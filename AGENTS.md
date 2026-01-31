# Repository Guidelines

## Project Structure & Module Organization
This is a single-package Go library (`regis3`) for parsing and exporting Windows `.REG` files.
- `*.go` at repo root: core package files (parser, tree model, writers).
- `*_test.go`: unit, parity, and round-trip tests (e.g., `parity_test.go`, `regis3_test.go`).
- `docs/`: architecture and overview references (`docs/overview.md`, `docs/architecture.mmd`/`.png`).

## Build, Test, and Development Commands
```bash
go test                    # run all tests
go test -v                 # verbose output
go test -testdata=/path    # tests against real .REG files
REGIS3_TESTDATA=/path go test
go fmt ./...               # format package files
gofmt -l .                 # check formatting
```
This repo has no external dependencies (standard library only).

## Coding Style & Naming Conventions
- Go standard style enforced by `gofmt` (tabs for indentation).
- Exported identifiers use `PascalCase`; unexported use `camelCase`.
- Keep files small and focused (e.g., `key.go`, `value.go`, `parser.go`).
- Avoid new dependencies unless essential to parsing/writing behavior.

## Testing Guidelines
- Tests use Go’s `testing` package (no external framework).
- Favor round-trip tests: parse → modify → export → parse.
- Use parity tests to validate compatibility with the C# implementation.
- Name tests `TestXxx` and keep fixtures under the optional `-testdata` path.

## Commit & Pull Request Guidelines
Recent history shows short, imperative summaries and occasional Conventional Commit style:
`feat(key): Add SetParent method for tree grafting`.
- Keep commit subjects concise and action-focused.
- If using scopes, prefer the affected area (`feat(parser):`, `fix(writer):`).

No formal PR template is present. Please include:
- A clear summary of changes and rationale.
- Test commands run and results.
- Sample `.reg` inputs/outputs if behavior changes.

## Architecture Notes
The core flow is parse → in-memory tree → export. See `CLAUDE.md` for file-level responsibilities and design notes before making structural changes.
