package exons

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

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

// Resource access values. Access says what the definition does WITH a resource,
// which is what a registry needs to decide whether a binding may be granted:
//   - read:  the definition only reads the resource (the default — an empty
//     access resolves to read, exactly as an empty scope resolves to org)
//   - write: the definition may modify it
const (
	ResourceAccessRead  = "read"
	ResourceAccessWrite = "write"
)

// ResourceKindPattern is the shape of ResourceRequirement.Kind: a lowercase
// token. exons attaches NO meaning to a kind — it is interpreted only by the
// runtime that binds it — so the one thing the format guarantees is that two
// runtimes can never disagree about it by CASE ("Corpus" vs "corpus"). The
// schema states the same pattern; the agreement test holds the two together.
const ResourceKindPattern = `^[a-z][a-z0-9_.-]*$`

// requirementCoordinateMarker is the substring that makes a value a concrete
// coordinate rather than a logical name. A resource's ref and kind refuse it:
// the design keeps tenant-specific locations (and the tenant ids they carry)
// in the registry's binding plane, so a definition exported from one tenant
// never names a location inside it.
//
// ⚠ IT IS A NARROW HEURISTIC AND IS DELIBERATELY NOT WIDENED: it catches
// URI-shaped coordinates ("s3://…", "https://…") and nothing else, so
// "vault:secret/x", "/mnt/docs" or "urn:…" pass. It is a guard against the
// common paste, not a proof that a value is logical — a registry must still
// treat every ref as a logical name and bind it, never dereference it.
const requirementCoordinateMarker = "://"

// Field labels naming which resource field a refusal is about, carried as the
// error's context so a coordinate found in kind is not reported against ref.
const (
	resourceFieldRef  = "ref"
	resourceFieldKind = "kind"
)

var resourceKindRegex = regexp.MustCompile(ResourceKindPattern)

// Bounds on the requirements block (defense-in-depth beyond the whole-document
// frontmatter size cap): cap list length and per-field length so the governance
// seam cannot be flooded with oversized or excessive entries.
//
// MaxRequirementFieldLen counts CHARACTERS (runes), not bytes, since v0.31.0 —
// the schema's maxLength counts characters, and a byte cap made the parser
// refuse a 512-character non-ASCII value the published schema accepted.
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
	// Resources declares logical resources (a document corpus, a folder, a
	// toolset…) the definition needs, bound to a concrete coordinate per
	// workspace by a registry. Added in v0.31.0.
	Resources []ResourceRequirement `yaml:"resources,omitempty" json:"resources,omitempty"`
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

// ResourceRequirement declares one logical resource requirement.
//
// ⚠ REF IS A LOGICAL NAME, NEVER THE COORDINATE. "product-docs" is portable;
// "s3://acme-tenant-42/docs" is a location inside one tenant, and a definition
// carrying it leaks that tenant into every export. A registry (aigentverse)
// binds the ref to a coordinate per workspace; Validate refuses any ref or kind
// containing "://" to keep that line where the design draws it.
type ResourceRequirement struct {
	// Ref is REQUIRED: the logical name a binding maps to a concrete coordinate.
	// Unique within resources, compared verbatim (never trimmed or case-folded).
	Ref string `yaml:"ref" json:"ref"`
	// Kind is REQUIRED: an opaque, runtime-interpreted type token ("corpus",
	// "folder", "toolset"). exons attaches no meaning to it beyond its shape,
	// ResourceKindPattern.
	Kind string `yaml:"kind" json:"kind"`
	// Access is ResourceAccessRead or ResourceAccessWrite; empty resolves to
	// read (EffectiveAccess).
	Access string `yaml:"access,omitempty" json:"access,omitempty"`
	// Scope uses the same org|user|per_call vocabulary as the other
	// requirements; empty resolves to org (EffectiveScope).
	Scope string `yaml:"scope,omitempty" json:"scope,omitempty"`
	// Purpose is optional human prose: why the definition needs the resource.
	Purpose string `yaml:"purpose,omitempty" json:"purpose,omitempty"`
}

// EffectiveAccess returns the access this requirement resolves to: the
// declared value, or ResourceAccessRead when none was declared. It does not
// validate — an out-of-vocabulary value is returned verbatim, and Validate is
// what refuses it.
func (r ResourceRequirement) EffectiveAccess() string {
	if r.Access == "" {
		return ResourceAccessRead
	}
	return r.Access
}

// EffectiveScope returns the scope this requirement resolves to: the declared
// value, or RequirementScopeOrg when none was declared. Like EffectiveAccess
// it returns an out-of-vocabulary value verbatim rather than correcting it.
func (r ResourceRequirement) EffectiveScope() string {
	if r.Scope == "" {
		return RequirementScopeOrg
	}
	return r.Scope
}

// isValidResourceAccess reports whether access is a recognized access value.
// An empty access is allowed (it resolves to read).
func isValidResourceAccess(access string) bool {
	switch access {
	case "", ResourceAccessRead, ResourceAccessWrite:
		return true
	default:
		return false
	}
}

// requirementFieldTooLong reports whether any value exceeds
// MaxRequirementFieldLen characters. Runes, not bytes: see the constant.
func requirementFieldTooLong(values ...string) bool {
	for _, v := range values {
		if utf8.RuneCountInString(v) > MaxRequirementFieldLen {
			return true
		}
	}
	return false
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
// capability and ref uniqueness, and a valid scope enum on every entry — and,
// for resources, a non-empty kind in ResourceKindPattern's shape, a valid
// access enum, and no coordinate ("://") in ref or kind. A nil
// Requirements is valid (the block is optional).
func (r *SpecRequirements) Validate() error {
	if r == nil {
		return nil
	}
	if len(r.MCP) > MaxRequirementEntries || len(r.Credentials) > MaxRequirementEntries ||
		len(r.Resources) > MaxRequirementEntries {
		return NewSpecValidationError(ErrMsgRequirementTooManyEntries, "")
	}

	seenCapabilities := make(map[string]struct{}, len(r.MCP))
	for _, m := range r.MCP {
		if m.Capability == "" {
			return NewSpecValidationError(ErrMsgRequirementCapabilityEmpty, "")
		}
		if requirementFieldTooLong(m.Capability, m.CredentialRef) {
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
		if requirementFieldTooLong(c.Ref, c.Provider) {
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

	return r.validateResources()
}

// validateResources checks the resources list. The whitespace policy is the
// other lists': values are compared VERBATIM — never trimmed — because every
// consumer keys bindings on the exact string, and a validator that trimmed
// would bless a ref no binding lookup can find.
func (r *SpecRequirements) validateResources() error {
	seen := make(map[string]struct{}, len(r.Resources))
	for _, res := range r.Resources {
		if res.Ref == "" {
			return NewSpecValidationError(ErrMsgRequirementResourceRefEmpty, "")
		}
		if res.Kind == "" {
			return NewSpecValidationError(ErrMsgRequirementResourceKindEmpty, res.Ref)
		}
		if requirementFieldTooLong(res.Ref, res.Kind, res.Purpose) {
			return NewSpecValidationError(ErrMsgRequirementFieldTooLong, res.Ref)
		}
		// Before the kind pattern, so a coordinate pasted into kind gets the
		// sentence that says where it belongs rather than a shape complaint.
		if strings.Contains(res.Ref, requirementCoordinateMarker) {
			return NewSpecValidationError(ErrMsgRequirementResourceCoordinate, resourceFieldRef+"="+res.Ref)
		}
		if strings.Contains(res.Kind, requirementCoordinateMarker) {
			return NewSpecValidationError(ErrMsgRequirementResourceCoordinate, resourceFieldKind+"="+res.Kind)
		}
		if !resourceKindRegex.MatchString(res.Kind) {
			return NewSpecValidationError(ErrMsgRequirementResourceKindForm, res.Ref)
		}
		if _, dup := seen[res.Ref]; dup {
			return NewSpecValidationError(ErrMsgRequirementResourceRefDup, res.Ref)
		}
		seen[res.Ref] = struct{}{}
		if !isValidResourceAccess(res.Access) {
			return NewSpecValidationError(ErrMsgRequirementResourceAccess, res.Ref)
		}
		if !isValidRequirementScope(res.Scope) {
			return NewSpecValidationError(ErrMsgRequirementScopeInvalid, res.Ref)
		}
	}
	return nil
}

// Clone returns a deep copy of the SpecRequirements block. copy() is a true deep
// copy here because MCPRequirement, CredentialRequirement and ResourceRequirement
// are all-scalar value types; if any gains a slice/map/pointer field, switch to per-element deep
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
	if r.Resources != nil {
		clone.Resources = make([]ResourceRequirement, len(r.Resources))
		copy(clone.Resources, r.Resources)
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
