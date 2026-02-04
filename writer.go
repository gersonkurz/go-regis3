package regis3

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"golang.org/x/text/encoding/unicode"
)

// RegWriter handles exporting KeyEntry trees to .REG format.
type RegWriter struct {
	Header   string
	Options  ExportOptions
	useUtf16 bool
}

// NewRegWriter creates a new RegWriter.
// For Windows 5.00 format, use HeaderWindows5 and useUtf16=true.
// For REGEDIT4 format, use HeaderRegedit4 and useUtf16=false.
func NewRegWriter(header string, options ExportOptions, useUtf16 bool) *RegWriter {
	return &RegWriter{
		Header:   header,
		Options:  options,
		useUtf16: useUtf16,
	}
}

// Write exports the key entry tree to the provided writer.
func (w *RegWriter) Write(out io.Writer, root *KeyEntry) error {
	var writer io.StringWriter

	if w.useUtf16 {
		// Use standard library for robust UTF-16LE encoding with BOM.
		// Everything written to this encoder will be correctly transformed.
		enc := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder()
		utf16Writer := enc.Writer(out)
		
		// If out is already a string writer, we still want to wrap it.
		// But io.Writer returned by enc.Writer doesn't implement StringWriter.
		writer = &stringWriterWrapper{utf16Writer}
	} else {
		// Plain ANSI/UTF-8 export.
		if sw, ok := out.(io.StringWriter); ok {
			writer = sw
		} else {
			writer = &stringWriterWrapper{out}
		}
	}

	// Write header
	if err := w.writeLine(writer, w.Header); err != nil {
		return err
	}
	if err := w.writeLine(writer, ""); err != nil {
		return err
	}

	return w.exportRecursive(writer, root)
}


func (w *RegWriter) exportRecursive(writer io.StringWriter, key *KeyEntry) error {
	noEmptyKeys := w.Options.NoEmptyKeys
	skipThisEntry := false

	if noEmptyKeys {
		skipThis := true
		if key.defaultValue != nil || len(key.values) > 0 || key.removeFlag {
			skipThis = false
		}
		skipThisEntry = skipThis
	}

	if !skipThisEntry && key.name != "" {
		path := key.GetPath()
		if key.removeFlag {
			if err := w.writeLine(writer, fmt.Sprintf("[-%s]", path)); err != nil {
				return err
			}
		} else {
			if err := w.writeLine(writer, fmt.Sprintf("[%s]", path)); err != nil {
				return err
			}

			// Export default value first
			if key.defaultValue != nil {
				if err := w.exportValue(writer, key.defaultValue); err != nil {
					return err
				}
			}

			// Export named values in sorted order
			names := make([]string, 0, len(key.values))
			for name := range key.values {
				names = append(names, name)
			}
			sort.Strings(names)

			for _, name := range names {
				if err := w.exportValue(writer, key.values[name]); err != nil {
					return err
				}
			}
		}
		if err := w.writeLine(writer, ""); err != nil {
			return err
		}
	}

	// Export subkeys in sorted order
	subKeyNames := make([]string, 0, len(key.subKeys))
	for name := range key.subKeys {
		subKeyNames = append(subKeyNames, name)
	}
	sort.Strings(subKeyNames)

	for _, name := range subKeyNames {
		if err := w.exportRecursive(writer, key.subKeys[name]); err != nil {
			return err
		}
	}

	return nil
}

func (w *RegWriter) exportValue(writer io.StringWriter, val *ValueEntry) error {
	name := "@"
	if !val.IsDefaultValue() {
		name = fmt.Sprintf("\"%s\"", EscapeString(val.Name()))
	}

	if val.removeFlag {
		return w.writeLine(writer, fmt.Sprintf("%s=-", name))
	}

	switch val.kind {
	case RegSz:
		return w.writeLine(writer, fmt.Sprintf("%s=\"%s\"", name, EscapeString(val.GetString(""))))
	case RegDword:
		return w.writeLine(writer, fmt.Sprintf("%s=dword:%08x", name, val.GetDword(0)))
	case RegEscapedDword:
		// Variables like $$VAR$$ are stored as escaped strings
		return w.writeLine(writer, fmt.Sprintf("%s=dword:%s", name, val.GetString("")))
	case RegEscapedQword:
		// QWORD with variable is hex(b):$$VAR$$
		return w.writeLine(writer, fmt.Sprintf("%s=hex(b):%s", name, val.GetString("")))
	default:
		return w.writeHexValue(writer, val, name)
	}
}

func (w *RegWriter) writeHexValue(writer io.StringWriter, val *ValueEntry, name string) error {
	var sb strings.Builder
	sb.WriteString(name)
	if val.kind == RegBinary {
		sb.WriteString("=hex:")
	} else {
		sb.WriteString(fmt.Sprintf("=hex(%x):", val.kind))
	}

	data := val.data
	lineLen := sb.Len()

	for i, b := range data {
		if i > 0 {
			if lineLen >= 75 {
				sb.WriteString(",\\\r\n  ")
				lineLen = 2
			} else {
				sb.WriteString(",")
				lineLen++
			}
		}
		hexStr := fmt.Sprintf("%02x", b)
		sb.WriteString(hexStr)
		lineLen += 2
	}

	return w.writeLine(writer, sb.String())
}

func (w *RegWriter) writeLine(writer io.StringWriter, line string) error {
	_, err := writer.WriteString(line + "\r\n")
	return err
}

// EscapeString escapes backslashes and double quotes for .REG files.
func EscapeString(input string) string {
	var sb strings.Builder
	for _, r := range input {
		if r == '"' {
			sb.WriteString(`\"`)
		} else if r == '\\' {
			sb.WriteString(`\\`)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

type stringWriterWrapper struct {
	w io.Writer
}

func (s *stringWriterWrapper) WriteString(str string) (int, error) {
	return s.w.Write([]byte(str))
}
