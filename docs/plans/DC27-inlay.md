# DC27-inlay — edit a stored definition without damaging it, and give "none" a spelling

> **Status:** built as **v0.40.0** on branch `feat/v0.40.0-resource-modes`; release pending.
> Driver: **vAudience/atlas#803** (owner ruling 2026-10-03: *"the upstream exons change is fine —
> we execute it; if there are any other improvements we should make to exons, this is the time"*).
> Also closes **go-exons#13** (deepr#385).
>
> *inlay*, n. — a piece set into a surface so that the surface around it is untouched.

---

## Why this cycle exists

atlas#803 makes an agent's setup (tools, corpora, MCP servers, skills, model) editable from the
composer, as an edit to the agent's exons definition in aigentverse. Two things in go-exons stood
in the way:

1. **"None" had no spelling for resources.** Tools could already say all three things
   (`tools.allow` absent / `[]` / a list, since v0.32.0). Resources could say *all* (declare
   nothing) and *listed* (declare entries) but not *none of this kind*.
2. **There was no safe way to change one field of a stored document.** Parse → edit → Serialize
   drops `credentials:` (ExportFull excludes them by design), drops every comment and re-sorts
   keys, and — through `Engine.Parse` — RENDERS `{~…~}` tags in the frontmatter, so an author's
   templated model is saved as today's value of the variable.

---

## What shipped

| Item | Change | Kind |
|---|---|---|
| 1a | `requirements.resource_modes: {kind: all\|listed\|none}`; `SpecRequirements.ResourceMode(kind)`; `Spec.ToolMode()` from `tools.allow` | additive |
| B1 | `PatchSource(src, edits...)` with typed `Set…` edits; `ErrPatchRefused` + six specific sentinels, `*PatchError` | additive |
| B3 | `tools.allow` / `mcp_servers[].tools`: non-empty, unique, ≤ `MaxToolAllowEntries` (512) | ⚠ **behaviour change** (narrowing) |
| B4 | `WithStrictRenderability()`: `Validate` reports at error severity what this engine can never execute | additive, opt-in |
| B5 | `docs/tools-and-resources.md` | docs |

Rejected by the owner and **not** built: glob syntax in `tools.allow`; a pluggable slug grammar
(go-exons#1 stays open); an agent image field.

---

## The decisions, and the evidence for each

**1. One spelling per fact.** `resource_modes` covers resources only. A `resource_modes.tools`
key, or a `tools.mode`, would let a document say `allow: [a]` and `mode: none` at once; every
consumer would then have to pick one, and a runtime and a UI would pick differently.
`Spec.ToolMode()` derives the mode from the one field that already said it.

**2. An absent mode is DERIVED, never "unset".** A document written before v0.40.0 has no
`resource_modes` key, and it meant something: entries of a kind → listed, none → all.
`ResourceMode(kind)` returns exactly that, so a consumer calls one accessor on every document.

**3. Contradictions are refused, not resolved.** `all`/`none` beside entries of the same kind
would leave entries a reader takes for a narrowing that no runtime applies; `listed` with no entry
is `none` written as an empty list. Both are cross-field rules JSON Schema cannot express, so the
agreement test gains one closed divergence reason (`parserOnlyResourceModes`) with three rows.

**4. PatchSource splices TEXT; it does not re-encode the document.** yaml.v3 can round-trip a
`yaml.Node` with comments, but it re-indents, drops blank lines and moves comments — "preserved"
would have meant "mostly". Instead the node tree is used only to LOCATE the edited entry
(key line → last content line of its value); only those lines are replaced, with the value
rendered at the entry's own column. Everything else is copied, so byte equality outside the edit
is true by construction, and a golden test pins it per edit.

**5. PatchSource never renders, and refuses rather than guesses.** It requires the source to pass
`exons.Parse` (which decodes the frontmatter as written; `Engine.Parse` is the one that renders).
A frontmatter that is YAML only after rendering cannot be edited without rendering it, so it is
refused with a message saying to quote the tag — the engine's documented advice already.

**6. The self-check is the safety net for every shape not anticipated.** After splicing, the
result must Parse/Validate AND decode to the original with the same edits applied in Go, compared
by canonical YAML encoding (the allow-lists keep nil apart from `[]` through their own
marshallers). Pruning an emptied block (`tools:` left with no keys) is reported by the text edit
and mirrored on the Go side per edit, so the two cannot disagree about it. A revert-check proves
the self-check is live: an edit whose Go twin disagrees with its text is refused.

**7. B3 is a narrowing in a minor release, stated.** Before v0.40.0 nothing validated the
allow-lists. A duplicate is harmless to a set-based runtime, but an empty name matches nothing and
an oversized list is a paste. 512 is atlas's per-turn capability ceiling, so no real narrowing is
near it. Consumers must re-validate stored documents before bumping (see the CHANGELOG).

**8. B4 is a separate option, and Parse does not change.** `WithStrictAttributes` judges whether a
tag is well formed for ANY engine, and its consumers (aigentverse) register resolvers after the
parse. Raising unknown tags to errors there would refuse every custom tag. Renderability is a
question about THIS engine, asked in `Validate`, so `ValidationResult` carries an unexported
`renderability` flag that only `Engine.Validate` sets; the strict-attributes walk from Parse
builds its own result without it (revert-checked). The env check calls the env resolver's own
`CheckRenderable`, which runs the same two checks `Resolve` runs, and every test row renders as
well as validates.

---

## Evidence

- Revert-checks (each guard broken, the named tests fail): PatchSource returning ExportFull
  instead of the splice; the self-check disabled; `entryLines` treating comment lines as
  terminators; the `none/all beside entries` arm; the `listed with nothing` arm; the derivation
  of `ResourceMode`; `ToolMode` reading `[]` as all; `Spec.Validate` not calling `Tools.Validate`;
  the prompt prohibition; renderability's unknown-tag arm; renderability's env arm; renderability
  leaking into the strict-attributes Parse walk.
- Agreement corpus: 105 rows (80 agree, 6 parser-stricter, 19 schema-stricter).
- `make ci-local` green on the branch: build, vet, gofmt, tidy, golangci-lint v2.12.2 (0 issues),
  race tests, total coverage 91.0 % (floor 88; root package 92.0 %).
