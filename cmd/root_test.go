package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

const testMakeLog = `make: Entering directory '/home/user/project'
mkdir build
arm-none-eabi-gcc -c -mcpu=cortex-m0 -mthumb -DNDEBUG -DUSE_HAL_DRIVER -DSTM32F030x6 -ICore/Inc -IDrivers/MM32F0140/Include -IDrivers/MM32F0140/HAL_Lib/Inc -IDrivers/CMSIS/Include -Og -Wall -fdata-sections -ffunction-sections -g -gdwarf-2 -MMD -MP -MF"build/system_mm32f0140.d" Drivers/MM32F0140/Source/system_mm32f0140.c -o build/system_mm32f0140.o
arm-none-eabi-gcc -c -mcpu=cortex-m0 -mthumb -DNDEBUG -DUSE_HAL_DRIVER -DSTM32F030x6 -ICore/Inc -IDrivers/MM32F0140/Include -IDrivers/MM32F0140/HAL_Lib/Inc -IDrivers/CMSIS/Include -Og -Wall -fdata-sections -ffunction-sections -g -gdwarf-2 -MMD -MP -MF"build/hal_comp.d" Drivers/MM32F0140/HAL_Lib/Src/hal_comp.c -o build/hal_comp.o
arm-none-eabi-gcc -c -mcpu=cortex-m0 -mthumb -DNDEBUG -DUSE_HAL_DRIVER -DSTM32F030x6 -ICore/Inc -IDrivers/MM32F0140/Include -IDrivers/MM32F0140/HAL_Lib/Inc -IDrivers/CMSIS/Include -Og -Wall -fdata-sections -ffunction-sections -g -gdwarf-2 -MMD -MP -MF"build/main.d" user/app/main.c -o build/main.o
arm-none-eabi-gcc build/system_mm32f0140.o build/hal_comp.o build/main.o -mcpu=cortex-m0 -mthumb -specs=nano.specs -Tmm32f0144c6p.ld -lc -lm -lnosys -Wl,-Map=build/project.map,--cref -Wl,--gc-sections -o build/project.elf
make: Leaving directory '/home/user/project'`

func TestRunGenerateSuccess(t *testing.T) {
	tempDir := t.TempDir()
	inputFilePath := filepath.Join(tempDir, "test.log")
	outputFilePath := filepath.Join(tempDir, "compile_commands.json")

	if err := os.WriteFile(inputFilePath, []byte(testMakeLog), 0644); err != nil {
		t.Fatalf("Failed to create test input file: %v", err)
	}

	var (
		testInputFile  string
		testOutputFile string
		testVerbose    bool
	)

	testCmd := &cobra.Command{
		Use: "yacd",
		RunE: func(cmd *cobra.Command, args []string) error {
			stdinHasData := false
			if testInputFile == "" && !stdinHasData {
				cmd.Help()
				return nil
			}

			if err := ValidateInputSources(testInputFile, "", stdinHasData); err != nil {
				return err
			}

			opts, err := PrepareOptions(testInputFile, testOutputFile, "", "", false, testVerbose, false)
			if err != nil {
				return err
			}

			reader, cleanup, err := PrepareReader(opts, stdinHasData)
			if err != nil {
				return err
			}
			defer cleanup()

			return ExecuteGeneration(&opts, reader)
		},
	}

	testCmd.Flags().StringVarP(&testInputFile, "input", "i", "", "Input make log file path")
	testCmd.Flags().StringVarP(&testOutputFile, "output", "o", "compile_commands.json", "Output file path")
	testCmd.Flags().BoolVarP(&testVerbose, "verbose", "v", false, "Verbose output")

	testCmd.SetArgs([]string{"--input", inputFilePath, "--output", outputFilePath, "--verbose"})

	if err := testCmd.Execute(); err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}

	if _, err := os.Stat(outputFilePath); os.IsNotExist(err) {
		t.Fatal("Output file not created")
	}
}

func TestRootCmdHelp(t *testing.T) {
	var testInputFile string
	testCmd := &cobra.Command{
		Use:   "yacd",
		Short: "Yet Another CompileDB",
		RunE: func(cmd *cobra.Command, args []string) error {
			stdinHasData := HasStdinData()
			if testInputFile == "" && !stdinHasData {
				cmd.Help()
				return nil
			}
			return nil
		},
	}

	testCmd.Flags().StringVarP(&testInputFile, "input", "i", "", "Input file path")
	testCmd.SetArgs([]string{})

	if err := testCmd.Execute(); err != nil {
		t.Errorf("Root command should not return error when showing help: %v", err)
	}
}
