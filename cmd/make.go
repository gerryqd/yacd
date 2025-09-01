package cmd

import (
	"io"
	"os/exec"
	"strings"

	"github.com/gerrywa/yacd/utils/errorutil"
)

// ExecuteMakeCommand executes the make command with -Bnkw flags and returns the output as a reader
func ExecuteMakeCommand(makeCmd string) (io.Reader, error) {
	// Parse the make command
	cmdParts := strings.Fields(makeCmd)
	if len(cmdParts) == 0 {
		return nil, errorutil.CreateEmptyInputError("make command")
	}

	// Extract the make executable and arguments
	makeExe := cmdParts[0]
	makeArgs := cmdParts[1:]

	// Add -Bnkw flags if not already present
	args := EnsureMakeFlags(makeArgs)

	// Create the command
	cmd := exec.Command(makeExe, args...)

	// Set up stdout pipe
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errorutil.WrapError(err, "failed to create stdout pipe")
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, errorutil.WrapError(err, "failed to start make command")
	}

	return stdoutPipe, nil
}

// EnsureMakeFlags adds -Bnkw flags to make arguments
func EnsureMakeFlags(args []string) []string {
	// Always add -Bnkw flags at the beginning
	result := []string{"-Bnkw"}

	// Add original arguments
	result = append(result, args...)

	return result
}
