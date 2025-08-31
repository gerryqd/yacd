package pathutil

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestIsAbsolutePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Unix absolute path",
			path:     "/home/user/project",
			expected: true,
		},
		{
			name:     "Windows absolute path",
			path:     "C:\\Users\\user\\project",
			expected: true,
		},
		{
			name:     "Relative path",
			path:     "relative/path",
			expected: false,
		},
		{
			name:     "Current directory",
			path:     ".",
			expected: false,
		},
		{
			name:     "Parent directory",
			path:     "../parent",
			expected: false,
		},
		{
			name:     "Backslash root",
			path:     "\\root",
			expected: true,
		},
		{
			name:     "Empty path",
			path:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAbsolutePath(tt.path)
			if result != tt.expected {
				t.Errorf("IsAbsolutePath(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestResolveRelativePath(t *testing.T) {
	tests := []struct {
		name         string
		baseDir      string
		relativePath string
		expected     string
	}{
		{
			name:         "Relative path resolution",
			baseDir:      "/project",
			relativePath: "src/main.c",
			expected:     "/project/src/main.c",
		},
		{
			name:         "Absolute path unchanged",
			baseDir:      "/project",
			relativePath: "/absolute/path.c",
			expected:     "/absolute/path.c",
		},
		{
			name:         "Current directory",
			baseDir:      "/project",
			relativePath: ".",
			expected:     "/project",
		},
		{
			name:         "Empty relative path",
			baseDir:      "/project",
			relativePath: "",
			expected:     "/project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveRelativePath(tt.baseDir, tt.relativePath)
			// Normalize paths for comparison
			expectedNorm := filepath.Clean(tt.expected)
			resultNorm := filepath.Clean(result)

			if ToSlashPath(resultNorm) != ToSlashPath(expectedNorm) {
				t.Errorf("ResolveRelativePath(%q, %q) = %q, expected %q",
					tt.baseDir, tt.relativePath, result, tt.expected)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Remove double slashes",
			path:     "/path//to/file",
			expected: "/path/to/file",
		},
		{
			name:     "Remove trailing slash",
			path:     "/path/to/dir/",
			expected: "/path/to/dir",
		},
		{
			name:     "Resolve dots",
			path:     "/path/./to/../file",
			expected: "/path/file",
		},
		{
			name:     "Current directory",
			path:     ".",
			expected: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePath(tt.path)
			expectedNorm := filepath.Clean(tt.expected)

			if ToSlashPath(result) != ToSlashPath(expectedNorm) {
				t.Errorf("NormalizePath(%q) = %q, expected %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsSourceFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "C source file",
			filename: "main.c",
			expected: true,
		},
		{
			name:     "C++ source file",
			filename: "main.cpp",
			expected: true,
		},
		{
			name:     "C++ alternative extension",
			filename: "main.cc",
			expected: true,
		},
		{
			name:     "Assembly file",
			filename: "startup.s",
			expected: true,
		},
		{
			name:     "Assembly file uppercase",
			filename: "startup.S",
			expected: true,
		},
		{
			name:     "Header file",
			filename: "main.h",
			expected: false,
		},
		{
			name:     "Object file",
			filename: "main.o",
			expected: false,
		},
		{
			name:     "No extension",
			filename: "Makefile",
			expected: false,
		},
		{
			name:     "Case insensitive",
			filename: "main.C",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSourceFile(tt.filename)
			if result != tt.expected {
				t.Errorf("IsSourceFile(%q) = %v, expected %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestJoinPaths(t *testing.T) {
	tests := []struct {
		name     string
		elements []string
		expected string
	}{
		{
			name:     "Simple join",
			elements: []string{"path", "to", "file"},
			expected: "path/to/file",
		},
		{
			name:     "With separators",
			elements: []string{"/base", "dir/", "/file"},
			expected: "/base/dir/file",
		},
		{
			name:     "Single element",
			elements: []string{"file"},
			expected: "file",
		},
		{
			name:     "Empty elements",
			elements: []string{"", "path", "", "file"},
			expected: "path/file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JoinPaths(tt.elements...)
			expectedNorm := filepath.Clean(tt.expected)

			if ToSlashPath(result) != ToSlashPath(expectedNorm) {
				t.Errorf("JoinPaths(%v) = %q, expected %q", tt.elements, result, tt.expected)
			}
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		expectedDir  string
		expectedFile string
	}{
		{
			name:         "Full path",
			path:         "/path/to/file.txt",
			expectedDir:  "/path/to/",
			expectedFile: "file.txt",
		},
		{
			name:         "Relative path",
			path:         "dir/file.txt",
			expectedDir:  "dir/",
			expectedFile: "file.txt",
		},
		{
			name:         "Just filename",
			path:         "file.txt",
			expectedDir:  "",
			expectedFile: "file.txt",
		},
		{
			name:         "Root path",
			path:         "/file.txt",
			expectedDir:  "/",
			expectedFile: "file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, file := SplitPath(tt.path)
			if dir != tt.expectedDir || file != tt.expectedFile {
				t.Errorf("SplitPath(%q) = (%q, %q), expected (%q, %q)",
					tt.path, dir, file, tt.expectedDir, tt.expectedFile)
			}
		})
	}
}

func TestEnsureDirectorySeparator(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Path without separator",
			path:     "/path/to/dir",
			expected: "/path/to/dir" + string(filepath.Separator),
		},
		{
			name:     "Path with separator",
			path:     "/path/to/dir/",
			expected: "/path/to/dir/",
		},
		{
			name:     "Empty path",
			path:     "",
			expected: "",
		},
		{
			name:     "Root path",
			path:     "/",
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnsureDirectorySeparator(tt.path)

			// On Windows, we accept both separators
			if runtime.GOOS == "windows" && strings.HasSuffix(tt.path, "/") {
				tt.expected = tt.path
			}

			if result != tt.expected {
				t.Errorf("EnsureDirectorySeparator(%q) = %q, expected %q",
					tt.path, result, tt.expected)
			}
		})
	}
}
