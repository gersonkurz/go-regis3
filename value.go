package regis3

import (
	"encoding/binary"
	"unicode/utf16"
)

// ValueEntry represents a registry value with name, type, and data.
// Data is stored internally as UTF-16LE bytes (matching Windows registry format).
type ValueEntry struct {
	name       string
	kind       uint32
	data       []byte
	removeFlag bool
}

// NewValueEntry creates a new ValueEntry.
func NewValueEntry(name string) *ValueEntry {
	return &ValueEntry{
		name: name,
		kind: RegTypeUnknown,
	}
}

// IsDefaultValue checks if this is the default (unnamed) value for a key.
func (v *ValueEntry) IsDefaultValue() bool {
	return v.name == ""
}

// Name returns the value name.
func (v *ValueEntry) Name() string {
	return v.name
}

// Kind returns the registry type.
func (v *ValueEntry) Kind() uint32 {
	return v.kind
}

// Data returns the raw data bytes.
func (v *ValueEntry) Data() []byte {
	return v.data
}

// RemoveFlag checks if this value should be removed.
func (v *ValueEntry) RemoveFlag() bool {
	return v.removeFlag
}

// SetRemoveFlag sets the remove flag.
func (v *ValueEntry) SetRemoveFlag(flag bool) {
	v.removeFlag = flag
}

// SetNone sets the value as REG_NONE.
func (v *ValueEntry) SetNone() {
	v.data = nil
	v.kind = RegNone
}

// SetDword sets the value as REG_DWORD.
func (v *ValueEntry) SetDword(val uint32) {
	v.data = make([]byte, 4)
	binary.LittleEndian.PutUint32(v.data, val)
	v.kind = RegDword
}

// SetQword sets the value as REG_QWORD.
func (v *ValueEntry) SetQword(val uint64) {
	v.data = make([]byte, 8)
	binary.LittleEndian.PutUint64(v.data, val)
	v.kind = RegQword
}

// SetString sets the value as REG_SZ.
func (v *ValueEntry) SetString(val string) {
	v.assignFromUtf8String(val, RegSz)
}

// SetExpandString sets the value as REG_EXPAND_SZ.
func (v *ValueEntry) SetExpandString(val string) {
	v.assignFromUtf8String(val, RegExpandSz)
}

// SetMultiString sets the value as REG_MULTI_SZ.
func (v *ValueEntry) SetMultiString(stringsList []string) {
	v.kind = RegMultiSz
	v.data = make([]byte, 0)

	for _, str := range stringsList {
		utf16Chars := utf16.Encode([]rune(str))
		// Append chars + null terminator
		for _, char := range utf16Chars {
			v.appendUint16(char)
		}
		v.appendUint16(0)
	}
	// Final null terminator
	v.appendUint16(0)
}

// SetBinaryType sets raw binary data with specified type.
func (v *ValueEntry) SetBinaryType(kind uint32, data []byte) {
	v.kind = kind
	v.data = make([]byte, len(data))
	copy(v.data, data)

	// Clean up string types if necessary (ensure proper null termination)
	if kind == RegSz || kind == RegExpandSz {
		// Strip trailing nulls (2 bytes = 1 null char)
		for len(v.data) >= 2 && v.data[len(v.data)-1] == 0 && v.data[len(v.data)-2] == 0 {
			v.data = v.data[:len(v.data)-2]
		}
		// Re-add single null terminator
		v.appendUint16(0)
	}
}

// SetEscapedDwordValue sets escaped DWORD value (variable name stored as string).
func (v *ValueEntry) SetEscapedDwordValue(variableRef string) {
	v.assignFromUtf8String(variableRef, RegEscapedDword)
}

// SetEscapedQwordValue sets escaped QWORD value (variable name stored as string).
func (v *ValueEntry) SetEscapedQwordValue(variableRef string) {
	v.assignFromUtf8String(variableRef, RegEscapedQword)
}

// GetDword returns the value as DWORD.
func (v *ValueEntry) GetDword(defaultValue uint32) uint32 {
	if v.kind != RegDword || len(v.data) < 4 {
		return defaultValue
	}
	return binary.LittleEndian.Uint32(v.data)
}

// GetQword returns the value as QWORD.
func (v *ValueEntry) GetQword(defaultValue uint64) uint64 {
	if v.kind != RegQword || len(v.data) < 8 {
		return defaultValue
	}
	return binary.LittleEndian.Uint64(v.data)
}

// GetString returns the value as string.
func (v *ValueEntry) GetString(defaultValue string) string {
	if !IsStringType(v.kind) && v.kind != RegEscapedDword && v.kind != RegEscapedQword {
		return defaultValue
	}

	// Note: An empty string in UTF-16LE would be 2 bytes (null terminator 00 00),
	// not 0 bytes. However, SetString("") produces 2 bytes, so this check only
	// affects externally-set data with truly zero bytes. Kept for safety.
	if len(v.data) == 0 {
		return defaultValue
	}

	// Convert bytes to uint16s
	u16s := make([]uint16, len(v.data)/2)
	for i := 0; i < len(u16s); i++ {
		u16s[i] = binary.LittleEndian.Uint16(v.data[i*2:])
	}

	// Strip null terminator
	if len(u16s) > 0 && u16s[len(u16s)-1] == 0 {
		u16s = u16s[:len(u16s)-1]
	}

	return string(utf16.Decode(u16s))
}

// GetMultiString returns the value as a list of strings.
func (v *ValueEntry) GetMultiString() []string {
	if v.kind != RegMultiSz || len(v.data) == 0 {
		return nil
	}

	var result []string
	u16s := make([]uint16, len(v.data)/2)
	for i := 0; i < len(u16s); i++ {
		u16s[i] = binary.LittleEndian.Uint16(v.data[i*2:])
	}

	start := 0
	for i, char := range u16s {
		if char == 0 {
			if i > start {
				result = append(result, string(utf16.Decode(u16s[start:i])))
			}
			start = i + 1
			// Double null means end of list
			if i+1 < len(u16s) && u16s[i+1] == 0 {
				break
			}
		}
	}
	return result
}

// Internal helpers

func (v *ValueEntry) assignFromUtf8String(val string, kind uint32) {
	v.kind = kind
	utf16Chars := utf16.Encode([]rune(val))
	v.data = make([]byte, (len(utf16Chars)+1)*2)

	for i, char := range utf16Chars {
		binary.LittleEndian.PutUint16(v.data[i*2:], char)
	}
	// Null terminator is already zero-initialized by make()
}

func (v *ValueEntry) appendUint16(val uint16) {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, val)
	v.data = append(v.data, b...)
}
