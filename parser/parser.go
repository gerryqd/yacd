package parser

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gerryqd/yacd/types"
)

const (
	// Regular expressions for make directory enter and exit messages
	makeDirEnterPattern = `^make(\[\d+\])?: Entering directory '(.+)'`
	makeDirLeavePattern = `^make(\[\d+\])?: Leaving directory '(.+)'`

	// Common C/C++ compilers (simplified pattern)
	commonCompilers = `(\w+-)*\w*-(gcc|g\+\+|clang|clang\+\+|cc)|\b(gcc|g\+\+|clang|clang\+\+|cc)\b`

	// Shell command chain pattern
	cdChainPattern = `^\s*cd\s+([^&]+)\s*&&\s*(.+)$`

	// Backtick command substitution pattern
	backtickPattern = "`([^`]*)`"

	// Echo patterns for extracting paths
	echoWithQuotesPattern    = `echo\s+['"]([^'"]+)['"]`
	echoWithoutQuotesPattern = `echo\s+([^\s]+)`

	// Shell redirection patterns
	redirectionPatterns = `\s+2>&1|\s+>&1|\s+>\S+|\s+>>\S+|\s+<\S+|\s+2>\S+|\s+2>>\S+|\s+\d+>&\d+|\s+\d+>\S+|\s+\d+>>\S+`
)

// Pre-compiled regexes for redirection removal
var redirectionRegex *regexp.Regexp

func init() {
	redirectionRegex = regexp.MustCompile(redirectionPatterns)
}

// Parser parser struct
type Parser struct {
	// Current working directory stack
	dirStack []string

	// Compiler detection regular expression
	compilerRegex *regexp.Regexp

	// Make directory enter regular expression
	makeDirEnterRegex *regexp.Regexp

	// Make directory exit regular expression
	makeDirLeaveRegex *regexp.Regexp

	// Pre-compiled regexes for command parsing
	cdChainRegex           *regexp.Regexp
	backtickRegex          *regexp.Regexp
	echoQuotedRegex        *regexp.Regexp
	echoUnquotedRegex      *regexp.Regexp

	// Parse options
	options types.ParseOptions
}

// NewParser creates a new parser
func NewParser(options types.ParseOptions) (*Parser, error) {
	compilerRegex, err := regexp.Compile(commonCompilers)
	if err != nil {
		return nil, fmt.Errorf("compiler regex compilation failed: %w", err)
	}

	makeDirEnterRegex, err := regexp.Compile(makeDirEnterPattern)
	if err != nil {
		return nil, fmt.Errorf("make directory enter regex compilation failed: %w", err)
	}

	makeDirLeaveRegex, err := regexp.Compile(makeDirLeavePattern)
	if err != nil {
		return nil, fmt.Errorf("make directory exit regex compilation failed: %w", err)
	}

	cdChainRegex, err := regexp.Compile(cdChainPattern)
	if err != nil {
		return nil, fmt.Errorf("cd chain regex compilation failed: %w", err)
	}

	backtickRegex, err := regexp.Compile(backtickPattern)
	if err != nil {
		return nil, fmt.Errorf("backtick regex compilation failed: %w", err)
	}

	echoQuotedRegex, err := regexp.Compile(echoWithQuotesPattern)
	if err != nil {
		return nil, fmt.Errorf("echo quoted regex compilation failed: %w", err)
	}

	echoUnquotedRegex, err := regexp.Compile(echoWithoutQuotesPattern)
	if err != nil {
		return nil, fmt.Errorf("echo unquoted regex compilation failed: %w", err)
	}

	return &Parser{
		dirStack:          make([]string, 0),
		compilerRegex:     compilerRegex,
		makeDirEnterRegex: makeDirEnterRegex,
		makeDirLeaveRegex: makeDirLeaveRegex,
		cdChainRegex:      cdChainRegex,
		backtickRegex:     backtickRegex,
		echoQuotedRegex:   echoQuotedRegex,
		echoUnquotedRegex: echoUnquotedRegex,
		options:           options,
	}, nil
}

// ParseMakeLog parses make log
func (p *Parser) ParseMakeLog(reader io.Reader) ([]types.MakeLogEntry, error) {
	var entries []types.MakeLogEntry
	scanner := bufio.NewScanner(reader)

	// Set base directory
	if p.options.BaseDir != "" {
		p.dirStack = append(p.dirStack, p.options.BaseDir)
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Skip lines starting with '#' (comments)
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Handle make directory changes
		if p.handleDirectoryChange(line) {
			continue
		}

		// Parse compilation commands
		if entry := p.parseCompileCommand(line); entry != nil {
			entries = append(entries, *entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	return entries, nil
}

// handleDirectoryChange handles directory changes
func (p *Parser) handleDirectoryChange(line string) bool {
	// Check if entering directory
	if matches := p.makeDirEnterRegex.FindStringSubmatch(line); matches != nil {
		dir := matches[2]
		p.dirStack = append(p.dirStack, dir)
		if p.options.Verbose {
			fmt.Printf("Entering directory: %s\n", dir)
		}
		return true
	}

	// Check if leaving directory
	if matches := p.makeDirLeaveRegex.FindStringSubmatch(line); matches != nil {
		if len(p.dirStack) > 0 {
			if p.options.Verbose {
				fmt.Printf("Leaving directory: %s\n", p.dirStack[len(p.dirStack)-1])
			}
			p.dirStack = p.dirStack[:len(p.dirStack)-1]
		}
		return true
	}

	return false
}

// parseCompileCommand parses compilation commands
func (p *Parser) parseCompileCommand(line string) *types.MakeLogEntry {
	// Skip echo commands that might contain compiler names but are not actual compilation commands
	if strings.HasPrefix(strings.TrimSpace(line), "echo ") {
		return nil
	}

	// Check for compiler command (simplified logic)
	if p.compilerRegex.MatchString(line) {
		// Handle shell command chains with cd && compiler
		if entry := p.parseShellCommandChain(line); entry != nil {
			return entry
		}
		return p.parseDirectCompileCommand(line)
	}

	return nil
}

// parseShellCommandChain handles shell command chains such as "cd dir && gcc ..."
func (p *Parser) parseShellCommandChain(line string) *types.MakeLogEntry {
	matches := p.cdChainRegex.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	cdDir := strings.TrimSpace(matches[1])
	compilerCommand := strings.TrimSpace(matches[2])

	if p.options.Verbose {
		fmt.Printf("Found cd command chain: cd %s && %s\n", cdDir, compilerCommand)
	}

	// Parse the compiler command part without directory inference
	entry := p.parseCompileCommandArgs(compilerCommand)
	if entry == nil {
		if p.options.Verbose {
			fmt.Printf("Failed to parse compiler command: %s\n", compilerCommand)
		}
		return nil
	}

	// Calculate the new working directory
	currentWorkingDir := ""
	if len(p.dirStack) > 0 {
		currentWorkingDir = p.dirStack[len(p.dirStack)-1]
	}
	newWorkingDir := p.resolveRelativePath(currentWorkingDir, cdDir)
	entry.WorkingDir = newWorkingDir

	if p.options.Verbose {
		fmt.Printf("Shell command parsed - Working dir: %s, Source: %s, Output: %s\n",
			entry.WorkingDir, entry.SourceFile, entry.OutputFile)
	}

	return entry
}

// parseCompileCommandArgs parses compiler arguments from a command line without setting working directory
func (p *Parser) parseCompileCommandArgs(line string) *types.MakeLogEntry {
	// Remove redirection operators before parsing
	cleanLine := p.removeRedirectionOperators(line)

	// Find the actual compiler in the command line
	compilerStartIndex := p.findCompilerStartIndex(cleanLine)
	if compilerStartIndex == -1 {
		return nil
	}

	// Extract the compiler command part
	compilerCommand := cleanLine[compilerStartIndex:]

	// Split command line arguments
	args := p.splitCommandLine(compilerCommand)
	if len(args) == 0 {
		return nil
	}

	compiler := args[0]

	// Additional validation: check if the compiler looks like a real compiler
	if p.isInvalidCompiler(compiler) {
		return nil
	}

	// Find source file and output file
	sourceFile, outputFile := p.extractFiles(args)
	if sourceFile == "" {
		return nil
	}

	return &types.MakeLogEntry{
		Compiler:   compiler,
		Args:       args,
		SourceFile: sourceFile,
		OutputFile: outputFile,
	}
}

// parseDirectCompileCommand parses direct compilation commands
func (p *Parser) parseDirectCompileCommand(line string) *types.MakeLogEntry {
	entry := p.parseCompileCommandArgs(line)
	if entry == nil {
		return nil
	}

	// Get current working directory from stack
	if len(p.dirStack) > 0 {
		entry.WorkingDir = p.dirStack[len(p.dirStack)-1]
	}

	return entry
}

// findCompilerStartIndex finds the start index of the actual compiler command
func (p *Parser) findCompilerStartIndex(line string) int {
	// Split the line into words to find the compiler
	words := strings.Fields(line)

	// Look for the compiler pattern in the words
	// Skip words that are compiler flags like -D, -I, etc.
	for i, word := range words {
		// Skip if it's a compiler flag or option
		if strings.HasPrefix(word, "-") {
			continue
		}

		// Check if it's a potential compiler name
		if p.compilerRegex.MatchString(word) {
			// Additional check: make sure it's not part of a macro definition like -DCPP_LOCATION="gcc"
			if i > 0 {
				prevWord := words[i-1]
				if strings.HasPrefix(prevWord, "-D") {
					continue
				}
			}
			// Found a compiler, calculate its position in the original line
			var prefix strings.Builder
			for j := 0; j < i; j++ {
				prefix.WriteString(words[j])
				prefix.WriteByte(' ')
			}
			return prefix.Len()
		}
	}

	// If no compiler found in words, try the direct approach
	if p.compilerRegex.MatchString(line) {
		return 0
	}

	return -1
}

// resolveRelativePath resolves a relative path against a base directory
func (p *Parser) resolveRelativePath(baseDir, relativePath string) string {
	if filepath.IsAbs(relativePath) {
		return relativePath
	}
	return filepath.Join(baseDir, relativePath)
}

// splitCommandLine splits command line, handling quotes and escape characters
func (p *Parser) splitCommandLine(line string) []string {
	var args []string
	var current strings.Builder
	inSingleQuotes := false
	inDoubleQuotes := false
	escaped := false

	for _, char := range line {
		switch {
		case escaped:
			current.WriteRune(char)
			escaped = false
		case char == '\\':
			if inSingleQuotes {
				current.WriteRune(char)
			} else {
				escaped = true
			}
		case char == '\'':
			if inDoubleQuotes {
				current.WriteRune(char)
			} else {
				inSingleQuotes = !inSingleQuotes
			}
		case char == '"':
			if inSingleQuotes {
				current.WriteRune(char)
			} else {
				inDoubleQuotes = !inDoubleQuotes
			}
		case char == ' ' && !inSingleQuotes && !inDoubleQuotes:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// extractFiles extracts source file and output file from arguments
func (p *Parser) extractFiles(args []string) (sourceFile, outputFile string) {
	for i, arg := range args {
		// Find output file (-o parameter)
		if arg == "-o" && i+1 < len(args) {
			outputFile = args[i+1]
			continue
		}

		// Find source file (usually the last .c or .cpp file)
		if p.isSourceFile(arg) {
			sourceFile = arg
		}
	}

	return sourceFile, outputFile
}

// isSourceFile determines if it is a source file
func (p *Parser) isSourceFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, validExt := range sourceExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

var sourceExts = []string{".c", ".cpp", ".cc", ".cxx", ".c++", ".s", ".asm"}

// removeRedirectionOperators removes shell redirection operators and processes backtick command substitutions from command line
func (p *Parser) removeRedirectionOperators(line string) string {
	// First, process backtick command substitutions to extract path information
	cleanLine := p.processBacktickSubstitution(line)

	// Remove all redirection patterns using pre-compiled regex
	cleanLine = redirectionRegex.ReplaceAllString(cleanLine, "")

	return strings.TrimSpace(cleanLine)
}

// processBacktickSubstitution processes backtick command substitutions and extracts path information
func (p *Parser) processBacktickSubstitution(line string) string {
	// Find all backtick matches using pre-compiled regex
	matches := p.backtickRegex.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return line
	}

	result := line
	for _, match := range matches {
		fullMatch := match[0] // The entire `...` part
		command := match[1]   // The content inside backticks

		// Extract path from echo command
		if extractedPath := p.extractPathFromCommand(command); extractedPath != "" {
			replacementText := extractedPath
			// Check if the character after the backtick is a letter/digit (indicating a filename)
			backtickEndIndex := strings.Index(result, fullMatch) + len(fullMatch)
			if backtickEndIndex < len(result) &&
				(result[backtickEndIndex] >= 'a' && result[backtickEndIndex] <= 'z' ||
					result[backtickEndIndex] >= 'A' && result[backtickEndIndex] <= 'Z' ||
					result[backtickEndIndex] >= '0' && result[backtickEndIndex] <= '9') {
				if !strings.HasSuffix(extractedPath, "/") {
					replacementText = extractedPath + " "
				}
			}
			result = strings.Replace(result, fullMatch, replacementText, 1)
		} else {
			result = strings.Replace(result, fullMatch, "", 1)
		}
	}

	return result
}

// extractPathFromCommand extracts path from shell command like "test -f 'file' || echo 'path'"
func (p *Parser) extractPathFromCommand(command string) string {
	// Pattern to match "echo 'path'" or "echo \"path\""
	matches := p.echoQuotedRegex.FindStringSubmatch(command)
	if len(matches) > 1 {
		return matches[1]
	}

	// Pattern to match "echo path" (without quotes)
	matches = p.echoUnquotedRegex.FindStringSubmatch(command)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// isInvalidCompiler checks if the compiler name is likely not a real compiler
func (p *Parser) isInvalidCompiler(compiler string) bool {
	if strings.HasPrefix(compiler, "-D") {
		return true
	}

	if strings.HasSuffix(compiler, ".cc") ||
		strings.HasSuffix(compiler, ".cpp") ||
		strings.HasSuffix(compiler, ".c") ||
		strings.HasSuffix(compiler, ".cxx") ||
		strings.HasSuffix(compiler, ".C") ||
		strings.HasSuffix(compiler, ".s") ||
		strings.HasSuffix(compiler, ".S") ||
		strings.HasSuffix(compiler, ".o") {
		return true
	}

	if strings.Contains(compiler, ".yy.tab.") ||
		strings.Contains(compiler, ".tab.") ||
		strings.Contains(compiler, ".lex.") {
		return true
	}

	return false
}

// ParseMakeLog is a convenience function to parse make log with full options
func ParseMakeLog(reader io.Reader, options types.ParseOptions) ([]types.MakeLogEntry, error) {
	parser, err := NewParser(options)
	if err != nil {
		return nil, err
	}

	return parser.ParseMakeLog(reader)
}
