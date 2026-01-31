package regis3

// Registry Value Types
// These constants match the standard Windows API values found in winnt.h
const (
	RegNone                     = 0
	RegSz                       = 1
	RegExpandSz                 = 2
	RegBinary                   = 3
	RegDword                    = 4
	RegDwordLittleEndian        = 4
	RegDwordBigEndian           = 5
	RegLink                     = 6
	RegMultiSz                  = 7
	RegResourceList             = 8
	RegFullResourceDescriptor   = 9
	RegResourceRequirementsList = 10
	RegQword                    = 11
	RegQwordLittleEndian        = 11
)

// pnq-specific Constants
const (
	// RegTypeUnknown is a sentinel value indicating unknown/uninitialized registry value type.
	RegTypeUnknown = 0xFFFFFFFF // -1

	// RegEscapedDword is a marker for escaped DWORD variables ($$VAR$$ syntax in .REG files).
	// The actual value is stored as a string to be expanded at write time.
	RegEscapedDword = 0xFFFFFFFE // -2

	// RegEscapedQword is a marker for escaped QWORD variables ($$VAR$$ syntax in .REG files).
	RegEscapedQword = 0xFFFFFFFD // -3
)

// ParseOptions configures the behavior of the .REG file parser.
type ParseOptions struct {
	// AllowHashtagComments allows # style line comments.
	// Standard .REG files don't support comments; enable this for hand-edited files.
	AllowHashtagComments bool

	// AllowSemicolonComments allows ; style line comments.
	// Standard .REG files don't support comments; enable this for hand-edited files.
	AllowSemicolonComments bool

	// IgnoreWhitespaces is more relaxed about whitespace in .REG files.
	// Recommended for manually edited files that may have extra spaces.
	IgnoreWhitespaces bool

	// AllowVariableSubstitution allows variable names for non-string values.
	// Enables syntax like: "value"=dword:$$VARIABLE$$
	// Useful for installer templates where some values are filled in at install time.
	AllowVariableSubstitution bool
}

// ExportOptions configures the behavior of the .REG file exporter.
type ExportOptions struct {
	// NoEmptyKeys skips keys that have no values when exporting.
	NoEmptyKeys bool
}

// IsStringType checks if a registry value type is a string type (REG_SZ or REG_EXPAND_SZ).
func IsStringType(kind uint32) bool {
	return kind == RegSz || kind == RegExpandSz
}
