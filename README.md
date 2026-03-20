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
Conditional:    {~exons.if eval="user.isAdmin"~}...{~/exons.if~}
Loop:           {~exons.for item="x" in="items"~}...{~/exons.for~}
Include:        {~exons.include template="header" /~}
Message:        {~exons.message role="system"~}...{~/exons.message~}
Raw:            {~exons.raw~}not parsed{~/exons.raw~}
Comment:        {~exons.comment~}removed{~/exons.comment~}
Escape:         \{~ produces literal {~
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
