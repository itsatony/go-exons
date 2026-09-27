package exons

import (
	"context"
	"errors"
	"testing"

	"github.com/itsatony/go-cuserr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// go-exons#11: Parse judged grammar only, so a tag its own resolver refuses parsed cleanly and then
// failed at every render. WithStrictAttributes makes Parse refuse exactly what Validate reports as
// an error; the default is unchanged.

const (
	strictIncludeNoTemplate = `{~exons.include ref="@org/frag" /~}`
	strictMessageNoRole     = `{~exons.message~}hello{~/exons.message~}`
	strictWellFormed        = `{~exons.message role="user"~}hi {~exons.var name="x" default="y" /~}{~/exons.message~}`
	strictNestedInIf        = "{~exons.if eval=\"true\"~}a{~exons.else~}{~exons.include ref=\"x\" /~}{~/exons.if~}"
	strictNestedInFor       = "{~exons.for item=\"i\" in=\"items\"~}{~exons.message~}x{~/exons.message~}{~/exons.for~}"
	strictNestedInSwitch    = "{~exons.switch eval=\"k\"~}{~exons.case value=\"a\"~}{~exons.include ref=\"x\" /~}{~/exons.case~}{~exons.casedefault~}ok{~/exons.casedefault~}{~/exons.switch~}"
	strictNestedInTag       = "{~exons.message role=\"user\"~}{~exons.include ref=\"x\" /~}{~/exons.message~}"
	strictUnknownTagOnly    = `{~custom.later /~}`
	strictTwoIssues         = `{~exons.include ref="a" /~} and {~exons.message~}x{~/exons.message~}`
	strictFrontmatterBadTag = "---\nname: t\ndescription: '{~exons.include ref=\"x\" /~}'\n---\nbody"
)

func TestWithStrictAttributes_Parse(t *testing.T) {
	cases := []struct {
		name       string
		source     string
		wantRefuse bool
		wantTag    string
		wantLine   int
	}{
		{"include without template", strictIncludeNoTemplate, true, TagNameInclude, 1},
		{"message without role", strictMessageNoRole, true, TagNameMessage, 1},
		{"well-formed message and var", strictWellFormed, false, "", 0},
		{"include in else branch", strictNestedInIf, true, TagNameInclude, 1},
		{"message in for body", strictNestedInFor, true, TagNameMessage, 1},
		{"include in switch case", strictNestedInSwitch, true, TagNameInclude, 1},
		{"include inside a message", strictNestedInTag, true, TagNameInclude, 1},
		{"unknown tag is only a warning", strictUnknownTagOnly, false, "", 0},
	}

	strict := MustNew(WithStrictAttributes())
	lenient := MustNew()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The default engine accepts every one of these: the behaviour strict mode changes.
			_, err := lenient.Parse(tc.source)
			require.NoError(t, err, "default Parse must stay grammar-only")

			// Strict Parse agrees with Validate's error verdict, both directions.
			res, err := strict.Validate(tc.source)
			require.NoError(t, err)
			assert.Equal(t, tc.wantRefuse, res.HasErrors(), "fixture must match Validate's verdict")

			tmpl, err := strict.Parse(tc.source)
			if !tc.wantRefuse {
				require.NoError(t, err)
				require.NotNil(t, tmpl)
				return
			}
			require.Error(t, err)
			assert.Nil(t, tmpl)
			assert.True(t, errors.Is(err, ErrStrictAttributes), "errors.Is sentinel")

			var sae *StrictAttributeError
			require.True(t, errors.As(err, &sae), "errors.As typed error")
			require.NotEmpty(t, sae.Issues)
			assert.Equal(t, tc.wantTag, sae.Issues[0].TagName)
			assert.Equal(t, tc.wantLine, sae.Issues[0].Position.Line)
			assert.Equal(t, SeverityError, sae.Issues[0].Severity)
			assert.Equal(t, res.Errors(), sae.Issues, "same issues Validate reports as errors")
			assert.Contains(t, err.Error(), tc.wantTag)
			assert.Contains(t, err.Error(), ErrMsgStrictAttributes)

			var ce *cuserr.CustomError
			require.True(t, errors.As(err, &ce))
			assert.Equal(t, ErrCodeValidation, ce.Code)
			tag, _ := ce.GetMetadata(MetaKeyTag)
			assert.Equal(t, tc.wantTag, tag)
		})
	}
}

// Strict mode judges the DOCUMENT, not a render: no error strategy, engine-wide or onerror=, lifts it.
func TestWithStrictAttributes_IgnoresErrorStrategy(t *testing.T) {
	_, err := MustNew(WithErrorStrategy(ErrorStrategyRemove), WithStrictAttributes()).Parse(strictIncludeNoTemplate)
	require.ErrorIs(t, err, ErrStrictAttributes)
	_, err = MustNew(WithStrictAttributes()).Parse(`{~exons.include ref="x" onerror="remove" /~}`)
	require.ErrorIs(t, err, ErrStrictAttributes)
}

func TestWithStrictAttributes_ParseBody(t *testing.T) {
	strict := MustNew(WithStrictAttributes())
	_, err := strict.ParseBody(strictIncludeNoTemplate)
	require.ErrorIs(t, err, ErrStrictAttributes)

	// A refused body must not be cached as a success.
	_, err = strict.ParseBody(strictIncludeNoTemplate)
	require.ErrorIs(t, err, ErrStrictAttributes)

	_, err = strict.ParseBody(strictWellFormed)
	require.NoError(t, err)

	_, err = MustNew().ParseBody(strictIncludeNoTemplate)
	require.NoError(t, err)
}

func TestWithStrictAttributes_AllIssuesCarried(t *testing.T) {
	_, err := MustNew(WithStrictAttributes()).Parse(strictTwoIssues)
	var sae *StrictAttributeError
	require.ErrorAs(t, err, &sae)
	require.Len(t, sae.Issues, 2)
	assert.Equal(t, TagNameInclude, sae.Issues[0].TagName)
	assert.Equal(t, TagNameMessage, sae.Issues[1].TagName)
	assert.Contains(t, sae.Error(), "(and 1 more)")
}

func TestWithStrictAttributes_RegisterTemplate(t *testing.T) {
	err := MustNew(WithStrictAttributes()).RegisterTemplate("bad", strictMessageNoRole)
	require.ErrorIs(t, err, ErrStrictAttributes)
	require.NoError(t, MustNew().RegisterTemplate("bad", strictMessageNoRole))
}

// Frontmatter tags are rendered through Parse during config extraction, so strict mode checks them.
// Under the default throw strategy that render already refuses the tag; under a lenient strategy it
// does not, and strict mode is what refuses it.
func TestWithStrictAttributes_Frontmatter(t *testing.T) {
	_, err := MustNew(WithErrorStrategy(ErrorStrategyKeepRaw)).Parse(strictFrontmatterBadTag)
	require.NoError(t, err, "lenient frontmatter render keeps the refused tag raw")

	_, err = MustNew(WithErrorStrategy(ErrorStrategyKeepRaw), WithStrictAttributes()).Parse(strictFrontmatterBadTag)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrStrictAttributes))
	assert.Contains(t, err.Error(), ErrMsgFrontmatterParse, "arrives as the frontmatter error")
	assert.Contains(t, err.Error(), TagNameInclude)
}

// The default engine's render still fails on these documents, which is why the option exists.
func TestWithStrictAttributes_DefaultRenderStillFails(t *testing.T) {
	tmpl, err := MustNew().Parse(strictIncludeNoTemplate)
	require.NoError(t, err)
	_, err = tmpl.Execute(context.Background(), nil)
	require.Error(t, err)
}

func TestStrictAttributeError_Shape(t *testing.T) {
	assert.Equal(t, ErrMsgStrictAttributes, (&StrictAttributeError{}).Error())
	assert.False(t, (&StrictAttributeError{}).Is(errors.New("other")))
	err := NewStrictAttributeError(nil)
	assert.ErrorIs(t, err, ErrStrictAttributes)
}

func TestValidationResult_ErrorsAndWarnings(t *testing.T) {
	res, err := MustNew().Validate(strictIncludeNoTemplate + strictUnknownTagOnly)
	require.NoError(t, err)

	errs := res.Errors()
	require.Len(t, errs, 1)
	assert.Equal(t, SeverityError, errs[0].Severity)
	assert.Equal(t, TagNameInclude, errs[0].TagName)

	warns := res.Warnings()
	require.Len(t, warns, 1)
	assert.Equal(t, SeverityWarning, warns[0].Severity)
	assert.Equal(t, ErrMsgUnknownTagInTemplate, warns[0].Message)

	assert.Len(t, res.Issues(), len(errs)+len(warns), "the two filters partition Issues")

	clean, err := MustNew().Validate(strictWellFormed)
	require.NoError(t, err)
	assert.Empty(t, clean.Errors())
	assert.Empty(t, clean.Warnings())
}
