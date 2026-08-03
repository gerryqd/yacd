package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gerryqd/yacd/types"
)

// Regression tests for issues found during code review.

func TestCommandQuotesArgumentsWithSpaces(t *testing.T) {
	// Arguments containing shell-special characters must be quoted when the
	// command is reconstructed as a plain string so it can be parsed back into
	// the same arguments.
	entries := []types.MakeLogEntry{
		{
			WorkingDir: "/proj",
			Compiler:   "gcc",
			Args:       []string{"gcc", "-c", "my file.c", "-o", "my file.o"},
			SourceFile: "my file.c",
			OutputFile: "my file.o",
		},
	}

	result, _ := GenerateCompilationDatabase(entries, &types.ParseOptions{})
	if len(result) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(result))
	}
	if result[0].Command != "gcc -c 'my file.c' -o 'my file.o'" {
		t.Errorf("Command = %q, expected quoted arguments", result[0].Command)
	}

	// Re-parsing the quoted command must yield the original arguments.
	parts := types.SplitCommandLine(result[0].Command)
	if len(parts) != 5 || parts[2] != "my file.c" || parts[4] != "my file.o" {
		t.Errorf("Round-trip failed, got %v", parts)
	}
}

func TestCommandQuotesSpecialCharacters(t *testing.T) {
	tests := []struct {
		arg      string
		expected string
	}{
		{"plain.c", "plain.c"},
		{"my file.c", "'my file.c'"},
		{"a'b.c", `'a'\''b.c'`},
		{"-DFOO=a>b", "-DFOO=a>b"},
		{"-DFOO='x'", `'-DFOO='\''x'\'''`},
		{"", "''"},
	}

	for _, tt := range tests {
		if got := quoteShellArg(tt.arg); got != tt.expected {
			t.Errorf("quoteShellArg(%q) = %q, expected %q", tt.arg, got, tt.expected)
		}
	}
}

func TestDeduplicateEntriesNoKeyCollision(t *testing.T) {
	// Keys are built with a NUL separator, so paths containing '|' must not
	// collide with different (WorkingDir, SourceFile) pairs.
	entries := []types.MakeLogEntry{
		{WorkingDir: "/a|b", Compiler: "gcc", Args: []string{"gcc", "-c", "c.c"}, SourceFile: "c.c"},
		{WorkingDir: "/a", Compiler: "gcc", Args: []string{"gcc", "-c", "b|c.c"}, SourceFile: "b|c.c"},
	}

	result := deduplicateEntries(entries)
	if len(result) != 2 {
		t.Errorf("Expected 2 entries, got %d (key collision)", len(result))
	}
}

func TestAddSysrootToArgsWithSplitInclude(t *testing.T) {
	cache := NewSysrootCache()
	cache.cache["test-gcc"] = sysrootResult{sysroot: "/test/sysroot"}

	// Include already present as a "-I <path>" pair must not be added again.
	args := []string{"test-gcc", "-I", "/test/sysroot/usr/include", "-c", "main.c"}
	result := addSysrootToArgs(args, "test-gcc", cache)
	if len(result) != 5 {
		t.Errorf("Expected 5 args (no duplicate include), got %d: %v", len(result), result)
	}
	for _, arg := range result {
		if arg == "-I/test/sysroot/usr/include" {
			t.Errorf("Duplicate joined include found: %v", result)
		}
	}
}

func TestWriteCompilationDatabaseOverwritesExisting(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "compile_commands.json")

	first := []types.CompilationEntry{
		{Directory: "/proj", Command: "gcc -c main.c", File: "/proj/main.c"},
	}
	if err := WriteCompilationDatabase(first, tmpFile); err != nil {
		t.Fatalf("First write failed: %v", err)
	}

	second := []types.CompilationEntry{
		{Directory: "/proj", Command: "gcc -c main.c", File: "/proj/main.c"},
		{Directory: "/proj", Command: "gcc -c util.c", File: "/proj/util.c"},
	}
	if err := WriteCompilationDatabase(second, tmpFile); err != nil {
		t.Fatalf("Overwrite failed: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}
	var result []types.CompilationEntry
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 entries after overwrite, got %d", len(result))
	}

	// No temp files should remain behind.
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(tmpFile), filepath.Base(tmpFile)+".tmp*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Errorf("Temporary files left behind: %v", matches)
	}
}

func TestAddSysrootIncludePathPreservesQuotedArgs(t *testing.T) {
	cache := NewSysrootCache()
	cache.cache["test-gcc"] = sysrootResult{sysroot: "/test/sysroot"}

	result := addSysrootIncludePath(`test-gcc -c "my file.c"`, "test-gcc", cache)
	if !strings.Contains(result, `'my file.c'`) {
		t.Errorf("Quoted argument not preserved in command: %s", result)
	}
	parts := types.SplitCommandLine(result)
	if len(parts) != 4 || parts[3] != "my file.c" {
		t.Errorf("Round-trip failed, got %v", parts)
	}
}
