#!/bin/bash

# yacd code quality check script
# Used to check code format, static analysis, test coverage, etc.

set -e

echo "yacd - Code Quality Check"
echo "========================"

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check function
check_step() {
    local step_name="$1"
    local cmd="$2"
    
    echo -e "${BLUE}Checking: $step_name${NC}"
    
    if eval "$cmd"; then
        echo -e "${GREEN}✓ $step_name passed${NC}"
        return 0
    else
        echo -e "${RED}✗ $step_name failed${NC}"
        return 1
    fi
}

# Main check process
main() {
    local exit_code=0
    
    echo "Starting code quality check..."
    echo
    
    # 1. Code format check
    if ! check_step "Code format check" "go fmt ./... | tee /dev/null && [ \${PIPESTATUS[0]} -eq 0 ]"; then
        echo -e "${YELLOW}Suggestion: run go fmt ./...${NC}"
        exit_code=1
    fi
    
    # 2. Static analysis
    if ! check_step "Static analysis" "go vet ./..."; then
        exit_code=1
    fi
    
    # 3. Compilation check
    if ! check_step "Compilation check" "go build -o /dev/null ."; then
        exit_code=1
    fi
    
    # 4. Test execution
    if ! check_step "Unit tests" "go test ./..."; then
        exit_code=1
    fi
    
    # 5. Test coverage
    echo -e "${BLUE}Checking: Test coverage${NC}"
    if go test -coverprofile=coverage.out ./... > /dev/null 2>&1; then
        coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
        echo -e "${GREEN}✓ Test coverage: $coverage${NC}"
        
        # Check if coverage meets minimum requirement (70%)
        coverage_num=$(echo $coverage | sed 's/%//')
        if (( $(echo "$coverage_num >= 70" | bc -l) )); then
            echo -e "${GREEN}✓ Coverage meets requirement (>= 70%)${NC}"
        else
            echo -e "${YELLOW}⚠ Coverage below recommended 70%${NC}"
        fi
    else
        echo -e "${RED}✗ Test coverage check failed${NC}"
        exit_code=1
    fi
    
    # 6. Module check
    if ! check_step "Module dependency check" "go mod verify"; then
        exit_code=1
    fi
    
    # 7. Race condition check
    if ! check_step "Race condition check" "go test -race ./..."; then
        exit_code=1
    fi
    
    # Clean up temporary files
    rm -f coverage.out
    
    echo
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}🎉 All checks passed! Code quality is good.${NC}"
    else
        echo -e "${RED}❌ Issues found, please fix and recheck.${NC}"
    fi
    
    return $exit_code
}

# Help information
show_help() {
    echo "Usage: $0 [options]"
    echo
    echo "Options:"
    echo "  -h, --help     Show help information"
    echo "  -v, --verbose  Verbose output"
    echo
    echo "This script performs the following checks:"
    echo "  - Code format check (go fmt)"
    echo "  - Static analysis (go vet)"
    echo "  - Compilation check"
    echo "  - Unit tests"
    echo "  - Test coverage"
    echo "  - Module dependency verification"
    echo "  - Race condition check"
}

# Parameter handling
case "${1:-}" in
    -h|--help)
        show_help
        exit 0
        ;;
    -v|--verbose)
        set -x
        ;;
esac

# Check if in Go project directory
if [ ! -f "go.mod" ]; then
    echo -e "${RED}Error: Current directory is not a Go project directory (go.mod file not found)${NC}"
    exit 1
fi

# Check if bc tool is installed (for floating point comparison)
if ! command -v bc &> /dev/null; then
    echo -e "${YELLOW}Warning: bc tool not found, skipping coverage threshold check${NC}"
fi

main "$@"