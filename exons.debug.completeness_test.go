package exons

import (
	"context"
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
