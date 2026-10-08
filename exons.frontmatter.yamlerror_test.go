package exons

import (
	"errors"
	"fmt"
	"testing"

	"github.com/itsatony/go-cuserr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// requireLocatedParseError asserts the stable half of a frontmatter parse error — the
// code, the message prefix, the cause — and returns it for the specific claim.
func requireLocatedParseError(t *testing.T, err error, docLine, fmLine int, key string) *cuserr.CustomError {
	t.Helper()
	require.Error(t, err)
	var ce *cuserr.CustomError
	require.True(t, errors.As(err, &ce), "a frontmatter parse error is a cuserr: %v", err)
	assert.Contains(t, ce.Message, ErrCodeConfig+": "+ErrMsgFrontmatterParse+": ",
		"the code and the prefix consumers match on are unchanged")

	line, ok := ce.GetMetadata(MetaKeyLine)
	require.True(t, ok)
	assert.Equal(t, fmt.Sprint(docLine), line, "MetaKeyLine is the DOCUMENT line")
	yl, ok := ce.GetMetadata(MetaKeyFrontmatterLine)
	require.True(t, ok)
	assert.Equal(t, fmt.Sprint(fmLine), yl, "MetaKeyFrontmatterLine is yaml.v3's own line")

	gotKey, ok := ce.GetMetadata(MetaKeyFrontmatterKey)
	if key == "" {
		assert.False(t, ok, "no key is invented")
	} else {
		require.True(t, ok)
		assert.Equal(t, key, gotKey)
	}
	return ce
}

// The issue's exact example (vAudience/aigentverse#258, nexus2_issue_reports#28): a
// German prose description containing ": ".
func TestFrontmatterParseError_AnUnquotedColonNamesTheKeyAndTheFix(t *testing.T) {
	doc := "---\nname: wochenrueckblick\ntype: prompt\ndescription: Wochenrückblick für den Kurs, gegliedert in Einzelmeldungen: Aktuelles aus der Branche\n---\nbody"
	_, err := Parse([]byte(doc))
	requireLocatedParseError(t, err, 4, 3, "description")

	assert.Equal(t,
		`EXONS_CONFIG: failed to parse YAML frontmatter: line 4 (frontmatter line 3), key "description": `+
			`the value of "description" contains ": " — wrap it in quotes, or write it as a block scalar (description: >-): `+
			`yaml: line 3: mapping values are not allowed in this context`,
		err.Error())
}

// The ": " on a continuation line of a multi-line plain scalar: the owner is the key
// above, and since the line could also be a mis-indented new key, both fixes are named.
func TestFrontmatterParseError_AColonOnAContinuationLineNamesTheOwningKey(t *testing.T) {
	doc := "---\nname: kurs\ndescription: Wochenrückblick mit\n  Einzelmeldungen: Aktuelles\n---\nbody"
	_, err := Parse([]byte(doc))
	requireLocatedParseError(t, err, 4, 3, "description")
	assert.Contains(t, err.Error(), `the value of "description" contains ": "`)
	assert.Contains(t, err.Error(), "; or, if line 4 starts a new key, fix its indentation")
}

func TestFrontmatterParseError_ANestedKeyIsNamedByItsPath(t *testing.T) {
	doc := "---\nname: kurs\ndescription: d\ntype: prompt\ninputs:\n  ton:\n    type: select\n    description: Ton: sachlich oder locker\n---\nbody"
	_, err := Parse([]byte(doc))
	requireLocatedParseError(t, err, 8, 7, "inputs.ton.description")
	assert.Contains(t, err.Error(), `(description: >-)`, "the fix names the LEAF key the author writes")
}

// A BOM (and leading spaces) before `---` is trimmed by Parse and shifts no line.
func TestFrontmatterParseError_ABOMPrefixedDocumentKeepsItsLines(t *testing.T) {
	doc := "\xef\xbb\xbf---\nname: kurs\ndescription: Einzelmeldungen: Aktuelles\n---\nbody"
	_, err := Parse([]byte(doc))
	requireLocatedParseError(t, err, 3, 2, "description")

	crlf := "---\r\nname: kurs\r\ndescription: Einzelmeldungen: Aktuelles\r\n---\r\nbody"
	_, err = Parse([]byte(crlf))
	requireLocatedParseError(t, err, 3, 2, "description")
}

// When something follows `---` on its own line, the frontmatter text starts ON line 1.
func TestFrontmatterParseError_TextAfterTheOpeningDelimiterIsLineOne(t *testing.T) {
	_, err := Parse([]byte("--- \nname: a: b\n---\nbody"))
	ce := requireLocatedParseError(t, err, 2, 2, "name")
	assert.NotContains(t, ce.Message, "frontmatter line", "the two lines agree, so only one is stated")
}

// A type error (the issue's first example is now accepted, so use one that is still
// wrong): the key path is named and yaml.v3's TypeError survives as the cause.
func TestFrontmatterParseError_ATypeErrorNamesTheKey(t *testing.T) {
	doc := "---\nname: kurs\ndescription: d\ntype: prompt\ninputs:\n  ton:\n    type: select\n    options:\n      - [sachlich, locker]\n---\nbody"
	_, err := Parse([]byte(doc))
	requireLocatedParseError(t, err, 9, 8, "inputs.ton.options")
	assert.Contains(t, err.Error(), "line 8: an option must be a string or a {value, label} mapping, not a sequence")
	assert.NotContains(t, err.Error(), "wrap it in quotes", "the quoting fix belongs to the ': ' error only")

	var te *yaml.TypeError
	assert.True(t, errors.As(err, &te), "errors.As still reaches yaml.v3's TypeError")

	_, err = Parse([]byte("---\nname: kurs\ndescription: d\ntype: [prompt]\n---\nbody"))
	requireLocatedParseError(t, err, 4, 3, "type")
}

// The issue's first example, which used to be the type error, now parses.
func TestFrontmatterParseError_TheIssuesStringListNoLongerFails(t *testing.T) {
	doc := "---\nname: kurs\ndescription: d\ntype: prompt\ninputs:\n  ton:\n    type: select\n    options: [sachlich, locker, begeistert]\n---\nbody"
	spec, err := Parse([]byte(doc))
	require.NoError(t, err)
	assert.Len(t, spec.Inputs["ton"].Options, 3)
}

// ParseYAMLSpec is handed bare YAML, so its lines ARE document lines.
func TestFrontmatterParseError_ParseYAMLSpecReportsItsOwnLines(t *testing.T) {
	_, err := ParseYAMLSpec("name: kurs\ndescription: Einzelmeldungen: Aktuelles\n")
	ce := requireLocatedParseError(t, err, 2, 2, "description")
	assert.NotContains(t, ce.Message, "frontmatter line")
}

// An error with no line keeps exactly the pre-v0.44.0 shape.
func TestFrontmatterParseError_AnErrorWithoutALineIsWrappedAsBefore(t *testing.T) {
	cause := errors.New("yaml: invalid leading UTF-8 octet")
	got := NewFrontmatterParseErrorAt(cause, "x", 2)
	assert.Equal(t, NewFrontmatterParseError(cause).Error(), got.Error())
	assert.Equal(t, NewFrontmatterParseError(nil), NewFrontmatterParseErrorAt(nil, "", 1))
}

func TestFrontmatterKeyAt(t *testing.T) {
	fm := "name: a\n\"quoted key\": b\nrequirements:\n  resources:\n    - ref: r\n      kind: true\n# a: comment\nlist:\n- x\n"
	cases := []struct {
		line          int
		unquotedColon bool
		path, leaf    string
		keyLine       int
	}{
		{1, false, "name", "name", 1},
		{2, false, "quoted key", "quoted key", 2},
		{5, false, "requirements.resources.ref", "ref", 5},
		{6, false, "requirements.resources.kind", "kind", 6},
		{7, false, "", "", 0},         // a comment line is attributed to nothing
		{9, false, "list", "list", 8}, // a sequence item at its key's own indentation
		{1, true, "", "", 0},          // `name: a` has no further ": ", and nothing is above it
		{0, false, "", "", 0},         // out of range
		{99, false, "", "", 0},        // out of range
	}
	for _, tc := range cases {
		path, leaf, keyLine := frontmatterKeyAt(fm, tc.line, tc.unquotedColon)
		assert.Equal(t, tc.path, path, "line %d", tc.line)
		assert.Equal(t, tc.leaf, leaf, "line %d", tc.line)
		assert.Equal(t, tc.keyLine, keyLine, "line %d", tc.line)
	}
}
