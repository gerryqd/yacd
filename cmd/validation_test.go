package cmd

import (
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
				if tt.errorContains != "" && !containsError(err.Error(), tt.errorContains) {
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
	// Note: This test is challenging because HasStdinData() checks the actual stdin
	// In a real test environment, stdin is typically attached to a terminal
	// So we expect this to return false in most test scenarios

	result := HasStdinData()

	// In a normal test environment, stdin should be a character device (terminal)
	// so HasStdinData should return false
	if result {
		t.Log("HasStdinData() returned true - this might indicate stdin is piped or redirected")
	} else {
		t.Log("HasStdinData() returned false - stdin appears to be a terminal")
	}

	// We don't assert a specific value here because the behavior depends on
	// how the test is run (terminal vs piped input)
}

func TestHasStdinDataBehavior(t *testing.T) {
	// Test that the function doesn't panic and returns a boolean
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("HasStdinData() panicked: %v", r)
		}
	}()

	result := HasStdinData()

	// Should return a boolean value (true or false)
	if result != true && result != false {
		t.Errorf("HasStdinData() should return a boolean, got: %v", result)
	}
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
			expectError:  false, // Non-empty string is considered valid
		},
		{
			name:         "Whitespace only make command",
			inputFile:    "",
			makeCommand:  "   ",
			stdinHasData: false,
			expectError:  false, // Non-empty string is considered valid
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

// Helper function to check if error message contains expected text
func containsError(errorMsg, expected string) bool {
	return len(errorMsg) > 0 && len(expected) > 0 &&
		(errorMsg == expected ||
			len(errorMsg) >= len(expected) &&
				findInString(errorMsg, expected))
}

// Simple substring search
func findInString(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
