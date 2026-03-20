package exons

import (
	"strings"
)

// ImportFromSkillMD parses a SKILL.md-formatted string into a Spec.
// The content must contain YAML frontmatter delimited by --- markers.
// Any content after the closing --- delimiter is set as the body.
func ImportFromSkillMD(content string) (*Spec, error) {
	if content == "" {
		return nil, NewImportError(ErrMsgSkillMDInvalidFormat, nil)
	}

	// Trim BOM and leading whitespace
	trimmed := strings.TrimLeft(content, "\xef\xbb\xbf \t")

	// Check for frontmatter delimiter
	if !strings.HasPrefix(trimmed, YAMLFrontmatterDelimiter) {
		return nil, NewImportError(ErrMsgSkillMDMissingFM, nil)
	}

	// Skip opening delimiter and newline
	afterOpening := trimmed[len(YAMLFrontmatterDelimiter):]
	if len(afterOpening) > 0 && afterOpening[0] == '\n' {
		afterOpening = afterOpening[1:]
	} else if len(afterOpening) > 1 && afterOpening[0] == '\r' && afterOpening[1] == '\n' {
		afterOpening = afterOpening[2:]
	}

	// Find closing delimiter
	closeIdx := strings.Index(afterOpening, "\n"+YAMLFrontmatterDelimiter)
	if closeIdx == -1 {
		return nil, NewImportError(ErrMsgSkillMDInvalidFormat, nil)
	}

	// Extract frontmatter YAML
	fmYAML := afterOpening[:closeIdx]

	// Extract body (after closing delimiter and newline)
	bodyStart := closeIdx + len("\n"+YAMLFrontmatterDelimiter)
	body := ""
	if bodyStart < len(afterOpening) {
		body = afterOpening[bodyStart:]
		if len(body) > 0 && body[0] == '\n' {
			body = body[1:]
		} else if len(body) > 1 && body[0] == '\r' && body[1] == '\n' {
			body = body[2:]
		}
	}

	// Parse YAML frontmatter
	spec, err := ParseYAMLSpec(fmYAML)
	if err != nil {
		return nil, NewImportError(ErrMsgSkillMDParseFailed, err)
	}
	if spec == nil {
		return nil, NewImportError(ErrMsgSkillMDParseFailed, nil)
	}

	spec.Body = body
	return spec, nil
}

// ExportToSkillMD serializes the Spec in SKILL.md format using
// Agent Skills export options (strips execution, extensions, and agent fields).
// Returns an error if the receiver is nil.
func (s *Spec) ExportToSkillMD() ([]byte, error) {
	if s == nil {
		return nil, NewExportError(ErrMsgExportFailed, nil)
	}
	return s.Serialize(AgentSkillsExportOptions())
}
