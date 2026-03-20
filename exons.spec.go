package exons

import (
	"github.com/itsatony/go-exons/execution"
	"github.com/itsatony/go-exons/genspec"
)

// Spec is the core configuration type parsed from .exons file frontmatter.
// It describes what an agent is: identity, execution parameters, tools,
// skills, constraints, and GenSpec metadata (memory, dispatch, verification).
//
// The Type field determines validation rules:
//   - "prompt": Simple template, no skills/tools/genspec
//   - "skill":  Reusable capability, may have memory/registry/verification
//   - "agent":  Full agent with all fields available
type Spec struct {
	// Identity
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Type        string `yaml:"type,omitempty" json:"type,omitempty"`

	// Execution
	Execution *execution.Config `yaml:"execution,omitempty" json:"execution,omitempty"`

	// Schema
	Inputs  map[string]*InputDef  `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Outputs map[string]*OutputDef `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	Sample  map[string]any        `yaml:"sample,omitempty" json:"sample,omitempty"`

	// Agent composition
	Skills      []SkillRef          `yaml:"skills,omitempty" json:"skills,omitempty"`
	Tools       *ToolsConfig        `yaml:"tools,omitempty" json:"tools,omitempty"`
	Constraints *ConstraintsConfig  `yaml:"constraints,omitempty" json:"constraints,omitempty"`
	Messages    []MessageTemplate   `yaml:"messages,omitempty" json:"messages,omitempty"`
	Context     map[string]any      `yaml:"context,omitempty" json:"context,omitempty"`
	Credentials map[string]*CredentialRef `yaml:"credentials,omitempty" json:"credentials,omitempty"`
	Credential  string              `yaml:"credential,omitempty" json:"credential,omitempty"`

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
