# go-exons Masterplan

Living document tracking dev-cycles toward the library's goals. Dev-cycles
are continuously numbered; earlier cycles (DC1–DC10) predate this document
and are recorded in [CHANGELOG.md](../CHANGELOG.md) as `0.x.0-dcN` releases.
The Skope-migration side-cycle DC-SE0 landed in 0.12–0.14.

## Goal

exons is the agent specification format for Go: a content-resistant template
language (`{~...~}`) plus a portable Spec document format for agents, skills,
and prompts — safe to author by hand and by LLMs.

## Cycle log

| Cycle | Version | Theme | Status |
|---|---|---|---|
| DC1–DC10 | 0.1–0.11 | Core engine, catalogs, compilation, A2A, hardening | shipped |
| DC-SE0 | 0.12–0.14 | Skope migration seam: requirements block, @org/name refs | shipped |
| DC11-verbatim | 0.15.0 | Syntax safety: nesting & examples-of-the-syntax | in progress |

## DC11-verbatim — Syntax safety (0.15.0)

Problem: templates could not safely contain examples of their own syntax —
raw blocks required lexically valid content, could not contain their own
close tag, round-tripped lossily, and markdown fences offered no protection
in SKILL.md-style bodies.

Design (from a comparative study of Handlebars `{{{{raw}}}}`, CommonMark
fence escalation, MDX inert fences, Jekyll's fence trap, Rust raw strings):

1. **Verbatim tilde fences** `{~~ ... ~~}` with markdown-style length
   escalation — lexer-level, byte-exact, backward-compatible (`{~~` was a
   hard lexer error before).
2. **Lexer-level named raw/comment** — verbatim scan to the first canonical
   close; fixes must-lex-cleanly and lossy reconstruction defects.
3. **Markdown fence inertness** as opt-in `WithMarkdownFences()` (never a
   silent global default), with ```` ```exons ```` live-fence opt-back-in,
   `Spec.ContentFormat` hint from `ImportFromSkillMD`, and `Validate()`
   lints.
4. Editor grammar rules for both mechanisms; normative
   [template-syntax.md](template-syntax.md) spec.

Also fixed in-cycle: inheritance re-lex ignored engine delimiter config;
stray top-level block-close was silently swallowed; raw reconstruction
hardcoded default delimiters; golangci-lint config migrated to v2.

Deliberately out of scope: inline code spans staying live (interpolation
inside `` `...` `` is common in prompts); fence semantics in the core lexer
by default.
