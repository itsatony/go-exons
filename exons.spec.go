package exons

import (
	"regexp"

	"github.com/itsatony/go-exons/execution"
	"github.com/itsatony/go-exons/genspec"
	"gopkg.in/yaml.v3"
)

// Spec is the core configuration type parsed from .exons file frontmatter.
// It describes what an agent is: identity, execution parameters, tools,
// skills, constraints, and GenSpec metadata (memory, dispatch, verification).
//
// Thread safety: Spec instances are safe for concurrent read access once constructed.
// Concurrent mutation requires external synchronization.
// Use Clone() to create independent copies when concurrent modification is needed.
//
// The Type field determines validation rules:
//   - "prompt": Simple template, no skills/tools/genspec
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

	// GenSpec — agent specification metadata
	GenSpec *genspec.GenSpec `yaml:"genspec,omitempty" json:"genspec,omitempty"`

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
		return nil, nil
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

	case DocumentTypeSkill:
		if len(s.Skills) > 0 {
			return NewSpecValidationError(ErrMsgSkillNoSkillsAllowed, s.Name)
		}

	case DocumentTypeAgent:
		// Agent-specific validation can be added here
	}

	// Validate execution config if present
	if s.Execution != nil {
		if err := s.Execution.Validate(); err != nil {
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

	// Clone inputs
	if s.Inputs != nil {
		clone.Inputs = make(map[string]*InputDef, len(s.Inputs))
		for k, v := range s.Inputs {
			inputClone := *v
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

	// Clone tools
	if s.Tools != nil {
		toolsCopy := *s.Tools
		if s.Tools.Functions != nil {
			toolsCopy.Functions = make([]*FunctionDef, len(s.Tools.Functions))
			for i, f := range s.Tools.Functions {
				fCopy := *f
				if f.Parameters != nil {
					fCopy.Parameters = deepCopyMap(f.Parameters)
				}
				toolsCopy.Functions[i] = &fCopy
			}
		}
		if s.Tools.MCPServers != nil {
			toolsCopy.MCPServers = make([]*MCPServer, len(s.Tools.MCPServers))
			for i, m := range s.Tools.MCPServers {
				mCopy := *m
				toolsCopy.MCPServers[i] = &mCopy
			}
		}
		if s.Tools.Allow != nil {
			toolsCopy.Allow = make([]string, len(s.Tools.Allow))
			copy(toolsCopy.Allow, s.Tools.Allow)
		}
		if s.Tools.ParallelToolCalls != nil {
			t := *s.Tools.ParallelToolCalls
			toolsCopy.ParallelToolCalls = &t
		}
		clone.Tools = &toolsCopy
	}

	// Clone constraints
	if s.Constraints != nil {
		constraintsCopy := *s.Constraints
		if s.Constraints.Behavioral != nil {
			constraintsCopy.Behavioral = make([]string, len(s.Constraints.Behavioral))
			copy(constraintsCopy.Behavioral, s.Constraints.Behavioral)
		}
		if s.Constraints.Safety != nil {
			constraintsCopy.Safety = make([]string, len(s.Constraints.Safety))
			copy(constraintsCopy.Safety, s.Constraints.Safety)
		}
		if s.Constraints.Operational != nil {
			opCopy := *s.Constraints.Operational
			if s.Constraints.Operational.MaxTurns != nil {
				t := *s.Constraints.Operational.MaxTurns
				opCopy.MaxTurns = &t
			}
			if s.Constraints.Operational.MaxTokensPerTurn != nil {
				t := *s.Constraints.Operational.MaxTokensPerTurn
				opCopy.MaxTokensPerTurn = &t
			}
			if s.Constraints.Operational.AllowedDomains != nil {
				opCopy.AllowedDomains = make([]string, len(s.Constraints.Operational.AllowedDomains))
				copy(opCopy.AllowedDomains, s.Constraints.Operational.AllowedDomains)
			}
			if s.Constraints.Operational.BlockedDomains != nil {
				opCopy.BlockedDomains = make([]string, len(s.Constraints.Operational.BlockedDomains))
				copy(opCopy.BlockedDomains, s.Constraints.Operational.BlockedDomains)
			}
			if s.Constraints.Operational.TimeoutSeconds != nil {
				t := *s.Constraints.Operational.TimeoutSeconds
				opCopy.TimeoutSeconds = &t
			}
			if s.Constraints.Operational.MaxToolCalls != nil {
				t := *s.Constraints.Operational.MaxToolCalls
				opCopy.MaxToolCalls = &t
			}
			constraintsCopy.Operational = &opCopy
		}
		clone.Constraints = &constraintsCopy
	}

	// Clone messages
	if s.Messages != nil {
		clone.Messages = make([]MessageTemplate, len(s.Messages))
		copy(clone.Messages, s.Messages)
	}

	// Clone context
	if s.Context != nil {
		clone.Context = deepCopyMap(s.Context)
	}

	// Clone credentials
	if s.Credentials != nil {
		clone.Credentials = make(map[string]*CredentialRef, len(s.Credentials))
		for k, v := range s.Credentials {
			credCopy := *v
			if v.Scopes != nil {
				credCopy.Scopes = make([]string, len(v.Scopes))
				copy(credCopy.Scopes, v.Scopes)
			}
			clone.Credentials[k] = &credCopy
		}
	}

	// Clone genspec (shallow copy — deep copy would require GenSpec.Clone())
	if s.GenSpec != nil {
		genspecCopy := *s.GenSpec
		clone.GenSpec = &genspecCopy
	}

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
		len(s.Credentials) == 0 && s.Credential == "" && s.GenSpec == nil
}

// IsGenSpec returns true if this Spec has GenSpec metadata set.
func (s *Spec) IsGenSpec() bool {
	if s == nil {
		return false
	}
	return s.GenSpec != nil && s.GenSpec.HasContent()
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

// deepCopyMap, deepCopyValue, deepCopySlice are defined in exons.context.go.
