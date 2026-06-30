package parser

import (
	"strings"
	"testing"

	"github.com/gerryqd/yacd/types"
)

func TestParseCrossCompilerCommand(t *testing.T) {
	inputCommand := "/home/gerrywa/eem210/berwick/maxwell/build/reborn/buildroot-isam-reborn-cavium-eem210/output/host/usr/bin/mips64-octeon-linux-gnu-gcc -g -Wall -DDPOE_ROLT -DBOARD_eem210 -g -fPIC -O0 -Wextra -Wall -Wbad-function-cast -Wcast-qual -Wchar-subscripts -Wmissing-prototypes -Wnested-externs -Wpointer-arith -Wshadow -Wstrict-prototypes -Wparentheses -Wswitch -fno-strict-aliasing -Wno-format-truncation -Wno-implicit-function-declaration -Wno-nested-externs -Wno-int-conversion -I/home/gerrywa/eem210/berwick/maxwell/build/reborn/buildroot-isam-reborn-cavium-eem210/output/host/opt/ext-toolchain/output/host/usr/include -DSIM_DELAY -DPPC -Wold-style-declaration -std=gnu89  -I. -I/home/gerrywa/eem210/berwick/maxwell/src/Dpoe_AI/dmlcore/common/include -I/home/gerrywa/eem210/berwick/maxwell/src/Dpoe_AI/dmlcore/build/mips64-octeon-linux/build/include -I/home/gerrywa/eem210/berwick/maxwell/build/reborn/buildroot-isam-reborn-cavium-eem210/output/staging/usr/local/include -I/home/gerrywa/eem210/berwick/maxwell/include -I../../../system-common/include/ -I../../../system-common/lib/liblogger/h -I./include/armstrong -I./include/macfie -I./src -I../Dpoe_AI/include/backTrace -I../DpoeConfD/export -I/home/gerrywa/eem210/berwick/maxwell/build/reborn/buildroot-isam-reborn-cavium-eem210/output/host/confd/include/ -I../Dpoe_AI/dmlcore/lib/libdmlutils -o /home/gerrywa/eem210/berwick/maxwell/src/Dpoe_AI/dmlcore/build/mips64-octeon-linux/obj/cms/cable_bundle.o -c ./src/cable_bundle.c"

	options := types.ParseOptions{
		Verbose: false,
	}

	p, err := NewParser(options)
	if err != nil {
		t.Fatalf("Error creating parser: %v", err)
	}

	entries, err := p.ParseMakeLog(strings.NewReader(inputCommand))
	if err != nil {
		t.Fatalf("Error parsing command: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("Expected at least 1 parsed entry, got 0 — cross-compiler regex may not match")
	}

	entry := entries[0]

	// Verify compiler detection (full path is preserved)
	if !strings.Contains(entry.Compiler, "mips64-octeon-linux-gnu-gcc") {
		t.Errorf("Compiler = %s, expected to contain mips64-octeon-linux-gnu-gcc", entry.Compiler)
	}

	// Verify source file extraction
	expectedSource := "./src/cable_bundle.c"
	if entry.SourceFile != expectedSource {
		t.Errorf("SourceFile = %s, expected %s", entry.SourceFile, expectedSource)
	}

	// Verify output file extraction
	expectedOutput := "/home/gerrywa/eem210/berwick/maxwell/src/Dpoe_AI/dmlcore/build/mips64-octeon-linux/obj/cms/cable_bundle.o"
	if entry.OutputFile != expectedOutput {
		t.Errorf("OutputFile = %s, expected %s", entry.OutputFile, expectedOutput)
	}

	// Verify args include the compiler as first element
	if len(entry.Args) == 0 {
		t.Fatal("Args should not be empty")
	}
	if !strings.Contains(entry.Args[0], "mips64-octeon-linux-gnu-gcc") {
		t.Errorf("Args[0] = %s, expected to contain mips64-octeon-linux-gnu-gcc", entry.Args[0])
	}

	// Verify compilation entry can be constructed
	compilationEntry := types.CompilationEntry{
		Directory: entry.WorkingDir,
		Command:   strings.Join(entry.Args, " "),
		File:      entry.SourceFile,
		Output:    entry.OutputFile,
	}
	if compilationEntry.File != expectedSource {
		t.Errorf("CompilationEntry.File = %s, expected %s", compilationEntry.File, expectedSource)
	}
}

func TestParseDollarParenSubstitution(t *testing.T) {
	p, err := NewParser(types.ParseOptions{})
	if err != nil {
		t.Fatalf("Error creating parser: %v", err)
	}

	// gcc with $(...) command substitution
	input := `gcc -I$(shell echo '/usr/include') -c main.c -o main.o`
	entries, err := p.ParseMakeLog(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Error parsing: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
	if entries[0].SourceFile != "main.c" {
		t.Errorf("SourceFile = %s, expected main.c", entries[0].SourceFile)
	}
}

func TestParseBacktickSubstitution(t *testing.T) {
	p, err := NewParser(types.ParseOptions{})
	if err != nil {
		t.Fatalf("Error creating parser: %v", err)
	}

	input := "gcc -I`echo '/opt/include'` -c main.c -o main.o"
	entries, err := p.ParseMakeLog(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Error parsing: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
	if entries[0].SourceFile != "main.c" {
		t.Errorf("SourceFile = %s, expected main.c", entries[0].SourceFile)
	}
}

func TestParseInvalidCompiler(t *testing.T) {
	p, err := NewParser(types.ParseOptions{})
	if err != nil {
		t.Fatalf("Error creating parser: %v", err)
	}

	tests := []struct {
		name string
		line string
	}{
		{"Source file named gcc", "echo gcc"},
		{"Tab file", "lex.tab.c -o output"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.parseCompileCommand(tt.line)
			_ = result // just ensure no panic
		})
	}
}

func TestParseMakeLogWithLinkCommand(t *testing.T) {
	p, err := NewParser(types.ParseOptions{BaseDir: "/project"})
	if err != nil {
		t.Fatalf("Error creating parser: %v", err)
	}

	makeLog := `make: Entering directory '/home/user/project'
gcc -c -Wall main.c -o main.o
gcc main.o util.o -o program
make: Leaving directory '/home/user/project'`

	entries, err := p.ParseMakeLog(strings.NewReader(makeLog))
	if err != nil {
		t.Fatalf("ParseMakeLog failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry (link command has no source), got %d", len(entries))
	}
}

func TestParseMakeLogWithDollarParenEcho(t *testing.T) {
	p, err := NewParser(types.ParseOptions{BaseDir: "/project"})
	if err != nil {
		t.Fatalf("Error creating parser: %v", err)
	}

	makeLog := `make: Entering directory '/home/user/project'
gcc -c -I$(shell echo '/opt/include') main.c -o main.o
make: Leaving directory '/home/user/project'`

	entries, err := p.ParseMakeLog(strings.NewReader(makeLog))
	if err != nil {
		t.Fatalf("ParseMakeLog failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
}
