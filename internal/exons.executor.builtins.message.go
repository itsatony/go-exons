package internal

import (
	"context"
	"strings"
)

// MessageResolver handles {~exons.message role="..."~}...{~/exons.message~} tags.
// It outputs content wrapped with special markers that can be extracted after execution.
// This resolver is thread-safe as it maintains no mutable state.
type MessageResolver struct{}

// Message marker format for extraction after template execution.
//
// Format: \x00MSG_START:<role>:<cache>:<content>\x00MSG_END\x00
//
// Components:
//   - \x00 (null byte): Non-printable delimiter to prevent collision with prompt content
//   - MSG_START: Literal marker identifier
//   - <role>: Message role (system|user|assistant|tool), always lowercase
//   - <cache>: Cache hint (true|false)
//   - <content>: Executed template content (may contain newlines, sanitized of null bytes)
//   - MSG_END: Literal end marker
//
// Example output: "\x00MSG_START:user:false:Hello world\x00MSG_END\x00"
//
// Security: Content is sanitized by the executor to remove null bytes (\x00) before
// being wrapped with markers. This prevents marker injection attacks where malicious
// content could attempt to inject fake messages.
//
// Extraction: Use ExtractMessages() to parse the executed output and recover
// structured messages for LLM API calls.
const (
	// MessageStartMarker marks the beginning of a message in executed output.
	// Format after this marker: <role>:<cache>:<content>
	MessageStartMarker = "\x00MSG_START:"

	// MessageEndMarker marks the end of a message in executed output.
	MessageEndMarker = "\x00MSG_END\x00"

	// MessageFieldSep separates fields (role, cache, content) within the message.
	MessageFieldSep = ":"
)

// TagName returns the tag name this resolver handles.
func (r *MessageResolver) TagName() string {
	return TagNameMessage
}

// Validate checks that the tag has valid attributes.
func (r *MessageResolver) Validate(attrs Attributes) error {
	// role is required
	role, hasRole := attrs.Get(AttrRole)
	if !hasRole || role == "" {
		return &BuiltinError{
			Message: ErrMsgMessageMissingRole,
			TagName: TagNameMessage,
		}
	}

	// Validate role value
	if !isValidRole(role) {
		return &BuiltinError{
			Message:  ErrMsgMessageInvalidRole,
			TagName:  TagNameMessage,
			Metadata: map[string]string{AttrRole: role},
		}
	}

	return nil
}

// Resolve wraps the content with message markers for later extraction.
// The content is already executed by the time this resolver is called.
// This resolver is special - it's called for block tags, and the content
// between the tags has already been processed by the executor.
func (r *MessageResolver) Resolve(ctx context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	role, _ := attrs.Get(AttrRole)
	cache, hasCache := attrs.Get(AttrCache)

	// Normalize role to lowercase for consistent storage
	// (validation already accepts case-insensitive input)
	role = strings.ToLower(role)

	// Build cache flag string
	cacheFlag := "false"
	if hasCache && strings.EqualFold(cache, AttrValueTrue) {
		cacheFlag = "true"
	}

	// For block tags, the content is passed via a special attribute or we return just markers
	// The executor will handle inserting the content between markers

	// Return just the start marker - the executor needs to wrap the content
	return MessageStartMarker + role + MessageFieldSep + cacheFlag + MessageFieldSep, nil
}

// isValidRole checks if the role is one of the allowed values.
// Roles are case-insensitive during validation.
func isValidRole(role string) bool {
	switch strings.ToLower(role) {
	case RoleSystem, RoleUser, RoleAssistant, RoleTool:
		return true
	default:
		return false
	}
}

// MessageInfo represents extracted message information.
type MessageInfo struct {
	Role    string // Message role: system, user, assistant, or tool
	Content string // Message content with leading/trailing whitespace trimmed
	Cache   bool   // Cache hint for this message
}

// ExtractMessages parses the executed template output and extracts structured messages.
// Returns nil if no messages are found.
//
// This function gracefully handles malformed markers by skipping them:
//   - Missing role separator: Message skipped (marker may be corrupted)
//   - Missing cache separator: Message skipped (marker may be corrupted)
//   - Missing end marker: Message skipped (unclosed message block)
//
// For debugging extraction issues, check that:
//  1. Template uses {~exons.message role="..."~}...{~/exons.message~} syntax
//  2. Role is one of: system, user, assistant, tool
//  3. No null bytes (\x00) in template content (they are sanitized)
//
// Example usage:
//
//	output, _ := tmpl.Execute(ctx, data)
//	messages := ExtractMessages(output)
//	for _, msg := range messages {
//	    fmt.Printf("[%s]: %s\n", msg.Role, msg.Content)
//	}
func ExtractMessages(output string) []MessageInfo {
	var messages []MessageInfo

	remaining := output
	for {
		// Find start marker
		startIdx := strings.Index(remaining, MessageStartMarker)
		if startIdx == -1 {
			break // No more messages - normal termination
		}

		// Find end of start marker (the second MessageFieldSep after role)
		markerStart := startIdx + len(MessageStartMarker)

		// Parse role:cache: from marker
		// Expected format: role:cache:content
		firstSep := strings.Index(remaining[markerStart:], MessageFieldSep)
		if firstSep == -1 {
			// Malformed marker: missing role separator
			// This indicates a corrupted or incomplete marker - skip to next potential marker
			remaining = remaining[markerStart:]
			continue
		}
		role := remaining[markerStart : markerStart+firstSep]

		cacheStart := markerStart + firstSep + 1
		secondSep := strings.Index(remaining[cacheStart:], MessageFieldSep)
		if secondSep == -1 {
			// Malformed marker: missing cache separator
			// This indicates a corrupted marker - skip to next potential marker
			remaining = remaining[cacheStart:]
			continue
		}
		cacheStr := remaining[cacheStart : cacheStart+secondSep]
		cache := strings.EqualFold(cacheStr, "true")

		contentStart := cacheStart + secondSep + 1

		// Find end marker
		endIdx := strings.Index(remaining[contentStart:], MessageEndMarker)
		if endIdx == -1 {
			// Malformed marker: missing end marker
			// This indicates an unclosed message block - skip rest of output
			// This shouldn't happen in well-formed output from the executor
			break
		}

		content := remaining[contentStart : contentStart+endIdx]

		messages = append(messages, MessageInfo{
			Role:    role,
			Content: strings.TrimSpace(content),
			Cache:   cache,
		})

		// Move past this message
		remaining = remaining[contentStart+endIdx+len(MessageEndMarker):]
	}

	return messages
}
