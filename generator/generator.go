package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gerryqd/yacd/types"
	"github.com/gerryqd/yacd/utils/errorutil"
)

// GenerateCompilationDatabase converts parsed make log entries to compilation database entries
func GenerateCompilationDatabase(entries []types.MakeLogEntry, options *types.ParseOptions) ([]types.CompilationEntry, int) {
	var compilationDB []types.CompilationEntry
	missingFiles := 0

	for i, entry := range entries {
		// Convert to compilation entry
		compilationEntry := types.CompilationEntry{
			Directory: entry.WorkingDir,
			Command:   strings.Join(append([]string{entry.Compiler}, entry.Args...), " "),
			File:      entry.SourceFile,
			Output:    entry.OutputFile,
		}

		// Apply path transformations if needed
		if options.UseRelativePaths {
			compilationEntry = convertToRelativePaths(compilationEntry, options.BaseDir)
		}

		// Add to database
		compilationDB = append(compilationDB, compilationEntry)

		// Check if source file exists
		filePath := compilationEntry.File
		// If using relative paths, we need to resolve the full path
		if options.UseRelativePaths && !filepath.IsAbs(filePath) {
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
