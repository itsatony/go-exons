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
| 1a | `requirements.resource_modes: {kind: all\|listed\|none}` (the narrowing; resources stays a declaration); `SpecRequirements.ResourceMode(kind)`; `Spec.ToolMode()` from `tools.allow` | additive |
| B1 | `PatchSource(src, edits...)` with typed `Set…` edits; `ErrPatchRefused` + six specific sentinels, `*PatchError` | additive |
| B3 | `Spec.ValidateStrict()` / `ToolsConfig.Validate()`: allow-lists non-empty, unique, ≤ `MaxToolAllowEntries` (512) — a writer's check; Parse stays tolerant | additive |
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

**2. resources declares, resource_modes narrows; an absent mode is ALL (review M2).** The first
draft derived an absent mode as `listed` when entries of the kind existed. That would have turned
every existing `requirements.resources` declaration into a narrowing the day a runtime read
`ResourceMode` — taking resources away from working agents on a library bump. Now
`ResourceMode(kind)` is `all` unless `resource_modes` says otherwise, and `resources` keeps its
meaning: what the definition needs.

**3. Contradictions are refused, not resolved.** `none` beside entries of the same kind declares a
need the mode forbids; `listed` with no entry is `none` written as an empty list. `all` beside
entries is consistent (needs declared, nothing narrowed). Both refusals are cross-field rules JSON
Schema cannot express, so the agreement test gains a closed divergence reason
(`parserOnlyResourceModes`).

**4. PatchSource splices TEXT; it does not re-encode the document.** An entry is its key line
plus every line indented deeper — `#` lines included, because inside a block or multi-line quoted
scalar they are content (review H2: the first draft skipped them and left value text behind as a
comment). Comment lines at or left of the key's column are never part of an entry. yaml.v3 can round-trip a
`yaml.Node` with comments, but it re-indents, drops blank lines and moves comments — "preserved"
would have meant "mostly". Instead the node tree is used only to LOCATE the edited entry
(key line → last content line of its value); only those lines are replaced, with the value
rendered at the entry's own column. Everything else is copied, so byte equality outside the edit
is true by construction, and a golden test pins it per edit.

**5. PatchSource never renders, and refuses rather than guesses.** It requires the source to pass
`exons.Parse` (which decodes the frontmatter as written; `Engine.Parse` is the one that renders).
A frontmatter that is YAML only after rendering cannot be edited without rendering it, so it is
refused with a message saying to quote the tag — the engine's documented advice already.

**6. The self-check is the safety net for every shape not anticipated.** It has two arms: the
decoded result equals the original with the edits applied in Go, and the result carries no
comment line (yaml.v3's comment fields, as a multiset) the source did not have — the class H2
belonged to. Line breaks yaml.v3 counts but a `\n` split does not (U+0085, U+2028, U+2029, a lone
`\r`) are refused up front, every line range is bounds-checked, and a recovered panic becomes
`ErrPatchInternal` (review H1). After splicing, the
result must Parse/Validate AND decode to the original with the same edits applied in Go, compared
by canonical YAML encoding (the allow-lists keep nil apart from `[]` through their own
marshallers). Pruning an emptied block (`tools:` left with no keys) is reported by the text edit
and mirrored on the Go side per edit, so the two cannot disagree about it. A revert-check proves
the self-check is live: an edit whose Go twin disagrees with its text is refused.

**7. The allow-list rules are a WRITER's check, not a reader's (review H3).** The first draft put
them in `Spec.Validate`, which made `Parse` refuse stored documents that work. They live in
`Spec.ValidateStrict()` now, for a registry's publish step; PatchSource runs it on its result, so
patching a document that breaks it must repair it. 512 is go-exons' own sanity bound on a written
list. The schema states the rules; the agreement test declares the divergence
(`schemaStricterToolLists`). No reader-visible behaviour changes in this release.

**8. B4 is a separate option, Parse does not change, and the severity is Execute's (review M1).**
A finding is an error only when the effective strategy (the tag's `onerror=`, else the engine's)
is throw; under any other strategy Execute carries on, so it is a warning. Tags in `if` / `for` /
`switch` bodies are judged as if reached. `WithStrictAttributes` judges whether a
tag is well formed for ANY engine, and its consumers (aigentverse) register resolvers after the
parse. Raising unknown tags to errors there would refuse every custom tag. Renderability is a
question about THIS engine, asked in `Validate`, so `ValidationResult` carries an unexported
`renderability` flag that only `Engine.Validate` sets; the strict-attributes walk from Parse
builds its own result without it (revert-checked). The env check calls the env resolver's own
`CheckRenderable`, which runs the same two checks `Resolve` runs, and every test row renders as
well as validates.

---

## Evidence

- Revert-checks (each guard broken, the named tests fail). Round 0: PatchSource returning
  ExportFull instead of the splice; the decode self-check disabled; the `none beside entries`
  arm; the `listed with nothing` arm; `ToolMode` reading `[]` as all; the prompt prohibition;
  renderability's unknown-tag and env arms; renderability leaking into the strict-attributes
  Parse walk. Review round 1: the foreign-line-break refusal; the line-range bounds check; the
  panic recover; `#` lines deeper than the key skipped again; the comment self-check; the strict
  check moved back into `Validate`; PatchSource skipping the strict check on its result; the
  severity ignoring `onerror=`/the engine strategy; entries deriving `listed` again; an
  intermediate failure reported as the source's fault; any CRLF meaning CRLF; the key-line
  comment contains-guard; boolean-looking key quoting; the edit cap.
- Agreement corpus: 105 rows (76 agree, 5 parser-stricter, 24 schema-stricter).
- Review round 1 (3 HIGH, 3 MEDIUM, 6 LOW) fixed with a test per finding; see the commit log.
- `make ci-local` green on the branch: build, vet, gofmt, tidy, golangci-lint v2.12.2 (0 issues),
  race tests, total coverage 91.1 % (floor 88; root package 92.0 %).
