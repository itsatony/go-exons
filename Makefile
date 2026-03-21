.PHONY: test lint cover build clean check fmt vet ci-check

# Run all tests with race detection
test:
	go test -v -race ./...

# Run linter (matches CI exactly)
lint:
	golangci-lint run ./...

# Run tests with coverage report
cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Build all packages
build:
	go build ./...

# Format code
fmt:
	gofmt -s -w .

# Vet code
vet:
	go vet ./...

# Run all checks (used by CI)
check: fmt vet lint test

# Validate CI workflows reference only existing paths
ci-check:
	@echo "Checking CI workflow path references..."
	@grep -rn '\./cmd/\|\.\/provider/\|\.\/storage/' .github/workflows/ 2>/dev/null && { echo "ERROR: CI workflows reference deleted paths"; exit 1; } || echo "OK: No stale path references in CI workflows"

# Clean build artifacts
clean:
	rm -rf bin/ dist/ tmp/ coverage.out coverage.html
