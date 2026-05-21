package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gerryqd/yacd/types"
)

func TestPrepareOptions(t *testing.T) {
	tests := []struct {
		name             string
		inputFile        string
		outputFile       string
		makeCommand      string
		baseDir          string
		useRelativePaths bool
		verbose          bool
		checkFunc        func(types.ParseOptions) bool
	}{
		{
			name:       "Basic options",
			inputFile:  "input.log",
			outputFile: "output.json",
			checkFunc: func(o types.ParseOptions) bool {
				return o.InputFile == "input.log" && o.OutputFile == "output.json"
			},
		},
		{
			name:        "With make command",
			outputFile:  "output.json",
			makeCommand: "make clean all",
			verbose:     true,
			checkFunc: func(o types.ParseOptions) bool {
				return o.MakeCommand == "make clean all" && o.Verbose
			},
		},
		{
			name:             "Explicit base directory",
			inputFile:        "input.log",
			outputFile:       "output.json",
			baseDir:          "/project/root",
			useRelativePaths: true,
			checkFunc: func(o types.ParseOptions) bool {
				return o.BaseDir == "/project/root" && o.UseRelativePaths
			},
		},
		{
			name:             "Auto base from output dir",
			inputFile:        "input.log",
			outputFile:       "build/output.json",
			useRelativePaths: true,
			checkFunc: func(o types.ParseOptions) bool {
				return o.BaseDir == "build"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, err := PrepareOptions(tt.inputFile, tt.outputFile, tt.makeCommand,
				tt.baseDir, tt.useRelativePaths, tt.verbose, false)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !tt.checkFunc(options) {
				t.Errorf("Options check failed: %+v", options)
			}
		})
	}
}

func TestPrepareOptionsWithCurrentDirectory(t *testing.T) {
	options, err := PrepareOptions("input.log", "output.json", "", "", true, false, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if options.BaseDir == "" {
		t.Errorf("BaseDir should not be empty when using relative paths")
	}
	if !filepath.IsAbs(options.BaseDir) {
		t.Errorf("BaseDir should be absolute, got: %s", options.BaseDir)
	}
}

func TestPrepareReaderValidation(t *testing.T) {
	_, cleanup, err := PrepareReader(types.ParseOptions{
		InputFile: "/non/existent/file.log",
	}, false)
	if err == nil || !strings.Contains(err.Error(), "file does not exist") {
		t.Errorf("Expected 'file does not exist' error, got: %v", err)
	}
	if cleanup != nil {
		cleanup()
	}
}
