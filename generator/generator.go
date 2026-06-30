package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gerryqd/yacd/types"
)

// sysrootResult holds the outcome of a single compiler sysroot lookup so that
// both the resolved path and any error can be memoized together.
type sysrootResult struct {
	sysroot string
	err     error
}

// SysrootCache caches compiler sysroot lookups within a single generation pass.
type SysrootCache struct {
	cache map[string]sysrootResult
}

// NewSysrootCache creates a new SysrootCache.
func NewSysrootCache() *SysrootCache {
	return &SysrootCache{cache: make(map[string]sysrootResult)}
}

// Get returns the cached sysroot for a compiler, looking it up if necessary.
// Both successful and failed lookups are memoized so that subsequent calls for
// the same compiler return a consistent result without re-running it.
func (sc *SysrootCache) Get(compilerPath string) (string, error) {
	if res, ok := sc.cache[compilerPath]; ok {
		return res.sysroot, res.err
	}

	res := sc.lookup(compilerPath)
	sc.cache[compilerPath] = res
	return res.sysroot, res.err
}

// lookup queries the compiler for its sysroot path.
func (sc *SysrootCache) lookup(compilerPath string) sysrootResult {
	cmd := exec.Command(compilerPath, "--print-sysroot")
	output, err := cmd.Output()
	if err != nil {
		return sysrootResult{"", fmt.Errorf("failed to execute %s --print-sysroot: %w", compilerPath, err)}
	}

	sysroot := strings.TrimSpace(string(output))
	if !isValidPath(sysroot) {
		return sysrootResult{"", fmt.Errorf("invalid sysroot path returned by compiler: %s", sysroot)}
	}

	return sysrootResult{sysroot, nil}
}

// getSysrootIncludePath looks up the compiler sysroot and returns the include path.
// Returns empty string and nil error if the sysroot is valid but empty.
func getSysrootIncludePath(compiler string, cache *SysrootCache) (string, error) {
	sysroot, err := cache.Get(compiler)
	if err != nil {
		return "", err
	}
	if sysroot == "" {
		return "", nil
	}
	return "-I" + filepath.Join(sysroot, "usr", "include"), nil
}

// GenerateCompilationDatabase converts parsed make log entries to compilation database entries.
// Returns the compilation database and a list of missing source files.
func GenerateCompilationDatabase(entries []types.MakeLogEntry, options *types.ParseOptions) ([]types.CompilationEntry, []string) {
	// A nil options pointer is treated as default options to avoid a nil-dereference
	// in this public API.
	if options == nil {
		options = &types.ParseOptions{}
	}

	// Deduplicate entries if enabled
	if options.Deduplicate {
		entries = deduplicateEntries(entries)
	}

	var compilationDB []types.CompilationEntry
	sysrootCache := NewSysrootCache()

	for i, entry := range entries {
		// Build command args
		var commandArgs []string
		if len(entry.Args) > 0 && entry.Args[0] == entry.Compiler {
			commandArgs = entry.Args
		} else {
			commandArgs = append([]string{entry.Compiler}, entry.Args...)
		}

		// Resolve paths
		workingDir := entry.WorkingDir
		sourceFile := entry.SourceFile
		outputFile := entry.OutputFile

		// Apply path transformations if needed
		if options.UseRelativePaths {
			baseDir := options.BaseDir
			if baseDir == "" {
				baseDir = workingDir
			}
			workingDir = getRelativePath(workingDir, baseDir)
			sourceFile = getRelativePath(sourceFile, baseDir)
			outputFile = getRelativePath(outputFile, baseDir)

			// Rebuild args with relative paths
			newArgs := make([]string, len(commandArgs))
			for j, arg := range commandArgs {
				if arg == entry.SourceFile {
					newArgs[j] = sourceFile
				} else if arg == entry.OutputFile {
					newArgs[j] = outputFile
				} else {
					newArgs[j] = arg
				}
			}
			commandArgs = newArgs
		}

		compilationEntry := types.CompilationEntry{
			Directory: workingDir,
			Command:   strings.Join(commandArgs, " "),
			File:      sourceFile,
			Output:    outputFile,
		}

		// Use arguments array format when enabled (preferred by clangd)
		if options.UseArguments {
			compilationEntry.Arguments = commandArgs
			compilationEntry.Command = ""
		}

		// Add sysroot include path only when explicitly enabled
		if options.AddSysroot {
			if options.UseArguments {
				compilationEntry.Arguments = addSysrootToArgs(compilationEntry.Arguments, entry.Compiler, sysrootCache)
			} else {
				compilationEntry.Command = addSysrootIncludePath(compilationEntry.Command, entry.Compiler, sysrootCache)
			}
		}

		compilationDB = append(compilationDB, compilationEntry)

		if options.Verbose {
			fmt.Printf("Entry %d: %s\n", i+1, compilationEntry.File)
		}
	}

	// Check for missing source files
	missingFiles := checkMissingFiles(compilationDB, options)

	return compilationDB, missingFiles
}

// deduplicateEntries removes duplicate entries based on (WorkingDir, SourceFile) key.
func deduplicateEntries(entries []types.MakeLogEntry) []types.MakeLogEntry {
	seen := make(map[string]struct{})
	result := make([]types.MakeLogEntry, 0, len(entries))
	for _, entry := range entries {
		key := entry.WorkingDir + "|" + entry.SourceFile
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			result = append(result, entry)
		}
	}
	return result
}

// checkMissingFiles checks source file existence.
func checkMissingFiles(db []types.CompilationEntry, options *types.ParseOptions) []string {
	var missing []string
	for i, entry := range db {
		filePath := entry.File
		if !filepath.IsAbs(filePath) {
			if filepath.IsAbs(entry.Directory) {
				filePath = filepath.Join(entry.Directory, filePath)
			} else if options.BaseDir != "" {
				filePath = filepath.Join(options.BaseDir, filePath)
			}
		}
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			missing = append(missing, fmt.Sprintf("%s (entry %d)", entry.File, i+1))
		}
	}
	return missing
}

// addSysrootToArgs adds sysroot include path to arguments array.
func addSysrootToArgs(args []string, compiler string, cache *SysrootCache) []string {
	includePath, err := getSysrootIncludePath(compiler, cache)
	if err != nil {
		fmt.Printf("Warning: Could not get valid sysroot for compiler %s: %v\n", compiler, err)
		return args
	}
	if includePath == "" {
		return args
	}

	for _, arg := range args {
		if arg == includePath {
			return args
		}
	}

	result := make([]string, 0, len(args)+1)
	result = append(result, args[0])
	result = append(result, includePath)
	result = append(result, args[1:]...)
	return result
}

// addSysrootIncludePath adds the sysroot include path to the command string.
func addSysrootIncludePath(command string, compiler string, cache *SysrootCache) string {
	includePath, err := getSysrootIncludePath(compiler, cache)
	if err != nil {
		fmt.Printf("Warning: Could not get valid sysroot for compiler %s: %v\n", compiler, err)
		return command
	}
	if includePath == "" {
		return command
	}

	// Use proper command splitting to handle quoted arguments
	parts := types.SplitCommandLine(command)
	for i, part := range parts {
		// Already present as a single "-I<path>" token.
		if part == includePath {
			return command
		}
		// Present as a space-separated "-I <path>" pair.
		if part == "-I" && i+1 < len(parts) && parts[i+1] == includePath[2:] {
			return command
		}
	}

	if len(parts) < 2 {
		return command + " " + includePath
	}

	return parts[0] + " " + includePath + " " + strings.Join(parts[1:], " ")
}

// getRelativePath converts an absolute path to a relative path based on baseDir
func getRelativePath(path, baseDir string) string {
	if !filepath.IsAbs(path) {
		return path
	}

	relPath, err := filepath.Rel(baseDir, path)
	if err != nil {
		return path
	}

	return relPath
}

// isValidPath checks if the given path is a valid directory path
func isValidPath(path string) bool {
	if path == "" {
		return false
	}

	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, ".") {
		info, err := os.Stat(path)
		return err == nil && info.IsDir()
	}

	return false
}

// WriteCompilationDatabase writes the compilation database to a JSON file atomically
func WriteCompilationDatabase(compilationDB []types.CompilationEntry, outputFile string) error {
	data, err := json.MarshalIndent(compilationDB, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal compilation database to JSON: %w", err)
	}
	data = append(data, '\n')

	tmpFile := outputFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", tmpFile, err)
	}

	if err := os.Rename(tmpFile, outputFile); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to rename to file %s: %w", outputFile, err)
	}

	return nil
}
