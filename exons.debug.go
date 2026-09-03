package exons

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/itsatony/go-exons/internal"
)

// DryRun output format constants.
const (
	dryRunHeader             = "=== Dry Run Result ===\n"
	dryRunValidLabel         = "Valid: %v\n"
	dryRunVariablesHeader    = "\nVariables (%d):\n"
	dryRunResolversHeader    = "\nResolvers (%d):\n"
	dryRunIncludesHeader     = "\nIncludes (%d):\n"
	dryRunInputsHeader       = "\nInputs (%d):\n"
	dryRunConditionalsHeader = "\nConditionals (%d):\n"
	dryRunSwitchesHeader     = "\nSwitches (%d):\n"
	dryRunLoopsHeader        = "\nLoops (%d):\n"
	dryRunMissingVarsHeader  = "\nMissing Variables (%d):\n"
	dryRunUnusedVarsHeader   = "\nUnused Variables (%d):\n"
	dryRunErrorsHeader       = "\nErrors (%d):\n"
	dryRunWarningsHeader     = "\nWarnings (%d):\n"
	dryRunOutputHeader       = "\n=== Placeholder Output ===\n"
	dryRunListPrefix         = "  - "
	dryRunNewline            = "\n"
)

// DryRun variable status labels.
const (
	dryRunStatusFound      = "found"
	dryRunStatusMissing    = "MISSING"
	dryRunStatusNotFound   = "NOT FOUND"
	dryRunStatusDefault    = "not found (default: %q)"
	dryRunStatusVarLine    = "  - %s [line %d]: %s\n"
	dryRunStatusSuggestion = "    Did you mean: %s?\n"
	dryRunStatusResSummary = "  - %s [line %d]\n"
	dryRunStatusIncLine    = "  - %s [line %d]: %s\n"
	dryRunStatusCondLine   = "  - %s [line %d]\n"
	dryRunStatusInputDecl  = "declared"
	dryRunStatusInputUndec = "UNDECLARED"
	dryRunStatusInputLine  = "  - %s [line %d]: %s\n"
	dryRunStatusSwitchLine = "  - %s [line %d]: %d case(s)\n"
	dryRunStatusLoopFound  = "source found"
	dryRunStatusLoopMiss   = "source NOT FOUND"
	dryRunStatusLoopLine   = "  - for %s in %s [line %d]: %s\n"
)

// DryRun warning format strings.
const (
	dryRunWarnInclude = "line %d: included template '%s' not found"
	dryRunWarnLoopSrc = "line %d: loop source '%s' not found in data"

	// dryRunErrInheritance reports that the analysed AST is the child template alone, because the
	// parent could not be resolved. It is an Error rather than a Warning: the reference collections
	// are INCOMPLETE when it appears, and a consumer must not conclude anything is unreferenced.
	dryRunErrInheritance = "inheritance could not be resolved, analysis covers this template only: %v"

	// dryRunErrNoEngine reports that a template which DOES extend a parent was analysed with no
	// engine to resolve it through, so the parent body was never walked. The damage is identical
	// to dryRunErrInheritance; until v0.23.0 there was no signal at all, and as of v0.24.0
	// EXECUTION refuses it too, so it is reported as invalid rather than merely incomplete.
	dryRunErrNoEngine = "template extends another but has no engine to resolve it, analysis covers this template only"

	// dryRunErrInheritanceInfo reports that the {~exons.extends~} declaration itself could not be
	// read, so analysis proceeded as though the template extends nothing, and the reported
	// references describe a document that is never the one executed. Parse-time extraction still
	// treats it as non-fatal — a template that cannot state its parent can still be inspected —
	// but as of v0.24.0 EXECUTION refuses it, so this too is reported as invalid.
	dryRunErrInheritanceInfo = "inheritance declaration could not be read, analysis assumes no parent: %v"

	// dryRunErrUnknownNode reports an AST node kind the walker does not know. The node AND ITS
	// ENTIRE SUBTREE go unanalysed, which is the widest incompleteness this type can carry.
	dryRunErrUnknownNode = "unknown AST node kind %T, its subtree was not analysed"

	// dryRunErrNilNode reports a nil node reached by the walk. A nil carries no type and no
	// position, so nothing can be said about what was skipped — which is precisely why it must
	// not be skipped silently.
	dryRunErrNilNode = "a nil AST node was reached, the content it stood for was not analysed"

	// dryRunErrMissingName reports a {~exons.var~} or {~exons.input~} carrying no name=
	// attribute. Neither tag requires one to parse. The reference IS still recorded, with an
	// empty Name — a reference site whose target is UNKNOWN, which must never read as a
	// reference to nothing.
	dryRunErrMissingName = "line %d:%d: {~%s~} has no name= attribute, its target is unknown"

	// dryRunErrExpression reports that an eval= expression could not be parsed, so the context
	// paths it references are UNKNOWN rather than absent. Nothing else in this library parses
	// expressions at analysis time — the template parser stores eval= opaquely and
	// Engine.Validate never inspects it — so if this does not report, nothing does.
	dryRunErrExpression = "line %d:%d: %s expression %q could not be parsed, its references are unknown: %v"
)

// Names for the positions an eval= expression can occupy, used by dryRunErrExpression so a
// reader is told WHICH expression failed and not merely that one did.
const (
	dryRunSiteConditionBranch = "conditional branch"
	dryRunSiteSwitch          = "switch"
	dryRunSiteSwitchCase      = "switch case"
)

// Placeholder format strings for dry-run output.
const (
	placeholderVar         = "{{%s}}"
	placeholderInclude     = "{{include:%s}}"
	placeholderTag         = "{{%s}}"
	placeholderIfOpen      = "{{if:%s}}"
	placeholderElseIf      = "{{elseif:%s}}"
	placeholderElse        = "{{else}}"
	placeholderIfClose     = "{{/if}}"
	placeholderForOpen     = "{{for:%s in %s}}"
	placeholderForClose    = "{{/for}}"
	placeholderSwitchOpen  = "{{switch:%s}}"
	placeholderCaseVal     = "{{case:%s}}"
	placeholderCaseEval    = "{{case eval:%s}}"
	placeholderDefault     = "{{default}}"
	placeholderSwitchClose = "{{/switch}}"
)

// Explain output format constants.
const (
	explainHeader       = "=== Template Explanation ===\n"
	explainASTHeader    = "\n--- AST Structure ---\n"
	explainVarsHeader   = "\n--- Variable Accesses ---\n"
	explainTimingHeader = "\n--- Timing ---\n"
	explainErrorHeader  = "\n--- Error ---\n  %v\n"
	explainOutputHeader = "\n--- Output ---\n"
	explainTimingTotal  = "  Total: %v\n"
	explainTimingExec   = "  Execution: %v\n"
	explainVarNotFound  = "NOT FOUND"
	explainVarDefault   = "not found, using default: %q"
	explainVarValue     = "= %v"
	explainVarLine      = "  [line %d] %s: %s\n"
)

// AST format constants.
const (
	astIndentUnit = "  "
	astRootLabel  = "%sRoot\n"
	astTextLabel  = "%sText: %q\n"
	astTagLabel   = "%sTag: %s"
	astBlockLabel = "%sBlock: %s (line %d)\n"
	astAttrsFmt   = " [%s]"
	astAttrPair   = "%s=%q"
	astLineFmt    = " (line %d)\n"
	astCondFmt    = "%sConditional: %s (line %d)\n"
	astThenLabel  = "%s  Then:\n"
	astElseLabel  = "%s  Else:\n"
	astElseIfFmt  = "%s  ElseIf: %s\n"
	astForFmt     = "%sFor: %s in %s"
	astForIndex   = " (index: %s)"
	astForLimit   = " (limit: %d)"
	astSwitchFmt  = "%sSwitch: %s (line %d)\n"
	astCaseVal    = "%s  Case: %s\n"
	astCaseEval   = "%s  Case eval: %s\n"
	astDefaultLbl = "%s  Default:\n"
)

// AST text truncation constants.
const (
	astTextMaxLen     = 40
	astTextTruncation = "..."
	astNewlineEscape  = "\\n"
	astNewlineChar    = "\n"
)

// Levenshtein similarity threshold.
const (
	levenshteinMaxSuggestions = 3
	levenshteinThresholdExtra = 2
)

// DryRunResult contains the results of a dry-run execution.
// Dry-run validates the template structure without executing resolvers.
//
// # The reference-completeness contract
//
// As of v0.22.0 the reference collections below are COMPLETE for statically-analysable references:
// every position in the template where a context path can be named is reported. This is the
// contract a consumer needs before it can safely conclude that a name is referenced NOWHERE — a
// conclusion that licenses telling an author their declaration is dead. Under-reporting turns that
// advice into an instruction to delete working code, so the guarantee is stated rather than
// implied, and exons.debug.completeness_test.go pins each clause.
//
// Reported positions: {~exons.var~} and {~exons.input~} anywhere, including nested inside block
// tags to arbitrary depth; every branch of a conditional including elseif and else; a switch's
// dispatch expression and every case arm's eval=; a loop's in= source; an include's attributes;
// any custom resolver's attributes; and all of the above inside {~exons.block~} bodies and inside
// the parent body a {~exons.extends~} child splices into.
//
// Deliberately NOT reported, each failing in the SAFE direction:
//
//   - A reference inside {~exons.raw~}, {~exons.comment~} or a {~~ ~~} fence. These spans are
//     consumed at the lexer and never become nodes. Correct: they are literal text, not references.
//   - Names bound by a loop variable are not distinguished from document inputs. An in-loop
//     shadow therefore reads as a reference, which over-reports USE and never under-reports it.
//   - Includes are not recursed. The include boundary does not propagate the caller's reserved
//     input root — buildChildData passes only the with= expansion and literal attributes — so a
//     parent's declared inputs are structurally unreachable inside an included template. The one
//     cross-boundary path, with="input.foo", is carried verbatim in IncludeReference.Attributes.
//   - A non-grammar attribute on a CONTROL-FLOW tag. parseConditional reads only eval=, parseFor
//     only item/in/index/limit, parseSwitch only eval= and parseSwitchCase only value=/eval=;
//     everything else on those tags is discarded before an AST node exists. So unlike a resolver
//     tag, this library structurally CANNOT report a future path-bearing attribute on
//     {~exons.if~}. Today that is vacuous because no such attribute exists — it is recorded here
//     so that adding one is understood to be a grammar change, not an additive one.
//
// The one case that is NOT closable by this library: a third-party resolver may define an
// attribute whose value names a context path, and this library cannot know that attribute's
// semantics. Every resolver's attributes are reported in full for exactly this reason. A consumer
// performing unreferenced analysis should treat any attribute value containing a reference to a
// name as a use of it — over-reporting use, never under-reporting it. As of v0.23.0 the same
// applies to {~exons.var~} and {~exons.input~}, whose attributes were computed and then dropped.
//
// Finally, and most importantly: the guarantee above holds only for a walk that REACHED
// everything. Call AnalysisComplete before concluding that any name is unreferenced.
type DryRunResult struct {
	// Valid reports that DryRun has POSITIVE PROOF ExecuteWithContext would fail on this
	// template. Today the only such proof is unresolvable inheritance, which
	// Template.ExecuteWithContext returns on unconditionally.
	//
	// Valid is NOT the completeness gate — use AnalysisComplete for that. Until v0.23.0 it was
	// derived from len(Errors), so it carried no information of its own and every new reason to
	// report an incomplete analysis silently widened the claim "this template is invalid". The
	// two are genuinely different questions: a template with an unparseable eval= in a branch it
	// never takes renders perfectly, and DryRun says so by returning Valid true and
	// AnalysisComplete false.
	Valid bool

	// Output is the template with placeholders for dynamic content
	Output string

	// Variables lists all variable references found in the template
	Variables []VariableReference

	// Resolvers lists all resolver invocations found in the template
	Resolvers []ResolverReference

	// Inputs lists all DECLARED-INPUT references ({~exons.input~}) found in the template.
	//
	// This is deliberately separate from Variables. An exons.var reference names an
	// arbitrary runtime context path and may legitimately resolve to anything the caller
	// supplies; an exons.input reference names something the DOCUMENT declared, so a
	// consumer can check it against the frontmatter and be right. Folding the two together
	// would throw away exactly the distinction the verb exists to create.
	Inputs []InputReference

	// Includes lists all template includes found
	Includes []IncludeReference

	// Conditionals lists all conditional blocks found
	Conditionals []ConditionalReference

	// Switches lists all {~exons.switch~} blocks found.
	//
	// Switches were invisible to DryRun entirely until v0.22.0 — neither the switch expression nor
	// any case's eval= or value= was recorded anywhere, so a document branching on a context path
	// through a switch reported that path as referenced nowhere. That is the shape of miss which
	// turns an advisory into a false accusation, which is why this is its own reported collection
	// rather than a flag on something else.
	Switches []SwitchReference

	// Loops lists all loop blocks found
	Loops []LoopReference

	// Errors is the ANALYSIS-COMPLETENESS channel: each entry names a place the walk could not
	// reach, and therefore a region of the template whose references are UNKNOWN rather than
	// absent.
	//
	// It is written through exactly one function, reportIncomplete, and
	// TestDryRunErrorsHaveExactlyOneWriter parses this package to keep it that way. The single
	// writer is the point rather than tidiness: this is the channel a consumer gates on before
	// concluding that a declared name is referenced NOWHERE — a conclusion whose remedy is to
	// delete an author's declaration — and a channel written from scattered points across a
	// recursive walk is one nobody can audit for completeness.
	//
	// Prefer AnalysisComplete to inspecting the length.
	Errors []string

	// Warnings contains non-fatal issues
	Warnings []string

	// MissingVariables lists variables that are referenced but not in data
	MissingVariables []string

	// UnusedVariables lists variables in data that are not referenced
	UnusedVariables []string
}

// AnalysisComplete reports whether the walk reached every part of the template — that is,
// whether the reference collections may be treated as exhaustive.
//
// This is the check to make BEFORE concluding that a declared name is referenced nowhere. That
// conclusion licenses telling an author to delete a declaration, so it may only be drawn from a
// complete walk; when this returns false some region went unanalysed and its references are
// unknown, not absent. Errors carries the reasons.
//
// It is deliberately not Valid. A template can be perfectly executable and still be incompletely
// analysable, and the two are reported separately for that reason.
func (r *DryRunResult) AnalysisComplete() bool {
	return len(r.Errors) == 0
}

// reportIncomplete records that the walk could not analyse something, so the reference
// collections are INCOMPLETE.
//
// This is the ONLY writer of Errors in this package, and TestDryRunErrorsHaveExactlyOneWriter
// parses the package's own source to keep it that way. See the Errors field for why a single
// auditable door matters more here than it would for an ordinary diagnostic list.
//
// It does NOT touch Valid. Incompleteness of the ANALYSIS and invalidity of the TEMPLATE are
// different claims; use reportIncompleteAndInvalid where one condition proves both.
func (r *DryRunResult) reportIncomplete(format string, args ...any) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, args...))
}

// reportIncompleteAndInvalid records an analysis-completeness failure whose cause ALSO proves
// that ExecuteWithContext would return an error for this template.
//
// Use it only where execution failure is CERTAIN, not merely possible. A malformed eval= in a
// branch that is never taken renders perfectly, and reporting such a template invalid is a false
// claim with no remedy attached. Today the only qualifying condition is unresolvable
// inheritance, which Template.ExecuteWithContext fails on unconditionally.
func (r *DryRunResult) reportIncompleteAndInvalid(format string, args ...any) {
	r.reportIncomplete(format, args...)
	r.Valid = false
}

// VariableReference represents a variable reference in a template.
type VariableReference struct {
	Name    string // Variable path (e.g., "user.name")
	Default string // Default value if specified

	// Attributes is the tag's full attribute map, verbatim.
	//
	// It is carried for the same reason ResolverReference and IncludeReference carry theirs: a
	// consumer performing unreferenced analysis must treat any attribute value naming a context
	// path as a USE of that name, and it cannot do so for attributes this library never handed
	// it. Today no attribute on {~exons.var~} names a path — name= is reported as Name, and
	// join= and format= take literals — so this is a guarantee held in reserve rather than one
	// currently load-bearing. It was computed and then dropped until v0.23.0, which is the shape
	// of asymmetry that becomes a false accusation the day an attribute does take a path.
	Attributes map[string]string

	Line        int      // Source line number
	Column      int      // Source column number
	HasDefault  bool     // Whether a default was specified
	InData      bool     // Whether the variable exists in provided data
	Suggestions []string // Similar variable names if not found
}

// ResolverReference represents a resolver invocation in a template.
type ResolverReference struct {
	TagName    string            // Resolver tag name
	Attributes map[string]string // Attributes passed to resolver
	Line       int               // Source line number
	Column     int               // Source column number
	Registered bool              // Whether resolver is registered
}

// InputReference represents a {~exons.input~} reference to a declared input.
//
// It is the sound answer to "which of this document's declared inputs does its body actually
// reference?" — sound because it comes from the parsed AST rather than from re-scanning the
// source. A consumer re-scanning source has to agree byte-for-byte with this library's lexer
// about how a reference is spelled, and gets it wrong for hyphenated names, names containing
// a quote or a backslash, tags spanning several lines, and tildes inside attribute values.
//
// Note also what CANNOT appear here: a reference inside {~exons.raw~}, {~exons.comment~} or a
// {~~ ~~} fence. Those spans are consumed at the LEXER, so no tag node is ever built for
// their contents — the exclusion is structural rather than a rule someone remembered to add.
type InputReference struct {
	Name    string // Declared input name, from the name= attribute
	Default string // Tag-level default= fallback, if specified

	// Attributes is the tag's full attribute map, verbatim. See VariableReference.Attributes for
	// why it is carried even though no attribute on {~exons.input~} names a context path today.
	Attributes map[string]string

	Line       int  // Source line number
	Column     int  // Source column number
	HasDefault bool // Whether a tag-level default= was specified

	// Declared reports whether the name appears in the frontmatter `inputs:` block.
	//
	// KNOWN LIMIT, and it errs toward accusation: this is answered from THIS template's spec,
	// while the walked AST is the inheritance-RESOLVED one and therefore contains the parent's
	// {~exons.input~} nodes too. An input declared by a parent and referenced in the parent body
	// reports Declared false — "the author mistyped this" — when the truth is "declared one
	// document up". Resolving parent specs is a separate design; until then a consumer treating
	// Declared false as a defect should require AnalysisComplete AND a template that does not
	// extend.
	Declared bool
}

// IncludeReference represents a template include in a template.
type IncludeReference struct {
	TemplateName string            // Name of included template
	Attributes   map[string]string // Additional attributes
	Line         int               // Source line number
	Column       int               // Source column number
	Exists       bool              // Whether template is registered
	Isolated     bool              // Whether isolate="true"
}

// ConditionalReference represents a conditional block in a template.
type ConditionalReference struct {
	Condition string // The eval expression of the FIRST branch; see Branches for the rest

	// Identifiers are the context paths referenced by Condition, resolved by this library's own
	// expression parser rather than by scanning the string. Convenience mirror of
	// Branches[0].Identifiers.
	Identifiers []string

	// Branches carries EVERY branch of the conditional, in source order, including elseif and
	// else. Before v0.22.0 only the first branch's condition survived into this struct — the rest
	// were reduced to the two booleans below — so a path referenced only in an elseif was reported
	// as referenced nowhere. Condition/HasElseIf/HasElse are retained unchanged for compatibility;
	// Branches is the complete answer.
	Branches []ConditionalBranchRef

	Line      int  // Source line number
	Column    int  // Source column number
	HasElseIf bool // Whether it has elseif branches
	HasElse   bool // Whether it has an else branch
}

// ConditionalBranchRef is one branch of a conditional — an if, an elseif, or an else.
//
// Until v0.23.0 this said branches carry no source position of their own. They always did; the
// position was simply WRONG for every branch after the first, because parseConditional built each
// one with the position of the tag that TERMINATED it — and the last branch with the position of
// the closing {~/exons.if~}. That same field is what executeConditional reports a failing branch
// expression at, so the defect was reaching users as wrong line numbers in runtime errors long
// before it reached this struct. Both are fixed; Line/Column below are the branch's own tag.
type ConditionalBranchRef struct {
	Condition   string   // The eval expression; empty for the else branch
	Identifiers []string // Context paths referenced by Condition, resolved by the expression parser
	IsElse      bool     // True for the final else branch
	Line        int      // Source line of this branch's own if/elseif/else tag
	Column      int      // Source column of this branch's own if/elseif/else tag
}

// SwitchReference represents a {~exons.switch~} block in a template.
type SwitchReference struct {
	Expression  string   // The eval= expression the switch dispatches on
	Identifiers []string // Context paths referenced by Expression

	Cases      []SwitchCaseRef // Every case arm, in source order
	HasDefault bool            // Whether a default arm is present

	Line   int // Source line number
	Column int // Source column number
}

// SwitchCaseRef is one arm of a switch.
//
// Value and Eval are mutually exclusive in the grammar: a case either compares the switch value
// against a literal string, or evaluates its own boolean expression. Only Eval can reference a
// context path, so Identifiers is always derived from Eval and is empty for a value-comparison
// arm — a literal is a literal, not a reference.
type SwitchCaseRef struct {
	Value       string   // Literal compared against the switch value
	Eval        string   // Boolean expression evaluated for this arm
	Identifiers []string // Context paths referenced by Eval
	Line        int      // Source line of this case's own tag
	Column      int      // Source column of this case's own tag
}

// LoopReference represents a loop block in a template.
type LoopReference struct {
	ItemVar  string // Loop item variable name
	IndexVar string // Loop index variable name
	Source   string // Source collection path
	Line     int    // Source line number
	Column   int    // Source column number
	Limit    int    // Loop limit if specified
	InData   bool   // Whether source exists in data
}

// ExplainResult contains detailed execution explanation.
type ExplainResult struct {
	// AST is a human-readable representation of the parsed AST
	AST string

	// Steps contains the execution steps in order
	Steps []ExecutionStep

	// Variables shows all variable accesses during execution
	Variables []VariableAccess

	// Resolvers shows all resolver invocations
	Resolvers []ResolverInvocation

	// Timing contains execution timing information
	Timing ExecutionTiming

	// Output is the final rendered output
	Output string

	// Error is set if execution failed
	Error error
}

// ExecutionStep represents a single step in template execution.
type ExecutionStep struct {
	StepNumber  int           // Step number (1-based)
	Type        string        // Step type (text, variable, resolver, conditional, loop, include)
	Description string        // Human-readable description
	Input       string        // Input to this step
	Output      string        // Output from this step
	Duration    time.Duration // Time taken for this step
	Line        int           // Source line number
	Column      int           // Source column number
}

// VariableAccess records a variable access during execution.
type VariableAccess struct {
	Path    string // Variable path accessed
	Value   any    // Value retrieved (or nil if not found)
	Found   bool   // Whether the variable was found
	Default string // Default value used (if any)
	Line    int    // Source line number
	Column  int    // Source column number
}

// ResolverInvocation records a resolver invocation during execution.
type ResolverInvocation struct {
	TagName    string            // Resolver tag name
	Attributes map[string]string // Attributes passed
	Output     string            // Output produced
	Error      error             // Error if any
	Duration   time.Duration     // Time taken
	Line       int               // Source line number
	Column     int               // Source column number
}

// ExecutionTiming contains timing information for execution.
type ExecutionTiming struct {
	Total        time.Duration // Total execution time
	Parsing      time.Duration // Time spent parsing
	Execution    time.Duration // Time spent executing
	ResolverTime time.Duration // Total time in resolvers
	VariableTime time.Duration // Total time resolving variables
}

// DryRun performs a dry-run of the template without executing resolvers.
// It validates the template structure and reports all dynamic elements.
func (t *Template) DryRun(ctx context.Context, data map[string]any) *DryRunResult {
	result := &DryRunResult{
		Valid:            true,
		Variables:        make([]VariableReference, 0),
		Resolvers:        make([]ResolverReference, 0),
		Inputs:           make([]InputReference, 0),
		Includes:         make([]IncludeReference, 0),
		Conditionals:     make([]ConditionalReference, 0),
		Switches:         make([]SwitchReference, 0),
		Loops:            make([]LoopReference, 0),
		Errors:           make([]string, 0),
		Warnings:         make([]string, 0),
		MissingVariables: make([]string, 0),
		UnusedVariables:  make([]string, 0),
	}

	// Track which data keys are used
	usedKeys := make(map[string]bool)

	// Collect available keys for suggestions
	availableKeys := collectAllKeys(data, "")

	// Walk the AST and collect references.
	//
	// The walk uses the INHERITANCE-RESOLVED AST, matching what ExecuteWithContext runs. A template
	// that extends another is, before resolution, a handful of block definitions with none of the
	// parent body around them — so analysing the raw AST reports the references of a document that
	// is never the one executed.
	astToWalk := t.dryRunAST(ctx, result)
	t.walkASTForDryRun(astToWalk, data, result, usedKeys, availableKeys)

	// Find missing variables
	missingSet := make(map[string]bool)
	for _, v := range result.Variables {
		if !v.InData && !v.HasDefault {
			missingSet[v.Name] = true
		}
	}
	for name := range missingSet {
		result.MissingVariables = append(result.MissingVariables, name)
	}
	sort.Strings(result.MissingVariables)

	// Find unused variables
	for _, key := range availableKeys {
		if !usedKeys[key] {
			// Only report top-level unused keys
			if !strings.Contains(key, PathSeparator) {
				result.UnusedVariables = append(result.UnusedVariables, key)
			}
		}
	}
	sort.Strings(result.UnusedVariables)

	// Generate placeholder output from the same AST that was analysed, so the reported references
	// and the reported output can never describe two different documents.
	//
	// The PREVIEW data carries the declared input defaults; the ANALYSIS data above deliberately
	// does not. A declared default is a fact about the document, so a preview showing "{{tone}}"
	// where the render produces "friendly" is simply wrong — and it was, until this release.
	// Seeding the analysis map instead would be a different and worse change: availableKeys is
	// derived from it, so every declared input would begin reporting as an UNUSED caller-supplied
	// variable. Nothing on the analysis side reads data for an input — InputReference has no
	// InData field, it has Declared — so this is the one site that needs it.
	result.Output = t.generatePlaceholderOutput(astToWalk, t.previewDataWithInputs(data))

	// Valid is deliberately NOT derived from Errors here.
	//
	// It used to be, which meant Valid carried no information of its own and every reason to
	// report an incomplete ANALYSIS silently restated itself as a claim that the TEMPLATE is
	// invalid. The writers now decide: reportIncompleteAndInvalid where the cause proves
	// execution would fail, reportIncomplete where it only proves the walk stopped short.
	return result
}

// dryRunAST returns the AST that DryRun should analyse — inheritance-resolved when the template
// extends another, and the raw AST otherwise.
//
// It mirrors ExecuteWithContext deliberately. Analysis that walks a different AST than execution
// walks is not analysis of the document that runs.
//
// Resolution failure is reported and then TOLERATED: the raw AST is analysed instead. DryRun's
// purpose is to tell an author what is in their template, and degrading to a partial answer with
// the reason attached is more useful than returning nothing. Callers deciding anything destructive
// from the reference collections must check Errors first — a resolution failure means the
// collections describe only the child, so a name used solely in the parent body is absent.
// It shares ONE resolution helper with ExecuteWithContext and Explain — see resolveInheritance.
// This function's whole job is now to translate an outcome into this channel's vocabulary.
func (t *Template) dryRunAST(ctx context.Context, result *DryRunResult) any {
	// nil execution context: DryRun holds a data map, not a *Context, so the parent comes from the
	// engine registry exactly as before per-call resolvers existed.
	resolvedAST, outcome, err := t.resolveInheritance(ctx, nil)

	switch outcome {
	case inheritanceNone, inheritanceResolved:
		return resolvedAST

	case inheritanceUnreadable:
		// Parse keeps an unreadable extends non-fatal because a template that cannot state its
		// parent can still be inspected. For ANALYSIS it means the collections below describe a
		// document that is never the one executed — exactly the claim this channel withholds.
		//
		// Now reportIncompleteAndInvalid rather than reportIncomplete: as of this release
		// ExecuteWithContext returns this same error, so it proves the template cannot run.
		result.reportIncompleteAndInvalid(dryRunErrInheritanceInfo, t.inheritanceErr)
		return resolvedAST

	case inheritanceNoEngine:
		// Extends, but no engine to resolve the parent through. The damage is identical to an
		// outright resolution failure — the parent body is never walked — and this too is now
		// fatal at execute, so it is reported as invalid rather than merely incomplete.
		result.reportIncompleteAndInvalid(dryRunErrNoEngine)
		return resolvedAST

	default: // inheritanceFailed
		// The condition that has always proved both: ExecuteWithContext returns this same error.
		result.reportIncompleteAndInvalid(dryRunErrInheritance, err)
		return resolvedAST
	}
}

// dryRunASTNodeKinds is the number of AST node kinds walkASTForDryRun handles explicitly. Like
// exprIdentifierNodeKinds it exists to make adding an eighth kind a deliberate act rather than a
// silent completeness hole — TestDryRunWalkerCoversEveryASTNodeKind asserts this number and names
// the kinds, so a reader who adds one is told where to add the case.
//
// Be honest about what it cannot do: a new kind declared in package internal will not fail this
// test by itself, because the AST types are unexported and cannot be enumerated from here. The
// default arm below is what makes such a kind LOUD at runtime; this constant is what makes
// handling it a decision someone wrote down.
const dryRunASTNodeKinds = 7

// walkASTForDryRun recursively walks the AST to collect dry-run information.
func (t *Template) walkASTForDryRun(node any, data map[string]any, result *DryRunResult, usedKeys map[string]bool, availableKeys []string) {
	// A nil carries neither a type to name nor a position to point at, so nothing can be said
	// about the content it stood for. That is the reason to report it, not a reason to skip it.
	//
	// This catches an UNTYPED nil only. A typed nil — (*internal.RootNode)(nil) boxed in an any —
	// is not == nil, matches its type case below, and panics on the first field access. No
	// producer reaches that state today: the parser never appends a nil child, and every nil
	// return from ResolveInheritance is paired with a non-nil error. A stated limit, not a hole
	// left open.
	if node == nil {
		result.reportIncomplete(dryRunErrNilNode)
		return
	}

	switch n := node.(type) {
	case *internal.RootNode:
		for _, child := range n.Children {
			t.walkASTForDryRun(child, data, result, usedKeys, availableKeys)
		}

	case *internal.TagNode:
		t.processTagNodeForDryRun(n, data, result, usedKeys, availableKeys)

	case *internal.ConditionalNode:
		t.processConditionalNodeForDryRun(n, data, result, usedKeys, availableKeys)

	case *internal.ForNode:
		t.processForNodeForDryRun(n, data, result, usedKeys, availableKeys)

	case *internal.SwitchNode:
		t.processSwitchNodeForDryRun(n, data, result, usedKeys, availableKeys)

	case *internal.BlockNode:
		// A block's body is ordinary template content and the executor executes it
		// (executeBlockNode). Omitting this case meant every reference inside every
		// {~exons.block~} was invisible to analysis.
		for _, child := range n.Children {
			t.walkASTForDryRun(child, data, result, usedKeys, availableKeys)
		}

	case *internal.TextNode:
		// Literal output. It references nothing, so there is nothing to collect — but the case
		// must be WRITTEN rather than left to fall off the end of the switch, because the
		// default arm below now treats an unhandled kind as a completeness failure. This is the
		// one node kind whose silence is correct.

	default:
		// Any kind this walker does not know, and its ENTIRE SUBTREE, would otherwise vanish
		// without a trace — the widest incompleteness this type can carry, and until v0.23.0 the
		// switch simply had no default. Reporting the concrete type is what makes the omission
		// findable; dryRunASTNodeKinds is what makes adding a kind a deliberate act.
		//
		// It is not proof the template cannot RUN: a kind added upstream would reach the
		// executor, which knows it, before it reached this switch, which does not.
		result.reportIncomplete(dryRunErrUnknownNode, node)
	}
}

// processTagNodeForDryRun processes a tag node for dry-run.
func (t *Template) processTagNodeForDryRun(n *internal.TagNode, data map[string]any, result *DryRunResult, usedKeys map[string]bool, availableKeys []string) {
	attrs := n.Attributes.Map()
	pos := n.Pos()
	line := pos.Line
	col := pos.Column

	switch n.Name {
	case TagNameVar:
		varName, hasName := n.Attributes.Get(AttrName)
		if !hasName {
			// Nothing in the parser or in Engine.Validate requires name=, so {~exons.var /~}
			// reaches analysis intact. The reference is still recorded below — over-reporting a
			// use is the safe direction — but its target is UNKNOWN, and an empty Name would
			// otherwise be indistinguishable from a reference to nothing.
			result.reportIncomplete(dryRunErrMissingName, line, col, n.Name)
		}
		defaultVal := n.Attributes.GetDefault(AttrDefault, "")
		hasDefault := n.Attributes.Has(AttrDefault)

		// Check if variable exists in data
		inData := hasPath(data, varName)
		if inData {
			markKeyUsed(usedKeys, varName)
		}

		// Find suggestions if not found
		var suggestions []string
		if !inData && !hasDefault {
			suggestions = findSimilarStrings(varName, availableKeys, levenshteinMaxSuggestions)
		}

		result.Variables = append(result.Variables, VariableReference{
			Name:        varName,
			Default:     defaultVal,
			Attributes:  attrs,
			Line:        line,
			Column:      col,
			HasDefault:  hasDefault,
			InData:      inData,
			Suggestions: suggestions,
		})

	case TagNameInclude:
		tmplName, _ := n.Attributes.Get(AttrTemplate)
		isolated := n.Attributes.GetDefault(AttrIsolate, "") == AttrValueTrue

		// Check if template exists
		exists := false
		if t.engine != nil {
			exists = t.engine.HasTemplate(tmplName)
		}

		result.Includes = append(result.Includes, IncludeReference{
			TemplateName: tmplName,
			Attributes:   attrs,
			Line:         line,
			Column:       col,
			Exists:       exists,
			Isolated:     isolated,
		})

		if !exists {
			result.Warnings = append(result.Warnings, fmt.Sprintf(dryRunWarnInclude, line, tmplName))
		}

	case TagNameInput:
		inputName, hasName := n.Attributes.Get(AttrName)
		if !hasName {
			// See the exons.var arm: name= is not required to parse, and an empty Name must not
			// read as a reference to nothing.
			result.reportIncomplete(dryRunErrMissingName, line, col, n.Name)
		}
		result.Inputs = append(result.Inputs, InputReference{
			Name:       inputName,
			Default:    n.Attributes.GetDefault(AttrDefault, ""),
			Attributes: attrs,
			Line:       line,
			Column:     col,
			HasDefault: n.Attributes.Has(AttrDefault),
			Declared:   t.declaresInput(inputName),
		})

	case TagNameRaw, TagNameComment:
		// No action needed for raw/comment

	default:
		// Custom resolver.
		//
		// Registered is ASKED, not assumed. It used to be hardcoded true with the comment
		// "assume registered since it parsed" — but the parser never consults the resolver
		// registry (any well-formed tag name parses), so a typo'd or genuinely unregistered
		// verb was reported as registered, and the one field a caller would use to catch it
		// always said everything was fine.
		//
		// An unresolved {~exons.extends~} or {~exons.parent~} also lands here and is reported as
		// an unregistered resolver that does not exist. That is left deliberately: it
		// OVER-reports, the safe direction, and whenever it happens dryRunErrInheritance or
		// dryRunErrNoEngine is already in Errors explaining the real cause.
		result.Resolvers = append(result.Resolvers, ResolverReference{
			TagName:    n.Name,
			Attributes: attrs,
			Line:       line,
			Column:     col,
			Registered: t.executor != nil && t.executor.HasResolver(n.Name),
		})
	}

	// Recurse into a BLOCK tag's children.
	//
	// This is the single most consequential completeness fix in DryRun's history, and it is one
	// line of intent. A tag may be self-closing ({~exons.var /~}, no children) or a block
	// ({~exons.message~}…{~/exons.message~}, children the executor executes — see executeTagNode).
	// Until v0.22.0 this function reported the tag itself and never descended, so
	//
	//	{~exons.message role="user"~}{~exons.input name="tone" /~}{~/exons.message~}
	//
	// produced ZERO input references. exons.message wraps essentially every real prompt body, so
	// the miss was not an edge case — it was the common case, and any consumer concluding
	// "referenced nowhere" from that would have been wrong about almost every document it saw.
	//
	// raw and comment are excluded because their spans are consumed at the LEXER: raw keeps its
	// body in RawContent as literal text and comment keeps nothing. Neither has parsed children,
	// so the exclusion is a statement of intent rather than a guard against anything reachable —
	// and it is the structural reason a reference inside {~exons.raw~} is correctly never reported.
	if n.Name != TagNameRaw && n.Name != TagNameComment {
		for _, child := range n.Children {
			t.walkASTForDryRun(child, data, result, usedKeys, availableKeys)
		}
	}
}

// processConditionalNodeForDryRun processes a conditional node for dry-run.
func (t *Template) processConditionalNodeForDryRun(n *internal.ConditionalNode, data map[string]any, result *DryRunResult, usedKeys map[string]bool, availableKeys []string) {
	pos := n.Pos()

	// Capture EVERY branch, not just the first.
	//
	// The previous implementation reduced branches 1..n to two booleans, which threw away every
	// elseif condition. A path referenced only in an elseif was then reported as referenced
	// nowhere — the exact miss that turns "declared but unreferenced" advice into a false
	// instruction to delete working code.
	hasElseIf := false
	hasElse := false
	firstCondition := ""
	branches := make([]ConditionalBranchRef, 0, len(n.Branches))

	for i, branch := range n.Branches {
		if i == 0 {
			firstCondition = branch.Condition
		} else if branch.IsElse {
			hasElse = true
		} else {
			hasElseIf = true
		}

		branchPos := branch.Pos
		branches = append(branches, ConditionalBranchRef{
			Condition: branch.Condition,
			Identifiers: result.expressionIdentifiers(
				branch.Condition, dryRunSiteConditionBranch, branchPos.Line, branchPos.Column,
			),
			IsElse: branch.IsElse,
			Line:   branchPos.Line,
			Column: branchPos.Column,
		})
	}

	firstIdentifiers := []string{}
	if len(branches) > 0 {
		firstIdentifiers = branches[0].Identifiers
	}

	result.Conditionals = append(result.Conditionals, ConditionalReference{
		Condition:   firstCondition,
		Identifiers: firstIdentifiers,
		Branches:    branches,
		Line:        pos.Line,
		Column:      pos.Column,
		HasElseIf:   hasElseIf,
		HasElse:     hasElse,
	})

	// Walk all branches
	for _, branch := range n.Branches {
		for _, child := range branch.Children {
			t.walkASTForDryRun(child, data, result, usedKeys, availableKeys)
		}
	}
}

// processForNodeForDryRun processes a for node for dry-run.
func (t *Template) processForNodeForDryRun(n *internal.ForNode, data map[string]any, result *DryRunResult, usedKeys map[string]bool, availableKeys []string) {
	pos := n.Pos()
	inData := hasPath(data, n.Source)
	if inData {
		markKeyUsed(usedKeys, n.Source)
	}

	result.Loops = append(result.Loops, LoopReference{
		ItemVar:  n.ItemVar,
		IndexVar: n.IndexVar,
		Source:   n.Source,
		Line:     pos.Line,
		Column:   pos.Column,
		Limit:    n.Limit,
		InData:   inData,
	})

	if !inData {
		result.Warnings = append(result.Warnings, fmt.Sprintf(dryRunWarnLoopSrc, pos.Line, n.Source))
	}

	// Walk body (with loop variables added conceptually)
	for _, child := range n.Children {
		t.walkASTForDryRun(child, data, result, usedKeys, availableKeys)
	}
}

// processSwitchNodeForDryRun processes a switch node for dry-run.
func (t *Template) processSwitchNodeForDryRun(n *internal.SwitchNode, data map[string]any, result *DryRunResult, usedKeys map[string]bool, availableKeys []string) {
	pos := n.Pos()

	// Record the switch itself. Until v0.22.0 this function walked case bodies and recorded
	// nothing about the switch — not the dispatch expression, not a single case — so a document
	// branching on a context path through a switch reported that path nowhere at all.
	cases := make([]SwitchCaseRef, 0, len(n.Cases))
	for _, c := range n.Cases {
		casePos := c.Pos
		cases = append(cases, SwitchCaseRef{
			Value: c.Value,
			Eval:  c.Eval,
			// Only Eval is an expression; Value is a literal compared against the switch value.
			Identifiers: result.expressionIdentifiers(
				c.Eval, dryRunSiteSwitchCase, casePos.Line, casePos.Column,
			),
			Line:   casePos.Line,
			Column: casePos.Column,
		})
	}

	result.Switches = append(result.Switches, SwitchReference{
		Expression: n.Expression,
		Identifiers: result.expressionIdentifiers(
			n.Expression, dryRunSiteSwitch, pos.Line, pos.Column,
		),
		Cases:      cases,
		HasDefault: n.Default != nil,
		Line:       pos.Line,
		Column:     pos.Column,
	})

	// Walk all cases
	for _, c := range n.Cases {
		for _, child := range c.Children {
			t.walkASTForDryRun(child, data, result, usedKeys, availableKeys)
		}
	}
	if n.Default != nil {
		for _, child := range n.Default.Children {
			t.walkASTForDryRun(child, data, result, usedKeys, availableKeys)
		}
	}
}

// generatePlaceholderOutput generates output with placeholders for dynamic content.
func (t *Template) generatePlaceholderOutput(node any, data map[string]any) string {
	var sb strings.Builder
	t.generatePlaceholders(node, data, &sb)
	return sb.String()
}

// generatePlaceholders recursively generates placeholder output.
func (t *Template) generatePlaceholders(node any, data map[string]any, sb *strings.Builder) {
	switch n := node.(type) {
	case *internal.RootNode:
		for _, child := range n.Children {
			t.generatePlaceholders(child, data, sb)
		}

	case *internal.TextNode:
		sb.WriteString(n.Content)

	case *internal.TagNode:
		switch n.Name {
		case TagNameVar:
			varName, _ := n.Attributes.Get(AttrName)
			defaultVal := n.Attributes.GetDefault(AttrDefault, "")

			// Try to get actual value
			if val, ok := getPath(data, varName); ok {
				fmt.Fprintf(sb, "%v", val)
			} else if defaultVal != "" {
				sb.WriteString(defaultVal)
			} else {
				fmt.Fprintf(sb, placeholderVar, varName)
			}

		case TagNameInput:
			// Mirrors the exons.var arm, reading through the reserved input root.
			//
			// Without this case an input fell to the default arm and previewed as the literal
			// "{{exons.input}}" — every input in the document rendering as the same opaque
			// token, with the one thing an author needs to see, the NAME, discarded. A preview
			// that cannot tell two inputs apart is not a preview of that document.
			inputName, _ := n.Attributes.Get(AttrName)
			defaultVal := n.Attributes.GetDefault(AttrDefault, "")

			if val, ok := getPath(data, ContextKeyInput+PathSeparator+inputName); ok {
				fmt.Fprintf(sb, "%v", val)
			} else if defaultVal != "" {
				sb.WriteString(defaultVal)
			} else {
				fmt.Fprintf(sb, placeholderVar, inputName)
			}

		case TagNameInclude:
			tmplName, _ := n.Attributes.Get(AttrTemplate)
			fmt.Fprintf(sb, placeholderInclude, tmplName)

		case TagNameRaw:
			sb.WriteString(n.RawContent)

		case TagNameComment:
			// Comments produce no output

		default:
			fmt.Fprintf(sb, placeholderTag, n.Name)
			// A block tag's children are content the executor renders. Emitting only the tag
			// placeholder made every nested reference vanish from the preview — the same omission
			// as the analysis-side one in processTagNodeForDryRun, in the surface an author reads.
			for _, child := range n.Children {
				t.generatePlaceholders(child, data, sb)
			}
		}

	case *internal.ConditionalNode:
		if len(n.Branches) > 0 {
			fmt.Fprintf(sb, placeholderIfOpen, n.Branches[0].Condition)
			for _, child := range n.Branches[0].Children {
				t.generatePlaceholders(child, data, sb)
			}
			for i := 1; i < len(n.Branches); i++ {
				branch := n.Branches[i]
				if branch.IsElse {
					sb.WriteString(placeholderElse)
				} else {
					fmt.Fprintf(sb, placeholderElseIf, branch.Condition)
				}
				for _, child := range branch.Children {
					t.generatePlaceholders(child, data, sb)
				}
			}
			sb.WriteString(placeholderIfClose)
		}

	case *internal.ForNode:
		fmt.Fprintf(sb, placeholderForOpen, n.ItemVar, n.Source)
		for _, child := range n.Children {
			t.generatePlaceholders(child, data, sb)
		}
		sb.WriteString(placeholderForClose)

	case *internal.SwitchNode:
		fmt.Fprintf(sb, placeholderSwitchOpen, n.Expression)
		for _, c := range n.Cases {
			if c.Value != "" {
				fmt.Fprintf(sb, placeholderCaseVal, c.Value)
			} else {
				fmt.Fprintf(sb, placeholderCaseEval, c.Eval)
			}
			for _, child := range c.Children {
				t.generatePlaceholders(child, data, sb)
			}
		}
		if n.Default != nil {
			sb.WriteString(placeholderDefault)
			for _, child := range n.Default.Children {
				t.generatePlaceholders(child, data, sb)
			}
		}
		sb.WriteString(placeholderSwitchClose)

	case *internal.BlockNode:
		// A block's body is content the executor renders (executeBlockNode). Without this case
		// the node matched nothing and the whole body rendered as EMPTY, while the analysis side
		// — which gained its BlockNode arm in v0.22.0 — reported every reference inside it. The
		// two halves of one DryRunResult described different documents, which is precisely the
		// parity the comment on the tag default arm above claims to keep. It was one arm short.
		for _, child := range n.Children {
			t.generatePlaceholders(child, data, sb)
		}
	}
}

// Explain provides detailed execution explanation for debugging.
func (t *Template) Explain(ctx context.Context, data map[string]any) *ExplainResult {
	result := &ExplainResult{
		Steps:     make([]ExecutionStep, 0),
		Variables: make([]VariableAccess, 0),
		Resolvers: make([]ResolverInvocation, 0),
	}

	startTime := time.Now()

	// Resolve inheritance through the SAME helper ExecuteWithContext uses.
	//
	// Explain used to skip resolution entirely and walk t.ast, so a template using `extends`
	// explained its own block definitions rather than the document that runs — the worst failure
	// mode there is for a debugging tool, because the discrepancy looks like the bug being
	// investigated. Everything below therefore uses astToExplain, never t.ast: the formatted AST,
	// the execution, and the variable-access collection must all describe one document.
	astToExplain, _, inheritErr := t.resolveInheritance(ctx, nil)

	// Generate AST representation
	result.AST = t.formatAST(astToExplain, 0)

	if inheritErr != nil {
		result.Error = inheritErr
		result.Timing = ExecutionTiming{Total: time.Since(startTime)}
		return result
	}

	// Execute with tracking.
	//
	// contextWithInputs must run here too, not only in ExecuteWithContext: Explain calls the
	// executor DIRECTLY and so bypasses that funnel. Without this line a document declaring
	// inputs would EXPLAIN differently than it RENDERS.
	execCtx := t.contextWithInputs(ctx, NewContextWithStrategy(data, t.config.errorStrategy))
	if t.engine != nil {
		execCtx = execCtx.WithEngine(t.engine)
	}

	execStart := time.Now()
	output, err := t.executor.Execute(ctx, astToExplain, execCtx)
	execDuration := time.Since(execStart)

	result.Output = output
	result.Error = err
	result.Timing = ExecutionTiming{
		Total:     time.Since(startTime),
		Execution: execDuration,
	}

	// Add variable accesses from context keys
	t.collectVariableAccesses(astToExplain, data, result)

	return result
}

// formatAST formats the AST as a human-readable string.
func (t *Template) formatAST(node any, depth int) string {
	indent := strings.Repeat(astIndentUnit, depth)
	var sb strings.Builder

	switch n := node.(type) {
	case *internal.RootNode:
		fmt.Fprintf(&sb, astRootLabel, indent)
		for _, child := range n.Children {
			sb.WriteString(t.formatAST(child, depth+1))
		}

	case *internal.TextNode:
		content := n.Content
		if len(content) > astTextMaxLen {
			content = content[:astTextMaxLen] + astTextTruncation
		}
		content = strings.ReplaceAll(content, astNewlineChar, astNewlineEscape)
		fmt.Fprintf(&sb, astTextLabel, indent, content)

	case *internal.TagNode:
		fmt.Fprintf(&sb, astTagLabel, indent, n.Name)
		if len(n.Attributes.Keys()) > 0 {
			attrs := make([]string, 0)
			for _, k := range n.Attributes.Keys() {
				v, _ := n.Attributes.Get(k)
				attrs = append(attrs, fmt.Sprintf(astAttrPair, k, v))
			}
			fmt.Fprintf(&sb, astAttrsFmt, strings.Join(attrs, ", "))
		}
		pos := n.Pos()
		fmt.Fprintf(&sb, astLineFmt, pos.Line)

		// A block tag's children are structure, and an AST dump that stops at the opening tag is
		// not a dump of the tree. Same omission as collectVariableAccesses above.
		if n.Name != TagNameRaw && n.Name != TagNameComment {
			for _, child := range n.Children {
				sb.WriteString(t.formatAST(child, depth+1))
			}
		}

	case *internal.BlockNode:
		fmt.Fprintf(&sb, astBlockLabel, indent, n.Name, n.Pos().Line)
		for _, child := range n.Children {
			sb.WriteString(t.formatAST(child, depth+1))
		}

	case *internal.ConditionalNode:
		pos := n.Pos()
		condition := ""
		if len(n.Branches) > 0 {
			condition = n.Branches[0].Condition
		}
		fmt.Fprintf(&sb, astCondFmt, indent, condition, pos.Line)
		for i, branch := range n.Branches {
			if i == 0 {
				fmt.Fprintf(&sb, astThenLabel, indent)
			} else if branch.IsElse {
				fmt.Fprintf(&sb, astElseLabel, indent)
			} else {
				fmt.Fprintf(&sb, astElseIfFmt, indent, branch.Condition)
			}
			for _, child := range branch.Children {
				sb.WriteString(t.formatAST(child, depth+2))
			}
		}

	case *internal.ForNode:
		pos := n.Pos()
		fmt.Fprintf(&sb, astForFmt, indent, n.ItemVar, n.Source)
		if n.IndexVar != "" {
			fmt.Fprintf(&sb, astForIndex, n.IndexVar)
		}
		if n.Limit > 0 {
			fmt.Fprintf(&sb, astForLimit, n.Limit)
		}
		fmt.Fprintf(&sb, astLineFmt, pos.Line)
		for _, child := range n.Children {
			sb.WriteString(t.formatAST(child, depth+1))
		}

	case *internal.SwitchNode:
		pos := n.Pos()
		fmt.Fprintf(&sb, astSwitchFmt, indent, n.Expression, pos.Line)
		for _, c := range n.Cases {
			if c.Value != "" {
				fmt.Fprintf(&sb, astCaseVal, indent, c.Value)
			} else {
				fmt.Fprintf(&sb, astCaseEval, indent, c.Eval)
			}
			for _, child := range c.Children {
				sb.WriteString(t.formatAST(child, depth+2))
			}
		}
		if n.Default != nil {
			fmt.Fprintf(&sb, astDefaultLbl, indent)
			for _, child := range n.Default.Children {
				sb.WriteString(t.formatAST(child, depth+2))
			}
		}
	}

	return sb.String()
}

// collectVariableAccesses collects variable accesses from the AST.
func (t *Template) collectVariableAccesses(node any, data map[string]any, result *ExplainResult) {
	switch n := node.(type) {
	case *internal.RootNode:
		for _, child := range n.Children {
			t.collectVariableAccesses(child, data, result)
		}

	case *internal.TagNode:
		if n.Name == TagNameVar {
			varName, _ := n.Attributes.Get(AttrName)
			defaultVal := n.Attributes.GetDefault(AttrDefault, "")
			pos := n.Pos()

			val, found := getPath(data, varName)
			result.Variables = append(result.Variables, VariableAccess{
				Path:    varName,
				Value:   val,
				Found:   found,
				Default: defaultVal,
				Line:    pos.Line,
				Column:  pos.Column,
			})
		}

		// Recurse into a block tag's children — the same omission processTagNodeForDryRun fixed
		// in v0.22.0, still present on the Explain side until v0.23.0. {~exons.message~} wraps
		// essentially every real prompt body, so Explain reported ZERO variable accesses for the
		// common case. raw and comment are excluded for the same structural reason as there:
		// their spans are consumed at the lexer and have no parsed children.
		if n.Name != TagNameRaw && n.Name != TagNameComment {
			for _, child := range n.Children {
				t.collectVariableAccesses(child, data, result)
			}
		}

	case *internal.BlockNode:
		for _, child := range n.Children {
			t.collectVariableAccesses(child, data, result)
		}

	case *internal.ConditionalNode:
		for _, branch := range n.Branches {
			for _, child := range branch.Children {
				t.collectVariableAccesses(child, data, result)
			}
		}

	case *internal.ForNode:
		for _, child := range n.Children {
			t.collectVariableAccesses(child, data, result)
		}

	case *internal.SwitchNode:
		for _, c := range n.Cases {
			for _, child := range c.Children {
				t.collectVariableAccesses(child, data, result)
			}
		}
		if n.Default != nil {
			for _, child := range n.Default.Children {
				t.collectVariableAccesses(child, data, result)
			}
		}
	}
}

// Helper functions

// hasPath checks if a path exists in data.
func hasPath(data map[string]any, path string) bool {
	_, ok := getPath(data, path)
	return ok
}

// getPath retrieves a value by dot-notation path.
func getPath(data map[string]any, path string) (any, bool) {
	if path == "" || data == nil {
		return nil, false
	}

	parts := strings.Split(path, PathSeparator)
	var current any = data

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]any:
			val, ok := v[part]
			if !ok {
				return nil, false
			}
			current = val
		case map[string]string:
			val, ok := v[part]
			if !ok {
				return nil, false
			}
			current = val
		default:
			return nil, false
		}
	}

	return current, true
}

// collectAllKeys collects all keys from nested maps with dot notation.
func collectAllKeys(data map[string]any, prefix string) []string {
	var keys []string
	for k, v := range data {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + PathSeparator + k
		}
		keys = append(keys, fullKey)

		// Recurse into nested maps.
		//
		// Only map[string]any, while getPath also traverses map[string]string — so the key set
		// under-covers, and MissingVariables/UnusedVariables/Suggestions under-report for data
		// shaped that way. Left as-is deliberately: every consequence fails in the SAFE
		// direction. An unlisted key can only cost a suggestion or an "unused" note; it can
		// never produce an accusation, which is the failure mode this file's contract guards.
		if nested, ok := v.(map[string]any); ok {
			keys = append(keys, collectAllKeys(nested, fullKey)...)
		}
	}
	return keys
}

// markKeyUsed marks a key and its parent keys as used.
func markKeyUsed(usedKeys map[string]bool, path string) {
	usedKeys[path] = true
	// Also mark parent paths
	parts := strings.Split(path, PathSeparator)
	for i := 1; i < len(parts); i++ {
		parentPath := strings.Join(parts[:i], PathSeparator)
		usedKeys[parentPath] = true
	}
}

// findSimilarStrings finds strings similar to target using Levenshtein distance.
func findSimilarStrings(target string, candidates []string, maxResults int) []string {
	type scored struct {
		str   string
		score int
	}

	var scoredCandidates []scored
	for _, c := range candidates {
		dist := levenshteinDistance(strings.ToLower(target), strings.ToLower(c))
		// Only include if reasonably similar (distance < half the length)
		if dist <= len(target)/2+levenshteinThresholdExtra {
			scoredCandidates = append(scoredCandidates, scored{c, dist})
		}
	}

	// Sort by distance
	sort.Slice(scoredCandidates, func(i, j int) bool {
		return scoredCandidates[i].score < scoredCandidates[j].score
	})

	// Return top results
	results := make([]string, 0, maxResults)
	for i := 0; i < len(scoredCandidates) && i < maxResults; i++ {
		results = append(results, scoredCandidates[i].str)
	}
	return results
}

// levenshteinDistance calculates the edit distance between two strings.
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create matrix
	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(a); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(b); j++ {
		matrix[0][j] = j
	}

	// Fill in the rest
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			matrix[i][j] = minOfThree(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(a)][len(b)]
}

// minOfThree returns the minimum of three integers.
func minOfThree(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// String returns a human-readable summary of the dry-run result.
func (r *DryRunResult) String() string {
	var sb strings.Builder

	sb.WriteString(dryRunHeader)
	fmt.Fprintf(&sb, dryRunValidLabel, r.Valid)

	if len(r.Variables) > 0 {
		fmt.Fprintf(&sb, dryRunVariablesHeader, len(r.Variables))
		for _, v := range r.Variables {
			status := dryRunStatusFound
			if !v.InData {
				if v.HasDefault {
					status = fmt.Sprintf(dryRunStatusDefault, v.Default)
				} else {
					status = dryRunStatusMissing
				}
			}
			fmt.Fprintf(&sb, dryRunStatusVarLine, v.Name, v.Line, status)
			if len(v.Suggestions) > 0 {
				fmt.Fprintf(&sb, dryRunStatusSuggestion, strings.Join(v.Suggestions, ", "))
			}
		}
	}

	if len(r.Resolvers) > 0 {
		fmt.Fprintf(&sb, dryRunResolversHeader, len(r.Resolvers))
		for _, res := range r.Resolvers {
			fmt.Fprintf(&sb, dryRunStatusResSummary, res.TagName, res.Line)
		}
	}

	// Inputs and Switches were collected from v0.21.0 and v0.22.0 respectively and printed by
	// nothing: the human-readable surface silently under-reported the two collections the input
	// work exists to produce, so an author reading String() saw a document with no inputs in it.
	if len(r.Inputs) > 0 {
		fmt.Fprintf(&sb, dryRunInputsHeader, len(r.Inputs))
		for _, in := range r.Inputs {
			status := dryRunStatusInputDecl
			if !in.Declared {
				status = dryRunStatusInputUndec
			}
			fmt.Fprintf(&sb, dryRunStatusInputLine, in.Name, in.Line, status)
		}
	}

	if len(r.Includes) > 0 {
		fmt.Fprintf(&sb, dryRunIncludesHeader, len(r.Includes))
		for _, inc := range r.Includes {
			status := dryRunStatusFound
			if !inc.Exists {
				status = dryRunStatusNotFound
			}
			fmt.Fprintf(&sb, dryRunStatusIncLine, inc.TemplateName, inc.Line, status)
		}
	}

	if len(r.Conditionals) > 0 {
		fmt.Fprintf(&sb, dryRunConditionalsHeader, len(r.Conditionals))
		for _, cond := range r.Conditionals {
			fmt.Fprintf(&sb, dryRunStatusCondLine, cond.Condition, cond.Line)
		}
	}

	if len(r.Switches) > 0 {
		fmt.Fprintf(&sb, dryRunSwitchesHeader, len(r.Switches))
		for _, sw := range r.Switches {
			fmt.Fprintf(&sb, dryRunStatusSwitchLine, sw.Expression, sw.Line, len(sw.Cases))
		}
	}

	if len(r.Loops) > 0 {
		fmt.Fprintf(&sb, dryRunLoopsHeader, len(r.Loops))
		for _, loop := range r.Loops {
			status := dryRunStatusLoopFound
			if !loop.InData {
				status = dryRunStatusLoopMiss
			}
			fmt.Fprintf(&sb, dryRunStatusLoopLine, loop.ItemVar, loop.Source, loop.Line, status)
		}
	}

	if len(r.MissingVariables) > 0 {
		fmt.Fprintf(&sb, dryRunMissingVarsHeader, len(r.MissingVariables))
		for _, v := range r.MissingVariables {
			sb.WriteString(dryRunListPrefix + v + dryRunNewline)
		}
	}

	if len(r.UnusedVariables) > 0 {
		fmt.Fprintf(&sb, dryRunUnusedVarsHeader, len(r.UnusedVariables))
		for _, v := range r.UnusedVariables {
			sb.WriteString(dryRunListPrefix + v + dryRunNewline)
		}
	}

	if len(r.Errors) > 0 {
		fmt.Fprintf(&sb, dryRunErrorsHeader, len(r.Errors))
		for _, e := range r.Errors {
			sb.WriteString(dryRunListPrefix + e + dryRunNewline)
		}
	}

	if len(r.Warnings) > 0 {
		fmt.Fprintf(&sb, dryRunWarningsHeader, len(r.Warnings))
		for _, w := range r.Warnings {
			sb.WriteString(dryRunListPrefix + w + dryRunNewline)
		}
	}

	sb.WriteString(dryRunOutputHeader)
	sb.WriteString(r.Output)
	sb.WriteString(dryRunNewline)

	return sb.String()
}

// String returns a human-readable summary of the explain result.
func (r *ExplainResult) String() string {
	var sb strings.Builder

	sb.WriteString(explainHeader)

	sb.WriteString(explainASTHeader)
	sb.WriteString(r.AST)

	if len(r.Variables) > 0 {
		sb.WriteString(explainVarsHeader)
		for _, v := range r.Variables {
			var status string
			if !v.Found {
				if v.Default != "" {
					status = fmt.Sprintf(explainVarDefault, v.Default)
				} else {
					status = explainVarNotFound
				}
			} else {
				status = fmt.Sprintf(explainVarValue, v.Value)
			}
			fmt.Fprintf(&sb, explainVarLine, v.Line, v.Path, status)
		}
	}

	sb.WriteString(explainTimingHeader)
	fmt.Fprintf(&sb, explainTimingTotal, r.Timing.Total)
	fmt.Fprintf(&sb, explainTimingExec, r.Timing.Execution)

	if r.Error != nil {
		fmt.Fprintf(&sb, explainErrorHeader, r.Error)
	}

	sb.WriteString(explainOutputHeader)
	sb.WriteString(r.Output)
	sb.WriteString(dryRunNewline)

	return sb.String()
}
