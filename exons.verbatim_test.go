package exons

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Verbatim tilde fences {~~ ... ~~}
// =============================================================================

func TestEngine_Execute_VerbatimFence(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("fence body renders literally", func(t *testing.T) {
		result, err := engine.Execute(ctx, `Use {~~{~exons.var name="x" /~}~~} to interpolate.`, nil)
		require.NoError(t, err)
		assert.Equal(t, `Use {~exons.var name="x" /~} to interpolate.`, result)
	})

	t.Run("fence contains full raw block example", func(t *testing.T) {
		result, err := engine.Execute(ctx, "{~~ {~exons.raw~}not parsed{~/exons.raw~} ~~}", nil)
		require.NoError(t, err)
		assert.Equal(t, " {~exons.raw~}not parsed{~/exons.raw~} ", result)
	})

	t.Run("escalated fence contains two-tilde close", func(t *testing.T) {
		result, err := engine.Execute(ctx, "{~~~ a fence looks like {~~ x ~~} done ~~~}", nil)
		require.NoError(t, err)
		assert.Equal(t, " a fence looks like {~~ x ~~} done ", result)
	})

	t.Run("escaped fence open is literal", func(t *testing.T) {
		result, err := engine.Execute(ctx, `\{~~not a fence~~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, `{~~not a fence~~}`, result)
	})

	t.Run("escape inside fence stays escaped", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~~ write \{~ to escape ~~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, ` write \{~ to escape `, result)
	})

	t.Run("tags around fence still render", func(t *testing.T) {
		result, err := engine.Execute(ctx,
			`{~exons.var name="a" /~}{~~ {~exons.var name="a" /~} ~~}{~exons.var name="a" /~}`,
			map[string]any{"a": "X"})
		require.NoError(t, err)
		assert.Equal(t, `X {~exons.var name="a" /~} X`, result)
	})

	t.Run("unterminated fence is a parse error", func(t *testing.T) {
		_, err := engine.Execute(ctx, "{~~ never closed", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgParseFailed)
	})

	t.Run("large fence body round-trips", func(t *testing.T) {
		body := strings.Repeat("{~exons.var name=\"x\" /~} and text\n", 5000)
		result, err := engine.Execute(ctx, "{~~"+body+"~~}", nil)
		require.NoError(t, err)
		assert.Equal(t, body, result)
	})
}

func TestEngine_Execute_VerbatimFence_CustomDelimitersInert(t *testing.T) {
	engine := MustNew(WithDelimiters("<<", ">>"))
	ctx := context.Background()

	result, err := engine.Execute(ctx, `{~~ plain text ~~} <<exons.var name="x" />>`, map[string]any{"x": "V"})
	require.NoError(t, err)
	assert.Equal(t, `{~~ plain text ~~} V`, result)
}

// =============================================================================
// Raw blocks: byte fidelity + keepRaw
// =============================================================================

func TestEngine_Execute_RawByteFidelity(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("lexically invalid example survives in raw", func(t *testing.T) {
		result, err := engine.Execute(ctx, "{~exons.raw~}a lone {~ and {~ 5 ~} fragment{~/exons.raw~}", nil)
		require.NoError(t, err)
		assert.Equal(t, "a lone {~ and {~ 5 ~} fragment", result)
	})

	t.Run("escape inside raw preserved byte-for-byte", func(t *testing.T) {
		result, err := engine.Execute(ctx, `{~exons.raw~}\{~ literal{~/exons.raw~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, `\{~ literal`, result)
	})

	t.Run("irregular whitespace and quotes preserved", func(t *testing.T) {
		body := `{~x   a='1'  b="2"/~}`
		result, err := engine.Execute(ctx, "{~exons.raw~}"+body+"{~/exons.raw~}", nil)
		require.NoError(t, err)
		assert.Equal(t, body, result)
	})

	t.Run("first close wins", func(t *testing.T) {
		result, err := engine.Execute(ctx, "{~exons.raw~}a{~exons.raw~}b{~/exons.raw~}", nil)
		require.NoError(t, err)
		assert.Equal(t, "a{~exons.raw~}b", result)
	})

	t.Run("unterminated raw is a parse error", func(t *testing.T) {
		_, err := engine.Execute(ctx, "{~exons.raw~}never closes", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgParseFailed)
	})

	t.Run("keepRaw returns original tag source", func(t *testing.T) {
		engine := MustNew(WithErrorStrategy(ErrorStrategyKeepRaw))
		source := `{~unknown.tag  attr='v' /~}`
		result, err := engine.Execute(ctx, source, nil)
		require.NoError(t, err)
		assert.Equal(t, source, result)
	})
}

// =============================================================================
// Markdown fence mode (WithMarkdownFences)
// =============================================================================

func TestEngine_Execute_MarkdownFences(t *testing.T) {
	ctx := context.Background()

	t.Run("inert fence hides tags, prose renders", func(t *testing.T) {
		engine := MustNew(WithMarkdownFences())
		source := "Hello {~exons.var name=\"who\" /~}!\n```\n{~exons.var name=\"who\" /~}\n```\ndone"
		result, err := engine.Execute(ctx, source, map[string]any{"who": "World"})
		require.NoError(t, err)
		assert.Equal(t, "Hello World!\n```\n{~exons.var name=\"who\" /~}\n```\ndone", result)
	})

	t.Run("exons info string renders live", func(t *testing.T) {
		engine := MustNew(WithMarkdownFences())
		source := "```exons\n{~exons.var name=\"who\" /~}\n```\n"
		result, err := engine.Execute(ctx, source, map[string]any{"who": "World"})
		require.NoError(t, err)
		assert.Equal(t, "```exons\nWorld\n```\n", result)
	})

	t.Run("tilde fence inert", func(t *testing.T) {
		engine := MustNew(WithMarkdownFences())
		source := "~~~\n{~exons.var name=\"who\" /~}\n~~~\n"
		result, err := engine.Execute(ctx, source, map[string]any{"who": "World"})
		require.NoError(t, err)
		assert.Equal(t, source, result)
	})

	t.Run("mode off renders inside fences", func(t *testing.T) {
		engine := MustNew()
		source := "```\n{~exons.var name=\"who\" /~}\n```\n"
		result, err := engine.Execute(ctx, source, map[string]any{"who": "World"})
		require.NoError(t, err)
		assert.Equal(t, "```\nWorld\n```\n", result)
	})

	t.Run("broken example syntax inside inert fence", func(t *testing.T) {
		engine := MustNew(WithMarkdownFences())
		source := "```\na lone {~ open and {~~}\n```\nok"
		result, err := engine.Execute(ctx, source, nil)
		require.NoError(t, err)
		assert.Equal(t, source, result)
	})

	t.Run("inheritance honors markdown mode", func(t *testing.T) {
		engine := MustNew(WithMarkdownFences())
		engine.MustRegisterTemplate("base",
			"P:{~exons.block name=\"content\"~}default{~/exons.block~}\n```\n{~exons.var name=\"x\" /~}\n```")
		result, err := engine.Execute(ctx,
			`{~exons.extends template="base" /~}{~exons.block name="content"~}child{~/exons.block~}`, nil)
		require.NoError(t, err)
		assert.Equal(t, "P:child\n```\n{~exons.var name=\"x\" /~}\n```", result)
	})
}

func TestEngine_Execute_MarkdownFences_CombinedSkillTemplate(t *testing.T) {
	// A realistic SKILL.md-style body: teaching examples in inert fences,
	// a live fence with interpolation, and a verbatim fence with raw-in-raw.
	engine := MustNew(WithMarkdownFences())
	ctx := context.Background()

	source := "# " + "{~exons.var name=\"title\" /~}\n\n" +
		"To interpolate, write:\n" +
		"```\n{~exons.var name=\"user\" default=\"Guest\" /~}\n```\n\n" +
		"Your generated config:\n" +
		"```exons\nname: {~exons.var name=\"title\" /~}\n```\n\n" +
		"Raw blocks look like {~~{~exons.raw~}x{~/exons.raw~}~~}.\n"

	want := "# Teach\n\n" +
		"To interpolate, write:\n" +
		"```\n{~exons.var name=\"user\" default=\"Guest\" /~}\n```\n\n" +
		"Your generated config:\n" +
		"```exons\nname: Teach\n```\n\n" +
		"Raw blocks look like {~exons.raw~}x{~/exons.raw~}.\n"

	result, err := engine.Execute(ctx, source, map[string]any{"title": "Teach"})
	require.NoError(t, err)
	assert.Equal(t, want, result)
}

// =============================================================================
// Validation lints
// =============================================================================

func TestEngine_Validate_MarkdownFenceLints(t *testing.T) {
	t.Run("tag-like syntax in inert fence warns", func(t *testing.T) {
		engine := MustNew(WithMarkdownFences())
		result, err := engine.Validate("```\n{~exons.var name=\"x\" /~}\n```\n")
		require.NoError(t, err)
		require.Len(t, result.Warnings(), 1)
		assert.Equal(t, WarnMsgTagLikeInInertFence, result.Warnings()[0].Message)
		assert.Equal(t, 1, result.Warnings()[0].Position.Line)
	})

	t.Run("live fence does not warn", func(t *testing.T) {
		engine := MustNew(WithMarkdownFences())
		result, err := engine.Validate("```exons\n{~exons.var name=\"x\" /~}\n```\n")
		require.NoError(t, err)
		assert.Empty(t, result.Warnings())
	})

	t.Run("unclosed fence warns", func(t *testing.T) {
		engine := MustNew(WithMarkdownFences())
		result, err := engine.Validate("text\n```\nnever closed")
		require.NoError(t, err)
		require.Len(t, result.Warnings(), 1)
		assert.Equal(t, WarnMsgUnclosedMarkdownFence, result.Warnings()[0].Message)
		assert.Equal(t, 2, result.Warnings()[0].Position.Line)
	})

	t.Run("fence without tags does not warn", func(t *testing.T) {
		engine := MustNew(WithMarkdownFences())
		result, err := engine.Validate("```go\nfmt.Println(\"hi\")\n```\n")
		require.NoError(t, err)
		assert.Empty(t, result.Warnings())
	})

	t.Run("option off emits no fence lints", func(t *testing.T) {
		engine := MustNew()
		result, err := engine.Validate("```\n{~exons.var name=\"x\" /~}\n```\n")
		require.NoError(t, err)
		for _, w := range result.Warnings() {
			assert.NotEqual(t, WarnMsgTagLikeInInertFence, w.Message)
			assert.NotEqual(t, WarnMsgUnclosedMarkdownFence, w.Message)
		}
	})
}

// =============================================================================
// Spec.ContentFormat
// =============================================================================

func TestImportFromSkillMD_SetsContentFormat(t *testing.T) {
	content := "---\nname: my-skill\ndescription: test\n---\n# Body\n"
	spec, err := ImportFromSkillMD(content)
	require.NoError(t, err)
	assert.Equal(t, ContentFormatMarkdown, spec.ContentFormat)
	assert.Equal(t, "# Body\n", spec.Body)

	clone := spec.Clone()
	assert.Equal(t, ContentFormatMarkdown, clone.ContentFormat)
}

func TestSpec_ContentFormat_SerializationRoundTrip(t *testing.T) {
	spec := &Spec{Name: "s", ContentFormat: ContentFormatMarkdown}
	data, err := spec.Serialize(nil)
	require.NoError(t, err)
	assert.Contains(t, string(data), "content_format: markdown")

	yamlPart := strings.TrimPrefix(string(data), YAMLFrontmatterDelimiter+"\n")
	yamlPart = strings.Split(yamlPart, "\n"+YAMLFrontmatterDelimiter)[0]
	parsed, err := ParseYAMLSpec(yamlPart)
	require.NoError(t, err)
	assert.Equal(t, ContentFormatMarkdown, parsed.ContentFormat)
}
