package regis3

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf16"
)

const (
	HeaderRegedit4 = "REGEDIT4"
	HeaderWindows5 = "Windows Registry Editor Version 5.00"
)

// Parse parses a .REG file content string.
// It automatically detects the header (REGEDIT4 or Windows Registry Editor Version 5.00).
func Parse(content string, options ImportOptions) (*KeyEntry, error) {
	trimmed := strings.TrimSpace(content)
	
	var header string
	if strings.HasPrefix(trimmed, HeaderRegedit4) {
		header = HeaderRegedit4
	} else if strings.HasPrefix(trimmed, HeaderWindows5) {
		header = HeaderWindows5
	} else {
		// Fallback or error? 
		// Let's assume Windows 5.00 if valid but maybe missing header? 
		// No, strict parsing requires header.
		// Check C++: It requires header match.
		
		// If content has BOM, it might mess up prefix check if not handled.
		// But Go strings are UTF-8. Caller is responsible for decoding.
		// If we assume standard .reg files, let's try to find the header in the first line.
		
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
func ParseFile(filename string, options ImportOptions) (*KeyEntry, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Detect encoding and convert to UTF-8 string
	content, err := decodeRegFileContent(data)
	if err != nil {
		return nil, err
	}

	return Parse(content, options)
}

// decodeRegFileContent converts raw bytes to UTF-8 string.
// Supports UTF-16LE (BOM) and ANSI/ASCII.
func decodeRegFileContent(data []byte) (string, error) {
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		// UTF-16LE BOM
		return decodeUtf16Le(data[2:]), nil
	}
	// Assume ANSI/ASCII/UTF-8
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
