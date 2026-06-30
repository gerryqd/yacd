package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gerryqd/yacd/generator"
	"github.com/gerryqd/yacd/parser"
	"github.com/gerryqd/yacd/types"
)

var noColor = os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"

// ExecuteGeneration executes the generation process with the given options and reader
func ExecuteGeneration(options *types.ParseOptions, reader io.Reader) error {
	entries, err := parser.ParseMakeLog(reader, *options)
	if err != nil {
		return fmt.Errorf("failed to parse make log: %w", err)
	}

	compilationDB, missingFiles := generator.GenerateCompilationDatabase(entries, options)

	if err := generator.WriteCompilationDatabase(compilationDB, options.OutputFile); err != nil {
		return fmt.Errorf("failed to write file %s: %w", options.OutputFile, err)
	}

	fmt.Println(strings.Repeat("-", 50))
	if len(missingFiles) > 0 {
		if noColor {
			fmt.Printf("Warning: %d entries have non-existent source files\n", len(missingFiles))
		} else {
			fmt.Printf("\033[33mWarning: %d entries have non-existent source files\033[0m\n", len(missingFiles))
		}
	}
	if noColor {
		fmt.Printf("Successfully generated %s with %d entries\n", options.OutputFile, len(compilationDB))
	} else {
		fmt.Printf("\033[32mSuccessfully generated %s with %d entries\033[0m\n", options.OutputFile, len(compilationDB))
	}
	fmt.Println(strings.Repeat("-", 50))
	return nil
}

// PrepareReader prepares the input reader based on options
func PrepareReader(options *types.ParseOptions, stdinHasData bool) (io.Reader, func(), error) {
	var reader io.Reader
	var cleanup func()

	if options.MakeCommand != "" {
		if options.Verbose {
			fmt.Printf("Executing make command: %s\n", options.MakeCommand)
		}

		cmd, err := ExecuteMakeCommand(options.MakeCommand)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to execute command %s: %w", options.MakeCommand, err)
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to execute command %s: %w", options.MakeCommand, err)
		}

		if err := cmd.Start(); err != nil {
			return nil, nil, fmt.Errorf("failed to execute command %s: %w", options.MakeCommand, err)
		}

		reader = stdout
		cleanup = func() {
			// A non-zero exit status is common for `make -Bnkw` dry runs, so it
			// is reported as a verbose warning rather than treated as fatal.
			if waitErr := cmd.Wait(); waitErr != nil && options.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: make command exited with error: %v\n", waitErr)
			}
		}
	} else if stdinHasData {
		if options.Verbose {
			fmt.Printf("Reading from stdin\n")
		}
		reader = os.Stdin
		cleanup = func() {}
	} else {
		if _, err := os.Stat(options.InputFile); os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("file does not exist: %s", options.InputFile)
		}

		file, err := os.Open(options.InputFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open file %s: %w", options.InputFile, err)
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
	useRelativePaths, verbose, addSysroot, useArguments, dedup bool) (types.ParseOptions, error) {

	if useRelativePaths && baseDir == "" {
		baseDir = filepath.Dir(outputFile)
		if baseDir == "." {
			var err error
			baseDir, err = filepath.Abs(".")
			if err != nil {
				return types.ParseOptions{}, fmt.Errorf("failed to get current working directory: %w", err)
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
		AddSysroot:       addSysroot,
		UseArguments:     useArguments,
		Deduplicate:      dedup,
	}

	return options, nil
}
