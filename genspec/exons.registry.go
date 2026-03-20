package genspec

// RegistrySpec provides metadata for agent catalogs and marketplaces.
type RegistrySpec struct {
	// Namespace is the unique identifier for this agent in the registry.
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty"`

	// Origin describes provenance: internal, external, or unknown.
	// This is a declaration of source, not a trust assertion.
	Origin string `yaml:"origin,omitempty" json:"origin,omitempty"`

	// Version is the semantic version of this agent specification.
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}

// Origin constants.
const (
	OriginInternal = "internal"
	OriginExternal = "external"
	OriginUnknown  = "unknown"
)
