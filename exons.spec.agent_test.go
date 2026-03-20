package exons

import (
	"testing"

	"github.com/itsatony/go-exons/execution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Spec.ValidateAsAgent
// =============================================================================

func TestValidateAsAgent_Valid(t *testing.T) {
	t.Run("agent with execution and body", func(t *testing.T) {
		spec := &Spec{
			Name: "test-agent",
			Type: DocumentTypeAgent,
			Execution: &execution.Config{
				Provider: ProviderOpenAI,
				Model:    "gpt-4",
			},
			Body: "Hello, I am an agent.",
		}
		err := spec.ValidateAsAgent()
		assert.NoError(t, err)
	})

	t.Run("agent with execution and messages", func(t *testing.T) {
		spec := &Spec{
			Name: "chat-agent",
			Type: DocumentTypeAgent,
			Execution: &execution.Config{
				Provider: ProviderAnthropic,
				Model:    "claude-sonnet-4-5",
			},
			Messages: []MessageTemplate{
				{Role: RoleSystem, Content: "You are a helpful assistant."},
				{Role: RoleUser, Content: "Hello"},
			},
		}
		err := spec.ValidateAsAgent()
		assert.NoError(t, err)
	})

	t.Run("agent with execution, body, and messages", func(t *testing.T) {
		spec := &Spec{
			Name: "full-agent",
			Type: DocumentTypeAgent,
			Execution: &execution.Config{
				Provider: ProviderOpenAI,
				Model:    "gpt-4",
			},
			Body: "Some body content",
			Messages: []MessageTemplate{
				{Role: RoleSystem, Content: "System message"},
			},
		}
		err := spec.ValidateAsAgent()
		assert.NoError(t, err)
	})
}

func TestValidateAsAgent_NilSpec(t *testing.T) {
	var spec *Spec
	err := spec.ValidateAsAgent()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgSpecNil)
}

func TestValidateAsAgent_NotAgent(t *testing.T) {
	t.Run("skill type returns error", func(t *testing.T) {
		spec := &Spec{
			Name: "my-skill",
			Type: DocumentTypeSkill,
			Execution: &execution.Config{
				Provider: ProviderOpenAI,
				Model:    "gpt-4",
			},
			Body: "skill body",
		}
		err := spec.ValidateAsAgent()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgNotAnAgent)
	})

	t.Run("prompt type returns error", func(t *testing.T) {
		spec := &Spec{
			Name: "my-prompt",
			Type: DocumentTypePrompt,
			Execution: &execution.Config{
				Provider: ProviderOpenAI,
				Model:    "gpt-4",
			},
			Body: "prompt body",
		}
		err := spec.ValidateAsAgent()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgNotAnAgent)
	})

	t.Run("default type (empty) is skill, returns error", func(t *testing.T) {
		spec := &Spec{
			Name: "no-type",
			Execution: &execution.Config{
				Provider: ProviderOpenAI,
				Model:    "gpt-4",
			},
			Body: "body",
		}
		err := spec.ValidateAsAgent()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgNotAnAgent)
	})
}

func TestValidateAsAgent_NoExecution(t *testing.T) {
	spec := &Spec{
		Name: "no-exec-agent",
		Type: DocumentTypeAgent,
		Body: "agent body",
	}
	err := spec.ValidateAsAgent()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgNoExecutionConfig)
}

func TestValidateAsAgent_NoBodyOrMessages(t *testing.T) {
	spec := &Spec{
		Name: "empty-agent",
		Type: DocumentTypeAgent,
		Execution: &execution.Config{
			Provider: ProviderOpenAI,
			Model:    "gpt-4",
		},
	}
	err := spec.ValidateAsAgent()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgAgentNoBodyOrMessages)
}

func TestValidateAsAgent_EmptyMessages(t *testing.T) {
	spec := &Spec{
		Name: "empty-messages-agent",
		Type: DocumentTypeAgent,
		Execution: &execution.Config{
			Provider: ProviderOpenAI,
			Model:    "gpt-4",
		},
		Messages: []MessageTemplate{},
	}
	err := spec.ValidateAsAgent()
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgAgentNoBodyOrMessages)
}
