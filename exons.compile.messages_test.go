package exons

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// CompiledSpec.ToOpenAIMessages
// =============================================================================

func TestCompiledSpec_ToOpenAIMessages_Nil(t *testing.T) {
	var cs *CompiledSpec
	result := cs.ToOpenAIMessages()
	assert.Nil(t, result)
}

func TestCompiledSpec_ToOpenAIMessages_Empty(t *testing.T) {
	cs := &CompiledSpec{}
	result := cs.ToOpenAIMessages()
	assert.Nil(t, result)
}

func TestCompiledSpec_ToOpenAIMessages_SystemAndUser(t *testing.T) {
	cs := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleSystem, Content: "You are helpful."},
			{Role: RoleUser, Content: "Hello!"},
		},
	}

	result := cs.ToOpenAIMessages()
	require.Len(t, result, 2)

	assert.Equal(t, RoleSystem, result[0][AttrRole])
	assert.Equal(t, "You are helpful.", result[0][ProviderMsgKeyContent])

	assert.Equal(t, RoleUser, result[1][AttrRole])
	assert.Equal(t, "Hello!", result[1][ProviderMsgKeyContent])
}

func TestCompiledSpec_ToOpenAIMessages_AllRoles(t *testing.T) {
	cs := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleSystem, Content: "System"},
			{Role: RoleUser, Content: "User"},
			{Role: RoleAssistant, Content: "Assistant"},
			{Role: RoleTool, Content: "Tool"},
		},
	}

	result := cs.ToOpenAIMessages()
	require.Len(t, result, 4)
	assert.Equal(t, RoleSystem, result[0][AttrRole])
	assert.Equal(t, RoleUser, result[1][AttrRole])
	assert.Equal(t, RoleAssistant, result[2][AttrRole])
	assert.Equal(t, RoleTool, result[3][AttrRole])
}

// =============================================================================
// CompiledSpec.ToAnthropicMessages
// =============================================================================

func TestCompiledSpec_ToAnthropicMessages_Nil(t *testing.T) {
	var cs *CompiledSpec
	result := cs.ToAnthropicMessages()
	assert.Nil(t, result)
}

func TestCompiledSpec_ToAnthropicMessages_Empty(t *testing.T) {
	cs := &CompiledSpec{}
	result := cs.ToAnthropicMessages()
	assert.Nil(t, result)
}

func TestCompiledSpec_ToAnthropicMessages_SystemExtracted(t *testing.T) {
	cs := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleSystem, Content: "You are a research assistant."},
			{Role: RoleUser, Content: "Find info about Go."},
		},
	}

	result := cs.ToAnthropicMessages()
	require.NotNil(t, result)

	// System extracted to top-level key
	system, ok := result[ProviderMsgKeySystem]
	require.True(t, ok)
	assert.Equal(t, "You are a research assistant.", system)

	// Non-system in messages array
	messages, ok := result[ProviderMsgKeyMessages]
	require.True(t, ok)
	msgs, ok := messages.([]map[string]any)
	require.True(t, ok)
	require.Len(t, msgs, 1)
	assert.Equal(t, RoleUser, msgs[0][AttrRole])
	assert.Equal(t, "Find info about Go.", msgs[0][ProviderMsgKeyContent])
}

func TestCompiledSpec_ToAnthropicMessages_MultipleSystemJoined(t *testing.T) {
	cs := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleSystem, Content: "First system instruction."},
			{Role: RoleSystem, Content: "Second system instruction."},
			{Role: RoleUser, Content: "Hello"},
		},
	}

	result := cs.ToAnthropicMessages()
	require.NotNil(t, result)

	system, ok := result[ProviderMsgKeySystem]
	require.True(t, ok)
	assert.Equal(t, "First system instruction.\n\nSecond system instruction.", system)
}

func TestCompiledSpec_ToAnthropicMessages_NoSystem(t *testing.T) {
	cs := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleUser, Content: "Hello"},
			{Role: RoleAssistant, Content: "Hi there"},
		},
	}

	result := cs.ToAnthropicMessages()
	require.NotNil(t, result)

	// No system key
	_, hasSystem := result[ProviderMsgKeySystem]
	assert.False(t, hasSystem)

	// Messages present
	messages, ok := result[ProviderMsgKeyMessages]
	require.True(t, ok)
	msgs, ok := messages.([]map[string]any)
	require.True(t, ok)
	require.Len(t, msgs, 2)
}

func TestCompiledSpec_ToAnthropicMessages_OnlySystem(t *testing.T) {
	cs := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleSystem, Content: "Just a system message."},
		},
	}

	result := cs.ToAnthropicMessages()
	require.NotNil(t, result)

	system, ok := result[ProviderMsgKeySystem]
	require.True(t, ok)
	assert.Equal(t, "Just a system message.", system)

	// No messages key since there are no non-system messages
	_, hasMessages := result[ProviderMsgKeyMessages]
	assert.False(t, hasMessages)
}

// =============================================================================
// CompiledSpec.ToGeminiContents
// =============================================================================

func TestCompiledSpec_ToGeminiContents_Nil(t *testing.T) {
	var cs *CompiledSpec
	result := cs.ToGeminiContents()
	assert.Nil(t, result)
}

func TestCompiledSpec_ToGeminiContents_Empty(t *testing.T) {
	cs := &CompiledSpec{}
	result := cs.ToGeminiContents()
	assert.Nil(t, result)
}

func TestCompiledSpec_ToGeminiContents_SystemInstruction(t *testing.T) {
	cs := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleSystem, Content: "You are a helpful AI."},
			{Role: RoleUser, Content: "Tell me a joke."},
		},
	}

	result := cs.ToGeminiContents()
	require.NotNil(t, result)

	// System instruction
	si, ok := result[ProviderMsgKeySystemInstruction]
	require.True(t, ok)
	siMap, ok := si.(map[string]any)
	require.True(t, ok)
	parts, ok := siMap[ProviderMsgKeyParts].([]map[string]any)
	require.True(t, ok)
	require.Len(t, parts, 1)
	assert.Equal(t, "You are a helpful AI.", parts[0][ProviderMsgKeyText])

	// Contents
	contents, ok := result[ProviderMsgKeyContents]
	require.True(t, ok)
	contentArr, ok := contents.([]map[string]any)
	require.True(t, ok)
	require.Len(t, contentArr, 1)
	assert.Equal(t, RoleUser, contentArr[0][AttrRole])
}

func TestCompiledSpec_ToGeminiContents_AssistantMappedToModel(t *testing.T) {
	cs := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleUser, Content: "Hello"},
			{Role: RoleAssistant, Content: "Hi!"},
		},
	}

	result := cs.ToGeminiContents()
	require.NotNil(t, result)

	contents, ok := result[ProviderMsgKeyContents]
	require.True(t, ok)
	contentArr, ok := contents.([]map[string]any)
	require.True(t, ok)
	require.Len(t, contentArr, 2)

	// First message: user
	assert.Equal(t, RoleUser, contentArr[0][AttrRole])

	// Second message: assistant → model
	assert.Equal(t, ProviderMsgKeyModelRole, contentArr[1][AttrRole])
}

func TestCompiledSpec_ToGeminiContents_NoSystem(t *testing.T) {
	cs := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleUser, Content: "Hello"},
		},
	}

	result := cs.ToGeminiContents()
	require.NotNil(t, result)

	_, hasSI := result[ProviderMsgKeySystemInstruction]
	assert.False(t, hasSI)

	contents, ok := result[ProviderMsgKeyContents]
	require.True(t, ok)
	contentArr, ok := contents.([]map[string]any)
	require.True(t, ok)
	require.Len(t, contentArr, 1)
}

func TestCompiledSpec_ToGeminiContents_MultipleSystemJoined(t *testing.T) {
	cs := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleSystem, Content: "Instruction one."},
			{Role: RoleSystem, Content: "Instruction two."},
			{Role: RoleUser, Content: "Go"},
		},
	}

	result := cs.ToGeminiContents()
	require.NotNil(t, result)

	si, ok := result[ProviderMsgKeySystemInstruction]
	require.True(t, ok)
	siMap, ok := si.(map[string]any)
	require.True(t, ok)
	parts, ok := siMap[ProviderMsgKeyParts].([]map[string]any)
	require.True(t, ok)
	require.Len(t, parts, 1)
	assert.Equal(t, "Instruction one.\n\nInstruction two.", parts[0][ProviderMsgKeyText])
}

// =============================================================================
// CompiledSpec.ToProviderMessages — routing
// =============================================================================

func TestCompiledSpec_ToProviderMessages_OpenAI(t *testing.T) {
	cs := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleUser, Content: "Hello"},
		},
	}

	result, err := cs.ToProviderMessages(ProviderOpenAI)
	require.NoError(t, err)
	msgs, ok := result.([]map[string]any)
	require.True(t, ok)
	require.Len(t, msgs, 1)
}

func TestCompiledSpec_ToProviderMessages_Azure(t *testing.T) {
	cs := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleUser, Content: "Hello"},
		},
	}

	result, err := cs.ToProviderMessages(ProviderAzure)
	require.NoError(t, err)
	msgs, ok := result.([]map[string]any)
	require.True(t, ok)
	require.Len(t, msgs, 1)
}

func TestCompiledSpec_ToProviderMessages_Anthropic(t *testing.T) {
	cs := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleSystem, Content: "System"},
			{Role: RoleUser, Content: "Hello"},
		},
	}

	result, err := cs.ToProviderMessages(ProviderAnthropic)
	require.NoError(t, err)
	m, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Contains(t, m, ProviderMsgKeySystem)
	assert.Contains(t, m, ProviderMsgKeyMessages)
}

func TestCompiledSpec_ToProviderMessages_Gemini(t *testing.T) {
	cs := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleUser, Content: "Hello"},
		},
	}

	for _, provider := range []string{ProviderGoogle, ProviderGemini, ProviderVertex} {
		t.Run(provider, func(t *testing.T) {
			result, err := cs.ToProviderMessages(provider)
			require.NoError(t, err)
			m, ok := result.(map[string]any)
			require.True(t, ok)
			assert.Contains(t, m, ProviderMsgKeyContents)
		})
	}
}

func TestCompiledSpec_ToProviderMessages_Unsupported(t *testing.T) {
	cs := &CompiledSpec{
		Messages: []CompiledMessage{
			{Role: RoleUser, Content: "Hello"},
		},
	}

	result, err := cs.ToProviderMessages("unknown-provider")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), ErrMsgUnsupportedMsgProvider)
}

func TestCompiledSpec_ToProviderMessages_NilCompiledSpec(t *testing.T) {
	var cs *CompiledSpec

	// OpenAI: nil
	result, err := cs.ToProviderMessages(ProviderOpenAI)
	require.NoError(t, err)
	assert.Nil(t, result)

	// Anthropic: nil
	result, err = cs.ToProviderMessages(ProviderAnthropic)
	require.NoError(t, err)
	assert.Nil(t, result)

	// Gemini: nil
	result, err = cs.ToProviderMessages(ProviderGemini)
	require.NoError(t, err)
	assert.Nil(t, result)
}
