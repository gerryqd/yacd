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
		{"Input file only", "test.log", "", false, false, ""},
		{"Make command only", "", "make clean all", false, false, ""},
		{"Stdin only", "", "", true, false, ""},
		{"No input", "", "", false, true, "no input source provided"},
		{"File and make", "test.log", "make clean all", false, true, "multiple input sources"},
		{"Whitespace only file", "   ", "", false, true, "no input source provided"},
		{"Whitespace only command", "", "   ", false, true, "no input source provided"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInputSources(tt.inputFile, tt.makeCommand, tt.stdinHasData)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Error = %v, expected to contain %s", err, tt.errorContains)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestHasStdinData(t *testing.T) {
	_ = HasStdinData()
}
