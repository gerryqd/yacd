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
			Command:   quoteCommand(commandArgs),
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
// A NUL separator is used so that paths containing '|' cannot collide.
func deduplicateEntries(entries []types.MakeLogEntry) []types.MakeLogEntry {
	seen := make(map[string]struct{})
	result := make([]types.MakeLogEntry, 0, len(entries))
	for _, entry := range entries {
		key := entry.WorkingDir + "\x00" + entry.SourceFile
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

	for i, arg := range args {
		// Already present as a single "-I<path>" token.
		if arg == includePath {
			return args
		}
		// Present as a space-separated "-I <path>" pair.
		if arg == "-I" && i+1 < len(args) && args[i+1] == includePath[2:] {
			return args
		}
	}

	result := make([]string, 0, len(args)+1)
	result = append(result, args[0])
	result = append(result, includePath)
	result = append(result, args[1:]...)
	return result
}

// addSysrootIncludePath adds the sysroot include path to a command string. It
// splits the string, delegates the insertion to addSysrootToArgs, and re-quotes
// the result so the command-string and arguments-array forms share a single
// code path instead of duplicating the presence check and insertion logic.
func addSysrootIncludePath(command string, compiler string, cache *SysrootCache) string {
	parts := types.SplitCommandLine(command)
	return quoteCommand(addSysrootToArgs(parts, compiler, cache))
}

// quoteShellArg wraps an argument in single quotes when it contains characters
// that would otherwise be split or interpreted by a shell when the command is
// reconstructed as a plain string.
func quoteShellArg(arg string) string {
	if arg == "" {
		return "''"
	}
	for _, ch := range arg {
		switch ch {
		case ' ', '\t', '\n', '\r', '\'', '"', '\\', '$', '`':
			return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
		}
	}
	return arg
}

// quoteCommand joins command arguments into a shell command string, quoting
// arguments that contain shell-special characters so the string can be parsed
// back into the same arguments.
func quoteCommand(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = quoteShellArg(arg)
	}
	return strings.Join(quoted, " ")
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

	// Use a unique temporary file in the target directory so concurrent runs
	// writing to the same output do not clobber each other's temp files.
	dir := filepath.Dir(outputFile)
	tmpFile, err := os.CreateTemp(dir, filepath.Base(outputFile)+".tmp*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file in %s: %w", dir, err)
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write file %s: %w", tmpName, err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close file %s: %w", tmpName, err)
	}
	if err := os.Chmod(tmpName, 0644); err != nil {
		return fmt.Errorf("failed to set permissions on file %s: %w", tmpName, err)
	}

	if err := os.Rename(tmpName, outputFile); err != nil {
		return fmt.Errorf("failed to rename to file %s: %w", outputFile, err)
	}

	return nil
}
