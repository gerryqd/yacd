package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gerryqd/yacd/types"
)

func TestSysrootCacheGetSuccessFromCache(t *testing.T) {
	cache := NewSysrootCache()
	// Pre-populate the cache with a successful lookup result.
	cache.cache["test-gcc"] = sysrootResult{sysroot: "/test/sysroot"}

	// First retrieval returns the cached sysroot with no error.
	sysroot, err := cache.Get("test-gcc")
	if err != nil {
		t.Fatalf("Expected no error from cached entry, got: %v", err)
	}
	if sysroot != "/test/sysroot" {
		t.Errorf("sysroot = %q, expected /test/sysroot", sysroot)
	}
}

func TestSysrootCacheMemoizesError(t *testing.T) {
	cache := NewSysrootCache()
	// Pre-populate with a memoized error to verify it is returned consistently.
	cache.cache["bad-gcc"] = sysrootResult{err: errMemoized}

	_, err := cache.Get("bad-gcc")
	if err != errMemoized {
		t.Errorf("Expected memoized error to be returned, got: %v", err)
	}
}

var errMemoized = memoizedErr{}

type memoizedErr struct{}

func (memoizedErr) Error() string { return "memoized lookup failure" }

func TestGetSysrootIncludePathSuccess(t *testing.T) {
	cache := NewSysrootCache()
	cache.cache["test-gcc"] = sysrootResult{sysroot: "/test/sysroot"}

	includePath, err := getSysrootIncludePath("test-gcc", cache)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	expected := "-I" + filepath.Join("/test/sysroot", "usr", "include")
	if includePath != expected {
		t.Errorf("includePath = %q, expected %q", includePath, expected)
	}
}

func TestGetSysrootIncludePathEmpty(t *testing.T) {
	cache := NewSysrootCache()
	// An empty sysroot result yields an empty include path with no error.
	cache.cache["empty-gcc"] = sysrootResult{}

	includePath, err := getSysrootIncludePath("empty-gcc", cache)
	if err != nil {
		t.Fatalf("Expected no error for empty sysroot, got: %v", err)
	}
	if includePath != "" {
		t.Errorf("Expected empty include path, got: %q", includePath)
	}
}

func TestAddSysrootToArgsAlreadyPresent(t *testing.T) {
	cache := NewSysrootCache()
	cache.cache["test-gcc"] = sysrootResult{sysroot: "/test/sysroot"}

	// When the sysroot include path is already present, the args are unchanged.
	args := []string{"test-gcc", "-I/test/sysroot/usr/include", "-c", "main.c"}
	result := addSysrootToArgs(args, "test-gcc", cache)
	if len(result) != len(args) {
		t.Errorf("Expected args unchanged (%d), got %d: %v", len(args), len(result), result)
	}
}

func TestAddSysrootIncludePathAlreadyPresent(t *testing.T) {
	cache := NewSysrootCache()
	cache.cache["test-gcc"] = sysrootResult{sysroot: "/test/sysroot"}

	// Single-token "-I<path>" already present.
	command := "test-gcc -I/test/sysroot/usr/include -c main.c"
	if result := addSysrootIncludePath(command, "test-gcc", cache); result != command {
		t.Errorf("Expected unchanged command, got: %q", result)
	}

	// Space-separated "-I <path>" already present.
	command2 := "test-gcc -I /test/sysroot/usr/include -c main.c"
	if result := addSysrootIncludePath(command2, "test-gcc", cache); result != command2 {
		t.Errorf("Expected unchanged command for space-separated -I, got: %q", result)
	}
}

func TestAddSysrootIncludePathShortCommand(t *testing.T) {
	cache := NewSysrootCache()
	cache.cache["test-gcc"] = sysrootResult{sysroot: "/test/sysroot"}

	// A command with a single token is handled gracefully.
	result := addSysrootIncludePath("test-gcc", "test-gcc", cache)
	if result == "test-gcc" {
		t.Error("Expected sysroot to be appended to short command")
	}
}

func TestAddSysrootIncludePathLookupError(t *testing.T) {
	cache := NewSysrootCache()
	cache.cache["bad-gcc"] = sysrootResult{err: errMemoized}

	// When the lookup fails, the original command is returned unchanged.
	command := "bad-gcc -c main.c"
	if result := addSysrootIncludePath(command, "bad-gcc", cache); result != command {
		t.Errorf("Expected unchanged command on lookup error, got: %q", result)
	}
}

func TestAddSysrootToArgsLookupError(t *testing.T) {
	cache := NewSysrootCache()
	cache.cache["bad-gcc"] = sysrootResult{err: errMemoized}

	args := []string{"bad-gcc", "-c", "main.c"}
	if result := addSysrootToArgs(args, "bad-gcc", cache); len(result) != len(args) {
		t.Errorf("Expected unchanged args on lookup error, got: %v", result)
	}
}

func TestGenerateCompilationDatabaseNilOptions(t *testing.T) {
	// A nil options pointer must not panic.
	entries := []types.MakeLogEntry{
		{WorkingDir: "/project", Compiler: "gcc", Args: []string{"gcc", "-c", "main.c"}, SourceFile: "main.c"},
	}
	result, _ := GenerateCompilationDatabase(entries, nil)
	if len(result) != 1 {
		t.Errorf("Expected 1 entry with nil options, got %d", len(result))
	}
}

func TestWriteCompilationDatabaseUnwritableDir(t *testing.T) {
	// Writing to a path inside a non-existent directory fails at the write step.
	entries := []types.CompilationEntry{
		{Directory: "/project", Command: "gcc -c main.c", File: "/project/main.c"},
	}
	badPath := filepath.Join(t.TempDir(), "no_such_dir", "out.json")
	if err := WriteCompilationDatabase(entries, badPath); err == nil {
		t.Error("Expected write error for non-existent directory, got nil")
	}
}

func TestWriteCompilationDatabaseRenameToDirectory(t *testing.T) {
	// When the target path is an existing directory, the temp file can be written
	// but renaming it onto a directory fails, exercising the rename-error path.
	dirTarget := filepath.Join(t.TempDir(), "target_dir")
	if err := os.Mkdir(dirTarget, 0755); err != nil {
		t.Fatal(err)
	}

	entries := []types.CompilationEntry{
		{Directory: "/project", Command: "gcc -c main.c", File: "/project/main.c"},
	}
	if err := WriteCompilationDatabase(entries, dirTarget); err == nil {
		t.Error("Expected rename error when target is a directory, got nil")
	}
}

func TestCheckMissingFilesWithBaseDir(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "main.c")
	if err := os.WriteFile(existingFile, []byte("int main(){}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Relative file path resolved against BaseDir.
	db := []types.CompilationEntry{
		{Directory: "rel", File: "main.c"},
	}

	missing := checkMissingFiles(db, &types.ParseOptions{BaseDir: tmpDir})
	if len(missing) != 0 {
		t.Errorf("Expected 0 missing files resolved via BaseDir, got %d: %v", len(missing), missing)
	}
}

func TestGetRelativePathError(t *testing.T) {
	// Two paths on different volume roots cannot be made relative on Windows;
	// on POSIX an un-relatable input is hard to construct, so we just ensure the
	// function returns a non-empty fallback for a normal absolute path.
	result := getRelativePath("/project/src/main.c", "/other/base")
	if result == "" {
		t.Error("Expected non-empty fallback path")
	}
}
