# Gollum Justfile
# https://github.com/casey/just

default:
    @just --list

# Build and install gollum to GOBIN
build:
    @go install ./cmd/gollum
    @ls -la $(which gollum)

# Run tests
test:
    go test -v -race ./...

# Run tests with coverage
test-coverage:
    go test -v -race -coverprofile=/tmp/gollum/coverage.out ./...
    go tool cover -html=/tmp/gollum/coverage.out -o coverage.html

# Run linters
lint:
    golangci-lint run ./...

# Run linters with fix
lint-fix:
    golangci-lint run --fix ./...

# Format code
fmt:
    go fmt ./...
    gofumpt -w .

# Tidy dependencies
tidy:
    go mod tidy

# Clean build artifacts
clean:
    rm -f coverage.out coverage.html

# Run gollum
run:
    go run ./cmd/gollum

# Development: build and run
dev: build run

# Run all checks before commit
check: fmt lint test

# Update dependencies
update:
    @go get -u ./...
    @go mod tidy

# Verify dependencies
verify:
    @go mod verify
