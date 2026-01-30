# go-regis3

A Go library for parsing, manipulating, and exporting Windows Registry (.REG) files.

## Why Does This Exist?

Most "registry libraries" wrap `RegOpenKeyEx` so you don't forget `RegCloseKey`. That's not what this is.

`go-regis3` is a complete toolkit for .REG file parsing, in-memory representation, and diff/merge operations:

- **State machine parser** that handles both REGEDIT4 (ANSI) and Windows Registry Editor 5.00 (UTF-16LE) formats
- **In-memory tree representation** (`KeyEntry`) with full navigation and manipulation
- **Diff/merge API**: `AskToAddValue()`, `AskToRemoveValue()` - designed for comparing registry snapshots
- **Round-trip fidelity**: parse → modify → export preserves structure
- **WiX XML generation** for MSI installer authoring
- Handles all the quirks - line continuation, hex encoding, escaped strings, variable substitution

## Sister Implementations

- **C#**: [github.com/gersonkurz/regdiff](https://github.com/gersonkurz/regdiff) - The original .NET implementation
- **C++**: [github.com/gersonkurz/pnq](https://github.com/gersonkurz/pnq) - Header-only C++ library (`pnq::regis3`)

All three implementations share the same design philosophy and API structure.

## Installation

```bash
go get github.com/gersonkurz/go-regis3
```

## Quick Start

```go
package main

import (
    "fmt"
    "strings"
    "github.com/gersonkurz/go-regis3"
)

func main() {
    // Parse a .REG file
    root, err := regis3.ParseFile("settings.reg", regis3.ImportOptionsNone)
    if err != nil {
        panic(err)
    }

    // Navigate the tree
    key := root.FindKey("Software\\MyApp")
    if key != nil {
        val := key.Values()["setting"]
        fmt.Println(val.GetString("default"))
    }

    // Export back to .REG format
    writer := regis3.NewRegWriter(regis3.HeaderWindows5, regis3.ExportOptionsNone, false)
    var sb strings.Builder
    writer.Write(&sb, root)
    fmt.Println(sb.String())
}
```

## API Overview

### Parsing

```go
// Parse from string
root, err := regis3.Parse(content, regis3.ImportOptionsNone)

// Parse from file (auto-detects UTF-16LE vs ANSI)
root, err := regis3.ParseFile("file.reg", regis3.ImportOptionsNone)
```

### Navigation

```go
// Find existing key (returns nil if not found)
key := root.FindKey("Software\\MyApp\\Settings")

// Find or create key path
key := root.FindOrCreateKey("Software\\MyApp\\Settings")

// Get values
values := key.Values()           // map[string]*ValueEntry
defaultVal := key.DefaultValue() // *ValueEntry or nil
subKeys := key.SubKeys()         // map[string]*KeyEntry
```

### Value Access

```go
val := key.Values()["myvalue"]

// Type-specific getters (with default fallback)
str := val.GetString("")
dword := val.GetDword(0)
qword := val.GetQword(0)
strs := val.GetMultiString()
raw := val.Data()

// Type checking
kind := val.Kind() // regis3.RegSz, RegDword, etc.
```

### Modification

```go
// Create/modify values
val := key.FindOrCreateValue("NewValue")
val.SetString("Hello")
val.SetDword(42)
val.SetExpandString("%PATH%\\bin")
val.SetMultiString([]string{"one", "two", "three"})
val.SetBinaryType(regis3.RegBinary, []byte{0xDE, 0xAD, 0xBE, 0xEF})

// Mark for deletion
key.SetRemoveFlag(true)
val.SetRemoveFlag(true)
```

### Diff/Merge Operations

```go
// For building diff trees
result.AskToAddKey(sourceKey)
result.AskToRemoveKey(sourceKey)
result.AskToAddValue(sourceKey, sourceValue)
result.AskToRemoveValue(sourceKey, sourceValue)
```

### Export

```go
// Export to .REG format
writer := regis3.NewRegWriter(regis3.HeaderWindows5, regis3.ExportOptionsNone, false)
writer.Write(file, root)

// Export to WiX XML
wixWriter := regis3.NewWixWriter()
wixWriter.Write(file, root)
```

## Import Options

| Option | Description |
|--------|-------------|
| `ImportOptionsNone` | Standard parsing |
| `AllowHashtagComments` | Allow `#` line comments |
| `AllowSemicolonComments` | Allow `;` line comments |
| `IgnoreWhitespaces` | Relaxed whitespace handling |
| `AllowVariableNamesForNonStringVariables` | Enable `dword:$$VAR$$` syntax |

## Export Options

| Option | Description |
|--------|-------------|
| `ExportOptionsNone` | Standard export |
| `NoEmptyKeys` | Skip keys with no values |

## Registry Value Types

Standard Windows types (matching `winnt.h`):

| Constant | Value | Description |
|----------|-------|-------------|
| `RegNone` | 0 | No type |
| `RegSz` | 1 | String |
| `RegExpandSz` | 2 | Expandable string (%VAR%) |
| `RegBinary` | 3 | Binary data |
| `RegDword` | 4 | 32-bit integer |
| `RegMultiSz` | 7 | Multi-string |
| `RegQword` | 11 | 64-bit integer |

Extended types for variable substitution:

| Constant | Description |
|----------|-------------|
| `RegEscapedDword` | DWORD with variable (e.g., `$$VAR$$`) |
| `RegEscapedQword` | QWORD with variable |

## Testing

```bash
# Run unit tests
go test

# Run with production test files
go test -testdata=/path/to/reg/files

# Or via environment variable
REGIS3_TESTDATA=/path/to/reg/files go test
```

## API Reference

### Package Functions

- `Parse(content string, options ImportOptions) (*KeyEntry, error)` - Parse .REG content
- `ParseFile(filename string, options ImportOptions) (*KeyEntry, error)` - Parse .REG file
- `NewRegWriter(header string, options ExportOptions, useUtf16 bool) *RegWriter` - Create .REG exporter
- `NewWixWriter() *WixWriter` - Create WiX XML exporter
- `EscapeString(input string) string` - Escape string for .REG format
- `IsStringType(kind uint32) bool` - Check if type is REG_SZ or REG_EXPAND_SZ

### KeyEntry Methods

| Method | Description |
|--------|-------------|
| `Name() string` | Key name (not full path) |
| `Parent() *KeyEntry` | Parent key |
| `SetParent(*KeyEntry)` | Set parent (for tree grafting) |
| `GetPath() string` | Full registry path |
| `SubKeys() map[string]*KeyEntry` | Child keys |
| `Values() map[string]*ValueEntry` | Named values |
| `DefaultValue() *ValueEntry` | Default (unnamed) value |
| `FindKey(path string) *KeyEntry` | Find existing key (read-only) |
| `FindOrCreateKey(path string) *KeyEntry` | Find or create key path |
| `FindOrCreateValue(name string) *ValueEntry` | Find or create value |
| `Clone(newParent *KeyEntry) *KeyEntry` | Deep copy |
| `RemoveFlag() bool` | Check if marked for deletion |
| `SetRemoveFlag(bool)` | Mark for deletion |
| `AskToAddKey(*KeyEntry) *KeyEntry` | Add key for diff/merge |
| `AskToRemoveKey(*KeyEntry) *KeyEntry` | Remove key for diff/merge |
| `AskToAddValue(*KeyEntry, *ValueEntry)` | Add value for diff/merge |
| `AskToRemoveValue(*KeyEntry, *ValueEntry)` | Remove value for diff/merge |

### ValueEntry Methods

| Method | Description |
|--------|-------------|
| `Name() string` | Value name |
| `IsDefaultValue() bool` | True if unnamed (default) value |
| `Kind() uint32` | Registry type constant |
| `Data() []byte` | Raw data bytes |
| `RemoveFlag() bool` | Check if marked for deletion |
| `SetRemoveFlag(bool)` | Mark for deletion |
| `GetString(default string) string` | Get as string |
| `GetDword(default uint32) uint32` | Get as DWORD |
| `GetQword(default uint64) uint64` | Get as QWORD |
| `GetMultiString() []string` | Get as string list |
| `SetNone()` | Set as REG_NONE |
| `SetString(string)` | Set as REG_SZ |
| `SetExpandString(string)` | Set as REG_EXPAND_SZ |
| `SetDword(uint32)` | Set as REG_DWORD |
| `SetQword(uint64)` | Set as REG_QWORD |
| `SetMultiString([]string)` | Set as REG_MULTI_SZ |
| `SetBinaryType(kind uint32, data []byte)` | Set raw binary with type |
| `SetEscapedDwordValue(string)` | Set DWORD variable reference |
| `SetEscapedQwordValue(string)` | Set QWORD variable reference |

### RegWriter

```go
type RegWriter struct {
    Header   string        // HeaderRegedit4 or HeaderWindows5
    Options  ExportOptions
}

func NewRegWriter(header string, options ExportOptions, useUtf16 bool) *RegWriter
func (w *RegWriter) Write(out io.Writer, root *KeyEntry) error
```

### WixWriter

```go
type WixWriter struct{}

func NewWixWriter() *WixWriter
func (w *WixWriter) Write(out io.Writer, root *KeyEntry) error
```

Generates WiX 4.x compatible XML fragments with:
- `<RegistryKey>` elements for key hierarchy
- `<RegistryValue>` elements for values
- `<RemoveRegistryKey>` for key deletions
- `<RemoveRegistryValue>` for value deletions

## License

MIT License - see LICENSE file for details.
