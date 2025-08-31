package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gerrywa/yacd/generator"
	"github.com/gerrywa/yacd/parser"
	"github.com/gerrywa/yacd/types"
	"github.com/spf13/cobra"
)

// runGenerateWithOptions is a helper function for testing
func runGenerateWithOptions(options types.ParseOptions) error {
	// Validate input file
	if options.InputFile == "" {
		return fmt.Errorf("input file must be specified")
	}

	// Check if input file exists
	if _, err := os.Stat(options.InputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", options.InputFile)
	}

	// Handle base directory
	if options.UseRelativePaths && options.BaseDir == "" {
		// If no base directory specified, use output file's directory
		options.BaseDir = filepath.Dir(options.OutputFile)
		if options.BaseDir == "." {
			var err error
			options.BaseDir, err = os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %w", err)
			}
		}
	}

	if options.Verbose {
		fmt.Printf("yacd - Yet Another CompileDB\n")
		fmt.Printf("Input file: %s\n", options.InputFile)
		fmt.Printf("Output file: %s\n", options.OutputFile)
		if options.UseRelativePaths {
			fmt.Printf("Using relative paths, base directory: %s\n", options.BaseDir)
		}
		fmt.Printf("Starting to parse...\n")
	}

	// Open input file
	file, err := os.Open(options.InputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	// Create parser
	p, err := parser.NewParser(options)
	if err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}

	// Parse make log
	entries, err := p.ParseMakeLog(file)
	if err != nil {
		return fmt.Errorf("failed to parse make log: %w", err)
	}

	if options.Verbose {
		fmt.Printf("Parsing completed, found %d compilation entries\n", len(entries))
	}

	// Create generator
	g := generator.NewGenerator(options)

	// Generate compilation database
	compileDB, err := g.GenerateCompileCommands(entries)
	if err != nil {
		return fmt.Errorf("failed to generate compilation database: %w", err)
	}

	// Validate compilation database
	if err := g.ValidateCompileDB(compileDB); err != nil {
		return fmt.Errorf("compilation database validation failed: %w", err)
	}

	// Write output file
	if err := g.WriteToFile(compileDB, options.OutputFile); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Successfully generated %s with %d compilation entries\n", options.OutputFile, len(compileDB))
	return nil
}

// Create test make log content
const testMakeLog = `make: Entering directory '/home/user/project'
mkdir build
arm-none-eabi-gcc -c -mcpu=cortex-m0 -mthumb -DNDEBUG -DUSE_HAL_DRIVER -DSTM32F030x6 -ICore/Inc -IDrivers/MM32F0140/Include -IDrivers/MM32F0140/HAL_Lib/Inc -IDrivers/CMSIS/Include -Og -Wall -fdata-sections -ffunction-sections -g -gdwarf-2 -MMD -MP -MF"build/system_mm32f0140.d" Drivers/MM32F0140/Source/system_mm32f0140.c -o build/system_mm32f0140.o
arm-none-eabi-gcc -c -mcpu=cortex-m0 -mthumb -DNDEBUG -DUSE_HAL_DRIVER -DSTM32F030x6 -ICore/Inc -IDrivers/MM32F0140/Include -IDrivers/MM32F0140/HAL_Lib/Inc -IDrivers/CMSIS/Include -Og -Wall -fdata-sections -ffunction-sections -g -gdwarf-2 -MMD -MP -MF"build/hal_comp.d" Drivers/MM32F0140/HAL_Lib/Src/hal_comp.c -o build/hal_comp.o
arm-none-eabi-gcc -c -mcpu=cortex-m0 -mthumb -DNDEBUG -DUSE_HAL_DRIVER -DSTM32F030x6 -ICore/Inc -IDrivers/MM32F0140/Include -IDrivers/MM32F0140/HAL_Lib/Inc -IDrivers/CMSIS/Include -Og -Wall -fdata-sections -ffunction-sections -g -gdwarf-2 -MMD -MP -MF"build/main.d" user/app/main.c -o build/main.o
arm-none-eabi-gcc build/system_mm32f0140.o build/hal_comp.o build/main.o -mcpu=cortex-m0 -mthumb -specs=nano.specs -Tmm32f0144c6p.ld -lc -lm -lnosys -Wl,-Map=build/project.map,--cref -Wl,--gc-sections -o build/project.elf
make: Leaving directory '/home/user/project'`

func TestRunGenerateSuccess(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create test input file
	inputFilePath := filepath.Join(tempDir, "test.log")
	outputFilePath := filepath.Join(tempDir, "compile_commands.json")

	if err := os.WriteFile(inputFilePath, []byte(testMakeLog), 0644); err != nil {
		t.Fatalf("Failed to create test input file: %v", err)
	}

	// Create local variables for this test
	var (
		testInputFile        string
		testOutputFile       string
		testUseRelativePaths bool
		testBaseDir          string
		testVerbose          bool
	)

	// Create a copy of root command for testing
	testCmd := &cobra.Command{
		Use: "yacd",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use local variables instead of global ones
			options := types.ParseOptions{
				InputFile:        testInputFile,
				OutputFile:       testOutputFile,
				UseRelativePaths: testUseRelativePaths,
				BaseDir:          testBaseDir,
				Verbose:          testVerbose,
			}
			return runGenerateWithOptions(options)
		},
	}

	testCmd.Flags().StringVarP(&testInputFile, "input", "i", "", "Input make log file path")
	testCmd.Flags().StringVarP(&testOutputFile, "output", "o", "compile_commands.json", "Output compile_commands.json file path")
	testCmd.Flags().BoolVarP(&testUseRelativePaths, "relative", "r", false, "Use relative paths")
	testCmd.Flags().StringVarP(&testBaseDir, "base-dir", "b", "", "Base directory path")
	testCmd.Flags().BoolVarP(&testVerbose, "verbose", "v", false, "Verbose output")

	// Set parameters
	testCmd.SetArgs([]string{
		"--input", inputFilePath,
		"--output", outputFilePath,
		"--verbose",
	})

	// Execute command
	err := testCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}

	// Check if output file exists
	if _, err := os.Stat(outputFilePath); os.IsNotExist(err) {
		t.Fatal("Output file not created")
	}

	// Validate output file content
	data, err := os.ReadFile(outputFilePath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var compileDB types.CompilationDatabase
	if err := json.Unmarshal(data, &compileDB); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Should have 3 compilation entries
	expectedCount := 3
	if len(compileDB) != expectedCount {
		t.Errorf("Generated %d compilation entries, expected %d", len(compileDB), expectedCount)
	}

	// Validate first entry
	if len(compileDB) > 0 {
		entry := compileDB[0]
		expectedDir := filepath.ToSlash("/home/user/project")
		actualDir := filepath.ToSlash(entry.Directory)
		if actualDir != expectedDir {
			t.Errorf("First entry directory = %s, expected %s", entry.Directory, expectedDir)
		}
		if !strings.Contains(entry.File, "system_mm32f0140.c") {
			t.Errorf("First entry file does not contain system_mm32f0140.c: %s", entry.File)
		}
		// Check Arguments field instead of Command
		if len(entry.Arguments) == 0 || !strings.Contains(entry.Arguments[0], "arm-none-eabi-gcc") {
			t.Errorf("First entry arguments do not contain arm-none-eabi-gcc: %v", entry.Arguments)
		}
	}
}

func TestRunGenerateWithRelativePaths(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create test input file
	inputFilePath := filepath.Join(tempDir, "test.log")
	outputFilePath := filepath.Join(tempDir, "compile_commands.json")

	if err := os.WriteFile(inputFilePath, []byte(testMakeLog), 0644); err != nil {
		t.Fatalf("Failed to create test input file: %v", err)
	}

	// Create local variables for this test
	var (
		testInputFile        string
		testOutputFile       string
		testUseRelativePaths bool
		testBaseDir          string
		testVerbose          bool
	)

	// Create a copy of root command for testing
	testCmd := &cobra.Command{
		Use: "yacd",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use local variables instead of global ones
			options := types.ParseOptions{
				InputFile:        testInputFile,
				OutputFile:       testOutputFile,
				UseRelativePaths: testUseRelativePaths,
				BaseDir:          testBaseDir,
				Verbose:          testVerbose,
			}
			return runGenerateWithOptions(options)
		},
	}

	testCmd.Flags().StringVarP(&testInputFile, "input", "i", "", "Input make log file path")
	testCmd.Flags().StringVarP(&testOutputFile, "output", "o", "compile_commands.json", "Output compile_commands.json file path")
	testCmd.Flags().BoolVarP(&testUseRelativePaths, "relative", "r", false, "Use relative paths")
	testCmd.Flags().StringVarP(&testBaseDir, "base-dir", "b", "", "Base directory path")
	testCmd.Flags().BoolVarP(&testVerbose, "verbose", "v", false, "Verbose output")

	// Set parameters (with relative paths)
	testCmd.SetArgs([]string{
		"--input", inputFilePath,
		"--output", outputFilePath,
		"--relative",
		"--base-dir", "/home/user",
	})

	// Execute command
	err := testCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}

	// Validate output file content
	data, err := os.ReadFile(outputFilePath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var compileDB types.CompilationDatabase
	if err := json.Unmarshal(data, &compileDB); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Validate relative paths are used
	if len(compileDB) > 0 {
		entry := compileDB[0]
		if entry.Directory != "project" {
			t.Errorf("First entry directory = %s, expected relative path 'project'", entry.Directory)
		}
	}
}

func TestRunGenerateInputFileNotExists(t *testing.T) {
	// Reset global variables
	inputFile, outputFile, useRelativePaths, baseDir, verbose = "", "compile_commands.json", false, "", false

	// Create a copy of root command for testing
	testCmd := &cobra.Command{
		Use:  "yacd",
		RunE: runGenerate,
	}

	testCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input make log file path")
	testCmd.Flags().StringVarP(&outputFile, "output", "o", "compile_commands.json", "Output compile_commands.json file path")
	testCmd.Flags().BoolVarP(&useRelativePaths, "relative", "r", false, "Use relative paths")
	testCmd.Flags().StringVarP(&baseDir, "base-dir", "b", "", "Base directory path")
	testCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Set non-existent input file
	testCmd.SetArgs([]string{
		"--input", "/non/existent/file.log",
		"--output", "compile_commands.json",
	})

	// Execute command, should fail
	err := testCmd.Execute()
	if err == nil {
		t.Fatal("Expected command to fail, but it succeeded")
	}

	if !strings.Contains(err.Error(), "file does not exist") {
		t.Errorf("Incorrect error message: %v", err)
	}
}

func TestRunGenerateEmptyInput(t *testing.T) {
	// Reset global variables
	inputFile, outputFile, useRelativePaths, baseDir, verbose = "", "compile_commands.json", false, "", false

	// Create a copy of root command for testing
	testCmd := &cobra.Command{
		Use:  "yacd",
		RunE: runGenerate,
	}

	testCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input make log file path")
	testCmd.Flags().StringVarP(&outputFile, "output", "o", "compile_commands.json", "Output compile_commands.json file path")
	testCmd.Flags().BoolVarP(&useRelativePaths, "relative", "r", false, "Use relative paths")
	testCmd.Flags().StringVarP(&baseDir, "base-dir", "b", "", "Base directory path")
	testCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Don't set input file
	testCmd.SetArgs([]string{
		"--output", "compile_commands.json",
	})

	// Execute command, should show help
	err := testCmd.Execute()
	if err != nil {
		t.Fatalf("Expected command to show help, but it failed: %v", err)
	}

	// This test just verifies that the command doesn't crash when no input is provided
	// The actual behavior is to show help, which is what we expect
}

func TestRunGenerateWithSubdirectoryOutput(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create test input file
	inputFilePath := filepath.Join(tempDir, "test.log")
	outputFilePath := filepath.Join(tempDir, "build", "output", "compile_commands.json")

	if err := os.WriteFile(inputFilePath, []byte(testMakeLog), 0644); err != nil {
		t.Fatalf("Failed to create test input file: %v", err)
	}

	// Create local variables for this test
	var (
		testInputFile        string
		testOutputFile       string
		testUseRelativePaths bool
		testBaseDir          string
		testVerbose          bool
	)

	// Create a copy of root command for testing
	testCmd := &cobra.Command{
		Use: "yacd",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use local variables instead of global ones
			options := types.ParseOptions{
				InputFile:        testInputFile,
				OutputFile:       testOutputFile,
				UseRelativePaths: testUseRelativePaths,
				BaseDir:          testBaseDir,
				Verbose:          testVerbose,
			}
			return runGenerateWithOptions(options)
		},
	}

	testCmd.Flags().StringVarP(&testInputFile, "input", "i", "", "Input make log file path")
	testCmd.Flags().StringVarP(&testOutputFile, "output", "o", "compile_commands.json", "Output compile_commands.json file path")
	testCmd.Flags().BoolVarP(&testUseRelativePaths, "relative", "r", false, "Use relative paths")
	testCmd.Flags().StringVarP(&testBaseDir, "base-dir", "b", "", "Base directory path")
	testCmd.Flags().BoolVarP(&testVerbose, "verbose", "v", false, "Verbose output")

	// Set parameters, output to subdirectory
	testCmd.SetArgs([]string{
		"--input", inputFilePath,
		"--output", outputFilePath,
	})

	// Execute command
	err := testCmd.Execute()
	if err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}

	// Check if output file exists
	if _, err := os.Stat(outputFilePath); os.IsNotExist(err) {
		t.Fatal("Output file not created")
	}

	// Check if subdirectory was created
	subDir := filepath.Dir(outputFilePath)
	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Fatal("Subdirectory not created")
	}
}

func TestRunGenerateEmptyMakeLog(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	// Create empty test input file
	inputFilePath := filepath.Join(tempDir, "empty.log")
	outputFilePath := filepath.Join(tempDir, "compile_commands.json")

	if err := os.WriteFile(inputFilePath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty test input file: %v", err)
	}

	// Create local variables for this test
	var (
		testInputFile        string
		testOutputFile       string
		testUseRelativePaths bool
		testBaseDir          string
		testVerbose          bool
	)

	// Create a copy of root command for testing
	testCmd := &cobra.Command{
		Use: "yacd",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use local variables instead of global ones
			options := types.ParseOptions{
				InputFile:        testInputFile,
				OutputFile:       testOutputFile,
				UseRelativePaths: testUseRelativePaths,
				BaseDir:          testBaseDir,
				Verbose:          testVerbose,
			}
			return runGenerateWithOptions(options)
		},
	}

	testCmd.Flags().StringVarP(&testInputFile, "input", "i", "", "Input make log file path")
	testCmd.Flags().StringVarP(&testOutputFile, "output", "o", "compile_commands.json", "Output compile_commands.json file path")
	testCmd.Flags().BoolVarP(&testUseRelativePaths, "relative", "r", false, "Use relative paths")
	testCmd.Flags().StringVarP(&testBaseDir, "base-dir", "b", "", "Base directory path")
	testCmd.Flags().BoolVarP(&testVerbose, "verbose", "v", false, "Verbose output")

	// Set parameters
	testCmd.SetArgs([]string{
		"--input", inputFilePath,
		"--output", outputFilePath,
	})

	// Execute command, should fail (because no compilation entries)
	err := testCmd.Execute()
	if err == nil {
		t.Fatal("Expected command to fail, but it succeeded")
	}

	if !strings.Contains(err.Error(), "compilation database is empty") {
		t.Errorf("Incorrect error message: %v", err)
	}
}
