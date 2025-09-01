package errorutil

import (
	"fmt"
)

// WrapError wraps an error with additional context message
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// WrapErrorf wraps an error with formatted context message
func WrapErrorf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	message := fmt.Sprintf(format, args...)
	return fmt.Errorf("%s: %w", message, err)
}

// NewError creates a new error with message
func NewError(message string) error {
	return fmt.Errorf("%s", message)
}

// NewErrorf creates a new error with formatted message
func NewErrorf(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

// Common error wrapping functions for frequently used patterns

// WrapFileError wraps file operation errors
func WrapFileError(err error, operation, filename string) error {
	return WrapErrorf(err, "failed to %s file %s", operation, filename)
}

// WrapParseError wraps parsing errors
func WrapParseError(err error, what string) error {
	return WrapErrorf(err, "failed to parse %s", what)
}

// WrapValidationError wraps validation errors
func WrapValidationError(err error, what string) error {
	return WrapErrorf(err, "validation failed for %s", what)
}

// WrapExecutionError wraps command execution errors
func WrapExecutionError(err error, command string) error {
	return WrapErrorf(err, "failed to execute command %s", command)
}

// WrapGenerationError wraps generation errors
func WrapGenerationError(err error, what string) error {
	return WrapErrorf(err, "failed to generate %s", what)
}

// WrapConversionError wraps conversion errors
func WrapConversionError(err error, from, to string) error {
	return WrapErrorf(err, "failed to convert %s to %s", from, to)
}

// CreateFileNotExistError creates file not exist error
func CreateFileNotExistError(filename string) error {
	return NewErrorf("file does not exist: %s", filename)
}

// CreateInvalidArgumentError creates invalid argument error
func CreateInvalidArgumentError(argument, reason string) error {
	return NewErrorf("invalid argument %s: %s", argument, reason)
}

// CreateUnsupportedError creates unsupported operation error
func CreateUnsupportedError(operation string) error {
	return NewErrorf("unsupported operation: %s", operation)
}

// CreateEmptyInputError creates empty input error
func CreateEmptyInputError(inputType string) error {
	return NewErrorf("%s is empty", inputType)
}

// CreateMutuallyExclusiveError creates mutually exclusive error
func CreateMutuallyExclusiveError(option1, option2 string) error {
	return NewErrorf("options %s and %s are mutually exclusive", option1, option2)
}

// IsError checks if err is not nil
func IsError(err error) bool {
	return err != nil
}
