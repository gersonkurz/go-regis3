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

// ImportOptions flags for .REG file parsing
type ImportOptions uint32

const (
	ImportOptionsNone ImportOptions = 0

	// AllowHashtagComments allows # style line comments
	AllowHashtagComments ImportOptions = 1

	// AllowSemicolonComments allows ; style line comments
	AllowSemicolonComments ImportOptions = 2

	// IgnoreWhitespaces is more relaxed about whitespaces in .REG files.
	// Recommended for manually edited files.
	IgnoreWhitespaces ImportOptions = 4

	// AllowVariableNamesForNonStringVariables allows variable names for non-string variables.
	// Enables syntax like: "value"=dword:$$VARIABLE$$
	AllowVariableNamesForNonStringVariables ImportOptions = 8
)

// Helper methods for ImportOptions
func (o ImportOptions) HasFlag(flag ImportOptions) bool {
	return (o & flag) == flag
}

// ExportOptions flags for .REG file generation
type ExportOptions uint32

const (
	ExportOptionsNone ExportOptions = 0

	// NoEmptyKeys skips keys that have no values.
	NoEmptyKeys ExportOptions = 1
)

// Helper methods for ExportOptions
func (o ExportOptions) HasFlag(flag ExportOptions) bool {
	return (o & flag) == flag
}

// IsStringType checks if a registry value type is a string type (REG_SZ or REG_EXPAND_SZ).
func IsStringType(kind uint32) bool {
	return kind == RegSz || kind == RegExpandSz
}
