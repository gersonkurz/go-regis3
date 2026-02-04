# GEMINI.md

## Project Overview

`go-regis3` is a Go library for parsing, manipulating, and exporting Windows Registry (.REG) files. It provides a complete toolkit for handling registry snapshots, offering features beyond simple Windows API wrappers.

### Key Technologies
- **Language:** Go (1.25.5+)
- **Encodings:** `golang.org/x/text` for handling UTF-16LE (Windows 5.00) and ANSI (REGEDIT4) formats.
- **Architecture:** Character-at-a-time state machine parser with an in-memory tree representation (`KeyEntry` and `ValueEntry`).

### Core Capabilities
- **State Machine Parser:** Robustly handles complex .REG syntax (line continuations, hex encoding, escaped strings). It uses a self-describing state graph where transitions are explicit.
- **Fidelity-First Data Model:** String values are stored as **UTF-16LE bytes internally** to ensure byte-for-byte round-trip fidelity and natural handling of embedded NULLs.
- **In-Memory Representation:** Navigate and modify registry trees using a clean API. Registry is case-insensitive but case-preserving.
- **Diff/Merge API:** Compare registry states and generate transition .REG files using the `AskTo*` naming convention.
- **WiX XML Generation:** Export registry settings directly to WiX 4.x compatible XML for MSI installer authoring.
- **Variable Substitution:** Supports `$$VARIABLE$$` syntax for deferring value assignment until install time.

## Design Philosophy

- **Leniency & Permissiveness:** The parser is designed to mirror `regedit.exe`'s behavior, meaning it is intentionally lenient with malformed data (e.g., missing trailing newlines, extra whitespace, or unterminated strings at EOF).
- **Deterministic Output:** Writers sort keys and values alphabetically to ensure consistent, diff-friendly output.
- **Two-Pass WiX Generation:** The `WixWriter` performs a removal pass (flat) followed by an addition pass (nested) to ensure clean state transitions during installation.
- **Cross-Language Consistency:** The API design (including naming like `AskTo*` and mutable map exposure) is kept consistent with its C# and C++ sibling implementations.

## Building and Running

### Prerequisites
- Go 1.25.5 or higher.

### Key Commands
- **Install Dependencies:**
  ```bash
  go mod download
  ```
- **Build:**
  ```bash
  go build ./...
  ```
- **Run Tests:**
  ```bash
  go test ./...
  ```
- **Run Production Tests:**
  If you have a collection of real-world .REG files, you can test the parser against them:
  ```bash
  go test -testdata=/path/to/reg/files
  # OR
  REGIS3_TESTDATA=/path/to/reg/files go test
  ```

## Development Conventions

### Code Style
- **Idiomatic Go:** Follow standard Go formatting (`gofmt`).
- **Error Handling:** Uses a custom `ParseError` type for detailed reporting, including line/column numbers and context snippets.
- **Map Exposure:** Returns `map[string]*T` directly for `SubKeys()` and `Values()`, allowing direct mutation to match sibling C# semantics.
- **Naming:** Keeps "Get" prefixes for typed accessors (`GetString()`, `GetDword()`) for clarity.

### Testing Practices
- **Unit Tests:** Located in `regis3_test.go` and `parity_test.go`.
- **Round-trip Testing:** Ensures `Parse -> Modify -> Export -> Parse` preserves semantic fidelity.
- **Data-Driven Tests:** Verified against real-world suites via `TestProductionFiles`.

### Project Structure
- `regis3.go`: Public API entry points.
- `parser.go`: Abstract state machine infrastructure (`ParserContext`).
- `reg_parser.go`: .REG-specific implementation (20+ states).
- `types.go`: Registry type constants and option structs.
- `key.go` & `value.go`: `KeyEntry` (tree nodes) and `ValueEntry` (data).
- `writer.go` & `reg_writer.go`: .REG file exporter.
- `wix_writer.go`: WiX XML exporter.
- `wow.go`: WOW64 detection helpers.

## Usage Highlights

### Parsing a File
```go
root, err := regis3.ParseFile("settings.reg", nil)
```

### Modifying and Exporting
```go
key := root.FindOrCreateKey("Software\\MyCompany\\MyApp")
key.FindOrCreateValue("Version").SetString("1.2.3")

writer := regis3.NewRegWriter(regis3.HeaderWindows5, regis3.ExportOptions{}, true)
writer.Write(os.Stdout, root)
```

### WiX Export
```go
wixWriter := regis3.NewWixWriter()
wixWriter.Write(os.Stdout, root)
```
