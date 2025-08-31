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
}
