package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/gerrywa/yacd/generator"
	"github.com/gerrywa/yacd/parser"
	"github.com/gerrywa/yacd/types"
	"github.com/gerrywa/yacd/utils/errorutil"
	"github.com/gerrywa/yacd/utils/pathutil"
)

// ExecuteGeneration executes the main generation logic
func ExecuteGeneration(options types.ParseOptions, reader io.Reader) error {
	if options.Verbose {
		printExecutionInfo(options)
	}

	// Create parser
	p, err := parser.NewParser(options)
	if err != nil {
		return errorutil.WrapError(err, "failed to create parser")
	}

	// Parse make log
	entries, err := p.ParseMakeLog(reader)
	if err != nil {
		return errorutil.WrapParseError(err, "make log")
	}

	if options.Verbose {
		fmt.Printf("Parsing completed, found %d compilation entries\n", len(entries))
	}

	// Create generator
	g := generator.NewGenerator(options)

	// Generate compilation database
	compileDB, err := g.GenerateCompileCommands(entries)
	if err != nil {
		return errorutil.WrapGenerationError(err, "compilation database")
	}

	// Validate compilation database
	if err := g.ValidateCompileDB(compileDB); err != nil {
		return errorutil.WrapValidationError(err, "compilation database")
	}

	// Write output file
	if err := g.WriteToFile(compileDB, options.OutputFile); err != nil {
		return errorutil.WrapFileError(err, "write", options.OutputFile)
	}

	fmt.Printf("Successfully generated %s with %d compilation entries\n",
		options.OutputFile, len(compileDB))
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

		var err error
		reader, err = ExecuteMakeCommand(options.MakeCommand)
		if err != nil {
			return nil, nil, errorutil.WrapExecutionError(err, "make command")
		}

		cleanup = func() {
			if closer, ok := reader.(io.Closer); ok {
				closer.Close()
			}
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

// printExecutionInfo prints execution information in verbose mode
func printExecutionInfo(options types.ParseOptions) {
	fmt.Printf("yacd - Yet Another CompileDB\n")
	if options.MakeCommand != "" {
		fmt.Printf("Make command: %s\n", options.MakeCommand)
	} else if options.InputFile == "" {
		fmt.Printf("Input source: stdin\n")
	} else {
		fmt.Printf("Input file: %s\n", options.InputFile)
	}
	fmt.Printf("Output file: %s\n", options.OutputFile)
	if options.UseRelativePaths {
		fmt.Printf("Using relative paths, base directory: %s\n", options.BaseDir)
	}
	fmt.Printf("Starting to parse...\n")
}
