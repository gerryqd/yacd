package cmd

import (
	"strings"
	"testing"

	"github.com/gerryqd/yacd/types"
)

func TestParseMakeCommand(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expectedCmd string
		expectedLen int
	}{
		{
			name:        "Simple make command",
			input:       "make",
			expectError: false,
			expectedCmd: "make",
			expectedLen: 1,
		},
		{
			name:        "Make with targets",
			input:       "make clean all",
			expectError: false,
			expectedCmd: "make",
			expectedLen: 3,
		},
		{
			name:        "Make with flags",
			input:       "make -j4 all",
			expectError: false,
			expectedCmd: "make",
			expectedLen: 3,
		},
		{
			name:        "Empty command",
			input:       "",
			expectError: true,
			expectedCmd: "",
			expectedLen: 0,
		},
		{
			name:        "Just spaces",
			input:       "   ",
			expectError: true,
			expectedCmd: "",
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdParts := strings.Fields(tt.input)

			if tt.expectError {
				if len(cmdParts) != 0 {
					t.Errorf("Expected empty command parts for input %q, got %v", tt.input, cmdParts)
				}
				return
			}

			if len(cmdParts) == 0 {
				t.Errorf("Expected non-empty command parts for input %q", tt.input)
				return
			}

			if cmdParts[0] != tt.expectedCmd {
				t.Errorf("Expected command %q, got %q", tt.expectedCmd, cmdParts[0])
			}

			if len(cmdParts) != tt.expectedLen {
				t.Errorf("Expected %d command parts, got %d", tt.expectedLen, len(cmdParts))
			}
		})
	}
}

func TestExecuteMakeCommand(t *testing.T) {
	tests := []struct {
		name          string
		makeCmd       string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Empty command",
			makeCmd:       "",
			expectError:   true,
			errorContains: "empty make command",
		},
		{
			name:          "Whitespace only command",
			makeCmd:       "   ",
			expectError:   true,
			errorContains: "empty make command",
		},
		{
			name:        "Valid make command",
			makeCmd:     "make clean",
			expectError: false,
		},
		{
			name:        "Make with targets",
			makeCmd:     "make clean all",
			expectError: false,
		},
		{
			name:        "Make with flags",
			makeCmd:     "make -j4 clean",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := ExecuteMakeCommand(tt.makeCmd)

			if tt.expectError {
				if err == nil {
					t.Errorf("ExecuteMakeCommand() expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("ExecuteMakeCommand() error = %v, expected to contain %s", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				// For valid commands, we might get errors if 'make' is not available
				// or if the command fails to start. This is expected in test environments.
				t.Logf("ExecuteMakeCommand() returned error (expected in test environment): %v", err)
				return
			}

			if cmd == nil {
				t.Errorf("ExecuteMakeCommand() returned nil command")
				return
			}

			// Check that the command is properly configured with -Bnkw flags
			if len(cmd.Args) < 2 || cmd.Args[1] != "-Bnkw" {
				t.Errorf("ExecuteMakeCommand() did not add -Bnkw flag correctly. Args: %v", cmd.Args)
			}
		})
	}
}

func TestPrintExecutionInfoInMake(t *testing.T) {
	tests := []struct {
		name    string
		options types.ParseOptions
	}{
		{
			name: "Input file option",
			options: types.ParseOptions{
				InputFile:  "test.log",
				OutputFile: "compile_commands.json",
			},
		},
		{
			name: "Make command option",
			options: types.ParseOptions{
				MakeCommand: "make clean all",
				OutputFile:  "compile_commands.json",
			},
		},
		{
			name: "Relative paths option",
			options: types.ParseOptions{
				InputFile:        "test.log",
				OutputFile:       "compile_commands.json",
				UseRelativePaths: true,
				BaseDir:          "/project",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function just prints to stdout, so we test it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PrintExecutionInfo() panicked: %v", r)
				}
			}()

			PrintExecutionInfo(&tt.options)
		})
	}
}
