package exons

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// ExecuteAndExtractMessages Tests
// =============================================================================

func TestTemplate_ExecuteAndExtractMessages(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("single user message", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="user"~}Hello!{~/exons.message~}`)
		require.NoError(t, err)
		msgs, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Equal(t, RoleUser, msgs[0].Role)
		assert.Equal(t, "Hello!", msgs[0].Content)
		assert.False(t, msgs[0].Cache)
	})

	t.Run("single system message", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="system"~}You are an assistant.{~/exons.message~}`)
		require.NoError(t, err)
		msgs, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Equal(t, RoleSystem, msgs[0].Role)
		assert.Equal(t, "You are an assistant.", msgs[0].Content)
	})

	t.Run("single assistant message", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="assistant"~}I can help!{~/exons.message~}`)
		require.NoError(t, err)
		msgs, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Equal(t, RoleAssistant, msgs[0].Role)
		assert.Equal(t, "I can help!", msgs[0].Content)
	})

	t.Run("single tool message", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="tool"~}Tool result here{~/exons.message~}`)
		require.NoError(t, err)
		msgs, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Equal(t, RoleTool, msgs[0].Role)
		assert.Equal(t, "Tool result here", msgs[0].Content)
	})

	t.Run("multi message conversation", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="system"~}System prompt{~/exons.message~}
{~exons.message role="user"~}User question{~/exons.message~}
{~exons.message role="assistant"~}Assistant answer{~/exons.message~}`)
		require.NoError(t, err)
		msgs, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
		require.NoError(t, err)
		require.Len(t, msgs, 3)
		assert.Equal(t, RoleSystem, msgs[0].Role)
		assert.Equal(t, RoleUser, msgs[1].Role)
		assert.Equal(t, RoleAssistant, msgs[2].Role)
		assert.Equal(t, "System prompt", msgs[0].Content)
		assert.Equal(t, "User question", msgs[1].Content)
		assert.Equal(t, "Assistant answer", msgs[2].Content)
	})

	t.Run("message with cache attribute true", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="system" cache="true"~}Cached{~/exons.message~}`)
		require.NoError(t, err)
		msgs, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.True(t, msgs[0].Cache)
	})

	t.Run("message with cache attribute false", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="user" cache="false"~}Not cached{~/exons.message~}`)
		require.NoError(t, err)
		msgs, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.False(t, msgs[0].Cache)
	})

	t.Run("message with variable interpolation", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="user"~}{~exons.var name="query" /~}{~/exons.message~}`)
		require.NoError(t, err)
		msgs, err := tmpl.ExecuteAndExtractMessages(ctx, map[string]any{"query": "What is Go?"})
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Equal(t, "What is Go?", msgs[0].Content)
	})

	t.Run("message with nested content", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="system"~}Hello {~exons.var name="name" /~}, you are {~exons.if eval="admin"~}an admin{~exons.else~}a user{~/exons.if~}.{~/exons.message~}`)
		require.NoError(t, err)
		msgs, err := tmpl.ExecuteAndExtractMessages(ctx, map[string]any{
			"name":  "Alice",
			"admin": true,
		})
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Equal(t, "Hello Alice, you are an admin.", msgs[0].Content)
	})

	t.Run("message with loop content", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="user"~}Items: {~exons.for item="x" in="items"~}{~exons.var name="x" /~} {~/exons.for~}{~/exons.message~}`)
		require.NoError(t, err)
		msgs, err := tmpl.ExecuteAndExtractMessages(ctx, map[string]any{
			"items": []any{"a", "b", "c"},
		})
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		assert.Contains(t, msgs[0].Content, "a")
		assert.Contains(t, msgs[0].Content, "b")
		assert.Contains(t, msgs[0].Content, "c")
	})

	t.Run("messages with text between them", func(t *testing.T) {
		tmpl, err := engine.Parse(`preamble text
{~exons.message role="system"~}System{~/exons.message~}
middle text
{~exons.message role="user"~}User{~/exons.message~}
trailing text`)
		require.NoError(t, err)
		msgs, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
		require.NoError(t, err)
		require.Len(t, msgs, 2)
		assert.Equal(t, "System", msgs[0].Content)
		assert.Equal(t, "User", msgs[1].Content)
	})

	t.Run("execution error propagates", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="user"~}{~exons.var name="missing" /~}{~/exons.message~}`)
		require.NoError(t, err)
		_, err = tmpl.ExecuteAndExtractMessages(ctx, nil)
		assert.Error(t, err)
	})
}

// =============================================================================
// ExtractMessagesFromOutput Tests
// =============================================================================

func TestExtractMessagesFromOutput_Detailed(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("returns nil for plain text", func(t *testing.T) {
		msgs := ExtractMessagesFromOutput("just text without messages")
		assert.Nil(t, msgs)
	})

	t.Run("returns nil for empty string", func(t *testing.T) {
		msgs := ExtractMessagesFromOutput("")
		assert.Nil(t, msgs)
	})

	t.Run("extracts messages from executed output", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="user"~}Hello{~/exons.message~}`)
		require.NoError(t, err)
		output, err := tmpl.Execute(ctx, nil)
		require.NoError(t, err)
		msgs := ExtractMessagesFromOutput(output)
		require.NotNil(t, msgs)
		require.Len(t, msgs, 1)
		assert.Equal(t, RoleUser, msgs[0].Role)
		assert.Equal(t, "Hello", msgs[0].Content)
	})

	t.Run("extracts multiple messages from output", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.message role="system"~}Sys{~/exons.message~}{~exons.message role="user"~}Usr{~/exons.message~}`)
		require.NoError(t, err)
		output, err := tmpl.Execute(ctx, nil)
		require.NoError(t, err)
		msgs := ExtractMessagesFromOutput(output)
		require.NotNil(t, msgs)
		require.Len(t, msgs, 2)
		assert.Equal(t, RoleSystem, msgs[0].Role)
		assert.Equal(t, RoleUser, msgs[1].Role)
	})
}

// =============================================================================
// Message Struct Tests
// =============================================================================

func TestMessage_Fields(t *testing.T) {
	msg := Message{
		Role:    RoleSystem,
		Content: "Hello world",
		Cache:   true,
	}
	assert.Equal(t, RoleSystem, msg.Role)
	assert.Equal(t, "Hello world", msg.Content)
	assert.True(t, msg.Cache)
}

// =============================================================================
// Full Chat Template Tests
// =============================================================================

func TestTemplate_ChatTemplate(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("complete chat template with frontmatter", func(t *testing.T) {
		source := `---
name: chat-assistant
description: A chat assistant
type: skill
---
{~exons.message role="system"~}
You are a helpful assistant. Your name is {~exons.var name="bot_name" default="ExonsBot" /~}.
{~/exons.message~}
{~exons.message role="user"~}
{~exons.var name="user_input" /~}
{~/exons.message~}`

		tmpl, err := engine.Parse(source)
		require.NoError(t, err)
		assert.True(t, tmpl.HasSpec())

		msgs, err := tmpl.ExecuteAndExtractMessages(ctx, map[string]any{
			"user_input": "What can you do?",
		})
		require.NoError(t, err)
		require.Len(t, msgs, 2)
		assert.Equal(t, RoleSystem, msgs[0].Role)
		assert.Contains(t, msgs[0].Content, "ExonsBot")
		assert.Equal(t, RoleUser, msgs[1].Role)
		assert.Equal(t, "What can you do?", msgs[1].Content)
	})

	t.Run("four role conversation", func(t *testing.T) {
		source := `{~exons.message role="system"~}System prompt{~/exons.message~}
{~exons.message role="user"~}User query{~/exons.message~}
{~exons.message role="assistant"~}Let me search for that.{~/exons.message~}
{~exons.message role="tool"~}Search result: found 3 items{~/exons.message~}`

		tmpl, err := engine.Parse(source)
		require.NoError(t, err)
		msgs, err := tmpl.ExecuteAndExtractMessages(ctx, nil)
		require.NoError(t, err)
		require.Len(t, msgs, 4)
		assert.Equal(t, RoleSystem, msgs[0].Role)
		assert.Equal(t, RoleUser, msgs[1].Role)
		assert.Equal(t, RoleAssistant, msgs[2].Role)
		assert.Equal(t, RoleTool, msgs[3].Role)
	})
}
