package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/gerryqd/yacd/types"
	"github.com/gerryqd/yacd/utils/errorutil"
)

// ExecuteMakeCommand executes a make command with -Bnkw flags
func ExecuteMakeCommand(makeCmd string) (*exec.Cmd, error) {
	// Split the command into parts
	parts := strings.Fields(makeCmd)
	if len(parts) == 0 {
		return nil, errorutil.NewError("empty make command")
	}

	// Ensure the command starts with "make"
	if parts[0] != "make" {
		return nil, errorutil.NewError("make command must start with 'make'")
	}

	// Add -Bnkw flags at the beginning (after "make")
	args := append([]string{"make", "-Bnkw"}, parts[1:]...)

	// Create command
	cmd := exec.Command(args[0], args[1:]...)
	return cmd, nil
}

// PrintExecutionInfo prints execution information based on the input source
func PrintExecutionInfo(options *types.ParseOptions) {
	fmt.Println("yacd - Yet Another CompileDB")

	// Print input source information
	if options.InputFile != "" {
		fmt.Printf("Input file: %s\n", options.InputFile)
	} else if options.MakeCommand != "" {
		fmt.Printf("Make command: %s\n", options.MakeCommand)
	} else {
		fmt.Println("Input source: stdin")
	}

	fmt.Printf("Output file: %s\n", options.OutputFile)

	if options.UseRelativePaths {
		fmt.Printf("Using relative paths, base directory: %s\n", options.BaseDir)
	}

	fmt.Println("Starting to parse...")
}
