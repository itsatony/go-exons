package exons

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// InputDef Label + Options, and Spec.RecommendedAgents (v0.16.0)
// =============================================================================

func TestParseYAMLSpec_InputLabelAndOptions(t *testing.T) {
	yaml := `
name: tone-picker
description: A prompt with a typed select input
type: prompt
inputs:
  tone:
    type: select
    label: Tone of voice
    required: true
    options:
      - value: formal
        label: Formal
      - value: casual
        label: Casual
  tags:
    type: multiselect
    options:
      - value: a
      - value: b
`
	spec, err := ParseYAMLSpec(yaml)
	require.NoError(t, err)
	require.NotNil(t, spec)

	require.Contains(t, spec.Inputs, "tone")
	tone := spec.Inputs["tone"]
	assert.Equal(t, InputTypeSelect, tone.Type)
	assert.Equal(t, "Tone of voice", tone.Label)
	assert.True(t, tone.Required)
	require.Len(t, tone.Options, 2)
	assert.Equal(t, "formal", tone.Options[0].Value)
	assert.Equal(t, "Formal", tone.Options[0].Label)

	require.Contains(t, spec.Inputs, "tags")
	tags := spec.Inputs["tags"]
	assert.Equal(t, InputTypeMultiselect, tags.Type)
	require.Len(t, tags.Options, 2)
	assert.Equal(t, "a", tags.Options[0].Value)
	assert.Empty(t, tags.Options[0].Label) // label falls back to value at the consumer
}

func TestParseYAMLSpec_RecommendedAgents(t *testing.T) {
	yaml := `
name: churn-deep-dive
description: A prompt made for specific coworkers
type: prompt
recommended_agents:
  - "@vai/mary"
  - "@vai/otto"
`
	spec, err := ParseYAMLSpec(yaml)
	require.NoError(t, err)
	require.NotNil(t, spec)
	require.Len(t, spec.RecommendedAgents, 2)
	assert.Equal(t, "@vai/mary", spec.RecommendedAgents[0])
	assert.Equal(t, "@vai/otto", spec.RecommendedAgents[1])
	// Must not leak into the inline Extensions catch-all.
	assert.NotContains(t, spec.Extensions, SpecFieldRecommendedAgents)
}

func TestSpec_Clone_OptionsAndRecommendedAgents(t *testing.T) {
	s := &Spec{
		Name: "test",
		Inputs: map[string]*InputDef{
			"tone": {Type: InputTypeSelect, Options: []InputOption{{Value: "formal", Label: "Formal"}}},
		},
		RecommendedAgents: []string{"@vai/mary"},
	}
	clone := s.Clone()

	// Mutate the original after cloning.
	s.Inputs["tone"].Options[0].Value = "changed"
	s.RecommendedAgents[0] = "@vai/changed"

	require.Len(t, clone.Inputs["tone"].Options, 1)
	assert.Equal(t, "formal", clone.Inputs["tone"].Options[0].Value)
	require.Len(t, clone.RecommendedAgents, 1)
	assert.Equal(t, "@vai/mary", clone.RecommendedAgents[0])
}

func TestSerialize_RoundTrip_OptionsAndRecommendedAgents(t *testing.T) {
	orig := &Spec{
		Name:        "round-trip",
		Description: "desc",
		Type:        DocumentTypePrompt,
		Inputs: map[string]*InputDef{
			"tone": {
				Type:    InputTypeSelect,
				Label:   "Tone",
				Options: []InputOption{{Value: "formal", Label: "Formal"}, {Value: "casual", Label: "Casual"}},
			},
		},
		RecommendedAgents: []string{"@vai/mary", "@vai/otto"},
	}

	data, err := orig.Serialize(DefaultSerializeOptions())
	require.NoError(t, err)

	got, err := ParseYAMLSpec(string(data))
	require.NoError(t, err)
	require.Contains(t, got.Inputs, "tone")
	assert.Equal(t, "Tone", got.Inputs["tone"].Label)
	require.Len(t, got.Inputs["tone"].Options, 2)
	assert.Equal(t, "casual", got.Inputs["tone"].Options[1].Value)
	assert.Equal(t, orig.RecommendedAgents, got.RecommendedAgents)
}

func TestSerialize_AgentSkillsExport_StripsRecommendedAgents(t *testing.T) {
	// recommended_agents is a go-exons extension key → excluded from the stripped
	// Agent-Skills-compatible export (IncludeExtensions=false).
	s := &Spec{Name: "p", Description: "d", RecommendedAgents: []string{"@vai/mary"}}
	m := s.buildSerializeMap(AgentSkillsExportOptions())
	assert.NotContains(t, m, SpecFieldRecommendedAgents)

	full := s.buildSerializeMap(DefaultSerializeOptions())
	assert.Contains(t, full, SpecFieldRecommendedAgents)
}
