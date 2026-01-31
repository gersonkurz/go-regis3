package regis3

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf16"

	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"
)

const (
	HeaderRegedit4 = "REGEDIT4"
	HeaderWindows5 = "Windows Registry Editor Version 5.00"
)

// Parse parses a .REG file content string.
// It automatically detects the header (REGEDIT4 or Windows Registry Editor Version 5.00).
// Pass nil for options to use defaults, or provide a ParseOptions struct.
func Parse(content string, options *ParseOptions) (*KeyEntry, error) {
	if options == nil {
		options = &ParseOptions{}
	}

	// If the caller provided a UTF-8 BOM string, strip it here to keep
	// header detection lenient and to avoid false "missing header" errors.
	content = strings.TrimPrefix(content, "\ufeff")
	trimmed := strings.TrimSpace(content)

	var header string
	if strings.HasPrefix(trimmed, HeaderRegedit4) {
		header = HeaderRegedit4
	} else if strings.HasPrefix(trimmed, HeaderWindows5) {
		header = HeaderWindows5
	} else {
		// If content has BOM, it might mess up prefix check if not handled.
		// But Go strings are UTF-8. Caller is responsible for decoding.
		// Try to find the header in the first line.
		lines := strings.SplitN(trimmed, "\n", 2)
		if len(lines) > 0 {
			firstLine := strings.TrimSpace(lines[0])
			if firstLine == HeaderRegedit4 {
				header = HeaderRegedit4
			} else if firstLine == HeaderWindows5 {
				header = HeaderWindows5
			}
		}
	}

	if header == "" {
		return nil, fmt.Errorf("unknown or missing .REG header")
	}

	parser := NewRegParser(header, options)
	return parser.Parse(content)
}

// ParseFile reads and parses a .REG file.
// It handles UTF-16LE to UTF-8 conversion if necessary.
// Pass nil for options to use defaults, or provide a ParseOptions struct.
func ParseFile(filename string, options *ParseOptions) (*KeyEntry, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Detect encoding and convert to UTF-8 string
	content, err := decodeRegFileContent(data, options)
	if err != nil {
		return nil, err
	}

	return Parse(content, options)
}

// decodeRegFileContent converts raw bytes to UTF-8 string.
// Supports UTF-16LE (BOM) and ANSI/ASCII.
func decodeRegFileContent(data []byte, options *ParseOptions) (string, error) {
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		// UTF-16LE BOM
		return decodeUtf16Le(data[2:]), nil
	}
	// UTF-8 BOM: treat as UTF-8 and strip the BOM.
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return string(data[3:]), nil
	}

	// Optional ANSI decoding (REGEDIT4 legacy). This is best-effort and intentionally
	// lenient: if the code page is unknown, fall back to raw bytes.
	if options != nil && options.AnsiCodePage != "" {
		decoded, err := decodeWithCodePage(data, options.AnsiCodePage)
		if err != nil {
			return "", err
		}
		return decoded, nil
	}

	// Assume ANSI/ASCII/UTF-8 (best-effort).
	return string(data), nil
}

func decodeUtf16Le(data []byte) string {
	// Simple UTF-16LE to UTF-8
	// Make sure length is even
	if len(data)%2 != 0 {
		data = append(data, 0)
	}

	u16s := make([]uint16, len(data)/2)
	for i := 0; i < len(u16s); i++ {
		u16s[i] = uint16(data[i*2]) | uint16(data[i*2+1])<<8
	}

	// unicode/utf16 Decode returns []rune
	// string([]rune) converts to UTF-8
	return string(utf16.Decode(u16s))
}

func decodeWithCodePage(data []byte, codePage string) (string, error) {
	enc, err := ianaindex.MIME.Encoding(codePage)
	if err != nil {
		return "", err
	}
	if enc == nil {
		return "", fmt.Errorf("unknown ANSI code page: %s", codePage)
	}

	reader := transform.NewReader(bytes.NewReader(data), enc.NewDecoder())
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
