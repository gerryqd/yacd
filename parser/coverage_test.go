package parser

import (
	"strings"
	"testing"

	"github.com/gerryqd/yacd/types"
)

func TestPackageParseMakeLog(t *testing.T) {
	makeLog := `make: Entering directory '/home/user/project'
gcc -c -Wall main.c -o main.o
make: Leaving directory '/home/user/project'`

	entries, err := ParseMakeLog(strings.NewReader(makeLog), types.ParseOptions{BaseDir: "/project"})
	if err != nil {
		t.Fatalf("ParseMakeLog failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
	if entries[0].SourceFile != "main.c" {
		t.Errorf("SourceFile = %q, expected main.c", entries[0].SourceFile)
	}
}

func TestPackageParseMakeLogNewParserError(t *testing.T) {
	// Package ParseMakeLog must propagate parser creation failures (none expected
	// here, but we exercise the error-returning code path).
	entries, err := ParseMakeLog(strings.NewReader("gcc -c main.c -o main.o"), types.ParseOptions{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
}

func TestParseMakeLogScannerError(t *testing.T) {
	p, err := NewParser(types.ParseOptions{BaseDir: "/project"})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// errReader always returns an error on Read, which surfaces as a scanner error.
	_, err = p.ParseMakeLog(&errReader{})
	if err == nil {
		t.Fatal("Expected error from failing reader, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read input") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

type errReader struct{}

func (errReader) Read(p []byte) (int, error) { return 0, errReaderErr }

type errReaderErrType struct{}

func (errReaderErrType) Error() string { return "simulated read failure" }

var errReaderErr = errReaderErrType{}

func TestResolveRelativePath(t *testing.T) {
	p := newTestParser(t)

	tests := []struct {
		name         string
		baseDir      string
		relativePath string
		expected     string
	}{
		{"Relative", "/project", "src/main.c", "/project/src/main.c"},
		{"Absolute preserved", "/project", "/abs/path/main.c", "/abs/path/main.c"},
		{"Empty base", "", "src/main.c", "src/main.c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.resolveRelativePath(tt.baseDir, tt.relativePath); got != tt.expected {
				t.Errorf("resolveRelativePath(%q, %q) = %q, expected %q", tt.baseDir, tt.relativePath, got, tt.expected)
			}
		})
	}
}

func TestExtractPathFromCommand(t *testing.T) {
	p := newTestParser(t)

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{"Single quoted", "test -f 'x' || echo '/opt/include'", "/opt/include"},
		{"Double quoted", `echo "/usr/include"`, "/usr/include"},
		{"Unquoted", "echo /usr/local/include", "/usr/local/include"},
		{"No echo", "ls -la", ""},
		{"Empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.extractPathFromCommand(tt.command); got != tt.expected {
				t.Errorf("extractPathFromCommand(%q) = %q, expected %q", tt.command, got, tt.expected)
			}
		})
	}
}

func TestIsInvalidCompiler(t *testing.T) {
	p := newTestParser(t)

	tests := []struct {
		name     string
		compiler string
		invalid  bool
	}{
		{"Valid gcc", "gcc", false},
		{"Valid cross gcc", "arm-none-eabi-gcc", false},
		{"Valid clang", "clang", false},
		{"D macro", "-DTEST", true},
		{"C source", "main.c", true},
		{"Cpp source", "main.cpp", true},
		{"CC source", "main.cc", true},
		{"Cxx source", "main.cxx", true},
		{"Assembly lower", "start.s", true},
		{"Assembly upper", "start.S", true},
		{"Object file", "main.o", true},
		{"Yacc tab", "parser.yy.tab.c", true},
		{"Tab file", "foo.tab.c", true},
		{"Lex file", "scanner.lex.yy.c", true},
		{"Uppercase C", "foo.C", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.isInvalidCompiler(tt.compiler); got != tt.invalid {
				t.Errorf("isInvalidCompiler(%q) = %v, expected %v", tt.compiler, got, tt.invalid)
			}
		})
	}
}

func TestFindCompilerStartIndexWithDMacro(t *testing.T) {
	p := newTestParser(t)

	// A compiler-like token immediately following a -D flag should be skipped so
	// that macro values resembling compiler names are not mistaken for the
	// compiler. The first "gcc" (after -DDEFINE) is skipped, and the real
	// cross-compiler is selected instead.
	line := "-DDEFINE gcc arm-none-eabi-gcc -c main.c -o main.o"
	idx := p.findCompilerStartIndex(line)
	if idx < 0 {
		t.Fatalf("Expected to find compiler, got -1")
	}
	compilerPart := line[idx:]
	if !strings.HasPrefix(compilerPart, "arm-none-eabi-gcc") {
		t.Errorf("Expected compiler to start at 'arm-none-eabi-gcc', got: %q", compilerPart)
	}
}

func TestProcessCommandSubstitution(t *testing.T) {
	p := newTestParser(t)

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"Dollar paren with echo", "gcc -I$(shell echo '/opt/include') -c main.c", "/opt/include"},
		{"Backtick with echo", "gcc -I`echo '/opt/include'` -c main.c", "/opt/include"},
		{"No substitution", "gcc -c main.c", "main.c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.processCommandSubstitution(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("processCommandSubstitution(%q) = %q, expected to contain %q", tt.input, result, tt.contains)
			}
		})
	}
}

func TestProcessBacktickSubstitutionWithFilenameSuffix(t *testing.T) {
	p := newTestParser(t)

	// When a backtick substitution is immediately followed by a filename (e.g.
	// `/include`file.c), a separating space must be inserted so the path and the
	// filename do not get merged.
	input := "gcc -I`echo '/opt/include'`file.c -c main.c"
	result := p.processBacktickSubstitution(input)
	if !strings.Contains(result, "/opt/include file.c") {
		t.Errorf("Expected space-inserted path, got: %q", result)
	}
}

func TestProcessDollarParenSubstitutionNoPath(t *testing.T) {
	p := newTestParser(t)

	// A $(...) substitution whose embedded command has no echo should be removed.
	input := "gcc $(date) -c main.c -o main.o"
	result := p.processDollarParenSubstitution(input)
	if strings.Contains(result, "$(date)") {
		t.Errorf("Expected $(date) to be removed, got: %q", result)
	}
}

func TestHandleDirectoryChangeLeaveEmptyStack(t *testing.T) {
	// Leaving a directory when the stack is empty must not panic and should be a no-op.
	p, err := NewParser(types.ParseOptions{})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	handled := p.handleDirectoryChange("make: Leaving directory '/some/dir'")
	if !handled {
		t.Error("Expected leaving-directory line to be handled")
	}
	if len(p.dirStack) != 0 {
		t.Errorf("Expected empty stack, got %v", p.dirStack)
	}
}

func TestParseShellCommandChainInvalidCompiler(t *testing.T) {
	p, err := NewParser(types.ParseOptions{})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	p.dirStack = []string{"/test"}

	// A cd chain whose compiler part is invalid (no source file) returns nil.
	result := p.parseShellCommandChain("cd src && gcc -c -Wall")
	if result != nil {
		t.Errorf("Expected nil for invalid compiler chain, got %+v", result)
	}
}

func TestParseMakeLogSkipsCommentsAndBlankLines(t *testing.T) {
	p, err := NewParser(types.ParseOptions{BaseDir: "/project"})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	makeLog := `# This is a comment

gcc -c -Wall main.c -o main.o

# Another comment
make: Leaving directory '/project'`

	entries, err := p.ParseMakeLog(strings.NewReader(makeLog))
	if err != nil {
		t.Fatalf("ParseMakeLog failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry (skipping comments/blanks), got %d", len(entries))
	}
}

func TestIsSourceFile(t *testing.T) {
	p := newTestParser(t)

	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"C file", "main.c", true},
		{"Cpp file", "main.cpp", true},
		{"CC file", "main.cc", true},
		{"Cxx file", "main.cxx", true},
		{"C++ file", "main.c++", true},
		{"Assembly", "start.s", true},
		{"ASM", "start.asm", true},
		{"Uppercase ext", "MAIN.C", true},
		{"Header", "main.h", false},
		{"Object", "main.o", false},
		{"No ext", "Makefile", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.isSourceFile(tt.filename); got != tt.expected {
				t.Errorf("isSourceFile(%q) = %v, expected %v", tt.filename, got, tt.expected)
			}
		})
	}
}
