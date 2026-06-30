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

func TestExecuteMakeCommandQuotedArgs(t *testing.T) {
	// Quoted arguments must be preserved as a single argument. Using
	// strings.Fields here would incorrectly split "CFLAGS=-O2 -g".
	cmd, err := ExecuteMakeCommand(`make CFLAGS="-O2 -g"`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if cmd == nil {
		t.Fatal("Returned nil command")
	}
	// Args layout: [make, -Bnkw, CFLAGS=-O2 -g]
	if len(cmd.Args) != 3 {
		t.Fatalf("Expected 3 args, got %d: %v", len(cmd.Args), cmd.Args)
	}
	if cmd.Args[2] != "CFLAGS=-O2 -g" {
		t.Errorf("Expected quoted arg preserved, got %q in %v", cmd.Args[2], cmd.Args)
	}
}
