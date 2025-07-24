# Build variables
VERSION := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
COMMIT := `git rev-parse --short HEAD 2>/dev/null || echo "unknown"`
DATE := `date -u +"%Y-%m-%dT%H:%M:%SZ"`
LDFLAGS := "-X main.version=" + VERSION + " -X main.commit=" + COMMIT + " -X main.date=" + DATE

# Default recipe
default: tidy build

# Build the binary
build:
    mkdir -p build
    go build -ldflags "{{LDFLAGS}}" -o build/bump-tui .

# Run the application
run:
    go run -ldflags "{{LDFLAGS}}" .

# Clean build artifacts
clean:
    rm -rf build/
    go clean

# Run tests
test:
    go test -v ./...

# Run tests with coverage
test-coverage:
    go test -v -race -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

# Run linting
lint:
    golangci-lint run --timeout=5m

# Run go vet
vet:
    go vet ./...

# Run full CI-equivalent checks
ci-test:
    just tidy
    just vet  
    just test-coverage
    just lint

# Tidy go modules
tidy:
    go mod tidy

# Install the binary to GOPATH/bin
install: build
    cp build/bump-tui $(GOPATH)/bin/

# Show help
help:
    @echo "Available recipes:"
    @echo "  build         - Build the binary"
    @echo "  run           - Run the application"  
    @echo "  clean         - Clean build artifacts"
    @echo "  test          - Run tests"
    @echo "  test-coverage - Run tests with coverage report"
    @echo "  lint          - Run golangci-lint"
    @echo "  vet           - Run go vet"
    @echo "  ci-test       - Run full CI-equivalent checks"
    @echo "  tidy          - Tidy go modules"
    @echo "  install       - Install binary to GOPATH/bin"
    @echo "  help          - Show this help"

# Development recipes
dev: tidy
    DEBUG=1 go run -ldflags "{{LDFLAGS}}" .

# Build for multiple platforms
build-all:
    mkdir -p build
    GOOS=linux GOARCH=amd64 go build -ldflags "{{LDFLAGS}}" -o build/bump-tui-linux-amd64 .
    GOOS=darwin GOARCH=amd64 go build -ldflags "{{LDFLAGS}}" -o build/bump-tui-darwin-amd64 .
    GOOS=darwin GOARCH=arm64 go build -ldflags "{{LDFLAGS}}" -o build/bump-tui-darwin-arm64 .
    GOOS=windows GOARCH=amd64 go build -ldflags "{{LDFLAGS}}" -o build/bump-tui-windows-amd64.exe .