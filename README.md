# go-exons

Declarative agent specification format for Go.

An `.exons` file describes a complete agent: identity, execution parameters, tools, memory, dispatch rules, safety constraints, and verification cases — using YAML frontmatter and a content-resistant `{~...~}` template syntax.

go-exons parses, validates, and serializes these specs. It does **not** execute against LLMs — that's the runtime's job.

```
go get github.com/itsatony/go-exons
```

## Quick Start

**1. Define an agent** (`hello.exons`):

```yaml
---
name: greeter
description: A friendly greeter agent
type: agent
execution:
  provider: openai
  model: gpt-4o
  temperature: 0.7
---
{~exons.message role="system"~}
You are a friendly greeter.
{~/exons.message~}

{~exons.message role="user"~}
Say hello to {~exons.var name="user_name" default="World" /~}
{~/exons.message~}
```

**2. Parse and use**:

```go
engine := exons.MustNew()
tmpl, _ := engine.Parse(source)

// Execute template and extract structured messages
messages, _ := tmpl.ExecuteAndExtractMessages(ctx, map[string]any{
    "user_name": "Alice",
})
// messages[0] → {Role: "system", Content: "You are a friendly greeter."}
// messages[1] → {Role: "user",   Content: "Say hello to Alice"}

// Access the parsed spec
spec := tmpl.Spec()
fmt.Println(spec.Name)            // "greeter"
fmt.Println(spec.Execution.Model) // "gpt-4o"
```

## The `.exons` Format

An `.exons` file has two parts: **YAML frontmatter** (configuration) and a **template body** (the prompt).

### Document Types

| Type | Description |
|---|---|
| `prompt` | Simple template — variables, conditionals, loops. No skills/tools/constraints. |
| `skill` | Reusable capability with inputs/outputs. May have memory, registry, verification. |
| `agent` | Full agent with tools, skills, constraints, metadata. All fields available. |

### Annotated Example

```yaml
---
name: dns-specialist
description: Deep DNS expert for Cloudflare zone management
type: agent

execution:
  provider: anthropic
  model: claude-sonnet-4-6
  temperature: 0.2
  max_tokens: 4096

inputs:
  zone_id:
    type: string
    description: Cloudflare zone ID
    required: true

tools:
  allow: [dns_list_records, dns_create_record, dns_update_record, dns_delete_record]
  functions:
    - name: check_propagation
      description: Check DNS propagation status worldwide
      parameters:
        type: object
        properties:
          domain: { type: string }
          record_type: { type: string, enum: [A, AAAA, CNAME, MX, TXT] }
        required: [domain]

memory:
  scope: dns-manager
  auto_recall: true
  auto_record: true

dispatch:
  trigger_keywords: [dns, domain, nameserver, propagation]
  trigger_description: Route DNS tasks to this agent
  cost_limit_usd: 0.50

verifications:
  - name: can-list-records
    prompt: "List all DNS records for the test zone"
    input: { zone_id: "test-zone-id" }
    expect:
      tool_calls: [dns_list_records]
      output_contains: "records"
    timeout_seconds: 30

registry:
  namespace: dns-manager
  origin: internal
  version: 1.2.0

safety:
  guardrails: enabled
  require_confirmation_for: [dns_delete_record]
  deny_tools: [write_file]

constraints:
  behavioral:
    - Always verify current state before making changes
  safety:
    - Never delete SOA or NS records
  operational:
    max_turns: 15
    timeout_seconds: 120
---
{~exons.message role="system"~}
You are a DNS specialist agent. When given a DNS task:
1. Read the current state before making changes.
2. Explain what you plan to change and why.
3. After changes, verify propagation.
{~/exons.message~}

{~exons.message role="user"~}
{~exons.var name="input.query" default="What DNS records exist?" /~}
{~/exons.message~}
```

## Template Syntax Reference

The `{~...~}` delimiter was chosen to never collide with prompt content (JSON, XML, Go templates, etc.).

| Tag | Example |
|---|---|
| Variable | `{~exons.var name="user.name" default="Guest" /~}` |
| Conditional | `{~exons.if eval="user.isAdmin"~}...{~exons.else~}...{~/exons.if~}` |
| Loop | `{~exons.for item="x" index="i" in="items"~}...{~/exons.for~}` |
| Include | `{~exons.include template="header" /~}` |
| Message | `{~exons.message role="system"~}...{~/exons.message~}` |
| Ref | `{~exons.ref slug="my-skill" /~}` |
| Switch | `{~exons.switch eval="x"~}{~exons.case value="a"~}...{~/exons.case~}{~/exons.switch~}` |
| Skills Catalog | `{~exons.skills_catalog /~}` |
| Tools Catalog | `{~exons.tools_catalog /~}` |
| Env | `{~exons.env name="API_KEY" default="none" /~}` |
| Extends | `{~exons.extends template="parent"~}` |
| Block | `{~exons.block name="content"~}...{~/exons.block~}` |
| Raw | `{~exons.raw~}not parsed{~/exons.raw~}` |
| Comment | `{~exons.comment~}removed from output{~/exons.comment~}` |
| Escape | `\{~` produces literal `{~` |

## Metadata Fields

Metadata describes agent behavior beyond prompts. These fields live at the YAML top level:

| Field | Allowed On | Purpose |
|---|---|---|
| `memory` | skill, agent | Scope, auto-recall, auto-record, read scopes |
| `dispatch` | agent | Trigger keywords, description, cost limits |
| `verifications` | all | Test cases with expected tool calls and outputs |
| `registry` | skill, agent | Namespace, origin (internal/external/unknown), version |
| `safety` | all | Guardrails, deny-tools, require-confirmation lists |

Go types: `MemorySpec`, `DispatchSpec`, `VerificationCase`, `RegistrySpec`, `SafetyConfig` — all with `Clone()` and `Validate()`.

## Execution Config

Provider-agnostic LLM parameters (32+ fields). Defined in the `execution/` package.

```yaml
execution:
  provider: openai
  model: gpt-4o
  temperature: 0.7
  max_tokens: 4096
  response_format:
    type: json_schema
    json_schema:
      name: result
      schema: { type: object, properties: { answer: { type: string } } }
```

Serializes to provider-specific formats:

```go
exec := spec.Execution
openAI, _ := exec.ProviderFormat("openai")
anthropic, _ := exec.ProviderFormat("anthropic")
gemini, _ := exec.ProviderFormat("gemini")
```

Supported providers: OpenAI, Anthropic, Gemini, vLLM, Mistral, Cohere.

## Working with Specs

### Parse & Validate

```go
// Parse from string
spec, _ := exons.Parse(source)

// Parse from file
spec, _ := exons.ParseFile("agent.exons")

// Validate
err := spec.Validate()

// Validate credential references
err := spec.ValidateCredentialRefs()
```

### Serialize & Export

```go
// Serialize to YAML+body string
output, _ := spec.Serialize(exons.DefaultSerializeOptions())

// Full export including credentials
output, _ := spec.Serialize(exons.FullExportWithCredentials())

// Agent Skills compatible export
output, _ := spec.ExportToSkillMD()
```

### Import

```go
// Import from .md, .zip, .prompty, or .genspec files
result, _ := exons.Import(data, "agent.zip")
spec := result.Spec

// Import a .prompty file (auto-converts {~prompty.~} tags to {~exons.~})
spec, _ := exons.ImportPrompty(promptyData)

// Import from SKILL.md format
spec, _ := exons.ImportFromSkillMD(content)
```

### Clone

```go
copy := spec.Clone() // Deep copy of all fields
```

## Template Engine

### Basics

```go
engine := exons.MustNew()

// Parse and execute
tmpl, _ := engine.Parse(source)
output, _ := tmpl.Execute(ctx, data)

// Register reusable templates
engine.MustRegisterTemplate("header", "Welcome to {~exons.var name=\"site\" /~}")
```

### Custom Resolvers

```go
type MyResolver struct{}
func (r *MyResolver) TagName() string { return "MyTag" }
func (r *MyResolver) Resolve(ctx context.Context, execCtx *exons.Context, attrs exons.Attributes) (string, error) {
    return "resolved", nil
}
func (r *MyResolver) Validate(attrs exons.Attributes) error { return nil }

engine.RegisterResolver(&MyResolver{})
```

### Custom Functions

```go
engine.RegisterFunc("shout", func(args ...any) (any, error) {
    return strings.ToUpper(fmt.Sprint(args[0])), nil
})
// Use in templates: {~exons.var name="x" /~} → shout(x)
```

### Message Extraction

```go
messages, _ := tmpl.ExecuteAndExtractMessages(ctx, data)
// Returns []Message with Role and Content fields
```

### Spec Resolution

```go
resolver := exons.NewMapSpecResolver()
resolver.Add("web-search", searchSpec, searchBody)
engine.SetSpecResolver(resolver)

// Now {~exons.ref slug="web-search" /~} resolves automatically
```

## Catalogs

```go
// Auto-generate skill/tool catalogs and inject into template context
result, _ := engine.ExecuteWithCatalogs(ctx, source, data, spec, exons.CatalogFormatDefault)
```

Formats: `default` (markdown), `detailed`, `compact`, `function_calling` (JSON schema).

## A2A Agent Cards

Generate [Google A2A protocol](https://github.com/google/a2a-spec) Agent Cards from Spec metadata. Pure metadata transformation — no template execution or network calls.

```go
card, _ := spec.CompileAgentCard(ctx, &exons.A2ACardOptions{
    URL:                  "https://agents.example.com/dns",
    ProviderOrganization: "Acme Corp",
    Resolver:             myResolver,
})
jsonBytes, _ := card.ToJSONPretty()
```

Metadata enriches Agent Cards: dispatch keywords become skill tags, registry version becomes the card version, safety config appears in card metadata.

## Token Estimation

```go
estimate, _ := exons.EstimateTokens(source, data)
// estimate.InputTokens, estimate.OutputTokens, estimate.TotalTokens
```

## Debug & Validation

```go
// AST validation (checks for unknown tags, missing attributes)
result := engine.Validate(source)

// Dry run (static analysis without execution)
dryRun, _ := tmpl.DryRun()

// Human-readable execution walkthrough
explanation, _ := tmpl.Explain(ctx, data)
```

## Editor Support

VS Code syntax highlighting for `.exons` files is included in `editor/vscode/`.

## Lineage

go-exons evolves from [go-prompty](https://github.com/itsatony/go-prompty), inheriting its battle-tested template engine (lexer, parser, expression evaluator) while redesigning the public API for the agent specification use case.

## License

MIT — see [LICENSE](LICENSE).
