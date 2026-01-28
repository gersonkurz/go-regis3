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

	input       []rune
	inputLen    int
	pos         int
	lastFile    string
	syntaxError error
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
	p.syntaxError = nil

	state := initialState

	for p.pos < p.inputLen {
		r := p.input[p.pos]
		
		// In C++, the loop increments ptr AFTER calling state.
		// So state sees current char.
		
		var err error
		if state != nil {
			state, err = state(r)
			if err != nil {
				return p.FormatError(err)
			}
		} else {
			// Should not happen if loop continues
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
func (p *ParserContext) FormatError(err error) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Parser failed at line %d, col %d:\n", p.Line, p.Column))
	
	if p.lastFile != "" {
		sb.WriteString(fmt.Sprintf("- in '%s'\n", p.lastFile))
	}

	sb.WriteString(err.Error())
	sb.WriteString("\n")

	// Show context
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
		lineStr := string(p.input[start:stop])
		sb.WriteString(">> ")
		sb.WriteString(lineStr)
		sb.WriteString("\n")
		
		sb.WriteString(">> ")
		for i := start; i < p.pos && i < p.inputLen; i++ {
			sb.WriteString(" ")
		}
		sb.WriteString("^")
		sb.WriteString("\n")
	}

	return fmt.Errorf("%s", sb.String())
}
