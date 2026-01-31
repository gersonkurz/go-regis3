# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

go-regis3 is a Go library for parsing, manipulating, and exporting Windows Registry (.REG) files. It is NOT a registry API wrapper - it's a complete .REG file parser and generator with diff/merge capabilities.

Sister implementations exist in C# (regdiff) and C++ (pnq::regis3) - all three share identical design philosophy and must produce compatible output.

## Build and Test Commands

```bash
# Run all tests
go test

# Run tests with verbose output
go test -v

# Run tests with production .REG files (optional)
go test -testdata=/path/to/reg/files
REGIS3_TESTDATA=/path/to/reg/files go test

# Format code (required before commits)
go fmt ./...

# Check formatting
gofmt -l .
```

## Architecture

### Package Structure (Single Package)

All code lives in the `regis3` package with no subpackages:

- `types.go` - Registry type constants (RegSz, RegDword, etc.) and option flags
- `key.go` - `KeyEntry` struct - registry key tree nodes with case-insensitive maps
- `value.go` - `ValueEntry` struct - registry values with UTF-16LE internal storage
- `parser.go` - Abstract state machine parser infrastructure (`ParserContext`)
- `reg_parser.go` - .REG file specific parser with 20+ state handlers
- `regis3.go` - Public API entry points: `Parse()`, `ParseFile()`
- `writer.go` - .REG file exporter (ANSI/Unicode output)
- `wix_writer.go` - WiX 4.x XML exporter for MSI authoring
- `wow.go` - WOW64 detection helpers

### Key Design Patterns

**UTF-16LE Internal Storage**: All string values are stored internally as UTF-16LE bytes (matching Windows registry). This ensures byte-for-byte round-trip fidelity.

**Case Insensitivity**: KeyEntry and ValueEntry preserve original names but use lowercase keys for internal maps:
```go
subKeys map[string]*KeyEntry  // key is lowercase(name)
name    string                // original name preserved
```

**State Machine Parser**: Character-by-character parsing with state transitions:
```go
type ParserState func(rune) (ParserState, error)
```

### Core Data Flow

1. **Parse**: `.REG` file → `RegParser` state machine → `KeyEntry` tree
2. **Manipulate**: Navigate/modify tree via `KeyEntry`/`ValueEntry` methods
3. **Export**: `KeyEntry` tree → `RegWriter` or `WixWriter` → output file

### Extended Types for Variable Substitution

The library supports `$$VAR$$` syntax in values:
- `RegEscapedDword` (0xFFFFFFFE) - DWORD with variable reference
- `RegEscapedQword` (0xFFFFFFFD) - QWORD with variable reference

## API Entry Points

```go
// Parsing
Parse(content string, options ImportOptions) (*KeyEntry, error)
ParseFile(filename string, options ImportOptions) (*KeyEntry, error)

// Writing
NewRegWriter(header string, options ExportOptions, useUtf16 bool) *RegWriter
NewWixWriter() *WixWriter
```

## Testing Strategy

- **Parity tests** (`parity_test.go`) - C# implementation compatibility
- **Unit tests** - Parser, writer, edge cases
- **Round-trip tests** - Parse → Modify → Export → Parse verification
- **Production file tests** - Against real .REG files (requires `-testdata` flag)

## Dependencies

No external dependencies - only Go standard library. This is intentional for easy integration.
