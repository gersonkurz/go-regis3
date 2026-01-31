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
    // Parse a .REG file (pass nil for default options)
    root, err := regis3.ParseFile("settings.reg", nil)
    if err != nil {
        panic(err)
    }

    // Navigate the tree
    key := root.FindKey("Software\\MyApp")
    if key != nil {
        val := key.Values()["setting"]
        fmt.Println(val.GetString("default"))
    }

    // Export back to .REG format (UTF-16LE for Windows 5.00 format)
    writer := regis3.NewRegWriter(regis3.HeaderWindows5, regis3.ExportOptions{}, true)
    var sb strings.Builder
    writer.Write(&sb, root)
    fmt.Println(sb.String())
}
```

## API Overview

### Getting Registry Data Into Memory

Your first decision is where the .REG content comes from. You have two options:

**From a file on disk** - the common case when you're working with exported registry snapshots:

```go
root, err := regis3.ParseFile("settings.reg", nil)
```

`ParseFile` handles encoding detection automatically. Windows .REG files come in two flavors: ANSI (older `REGEDIT4` format) and UTF-16LE with BOM (modern `Windows Registry Editor Version 5.00` format). You don't need to care which one you have.

**From a string** - useful when the content doesn't live on disk. Maybe you extracted it from a ZIP archive, received it over a network, or generated it programmatically:

```go
content := `Windows Registry Editor Version 5.00

[HKEY_CURRENT_USER\Software\MyApp]
"Setting"="Value"
`
root, err := regis3.Parse(content, nil)
```

Both functions return a `*KeyEntry` - the root of a tree representing the registry structure.

### Understanding the Tree Structure

A parsed .REG file becomes a tree of `KeyEntry` nodes. Each key can have:
- **Subkeys** - child `KeyEntry` nodes (like folders)
- **Values** - `ValueEntry` nodes containing actual data (like files)
- **A default value** - the unnamed `@` value that every key can have

Think of it like a filesystem: keys are directories, values are files, and the default value is like a directory's own metadata.

The root you get back typically represents a registry hive like `HKEY_CURRENT_USER` or `HKEY_LOCAL_MACHINE`. If your .REG file contains multiple hives, the root will be an unnamed container with each hive as a child.

### Navigating to a Specific Key

If you know the path you're looking for, use `FindKey`:

```go
key := root.FindKey("Software\\MyApp\\Settings")
if key == nil {
    // Key doesn't exist in this .REG file
}
```

The path is relative to whatever `root` represents. If `root` is `HKEY_CURRENT_USER`, then `FindKey("Software\\MyApp")` looks for `HKEY_CURRENT_USER\Software\MyApp`.

`FindKey` is read-only - it returns `nil` if the path doesn't exist rather than creating it. This is what you want when exploring an existing file.

### Enumerating Keys and Values

Often you don't know exactly what's in a .REG file and need to explore it. The tree exposes its contents through accessor methods:

```go
// Iterate over all subkeys
for name, subkey := range key.SubKeys() {
    fmt.Printf("Subkey: %s (full path: %s)\n", name, subkey.GetPath())
}

// Iterate over all named values
for name, value := range key.Values() {
    fmt.Printf("Value: %s = %v\n", name, value.GetString(""))
}

// Check for the default value (the @ value in .REG syntax)
if def := key.DefaultValue(); def != nil {
    fmt.Printf("Default value: %s\n", def.GetString(""))
}
```

Note: The map keys are **lowercase** for case-insensitive lookup, but each entry preserves its original name via `Name()`. The Windows Registry is case-insensitive but case-preserving, and go-regis3 mirrors this behavior.

### Reading Value Data

Values have different types (string, DWORD, binary, etc.), and you access them through type-specific getters. Each getter takes a default value to return if the type doesn't match:

```go
// String values (REG_SZ, REG_EXPAND_SZ)
str := value.GetString("fallback")

// Integer values
dword := value.GetDword(0)    // REG_DWORD (32-bit)
qword := value.GetQword(0)    // REG_QWORD (64-bit)

// Multi-string (REG_MULTI_SZ) - returns nil if wrong type
strings := value.GetMultiString()

// Raw bytes for anything else
raw := value.Data()
```

If you need to check the type explicitly:

```go
switch value.Kind() {
case regis3.RegSz:
    // It's a string
case regis3.RegDword:
    // It's a 32-bit integer
case regis3.RegBinary:
    // It's raw binary data
// ... etc
}
```

### Creating and Modifying Data

When you want to build or modify a registry tree, use `FindOrCreateKey` and `FindOrCreateValue`. Unlike the read-only `FindKey`, these create missing nodes:

```go
// This creates the entire path if it doesn't exist
key := root.FindOrCreateKey("Software\\MyApp\\Settings")

// Create or update a value
val := key.FindOrCreateValue("InstallPath")
val.SetString("C:\\Program Files\\MyApp")

// Other value types
key.FindOrCreateValue("LaunchCount").SetDword(42)
key.FindOrCreateValue("SearchPaths").SetMultiString([]string{"C:\\bin", "D:\\tools"})
key.FindOrCreateValue("License").SetBinaryType(regis3.RegBinary, licenseBytes)
```

To mark something for deletion (which exports as `[-KEY]` or `"value"=-`):

```go
key.SetRemoveFlag(true)    // Marks the entire key for deletion
value.SetRemoveFlag(true)  // Marks just this value for deletion
```

### Variable Substitution (`$$VAR$$` Syntax)

Sometimes you need registry values that aren't fully known until install time. Consider an installer that writes mostly fixed settings, but some values - like a license key, installation path, or machine-specific ID - come from an INI file or are generated during setup.

go-regis3 supports a `$$VARIABLE$$` syntax for this. Instead of a literal value, you write a placeholder that your installer resolves later:

```reg
[HKEY_LOCAL_MACHINE\SOFTWARE\MyApp]
"InstallPath"="C:\\Program Files\\MyApp"
"LicenseKey"=dword:$$LICENSE_DWORD$$
"MachineId"=hex(b):$$MACHINE_QWORD$$
```

To parse files containing this syntax, enable the option:

```go
root, err := regis3.ParseFile("template.reg", &regis3.ParseOptions{
    AllowVariableSubstitution: true,
})
```

The parser stores these as special types (`RegEscapedDword`, `RegEscapedQword`) with the variable name as the payload. When you export, the `$$VAR$$` syntax is preserved:

```go
val := key.FindOrCreateValue("LicenseKey")
val.SetEscapedDwordValue("$$LICENSE_DWORD$$")  // Stored as variable reference, not a number
```

Your installer then:
1. Parses the template .REG file
2. Walks the tree looking for escaped values (`val.Kind() == regis3.RegEscapedDword`)
3. Replaces them with actual values from your INI file or generator
4. Exports the final .REG or applies it to the registry

This keeps your .REG templates readable and version-controllable while deferring machine-specific values to install time.

### Building Diff Trees

The `AskTo*` methods are designed for comparing two registry snapshots and building a diff. They copy nodes from a source tree into a result tree:

```go
diff := regis3.NewKeyEntry(nil, "")  // Empty result tree

// When comparing old vs new, for each difference:
diff.AskToAddKey(newKey)           // Key was added
diff.AskToRemoveKey(oldKey)        // Key was removed
diff.AskToAddValue(parentKey, newValue)     // Value was added/changed
diff.AskToRemoveValue(parentKey, oldValue)  // Value was removed
```

The result is a tree containing only the changes, which you can export to create a .REG file that transforms "old" into "new".

### Exporting Back to .REG Format

Once you have a tree (parsed, modified, or built from scratch), export it. The header and encoding must match:

```go
// Modern format: UTF-16LE with BOM (recommended)
writer := regis3.NewRegWriter(regis3.HeaderWindows5, regis3.ExportOptions{}, true)

// Legacy format: ANSI encoding
writer := regis3.NewRegWriter(regis3.HeaderRegedit4, regis3.ExportOptions{}, false)

// Skip keys that have no values
writer := regis3.NewRegWriter(regis3.HeaderWindows5, regis3.ExportOptions{NoEmptyKeys: true}, true)

file, _ := os.Create("output.reg")
defer file.Close()
writer.Write(file, root)
```

### Exporting to WiX XML for MSI Installers

If you're building Windows installers with [WiX](https://wixtoolset.org/) (Windows Installer XML), you often need to include registry operations. Instead of hand-writing XML fragments, you can author your registry settings as a .REG file - which is easier to read, test, and diff - then convert it to WiX format:

```go
wixWriter := regis3.NewWixWriter()
wixWriter.Write(file, root)
```

Given this .REG content:

```reg
[HKEY_LOCAL_MACHINE\SOFTWARE\MyApp]
"InstallPath"="C:\Program Files\MyApp"
"Version"=dword:00000001

[-HKEY_LOCAL_MACHINE\SOFTWARE\MyApp\OldFeature]
```

The WiX writer produces:

```xml
<RemoveRegistryKey Action="removeOnInstall" Root="HKLM" Key="SOFTWARE\MyApp\OldFeature"/>
<RegistryKey Root="HKLM" Key="SOFTWARE\MyApp">
  <RegistryValue Name="InstallPath" Value="C:\Program Files\MyApp" Type="string"/>
  <RegistryValue Name="Version" Value="1" Type="integer"/>
</RegistryKey>
```

The output uses WiX 4.x syntax and handles:
- **Key hierarchy** - nested `<RegistryKey>` elements
- **All value types** - strings, integers, binary, multi-string, expandable strings
- **Deletions** - `<RemoveRegistryKey>` and `<RemoveRegistryValue>` elements
- **Root mapping** - `HKEY_LOCAL_MACHINE` → `HKLM`, `HKEY_CURRENT_USER` → `HKCU`, etc.

The writer does two passes: first emitting all removal operations (so they execute before additions), then the key/value additions as a nested hierarchy. This matches how you'd typically structure WiX components.

## Parse Options

Pass `nil` for default parsing behavior, or provide a `*ParseOptions` struct:

```go
type ParseOptions struct {
    AllowHashtagComments      bool  // Allow # line comments
    AllowSemicolonComments    bool  // Allow ; line comments
    IgnoreWhitespaces         bool  // Relaxed whitespace handling
    AllowVariableSubstitution bool  // Enable dword:$$VAR$$ syntax
    AnsiCodePage              string // Optional ANSI code page (e.g., "windows-1252") for REGEDIT4 files
}
```

Example with multiple options:

```go
root, err := regis3.Parse(content, &regis3.ParseOptions{
    AllowHashtagComments: true,
    IgnoreWhitespaces:    true,
})
```

## Export Options

Pass `ExportOptions{}` for default export behavior:

```go
type ExportOptions struct {
    NoEmptyKeys bool  // Skip keys that have no values
}
```

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

- `Parse(content string, options *ParseOptions) (*KeyEntry, error)` - Parse .REG content
- `ParseFile(filename string, options *ParseOptions) (*KeyEntry, error)` - Parse .REG file
- `NewRegWriter(header string, options ExportOptions, useUtf16 bool) *RegWriter` - Create .REG exporter
- `NewWixWriter() *WixWriter` - Create WiX XML exporter
- `EscapeString(input string) string` - Escape string for .REG format
- `IsStringType(kind uint32) bool` - Check if type is REG_SZ or REG_EXPAND_SZ
- `IsParseError(err error) *ParseError` - Check if error is a ParseError

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
