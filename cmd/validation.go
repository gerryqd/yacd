package cmd

import (
	"os"

	"github.com/gerrywa/yacd/utils/errorutil"
)

// ValidateInputSources validates that exactly one input source is specified
func ValidateInputSources(inputFile, makeCommand string, stdinHasData bool) error {
	// Check if stdin has data (for pipe input)
	if !stdinHasData {
		stdinHasData = HasStdinData()
	}

	// Validate parameters - either input file, make command, or stdin must be specified
	if inputFile == "" && makeCommand == "" && !stdinHasData {
		return errorutil.CreateInvalidArgumentError("input",
			"either input file (-i), make command (-n), or stdin pipe must be specified")
	}

	// Ensure mutual exclusivity
	inputSources := 0
	if inputFile != "" {
		inputSources++
	}
	if makeCommand != "" {
		inputSources++
	}
	if stdinHasData {
		inputSources++
	}
	if inputSources > 1 {
		return errorutil.NewError("only one input source is allowed: input file (-i), make command (-n), or stdin pipe")
	}

	return nil
}

// HasStdinData checks if stdin has data available (for pipe input)
func HasStdinData() bool {
	// Get file info for stdin
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	// Check if stdin is a pipe or has data
	// On Unix: ModeCharDevice indicates terminal input
	// On Windows: we check if it's not a character device
	return (stat.Mode() & os.ModeCharDevice) == 0
}
