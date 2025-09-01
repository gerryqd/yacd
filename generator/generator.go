package generator

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gerrywa/yacd/types"
	"github.com/gerrywa/yacd/utils/errorutil"
	"github.com/gerrywa/yacd/utils/pathutil"
)

// Generator compilation database generator
type Generator struct {
	options types.ParseOptions
}

// NewGenerator creates a new generator
func NewGenerator(options types.ParseOptions) *Generator {
	return &Generator{
		options: options,
	}
}

// GenerateCompileCommands generates compile_commands.json from make log entries
func (g *Generator) GenerateCompileCommands(entries []types.MakeLogEntry) (types.CompilationDatabase, error) {
	var compileDB types.CompilationDatabase

	for _, entry := range entries {
		compEntry, err := g.convertToCompilationEntry(entry)
		if err != nil {
			if g.options.Verbose {
				fmt.Printf("Warning: failed to convert entry: %v\n", err)
			}
			continue
		}

		compileDB = append(compileDB, *compEntry)
	}

	return compileDB, nil
}

// convertToCompilationEntry converts MakeLogEntry to CompilationEntry
func (g *Generator) convertToCompilationEntry(entry types.MakeLogEntry) (*types.CompilationEntry, error) {
	// Handle working directory
	directory := entry.WorkingDir
	if directory == "" {
		directory = "."
	}

	// Handle source file path
	sourceFile := entry.SourceFile
	// Check for both Windows absolute paths and Unix absolute paths
	// Also handle Windows-style paths that start with backslash
	isAbsolute := pathutil.IsAbsolutePath(sourceFile)
	if !isAbsolute && directory != "" {
		sourceFile = pathutil.JoinPaths(directory, sourceFile)
	}

	// Use relative paths if needed
	if g.options.UseRelativePaths && g.options.BaseDir != "" {
		if relPath, err := pathutil.ToRelativePath(g.options.BaseDir, sourceFile); err == nil {
			sourceFile = relPath
		}

		if relDir, err := pathutil.ToRelativePath(g.options.BaseDir, directory); err == nil {
			directory = relDir
		}
	}

	// Clean and normalize paths
	sourceFile = pathutil.NormalizePath(sourceFile)
	directory = pathutil.NormalizePath(directory)

	// Handle output file path
	output := entry.OutputFile
	if output != "" {
		// Check if output path is absolute (considering both Unix and Windows styles)
		outputIsAbsolute := pathutil.IsAbsolutePath(output)
		if !outputIsAbsolute && entry.WorkingDir != "" {
			output = pathutil.JoinPaths(entry.WorkingDir, output)
		}
		if g.options.UseRelativePaths && g.options.BaseDir != "" {
			if relPath, err := pathutil.ToRelativePath(g.options.BaseDir, output); err == nil {
				output = relPath
			}
		}
		output = pathutil.NormalizePath(output)
	}

	return &types.CompilationEntry{
		Directory: directory,
		Arguments: entry.Args,
		File:      sourceFile,
		Output:    output,
	}, nil
}

// WriteToFile writes compilation database to file
func (g *Generator) WriteToFile(compileDB types.CompilationDatabase, filename string) error {
	// Create output directory if it doesn't exist
	dir := pathutil.GetDirectoryFromPath(filename)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errorutil.WrapError(err, "failed to create output directory")
		}
	}

	// Serialize database to JSON
	data, err := json.MarshalIndent(compileDB, "", "  ")
	if err != nil {
		return errorutil.WrapError(err, "failed to serialize compilation database")
	}

	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return errorutil.WrapFileError(err, "write", filename)
	}

	if g.options.Verbose {
		fmt.Printf("Successfully generated compile_commands.json: %s\n", filename)
		fmt.Printf("Contains %d compilation entries\n", len(compileDB))
	}

	return nil
}

// ValidateCompileDB validates the compilation database
func (g *Generator) ValidateCompileDB(compileDB types.CompilationDatabase) error {
	if len(compileDB) == 0 {
		return errorutil.CreateEmptyInputError("compilation database")
	}

	missingFiles := 0
	for i, entry := range compileDB {
		if entry.Directory == "" {
			return errorutil.NewErrorf("entry %d: working directory cannot be empty", i)
		}

		if entry.File == "" {
			return errorutil.NewErrorf("entry %d: source file path cannot be empty", i)
		}

		if len(entry.Arguments) == 0 {
			return errorutil.NewErrorf("entry %d: arguments cannot be empty", i)
		}

		// Check if source file exists
		filePath := entry.File
		// The file path in compile_commands.json should be absolute or properly resolved
		// For validation, we use the file path as-is since generator already handled path resolution

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			missingFiles++
			fmt.Printf("Warning: source file does not exist: %s (entry %d)\n", entry.File, i+1)
			if g.options.Verbose {
				fmt.Printf("  Resolved path: %s\n", filePath)
				fmt.Printf("  Working directory: %s\n", entry.Directory)
			}
		}
	}

	if missingFiles > 0 {
		fmt.Printf("Warning: %d source file(s) do not exist\n", missingFiles)
	}

	return nil
}
