package exons

import "github.com/itsatony/go-exons/internal"

// The untrusted-input fence (vAudience/atlas#819, cycle U1). See internal/exons.untrusted.go for
// the design; this file is the public surface.
//
// A caller marks a value as external data by binding NewUntrustedValue(name, value) instead of
// the value — under the reserved `input` root, in the flat data map, or both (share ONE sealed
// value so the placement count is shared). A document can also declare an input
// `untrusted: true` (InputDef.Untrusted), and the engine seals it itself.
//
// What a sealed value does on every path:
//
//   - {~exons.input name="x" /~} renders UntrustedNotice and the fenced block the FIRST time,
//     a back-reference without the data every later time in the same render;
//   - {~exons.var name="x" /~}, and any path through it (`x.payload`), is REFUSED
//     (ErrMsgUntrustedVarRead) — it would render without the fence;
//   - everything else (an expression, a loop, a debug dump, a %v, JSON) sees an opaque value
//     whose rendering is UntrustedPlaceholder.
//
// The data inside the fence is neutralised (NeutraliseUntrusted): no value can close the block,
// open a new one, or produce a line that looks like the fence.

// UntrustedValue is a sealed piece of external data. Build it with NewUntrustedValue.
type UntrustedValue = internal.UntrustedValue

// NewUntrustedValue seals val as untrusted external data under the input name `name`. A string is
// fenced verbatim; any other value as indented JSON with byte slices withheld. The fence's source
// label is val["source"] when val is a map carrying a non-empty string there, else the name.
// Sealing an already-sealed value returns it unchanged.
//
// ⚠ Build one per render: the value counts its own placements (Placements), which is how a host
// tells whether the template placed it and how a second placement renders without the data.
func NewUntrustedValue(name string, val any) *UntrustedValue {
	return internal.NewUntrustedValue(name, val)
}

// NeutraliseUntrusted rewrites untrusted text so it cannot close or open the fence: fence words
// touching a bracket (`[Ende]`, `⟦ENDE`, homoglyph and zero-width variants included) are
// replaced, the fence's own bracket glyphs become plain ASCII brackets, and control characters
// other than \n, \r and \t are dropped.
func NeutraliseUntrusted(text string) string {
	return internal.NeutraliseUntrusted(text)
}

// StripUntrusted removes every engine-written untrusted rendering from text — the notice line,
// each fenced block, each back-reference — and returns what remains, trimmed: the instruction.
// A host uses it wherever the data must not go (an agent's system frame, @-mention addressing).
// It is exact: the neutraliser guarantees no data or label can contain the fence's glyphs.
func StripUntrusted(text string) string {
	return internal.StripUntrusted(text)
}

const (
	// UntrustedNotice is the fixed line that precedes untrusted data, once per placement.
	UntrustedNotice = internal.UntrustedNotice
	// UntrustedClose is the fence's closing line.
	UntrustedClose = internal.UntrustedClose
	// UntrustedPlaceholder is what a sealed value renders as on every path except exons.input.
	UntrustedPlaceholder = internal.UntrustedPlaceholder
	// UntrustedSourceMaxRunes bounds the source label on the fence's opening line.
	UntrustedSourceMaxRunes = internal.UntrustedSourceMaxRunes
	// UntrustedRepeat is the fixed back-reference a second placement renders (no label, no data).
	UntrustedRepeat = internal.UntrustedRepeat
	// ErrMsgUntrustedOutsideUserMessage is the refusal exons.input answers for a sealed value
	// placed inside a system, assistant or tool {~exons.message~} block.
	ErrMsgUntrustedOutsideUserMessage = internal.ErrMsgUntrustedOutsideUserMessage
	// ErrMsgUntrustedVarRead is the refusal exons.var answers for a path that reaches a sealed value.
	ErrMsgUntrustedVarRead = internal.ErrMsgUntrustedVarRead
)
