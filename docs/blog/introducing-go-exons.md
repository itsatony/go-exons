# Introducing go-exons: The Declarative Agent Specification Format for Go

Every AI agent framework has its own way of defining agents. CrewAI uses YAML with role/goal/backstory patterns. LangChain chains functions in Python. Semantic Kernel defines agents in C#/.NET config. Each format is tightly coupled to its runtime, making agents non-portable and hard to audit.

**go-exons** is a different approach: a declarative specification format that describes *what* an agent is — not how it runs. A single `.exons` file captures everything about an agent: identity, execution parameters, tools, memory, safety constraints, verification cases, and the prompt itself. Pure Go. No runtime. No LLM calls. Just parsing, validation, and serialization.

```
go get github.com/itsatony/go-exons
```

## The Problem

If you're building agent systems in Go, you've probably hit these walls:

**Template collisions.** Your agent prompts contain JSON examples, XML tags, Go template syntax, or code snippets. Every template engine using `{{ }}`, `${}`, or `{variable}` collides with your content, forcing you to escape everything. One missed escape and your prompt breaks silently.

**Provider lock-in.** You define tool schemas for OpenAI's format. Now you need to support Anthropic. And Gemini. Each has a slightly different JSON structure for the same concept. You end up writing adapter code for every provider.

**Configuration sprawl.** Your agent's identity lives in one file, its tools in another, its safety rules in a third, its prompt in a fourth. Code review means chasing across files. Git diffs are meaningless. There's no single source of truth.

**No governance story.** Regulators and security teams ask: "Show me what this agent can do, what it can't do, and how you verify that." You point at scattered code and configuration. They're not impressed.

## The Solution: `.exons` Files

An `.exons` file uses YAML frontmatter for configuration and a content-resistant `{~...~}` template syntax for the prompt:

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

tools:
  allow: [dns_list_records, dns_create_record, dns_delete_record]
  functions:
    - name: check_propagation
      description: Check DNS propagation status worldwide
      parameters:
        type: object
        properties:
          domain: { type: string }
          record_type: { type: string, enum: [A, AAAA, CNAME, MX, TXT] }
        required: [domain]

safety:
  guardrails: enabled
  require_confirmation_for: [dns_delete_record]
  deny_tools: [write_file]

verifications:
  - name: can-list-records
    prompt: "List all DNS records for the test zone"
    expect:
      tool_calls: [dns_list_records]
      output_contains: "records"
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

One file. Complete agent. Diffable. Reviewable. Testable.

## Why `{~...~}`?

The tilde delimiter was chosen deliberately. Consider what happens when your agent prompt contains examples:

```
Agent prompt containing JSON: {"key": "value"}
Agent prompt containing Go templates: {{.Name}}
Agent prompt containing shell: ${HOME}/config
Agent prompt containing Jinja: {{ user.name }}
Agent prompt containing XML: <tool name="search">
```

With Jinja or Go templates, you'd need to escape every one of these. With `{~...~}`, the content passes through untouched. The probability of `{~` appearing naturally in any prompt content is effectively zero.

This isn't a minor convenience — it's a **correctness guarantee**. For agents that handle code, schemas, or structured data, content-resistant syntax eliminates an entire class of silent failures.

## Key Features

### Parse, Validate, Execute Templates

```go
engine := exons.MustNew()
tmpl, _ := engine.Parse(source)

// Execute and extract structured messages
messages, _ := tmpl.ExecuteAndExtractMessages(ctx, map[string]any{
    "user_name": "Alice",
})
// messages[0] → {Role: "system", Content: "You are a DNS specialist..."}
// messages[1] → {Role: "user",   Content: "What DNS records exist?"}
```

### Export Tools to Any Provider

Define tools once, export to six provider formats:

```go
spec, _ := exons.Parse(data)
for _, fn := range spec.Tools.Functions {
    openai := fn.ToOpenAITool()       // {"type":"function","function":{...}}
    anthropic := fn.ToAnthropicTool() // {"name":...,"input_schema":{...}}
    gemini := fn.ToGeminiTool()       // {"name":...,"parameters":{...}}
    mcp := fn.ToMCPTool()             // {"name":...,"inputSchema":{...}}
    cohere := fn.ToCohereTool()       // {"name":...,"parameter_definitions":{...}}
}
```

Or use batch methods: `spec.Tools.ToOpenAITools()`, `spec.Tools.ToAnthropicTools()`, etc.

### Multi-Provider Execution Config

Define execution parameters once in the spec, serialize to any provider's API format:

```go
exec := spec.Execution
openAIParams, _ := exec.ProviderFormat("openai")
anthropicParams, _ := exec.ProviderFormat("anthropic")
geminiParams, _ := exec.ProviderFormat("gemini")
```

Supports OpenAI, Anthropic, Gemini, vLLM, Mistral, and Cohere.

### A2A Agent Cards

Generate [Google A2A protocol](https://github.com/google/a2a-spec) Agent Cards directly from spec metadata:

```go
card, _ := spec.CompileAgentCard(ctx, &exons.A2ACardOptions{
    URL:                  "https://agents.example.com/dns",
    ProviderOrganization: "Acme Corp",
})
jsonBytes, _ := card.ToJSONPretty()
```

Dispatch keywords become skill tags. Registry version becomes the card version. Safety config appears in card metadata. Pure metadata transformation — no LLM calls.

### JSON Schema Validation

A JSON Schema for `.exons` files ships with the library at `schema/exons.schema.json`. Wire it into your VS Code YAML extension for autocomplete and validation, or use it in CI to validate agent definitions before deployment.

### Security by Default

The template engine is security-hardened:

- **Env var denylist**: `{~exons.env~}` blocks `*_KEY`, `*_SECRET`, `*_TOKEN`, `*_PASSWORD` patterns by default. Override explicitly with `WithEnvAllowlist` or `WithEnvDenylist`.
- **Output size limits**: Rendered output is capped at 10MB by default to prevent memory exhaustion.
- **Zip import protection**: Path traversal and decompression bomb defenses built in.
- **Recursion limits**: Template inclusion, inheritance, and ref resolution all have configurable depth limits.

### Debug and Introspection

```go
// Dry run — static analysis without execution
dryRun, _ := tmpl.DryRun()

// Human-readable execution walkthrough
explanation, _ := tmpl.Explain(ctx, data)

// AST validation
result := engine.Validate(source)
```

## Document Types

| Type | Description |
|---|---|
| `prompt` | Simple template — variables, conditionals, loops. No skills/tools. |
| `skill` | Reusable capability with inputs/outputs. May have memory, registry. |
| `agent` | Full agent with tools, skills, constraints, all metadata fields. |

## Getting Started

The `examples/` directory contains 7 runnable examples covering the core workflows:

1. **Basic prompt** — parse and execute a template
2. **Chat agent** — extract structured messages
3. **Custom resolver** — extend the template engine
4. **Tool export** — export to all provider formats
5. **Template composition** — compose agents from skills
6. **Validation and debug** — validate, dry-run, explain
7. **A2A agent card** — generate discovery metadata

Each is a standalone Go program: `cd examples/01-basic-prompt && go run .`

## What's Next

- **Multi-language ports**: Python, TypeScript, C#, and Java implementations are planned. The JSON Schema ensures format compatibility across languages.
- **exons-run**: A CLI + MCP server that executes `.exons` files against LLM providers, bridging the spec format with actual agent execution.
- **Ecosystem integrations**: Adapters for Go agent frameworks (Google ADK, Eino, LangChainGo) to load `.exons` files as agent definitions.
- **SchemaStore.org + VS Code Marketplace**: Publishing the JSON Schema and VS Code extension for broader discoverability.

## The Bigger Picture

The agent specification space is fragmenting. Oracle, Microsoft, IBM, and others have all released their own formats. None has achieved the equivalent of OpenAPI for APIs. None has a native Go implementation.

go-exons aims to be that standard for Go — and with multi-language ports, potentially beyond. One file format. Any provider. Any language. Content-resistant by design.

**MIT licensed. Production-ready. [Get started on GitHub.](https://github.com/itsatony/go-exons)**
