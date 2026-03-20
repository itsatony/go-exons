# go-exons

**Exons define what agents do.** The functional coding regions of an agent's DNA.

go-exons is a declarative agent specification format for Go. An `.exons` file describes a complete agent: identity, execution parameters, tools, memory, dispatch rules, verification cases, and more — using YAML frontmatter and a content-resistant `{~...~}` template syntax.

```
go get github.com/itsatony/go-exons
```

## Quick Start

**1. Define an agent** (`hello.exons`):

```yaml
---
name: greeter
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

**2. Parse and execute**:

```go
engine := exons.MustNew()
tmpl, _ := engine.Parse(source) // source is the .exons content above

// Execute and extract structured messages for LLM API calls
messages, _ := tmpl.ExecuteAndExtractMessages(ctx, map[string]any{
    "user_name": "Alice",
})
// messages → [{Role: "system", Content: "You are a friendly greeter."},
//             {Role: "user", Content: "Say hello to Alice"}]

// Access the parsed spec (frontmatter)
spec := tmpl.Spec()
fmt.Println(spec.Name)             // "greeter"
fmt.Println(spec.Execution.Model)  // "gpt-4o"
```

## What Problem Does This Solve?

| Without go-exons | With go-exons |
|---|---|
| Agent config scattered across code | Single `.exons` file per agent |
| Go's `{{}}` templates collide with JSON/XML in prompts | `{~...~}` delimiters never collide |
| Provider-specific config hardcoded | Multi-provider serialization built in |
| Manual message assembly per provider | `CompileAgent` → `ToOpenAIMessages()` in one pipeline |
| No standard for agent metadata | GenSpec: memory, dispatch, verification, safety |
| Test definitions separate from spec | Verification cases travel with the agent |

## The `.exons` Format

An `.exons` file has two parts: YAML frontmatter (configuration) and a template body (the prompt).

### Document Types

| Type | Description |
|---|---|
| `prompt` | Simple template — variables, conditionals, loops |
| `skill` | Reusable capability with inputs/outputs |
| `agent` | Full agent with tools, skills, constraints, GenSpec |

### GenSpec — Agent Specification Metadata

GenSpec fields describe *what an agent is*, not just *what it says*:

```yaml
genspec:
  memory:
    scope: my-agent
    auto_recall: true
  dispatch:
    trigger_keywords: [dns, domain]
    trigger_description: Route DNS tasks to this agent
  verifications:
    - name: basic-check
      prompt: "List all records"
      expect:
        tool_calls: [list_records]
  registry:
    namespace: my-agent
    origin: internal
    version: 1.0.0
  safety:
    guardrails: enabled
    deny_tools: [dangerous_tool]
```

### Template Syntax

```
Variable:       {~exons.var name="user.name" default="Guest" /~}
Conditional:    {~exons.if eval="user.isAdmin"~}...{~exons.else~}...{~/exons.if~}
Loop:           {~exons.for item="x" in="items"~}...{~/exons.for~}
Include:        {~exons.include template="header" /~}
Message:        {~exons.message role="system"~}...{~/exons.message~}
Ref:            {~exons.ref slug="my-skill" /~}
Switch:         {~exons.switch eval="x"~}{~exons.case value="a"~}...{~/exons.case~}{~/exons.switch~}
Skills Catalog: {~exons.skills_catalog /~}
Tools Catalog:  {~exons.tools_catalog /~}
Env:            {~exons.env name="API_KEY" default="none" /~}
Extends:        {~exons.extends template="parent"~}
Block:          {~exons.block name="content"~}...{~/exons.block~}
Raw:            {~exons.raw~}not parsed{~/exons.raw~}
Comment:        {~exons.comment~}removed{~/exons.comment~}
Escape:         \{~ produces literal {~
```

### Catalog Generation & Spec Resolution

```go
// Register specs for cross-referencing
resolver := exons.NewMapSpecResolver()
resolver.Add("web-search", searchSpec, searchBody)
engine.SetSpecResolver(resolver)

// Auto-generate skill/tool catalogs and inject into template
result, _ := engine.ExecuteWithCatalogs(ctx, source, data, agentSpec, exons.CatalogFormatDefault)

// Or generate catalogs manually
skillsCatalog, _ := exons.GenerateSkillsCatalog(ctx, skills, resolver, exons.CatalogFormatDetailed)
toolsCatalog, _ := exons.GenerateToolsCatalog(tools, exons.CatalogFormatFunctionCalling)
```

### Agent Compilation

Compile an agent spec into provider-ready API payloads:

```go
// Parse an .exons file
spec, _ := exons.ParseFile("research-agent.exons")

// Compile into messages + execution config + tools + constraints
compiled, _ := spec.CompileAgent(ctx, map[string]any{"query": "climate change"}, &exons.CompileOptions{
    Resolver: resolver, // for skill resolution
})

// Convert to provider-specific format
openAIMessages := compiled.ToOpenAIMessages()       // []map[string]any
anthropicPayload := compiled.ToAnthropicMessages()   // {system, messages}
geminiContents := compiled.ToGeminiContents()         // {system_instruction, contents}

// Or auto-dispatch by provider name
payload, _ := compiled.ToProviderMessages("openai")
```

Activate a specific skill (injects content into system/user messages):

```go
compiled, _ := spec.ActivateSkill(ctx, "web-search", input, opts)
```

Validate without executing (dry run):

```go
result := spec.AgentDryRun(ctx, opts)
if !result.OK() {
    fmt.Println(result.String()) // lists all issues
}
```

High-level convenience wrapper:

```go
executor := exons.NewAgentExecutor(exons.WithAgentResolver(resolver))
compiled, _ := executor.Execute(ctx, source, input)
compiled, _ := executor.ExecuteFile(ctx, "agent.exons", input)
```

### Import / Export

```go
// Import from .md or .zip
result, _ := exons.Import(data, "agent.zip")
spec := result.Spec

// Export to zip archive
zipData, _ := exons.ExportDirectory(spec, resources)

// SKILL.md format (Agent Skills compatible)
spec, _ := exons.ImportFromSkillMD(content)
data, _ := spec.ExportToSkillMD()
```

## Multi-Provider Support

ExecutionConfig serializes to provider-specific formats:

- **OpenAI** / Azure
- **Anthropic** (Claude)
- **Gemini** (Google)
- **vLLM**
- **Mistral**
- **Cohere**

## Editor Support

VS Code syntax highlighting is included in `editor/vscode/`. See the [editor README](editor/vscode/) for installation.

## Lineage

go-exons evolves from [go-prompty](https://github.com/itsatony/go-prompty), inheriting its battle-tested template engine (lexer, parser, expression evaluator) while redesigning the public API for the agent specification use case.

## License

MIT — see [LICENSE](LICENSE).
