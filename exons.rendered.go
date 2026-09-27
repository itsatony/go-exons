package exons

import (
	"context"
	"strings"
	"unicode"

	"github.com/itsatony/go-exons/internal"
)

// RenderedBody is text that go-exons RENDERED, carried in a type only go-exons can fill. It is
// what a RenderedSpecResolver returns for a reference whose body it has already rendered, and
// it is the one way to hand {~exons.ref~} text whose message markers must SURVIVE the splice.
//
// ⛔ WHY A TYPE AND NOT AN OPTION (go-exons#9). Since v0.34.1 only the message tag may emit a
// message marker: every leaf of a render is stripped of the NUL delimiter, so a value can never
// forge a message. A reference spliced verbatim is stripped too, because a plain string could be
// anything — and that silently deleted the messages of a body that WAS a go-exons render
// (aigentverse's resolver renders each child itself, under its own budgets). An engine option
// ("this resolver's strings are rendered") would have fixed that by TRUSTING the resolver about
// every string it ever returns, including the ones it did not render: a placeholder for a
// missing child, a heading assembled from recipe data, a cache entry written by other code.
// Any NUL in one of those would be a forged message again, and nothing would say so.
//
// A RenderedBody makes the claim CHECKABLE instead of trusted. Its field is unexported, so the
// only producers are this package's own:
//
//   - Engine.ExecuteRendered / Template.ExecuteRendered — the output of a render, whose every
//     NUL was written by a message tag;
//   - RenderedText — text that was NOT rendered, stripped of every NUL on the way in;
//   - JoinRendered and TrimRightSpace — which only concatenate or trim whitespace, and so can
//     neither open nor close a marker.
//
// By induction every NUL inside any RenderedBody belongs to a message marker a message tag
// wrote. A plain string keeps being stripped wherever it is spliced; nothing about the
// v0.34.1 hardening is weakened.
//
// The zero value is the empty body.
type RenderedBody struct {
	text string
}

// String returns the rendered text, markers included. Hand it to ExtractMessagesFromOutput or
// StripMessageMarkers exactly as you would the result of Engine.Execute.
func (b RenderedBody) String() string {
	return b.text
}

// Len returns the length of the rendered text in bytes, markers included — for a caller that
// budgets what it emits.
func (b RenderedBody) Len() int {
	return len(b.text)
}

// IsEmpty reports whether the body carries no text at all.
func (b RenderedBody) IsEmpty() bool {
	return b.text == ""
}

// TrimRightSpace returns the body without trailing Unicode whitespace. NUL is not whitespace, so
// no marker is ever cut.
func (b RenderedBody) TrimRightSpace() RenderedBody {
	return RenderedBody{text: strings.TrimRightFunc(b.text, unicode.IsSpace)}
}

// RenderedText wraps text that go-exons did NOT render — glue between rendered pieces, a
// heading, a placeholder — so it can be joined with rendered bodies. ⛔ Every NUL is removed:
// unrendered text never carries a message marker, whatever it looks like.
func RenderedText(s string) RenderedBody {
	return RenderedBody{text: internal.StripMarkerBytes(s)}
}

// JoinRendered concatenates bodies in order. Each part's markers are complete (a message tag
// writes its start and end marker together), and every other byte is NUL-free, so the result is
// a shape one render could have produced itself — the render of the parts' templates written one
// after another. Concatenation cannot join two halves into a message no tag wrote.
func JoinRendered(parts ...RenderedBody) RenderedBody {
	switch len(parts) {
	case 0:
		return RenderedBody{}
	case 1:
		return parts[0]
	}
	n := 0
	for _, p := range parts {
		n += len(p.text)
	}
	var b strings.Builder
	b.Grow(n)
	for _, p := range parts {
		b.WriteString(p.text)
	}
	return RenderedBody{text: b.String()}
}

// ExecuteRendered is Execute returning a RenderedBody: the same render, the same output, in the
// type a RenderedSpecResolver hands back for a reference. See RenderedBody.
func (e *Engine) ExecuteRendered(ctx context.Context, source string, data map[string]any) (RenderedBody, error) {
	out, err := e.Execute(ctx, source, data)
	if err != nil {
		return RenderedBody{}, err
	}
	return RenderedBody{text: out}, nil
}

// ExecuteRendered is Execute returning a RenderedBody. See RenderedBody.
func (t *Template) ExecuteRendered(ctx context.Context, data map[string]any) (RenderedBody, error) {
	out, err := t.Execute(ctx, data)
	if err != nil {
		return RenderedBody{}, err
	}
	return RenderedBody{text: out}, nil
}

// RenderedSpecResolver is a SpecResolver that returns references ALREADY RENDERED, as
// RenderedBody values. It is optional: implement it alongside ResolveSpec and {~exons.ref~}
// calls ResolveRenderedSpec instead, splicing the body as-is — no second render, and the body's
// message markers kept.
//
// ⚠ Implementing it is the declaration; no engine option is needed and WithRefVerbatim() makes
// no difference to such a resolver. As with WithRefVerbatim, RefMaxDepth and the circular-chain
// check DO NOT APPLY, because go-exons does not recurse into a body it did not render: the
// resolver owns its own depth, cycle and size bounds.
//
// Position is honoured. At the top level of a document the body's messages stay messages.
// Spliced INSIDE an {~exons.message~}, the body contributes its content only — its markers are
// removed, not left as words — the same flatten rule a nested message has followed since
// v0.34.1.
//
// ResolveSpec is still required (SetSpecResolver takes a SpecResolver, and other readers such as
// the skills catalog use it); only {~exons.ref~} prefers ResolveRenderedSpec.
type RenderedSpecResolver interface {
	SpecResolver
	// ResolveRenderedSpec returns the rendered body of the referenced spec, or an error if it
	// cannot be resolved (reported as "referenced spec not found").
	ResolveRenderedSpec(ctx context.Context, slug string, version string) (RenderedBody, error)
}

// spliceRendered returns a rendered reference body in the shape its position needs: markers kept
// at the top level, flattened to content inside an enclosing message.
func spliceRendered(ctx context.Context, body RenderedBody) string {
	if internal.InsideMessage(ctx) {
		return StripMessageMarkers(body.text)
	}
	return body.text
}
