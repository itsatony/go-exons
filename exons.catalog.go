package exons

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// skillCatalogEntry holds resolved skill information for catalog generation.
type skillCatalogEntry struct {
	slug        string
	description string
	injection   string
}

// generateSkillsCatalog generates a text catalog of skills in the specified format.
// The resolver is used to look up skill descriptions; resolution failures are non-fatal
// (the skill appears with an empty description). If resolver is nil, all descriptions
// will be empty.
//
// Supported formats:
//   - CatalogFormatDefault (""): Markdown list with descriptions
//   - CatalogFormatDetailed: Full description with injection mode
//   - CatalogFormatCompact: Semicolon-separated single-line entries
//   - CatalogFormatFunctionCalling: Not supported for skills (returns error)
func generateSkillsCatalog(ctx context.Context, skills []SkillRef, resolver SpecResolver, format CatalogFormat) (string, error) {
	if len(skills) == 0 {
		return "", nil
	}

	// Build entries by resolving descriptions
	entries := make([]skillCatalogEntry, 0, len(skills))
	for _, ref := range skills {
		entry := skillCatalogEntry{
			slug:      ref.Slug,
			injection: ref.Injection,
		}

		// Resolve description (non-fatal on failure)
		if resolver != nil && ref.Slug != "" {
			spec, _, err := resolver.ResolveSpec(ctx, ref.Slug, RefVersionLatest)
			if err == nil && spec != nil {
				entry.description = spec.Description
			}
		}

		entries = append(entries, entry)
	}

	switch format {
	case CatalogFormatDefault:
		return generateSkillsCatalogDefault(entries), nil
	case CatalogFormatDetailed:
		return generateSkillsCatalogDetailed(entries), nil
	case CatalogFormatCompact:
		return generateSkillsCatalogCompact(entries), nil
	case CatalogFormatFunctionCalling:
		return "", NewCatalogError(ErrMsgCatalogFuncCallingSkills, nil)
	default:
		return "", NewCatalogFormatError(ErrMsgCatalogUnknownFormat, string(format))
	}
}

// generateToolsCatalog generates a text catalog of tools in the specified format.
// Returns an empty string when tools is nil or has no tool definitions.
//
// Supported formats:
//   - CatalogFormatDefault (""): Markdown list with descriptions
//   - CatalogFormatDetailed: Full description with JSON parameters and MCP servers
//   - CatalogFormatCompact: Semicolon-separated single-line entries
//   - CatalogFormatFunctionCalling: JSON array of OpenAI-compatible tool definitions
func generateToolsCatalog(tools *ToolsConfig, format CatalogFormat) (string, error) {
	if !tools.HasTools() {
		return "", nil
	}

	switch format {
	case CatalogFormatDefault:
		return generateToolsCatalogDefault(tools), nil
	case CatalogFormatDetailed:
		return generateToolsCatalogDetailed(tools), nil
	case CatalogFormatCompact:
		return generateToolsCatalogCompact(tools), nil
	case CatalogFormatFunctionCalling:
		return generateToolsCatalogFunctionCalling(tools)
	default:
		return "", NewCatalogFormatError(ErrMsgCatalogUnknownFormat, string(format))
	}
}

// generateSkillsCatalogDefault produces a markdown list of skills.
func generateSkillsCatalogDefault(entries []skillCatalogEntry) string {
	var sb strings.Builder
	sb.WriteString(CatalogHeaderSkills)
	for _, e := range entries {
		sb.WriteString(CatalogMDListItem)
		sb.WriteString(e.slug)
		sb.WriteString(CatalogMDBoldClose)
		if e.description != "" {
			sb.WriteString(CatalogMDColonSep)
			sb.WriteString(e.description)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// generateSkillsCatalogDetailed produces a detailed markdown catalog including
// description and injection mode.
func generateSkillsCatalogDetailed(entries []skillCatalogEntry) string {
	var sb strings.Builder
	sb.WriteString(CatalogHeaderSkills)
	for _, e := range entries {
		sb.WriteString(CatalogMDHeading3)
		sb.WriteString(e.slug)
		sb.WriteString("\n")
		if e.description != "" {
			sb.WriteString(e.description)
			sb.WriteString("\n")
		}
		if e.injection != "" {
			sb.WriteString(CatalogMDInjectionPfx)
			sb.WriteString(e.injection)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// generateSkillsCatalogCompact produces a compact semicolon-separated format for skills.
func generateSkillsCatalogCompact(entries []skillCatalogEntry) string {
	parts := make([]string, 0, len(entries))
	for _, e := range entries {
		part := e.slug
		if e.description != "" {
			part += CatalogMDDashSep + truncateString(e.description, CatalogCompactDescriptionMaxLen)
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, CatalogCompactSep)
}

// generateToolsCatalogDefault produces a markdown list of tool functions and MCP servers.
func generateToolsCatalogDefault(tools *ToolsConfig) string {
	var sb strings.Builder
	sb.WriteString(CatalogHeaderTools)

	for _, fn := range tools.Functions {
		if fn == nil {
			continue
		}
		sb.WriteString(CatalogMDListItem)
		sb.WriteString(fn.Name)
		sb.WriteString(CatalogMDBoldClose)
		if fn.Description != "" {
			sb.WriteString(CatalogMDColonSep)
			sb.WriteString(fn.Description)
		}
		sb.WriteString("\n")
	}

	for _, mcp := range tools.MCPServers {
		if mcp == nil {
			continue
		}
		sb.WriteString(CatalogMDMCPListPfx)
		sb.WriteString(mcp.Name)
		sb.WriteString(CatalogMDBoldClose)
		if mcp.URL != "" {
			sb.WriteString(CatalogMDParenOpen)
			sb.WriteString(mcp.URL)
			sb.WriteString(CatalogMDParenClose)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateToolsCatalogDetailed produces a detailed markdown catalog with JSON parameters
// for functions and URL info for MCP servers.
func generateToolsCatalogDetailed(tools *ToolsConfig) string {
	var sb strings.Builder
	sb.WriteString(CatalogHeaderTools)

	for _, fn := range tools.Functions {
		if fn == nil {
			continue
		}
		sb.WriteString(CatalogMDHeading3)
		sb.WriteString(fn.Name)
		sb.WriteString("\n")
		if fn.Description != "" {
			sb.WriteString(fn.Description)
			sb.WriteString("\n")
		}
		if fn.Parameters != nil {
			paramJSON, err := json.MarshalIndent(fn.Parameters, "", JSONIndentDefault)
			if err == nil {
				sb.WriteString(CatalogMDCodeBlockOpen)
				sb.WriteString(string(paramJSON))
				sb.WriteString(CatalogMDCodeBlockEnd)
			}
		}
		sb.WriteString("\n")
	}

	for _, mcp := range tools.MCPServers {
		if mcp == nil {
			continue
		}
		sb.WriteString(CatalogMDMCPDetailPfx)
		sb.WriteString(mcp.Name)
		sb.WriteString("\n")
		if mcp.URL != "" {
			sb.WriteString(CatalogMDURLPfx)
			sb.WriteString(mcp.URL)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateToolsCatalogCompact produces a compact semicolon-separated format for tools.
func generateToolsCatalogCompact(tools *ToolsConfig) string {
	parts := make([]string, 0, len(tools.Functions)+len(tools.MCPServers))

	for _, fn := range tools.Functions {
		if fn == nil {
			continue
		}
		part := fn.Name
		if fn.Description != "" {
			part += CatalogMDDashSep + truncateString(fn.Description, CatalogCompactDescriptionMaxLen)
		}
		parts = append(parts, part)
	}

	for _, mcp := range tools.MCPServers {
		if mcp == nil {
			continue
		}
		part := CatalogMDMCPPfx + mcp.Name
		parts = append(parts, part)
	}

	return strings.Join(parts, CatalogCompactSep)
}

// generateToolsCatalogFunctionCalling produces a JSON array of OpenAI-compatible
// tool definitions. Only function definitions are included (not MCP servers).
func generateToolsCatalogFunctionCalling(tools *ToolsConfig) (string, error) {
	toolDefs := make([]map[string]any, 0, len(tools.Functions))
	for _, fn := range tools.Functions {
		if fn == nil {
			continue
		}
		toolDefs = append(toolDefs, fn.ToOpenAITool())
	}

	data, err := json.MarshalIndent(toolDefs, "", JSONIndentDefault)
	if err != nil {
		return "", NewCatalogError(ErrMsgCatalogGenerationFailed, err)
	}
	return string(data), nil
}

// truncateString truncates a string to maxLen runes, appending "..." if truncated.
// Operates on Unicode code points (runes) to avoid splitting multi-byte characters.
func truncateString(s string, maxLen int) string {
	runeCount := utf8.RuneCountInString(s)
	if runeCount <= maxLen {
		return s
	}
	if maxLen <= len(TruncationSuffix) {
		return string([]rune(s)[:maxLen])
	}
	return string([]rune(s)[:maxLen-len(TruncationSuffix)]) + TruncationSuffix
}
