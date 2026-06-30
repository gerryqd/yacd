package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestGenerateCompilationDatabaseWithArgumentsFormat(t *testing.T) {
	entries := []types.MakeLogEntry{
		{WorkingDir: "/project", Compiler: "gcc", Args: []string{"gcc", "-c", "main.c", "-o", "main.o"}, SourceFile: "main.c", OutputFile: "main.o"},
	}

	result, _ := GenerateCompilationDatabase(entries, &types.ParseOptions{UseArguments: true})
	if len(result) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(result))
	}
	if result[0].Command != "" {
		t.Errorf("Command should be empty when UseArguments is true, got: %s", result[0].Command)
	}
	if len(result[0].Arguments) != 5 {
		t.Errorf("Expected 5 arguments, got %d", len(result[0].Arguments))
	}
	if result[0].Arguments[0] != "gcc" {
		t.Errorf("First argument should be gcc, got: %s", result[0].Arguments[0])
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

func TestWriteCompilationDatabaseWithArguments(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test_compile_args.json")
	defer os.Remove(tmpFile)

	args := []string{"gcc", "-c", "main.c", "-o", "main.o"}
	entries := []types.CompilationEntry{
		{Directory: "/project", Arguments: args, File: "/project/main.c", Output: "/project/main.o"},
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

	if len(result) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(result))
	}
	if len(result[0].Arguments) != 5 {
		t.Errorf("Expected 5 arguments, got %d", len(result[0].Arguments))
	}
}

func TestDeduplicateEntries(t *testing.T) {
	entries := []types.MakeLogEntry{
		{WorkingDir: "/project", Compiler: "gcc", Args: []string{"gcc", "-c", "main.c"}, SourceFile: "main.c"},
		{WorkingDir: "/project", Compiler: "gcc", Args: []string{"gcc", "-c", "main.c", "-O2"}, SourceFile: "main.c"},
		{WorkingDir: "/project", Compiler: "gcc", Args: []string{"gcc", "-c", "util.c"}, SourceFile: "util.c"},
	}

	result := deduplicateEntries(entries)
	if len(result) != 2 {
		t.Errorf("Expected 2 entries after dedup, got %d", len(result))
	}
	if result[0].SourceFile != "main.c" {
		t.Errorf("First entry source = %s, expected main.c", result[0].SourceFile)
	}
	if result[1].SourceFile != "util.c" {
		t.Errorf("Second entry source = %s, expected util.c", result[1].SourceFile)
	}
}

func TestDeduplicateEntriesEmpty(t *testing.T) {
	result := deduplicateEntries(nil)
	if len(result) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(result))
	}
}

func TestCheckMissingFilesWithExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "main.c")
	if err := os.WriteFile(existingFile, []byte("int main(){}"), 0644); err != nil {
		t.Fatal(err)
	}

	db := []types.CompilationEntry{
		{Directory: tmpDir, File: existingFile},
	}

	missing := checkMissingFiles(db, &types.ParseOptions{})
	if len(missing) != 0 {
		t.Errorf("Expected 0 missing files, got %d: %v", len(missing), missing)
	}
}

func TestCheckMissingFilesWithNonExistentFile(t *testing.T) {
	db := []types.CompilationEntry{
		{Directory: "/nonexistent", File: "/nonexistent/missing.c"},
	}

	missing := checkMissingFiles(db, &types.ParseOptions{})
	if len(missing) != 1 {
		t.Errorf("Expected 1 missing file, got %d", len(missing))
	}
}

func TestIsValidPath(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		valid bool
	}{
		{"Empty", "", false},
		{"Relative file", "main.c", false},
		{"Existing dir", t.TempDir(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidPath(tt.path); got != tt.valid {
				t.Errorf("isValidPath(%q) = %v, expected %v", tt.path, got, tt.valid)
			}
		})
	}
}

func TestSysrootCache(t *testing.T) {
	cache := NewSysrootCache()
	if cache == nil {
		t.Fatal("NewSysrootCache returned nil")
	}

	// Test with a non-existent compiler (should error)
	_, err := cache.Get("/nonexistent/compiler-gcc")
	if err == nil {
		t.Error("Expected error for non-existent compiler")
	}
}

func TestGenerateCompilationDatabaseWithDedup(t *testing.T) {
	entries := []types.MakeLogEntry{
		{WorkingDir: "/project", Compiler: "gcc", Args: []string{"gcc", "-c", "main.c"}, SourceFile: "main.c", OutputFile: "main.o"},
		{WorkingDir: "/project", Compiler: "gcc", Args: []string{"gcc", "-c", "main.c", "-O2"}, SourceFile: "main.c", OutputFile: "main.o"},
		{WorkingDir: "/project", Compiler: "gcc", Args: []string{"gcc", "-c", "util.c"}, SourceFile: "util.c", OutputFile: "util.o"},
	}

	result, _ := GenerateCompilationDatabase(entries, &types.ParseOptions{Deduplicate: true})
	if len(result) != 2 {
		t.Errorf("Expected 2 entries after dedup, got %d", len(result))
	}
}

func TestGenerateCompilationDatabaseWithoutDedup(t *testing.T) {
	entries := []types.MakeLogEntry{
		{WorkingDir: "/project", Compiler: "gcc", Args: []string{"gcc", "-c", "main.c"}, SourceFile: "main.c", OutputFile: "main.o"},
		{WorkingDir: "/project", Compiler: "gcc", Args: []string{"gcc", "-c", "main.c", "-O2"}, SourceFile: "main.c", OutputFile: "main.o"},
	}

	result, _ := GenerateCompilationDatabase(entries, &types.ParseOptions{Deduplicate: false})
	if len(result) != 2 {
		t.Errorf("Expected 2 entries without dedup, got %d", len(result))
	}
}

func TestAddSysrootToArgs(t *testing.T) {
	cache := NewSysrootCache()
	// Pre-populate cache to avoid executing a real compiler
	cache.cache["test-gcc"] = sysrootResult{sysroot: "/test/sysroot"}

	args := []string{"test-gcc", "-c", "main.c"}
	result := addSysrootToArgs(args, "test-gcc", cache)
	if len(result) != 4 {
		t.Errorf("Expected 4 args after adding sysroot, got %d: %v", len(result), result)
	}
	if result[1] != "-I/test/sysroot/usr/include" {
		t.Errorf("Expected sysroot include path at index 1, got: %s", result[1])
	}
}

func TestAddSysrootIncludePath(t *testing.T) {
	cache := NewSysrootCache()
	cache.cache["test-gcc"] = sysrootResult{sysroot: "/test/sysroot"}

	result := addSysrootIncludePath("test-gcc -c main.c", "test-gcc", cache)
	if result == "test-gcc -c main.c" {
		t.Error("Expected sysroot to be added to command")
	}
	if !strings.Contains(result, "-I/test/sysroot/usr/include") {
		t.Errorf("Expected sysroot include in result: %s", result)
	}
}
