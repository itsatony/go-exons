# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.11.0-dc10] - 2026-03-21

### BREAKING
- **Default env var denylist active**: `{~exons.env~}` now blocks environment variables matching common secret patterns (`*_KEY`, `*_SECRET`, `*_TOKEN`, `*_PASSWORD`, `*_PASS`, `*_CREDENTIAL`, `*_PASSPHRASE`, `*_DSN`, `*_CONN_STRING`) by default. Override with `WithEnvDenylist(nil)` or `WithEnvAllowlist(...)`.
- **`DefaultEnvDenyPatterns` is now a function** returning a fresh copy, not a mutable `var` slice. Callers using the old var form need to add `()`.
- **Zip document size limit**: `ImportDirectory` now limits document files (SKILL.md, AGENT.md, PROMPT.md) to 10MB (`MaxImportDocumentSize`). Previously unlimited.
- **Zip path traversal rejection**: `ImportDirectory` rejects resource entries with path traversal (`../`) or absolute paths. `ExportDirectory` returns an error for such paths.

### Added
- `FunctionDef.ToAnthropicTool()` — Anthropic `input_schema` format
- `FunctionDef.ToGeminiTool()` — Gemini `parameters` format (flat, no wrapper)
- `FunctionDef.ToMCPTool()` — MCP `inputSchema` format (camelCase)
- `FunctionDef.ToMistralTool()` — Mistral (OpenAI-compatible) format
- `FunctionDef.ToCohereTool()` — Cohere `parameter_definitions` format
- `ToolsConfig.ToOpenAITools()`, `ToAnthropicTools()`, `ToGeminiTools()`, `ToMCPTools()`, `ToMistralTools()`, `ToCohereTools()` — batch export methods
- `WithEnvAllowlist(patterns)` — restrict `{~exons.env~}` to matching glob patterns only
- `WithEnvDenylist(patterns)` — block `{~exons.env~}` for matching glob patterns
- `WithEnvDisabled()` — completely disable `{~exons.env~}` tag
- `WithMaxOutputSize(size)` — cap rendered template output (default 10MB)
- `DefaultEnvDenyPatterns()` — returns default deny patterns (immutable function)
- `MaxImportDocumentSize` constant (10MB) for zip document file limits
- JSON Schema for `.exons` format at `schema/exons.schema.json` (draft 2020-12, 28 `$defs`)
- 7 standalone Go examples in `examples/01-*` through `examples/07-*`
- Blog post: `docs/blog/introducing-go-exons.md`

### Security
- Env var access denied by default for common secret patterns (defense in depth)
- Zip bomb prevention: document files limited to 10MB
- Zip path traversal: reject `../` and absolute paths in import; error on export
- Output size enforcement: `executeNodes` enforces `MaxOutputSize` (10MB default)
- Invalid glob patterns in env config fail loudly (no silent bypass)

### Fixed
- README: `EstimateTokens` example now matches actual signature `EstimateTokens(text string) *TokenEstimate`
- README: `RegisterFunc` example now shows correct `*Func` struct form

## [0.10.0-dc9] - 2026-03-20

### BREAKING
- Removed `cmd/exons/` CLI stub (42 LOC, hardcoded stale version, 0% coverage)
- Removed `provider/` and `storage/` empty package stubs
- Removed 56 Get*/Has* methods from `execution.Config` (fields are exported — use direct access)
- Unexported: `GenerateSkillsCatalog`, `GenerateToolsCatalog`, `GetStandardFields`, `GetExonsFields`, `GetExtensionAs`, `StripExtensions`

### Fixed
- `MetaKeyMaxDepth` misuse in spec length validation errors — now uses `MetaKeyMaxLength`
- `NewCredentialValidationError` hardcoded wrong message — now uses generic `ErrMsgCredentialValidationFailed`
- `RegisterTemplate` held write lock during Parse — parse now happens outside lock
- `Context.With*` methods used write lock for read-only ops — now uses RLock
- `knownSpecFields` silently swallowed extension data for non-existent Spec fields (license, compatibility, allowed_tools, metadata, requirements)
- Discarded error in `newTemplateWithConfig` now handled explicitly (nil on error, preserving fail-safe behavior)

### Removed
- 9 dead error constants (`ErrMsgSkillNotFound`, `ErrMsgSkillRefAmbiguous`, `ErrMsgSkillRefInvalidVersion`, `ErrMsgInvalidSkillInjection`, `ErrMsgMessageTemplateNoBody`, `ErrMsgNoDocumentResolver`, `ErrMsgInvalidSkopeSlug`, `ErrMsgInvalidVisibility`, `ErrMsgVersionNumberNegative`)
- 1 dead error constant in root package (`ErrMsgEngineNotAvailable` — internal/ has its own)
- 2 dead error constants (`ErrMsgExtensionNotFound`, `ErrMsgExtensionCastFailed`) — only used by deleted functions
- 2 dead error constructors (`NewEngineNotAvailableError`, `NewAgentValidationError`)
- 1 dead error code (`ErrCodeAgent`)
- 5 dead SpecField constants (`SpecFieldLicense`, `SpecFieldCompatibility`, `SpecFieldAllowedTools`, `SpecFieldMetadata`, `SpecFieldRequirements`)
- 4 dead functions after API surface reduction: `getExtensionAs[T]`, `getStandardFields`, `getExonsFields`, `stripExtensions` (test-only, zero production callers)

## [0.9.0-dc8] - 2026-03-20

### Removed

#### BREAKING: Compilation Layer Removed
- **BREAKING**: Removed `CompileAgent()` — agent compilation is the runtime's responsibility
- **BREAKING**: Removed `CompiledSpec` type and `CompiledMessage` type
- **BREAKING**: Removed `Compile()` on Spec and Template
- **BREAKING**: Removed `ActivateSkill()` — skill injection belongs in the runtime
- **BREAKING**: Removed `AgentDryRun()` and `AgentDryRunResult` — validation without compilation is the consumer's job
- **BREAKING**: Removed `AgentExecutor` wrapper (`Execute`, `ExecuteFile`, `ExecuteSpec`)
- **BREAKING**: Removed `CompileOptions` type and functional option constructors (`WithResolver`, `WithCompileEngine`, etc.)
- **BREAKING**: Removed `ToOpenAIMessages()`, `ToAnthropicMessages()`, `ToGeminiContents()`, `ToProviderMessages()` — provider message formatting belongs in the runtime
- **BREAKING**: Removed `Template.Compile()` and `Template.CompileAgent()` delegation methods
- **BREAKING**: Removed `ValidateAsAgent()` method on Spec
- **BREAKING**: Removed error constructors: `NewCompilationError`, `NewCompileMessageError`, `NewCompileSkillError`, `NewCompileBodyError`, `NewProviderMessageError`, `NewSkillNotFoundError`
- **BREAKING**: Removed `ErrCodeCompile` error code
- **BREAKING**: Removed ~30 compilation-only constants (error messages, dry run categories, provider message keys, injection markers)

#### Files Deleted
- `exons.compile.go` (467 lines)
- `exons.compile.messages.go` (123 lines)
- `exons.compile.dryrun.go` (156 lines)
- `exons.agent.executor.go` (111 lines)
- `exons.compile_test.go` (1,372 lines)
- `exons.compile.integration_test.go` (1,082 lines)
- `exons.compile.messages_test.go` (371 lines)
- `exons.compile.dryrun_test.go` (473 lines)
- `exons.spec.agent_test.go`

### Changed
- README.md rewritten — focuses on parse/validate/serialize/execute flow, no compilation references
- CLAUDE.md updated — removed compilation section and files from package structure
- `ValidateCredentialRefs()` simplified — no longer wraps errors with compilation-specific constructors
- Internal catalog resolver comments updated to remove CompileAgent references
- Internal hint updated: `CompileOptions.Resolver` reference replaced with `Engine.SetSpecResolver()`

### Retained
- Template engine (`Engine.Parse`, `Engine.Execute`, `ExecuteAndExtractMessages`)
- Spec parsing, validation, serialization, clone
- `execution.Config` with all 32+ fields and 6 provider serializers
- A2A Agent Card generation (`CompileAgentCard`)
- Catalog generation (`GenerateSkillsCatalog`, `GenerateToolsCatalog`)
- Import/Export (`.md`, `.zip`, `.prompty`, `.genspec`)
- All metadata types (memory, dispatch, verifications, registry, safety)
- SkillRef, ToolsConfig, ConstraintsConfig types (YAML spec format)
- SkillInjection type and constants (part of SkillRef YAML field)
- Role constants, context key constants (used by template engine)
- Root coverage: 90.6%, internal: 91.1%, execution: 92.1%

## [0.8.0-dc7] - 2026-03-20

### Changed

#### Part A: Flatten GenSpec into Spec (BREAKING)
- **BREAKING**: Removed `genspec/` sub-package entirely — all types moved to root `exons` package
- **BREAKING**: Replaced `GenSpec *genspec.GenSpec` field on `Spec` with 5 flat fields:
  - `Memory *MemorySpec` (yaml: `memory:`)
  - `Dispatch *DispatchSpec` (yaml: `dispatch:`)
  - `Verifications []VerificationCase` (yaml: `verifications:`)
  - `Registry *RegistrySpec` (yaml: `registry:`)
  - `Safety *SafetyConfig` (yaml: `safety:`)
- **BREAKING**: YAML format changed from nested `genspec:` wrapper to flat top-level fields
- **BREAKING**: Removed `GenSpecVersion` constant, `SpecFieldGenSpec` constant, `A2AMetaKeyGenSpecVersion` constant
- **BREAKING**: `SerializeOptions.IncludeGenSpec` renamed to `IncludeMetadata`
- **BREAKING**: `ErrCodeGenSpec` renamed to `ErrCodeMetadata`
- Added `HasMetadata()` method on Spec — replaces `IsGenSpec()` conceptually
- `IsGenSpec()` kept as deprecated alias for `HasMetadata()`
- Added proper `Clone()` methods on all metadata types (fixes shallow-copy defect)
- Origin constants (`OriginInternal`, `OriginExternal`, `OriginUnknown`) and guardrails constants (`GuardrailsEnabled`, `GuardrailsDisabled`) now in root package
- A2A metadata: `genspec.version` key removed from agent card metadata; safety/dispatch enrichment unchanged
- New spec field constants: `SpecFieldMemory`, `SpecFieldDispatch`, `SpecFieldVerifications`, `SpecFieldRegistry`, `SpecFieldSafety`

#### Part B: Prompty Auto-Import
- `ImportPrompty(data)` converts `.prompty` files to valid `Spec` instances
- Tag namespace conversion: `{~prompty.` → `{~exons.` (opening and closing tags)
- YAML field remapping: `delegation` → `dispatch`, `tests` → `verifications`, `plugin` → `registry` (with `trust_level` → `origin`)
- `genspec:` wrapper auto-flattened to top-level fields
- Extra prompty fields (`license`, `compatibility`, `allowed_tools`, `metadata`, `requirements`) moved to `extensions`
- `.prompty` and `.genspec` file extensions recognized by `Import()`
- `isPromptyContent()` auto-detection helper for content with `{~prompty.` tags

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
