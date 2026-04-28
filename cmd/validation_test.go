package cmd

import (
	"strings"
	"testing"
)

func TestValidateInputSources(t *testing.T) {
	tests := []struct {
		name          string
		inputFile     string
		makeCommand   string
		stdinHasData  bool
		expectError   bool
		errorContains string
	}{
		{
			name:         "Valid input file only",
			inputFile:    "test.log",
			makeCommand:  "",
			stdinHasData: false,
			expectError:  false,
		},
		{
			name:         "Valid make command only",
			inputFile:    "",
			makeCommand:  "make clean all",
			stdinHasData: false,
			expectError:  false,
		},
		{
			name:         "Valid stdin only",
			inputFile:    "",
			makeCommand:  "",
			stdinHasData: true,
			expectError:  false,
		},
		{
			name:          "No input source specified",
			inputFile:     "",
			makeCommand:   "",
			stdinHasData:  false,
			expectError:   true,
			errorContains: "no input source provided, please specify one of: -i/--input, -n/--dry-run, or provide input via stdin",
		},
		{
			name:          "Multiple input sources - file and make command",
			inputFile:     "test.log",
			makeCommand:   "make clean all",
			stdinHasData:  false,
			expectError:   true,
			errorContains: "multiple input sources provided, please specify only one of: -i/--input, -n/--dry-run, or stdin",
		},
		{
			name:          "Multiple input sources - file and stdin",
			inputFile:     "test.log",
			makeCommand:   "",
			stdinHasData:  true,
			expectError:   true,
			errorContains: "multiple input sources provided, please specify only one of: -i/--input, -n/--dry-run, or stdin",
		},
		{
			name:          "Multiple input sources - make command and stdin",
			inputFile:     "",
			makeCommand:   "make clean all",
			stdinHasData:  true,
			expectError:   true,
			errorContains: "multiple input sources provided, please specify only one of: -i/--input, -n/--dry-run, or stdin",
		},
		{
			name:          "All three input sources specified",
			inputFile:     "test.log",
			makeCommand:   "make clean all",
			stdinHasData:  true,
			expectError:   true,
			errorContains: "multiple input sources provided, please specify only one of: -i/--input, -n/--dry-run, or stdin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInputSources(tt.inputFile, tt.makeCommand, tt.stdinHasData)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateInputSources() expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("ValidateInputSources() error = %v, expected to contain %s", err, tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateInputSources() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestHasStdinData(t *testing.T) {
	// Note: This test just verifies the function does not panic.
	// The return value depends on the test environment (terminal vs piped input).
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("HasStdinData() panicked: %v", r)
		}
	}()

	_ = HasStdinData()
}

func TestValidateInputSourcesEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		inputFile    string
		makeCommand  string
		stdinHasData bool
		expectError  bool
	}{
		{
			name:         "Empty strings and false stdin",
			inputFile:    "",
			makeCommand:  "",
			stdinHasData: false,
			expectError:  true,
		},
		{
			name:         "Whitespace only input file",
			inputFile:    "   ",
			makeCommand:  "",
			stdinHasData: false,
			expectError:  true, // Whitespace-only is trimmed to empty
		},
		{
			name:         "Whitespace only make command",
			inputFile:    "",
			makeCommand:  "   ",
			stdinHasData: false,
			expectError:  true, // Whitespace-only is trimmed to empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInputSources(tt.inputFile, tt.makeCommand, tt.stdinHasData)

			if tt.expectError && err == nil {
				t.Errorf("ValidateInputSources() expected error, got nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("ValidateInputSources() unexpected error = %v", err)
			}
		})
	}
}
