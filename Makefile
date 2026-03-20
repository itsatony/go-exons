.PHONY: test lint cover build clean check fmt vet

# Run all tests with race detection
test:
	go test -v -race ./...

# Run linter
lint:
	golangci-lint run

# Run tests with coverage report
cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Build all packages
build:
	go build ./...

# Build CLI binary
build-cli:
	go build -o bin/exons ./cmd/exons/

# Format code
fmt:
	gofmt -s -w .

# Vet code
vet:
	go vet ./...

# Run all checks (used by CI)
check: fmt vet lint test

# Clean build artifacts
clean:
	rm -rf bin/ dist/ tmp/ coverage.out coverage.html
