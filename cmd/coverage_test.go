package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gerryqd/yacd/types"
)

func TestGetGitCommitDefault(t *testing.T) {
	// GitCommit defaults to "unknown" when not injected at build time.
	if got := GetGitCommit(); got != "unknown" {
		t.Errorf("GetGitCommit() = %q, expected \"unknown\"", got)
	}
}

func TestRunGenerateVersionFlag(t *testing.T) {
	// Capture stdout to verify version output.
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	// Reset and set the version flag.
	origShowVersion := showVersion
	showVersion = true
	defer func() {
		showVersion = origShowVersion
		os.Stdout = origStdout
	}()

	if err := runGenerate(rootCmd, []string{}); err != nil {
		t.Fatalf("runGenerate with version flag returned error: %v", err)
	}

	w.Close()
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		buf.ReadFrom(r)
		close(done)
	}()
	<-done

	output := buf.String()
	if !strings.Contains(output, "yacd version") {
		t.Errorf("Expected version output, got: %q", output)
	}
	if !strings.Contains(output, Version) {
		t.Errorf("Expected version %q in output, got: %q", Version, output)
	}
}

func TestPrepareReaderFileBranch(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "input.log")
	if err := os.WriteFile(inputPath, []byte("gcc -c main.c -o main.o\n"), 0644); err != nil {
		t.Fatal(err)
	}

	reader, cleanup, err := PrepareReader(&types.ParseOptions{InputFile: inputPath}, false)
	if err != nil {
		t.Fatalf("PrepareReader failed: %v", err)
	}
	defer cleanup()
	if reader == nil {
		t.Fatal("Expected non-nil reader")
	}
}

func TestPrepareReaderStdinBranch(t *testing.T) {
	// stdinHasData=true selects the os.Stdin branch.
	reader, cleanup, err := PrepareReader(&types.ParseOptions{}, true)
	if err != nil {
		t.Fatalf("PrepareReader failed: %v", err)
	}
	defer cleanup()
	if reader == nil {
		t.Fatal("Expected non-nil reader")
	}
}

func TestExecuteGenerationWithMissingFiles(t *testing.T) {
	// A log referencing a non-existent source file exercises the missing-files
	// warning path while still producing output.
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "compile_commands.json")

	options := types.ParseOptions{
		OutputFile: outputPath,
	}

	makeLog := "gcc -c /nonexistent/missing.c -o /nonexistent/missing.o\n"
	if err := ExecuteGeneration(&options, strings.NewReader(makeLog)); err != nil {
		t.Fatalf("ExecuteGeneration failed: %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("Expected output file to be created even with missing sources")
	}
}

func TestExecuteGenerationParseError(t *testing.T) {
	// ExecuteGeneration must surface a parse/read error from the reader.
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "compile_commands.json")
	options := types.ParseOptions{OutputFile: outputPath}

	err := ExecuteGeneration(&options, &failingReader{})
	if err == nil {
		t.Fatal("Expected error from failing reader, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse make log") {
		t.Errorf("Unexpected error: %v", err)
	}
}

type failingReader struct{}

func (failingReader) Read(p []byte) (int, error) { return 0, failingReadErr }

type failingReadErrType struct{}

func (failingReadErrType) Error() string { return "simulated read failure" }

var failingReadErr = failingReadErrType{}

func TestRunGenerateFullPipeline(t *testing.T) {
	// Drive runGenerate through the complete generation pipeline by setting the
	// package-level flag variables and forcing HasStdinData() to return false.
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "build.log")
	outputPath := filepath.Join(tmpDir, "compile_commands.json")
	if err := os.WriteFile(inputPath, []byte("gcc -c -Wall main.c -o main.o\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Redirect stdin to /dev/null so HasStdinData() reports no piped data.
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("Failed to open %s: %v", os.DevNull, err)
	}
	defer devNull.Close()

	origStdin := os.Stdin
	os.Stdin = devNull
	defer func() { os.Stdin = origStdin }()

	// Snapshot and restore the package-level flag variables.
	origInput, origOutput := inputFile, outputFile
	origMakeCmd := makeCommand
	defer func() {
		inputFile, outputFile, makeCommand = origInput, origOutput, origMakeCmd
	}()

	inputFile = inputPath
	outputFile = outputPath
	makeCommand = ""

	if err := runGenerate(rootCmd, nil); err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("Expected output file to be created")
	}
}

func TestRunGenerateMultipleInputError(t *testing.T) {
	// Setting both an input file and stdin data must produce a validation error.
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("Failed to open %s: %v", os.DevNull, err)
	}
	defer devNull.Close()

	origStdin := os.Stdin
	os.Stdin = devNull
	defer func() { os.Stdin = origStdin }()

	origInput, origMakeCmd := inputFile, makeCommand
	defer func() { inputFile, makeCommand = origInput, origMakeCmd }()

	// Both input sources set -> ValidateInputSources returns an error.
	inputFile = "some.log"
	makeCommand = "make all"
	err = runGenerate(rootCmd, nil)
	if err == nil {
		t.Fatal("Expected error for multiple input sources, got nil")
	}
	if !strings.Contains(err.Error(), "multiple input sources") {
		t.Errorf("Unexpected error: %v", err)
	}
}
