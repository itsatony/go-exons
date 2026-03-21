package exons

import (
	"regexp"

	"github.com/itsatony/go-exons/execution"
	"gopkg.in/yaml.v3"
)

// Spec is the core configuration type parsed from .exons file frontmatter.
// It describes what an agent is: identity, execution parameters, tools,
// skills, constraints, and metadata (memory, dispatch, verification, registry, safety).
//
// Thread safety: Spec instances are safe for concurrent read access once constructed.
// Concurrent mutation requires external synchronization.
// Use Clone() to create independent copies when concurrent modification is needed.
//
// The Type field determines validation rules:
//   - "prompt": Simple template, no skills/tools/metadata
//   - "skill":  Reusable capability, may have memory/registry/verification
//   - "agent":  Full agent with all fields available
type Spec struct {
	// Identity
	Name        string       `yaml:"name" json:"name"`
	Description string       `yaml:"description,omitempty" json:"description,omitempty"`
	Type        DocumentType `yaml:"type,omitempty" json:"type,omitempty"`

	// Execution
	Execution *execution.Config `yaml:"execution,omitempty" json:"execution,omitempty"`

	// Schema
	Inputs  map[string]*InputDef  `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Outputs map[string]*OutputDef `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	Sample  map[string]any        `yaml:"sample,omitempty" json:"sample,omitempty"`

	// Agent composition
	Skills      []SkillRef                `yaml:"skills,omitempty" json:"skills,omitempty"`
	Tools       *ToolsConfig              `yaml:"tools,omitempty" json:"tools,omitempty"`
	Constraints *ConstraintsConfig        `yaml:"constraints,omitempty" json:"constraints,omitempty"`
	Messages    []MessageTemplate         `yaml:"messages,omitempty" json:"messages,omitempty"`
	Context     map[string]any            `yaml:"context,omitempty" json:"context,omitempty"`
	Credentials map[string]*CredentialRef `yaml:"credentials,omitempty" json:"credentials,omitempty"`
	Credential  string                    `yaml:"credential,omitempty" json:"credential,omitempty"`

	// Metadata — agent specification metadata (flattened from genspec/)
	Memory        *MemorySpec        `yaml:"memory,omitempty" json:"memory,omitempty"`
	Dispatch      *DispatchSpec      `yaml:"dispatch,omitempty" json:"dispatch,omitempty"`
	Verifications []VerificationCase `yaml:"verifications,omitempty" json:"verifications,omitempty"`
	Registry      *RegistrySpec      `yaml:"registry,omitempty" json:"registry,omitempty"`
	Safety        *SafetyConfig      `yaml:"safety,omitempty" json:"safety,omitempty"`

	// Extensions — catch-all for unknown YAML keys
	Extensions map[string]any `yaml:",inline" json:"extensions,omitempty"`

	// Body — template content after frontmatter
	Body string `yaml:"-" json:"body,omitempty"`
}

// InputDef defines an input parameter for the spec.
type InputDef struct {
	Type        string `yaml:"type,omitempty" json:"type,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool   `yaml:"required,omitempty" json:"required,omitempty"`
	Default     any    `yaml:"default,omitempty" json:"default,omitempty"`
}

// OutputDef defines an output parameter for the spec.
type OutputDef struct {
	Type        string `yaml:"type,omitempty" json:"type,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// MessageTemplate defines a message in the spec frontmatter.
type MessageTemplate struct {
	Role    string `yaml:"role" json:"role"`
	Content string `yaml:"content" json:"content"`
	Cache   bool   `yaml:"cache,omitempty" json:"cache,omitempty"`
}

// specSlugRegex is the compiled regex for slug validation.
var specSlugRegex = regexp.MustCompile(SpecSlugPattern)

// isValidDocumentType checks if a DocumentType value is valid.
func isValidDocumentType(dt DocumentType) bool {
	switch dt {
	case DocumentTypePrompt, DocumentTypeSkill, DocumentTypeAgent:
		return true
	default:
		return false
	}
}

// ParseYAMLSpec parses YAML data into a Spec.
// Returns an error if the YAML data exceeds DefaultMaxFrontmatterSize (DoS protection).
func ParseYAMLSpec(yamlData string) (*Spec, error) {
	if yamlData == "" {
		return nil, NewFrontmatterError(ErrMsgFrontmatterEmpty, Position{}, nil)
	}

	// Check size limit to prevent DoS via large YAML
	if len(yamlData) > int(DefaultMaxFrontmatterSize) {
		return nil, NewFrontmatterError(ErrMsgFrontmatterTooLarge, Position{Line: 1, Column: 1}, nil)
	}

	var spec Spec
	if err := yaml.Unmarshal([]byte(yamlData), &spec); err != nil {
		return nil, NewFrontmatterParseError(err)
	}
	return &spec, nil
}

// Validate checks the spec configuration for required fields and constraints.
// Returns an error if validation fails, nil if valid.
func (s *Spec) Validate() error {
	if s == nil {
		return NewSpecNameRequiredError()
	}

	// Validate name (required, max length, slug format)
	if s.Name == "" {
		return NewSpecNameRequiredError()
	}
	if len(s.Name) > SpecNameMaxLength {
		return NewSpecNameTooLongError(s.Name, SpecNameMaxLength)
	}
	if !specSlugRegex.MatchString(s.Name) {
		return NewSpecNameInvalidFormatError(s.Name)
	}

	// Validate description (required, max length)
	if s.Description == "" {
		return NewSpecDescriptionRequiredError()
	}
	if len(s.Description) > SpecDescriptionMaxLength {
		return NewSpecDescriptionTooLongError(SpecDescriptionMaxLength)
	}

	// Validate document type if set
	if s.Type != "" && !isValidDocumentType(s.Type) {
		return NewSpecValidationError(ErrMsgInvalidDocumentType, string(s.Type))
	}

	// Type-specific validation
	effectiveType := s.EffectiveType()
	switch effectiveType {
	case DocumentTypePrompt:
		if len(s.Skills) > 0 {
			return NewSpecValidationError(ErrMsgPromptNoSkillsAllowed, s.Name)
		}
		if s.Tools != nil && len(s.Tools.Functions) > 0 {
			return NewSpecValidationError(ErrMsgPromptNoToolsAllowed, s.Name)
		}
		if s.Constraints != nil {
			return NewSpecValidationError(ErrMsgPromptNoConstraints, s.Name)
		}
		// Prompt type does not support memory, dispatch, or registry
		if s.Memory != nil {
			return NewSpecValidationError(ErrMsgPromptNoMemory, s.Name)
		}
		if s.Dispatch != nil {
			return NewSpecValidationError(ErrMsgPromptNoDispatch, s.Name)
		}
		if s.Registry != nil {
			return NewSpecValidationError(ErrMsgPromptNoRegistry, s.Name)
		}

	case DocumentTypeSkill:
		if len(s.Skills) > 0 {
			return NewSpecValidationError(ErrMsgSkillNoSkillsAllowed, s.Name)
		}
		// Skill type does not support dispatch
		if s.Dispatch != nil {
			return NewSpecValidationError(ErrMsgSkillNoDispatch, s.Name)
		}

	case DocumentTypeAgent:
		// Agent supports all fields
	}

	// Validate execution config if present
	if s.Execution != nil {
		if err := s.Execution.Validate(); err != nil {
			return err
		}
	}

	// Validate metadata types if present
	if s.Memory != nil {
		if err := s.Memory.Validate(); err != nil {
			return err
		}
	}
	if s.Dispatch != nil {
		if err := s.Dispatch.Validate(); err != nil {
			return err
		}
	}
	for i := range s.Verifications {
		if err := s.Verifications[i].Validate(); err != nil {
			return err
		}
	}
	if s.Registry != nil {
		if err := s.Registry.Validate(); err != nil {
			return err
		}
	}
	if s.Safety != nil {
		if err := s.Safety.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ValidateOptional performs validation only if the spec has enough fields to
// indicate it is a well-formed document. Specs with Execution, Type, or a
// Name are validated; bare frontmatter with none of these fields is silently accepted.
func (s *Spec) ValidateOptional() error {
	if s == nil {
		return nil
	}
	// If the spec has specific fields, require full validation
	if s.Execution != nil || s.Type != "" || s.Name != "" || len(s.Credentials) > 0 {
		return s.Validate()
	}
	return nil
}

// GetSlug returns the spec name as the slug identifier.
func (s *Spec) GetSlug() string {
	if s == nil {
		return ""
	}
	return s.Name
}

// EffectiveType returns the document type, defaulting to "skill" if not set.
func (s *Spec) EffectiveType() DocumentType {
	if s == nil || s.Type == "" {
		return DocumentTypeSkill
	}
	return s.Type
}

// Clone creates a deep copy of the Spec.
func (s *Spec) Clone() *Spec {
	if s == nil {
		return nil
	}

	clone := &Spec{
		Name:        s.Name,
		Description: s.Description,
		Type:        s.Type,
		Credential:  s.Credential,
		Body:        s.Body,
	}

	// Clone execution (delegates to Config.Clone)
	if s.Execution != nil {
		clone.Execution = s.Execution.Clone()
	}

	// Clone inputs (deep-copy Default which is type any)
	if s.Inputs != nil {
		clone.Inputs = make(map[string]*InputDef, len(s.Inputs))
		for k, v := range s.Inputs {
			inputClone := *v
			inputClone.Default = deepCopyValue(v.Default)
			clone.Inputs[k] = &inputClone
		}
	}

	// Clone outputs
	if s.Outputs != nil {
		clone.Outputs = make(map[string]*OutputDef, len(s.Outputs))
		for k, v := range s.Outputs {
			outputClone := *v
			clone.Outputs[k] = &outputClone
		}
	}

	// Clone sample
	if s.Sample != nil {
		clone.Sample = deepCopyMap(s.Sample)
	}

	// Clone skills
	if s.Skills != nil {
		clone.Skills = make([]SkillRef, len(s.Skills))
		copy(clone.Skills, s.Skills)
	}

	// Clone tools (delegates to ToolsConfig.Clone)
	clone.Tools = s.Tools.Clone()

	// Clone constraints (delegates to ConstraintsConfig.Clone)
	clone.Constraints = s.Constraints.Clone()

	// Clone messages
	if s.Messages != nil {
		clone.Messages = make([]MessageTemplate, len(s.Messages))
		copy(clone.Messages, s.Messages)
	}

	// Clone context
	if s.Context != nil {
		clone.Context = deepCopyMap(s.Context)
	}

	// Clone credentials (delegates to CredentialRef.Clone)
	if s.Credentials != nil {
		clone.Credentials = make(map[string]*CredentialRef, len(s.Credentials))
		for k, v := range s.Credentials {
			clone.Credentials[k] = v.Clone()
		}
	}

	// Clone metadata fields (deep copy via per-type Clone methods)
	clone.Memory = s.Memory.Clone()
	clone.Dispatch = s.Dispatch.Clone()
	if s.Verifications != nil {
		clone.Verifications = make([]VerificationCase, len(s.Verifications))
		for i := range s.Verifications {
			clone.Verifications[i] = s.Verifications[i].Clone()
		}
	}
	clone.Registry = s.Registry.Clone()
	clone.Safety = s.Safety.Clone()

	// Clone extensions
	if s.Extensions != nil {
		clone.Extensions = deepCopyMap(s.Extensions)
	}

	return clone
}

// HasExecution returns true if execution config is present.
func (s *Spec) HasExecution() bool {
	return s != nil && s.Execution != nil
}

// HasTools returns true if tools config is present with at least one definition.
func (s *Spec) HasTools() bool {
	return s != nil && s.Tools != nil && (len(s.Tools.Functions) > 0 || len(s.Tools.MCPServers) > 0)
}

// HasSkills returns true if any skills are referenced.
func (s *Spec) HasSkills() bool {
	return s != nil && len(s.Skills) > 0
}

// HasConstraints returns true if constraints config is present.
func (s *Spec) HasConstraints() bool {
	return s != nil && s.Constraints != nil
}

// HasExtensions returns true if any extensions are set.
func (s *Spec) HasExtensions() bool {
	return s != nil && len(s.Extensions) > 0
}

// HasCredentials returns true if any credential references are defined.
func (s *Spec) HasCredentials() bool {
	return s != nil && len(s.Credentials) > 0
}

// IsAgentSkillsCompatible returns true if the spec contains only
// standard fields (no execution, extensions, or agent-specific config).
func (s *Spec) IsAgentSkillsCompatible() bool {
	if s == nil {
		return true
	}
	return s.Execution == nil && len(s.Extensions) == 0 && s.Type == "" &&
		len(s.Skills) == 0 && s.Tools == nil && s.Constraints == nil && len(s.Messages) == 0 &&
		len(s.Credentials) == 0 && s.Credential == "" &&
		s.Memory == nil && s.Dispatch == nil && len(s.Verifications) == 0 &&
		s.Registry == nil && s.Safety == nil
}

// HasMetadata returns true if this Spec has any metadata fields set
// (memory, dispatch, verifications, registry, safety).
func (s *Spec) HasMetadata() bool {
	if s == nil {
		return false
	}
	return s.Memory != nil || s.Dispatch != nil || len(s.Verifications) > 0 ||
		s.Registry != nil || s.Safety != nil
}

// IsAgent returns true if this Spec is of type agent.
func (s *Spec) IsAgent() bool {
	if s == nil {
		return false
	}
	return s.Type == DocumentTypeAgent
}

// IsSkill returns true if this Spec is of type skill.
func (s *Spec) IsSkill() bool {
	if s == nil {
		return false
	}
	return s.Type == DocumentTypeSkill
}

// IsPrompt returns true if this Spec is of type prompt.
func (s *Spec) IsPrompt() bool {
	if s == nil {
		return false
	}
	return s.Type == DocumentTypePrompt
}

// ValidateCredentialRefs validates the credential map, default label, and skill credential labels.
// Returns an error if the default credential label doesn't exist in the map, or if any
// credential ref fails validation, or if any skill references a credential label not in the map.
func (s *Spec) ValidateCredentialRefs() error {
	if s == nil {
		return nil
	}

	// Validate each credential ref
	for label, cred := range s.Credentials {
		if cred == nil {
			continue
		}
		if err := cred.Validate(); err != nil {
			return NewCredentialValidationError(label, err)
		}
	}

	// Validate default credential label resolves
	if s.Credential != "" && len(s.Credentials) > 0 {
		if _, exists := s.Credentials[s.Credential]; !exists {
			return NewSpecValidationError(ErrMsgCredentialMissingRef, s.Credential)
		}
	}

	// Validate skill credential labels
	for _, skill := range s.Skills {
		if skill.Credential != "" && len(s.Credentials) > 0 {
			if _, exists := s.Credentials[skill.Credential]; !exists {
				return NewSpecValidationError(ErrMsgCredentialMissingRef, skill.Credential)
			}
		}
	}

	return nil
}

// deepCopyMap, deepCopyValue, deepCopySlice are defined in exons.context.go.
