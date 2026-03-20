package exons

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// Serialization error messages
const (
	ErrMsgSerializeFailed = "serialization failed"
	ErrMsgSerializeYAML   = "YAML marshaling failed"
)

// SerializeOptions configures spec serialization.
type SerializeOptions struct {
	// IncludeExecution includes the execution config in output
	IncludeExecution bool
	// IncludeExtensions includes extension fields (non-standard YAML keys) in output
	IncludeExtensions bool
	// IncludeAgentFields includes agent-specific fields (type, skills, tools, constraints, messages)
	IncludeAgentFields bool
	// IncludeContext includes the context map in output
	IncludeContext bool
	// IncludeCredentials includes credential references in output.
	// Default is false (safe default -- credentials are sensitive metadata).
	IncludeCredentials bool
	// IncludeGenSpec includes the GenSpec metadata in output
	IncludeGenSpec bool
}

// DefaultSerializeOptions returns the default serialization options (all included
// except credentials).
func DefaultSerializeOptions() *SerializeOptions {
	return &SerializeOptions{
		IncludeExecution:   true,
		IncludeExtensions:  true,
		IncludeAgentFields: true,
		IncludeContext:     true,
		IncludeGenSpec:     true,
	}
}

// AgentSkillsExportOptions returns options for Agent Skills compatible export.
// This strips all non-standard fields.
func AgentSkillsExportOptions() *SerializeOptions {
	return &SerializeOptions{
		IncludeExecution:   false,
		IncludeExtensions:  false,
		IncludeAgentFields: false,
		IncludeContext:     false,
		IncludeCredentials: false,
		IncludeGenSpec:     false,
	}
}

// FullExportWithCredentials returns serialization options with all fields
// including credentials.
func FullExportWithCredentials() *SerializeOptions {
	return &SerializeOptions{
		IncludeExecution:   true,
		IncludeExtensions:  true,
		IncludeAgentFields: true,
		IncludeContext:     true,
		IncludeCredentials: true,
		IncludeGenSpec:     true,
	}
}

// Serialize outputs the Spec as a YAML frontmatter + body document.
// If opts is nil, DefaultSerializeOptions is used.
func (s *Spec) Serialize(opts *SerializeOptions) ([]byte, error) {
	if s == nil {
		return nil, nil
	}

	if opts == nil {
		opts = DefaultSerializeOptions()
	}

	// Build a serializable map based on options
	exportData := s.buildSerializeMap(opts)

	yamlBytes, err := yaml.Marshal(exportData)
	if err != nil {
		return nil, NewSerializeError(ErrMsgSerializeYAML, err)
	}

	var sb strings.Builder
	sb.WriteString(YAMLFrontmatterDelimiter)
	sb.WriteString("\n")
	sb.Write(yamlBytes)
	sb.WriteString(YAMLFrontmatterDelimiter)
	sb.WriteString("\n")
	if s.Body != "" {
		sb.WriteString(s.Body)
	}

	return []byte(sb.String()), nil
}

// ExportAgentSkill serializes the spec with only Agent Skills compatible fields.
func (s *Spec) ExportAgentSkill() ([]byte, error) {
	return s.Serialize(AgentSkillsExportOptions())
}

// ExportFull serializes the spec with all standard fields.
// Credentials are excluded for safety; use FullExportWithCredentials() to include them.
func (s *Spec) ExportFull() ([]byte, error) {
	return s.Serialize(DefaultSerializeOptions())
}

// knownSpecFields is the set of all known Spec struct YAML field names.
// Extensions with these keys are skipped during serialization to prevent overwriting.
var knownSpecFields = map[string]bool{
	SpecFieldName:          true,
	SpecFieldDescription:   true,
	SpecFieldLicense:       true,
	SpecFieldCompatibility: true,
	SpecFieldAllowedTools:  true,
	SpecFieldMetadata:      true,
	SpecFieldInputs:        true,
	SpecFieldOutputs:       true,
	SpecFieldSample:        true,
	SpecFieldType:          true,
	SpecFieldExecution:     true,
	SpecFieldExtensions:    true,
	SpecFieldSkills:        true,
	SpecFieldTools:         true,
	SpecFieldContext:        true,
	SpecFieldConstraints:   true,
	SpecFieldMessages:      true,
	SpecFieldCredentials:   true,
	SpecFieldCredential:    true,
	SpecFieldRequirements:  true,
	SpecFieldGenSpec:       true,
}

// buildSerializeMap creates a map for YAML serialization.
func (s *Spec) buildSerializeMap(opts *SerializeOptions) map[string]any {
	m := make(map[string]any)

	// Standard fields (always included)
	if s.Name != "" {
		m[SpecFieldName] = s.Name
	}
	if s.Description != "" {
		m[SpecFieldDescription] = s.Description
	}

	// Type (include if agent fields are included or if type is explicitly set)
	if opts.IncludeAgentFields && s.Type != "" {
		m[SpecFieldType] = string(s.Type)
	}

	// Execution config
	if opts.IncludeExecution && s.Execution != nil {
		m[SpecFieldExecution] = s.Execution
	}

	// Extensions (non-standard YAML keys, written as top-level keys)
	// Skip keys that match known Spec fields to prevent overwriting.
	if opts.IncludeExtensions && len(s.Extensions) > 0 {
		for k, v := range s.Extensions {
			if !knownSpecFields[k] {
				m[k] = v
			}
		}
	}

	// Inputs/Outputs/Sample (always included)
	if len(s.Inputs) > 0 {
		m[SpecFieldInputs] = s.Inputs
	}
	if len(s.Outputs) > 0 {
		m[SpecFieldOutputs] = s.Outputs
	}
	if len(s.Sample) > 0 {
		m[SpecFieldSample] = s.Sample
	}

	// Agent-specific fields
	if opts.IncludeAgentFields {
		if len(s.Skills) > 0 {
			m[SpecFieldSkills] = s.Skills
		}
		if s.Tools != nil && (len(s.Tools.Functions) > 0 || len(s.Tools.MCPServers) > 0) {
			m[SpecFieldTools] = s.Tools
		}
		if s.Constraints != nil {
			m[SpecFieldConstraints] = s.Constraints
		}
		if len(s.Messages) > 0 {
			m[SpecFieldMessages] = s.Messages
		}
	}

	// Context
	if opts.IncludeContext && len(s.Context) > 0 {
		m[SpecFieldContext] = s.Context
	}

	// Credentials (gated by IncludeCredentials -- safe default is off)
	if opts.IncludeCredentials {
		if len(s.Credentials) > 0 {
			m[SpecFieldCredentials] = s.Credentials
		}
		if s.Credential != "" {
			m[SpecFieldCredential] = s.Credential
		}
	}

	// GenSpec (gated by IncludeGenSpec)
	if opts.IncludeGenSpec && s.GenSpec != nil {
		m[SpecFieldGenSpec] = s.GenSpec
	}

	return m
}
