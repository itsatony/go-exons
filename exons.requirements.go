package exons

// Requirement scope values. They resolve the MCP/credential "mixed bag":
//   - org:      one shared credential for the whole org
//   - user:     resolved per invoking identity at runtime (same portable
//     definition, different secret per caller)
//   - per_call: supplied at invocation, never stored
//
// The portable definition is identical across all three; only resolution differs.
const (
	RequirementScopeOrg     = "org"
	RequirementScopeUser    = "user"
	RequirementScopePerCall = "per_call"
)

// Bounds on the requirements block (defense-in-depth beyond the whole-document
// frontmatter size cap): cap list length and per-field length so the governance
// seam cannot be flooded with oversized or excessive entries.
const (
	MaxRequirementEntries  = 256
	MaxRequirementFieldLen = 512
)

// SpecRequirements is an additive, portable block declaring the abstract capability
// and credential needs of a definition WITHOUT binding them. It is the seam that
// keeps a definition portable while letting governance and authoring-time
// preflight have teeth: it carries abstract capabilities and logical credential
// refs + scope — never server URLs, never secrets. Resolution (capability →
// concrete MCP server, ref → secret) happens at runtime in other systems.
type SpecRequirements struct {
	// MCP declares abstract MCP capabilities the definition needs.
	MCP []MCPRequirement `yaml:"mcp,omitempty" json:"mcp,omitempty"`
	// Credentials declares logical credential refs the definition needs.
	Credentials []CredentialRequirement `yaml:"credentials,omitempty" json:"credentials,omitempty"`
}

// MCPRequirement declares one abstract MCP capability requirement. Capability is
// an abstract name (e.g. "dns-management"), not a server URL — a router resolves
// it to a concrete server at runtime. CredentialRef is a logical name, not a
// secret.
type MCPRequirement struct {
	Capability    string `yaml:"capability" json:"capability"`
	CredentialRef string `yaml:"credential_ref,omitempty" json:"credential_ref,omitempty"`
	Scope         string `yaml:"scope,omitempty" json:"scope,omitempty"`
}

// CredentialRequirement declares one logical credential requirement. Ref is a
// logical name the binding plane maps to a resolver coordinate; the resolver
// releases the secret at runtime. The definition never carries the secret.
type CredentialRequirement struct {
	Ref      string `yaml:"ref" json:"ref"`
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`
	Scope    string `yaml:"scope,omitempty" json:"scope,omitempty"`
}

// isValidRequirementScope reports whether scope is a recognized scope value. An
// empty scope is allowed (it defaults during resolution).
func isValidRequirementScope(scope string) bool {
	switch scope {
	case "", RequirementScopeOrg, RequirementScopeUser, RequirementScopePerCall:
		return true
	default:
		return false
	}
}

// Validate checks the requirements block shape: non-empty capabilities/refs,
// capability and ref uniqueness, and a valid scope enum on every entry. A nil
// Requirements is valid (the block is optional).
func (r *SpecRequirements) Validate() error {
	if r == nil {
		return nil
	}
	if len(r.MCP) > MaxRequirementEntries || len(r.Credentials) > MaxRequirementEntries {
		return NewSpecValidationError(ErrMsgRequirementTooManyEntries, "")
	}

	seenCapabilities := make(map[string]struct{}, len(r.MCP))
	for _, m := range r.MCP {
		if m.Capability == "" {
			return NewSpecValidationError(ErrMsgRequirementCapabilityEmpty, "")
		}
		if len(m.Capability) > MaxRequirementFieldLen || len(m.CredentialRef) > MaxRequirementFieldLen {
			return NewSpecValidationError(ErrMsgRequirementFieldTooLong, m.Capability)
		}
		if _, dup := seenCapabilities[m.Capability]; dup {
			return NewSpecValidationError(ErrMsgRequirementCapabilityDup, m.Capability)
		}
		seenCapabilities[m.Capability] = struct{}{}
		if !isValidRequirementScope(m.Scope) {
			return NewSpecValidationError(ErrMsgRequirementScopeInvalid, m.Capability)
		}
	}

	seenRefs := make(map[string]struct{}, len(r.Credentials))
	for _, c := range r.Credentials {
		if c.Ref == "" {
			return NewSpecValidationError(ErrMsgRequirementCredRefEmpty, "")
		}
		if len(c.Ref) > MaxRequirementFieldLen || len(c.Provider) > MaxRequirementFieldLen {
			return NewSpecValidationError(ErrMsgRequirementFieldTooLong, c.Ref)
		}
		if _, dup := seenRefs[c.Ref]; dup {
			return NewSpecValidationError(ErrMsgRequirementCredRefDup, c.Ref)
		}
		seenRefs[c.Ref] = struct{}{}
		if !isValidRequirementScope(c.Scope) {
			return NewSpecValidationError(ErrMsgRequirementScopeInvalid, c.Ref)
		}
	}

	return nil
}

// Clone returns a deep copy of the SpecRequirements block. copy() is a true deep
// copy here because MCPRequirement and CredentialRequirement are all-scalar value
// types; if either gains a slice/map/pointer field, switch to per-element deep
// copies (as Spec.Clone does) to preserve the deep-copy guarantee.
func (r *SpecRequirements) Clone() *SpecRequirements {
	if r == nil {
		return nil
	}
	clone := &SpecRequirements{}
	if r.MCP != nil {
		clone.MCP = make([]MCPRequirement, len(r.MCP))
		copy(clone.MCP, r.MCP)
	}
	if r.Credentials != nil {
		clone.Credentials = make([]CredentialRequirement, len(r.Credentials))
		copy(clone.Credentials, r.Credentials)
	}
	return clone
}

// ValidateRequirements validates the spec's requirements block (shape, ref
// uniqueness, scope enum). It is safe to call on a nil spec or a spec without a
// requirements block.
func (s *Spec) ValidateRequirements() error {
	if s == nil {
		return nil
	}
	return s.Requirements.Validate()
}
