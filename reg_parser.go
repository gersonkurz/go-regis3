package regis3

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// RegParser parses Windows .REG files.
type RegParser struct {
	*ParserContext

	options          *ParseOptions
	expectedHeader   string
	bracketsExpected int
	result           *KeyEntry
	currentKey       *KeyEntry
	currentValue     *ValueEntry
	currentDataKind  uint32
}

// NewRegParser creates a new parser.
func NewRegParser(expectedHeader string, options *ParseOptions) *RegParser {
	if options == nil {
		options = &ParseOptions{}
	}
	return &RegParser{
		ParserContext:  NewParserContext(),
		expectedHeader: expectedHeader,
		options:        options,
		result:         NewKeyEntry(nil, ""), // Root key
	}
}

// Parse parses the given content.
func (p *RegParser) Parse(content string) (*KeyEntry, error) {
	err := p.ParseText(content, p.stateExpectHeader)
	if err != nil {
		return nil, err
	}
	// Lenient EOF handling: regedit is forgiving, so we finalize any pending
	// tokens instead of failing on missing trailing newlines or incomplete lines.
	if err := p.finalizeAtEOF(); err != nil {
		return nil, err
	}
	p.cleanup()
	return p.result, nil
}

// cleanup recursively unwraps single-child keys with no values.
func (p *RegParser) cleanup() {
	// Unwrap single-child keys with no values (match C# behavior: unwrap once)
	if p.result != nil &&
		len(p.result.subKeys) == 1 &&
		len(p.result.values) == 0 &&
		p.result.defaultValue == nil {

		var child *KeyEntry
		for _, v := range p.result.subKeys {
			child = v
			break
		}
		p.result = child
		p.result.parent = nil // Detach
	}
}

// Helpers

func (p *RegParser) allowVariableSubstitution() bool {
	return p.options.AllowVariableSubstitution
}

func (p *RegParser) allowSemicolonComments() bool {
	return p.options.AllowSemicolonComments
}

func (p *RegParser) allowHashtagComments() bool {
	return p.options.AllowHashtagComments
}

func (p *RegParser) ignoreWhitespaces() bool {
	return p.options.IgnoreWhitespaces
}

func (p *RegParser) isWhitespace(r rune) bool {
	if p.ignoreWhitespaces() {
		return r == ' ' || r == '\t'
	}
	return false
}

func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') ||
		(r >= 'A' && r <= 'F') ||
		(r >= 'a' && r <= 'f')
}

func (p *RegParser) decodeCurrentHexValue() (ParserState, error) {
	val := p.BufferAsString()
	p.BufferClear()

	result, err := strconv.ParseUint(val, 16, 32)
	if err != nil {
		return nil, fmt.Errorf("'%s' is not a valid hex integer", val)
	}

	p.currentValue.SetDword(uint32(result))
	return p.stateExpectNewline, nil
}

func (p *RegParser) decodeCurrentVariableDefinedHexValue() (ParserState, error) {
	val := p.BufferAsString()
	if p.currentDataKind == RegQword {
		p.currentValue.SetEscapedQwordValue(val)
	} else {
		// Default to Dword
		p.currentValue.SetEscapedDwordValue(val)
	}
	p.BufferClear()
	return p.stateExpectNewline, nil
}

func (p *RegParser) createByteArrayFromBuffer() []byte {
	input := p.BufferAsString()

	// Pad to even length if necessary (though parser should enforce pairs usually)
	// Leniency: regedit accepts messy input; we mirror that by padding instead of failing.
	if len(input)%2 != 0 {
		input += "0"
	}

	data, _ := hex.DecodeString(input)
	return data
}

// finalizeAtEOF attempts to complete the current parse state when the file ends
// without a trailing newline. This is intentionally permissive to match regedit's
// lenient behavior and to tolerate legacy/bad data.
func (p *RegParser) finalizeAtEOF() error {
	state := p.LastState()
	if state == nil {
		return nil
	}

	statePtr := func(s ParserState) uintptr {
		if s == nil {
			return 0
		}
		return reflect.ValueOf(s).Pointer()
	}

	current := statePtr(state)
	if current == statePtr(p.stateExpectHeader) {
		// Allow header without trailing newline.
		header := p.BufferAsString()
		if header == "" {
			return nil
		}
		if header != p.expectedHeader {
			return fmt.Errorf(".REG file expected header '%s', got '%s' instead", p.expectedHeader, header)
		}
		return nil
	}
	if current == statePtr(p.stateExpectHexIntegerValue) {
		// Accept trailing DWORD without newline.
		if p.BufferAsString() != "" {
			_, err := p.decodeCurrentHexValue()
			return err
		}
	}
	if current == statePtr(p.stateExpectVariableDefinedHexValue) {
		// Accept trailing $$VAR$$ without newline.
		if p.BufferAsString() != "" {
			_, err := p.decodeCurrentVariableDefinedHexValue()
			return err
		}
	}
	if current == statePtr(p.stateExpectMultibyteValueDefinition) {
		// Accept trailing hex blob without newline.
		if p.BufferAsString() != "" && p.currentValue != nil {
			p.currentValue.SetBinaryType(p.currentDataKind, p.createByteArrayFromBuffer())
			p.BufferClear()
		}
	}
	if current == statePtr(p.stateExpectStringValueDefinition) {
		// Accept unterminated string at EOF (lenient).
		if p.currentValue != nil {
			p.currentValue.SetString(p.BufferAsString())
			p.BufferClear()
		}
	}
	if current == statePtr(p.stateExpectKeyPath) {
		// Accept key declaration missing closing bracket at EOF (lenient).
		if p.BufferAsString() != "" {
			p.currentKey = p.result.FindOrCreateKey(p.BufferAsString())
			p.currentKey.SetDeclared(true)
			p.BufferClear()
		}
	}

	return nil
}

// State Handlers

// stateExpectHeader validates the .REG file header (e.g., "REGEDIT4").
func (p *RegParser) stateExpectHeader(r rune) (ParserState, error) {
	if r == '\r' {
		header := p.BufferAsString()
		if header != p.expectedHeader {
			return nil, fmt.Errorf(".REG file expected header '%s', got '%s' instead", p.expectedHeader, header)
		}
		return p.stateExpectNewline, nil
	} else if r == '\n' {
		header := p.BufferAsString()
		if header != p.expectedHeader {
			return nil, fmt.Errorf(".REG file expected header '%s', got '%s' instead", p.expectedHeader, header)
		}
		return p.stateExpectStartOfLine, nil
	}
	p.BufferAppend(r)
	return p.stateExpectHeader, nil
}

// stateExpectCarriageReturn handles the end of a line, allowing for trailing whitespace or comments.
func (p *RegParser) stateExpectCarriageReturn(r rune) (ParserState, error) {
	if r == '\r' {
		return p.stateExpectStartOfLine, nil
	} else if r == '\n' {
		return p.stateExpectStartOfLine, nil
	} else if r == ' ' || r == '\t' {
		// ignore trailing whitespace
		return p.stateExpectCarriageReturn, nil
	} else if r == '#' {
		if !p.allowHashtagComments() {
			return nil, fmt.Errorf("'#' is not valid here; set AllowHashtagComments to use # as a comment marker")
		}
		return p.stateExpectCommentUntilEndOfLine, nil
	} else if r == ';' {
		if !p.allowSemicolonComments() {
			return nil, fmt.Errorf("';' is not valid here; set AllowSemicolonComments to use ; as a comment marker")
		}
		return p.stateExpectCommentUntilEndOfLine, nil
	}
	return nil, fmt.Errorf("expected carriage return but got '%c' instead", r)
}

// stateExpectNewline consumes CR/LF sequences to reach the next line.
func (p *RegParser) stateExpectNewline(r rune) (ParserState, error) {
	if r == '\n' {
		return p.stateExpectStartOfLine, nil
	}
	// Consume anything until \n (handles \r\n and other trailing chars)
	return p.stateExpectNewline, nil
}

// stateExpectStartOfLine determines the type of line being parsed (Key, Value, or Comment).
// Examples:
//
//	[HKEY_LOCAL_MACHINE\Software]  -> Key path
//	"MyValue"="Data"               -> Value definition
//	@="Default"                    -> Default value
//	# Comment                      -> Comment
func (p *RegParser) stateExpectStartOfLine(r rune) (ParserState, error) {
	if r == '\r' || r == '\n' {
		// blank line
		return p.stateExpectStartOfLine, nil
	} else if p.isWhitespace(r) {
		// Leniency: allow leading whitespace even at start-of-line.
		return p.stateExpectStartOfLine, nil
	} else if r == '[' {
		p.BufferClear()
		p.bracketsExpected = 0
		return p.stateExpectKeyPath, nil
	} else if r == '@' {
		if p.currentKey == nil {
			return nil, fmt.Errorf("value definition before any key declaration")
		}
		p.currentValue = p.currentKey.FindOrCreateValue("")
		return p.stateExpectEqualSign, nil
	} else if r == '"' {
		if p.currentKey == nil {
			return nil, fmt.Errorf("value definition before any key declaration")
		}
		p.BufferClear()
		return p.stateExpectValueNameDefinition, nil
	} else if r == '#' && p.allowHashtagComments() {
		p.BufferClear()
		return p.stateExpectCommentUntilEndOfLine, nil
	} else if r == ';' && p.allowSemicolonComments() {
		p.BufferClear()
		return p.stateExpectCommentUntilEndOfLine, nil
	}
	return nil, fmt.Errorf("unexpected character '%c' at start of line", r)
}

// stateExpectCommentUntilEndOfLine consumes characters until the end of the line.
func (p *RegParser) stateExpectCommentUntilEndOfLine(r rune) (ParserState, error) {
	if r == '\n' {
		return p.stateExpectStartOfLine, nil
	}
	return p.stateExpectCommentUntilEndOfLine, nil
}

// stateExpectValueNameDefinition parses the name of a registry value.
// Example: "MyValue"=...
func (p *RegParser) stateExpectValueNameDefinition(r rune) (ParserState, error) {
	if r == '"' {
		p.currentValue = p.currentKey.FindOrCreateValue(p.BufferAsString())
		return p.stateExpectEqualSign, nil
	} else if r == '\\' {
		return p.stateExpectQuotedCharInValueName, nil
	}
	p.BufferAppend(r)
	return p.stateExpectValueNameDefinition, nil
}

// stateExpectQuotedCharInValueName handles escaped characters within a value name.
func (p *RegParser) stateExpectQuotedCharInValueName(r rune) (ParserState, error) {
	p.BufferAppend(r)
	return p.stateExpectValueNameDefinition, nil
}

// stateExpectEqualSign waits for the '=' separator or '-' for deletion.
// Examples:
//
//	"Value"=...
//	"Value"=-
func (p *RegParser) stateExpectEqualSign(r rune) (ParserState, error) {
	if r == '=' {
		p.BufferClear()
		return p.stateExpectStartOfValueDefinition, nil
	} else if r == '-' {
		// "value"=- means delete
		p.currentValue.SetRemoveFlag(true)
		return p.stateExpectCarriageReturn, nil
	} else if p.isWhitespace(r) {
		return p.stateExpectEqualSign, nil
	}
	return nil, fmt.Errorf("expected '=' but got '%c' instead", r)
}

// stateExpectStartOfValueDefinition dispatches based on the value type.
// Examples:
//
//	...="String"     -> String
//	...=dword:0001   -> DWORD
//	...=hex:01,02    -> Binary
func (p *RegParser) stateExpectStartOfValueDefinition(r rune) (ParserState, error) {
	if r == '"' {
		p.BufferClear()
		return p.stateExpectStringValueDefinition, nil
	} else if r == '-' {
		// @=- means delete default value
		p.currentValue.SetRemoveFlag(true)
		return p.stateExpectCarriageReturn, nil
	} else if p.isWhitespace(r) {
		return p.stateExpectStartOfValueDefinition, nil
	} else if r == ':' {
		typeName := strings.ToLower(p.BufferAsString())
		p.BufferClear()

		if typeName == "dword" {
			p.currentDataKind = RegDword
			return p.stateExpectHexIntegerValue, nil
		} else if strings.HasPrefix(typeName, "hex(") && strings.HasSuffix(typeName, ")") {
			// hex(N): format
			tokens := strings.Split(typeName, "(")
			if len(tokens) >= 2 {
				hexStr := strings.TrimSuffix(tokens[1], ")")
				dataKind, err := strconv.ParseUint(hexStr, 16, 32)
				if err != nil {
					return nil, fmt.Errorf("'%s' is not a valid hex() kind", hexStr)
				}
				p.currentDataKind = uint32(dataKind)
				return p.stateExpectStartOfMultibyteValue, nil
			}
		} else if typeName == "hex" {
			p.currentDataKind = RegBinary
			return p.stateExpectStartOfMultibyteValue, nil
		} else {
			return nil, fmt.Errorf("value type '%s' is not supported", typeName)
		}
	}
	p.BufferAppend(r)
	return p.stateExpectStartOfValueDefinition, nil
}

// stateExpectHexIntegerValue parses a 32-bit hex integer (REG_DWORD).
// Example: ...=dword:00000001
func (p *RegParser) stateExpectHexIntegerValue(r rune) (ParserState, error) {
	if r == '\r' {
		return p.decodeCurrentHexValue()
	} else if r == '\n' {
		// Handle LF directly
		_, err := p.decodeCurrentHexValue()
		if err != nil {
			return nil, err
		}
		return p.stateExpectStartOfLine, nil
	} else if isHexDigit(r) {
		p.BufferAppend(r)
		return p.stateExpectHexIntegerValue, nil
	} else if p.isWhitespace(r) {
		// ignore
		return p.stateExpectHexIntegerValue, nil
	} else if r == '#' && p.allowHashtagComments() {
		if _, err := p.decodeCurrentHexValue(); err != nil {
			return nil, err
		}
		return p.stateExpectCommentUntilEndOfLine, nil
	} else if r == ';' && p.allowSemicolonComments() {
		if _, err := p.decodeCurrentHexValue(); err != nil {
			return nil, err
		}
		return p.stateExpectCommentUntilEndOfLine, nil
	} else if r == '$' && p.allowVariableSubstitution() {
		p.BufferAppend(r)
		return p.stateExpectVariableDefinedHexValue, nil
	}
	return nil, fmt.Errorf("'%c' is not a valid hex digit", r)
}

// stateExpectVariableDefinedHexValue handles escaped variables in hex values.
// Example: ...=dword:$$MY_VARIABLE$$
func (p *RegParser) stateExpectVariableDefinedHexValue(r rune) (ParserState, error) {
	if r == '\r' {
		return p.decodeCurrentVariableDefinedHexValue()
	} else if r == '\n' {
		// Handle LF directly
		_, err := p.decodeCurrentVariableDefinedHexValue()
		if err != nil {
			return nil, err
		}
		return p.stateExpectStartOfLine, nil
	} else if r == '#' && p.allowHashtagComments() {
		if _, err := p.decodeCurrentVariableDefinedHexValue(); err != nil {
			return nil, err
		}
		return p.stateExpectCommentUntilEndOfLine, nil
	} else if r == ';' && p.allowSemicolonComments() {
		if _, err := p.decodeCurrentVariableDefinedHexValue(); err != nil {
			return nil, err
		}
		return p.stateExpectCommentUntilEndOfLine, nil
	}
	p.BufferAppend(r)
	return p.stateExpectVariableDefinedHexValue, nil
}

// stateExpectStringValueDefinition parses a quoted string value.
// Example: ...="Hello World"
func (p *RegParser) stateExpectStringValueDefinition(r rune) (ParserState, error) {
	if r == '"' {
		p.currentValue.SetString(p.BufferAsString())
		return p.stateExpectCarriageReturn, nil
	} else if r == '\\' {
		return p.stateExpectQuotedCharInStringValue, nil
	}
	p.BufferAppend(r)
	return p.stateExpectStringValueDefinition, nil
}

// stateExpectQuotedCharInStringValue handles escaped characters within a string.
// Example: ...="Line 1\\nLine 2"
func (p *RegParser) stateExpectQuotedCharInStringValue(r rune) (ParserState, error) {
	p.BufferAppend(r)
	return p.stateExpectStringValueDefinition, nil
}

// stateExpectKeyPath parses a registry key path.
// Example: [HKEY_LOCAL_MACHINE\Software]
func (p *RegParser) stateExpectKeyPath(r rune) (ParserState, error) {
	if r == '[' {
		p.bracketsExpected++
		p.BufferAppend(r)
	} else if r == ']' {
		if p.bracketsExpected == 0 {
			p.currentKey = p.result.FindOrCreateKey(p.BufferAsString())
			p.currentKey.SetDeclared(true)
			return p.stateExpectCarriageReturn, nil
		} else {
			p.bracketsExpected--
			p.BufferAppend(r)
		}
	} else {
		p.BufferAppend(r)
	}
	return p.stateExpectKeyPath, nil
}

// stateExpectStartOfMultibyteValue parses the first digit of a hex data sequence.
func (p *RegParser) stateExpectStartOfMultibyteValue(r rune) (ParserState, error) {
	if r == '\r' {
		return p.stateExpectMultibyteValueDefinition(r)
	} else if r == '\n' {
		return p.stateExpectMultibyteValueDefinition(r)
	} else if isHexDigit(r) {
		p.BufferAppend(r)
		return p.stateExpectMultibyteValueDefinition, nil
	} else if r == '$' {
		if p.allowVariableSubstitution() {
			p.BufferAppend(r)
			return p.stateExpectVariableDefinedHexValue, nil
		}
	}
	return nil, fmt.Errorf("expected hex digit at start of binary value")
}

// stateExpectMultibyteValueDefinition parses comma-separated hex bytes.
// Example: ...=hex:01,02,03
func (p *RegParser) stateExpectMultibyteValueDefinition(r rune) (ParserState, error) {
	if r == ',' {
		// separator, ignore
		return p.stateExpectMultibyteValueDefinition, nil
	} else if r == '\\' {
		return p.stateExpectNewlineFollowedByMultibyte, nil
	} else if r == ' ' || r == '\t' {
		// whitespace
		return p.stateExpectMultibyteValueDefinition, nil
	} else if r == '\r' {
		p.currentValue.SetBinaryType(p.currentDataKind, p.createByteArrayFromBuffer())
		p.BufferClear()
		return p.stateExpectNewline, nil
	} else if r == '\n' {
		// Handle LF directly
		p.currentValue.SetBinaryType(p.currentDataKind, p.createByteArrayFromBuffer())
		p.BufferClear()
		return p.stateExpectStartOfLine, nil
	} else if r == '#' && p.allowHashtagComments() {
		// Allow comments after multibyte values (lenient).
		p.currentValue.SetBinaryType(p.currentDataKind, p.createByteArrayFromBuffer())
		p.BufferClear()
		return p.stateExpectCommentUntilEndOfLine, nil
	} else if r == ';' && p.allowSemicolonComments() {
		// Allow comments after multibyte values (lenient).
		p.currentValue.SetBinaryType(p.currentDataKind, p.createByteArrayFromBuffer())
		p.BufferClear()
		return p.stateExpectCommentUntilEndOfLine, nil
	} else if isHexDigit(r) {
		p.BufferAppend(r)
		return p.stateExpectMultibyteValueDefinition, nil
	}
	return nil, fmt.Errorf("unexpected character '%c' in hex data", r)
}

// stateExpectNewlineFollowedByMultibyte handles backslash line continuations in hex data.
// Example:
//
//	"Binary"=hex:01,02,\
//	  03,04
func (p *RegParser) stateExpectNewlineFollowedByMultibyte(r rune) (ParserState, error) {
	if r == '\r' {
		// ignore
		return p.stateExpectNewlineFollowedByMultibyte, nil
	} else if r == '\n' {
		return p.stateExpectMultibyteValueDefinition, nil
	} else if r == ' ' || r == '\t' {
		// ignore trailing spaces after backslash
		return p.stateExpectNewlineFollowedByMultibyte, nil
	}
	return nil, fmt.Errorf("expected newline to follow trailing backslash")
}

