package regis3

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf16"
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
	// Write BOM if UTF-16LE
	if w.useUtf16 {
		if _, err := out.Write([]byte{0xFF, 0xFE}); err != nil {
			return err
		}
	}

	// Internal writer to handle encoding if needed
	var writer io.StringWriter
	if sw, ok := out.(io.StringWriter); ok {
		writer = sw
	} else {
		writer = &stringWriterWrapper{out}
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
	noEmptyKeys := w.Options.HasFlag(NoEmptyKeys)
	skipThisEntry := false

	if noEmptyKeys {
		skip_this := true
		if key.defaultValue != nil || len(key.values) > 0 || key.removeFlag {
			skip_this = false
		}
		skipThisEntry = skip_this
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
	if w.useUtf16 {
		// Manual UTF-16LE conversion
		runes := []rune(line + "\r\n")
		u16s := utf16.Encode(runes)
		b := make([]byte, len(u16s)*2)
		for i, u := range u16s {
			b[i*2] = byte(u)
			b[i*2+1] = byte(u >> 8)
		}
		// Since writer is io.StringWriter but we want to write bytes for UTF-16LE
		// we need to bypass the StringWriter and use the underlying Writer if possible.
		// However, w.writeLine is designed for the high-level logic.
		// Let's rethink. If useUtf16 is true, we should probably wrap the output in a encoder.
		// But for now, let's keep it simple.
		if wrapper, ok := writer.(*stringWriterWrapper); ok {
			_, err := wrapper.w.Write(b)
			return err
		}
		// If it's a real StringWriter (like strings.Builder), we can't easily write raw bytes 
		// that aren't valid UTF-8.
		// Actually, in Go, strings can contain arbitrary bytes.
		_, err := writer.WriteString(string(b))
		return err
	}
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
