package pathutil

import (
	"path/filepath"
	"runtime"
	"strings"
)

// IsAbsolutePath checks if a path is absolute (cross-platform)
// This function handles both Unix-style and Windows-style absolute paths
func IsAbsolutePath(path string) bool {
	// Standard filepath.IsAbs check
	if filepath.IsAbs(path) {
		return true
	}

	// Check for Unix-style absolute paths (starting with '/')
	// This is important for cross-platform compatibility
	if len(path) > 0 && path[0] == '/' {
		return true
	}

	// Check for Windows-style paths starting with backslash
	if len(path) > 0 && path[0] == '\\' {
		return true
	}

	return false
}

// ResolveRelativePath resolves a relative path against a base directory
func ResolveRelativePath(baseDir, relativePath string) string {
	if IsAbsolutePath(relativePath) {
		return relativePath
	}
	return filepath.Join(baseDir, relativePath)
}

// ToRelativePath converts an absolute path to relative path based on baseDir
func ToRelativePath(basePath, targetPath string) (string, error) {
	return filepath.Rel(basePath, targetPath)
}

// NormalizePath cleans and normalizes a path for consistent usage
func NormalizePath(path string) string {
	return filepath.Clean(path)
}

// ToSlashPath converts path separators to forward slashes for consistent comparison
func ToSlashPath(path string) string {
	return filepath.ToSlash(path)
}

// JoinPaths joins path elements and returns a clean path
func JoinPaths(elements ...string) string {
	return filepath.Clean(filepath.Join(elements...))
}

// GetDirectoryFromPath returns the directory part of a file path
func GetDirectoryFromPath(filePath string) string {
	return filepath.Dir(filePath)
}

// SplitPath splits a path into directory and filename components
func SplitPath(path string) (dir, file string) {
	return filepath.Split(path)
}

// HasExtension checks if a path has the given file extension
func HasExtension(path, ext string) bool {
	return strings.HasSuffix(strings.ToLower(path), strings.ToLower(ext))
}

// IsSourceFile checks if a file is a source code file based on extension
func IsSourceFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	sourceExts := []string{".c", ".cpp", ".cc", ".cxx", ".c++", ".s", ".S", ".asm"}

	for _, validExt := range sourceExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

// GetWorkingDirectory returns current working directory or empty string on error
func GetWorkingDirectory() string {
	if wd, err := filepath.Abs("."); err == nil {
		return wd
	}
	return ""
}

// EnsureDirectorySeparator ensures path ends with directory separator
func EnsureDirectorySeparator(path string) string {
	if path == "" {
		return path
	}

	sep := string(filepath.Separator)
	if runtime.GOOS == "windows" {
		// On Windows, also accept forward slash
		if !strings.HasSuffix(path, sep) && !strings.HasSuffix(path, "/") {
			return path + sep
		}
	} else {
		if !strings.HasSuffix(path, sep) {
			return path + sep
		}
	}
	return path
}
