# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.22.0] - 2026-08-08

`DryRun` becomes complete, and documents the contract it now satisfies.

`DryRun` is the only sound static-analysis surface a consumer has — the AST lives in `package
internal` and is unimportable. So a consumer deciding *"this declared input is referenced
nowhere"* can only be as right as `DryRun` is complete, and that verdict licenses telling an author
to delete working code. Under-reporting a reference is therefore not a milder bug here; it is a
different and worse one. That asymmetry is why this release ships a **stated** completeness
contract on `DryRunResult` rather than merely more coverage.

### Fixed

- **Block-tag children were never walked.** `processTagNodeForDryRun` reported a tag and did not
  descend into its children, so `{~exons.message role="user"~}{~exons.input name="tone"
  /~}{~/exons.message~}` produced **zero** input references. `exons.message` wraps essentially
  every real prompt body, making this the common case rather than an edge case. It was entirely
  untested — the full suite stayed green after the fix. The same omission is fixed in
  `generatePlaceholders`, where it made nested content vanish from the preview an author reads.
- **`elseif` conditions were discarded.** Branches after the first were reduced to two booleans, so
  a path referenced only in an `elseif` was reported as referenced nowhere.
- **Switches were invisible in their entirety.** Neither the dispatch expression nor any case's
  `eval=`/`value=` was recorded anywhere, and there was no `SwitchReference` type at all.
- **`BlockNode` had no walker arm**, so every reference inside every `{~exons.block~}` was unseen.
- **`DryRun` analysed the raw AST** while `ExecuteWithContext` runs the inheritance-resolved one —
  an `{~exons.extends~}` child was analysed as a document that never runs. Resolution failure is
  now reported through `Errors` and tolerated, degrading to a partial answer with the reason
  attached.

### Added

- `ExpressionIdentifiers(expression string) ([]string, error)` — returns every context path an
  `eval=` expression references, resolved by this library's own expression parser. A consumer
  scanning the raw condition string has to agree byte-for-byte with the lexer about how a reference
  is spelled, and gets it wrong for dotted paths inside call arguments, identifiers adjacent to
  operators, and paths in a boolean's right-hand branch. Identifiers are whole dotted paths,
  de-duplicated and sorted; function names are excluded.
- `DryRunResult.Switches` (`SwitchReference`, `SwitchCaseRef`).
- `ConditionalReference.Branches` (`ConditionalBranchRef`) and `.Identifiers`. The existing
  `Condition`, `HasElseIf` and `HasElse` fields keep their exact meaning — this release is additive.

### Documented

- The **reference-completeness contract** on `DryRunResult`: what is guaranteed reported, what is
  deliberately not, and the one case that is not closable by this library — a third-party resolver
  whose arbitrary attribute names a context path. Every omission fails in the *safe* direction
  (over-reports use, never under-reports it): inert `exons.raw`/`exons.comment` spans, loop-variable
  shadowing, and include recursion (structurally moot — the include boundary does not propagate the
  caller's reserved `input` root, so a parent's inputs are unreachable inside an included template).

## [0.21.2] - 2026-08-08

Found in re-review of v0.21.1. No API change.

### Fixed

- **v0.21.1's binary guarantee was falsely *complete*: map KEYS were not swept.** A slice cannot be
  a map key, but a byte **array** can — so `map[[16]byte]string` (a digest-keyed map) rendered its
  keys as `104, 101, …` while every value was correctly withheld. This is the v0.21.0 mistake at
  smaller scale, and the reason it was worth another release: a redactor that states an absolute
  and misses one path is worse than one that states a scope, because the claim is what the next
  reader trusts instead of re-deriving it.

### Documentation

No behaviour change; two comments overstated what they had fixed.

- `contextWithInputs` claimed to have restored a thread-safety property. What it restored is
  **agreement** with `Spec.BindInputs`: `deepCopyValue` copies the YAML/JSON shapes exhaustively and
  returns structs, pointers, `[]byte` and non-string-keyed maps **by reference**. Widening that
  belongs in `deepCopyValue`, not in two call sites that would then have to agree by memory.
- `exons.input` reaches its depth bound at a shallower *data* depth than `exons.var`, because the
  sweep flattens pointers while `renderValue` spends a level on each deref. A non-binary value
  behind several pointers can therefore elide under one verb and render whole under the other.
  Accepted and now stated: over-eliding is the safe direction, and not incrementing on pointers is
  what `type P *P; p = &p` turns into a hang.

## [0.21.1] - 2026-08-08

Fixes to v0.21.0's `exons.input`, found in review. No API change.

### Fixed

- **`exons.input`'s binary guarantee had a hole, and it was the shape a Go caller reaches for
  first.** `withholdBinary` recursed through slices, maps and pointers but had **no
  `reflect.Struct` arm** — while `renderValue` has an explicit one that walks exported fields,
  and renders any uint8-element slice as `string(rv.Bytes())`. So binding a declared input to
  `struct{ Name string; Body []byte }` pasted the **entire file body into the prompt**, through
  the one function written to prevent precisely that. The sweep now mirrors `renderValue`'s
  traversal kind for kind, and a new test pins that parity with a **positive** assertion on the
  withheld marker — its predecessor asserted only that the secret string was absent, which an
  untraversed byte array passes while leaking every byte.
  - Byte **arrays** (`[32]byte`) escaped as well — the uint8 check was gated on
    `Kind() == Slice` — and rendered as a list of small integers.
  - The depth bound **failed open**: the sweep flattens pointers, so the rebuilt value can be
    shallower than the one swept, and a byte slice handed back raw at the sweep's bound was
    reached by `renderValue` well inside its own. It now elides at the bound.
  - The old `reflect.Interface` arm was dead code: `reflect.ValueOf` takes an `any` and always
    reports the dynamic kind.
  - ⚠ A byte slice is withheld **even when it holds UTF-8 text**. The shape carries no evidence
    of which it is, and a text file read into a `[]byte` is as much an accidental paste as a
    PDF. A caller that means to inline a document's text binds a `string`.
- **The file manifest claimed lists that were not files.** The recognizer required only "every
  element is a map with a non-empty `name`", which describes a list of **named objects** — one
  of the most ordinary shapes a caller can bind. `[{"name":"GPT-4","provider":"openai"}, …]`
  rendered as a bullet list with `provider` **silently dropped**. It now requires every
  element's keys to be drawn from `{name, mime_type, size_bytes}` *and* at least one element to
  carry a file-specific key; a list of bare `{"name": …}` maps is too ambiguous to claim and
  falls through to `renderValue`, which preserves it whole.
- **`contextWithInputs` merged caller bindings shallowly.** `Context.Get` returns the *live*
  value, and the map injection replaces came from `Context.Data()`, which deep-copies — so the
  shallow merge silently weakened a thread-safety property the context documents, on the one
  path built to run concurrently. `Spec.BindInputs`, the sibling path, already deep-copied.
- **A nil `input` root skipped injection entirely.** `data["input"] = nil` means "nothing is
  bound", not "`input` is my own variable", but it took the not-a-map exit — so a document's
  declared **defaults silently did not apply**, the single outcome the feature exists to
  prevent.
- **An `error` bound to a declared input rendered as a struct dump.** The pointer arm
  dereferenced `*errorString` before `renderValue` could reach its `error` case. Values that
  render through their own method (`time.Time`, `fmt.Stringer`, `error`) now short-circuit the
  sweep — which also makes the new struct arm safe, since `time.Time` is a struct of unexported
  fields and would otherwise have rebuilt as an empty map.

## [0.21.0] - 2026-08-08

A declared input becomes its own word — and, more consequentially, the frontmatter
`inputs:` block stops being inert.

Until now nothing in `Execute` ever read `Spec.Inputs`, and `InputDef.Default` had no
application site anywhere in the library: a declared input was a promise to a form
builder and nothing more. The only way to actually *use* one was `{~exons.var~}`, the
same verb that reads arbitrary runtime data — which is why no tool could tell a mistyped
input from a legitimate context variable, and why "this input is declared but never
referenced" was undecidable no matter how good the analysis.

### Added
- **`{~exons.input name="x" /~}`** — a reference to a value the *document* declared.
  `exons.var` is unchanged and is **not** deprecated: it remains the context-variable
  verb. The defect was that it was *also* the input verb.
- **Declared inputs are injected under the reserved `input` context root**, with
  `InputDef.Default` applied wherever the caller bound nothing. So
  `{~exons.input name="tone"~}` *is* the path `input.tone`, and `eval="input.verbose"` /
  `in="input.sources"` work with **zero grammar change** to `exons.if` / `exons.for`. A
  verb with its own private lookup space would have been invisible to control flow — a
  half-language where you can print an input but cannot branch on it.
- **`Spec.BindInputs`** (apply declared defaults to caller values, producing the map to
  place under the reserved key) and **`Spec.ValidateInputBinding`** (enforce `required`,
  declared `options` membership, and `max_files`) — the pre-render binding contract. This
  is why `exons.input` **refuses** a `required=` attribute: render time is too late to ask
  a user for a value, and a `required=` inside an unreached `exons.if` branch would
  enforce or not depending on data.
- **`DryRunResult.Inputs []InputReference`** — the sound answer to "which declared inputs
  does this body reference?", from the parsed AST rather than a source re-scan. A consumer
  re-scanning source must agree byte-for-byte with this library's lexer, and gets it wrong
  for hyphenated names, names containing a quote or a backslash, multi-line tags, and
  tildes inside attribute values. Kept separate from `Variables` on purpose: folding the
  two together discards exactly the distinction the verb exists to create.
- **`AttrJoin`, `AttrTz`, `AttrLayout` mirrored into the public constants.** They shipped
  internally in v0.18.0/v0.20.0 and were never exported, so a caller reading the public
  package could not spell an attribute the executor accepts.

### Changed
- **`DryRun`'s `ResolverReference.Registered` is now asked of the registry** instead of
  hardcoded `true` with the comment "assume registered since it parsed". The parser never
  consults the resolver registry — any well-formed tag name parses — so an unregistered or
  typo'd verb was reported as registered, and the one field a caller would use to catch it
  always said everything was fine.
- **`Template.Explain` now applies input injection.** It calls the executor directly and
  so bypassed the `ExecuteWithContext` funnel; without this a document declaring inputs
  would *explain* differently than it *renders*, which is the worst failure mode there is
  for a debugging tool, because the discrepancy looks like the bug being investigated.

### Behaviour changes ⚠

Enumerated by tag pattern rather than prose, because that is what a corpus can be swept
for. Each applies **only** to a document that declares `inputs:`.

| pattern | before | after |
| --- | --- | --- |
| `{~exons.var name="input.x"~}` | miss | hit — renders the default, or empty |
| the same with `onerror="keepraw"` | literal tag text | the value |
| `eval="input"` | falsy (absent) | **truthy** — a non-empty map |
| any `exons.var` miss | did-you-mean list | now also contains `"input"` |
| `TokenCount` | N | N + the rendered defaults |

`data["input"] = "a string"` is **not** affected. Injection no-ops when the reserved root
already holds a non-map, because `input` is the most idiomatic key name in a prompt
library and `exons.include` copies its non-reserved attributes into the child data as
strings — so `{~exons.include template="x" input="y" /~}` would otherwise let a
*document* reach the failure path.

A declared input with neither a bound value nor a default lands **present and nil**,
not absent. Every consumer already does the right thing with nil — it renders empty,
evaluates falsy, and iterates zero times — so an unbound optional multiselect behaves
sanely with no executor change. The payoff is the equivalence **present ⇔ declared**,
which is what lets an absent name be reported as the author typo it is.

### Security
- **`exons.input` withholds byte slices at every depth.** `renderValue` matches a slice on
  its element *kind* and returns `string(rv.Bytes())` for `uint8` — deliberately, so
  `json.RawMessage` renders as text. That arm is correct for `exons.var` and catastrophic
  for an uploaded file: a caller binding the bytes would paste the entire file body into
  the prompt, by accident rather than by misuse. Dispatch is by value *shape*, not by the
  declared `type:`, so it fails closed even when an author mislabels an upload. A list of
  `{name, mime_type, size_bytes}` maps renders as a filename manifest instead.
- **Declared defaults are deep-copied at injection.** A `*Template` is built to be executed
  many times, concurrently, and `InputDef.Default` holds YAML-decoded values — aliasing a
  map or slice default into a live context would share it across renders.

### Known gaps (documented, not fixed here)
- A resolver's `Validate` is **never invoked by the executor**. `exons.input` therefore
  routes its attribute checks through one helper that both `Validate` and `Resolve` call,
  so a refusal cannot be dead code.
- `Template.Explain` still bypasses **inheritance** resolution, so a template using
  `extends` explains its own AST rather than the spliced one. Predates this release.
- `exons.if` / `exons.for` / `exons.switch` fail **unconditionally** on a condition error,
  a missing collection path, or a non-iterable value: their parse discards the tag
  attributes, so `onerror=`/`default=` are structurally unreachable on a block tag.
  Present-as-nil removes the common input-shaped case from that path.
- `extends` does not merge frontmatter, so a parent's `inputs:` are invisible and the
  parent's body resolves against the **child's** declarations.

## [0.20.0] - 2026-08-07

The two halves of v0.19.0's input vocabulary that could not actually be *used*: the
order the author asked their questions in, and a legible rendering of the values the
new kinds bind. Both were found downstream, in atlas's typed prompt-form work, where
each one is visible to an end user.

### Fixed
- **A list or object value reached the model as Go debug output.** `{~exons.var~}`
  fell through to `fmt.Sprintf("%v")` for every non-scalar, so a `multiselect` or
  `sort` value rendered as `[cost speed]` and an `associate` value as
  `[map[left:eu right:gdpr]]` — inside a sentence, addressed to a language model.
  Those are precisely the kinds v0.19.0 introduced. Values now render as prose: a
  list as `cost, speed`, an object as `beta: 2, gamma: 3` (keys **sorted**, because a
  Go map has no order and range order would make the same value produce a different
  prompt each run), and a nested composite with delimiters (`[a, b]`,
  `(left: eu, right: gdpr)`) so a list of pairs cannot smear into one comma-run.
  Scalars, `nil` and `fmt.Stringer` are unchanged. A byte slice renders as text
  (matched on the element kind, so `json.RawMessage` and other named types count);
  a `time.Time` renders RFC 3339 rather than through its debug `String()`; a struct
  renders `Field: value` in declaration order; only kinds that cannot contain another
  value (chan, func, complex) still reach `%v`.
  ⚠ **This is a behaviour change for any consumer that was parsing the `%v` form** —
  it is a MINOR bump for that reason, and the test asserting `"[a b]"` had been
  pinning the defect as if it were the contract.
- **The depth bound did not bound anything a cycle could reach.** It incremented only
  on slice elements and map entries, so `var i any; i = &i` recursed through the
  pointer arm until the stack died — and the fallback the comment promised would
  "terminate" was `fmt.Sprintf("%v")`, which does not detect a slice containing
  itself either. Every recursive arm now counts, and past the bound the value is
  elided (`…`) rather than handed to fmt. ⚠ The test that was meant to cover this
  used a struct field, which landed on the untraversed arm — it passed against the
  hole.
- **`Serialize` / `ExportFull` silently re-alphabetized the inputs.** They build a Go
  map and `yaml.Marshal` sorts a map's keys, so an export → re-import round trip
  destroyed the authored order this release exists to preserve, on the documented
  export path. `input_order` is now emitted. ⚠ The round-trip test missed it by
  calling `yaml.Marshal(spec)` directly, where the struct tag does the work.
- **A merge key inside `inputs:` made a previously-valid document fail to parse.**
  `<<` is a literal key in the YAML node and never a key in the decoded map, so the
  derived order contained `"<<"` and validation rejected it. The derived order is now
  filtered against the decoded map, which is also what makes "a derived order cannot
  fail validation" true rather than merely asserted. An `inputs: *alias` is likewise
  followed rather than silently dropped to alphabetical.
- **The prompty importer fabricated an alphabetical order and presented it as
  authored.** It round-trips frontmatter through a Go map before `Parse` sees it, so
  the order `Parse` derived was already sorted — indistinguishable downstream from an
  author's. The order is read from the source document instead.
- **`Spec.Inputs` is a Go map, so the authored order of the inputs was destroyed at
  unmarshal** and no consumer could recover it. A form is a sequence of questions, and
  every downstream projection had to sort by key and say so — asking for a zip code
  before an address, with no way to do better. `Spec.UnmarshalYAML` now records the
  YAML mapping's key order.

### Added
- **`Spec.InputOrder`** (`input_order:`) and **`Spec.OrderedInputKeys()`**. Existing
  documents gain the order with no authoring change. `OrderedInputKeys()` is the
  accessor consumers should walk: it is **total** where the field is partial —
  ordered keys first, then any remainder sorted — so even a `Spec` assembled in Go
  yields a stable sequence instead of randomised map iteration. The order serializes
  in both YAML and JSON, so it survives a round-trip through a store that keeps only
  the projection, and a hand-written `input_order` overrides the derived one.
- **`join` attribute on `{~exons.var~}`** — the separator between a list value's
  elements (`join=" > "` → `cost > speed > quality`, for a `sort` value whose ranking
  is the point). Top level only: reusing it one level down would spell two different
  relations the same way. Defaults to `", "`.
- `Spec.Validate()` rejects a **hand-written** `input_order` that names an undeclared
  input, or names one twice — an author's contradiction, which would otherwise silently
  drop a field to the end of the form. A derived order cannot fail either check, and
  no previously-parsing document can newly fail: `input_order` is a new key.
- `input_order` declared in `schema/exons.schema.json` (the top level is
  `additionalProperties: true`, so this is editor completion, not a validity fix).

### Changed
- `Spec` now has a custom `UnmarshalYAML`. It decodes through a `type rawSpec Spec`
  alias — no methods, identical field tags — so every existing decode behaviour is
  what it was, including the `yaml:",inline"` Extensions catch-all. The method only
  adds `InputOrder`. `Clone()` copies it.

## [0.19.0] - 2026-08-06

A prompt `subtype` discriminator and a complete input-kind vocabulary. Consumed by
aigentverse MP-STENCIL (DC103-stencil / DC104-quill), where prompts split into
composable *fragments* and executable *prompt-templates*.

Both additions are **declaration-only and advisory**. `Spec.Validate()` gained no new
checks and rejects nothing it previously accepted: go-exons declares, the executing
system validates. That is a deliberate contract, not an omission — an enforced enum
here would break every consumer pinning a newer vocabulary than the library.

### Added
- `Spec.Subtype` (`subtype:`), refining `Type`. Meaningful today only for
  `DocumentTypePrompt`, with `SubtypePromptFragment` / `SubtypePromptTemplate`. Empty
  means unspecified. Serialized **only alongside `type`** (same `IncludeAgentFields`
  gate) — a subtype that refines nothing is meaningless, so the two never travel apart.
- Input kind constants completing the `InputDef.Type` vocabulary: `InputTypeText`,
  `InputTypeNumber`, `InputTypeBoolean`, `InputTypeFileUpload`, `InputTypeSort`,
  `InputTypeAssociate` — joining the existing `InputTypeSelect` / `InputTypeMultiselect`.
- `InputDef.AssociateWith` (`associate_with:`) — the right-hand set for `associate`,
  whose bound value is a **many-to-many** set of pairs with `Options`.
- `InputDef.Accept`, `InputDef.MaxSizeBytes`, `InputDef.MaxFiles` for `file-upload`.
  `MaxSizeBytes` bounds each **individual** file; `MaxFiles` bounds **how many** — two
  distinct limits, and enforcing only one of them is the bug they exist to prevent.
- `A2AMIMEApplicationOctetStream`, for a file-upload input's opaque binary value.
- README sections documenting both the `subtype` discriminator and the full input-kind
  table. The `requirements` block (0.13.0) shipped with no README section; this closes
  that pattern rather than repeating it.

### Fixed
- **`schema/exons.schema.json` rejected valid documents.** `InputDef` is
  `additionalProperties: false` but never gained `label` / `options` when those shipped
  in 0.16.0, so any editor validating against the published schema flagged a correct
  `select` input as invalid. Its `type` enum was likewise missing `select` /
  `multiselect`. The enum is now **removed** rather than extended — an enum contradicts
  an advisory, open vocabulary — and replaced by `examples` plus a documented list.
- The schema root was missing `content_format` and `recommended_agents`, both long
  present on `Spec`. (The root is `additionalProperties: true`, so these were tolerated
  rather than rejected — undocumented, not broken.)
- `Spec.IsAgentSkillsCompatible()` now accounts for `Subtype`. Agent Skills has never
  heard of the field, so a spec carrying one is not compatible — checked explicitly
  rather than inferred from `Type == ""`, since a subtype can be set with no type.

### Notes
- `Spec.Clone()` deep-copies the new `AssociateWith` and `Accept` slices. A shallow
  copy here is a silent aliasing bug, and it is asserted by test.
- `inputTypeToMIME` now enumerates every kind instead of falling through to
  `text/plain`. The structured kinds bind a list or a set of pairs, and reporting those
  as `text/plain` told an A2A consumer it could send a bare string for an input that can
  never be one. The `default:` arm stays `text/plain` because the vocabulary is open.

## [0.18.0] - 2026-07-28

New `{~exons.now~}` built-in **output** tag — prints a formatted reference time into
body text (curated named formats `iso`/`date`/`datetime`/`time`/`year`/`month`/`day`/
`weekday`/`unix`/`rfc1123`/`date-de`, optional `tz=` IANA zone, `layout=` raw-Go escape
hatch). Distinct from the eval-only date/time expression functions, which have no output
path. The reference time is seeded per render via the reserved `ContextKeyReferenceTime`
data key, falling back to `time.Now()`. Consumed by aigentverse DC92-chronos.

(Entry reconstructed in 0.19.0 — the 0.18.0 release shipped without one.)

## [0.17.0] - 2026-07-19

A2A Agent Card upgraded to **spec v1.0.1** (`github.com/a2aproject/A2A`
`specification/a2a.proto` @ tag v1.0.1). Consumed by aigentverse DC82-emissary as the
enterprise-registry bridge (Google Cloud Agent Registry / AGNTCY ingest declaration cards).

### Changed (breaking — pre-1.0, no compatibility shim)
- `a2a.AgentCard` now models v1.0.1: transport moved from `url` + top-level
  `protocolVersion` into a required `supportedInterfaces[]` (`AgentInterface{url,
  protocolBinding, protocolVersion}`; protocol version is **per interface**). There is
  no top-level `metadata` — vendor data rides in `capabilities.extensions[]`
  (`AgentExtension{uri, description, required, params}`). `security` → `securityRequirements`.
  New fields: `documentationUrl`, `iconUrl`, `signatures[]`.
- `AgentSkill` gains required `description`/`tags` + `examples`/`securityRequirements`;
  `AgentCapabilities` gains `extensions[]`/`extendedAgentCard` and uses `*bool` for
  `streaming`/`pushNotifications` (unset ⇒ omitted, never an explicit default).
- `CompileAgentCard`/`A2ACardOptions`: takes `SupportedInterfaces` + `Extensions`
  instead of a required `URL` (removed `ErrMsgA2ACardMissingURL`); synthesizes a skill
  from the agent when the spec declares none; skill description falls back to the
  name so the required field is never blank; safety/dispatch/a2a-prefixed enrichment
  now rides as one go-exons metadata extension (`A2AExtensionURIGoExonsMetadata`).
- `A2AProtocolVersionDefault` is now `"1.0"` (the per-interface default).

### Added
- **§8.4 detached-JWS signing (RFC 7515), key-injected:** `AgentCard.CanonicalPayload`
  (RFC-8785/JCS via `github.com/gowebpki/jcs`, signatures excluded), `EncodeProtectedHeader`
  (`alg:EdDSA, typ:JOSE, kid, jku`), `JWSSigningInput`, `AttachDetachedSignature`,
  `VerifySignatures` — the package produces/consumes the signature material but the key
  operation is injected by the caller (go-exons never holds a key).
- **Offline conformance:** `AgentCard.Validate` checks the pinned v1.0.1 required-field
  rules (`A2ASpecSource` records the source tag); zero-network.

### Dependencies
- Added `github.com/gowebpki/jcs` v1.0.1 (RFC 8785 JSON Canonicalization).

## [0.15.0-dc11] - 2026-07-12

Syntax-safety cycle: templates can now safely contain examples of their own
syntax. Normative lexical spec: [docs/template-syntax.md](docs/template-syntax.md).

### Added

- **Verbatim tilde fences `{~~ ... ~~}`**: a `{` + k tildes (k ≥ 2) opens a
  fence closed by the first maximal run of exactly k tildes + `}`; the body
  is emitted byte-for-byte with no tag/escape processing and may contain
  lexically invalid syntax, full raw blocks, anything. Body contains `~~}`?
  Add a tilde per side (`{~~~ ... ~~~}`), like markdown fence escalation.
  Unterminated fences are hard lexer errors (position + tilde count).
  Backward-compatible by construction: `{~~` was previously a hard lexer
  error ("invalid tag name"), not literal text, so no valid template changes
  meaning. Disabled under custom `WithDelimiters` (stays plain text).
- **`WithMarkdownFences()` engine option**: markdown fenced code blocks
  (CommonMark subset: ```` ``` ````/`~~~`, ≤3-space indent, ≥-length same-char
  closer) become inert regions — tags/escapes/fences inside render as literal
  text. A fence whose info string starts with `exons` (```` ```exons ````)
  renders live. Unclosed fences are inert to EOF (matching markdown
  renderers) with a `Validate()` warning. Inline code spans stay live.
- **`Spec.ContentFormat`** (`content_format` in YAML/JSON) — content-format
  hint; `ImportFromSkillMD` sets it to `"markdown"` so consumers know to
  render the body with `WithMarkdownFences()`.
- **`Validate()` markdown-fence lints** (`WarnMsgTagLikeInInertFence`,
  `WarnMsgUnclosedMarkdownFence`), emitted even when the template fails to
  parse.
- VS Code grammar: verbatim-fence scope (backreference-matched tilde runs)
  and inert/live markdown code-fence scopes (closer length approximated —
  TextMate cannot express "at least as long").
- `docs/template-syntax.md` (normative lexical grammar + precedence table)
  and `docs/masterplan.md` (cycle log).

### BREAKING

- **Raw/comment blocks now scan verbatim at the lexer level.** Consequences:
  - Only the canonical close (`{~/exons.raw~}`, no interior whitespace) ends
    a block; `{~/ exons.raw ~}` is now body content.
  - Escapes inside raw are preserved byte-for-byte (`\{~` stays `\{~`;
    previously it was un-escaped to `{~`).
  - Body content no longer needs to lex cleanly — broken teaching examples
    (`{~ 5 ~}`, a lone `{~`) are now legal raw content.
  - Unterminated raw/comment is a lexer error (was a parser mismatched-tag
    error).
  - Nested raw no longer errors: the first close wins and inner openers are
    literal text. Removed `NewNestedRawBlockError` and `ErrMsgNestedRawBlock`
    from the public API. Use `{~~ ... ~~}` to embed a complete raw block.
- A stray top-level block close (`{~/x~}` with no open) is now a parse error;
  previously everything after it was silently dropped.
- Internal (pre-release): `internal.NewParserWithSource` and
  `internal.NewInheritanceResolver` take a `LexerConfig`.

### Fixed

- Raw-block content round-trips byte-for-byte; the old parser-level
  reconstruction canonicalized quotes/whitespace and hardcoded default
  delimiters (broken under `WithDelimiters`).
- Template inheritance re-lexed parent templates with default config,
  ignoring `WithDelimiters` (and now `WithMarkdownFences`); the engine lexer
  config is threaded through.
- Migrated `.golangci.yml` to golangci-lint v2 and fixed the staticcheck
  findings it surfaced (`fmt.Fprintf` over `WriteString(fmt.Sprintf(...))`).
- All markdown import entry points now agree on `ContentFormat`: `Import`
  (`.md`) and `ImportDirectory` (zip) mark specs `"markdown"` exactly like
  `ImportFromSkillMD`.
- `content_format` is serialized only when `IncludeExtensions` is set, so
  Agent Skills compatible exports (`ExportToSkillMD`) stay free of
  non-standard frontmatter keys.

## [0.14.0] - 2026-06-21

### Added

- `exons.ref` now accepts namespace-qualified `@org/name` slugs (e.g.
  `{~exons.ref slug="@aigentverse/source-scout" /~}`) in addition to bare slugs. The
  `@org/name` form is the portable cross-namespace reference contract used by registries that
  address specs by org and name; previously the slug validator (`^[a-z][a-z0-9-]*$`) rejected
  it before the resolver was consulted, so such references could not be composed at all. A
  trailing `@version` still parses (split on the last `@`). Bare slugs are unchanged.

## [0.13.1] - 2026-05-30

### Added
- Bounds on the `requirements` block (`MaxRequirementEntries`, `MaxRequirementFieldLen`) so the governance seam cannot be flooded; documented the `Clone` deep-copy guarantee.

## [0.13.0] - 2026-05-30

### Added
- **Portable `requirements` block** (`Spec.Requirements *SpecRequirements`): an
  additive, optional top-level block declaring abstract MCP capabilities and
  logical credential refs (+ `scope`: `org`/`user`/`per_call`) **without binding
  them** — never server URLs, never secrets. The portable seam that lets
  downstream governance and authoring-time preflight reason about a definition's
  needs while keeping it runtime-agnostic.
  - Types: `SpecRequirements`, `MCPRequirement{capability, credential_ref, scope}`,
    `CredentialRequirement{ref, provider, scope}`; scope constants
    `RequirementScopeOrg`/`User`/`PerCall`.
  - `(*Spec).ValidateRequirements()` and `(*SpecRequirements).Validate()` check
    shape (non-empty capability/ref), capability + ref uniqueness, and scope enum.
    Wired into `Spec.Validate()` and `Spec.Clone()`.
  - Named distinctly from the existing skill-activation `Requirements` type.

## [0.11.0-dc10] - 2026-03-21

### BREAKING
- **Default env var denylist active**: `{~exons.env~}` now blocks environment variables matching common secret patterns (`*_KEY`, `*_SECRET`, `*_TOKEN`, `*_PASSWORD`, `*_PASS`, `*_CREDENTIAL`, `*_PASSPHRASE`, `*_DSN`, `*_CONN_STRING`) by default. Override with `WithEnvDenylist(nil)` or `WithEnvAllowlist(...)`.
- **`DefaultEnvDenyPatterns` is now a function** returning a fresh copy, not a mutable `var` slice. Callers using the old var form need to add `()`.
- **Zip document size limit**: `ImportDirectory` now limits document files (SKILL.md, AGENT.md, PROMPT.md) to 10MB (`MaxImportDocumentSize`). Previously unlimited.
- **Zip path traversal rejection**: `ImportDirectory` rejects resource entries with path traversal (`../`) or absolute paths. `ExportDirectory` returns an error for such paths.

### Added
- `FunctionDef.ToAnthropicTool()` — Anthropic `input_schema` format
- `FunctionDef.ToGeminiTool()` — Gemini `parameters` format (flat, no wrapper)
- `FunctionDef.ToMCPTool()` — MCP `inputSchema` format (camelCase)
- `FunctionDef.ToMistralTool()` — Mistral (OpenAI-compatible) format
- `FunctionDef.ToCohereTool()` — Cohere `parameter_definitions` format
- `ToolsConfig.ToOpenAITools()`, `ToAnthropicTools()`, `ToGeminiTools()`, `ToMCPTools()`, `ToMistralTools()`, `ToCohereTools()` — batch export methods
- `WithEnvAllowlist(patterns)` — restrict `{~exons.env~}` to matching glob patterns only
- `WithEnvDenylist(patterns)` — block `{~exons.env~}` for matching glob patterns
- `WithEnvDisabled()` — completely disable `{~exons.env~}` tag
- `WithMaxOutputSize(size)` — cap rendered template output (default 10MB)
- `DefaultEnvDenyPatterns()` — returns default deny patterns (immutable function)
- `MaxImportDocumentSize` constant (10MB) for zip document file limits
- JSON Schema for `.exons` format at `schema/exons.schema.json` (draft 2020-12, 28 `$defs`)
- 7 standalone Go examples in `examples/01-*` through `examples/07-*`
- Blog post: `docs/blog/introducing-go-exons.md`

### Security
- Env var access denied by default for common secret patterns (defense in depth)
- Zip bomb prevention: document files limited to 10MB
- Zip path traversal: reject `../` and absolute paths in import; error on export
- Output size enforcement: `executeNodes` enforces `MaxOutputSize` (10MB default)
- Invalid glob patterns in env config fail loudly (no silent bypass)

### Fixed
- README: `EstimateTokens` example now matches actual signature `EstimateTokens(text string) *TokenEstimate`
- README: `RegisterFunc` example now shows correct `*Func` struct form

## [0.10.0-dc9] - 2026-03-20

### BREAKING
- Removed `cmd/exons/` CLI stub (42 LOC, hardcoded stale version, 0% coverage)
- Removed `provider/` and `storage/` empty package stubs
- Removed 56 Get*/Has* methods from `execution.Config` (fields are exported — use direct access)
- Unexported: `GenerateSkillsCatalog`, `GenerateToolsCatalog`, `GetStandardFields`, `GetExonsFields`, `GetExtensionAs`, `StripExtensions`

### Fixed
- `MetaKeyMaxDepth` misuse in spec length validation errors — now uses `MetaKeyMaxLength`
- `NewCredentialValidationError` hardcoded wrong message — now uses generic `ErrMsgCredentialValidationFailed`
- `RegisterTemplate` held write lock during Parse — parse now happens outside lock
- `Context.With*` methods used write lock for read-only ops — now uses RLock
- `knownSpecFields` silently swallowed extension data for non-existent Spec fields (license, compatibility, allowed_tools, metadata, requirements)
- Discarded error in `newTemplateWithConfig` now handled explicitly (nil on error, preserving fail-safe behavior)

### Removed
- 9 dead error constants (`ErrMsgSkillNotFound`, `ErrMsgSkillRefAmbiguous`, `ErrMsgSkillRefInvalidVersion`, `ErrMsgInvalidSkillInjection`, `ErrMsgMessageTemplateNoBody`, `ErrMsgNoDocumentResolver`, `ErrMsgInvalidSkopeSlug`, `ErrMsgInvalidVisibility`, `ErrMsgVersionNumberNegative`)
- 1 dead error constant in root package (`ErrMsgEngineNotAvailable` — internal/ has its own)
- 2 dead error constants (`ErrMsgExtensionNotFound`, `ErrMsgExtensionCastFailed`) — only used by deleted functions
- 2 dead error constructors (`NewEngineNotAvailableError`, `NewAgentValidationError`)
- 1 dead error code (`ErrCodeAgent`)
- 5 dead SpecField constants (`SpecFieldLicense`, `SpecFieldCompatibility`, `SpecFieldAllowedTools`, `SpecFieldMetadata`, `SpecFieldRequirements`)
- 4 dead functions after API surface reduction: `getExtensionAs[T]`, `getStandardFields`, `getExonsFields`, `stripExtensions` (test-only, zero production callers)

## [0.9.0-dc8] - 2026-03-20

### Removed

#### BREAKING: Compilation Layer Removed
- **BREAKING**: Removed `CompileAgent()` — agent compilation is the runtime's responsibility
- **BREAKING**: Removed `CompiledSpec` type and `CompiledMessage` type
- **BREAKING**: Removed `Compile()` on Spec and Template
- **BREAKING**: Removed `ActivateSkill()` — skill injection belongs in the runtime
- **BREAKING**: Removed `AgentDryRun()` and `AgentDryRunResult` — validation without compilation is the consumer's job
- **BREAKING**: Removed `AgentExecutor` wrapper (`Execute`, `ExecuteFile`, `ExecuteSpec`)
- **BREAKING**: Removed `CompileOptions` type and functional option constructors (`WithResolver`, `WithCompileEngine`, etc.)
- **BREAKING**: Removed `ToOpenAIMessages()`, `ToAnthropicMessages()`, `ToGeminiContents()`, `ToProviderMessages()` — provider message formatting belongs in the runtime
- **BREAKING**: Removed `Template.Compile()` and `Template.CompileAgent()` delegation methods
- **BREAKING**: Removed `ValidateAsAgent()` method on Spec
- **BREAKING**: Removed error constructors: `NewCompilationError`, `NewCompileMessageError`, `NewCompileSkillError`, `NewCompileBodyError`, `NewProviderMessageError`, `NewSkillNotFoundError`
- **BREAKING**: Removed `ErrCodeCompile` error code
- **BREAKING**: Removed ~30 compilation-only constants (error messages, dry run categories, provider message keys, injection markers)

#### Files Deleted
- `exons.compile.go` (467 lines)
- `exons.compile.messages.go` (123 lines)
- `exons.compile.dryrun.go` (156 lines)
- `exons.agent.executor.go` (111 lines)
- `exons.compile_test.go` (1,372 lines)
- `exons.compile.integration_test.go` (1,082 lines)
- `exons.compile.messages_test.go` (371 lines)
- `exons.compile.dryrun_test.go` (473 lines)
- `exons.spec.agent_test.go`

### Changed
- README.md rewritten — focuses on parse/validate/serialize/execute flow, no compilation references
- CLAUDE.md updated — removed compilation section and files from package structure
- `ValidateCredentialRefs()` simplified — no longer wraps errors with compilation-specific constructors
- Internal catalog resolver comments updated to remove CompileAgent references
- Internal hint updated: `CompileOptions.Resolver` reference replaced with `Engine.SetSpecResolver()`

### Retained
- Template engine (`Engine.Parse`, `Engine.Execute`, `ExecuteAndExtractMessages`)
- Spec parsing, validation, serialization, clone
- `execution.Config` with all 32+ fields and 6 provider serializers
- A2A Agent Card generation (`CompileAgentCard`)
- Catalog generation (`GenerateSkillsCatalog`, `GenerateToolsCatalog`)
- Import/Export (`.md`, `.zip`, `.prompty`, `.genspec`)
- All metadata types (memory, dispatch, verifications, registry, safety)
- SkillRef, ToolsConfig, ConstraintsConfig types (YAML spec format)
- SkillInjection type and constants (part of SkillRef YAML field)
- Role constants, context key constants (used by template engine)
- Root coverage: 90.6%, internal: 91.1%, execution: 92.1%

## [0.8.0-dc7] - 2026-03-20

### Changed

#### Part A: Flatten GenSpec into Spec (BREAKING)
- **BREAKING**: Removed `genspec/` sub-package entirely — all types moved to root `exons` package
- **BREAKING**: Replaced `GenSpec *genspec.GenSpec` field on `Spec` with 5 flat fields:
  - `Memory *MemorySpec` (yaml: `memory:`)
  - `Dispatch *DispatchSpec` (yaml: `dispatch:`)
  - `Verifications []VerificationCase` (yaml: `verifications:`)
  - `Registry *RegistrySpec` (yaml: `registry:`)
  - `Safety *SafetyConfig` (yaml: `safety:`)
- **BREAKING**: YAML format changed from nested `genspec:` wrapper to flat top-level fields
- **BREAKING**: Removed `GenSpecVersion` constant, `SpecFieldGenSpec` constant, `A2AMetaKeyGenSpecVersion` constant
- **BREAKING**: `SerializeOptions.IncludeGenSpec` renamed to `IncludeMetadata`
- **BREAKING**: `ErrCodeGenSpec` renamed to `ErrCodeMetadata`
- Added `HasMetadata()` method on Spec — replaces `IsGenSpec()` conceptually
- `IsGenSpec()` kept as deprecated alias for `HasMetadata()`
- Added proper `Clone()` methods on all metadata types (fixes shallow-copy defect)
- Origin constants (`OriginInternal`, `OriginExternal`, `OriginUnknown`) and guardrails constants (`GuardrailsEnabled`, `GuardrailsDisabled`) now in root package
- A2A metadata: `genspec.version` key removed from agent card metadata; safety/dispatch enrichment unchanged
- New spec field constants: `SpecFieldMemory`, `SpecFieldDispatch`, `SpecFieldVerifications`, `SpecFieldRegistry`, `SpecFieldSafety`

#### Part B: Prompty Auto-Import
- `ImportPrompty(data)` converts `.prompty` files to valid `Spec` instances
- Tag namespace conversion: `{~prompty.` → `{~exons.` (opening and closing tags)
- YAML field remapping: `delegation` → `dispatch`, `tests` → `verifications`, `plugin` → `registry` (with `trust_level` → `origin`)
- `genspec:` wrapper auto-flattened to top-level fields
- Extra prompty fields (`license`, `compatibility`, `allowed_tools`, `metadata`, `requirements`) moved to `extensions`
- `.prompty` and `.genspec` file extensions recognized by `Import()`
- `isPromptyContent()` auto-detection helper for content with `{~prompty.` tags

## [0.7.0-dc6] - 2026-03-20

### Added

#### A2A Agent Card Generation
- `Spec.CompileAgentCard(ctx, opts)` — generates Google A2A protocol Agent Cards from Spec configuration (DC-6)
- Pure metadata transformation — no template execution or network communication
- `A2ACardOptions` struct — URL, provider info, version, capabilities, security, resolver

#### `a2a/` Package Types
- `a2a.AgentCard` — full Agent Card (v0.3 spec) with name, URL, skills, capabilities, security, metadata
- `a2a.Provider` — organization identification
- `a2a.Capabilities` — streaming and push notification support
- `a2a.Skill` — skill advertisement with ID, name, description, tags, input/output modes
- `AgentCard.ToJSON()` and `AgentCard.ToJSONPretty()` — JSON serialization

#### Auto-Detection
- Skills mapped from `SkillRef` entries; descriptions resolved via `SpecResolver` (non-fatal fallback)
- Streaming capability detected from `execution.Config.Streaming.Enabled`
- Input modes inferred from `Spec.Inputs` types (string→"text/plain", object→"application/json")
- Output modes inferred from execution modality (text→"text/plain", image→"image/png", audio→"audio/mpeg")
- A2A-prefixed extensions (`a2a.*`) merged into card metadata

#### GenSpec Enrichment (unique to go-exons)
- `dispatch.TriggerKeywords` → appended to each A2A skill's Tags
- `registry.Version` → used as agent card Version (fallback after opts)
- `safety.Guardrails`, `safety.DenyTools`, `safety.RequireConfirmationFor` → card metadata
- `genspec.Version` → card metadata under `genspec.version`
- `dispatch.TriggerDescription` → card metadata under `dispatch.trigger_description`

#### Constants and Errors
- A2A metadata key constants: `A2AMetaKeySafetyGuardrails`, `A2AMetaKeySafetyDenyTools`, `A2AMetaKeySafetyConfirmation`, `A2AMetaKeyGenSpecVersion`, `A2AMetaKeyDispatchDescription`
- `NewA2AError()` constructor wrapping cuserr with `ErrCodeA2A`
- Helper functions: `modalityToMIME`, `inputTypeToMIME`, `sortedStringKeys`

#### Testing
- 40+ test functions in `exons.a2a_test.go` covering all paths
- Root coverage: 91.1%, total: 90.7%

## [0.6.0-dc5] - 2026-03-20

### Added
- Full agent compilation pipeline: `Spec.CompileAgent()` produces `CompiledSpec` with messages, execution, tools, constraints (DC-5)
- `Spec.Compile()` — simple body compilation through template engine
- `Spec.ActivateSkill()` — skill activation with injection into system_prompt, user_context, or none
- `Spec.AgentDryRun()` — 6-step validation collecting ALL issues without stopping at first error
- `CompiledMessage` type — compilation output (distinct from `Message` for template output extraction)
- `CompileOptions` with functional options: `WithResolver`, `WithCompileEngine`, `WithSkillsCatalogFormat`, `WithToolsCatalogFormat`
- Provider message serialization: `CompiledSpec.ToOpenAIMessages()`, `ToAnthropicMessages()`, `ToGeminiContents()`, `ToProviderMessages(provider)`
- `AgentExecutor` convenience wrapper: `Execute`, `ExecuteFile`, `ExecuteSpec`, `ActivateSkill`
- `AgentDryRunResult` with `OK()`, `HasErrors()`, `String()` methods
- Clone methods on types: `ToolsConfig.Clone()`, `ConstraintsConfig.Clone()`, `OperationalConstraints.Clone()`, `CredentialRef.Clone()`, `CredentialRef.Validate()`
- `Spec.ValidateCredentialRefs()` — validates credential map, default label, and skill credential labels
- Error constructors: `NewCompilationError`, `NewCompileMessageError`, `NewCompileSkillError`, `NewCompileBodyError`, `NewProviderMessageError`, `NewSkillNotFoundError`
- 25+ compile/DryRun/provider constants
- 130+ new test functions across 4 test files
- Root coverage: 89.5%, internal: 91.1%, execution: 92.1%

### Changed
- `Template.Compile()` and `Template.CompileAgent()` now delegate to `Spec` methods (stubs removed)
- `Spec.Clone()` now delegates to standalone `ToolsConfig.Clone()`, `ConstraintsConfig.Clone()`, `CredentialRef.Clone()`

### Removed
- `ErrMsgCompileNotAvailable` constant and `NewCompileNotAvailableError()` — replaced by real compilation
- `CompiledSpec` and `CompileOptions` placeholder definitions from `exons.template.go` — moved to `exons.compile.go`

## [0.5.0-dc4] - 2026-03-20

### Added
- Catalog generation API: `GenerateSkillsCatalog()` and `GenerateToolsCatalog()` with 4 formats (default, detailed, compact, function_calling) (DC-4)
- `NoopSpecResolver` — default SpecResolver that always returns not-found errors
- `MapSpecResolver` — thread-safe in-memory SpecResolver with `Add`, `AddMulti`, `Remove`, `Has`, `Count`
- `Engine.SetSpecResolver()` / `Engine.GetSpecResolver()` — configure spec resolver on Engine
- `Engine.ExecuteWithCatalogs()` — auto-generates skill/tool catalog strings and injects into context
- Auto-injection: Engine automatically injects SpecResolver into execution Context for `{~exons.ref~}` resolution
- `Import()` / `ImportDirectory()` — import from `.md` or `.zip` archives (SKILL.md/AGENT.md/PROMPT.md)
- `ExportDirectory()` — export Spec + resources to `.zip` archive
- `ImportFromSkillMD()` — parse SKILL.md format (frontmatter + body)
- `Spec.ExportToSkillMD()` — serialize with Agent Skills compatible fields only
- `Spec.StripExtensions()` — clone with execution/extensions/agent-fields removed
- `Spec.ValidateAsAgent()` — validate spec has agent type, execution config, and body/messages
- `ToolsConfig.HasTools()` — check if tools config has any functions or MCP servers
- `FunctionDef.ToOpenAITool()` — OpenAI-compatible tool definition map
- `ImportResult` struct for import results with Spec and Resources
- `MapSpecResolverEntry` struct for bulk resolver population
- 42 end-to-end integration tests verifying all 14 template tags through public Engine API
- Error constructors: `NewCatalogError`, `NewExportError`, `NewImportError`, `NewAgentValidationError`
- Root coverage: 88.8%, internal: 91.1%, execution: 92.1%

### Changed
- `Engine.Execute()` now auto-injects SpecResolver into context when configured
- `Engine.ExecuteTemplate()` now auto-injects SpecResolver for nested templates
- Version bumped to 0.5.0

## [0.4.0-dc3] - 2026-03-20

### Added
- Full `execution.Config` with 32 fields covering all major LLM providers (DC-3)
- Provider serialization: `ToOpenAI()`, `ToAnthropic()`, `ToGemini()`, `ToVLLM()`, `ToMistral()`, `ToCohere()`
- `Config.Validate()` — validates all field ranges and delegates to sub-type validators
- `Config.Clone()` — deep copy of all pointer/slice/map/nested config fields
- `Config.Merge()` — 3-layer precedence merge (other wins for non-nil fields)
- `GetEffectiveProvider()` — auto-detect provider from model name and config shape
- Get/Has pairs for all ~30 optional config fields
- `ProviderFormat(provider)` — dispatch to provider-specific response format
- `Config.JSON()` / `Config.YAML()` — convenience serialization methods
- Sub-types: `ThinkingConfig`, `ResponseFormat`, `JSONSchemaSpec`, `EnumConstraint`, `GuidedDecoding`
- Media types: `ImageConfig` (11 fields), `AudioConfig` (6), `EmbeddingConfig` (7), `StreamingConfig`, `AsyncConfig`
- Schema helpers: `GeminiTaskType()`, `CohereUpperCase()`, `ensureAdditionalPropertiesFalse()`
- Model detection: `isOpenAIModel`, `isAnthropicModel`, `isGeminiModel`, `isMistralModel`, `isCohereModel`
- `Spec.Serialize(opts)` — YAML frontmatter + body export with configurable field inclusion
- `SerializeOptions` with `IncludeExecution`, `IncludeExtensions`, `IncludeAgentFields`, `IncludeContext`, `IncludeCredentials`, `IncludeGenSpec`
- Factory functions: `DefaultSerializeOptions()`, `AgentSkillsExportOptions()`, `FullExportWithCredentials()`
- `Parse(data)` / `ParseFile(path)` / `MustParse(data)` — standalone `.exons` file parsing
- Extension helpers: `GetExtension`, `SetExtension`, `RemoveExtension`, `GetExtensionAs[T]`
- `GetStandardFields()` / `GetExonsFields()` — field classification helpers
- `CompiledSpec` fields typed: `Execution *execution.Config`, `Tools *ToolsConfig`, `Constraints *ConstraintsConfig`
- `Spec.Clone()` delegates to `Config.Clone()` (replaces 25 lines of manual copying)
- `Spec.Validate()` delegates to `Config.Validate()` for execution config validation
- execution/ package: 92.1% coverage, root package: 88.3% coverage

### Changed
- **BREAKING**: `execution.Config.Stop` renamed to `StopSequences` (yaml: `stop_sequences`)
- Version bumped to 0.4.0

## [0.3.0-dc2] - 2026-03-20

### Added
- Root package public API wrapping internal engine (DC-2)
- `Engine` — main entry point: `New()`, `MustNew()`, `Parse()`, `Execute()`
- `Template` — parsed template: `Execute()`, `ExecuteWithContext()`, `ExecuteAndExtractMessages()`
- `Context` — execution context with dot-notation paths, typed getters, parent-child scoping
- `Resolver` / `SpecResolver` / `Attributes` interfaces for custom tag handlers
- `ResolverFunc` convenience type for function-based resolvers
- `Func` type for custom expression functions
- `Spec` — YAML frontmatter parsing (`ParseYAMLSpec`), validation, `Clone()`
- `CompiledSpec` placeholder type (compile stubs return error until DC-5)
- `Message` type and `ExtractMessagesFromOutput()` for LLM API integration
- `ValidationResult` / `Engine.Validate()` — AST-walking template validator
- `HookRegistry` — simplified hook system (10 hook points, no access-control deps)
- `LoggingHook` and `TimingHook` factory functions
- `DryRunResult` / `Template.DryRun()` — static analysis without execution
- `ExplainResult` / `Template.Explain()` — human-readable execution walkthrough
- `TokenEstimate` / `EstimateTokens()` — token count estimation with cost budgeting
- `ErrorStrategy` type with `ParseErrorStrategy()` and `IsValidErrorStrategy()`
- `ValidationSeverity` type
- Error constructors via `go-cuserr` for all error categories
- `Position` type for source location tracking
- Functional options: `WithDelimiters()`, `WithErrorStrategy()`, `WithMaxDepth()`, `WithLogger()`
- `TemplateRunner` interface shared by Engine (and future StorageEngine)
- 579 root package tests, 86.7% coverage, all passing with `-race`

### Changed
- Version bumped to 0.3.0

## [0.2.0-dc1] - 2026-03-20

### Added
- Initial project structure
- Core template engine (lexer, parser, executor) from go-prompty lineage
- `.exons` file format with YAML frontmatter and `{~...~}` template syntax
- Spec type with GenSpec support (memory, dispatch, verification, registry, safety)
- Execution configuration with multi-provider serialization
- Provider packages: OpenAI, Anthropic, Gemini, vLLM, Mistral, Cohere
- A2A Agent Card generation
- Storage interfaces with in-memory implementation
- VS Code syntax highlighting for `.exons` files
- CLI tool (`exons`)
- 476 internal tests, 91.1% coverage
