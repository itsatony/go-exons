# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.7.0-dc6] - 2026-03-20

### Added

#### A2A Agent Card Generation
- `Spec.CompileAgentCard(ctx, opts)` — generates Google A2A protocol Agent Cards from Spec configuration (DC-6)
- Pure metadata transformation — no template execution or network communication
- `A2ACardOptions` struct — URL, provider info, version, capabilities, security, resolver

#### `a2a/` Package Types
- `a2a.AgentCard` — full Agent Card (v0.3 spec) with name, URL, skills, capabilities, security, metadata
- `a2a.Provider` — organization identification
- `a2a.Capabilities` — streaming and push notification support
- `a2a.Skill` — skill advertisement with ID, name, description, tags, input/output modes
- `AgentCard.ToJSON()` and `AgentCard.ToJSONPretty()` — JSON serialization

#### Auto-Detection
- Skills mapped from `SkillRef` entries; descriptions resolved via `SpecResolver` (non-fatal fallback)
- Streaming capability detected from `execution.Config.Streaming.Enabled`
- Input modes inferred from `Spec.Inputs` types (string→"text/plain", object→"application/json")
- Output modes inferred from execution modality (text→"text/plain", image→"image/png", audio→"audio/mpeg")
- A2A-prefixed extensions (`a2a.*`) merged into card metadata

#### GenSpec Enrichment (unique to go-exons)
- `dispatch.TriggerKeywords` → appended to each A2A skill's Tags
- `registry.Version` → used as agent card Version (fallback after opts)
- `safety.Guardrails`, `safety.DenyTools`, `safety.RequireConfirmationFor` → card metadata
- `genspec.Version` → card metadata under `genspec.version`
- `dispatch.TriggerDescription` → card metadata under `dispatch.trigger_description`

#### Constants and Errors
- A2A metadata key constants: `A2AMetaKeySafetyGuardrails`, `A2AMetaKeySafetyDenyTools`, `A2AMetaKeySafetyConfirmation`, `A2AMetaKeyGenSpecVersion`, `A2AMetaKeyDispatchDescription`
- `NewA2AError()` constructor wrapping cuserr with `ErrCodeA2A`
- Helper functions: `modalityToMIME`, `inputTypeToMIME`, `sortedStringKeys`

#### Testing
- 40+ test functions in `exons.a2a_test.go` covering all paths
- Root coverage: 91.1%, total: 90.7%

## [0.6.0-dc5] - 2026-03-20

### Added
- Full agent compilation pipeline: `Spec.CompileAgent()` produces `CompiledSpec` with messages, execution, tools, constraints (DC-5)
- `Spec.Compile()` — simple body compilation through template engine
- `Spec.ActivateSkill()` — skill activation with injection into system_prompt, user_context, or none
- `Spec.AgentDryRun()` — 6-step validation collecting ALL issues without stopping at first error
- `CompiledMessage` type — compilation output (distinct from `Message` for template output extraction)
- `CompileOptions` with functional options: `WithResolver`, `WithCompileEngine`, `WithSkillsCatalogFormat`, `WithToolsCatalogFormat`
- Provider message serialization: `CompiledSpec.ToOpenAIMessages()`, `ToAnthropicMessages()`, `ToGeminiContents()`, `ToProviderMessages(provider)`
- `AgentExecutor` convenience wrapper: `Execute`, `ExecuteFile`, `ExecuteSpec`, `ActivateSkill`
- `AgentDryRunResult` with `OK()`, `HasErrors()`, `String()` methods
- Clone methods on types: `ToolsConfig.Clone()`, `ConstraintsConfig.Clone()`, `OperationalConstraints.Clone()`, `CredentialRef.Clone()`, `CredentialRef.Validate()`
- `Spec.ValidateCredentialRefs()` — validates credential map, default label, and skill credential labels
- Error constructors: `NewCompilationError`, `NewCompileMessageError`, `NewCompileSkillError`, `NewCompileBodyError`, `NewProviderMessageError`, `NewSkillNotFoundError`
- 25+ compile/DryRun/provider constants
- 130+ new test functions across 4 test files
- Root coverage: 89.5%, internal: 91.1%, execution: 92.1%

### Changed
- `Template.Compile()` and `Template.CompileAgent()` now delegate to `Spec` methods (stubs removed)
- `Spec.Clone()` now delegates to standalone `ToolsConfig.Clone()`, `ConstraintsConfig.Clone()`, `CredentialRef.Clone()`

### Removed
- `ErrMsgCompileNotAvailable` constant and `NewCompileNotAvailableError()` — replaced by real compilation
- `CompiledSpec` and `CompileOptions` placeholder definitions from `exons.template.go` — moved to `exons.compile.go`

## [0.5.0-dc4] - 2026-03-20

### Added
- Catalog generation API: `GenerateSkillsCatalog()` and `GenerateToolsCatalog()` with 4 formats (default, detailed, compact, function_calling) (DC-4)
- `NoopSpecResolver` — default SpecResolver that always returns not-found errors
- `MapSpecResolver` — thread-safe in-memory SpecResolver with `Add`, `AddMulti`, `Remove`, `Has`, `Count`
- `Engine.SetSpecResolver()` / `Engine.GetSpecResolver()` — configure spec resolver on Engine
- `Engine.ExecuteWithCatalogs()` — auto-generates skill/tool catalog strings and injects into context
- Auto-injection: Engine automatically injects SpecResolver into execution Context for `{~exons.ref~}` resolution
- `Import()` / `ImportDirectory()` — import from `.md` or `.zip` archives (SKILL.md/AGENT.md/PROMPT.md)
- `ExportDirectory()` — export Spec + resources to `.zip` archive
- `ImportFromSkillMD()` — parse SKILL.md format (frontmatter + body)
- `Spec.ExportToSkillMD()` — serialize with Agent Skills compatible fields only
- `Spec.StripExtensions()` — clone with execution/extensions/agent-fields removed
- `Spec.ValidateAsAgent()` — validate spec has agent type, execution config, and body/messages
- `ToolsConfig.HasTools()` — check if tools config has any functions or MCP servers
- `FunctionDef.ToOpenAITool()` — OpenAI-compatible tool definition map
- `ImportResult` struct for import results with Spec and Resources
- `MapSpecResolverEntry` struct for bulk resolver population
- 42 end-to-end integration tests verifying all 14 template tags through public Engine API
- Error constructors: `NewCatalogError`, `NewExportError`, `NewImportError`, `NewAgentValidationError`
- Root coverage: 88.8%, internal: 91.1%, execution: 92.1%

### Changed
- `Engine.Execute()` now auto-injects SpecResolver into context when configured
- `Engine.ExecuteTemplate()` now auto-injects SpecResolver for nested templates
- Version bumped to 0.5.0

## [0.4.0-dc3] - 2026-03-20

### Added
- Full `execution.Config` with 32 fields covering all major LLM providers (DC-3)
- Provider serialization: `ToOpenAI()`, `ToAnthropic()`, `ToGemini()`, `ToVLLM()`, `ToMistral()`, `ToCohere()`
- `Config.Validate()` — validates all field ranges and delegates to sub-type validators
- `Config.Clone()` — deep copy of all pointer/slice/map/nested config fields
- `Config.Merge()` — 3-layer precedence merge (other wins for non-nil fields)
- `GetEffectiveProvider()` — auto-detect provider from model name and config shape
- Get/Has pairs for all ~30 optional config fields
- `ProviderFormat(provider)` — dispatch to provider-specific response format
- `Config.JSON()` / `Config.YAML()` — convenience serialization methods
- Sub-types: `ThinkingConfig`, `ResponseFormat`, `JSONSchemaSpec`, `EnumConstraint`, `GuidedDecoding`
- Media types: `ImageConfig` (11 fields), `AudioConfig` (6), `EmbeddingConfig` (7), `StreamingConfig`, `AsyncConfig`
- Schema helpers: `GeminiTaskType()`, `CohereUpperCase()`, `ensureAdditionalPropertiesFalse()`
- Model detection: `isOpenAIModel`, `isAnthropicModel`, `isGeminiModel`, `isMistralModel`, `isCohereModel`
- `Spec.Serialize(opts)` — YAML frontmatter + body export with configurable field inclusion
- `SerializeOptions` with `IncludeExecution`, `IncludeExtensions`, `IncludeAgentFields`, `IncludeContext`, `IncludeCredentials`, `IncludeGenSpec`
- Factory functions: `DefaultSerializeOptions()`, `AgentSkillsExportOptions()`, `FullExportWithCredentials()`
- `Parse(data)` / `ParseFile(path)` / `MustParse(data)` — standalone `.exons` file parsing
- Extension helpers: `GetExtension`, `SetExtension`, `RemoveExtension`, `GetExtensionAs[T]`
- `GetStandardFields()` / `GetExonsFields()` — field classification helpers
- `CompiledSpec` fields typed: `Execution *execution.Config`, `Tools *ToolsConfig`, `Constraints *ConstraintsConfig`
- `Spec.Clone()` delegates to `Config.Clone()` (replaces 25 lines of manual copying)
- `Spec.Validate()` delegates to `Config.Validate()` for execution config validation
- execution/ package: 92.1% coverage, root package: 88.3% coverage

### Changed
- **BREAKING**: `execution.Config.Stop` renamed to `StopSequences` (yaml: `stop_sequences`)
- Version bumped to 0.4.0

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
