package parser

import (
	"strings"
	"testing"

	"github.com/gerryqd/yacd/types"
)

func newTestParser(t *testing.T) *Parser {
	t.Helper()
	p, err := NewParser(types.ParseOptions{})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	return p
}

func TestNewParser(t *testing.T) {
	p, err := NewParser(types.ParseOptions{})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	if p == nil || p.compilerRegex == nil || p.makeDirEnterRegex == nil || p.makeDirLeaveRegex == nil {
		t.Fatal("Parser or its regex fields should not be nil")
	}
}

func TestHandleDirectoryChange(t *testing.T) {
	tests := []struct {
		name          string
		lines         []string
		shouldHandle  []bool
		expectedStack []string
	}{
		{
			"Enter directory",
			[]string{"make: Entering directory '/home/user/project'"},
			[]bool{true},
			[]string{"/test", "/home/user/project"},
		},
		{
			"Nested enter",
			[]string{
				"make: Entering directory '/home/user/project'",
				"make[1]: Entering directory '/home/user/project/subdir'",
			},
			[]bool{true, true},
			[]string{"/test", "/home/user/project", "/home/user/project/subdir"},
		},
		{
			"Leave directory",
			[]string{
				"make: Entering directory '/home/user/project'",
				"make[1]: Entering directory '/home/user/project/subdir'",
				"make: Leaving directory '/home/user/project/subdir'",
			},
			[]bool{true, true, true},
			[]string{"/test", "/home/user/project"},
		},
		{
			"Regular command",
			[]string{"gcc -c main.c -o main.o"},
			[]bool{false},
			[]string{"/test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewParser(types.ParseOptions{BaseDir: "/test"})
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}
			p.dirStack = []string{"/test"}

			for i, line := range tt.lines {
				if p.handleDirectoryChange(line) != tt.shouldHandle[i] {
					t.Errorf("Line %d: handle result mismatch", i)
				}
			}

			if len(p.dirStack) != len(tt.expectedStack) {
				t.Fatalf("Stack length = %d, expected %d (got %v)", len(p.dirStack), len(tt.expectedStack), p.dirStack)
			}
			for i, expected := range tt.expectedStack {
				if p.dirStack[i] != expected {
					t.Errorf("Stack[%d] = %s, expected %s", i, p.dirStack[i], expected)
				}
			}
		})
	}
}

func TestSplitCommandLine(t *testing.T) {
	p := newTestParser(t)

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"Simple", "gcc -c main.c -o main.o", []string{"gcc", "-c", "main.c", "-o", "main.o"}},
		{"Double quoted", `gcc -DMESSAGE="Hello World" -c main.c`, []string{"gcc", "-DMESSAGE=Hello World", "-c", "main.c"}},
		{"Single quoted", `gcc -DMESSAGE='Hello World' -c main.c`, []string{"gcc", "-DMESSAGE=Hello World", "-c", "main.c"}},
		{"Escaped", `gcc -DMESSAGE=\"Hello\" -c main.c`, []string{"gcc", `-DMESSAGE="Hello"`, "-c", "main.c"}},
		{"ARM command", "arm-none-eabi-gcc -c -mcpu=cortex-m0 -DSTM32F030x6 -ICore/Inc main.c -o main.o",
			[]string{"arm-none-eabi-gcc", "-c", "-mcpu=cortex-m0", "-DSTM32F030x6", "-ICore/Inc", "main.c", "-o", "main.o"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.splitCommandLine(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("Length = %d, expected %d\nGot: %v\nExp: %v", len(result), len(tt.expected), result, tt.expected)
			}
			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("[%d] = %q, expected %q", i, result[i], exp)
				}
			}
		})
	}
}

func TestExtractFiles(t *testing.T) {
	p := newTestParser(t)

	tests := []struct {
		name   string
		args   []string
		source string
		output string
	}{
		{"C file", []string{"gcc", "-c", "main.c", "-o", "main.o"}, "main.c", "main.o"},
		{"No output", []string{"gcc", "-c", "main.c"}, "main.c", ""},
		{"Complex path", []string{"gcc", "-c", "src/utils/helper.c", "-o", "build/helper.o"}, "src/utils/helper.c", "build/helper.o"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, out := p.extractFiles(tt.args)
			if src != tt.source {
				t.Errorf("Source = %q, expected %q", src, tt.source)
			}
			if out != tt.output {
				t.Errorf("Output = %q, expected %q", out, tt.output)
			}
		})
	}
}

func TestParseCompileCommand(t *testing.T) {
	p, err := NewParser(types.ParseOptions{BaseDir: "/project"})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	p.dirStack = append(p.dirStack, "/project/build")

	tests := []struct {
		name          string
		line          string
		expectNil     bool
		expectCompiler string
		expectSource   string
		expectOutput   string
	}{
		{"GCC", "gcc -c -Wall main.c -o main.o", false, "gcc", "main.c", "main.o"},
		{"ARM", "arm-none-eabi-gcc -c -mcpu=cortex-m0 -mthumb main.c -o build/main.o", false, "arm-none-eabi-gcc", "main.c", "build/main.o"},
		{"Non-compile", "mkdir -p build", true, "", "", ""},
		{"Link (no source)", "gcc main.o util.o -o program", true, "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.parseCompileCommand(tt.line)
			if tt.expectNil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("Expected non-nil result")
			}
			if result.Compiler != tt.expectCompiler {
				t.Errorf("Compiler = %q, expected %q", result.Compiler, tt.expectCompiler)
			}
			if result.SourceFile != tt.expectSource {
				t.Errorf("SourceFile = %q, expected %q", result.SourceFile, tt.expectSource)
			}
			if result.OutputFile != tt.expectOutput {
				t.Errorf("OutputFile = %q, expected %q", result.OutputFile, tt.expectOutput)
			}
		})
	}
}

func TestParseMakeLog(t *testing.T) {
	p, err := NewParser(types.ParseOptions{BaseDir: "/project"})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	makeLog := `make: Entering directory '/home/user/project'
mkdir build
gcc -c -Wall main.c -o main.o
gcc -c -Wall util.c -o util.o
gcc main.o util.o -o program
make: Leaving directory '/home/user/project'`

	entries, err := p.ParseMakeLog(strings.NewReader(makeLog))
	if err != nil {
		t.Fatalf("ParseMakeLog failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}
	if entries[0].SourceFile != "main.c" {
		t.Errorf("First entry source = %q, expected main.c", entries[0].SourceFile)
	}
	if entries[0].WorkingDir != "/home/user/project" {
		t.Errorf("First entry dir = %q, expected /home/user/project", entries[0].WorkingDir)
	}
	if entries[1].SourceFile != "util.c" {
		t.Errorf("Second entry source = %q, expected util.c", entries[1].SourceFile)
	}
}

func TestParseMakeLogWithComments(t *testing.T) {
	p, err := NewParser(types.ParseOptions{BaseDir: "/project"})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	makeLog := `# comment
make: Entering directory '/home/user/project'
# another comment
gcc -c -Wall main.c -o main.o
gcc -c -Wall util.c -o util.o
# trailing
make: Leaving directory '/home/user/project'`

	entries, err := p.ParseMakeLog(strings.NewReader(makeLog))
	if err != nil {
		t.Fatalf("ParseMakeLog failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}
}

func TestMakeCDirectoryHandling(t *testing.T) {
	p, err := NewParser(types.ParseOptions{})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	makeLog := `make: Entering directory '/project/root'
make -C subdir1 all
make[1]: Entering directory '/project/root/subdir1'
gcc -c -Wall main1.c -o main1.o
gcc -c -Wall util1.c -o util1.o
make[1]: Leaving directory '/project/root/subdir1'
make -C subdir2 all
make[2]: Entering directory '/project/root/subdir2'
gcc -c -Wall main2.c -o main2.o
gcc -c -Wall util2.c -o util2.o
make[2]: Leaving directory '/project/root/subdir2'
make: Leaving directory '/project/root'`

	entries, err := p.ParseMakeLog(strings.NewReader(makeLog))
	if err != nil {
		t.Fatalf("ParseMakeLog failed: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("Expected 4 entries, got %d", len(entries))
	}

	expectedDirs := []string{
		"/project/root/subdir1",
		"/project/root/subdir1",
		"/project/root/subdir2",
		"/project/root/subdir2",
	}
	expectedSources := []string{"main1.c", "util1.c", "main2.c", "util2.c"}

	for i, entry := range entries {
		if entry.WorkingDir != expectedDirs[i] {
			t.Errorf("Entry %d dir = %q, expected %q", i, entry.WorkingDir, expectedDirs[i])
		}
		if entry.SourceFile != expectedSources[i] {
			t.Errorf("Entry %d source = %q, expected %q", i, entry.SourceFile, expectedSources[i])
		}
	}
}

func TestRemoveRedirectionOperators(t *testing.T) {
	p := newTestParser(t)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"2>&1", "gcc main.c -o main 2>&1", "gcc main.c -o main"},
		{"stdout redirect", "gcc -c file.c -o file.o >output.log", "gcc -c file.c -o file.o"},
		{"stderr redirect", "arm-none-eabi-gcc -c src.c -o obj.o 2>error.log", "arm-none-eabi-gcc -c src.c -o obj.o"},
		{"No redirect", "gcc normal.c -o normal", "gcc normal.c -o normal"},
		{"Multiple", "gcc -c file.c -o file.o >output.log 2>&1", "gcc -c file.c -o file.o"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := p.removeRedirectionOperators(tt.input); result != tt.expected {
				t.Errorf("Got %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestShellCommandChain(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		dir      string
		source   string
		output   string
	}{
		{"cd and gcc", "cd src && gcc -c main.c -o main.o", "/test/src", "main.c", "main.o"},
		{"cd with flags", "cd lib && gcc -c utils.c -o utils.o -I../include -Wall", "/test/lib", "utils.c", "utils.o"},
		{"no cd", "gcc -c normal.c -o normal.o", "/test", "normal.c", "normal.o"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewParser(types.ParseOptions{})
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}
			p.dirStack = []string{"/test"}

			result := p.parseCompileCommand(tt.line)
			if result == nil {
				t.Fatal("Expected non-nil result")
			}
			if result.WorkingDir != tt.dir {
				t.Errorf("WorkingDir = %q, expected %q", result.WorkingDir, tt.dir)
			}
			if result.SourceFile != tt.source {
				t.Errorf("SourceFile = %q, expected %q", result.SourceFile, tt.source)
			}
			if result.OutputFile != tt.output {
				t.Errorf("OutputFile = %q, expected %q", result.OutputFile, tt.output)
			}
		})
	}
}

func TestShellCommandChainNotCompile(t *testing.T) {
	p, err := NewParser(types.ParseOptions{})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	p.dirStack = []string{"/test"}

	result := p.parseCompileCommand("cd src && echo hello")
	if result != nil {
		t.Errorf("Expected nil for non-compile cd chain, got %+v", result)
	}
}

func TestParseMakeLogWithEchoCommands(t *testing.T) {
	p, err := NewParser(types.ParseOptions{BaseDir: "/project"})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	makeLog := `make: Entering directory '/home/user/project'
echo "Compiling main.c"
gcc -c -Wall main.c -o main.o
echo "Compiling util.c"
gcc -c -Wall util.c -o util.o
make: Leaving directory '/home/user/project'`

	entries, err := p.ParseMakeLog(strings.NewReader(makeLog))
	if err != nil {
		t.Fatalf("ParseMakeLog failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}
}

func TestFindCompilerStartIndex(t *testing.T) {
	p := newTestParser(t)

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"Simple gcc", "gcc -c main.c -o main.o", 0},
		{"With prefix", "/path/to/check /path/to/check -p arm-linux-gnu-gcc -c main.c -o main.o", 33},
		{"Multiple prefixes", "tool1 tool2 gcc -c main.c -o main.o", 12},
		{"No compiler", "mkdir build && cd build", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := p.findCompilerStartIndex(tt.input); result != tt.expected {
				t.Errorf("Got %d, expected %d for %q", result, tt.expected, tt.input)
			}
		})
	}
}

func TestParseCompileCommandWithPrefix(t *testing.T) {
	p, err := NewParser(types.ParseOptions{BaseDir: "/project"})
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	p.dirStack = append(p.dirStack, "/project/build")

	tests := []struct {
		name          string
		line          string
		expectCompiler string
		expectSource   string
	}{
		{"check prefix", "path/to/check path/to/check -p gcc -DTEST=1 -c main.c -o main.o", "gcc", "main.c"},
		{"complex prefix", "/tools/preprocessor /tools/preprocessor -flags arm-linux-gnueabi-gcc -DARCH=arm -Wall -c file.c -o file.o", "arm-linux-gnueabi-gcc", "file.c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.parseCompileCommand(tt.line)
			if result == nil {
				t.Fatal("Expected non-nil result")
			}
			if result.Compiler != tt.expectCompiler {
				t.Errorf("Compiler = %q, expected %q", result.Compiler, tt.expectCompiler)
			}
			if result.SourceFile != tt.expectSource {
				t.Errorf("SourceFile = %q, expected %q", result.SourceFile, tt.expectSource)
			}
		})
	}
}
