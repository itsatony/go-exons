package internal

import (
	"strings"
	"unicode/utf8"
)

// Legacy config block tag constants (kept for migration error messages)
const (
	// ConfigBlockOpen is the opening tag for legacy JSON config blocks
	ConfigBlockOpen = "{~exons.config~}"
	// ConfigBlockClose is the closing tag for legacy JSON config blocks
	ConfigBlockClose = "{~/exons.config~}"
)

// FrontmatterResult holds the result of extracting YAML frontmatter from source.
type FrontmatterResult struct {
	// FrontmatterYAML contains the YAML string from inside the frontmatter delimiters.
	// Empty if no frontmatter was found.
	FrontmatterYAML string
	// TemplateBody contains the template source after the frontmatter.
	// If no frontmatter, this is the original source.
	TemplateBody string
	// FrontmatterPosition is the position where the frontmatter started.
	// Zero position if no frontmatter was found.
	FrontmatterPosition Position
	// HasFrontmatter indicates whether YAML frontmatter was found.
	HasFrontmatter bool
}

// HasConfig returns whether frontmatter was found (backward compatibility alias).
// Deprecated: Use HasFrontmatter instead.
func (r *FrontmatterResult) HasConfig() bool {
	return r.HasFrontmatter
}

// ConfigJSON returns the frontmatter YAML content (backward compatibility alias).
// Deprecated: Use FrontmatterYAML instead.
func (r *FrontmatterResult) ConfigJSON() string {
	return r.FrontmatterYAML
}

// ConfigPosition returns the frontmatter position (backward compatibility alias).
// Deprecated: Use FrontmatterPosition instead.
func (r *FrontmatterResult) ConfigPosition() Position {
	return r.FrontmatterPosition
}

// ConfigBlockResult is an alias for backward compatibility during migration.
// Deprecated: Use FrontmatterResult instead.
type ConfigBlockResult = FrontmatterResult

// ExtractYAMLFrontmatter finds and extracts YAML frontmatter from source.
// The frontmatter must appear at the very start of the source (after optional BOM/whitespace).
//
// Format:
//
//	---
//	name: my-template
//	model:
//	  name: gpt-4
//	---
//	Template body here...
//
// Returns:
//   - FrontmatterResult containing the extracted YAML and remaining template body
//   - error if the frontmatter is malformed (e.g., unclosed)
func ExtractYAMLFrontmatter(source string) (*FrontmatterResult, error) {
	result := &FrontmatterResult{
		TemplateBody: source,
	}

	// Skip optional BOM (byte order mark) and leading whitespace
	trimmedStart := skipBOMAndWhitespace(source)

	// Check for legacy JSON config block and provide migration hint
	remaining := source[trimmedStart:]
	if strings.HasPrefix(remaining, ConfigBlockOpen) {
		pos := calculatePosition(source[:trimmedStart])
		return nil, &ConfigError{
			Message:  ErrMsgLegacyJSONConfigDetected,
			Position: pos,
		}
	}

	// Check for YAML frontmatter delimiter (must be exactly "---" followed by newline)
	if !hasYAMLFrontmatterStart(remaining) {
		// No frontmatter - return source as template body
		return result, nil
	}

	// Calculate position of frontmatter
	result.FrontmatterPosition = calculatePosition(source[:trimmedStart])
	result.HasFrontmatter = true

	// Find the content start (after the opening delimiter and newline)
	contentStart := trimmedStart + findNewlineAfterDelimiter(remaining)

	// Find the closing delimiter "---" on its own line
	closeIdx := findClosingFrontmatterDelimiter(source[contentStart:])
	if closeIdx == -1 {
		// Frontmatter not closed
		pos := calculatePosition(source[:contentStart])
		return nil, &ConfigError{
			Message:  ErrMsgFrontmatterUnclosed,
			Position: pos,
		}
	}

	// Extract YAML content (between delimiters)
	result.FrontmatterYAML = strings.TrimSpace(source[contentStart : contentStart+closeIdx])

	// Template body is everything after the closing delimiter line
	bodyStart := contentStart + closeIdx + findNewlineAfterDelimiter(source[contentStart+closeIdx:])
	result.TemplateBody = source[bodyStart:]

	return result, nil
}

// ExtractConfigBlock is kept for backward compatibility.
// It now extracts YAML frontmatter instead of JSON config blocks.
// Deprecated: Use ExtractYAMLFrontmatter instead.
func ExtractConfigBlock(source string, config LexerConfig) (*FrontmatterResult, error) {
	return ExtractYAMLFrontmatter(source)
}

// skipBOMAndWhitespace skips any UTF-8 BOM and leading whitespace,
// returning the byte offset where content starts.
func skipBOMAndWhitespace(source string) int {
	offset := 0

	// Skip UTF-8 BOM if present (EF BB BF)
	if strings.HasPrefix(source, "\xef\xbb\xbf") {
		offset = 3
	}

	// Skip leading whitespace (but NOT newlines - they matter for frontmatter detection)
	for offset < len(source) {
		r, size := utf8.DecodeRuneInString(source[offset:])
		if r == ' ' || r == '\t' {
			offset += size
		} else {
			break
		}
	}

	return offset
}

// hasYAMLFrontmatterStart checks if the string starts with "---" followed by newline.
func hasYAMLFrontmatterStart(s string) bool {
	if !strings.HasPrefix(s, YAMLFrontmatterDelimiter) {
		return false
	}
	// Must be followed by newline (LF or CRLF) or be just "---\n"
	rest := s[len(YAMLFrontmatterDelimiter):]
	return len(rest) > 0 && (rest[0] == '\n' || (len(rest) > 1 && rest[0] == '\r' && rest[1] == '\n'))
}

// findNewlineAfterDelimiter returns the offset to the start of content after "---\n" or "---\r\n".
func findNewlineAfterDelimiter(s string) int {
	if !strings.HasPrefix(s, YAMLFrontmatterDelimiter) {
		return 0
	}
	offset := len(YAMLFrontmatterDelimiter)
	if len(s) > offset && s[offset] == '\r' {
		offset++
	}
	if len(s) > offset && s[offset] == '\n' {
		offset++
	}
	return offset
}

// findClosingFrontmatterDelimiter finds the closing "---" that appears at the start of a line.
// Returns the byte offset of the "---" from the start of the search string, or -1 if not found.
func findClosingFrontmatterDelimiter(s string) int {
	// Search for "---" at the beginning of a line
	pos := 0
	for pos < len(s) {
		// Check if we're at the start of a line with "---"
		if strings.HasPrefix(s[pos:], YAMLFrontmatterDelimiter) {
			// Verify it's followed by newline or end of string
			afterDelim := pos + len(YAMLFrontmatterDelimiter)
			if afterDelim >= len(s) || s[afterDelim] == '\n' || s[afterDelim] == '\r' {
				return pos
			}
		}
		// Move to next line
		nextNewline := strings.Index(s[pos:], "\n")
		if nextNewline == -1 {
			break
		}
		pos += nextNewline + 1
	}
	return -1
}

// calculatePosition calculates the Position (line, column, offset) for a given prefix string.
func calculatePosition(prefix string) Position {
	pos := Position{
		Offset: len(prefix),
		Line:   1,
		Column: 1,
	}

	for i := 0; i < len(prefix); i++ {
		if prefix[i] == '\n' {
			pos.Line++
			pos.Column = 1
		} else {
			pos.Column++
		}
	}

	return pos
}

// ConfigError represents an error during config/frontmatter extraction.
type ConfigError struct {
	Message  string
	Position Position
	Cause    error
}

// Error implements the error interface.
func (e *ConfigError) Error() string {
	if e.Position.Line > 0 {
		return e.Message + " at " + e.Position.String()
	}
	return e.Message
}

// Unwrap returns the underlying cause.
func (e *ConfigError) Unwrap() error {
	return e.Cause
}

// NewConfigError creates a new config error with position.
func NewConfigError(message string, pos Position, cause error) *ConfigError {
	return &ConfigError{
		Message:  message,
		Position: pos,
		Cause:    cause,
	}
}
