# AGENTS.md

This file provides guidance to agents when working with code in this repository.

## Build/Lint/Test Commands

- `make build` - Build the binary
- `make test` - Run all tests
- `make test-coverage` - Run tests with coverage report
- `make fmt` - Format code
- `make vet` - Static analysis
- `make lint` - Lint checks (requires golint installation)
- `./scripts/quality-check.sh` - Run comprehensive code quality checks

## Code Style Guidelines

- Error handling: Use `fmt.Errorf` with `%w` verb for wrapped errors with context
- Path handling: Use `path/filepath` for cross-platform path operations
- Command line splitting: Use `types.SplitCommandLine` for proper quote/escape handling
- File I/O: Always handle errors properly and close resources
- Testing: Use table-driven tests where appropriate
- Naming: Use descriptive names for variables and functions
- Comments: Write clear comments for exported functions and complex logic

## Project-Specific Patterns

- Command structure follows Cobra framework patterns
- Input validation happens in `cmd/validation.go`
- Execution logic is in `cmd/execute.go`
- Make log parsing is in `parser` package
- JSON generation is in `generator` package
- Shared types and utilities are in `types` package
- All regexes are pre-compiled at package level in `parser` package
- Sysroot cache is encapsulated in `generator.SysrootCache` struct
- `ParseOptions` should be passed as pointer for consistency
- Generator supports both `command` (string) and `arguments` (array) output formats
- Generator supports deduplication via `Deduplicate` option
- Generator checks source file existence sequentially in `checkMissingFiles`

## Testing Specifics

- Tests are organized by package
- Use `t.TempDir()` for temporary test directories
- Test files should be in the same package as the code being tested
- Run individual tests with: `go test -v ./package/path -run TestName`
