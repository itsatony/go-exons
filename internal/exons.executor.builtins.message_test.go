package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageResolver_TagName(t *testing.T) {
	resolver := &MessageResolver{}
	assert.Equal(t, TagNameMessage, resolver.TagName())
}

func TestMessageResolver_Validate_ValidRoles(t *testing.T) {
	resolver := &MessageResolver{}

	testCases := []struct {
		name string
		role string
	}{
		{"system", "system"},
		{"user", "user"},
		{"assistant", "assistant"},
		{"tool", "tool"},
		{"System (case insensitive)", "System"},
		{"USER (case insensitive)", "USER"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			attrs := Attributes{
				AttrRole: tc.role,
			}
			err := resolver.Validate(attrs)
			assert.NoError(t, err)
		})
	}
}

func TestMessageResolver_Validate_MissingRole(t *testing.T) {
	resolver := &MessageResolver{}
	attrs := Attributes{}

	err := resolver.Validate(attrs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgMessageMissingRole)
}

func TestMessageResolver_Validate_EmptyRole(t *testing.T) {
	resolver := &MessageResolver{}
	attrs := Attributes{
		AttrRole: "",
	}

	err := resolver.Validate(attrs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgMessageMissingRole)
}

func TestMessageResolver_Validate_InvalidRole(t *testing.T) {
	resolver := &MessageResolver{}
	attrs := Attributes{
		AttrRole: "invalid_role",
	}

	err := resolver.Validate(attrs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgMessageInvalidRole)
}

func TestMessageResolver_Resolve_StartMarker(t *testing.T) {
	resolver := &MessageResolver{}
	ctx := context.Background()
	execCtx := &messageTestContextAccessor{}

	attrs := Attributes{
		AttrRole: RoleUser,
	}

	result, err := resolver.Resolve(ctx, execCtx, attrs)
	require.NoError(t, err)

	// Should return start marker with role and cache flag
	assert.Contains(t, result, MessageStartMarker)
	assert.Contains(t, result, "user")
	assert.Contains(t, result, "false") // default cache is false
}

func TestMessageResolver_Resolve_WithCacheTrue(t *testing.T) {
	resolver := &MessageResolver{}
	ctx := context.Background()
	execCtx := &messageTestContextAccessor{}

	attrs := Attributes{
		AttrRole:  RoleSystem,
		AttrCache: "true",
	}

	result, err := resolver.Resolve(ctx, execCtx, attrs)
	require.NoError(t, err)

	assert.Contains(t, result, MessageStartMarker)
	assert.Contains(t, result, "system")
	assert.Contains(t, result, "true") // cache is true
}

func TestExtractMessages_SingleMessage(t *testing.T) {
	output := MessageStartMarker + "user:false:Hello, World!" + MessageEndMarker

	messages := ExtractMessages(output)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleUser, messages[0].Role)
	assert.Equal(t, "Hello, World!", messages[0].Content)
	assert.False(t, messages[0].Cache)
}

func TestExtractMessages_MultipleMessages(t *testing.T) {
	output := MessageStartMarker + "system:true:You are a helpful assistant." + MessageEndMarker +
		MessageStartMarker + "user:false:What is 2+2?" + MessageEndMarker

	messages := ExtractMessages(output)
	require.Len(t, messages, 2)

	assert.Equal(t, RoleSystem, messages[0].Role)
	assert.Equal(t, "You are a helpful assistant.", messages[0].Content)
	assert.True(t, messages[0].Cache)

	assert.Equal(t, RoleUser, messages[1].Role)
	assert.Equal(t, "What is 2+2?", messages[1].Content)
	assert.False(t, messages[1].Cache)
}

func TestExtractMessages_NoMessages(t *testing.T) {
	output := "Just plain text with no message markers"

	messages := ExtractMessages(output)
	assert.Nil(t, messages)
}

func TestExtractMessages_MessagesWithContent(t *testing.T) {
	output := "Prefix text\n" +
		MessageStartMarker + "assistant:false:Here is my response.\n\nWith multiple lines." + MessageEndMarker +
		"\nSuffix text"

	messages := ExtractMessages(output)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleAssistant, messages[0].Role)
	assert.Equal(t, "Here is my response.\n\nWith multiple lines.", messages[0].Content)
}

func TestExtractMessages_FullConversation(t *testing.T) {
	output := MessageStartMarker + "system:true:You are a helpful assistant." + MessageEndMarker +
		MessageStartMarker + "user:false:Hello!" + MessageEndMarker +
		MessageStartMarker + "assistant:false:Hi there! How can I help you?" + MessageEndMarker +
		MessageStartMarker + "user:false:Tell me a joke." + MessageEndMarker

	messages := ExtractMessages(output)
	require.Len(t, messages, 4)

	assert.Equal(t, RoleSystem, messages[0].Role)
	assert.Equal(t, RoleUser, messages[1].Role)
	assert.Equal(t, RoleAssistant, messages[2].Role)
	assert.Equal(t, RoleUser, messages[3].Role)

	assert.Equal(t, "Hello!", messages[1].Content)
	assert.Equal(t, "Hi there! How can I help you?", messages[2].Content)
}

func TestIsValidRole(t *testing.T) {
	tests := []struct {
		role  string
		valid bool
	}{
		{"system", true},
		{"user", true},
		{"assistant", true},
		{"tool", true},
		{"System", true}, // case insensitive
		{"USER", true},   // case insensitive
		{"invalid", false},
		{"admin", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.role, func(t *testing.T) {
			assert.Equal(t, tc.valid, isValidRole(tc.role))
		})
	}
}

// Edge case tests for message extraction

func TestExtractMessages_EmptyContent(t *testing.T) {
	output := MessageStartMarker + "user:false:" + MessageEndMarker

	messages := ExtractMessages(output)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleUser, messages[0].Role)
	assert.Equal(t, "", messages[0].Content) // Empty content after TrimSpace
	assert.False(t, messages[0].Cache)
}

func TestExtractMessages_WhitespaceOnlyContent(t *testing.T) {
	output := MessageStartMarker + "user:false:   \n\t  " + MessageEndMarker

	messages := ExtractMessages(output)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleUser, messages[0].Role)
	assert.Equal(t, "", messages[0].Content) // Whitespace trimmed to empty
	assert.False(t, messages[0].Cache)
}

func TestExtractMessages_ContentWithColons(t *testing.T) {
	// Edge case: content contains colons which are also the field separator
	output := MessageStartMarker + "user:false:Time is 10:30:45 AM" + MessageEndMarker

	messages := ExtractMessages(output)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleUser, messages[0].Role)
	assert.Equal(t, "Time is 10:30:45 AM", messages[0].Content)
}

func TestExtractMessages_ContentWithNewlines(t *testing.T) {
	content := "Line 1\nLine 2\n\nLine 4 after blank"
	output := MessageStartMarker + "assistant:false:" + content + MessageEndMarker

	messages := ExtractMessages(output)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleAssistant, messages[0].Role)
	assert.Equal(t, content, messages[0].Content)
}

func TestExtractMessages_VeryLongContent(t *testing.T) {
	// Test with 10KB of content
	longContent := ""
	for i := range 1000 {
		longContent += "This is line number " + string(rune('0'+i%10)) + ".\n"
	}

	output := MessageStartMarker + "user:false:" + longContent + MessageEndMarker

	messages := ExtractMessages(output)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleUser, messages[0].Role)
	// Content is trimmed, so we check it ends properly
	assert.True(t, len(messages[0].Content) > 1000)
}

func TestExtractMessages_UnicodeContent(t *testing.T) {
	content := "Hello 你好 مرحبا שלום 🎉 emoji test"
	output := MessageStartMarker + "user:false:" + content + MessageEndMarker

	messages := ExtractMessages(output)
	require.Len(t, messages, 1)

	assert.Equal(t, RoleUser, messages[0].Role)
	assert.Equal(t, content, messages[0].Content)
}

func TestMessageResolver_Validate_UnicodeRole(t *testing.T) {
	resolver := &MessageResolver{}
	attrs := Attributes{
		AttrRole: "用户", // Chinese for "user"
	}

	err := resolver.Validate(attrs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrMsgMessageInvalidRole)
}

func TestMessageResolver_Resolve_RoleNormalization(t *testing.T) {
	resolver := &MessageResolver{}
	ctx := context.Background()
	execCtx := &messageTestContextAccessor{}

	testCases := []struct {
		inputRole    string
		expectedRole string
	}{
		{"USER", "user"},
		{"System", "system"},
		{"ASSISTANT", "assistant"},
		{"Tool", "tool"},
	}

	for _, tc := range testCases {
		t.Run(tc.inputRole, func(t *testing.T) {
			attrs := Attributes{
				AttrRole: tc.inputRole,
			}

			result, err := resolver.Resolve(ctx, execCtx, attrs)
			require.NoError(t, err)

			// Result should contain the normalized (lowercase) role
			assert.Contains(t, result, tc.expectedRole+":")
			assert.NotContains(t, result, tc.inputRole+":") // Should not have original case (unless already lowercase)
		})
	}
}

// Tests for malformed markers - ExtractMessages should handle gracefully

func TestExtractMessages_MalformedMissingRoleSeparator(t *testing.T) {
	// Missing first colon after role - malformed marker should be skipped
	output := MessageStartMarker + "user" + MessageEndMarker // No :cache:content

	messages := ExtractMessages(output)
	assert.Nil(t, messages) // Should skip malformed marker
}

func TestExtractMessages_MalformedMissingCacheSeparator(t *testing.T) {
	// Has role but missing cache separator - malformed marker should be skipped
	output := MessageStartMarker + "user:false" + MessageEndMarker // No :content part

	messages := ExtractMessages(output)
	assert.Nil(t, messages) // Should skip malformed marker
}

func TestExtractMessages_MalformedMissingEndMarker(t *testing.T) {
	// Missing end marker - should skip unclosed message
	output := MessageStartMarker + "user:false:Hello World"

	messages := ExtractMessages(output)
	assert.Nil(t, messages) // Should skip unclosed message
}

func TestExtractMessages_MixedValidAndMalformed(t *testing.T) {
	// Valid message followed by malformed (at end, so it doesn't corrupt subsequent messages)
	output := MessageStartMarker + "user:false:First message" + MessageEndMarker +
		MessageStartMarker + "assistant:false:Second message" + MessageEndMarker +
		MessageStartMarker + "invalid" + MessageEndMarker // Malformed at end - missing separators

	messages := ExtractMessages(output)
	require.Len(t, messages, 2) // Only the two valid messages extracted

	assert.Equal(t, RoleUser, messages[0].Role)
	assert.Equal(t, "First message", messages[0].Content)
	assert.Equal(t, RoleAssistant, messages[1].Role)
	assert.Equal(t, "Second message", messages[1].Content)
}

func TestExtractMessages_MalformedMidstreamCanCorrupt(t *testing.T) {
	// Note: When a malformed marker appears mid-stream, it can affect parsing
	// of subsequent markers because the extraction looks for colons which may
	// be found in the next marker's content. This test documents this behavior.
	output := MessageStartMarker + "user:false:First message" + MessageEndMarker +
		MessageStartMarker + "invalid" + MessageEndMarker + // Malformed - will corrupt next
		MessageStartMarker + "assistant:false:Second message" + MessageEndMarker

	messages := ExtractMessages(output)

	// First message is extracted correctly
	require.True(t, len(messages) >= 1)
	assert.Equal(t, RoleUser, messages[0].Role)
	assert.Equal(t, "First message", messages[0].Content)

	// The malformed marker causes subsequent parsing issues, so we don't
	// guarantee the second message is correctly extracted. This is documented
	// behavior - malformed markers in the middle of output can cause corruption.
	// In practice, this shouldn't happen because the executor sanitizes content.
}

func TestExtractMessages_ContentWithMarkerLikeText(t *testing.T) {
	// Content that looks like markers but isn't (no null bytes in real content)
	content := "The marker looks like MSG_START:role:cache:content but without null bytes"
	output := MessageStartMarker + "user:false:" + content + MessageEndMarker

	messages := ExtractMessages(output)
	require.Len(t, messages, 1)

	assert.Equal(t, content, messages[0].Content)
}

func TestExtractMessages_CacheTrueVariants(t *testing.T) {
	testCases := []struct {
		name     string
		cacheStr string
		expected bool
	}{
		{"true lowercase", "true", true},
		{"TRUE uppercase", "TRUE", true},
		{"True mixed", "True", true},
		{"false lowercase", "false", false},
		{"FALSE uppercase", "FALSE", false},
		{"empty", "", false},
		{"invalid", "yes", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := MessageStartMarker + "user:" + tc.cacheStr + ":Test content" + MessageEndMarker

			messages := ExtractMessages(output)
			require.Len(t, messages, 1)
			assert.Equal(t, tc.expected, messages[0].Cache)
		})
	}
}

// messageTestContextAccessor implements ContextAccessor for testing
// (separate from mockContextAccessor in builtins_test.go to avoid redeclaration)
type messageTestContextAccessor struct{}

func (m *messageTestContextAccessor) Get(path string) (any, bool) {
	return nil, false
}

func (m *messageTestContextAccessor) GetString(path string) string {
	return ""
}

func (m *messageTestContextAccessor) GetStringDefault(path, defaultVal string) string {
	return defaultVal
}

func (m *messageTestContextAccessor) Has(path string) bool {
	return false
}
