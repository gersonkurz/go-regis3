# go-regis3 build automation

# Default recipe: show available commands
default:
    @just --list

# Build the library to check for compilation errors
build:
    go build ./...

# Run standard unit tests
test:
    go test ./...

# Run tests with verbose output
test-v:
    go test -v ./...

# Run production tests with external .reg files
# Usage: just test-prod /path/to/reg/files
test-prod folder:
    go test -v -testdata={{folder}}

# Run tests with race detector
test-race:
    go test -race ./...

# Format code
fmt:
    gofmt -w .

# Run go vet (static analysis)
vet:
    go vet ./...

# Check formatting without modifying files
fmt-check:
    @gofmt -l . | grep -q . && echo "Code not formatted. Run 'just fmt'" && exit 1 || echo "Code is formatted"

# Run all quality checks
check: fmt-check vet test

# Clean build and test cache
clean:
    go clean -testcache
    go clean ./...

# Update dependencies
tidy:
    go mod tidy
