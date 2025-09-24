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
	"github.com/gerryqd/yacd/utils/errorutil"
)

// Global cache for compiler sysroots
var (
	compilerSysrootCache = make(map[string]string)
	cacheMutex           sync.RWMutex
)

// GenerateCompilationDatabase converts parsed make log entries to compilation database entries
func GenerateCompilationDatabase(entries []types.MakeLogEntry, options *types.ParseOptions) ([]types.CompilationEntry, int) {
	var compilationDB []types.CompilationEntry
	missingFiles := 0

	for i, entry := range entries {
		// Convert to compilation entry
		// Check if entry.Args already includes the compiler as the first element
		var commandArgs []string
		if len(entry.Args) > 0 && entry.Args[0] == entry.Compiler {
			// Args already includes the compiler, so just use entry.Args
			commandArgs = entry.Args
		} else {
			// Args doesn't include the compiler, so prepend it
			commandArgs = append([]string{entry.Compiler}, entry.Args...)
		}

		compilationEntry := types.CompilationEntry{
			Directory: entry.WorkingDir,
			Command:   strings.Join(commandArgs, " "),
			File:      entry.SourceFile,
			Output:    entry.OutputFile,
		}

		// Add sysroot include path if possible
		compilationEntry.Command = addSysrootIncludePath(compilationEntry.Command, entry.Compiler)

		// Apply path transformations if needed
		if options.UseRelativePaths {
			compilationEntry = convertToRelativePaths(compilationEntry, options.BaseDir)
		}

		// Add to database
		compilationDB = append(compilationDB, compilationEntry)

		// Check if source file exists
		filePath := compilationEntry.File
		if !filepath.IsAbs(filePath) {
			// For relative paths, try to resolve using the directory
			if filepath.IsAbs(compilationEntry.Directory) {
				filePath = filepath.Join(compilationEntry.Directory, filePath)
			} else if options.BaseDir != "" {
				filePath = filepath.Join(options.BaseDir, filePath)
			}
		}

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			missingFiles++
			// Print warning message with "Warning:" in yellow and the rest in normal color
			// Always print this warning, not just in verbose mode
			fmt.Printf("\033[33mWarning:\033[0m source file does not exist: %s (entry %d)\n", compilationEntry.File, i+1)
		}

		// Print verbose information if requested
		if options.Verbose {
			fmt.Printf("Entry %d: %s\n", i+1, compilationEntry.File)
		}
	}

	return compilationDB, missingFiles
}

// convertToRelativePaths converts absolute paths to relative paths based on baseDir
func convertToRelativePaths(entry types.CompilationEntry, baseDir string) types.CompilationEntry {
	// If no base directory is provided, try to infer it from the entry's directory
	if baseDir == "" {
		baseDir = entry.Directory
	}

	// Convert paths to relative
	relativeEntry := entry
	relativeEntry.Directory = getRelativePath(entry.Directory, baseDir)
	relativeEntry.File = getRelativePath(entry.File, baseDir)
	relativeEntry.Output = getRelativePath(entry.Output, baseDir)

	// Update command with relative paths
	relativeEntry.Command = strings.ReplaceAll(entry.Command, entry.File, relativeEntry.File)
	relativeEntry.Command = strings.ReplaceAll(entry.Command, entry.Output, relativeEntry.Output)

	return relativeEntry
}

// getRelativePath converts an absolute path to a relative path based on baseDir
func getRelativePath(path, baseDir string) string {
	// If path is already relative, return as is
	if !filepath.IsAbs(path) {
		return path
	}

	// Get relative path
	relPath, err := filepath.Rel(baseDir, path)
	if err != nil {
		// If we can't get a relative path, return the original
		return path
	}

	return relPath
}

// isValidPath checks if the given path is a valid directory path
func isValidPath(path string) bool {
	// Basic validation: non-empty, looks like a path
	if path == "" {
		return false
	}

	// Check if it's an absolute path or starts with common path indicators
	if strings.HasPrefix(path, "/") {
		// For absolute paths, check if it exists and is a directory
		info, err := os.Stat(path)
		return err == nil && info.IsDir()
	} else if strings.HasPrefix(path, ".") || strings.HasPrefix(path, "..") {
		// For relative paths, check if it exists and is a directory
		info, err := os.Stat(path)
		return err == nil && info.IsDir()
	}

	// If it doesn't look like a valid path format, it's probably not valid
	return false
}

// getCompilerSysroot returns the sysroot path for a given compiler
// It uses a cache to avoid repeated execution of the same command
func getCompilerSysroot(compilerPath string) (string, error) {
	// Check cache first
	cacheMutex.RLock()
	if sysroot, exists := compilerSysrootCache[compilerPath]; exists {
		cacheMutex.RUnlock()
		return sysroot, nil
	}
	cacheMutex.RUnlock()

	// Execute compiler with --print-sysroot option
	cmd := exec.Command(compilerPath, "--print-sysroot")
	output, err := cmd.Output()
	if err != nil {
		// If the command fails, return an empty string and the error
		return "", fmt.Errorf("failed to execute %s --print-sysroot: %w", compilerPath, err)
	}

	// Trim the output to get the sysroot path
	sysroot := strings.TrimSpace(string(output))

	// Validate the sysroot path
	if !isValidPath(sysroot) {
		// If the path is not valid, return empty string (no sysroot to add)
		// Cache the empty result to avoid repeated attempts
		cacheMutex.Lock()
		compilerSysrootCache[compilerPath] = ""
		cacheMutex.Unlock()
		return "", fmt.Errorf("invalid sysroot path returned by compiler: %s", sysroot)
	}

	// Cache the result
	cacheMutex.Lock()
	compilerSysrootCache[compilerPath] = sysroot
	cacheMutex.Unlock()

	return sysroot, nil
}

// addSysrootIncludePath adds the sysroot include path to the command if it doesn't already exist
func addSysrootIncludePath(command string, compiler string) string {
	// Get the sysroot path from the compiler
	sysroot, err := getCompilerSysroot(compiler)
	if err != nil {
		// If we can't get the sysroot, return the original command
		fmt.Printf("Warning: Could not get valid sysroot for compiler %s: %v\n", compiler, err)
		return command
	}

	// If sysroot is empty, return the original command
	if sysroot == "" {
		return command
	}

	// Check if the sysroot include path is already in the command
	sysrootIncludePath := "-I" + filepath.Join(sysroot, "usr", "include")

	// Split command into parts to check for the exact include path
	parts := strings.Fields(command)
	for _, part := range parts {
		if part == sysrootIncludePath {
			// If the path is already in the command, return the original command
			return command
		}
	}

	// Also check for the case where the include path is part of a combined argument like "-I/path"
	for _, part := range parts {
		if strings.HasPrefix(part, "-I") && strings.HasSuffix(part, filepath.Join(sysroot, "usr", "include")) {
			// If the path is already in the command, return the original command
			return command
		}
	}

	// Add the sysroot include path to the command
	// Find the position after the compiler name to insert the include path
	// We'll add it after the compiler name but before other options
	cmdParts := strings.SplitN(command, " ", 2)
	if len(cmdParts) < 2 {
		// If there's only the compiler name, just append the include path
		return command + " " + sysrootIncludePath
	}

	// Insert the sysroot include path after the compiler name
	return cmdParts[0] + " " + sysrootIncludePath + " " + cmdParts[1]
}

// WriteCompilationDatabase writes the compilation database to a JSON file
func WriteCompilationDatabase(compilationDB []types.CompilationEntry, outputFile string) error {
	// Create or truncate the output file
	file, err := os.Create(outputFile)
	if err != nil {
		return errorutil.WrapFileError(err, "create", outputFile)
	}
	defer file.Close()

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(compilationDB, "", "  ")
	if err != nil {
		return errorutil.WrapError(err, "failed to marshal compilation database to JSON")
	}

	// Write to file
	_, err = file.Write(data)
	if err != nil {
		return errorutil.WrapFileError(err, "write to", outputFile)
	}

	// Add newline at end of file
	_, err = file.WriteString("\n")
	if err != nil {
		return errorutil.WrapFileError(err, "write newline to", outputFile)
	}

	return nil
}
