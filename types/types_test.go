package types

import (
	"testing"
)

func TestCompilationEntry(t *testing.T) {
	// Test basic CompilationEntry creation
	entry := CompilationEntry{
		Directory: "/test/dir",
		Arguments: []string{"gcc", "-c", "main.c"},
		File:      "main.c",
		Output:    "main.o",
	}

	if entry.Directory != "/test/dir" {
		t.Errorf("Directory = %s, expected /test/dir", entry.Directory)
	}

	if len(entry.Arguments) != 3 {
		t.Errorf("Arguments length = %d, expected 3", len(entry.Arguments))
	}

	if entry.File != "main.c" {
		t.Errorf("File = %s, expected main.c", entry.File)
	}

	if entry.Output != "main.o" {
		t.Errorf("Output = %s, expected main.o", entry.Output)
	}
}

func TestMakeLogEntry(t *testing.T) {
	// Test basic MakeLogEntry creation
	entry := MakeLogEntry{
		WorkingDir: "/project",
		Compiler:   "gcc",
		Args:       []string{"gcc", "-c", "test.c"},
		SourceFile: "test.c",
		OutputFile: "test.o",
	}

	if entry.WorkingDir != "/project" {
		t.Errorf("WorkingDir = %s, expected /project", entry.WorkingDir)
	}

	if entry.Compiler != "gcc" {
		t.Errorf("Compiler = %s, expected gcc", entry.Compiler)
	}

	if entry.SourceFile != "test.c" {
		t.Errorf("SourceFile = %s, expected test.c", entry.SourceFile)
	}

	if entry.OutputFile != "test.o" {
		t.Errorf("OutputFile = %s, expected test.o", entry.OutputFile)
	}
}

func TestParseOptions(t *testing.T) {
	// Test basic ParseOptions creation
	options := ParseOptions{
		InputFile:        "test.log",
		OutputFile:       "compile_commands.json",
		UseRelativePaths: true,
		BaseDir:          "/base",
		Verbose:          true,
	}

	if options.InputFile != "test.log" {
		t.Errorf("InputFile = %s, expected test.log", options.InputFile)
	}

	if options.OutputFile != "compile_commands.json" {
		t.Errorf("OutputFile = %s, expected compile_commands.json", options.OutputFile)
	}

	if !options.UseRelativePaths {
		t.Errorf("UseRelativePaths = %v, expected true", options.UseRelativePaths)
	}

	if options.BaseDir != "/base" {
		t.Errorf("BaseDir = %s, expected /base", options.BaseDir)
	}

	if !options.Verbose {
		t.Errorf("Verbose = %v, expected true", options.Verbose)
	}
}

func TestCompilationDatabase(t *testing.T) {
	// Test CompilationDatabase creation and manipulation
	var db CompilationDatabase

	// Test empty database
	if len(db) != 0 {
		t.Errorf("Empty database length = %d, expected 0", len(db))
	}

	// Test adding entries
	entry1 := CompilationEntry{
		Directory: "/test1",
		File:      "file1.c",
	}
	entry2 := CompilationEntry{
		Directory: "/test2",
		File:      "file2.c",
	}

	db = append(db, entry1, entry2)

	if len(db) != 2 {
		t.Errorf("Database length = %d, expected 2", len(db))
	}

	if db[0].File != "file1.c" {
		t.Errorf("First entry file = %s, expected file1.c", db[0].File)
	}

	if db[1].File != "file2.c" {
		t.Errorf("Second entry file = %s, expected file2.c", db[1].File)
	}
}
