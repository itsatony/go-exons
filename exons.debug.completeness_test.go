package exons

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests pin DryRun's COMPLETENESS — the property that every place a context path can be
// referenced is reported. They are separate from exons.debug_test.go because they guard a
// different thing: those tests assert DryRun reports what it finds correctly, these assert it
// finds everything there is.
//
// The distinction matters because of who consumes the answer. A consumer building a
// "this declared input is never referenced" advisory turns a MISSED reference into an instruction
// to delete working code. Under-reporting is therefore not a smaller version of a bug here, it is
// a different and worse one, and every test below was verified to FAIL before the fix it guards.

// =============================================================================
// Block-tag children — the highest-severity gap
// =============================================================================

func TestDryRunDescendsIntoBlockTagChildren(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// exons.message wraps essentially every real prompt body. Before v0.22.0
	// processTagNodeForDryRun reported the tag and never descended, so this returned zero
	// references — meaning the miss was the COMMON case, not an edge case.
	t.Run("input reference inside exons.message is found", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="user"~}{~exons.input name="tone" /~}{~/exons.message~}`)
		require.NoError(t, err)

		result := tmpl.DryRun(ctx, nil)

		require.Len(t, result.Inputs, 1, "an input nested in a block tag must be reported")
		assert.Equal(t, "tone", result.Inputs[0].Name)
	})

	t.Run("var reference inside exons.message is found", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="system"~}{~exons.var name="persona" /~}{~/exons.message~}`)
		require.NoError(t, err)

		result := tmpl.DryRun(ctx, nil)

		require.Len(t, result.Variables, 1)
		assert.Equal(t, "persona", result.Variables[0].Name)
	})

	t.Run("nesting is walked to arbitrary depth", func(t *testing.T) {
		tmpl, err := engine.Parse(
			`{~exons.message role="user"~}{~exons.if eval="a"~}{~exons.message role="user"~}{~exons.input name="deep" /~}{~/exons.message~}{~/exons.if~}{~/exons.message~}`)
		require.NoError(t, err)

		result := tmpl.DryRun(ctx, nil)

		require.Len(t, result.Inputs, 1)
		assert.Equal(t, "deep", result.Inputs[0].Name)
	})

	// The counterpart guarantee. A reference inside exons.raw is NOT a reference — the span is
	// consumed at the lexer, so no tag node is ever built. This is the structural agreement that
	// replaces two hand-maintained inert-span regexes, and the child recursion above must not
	// have broken it.
	t.Run("a reference inside exons.raw is still not reported", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.raw~}{~exons.input name="literal" /~}{~/exons.raw~}`)
		require.NoError(t, err)

		result := tmpl.DryRun(ctx, nil)

		assert.Empty(t, result.Inputs, "an exons.raw span is literal text, never a reference")
	})
}

// =============================================================================
// Conditional branches — elseif conditions were discarded
// =============================================================================

func TestDryRunCapturesEveryConditionalBranch(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	tmpl, err := engine.Parse(
		`{~exons.if eval="input.a"~}1{~exons.elseif eval="input.b"~}2{~exons.elseif eval="input.c"~}3{~exons.else~}4{~/exons.if~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(ctx, nil)
	require.Len(t, result.Conditionals, 1)
	cond := result.Conditionals[0]

	// The legacy fields keep their meaning exactly.
	assert.Equal(t, "input.a", cond.Condition)
	assert.True(t, cond.HasElseIf)
	assert.True(t, cond.HasElse)

	// The complete answer: four branches, three of them carrying conditions.
	require.Len(t, cond.Branches, 4, "every branch must be reported, including else")
	assert.Equal(t, "input.a", cond.Branches[0].Condition)
	assert.Equal(t, "input.b", cond.Branches[1].Condition)
	assert.Equal(t, "input.c", cond.Branches[2].Condition)
	assert.True(t, cond.Branches[3].IsElse)
	assert.Empty(t, cond.Branches[3].Condition, "an else branch has no condition")

	// And the identifiers are resolved, so a consumer never re-parses the expression grammar.
	assert.Equal(t, []string{"input.b"}, cond.Branches[1].Identifiers)
	assert.Equal(t, []string{"input.a"}, cond.Identifiers, "Identifiers mirrors the first branch")
}

// =============================================================================
// Switches — previously invisible in their entirety
// =============================================================================

func TestDryRunReportsSwitches(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("dispatch expression and case arms are all reported", func(t *testing.T) {
		tmpl, err := engine.Parse(
			`{~exons.switch eval="input.mode"~}` +
				`{~exons.case value="fast"~}F{~/exons.case~}` +
				`{~exons.case eval="input.thorough"~}T{~/exons.case~}` +
				`{~exons.casedefault~}D{~/exons.casedefault~}` +
				`{~/exons.switch~}`)
		require.NoError(t, err)

		result := tmpl.DryRun(ctx, nil)

		require.Len(t, result.Switches, 1, "a switch must be reported at all")
		sw := result.Switches[0]
		assert.Equal(t, "input.mode", sw.Expression)
		assert.Equal(t, []string{"input.mode"}, sw.Identifiers)
		assert.True(t, sw.HasDefault)

		require.Len(t, sw.Cases, 2)
		assert.Equal(t, "fast", sw.Cases[0].Value)
		assert.Empty(t, sw.Cases[0].Identifiers, "a value arm compares a literal and references nothing")
		assert.Equal(t, "input.thorough", sw.Cases[1].Eval)
		assert.Equal(t, []string{"input.thorough"}, sw.Cases[1].Identifiers)
	})

	t.Run("references inside case bodies are still walked", func(t *testing.T) {
		tmpl, err := engine.Parse(
			`{~exons.switch eval="m"~}{~exons.case value="a"~}{~exons.input name="in_case" /~}{~/exons.case~}{~/exons.switch~}`)
		require.NoError(t, err)

		result := tmpl.DryRun(ctx, nil)

		require.Len(t, result.Inputs, 1)
		assert.Equal(t, "in_case", result.Inputs[0].Name)
	})
}

// =============================================================================
// ExpressionIdentifiers — the public extractor
// =============================================================================

func TestExpressionIdentifiers(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		want       []string
	}{
		{"empty", "", []string{}},
		{"whitespace only", "   ", []string{}},
		{"bare identifier", "input.verbose", []string{"input.verbose"}},
		{"dotted path stays whole", "input.nested.deep", []string{"input.nested.deep"}},
		{"binary both sides", "input.a && input.b", []string{"input.a", "input.b"}},
		{"unary", "!input.disabled", []string{"input.disabled"}},
		{"comparison against literal", `input.mode == "fast"`, []string{"input.mode"}},
		{"literal only", `"x" == "y"`, []string{}},
		{"call argument", "len(input.items) > 0", []string{"input.items"}},
		{"function name is not an identifier", "now()", []string{}},
		{"nested calls", "upper(trim(input.name))", []string{"input.name"}},
		{"deduplicated", "input.a && input.a", []string{"input.a"}},
		{"sorted", "input.z && input.a", []string{"input.a", "input.z"}},
		{"no spaces around operator", "input.a&&input.b", []string{"input.a", "input.b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpressionIdentifiers(tt.expression)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	// A malformed expression must return the error rather than an empty set. The distinction is
	// the whole safety story for a consumer: "references nothing" and "could not tell" must not
	// look the same, because only the first licenses saying an input is unused.
	t.Run("malformed expression errors rather than reporting no references", func(t *testing.T) {
		_, err := ExpressionIdentifiers("input.a &&")
		require.Error(t, err)
	})
}

// TestExpressionIdentifierWalkerCoversEveryNodeKind fails when a sixth expression node kind is
// added without teaching collectExprIdentifiers about it.
//
// It cannot force the walker to be correct — only make an omission a conscious act. That is worth
// having anyway: a silently unhandled node kind returns an INCOMPLETE identifier set, and the
// consumer of an incomplete set accuses an author of leaving an input unused.
func TestExpressionIdentifierWalkerCoversEveryNodeKind(t *testing.T) {
	// The five kinds are Literal, Identifier, Unary, Binary, Call. Their String() method is the
	// only enumeration the internal package exposes, so the count is asserted against the
	// constant that documents the walker's coverage.
	assert.Equal(t, 5, exprIdentifierNodeKinds,
		"a new expression node kind must be added to collectExprIdentifiers before this constant is bumped")
}

// =============================================================================
// Template inheritance — analysis must describe the document that RUNS
// =============================================================================

func TestDryRunAnalysesTheInheritanceResolvedAST(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	require.NoError(t, engine.RegisterTemplate("base-completeness",
		`{~exons.var name="from_parent" /~}{~exons.block name="content"~}default{~/exons.block~}`))

	tmpl, err := engine.Parse(
		`{~exons.extends template="base-completeness" /~}{~exons.block name="content"~}{~exons.input name="from_child" /~}{~/exons.block~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(ctx, nil)

	// The child's own block body — previously unreachable because BlockNode had no walker case.
	inputNames := make([]string, 0, len(result.Inputs))
	for _, in := range result.Inputs {
		inputNames = append(inputNames, in.Name)
	}
	assert.Contains(t, inputNames, "from_child", "a reference inside an exons.block must be reported")

	// The parent body spliced around it — previously unreachable because DryRun walked the raw AST
	// while ExecuteWithContext walks the resolved one.
	varNames := make([]string, 0, len(result.Variables))
	for _, v := range result.Variables {
		varNames = append(varNames, v.Name)
	}
	assert.Contains(t, varNames, "from_parent",
		"analysis must cover the parent body that execution will render")
}

// =============================================================================
// v0.23.0 — Errors as the analysis-completeness channel
//
// The tests above pin that DryRun FINDS everything. These pin the other half of the same
// contract: that when it cannot, it SAYS SO. A walk that gives up silently is indistinguishable
// from a walk that found nothing, and "found nothing" is the accusation.
// =============================================================================

func TestDryRunReportsUnparseableExpressions(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// Before v0.23.0 every one of these returned Identifiers: [] with Errors empty. Nothing else
	// in the library parses an expression at analysis time, so nothing else could have reported
	// it: the template parser stores eval= opaquely and Engine.Validate never inspects it.
	malformed := []struct {
		name     string
		source   string
		wantSite string
	}{
		{
			name:     "if condition",
			source:   `{~exons.if eval="input.a &&"~}x{~/exons.if~}`,
			wantSite: dryRunSiteConditionBranch,
		},
		{
			name:     "elseif condition",
			source:   `{~exons.if eval="input.a"~}x{~exons.elseif eval="|| input.b"~}y{~/exons.if~}`,
			wantSite: dryRunSiteConditionBranch,
		},
		{
			name:     "switch dispatch expression",
			source:   `{~exons.switch eval="&& x"~}{~exons.case value="a"~}A{~/exons.case~}{~/exons.switch~}`,
			wantSite: dryRunSiteSwitch,
		},
		{
			name:     "switch case eval",
			source:   `{~exons.switch eval="input.mode"~}{~exons.case eval="x ||"~}A{~/exons.case~}{~/exons.switch~}`,
			wantSite: dryRunSiteSwitchCase,
		},
	}

	for _, tc := range malformed {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := engine.Parse(tc.source)
			require.NoError(t, err, "the TEMPLATE parses — only the expression inside it does not")

			result := tmpl.DryRun(ctx, nil)

			require.Len(t, result.Errors, 1, "an unparseable expression must be reported exactly once")
			assert.Contains(t, result.Errors[0], tc.wantSite,
				"the report must name WHICH expression failed, not merely that one did")
			assert.False(t, result.AnalysisComplete(),
				"references are UNKNOWN here, and a consumer must be able to tell that from 'absent'")

			// The load-bearing assertion of the whole cycle. An expression in a branch that is
			// never taken renders perfectly, so DryRun has no proof this template cannot run.
			assert.True(t, result.Valid,
				"an unparseable expression proves the ANALYSIS is incomplete, not that the TEMPLATE is invalid")
		})
	}

	// Negative controls. These pin "empty is not malformed", which is the distinction the fix
	// could most easily over-shoot: an else branch carries no condition at all, and a value= case
	// carries a literal rather than an expression.
	wellFormed := []struct {
		name   string
		source string
	}{
		{"else branch has no condition", `{~exons.if eval="input.a"~}x{~exons.else~}y{~/exons.if~}`},
		{"value case is a literal, not an expression", `{~exons.switch eval="input.m"~}{~exons.case value="a"~}A{~/exons.case~}{~/exons.switch~}`},
	}

	for _, tc := range wellFormed {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := engine.Parse(tc.source)
			require.NoError(t, err)

			result := tmpl.DryRun(ctx, nil)

			assert.Empty(t, result.Errors, "an absent expression is not a malformed one")
			assert.True(t, result.AnalysisComplete())
		})
	}
}

func TestDryRunValidAndAnalysisCompleteAreDifferentQuestions(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// Until v0.23.0 Valid was assigned from len(Errors) > 0, so these two could never disagree
	// and Valid carried no information of its own. This test is the doc-as-test for the split.
	t.Run("executable but not fully analysable", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.if eval="input.a"~}x{~exons.elseif eval="&&"~}y{~/exons.if~}`)
		require.NoError(t, err)

		result := tmpl.DryRun(ctx, nil)

		assert.True(t, result.Valid)
		assert.False(t, result.AnalysisComplete())
	})

	t.Run("neither executable nor analysable", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.extends template="no-such-parent-anywhere" /~}`)
		require.NoError(t, err)

		result := tmpl.DryRun(ctx, nil)

		assert.False(t, result.Valid, "unresolvable inheritance is proof ExecuteWithContext fails")
		assert.False(t, result.AnalysisComplete())
	})

	t.Run("clean template", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.var name="a" /~}`)
		require.NoError(t, err)

		result := tmpl.DryRun(ctx, nil)

		assert.True(t, result.Valid)
		assert.True(t, result.AnalysisComplete())
	})
}

func TestDryRunReportsUnanalysableInheritance(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("extends with no engine to resolve through", func(t *testing.T) {
		require.NoError(t, engine.RegisterTemplate("base-no-engine", `{~exons.var name="p" /~}`))
		tmpl, err := engine.Parse(`{~exons.extends template="base-no-engine" /~}body`)
		require.NoError(t, err)

		// Same package, so the unexported field is reachable. This branch existed with no signal
		// at all: the parent body simply went unanalysed and Errors stayed empty.
		tmpl.engine = nil

		result := tmpl.DryRun(ctx, nil)

		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0], "no engine")
		assert.False(t, result.AnalysisComplete())
		assert.True(t, result.Valid,
			"ExecuteWithContext takes the same branch and renders the child body, so this is no proof of failure")
	})

	t.Run("unreadable extends declaration", func(t *testing.T) {
		// Two extends tags: ExtractInheritanceInfo fails, and until v0.23.0 the error was
		// discarded so DryRun believed the template extends nothing at all.
		tmpl, err := engine.Parse(
			`{~exons.extends template="a" /~}{~exons.extends template="b" /~}body`)
		require.NoError(t, err)

		result := tmpl.DryRun(ctx, nil)

		require.NotEmpty(t, result.Errors)
		assert.False(t, result.AnalysisComplete())
	})
}

func TestDryRunWalkerCoversEveryASTNodeKind(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("an unknown node kind is reported, not dropped", func(t *testing.T) {
		result := &DryRunResult{}
		tmpl, err := engine.Parse(`x`)
		require.NoError(t, err)

		type unknownNode struct{}
		tmpl.walkASTForDryRun(unknownNode{}, nil, result, map[string]bool{}, nil)

		require.Len(t, result.Errors, 1,
			"an unhandled node kind takes its entire subtree with it and must never be silent")
		assert.Contains(t, result.Errors[0], "unknownNode", "the report must name the concrete type")
	})

	t.Run("a nil node is reported", func(t *testing.T) {
		result := &DryRunResult{}
		tmpl, err := engine.Parse(`x`)
		require.NoError(t, err)

		tmpl.walkASTForDryRun(nil, nil, result, map[string]bool{}, nil)

		require.Len(t, result.Errors, 1)
	})

	// The regression control for the new default arm. Text nodes are by far the most common kind
	// in any document; if the explicit TextNode case were ever removed, the default arm would
	// report an error for every line of prose and this goes red immediately.
	t.Run("every ordinary node kind stays silent", func(t *testing.T) {
		tmpl, err := engine.Parse(
			`prose {~exons.var name="v" /~}` +
				`{~exons.if eval="a"~}x{~/exons.if~}` +
				`{~exons.for item="i" in="items"~}y{~/exons.for~}` +
				`{~exons.switch eval="m"~}{~exons.case value="c"~}z{~/exons.case~}{~/exons.switch~}` +
				`{~exons.block name="b"~}w{~/exons.block~}`)
		require.NoError(t, err)

		result := tmpl.DryRun(ctx, nil)

		assert.Empty(t, result.Errors, "an ordinary document must not trip the unknown-kind arm")
	})

	assert.Equal(t, 7, dryRunASTNodeKinds,
		"a new AST node kind must gain a case in walkASTForDryRun before this constant is bumped")
}

func TestDryRunReportsTagWithoutName(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// Neither the parser nor Engine.Validate requires name=, so these reach analysis intact and
	// used to record a reference whose target was the empty string — reported as if known.
	for _, tc := range []struct{ name, source string }{
		{"exons.var", `{~exons.var /~}`},
		{"exons.input", `{~exons.input /~}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := engine.Parse(tc.source)
			require.NoError(t, err)

			result := tmpl.DryRun(ctx, nil)

			require.Len(t, result.Errors, 1)
			assert.Contains(t, result.Errors[0], tc.name)
			assert.False(t, result.AnalysisComplete())

			// Still recorded: over-reporting a use is the safe direction.
			assert.True(t, len(result.Variables)+len(result.Inputs) == 1,
				"the reference site is still reported, it is its TARGET that is unknown")
		})
	}
}

func TestDryRunReportsVarAndInputAttributes(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// The attribute map was computed once for all arms and then consumed by only two of the four.
	// A consumer doing unreferenced analysis must treat any attribute value naming a path as a
	// use, and cannot do so for attributes it was never handed.
	tmpl, err := engine.Parse(
		`{~exons.var name="a" join=", " /~}{~exons.input name="t" default="warm" /~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(ctx, nil)

	require.Len(t, result.Variables, 1)
	assert.Equal(t, map[string]string{"name": "a", "join": ", "}, result.Variables[0].Attributes)

	require.Len(t, result.Inputs, 1)
	assert.Equal(t, map[string]string{"name": "t", "default": "warm"}, result.Inputs[0].Attributes)
}

func TestDryRunPlaceholderOutputRendersBlockBodies(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// The analysis side gained its BlockNode arm in v0.22.0 and the preview side did not, so one
	// DryRunResult described two different documents: references inside the block were reported
	// while the block's body rendered as nothing.
	tmpl, err := engine.Parse(`{~exons.block name="x"~}BODY{~/exons.block~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(ctx, nil)

	assert.Contains(t, result.Output, "BODY",
		"the preview must render what the executor would render")
}

func TestDryRunPlaceholderOutputNamesTheInput(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// exons.input fell to the resolver default arm and previewed as the literal "{{exons.input}}",
	// so every input in a document rendered as the same opaque token.
	tmpl, err := engine.Parse(`{~exons.input name="max_bullets" /~}`)
	require.NoError(t, err)

	result := tmpl.DryRun(ctx, nil)

	assert.Contains(t, result.Output, "max_bullets")
	assert.NotContains(t, result.Output, TagNameInput,
		"the placeholder must name the input, not the verb")
}

func TestDryRunStringReportsInputsAndSwitches(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// Both collections were populated and printed by nothing, so an author reading String() saw a
	// document with no inputs in it.
	tmpl, err := engine.Parse(
		`{~exons.input name="tone" /~}{~exons.switch eval="m"~}{~exons.case value="a"~}A{~/exons.case~}{~/exons.switch~}`)
	require.NoError(t, err)

	out := tmpl.DryRun(ctx, nil).String()

	assert.Contains(t, out, "Inputs (1)")
	assert.Contains(t, out, "tone")
	assert.Contains(t, out, "Switches (1)")
}

func TestExplainDescendsIntoBlockTagChildren(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// The same two gaps DryRun closed in v0.22.0 were still open on the Explain side: no
	// recursion into a tag's children and no BlockNode case. exons.message wraps essentially
	// every real prompt body, so Explain reported ZERO variable accesses for the common case.
	tmpl, err := engine.Parse(
		`{~exons.message role="user"~}{~exons.var name="persona" /~}{~/exons.message~}`)
	require.NoError(t, err)

	result := tmpl.Explain(ctx, map[string]any{"persona": "helpful"})

	paths := make([]string, 0, len(result.Variables))
	for _, v := range result.Variables {
		paths = append(paths, v.Path)
	}
	assert.Contains(t, paths, "persona",
		"a variable nested in a block tag must appear in the explanation")
}

func TestDryRunErrorsHaveExactlyOneWriter(t *testing.T) {
	// Errors is the channel a consumer gates on before concluding a name is referenced NOWHERE —
	// a conclusion whose remedy is to delete an author's declaration. A channel written from
	// scattered points across a recursive walk is one nobody can audit for completeness, so the
	// single door is a property worth enforcing mechanically rather than remembering.
	//
	// This parses the package's own source rather than grepping it, so a write spelled across
	// several lines or hidden behind different whitespace cannot slip past.
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	require.NoError(t, err)

	writers := make(map[string]bool)

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			var enclosing string
			ast.Inspect(file, func(n ast.Node) bool {
				if fn, ok := n.(*ast.FuncDecl); ok {
					enclosing = fn.Name.Name
					return true
				}
				assign, ok := n.(*ast.AssignStmt)
				if !ok {
					return true
				}
				for _, lhs := range assign.Lhs {
					sel, ok := lhs.(*ast.SelectorExpr)
					if ok && sel.Sel.Name == "Errors" {
						writers[enclosing] = true
					}
				}
				return true
			})
		}
	}

	assert.Equal(t, map[string]bool{"reportIncomplete": true}, writers,
		"DryRunResult.Errors must be written only through reportIncomplete — add a call to it "+
			"rather than appending directly, so the completeness channel stays auditable")
}

func TestConditionalBranchPositionsPointAtTheirOwnTag(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	// parseConditional built every branch after the first with the position of the tag that
	// TERMINATED it — and the last branch with the position of the closing {~/exons.if~}. The
	// same field is what executeConditional reports a failing branch expression at, so this was
	// reaching users as wrong line numbers in runtime errors before it ever reached DryRun.
	//
	// One tag per line, so a position is unambiguous.
	tmpl, err := engine.Parse("{~exons.if eval=\"a\"~}\n" + // line 1
		"x\n" + // line 2
		"{~exons.elseif eval=\"b\"~}\n" + // line 3
		"y\n" + // line 4
		"{~exons.else~}\n" + // line 5
		"z\n" + // line 6
		"{~/exons.if~}") // line 7
	require.NoError(t, err)

	result := tmpl.DryRun(ctx, nil)

	require.Len(t, result.Conditionals, 1)
	branches := result.Conditionals[0].Branches
	require.Len(t, branches, 3)

	assert.Equal(t, 1, branches[0].Line, "the if branch")
	assert.Equal(t, 3, branches[1].Line, "the elseif branch, not the else that follows it")
	assert.Equal(t, 5, branches[2].Line, "the else branch, not the closing tag that follows it")
}
