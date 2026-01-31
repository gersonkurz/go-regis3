package regis3

import (
	"fmt"
	"strings"
)

// ParseError represents an error that occurred during .REG file parsing.
// It includes location information (line, column) and context from the input.
type ParseError struct {
	// Line number where the error occurred (1-based).
	Line int

	// Column number where the error occurred (1-based).
	Column int

	// Message describing what went wrong.
	Message string

	// Context shows the problematic line from the input.
	Context string

	// ContextPointer is a line with a ^ pointing to the error position.
	ContextPointer string
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("parse error at line %d, col %d: %s", e.Line, e.Column, e.Message))

	if e.Context != "" {
		sb.WriteString("\n>> ")
		sb.WriteString(e.Context)
		if e.ContextPointer != "" {
			sb.WriteString("\n>> ")
			sb.WriteString(e.ContextPointer)
		}
	}

	return sb.String()
}

// Unwrap returns nil as ParseError doesn't wrap other errors.
func (e *ParseError) Unwrap() error {
	return nil
}

// IsParseError checks if an error is a ParseError and returns it.
// Returns nil if the error is not a ParseError.
func IsParseError(err error) *ParseError {
	if pe, ok := err.(*ParseError); ok {
		return pe
	}
	return nil
}
