package parser

import (
	"strings"
	"testing"

	"github.com/gerryqd/yacd/types"
)

// Regression tests for issues found during code review.

func TestParseRedirectionWithSpaceAfterOperator(t *testing.T) {
	// "> /dev/null" (space after operator) must be stripped along with the
	// redirect target, and must not leak into the parsed arguments.
	p, err := NewParser(types.ParseOptions{})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	line := "gcc -c foo.c -o foo.o > /dev/null 2>&1"
	entry := p.parseCompileCommand(line)
	if entry == nil {
		t.Fatal("Expected entry to be parsed, got nil")
	}
	if entry.SourceFile != "foo.c" {
		t.Errorf("SourceFile = %q, expected foo.c", entry.SourceFile)
	}
	for _, arg := range entry.Args {
		if arg == ">" || arg == "/dev/null" {
			t.Errorf("Redirection token leaked into args: %v", entry.Args)
		}
	}

	// Variants with space after the operator.
	for _, variant := range []string{
		"gcc -c foo.c 2> err.log",
		"gcc -c foo.c 2>> err.log",
		"gcc -c foo.c >> out.log",
		"gcc -c foo.c < in.txt",
		"gcc -c foo.c 2>& 1",
		"gcc -c foo.c >> out.log 2>> err.log",
		"gcc -c foo.c 2> err.log > out.log",
	} {
		if e := p.parseCompileCommand(variant); e == nil {
			t.Errorf("Expected parse success for %q", variant)
			continue
		} else {
			// The redirect target must not leak into the arguments.
			joined := strings.Join(e.Args, " ")
			for _, leaked := range []string{"out.log", "err.log", "in.txt", ">", "2>", ">>", "<"} {
				if strings.Contains(joined, leaked) {
					t.Errorf("%q leaked %q into args: %v", variant, leaked, e.Args)
					break
				}
			}
		}
	}
}

func TestParseRedirectionInsideQuotes(t *testing.T) {
	// A ">" inside a quoted argument is part of the argument, not a redirect.
	p, err := NewParser(types.ParseOptions{})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	line := `gcc -DFOO="a > b" -c foo.c`
	entry := p.parseCompileCommand(line)
	if entry == nil {
		t.Fatal("Expected entry to be parsed, got nil")
	}
	if len(entry.Args) != 4 {
		t.Fatalf("Expected 4 args, got %v", entry.Args)
	}
	if entry.Args[1] != "-DFOO=a > b" {
		t.Errorf("Args[1] = %q, expected %q", entry.Args[1], "-DFOO=a > b")
	}

	// Single-quoted variant.
	line2 := `gcc -DFOO='x > y' -c foo.c`
	entry2 := p.parseCompileCommand(line2)
	if entry2 == nil {
		t.Fatal("Expected entry to be parsed, got nil")
	}
	if entry2.Args[1] != "-DFOO=x > y" {
		t.Errorf("Args[1] = %q, expected %q", entry2.Args[1], "-DFOO=x > y")
	}
}

func TestParseEchoLineWithMultipleSpaces(t *testing.T) {
	p, err := NewParser(types.ParseOptions{})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// echo followed by multiple spaces must still be recognized as an echo
	// line, not parsed as a compile command.
	for _, line := range []string{
		"echo   gcc -c foo.c",
		"echo gcc -c foo.c",
		"@echo gcc -c foo.c",
	} {
		if entry := p.parseCompileCommand(line); entry != nil {
			t.Errorf("Expected nil for echo line %q, got %+v", line, entry)
		}
	}
}

func TestParseBacktickEchoWithDashN(t *testing.T) {
	// `echo -n /path` must extract /path, not the -n option.
	p, err := NewParser(types.ParseOptions{})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	p.dirStack = []string{"/proj"}

	line := "gcc -c `echo -n /proj`/foo.c"
	entry := p.parseCompileCommand(line)
	if entry == nil {
		t.Fatal("Expected entry to be parsed, got nil")
	}
	if entry.SourceFile != "/proj/foo.c" {
		t.Errorf("SourceFile = %q, expected %q", entry.SourceFile, "/proj/foo.c")
	}

	// echo -e variant.
	line2 := "gcc -c `echo -e /proj`/foo.c"
	entry2 := p.parseCompileCommand(line2)
	if entry2 == nil {
		t.Fatal("Expected entry to be parsed, got nil")
	}
	if entry2.SourceFile != "/proj/foo.c" {
		t.Errorf("SourceFile = %q, expected %q", entry2.SourceFile, "/proj/foo.c")
	}

	// Bare "echo -n" with no path must not produce a bogus path.
	line3 := "gcc -c `echo -n`/foo.c"
	entry3 := p.parseCompileCommand(line3)
	if entry3 == nil {
		t.Fatal("Expected entry to be parsed, got nil")
	}
	if entry3.SourceFile != "/foo.c" {
		t.Errorf("SourceFile = %q, expected %q", entry3.SourceFile, "/foo.c")
	}
}

func TestParseWindowsDrivePath(t *testing.T) {
	p, err := NewParser(types.ParseOptions{})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	p.dirStack = []string{`C:\proj`}

	line := `C:\mingw\bin\gcc.exe -c foo.c`
	entry := p.parseCompileCommand(line)
	if entry == nil {
		t.Fatal("Expected entry to be parsed, got nil")
	}
	if entry.Compiler != `C:\mingw\bin\gcc.exe` {
		t.Errorf("Compiler = %q, expected %q", entry.Compiler, `C:\mingw\bin\gcc.exe`)
	}

	// POSIX-style escaping must still work.
	line2 := `gcc -c my\ file.c`
	entry2 := p.parseCompileCommand(line2)
	if entry2 == nil {
		t.Fatal("Expected entry to be parsed, got nil")
	}
	if entry2.SourceFile != "my file.c" {
		t.Errorf("SourceFile = %q, expected %q", entry2.SourceFile, "my file.c")
	}
}

func TestParseMultipleSourceFilesSkipped(t *testing.T) {
	p, err := NewParser(types.ParseOptions{})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// A line compiling several sources cannot be represented by a single
	// compilation entry, so it must be skipped instead of producing a
	// misleading entry for the last source only.
	for _, line := range []string{
		"gcc -c a.c b.c",
		"gcc -c a.c b.c -o app",
		"gcc a.c b.c -o app",
	} {
		if entry := p.parseCompileCommand(line); entry != nil {
			t.Errorf("Expected nil for multi-source line %q, got %+v", line, entry)
		}
	}

	// Single-source lines must still parse.
	entry := p.parseCompileCommand("gcc -c a.c -o a.o")
	if entry == nil {
		t.Fatal("Expected entry for single-source line")
	}
	if entry.SourceFile != "a.c" {
		t.Errorf("SourceFile = %q, expected a.c", entry.SourceFile)
	}
}

func TestParseRedirectionEndToEnd(t *testing.T) {
	// Full pipeline: make log line with redirects produces a clean entry.
	p, err := NewParser(types.ParseOptions{BaseDir: "/proj"})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	makeLog := `make: Entering directory '/proj'
gcc -Wall -c src/main.c -o obj/main.o > /dev/null 2>&1
make: Leaving directory '/proj'`

	entries, err := p.ParseMakeLog(strings.NewReader(makeLog))
	if err != nil {
		t.Fatalf("ParseMakeLog failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
	if entries[0].SourceFile != "src/main.c" {
		t.Errorf("SourceFile = %q, expected src/main.c", entries[0].SourceFile)
	}
	joined := strings.Join(entries[0].Args, " ")
	if strings.Contains(joined, ">") || strings.Contains(joined, "/dev/null") {
		t.Errorf("Redirection leaked into args: %q", joined)
	}
}
