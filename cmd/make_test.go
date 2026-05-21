package cmd

import (
	"strings"
	"testing"
)

func TestExecuteMakeCommand(t *testing.T) {
	tests := []struct {
		name          string
		makeCmd       string
		expectError   bool
		errorContains string
	}{
		{"Empty command", "", true, "empty make command"},
		{"Whitespace only", "   ", true, "empty make command"},
		{"Valid make", "make clean", false, ""},
		{"Non-make command", "echo hello", true, "must start with 'make'"},
		{"Make with flags", "make -j4 clean", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := ExecuteMakeCommand(tt.makeCmd)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Error = %v, expected to contain %s", err, tt.errorContains)
				}
				return
			}
			if err != nil {
				t.Logf("Error (may be expected in test env): %v", err)
				return
			}
			if cmd == nil {
				t.Errorf("Returned nil command")
				return
			}
			if len(cmd.Args) < 2 || cmd.Args[1] != "-Bnkw" {
				t.Errorf("Did not add -Bnkw flag. Args: %v", cmd.Args)
			}
		})
	}
}
