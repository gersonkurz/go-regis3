# go-regis3 Design Document

**Author:** Gemini (Google)
**Date:** 2026-01-28
**Status:** In Progress

## 1. Overview

`go-regis3` is a Go port of the `regis3` library (originally C#, then C++). It provides functionality to parse, modify, and export Windows Registry files (`.reg`) and interact with the Windows Registry.

The primary goal is **strict compatibility** with the existing C# and C++ implementations to ensure `msis` (the MSI creation tool) produces identical output.

## 2. Architecture

The Go implementation mirrors the C++ `pnq::regis3` architecture, utilizing a state machine parser for robustness against the complexities of the .REG file format.

### 2.1. Package Structure

```
go-regis3/
├── regis3.go           # Main entry points (Parse, ParseFile)
├── types.go            # Registry constants (RegSz, ImportOptions, etc.)
├── value.go            # ValueEntry struct and methods (UTF-16LE internal storage)
├── key.go              # KeyEntry struct and methods (Case-insensitive logic)
├── parser.go           # Abstract state machine parser context
├── reg_parser.go       # .REG file specific parser implementation
├── writer.go           # .REG file writer (ANSI/Unicode)
├── wix_writer.go       # WiX XML output generator
└── regis3_test.go      # Tests
```

### 2.2. Core Data Structures

*   **Internal Storage:** `ValueEntry` stores data as `[]byte` (UTF-16LE for strings, little-endian for integers). This ensures byte-for-byte fidelity with Windows registry formats and simplifies round-trip writing.
*   **Value Types:**
    -   `RegSz`, `RegExpandSz`: Stored as UTF-16LE + null terminator.
    -   `RegMultiSz`: Stored as multiple UTF-16LE strings, each null-terminated, with a final double null.
    -   `RegDword`, `RegQword`: Stored as 4 or 8 bytes in little-endian.
    -   `RegBinary`: Raw bytes.
    -   `RegEscapedDword`, `RegEscapedQword`: Custom types for `$VAR$` syntax, stored as UTF-16LE strings internally but flagged for special formatting during export.
*   **Case Insensitivity:** `KeyEntry` uses lowercase keys for its internal `subKeys` and `values` maps to handle Windows registry case-insensitivity, but preserves the original name in the `name` field.

### 2.3. The Parser (`parser.go` & `reg_parser.go`)

We will port the `abstract_parser` and `regfile_parser` classes from the C++ implementation.

**Format Specifications:**
-   **Line Continuation:** Backslash (`\`) at EOL continues the value definition.
-   **Default Value:** `@=` maps to empty string name.
-   **Delete Markers:** `[-Key]` and `"Val"=-` set `RemoveFlag`.
-   **Encoding:** UTF-16LE BOM detection with fallback to ANSI (REGEDIT4).
-   **Variables:** `$VAR$` preserved in `RegEscaped*` types.
-   **Unwrapping:** Single-root keys without values are unwrapped to match C# behavior.

**State Machine Approach:**
The parser maintains a `currentState` function pointer (or method value in Go). It reads the input character by character, and the current state function determines the transition.


## 3. Implementation Milestones

### Phase 1: Foundation (Completed)
- [x] Define `types.go`, `key.go`, `value.go`.
- [x] Implement `parser.go` (state machine scaffolding).
- [x] Implement `reg_parser.go` (C++ port).
- [x] Implement `regis3.go` public API.
- [x] Basic unit tests (`regis3_test.go`).
- [x] Review against C# implementation (gaps filled: `AskToRemoveKey` etc.).

### Phase 2: Export & Parity (Current Focus)

**Milestone 2.1: .REG File Writer (`writer.go`)**
- [x] Implement `RegWriter` struct.
- [x] Support `Format5` (Windows 5.00, UTF-16LE).
- [x] Support `Format4` (REGEDIT4, ANSI).
- [x] Implement value formatting:
    - [x] `REG_SZ` (quoted, escaped).
    - [x] `REG_DWORD` (dword:xxxxxxxx).
    - [x] `REG_BINARY`, `REG_NONE` (hex:...).
    - [x] `REG_MULTI_SZ`, `REG_EXPAND_SZ` (hex(t):...).
- [x] Handle "Delete" markers (`[-Key]`, `"Val"=-`).

**Milestone 2.2: Round-Trip Verification**
- [x] Create `TestRoundTrip`: Parse File -> Write File -> Parse File -> Compare.
- [x] Verify idempotency: `Write(Parse(X))` should match `X` (semantically).
- [x] Add `TestEscapedValues` for `$$VAR$$` support.

**Milestone 2.3: WiX XML Generation (`wix_writer.go`)**
- [ ] Implement `ToWixXML(io.Writer)`.
- [ ] Map Registry types to WiX `Type` attributes (`string`, `integer`, `binary`, `multiString`, `expandable`).
- [ ] Handle `$$VAR$$` pass-through (ensure it's not double-escaped).
- [ ] Generate `<RegistryKey>` hierarchy and `<RegistryValue>` elements.

**Milestone 2.4: Edge Case Hardening**
- [x] Port `EscapedString` tests from C#.
- [x] Port `Whitespace` handling tests.
- [x] Validate `WOW64` behavior (detection implemented in `wow.go`).

### Phase 3: Integration
1.  Golden tests comparing output against C# `regis3`.
2.  Integration with `msis` (Go rewrite).

## 4. Specific Go Design Choices

- **Error Handling:** Explicit error returns (`error`), no exceptions.
- **Strings:** All internal storage in UTF-8 (`string`). Conversion happens at I/O boundaries.
- **Dependencies:** Minimal. `golang.org/x/text` (std lib subrepo) used for encoding support if needed, otherwise `unicode/utf16`.