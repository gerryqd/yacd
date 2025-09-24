package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gerryqd/yacd/types"
)

func TestParseCrossCompilerCommand(t *testing.T) {
	// The input command to parse
	inputCommand := "/home/gerrywa/eem210/berwick/maxwell/build/reborn/buildroot-isam-reborn-cavium-eem210/output/host/usr/bin/mips64-octeon-linux-gnu-gcc -g -Wall -DDPOE_ROLT -DBOARD_eem210 -g -fPIC -O0 -Wextra -Wall -Wbad-function-cast -Wcast-qual -Wchar-subscripts -Wmissing-prototypes -Wnested-externs -Wpointer-arith -Wshadow -Wstrict-prototypes -Wparentheses -Wswitch -fno-strict-aliasing -Wno-format-truncation -Wno-implicit-function-declaration -Wno-nested-externs -Wno-int-conversion -I/home/gerrywa/eem210/berwick/maxwell/build/reborn/buildroot-isam-reborn-cavium-eem210/output/host/opt/ext-toolchain/output/host/usr/include -DSIM_DELAY -DPPC -Wold-style-declaration -std=gnu89  -I. -I/home/gerrywa/eem210/berwick/maxwell/src/Dpoe_AI/dmlcore/common/include -I/home/gerrywa/eem210/berwick/maxwell/src/Dpoe_AI/dmlcore/build/mips64-octeon-linux/build/include -I/home/gerrywa/eem210/berwick/maxwell/build/reborn/buildroot-isam-reborn-cavium-eem210/output/staging/usr/local/include -I/home/gerrywa/eem210/berwick/maxwell/include -I../../../system-common/include/ -I../../../system-common/lib/liblogger/h -I./include/armstrong -I./include/macfie -I./src -I../Dpoe_AI/include/backTrace -I../DpoeConfD/export -I/home/gerrywa/eem210/berwick/maxwell/build/reborn/buildroot-isam-reborn-cavium-eem210/output/host/confd/include/ -I../Dpoe_AI/dmlcore/lib/libdmlutils -o /home/gerrywa/eem210/berwick/maxwell/src/Dpoe_AI/dmlcore/build/mips64-octeon-linux/obj/cms/cable_bundle.o -c ./src/cable_bundle.c"

	// Create a new parser with verbose output to see what happens
	options := types.ParseOptions{
		Verbose: true,
	}

	p, err := NewParser(options)
	if err != nil {
		t.Fatalf("Error creating parser: %v", err)
	}

	// Parse the command by simulating it as a single line in a reader
	entries, err := p.ParseMakeLog(strings.NewReader(inputCommand))
	if err != nil {
		t.Fatalf("Error parsing command: %v", err)
	}

	// Display the results
	fmt.Printf("Parsed %d entries:\n", len(entries))
	for i, entry := range entries {
		fmt.Printf("Entry %d:\n", i+1)
		fmt.Printf("  WorkingDir: %s\n", entry.WorkingDir)
		fmt.Printf(" Compiler: %s\n", entry.Compiler)
		fmt.Printf(" Args: %v\n", entry.Args)
		fmt.Printf("  SourceFile: %s\n", entry.SourceFile)
		fmt.Printf("  OutputFile: %s\n", entry.OutputFile)
		fmt.Println()
	}

	// Generate the expected record content based on the parsed data
	if len(entries) > 0 {
		entry := entries[0]
		fmt.Println("Generated record content:")
		fmt.Printf("Compiler: %s\n", entry.Compiler)
		fmt.Printf("Source File: %s\n", entry.SourceFile)
		fmt.Printf("Output File: %s\n", entry.OutputFile)
		fmt.Printf("Working Directory: %s\n", entry.WorkingDir)
		fmt.Printf("Number of Arguments: %d\n", len(entry.Args))

		// Show a more detailed representation
		fmt.Println("\nDetailed Compilation Entry (for compile_commands.json):")
		compilationEntry := types.CompilationEntry{
			Directory: entry.WorkingDir,
			Command:   inputCommand, // The full command
			File:      entry.SourceFile,
			Output:    entry.OutputFile,
		}
		fmt.Printf("%+v\n", compilationEntry)
	} else {
		t.Log("No entries were parsed - this indicates the compiler regex doesn't match cross-compilers")
	}
}
