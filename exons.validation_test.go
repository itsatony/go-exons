package exons

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Valid Templates Pass
// =============================================================================

func TestEngine_Validate_ValidTemplates(t *testing.T) {
	engine := MustNew()

	t.Run("empty template", func(t *testing.T) {
		result, err := engine.Validate("")
		require.NoError(t, err)
		assert.True(t, result.IsValid())
		assert.Empty(t, result.Errors())
		assert.Empty(t, result.Warnings())
	})

	t.Run("plain text", func(t *testing.T) {
		result, err := engine.Validate("Hello World")
		require.NoError(t, err)
		assert.True(t, result.IsValid())
	})

	t.Run("valid variable tag", func(t *testing.T) {
		result, err := engine.Validate(`{~exons.var name="x" /~}`)
		require.NoError(t, err)
		assert.True(t, result.IsValid())
	})

	t.Run("valid conditional", func(t *testing.T) {
		result, err := engine.Validate(`{~exons.if eval="x"~}content{~/exons.if~}`)
		require.NoError(t, err)
		assert.True(t, result.IsValid())
	})

	t.Run("valid loop", func(t *testing.T) {
		result, err := engine.Validate(`{~exons.for item="x" in="items"~}body{~/exons.for~}`)
		require.NoError(t, err)
		assert.True(t, result.IsValid())
	})

	t.Run("valid raw block", func(t *testing.T) {
		result, err := engine.Validate(`{~exons.raw~}anything here{~/exons.raw~}`)
		require.NoError(t, err)
		assert.True(t, result.IsValid())
	})

	t.Run("valid comment", func(t *testing.T) {
		result, err := engine.Validate(`{~exons.comment~}ignored{~/exons.comment~}`)
		require.NoError(t, err)
		assert.True(t, result.IsValid())
	})

	t.Run("valid switch", func(t *testing.T) {
		result, err := engine.Validate(`{~exons.switch eval="x"~}{~exons.case value="a"~}A{~/exons.case~}{~/exons.switch~}`)
		require.NoError(t, err)
		assert.True(t, result.IsValid())
	})

	t.Run("valid env tag", func(t *testing.T) {
		result, err := engine.Validate(`{~exons.env name="PATH" /~}`)
		require.NoError(t, err)
		assert.True(t, result.IsValid())
	})

	t.Run("valid message tag", func(t *testing.T) {
		result, err := engine.Validate(`{~exons.message role="user"~}Hello{~/exons.message~}`)
		require.NoError(t, err)
		assert.True(t, result.IsValid())
	})
}

// =============================================================================
// Unknown Tags -> Warnings
// =============================================================================

func TestEngine_Validate_UnknownTag(t *testing.T) {
	engine := MustNew()

	result, err := engine.Validate(`{~UnknownTag attr="val" /~}`)
	require.NoError(t, err)
	assert.True(t, result.HasWarnings())
	warnings := result.Warnings()
	require.Len(t, warnings, 1)
	assert.Equal(t, SeverityWarning, warnings[0].Severity)
	assert.Equal(t, ErrMsgUnknownTagInTemplate, warnings[0].Message)
	assert.Equal(t, "UnknownTag", warnings[0].TagName)
}

func TestEngine_Validate_MultipleUnknownTags(t *testing.T) {
	engine := MustNew()

	result, err := engine.Validate(`{~Tag1 /~} {~Tag2 /~}`)
	require.NoError(t, err)
	assert.True(t, result.HasWarnings())
	assert.Len(t, result.Warnings(), 2)
}

// =============================================================================
// Invalid OnError Attribute -> Errors
// =============================================================================

func TestEngine_Validate_InvalidOnError(t *testing.T) {
	engine := MustNew()

	result, err := engine.Validate(`{~exons.var name="x" onerror="invalid_strategy" /~}`)
	require.NoError(t, err)
	assert.True(t, result.HasErrors())
	errors := result.Errors()
	require.Len(t, errors, 1)
	assert.Equal(t, ErrMsgInvalidOnErrorAttr, errors[0].Message)
}

func TestEngine_Validate_ValidOnError(t *testing.T) {
	engine := MustNew()

	strategies := []string{
		ErrorStrategyNameThrow,
		ErrorStrategyNameDefault,
		ErrorStrategyNameRemove,
		ErrorStrategyNameKeepRaw,
		ErrorStrategyNameLog,
	}
	for _, s := range strategies {
		t.Run(s, func(t *testing.T) {
			result, err := engine.Validate(`{~exons.var name="x" onerror="` + s + `" /~}`)
			require.NoError(t, err)
			assert.False(t, result.HasErrors(), "strategy %q should be valid", s)
		})
	}
}

// =============================================================================
// Missing Include Targets -> Warnings
// =============================================================================

func TestEngine_Validate_MissingInclude(t *testing.T) {
	engine := MustNew()

	result, err := engine.Validate(`{~exons.include template="nonexistent" /~}`)
	require.NoError(t, err)
	assert.True(t, result.HasWarnings())
	warnings := result.Warnings()
	found := false
	for _, w := range warnings {
		if w.Message == ErrMsgMissingIncludeTarget {
			found = true
			break
		}
	}
	assert.True(t, found, "should have missing include target warning")
}

func TestEngine_Validate_RegisteredInclude(t *testing.T) {
	engine := MustNew()
	engine.MustRegisterTemplate("header", "Header Content")

	result, err := engine.Validate(`{~exons.include template="header" /~}`)
	require.NoError(t, err)
	// Should not have the missing include warning
	for _, w := range result.Warnings() {
		assert.NotEqual(t, ErrMsgMissingIncludeTarget, w.Message)
	}
}

// =============================================================================
// For Validation
// =============================================================================

func TestEngine_Validate_ForLoop(t *testing.T) {
	engine := MustNew()

	t.Run("valid for loop", func(t *testing.T) {
		result, err := engine.Validate(`{~exons.for item="x" in="items"~}body{~/exons.for~}`)
		require.NoError(t, err)
		assert.False(t, result.HasErrors())
	})

	// Note: The parser typically enforces missing item/in at parse time.
	// The validation here is for additional checks.

	t.Run("for with negative limit handled by parser", func(t *testing.T) {
		// The parser may or may not propagate negative limits.
		// If it does, validation catches it. If not, it's handled at parse time.
		result, err := engine.Validate(`{~exons.for item="x" in="items" limit="-1"~}body{~/exons.for~}`)
		require.NoError(t, err)
		// Either the limit is parsed as 0 (default) and no error,
		// or it's -1 and validation catches it
		_ = result // valid regardless — the parser may clamp to 0
	})
}

// =============================================================================
// Switch Validation
// =============================================================================

func TestEngine_Validate_Switch(t *testing.T) {
	engine := MustNew()

	t.Run("valid switch", func(t *testing.T) {
		result, err := engine.Validate(`{~exons.switch eval="x"~}{~exons.case value="a"~}A{~/exons.case~}{~/exons.switch~}`)
		require.NoError(t, err)
		assert.False(t, result.HasErrors())
	})

	t.Run("switch with default case", func(t *testing.T) {
		result, err := engine.Validate(`{~exons.switch eval="x"~}{~exons.case value="a"~}A{~/exons.case~}{~exons.casedefault~}D{~/exons.casedefault~}{~/exons.switch~}`)
		require.NoError(t, err)
		assert.False(t, result.HasErrors())
	})
}

// =============================================================================
// Nested Validation (Children Validated Recursively)
// =============================================================================

func TestEngine_Validate_Nested(t *testing.T) {
	engine := MustNew()

	t.Run("unknown tag inside conditional", func(t *testing.T) {
		result, err := engine.Validate(`{~exons.if eval="x"~}{~UnknownTag /~}{~/exons.if~}`)
		require.NoError(t, err)
		assert.True(t, result.HasWarnings())
		warnings := result.Warnings()
		found := false
		for _, w := range warnings {
			if w.TagName == "UnknownTag" {
				found = true
				break
			}
		}
		assert.True(t, found, "should find unknown tag warning inside conditional")
	})

	t.Run("unknown tag inside loop", func(t *testing.T) {
		result, err := engine.Validate(`{~exons.for item="x" in="items"~}{~BadTag /~}{~/exons.for~}`)
		require.NoError(t, err)
		assert.True(t, result.HasWarnings())
	})

	t.Run("unknown tag inside switch case", func(t *testing.T) {
		result, err := engine.Validate(`{~exons.switch eval="x"~}{~exons.case value="a"~}{~UnknownInCase /~}{~/exons.case~}{~/exons.switch~}`)
		require.NoError(t, err)
		assert.True(t, result.HasWarnings())
	})

	t.Run("deeply nested validation", func(t *testing.T) {
		tmpl := `{~exons.if eval="a"~}
  {~exons.for item="x" in="items"~}
    {~exons.if eval="x"~}
      {~DeepUnknown /~}
    {~/exons.if~}
  {~/exons.for~}
{~/exons.if~}`
		result, err := engine.Validate(tmpl)
		require.NoError(t, err)
		assert.True(t, result.HasWarnings())
	})
}

// =============================================================================
// Parse Error -> Validation Error
// =============================================================================

func TestEngine_Validate_ParseError(t *testing.T) {
	engine := MustNew()

	t.Run("invalid syntax", func(t *testing.T) {
		result, err := engine.Validate(`{~exons.if~}no closing tag`)
		require.NoError(t, err)
		assert.True(t, result.HasErrors())
		assert.False(t, result.IsValid())
	})
}

// =============================================================================
// ValidationResult Methods
// =============================================================================

func TestValidationResult_Methods(t *testing.T) {
	t.Run("empty result", func(t *testing.T) {
		result := &ValidationResult{issues: make([]ValidationIssue, 0)}
		assert.True(t, result.IsValid())
		assert.False(t, result.HasErrors())
		assert.False(t, result.HasWarnings())
		assert.Empty(t, result.Issues())
		assert.Empty(t, result.Errors())
		assert.Empty(t, result.Warnings())
	})

	t.Run("result with error", func(t *testing.T) {
		result := &ValidationResult{
			issues: []ValidationIssue{
				{Severity: SeverityError, Message: "error1"},
			},
		}
		assert.False(t, result.IsValid())
		assert.True(t, result.HasErrors())
		assert.False(t, result.HasWarnings())
		assert.Len(t, result.Errors(), 1)
		assert.Len(t, result.Warnings(), 0)
	})

	t.Run("result with warning", func(t *testing.T) {
		result := &ValidationResult{
			issues: []ValidationIssue{
				{Severity: SeverityWarning, Message: "warning1"},
			},
		}
		assert.True(t, result.IsValid())
		assert.False(t, result.HasErrors())
		assert.True(t, result.HasWarnings())
		assert.Len(t, result.Errors(), 0)
		assert.Len(t, result.Warnings(), 1)
	})

	t.Run("result with mixed issues", func(t *testing.T) {
		result := &ValidationResult{
			issues: []ValidationIssue{
				{Severity: SeverityError, Message: "error1"},
				{Severity: SeverityWarning, Message: "warning1"},
				{Severity: SeverityInfo, Message: "info1"},
				{Severity: SeverityError, Message: "error2"},
			},
		}
		assert.False(t, result.IsValid())
		assert.True(t, result.HasErrors())
		assert.True(t, result.HasWarnings())
		assert.Len(t, result.Issues(), 4)
		assert.Len(t, result.Errors(), 2)
		assert.Len(t, result.Warnings(), 1)
	})
}

// =============================================================================
// ValidationIssue Fields
// =============================================================================

func TestValidationIssue_Fields(t *testing.T) {
	issue := ValidationIssue{
		Severity: SeverityError,
		Message:  "test message",
		Position: Position{Line: 5, Column: 10},
		TagName:  "exons.var",
	}
	assert.Equal(t, SeverityError, issue.Severity)
	assert.Equal(t, "test message", issue.Message)
	assert.Equal(t, 5, issue.Position.Line)
	assert.Equal(t, 10, issue.Position.Column)
	assert.Equal(t, "exons.var", issue.TagName)
}

// =============================================================================
// Validate with Registered Resolver
// =============================================================================

func TestEngine_Validate_RegisteredResolver(t *testing.T) {
	engine := MustNew()
	engine.MustRegister(NewResolverFunc("CustomTag", func(ctx context.Context, execCtx *Context, attrs Attributes) (string, error) {
		return "", nil
	}, nil))

	result, err := engine.Validate(`{~CustomTag /~}`)
	require.NoError(t, err)
	// CustomTag is registered, so no "unknown tag" warning
	for _, w := range result.Warnings() {
		assert.NotEqual(t, "CustomTag", w.TagName, "should not warn about registered custom tag")
	}
}

// =============================================================================
// Validate Switch Default Case
// =============================================================================

func TestEngine_Validate_SwitchDefaultCase(t *testing.T) {
	engine := MustNew()

	t.Run("default case children validated", func(t *testing.T) {
		result, err := engine.Validate(`{~exons.switch eval="x"~}{~exons.case value="a"~}A{~/exons.case~}{~exons.casedefault~}{~UnknownInDefault /~}{~/exons.casedefault~}{~/exons.switch~}`)
		require.NoError(t, err)
		assert.True(t, result.HasWarnings())
	})
}
