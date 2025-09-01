package parser

import (
	"strings"
	"testing"

	"github.com/gerryqd/yacd/types"
)

func TestNewParser(t *testing.T) {
	options := types.ParseOptions{
		InputFile:  "test.log",
		OutputFile: "compile_commands.json",
		Verbose:    false,
	}

	parser, err := NewParser(options)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	if parser == nil {
		t.Fatal("Parser should not be nil")
	}

	if parser.compilerRegex == nil {
		t.Fatal("Compiler regex should not be nil")
	}

	if parser.makeDirEnterRegex == nil {
		t.Fatal("Make directory enter regex should not be nil")
	}

	if parser.makeDirLeaveRegex == nil {
		t.Fatal("Make directory leave regex should not be nil")
	}
}

func TestHandleDirectoryChange(t *testing.T) {
	tests := []struct {
		name          string
		lines         []string // Multiple lines to execute in sequence
		shouldHandle  []bool   // Expected results for each line
		expectedStack []string // Final expected stack
	}{
		{
			name:          "Enter directory",
			lines:         []string{"make: Entering directory '/home/user/project'"},
			shouldHandle:  []bool{true},
			expectedStack: []string{"/test", "/home/user/project"},
		},
		{
			name: "Enter directory (with number)",
			lines: []string{
				"make: Entering directory '/home/user/project'",
				"make[1]: Entering directory '/home/user/project/subdir'",
			},
			shouldHandle:  []bool{true, true},
			expectedStack: []string{"/test", "/home/user/project", "/home/user/project/subdir"},
		},
		{
			name: "Leave directory",
			lines: []string{
				"make: Entering directory '/home/user/project'",
				"make[1]: Entering directory '/home/user/project/subdir'",
				"make: Leaving directory '/home/user/project/subdir'",
			},
			shouldHandle:  []bool{true, true, true},
			expectedStack: []string{"/test", "/home/user/project"},
		},
		{
			name: "Leave directory (with number)",
			lines: []string{
				"make: Entering directory '/home/user/project'",
				"make[1]: Leaving directory '/home/user/project'",
			},
			shouldHandle:  []bool{true, true},
			expectedStack: []string{"/test"},
		},
		{
			name:          "Regular command",
			lines:         []string{"gcc -c main.c -o main.o"},
			shouldHandle:  []bool{false},
			expectedStack: []string{"/test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh parser for each test
			options := types.ParseOptions{BaseDir: "/test"}
			parser, err := NewParser(options)
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			// Initialize the directory stack with BaseDir
			if options.BaseDir != "" {
				parser.dirStack = append(parser.dirStack, options.BaseDir)
			}

			// Process all lines in sequence
			for i, line := range tt.lines {
				handled := parser.handleDirectoryChange(line)
				if handled != tt.shouldHandle[i] {
					t.Errorf("handleDirectoryChange() line %d = %v, expected %v", i, handled, tt.shouldHandle[i])
				}
			}

			// Check final directory stack
			if len(parser.dirStack) != len(tt.expectedStack) {
				t.Errorf("Directory stack length = %d, expected %d", len(parser.dirStack), len(tt.expectedStack))
				t.Errorf("Actual stack: %v", parser.dirStack)
				t.Errorf("Expected stack: %v", tt.expectedStack)
				return
			}

			for i, expected := range tt.expectedStack {
				if parser.dirStack[i] != expected {
					t.Errorf("Directory stack[%d] = %s, expected %s", i, parser.dirStack[i], expected)
				}
			}
		})
	}
}

func TestSplitCommandLine(t *testing.T) {
	options := types.ParseOptions{}
	parser, err := NewParser(options)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Simple command",
			input:    "gcc -c main.c -o main.o",
			expected: []string{"gcc", "-c", "main.c", "-o", "main.o"},
		},
		{
			name:     "Command with quoted arguments",
			input:    "gcc -DMESSAGE=\"Hello World\" -c main.c",
			expected: []string{"gcc", "-DMESSAGE=Hello World", "-c", "main.c"},
		},
		{
			name:     "Command with single quoted arguments",
			input:    "gcc -DMESSAGE='Hello World' -c main.c",
			expected: []string{"gcc", "-DMESSAGE=Hello World", "-c", "main.c"},
		},
		{
			name:     "Command with escape characters",
			input:    "gcc -DMESSAGE=\\\"Hello\\\" -c main.c",
			expected: []string{"gcc", "-DMESSAGE=\"Hello\"", "-c", "main.c"},
		},
		{
			name:     "Complex ARM compilation command",
			input:    "arm-none-eabi-gcc -c -mcpu=cortex-m0 -DSTM32F030x6 -ICore/Inc main.c -o main.o",
			expected: []string{"arm-none-eabi-gcc", "-c", "-mcpu=cortex-m0", "-DSTM32F030x6", "-ICore/Inc", "main.c", "-o", "main.o"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.splitCommandLine(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitCommandLine() returned length %d, expected %d", len(result), len(tt.expected))
				t.Errorf("Returned: %v", result)
				t.Errorf("Expected: %v", tt.expected)
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("splitCommandLine()[%d] = %s, expected %s", i, result[i], expected)
				}
			}
		})
	}
}

func TestExtractFiles(t *testing.T) {
	options := types.ParseOptions{}
	parser, err := NewParser(options)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	tests := []struct {
		name           string
		args           []string
		expectedSource string
		expectedOutput string
	}{
		{
			name:           "Simple C file compilation",
			args:           []string{"gcc", "-c", "main.c", "-o", "main.o"},
			expectedSource: "main.c",
			expectedOutput: "main.o",
		},
		{
			name:           "C++ file compilation",
			args:           []string{"g++", "-c", "main.cpp", "-o", "main.o"},
			expectedSource: "main.cpp",
			expectedOutput: "main.o",
		},
		{
			name:           "Assembly file compilation",
			args:           []string{"gcc", "-c", "startup.s", "-o", "startup.o"},
			expectedSource: "startup.s",
			expectedOutput: "startup.o",
		},
		{
			name:           "No output file",
			args:           []string{"gcc", "-c", "main.c"},
			expectedSource: "main.c",
			expectedOutput: "",
		},
		{
			name:           "Complex path",
			args:           []string{"gcc", "-c", "src/utils/helper.c", "-o", "build/helper.o"},
			expectedSource: "src/utils/helper.c",
			expectedOutput: "build/helper.o",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceFile, outputFile := parser.extractFiles(tt.args)
			if sourceFile != tt.expectedSource {
				t.Errorf("extractFiles() sourceFile = %s, expected %s", sourceFile, tt.expectedSource)
			}
			if outputFile != tt.expectedOutput {
				t.Errorf("extractFiles() outputFile = %s, expected %s", outputFile, tt.expectedOutput)
			}
		})
	}
}

func TestIsSourceFile(t *testing.T) {
	options := types.ParseOptions{}
	parser, err := NewParser(options)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	tests := []struct {
		filename string
		expected bool
	}{
		{"main.c", true},
		{"main.cpp", true},
		{"main.cc", true},
		{"main.cxx", true},
		{"main.c++", true},
		{"startup.s", true},
		{"startup.S", true},
		{"assembly.asm", true},
		{"main.h", false},
		{"main.hpp", false},
		{"main.o", false},
		{"libtest.a", false},
		{"program", false},
		{"Makefile", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := parser.isSourceFile(tt.filename)
			if result != tt.expected {
				t.Errorf("isSourceFile(%s) = %v, expected %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestParseCompileCommand(t *testing.T) {
	options := types.ParseOptions{BaseDir: "/project"}
	parser, err := NewParser(options)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Set working directory
	parser.dirStack = append(parser.dirStack, "/project/build")

	tests := []struct {
		name     string
		line     string
		expected *types.MakeLogEntry
	}{
		{
			name: "GCC compilation command",
			line: "gcc -c -Wall main.c -o main.o",
			expected: &types.MakeLogEntry{
				WorkingDir: "/project/build",
				Compiler:   "gcc",
				Args:       []string{"gcc", "-c", "-Wall", "main.c", "-o", "main.o"},
				SourceFile: "main.c",
				OutputFile: "main.o",
			},
		},
		{
			name: "ARM compilation command",
			line: "arm-none-eabi-gcc -c -mcpu=cortex-m0 -mthumb -DNDEBUG -DUSE_HAL_DRIVER -DSTM32F030x6 -ICore/Inc main.c -o build/main.o",
			expected: &types.MakeLogEntry{
				WorkingDir: "/project/build",
				Compiler:   "arm-none-eabi-gcc",
				Args:       []string{"arm-none-eabi-gcc", "-c", "-mcpu=cortex-m0", "-mthumb", "-DNDEBUG", "-DUSE_HAL_DRIVER", "-DSTM32F030x6", "-ICore/Inc", "main.c", "-o", "build/main.o"},
				SourceFile: "main.c",
				OutputFile: "build/main.o",
			},
		},
		{
			name:     "Non-compilation command",
			line:     "mkdir -p build",
			expected: nil,
		},
		{
			name:     "Linking command (no source file)",
			line:     "gcc main.o util.o -o program",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.parseCompileCommand(tt.line)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("parseCompileCommand() = %v, expected nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("parseCompileCommand() = nil, expected non-nil")
			}

			if result.WorkingDir != tt.expected.WorkingDir {
				t.Errorf("WorkingDir = %s, expected %s", result.WorkingDir, tt.expected.WorkingDir)
			}

			if result.Compiler != tt.expected.Compiler {
				t.Errorf("Compiler = %s, expected %s", result.Compiler, tt.expected.Compiler)
			}

			if result.SourceFile != tt.expected.SourceFile {
				t.Errorf("SourceFile = %s, expected %s", result.SourceFile, tt.expected.SourceFile)
			}

			if result.OutputFile != tt.expected.OutputFile {
				t.Errorf("OutputFile = %s, expected %s", result.OutputFile, tt.expected.OutputFile)
			}

			if len(result.Args) != len(tt.expected.Args) {
				t.Errorf("Args length = %d, expected %d", len(result.Args), len(tt.expected.Args))
				return
			}

			for i, arg := range tt.expected.Args {
				if result.Args[i] != arg {
					t.Errorf("Args[%d] = %s, expected %s", i, result.Args[i], arg)
				}
			}
		})
	}
}

func TestParseMakeLog(t *testing.T) {
	options := types.ParseOptions{BaseDir: "/project"}
	parser, err := NewParser(options)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Simulate real make log
	makeLog := `make: Entering directory '/home/user/project'
mkdir build
gcc -c -Wall main.c -o main.o
gcc -c -Wall util.c -o util.o
gcc main.o util.o -o program
make: Leaving directory '/home/user/project'`

	reader := strings.NewReader(makeLog)
	entries, err := parser.ParseMakeLog(reader)
	if err != nil {
		t.Fatalf("ParseMakeLog() failed: %v", err)
	}

	// Should parse 2 compilation entries
	expectedCount := 2
	if len(entries) != expectedCount {
		t.Errorf("Parsed %d entries, expected %d", len(entries), expectedCount)
	}

	// Check first entry
	if len(entries) > 0 {
		entry := entries[0]
		if entry.SourceFile != "main.c" {
			t.Errorf("First entry source file = %s, expected main.c", entry.SourceFile)
		}
		if entry.WorkingDir != "/home/user/project" {
			t.Errorf("First entry working directory = %s, expected /home/user/project", entry.WorkingDir)
		}
	}

	// Check second entry
	if len(entries) > 1 {
		entry := entries[1]
		if entry.SourceFile != "util.c" {
			t.Errorf("Second entry source file = %s, expected util.c", entry.SourceFile)
		}
	}
}

func TestParseMakeLogWithComments(t *testing.T) {
	options := types.ParseOptions{BaseDir: "/project"}
	parser, err := NewParser(options)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Simulate make log with comment lines starting with '#'
	makeLog := `# This is a comment line
make: Entering directory '/home/user/project'
# Another comment
mkdir build
# Comment before compilation
gcc -c -Wall main.c -o main.o
# More comments
gcc -c -Wall util.c -o util.o
# Final comment
gcc main.o util.o -o program
make: Leaving directory '/home/user/project'
# End comment`

	reader := strings.NewReader(makeLog)
	entries, err := parser.ParseMakeLog(reader)
	if err != nil {
		t.Fatalf("ParseMakeLog() failed: %v", err)
	}

	// Should parse 2 compilation entries (comments should be ignored)
	expectedCount := 2
	if len(entries) != expectedCount {
		t.Errorf("Parsed %d entries, expected %d", len(entries), expectedCount)
	}

	// Check first entry
	if len(entries) > 0 {
		entry := entries[0]
		if entry.SourceFile != "main.c" {
			t.Errorf("First entry source file = %s, expected main.c", entry.SourceFile)
		}
		if entry.WorkingDir != "/home/user/project" {
			t.Errorf("First entry working directory = %s, expected /home/user/project", entry.WorkingDir)
		}
	}

	// Check second entry
	if len(entries) > 1 {
		entry := entries[1]
		if entry.SourceFile != "util.c" {
			t.Errorf("Second entry source file = %s, expected util.c", entry.SourceFile)
		}
	}
}

func TestMakeCDirectoryHandling(t *testing.T) {
	options := types.ParseOptions{}
	parser, err := NewParser(options)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Simulate make log with make -C commands
	makeLog := `make: Entering directory '/project/root'
make -C subdir1 all
make[1]: Entering directory '/project/root/subdir1'
gcc -c -Wall main1.c -o main1.o
gcc -c -Wall util1.c -o util1.o
gcc main1.o util1.o -o program1
make[1]: Leaving directory '/project/root/subdir1'
make -C subdir2 all
make[2]: Entering directory '/project/root/subdir2'
gcc -c -Wall main2.c -o main2.o
gcc -c -Wall util2.c -o util2.o
gcc main2.o util2.o -o program2
make[2]: Leaving directory '/project/root/subdir2'
make: Leaving directory '/project/root'`

	reader := strings.NewReader(makeLog)
	entries, err := parser.ParseMakeLog(reader)
	if err != nil {
		t.Fatalf("ParseMakeLog() failed: %v", err)
	}

	// Should parse 4 compilation entries (2 from each subdirectory)
	expectedCount := 4
	if len(entries) != expectedCount {
		t.Errorf("Parsed %d entries, expected %d", len(entries), expectedCount)
	}

	// Check first entry from subdir1
	if len(entries) > 0 {
		entry := entries[0]
		if entry.SourceFile != "main1.c" {
			t.Errorf("First entry source file = %s, expected main1.c", entry.SourceFile)
		}
		if entry.WorkingDir != "/project/root/subdir1" {
			t.Errorf("First entry working directory = %s, expected /project/root/subdir1", entry.WorkingDir)
		}
	}

	// Check second entry from subdir1
	if len(entries) > 1 {
		entry := entries[1]
		if entry.SourceFile != "util1.c" {
			t.Errorf("Second entry source file = %s, expected util1.c", entry.SourceFile)
		}
		if entry.WorkingDir != "/project/root/subdir1" {
			t.Errorf("Second entry working directory = %s, expected /project/root/subdir1", entry.WorkingDir)
		}
	}

	// Check third entry from subdir2
	if len(entries) > 2 {
		entry := entries[2]
		if entry.SourceFile != "main2.c" {
			t.Errorf("Third entry source file = %s, expected main2.c", entry.SourceFile)
		}
		if entry.WorkingDir != "/project/root/subdir2" {
			t.Errorf("Third entry working directory = %s, expected /project/root/subdir2", entry.WorkingDir)
		}
	}

	// Check fourth entry from subdir2
	if len(entries) > 3 {
		entry := entries[3]
		if entry.SourceFile != "util2.c" {
			t.Errorf("Fourth entry source file = %s, expected util2.c", entry.SourceFile)
		}
		if entry.WorkingDir != "/project/root/subdir2" {
			t.Errorf("Fourth entry working directory = %s, expected /project/root/subdir2", entry.WorkingDir)
		}
	}
}

func TestRemoveRedirectionOperators(t *testing.T) {
	options := types.ParseOptions{}
	parser, err := NewParser(options)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Remove 2>&1",
			input:    "gcc main.c -o main 2>&1",
			expected: "gcc main.c -o main",
		},
		{
			name:     "Remove stdout redirection",
			input:    "gcc -c file.c -o file.o >output.log",
			expected: "gcc -c file.c -o file.o",
		},
		{
			name:     "Remove append redirection",
			input:    "gcc -Wall test.c -o test >>build.log",
			expected: "gcc -Wall test.c -o test",
		},
		{
			name:     "Remove stderr redirection",
			input:    "arm-none-eabi-gcc -c src.c -o obj.o 2>error.log",
			expected: "arm-none-eabi-gcc -c src.c -o obj.o",
		},
		{
			name:     "Remove stdin redirection",
			input:    "clang++ hello.cpp -o hello <input.txt",
			expected: "clang++ hello.cpp -o hello",
		},
		{
			name:     "Remove multiple redirections",
			input:    "gcc -c file.c -o file.o >output.log 2>&1",
			expected: "gcc -c file.c -o file.o",
		},
		{
			name:     "No redirection to remove",
			input:    "gcc normal.c -o normal",
			expected: "gcc normal.c -o normal",
		},
		{
			name:     "Remove numbered fd redirection",
			input:    "gcc test.c -o test 3>debug.log",
			expected: "gcc test.c -o test",
		},
		{
			name:     "Remove fd to fd redirection",
			input:    "gcc main.c -o main 3>&2",
			expected: "gcc main.c -o main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.removeRedirectionOperators(tt.input)
			if result != tt.expected {
				t.Errorf("removeRedirectionOperators() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

// TestShellCommandChain tests parsing of shell command chains like "cd dir && gcc ..."
func TestShellCommandChain(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected *types.MakeLogEntry
		should   bool
	}{
		{
			name: "Simple cd and gcc command",
			line: "cd src && gcc -c main.c -o main.o",
			expected: &types.MakeLogEntry{
				WorkingDir: "/test/src",
				Compiler:   "gcc",
				Args:       []string{"gcc", "-c", "main.c", "-o", "main.o"},
				SourceFile: "main.c",
				OutputFile: "main.o",
			},
			should: true,
		},
		{
			name: "cd with complex gcc command",
			line: "cd lib && gcc -c utils.c -o utils.o -I../include -Wall",
			expected: &types.MakeLogEntry{
				WorkingDir: "/test/lib",
				Compiler:   "gcc",
				Args:       []string{"gcc", "-c", "utils.c", "-o", "utils.o", "-I../include", "-Wall"},
				SourceFile: "utils.c",
				OutputFile: "utils.o",
			},
			should: true,
		},
		{
			name: "Regular gcc command without cd",
			line: "gcc -c normal.c -o normal.o",
			expected: &types.MakeLogEntry{
				WorkingDir: "/test",
				Compiler:   "gcc",
				Args:       []string{"gcc", "-c", "normal.c", "-o", "normal.o"},
				SourceFile: "normal.c",
				OutputFile: "normal.o",
			},
			should: true,
		},
		{
			name:     "Not a compilation command",
			line:     "cd src && echo hello",
			expected: nil,
			should:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := types.ParseOptions{
				InputFile:  "test.log",
				OutputFile: "compile_commands.json",
				Verbose:    false,
			}

			parser, err := NewParser(options)
			if err != nil {
				t.Fatalf("Failed to create parser: %v", err)
			}

			// Set up directory stack
			parser.dirStack = []string{"/test"}
			parser.directoryHistory = []string{"/test"}

			result := parser.parseCompileCommand(test.line)

			if test.should {
				if result == nil {
					t.Errorf("Expected to parse command, but got nil")
					return
				}

				if result.WorkingDir != test.expected.WorkingDir {
					// On Windows, paths use backslashes, so we need to normalize for comparison
					expectedNormalized := strings.ReplaceAll(test.expected.WorkingDir, "/", "\\")
					if result.WorkingDir != expectedNormalized {
						t.Errorf("WorkingDir = %s, expected %s (or %s)", result.WorkingDir, test.expected.WorkingDir, expectedNormalized)
					}
				}

				if result.Compiler != test.expected.Compiler {
					t.Errorf("Compiler = %s, expected %s", result.Compiler, test.expected.Compiler)
				}

				if len(result.Args) != len(test.expected.Args) {
					t.Errorf("Args length = %d, expected %d", len(result.Args), len(test.expected.Args))
				} else {
					for i, arg := range result.Args {
						if arg != test.expected.Args[i] {
							t.Errorf("Args[%d] = %s, expected %s", i, arg, test.expected.Args[i])
						}
					}
				}

				if result.SourceFile != test.expected.SourceFile {
					// On Windows, paths use backslashes, so we need to normalize for comparison
					expectedNormalized := strings.ReplaceAll(test.expected.SourceFile, "/", "\\")
					if result.SourceFile != expectedNormalized {
						t.Errorf("SourceFile = %s, expected %s (or %s)", result.SourceFile, test.expected.SourceFile, expectedNormalized)
					}
				}

				if result.OutputFile != test.expected.OutputFile {
					// On Windows, paths use backslashes, so we need to normalize for comparison
					expectedNormalized := strings.ReplaceAll(test.expected.OutputFile, "/", "\\")
					if result.OutputFile != expectedNormalized {
						t.Errorf("OutputFile = %s, expected %s (or %s)", result.OutputFile, test.expected.OutputFile, expectedNormalized)
					}
				}
			} else {
				if result != nil {
					t.Errorf("Expected nil result, but got: %+v", result)
				}
			}
		})
	}
}

func TestParseMakeLogWithEchoCommands(t *testing.T) {
	options := types.ParseOptions{BaseDir: "/project"}
	parser, err := NewParser(options)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Simulate make log with echo commands containing "Compiling" messages
	makeLog := `make: Entering directory '/home/user/project'
echo "/bin/date +%T Compiling /usr/bin/tput -Txterm smso/home/user/project/main.c/usr/bin/tput -Txterm sgr0 2>/dev/null"
gcc -c -Wall main.c -o main.o
echo "/bin/date +%T Compiling /usr/bin/tput -Txterm smso/home/user/project/util.c/usr/bin/tput -Txterm sgr0 2>/dev/null"
gcc -c -Wall util.c -o util.o
gcc main.o util.o -o program
make: Leaving directory '/home/user/project'`

	reader := strings.NewReader(makeLog)
	entries, err := parser.ParseMakeLog(reader)
	if err != nil {
		t.Fatalf("ParseMakeLog() failed: %v", err)
	}

	// Should parse 2 compilation entries (echo commands should be ignored)
	expectedCount := 2
	if len(entries) != expectedCount {
		t.Errorf("Parsed %d entries, expected %d", len(entries), expectedCount)
	}

	// Check first entry
	if len(entries) > 0 {
		entry := entries[0]
		if entry.SourceFile != "main.c" {
			t.Errorf("First entry source file = %s, expected main.c", entry.SourceFile)
		}
		if entry.WorkingDir != "/home/user/project" {
			t.Errorf("First entry working directory = %s, expected /home/user/project", entry.WorkingDir)
		}
	}

	// Check second entry
	if len(entries) > 1 {
		entry := entries[1]
		if entry.SourceFile != "util.c" {
			t.Errorf("Second entry source file = %s, expected util.c", entry.SourceFile)
		}
	}
}

func TestFindCompilerStartIndex(t *testing.T) {
	options := types.ParseOptions{}
	parser, err := NewParser(options)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "Simple gcc command",
			input:    "gcc -c main.c -o main.o",
			expected: 0,
		},
		{
			name:     "Command with check tool prefix",
			input:    "/path/to/check /path/to/check -p arm-linux-gnu-gcc -c main.c -o main.o",
			expected: 33, // Index where "arm-linux-gnu-gcc" starts
		},
		{
			name:     "Command with multiple prefixes",
			input:    "tool1 tool2 gcc -c main.c -o main.o",
			expected: 12, // Index where "gcc" starts (including spaces)
		},
		{
			name:     "No compiler found",
			input:    "mkdir build && cd build",
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.findCompilerStartIndex(tt.input)
			if result != tt.expected {
				t.Errorf("findCompilerStartIndex() = %d, expected %d", result, tt.expected)
				t.Errorf("Input: %s", tt.input)
			}
		})
	}
}

func TestParseCompileCommandWithPrefix(t *testing.T) {
	options := types.ParseOptions{BaseDir: "/project"}
	parser, err := NewParser(options)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	// Set working directory
	parser.dirStack = append(parser.dirStack, "/project/build")

	tests := []struct {
		name     string
		line     string
		expected *types.MakeLogEntry
	}{
		{
			name: "Command with check tool prefix",
			line: "path/to/check path/to/check -p gcc -DTEST=1 -c main.c -o main.o",
			expected: &types.MakeLogEntry{
				WorkingDir: "/project/build",
				Compiler:   "gcc",
				Args: []string{
					"gcc",
					"-DTEST=1",
					"-c",
					"main.c",
					"-o",
					"main.o",
				},
				SourceFile: "main.c",
				OutputFile: "main.o",
			},
		},
		{
			name: "Command with complex prefix",
			line: "/tools/preprocessor /tools/preprocessor -flags arm-linux-gnueabi-gcc -DARCH=arm -Wall -c file.c -o file.o",
			expected: &types.MakeLogEntry{
				WorkingDir: "/project/build",
				Compiler:   "arm-linux-gnueabi-gcc",
				Args: []string{
					"arm-linux-gnueabi-gcc",
					"-DARCH=arm",
					"-Wall",
					"-c",
					"file.c",
					"-o",
					"file.o",
				},
				SourceFile: "file.c",
				OutputFile: "file.o",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.parseCompileCommand(tt.line)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("parseCompileCommand() = %v, expected nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("parseCompileCommand() = nil, expected non-nil")
			}

			if result.WorkingDir != tt.expected.WorkingDir {
				t.Errorf("WorkingDir = %s, expected %s", result.WorkingDir, tt.expected.WorkingDir)
			}

			if result.Compiler != tt.expected.Compiler {
				t.Errorf("Compiler = %s, expected %s", result.Compiler, tt.expected.Compiler)
			}

			if result.SourceFile != tt.expected.SourceFile {
				t.Errorf("SourceFile = %s, expected %s", result.SourceFile, tt.expected.SourceFile)
			}

			if result.OutputFile != tt.expected.OutputFile {
				t.Errorf("OutputFile = %s, expected %s", result.OutputFile, tt.expected.OutputFile)
			}

			// Compare args
			if len(result.Args) != len(tt.expected.Args) {
				t.Errorf("Args length = %d, expected %d", len(result.Args), len(tt.expected.Args))
				t.Errorf("Actual args: %v", result.Args)
				t.Errorf("Expected args: %v", tt.expected.Args)
				return
			}

			for i, arg := range tt.expected.Args {
				if result.Args[i] != arg {
					t.Errorf("Args[%d] = %s, expected %s", i, result.Args[i], arg)
				}
			}
		})
	}
}
