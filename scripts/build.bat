@echo off
REM yacd build script for Windows
echo Building yacd for Windows...

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo Error: Go is not installed or not in PATH
    exit /b 1
)

REM Create build directory
if not exist build mkdir build

REM Get version information
for /f "tokens=*" %%i in ('git describe --tags --always --dirty 2^>nul') do set VERSION=%%i
if "%VERSION%"=="" set VERSION=dev

for /f "tokens=*" %%i in ('git rev-parse --short HEAD 2^>nul') do set COMMIT=%%i
if "%COMMIT%"=="" set COMMIT=unknown

REM Build binary
echo Building with version: %VERSION%
go build -ldflags "-X main.Version=%VERSION% -X main.Commit=%COMMIT%" -o build\yacd.exe .

if errorlevel 1 (
    echo Build failed!
    exit /b 1
)

echo Build successful! Binary created at build\yacd.exe