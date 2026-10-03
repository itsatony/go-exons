# Tools, MCP and resources: which field says what

An agent definition has **four** places that mention tools or MCP servers. They are not
synonyms. Each answers a different question, is resolved by a different party, and survives a
copy between tenants differently. Reading one as another is how a runtime ends up enforcing one
list while a UI displays another.

| Field | What it is | Names a concrete location? | Who resolves it | Declaration or narrowing? |
|---|---|---|---|---|
| `tools.functions[]` | Inline tool **definitions**: name, description, JSON-Schema parameters | No | Nobody — the document carries the whole definition | **Declaration** (adds tools) |
| `tools.mcp_servers[]` | A **URL-bound** MCP server: `name`, `url`, `transport`, optional `tools` | **Yes** — the URL | The runtime connects to the URL as written | **Declaration** (adds a server's tools); its `tools` list **narrows that one server** |
| `tools.allow` | Allow-list over the tool names offered **from every source** (inline functions, MCP servers, tools a runtime adds itself) | No | The runtime, by exact name match | **Narrowing** only — never adds a tool |
| `requirements.mcp[]` | An **abstract MCP capability** the definition needs (`dns-management`), with a logical credential ref and scope | No | A router maps the capability to a concrete server at runtime | **Declaration of need** (portable; adds nothing by itself) |
| `requirements.resources[]` with `kind: mcp_server` (or any kind) | A **logical resource** the definition may use (`ref: crm-tools`, `kind: mcp_server`) | No — `://` is refused | A registry binds the ref to a concrete coordinate per workspace (e.g. aigentverse `/resource-bindings`) | **Narrowing**: the definition may use the listed resources of that kind and no other |
| `requirements.resource_modes.<kind>` (v0.40.0) | Per kind: `all`, `listed` or `none` | No | Read through `SpecRequirements.ResourceMode(kind)` | Says whether the kind is unrestricted, narrowed to the list, or closed |

## The two narrowings, and why each has three answers

A narrowing must be able to say three different things: *everything*, *only these*, and
*nothing*. Absent and empty must therefore mean different things, and a round trip must keep them
apart.

**Tools** already could, through `tools.allow` alone (since v0.32.0 the empty form survives a
round trip):

| `tools.allow` | `Spec.ToolMode()` | Meaning |
|---|---|---|
| absent (or no `tools:` block) | `all` | no narrowing — every tool from every source |
| `[]` | `none` | no tools at all |
| `[a, b]` | `listed` | only `a` and `b` |

There is deliberately **no** `resource_modes.tools` or any second spelling for tools: two
spellings for one fact is a document that can say both at once.

**Resources** could not: an absent `resources` list meant "no narrowing", and there was no way to
write "none of this kind". `requirements.resource_modes` adds it, per kind:

```yaml
requirements:
  resources:
    - ref: churn-docs
      kind: corpus
  resource_modes:
    corpus: listed       # only churn-docs (the default when entries of the kind exist)
    mcp_server: none     # no MCP server resources at all
    # folder: absent → all (the default when no entry of the kind exists)
```

`ResourceMode(kind)` returns the declared mode, or derives it exactly as a pre-v0.40.0 document
always meant it: entries of the kind present → `listed`, none → `all`. `Parse` refuses `all` or
`none` beside entries of the same kind (the entries would be dead text a reader mistakes for a
narrowing) and `listed` with no entry of the kind (that is `none` spelled as a list nobody wrote).
A prompt refuses the key, as it refuses `requirements.environment`.

## Which to write

- The agent should be offered **fewer tools** than its runtime has → `tools.allow`.
- The agent needs a capability any compliant server could provide, and the definition must stay
  portable → `requirements.mcp[]`.
- The agent may use **specific** MCP servers that each workspace binds for itself →
  `requirements.resources[]` with `kind: mcp_server`, and `resource_modes.mcp_server` to say
  `none` when it may use none.
- The agent is pinned to one server at one address, and will never be copied into another
  tenant → `tools.mcp_servers[]`. The URL travels with every export; prefer a resource binding
  wherever the definition is shared.

go-exons attaches no meaning to a resource `kind` beyond its shape (`ResourceKindPattern`);
`corpus`, `toolset` and `mcp_server` are conventions of the runtimes that bind them.

## Editing these fields without damaging the document

`exons.PatchSource(src, edits...)` (v0.40.0) edits `tools.allow`, `requirements.resources`,
`requirements.resource_modes.<kind>`, `skills`, `execution.provider` / `model` /
`reasoning_effort` and `display_name` in the frontmatter text, leaving comments, key order,
unknown keys, credentials and templated values untouched. Prefer it over Parse → Serialize for a
stored definition: `ExportFull` drops credentials and comments, and `Engine.Parse` renders
frontmatter tags.

```go
out, err := exons.PatchSource(src,
    exons.SetToolsAllow([]string{}),                         // none
    exons.SetResourceMode("mcp_server", exons.ResourceModeNone),
)
if errors.Is(err, exons.ErrPatchRefused) { /* nothing was written */ }
```
