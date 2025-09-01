package cmd

import (
	"os"

	"github.com/gerryqd/yacd/utils/errorutil"
)

// ValidateInputSources validates that exactly one input source is provided
func ValidateInputSources(inputFile, makeCommand string, stdinHasData bool) error {
	// Count how many input sources are provided
	inputCount := 0
	if inputFile != "" {
		inputCount++
	}
	if makeCommand != "" {
		inputCount++
	}
	if stdinHasData {
		inputCount++
	}

	// Must have exactly one input source
	if inputCount == 0 {
		return errorutil.NewError("no input source provided, please specify one of: -i/--input, -n/--dry-run, or provide input via stdin")
	}
	if inputCount > 1 {
		return errorutil.NewError("multiple input sources provided, please specify only one of: -i/--input, -n/--dry-run, or stdin")
	}

	return nil
}

// HasStdinData checks if stdin has data available
func HasStdinData() bool {
	// This is a simplified check - in practice, you might want to use a more robust method
	// such as checking if stdin is a pipe or has data available
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}
