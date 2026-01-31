package regis3

import (
	"fmt"
	"strings"
)

// ParserState is a function representing a state in the state machine.
// It takes a rune and returns the next state, or an error.
type ParserState func(r rune) (ParserState, error)

// ParserContext provides the base functionality for a state-machine parser.
// It mimics the C++ abstract_parser class.
type ParserContext struct {
	Line   int
	Column int
	Index  int
	Buffer []rune

	input     []rune
	inputLen  int
	pos       int
	lastState ParserState
}

// NewParserContext creates a new ParserContext.
func NewParserContext() *ParserContext {
	return &ParserContext{
		Buffer: make([]rune, 0, 1024),
	}
}

// ParseText processes the input string starting with the given initial state.
func (p *ParserContext) ParseText(text string, initialState ParserState) error {
	p.input = []rune(text)
	p.inputLen = len(p.input)
	p.pos = 0
	p.Line = 1
	p.Column = 1
	p.Index = 0
	p.Buffer = p.Buffer[:0]

	state := initialState
	p.lastState = state

	for p.pos < p.inputLen {
		r := p.input[p.pos]

		var err error
		if state != nil {
			state, err = state(r)
			if err != nil {
				return p.FormatError(err)
			}
			p.lastState = state
		} else {
			break
		}

		if r == '\n' {
			p.Line++
			p.Column = 0
		}
		p.Column++
		p.Index++
		p.pos++
	}

	return nil
}

// LastState returns the parser's last known state function.
// This is used by concrete parsers to apply lenient EOF finalization.
func (p *ParserContext) LastState() ParserState {
	return p.lastState
}

// BufferAsString returns the current buffer contents as a string.
func (p *ParserContext) BufferAsString() string {
	return string(p.Buffer)
}

// BufferAppend appends a rune to the buffer.
func (p *ParserContext) BufferAppend(r rune) {
	p.Buffer = append(p.Buffer, r)
}

// BufferClear clears the buffer.
func (p *ParserContext) BufferClear() {
	p.Buffer = p.Buffer[:0]
}

// FormatError augments an error with context information (line, column, snippet).
// Returns a *ParseError with location and context details.
func (p *ParserContext) FormatError(err error) error {
	pe := &ParseError{
		Line:    p.Line,
		Column:  p.Column,
		Message: err.Error(),
	}

	// Extract context line
	start := p.pos
	if start >= p.inputLen {
		start = p.inputLen - 1
	}
	if start < 0 {
		start = 0
	}

	// Back up to beginning of line
	for start > 0 && p.input[start-1] != '\n' {
		start--
	}

	stop := p.pos
	for stop < p.inputLen && p.input[stop] != 0 && p.input[stop] != '\r' && p.input[stop] != '\n' {
		stop++
	}

	if start < p.inputLen && start <= stop {
		pe.Context = string(p.input[start:stop])

		// Build pointer line
		var sb strings.Builder
		for i := start; i < p.pos && i < p.inputLen; i++ {
			sb.WriteString(" ")
		}
		sb.WriteString("^")
		pe.ContextPointer = sb.String()
	}

	return pe
}

// Deprecated: Use ParseError fields directly for programmatic access.
// This function is kept for backward compatibility with code that uses fmt.Errorf style errors.
func formatErrorLegacy(line, column int, message string) error {
	return fmt.Errorf("Parser failed at line %d, col %d: %s", line, column, message)
}
