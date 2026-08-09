# go-exons Masterplan

Living document tracking dev-cycles toward the library's goals. Dev-cycles
are continuously numbered; earlier cycles (DC1–DC10) predate this document
and are recorded in [CHANGELOG.md](../CHANGELOG.md) as `0.x.0-dcN` releases.
The Skope-migration side-cycle DC-SE0 landed in 0.12–0.14. Releases 0.16.0
through 0.22.0 shipped as consumer-driven increments without dev-cycle slugs and
are listed by version; [CHANGELOG.md](../CHANGELOG.md) is their record of detail.

## Goal

exons is the agent specification format for Go: a content-resistant template
language (`{~...~}`) plus a portable Spec document format for agents, skills,
and prompts — safe to author by hand and by LLMs.

## Cycle log

| Cycle | Version | Theme | Status |
|---|---|---|---|
| DC1–DC10 | 0.1–0.11 | Core engine, catalogs, compilation, A2A, hardening | shipped |
| DC-SE0 | 0.12–0.14 | Skope migration seam: requirements block, @org/name refs | shipped |
| DC11-verbatim | 0.15.0 | Syntax safety: nesting & examples-of-the-syntax | shipped 2026-07-13 |
| — | 0.16.0 | `InputDef.label` / `options`, `Spec.recommended_agents` | shipped 2026-07-14 |
| — | 0.17.0 | A2A Agent Card v1.0.1 + RFC-7515 §8.4 detached-JWS signing | shipped 2026-07-19 |
| — | 0.18.0 | `{~exons.now~}` built-in date/time output tag | shipped 2026-07-28 |
| — | 0.19.0 | Prompt `subtype` + the complete input-kind vocabulary | shipped 2026-08-06 |
| — | 0.20.0 | Authored input order survives; list/object values render as prose | shipped 2026-08-07 |
| — | 0.21.0 | `{~exons.input~}`: `inputs:` stops being inert (reserved `input` root) | shipped 2026-08-08 |
| — | 0.21.1–0.21.2 | The binary-withholding sweep made complete (structs, byte arrays, map keys) | shipped 2026-08-08 |
| — | 0.22.0 | `DryRun` reference completeness + `ExpressionIdentifiers` | shipped 2026-08-08 |
| DC12-promulgate | 0.23.0 | DryRun's `Errors` becomes the analysis-completeness channel; `main` gets a clean release line | shipped 2026-08-09 |
| DC13-entail | 0.24.0 | The executor honours what the document declared: `extends` merges declarations, block constructs get error recourse, a resolver's `Validate` is invoked | shipped 2026-08-09 |

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
