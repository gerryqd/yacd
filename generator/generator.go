package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gerryqd/yacd/types"
)

// Global cache for compiler sysroots
var (
	compilerSysrootCache = make(map[string]string)
	cacheMutex           sync.RWMutex
)

// GenerateCompilationDatabase converts parsed make log entries to compilation database entries.
// Returns the compilation database and a list of missing source files.
func GenerateCompilationDatabase(entries []types.MakeLogEntry, options *types.ParseOptions) ([]types.CompilationEntry, []string) {
	var compilationDB []types.CompilationEntry
	var missingFiles []string

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

		// Add sysroot include path only when explicitly enabled
		if options.AddSysroot {
			compilationEntry.Command = addSysrootIncludePath(compilationEntry.Command, entry.Compiler)
		}

		compilationDB = append(compilationDB, compilationEntry)

		// Check if source file exists
		filePath := compilationEntry.File
		if !filepath.IsAbs(filePath) {
			if filepath.IsAbs(compilationEntry.Directory) {
				filePath = filepath.Join(compilationEntry.Directory, filePath)
			} else if options.BaseDir != "" {
				filePath = filepath.Join(options.BaseDir, filePath)
			}
		}

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			missingFiles = append(missingFiles, fmt.Sprintf("%s (entry %d)", compilationEntry.File, i+1))
		}

		if options.Verbose {
			fmt.Printf("Entry %d: %s\n", i+1, compilationEntry.File)
		}
	}

	return compilationDB, missingFiles
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

// getCompilerSysroot returns the sysroot path for a given compiler
func getCompilerSysroot(compilerPath string) (string, error) {
	cacheMutex.RLock()
	if sysroot, exists := compilerSysrootCache[compilerPath]; exists {
		cacheMutex.RUnlock()
		return sysroot, nil
	}
	cacheMutex.RUnlock()

	cmd := exec.Command(compilerPath, "--print-sysroot")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute %s --print-sysroot: %w", compilerPath, err)
	}

	sysroot := strings.TrimSpace(string(output))

	if !isValidPath(sysroot) {
		cacheMutex.Lock()
		compilerSysrootCache[compilerPath] = ""
		cacheMutex.Unlock()
		return "", fmt.Errorf("invalid sysroot path returned by compiler: %s", sysroot)
	}

	cacheMutex.Lock()
	compilerSysrootCache[compilerPath] = sysroot
	cacheMutex.Unlock()

	return sysroot, nil
}

// addSysrootIncludePath adds the sysroot include path to the command if it doesn't already exist
func addSysrootIncludePath(command string, compiler string) string {
	sysroot, err := getCompilerSysroot(compiler)
	if err != nil {
		fmt.Printf("Warning: Could not get valid sysroot for compiler %s: %v\n", compiler, err)
		return command
	}

	if sysroot == "" {
		return command
	}

	sysrootIncludePath := "-I" + filepath.Join(sysroot, "usr", "include")

	parts := strings.Fields(command)
	for _, part := range parts {
		if part == sysrootIncludePath {
			return command
		}
	}

	for _, part := range parts {
		if strings.HasPrefix(part, "-I") && strings.HasSuffix(part, filepath.Join(sysroot, "usr", "include")) {
			return command
		}
	}

	cmdParts := strings.SplitN(command, " ", 2)
	if len(cmdParts) < 2 {
		return command + " " + sysrootIncludePath
	}

	return cmdParts[0] + " " + sysrootIncludePath + " " + cmdParts[1]
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
