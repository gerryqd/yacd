package types

// CompilationEntry represents a single compilation entry in compile_commands.json
type CompilationEntry struct {
	// Working directory where the compiler is executed
	Directory string `json:"directory"`

	// Command line executed by the compiler
	Command string `json:"command,omitempty"`

	// Compiler argument array (mutually exclusive with command field)
	Arguments []string `json:"arguments,omitempty"`

	// Absolute path to the source file
	File string `json:"file"`

	// Output file path (optional)
	Output string `json:"output,omitempty"`
}

// CompilationDatabase represents the entire compilation database
type CompilationDatabase []CompilationEntry

// MakeLogEntry represents a compilation record parsed from make log
type MakeLogEntry struct {
	// Working directory
	WorkingDir string

	// Compiler executable
	Compiler string

	// Compiler arguments
	Args []string

	// Source file path
	SourceFile string

	// Output file path
	OutputFile string
}

// ParseOptions parsing options
type ParseOptions struct {
	// Input file path
	InputFile string

	// Output file path
	OutputFile string

	// Make command to execute
	MakeCommand string

	// Whether to use relative paths
	UseRelativePaths bool

	// Base directory
	BaseDir string

	// Whether to enable verbose output
	Verbose bool

	// Whether to add compiler sysroot include path to commands
	AddSysroot bool

	// Whether to output arguments array format (preferred by clangd)
	UseArguments bool

	// Whether to deduplicate entries
	Deduplicate bool
}

// SplitCommandLine splits a command line string into individual arguments,
// handling single quotes, double quotes, and escape characters.
func SplitCommandLine(line string) []string {
	var args []string
	var current []rune
	inSingleQuotes := false
	inDoubleQuotes := false
	escaped := false

	for _, char := range line {
		switch {
		case escaped:
			current = append(current, char)
			escaped = false
		case char == '\\':
			if inSingleQuotes {
				current = append(current, char)
			} else if isWindowsDrivePath(current) {
				// Windows drive-letter paths use backslash as a separator, not
				// an escape character (e.g. C:\mingw\bin\gcc.exe).
				current = append(current, char)
			} else {
				escaped = true
			}
		case char == '\'':
			if inDoubleQuotes {
				current = append(current, char)
			} else {
				inSingleQuotes = !inSingleQuotes
			}
		case char == '"':
			if inSingleQuotes {
				current = append(current, char)
			} else {
				inDoubleQuotes = !inDoubleQuotes
			}
		case char == ' ' && !inSingleQuotes && !inDoubleQuotes:
			if len(current) > 0 {
				args = append(args, string(current))
				current = current[:0]
			}
		default:
			current = append(current, char)
		}
	}

	if len(current) > 0 {
		args = append(args, string(current))
	}

	return args
}

// isWindowsDrivePath reports whether the token built so far contains a
// Windows drive-letter prefix (e.g. "C:\" or "-IC:\"), in which case
// backslashes are path separators rather than escape characters.
func isWindowsDrivePath(token []rune) bool {
	for i := 0; i+1 < len(token); i++ {
		c := token[i]
		if token[i+1] == ':' && ((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return true
		}
	}
	return false
}
