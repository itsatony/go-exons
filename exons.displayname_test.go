package exons

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/itsatony/go-cuserr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// =============================================================================
// Spec.DisplayName (v0.38.0)
// =============================================================================

const displayNameDoc = `---
name: churn-analyst
display_name: Churn Analyst (DACH)
description: Finds why customers leave
type: agent
---
Body.
`

func TestDisplayName_ParsesAsTypedFieldNotExtension(t *testing.T) {
	spec, err := ParseYAMLSpec("name: churn-analyst\ndisplay_name: Churn Analyst\ndescription: d\ntype: agent\n")
	require.NoError(t, err)
	assert.Equal(t, "Churn Analyst", spec.DisplayName)
	// X13's trap in reverse: a typed field consumes its key from Extensions.
	_, inExtensions := spec.Extensions[SpecFieldDisplayName]
	assert.False(t, inExtensions, "display_name must not also land in Extensions")
	require.NoError(t, spec.Validate())
}

func TestDisplayName_MaxLengthCountsCharacters(t *testing.T) {
	base := func(dn string) *Spec {
		return &Spec{Name: "doc", Description: "d", Type: DocumentTypeAgent, DisplayName: dn}
	}
	// 80 umlauts are 160 bytes: a byte cap would refuse them.
	require.NoError(t, base(strings.Repeat("ä", SpecDisplayNameMaxLength)).Validate())

	err := base(strings.Repeat("ä", SpecDisplayNameMaxLength+1)).Validate()
	require.Error(t, err)
	var ce *cuserr.CustomError
	require.True(t, errors.As(err, &ce))
	assert.Contains(t, err.Error(), ErrMsgSpecDisplayNameTooLong)
	md, _ := ce.GetMetadata(MetaKeyMaxLength)
	assert.Equal(t, "80", md)
}

func TestDisplayName_OptionalAndFreeForm(t *testing.T) {
	for _, dn := range []string{"", "Upper Lower", "Kündigungs-Analyst 2.0 — Beta!", "@not/a-ref"} {
		s := &Spec{Name: "doc", Description: "d", Type: DocumentTypeSkill, DisplayName: dn}
		assert.NoError(t, s.Validate(), "display_name %q", dn)
	}
}

func TestEffectiveDisplayName(t *testing.T) {
	assert.Equal(t, "", (*Spec)(nil).EffectiveDisplayName())
	assert.Equal(t, "slug", (&Spec{Name: "slug"}).EffectiveDisplayName())
	assert.Equal(t, "slug", (&Spec{Name: "slug", DisplayName: "   "}).EffectiveDisplayName())
	assert.Equal(t, "Nice Name", (&Spec{Name: "slug", DisplayName: "  Nice Name \n"}).EffectiveDisplayName())
}

func TestDisplayName_Clone(t *testing.T) {
	s := &Spec{Name: "a", DisplayName: "A"}
	assert.Equal(t, "A", s.Clone().DisplayName)
}

func TestDisplayName_SerializeFullKeepsAgentSkillsStrips(t *testing.T) {
	s, err := Parse([]byte(displayNameDoc))
	require.NoError(t, err)

	full, err := s.ExportFull()
	require.NoError(t, err)
	fm := frontmatterMap(t, full)
	assert.Equal(t, "Churn Analyst (DACH)", fm[SpecFieldDisplayName])
	_, dupe := fm["extensions"]
	assert.False(t, dupe)

	skillExport, err := s.ExportAgentSkill()
	require.NoError(t, err)
	_, present := frontmatterMap(t, skillExport)[SpecFieldDisplayName]
	assert.False(t, present, "the Agent Skills export's top-level keys are a closed set")
}

func TestDisplayName_ExtensionKeyCannotOverwrite(t *testing.T) {
	s := &Spec{Name: "a", Description: "d", DisplayName: "Typed",
		Extensions: map[string]any{SpecFieldDisplayName: "Stale"}}
	out, err := s.ExportFull()
	require.NoError(t, err)
	assert.Equal(t, "Typed", frontmatterMap(t, out)[SpecFieldDisplayName])
}

func TestDisplayName_A2ACardUsesItWithSlugFallback(t *testing.T) {
	withDN := &Spec{Name: "churn-analyst", DisplayName: "Churn Analyst", Description: "d", Type: DocumentTypeAgent}
	card, err := withDN.CompileAgentCard(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "Churn Analyst", card.Name)
	require.Len(t, card.Skills, 1)
	assert.Equal(t, "churn-analyst", card.Skills[0].ID, "the synthesized skill's ID stays the slug")
	assert.Equal(t, "Churn Analyst", card.Skills[0].Name)

	bare := &Spec{Name: "churn-analyst", Description: "d", Type: DocumentTypeAgent}
	card, err = bare.CompileAgentCard(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "churn-analyst", card.Name)
}

// frontmatterMap decodes the YAML frontmatter of a serialized document.
func frontmatterMap(t *testing.T, doc []byte) map[string]any {
	t.Helper()
	parts := strings.SplitN(string(doc), YAMLFrontmatterDelimiter+"\n", 3)
	require.Len(t, parts, 3)
	var m map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(parts[1]), &m))
	return m
}
