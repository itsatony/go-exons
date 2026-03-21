# Contributing to go-exons

Thank you for considering contributing to go-exons. This document covers the development setup, code conventions, and PR process.

## Development Setup

```bash
# Clone
git clone https://github.com/itsatony/go-exons.git
cd go-exons

# Verify
go build ./...
go test -v -race ./...

# Lint (requires golangci-lint)
# https://golangci-lint.run/welcome/install/
golangci-lint run
```

## Code Conventions

### File Naming

All Go files follow the pattern: `exons.{type}.{module}.{variant}.go`

```
exons.engine.go           # Engine type
exons.spec.go             # Spec type
exons.spec.validate.go    # Spec validation
exons.lexer.go            # Lexer (in internal/)
exons.executor.builtins.go # Built-in resolvers (in internal/)
```

### Package Structure

- Root package (`package exons`) contains the public API
- `internal/` uses `package internal` — flat, single package
- Sub-packages (`execution/`, `a2a/`, etc.) each have their own package name

### Constants

**Every** string literal must be a named constant. No magic strings.

```go
// Correct
const ErrMsgMissingName = "name is required"
return cuserr.NewValidationError(FieldName, ErrMsgMissingName)

// Wrong
return fmt.Errorf("name is required")
```

### Error Handling

Use `go-cuserr` for all errors with constant message strings:

```go
const ErrMsgInvalidScope = "memory scope must match slug pattern"

if !isValidSlug(scope) {
    return cuserr.NewValidationError(FieldScope, ErrMsgInvalidScope)
}
```

### Testing

- Unit tests: >80% coverage, table-driven
- Always run with race detection: `go test -race`
- Use `stretchr/testify` for assertions

### Logging

Use `log/slog` (stdlib). No external logging dependencies.

## Pull Request Process

1. Fork the repository
2. Create a feature branch from `main`
3. Write tests for new functionality
4. Ensure `make check` passes (fmt, vet, lint, test)
5. Open a PR with a clear description of what and why

## Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). Be respectful.
