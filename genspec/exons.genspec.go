// Package genspec provides the GenSpec container and types for agent specification
// metadata: memory configuration, dispatch rules, verification cases, registry
// metadata, and safety configuration.
//
// GenSpec fields are only valid on certain document types:
//
//	| Type    | Memory | Dispatch | Verification | Registry |
//	|---------|--------|----------|--------------|----------|
//	| prompt  | NO     | NO       | YES          | NO       |
//	| skill   | YES    | NO       | YES          | YES      |
//	| agent   | YES    | YES      | YES          | YES      |
package genspec

// GenSpec is the container for agent specification metadata.
// It is a single field on [exons.Spec], keeping the core type focused.
type GenSpec struct {
	// Version of the genspec format. Currently "1".
	Version string `yaml:"version,omitempty" json:"version,omitempty"`

	// Memory configuration for agent state persistence.
	Memory *MemorySpec `yaml:"memory,omitempty" json:"memory,omitempty"`

	// Dispatch rules for multi-agent routing.
	Dispatch *DispatchSpec `yaml:"dispatch,omitempty" json:"dispatch,omitempty"`

	// Verification cases for declarative agent testing.
	Verifications []VerificationCase `yaml:"verifications,omitempty" json:"verifications,omitempty"`

	// Registry metadata for agent catalogs and marketplaces.
	Registry *RegistrySpec `yaml:"registry,omitempty" json:"registry,omitempty"`

	// Safety configuration for runtime guardrails.
	Safety *SafetyConfig `yaml:"safety,omitempty" json:"safety,omitempty"`
}

// HasContent returns true if any GenSpec field is populated.
func (g *GenSpec) HasContent() bool {
	if g == nil {
		return false
	}
	return g.Memory != nil ||
		g.Dispatch != nil ||
		len(g.Verifications) > 0 ||
		g.Registry != nil ||
		g.Safety != nil
}
