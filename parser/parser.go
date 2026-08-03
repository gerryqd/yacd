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

	// Echo patterns for extracting paths. The unquoted pattern skips echo
	// options such as -n or -e before capturing the path argument.
	echoWithQuotesPattern    = `echo\s+['"]([^'"]+)['"]`
	echoWithoutQuotesPattern = `echo\s+(?:-[a-zA-Z]+\s+)*([^\s]+)`

	// Echo line pattern used to skip echo commands that merely display a
	// command instead of executing it. Handles multiple spaces after "echo"
	// and an optional leading '@' (make silent prefix).
	echoLinePattern = `^\s*@?echo(\s+|$)`

	// Shell redirection patterns. Multi-character operators (>>, 2>>, n>&m)
	// must be listed before their single-character prefixes so an operand
	// following a space (e.g. ">> out.log") is removed together with the
	// operator instead of leaking. Quoted segments are protected before
	// removal so redirects inside quotes survive.
	redirectionPatterns = `\s+2>&1|\s+\d+>&\s*\d+|\s+>&\s*\d+|\s+>>\s*\S+|\s+2>>\s*\S+|\s+\d+>>\s*\S+|\s+2>\s*\S+|\s+\d+>\s*\S+|\s+>\s*\S+|\s+<\s*\S+`

	// Quoted segment pattern for protecting quoted content from redirection
	// removal. Handles escaped quotes inside double-quoted strings.
	quotedSegmentPattern = `"[^"\\]*(?:\\.[^"\\]*)*"|'[^']*'`
)

// Pre-compiled regexes for redirection removal

func init() {
	redirectionRegex = regexp.MustCompile(redirectionPatterns)
	compilerRegex = regexp.MustCompile(commonCompilers)
	makeDirEnterRegex = regexp.MustCompile(makeDirEnterPattern)
	makeDirLeaveRegex = regexp.MustCompile(makeDirLeavePattern)
	cdChainRegex = regexp.MustCompile(cdChainPattern)
	backtickRegex = regexp.MustCompile(backtickPattern)
	echoQuotedRegex = regexp.MustCompile(echoWithQuotesPattern)
	echoUnquotedRegex = regexp.MustCompile(echoWithoutQuotesPattern)
	echoLineRegex = regexp.MustCompile(echoLinePattern)
	quotedSegmentRegex = regexp.MustCompile(quotedSegmentPattern)
	dollarParenRegex = regexp.MustCompile(`\$\(([^)]*)\)`)
}

var (
	redirectionRegex   *regexp.Regexp
	compilerRegex      *regexp.Regexp
	makeDirEnterRegex  *regexp.Regexp
	makeDirLeaveRegex  *regexp.Regexp
	cdChainRegex       *regexp.Regexp
	backtickRegex      *regexp.Regexp
	echoQuotedRegex    *regexp.Regexp
	echoUnquotedRegex  *regexp.Regexp
	echoLineRegex      *regexp.Regexp
	quotedSegmentRegex *regexp.Regexp
	dollarParenRegex   *regexp.Regexp
)

// Parser parser struct
type Parser struct {
	// Parse options
	options types.ParseOptions

	// Current working directory stack
	dirStack []string
}

// NewParser creates a new parser
func NewParser(options types.ParseOptions) (*Parser, error) {
	return &Parser{
		dirStack: make([]string, 0),
		options:  options,
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
	if matches := makeDirEnterRegex.FindStringSubmatch(line); matches != nil {
		dir := matches[2]
		p.dirStack = append(p.dirStack, dir)
		if p.options.Verbose {
			fmt.Printf("Entering directory: %s\n", dir)
		}
		return true
	}

	// Check if leaving directory
	if matches := makeDirLeaveRegex.FindStringSubmatch(line); matches != nil {
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
	// Skip echo commands that might contain compiler names but are not actual
	// compilation commands. Uses a regex so multiple spaces after "echo" are
	// handled correctly.
	if echoLineRegex.MatchString(line) {
		return nil
	}

	// Check for compiler command (simplified logic)
	if compilerRegex.MatchString(line) {
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
	matches := cdChainRegex.FindStringSubmatch(line)
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
	args := types.SplitCommandLine(compilerCommand)
	if len(args) == 0 {
		return nil
	}

	compiler := args[0]

	// Additional validation: check if the compiler looks like a real compiler
	if p.isInvalidCompiler(compiler) {
		return nil
	}

	// Find source file and output file
	sources, outputFile := p.extractSourceFiles(args)
	if len(sources) == 0 {
		return nil
	}
	if len(sources) > 1 {
		// A single compilation entry cannot represent multiple source files.
		// Skip such lines to avoid generating misleading entries.
		if p.options.Verbose {
			fmt.Printf("Skipping line with multiple source files: %v\n", sources)
		}
		return nil
	}
	sourceFile := sources[0]

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
	// Scan words in the line, tracking byte offset of each word
	pos := 0
	var prevWord string
	for pos < len(line) {
		// Skip leading whitespace
		for pos < len(line) && line[pos] == ' ' {
			pos++
		}
		if pos >= len(line) {
			break
		}

		// Extract next word
		start := pos
		for pos < len(line) && line[pos] != ' ' {
			pos++
		}
		word := line[start:pos]

		// Skip flags
		if strings.HasPrefix(word, "-") {
			prevWord = word
			continue
		}

		if compilerRegex.MatchString(word) {
			// Skip if preceded by -D macro definition
			if strings.HasPrefix(prevWord, "-D") {
				prevWord = word
				continue
			}
			return start
		}
		prevWord = word
	}

	// If no compiler found in words, try the direct approach
	if compilerRegex.MatchString(line) {
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

// extractSourceFiles extracts all source files and the output file from arguments
func (p *Parser) extractSourceFiles(args []string) ([]string, string) {
	var sources []string
	outputFile := ""
	for i, arg := range args {
		// Find output file (-o parameter)
		if arg == "-o" && i+1 < len(args) {
			outputFile = args[i+1]
			continue
		}

		// Find source files
		if p.isSourceFile(arg) {
			sources = append(sources, arg)
		}
	}

	return sources, outputFile
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
	// Process command substitutions to extract path information
	cleanLine := p.processCommandSubstitution(line)

	// Protect quoted segments so redirects inside quotes are never removed
	cleanLine, quotedSegments := protectQuotedSegments(cleanLine)

	// Remove all redirection patterns using pre-compiled regex
	cleanLine = redirectionRegex.ReplaceAllString(cleanLine, "")

	// Restore quoted segments
	cleanLine = restoreQuotedSegments(cleanLine, quotedSegments)

	return strings.TrimSpace(cleanLine)
}

// protectQuotedSegments replaces quoted substrings with placeholder markers so
// that redirection removal never touches content inside quotes. It returns the
// transformed line and the original quoted segments for later restoration.
func protectQuotedSegments(line string) (string, []string) {
	var segments []string
	transformed := quotedSegmentRegex.ReplaceAllStringFunc(line, func(match string) string {
		idx := len(segments)
		segments = append(segments, match)
		return fmt.Sprintf("\x01%d\x01", idx)
	})
	return transformed, segments
}

// restoreQuotedSegments replaces placeholder markers with the original quoted
// segments in a single left-to-right pass using a Replacer, instead of rescanning
// the whole line once per segment.
func restoreQuotedSegments(line string, segments []string) string {
	if len(segments) == 0 {
		return line
	}
	pairs := make([]string, 0, len(segments)*2)
	for i, segment := range segments {
		pairs = append(pairs, fmt.Sprintf("\x01%d\x01", i), segment)
	}
	return strings.NewReplacer(pairs...).Replace(line)
}

// processCommandSubstitution handles both $(...) and backtick `...` command
// substitutions, extracting path information from embedded echo commands.
func (p *Parser) processCommandSubstitution(line string) string {
	// First handle $(...) substitutions
	line = p.processDollarParenSubstitution(line)
	// Then handle backtick `...` substitutions
	line = p.processBacktickSubstitution(line)
	return line
}

// processDollarParenSubstitution handles $(...) command substitution patterns
func (p *Parser) processDollarParenSubstitution(line string) string {
	matches := dollarParenRegex.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return line
	}

	result := line
	for _, match := range matches {
		fullMatch := match[0]
		command := match[1]
		if extractedPath := p.extractPathFromCommand(command); extractedPath != "" {
			result = strings.Replace(result, fullMatch, extractedPath, 1)
		} else {
			result = strings.Replace(result, fullMatch, "", 1)
		}
	}
	return result
}

// processBacktickSubstitution processes backtick command substitutions and extracts path information
func (p *Parser) processBacktickSubstitution(line string) string {
	// Find all backtick matches using pre-compiled regex
	matches := backtickRegex.FindAllStringSubmatch(line, -1)
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
	matches := echoQuotedRegex.FindStringSubmatch(command)
	if len(matches) > 1 {
		return matches[1]
	}

	// Pattern to match "echo path" (without quotes), skipping echo options
	// such as -n or -e. A capture that still looks like an option (e.g. a bare
	// "echo -n") is not a path.
	matches = echoUnquotedRegex.FindStringSubmatch(command)
	if len(matches) > 1 && !strings.HasPrefix(matches[1], "-") {
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
