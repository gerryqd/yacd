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
	entries := []types.MakeLogEntry{
		{WorkingDir: "/project", Compiler: "gcc", Args: []string{"gcc", "-c", "main.c", "-o", "main.o"}, SourceFile: "main.c", OutputFile: "main.o"},
		{WorkingDir: "/project", Compiler: "g++", Args: []string{"g++", "-c", "util.cpp", "-o", "util.o"}, SourceFile: "util.cpp", OutputFile: "util.o"},
	}

	result, _ := GenerateCompilationDatabase(entries, &types.ParseOptions{})
	if len(result) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(result))
	}
}

func TestGenerateCompilationDatabaseWithRelativePaths(t *testing.T) {
	baseDir := "/project"
	if runtime.GOOS == "windows" {
		baseDir = `C:\project`
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

	result, _ := GenerateCompilationDatabase(entries, &types.ParseOptions{
		UseRelativePaths: true,
		BaseDir:          baseDir,
	})

	if len(result) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(result))
	}

	expectedFile := filepath.ToSlash(filepath.Join("build", "main.c"))
	actualFile := filepath.ToSlash(result[0].File)
	if actualFile != expectedFile {
		t.Errorf("File = %s, expected %s", actualFile, expectedFile)
	}
}

func TestGetRelativePath(t *testing.T) {
	baseDir := "/project"
	if runtime.GOOS == "windows" {
		baseDir = `C:\project`
	}

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"Subdirectory", filepath.Join(baseDir, "build", "main.c"), "build/main.c"},
		{"Same directory", filepath.Join(baseDir, "main.c"), "main.c"},
		{"Already relative", "main.c", "main.c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filepath.ToSlash(getRelativePath(tt.path, baseDir))
			expected := filepath.ToSlash(tt.expected)
			if result != expected {
				t.Errorf("Got %s, expected %s", result, expected)
			}
		})
	}
}

func TestWriteCompilationDatabase(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test_compile_commands.json")
	defer os.Remove(tmpFile)

	entries := []types.CompilationEntry{
		{Directory: "/project", Command: "gcc -c main.c -o main.o", File: "/project/main.c", Output: "/project/main.o"},
	}

	if err := WriteCompilationDatabase(entries, tmpFile); err != nil {
		t.Fatalf("WriteCompilationDatabase error: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	var result []types.CompilationEntry
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(result) != 1 || result[0].Command != "gcc -c main.c -o main.o" {
		t.Errorf("Unexpected result: %+v", result)
	}
}
