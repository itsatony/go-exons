# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0-dc2] - 2026-03-20

### Added
- Root package public API wrapping internal engine (DC-2)
- `Engine` — main entry point: `New()`, `MustNew()`, `Parse()`, `Execute()`
- `Template` — parsed template: `Execute()`, `ExecuteWithContext()`, `ExecuteAndExtractMessages()`
- `Context` — execution context with dot-notation paths, typed getters, parent-child scoping
- `Resolver` / `SpecResolver` / `Attributes` interfaces for custom tag handlers
- `ResolverFunc` convenience type for function-based resolvers
- `Func` type for custom expression functions
- `Spec` — YAML frontmatter parsing (`ParseYAMLSpec`), validation, `Clone()`
- `CompiledSpec` placeholder type (compile stubs return error until DC-5)
- `Message` type and `ExtractMessagesFromOutput()` for LLM API integration
- `ValidationResult` / `Engine.Validate()` — AST-walking template validator
- `HookRegistry` — simplified hook system (10 hook points, no access-control deps)
- `LoggingHook` and `TimingHook` factory functions
- `DryRunResult` / `Template.DryRun()` — static analysis without execution
- `ExplainResult` / `Template.Explain()` — human-readable execution walkthrough
- `TokenEstimate` / `EstimateTokens()` — token count estimation with cost budgeting
- `ErrorStrategy` type with `ParseErrorStrategy()` and `IsValidErrorStrategy()`
- `ValidationSeverity` type
- Error constructors via `go-cuserr` for all error categories
- `Position` type for source location tracking
- Functional options: `WithDelimiters()`, `WithErrorStrategy()`, `WithMaxDepth()`, `WithLogger()`
- `TemplateRunner` interface shared by Engine (and future StorageEngine)
- 579 root package tests, 86.7% coverage, all passing with `-race`

### Changed
- Version bumped to 0.3.0

## [0.2.0-dc1] - 2026-03-20

### Added
- Initial project structure
- Core template engine (lexer, parser, executor) from go-prompty lineage
- `.exons` file format with YAML frontmatter and `{~...~}` template syntax
- Spec type with GenSpec support (memory, dispatch, verification, registry, safety)
- Execution configuration with multi-provider serialization
- Provider packages: OpenAI, Anthropic, Gemini, vLLM, Mistral, Cohere
- A2A Agent Card generation
- Storage interfaces with in-memory implementation
- VS Code syntax highlighting for `.exons` files
- CLI tool (`exons`)
- 476 internal tests, 91.1% coverage
