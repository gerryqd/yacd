package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gerryqd/yacd/generator"
	"github.com/gerryqd/yacd/parser"
	"github.com/gerryqd/yacd/types"
	"github.com/gerryqd/yacd/utils/errorutil"
	"github.com/gerryqd/yacd/utils/pathutil"
)

// ExecuteGeneration executes the generation process with the given options and reader
func ExecuteGeneration(options *types.ParseOptions, reader io.Reader) error {
	// Parse make log
	entries, err := parser.ParseMakeLog(reader, options.Verbose)
	if err != nil {
		return errorutil.WrapParseError(err, "failed to parse make log")
	}

	// Generate compilation database
	compilationDB, warningCount := generator.GenerateCompilationDatabase(entries, options)

	// Write to file
	if err := generator.WriteCompilationDatabase(compilationDB, options.OutputFile); err != nil {
		return errorutil.WrapFileError(err, "write compilation database to", options.OutputFile)
	}

	// Print summary with improved formatting
	fmt.Println(strings.Repeat("-", 50))
	if warningCount > 0 {
		fmt.Printf("\033[33mWarning: %d entries have non-existent source files\033[0m\n", warningCount)
	}
	fmt.Printf("\033[32mSuccessfully generated %s with %d entries\033[0m\n", options.OutputFile, len(compilationDB))
	fmt.Println(strings.Repeat("-", 50))
	return nil
}

// PrepareReader prepares the input reader based on options
func PrepareReader(options types.ParseOptions, stdinHasData bool) (io.Reader, func(), error) {
	var reader io.Reader
	var cleanup func()

	// Handle make command execution
	if options.MakeCommand != "" {
		if options.Verbose {
			fmt.Printf("Executing make command: %s\n", options.MakeCommand)
		}

		cmd, err := ExecuteMakeCommand(options.MakeCommand)
		if err != nil {
			return nil, nil, errorutil.WrapExecutionError(err, options.MakeCommand)
		}

		// Execute command and get stdout reader
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, nil, errorutil.WrapExecutionError(err, options.MakeCommand)
		}

		// Start command
		if err := cmd.Start(); err != nil {
			return nil, nil, errorutil.WrapExecutionError(err, options.MakeCommand)
		}

		reader = stdout
		cleanup = func() {
			cmd.Wait() // Wait for command to finish
		}
	} else if stdinHasData {
		// Handle stdin input
		if options.Verbose {
			fmt.Printf("Reading from stdin\n")
		}
		reader = os.Stdin
		cleanup = func() {} // No cleanup needed for stdin
	} else {
		// Handle input file
		if _, err := os.Stat(options.InputFile); os.IsNotExist(err) {
			return nil, nil, errorutil.CreateFileNotExistError(options.InputFile)
		}

		file, err := os.Open(options.InputFile)
		if err != nil {
			return nil, nil, errorutil.WrapFileError(err, "open", options.InputFile)
		}

		reader = file
		cleanup = func() {
			file.Close()
		}
	}

	return reader, cleanup, nil
}

// PrepareOptions prepares and validates parse options
func PrepareOptions(inputFile, outputFile, makeCommand, baseDir string,
	useRelativePaths, verbose bool) (types.ParseOptions, error) {

	// Handle base directory
	if useRelativePaths && baseDir == "" {
		// If no base directory specified, use output file's directory
		baseDir = pathutil.GetDirectoryFromPath(outputFile)
		if baseDir == "." {
			baseDir = pathutil.GetWorkingDirectory()
			if baseDir == "" {
				return types.ParseOptions{}, errorutil.NewError("failed to get current working directory")
			}
		}
	}

	options := types.ParseOptions{
		InputFile:        inputFile,
		OutputFile:       outputFile,
		MakeCommand:      makeCommand,
		UseRelativePaths: useRelativePaths,
		BaseDir:          baseDir,
		Verbose:          verbose,
	}

	return options, nil
}
