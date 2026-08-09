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

An optional `subtype` refines `type`. Today it is meaningful only for `prompt`:

| Subtype | Meaning |
|---|---|
| `fragment` | A small composable piece that exists to be *referenced* by something larger. |
| `template` | A pre-written prompt carrying runtime `inputs`, rendered per-run. |

`subtype` is **advisory** — go-exons records it and never rejects an unrecognised value.
The consumer that stores or executes the document enforces its own vocabulary. It is
serialized only alongside `type`, since a subtype that refines nothing is meaningless.

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
| Variable | `{~exons.var name="user.name" default="Guest" join=", " /~}` |
| Input | `{~exons.input name="tone" join=", " /~}` — a value the document **declared** |
| Conditional | `{~exons.if eval="user.isAdmin"~}...{~exons.else~}...{~/exons.if~}` |
| Loop | `{~exons.for item="x" index="i" in="items"~}...{~/exons.for~}` |
| Include | `{~exons.include template="header" /~}` |
| Message | `{~exons.message role="system"~}...{~/exons.message~}` |
| Ref | `{~exons.ref slug="my-skill" /~}` |
| Switch | `{~exons.switch eval="x"~}{~exons.case value="a"~}...{~/exons.case~}{~/exons.switch~}` |
| Skills Catalog | `{~exons.skills_catalog /~}` |
| Tools Catalog | `{~exons.tools_catalog /~}` |
| Env | `{~exons.env name="API_KEY" default="none" /~}` |
| Now | `{~exons.now format="date" tz="Europe/Berlin" /~}` |
| Extends | `{~exons.extends template="parent"~}` |
| Block | `{~exons.block name="content"~}...{~/exons.block~}` |
| Raw | `{~exons.raw~}not parsed{~/exons.raw~}` |
| Comment | `{~exons.comment~}removed from output{~/exons.comment~}` |
| Escape | `\{~` produces literal `{~` |
| Verbatim fence | `{~~ ... ~~}` emits its body byte-for-byte |

### `exons.input` vs `exons.var`

They look alike and are not interchangeable. `exons.var` reads **anything in the
execution context** — the caller's data, an include's `with=`, a loop's inherited
scope. `exons.input` reads only what the **document declared** in its frontmatter
`inputs:` block.

```yaml
---
name: summarizer
description: Summarize text into bullets.
type: prompt
subtype: template
inputs:
  text:
    type: text
  max_bullets:
    type: number
    default: 5
---
Summarize into at most {~exons.input name="max_bullets" /~} bullets.

{~exons.input name="text" /~}
```

Declared inputs are injected into the context under the reserved root `input` before
execution, with each `default:` applied wherever the caller bound nothing. So
`{~exons.input name="max_bullets"~}` *is* the path `input.max_bullets`, and control
flow reaches declared inputs with no special syntax:

```
{~exons.if eval="input.verbose"~}...{~/exons.if~}
{~exons.for item="s" in="input.sources"~}...{~/exons.for~}
```

Why the split exists: before it, both were spelled `{~exons.var~}`, so nothing could
distinguish a mistyped input from a legitimate context variable — which made "this
declared input is never referenced" undecidable for any tool, however careful. With
two verbs the question is answerable by construction, and
`Template.DryRun(...).Inputs` is the surface that answers it.

That answer is only sound when the analysis was **complete**: check
`result.AnalysisComplete()` before concluding a declared name is referenced nowhere,
because that conclusion licenses telling an author to delete a declaration, and a
region the walk could not reach holds references that are *unknown*, not *absent*.
`DryRunResult.Errors` carries the reasons.

Three properties worth knowing:

- A declared input that is neither bound nor defaulted is **present and nil**, so it
  renders empty, evaluates falsy, and iterates zero times. Because *present ⇔
  declared*, an **absent** name is an author typo, and `exons.input` says so.
- Bind values with `Spec.BindInputs`, and check them with `Spec.ValidateInputBinding`
  **before** rendering. That is why `exons.input` refuses a `required=` attribute —
  render time is too late to ask a user for a value.
- A value bound to an input is swept for byte slices at every depth and they are
  withheld, so an uploaded file's body can never reach the prompt by accident. A list
  of `{name, mime_type, size_bytes}` maps renders as a filename manifest.

### How a value renders

`{~exons.var~}` substitutes text into a prompt a language model will read, so the
rendering is prose-shaped, not Go-shaped:

| Value | Renders as |
|---|---|
| a string, number, boolean | itself (`19.99`, `true`) |
| `nil`, a nil slice/map | the empty string |
| a list | `cost, speed` — element order preserved |
| an object | `beta: 2, gamma: 3` — keys **sorted**, because a Go map has none |
| a list of objects | `(left: eu, right: gdpr), (left: us, right: ccpa)` |
| a `fmt.Stringer` | whatever it says |

Nested composites keep delimiters (`[…]`, `(…)`) so a list of pairs cannot smear
into one comma-run; the top level is unwrapped because that is what reads correctly
inside a sentence. `join` overrides the top-level separator — useful for a `sort`
value, whose ranking is the point:

```
Rank, best first: {~exons.var name="criteria" join=" > " /~}
→ Rank, best first: cost > speed > quality
```

### `exons.now`

`{~exons.now~}` prints a formatted timestamp into the body. The date/time *expression*
functions (`now`, `formatDate`, `year`, …) only exist inside `eval=`, so before this tag
there was no way to drop today's date into a prompt.

`format=` names a format from a **closed vocabulary** — it is not a Go layout string:

| `format=` | Renders as |
|---|---|
| omitted, or `iso` | `2026-07-21T14:30:00Z` (RFC-3339) |
| `date` | `2026-07-21` |
| `datetime` | `2026-07-21 14:30:00` |
| `time` | `14:30:00` |
| `year` | `2026` |
| `month` | `07` |
| `day` | `21` |
| `weekday` | `Tuesday` |
| `unix` | seconds since the epoch |
| `rfc1123` | `Tue, 21 Jul 2026 14:30:00 UTC` |
| `date-de` | `21.07.2026` |

`layout=` is the escape hatch for anything the vocabulary does not cover, and it takes a
raw Go layout. When both are present **`layout=` wins** and `format=` is ignored:

```
{~exons.now layout="Mon Jan 2" /~}   → Tue Jul 21
```

`tz=` takes an IANA name. The default is **UTC**, not the host's local zone: the format
names carry no offset, so a naked wall clock would be ambiguous for a document rendered
across locales.

```
{~exons.now format="datetime" tz="Europe/Berlin" /~}   → 2026-07-21 16:30:00
```

Every attribute is optional, and an unknown format name or unloadable timezone is reported
from *resolve* time rather than parse time — so the engine's configured error strategy
governs it, instead of hard-failing a template a lenient strategy would otherwise render.

The reference time is seeded once per render, so every `{~exons.now~}` in one render agrees
and a test can pin an exact instant. Unseeded, it falls back to `time.Now()`.

### Writing exons syntax as content

Three escape layers, from smallest to largest (full spec: [docs/template-syntax.md](docs/template-syntax.md)):

| Mechanism | Use for | Notes |
|---|---|---|
| `\{~` | a single literal `{~` | one-off inline escape |
| `{~~ ... ~~}` | any verbatim region | body is emitted byte-for-byte and may contain broken syntax, raw blocks, anything; if the body contains `~~}`, add one more tilde per side (`{~~~ ... ~~~}`, etc.) |
| `{~exons.raw~}...{~/exons.raw~}` | self-documenting verbatim | scans to the **first** `{~/exons.raw~}` — cannot contain its own close; use a `{~~` fence for that |

For markdown-format templates (SKILL.md-style bodies), enable `WithMarkdownFences()`: fenced code blocks (```` ``` ```` or `~~~`) become inert so syntax examples never execute, and a fence renders live when its info string starts with `exons` (```` ```exons ````). `ImportFromSkillMD` marks specs with `ContentFormat: "markdown"` so consumers know to enable it. `Validate()` warns when tag-like syntax sits in an inert fence.

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

## Input Kinds

`inputs` declares the parameters a document takes. Each entry is an `InputDef` whose
`type` names an input **kind** — enough for a consumer to render a form control.

| Kind | Bound value | Reads |
|---|---|---|
| `text` | a string | — |
| `number` | a number | — |
| `boolean` | true/false | — |
| `select` | one option value | `options` |
| `multiselect` | a list of option values | `options` |
| `sort` | `options` reordered — declared order is the initial order | `options` |
| `associate` | many-to-many `(options, associate_with)` pairs | `options`, `associate_with` |
| `file-upload` | uploaded files | `accept`, `max_size_bytes`, `max_files` |

`max_size_bytes` bounds each **individual** file; `max_files` bounds **how many**.

```yaml
inputs:
  months:
    type: multiselect
    label: Months to analyse
    options:
      - { value: jan, label: January }
      - { value: feb, label: February }
  report:
    type: file-upload
    accept: ["application/pdf"]
    max_size_bytes: 10485760
    max_files: 3
  owners:
    type: associate
    options:        [{ value: region }]
    associate_with: [{ value: analyst }]
```

**The kind vocabulary is advisory and open.** `Spec.Validate()` does not inspect an
input's *kind* at all — go-exons *declares*, the executing system *validates*. An
unknown kind is legal and forward-compatible; a consumer that does not recognise one
should degrade it to a plain text control rather than reject the document. `inputs`
carries declarations only — never values, never uploaded content.

### Input order

`Spec.Inputs` is a Go map, so the order you wrote the inputs in does not survive the
YAML unmarshal — and a form is a sequence of questions, not a set. Parsing therefore
records the authored key order in **`Spec.InputOrder`**, and consumers read it
through **`Spec.OrderedInputKeys()`**:

```go
spec, _ := exons.ParseFile("intake.exons")
for _, key := range spec.OrderedInputKeys() {   // authored order, not alphabetical
    in := spec.Inputs[key]
    // …render the control
}
```

`OrderedInputKeys()` is **total**: it returns every declared input exactly once —
the ordered ones first, then anything unordered sorted by key — so a `Spec` built in
Go with no order at all still yields a stable sequence instead of Go's randomised map
iteration. Prefer it over `range spec.Inputs` everywhere a human sees the result.

The order serializes as `input_order`, so it survives a JSON or YAML round-trip
through a store that keeps only the projection. You may also write it by hand, which
overrides the authored order:

```yaml
input_order: [full_name, address, zip_code]
inputs:
  zip_code:  { type: text }
  full_name: { type: text }
  address:   { type: text }
```

A hand-written `input_order` is the one input-related thing `Validate()` does check:
naming an input that is not declared, or naming one twice, is an author's
contradiction and is rejected. A derived order cannot fail either check.

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
engine.RegisterFunc(&exons.Func{
    Name:    "shout",
    MinArgs: 1,
    MaxArgs: 1,
    Fn: func(args []any) (any, error) {
        return strings.ToUpper(fmt.Sprint(args[0])), nil
    },
})
// Use in expressions: {~exons.if eval="shout(name) == 'ALICE'"~}...{~/exons.if~}
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
// Estimate tokens for raw text
estimate := exons.EstimateTokens(source)
// estimate.EstimatedGPT, estimate.EstimatedClaude, estimate.EstimatedGeneric

// Or estimate after template execution
estimate, _ := tmpl.EstimateTokens(ctx, data)
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

## Tool Format Export

Define tools once in the spec, export to any provider's format:

```go
spec, _ := exons.Parse(data)
for _, fn := range spec.Tools.Functions {
    openai := fn.ToOpenAITool()       // {"type":"function","function":{...}}
    anthropic := fn.ToAnthropicTool() // {"name":...,"input_schema":{...}}
    gemini := fn.ToGeminiTool()       // {"name":...,"parameters":{...}}
    mcp := fn.ToMCPTool()             // {"name":...,"inputSchema":{...}}
    cohere := fn.ToCohereTool()       // {"name":...,"parameter_definitions":{...}}
    mistral := fn.ToMistralTool()     // OpenAI-compatible format
}

// Or use batch methods on ToolsConfig
openAITools := spec.Tools.ToOpenAITools()
anthropicTools := spec.Tools.ToAnthropicTools()
```

## Security

The template engine is security-hardened by default:

- **Env var access control**: `{~exons.env~}` blocks common secret patterns (`*_KEY`, `*_SECRET`, `*_TOKEN`, `*_PASSWORD`, etc.) by default. Configure with `WithEnvAllowlist`, `WithEnvDenylist`, or `WithEnvDisabled`.
- **Output size limits**: Rendered output capped at 10MB by default (`WithMaxOutputSize` to override).
- **Zip import protection**: Path traversal and decompression bomb defenses built in.
- **Recursion limits**: Template inclusion, inheritance, and ref resolution all have configurable depth limits.

## JSON Schema

A JSON Schema for validating `.exons` YAML frontmatter ships at `schema/exons.schema.json`. Use it with VS Code's YAML extension or in CI pipelines.

## Examples

The `examples/` directory contains 8 standalone Go programs covering core workflows. Each is runnable with `go run .`:

1. `01-basic-prompt` — Parse and execute a template
2. `02-chat-agent` — Extract structured messages
3. `03-custom-resolver` — Extend the template engine
4. `04-tool-export` — Export tools to all provider formats
5. `05-template-composition` — Compose agents from skills via SpecResolver
6. `06-validation-and-debug` — Validate, dry-run, explain
7. `07-a2a-agent-card` — Generate A2A Agent Cards
8. `08-syntax-safety` — Write exons syntax as content: markdown fences, verbatim fences, raw blocks

Alongside them, `examples/dns-specialist.exons` is a worked agent document — the full frontmatter surface (execution, inputs, tools, memory, dispatch, verifications, registry, safety, constraints) on one realistic agent, for reading rather than running.

## Editor Support

VS Code syntax highlighting for `.exons` files is included in `editor/vscode/`.

## Lineage

go-exons evolves from [go-prompty](https://github.com/itsatony/go-prompty), inheriting its battle-tested template engine (lexer, parser, expression evaluator) while redesigning the public API for the agent specification use case.

## License

MIT — see [LICENSE](LICENSE).
