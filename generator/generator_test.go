package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gerrywa/yacd/types"
)

func TestNewGenerator(t *testing.T) {
	options := types.ParseOptions{
		OutputFile: "compile_commands.json",
		Verbose:    false,
	}

	generator := NewGenerator(options)
	if generator == nil {
		t.Fatal("Generator should not be nil")
	}

	if generator.options.OutputFile != options.OutputFile {
		t.Errorf("Output file = %s, expected %s", generator.options.OutputFile, options.OutputFile)
	}
}

func TestConvertToCompilationEntry(t *testing.T) {
	tests := []struct {
		name     string
		options  types.ParseOptions
		entry    types.MakeLogEntry
		expected types.CompilationEntry
	}{
		{
			name: "Basic conversion",
			options: types.ParseOptions{
				UseRelativePaths: false,
			},
			entry: types.MakeLogEntry{
				WorkingDir: "/project/build",
				Compiler:   "gcc",
				Args:       []string{"gcc", "-c", "main.c", "-o", "main.o"},
				SourceFile: "main.c",
				OutputFile: "main.o",
			},
			expected: types.CompilationEntry{
				Directory: "/project/build",
				Arguments: []string{"gcc", "-c", "main.c", "-o", "main.o"},
				File:      "/project/build/main.c",
				Output:    "/project/build/main.o",
			},
		},
		{
			name: "Relative path conversion",
			options: types.ParseOptions{
				UseRelativePaths: true,
				BaseDir:          "/project",
			},
			entry: types.MakeLogEntry{
				WorkingDir: "/project/build",
				Compiler:   "gcc",
				Args:       []string{"gcc", "-c", "main.c", "-o", "main.o"},
				SourceFile: "main.c",
				OutputFile: "main.o",
			},
			expected: types.CompilationEntry{
				Directory: "build",
				Arguments: []string{"gcc", "-c", "main.c", "-o", "main.o"},
				File:      "build/main.c",
				Output:    "build/main.o",
			},
		},
		{
			name: "Absolute path source file",
			options: types.ParseOptions{
				UseRelativePaths: false,
			},
			entry: types.MakeLogEntry{
				WorkingDir: "/project/build",
				Compiler:   "gcc",
				Args:       []string{"gcc", "-c", "/project/src/main.c", "-o", "main.o"},
				SourceFile: "/project/src/main.c",
				OutputFile: "main.o",
			},
			expected: types.CompilationEntry{
				Directory: "/project/build",
				Arguments: []string{"gcc", "-c", "/project/src/main.c", "-o", "main.o"},
				File:      "/project/src/main.c", // Absolute path source file should not be modified
				Output:    "/project/build/main.o",
			},
		},
		{
			name: "Empty working directory",
			options: types.ParseOptions{
				UseRelativePaths: false,
			},
			entry: types.MakeLogEntry{
				WorkingDir: "",
				Compiler:   "gcc",
				Args:       []string{"gcc", "-c", "main.c", "-o", "main.o"},
				SourceFile: "main.c",
				OutputFile: "main.o",
			},
			expected: types.CompilationEntry{
				Directory: ".",
				Arguments: []string{"gcc", "-c", "main.c", "-o", "main.o"},
				File:      "main.c",
				Output:    "main.o",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewGenerator(tt.options)
			result, err := generator.convertToCompilationEntry(tt.entry)
			if err != nil {
				t.Fatalf("convertToCompilationEntry() failed: %v", err)
			}

			if filepath.ToSlash(result.Directory) != filepath.ToSlash(tt.expected.Directory) {
				t.Errorf("Directory = %s, expected %s", result.Directory, tt.expected.Directory)
			}

			if len(result.Arguments) != len(tt.expected.Arguments) {
				t.Errorf("Arguments length = %d, expected %d", len(result.Arguments), len(tt.expected.Arguments))
			} else {
				for i, arg := range tt.expected.Arguments {
					if result.Arguments[i] != arg {
						t.Errorf("Arguments[%d] = %s, expected %s", i, result.Arguments[i], arg)
					}
				}
			}

			if filepath.ToSlash(result.File) != filepath.ToSlash(tt.expected.File) {
				t.Errorf("File = %s, expected %s", result.File, tt.expected.File)
			}

			if filepath.ToSlash(result.Output) != filepath.ToSlash(tt.expected.Output) {
				t.Errorf("Output = %s, expected %s", result.Output, tt.expected.Output)
			}
		})
	}
}

func TestGenerateCompileCommands(t *testing.T) {
	options := types.ParseOptions{
		UseRelativePaths: false,
		Verbose:          false,
	}

	generator := NewGenerator(options)

	entries := []types.MakeLogEntry{
		{
			WorkingDir: "/project",
			Compiler:   "gcc",
			Args:       []string{"gcc", "-c", "main.c", "-o", "main.o"},
			SourceFile: "main.c",
			OutputFile: "main.o",
		},
		{
			WorkingDir: "/project",
			Compiler:   "gcc",
			Args:       []string{"gcc", "-c", "util.c", "-o", "util.o"},
			SourceFile: "util.c",
			OutputFile: "util.o",
		},
	}

	compileDB, err := generator.GenerateCompileCommands(entries)
	if err != nil {
		t.Fatalf("GenerateCompileCommands() failed: %v", err)
	}

	expectedCount := 2
	if len(compileDB) != expectedCount {
		t.Errorf("Generated %d compilation entries, expected %d", len(compileDB), expectedCount)
	}

	// Check first entry
	if len(compileDB) > 0 {
		entry := compileDB[0]
		if filepath.ToSlash(entry.Directory) != "/project" {
			t.Errorf("First entry directory = %s, expected /project", entry.Directory)
		}
		if filepath.ToSlash(entry.File) != "/project/main.c" {
			t.Errorf("First entry file = %s, expected /project/main.c", entry.File)
		}
		if len(entry.Arguments) == 0 || entry.Arguments[0] != "gcc" {
			t.Errorf("First entry compiler = %v, expected gcc", entry.Arguments)
		}
	}
}

func TestValidateCompileDB(t *testing.T) {
	options := types.ParseOptions{}
	generator := NewGenerator(options)

	tests := []struct {
		name      string
		compileDB types.CompilationDatabase
		wantError bool
	}{
		{
			name: "Valid compilation database",
			compileDB: types.CompilationDatabase{
				{
					Directory: "/project",
					Arguments: []string{"gcc", "-c", "main.c", "-o", "main.o"},
					File:      "main.c",
					Output:    "main.o",
				},
			},
			wantError: false,
		},
		{
			name:      "Empty compilation database",
			compileDB: types.CompilationDatabase{},
			wantError: true,
		},
		{
			name: "Missing directory",
			compileDB: types.CompilationDatabase{
				{
					Directory: "",
					Arguments: []string{"gcc", "-c", "main.c", "-o", "main.o"},
					File:      "main.c",
				},
			},
			wantError: true,
		},
		{
			name: "Missing file",
			compileDB: types.CompilationDatabase{
				{
					Directory: "/project",
					Arguments: []string{"gcc", "-c", "main.c", "-o", "main.o"},
					File:      "",
				},
			},
			wantError: true,
		},
		{
			name: "Missing arguments",
			compileDB: types.CompilationDatabase{
				{
					Directory: "/project",
					Arguments: []string{},
					File:      "main.c",
				},
			},
			wantError: true,
		},
		{
			name: "Has arguments without command (valid)",
			compileDB: types.CompilationDatabase{
				{
					Directory: "/project",
					Arguments: []string{"gcc", "-c", "main.c"},
					File:      "main.c",
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.ValidateCompileDB(tt.compileDB)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateCompileDB() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestWriteToFile(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "compile_commands.json")

	options := types.ParseOptions{
		OutputFile: outputFile,
		Verbose:    false,
	}

	generator := NewGenerator(options)

	compileDB := types.CompilationDatabase{
		{
			Directory: "/project",
			Arguments: []string{"gcc", "-c", "main.c", "-o", "main.o"},
			File:      "main.c",
			Output:    "main.o",
		},
		{
			Directory: "/project",
			Arguments: []string{"gcc", "-c", "util.c", "-o", "util.o"},
			File:      "util.c",
			Output:    "util.o",
		},
	}

	// Write to file
	err := generator.WriteToFile(compileDB, outputFile)
	if err != nil {
		t.Fatalf("WriteToFile() failed: %v", err)
	}

	// Check if file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatal("Output file not created")
	}

	// Read and validate file content
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var readDB types.CompilationDatabase
	if err := json.Unmarshal(data, &readDB); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(readDB) != len(compileDB) {
		t.Errorf("Read %d entries, expected %d", len(readDB), len(compileDB))
	}

	// Validate content
	for i, expected := range compileDB {
		if i >= len(readDB) {
			break
		}

		actual := readDB[i]
		if actual.Directory != expected.Directory {
			t.Errorf("Entry %d: Directory = %s, expected %s", i, actual.Directory, expected.Directory)
		}
		// Compare Arguments array
		if len(actual.Arguments) != len(expected.Arguments) {
			t.Errorf("Entry %d: Arguments length = %d, expected %d", i, len(actual.Arguments), len(expected.Arguments))
		} else {
			for j, arg := range expected.Arguments {
				if actual.Arguments[j] != arg {
					t.Errorf("Entry %d Arguments[%d] = %s, expected %s", i, j, actual.Arguments[j], arg)
				}
			}
		}
		if actual.File != expected.File {
			t.Errorf("Entry %d: File = %s, expected %s", i, actual.File, expected.File)
		}
		if actual.Output != expected.Output {
			t.Errorf("Entry %d: Output = %s, expected %s", i, actual.Output, expected.Output)
		}
	}
}

func TestWriteToFileWithSubdirectory(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "build", "output")
	outputFile := filepath.Join(subDir, "compile_commands.json")

	options := types.ParseOptions{
		OutputFile: outputFile,
		Verbose:    false,
	}

	generator := NewGenerator(options)

	compileDB := types.CompilationDatabase{
		{
			Directory: "/project",
			Arguments: []string{"gcc", "-c", "main.c", "-o", "main.o"},
			File:      "main.c",
		},
	}

	// Write to file (should automatically create subdirectory)
	err := generator.WriteToFile(compileDB, outputFile)
	if err != nil {
		t.Fatalf("WriteToFile() failed: %v", err)
	}

	// Check if file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatal("Output file not created")
	}

	// Check if subdirectory was created
	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Fatal("Subdirectory not created")
	}
}
