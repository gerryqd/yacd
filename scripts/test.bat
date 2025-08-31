@echo off
REM yacd test script for Windows
echo Running yacd tests on Windows...

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo Error: Go is not installed or not in PATH
    exit /b 1
)

REM Format code
echo Formatting code...
go fmt ./...

REM Run static analysis
echo Running static analysis...
go vet ./...

REM Run tests
echo Running tests...
go test -v ./...

if errorlevel 1 (
    echo Tests failed!
    exit /b 1
)

echo All tests passed!