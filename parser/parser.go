package parser

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/gerryqd/yacd/types"
	"github.com/gerryqd/yacd/utils/pathutil"
)

const (
	// Regular expressions for make directory enter and exit messages
	makeDirEnterPattern = `^make(\[\d+\])?: Entering directory '(.+)'`
	makeDirLeavePattern = `^make(\[\d+\])?: Leaving directory '(.+)'`

	// Common C/C++ compilers (simplified pattern)
	commonCompilers = `(gcc|g\+\+|clang|clang\+\+|cc)`
)

// Parser parser struct
type Parser struct {
	// Current working directory stack
	dirStack []string

	// Directory history to track all directories we've seen
	directoryHistory []string

	// Compiler detection regular expression
	compilerRegex *regexp.Regexp

	// Make directory enter regular expression
	makeDirEnterRegex *regexp.Regexp

	// Make directory exit regular expression
	makeDirLeaveRegex *regexp.Regexp

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

	return &Parser{
		dirStack:          make([]string, 0),
		directoryHistory:  make([]string, 0),
		compilerRegex:     compilerRegex,
		makeDirEnterRegex: makeDirEnterRegex,
		makeDirLeaveRegex: makeDirLeaveRegex,
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
		// Also add to history
		p.directoryHistory = append(p.directoryHistory, dir)
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

// parseShellCommandChain handles shell command chains like "cd dir && gcc ..."
func (p *Parser) parseShellCommandChain(line string) *types.MakeLogEntry {
	// Look for patterns like "cd <dir> && <compiler> ..."
	cdPattern := regexp.MustCompile(`^\s*cd\s+([^&]+)\s*&&\s*(.+)$`)
	matches := cdPattern.FindStringSubmatch(line)
	if matches == nil {
		if p.options.Verbose {
			fmt.Printf("No shell command chain pattern matched for: %s\n", line)
		}
		return nil
	}

	cdDir := strings.TrimSpace(matches[1])
	compilerCommand := strings.TrimSpace(matches[2])

	if p.options.Verbose {
		fmt.Printf("Found cd command chain: cd %s && %s\n", cdDir, compilerCommand)
	}

	// Parse the compiler command part without directory inference
	entry := p.parseCompilerCommandOnly(compilerCommand)
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

	// Don't modify source and output file paths here
	// Let the generator handle path resolution based on the working directory

	if p.options.Verbose {
		fmt.Printf("Shell command parsed - Working dir: %s, Source: %s, Output: %s\n",
			entry.WorkingDir, entry.SourceFile, entry.OutputFile)
	}

	return entry
}

// parseCompilerCommandOnly parses a compiler command without directory inference
func (p *Parser) parseCompilerCommandOnly(line string) *types.MakeLogEntry {
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

	// Find source file and output file
	sourceFile, outputFile := p.extractFiles(args)
	if sourceFile == "" {
		return nil
	}

	return &types.MakeLogEntry{
		WorkingDir: "", // Will be set by caller
		Compiler:   compiler,
		Args:       args,
		SourceFile: sourceFile,
		OutputFile: outputFile,
	}
}

// findCompilerStartIndex finds the start index of the actual compiler command
func (p *Parser) findCompilerStartIndex(line string) int {
	// Split the line into words to find the compiler
	words := strings.Fields(line)

	// Look for the compiler pattern in the words
	for i, word := range words {
		if p.compilerRegex.MatchString(word) {
			// Found a compiler, calculate its position in the original line
			// Reconstruct the prefix to find the exact position
			prefix := ""
			for j := 0; j < i; j++ {
				prefix += words[j] + " "
			}
			return len(prefix)
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
	return pathutil.ResolveRelativePath(baseDir, relativePath)
}

// parseDirectCompileCommand parses direct compilation commands
func (p *Parser) parseDirectCompileCommand(line string) *types.MakeLogEntry {
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

	// Find source file and output file
	sourceFile, outputFile := p.extractFiles(args)
	if sourceFile == "" {
		return nil
	}

	// Get current working directory from stack
	workingDir := ""
	if len(p.dirStack) > 0 {
		workingDir = p.dirStack[len(p.dirStack)-1]
	}

	return &types.MakeLogEntry{
		WorkingDir: workingDir,
		Compiler:   compiler,
		Args:       args,
		SourceFile: sourceFile,
		OutputFile: outputFile,
	}
}

// splitCommandLine splits command line, handling quotes and escape characters
func (p *Parser) splitCommandLine(line string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false
	escaped := false

	for i, char := range line {
		switch {
		case escaped:
			current.WriteRune(char)
			escaped = false
		case char == '\\':
			escaped = true
		case char == '"' || char == '\'':
			inQuotes = !inQuotes
		case char == ' ' && !inQuotes:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}

		// Handle end of line case
		if i == len(line)-1 && current.Len() > 0 {
			args = append(args, current.String())
		}
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
	return pathutil.IsSourceFile(filename)
}

// removeRedirectionOperators removes shell redirection operators and processes backtick command substitutions from command line
func (p *Parser) removeRedirectionOperators(line string) string {
	// First, process backtick command substitutions to extract path information
	cleanLine := p.processBacktickSubstitution(line)

	// Common redirection patterns to remove
	// Use word boundaries and more precise matching
	redirectionPatterns := []string{
		`\s+2>&1`,     // stderr to stdout
		`\s+>&1`,      // stdout to fd 1
		`\s+>\S+`,     // stdout to file
		`\s+>>\S+`,    // stdout append to file
		`\s+<\S+`,     // stdin from file
		`\s+2>\S+`,    // stderr to file
		`\s+2>>\S+`,   // stderr append to file
		`\s+\d+>&\d+`, // fd redirection
		`\s+\d+>\S+`,  // numbered fd to file
		`\s+\d+>>\S+`, // numbered fd append to file
		`\s+2>&1$`,    // stderr to stdout at end
		`\s+>&1$`,     // stdout to fd 1 at end
	}

	for _, pattern := range redirectionPatterns {
		re := regexp.MustCompile(pattern)
		cleanLine = re.ReplaceAllString(cleanLine, "")
	}

	return strings.TrimSpace(cleanLine)
}

// processBacktickSubstitution processes backtick command substitutions and extracts path information
func (p *Parser) processBacktickSubstitution(line string) string {
	// Pattern to match backtick command substitutions
	backtickPattern := regexp.MustCompile("`([^`]*)`")

	// Find all backtick matches
	matches := backtickPattern.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return line // No backticks found
	}

	result := line
	for _, match := range matches {
		fullMatch := match[0] // The entire `...` part
		command := match[1]   // The content inside backticks

		// Extract path from echo command
		if extractedPath := p.extractPathFromCommand(command); extractedPath != "" {
			// Ensure there's a space between path and following content if needed
			replacementText := extractedPath
			// Check if the character after the backtick is a letter/digit (indicating a filename)
			backtickEndIndex := strings.Index(result, fullMatch) + len(fullMatch)
			if backtickEndIndex < len(result) &&
				(result[backtickEndIndex] >= 'a' && result[backtickEndIndex] <= 'z' ||
					result[backtickEndIndex] >= 'A' && result[backtickEndIndex] <= 'Z' ||
					result[backtickEndIndex] >= '0' && result[backtickEndIndex] <= '9') {
				// Add a space if path doesn't end with / and next char is alphanumeric
				if !strings.HasSuffix(extractedPath, "/") {
					replacementText = extractedPath + " "
				}
			}
			// Replace the backtick part with the extracted path
			result = strings.Replace(result, fullMatch, replacementText, 1)
		} else {
			// If no path found, remove the backtick part
			result = strings.Replace(result, fullMatch, "", 1)
		}
	}

	return result
}

// extractPathFromCommand extracts path from shell command like "test -f 'file' || echo 'path'"
// NOTE: This function assumes the command is in the form of "test -f 'file' || echo 'path'"
//
//	and this is only a workaround, which works well in quite a few cases.
func (p *Parser) extractPathFromCommand(command string) string {
	// Pattern to match "echo 'path'" or "echo \"path\""
	echoPattern := regexp.MustCompile(`echo\s+['"]([^'"]+)['"]`)
	matches := echoPattern.FindStringSubmatch(command)
	if len(matches) > 1 {
		return matches[1]
	}

	// Pattern to match "echo path" (without quotes)
	echoPatternNoQuotes := regexp.MustCompile(`echo\s+([^\s]+)`)
	matches = echoPatternNoQuotes.FindStringSubmatch(command)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// GetCurrentDirectory gets current working directory
func (p *Parser) GetCurrentDirectory() string {
	if len(p.dirStack) == 0 {
		return ""
	}
	return p.dirStack[len(p.dirStack)-1]
}

// ParseMakeLog is a convenience function to parse make log with default options
func ParseMakeLog(reader io.Reader, verbose bool) ([]types.MakeLogEntry, error) {
	options := types.ParseOptions{
		Verbose: verbose,
	}

	parser, err := NewParser(options)
	if err != nil {
		return nil, err
	}

	return parser.ParseMakeLog(reader)
}
