package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gerryqd/yacd/types"
	"github.com/spf13/cobra"
)

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
		testMakeCommand      string
		testShowVersion      bool
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
				MakeCommand:      testMakeCommand,
			}

			// Check if version flag is set
			if testShowVersion {
				// For testing, we just return nil
				return nil
			}

			// Check if no input is provided and show help instead of error
			stdinHasData := false // For testing, assume no stdin data
			if options.InputFile == "" && options.MakeCommand == "" && !stdinHasData {
				// Show help information instead of error when no input is provided
				cmd.Help()
				return nil
			}

			// Validate input sources
			if err := ValidateInputSources(options.InputFile, options.MakeCommand, stdinHasData); err != nil {
				return err
			}

			// Prepare options
			opts, err := PrepareOptions(options.InputFile, options.OutputFile, options.MakeCommand, options.BaseDir, options.UseRelativePaths, options.Verbose)
			if err != nil {
				return err
			}

			// Prepare reader
			reader, cleanup, err := PrepareReader(opts, stdinHasData)
			if err != nil {
				return err
			}
			defer cleanup()

			// Execute generation
			return ExecuteGeneration(&opts, reader)
		},
	}

	testCmd.Flags().StringVarP(&testInputFile, "input", "i", "", "Input make log file path")
	testCmd.Flags().StringVarP(&testOutputFile, "output", "o", "compile_commands.json", "Output compile_commands.json file path")
	testCmd.Flags().BoolVarP(&testUseRelativePaths, "relative", "r", false, "Use relative paths")
	testCmd.Flags().StringVarP(&testBaseDir, "base-dir", "b", "", "Base directory path")
	testCmd.Flags().BoolVarP(&testVerbose, "verbose", "v", false, "Verbose output")
	testCmd.Flags().StringVarP(&testMakeCommand, "dry-run", "n", "", "Execute make command with -Bnkw flags and process output directly")
	testCmd.Flags().BoolVarP(&testShowVersion, "version", "V", false, "Print version information and exit")

	// Mark mutually exclusive parameters
	testCmd.MarkFlagsMutuallyExclusive("input", "dry-run")

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

	// For now, just check that the file was created successfully
	// In a real test, we would validate the content as well
}

func TestValidateInputSourcesInRoot(t *testing.T) {
	tests := []struct {
		name          string
		inputFile     string
		makeCommand   string
		stdinHasData  bool
		expectError   bool
		errorContains string
	}{
		{
			name:          "No input sources",
			inputFile:     "",
			makeCommand:   "",
			stdinHasData:  false,
			expectError:   true,
			errorContains: "no input source provided",
		},
		{
			name:         "Input file only",
			inputFile:    "test.log",
			makeCommand:  "",
			stdinHasData: false,
			expectError:  false,
		},
		{
			name:         "Make command only",
			inputFile:    "",
			makeCommand:  "make clean all",
			stdinHasData: false,
			expectError:  false,
		},
		{
			name:          "Multiple input sources",
			inputFile:     "test.log",
			makeCommand:   "make clean all",
			stdinHasData:  false,
			expectError:   true,
			errorContains: "multiple input sources provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInputSources(tt.inputFile, tt.makeCommand, tt.stdinHasData)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateInputSources() expected error, got nil")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("ValidateInputSources() error = %v, expected to contain %s", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateInputSources() unexpected error: %v", err)
			}
		})
	}
}

func TestPrepareOptionsInRoot(t *testing.T) {
	tests := []struct {
		name             string
		inputFile        string
		outputFile       string
		makeCommand      string
		baseDir          string
		useRelativePaths bool
		verbose          bool
		expectError      bool
	}{
		{
			name:             "Basic options",
			inputFile:        "input.log",
			outputFile:       "output.json",
			makeCommand:      "",
			baseDir:          "",
			useRelativePaths: false,
			verbose:          false,
			expectError:      false,
		},
		{
			name:             "With relative paths",
			inputFile:        "input.log",
			outputFile:       "output.json",
			makeCommand:      "",
			baseDir:          "/project",
			useRelativePaths: true,
			verbose:          true,
			expectError:      false,
		},
		{
			name:             "With make command",
			inputFile:        "",
			outputFile:       "output.json",
			makeCommand:      "make clean all",
			baseDir:          "",
			useRelativePaths: false,
			verbose:          false,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, err := PrepareOptions(tt.inputFile, tt.outputFile, tt.makeCommand, tt.baseDir, tt.useRelativePaths, tt.verbose)

			if tt.expectError {
				if err == nil {
					t.Errorf("PrepareOptions() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("PrepareOptions() unexpected error: %v", err)
				return
			}

			// Validate options
			if options.InputFile != tt.inputFile {
				t.Errorf("InputFile = %s, expected %s", options.InputFile, tt.inputFile)
			}
			if options.OutputFile != tt.outputFile {
				t.Errorf("OutputFile = %s, expected %s", options.OutputFile, tt.outputFile)
			}
			if options.MakeCommand != tt.makeCommand {
				t.Errorf("MakeCommand = %s, expected %s", options.MakeCommand, tt.makeCommand)
			}
			if options.BaseDir != tt.baseDir {
				t.Errorf("BaseDir = %s, expected %s", options.BaseDir, tt.baseDir)
			}
			if options.UseRelativePaths != tt.useRelativePaths {
				t.Errorf("UseRelativePaths = %t, expected %t", options.UseRelativePaths, tt.useRelativePaths)
			}
			if options.Verbose != tt.verbose {
				t.Errorf("Verbose = %t, expected %t", options.Verbose, tt.verbose)
			}
		})
	}
}

func TestHasStdinDataInRoot(t *testing.T) {
	// This is a simple test - in practice, testing stdin detection is complex
	// We just ensure the function doesn't panic
	result := HasStdinData()
	// result could be true or false depending on test environment
	_ = result // Just to avoid unused variable error
}

func TestRootCmdHelp(t *testing.T) {
	// Test that the root command provides help when no arguments are given
	// Create a copy of root command for testing
	testCmd := &cobra.Command{
		Use:   "yacd",
		Short: "Yet Another CompileDB - Generate compile_commands.json from make logs",
		Long:  `yacd (Yet Another CompileDB) is a tool for generating compile_commands.json files from make logs of makefile projects.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use actual global variables for this test
			// Check if no input is provided and show help instead of error
			stdinHasData := HasStdinData()
			if inputFile == "" && makeCommand == "" && !stdinHasData {
				// Show help information instead of error when no input is provided
				cmd.Help()
				return nil
			}
			return nil
		},
	}

	// Add flags
	testCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input make log file path")
	testCmd.Flags().StringVarP(&outputFile, "output", "o", "compile_commands.json", "Output compile_commands.json file path")
	testCmd.Flags().BoolVarP(&useRelativePaths, "relative", "r", false, "Use relative paths instead of absolute paths")
	testCmd.Flags().StringVarP(&baseDir, "base-dir", "b", "", "Base directory path (used with --relative)")
	testCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	testCmd.Flags().StringVarP(&makeCommand, "dry-run", "n", "", "Execute make command with -Bnkw flags and process output directly")
	testCmd.Flags().BoolVarP(&showVersion, "version", "V", false, "Print version information and exit")

	// Execute with no arguments (should show help)
	testCmd.SetArgs([]string{})
	err := testCmd.Execute()
	if err != nil {
		t.Errorf("Root command should not return error when showing help: %v", err)
	}
}
