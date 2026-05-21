package cmd

import (
	"fmt"
	"os/exec"
	"strings"
)

// ExecuteMakeCommand executes a make command with -Bnkw flags
func ExecuteMakeCommand(makeCmd string) (*exec.Cmd, error) {
	// Split the command into parts
	parts := strings.Fields(makeCmd)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty make command")
	}

	// Ensure the command starts with "make"
	if parts[0] != "make" {
		return nil, fmt.Errorf("make command must start with 'make'")
	}

	// Add -Bnkw flags at the beginning (after "make")
	args := append([]string{"make", "-Bnkw"}, parts[1:]...)

	// Create command
	cmd := exec.Command(args[0], args[1:]...)
	return cmd, nil
}
