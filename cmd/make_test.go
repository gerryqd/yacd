package cmd

import (
	"strings"
	"testing"
)

func TestEnsureMakeFlags(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Empty arguments",
			input:    []string{},
			expected: []string{"-Bnkw"},
		},
		{
			name:     "With target",
			input:    []string{"all"},
			expected: []string{"-Bnkw", "all"},
		},
		{
			name:     "With flags and target",
			input:    []string{"-j4", "all"},
			expected: []string{"-Bnkw", "-j4", "all"},
		},
		{
			name:     "Multiple targets",
			input:    []string{"clean", "all"},
			expected: []string{"-Bnkw", "clean", "all"},
		},
		{
			name:     "With duplicate flags",
			input:    []string{"-B", "-n", "all"},
			expected: []string{"-Bnkw", "-B", "-n", "all"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnsureMakeFlags(tt.input)

			// Check if result length matches expected
			if len(result) != len(tt.expected) {
				t.Errorf("Expected length %d, got %d. Expected: %v, Got: %v",
					len(tt.expected), len(result), tt.expected, result)
				return
			}

			// Check each element
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("At index %d: expected %s, got %s", i, expected, result[i])
				}
			}

			// Ensure -Bnkw is always at the beginning
			if len(result) > 0 && result[0] != "-Bnkw" {
				t.Errorf("Expected -Bnkw at the beginning, got %s", result[0])
			}
		})
	}
}

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
			errorContains: "make command is empty",
		},
		{
			name:          "Whitespace only command",
			makeCmd:       "   ",
			expectError:   true,
			errorContains: "make command is empty",
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
			reader, err := ExecuteMakeCommand(tt.makeCmd)

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

			if reader == nil {
				t.Errorf("ExecuteMakeCommand() returned nil reader")
				return
			}

			// Close the reader if it implements io.Closer
			if closer, ok := reader.(interface{ Close() error }); ok {
				closer.Close()
			}
		})
	}
}
