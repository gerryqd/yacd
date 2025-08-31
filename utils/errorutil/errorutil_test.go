package errorutil

import (
	"errors"
	"strings"
	"testing"
)

func TestWrapError(t *testing.T) {
	originalErr := errors.New("original error")

	tests := []struct {
		name             string
		err              error
		message          string
		expectedNil      bool
		expectedContains []string
	}{
		{
			name:             "Wrap non-nil error",
			err:              originalErr,
			message:          "additional context",
			expectedNil:      false,
			expectedContains: []string{"additional context", "original error"},
		},
		{
			name:        "Wrap nil error",
			err:         nil,
			message:     "additional context",
			expectedNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.err, tt.message)

			if tt.expectedNil {
				if result != nil {
					t.Errorf("WrapError() expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("WrapError() expected error, got nil")
				return
			}

			errStr := result.Error()
			for _, expected := range tt.expectedContains {
				if !strings.Contains(errStr, expected) {
					t.Errorf("WrapError() error %q does not contain %q", errStr, expected)
				}
			}
		})
	}
}

func TestWrapErrorf(t *testing.T) {
	originalErr := errors.New("original error")

	tests := []struct {
		name             string
		err              error
		format           string
		args             []interface{}
		expectedNil      bool
		expectedContains []string
	}{
		{
			name:             "Wrap with formatted message",
			err:              originalErr,
			format:           "failed to process %s with %d items",
			args:             []interface{}{"file.txt", 42},
			expectedNil:      false,
			expectedContains: []string{"failed to process file.txt with 42 items", "original error"},
		},
		{
			name:        "Wrap nil error with format",
			err:         nil,
			format:      "failed to process %s",
			args:        []interface{}{"file.txt"},
			expectedNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapErrorf(tt.err, tt.format, tt.args...)

			if tt.expectedNil {
				if result != nil {
					t.Errorf("WrapErrorf() expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("WrapErrorf() expected error, got nil")
				return
			}

			errStr := result.Error()
			for _, expected := range tt.expectedContains {
				if !strings.Contains(errStr, expected) {
					t.Errorf("WrapErrorf() error %q does not contain %q", errStr, expected)
				}
			}
		})
	}
}

func TestNewError(t *testing.T) {
	message := "test error message"
	result := NewError(message)

	if result == nil {
		t.Errorf("NewError() expected error, got nil")
		return
	}

	if result.Error() != message {
		t.Errorf("NewError() = %q, expected %q", result.Error(), message)
	}
}

func TestNewErrorf(t *testing.T) {
	format := "error with %s and %d"
	args := []interface{}{"string", 42}
	expected := "error with string and 42"

	result := NewErrorf(format, args...)

	if result == nil {
		t.Errorf("NewErrorf() expected error, got nil")
		return
	}

	if result.Error() != expected {
		t.Errorf("NewErrorf() = %q, expected %q", result.Error(), expected)
	}
}

func TestWrapFileError(t *testing.T) {
	originalErr := errors.New("permission denied")
	operation := "open"
	filename := "test.txt"

	result := WrapFileError(originalErr, operation, filename)

	if result == nil {
		t.Errorf("WrapFileError() expected error, got nil")
		return
	}

	errStr := result.Error()
	expectedContains := []string{"failed to open file test.txt", "permission denied"}

	for _, expected := range expectedContains {
		if !strings.Contains(errStr, expected) {
			t.Errorf("WrapFileError() error %q does not contain %q", errStr, expected)
		}
	}
}

func TestWrapParseError(t *testing.T) {
	originalErr := errors.New("invalid syntax")
	what := "make log"

	result := WrapParseError(originalErr, what)

	if result == nil {
		t.Errorf("WrapParseError() expected error, got nil")
		return
	}

	errStr := result.Error()
	expectedContains := []string{"failed to parse make log", "invalid syntax"}

	for _, expected := range expectedContains {
		if !strings.Contains(errStr, expected) {
			t.Errorf("WrapParseError() error %q does not contain %q", errStr, expected)
		}
	}
}

func TestCreateFileNotExistError(t *testing.T) {
	filename := "nonexistent.txt"
	result := CreateFileNotExistError(filename)

	if result == nil {
		t.Errorf("CreateFileNotExistError() expected error, got nil")
		return
	}

	expected := "file does not exist: nonexistent.txt"
	if result.Error() != expected {
		t.Errorf("CreateFileNotExistError() = %q, expected %q", result.Error(), expected)
	}
}

func TestCreateInvalidArgumentError(t *testing.T) {
	argument := "--input"
	reason := "file not found"
	result := CreateInvalidArgumentError(argument, reason)

	if result == nil {
		t.Errorf("CreateInvalidArgumentError() expected error, got nil")
		return
	}

	expected := "invalid argument --input: file not found"
	if result.Error() != expected {
		t.Errorf("CreateInvalidArgumentError() = %q, expected %q", result.Error(), expected)
	}
}

func TestCreateMutuallyExclusiveError(t *testing.T) {
	option1 := "--input"
	option2 := "--dry-run"
	result := CreateMutuallyExclusiveError(option1, option2)

	if result == nil {
		t.Errorf("CreateMutuallyExclusiveError() expected error, got nil")
		return
	}

	expected := "options --input and --dry-run are mutually exclusive"
	if result.Error() != expected {
		t.Errorf("CreateMutuallyExclusiveError() = %q, expected %q", result.Error(), expected)
	}
}

func TestIsError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Non-nil error",
			err:      errors.New("test error"),
			expected: true,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsError(tt.err)
			if result != tt.expected {
				t.Errorf("IsError() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
