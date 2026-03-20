package exons

import (
	"strings"
)

// ToOpenAIMessages returns messages in OpenAI API format.
// Format: []map[string]any where each map has "role" and "content" keys.
// Returns nil if the CompiledSpec is nil or has no messages.
func (cs *CompiledSpec) ToOpenAIMessages() []map[string]any {
	if cs == nil || len(cs.Messages) == 0 {
		return nil
	}

	messages := make([]map[string]any, 0, len(cs.Messages))
	for _, msg := range cs.Messages {
		messages = append(messages, map[string]any{
			AttrRole:              msg.Role,
			ProviderMsgKeyContent: msg.Content,
		})
	}
	return messages
}

// ToAnthropicMessages returns messages in Anthropic API format.
// Format: map[string]any with "system" (string) and "messages" ([]map[string]any) keys.
// System messages are extracted to a top-level "system" key; multiple system messages
// are joined with "\n\n". Non-system messages are placed in the "messages" array.
// Returns nil if the CompiledSpec is nil or has no messages.
func (cs *CompiledSpec) ToAnthropicMessages() map[string]any {
	if cs == nil || len(cs.Messages) == 0 {
		return nil
	}

	var systemParts []string
	nonSystem := make([]map[string]any, 0, len(cs.Messages))

	for _, msg := range cs.Messages {
		if msg.Role == RoleSystem {
			systemParts = append(systemParts, msg.Content)
		} else {
			nonSystem = append(nonSystem, map[string]any{
				AttrRole:              msg.Role,
				ProviderMsgKeyContent: msg.Content,
			})
		}
	}

	result := make(map[string]any)
	if len(systemParts) > 0 {
		result[ProviderMsgKeySystem] = strings.Join(systemParts, SystemMessageSeparator)
	}
	if len(nonSystem) > 0 {
		result[ProviderMsgKeyMessages] = nonSystem
	}
	return result
}

// ToGeminiContents returns messages in Gemini API format.
// Format: map[string]any with optional "system_instruction" and "contents" array.
// System messages go to "system_instruction.parts[0].text".
// The "assistant" role is mapped to "model".
// Non-system messages go to "contents" array with "parts[0].text" format.
// Returns nil if the CompiledSpec is nil or has no messages.
func (cs *CompiledSpec) ToGeminiContents() map[string]any {
	if cs == nil || len(cs.Messages) == 0 {
		return nil
	}

	var systemParts []string
	contents := make([]map[string]any, 0, len(cs.Messages))

	for _, msg := range cs.Messages {
		if msg.Role == RoleSystem {
			systemParts = append(systemParts, msg.Content)
		} else {
			role := msg.Role
			if role == RoleAssistant {
				role = ProviderMsgKeyModelRole
			}
			contents = append(contents, map[string]any{
				AttrRole: role,
				ProviderMsgKeyParts: []map[string]any{
					{ProviderMsgKeyText: msg.Content},
				},
			})
		}
	}

	result := make(map[string]any)
	if len(systemParts) > 0 {
		systemText := strings.Join(systemParts, SystemMessageSeparator)
		result[ProviderMsgKeySystemInstruction] = map[string]any{
			ProviderMsgKeyParts: []map[string]any{
				{ProviderMsgKeyText: systemText},
			},
		}
	}
	if len(contents) > 0 {
		result[ProviderMsgKeyContents] = contents
	}
	return result
}

// ToProviderMessages dispatches to the correct provider format based on the provider name.
// Supported providers:
//   - "openai", "azure" → ToOpenAIMessages()
//   - "anthropic" → ToAnthropicMessages()
//   - "google", "gemini", "vertex" → ToGeminiContents()
//
// Returns an error for unsupported providers.
func (cs *CompiledSpec) ToProviderMessages(provider string) (any, error) {
	switch provider {
	case ProviderOpenAI, ProviderAzure:
		return cs.ToOpenAIMessages(), nil
	case ProviderAnthropic:
		return cs.ToAnthropicMessages(), nil
	case ProviderGoogle, ProviderGemini, ProviderVertex:
		return cs.ToGeminiContents(), nil
	default:
		return nil, NewProviderMessageError(provider)
	}
}
