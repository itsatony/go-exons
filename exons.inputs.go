package exons

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/itsatony/go-exons/internal"
)

// This file makes the frontmatter `inputs:` block MEAN something at execution time.
//
// Until v0.21.0 Spec.Inputs was entirely inert: nothing in Execute ever read it, and
// InputDef.Default had no application site anywhere in the library. A declared input was a
// promise to a form builder and nothing more, so the only way to actually USE one was
// {~exons.var~} — the same verb that reads arbitrary runtime data, which is why no tool could
// tell a mistyped input from a legitimate context variable.
//
// Injection under a reserved root fixes that at the root. Because `input` is an ordinary
// context path, exons.if and exons.for reach declared inputs with no grammar change at all:
// eval="input.verbose" and in="input.sources" simply work.

// contextWithInputs returns a context in which every input this template DECLARES is readable
// under the reserved `input` root, with InputDef.Default supplied wherever the caller bound
// nothing. It never mutates the given context or the caller's data map.
//
// Four rules, each load-bearing:
//
//  1. NO-OP WHEN data["input"] IS NOT A MAP. `input` is an extremely likely ordinary variable
//     name in a prompt — data["input"] = "the user's question" is idiomatic — and an include
//     copies its non-reserved attributes into the child data as STRINGS, so
//     {~exons.include template="x" input="y" /~} lets a DOCUMENT put a string there. Rather
//     than fail, such a template renders exactly as it did before this feature existed.
//
//     An explicit NIL is not that case and must not take this exit. data["input"] = nil says
//     "nothing is bound", not "input is my own variable", and bailing on it skipped injection
//     entirely — so a document's declared defaults silently did not apply, which is the one
//     outcome this whole file exists to prevent. A nil root proceeds as an empty binding.
//
//  2. MERGE PER KEY, CALLER WINS. Only absent (or nil, or empty-string) keys receive the
//     declared default. Replacing the caller's map wholesale would mean that adding an input
//     to a spec silently unbinds it for every existing caller.
//
//  3. DEEP-COPY EVERY DEFAULT. A *Template is built to be executed many times, concurrently.
//     InputDef.Default holds YAML-decoded values, so aliasing a map or slice default into a
//     live context would share it across renders — a cross-request contamination bug, not a
//     tidiness issue.
//
//  4. PRESENT-AS-NIL IS THE UNIVERSAL ZERO. A declared input with neither a bound value nor a
//     default lands as nil, PRESENT. Every consumer already does the right thing with nil: it
//     renders as the empty string, evaluates falsy in a condition, and iterates zero times in
//     a loop — so an unbound optional multiselect behaves sanely with no executor change. The
//     payoff is the equivalence PRESENT ⇔ DECLARED, which is what lets exons.input report an
//     undeclared name as the author error it is.
func (t *Template) contextWithInputs(ctx context.Context, execCtx *Context) *Context {
	// Error discarded: injection has no error channel. For every chain this walk cannot complete
	// the render is refused anyway — resolveInheritance rejects an unreadable, engine-less,
	// circular, missing-parent or over-deep chain before executing a node. The one walk failure
	// that does NOT stop the render is a non-templateProvider executor, where the parent's
	// declarations are unreadable while its body still resolves; there the defaults of an
	// inherited input simply do not apply, exactly as before v0.24.0. See mergedInputs.
	//
	// execCtx goes into the walk because it may carry the per-call parent resolver; the walk
	// reads it before the nil check below because the resolver is a property of the context, and
	// a nil context has none.
	inputs, _ := t.mergedInputs(ctx, execCtx)
	if execCtx == nil || len(inputs) == 0 {
		return execCtx
	}

	// Read through the parent chain: when ExecuteWithContext is handed a child context whose
	// parent carries the bindings, writing a fresh map into the child would SHADOW them
	// entirely — path lookup is all-or-nothing per scope, not a per-key overlay.
	bound, hasBinding := execCtx.Get(ContextKeyInput)
	binding, ok := asBindingMap(bound)
	if hasBinding && bound != nil && !ok {
		return execCtx // rule 1
	}

	// Deep-copied for the same reason the defaults below are, and it was an omission that these
	// were not: Context.Get returns the LIVE value, not a copy, so aliasing a caller's nested map
	// in here would share it across every render that context feeds. The map this replaces came
	// from Context.Data(), which deep-copies — so a shallow merge here silently WEAKENED a
	// thread-safety property the context documents. Spec.BindInputs, the sibling path, deep-copies
	// caller values already; the two now agree.
	//
	// ⚠ "Agree" is the accurate claim, NOT "safe for every shape". deepCopyValue copies the
	// YAML/JSON shapes exhaustively and returns everything else — a struct, a pointer, a []byte, a
	// map with non-string keys — BY REFERENCE, as its own comment says. A Go caller binding one of
	// those to a declared input therefore still shares it live across concurrent renders. That is
	// a property of deepCopyValue, unchanged here and identical on both paths; widening it belongs
	// there, not in two call sites that would then have to agree by memory.
	merged := mergeInputBinding(binding, inputs)

	data := execCtx.Data() // already a deep copy of the direct data
	if data == nil {
		data = make(map[string]any, 1)
	}
	data[ContextKeyInput] = merged
	return execCtx.withData(data)
}

// mergeInputBinding applies rules 2, 3 and 4 above to one binding against one declaration set.
// It is factored out because contextWithInputs is no longer its only caller — DryRun's placeholder
// preview needs the SAME answer, and a preview that disagrees with the render about a default is
// the failure this cycle exists to remove.
func mergeInputBinding(binding map[string]any, inputs map[string]*InputDef) map[string]any {
	merged := make(map[string]any, len(binding)+len(inputs))
	for k, v := range binding {
		merged[k] = deepCopyValue(v)
	}
	for name, def := range inputs {
		if def == nil {
			continue
		}
		if v, present := merged[name]; present && !isUnboundInputValue(v) {
			continue // rule 2
		}
		merged[name] = deepCopyValue(def.Default) // rules 3 and 4
	}
	return merged
}

// mergedInputs returns every input DECLARED anywhere in this template's extends chain, resolved
// by one rule:
//
//	The COMPOSING document is the authority; the COMPOSED document supplies the fallback.
//
// A child's `inputs:` entry overrides a parent's of the same name. This deliberately mirrors
// aigentverse ADR-141, so the library's inheritance story and the registry's composition story
// state the same thing rather than two things a reader must reconcile.
//
// Why this is needed at all: a *Template owns exactly one *Spec — its own. Spec has no `extends:`
// key (inheritance is expressed only by the body tag), so nothing in the frontmatter layer knows a
// parent exists. Meanwhile ResolveInheritance splices the parent's BODY into the child. The parent
// body's {~exons.input~} tags therefore executed against the CHILD's declarations, and since
// exons.input reports an undeclared name as a hard error rather than a blank, a parent that used
// its own declared input could not render at all once extended.
//
// Bounds are the resolver's, not a second set: the same maxDepth and the same "this parent is
// already in the chain" rule as InheritanceResolver.ResolveInheritance. A differently-bounded walk
// of the same chain is a second chance to disagree about what a cycle is.
//
// The returned map is ALWAYS usable, even when the error is non-nil: a chain that stopped short
// still yields every declaration the walk did reach. The three internal callers discard the error
// deliberately — each answers a question with no error channel (does the document declare x? what
// is in scope?), and a chain this walk cannot complete is one ResolveInheritance refuses outright
// at execute, where the reason is reported properly. The two EXPORTED accessors do NOT discard it:
// a partial contract handed to a form builder or a wire projection is a plausible lie, and a
// caller that cannot learn the chain is broken has no way to avoid publishing it.
func (t *Template) mergedInputs(ctx context.Context, execCtx *Context) (map[string]*InputDef, error) {
	ancestors, err := t.ancestorSpecs(ctx, execCtx)
	return t.mergeWithAncestors(ancestors), err
}

// mergeWithAncestors applies the composing-document-is-the-authority rule to an ALREADY-WALKED
// ancestor list, so a caller needing both the merged map and the ancestors themselves pays for
// exactly one traversal.
func (t *Template) mergeWithAncestors(ancestors []*Spec) map[string]*InputDef {
	var own map[string]*InputDef
	if t.spec != nil {
		own = t.spec.Inputs
	}

	if len(ancestors) == 0 {
		return own
	}

	// Walk ROOT-PARENT FIRST and this template LAST, so the nearer document overwrites the
	// further one and the child ends up authoritative.
	merged := make(map[string]*InputDef, len(own)+len(ancestors[0].Inputs))
	for i := len(ancestors) - 1; i >= 0; i-- {
		for name, def := range ancestors[i].Inputs {
			merged[name] = def
		}
	}
	for name, def := range own {
		merged[name] = def
	}
	return merged
}

// ancestorSpecs returns the specs of this template's extends chain, NEAREST-PARENT FIRST, skipping
// ancestors that declare no inputs. It is the single chain walk behind mergedInputs and
// DeclaredInputKeys.
//
// Bounds are the resolver's, not a second set: the same maxDepth and the same "this parent is
// already in the chain" rule as InheritanceResolver.ResolveInheritance. A differently-bounded walk
// of the same chain is a second chance to disagree about what a cycle is.
//
// The error names WHY the walk stopped short of the chain's natural end, and the ancestors
// returned alongside it are still every spec the walk did reach. The rule it enforces:
//
//	ancestorSpecs errors in exactly the cases where ExecuteWithContext refuses to render.
//
// That equivalence is the whole point of reporting at all. Without it DeclaredInputs would be a
// THIRD view of one template, agreeing with neither Execute nor DryRun — the "one rule, two
// implementations" divergence this track exists to remove. The one case with no execute-side
// counterpart is a TemplateExecutor that is not a templateProvider: that chain RESOLVES (source
// is available) while its declarations stay unreadable, so it is reported as its own reason.
//
// WHERE the parents come from follows the execute walk's rule exactly (see inheritanceResolverFor):
// a per-call TemplateSourceResolver on execCtx wins outright when set; otherwise the engine
// registry. Under the per-call resolver each parent's SOURCE is fetched and parsed into a
// throwaway *Template with this template's own engine configuration — never registered, never
// cached — and its spec read; the bounds and the cycle rule are the ones below, unchanged, so the
// two sources share one walk rather than each having its own. execCtx may be nil.
func (t *Template) ancestorSpecs(ctx context.Context, execCtx *Context) ([]*Spec, error) {
	// inheritanceErr FIRST, and the order is the whole correctness of this guard.
	// newTemplateWithConfig sets inheritanceInfo to nil whenever extraction failed, so
	// `inheritanceErr != nil` IMPLIES `inheritanceInfo == nil` — testing the info first makes the
	// error branch unreachable and hands back a clean, complete-LOOKING contract with a nil error
	// for a document ExecuteWithContext refuses. That is worse than the gap this rule closes: it
	// invites a caller to trust "nil error ⇒ safe to publish" precisely where the trust is false.
	// resolveInheritance checks in this order for the same reason.
	if t.inheritanceErr != nil {
		return nil, NewInheritanceError(ErrMsgInheritanceUnreadable, t.inheritanceErr)
	}
	if t.inheritanceInfo == nil {
		return nil, nil // extends nothing — the chain's natural end, at length zero
	}
	lookup, err := t.ancestorLookupFor(execCtx)
	if err != nil {
		return nil, err
	}

	maxDepth := t.config.maxDepth
	if maxDepth <= 0 {
		maxDepth = internal.DefaultMaxInheritanceDepth
	}

	var ancestors []*Spec
	seen := make(map[string]bool)
	current := t
	// The bound is checked the way ResolveInheritance checks it — when a parent at this depth is
	// DEMANDED, not after one has been taken. Bounding the loop itself instead (`depth <= maxDepth`)
	// silently converts "the chain ended exactly at the bound" into "the chain exceeded the bound",
	// so a chain of maxDepth+1 parents that Execute resolves happily reported DepthExceeded here.
	// One rule, two implementations, disagreeing at the boundary — which is the whole failure mode
	// this walk was written to avoid by reusing the resolver's bounds rather than inventing its own.
	for depth := 0; ; depth++ {
		if current.inheritanceInfo == nil {
			return ancestors, nil // walked to the root — the chain's natural end
		}
		if depth > maxDepth {
			return ancestors, NewInheritanceChainError(
				internal.ErrMsgInheritanceDepthExceeded, current.inheritanceInfo.ParentTemplate)
		}
		parentName := current.inheritanceInfo.ParentTemplate
		if seen[parentName] {
			// Circular chain — the same rule, and the same words, ResolveInheritance applies.
			return ancestors, NewInheritanceChainError(internal.ErrMsgCircularInheritance, parentName)
		}
		seen[parentName] = true

		parent, err := lookup(ctx, parentName)
		if err != nil {
			return ancestors, err
		}
		if parent.spec != nil && len(parent.spec.Inputs) > 0 {
			ancestors = append(ancestors, parent.spec)
		}
		current = parent
	}
}

// ancestorLookup fetches one named parent as a parsed *Template, or returns the chain error that
// explains why it could not.
type ancestorLookup func(ctx context.Context, parentName string) (*Template, error)

// templateParser is the optional form of TemplateExecutor that can parse a source string with its
// own configuration. *Engine satisfies it. The specs walk needs it under a per-call resolver,
// which hands back SOURCE: the parent's declarations live in its frontmatter, and only a parse
// with this template's own delimiters and env rules reads them the way Engine.Parse would.
type templateParser interface {
	Parse(source string) (*Template, error)
}

// ancestorLookupFor chooses the specs walk's source of parents by the execute walk's rule, so the
// two walks can never disagree about WHERE a parent comes from:
//
//  1. a per-call TemplateSourceResolver on execCtx — the parent's source is fetched through it and
//     parsed into a throwaway *Template by this template's engine. A lookup FAILURE (err) becomes
//     an inheritance error carrying that cause; found=false becomes template-not-found. Neither
//     is rewritten as the other.
//  2. the engine registry via templateProvider, as before.
//
// Under (1) an engine is still needed, for the parse: a template a consumer holds always has one
// (Engine.Parse is the only constructor), so the failure below names the one honest reason the
// declarations are unreadable while the render would still resolve — the same reason the
// non-templateProvider arm has always reported.
func (t *Template) ancestorLookupFor(execCtx *Context) (ancestorLookup, error) {
	if execCtx != nil {
		if r := execCtx.TemplateSourceResolver(); r != nil {
			parser, ok := t.engine.(templateParser)
			if !ok {
				return nil, NewInheritanceError(ErrMsgInheritanceSpecsUnavailable, nil)
			}
			return func(ctx context.Context, parentName string) (*Template, error) {
				source, found, err := r.ResolveTemplateSource(ctx, parentName)
				if err != nil {
					return nil, NewInheritanceResolutionError(parentName, err)
				}
				if !found {
					return nil, NewInheritanceChainError(ErrMsgTemplateNotFound, parentName)
				}
				parent, err := parser.Parse(source)
				if err != nil {
					return nil, NewInheritanceResolutionError(parentName, err)
				}
				return parent, nil
			}, nil
		}
	}

	if t.engine == nil {
		return nil, NewInheritanceError(ErrMsgInheritanceNoEngine, nil)
	}
	provider, ok := t.engine.(templateProvider)
	if !ok {
		// A third-party TemplateExecutor hands back source, not a parsed *Spec. Re-parsing it
		// here would build a second, differently-configured engine's idea of the parent — see
		// design decision A-1. The child's own declarations remain correct, and are returned;
		// what is NOT correct is calling that partial set the document's whole contract.
		return nil, NewInheritanceError(ErrMsgInheritanceSpecsUnavailable, nil)
	}
	return func(_ context.Context, parentName string) (*Template, error) {
		parent, found := provider.GetTemplate(parentName)
		if !found {
			return nil, NewInheritanceChainError(ErrMsgTemplateNotFound, parentName)
		}
		return parent, nil
	}, nil
}

// DeclaredInputs returns every input this document declares, INCLUDING those inherited through
// {~exons.extends~}, resolved by the rule mergedInputs states: the composing document is the
// authority, the composed document supplies the fallback.
//
// Template.Spec() deliberately returns only the document's OWN frontmatter — it is the parse
// result, and widening it would make Spec() report fields that appear nowhere in the source it
// came from. But a consumer building a form, or projecting a wire contract, needs the merged set:
// with Spec() alone an extending document silently omits every field its parent declares. That
// gap is the reason this accessor exists.
//
// The map and every *InputDef in it are COPIES. Handing back the live pointers would let a caller
// reach through this accessor into a registered PARENT's parsed spec — a template it never named —
// and corrupt every future and concurrent render of it. A form builder that normalizes a default
// in place is an ordinary thing to write, so the aliasing is not a rule worth stating; it is a
// footgun worth removing.
//
// A NON-NIL ERROR MEANS THE CONTRACT IS INCOMPLETE, and the map is what the chain walk reached
// before it stopped. It is returned rather than withheld so a caller may still display what is
// known — but it must not be PUBLISHED as the document's contract, because a cycle, a missing
// parent or an over-deep chain each produce a well-formed map for a document ExecuteWithContext
// refuses to render. That was the shape of this accessor before v0.24.0 shipped it, and a
// consumer projecting a wire contract had no way to tell.
//
// Nil with a nil error is a valid, empty result: the document declares nothing and inherits
// nothing.
func (t *Template) DeclaredInputs() (map[string]*InputDef, error) {
	return t.DeclaredInputsWithContext(context.Background(), nil)
}

// DeclaredInputsWithContext is DeclaredInputs with the parents of the extends chain resolved the way
// ExecuteWithContext(ctx, execCtx) would resolve them: through execCtx's TemplateSourceResolver
// when one is set, else through the engine registry. A nil execCtx is exactly DeclaredInputs.
//
// A host whose parents live behind a per-request identity must use this form, or the contract it
// publishes will describe the child alone — while the render, handed the same context, splices in
// a parent declaring inputs the form never asked for.
func (t *Template) DeclaredInputsWithContext(ctx context.Context, execCtx *Context) (map[string]*InputDef, error) {
	merged, err := t.mergedInputs(ctx, execCtx)
	if len(merged) == 0 {
		return nil, err
	}
	out := make(map[string]*InputDef, len(merged))
	for name, def := range merged {
		out[name] = cloneInputDef(def)
	}
	return out, err
}

// ValidateInputBinding checks a caller's bound values against every input this document
// DECLARES OR INHERITS, and is the accessor a runtime should reach for. Same rules and same
// wording as Spec.ValidateInputBinding — it delegates per value — over a different input set.
//
// ⛔ THE DIFFERENCE IS THE WHOLE POINT, AND THE SPEC-LEVEL ONE IS A TRAP FOR A TEMPLATE.
// Spec.ValidateInputBinding walks s.Inputs, which is the document's OWN frontmatter. A document
// using `extends:` declares inputs its ancestors own, and contextWithInputs binds those happily —
// so validating a template through its Spec silently skips every INHERITED required input while
// the executor goes on treating them as declared. That is one rule with two implementations, on
// the one path a runtime uses to decide whether a caller's form was filled in.
//
// ⚠ VALIDATE THE CALLER'S OWN BAG, NEVER A MERGED AMBIENT ONE. A host that folds its own
// variables (a clock, a locale, a theme) into the same map before calling here will have a
// document declaring a `select` named `theme` checked against the host's theme string — and
// refused on a perfectly healthy call. The values passed here must be exactly what the caller
// supplied.
//
// The error is the resolution error from the extends chain, if any; it is returned ALONGSIDE
// whatever violations were found rather than instead of them, for DeclaredInputs' stated reason —
// a partial chain still yields a usable set, and a caller deciding whether to refuse a request
// wants both facts. ⚠ A caller must not read "no violations" as acceptance while that error is
// non-nil: the set that produced it is incomplete, so the answer is "not proven", not "clean".
//
// An empty (or nil) violation slice with a nil error means the binding is acceptable.
func (t *Template) ValidateInputBinding(values map[string]any) ([]error, error) {
	return t.ValidateInputBindingWithContext(context.Background(), nil, values)
}

// ValidateInputBindingWithContext is ValidateInputBinding over the declaration set
// DeclaredInputsWithContext(ctx, execCtx) reports — the inherited inputs included, from whichever
// source of parents execCtx names. A nil execCtx is exactly ValidateInputBinding.
func (t *Template) ValidateInputBindingWithContext(ctx context.Context, execCtx *Context, values map[string]any) ([]error, error) {
	declared, err := t.DeclaredInputsWithContext(ctx, execCtx)
	if len(declared) == 0 {
		return nil, err
	}
	// A synthetic Spec is deliberately NOT used here. It would work today — OrderedInputKeys
	// falls back to the sorted key set when InputOrder is empty — but it would make this
	// function's correctness depend on which OTHER Spec fields that method happens to read,
	// which is exactly the kind of coupling a later refactor breaks silently.
	var errs []error
	for _, name := range sortedInputNames(declared) {
		def := declared[name]
		if def == nil {
			continue
		}
		val, present := values[name]
		if !present || !satisfiesRequired(val) {
			// A declared default satisfies requiredness — the value is never actually
			// absent at render, so refusing here would make an unsubmittable form.
			if def.Required && def.Default == nil {
				errs = append(errs, fmt.Errorf(ErrFmtInputRequired, name))
			}
			// ⚠ An UNBOUND value skips the per-value checks (there is nothing to check), but
			// an EMPTY COLLECTION is a bound value that simply does not answer a required
			// field — and it has nothing to check either: every per-value rule (option
			// membership, max_files) is vacuous on an empty list.
			continue
		}
		errs = append(errs, validateInputValue(name, def, val)...)
	}
	return errs, err
}

// sortedInputNames orders a merged input set. The merge is a map, so range order would make the
// same binding report its violations in a different order on each run — and a caller rendering
// them to a user, or a test pinning them, would see a value that is correct and unstable.
func sortedInputNames(inputs map[string]*InputDef) []string {
	out := make([]string, 0, len(inputs))
	for name := range inputs {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// cloneInputDef deep-copies an input declaration: every value field, all three option/accept
// slices (whose elements are pure value structs), and the decoded Default. A nil def clones to nil.
//
// The Default carries deepCopyValue's caveat, and it is stated here rather than assumed: that
// helper copies the YAML/JSON shapes exhaustively and returns a struct, a pointer or a []byte BY
// REFERENCE. Today no such value can occur — InputDef.Default is only ever populated by the
// frontmatter YAML unmarshal, which yields none of those shapes — so the copy is total in
// practice. It is not total by CONSTRUCTION, and a future path that sets Default from Go code
// would quietly reintroduce the aliasing this function exists to remove.
func cloneInputDef(def *InputDef) *InputDef {
	if def == nil {
		return nil
	}
	out := *def
	out.Default = deepCopyValue(def.Default)
	if def.Options != nil {
		out.Options = append([]InputOption(nil), def.Options...)
	}
	if def.AssociateWith != nil {
		out.AssociateWith = append([]InputOption(nil), def.AssociateWith...)
	}
	if def.Accept != nil {
		out.Accept = append([]string(nil), def.Accept...)
	}
	return &out
}

// DeclaredInputKeys returns the names from DeclaredInputs in presentation order: this document's
// own declarations in the author order Spec.OrderedInputKeys preserves, then each ancestor's
// remaining names nearest-parent first, a name keeping the position of its first appearance.
//
// The document you opened leads and inherited fields follow — the same "composing document is the
// authority" principle applied to presentation rather than to precedence. Note this is ordering
// only: a name the child restates takes the CHILD's definition wherever it appears in the list.
//
// The error carries the same meaning it carries on DeclaredInputs, and for the same reason: a
// list of field names IS a contract when it is the thing a form is built from.
func (t *Template) DeclaredInputKeys() ([]string, error) {
	// One walk, two orderings — the merged map and the ancestor order both come from it. Reading
	// mergedInputs() and then ancestorSpecs() separately would walk the chain twice.
	ancestors, err := t.ancestorSpecs(context.Background(), nil)
	merged := t.mergeWithAncestors(ancestors)
	if len(merged) == 0 {
		return nil, err
	}

	out := make([]string, 0, len(merged))
	placed := make(map[string]struct{}, len(merged))
	appendFrom := func(keys []string) {
		for _, k := range keys {
			if _, already := placed[k]; already {
				continue
			}
			if _, declared := merged[k]; !declared {
				continue
			}
			placed[k] = struct{}{}
			out = append(out, k)
		}
	}

	appendFrom(t.spec.OrderedInputKeys())
	for _, ancestor := range ancestors {
		appendFrom(ancestor.OrderedInputKeys())
	}

	// A name reachable through neither ordering cannot occur today — every entry in merged comes
	// from one of those specs, and OrderedInputKeys is total over its own Inputs. Sorting the
	// remainder rather than dropping it keeps that a fact about the data instead of an assumption
	// this function would silently rely on.
	rest := make([]string, 0)
	for k := range merged {
		if _, already := placed[k]; !already {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	return append(out, rest...), err
}

// previewDataWithInputs returns data with every declared input readable under the reserved `input`
// root, applying the same merge contextWithInputs applies — so DryRun previews the document the
// executor would actually render.
//
// It never mutates the caller's map, and it takes the same rule-1 exit: if `input` is already
// occupied by something that is not a binding map, the document is using `input` as an ordinary
// variable name and the preview leaves it exactly as it found it.
func (t *Template) previewDataWithInputs(data map[string]any) map[string]any {
	// Error discarded for the reason contextWithInputs discards it, and with the same one
	// exception: this mirrors injection exactly, and dryRunAST reports an unresolvable chain
	// through its own completeness channel rather than through this map.
	inputs, _ := t.mergedInputs(context.Background(), nil)
	if len(inputs) == 0 {
		return data
	}
	binding, ok := asBindingMap(data[ContextKeyInput])
	if bound, present := data[ContextKeyInput]; present && bound != nil && !ok {
		return data // rule 1
	}

	out := make(map[string]any, len(data)+1)
	for k, v := range data {
		out[k] = v
	}
	out[ContextKeyInput] = mergeInputBinding(binding, inputs)
	return out
}

// asBindingMap normalises the value found under the reserved root. A map[string]string is
// accepted as well as map[string]any because path lookup traverses both and callers do
// produce the narrower one. Anything else — including nil — yields ok=false.
func asBindingMap(val any) (map[string]any, bool) {
	switch m := val.(type) {
	case nil:
		return map[string]any{}, false
	case map[string]any:
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[k] = v
		}
		return out, true
	case map[string]string:
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[k] = v
		}
		return out, true
	default:
		return nil, false
	}
}

// isUnboundInputValue reports whether a caller-supplied value should still receive the
// declared default. A bound empty string counts as unbound, matching how {~exons.env~} treats
// an empty variable: for the form-driven callers this exists for, an untouched field and a
// cleared field are the same gesture.
func isUnboundInputValue(val any) bool {
	if val == nil {
		return true
	}
	s, ok := val.(string)
	return ok && s == ""
}

// satisfiesRequired reports whether a caller's value counts as ANSWERING a required input.
//
// ⛔ IT IS DELIBERATELY NOT isUnboundInputValue, AND THE DIFFERENCE IS A REAL HOLE THAT WAS
// MEASURED. That predicate answers a different question — "should this value still receive the
// declared default?" — and treats only nil and "" as unbound, because for a form-driven caller
// an untouched field and a cleared field are the same gesture. Reused for requiredness it makes
// the check a NO-OP for every collection kind: a caller sending an explicit `[]` for a required
// file-upload, multiselect, sort or associate is "present", nothing else constrains it, and the
// binding validates clean while the rendered prompt still has a hole in it.
//
// ⭐ THAT IS NOT A CORNER CASE: a required `file-upload` is the field type atlas#353 was reported
// against, and `{"papers": []}` is exactly what a stored routine or a curl sends.
//
// An empty collection is therefore NOT an answer to a required field. This matches what every
// form implementation already does — a required file field means "at least one file" — and it is
// scoped to the REQUIRED check alone: binding still uses isUnboundInputValue, so a deliberately
// emptied multiselect keeps its cleared value rather than silently reverting to a default.
func satisfiesRequired(val any) bool {
	if isUnboundInputValue(val) {
		return false
	}
	switch v := val.(type) {
	case []any:
		return len(v) > 0
	case []string:
		return len(v) > 0
	case map[string]any:
		return len(v) > 0
	}
	// ⚠ Reflection is deliberately NOT used for the general case. The values reaching here come
	// from JSON or YAML decoding, whose only composite shapes are the three above; a reflective
	// "is it empty" would additionally start judging caller structs whose emptiness this package
	// has no business defining.
	return true
}

// declaresInput reports whether the given input name is declared anywhere in this template's
// extends chain. A dotted reference (name="user.email" on a structured input) is judged by its
// first segment, since that is the key the frontmatter declares.
//
// It reads mergedInputs for the same reason contextWithInputs does, and that agreement is the
// point: reporting Declared: false for a name the executor happily binds would be one rule with
// two implementations, which is the defect class this whole track removes.
func (t *Template) declaresInput(name string) bool {
	if name == "" {
		return false
	}
	root, _, _ := strings.Cut(name, PathSeparator)
	// Error discarded: this answers a yes/no question with no error channel, and a name found in
	// a partially-walked chain is still genuinely declared. See mergedInputs.
	merged, _ := t.mergedInputs(context.Background(), nil)
	_, ok := merged[root]
	return ok
}

// BindInputs applies this spec's declared defaults to a set of caller-supplied values and
// returns the map to place under the reserved `input` key of Execute's data — so a caller
// never has to hand-roll the namespace or re-implement the default rules:
//
//	data := map[string]any{exons.ContextKeyInput: spec.BindInputs(formValues)}
//
// The returned map is independent of both the spec and the input map. Passing nil is fine and
// yields the defaults alone. Undeclared keys are preserved, not dropped — validating them is
// ValidateInputBinding's job, and silently discarding a caller's value would be worse than
// reporting it.
func (s *Spec) BindInputs(values map[string]any) map[string]any {
	out := make(map[string]any, len(values)+len(s.Inputs))
	for k, v := range values {
		out[k] = deepCopyValue(v)
	}
	for name, def := range s.Inputs {
		if def == nil {
			continue
		}
		if v, present := out[name]; present && !isUnboundInputValue(v) {
			continue
		}
		out[name] = deepCopyValue(def.Default)
	}
	return out
}

// ValidateInputBinding checks caller-supplied values against what the spec declared, and is
// the supported way to enforce a required input.
//
// It runs BEFORE any render, on purpose. Render time is far too late to ask a user for a
// missing value, and a check expressed as a tag attribute would fire or not depending on
// whether the tag sat inside a branch that happened to be taken — the same document would be
// "valid" or not depending on data. That is why {~exons.input~} refuses a `required=`
// attribute and points here instead.
//
// Returns every violation rather than the first, so a form can mark all its bad fields in one
// pass. An empty (or nil) result means the binding is acceptable.
func (s *Spec) ValidateInputBinding(values map[string]any) []error {
	var errs []error
	for _, name := range s.OrderedInputKeys() {
		def := s.Inputs[name]
		if def == nil {
			continue
		}
		val, present := values[name]
		if !present || !satisfiesRequired(val) {
			// A declared default satisfies requiredness — the value is never actually
			// absent at render, so refusing here would make an unsubmittable form.
			if def.Required && def.Default == nil {
				errs = append(errs, fmt.Errorf(ErrFmtInputRequired, name))
			}
			// ⚠ An UNBOUND value skips the per-value checks (there is nothing to check), but
			// an EMPTY COLLECTION is a bound value that simply does not answer a required
			// field — and it has nothing to check either: every per-value rule (option
			// membership, max_files) is vacuous on an empty list.
			continue
		}
		errs = append(errs, validateInputValue(name, def, val)...)
	}
	return errs
}

// validateInputValue checks one bound value against one declaration. Only constraints the
// SPEC states are enforced; the input-kind vocabulary itself stays open, because a document
// may be written against a newer version of this library than the one reading it.
func validateInputValue(name string, def *InputDef, val any) []error {
	var errs []error

	switch def.Type {
	case InputTypeSelect:
		if s, ok := val.(string); ok {
			if !optionAllows(def.Options, s) {
				errs = append(errs, fmt.Errorf(ErrFmtInputNotAnOption, name, s))
			}
		}
	case InputTypeMultiselect, InputTypeSort:
		for _, item := range asStringList(val) {
			if !optionAllows(def.Options, item) {
				errs = append(errs, fmt.Errorf(ErrFmtInputNotAnOption, name, item))
			}
		}
	case InputTypeFileUpload:
		if def.MaxFiles > 0 {
			if n, ok := boundLength(val); ok && n > def.MaxFiles {
				errs = append(errs, fmt.Errorf(ErrFmtInputTooManyFiles, name, n, def.MaxFiles))
			}
		}
	}
	return errs
}

// optionAllows reports whether v is one of the declared option values. An input that declares
// no options constrains nothing — a consumer degrades such a kind to free text, so validating
// against an empty set would reject every value.
func optionAllows(options []InputOption, v string) bool {
	if len(options) == 0 {
		return true
	}
	for _, opt := range options {
		if opt.Value == v {
			return true
		}
	}
	return false
}

// asStringList coerces a bound list value to the strings it contains, ignoring non-string
// elements rather than reporting them: the value vocabulary is open, and an unrecognised
// element shape is not evidence of an option violation.
func asStringList(val any) []string {
	switch v := val.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// boundLength returns the element count of a bound list value.
func boundLength(val any) (int, bool) {
	switch v := val.(type) {
	case []any:
		return len(v), true
	case []string:
		return len(v), true
	default:
		return 0, false
	}
}
