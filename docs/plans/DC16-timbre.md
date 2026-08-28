# DC16-timbre — an agent can say how it should sound

> **Status:** ✅ **SHIPPED as v0.27.0** (2026-08-28). Closes issue **#2**.
> `ci-local` green: 0 lint issues, race clean, coverage **90.9 %** (floor 88).
> Downstream: `vAudience/aigentverse` re-publishes this schema byte-identically and consumes the
> block in its own cycle (that repo's issue #32).
>
> *timbre*, n. — the quality that distinguishes one voice from another when pitch and loudness are
> the same. Exactly what a spec could not state.

---

## Why this cycle exists

An exons spec could describe **what** an agent is and **which LLM** runs it. It could not describe
**how it should sound when it is read aloud**. In a product where agents are coworkers you talk to,
that means every coworker is read in the workspace's one voice, and the agent's author — the person
who decided everything else about it — has no say.

Issue #2 filed it with the schema evidence and two options. **A** — a typed top-level `speech:`
block, symmetric with the `transcription:` convention — was taken.

⚠ **The issue's own framing needed one correction, which it had already made in fairness to the
schema:** `voice` *is* expressible today, as `execution.audio.voice`. What is missing is everything
that makes a named voice usable — the engine that defines the name, the delivery steering, and the
jurisdiction it may be produced in.

---

## The decisions, and the evidence for each

**1. A top-level block, not a widened `AudioConfig`.** `execution.*` parameterises the call that
produces the document's OUTPUT — `execution.audio` is for an agent whose output *is* audio (modality
`audio_speech`, `schema:738-744`). `speech:` answers a different question: given text this document
produced by any means, in what voice should a consumer read it back? And `ExecutionConfig` is
`additionalProperties: false` around exactly one provider/model pair — the LLM's — so a voice named
there has **no engine that defines it**.

⚠ **There is deliberately NO fallback between `speech.voice` and `execution.audio.voice`.** Nothing
here supersedes that key and nothing reads through to it. A silent fallback would make one key's
meaning depend on which consumer happened to read it.

**2. `SpeechConfig` has `Clone()` and no `Validate()`, and that is the decision.** go-exons stores
declarations; a consumer resolves them. The bounds that exist (speed 0.25–4.0, the `output_format`
vocabulary) are stated in the published schema, which is the contract editors and CI read. A Go
refusal would additionally be a **narrowing shipped in a MINOR release**: before v0.27.0 a `speech:`
block landed inertly in `Spec.Extensions`, so a document carrying one already parses *and validates*
today, and refusing it now would break a document that works. A `Validate()` that returns nil
unconditionally was rejected for the opposite reason — a check that cannot fire reads as protection
and is not.

**3. `transcription:` is declared in the SCHEMA and deliberately NOT a `Spec` field.** This is the
sharpest thing in the cycle, and it inverts what the issue asked for.

A typed field **consumes its key** from the `yaml:",inline"` Extensions map. `transcription:` is a
live convention read as `spec.Extensions["transcription"]` (go-vaibstract's
`ParseExonsTranscriptionSpec`, `exons_transcription.go:56`), so typing it here would empty that key
and **every transcription-configured document would silently lose its provider, model and region at
the consumer's next pin bump** — no error, no warning, a fall back to defaults. The schema `$def`
gives editors and CI the contract without moving the value; `TestTranscriptionStaysInExtensions`
pins both halves, and its reflection arm refuses the field under *any* tag.

⚠ The keys are derived from the consuming reader's own **constants** (`go-vaibstract/constants.go`
:5398-5414), not from prose. The sibling convention `transcription_verifications` is real too and is
deliberately left undeclared rather than guessed at.

**4. `region` is free-form; neither provider field reuses the LLM enum.** Downstream the region value
is provider-interpreted — `eu`, `us`, `eu-central-1`, `europe-west4` — so an enum would refuse values
that work today. And `ExecutionConfig.provider`'s enum (`openai, anthropic, google, gemini, vertex,
vllm, azure, mistral, cohere`) carries no TTS- or STT-only vendor, so reusing it would refuse
`elevenlabs` and `speechmatics`.

---

## The defect this cycle found on its own lines

**`requirements:` has been dropped by every export since it was introduced.** `Spec.Requirements`
(DC-SE0) had no `SpecFieldRequirements` constant, no `knownSpecFields` entry and no
`buildSerializeMap` arm — so the typed field ate the key from Extensions and nothing wrote it back.
`Serialize`, `ExportFull` and every caller of them silently produced a document with no requirements
block.

That is precisely the trap `speech:` had to avoid, sitting on the exact lines that had to change to
avoid it, which is why it is fixed here rather than filed. Both blocks are emitted under
`IncludeMetadata` — which also keeps both **out** of the Agent-Skills card
(`AgentSkillsExportOptions` sets it false), whose portable fields are a closed vocabulary where an
extra key is a conformance failure downstream.

⚠ **The guard has to go through `ExportFull`, never `yaml.Marshal(spec)`** — the struct tag does the
work in a direct marshal, so a direct-marshal test passes against a `buildSerializeMap` that emits
nothing. That is exactly how this gap survived, and `exons.serialize.go:197-206` already carried a
comment saying so about a *different* field.

---

## What shipped

| Area | Change |
|---|---|
| `schema/exons.schema.json` | root `speech` + `transcription` properties; `SpeechConfig`, `TranscriptionConfig`, `VocabularyBias`, `BiasTerm` `$defs`, all `additionalProperties: false` |
| `exons.spec.speech.go` | `SpeechConfig` + `Clone()`, and the reasoning above as comments where the next reader will be standing |
| `exons.spec.go` | `Speech *SpeechConfig`; `Clone()` arm; the transcription prohibition recorded at the field |
| `exons.constants.go` | `SpecFieldSpeech`, `SpecFieldRequirements` |
| `exons.serialize.go` | `knownSpecFields` + `buildSerializeMap` arms for both |
| `exons.speech_test.go` | five guards (below) |
| `schema/schema_test.go` | the four new `$defs` in `expectedDefs` **and** `strictDefs`; the two new root properties; the root's own `additionalProperties: true` |
| docs | `README.md` metadata table, `examples/dns-specialist.exons`, `CHANGELOG.md`, `versions.yaml`, this file, and the **DC15-unframe cycle-log row that was never added** |

## What the tests prove — and how they were probed

- `TestSpeechDecodesIntoTheTypedFieldAndNotIntoExtensions` — the value **and** the key's departure
  from `Extensions`. Asserting the value alone would not prove the move, which is the part that
  changes for every consumer.
- `TestTranscriptionStaysInExtensions` — the inverse, with a reflection arm. **Probed** by adding a
  `Transcription` field: fails.
- `TestFullExportRoundTripsSpeechAndRequirements` — through `ExportFull`. **Probed** by deleting both
  emission arms: fails.
- `TestAgentSkillExportCarriesNeitherSpeechNorRequirements` — two absence assertions **paired with a
  positive one**, because an absence check alone passes against an export that produced nothing.
- `TestShippedExampleDocumentParses` — the README's reference document, which nothing parsed before.
- The root-openness assertion was **probed** by flipping the schema to `false`: fails.

## Stated non-scope

- No `Validate()` on `SpeechConfig` (decision 2), and no runtime schema validation anywhere in this
  repo — the schema remains a published artifact for consumers.
- `transcription_verifications` stays undeclared.
- The render path is untouched, by inspection: `Spec` reaches the engine at four sites
  (`exons.engine.go:170,266,278`, `exons.inputs.go`), none of which enumerate fields, and the
  executor never sees a `Spec` at all. A downstream "render revision" constant does not move.
