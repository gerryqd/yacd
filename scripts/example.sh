#!/bin/bash

# yacd usage example script
# Demonstrates how to use yacd to generate compile_commands.json from make logs

set -e

echo "yacd - Yet Another CompileDB Usage Example"
echo "==========================================="

# Check if in project root directory
if [ ! -f "go.mod" ] || [ ! -d "test_data" ]; then
    echo "Error: Please run this script from yacd project root directory"
    exit 1
fi

# Build project
echo "1. Building yacd..."
if command -v go &> /dev/null; then
    go build -o yacd .
    echo "✓ Build completed"
else
    echo "Warning: Go not installed, please build project manually"
    echo "You can use the following command:"
    echo "  go build -o yacd ."
    exit 1
fi

# Check test data
echo ""
echo "2. Checking test data..."
if [ -f "test_data/loga.txt" ]; then
    echo "✓ Found test data file: test_data/loga.txt"
    lines=$(wc -l < test_data/loga.txt)
    echo "  File contains $lines lines"
else
    echo "✗ Test data file does not exist"
    exit 1
fi

# Run yacd
echo ""
echo "3. Running yacd to generate compile_commands.json..."
./yacd -i test_data/loga.txt -o compile_commands.json -v

# Verify output
echo ""
echo "4. Verifying output file..."
if [ -f "compile_commands.json" ]; then
    echo "✓ Successfully generated compile_commands.json"
    
    # Check JSON format
    if command -v python3 &> /dev/null; then
        if python3 -m json.tool compile_commands.json > /dev/null 2>&1; then
            echo "✓ JSON format is valid"
        else
            echo "✗ JSON format is invalid"
            exit 1
        fi
    fi
    
    # Show statistics
    entries=$(grep -c '"directory"' compile_commands.json || echo "0")
    echo "✓ Contains $entries compilation entries"
    
    # Show file size
    size=$(du -h compile_commands.json | cut -f1)
    echo "✓ File size: $size"
    
else
    echo "✗ Output file not generated"
    exit 1
fi

# Show example content
echo ""
echo "5. Example output file content:"
echo "-------------------------------"
if command -v python3 &> /dev/null; then
    python3 -c "
import json
with open('compile_commands.json', 'r') as f:
    data = json.load(f)
    if data:
        print(json.dumps(data[0], indent=2, ensure_ascii=False))
        if len(data) > 1:
            print(f'... plus {len(data) - 1} more entries')
    else:
        print('File is empty')
"
else
    head -20 compile_commands.json
fi

# Cleanup
echo ""
echo "6. Cleaning up temporary files..."
echo "Generated files remain in current directory:"
echo "  - compile_commands.json"
echo ""

echo "Example run completed!"
echo ""
echo "You can now:"
echo "  1. View the generated compile_commands.json file"
echo "  2. Copy it to your C/C++ project root directory"
echo "  3. Get better code intelligence in editors that support clangd or C/C++ extensions"
echo ""
echo "For more usage information see:"
echo "  ./yacd --help"
echo "  cat README.md"