package exons

import (
	"sort"
	"strings"

	"github.com/itsatony/go-exons/internal"
)

// exprIdentifierNodeKinds is the number of expression AST node kinds this package knows how to
// walk. It exists to make adding a sixth kind a COMPILE-TIME-adjacent event rather than a silent
// completeness hole: TestExpressionIdentifierWalkerCoversEveryNodeKind asserts this equals the
// number of ExprNodeType constants, so a new node kind fails a test that names this constant
// instead of quietly returning an incomplete identifier set.
//
// The failure mode being guarded is not cosmetic. A consumer uses ExpressionIdentifiers to decide
// whether a declared input is referenced anywhere; a missed identifier means "not referenced",
// which is an accusation, not silence.
const exprIdentifierNodeKinds = 5

// ExpressionIdentifiers returns every context path referenced by an exons expression — the strings
// that appear in eval= on {~exons.if~}, {~exons.elseif~}, {~exons.switch~} and {~exons.case~}.
//
// It exists so a consumer never has to re-parse the expression grammar. That is not a convenience:
// the expression tokenizer, parser and AST all live in package internal and are unimportable, so a
// consumer scanning the raw condition string with its own regex has to agree byte-for-byte with
// this library's lexer about how a reference is spelled — and gets it wrong for dotted paths inside
// function-call arguments, identifiers adjacent to operators without spaces, and paths appearing
// only in the right-hand branch of a boolean. Every one of those misses reports a REFERENCED name
// as unreferenced.
//
// Identifiers are returned as whole dotted paths, because that is how the tokenizer reads them:
// readIdentifier accepts '.', so "input.verbose" is a single identifier and never the two tokens
// "input" and "verbose". A caller matching a reserved root therefore tests a "input." prefix
// (or exact equality with "input") and does not need to reassemble anything.
//
// Function NAMES are deliberately excluded — in len(input.items) the identifier is "input.items"
// and "len" is a call into the function registry, not a context path.
//
// The result is de-duplicated and sorted, so it is stable to compare and safe to fold into a set.
// An empty or whitespace-only expression yields an empty slice and no error. A malformed
// expression returns the parse error: callers that are performing advisory analysis should treat
// an error as "this expression's references are UNKNOWN" and must not conclude that anything is
// unreferenced from it.
func ExpressionIdentifiers(expression string) ([]string, error) {
	if strings.TrimSpace(expression) == "" {
		return []string{}, nil
	}

	node, err := internal.ParseExpression(expression)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	identifiers := make([]string, 0)
	collectExprIdentifiers(node, seen, &identifiers)
	sort.Strings(identifiers)

	return identifiers, nil
}

// collectExprIdentifiers walks an expression AST accumulating identifier paths.
//
// The walk is total over the five node kinds by construction: literals carry no identifiers, and
// every other kind is a container whose children are all visited. See exprIdentifierNodeKinds for
// how a sixth kind is caught.
func collectExprIdentifiers(node internal.ExprNode, seen map[string]bool, out *[]string) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *internal.IdentifierNode:
		if !seen[n.Name] {
			seen[n.Name] = true
			*out = append(*out, n.Name)
		}

	case *internal.UnaryNode:
		collectExprIdentifiers(n.Right, seen, out)

	case *internal.BinaryNode:
		collectExprIdentifiers(n.Left, seen, out)
		collectExprIdentifiers(n.Right, seen, out)

	case *internal.CallNode:
		// n.Name is a function name, not a context path — see the doc comment.
		for _, arg := range n.Args {
			collectExprIdentifiers(arg, seen, out)
		}

	case *internal.LiteralNode:
		// A literal references nothing.
	}
}

// expressionIdentifiers is the analysis-side wrapper used by DryRun: it resolves the context paths
// an eval= expression references, and REPORTS through Errors when it cannot.
//
// It replaces expressionIdentifiersOrEmpty, which returned an empty slice on a parse failure and
// justified it with the claim that "a template whose condition does not parse is already reported
// through Errors by the parser". That was false on both halves. ParseExpression runs only in the
// evaluator — the template parser stores eval= as an opaque string and Engine.Validate never
// inspects it — so nothing else in this library diagnoses a malformed condition at analysis time.
// The result was an empty identifier set indistinguishable from "references nothing", handed to
// exactly the consumer this file's exprIdentifierNodeKinds comment says must never be misled that
// way.
//
// The signature takes the channel rather than returning an error on purpose. The bug was never a
// missing error value — ExpressionIdentifiers has always returned one — it was three call sites
// each independently deciding to drop it, and `ids, _ :=` is a lint violation nowhere. Receiving
// the result makes reporting unconditional and leaves the caller no decision to get wrong.
//
// site names which expression failed (see dryRunSite*); line and col locate it.
//
// The returned slice is still empty on failure. That is now safe, because AnalysisComplete has
// already turned false and an empty slice can no longer be read as "references nothing".
func (r *DryRunResult) expressionIdentifiers(expression, site string, line, col int) []string {
	identifiers, err := ExpressionIdentifiers(expression)
	if err != nil {
		r.reportIncomplete(dryRunErrExpression, line, col, site, expression, err)
		return []string{}
	}
	return identifiers
}
