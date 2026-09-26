package exons

import (
	"context"
	"log/slog"
)

// RenderSpecRef resolves a reference AND renders it, implementing internal.SpecRefRenderer.
//
// This is the whole of vAudience/atlas#696. Until v0.34.0 {~exons.ref~} returned the referenced
// document's body and the executor spliced it into the output as TEXT, so a fragment's own
// {~exons.ref~}, {~exons.var~} or {~exons.now~} reached the reader as literal tags — while
// {~exons.include~}, the tag that looks like its sibling, executed what it included. The two
// guards in the built-in (RefMaxDepth and the circular-chain check) could never fire, because
// nothing had ever pushed a frame for them to read.
//
// Three decisions are worth stating, because each could defensibly have gone the other way:
//
//   - SCOPE. The referenced body renders in a context derived from the CALLER's, not an isolated
//     one. Verbatim splicing already implied caller scope — the body became part of the caller's
//     document — and a shared fragment that cannot read the document it was pulled into is not
//     worth pulling in. The referenced document's own declared input DEFAULTS fill in underneath,
//     where the caller bound nothing, through the SAME mergeInputBinding the top-level render
//     uses. One rule, not two that can disagree.
//
//   - THE FRAME IS PUSHED HERE, and only here. depth+1 and chain+slug travel on the derived
//     context, so the built-in's guards read a real depth and a real chain. A cycle now refuses
//     and names its chain, which NewRefCircularError has always been able to say and has never
//     been asked to.
//
//   - VERBATIM STAYS REACHABLE. WithRefVerbatim() keeps the old splice for a SpecResolver that
//     returns ALREADY-RENDERED text (aigentverse's prismSpecResolver is the one known case).
//     ⛔ It is a migration path, not the destination: re-rendering rendered text is a no-op only
//     until that text contains a literal {~…~}, and then it is an unknown-tag failure where
//     there used to be inert prose. The fix for such a resolver is to return the raw body and
//     let this function do the chain.
func (a *SpecResolverAdapter) RenderSpecRef(
	ctx context.Context,
	execCtx interface{},
	slug string,
	version string,
) (out string, lookupFailed bool, err error) {
	spec, body, err := a.resolver.ResolveSpec(ctx, slug, version)
	if err != nil {
		// ⛔ Reported through the SECOND return, never by wrapping a sentinel into the error.
		// BuiltinError and ExecutorError both Unwrap to their cause, so a sentinel placed here
		// stays reachable by errors.Is from every enclosing frame, and a nested miss made four
		// outer references each claim "not found" about a slug that resolves.
		return "", true, err
	}

	// A context this adapter cannot derive from is one it cannot push a frame onto, and a frame
	// that is not pushed is a depth limit that does not bound and a cycle that does not refuse.
	// Splicing verbatim is then the only honest answer.
	parent, ok := execCtx.(*Context)
	if !ok || parent == nil {
		return body, false, nil
	}

	// ⛔ The engine comes from the CONTEXT, never from a field on this adapter.
	//
	// The adapter was almost given an engine at construction, and that would have shipped the
	// fix INERT for at least one consumer: vaichat2 builds its own adapter with the exported
	// NewSpecResolverAdapter and injects it per execution (vchat.resolver.spec.go:150, :340)
	// rather than going through Engine.SetSpecResolver. A constructor-held engine would have
	// been nil there, the ref would have spliced verbatim, and every test here would still have
	// been green. ⭐ Wired on one side and read by nobody is this stack's most-repeated defect;
	// reading the engine from the context that is already carrying it makes the wiring
	// impossible to get wrong.
	engine, ok := parent.Engine().(*Engine)
	if !ok || engine == nil {
		// ⚠ Said out loud rather than degraded quietly. A host that seeds its own TemplateExecutor
		// onto the context gets the pre-v0.34.0 splice back with no error and no output anyone
		// would question — literal tags read as an authoring mistake, not as a wiring one.
		slog.Default().Warn(LogMsgRefVerbatimNoEngine,
			slog.String(LogFieldSpecSlug, slug),
			slog.String(LogFieldSpecVersion, version))
		return body, false, nil
	}

	// The operator asked for the old splice — see WithRefVerbatim.
	if engine.config.refVerbatim {
		return body, false, nil
	}

	// ParseBody, never Parse: the resolver handed back a BODY, and Parse would offer its first
	// line to a YAML scanner. A fragment opening with a markdown horizontal rule is the case.
	tmpl, err := engine.ParseBody(body)
	if err != nil {
		return "", false, err
	}
	out, err = tmpl.ExecuteWithContext(ctx, refChildContext(parent, spec, slug))
	return out, false, err
}

// refChildContext derives the context a referenced body renders in: the caller's, with the
// reference frame pushed and the referenced document's input defaults layered underneath.
//
// Built from ONE readState/fromState pair rather than by chaining WithRefDepth → WithRefChain →
// withData. Each of those deep-copies the data map, so the obvious spelling copied the caller's
// whole context three times per reference — on a path that now runs for every ref in every
// render, where it used to run not at all.
func refChildContext(parent *Context, spec *Spec, slug string) *Context {
	// Built fresh rather than appended in place. parent.RefChain() hands back the live slice, and
	// two sibling references at the same depth would each append into the same spare capacity.
	prev := parent.RefChain()
	chain := make([]string, 0, len(prev)+1)
	chain = append(chain, prev...)
	chain = append(chain, slug)

	// Read the caller's binding BEFORE deriving: Get reads through the parent chain, and the
	// derived context shares the parent rather than being one.
	binding, applyInputs := refInputBinding(parent, spec)

	st := parent.readState()
	st.refDepth++
	st.refChain = chain
	child := fromState(st)

	if applyInputs {
		// Written directly because `child` is not yet shared with anything — fromState deep-copied
		// the map and this is the only reference to it. Going back through withData would copy it
		// a second time for no reader.
		if child.data == nil {
			child.data = make(map[string]any, 1)
		}
		child.data[ContextKeyInput] = binding
	}
	return child
}

// refInputBinding layers the referenced document's declared input defaults under whatever the
// caller has already bound. The caller always wins; the reference supplies the fallback — the
// same direction as template inheritance, where the composing document is the authority.
//
// The second return is false when there is nothing to apply, which includes the case where the
// caller bound something under "input" that is not a binding map: contextWithInputs leaves such a
// context alone rather than overwriting the caller's value, and so does this.
func refInputBinding(parent *Context, spec *Spec) (map[string]any, bool) {
	if parent == nil || spec == nil || len(spec.Inputs) == 0 {
		return nil, false
	}

	bound, hasBinding := parent.Get(ContextKeyInput)
	binding, ok := asBindingMap(bound)
	if hasBinding && bound != nil && !ok {
		return nil, false
	}

	return mergeInputBinding(binding, spec.Inputs), true
}
