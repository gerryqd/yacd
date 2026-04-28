package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gerryqd/yacd/types"
)

func TestGenerateCompilationDatabase(t *testing.T) {
	tests := []struct {
		name     string
		entries  []types.MakeLogEntry
		options  *types.ParseOptions
		expected int
	}{
		{
			name: "Basic conversion",
			entries: []types.MakeLogEntry{
				{
					WorkingDir: "/project",
					Compiler:   "gcc",
					Args:       []string{"gcc", "-c", "main.c", "-o", "main.o"},
					SourceFile: "main.c",
					OutputFile: "main.o",
				},
			},
			options: &types.ParseOptions{
				UseRelativePaths: false,
				Verbose:          false,
			},
			expected: 1,
		},
		{
			name: "Multiple entries",
			entries: []types.MakeLogEntry{
				{
					WorkingDir: "/project",
					Compiler:   "gcc",
					Args:       []string{"gcc", "-c", "main.c", "-o", "main.o"},
					SourceFile: "main.c",
					OutputFile: "main.o",
				},
				{
					WorkingDir: "/project",
					Compiler:   "g++",
					Args:       []string{"g++", "-c", "util.cpp", "-o", "util.o"},
					SourceFile: "util.cpp",
					OutputFile: "util.o",
				},
			},
			options: &types.ParseOptions{
				UseRelativePaths: false,
				Verbose:          false,
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := GenerateCompilationDatabase(tt.entries, tt.options)
			if len(result) != tt.expected {
				t.Errorf("GenerateCompilationDatabase() = %d entries, expected %d", len(result), tt.expected)
			}
		})
	}
}

func TestGenerateCompilationDatabaseWithRelativePaths(t *testing.T) {
	// Use platform-specific paths for testing
	var baseDir string
	if runtime.GOOS == "windows" {
		baseDir = `C:\project`
	} else {
		baseDir = "/project"
	}

	entries := []types.MakeLogEntry{
		{
			WorkingDir: filepath.Join(baseDir, "build"),
			Compiler:   "gcc",
			Args:       []string{"gcc", "-c", filepath.Join(baseDir, "build", "main.c"), "-o", filepath.Join(baseDir, "build", "main.o")},
			SourceFile: filepath.Join(baseDir, "build", "main.c"),
			OutputFile: filepath.Join(baseDir, "build", "main.o"),
		},
	}

	options := &types.ParseOptions{
		UseRelativePaths: true,
		BaseDir:          baseDir,
		Verbose:          false,
	}

	result, _ := GenerateCompilationDatabase(entries, options)
	if len(result) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(result))
	}

	entry := result[0]

	// Directory should be relative
	expectedDir := filepath.ToSlash(filepath.Join("build"))
	actualDir := filepath.ToSlash(entry.Directory)
	if actualDir != expectedDir {
		t.Errorf("Directory = %s, expected %s", actualDir, expectedDir)
	}

	// File path should be relative
	expectedFile := filepath.ToSlash(filepath.Join("build", "main.c"))
	actualFile := filepath.ToSlash(entry.File)
	if actualFile != expectedFile {
		t.Errorf("File = %s, expected %s", actualFile, expectedFile)
	}

	// Output path should be relative
	expectedOutput := filepath.ToSlash(filepath.Join("build", "main.o"))
	actualOutput := filepath.ToSlash(entry.Output)
	if actualOutput != expectedOutput {
		t.Errorf("Output = %s, expected %s", actualOutput, expectedOutput)
	}

	// Command should contain relative paths, not absolute
	if filepath.IsAbs(entry.File) && filepath.ToSlash(entry.File) != expectedFile {
		t.Errorf("Command should use relative paths: %s", entry.Command)
	}
}

func TestGenerateCompilationDatabaseWithSameDirRelativePaths(t *testing.T) {
	var baseDir string
	if runtime.GOOS == "windows" {
		baseDir = `C:\project`
	} else {
		baseDir = "/project"
	}

	entries := []types.MakeLogEntry{
		{
			WorkingDir: baseDir,
			Compiler:   "gcc",
			Args:       []string{"gcc", "-c", filepath.Join(baseDir, "main.c"), "-o", filepath.Join(baseDir, "main.o")},
			SourceFile: filepath.Join(baseDir, "main.c"),
			OutputFile: filepath.Join(baseDir, "main.o"),
		},
	}

	options := &types.ParseOptions{
		UseRelativePaths: true,
		BaseDir:          baseDir,
		Verbose:          false,
	}

	result, _ := GenerateCompilationDatabase(entries, options)
	if len(result) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(result))
	}

	entry := result[0]

	if filepath.ToSlash(entry.Directory) != "." {
		t.Errorf("Directory = %s, expected .", entry.Directory)
	}

	if filepath.ToSlash(entry.File) != "main.c" {
		t.Errorf("File = %s, expected main.c", entry.File)
	}

	if filepath.ToSlash(entry.Output) != "main.o" {
		t.Errorf("Output = %s, expected main.o", entry.Output)
	}
}

func TestGetRelativePath(t *testing.T) {
	// Use platform-specific paths for testing
	var baseDir string
	if runtime.GOOS == "windows" {
		baseDir = `C:\project`
	} else {
		baseDir = "/project"
	}

	tests := []struct {
		name     string
		path     string
		baseDir  string
		expected string
	}{
		{
			name:     "Basic relative path",
			path:     filepath.Join(baseDir, "build", "main.c"),
			baseDir:  baseDir,
			expected: "build/main.c",
		},
		{
			name:     "Same directory",
			path:     filepath.Join(baseDir, "main.c"),
			baseDir:  baseDir,
			expected: "main.c",
		},
		{
			name:     "Relative path unchanged",
			path:     "main.c",
			baseDir:  baseDir,
			expected: "main.c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getRelativePath(tt.path, tt.baseDir)
			resultNormalized := filepath.ToSlash(result)
			expectedNormalized := filepath.ToSlash(tt.expected)
			if resultNormalized != expectedNormalized {
				t.Errorf("getRelativePath(%s, %s) = %s, expected %s", tt.path, tt.baseDir, resultNormalized, expectedNormalized)
			}
		})
	}
}

func TestWriteCompilationDatabase(t *testing.T) {
	// Create temporary file for testing
	tmpFile := filepath.Join(os.TempDir(), "test_compile_commands.json")
	defer os.Remove(tmpFile)

	entries := []types.CompilationEntry{
		{
			Directory: "/project",
			Command:   "gcc -c main.c -o main.o",
			File:      "/project/main.c",
			Output:    "/project/main.o",
		},
	}

	err := WriteCompilationDatabase(entries, tmpFile)
	if err != nil {
		t.Fatalf("WriteCompilationDatabase() error = %v", err)
	}

	// Check that file was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatal("Output file was not created")
	}

	// Read and parse the file
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var result []types.CompilationEntry
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 entry in output, got %d", len(result))
	}

	if result[0].Directory != "/project" {
		t.Errorf("Directory = %s, expected /project", result[0].Directory)
	}

	if result[0].Command != "gcc -c main.c -o main.o" {
		t.Errorf("Command = %s, expected 'gcc -c main.c -o main.o'", result[0].Command)
	}
}
