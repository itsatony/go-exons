.PHONY: test lint cover build clean check fmt fmt-check vet ci-check ci-local tidy-check

# Minimum statement coverage for the module, enforced by ci-local.
#
# Set from the MEASURED baseline at v0.23.0 (90.9%) with headroom, not from an aspiration: a floor
# nobody can meet gets lowered until it means nothing, and a floor pinned to exactly today's number
# turns an honest refactor into a red build.
COVERAGE_THRESHOLD := 88

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

# Format code (MUTATES the working tree — never use this as a gate)
fmt:
	gofmt -s -w .

# Assert the tree is already formatted.
#
# This exists because `check` used to run `fmt`, which rewrites files and then reports success.
# A gate that passes by editing your code cannot fail on unformatted code, which is the one thing
# it was there to catch.
fmt-check:
	@unformatted=$$(gofmt -s -l . | grep -v '^examples/' || true); \
	if [ -n "$$unformatted" ]; then \
		echo "ERROR: not gofmt-simplified:"; echo "$$unformatted"; exit 1; \
	fi; \
	echo "OK: gofmt clean"

# Assert go.mod/go.sum are tidy without leaving them tidied.
tidy-check:
	@cp go.mod go.mod.cibak; cp go.sum go.sum.cibak; \
	go mod tidy; \
	status=0; \
	if ! diff -q go.mod go.mod.cibak >/dev/null || ! diff -q go.sum go.sum.cibak >/dev/null; then \
		echo "ERROR: go.mod/go.sum are not tidy — run 'go mod tidy' and commit the result"; status=1; \
	fi; \
	mv go.mod.cibak go.mod; mv go.sum.cibak go.sum; \
	if [ $$status -eq 0 ]; then echo "OK: go.mod tidy"; fi; \
	exit $$status

# Vet code
vet:
	go vet ./...

# Run all checks (used by CI)
check: fmt vet lint test

# The local gate. Kept as close as possible to .github/workflows/ci.yml, and non-mutating
# throughout so that a green run here means the same thing a green run in CI does.
ci-local: build vet fmt-check tidy-check lint
	@echo "==> tests (race) + coverage floor $(COVERAGE_THRESHOLD)%"
	@go test -race -coverprofile=coverage.out ./...
	@total=$$(go tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/,"",$$3); print $$3}'); \
	echo "==> total coverage: $$total% (floor $(COVERAGE_THRESHOLD)%)"; \
	awk -v t="$$total" -v f="$(COVERAGE_THRESHOLD)" \
		'BEGIN { if (t+0 < f+0) { printf "ERROR: coverage %.1f%% is below the %s%% floor\n", t, f; exit 1 } }'
	@echo "==> ci-local PASSED"

# Validate CI workflows reference only existing paths
ci-check:
	@echo "Checking CI workflow path references..."
	@grep -rn '\./cmd/\|\.\/provider/\|\.\/storage/' .github/workflows/ 2>/dev/null && { echo "ERROR: CI workflows reference deleted paths"; exit 1; } || echo "OK: No stale path references in CI workflows"

# Clean build artifacts
clean:
	rm -rf bin/ dist/ tmp/ coverage.out coverage.html
