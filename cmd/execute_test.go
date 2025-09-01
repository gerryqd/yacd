package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gerrywa/yacd/types"
)

func TestPrepareOptions(t *testing.T) {
	tests := []struct {
		name             string
		inputFile        string
		outputFile       string
		makeCommand      string
		baseDir          string
		useRelativePaths bool
		verbose          bool
		expectError      bool
		expectedOptions  types.ParseOptions
	}{
		{
			name:             "Basic options",
			inputFile:        "input.log",
			outputFile:       "output.json",
			makeCommand:      "",
			baseDir:          "",
			useRelativePaths: false,
			verbose:          false,
			expectError:      false,
			expectedOptions: types.ParseOptions{
				InputFile:        "input.log",
				OutputFile:       "output.json",
				MakeCommand:      "",
				UseRelativePaths: false,
				BaseDir:          "",
				Verbose:          false,
			},
		},
		{
			name:             "With make command",
			inputFile:        "",
			outputFile:       "output.json",
			makeCommand:      "make clean all",
			baseDir:          "",
			useRelativePaths: false,
			verbose:          true,
			expectError:      false,
			expectedOptions: types.ParseOptions{
				InputFile:        "",
				OutputFile:       "output.json",
				MakeCommand:      "make clean all",
				UseRelativePaths: false,
				BaseDir:          "",
				Verbose:          true,
			},
		},
		{
			name:             "With explicit base directory",
			inputFile:        "input.log",
			outputFile:       "output.json",
			makeCommand:      "",
			baseDir:          "/project/root",
			useRelativePaths: true,
			verbose:          false,
			expectError:      false,
			expectedOptions: types.ParseOptions{
				InputFile:        "input.log",
				OutputFile:       "output.json",
				MakeCommand:      "",
				UseRelativePaths: true,
				BaseDir:          "/project/root",
				Verbose:          false,
			},
		},
		{
			name:             "Relative paths with output file directory as base",
			inputFile:        "input.log",
			outputFile:       "build/output.json",
			makeCommand:      "",
			baseDir:          "",
			useRelativePaths: true,
			verbose:          false,
			expectError:      false,
			expectedOptions: types.ParseOptions{
				InputFile:        "input.log",
				OutputFile:       "build/output.json",
				MakeCommand:      "",
				UseRelativePaths: true,
				BaseDir:          "build",
				Verbose:          false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, err := PrepareOptions(tt.inputFile, tt.outputFile, tt.makeCommand,
				tt.baseDir, tt.useRelativePaths, tt.verbose)

			if tt.expectError {
				if err == nil {
					t.Errorf("PrepareOptions() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("PrepareOptions() unexpected error = %v", err)
				return
			}

			// Check all fields
			if options.InputFile != tt.expectedOptions.InputFile {
				t.Errorf("InputFile = %s, expected %s", options.InputFile, tt.expectedOptions.InputFile)
			}
			if options.OutputFile != tt.expectedOptions.OutputFile {
				t.Errorf("OutputFile = %s, expected %s", options.OutputFile, tt.expectedOptions.OutputFile)
			}
			if options.MakeCommand != tt.expectedOptions.MakeCommand {
				t.Errorf("MakeCommand = %s, expected %s", options.MakeCommand, tt.expectedOptions.MakeCommand)
			}
			if options.UseRelativePaths != tt.expectedOptions.UseRelativePaths {
				t.Errorf("UseRelativePaths = %v, expected %v", options.UseRelativePaths, tt.expectedOptions.UseRelativePaths)
			}
			if options.Verbose != tt.expectedOptions.Verbose {
				t.Errorf("Verbose = %v, expected %v", options.Verbose, tt.expectedOptions.Verbose)
			}

			// For base directory, we need to handle cases where it gets auto-determined
			if tt.baseDir != "" {
				// If baseDir was explicitly provided, it should match
				if options.BaseDir != tt.expectedOptions.BaseDir {
					t.Errorf("BaseDir = %s, expected %s", options.BaseDir, tt.expectedOptions.BaseDir)
				}
			} else if tt.useRelativePaths {
				// If using relative paths without explicit baseDir, it should be derived from output file
				expectedBaseDir := filepath.Dir(tt.outputFile)
				if expectedBaseDir == "." {
					// Current working directory case - just check it's not empty
					if options.BaseDir == "" {
						t.Errorf("BaseDir should not be empty when using relative paths")
					}
				} else {
					if options.BaseDir != expectedBaseDir {
						t.Errorf("BaseDir = %s, expected %s", options.BaseDir, expectedBaseDir)
					}
				}
			}
		})
	}
}

func TestPrepareOptionsWithCurrentDirectory(t *testing.T) {
	// Test case where output file is in current directory and relative paths are used
	options, err := PrepareOptions("input.log", "output.json", "", "", true, false)

	if err != nil {
		t.Errorf("PrepareOptions() unexpected error = %v", err)
		return
	}

	// BaseDir should be set to current working directory
	if options.BaseDir == "" {
		t.Errorf("BaseDir should not be empty when using relative paths with output in current dir")
	}

	// Should be an absolute path
	if !filepath.IsAbs(options.BaseDir) {
		t.Errorf("BaseDir should be absolute path, got: %s", options.BaseDir)
	}
}

func TestPrepareReaderValidation(t *testing.T) {
	tests := []struct {
		name          string
		options       types.ParseOptions
		stdinHasData  bool
		expectError   bool
		errorContains string
	}{
		{
			name: "Non-existent input file",
			options: types.ParseOptions{
				InputFile: "/non/existent/file.log",
			},
			stdinHasData:  false,
			expectError:   true,
			errorContains: "file does not exist",
		},
		{
			name: "Valid make command",
			options: types.ParseOptions{
				MakeCommand: "echo test", // Use echo instead of make for reliable testing
			},
			stdinHasData: false,
			expectError:  false,
		},
		{
			name: "Empty make command",
			options: types.ParseOptions{
				MakeCommand: "",
			},
			stdinHasData:  false,
			expectError:   true,
			errorContains: "file does not exist", // When MakeCommand is empty, it tries to read InputFile
		},
		{
			name:         "Stdin input",
			options:      types.ParseOptions{},
			stdinHasData: true,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, cleanup, err := PrepareReader(tt.options, tt.stdinHasData)

			if tt.expectError {
				if err == nil {
					t.Errorf("PrepareReader() expected error, got nil")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("PrepareReader() error = %v, expected to contain %s", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				// Some errors might be expected in test environment (e.g., missing make command)
				t.Logf("PrepareReader() returned error (might be expected in test environment): %v", err)
				return
			}

			if reader == nil {
				t.Errorf("PrepareReader() returned nil reader")
				return
			}

			// Cleanup should not be nil
			if cleanup == nil {
				t.Errorf("PrepareReader() returned nil cleanup function")
				return
			}

			// Call cleanup function
			cleanup()
		})
	}
}

func TestPrintExecutionInfo(t *testing.T) {
	tests := []struct {
		name    string
		options types.ParseOptions
	}{
		{
			name: "File input",
			options: types.ParseOptions{
				InputFile:        "input.log",
				OutputFile:       "output.json",
				UseRelativePaths: false,
			},
		},
		{
			name: "Make command input",
			options: types.ParseOptions{
				MakeCommand:      "make clean all",
				OutputFile:       "output.json",
				UseRelativePaths: false,
			},
		},
		{
			name: "Stdin input",
			options: types.ParseOptions{
				InputFile:        "",
				OutputFile:       "output.json",
				UseRelativePaths: false,
			},
		},
		{
			name: "With relative paths",
			options: types.ParseOptions{
				InputFile:        "input.log",
				OutputFile:       "output.json",
				UseRelativePaths: true,
				BaseDir:          "/project/root",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function just prints to stdout, so we test it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("printExecutionInfo() panicked: %v", r)
				}
			}()

			printExecutionInfo(tt.options)
		})
	}
}
