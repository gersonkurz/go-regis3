# Findings

## High

- **EOF handling silently drops/accepts incomplete input.** The parser never finalizes the last token at EOF, so a file missing a trailing newline can skip parsing the final DWORD/hex value without error; header-only files without newline can pass without validation. This is due to `ParseText` not invoking a terminal state and `stateExpectHeader` only validating on CR/LF. `parser.go:32`, `reg_parser.go:134`, `reg_parser.go:289`, `reg_parser.go:398`.
- **REGEDIT4 encoding is treated as UTF-8.** `ParseFile` treats non-UTF16 content as UTF-8/ASCII, but REGEDIT4 is ANSI in the system codepage. Non-ASCII content will decode incorrectly, and `RegWriter` with `useUtf16=false` will emit UTF-8 bytes rather than ANSI. `regis3.go:71`, `writer.go:202`.

## Medium

- **UTF-8 BOM is not stripped.** If the caller uses `Parse` on a UTF-8 BOM file (or `ParseFile` on a UTF-8 BOM without UTF-16LE), header detection can fail because the BOM stays in the string. `regis3.go:23`, `regis3.go:71`.
- **Odd-length hex payloads are silently padded.** `createByteArrayFromBuffer` pads odd digit counts with `"0"` and ignores decode errors, masking malformed input. This should likely hard-fail or at least surface a parse error. `reg_parser.go:120`.
- **IgnoreWhitespaces doesn?t allow leading line whitespace.** Even with `IgnoreWhitespaces`, lines that start with spaces/tabs (before `[` or `"`) are rejected. `reg_parser.go:182`.
- **Comments aren?t allowed after multibyte values.** `AllowHashtagComments`/`AllowSemicolonComments` work for DWORDs but not for `hex(...)` values, which can be surprising. `reg_parser.go:398`.

## Low

- **`GetMultiString` drops empty elements.** If a REG_MULTI_SZ includes empty strings between separators, they are lost (`i > start` guard). This can break round-trip fidelity for edge cases. `value.go:172`.
- **`IsParseError` doesn?t unwrap.** It only type-asserts, so wrapped parse errors won?t be detected. `errors.go:49`.
- **Internal data exposure isn?t fully documented.** `KeyEntry.SubKeys/Values` explicitly return mutable maps, but `ValueEntry.Data` also returns the underlying slice without a warning. `value.go:40`, `key.go:45`.
- **WiX QWORD semantics are unclear.** `RegQword` falls into the default ?binary? path; escaped QWORD uses `Type="integer"` with a variable string. Confirm WiX expects these mappings for 64-bit values. `wix_writer.go:232`.

## Praise

- **Architecture clarity.** The single-package layout and state-machine parser are exceptionally easy to reason about; docs are first-class.
- **API consistency.** The C#/C++/Go parity in naming and behaviors is thoughtful and well preserved.
- **Tests are focused and pragmatic.** Round-trip, deletion, whitespace, escaping, and WiX output are covered with minimal noise.
