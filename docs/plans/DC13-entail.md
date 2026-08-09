# DC13-entail — the executor honours what the document declared

> **Status:** PLANNED (2026-08-09) · target release **v0.24.0**
> **Theme:** two things the executor throws away — the *declarations it inherits* and the
> *escape hatches it was handed at parse* — plus the two refusals that were never reachable.
>
> *entail*, v. — (law) to settle property on a line of heirs; (general) to have as a necessary
> consequence. Both halves of this cycle.

---

## Why this cycle exists

Releases 0.19.0 → 0.23.0 built the typed-input format: `subtype`, the eight input kinds, author
order, `{~exons.input~}` and the reserved `input` context root, then `DryRun` completeness and its
analysis-completeness channel. aigentverse consumed all of it across DC103 → DC110 and MP-LEXEME
closed.

The format is finished. **The executor is not.** v0.21.0's CHANGELOG carries a *"Known gaps
(documented, not fixed here)"* block with four entries, and every one of them is a place where the
executor discards something the parse already produced. Two more of the same class were found while
planning this cycle and had not been recorded anywhere.

The unifying defect: **a value is computed, and then dropped on the floor between the phase that
computed it and the phase that needs it.** Attributes are parsed and discarded. A parent Spec is
loaded and never consulted. A refusal is written and never invoked. An error is stored on a struct
field that exactly one caller reads.

---

## Scope — six defects, two workstreams

| # | Defect | Recorded? | Workstream |
|---|---|---|---|
| 1 | `extends` does not merge frontmatter — a parent's `inputs:` are invisible; the parent's body resolves against the **child's** declarations | v0.21.0 Known gaps | **A — lineage** |
| 2 | **NEW** — the inheritance resolver lexes the parent's **raw source including frontmatter**, so a parent with frontmatter splices `---\n…\n---` into the child's output as literal text | ✗ unrecorded | **A — lineage** |
| 3 | `Template.Explain` bypasses inheritance resolution — a document **explains** differently than it **renders** | v0.21.0 Known gaps | **A — lineage** |
| 4 | **NEW** — two inheritance failures that `dryRunAST` reports are silently swallowed by `ExecuteWithContext`: a discarded `inheritanceErr`, and an engine-less `extends` that is a no-op returning `nil` | ✗ half-recorded | **A — lineage** |
| 5 | `exons.if` / `exons.for` / `exons.switch` fail **unconditionally** — their parse discards the tag attributes, so `onerror=` / `default=` are structurally unreachable on a block tag | v0.21.0 Known gaps | **B — recourse** |
| 6 | A resolver's `Validate` is **never invoked by the executor**, so a refusal expressed only there is dead code | v0.21.0 Known gaps | **B — recourse** |

Defects 2 and 4 are not scope creep. Both sit **on the code path the other fixes must touch**, and
each one would make its neighbour's fix look correct while the output stayed wrong. Fixing the
frontmatter *merge* without fixing the frontmatter *splice* (2) produces a document whose inputs
finally bind and whose body now carries a stray YAML block.

---

# Workstream A — lineage

## A.1 The three facts that determine the design

**(a) A `*Template` owns exactly one `*Spec` — its own.** `Engine.Parse`
(`exons.engine.go:119`) splits frontmatter at `:127`, parses it to a `*Spec` at `:143-163`, and
hands both to `newTemplateWithConfig` at `:190`. `Spec` has **no `extends:` key** — inheritance is
expressed only by the body tag — so nothing in the frontmatter layer knows a parent exists.

**(b) Input injection runs BEFORE the parent is even identified.**
`ExecuteWithContext` (`exons.template.go:62`):

```go
execCtx = t.contextWithInputs(execCtx)              // :64  ← builds `input` root from t.spec.Inputs
…
resolvedAST, err := resolver.ResolveInheritance(…)  // :82  ← only NOW is the parent discovered
```

`contextWithInputs` (`exons.inputs.go:52`) reads `t.spec.Inputs` and nothing else (`:83`).
`ResolveInheritance` returns a `*RootNode` and nothing else. **Reordering alone does not fix it** —
`t.spec` simply does not contain the parent's declarations.

The failure surface is a hard error, not a silent blank: `exons.input`
(`internal/exons.executor.builtins.input.go:112-114`) does `accessor.Get("input." + name)` and on a
miss returns `newInputNotDeclaredError`. So **a parent body that uses its own declared input fails
to render at all** once it is spliced into a child.

**(c) The parent's parsed `*Spec` is already in memory, and the internal resolver cannot see it.**
`Engine.RegisterTemplate` (`exons.engine.go:334`) stores the full `*Template`; `Engine.GetTemplate`
(`:383`) returns it. But the internal resolver's only hook is
`TemplateSourceResolver.GetTemplateSource(name) (string, bool)`
(`internal/exons.executor.inheritance.go:17`), mirrored by the **exported**
`TemplateExecutor` interface (`exons.context.go:20`).

> **Design decision A-1 — the merge happens in package `exons`, not `internal`.**
> Widening `TemplateExecutor` with a `GetTemplateSpec` method is a **breaking change for every
> external implementer** of an exported interface, on a library whose whole v0.23.0 story was
> earning a clean release line. Walking the extends chain in `ExecuteWithContext` via
> `Engine.GetTemplate` needs **no interface change and no re-parse**. Take that.

## A.2 Defect 2 — the parent's frontmatter is spliced as literal text

`Engine.GetTemplateSource` (`exons.engine.go:394-403`) returns `tmpl.Source()` — the **original**
source. `Template.TemplateBody()` (`exons.template.go:108`) is the stripped one and is not used
here. `parseTemplateWithInheritance` (`internal/exons.executor.inheritance.go:94`) lexes that string
directly, and **the lexer has no frontmatter awareness** — stripping happens only in
`internal.ExtractConfigBlock`, called from exactly one place, `Engine.Parse`.

So today: a parent template carrying YAML frontmatter emits its own `---\n…\n---` into the child's
rendered output as text nodes.

**Why this has never been seen:** every inheritance test in the repo uses a bare body. There is no
`testdata/` directory and no `.exons` fixture anywhere — all templates are inline Go string
literals, and none of the ~100 inheritance tests gives a parent frontmatter.

> **This is the cycle's cautionary lesson and it goes in the CHANGELOG:** a subsystem with ~100
> tests had **zero** covering the interaction between its two headline features. Feature coverage
> is not interaction coverage.

**Fix:** the resolver must lex the parent's **body**. Preferred shape — hand the resolver the
already-stripped body rather than teaching it to strip, since `ExtractConfigBlock` is the single
statement of what frontmatter *is* and a second one may disagree with it.

## A.3 Defect 1 — merge the declaration chain

**Rule (stated once, mirroring aiv ADR-141 so the two composition stories agree):**

> **The COMPOSING document is the authority; the COMPOSED document supplies the fallback.**
> A child's `inputs:` declaration overrides a parent's of the same name. The union is walked
> **root-parent-first**, child last, so the child wins — the same first-wins-after-reversal
> ordering aiv's `collectTileInputs` uses.

This must be applied consistently at **four** reader sites or they will contradict each other:

| Site | File:line | Today reads |
|---|---|---|
| Context injection | `exons.inputs.go:83` | `t.spec.Inputs` |
| `declaresInput` | `exons.inputs.go:140` | `t.spec.Inputs` |
| `DryRun` → `InputReference.Declared` | `exons.debug.go:822` | via `declaresInput` |
| `Explain` placeholder resolution | `exons.debug.go:1050` | `getPath(data, "input."+name)` |

Fixing only the first produces an executor that binds an input while `DryRun` reports it
`Declared: false` — one rule, two implementations, which is the failure this whole track exists to
remove. **One helper computes the merged inputs map; all four call it.**

Depth and cycle limits must be the resolver's existing ones (`maxDepth`, the circular-chain guard at
`internal/exons.executor.inheritance.go:51-62`) — a second, differently-bounded walk of the same
chain is a second chance to disagree about what a cycle is.

## A.4 Defects 3 + 4 — one resolution helper, three callers

There are three callers of inheritance resolution and they currently disagree in three ways:

| Failure | `dryRunAST` (`exons.debug.go:629-670`) | `ExecuteWithContext` (`exons.template.go:62`) | `Explain` (`exons.debug.go:1138`) |
|---|---|---|---|
| `ExtractInheritanceInfo` failed | reports `dryRunErrInheritanceInfo` `:635` | **discarded** — `inheritanceErr` set at `:33-36`, read by nobody | n/a |
| `extends` but no engine | reports `dryRunErrNoEngine` `:648` | **silent no-op** — the `&& t.engine != nil` conjunct at `:73` | n/a |
| `ResolveInheritance` errored | reported `:661` | correctly returned `:83` | n/a |
| Resolution at all | yes | yes | **no — uses `t.ast` at `:1142`, `:1166`, `:1176`** |

**Fix:** extract the resolution into one reporter-agnostic helper returning
`(*internal.RootNode, error)`, and route all three callers through it. `Explain` surfaces the error
into `ExplainResult.Error` (`exons.debug.go:484`).

> ⚠ **`reportIncomplete` (`exons.debug.go:315`) is the ONLY writer of `DryRunResult.Errors`, and
> `TestDryRunErrorsHaveExactlyOneWriter` parses this package's own AST to keep it that way.** Any
> new error path in `exons.debug.go` must go through `reportIncomplete` /
> `reportIncompleteAndInvalid` or that test fails — correctly.

> **Design decision A-2 — an engine-less `extends` becomes an error at execute.**
> Today it renders the child's bare block bodies and returns `nil`. That is not a degraded render;
> it is a *different document* reported as success. `dryRunAST:648` already concedes the damage is
> "identical to an outright resolution failure". Promote it. This is a **behaviour change** and
> gets its own CHANGELOG paragraph, not a bullet.

---

# Workstream B — recourse

## B.1 Defect 5 — the escape hatch that never arrives

`parseTag` (`internal/exons.parser.go:114`) parses attributes once at `:128` and passes the full map
into `parseBlockTag` (`:156`), which dispatches to `parseConditional(attrs, pos)` `:163`,
`parseFor(attrs, pos)` `:173`, `parseSwitch(attrs, pos)` `:178`. **Each reads only its own keys and
lets `attrs` fall out of scope** — `parseConditional:233` (`eval`), `parseFor:392-408`
(`item`/`in`/`index`/`limit`), `parseSwitch:479` (`eval`). `onerror=` and `default=` were parsed,
carried into the function, and dropped.

The node types confirm it — `ConditionalNode` (`internal/exons.parser.ast.go:226`), `ForNode`
(`:283`), `SwitchNode` (`:330`), plus `ConditionalBranch` (`:232`) and `SwitchCase` (`:337`) have
**no `Attributes` and no `RawSource` field**. `TagNode` (`:79`) has both.

**The fatal sites — six, not the four previously believed** (all
`internal/exons.executor.go`, all returning `"", err` without touching the funnel):

| Line | Condition |
|---|---|
| `:165` | `exons.if` — branch condition failed to evaluate |
| `:194` | `exons.for` — collection path not found |
| `:200` | `exons.for` — value not iterable |
| `:261` | `exons.switch` — dispatch expression failed |
| `:282` | **missed previously** — `exons.case` `eval=` failed |
| `:218`, `:238` | **missed previously** — context does not implement `ChildContextCreator` |

`handleTagError` is called from exactly two places in the repo — `:418` and `:424`, both inside
`executeTag` — and it *could not* be called from the block executors: its first parameter is
`*TagNode`.

**Fix, in three steps:**

1. Add `Attributes` to `ConditionalNode`, `ForNode`, `SwitchNode`, and to `ConditionalBranch` /
   `SwitchCase` (so `onerror=` on an individual `elseif` or `case` is honoured — today `nextAttrs`
   is read for `eval` and discarded, `parser.go:262`/`:288`).
2. Generalise `handleTagError` off `*TagNode` onto `(name string, attrs Attributes, rawSource
   string, …)` or a minimal interface, so one funnel serves both shapes. **Do not fork a second
   funnel** — two implementations of the error strategy is the defect class this cycle is closing.
3. Route all six sites through it.

> ⚠ **Threading discipline in `parseConditional` (`:245-296`).** v0.23.0 *just* fixed this function:
> `parseConditionalBranch` returns the position of the tag that **terminated** the branch, so the
> fix captures `branchPos := nextPos` **before** the recursive call (`:257`, `:275`). `nextAttrs` is
> overwritten by that same call at `:267`/`:284` — **the branch's own attributes must be captured
> alongside `branchPos`, not read after.** Getting this wrong reproduces the exact bug v0.23.0 just
> fixed, one field over.

> **Design decision B-1 — `onerror="keepraw"` requires `RawSource`, and block tags have none.**
> `RawSource` is captured only on `TagNode` paths (`parser.go:142`, `:224`, `:673`). `parseSwitch`
> has the close-tag offset available at `:505-509`; `parseFor` and `parseConditional` swallow the
> close token inside their body helpers and need plumbing. **Capture it properly for all three** —
> the alternative is `keepraw` silently degrading to `""` via `handleTagError`'s existing fallback,
> which is a strategy that lies about what it did.

## B.2 Defect 6 — the refusal that is dead code

`Resolver.Validate` (`exons.resolver.go:9`) carries the doc comment *"Called during parsing to catch
errors early."* **That is false.** The only two non-test call sites are the pass-through adapter
(`exons.engine.go:506`) and `validateTagNode` (`exons.validation.go:201`), reached only from the
opt-in `Engine.Validate(source)` lint API. `grep Validate internal/exons.executor*.go` → zero hits.

**Planning turned up which resolver this actually costs, and it is exactly one.** Four of the five
built-ins with a meaningful `Validate` already duplicate the check into `Resolve` — `var`
(`builtins.go`, missing-`name` → error), `include` (missing-`template` → error), `env`
(missing-`name` → error), `ref` (missing/invalid `slug` → error). `raw`, `now` and both catalog
resolvers return `nil` from `Validate` (`now` deliberately — see its comment at
`builtins.now.go:38-42`).

**`MessageResolver` is the exception, and it is a live correctness bug:**

```go
// Validate — internal/exons.executor.builtins.message.go
role, hasRole := attrs.Get(AttrRole)
if !hasRole || role == "" { return … ErrMsgMessageMissingRole }
if !isValidRole(role)     { return … ErrMsgMessageInvalidRole }   // system|user|assistant|tool

// Resolve — same file
role, _ := attrs.Get(AttrRole)          // ← error ignored, isValidRole never called
…
return MessageStartMarker + role + MessageFieldSep + cacheFlag + MessageFieldSep, nil
```

So `{~exons.message~}` with **no role**, or with `role="bogus"`, renders a **malformed message
marker** — which `ExecuteAndExtractMessages` then parses into a `MessageInfo` with an empty or
invalid role, which a runtime hands to an LLM chat API. `Engine.Validate` rejects both. **The lint
is stricter than the renderer** — the same polarity as the aiv ADR-138 defect, in the library rather
than the registry.

> **Design decision B-2 — invoke `Validate` in `executeTag`, routed through `handleTagError`.**
> Inserted between `:418` and `:421`. Two consequences, both stated deliberately:
> - **Governable.** Because it goes through the funnel, a refusal obeys `onerror=` and the context
>   strategy, rather than being an unconditional hard stop. This is what
>   `NowResolver.Validate`'s comment was worried about, and the funnel answers it.
> - **Behaviour change for third-party resolvers** whose `Validate` is stricter than their
>   `Resolve`. Stated in the CHANGELOG as a contract change, not a bullet. The alternative — delete
>   the claim and document `Validate` as lint-only — leaves `exons.message` emitting malformed
>   markers, which is the worse of the two.
>
> Cost is one interface call plus O(1) map lookups per tag per render, including per for-loop
> iteration. All built-in `Validate` bodies are `Has`/`Get` checks. Correct the false doc comment on
> `Resolver.Validate` either way. Check `ResolverFunc`'s optional `validate` field
> (`exons.resolver.go:57`) for nil-handling before making this load-bearing.

---

# Workstream C — the teaching surface

`examples/` stops at `08-syntax-safety`. **There is no runnable example of `subtype:`, `inputs:`, or
`{~exons.input~}`** — the format's headline feature across three releases. The exons.ai Inputs page
carries inline snippets instead, which are not compiled and not tested.

**Deliverable: `examples/09-typed-inputs/`** — its own `go.mod` like every sibling, a
`.exons` document declaring several input kinds, and a `main.go` that binds values and renders.

It must exercise, as its whole point, the things this cycle fixes:

- an **optional list input** iterated with `{~exons.for in="input.sources"~}` — present-as-nil, so it
  iterates zero times when unbound (the case MP-LEXEME OQ3 discovered was already benign);
- an **`onerror=`/`default=` on a block tag** — impossible before Workstream B;
- **a parent with frontmatter-declared inputs** extended by a child — impossible before Workstream A,
  and the interaction test the ~100 existing inheritance tests never had.

---

## Test plan

The repo has no `testdata/` and no fixture files; every template is an inline Go string literal
with `testify` `require`/`assert`. Follow that. Naming: `exons.<area>.<subarea>_test.go` mirroring
the source file.

**Every regression test below must be verified RED before the fix** — the DC109-tribune lesson: a
regression test that does not fail without the fix is not a guard.

| Test | Pins |
|---|---|
| `exons.inheritance.frontmatter_test.go` — parent declares `inputs:`, child extends, parent body uses `{~exons.input~}` | Defect 1. Today: `newInputNotDeclaredError` |
| same file — parent has frontmatter, assert the rendered output contains **no** `---` | Defect 2 |
| same file — child overrides a parent input of the same name | the composing-document-is-authority rule |
| `DryRun` on the same child reports the parent's input `Declared: true` | the four-reader-sites rule — this is the one that catches a partial fix |
| `Explain` on an `extends` child — output matches `ExecuteWithContext` | Defect 3 |
| a template with a malformed `extends` — `ExecuteWithContext` errors instead of rendering the bare child | Defect 4a |
| an engine-less `extends` — errors instead of returning `nil` | Defect 4b (behaviour change) |
| `onerror="default" default="…"` on each of `exons.if`, `exons.for`, `exons.switch` × each of the six fatal sites | Defect 5 |
| `onerror=` on an individual `elseif` and on an individual `case` | the branch-attribute threading |
| `onerror="keepraw"` on each block tag emits the full original source | B-1 |
| `{~exons.message~}` with no role, and with `role="bogus"` | Defect 6 — assert the error, and assert `onerror="remove"` suppresses it |

**Gate:** `make ci-local` — `build vet fmt-check tidy-check lint` + `go test -race -coverprofile`
with a **coverage floor of 88%**. Note `make check` is *mutating* (`fmt` rewrites files); `ci-local`
is the gate.

---

## Sequencing

1. **B first.** Self-contained, no API-shape questions, and its funnel is what A-2's error promotion
   wants to be governable by. Parser/AST change + executor rewiring + `Validate` invocation.
2. **A second.** Defect 2 (the splice) before Defect 1 (the merge) — a merge validated against an
   output still carrying stray YAML validates nothing. Then the shared resolution helper, then the
   three callers.
3. **C last.** The example is the acceptance test for both: it cannot be written honestly until
   both land.
4. `/review` at HIGH effort, fix-all until re-review is clean, then `/dev-cycle-end`.

## Release

**v0.24.0** — minor, not patch. Two deliberate behaviour changes (A-2 engine-less extends, B-2
executor-invoked `Validate`) plus new AST fields. No exported-API break: the AST node types live in
`package internal` and are unreachable outside the module; the root package's own `Attributes` is a
*separate interface* (`exons.resolver.go:35`) bridged by `internalAttributesAdapter` — do not
conflate them.

**Downstream:** aigentverse re-pins. Defects 1–4 reach aiv users directly (aiv ships
`exons.extends` in its known-verb vocabulary, so an author can reach the inheritance path today).
Defect 6 reaches every aiv message-extraction surface. Update `docs/masterplan.md`'s cycle log and
`CHANGELOG.md`; retire the four resolved entries from the v0.21.0 *Known gaps* block by superseding
them in the new release's notes.

## Explicitly NOT in this cycle

- **Restructuring `go-exons` into importable sub-packages.** Asked and answered 2026-08-08 in
  aiv's `mp-lexeme.md`: exporting the AST freezes every node field under semver on a pre-1.0 module
  still gaining verbs, and consumers want an *answer*, not the nodes.
- **Reviving aiv lint rule (i)** (declared-but-unreferenced). Cut permanently on evidence in
  DC107-lexicon / ADR-143. Nothing here changes that — `{~exons.ref~}` remains invisible to `DryRun`
  and reviving the rule needs closure-aware evidence at an authoring surface, not a better analyzer.
- **`extends` as a frontmatter key.** Inheritance stays body-expressed. Adding `Spec.Extends` would
  create a second, silently-disagreeing statement of who the parent is.
