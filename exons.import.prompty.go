package exons

import (
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"
)

// File extension constants for prompty/genspec files.
const (
	FileExtensionPrompty = ".prompty"
	FileExtensionGenSpec = ".genspec"
)

// Tag namespace constants for conversion.
const (
	PromptyTagOpenPrefix  = "{~prompty."
	PromptyTagClosePrefix = "{~/prompty."
	ExonsTagOpenPrefix    = "{~exons."
	ExonsTagClosePrefix   = "{~/exons."
)

// Import error messages for prompty conversion.
const (
	ErrMsgImportPromptyFailed      = "prompty import failed"
	ErrMsgImportPromptyParseFailed = "prompty YAML parsing failed"
)

// Prompty field names that need remapping.
const (
	promptyFieldDelegation  = "delegation"
	promptyFieldTests       = "tests"
	promptyFieldPlugin      = "plugin"
	promptyFieldTrustLevel  = "trust_level"
	promptyFieldUserMessage = "user_message"
	promptyFieldGenSpec     = "genspec"
	promptyFieldVersion     = "version"
)

// Exons target field names for remapping (YAML map keys).
const (
	remapTargetPrompt = "prompt"
	remapTargetOrigin = "origin"
)

// BOM and whitespace constants for frontmatter parsing.
const (
	bomAndWhitespace = "\xef\xbb\xbf \t\r\n"
)

// ImportPrompty converts raw .prompty or .genspec file content into a Spec.
// It performs:
//  1. Tag namespace conversion: {~prompty.X~} → {~exons.X~}
//  2. YAML field remapping: delegation→dispatch, tests→verifications, plugin→registry
//  3. genspec: wrapper flattening (promotes children to top level)
//
// Extra prompty-only fields (license, compatibility, etc.) are preserved as
// top-level YAML keys and captured by Spec.Extensions during parsing.
//
// Returns the parsed Spec or an error if conversion or parsing fails.
func ImportPrompty(data []byte) (*Spec, error) {
	if len(data) == 0 {
		return nil, NewImportError(ErrMsgImportPromptyFailed, nil)
	}

	// Split into frontmatter and body
	content := string(data)
	fmYAML, body, hasFrontmatter := splitPromptyFrontmatter(content)

	// Convert tags in body
	body = convertPromptyTags(body)

	if !hasFrontmatter || fmYAML == "" {
		// No frontmatter — pass body-only content through to Parse()
		spec, err := Parse([]byte(body))
		if err != nil {
			return nil, NewImportError(ErrMsgImportPromptyFailed, err)
		}
		return spec, nil
	}

	// Parse YAML into raw map for remapping
	var rawMap map[string]any
	if err := yaml.Unmarshal([]byte(fmYAML), &rawMap); err != nil {
		return nil, NewImportError(ErrMsgImportPromptyParseFailed, err)
	}
	if rawMap == nil {
		rawMap = make(map[string]any)
	}

	// Remap fields
	rawMap = remapPromptyFields(rawMap)

	// Flatten genspec: wrapper
	rawMap = flattenGenSpecWrapper(rawMap)

	// Re-serialize the remapped YAML
	yamlBytes, err := yaml.Marshal(rawMap)
	if err != nil {
		return nil, NewImportError(ErrMsgImportPromptyParseFailed, err)
	}

	// Reconstruct as frontmatter + body document
	var sb strings.Builder
	sb.WriteString(YAMLFrontmatterDelimiter)
	sb.WriteString("\n")
	sb.Write(yamlBytes)
	sb.WriteString(YAMLFrontmatterDelimiter)
	sb.WriteString("\n")
	if body != "" {
		sb.WriteString(body)
	}

	// Parse through standard Parse() function
	spec, err := Parse([]byte(sb.String()))
	if err != nil {
		return nil, NewImportError(ErrMsgImportPromptyFailed, err)
	}

	return spec, nil
}

// isPromptyContent returns true if the content contains {~prompty. tags,
// indicating it is a prompty-format document.
func isPromptyContent(data []byte) bool {
	return bytes.Contains(data, []byte(PromptyTagOpenPrefix))
}

// convertPromptyTags replaces all {~prompty. and {~/prompty. tag prefixes
// with {~exons. and {~/exons. respectively.
func convertPromptyTags(content string) string {
	result := strings.ReplaceAll(content, PromptyTagOpenPrefix, ExonsTagOpenPrefix)
	result = strings.ReplaceAll(result, PromptyTagClosePrefix, ExonsTagClosePrefix)
	return result
}

// splitPromptyFrontmatter splits content into YAML frontmatter and body.
// Returns (yaml, body, hasFrontmatter).
func splitPromptyFrontmatter(content string) (string, string, bool) {
	trimmed := strings.TrimLeft(content, bomAndWhitespace)

	if !strings.HasPrefix(trimmed, YAMLFrontmatterDelimiter) {
		// No frontmatter — entire content is body
		return "", content, false
	}

	// Skip opening delimiter
	afterOpen := trimmed[len(YAMLFrontmatterDelimiter):]
	if len(afterOpen) > 0 && afterOpen[0] == '\n' {
		afterOpen = afterOpen[1:]
	} else if len(afterOpen) > 1 && afterOpen[0] == '\r' && afterOpen[1] == '\n' {
		afterOpen = afterOpen[2:]
	}

	// Find closing delimiter
	closeIdx := strings.Index(afterOpen, "\n"+YAMLFrontmatterDelimiter)
	if closeIdx == -1 {
		// No closing delimiter — treat as no frontmatter
		return "", content, false
	}

	fmYAML := afterOpen[:closeIdx]

	// Body starts after closing delimiter
	bodyStart := closeIdx + len("\n"+YAMLFrontmatterDelimiter)
	body := ""
	if bodyStart < len(afterOpen) {
		body = afterOpen[bodyStart:]
		if len(body) > 0 && body[0] == '\n' {
			body = body[1:]
		} else if len(body) > 1 && body[0] == '\r' && body[1] == '\n' {
			body = body[2:]
		}
	}

	return fmYAML, body, true
}

// remapPromptyFields converts prompty-specific field names to exons equivalents.
//   - delegation → dispatch
//   - tests → verifications
//   - plugin → registry (with trust_level → origin)
//   - Extra prompty-only fields → extensions
func remapPromptyFields(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))

	for k, v := range m {
		switch k {
		case promptyFieldDelegation:
			result[SpecFieldDispatch] = v

		case promptyFieldTests:
			result[SpecFieldVerifications] = remapTestsToVerifications(v)

		case promptyFieldPlugin:
			result[SpecFieldRegistry] = remapPluginToRegistry(v)

		default:
			// Extra prompty-only fields remain as top-level keys.
			// They will be captured by Spec.Extensions (yaml:",inline") during parsing.
			result[k] = v
		}
	}

	return result
}

// remapTestsToVerifications converts prompty test cases to verification format.
// The main difference is field naming: user_message → prompt.
func remapTestsToVerifications(v any) any {
	tests, ok := v.([]any)
	if !ok {
		return v
	}

	verifications := make([]any, 0, len(tests))
	for _, test := range tests {
		testMap, ok := test.(map[string]any)
		if !ok {
			verifications = append(verifications, test)
			continue
		}

		vCase := make(map[string]any, len(testMap))
		for tk, tv := range testMap {
			if tk == promptyFieldUserMessage {
				vCase[remapTargetPrompt] = tv
			} else {
				vCase[tk] = tv
			}
		}
		verifications = append(verifications, vCase)
	}

	return verifications
}

// remapPluginToRegistry converts plugin metadata to registry format.
// Maps trust_level → origin.
func remapPluginToRegistry(v any) any {
	pluginMap, ok := v.(map[string]any)
	if !ok {
		return v
	}

	registry := make(map[string]any, len(pluginMap))
	for pk, pv := range pluginMap {
		if pk == promptyFieldTrustLevel {
			registry[remapTargetOrigin] = pv
		} else {
			registry[pk] = pv
		}
	}
	return registry
}

// flattenGenSpecWrapper promotes children of a "genspec:" key to the top level.
// If a genspec wrapper is present, its children are merged into the parent map.
func flattenGenSpecWrapper(m map[string]any) map[string]any {
	gsRaw, exists := m[promptyFieldGenSpec]
	if !exists {
		return m
	}

	gsMap, ok := gsRaw.(map[string]any)
	if !ok {
		return m
	}

	// Remove the genspec wrapper
	delete(m, promptyFieldGenSpec)

	// Remove version field (genspec-specific, not needed in exons)
	delete(gsMap, promptyFieldVersion)

	// Promote children, applying field remapping
	for k, v := range gsMap {
		switch k {
		case promptyFieldDelegation:
			m[SpecFieldDispatch] = v
		case promptyFieldTests:
			m[SpecFieldVerifications] = remapTestsToVerifications(v)
		case promptyFieldPlugin:
			m[SpecFieldRegistry] = remapPluginToRegistry(v)
		default:
			// Direct promotion (memory, dispatch, verifications, registry, safety)
			m[k] = v
		}
	}

	return m
}
