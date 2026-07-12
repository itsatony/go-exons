# exons Template Syntax — Lexical Specification

Normative reference for the exons template lexical grammar (v0.15.0+).
For tag semantics (var, if, for, ...) see the [README syntax reference](../README.md#template-syntax-reference).

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
