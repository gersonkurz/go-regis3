/*
Package regis3 provides a high-fidelity toolkit for parsing, manipulating,
and exporting Windows Registry (.REG) files.

Unlike standard registry libraries that wrap the Windows Registry API (RegOpenKeyEx, etc.),
go-regis3 treats .REG files as structured text documents. It uses a character-at-a-time
state machine parser to build an in-memory tree representation (KeyEntry) that can be
manipulated, compared (diff/merge), and exported back to .REG or WiX XML formats.

Core Features:
  - Supports both REGEDIT4 (ANSI) and Windows Registry Editor 5.00 (UTF-16LE) formats.
  - High-fidelity internal storage using UTF-16LE to preserve exact byte sequences.
  - Case-insensitive but case-preserving key and value handling.
  - Advanced features like variable substitution ($$VAR$$ syntax) and WiX XML generation.
  - Robust error reporting with line/column numbers and context snippets.

The library is designed for cross-platform use, allowing registry files to be
processed on non-Windows systems (e.g., for installer generation or snapshot analysis).
*/
package regis3
