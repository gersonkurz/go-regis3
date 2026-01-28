package regis3

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// RegParser parses Windows .REG files.
type RegParser struct {
	*ParserContext

	options              ImportOptions
	expectedHeader       string
	bracketsExpected     int
	result               *KeyEntry
	currentKey           *KeyEntry
	currentValue         *ValueEntry
	currentDataKind      uint32
}

// NewRegParser creates a new parser.
func NewRegParser(expectedHeader string, options ImportOptions) *RegParser {
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
	p.cleanup()
	return p.result, nil
}

// cleanup recursively unwraps single-child keys with no values.
func (p *RegParser) cleanup() {
	// Unwrap single-child keys with no values (match C# behavior: unwrap once?)
	// C# code:
	// if ((Result.Keys.Count == 1) && (Result.Values.Count == 0)) ...
	// It seems to be just ONE level.
	
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

func (p *RegParser) allowVariableNamesForNonStringVariables() bool {
	return p.options.HasFlag(AllowVariableNamesForNonStringVariables)
}

func (p *RegParser) allowSemicolonComments() bool {
	return p.options.HasFlag(AllowSemicolonComments)
}

func (p *RegParser) allowHashtagComments() bool {
	return p.options.HasFlag(AllowHashtagComments)
}

func (p *RegParser) ignoreWhitespaces() bool {
	return p.options.HasFlag(IgnoreWhitespaces)
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

	// Parse hex string to uint32
	// val might be empty or contain non-hex chars if logic is wrong, but caller guards it
	
	// C++ uses string::from_hex_string which handles "0x" prefix optionally?
	// .reg files usually have "dword:00000001"
	
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
	if len(input)%2 != 0 {
		input += "0"
	}

	data, _ := hex.DecodeString(input)
	return data
}


// State Handlers

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

func (p *RegParser) stateExpectCarriageReturn(r rune) (ParserState, error) {
	if r == '\r' {
		return p.stateExpectStartOfLine, nil
	} else if r == '\n' {
		return p.stateExpectStartOfLine, nil
	} else if r == ' ' || r == '\t' {
		// ignore trailing whitespace
		return p.stateExpectCarriageReturn, nil
	} else if r == '#' || r == ';' {
		return p.stateExpectCommentUntilEndOfLine, nil
	}
	return nil, fmt.Errorf("Expected carriage return but got '%c' instead", r)
}

func (p *RegParser) stateExpectNewline(r rune) (ParserState, error) {
	if r == '\n' {
		return p.stateExpectStartOfLine, nil
	}
	// C++: return true (stay in state) if not newline? 
	// "if (c == '\n') return set_current_state(...); return true;"
	// So if it's NOT newline, it just continues. 
	// Wait, standard Windows EOL is \r\n.
	// if stateExpectHeader returns stateExpectNewline, the next char SHOULD be \n.
	// If it's not, we might be in trouble or it's just ignoring junk?
	// C++ implementation:
	// bool state_expect_newline(char c) { if (c == '\n') set_state(...); return true; }
	// This implies it consumes anything until \n.
	return p.stateExpectNewline, nil
}

func (p *RegParser) stateExpectStartOfLine(r rune) (ParserState, error) {
	if r == '\r' || r == '\n' {
		// blank line
		return p.stateExpectStartOfLine, nil
	} else if r == '[' {
		p.BufferClear()
		p.bracketsExpected = 0
		return p.stateExpectKeyPath, nil
	} else if r == '@' {
		p.currentValue = p.currentKey.FindOrCreateValue("")
		return p.stateExpectEqualSign, nil
	} else if r == '"' {
		p.BufferClear()
		return p.stateExpectValueNameDefinition, nil
	} else if r == '#' && p.allowHashtagComments() {
		p.BufferClear()
		return p.stateExpectCommentUntilEndOfLine, nil
	} else if r == ';' && p.allowSemicolonComments() {
		p.BufferClear()
		return p.stateExpectCommentUntilEndOfLine, nil
	}
	return nil, fmt.Errorf("Unexpected character '%c' at start of line", r)
}

func (p *RegParser) stateExpectCommentUntilEndOfLine(r rune) (ParserState, error) {
	if r == '\n' {
		return p.stateExpectStartOfLine, nil
	}
	return p.stateExpectCommentUntilEndOfLine, nil
}

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

func (p *RegParser) stateExpectQuotedCharInValueName(r rune) (ParserState, error) {
	p.BufferAppend(r)
	return p.stateExpectValueNameDefinition, nil
}

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
	return nil, fmt.Errorf("Expected '=' but got '%c' instead", r)
}

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
			// fmt.Printf("DEBUG: stateExpectStartOfValueDefinition: typeName=%s, currentDataKind=%d\n", typeName, p.currentDataKind)
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
			return nil, fmt.Errorf("Value type '%s' not supported (buffer was '%s')", typeName, p.BufferAsString())
		}
	}
	p.BufferAppend(r)
	return p.stateExpectStartOfValueDefinition, nil
}

func (p *RegParser) stateExpectHexIntegerValue(r rune) (ParserState, error) {
	if r == '\r' {
		return p.decodeCurrentHexValue()
	} else if r == '\n' {
		// Handle LF directly
		_, err := p.decodeCurrentHexValue()
		if err != nil {
			return nil, err
		}
		// decodeCurrentHexValue returns stateExpectNewline, but we already have newline
		// so skip it?
		// decodeCurrentHexValue logic:
		// p.currentValue.SetDword(uint32(result))
		// return p.stateExpectNewline, nil
		
		// Since we hit \n, we are effectively done with the line.
		return p.stateExpectStartOfLine, nil
	} else if isHexDigit(r) {
		p.BufferAppend(r)
		return p.stateExpectHexIntegerValue, nil
	} else if p.isWhitespace(r) {
		// ignore
		return p.stateExpectHexIntegerValue, nil
	} else if r == '#' && p.allowHashtagComments() {
		p.decodeCurrentHexValue()
		return p.stateExpectCommentUntilEndOfLine, nil
	} else if r == ';' && p.allowSemicolonComments() {
		p.decodeCurrentHexValue()
		return p.stateExpectCommentUntilEndOfLine, nil
	} else if r == '$' && p.allowVariableNamesForNonStringVariables() {
		p.BufferAppend(r)
		return p.stateExpectVariableDefinedHexValue, nil
	}
	return nil, fmt.Errorf("'%c' is not a valid hex digit", r)
}

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
		p.decodeCurrentVariableDefinedHexValue()
		return p.stateExpectCommentUntilEndOfLine, nil
	} else if r == ';' && p.allowSemicolonComments() {
		p.decodeCurrentVariableDefinedHexValue()
		return p.stateExpectCommentUntilEndOfLine, nil
	}
	p.BufferAppend(r)
	return p.stateExpectVariableDefinedHexValue, nil
}

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

func (p *RegParser) stateExpectQuotedCharInStringValue(r rune) (ParserState, error) {
	p.BufferAppend(r)
	return p.stateExpectStringValueDefinition, nil
}

func (p *RegParser) stateExpectKeyPath(r rune) (ParserState, error) {
	if r == '[' {
		p.bracketsExpected++
		p.BufferAppend(r)
	} else if r == ']' {
		if p.bracketsExpected == 0 {
			p.currentKey = p.result.FindOrCreateKey(p.BufferAsString())
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

func (p *RegParser) stateExpectStartOfMultibyteValue(r rune) (ParserState, error) {
	if r == '\r' {
		return p.stateExpectMultibyteValueDefinition(r)
	} else if r == '\n' {
		// This is technically invalid (CR expected before LF for empty hex?) 
		// But let's handle it by delegating to definition
		return p.stateExpectMultibyteValueDefinition(r)
	} else if isHexDigit(r) {
		p.BufferAppend(r)
		return p.stateExpectMultibyteValueDefinition, nil
	} else if r == '$' {
		if p.allowVariableNamesForNonStringVariables() {
			p.BufferAppend(r)
			return p.stateExpectVariableDefinedHexValue, nil
		}
	}
	return nil, fmt.Errorf("Expected hex digit at start of binary value")
}

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
	} else if isHexDigit(r) {
		p.BufferAppend(r)
		return p.stateExpectMultibyteValueDefinition, nil
	}
	return nil, fmt.Errorf("Unexpected character '%c' in hex data", r)
}

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
	return nil, fmt.Errorf("Expected newline to follow trailing backslash")
}

