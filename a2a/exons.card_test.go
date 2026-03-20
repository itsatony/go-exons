package a2a

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentCard_ToJSON_NilReceiver(t *testing.T) {
	var card *AgentCard
	data, err := card.ToJSON()
	assert.NoError(t, err)
	assert.Nil(t, data)
}

func TestAgentCard_ToJSON_RoundTrip(t *testing.T) {
	card := &AgentCard{
		Name:            "test-agent",
		URL:             "https://example.com",
		ProtocolVersion: "0.3.0",
		Skills: []Skill{
			{ID: "search", Name: "search", Description: "Searches the web"},
		},
	}
	data, err := card.ToJSON()
	require.NoError(t, err)

	var parsed AgentCard
	require.NoError(t, json.Unmarshal(data, &parsed))
	assert.Equal(t, "test-agent", parsed.Name)
	assert.Equal(t, "https://example.com", parsed.URL)
	assert.Len(t, parsed.Skills, 1)
	assert.Equal(t, "search", parsed.Skills[0].ID)
}

func TestAgentCard_ToJSONPretty_NilReceiver(t *testing.T) {
	var card *AgentCard
	data, err := card.ToJSONPretty()
	assert.NoError(t, err)
	assert.Nil(t, data)
}

func TestAgentCard_ToJSONPretty_Indented(t *testing.T) {
	card := &AgentCard{
		Name:            "test-agent",
		URL:             "https://example.com",
		ProtocolVersion: "0.3.0",
	}
	data, err := card.ToJSONPretty()
	require.NoError(t, err)
	assert.Contains(t, string(data), "\n")
	assert.Contains(t, string(data), "  ")

	var parsed AgentCard
	require.NoError(t, json.Unmarshal(data, &parsed))
	assert.Equal(t, "test-agent", parsed.Name)
}

func TestAgentCard_ToJSON_OmitsEmptyFields(t *testing.T) {
	card := &AgentCard{
		Name:            "minimal",
		URL:             "https://example.com",
		ProtocolVersion: "0.3.0",
	}
	data, err := card.ToJSON()
	require.NoError(t, err)
	s := string(data)
	assert.NotContains(t, s, "description")
	assert.NotContains(t, s, "skills")
	assert.NotContains(t, s, "metadata")
	assert.NotContains(t, s, "provider")
}
