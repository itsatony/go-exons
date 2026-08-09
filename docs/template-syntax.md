# exons Template Syntax — Lexical Specification

Normative reference for the exons template lexical grammar (v0.24.0).
For tag semantics (var, if, for, ...) see the [README syntax reference](../README.md#template-syntax-reference).
The last two sections cover the two built-in tags whose attribute vocabularies
are closed (`exons.now`, `exons.input`) and the frontmatter keys they read.

## Delimiters

| Construct | Default form | Derivation under custom delimiters |
|---|---|---|
| Tag open | `{~` | `OpenDelim` (`WithDelimiters`) |
| Tag close | `~}` | `CloseDelim` |
| Self-close | `/~}` | `"/" + CloseDelim` |
| Block close | `{~/` | `OpenDelim + "/"` |
| Escape | `\{~` | `"\" + OpenDelim` |

Tag names match `[A-Za-z_][A-Za-z0-9_.-]*`. Attribute values are quoted
(single or double); inside a quoted value, delimiter sequences (`~}`, `{~`,
`{~~`) are ordinary characters, and `\"`/`\\` un-escape.

## Escape: `\{~`

A backslash escapes exactly the two-byte open delimiter and emits a literal
`{~`. Because every construct (tag open, block close, verbatim fence) begins
with `{~`, the escape uniformly neutralizes all of them; the rest of the
input lexes normally. `\{~~x~~}` therefore renders as literal `{~~x~~}`.

There is no escape for `~}` — a bare `~}` in text is already literal.

## Verbatim tilde fences: `{~~ ... ~~}`

The general mechanism for writing exons syntax *as content* (docs, examples,
templates that generate templates).

- **Open**: `{` followed by a maximal run of k tildes, k ≥ 2.
- **Close**: the first subsequent *maximal run of exactly k tildes*
  immediately followed by `}`. Maximal means the run is not part of a longer
  run — `~~~}` does **not** close a `{~~` fence.
- **Body**: every byte between open and close, emitted verbatim as text. No
  escape, tag, or nested-fence processing applies inside. The body may
  contain lexically invalid fragments (a lone `{~`, `{~ 5 ~}`), full raw
  blocks, anything.
- **Escalation rule**: if the body must contain `~~}`, use `{~~~ ... ~~~}`;
  one more tilde per side per level, exactly like markdown fence lengths.
- **Empty bodies are inexpressible**: in `{~~}` and `{~~~~}` the tilde runs
  merge into the opener, leaving no closing run — both are unterminated-fence
  errors. The minimal body is one non-tilde byte (`{~~ ~~}` emits a space).
  Use `\{~` for degenerate cases.
- **Unterminated fence**: hard lexer error reporting the tilde count and the
  open position. A stray `{~~` can never silently swallow the document.
- **Custom delimiters**: the fence family extends the *default* delimiter
  alphabet only. Under `WithDelimiters`, `{~~` is plain text.
- Not line-anchored: fences work inline, in JSON bodies, anywhere.

## Named verbatim blocks: `exons.raw` and `exons.comment`

`{~exons.raw~}...{~/exons.raw~}` (body emitted verbatim) and
`{~exons.comment~}...{~/exons.comment~}` (body discarded).

- The body is scanned **byte-for-byte at the lexer level** until the first
  canonical close sequence (`{~/exons.raw~}` / `{~/exons.comment~}`, built
  from the configured delimiters). Content round-trips exactly: escapes stay
  escaped, whitespace and quote styles are preserved, and lexically invalid
  fragments are allowed.
- **First close wins.** A nested opener is literal body content; the block
  cannot contain its own close sequence. To show a complete raw block as
  content, wrap it in a tilde fence: `{~~{~exons.raw~}x{~/exons.raw~}~~}`.
- Only the canonical close form ends the block — `{~/ exons.raw ~}` (interior
  whitespace) is body content.
- Attributes on the opener are lexed and ignored.
- Self-closing `{~exons.raw /~}` has no body and renders empty.
- An unterminated block is a hard lexer error naming the missing close tag
  and the open position.

## Markdown fence mode: `WithMarkdownFences()`

Opt-in engine option for markdown-format templates (SKILL.md-style bodies).
`ImportFromSkillMD` sets `Spec.ContentFormat = "markdown"` as the signal to
enable it.

When on, fenced code blocks per a CommonMark subset are **inert**: exons
tags, escapes, and tilde fences inside pass through as literal text.

- **Opening fence line**: up to 3 spaces of indent, then a run of ≥ 3
  backticks or ≥ 3 tildes, then an optional info string. A backtick fence
  whose info string contains a backtick is not a fence (CommonMark rule).
- **Closing fence line**: same character, run at least as long as the opener,
  nothing but whitespace after. A shorter run, the other character, or
  trailing text makes the line fence content.
- **Live fences**: a fence renders normally when the first
  whitespace-separated word of its info string is `exons`
  (```` ```exons ````). Use this to emit a code block with interpolated
  values.
- **Unclosed fence**: inert to end of input (matching real markdown
  renderers), not an error. `Validate()` emits a warning.
- Indented (4-space) code blocks and inline code spans (`` `x` ``) are *not*
  inert — inline interpolation like `` `{~exons.var name="cmd" /~}` `` stays
  live.
- CRLF sources are handled; a trailing `\r` counts as line-end whitespace.

### Validation lints

With the option enabled, `Engine.Validate` warns (never errors) when:

- an inert fence body contains the open delimiter — likely an example that
  was meant to render (`add 'exons' info string to render`), and
- a fence is unclosed (everything after it is inert).

## Precedence

At any position, the lexer resolves constructs in this order:

1. **Markdown-inert region** (fence mode only) — consumed wholesale as text.
2. **Escape** `\{~`.
3. **Verbatim tilde fence** `{~~...` (k ≥ 2 tildes).
4. **Block close** `{~/`.
5. **Tag open** `{~` (raw/comment openers then switch to verbatim scanning).
6. Plain text.

A construct that opens first owns its bytes until its own close, without
regard to later boundaries: a raw block opened before a markdown fence
consumes through it (so a `{~/exons.raw~}` inside that fence still closes the
raw block), and fence/raw content never opens markdown regions.

## Error summary

| Input | Error |
|---|---|
| `{~` never closed | `unterminated tag` |
| `{~~` (k ≥ 2) without matching close | `unterminated verbatim fence: expected closing run of exactly k tildes followed by '}'` |
| `{~exons.raw~}` / `{~exons.comment~}` without canonical close | `unterminated verbatim block: missing closing tag "..."` |
| Stray top-level `{~/x~}` | `unexpected token` |
| Unclosed markdown fence (fence mode) | no error; `Validate()` warning |

## Error recourse: `onerror=` and `default=`

Recourse is an **execution-time** mechanism: when a construct fails *while
rendering*, `onerror=` selects what the renderer emits in its place instead of
aborting the render. The lexer and parser errors above are outside it — a
construct that does not parse never reaches execution.

| `onerror=` | Emits | Notes |
|---|---|---|
| `throw` | nothing — the error propagates and the render fails | The default, and the fallback for any unrecognized value. |
| `default` | the construct's `default=` attribute | Empty string when no `default=` is present, i.e. `remove` with an escape hatch. |
| `remove` | empty string | |
| `keepraw` | the construct's original source, verbatim | Empty string when no source was captured — it fails closed rather than emit a wrong slice of the document. |
| `log` | empty string | Plus one `WARN` line naming the tag and the error. |

The vocabulary is closed. An unrecognized value parses to `throw`, and
`Engine.Validate` reports it as an **error** on every shape that honours the
attribute: a tag, `exons.if` / `exons.for` / `exons.switch`, and an individual
`exons.elseif` / `exons.case` (reported at the branch's own position, not the
opening tag's).

That coverage matters more than it looks. A typo does **not** degrade to the
context's configured leniency — resolution stops as soon as the key is present,
so `onerror="remov"` hard-fails under a renderer configured never to hard-fail,
and the misspelling is the only evidence. Before v0.24.0 the lint checked tags
alone, which was correct while a block construct's `onerror=` was inert; it
became a gap the moment the attribute started being honoured.

`default=` is read **only** by `onerror="default"`. On `exons.var` and
`exons.input` the same attribute name carries a second, resolver-level meaning
(a lookup miss / an empty declared value); that path returns a value instead of
an error, so the two readings never fire for one failure.

### Resolution order

1. The failing construct's own `onerror=` — for a branch, the branch's first
   (see below).
2. The **context default**: the strategy the execution context reports. A
   context supplies one by implementing `ErrorStrategy() int`; the built-in
   `Context` does, seeded at `Execute` from the engine's `WithErrorStrategy`
   (itself defaulting to `throw`). A loop body's child context inherits it.
3. `throw`.

A declared-but-unrecognized `onerror=` stops at step 1 and resolves to `throw`;
it does **not** fall through to the context default.

### Governed failures

| Construct | Failure |
|---|---|
| any tag | no resolver registered for the tag name |
| any tag | the resolver's `Validate` refuses the attributes (v0.24.0 — see below) |
| any tag | the resolver's `Resolve` returns an error |
| `exons.if` / `exons.elseif` | the branch condition fails to evaluate (v0.24.0) |
| `exons.for` | the `in=` path is not found in the context (v0.24.0) |
| `exons.for` | the value found is not iterable (v0.24.0) |
| `exons.for` | the host context cannot create a child context — it does not implement child creation, or the child it returns is not a context accessor (v0.24.0) |
| `exons.switch` | the dispatch `eval=` expression fails to evaluate (v0.24.0) |
| `exons.case` | the case's `eval=` expression fails to evaluate (v0.24.0) |

Everything else remains a hard failure: parse-time refusals (a missing `eval=`,
`item=` or `in=`, a non-numeric or negative `limit=`, an unclosed or mismatched
construct) precede execution, and an error raised *inside* a selected branch or
loop body belongs to the failing node's own site — the enclosing construct's
`onerror=` does not catch it.

### Block constructs (v0.24.0)

Before v0.24.0 the parse of `exons.if` / `exons.for` / `exons.switch` read the
keys it needed and let the attribute map fall out of scope, so `onerror=` and
`default=` on these three were lexed, parsed, and then structurally
unreachable: every failure marked v0.24.0 above was an unconditional render
abort no matter what the author wrote. The three constructs now retain their
opening tag's full attribute map and route those failures through the same
mechanism as a tag.

`exons.else` and `exons.casedefault` evaluate nothing and therefore cannot fail
this way; `onerror=` on them is inert.

### Branch-level `onerror=` (v0.24.0)

An individual `exons.elseif` and an individual `exons.case` may carry their own
`onerror=` / `default=`. For a failing branch the attribute maps are consulted
in order — **the branch's own first, the enclosing construct's second** — so a
branch-level declaration overrides the construct-level one, and a construct-level
declaration still governs every branch that declares nothing.

The two keys resolve **independently**: a branch declaring only `onerror=` still
picks up the enclosing construct's `default=`.

```
{~exons.if eval="a" onerror="default" default="(from if)"~}A
{~exons.elseif eval="b" onerror="default" default="(from elseif)"~}B
{~exons.else~}C{~/exons.if~}
```

### `keepraw` on a block construct

`keepraw` emits the **entire enclosing construct**, from its opening tag through
its closing tag — body, branches, cases and all — not just the failing branch.
`executeConditional` / `executeSwitch` / `executeFor` each return one string for
the whole construct, so a per-branch slice would have nothing to splice into.

```
{~exons.if eval="nope(" onerror="keepraw"~}body{~/exons.if~}
```
renders as that exact source text, `{~/exons.if~}` included.

The construct's source is captured through the token that closed it. Where that
capture is not possible the strategy degrades to the empty string rather than
emit a partial construct.

### Resolver `Validate` refusals (v0.24.0)

A resolver's `Validate` is now called by the executor immediately before
`Resolve`, and a refusal is routed through this mechanism rather than being an
unconditional stop — so `onerror=` governs it exactly as it governs a `Resolve`
error. Until v0.24.0 the executor never called `Validate`, which made a refusal
expressed only there dead code on every render and made the opt-in
`Engine.Validate` lint *stricter* than the renderer. `Validate` now runs once
per tag per render, including once per iteration inside an `exons.for` body;
implementations must stay cheap.

A refused block tag emits its recourse value and its children are **not**
rendered.

## Built-in output tag: `{~exons.now~}` (v0.18.0)

Prints a formatted reference time into body text. Distinct from the date/time
*expression* functions (`now`, `formatDate`, `year`, ...), which are usable only
in `eval=` and have no output path.

The reference time is seeded once per render by the caller under the reserved
data key `_refTime` (`ContextKeyReferenceTime`), so every `{~exons.now~}` in one
render agrees and a test can pin an exact instant. Unseeded, it falls back to
`time.Now()`.

| Attribute | Meaning |
|---|---|
| `format` | A **name** from the closed vocabulary below — never a Go layout string. Absent or empty means `iso`. |
| `layout` | A raw Go reference layout (`"Mon Jan 2"`). **Wins over `format`** when non-empty: the escape hatch for a format the named set does not cover. |
| `tz` | An IANA zone name (`Europe/Berlin`), resolved with `time.LoadLocation`. |

Timezone is applied first, then the format. **The default zone is UTC** — absent
`tz`, the instant is converted to UTC rather than left in the seeded zone.

| `format` | Rendering | Example |
|---|---|---|
| `iso` (default) | RFC 3339 | `2026-07-21T14:30:00Z` |
| `date` | `2006-01-02` | `2026-07-21` |
| `datetime` | `2006-01-02 15:04:05` | `2026-07-21 14:30:00` |
| `time` | `15:04:05` | `14:30:00` |
| `year` | `2006` | `2026` |
| `month` | `01` (zero-padded) | `07` |
| `day` | `02` (zero-padded) | `21` |
| `weekday` | English weekday name | `Tuesday` |
| `unix` | seconds since the epoch | `1784644200` |
| `rfc1123` | RFC 1123 | `Tue, 21 Jul 2026 14:30:00 UTC` |
| `date-de` | `02.01.2006` | `21.07.2026` |

```
{~exons.now /~}
{~exons.now format="date" /~}
{~exons.now format="datetime" tz="Europe/Berlin" /~}
{~exons.now layout="Mon Jan 2" /~}
```

Every attribute is optional, so `Validate` accepts the tag unconditionally. An
unrecognized `format` name and an unloadable `tz` are reported from `Resolve`,
which routes them through the engine's configured error strategy — a parse-time
error would hard-fail a template a lenient strategy would otherwise render.

## Built-in output tag: `{~exons.input~}` (v0.21.0)

References an input the document **declared itself**, in its frontmatter
`inputs:` block.

**`{~exons.var~}` is not deprecated and is unchanged.** It remains the
context-variable verb and still reads any path the runtime supplies. Before
v0.21.0 both jobs — "read a value the runtime happened to pass" and "read a value
this document declared" — were spelled `exons.var`; the split gives two jobs two
names. Nothing an existing document does with `exons.var` needs to change; the
new verb is a way to say something the old one could not distinguish.

Declared inputs are injected into the render context under a **reserved `input`
root**, with each `InputDef.Default` applied wherever the caller bound nothing.
So `{~exons.input name="tone" /~}` *is* the path `input.tone`, and
`eval="input.verbose"` / `in="input.sources"` work on `exons.if` / `exons.for`
with **no grammar change**. A private lookup space would have made the verb
invisible to control flow.

A declared input with neither a bound value nor a default is **present and nil**,
not absent: it renders empty, evaluates falsy, and a loop over it runs zero
times. The payoff is the equivalence *present ⇔ declared*, which is what makes a
name that is absent under the root provably an author typo — reported with
did-you-mean suggestions drawn from the declared names.

| Attribute | Meaning |
|---|---|
| `name` | **Required.** The declared input's key. |
| `default` | Used when the input is present but nil or empty. (Contrast `exons.var`, which consults `default=` on a lookup *miss* — after injection a declared input never misses.) |
| `join` | Separator between the elements of a list value. |
| `required` | **Refused** — an error, not ignored. `required:` in the frontmatter is the single source of truth; render time is too late to ask a user for a value, and a `required=` inside an unreached `exons.if` branch would enforce or not depending on data. Use `Spec.ValidateInputBinding`, which runs before any render. |

```
{~exons.input name="tone" /~}
{~exons.input name="sources" join=" > " /~}
{~exons.if eval="input.verbose"~}...{~/exons.if~}
{~exons.for each="s" in="input.sources"~}...{~/exons.for~}
```

## Frontmatter keys read by these tags

| Key | Since | Meaning |
|---|---|---|
| `subtype` | v0.19.0 | Refines `type: prompt` only: `fragment` (a composable piece meant to be *referenced*) or `template` (carries `inputs:`, meant to be *executed* with per-run values). Empty means unspecified. |
| `recommended_agents` | v0.16.0 | A list of curatorial "made for @org/name" associations. Carried verbatim; never resolved by this library. |

`inputs:` maps a name to an `InputDef`. The kind vocabulary (v0.19.0, completing
`select`/`multiselect`) is **advisory and open** — `Spec.Validate()` rejects no
kind, because an enforced enum would break a consumer pinning a newer vocabulary
than the library:

| `type` | Bound value |
|---|---|
| `text` | free-form single- or multi-line string |
| `number` | numeric value |
| `boolean` | true/false toggle |
| `select` | exactly one of `options` |
| `multiselect` | zero or more of `options` |
| `file-upload` | one or more uploaded files |
| `sort` | `options`, reordered — the declared order is the initial ranking |
| `associate` | many-to-many pairs of (`options`, `associate_with`) |

Modifiers, alongside `required:`, `default:` and `label:` (v0.16.0, a form label;
presentation only, consumers fall back to the input key when it is empty):

| Modifier | Applies to | Meaning |
|---|---|---|
| `description` | any | Human-facing help text. |
| `options` | `select`, `multiselect`, `sort`, `associate` | Selectable values (`{value, label}`). Order is significant for `sort`. |
| `associate_with` | `associate` | The right-hand set. |
| `accept` | `file-upload` | Media types or extensions (`application/pdf`, `.csv`), verbatim in the spirit of the HTML `accept` attribute. Empty means the author declared no restriction — not that any file is safe. |
| `max_size_bytes` | `file-upload` | Caps an **individual** file. Zero means unspecified. |
| `max_files` | `file-upload` | Caps **how many** files. Zero means unspecified. |
