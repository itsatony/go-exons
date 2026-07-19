// Package a2a provides Google A2A protocol Agent Card types for agent discovery
// and orchestration. Types in this package are pure data structures with JSON tags —
// compilation logic lives in the root exons package.
//
// This is a pure metadata representation — no template execution or network
// communication occurs within this package.
//
// The types model A2A **v1.0.1** (github.com/a2aproject/A2A, specification/a2a.proto
// @ tag v1.0.1). The wire form is protojson lowerCamelCase. Notable shape vs the
// retired v0.3 card: transport moved from a single `url`+`preferredTransport` into a
// required `supportedInterfaces[]`, `protocolVersion` is now **per interface**, there
// is **no top-level `metadata`** (vendor data rides in `capabilities.extensions[]`),
// and cards may carry RFC-7515 JWS `signatures[]` (§8.4). See exons.card.validate.go
// for the pinned conformance rules.
package a2a

// AgentCard is a self-describing manifest for an agent (A2A v1.0.1 AgentCard).
//
// Thread safety: AgentCard instances are safe for concurrent read access once
// constructed. Use value copies for concurrent mutation.
//
// Required fields (per the proto's REQUIRED field_behavior) are emitted without
// omitempty so a conformant card always carries them; optional fields omitempty so
// the served card carries no explicit default values (which keeps §8.4 verification
// stable — a verifier that strips defaults reproduces our canonical payload byte for
// byte).
type AgentCard struct {
	// Name is the agent's human-readable display name (required).
	Name string `json:"name"`
	// Description explains the agent's purpose (required).
	Description string `json:"description"`
	// SupportedInterfaces is the ordered list of transport interfaces; the first
	// entry is preferred (required, MUST be non-empty). A declaration-only card
	// (aigentverse's case — Definition-plane, no runtime endpoint) still carries one
	// interface pointing at the registry definition URL with an open-form binding.
	SupportedInterfaces []AgentInterface `json:"supportedInterfaces"`
	// Provider identifies the organization publishing the agent.
	Provider *AgentProvider `json:"provider,omitempty"`
	// Version is the agent implementation version, e.g. "1.0.0" (required).
	Version string `json:"version"`
	// DocumentationURL points to additional documentation about the agent.
	DocumentationURL string `json:"documentationUrl,omitempty"`
	// Capabilities advertises supported protocol features + extensions (required).
	Capabilities *AgentCapabilities `json:"capabilities,omitempty"`
	// SecuritySchemes defines inbound authentication methods (OpenAPI-shaped, so
	// map[string]any because schemes vary widely per type).
	SecuritySchemes map[string]any `json:"securitySchemes,omitempty"`
	// SecurityRequirements lists the security schemes required to contact the agent.
	SecurityRequirements []map[string][]string `json:"securityRequirements,omitempty"`
	// DefaultInputModes lists accepted input media types (required).
	DefaultInputModes []string `json:"defaultInputModes"`
	// DefaultOutputModes lists produced output media types (required).
	DefaultOutputModes []string `json:"defaultOutputModes"`
	// Skills lists the agent's advertised abilities (required, MUST be non-empty).
	Skills []AgentSkill `json:"skills"`
	// Signatures carries RFC-7515 JWS signatures over this card (§8.4). Excluded
	// from the canonical payload during signing/verification.
	Signatures []AgentCardSignature `json:"signatures,omitempty"`
	// IconURL points to an icon for the agent.
	IconURL string `json:"iconUrl,omitempty"`
}

// AgentInterface is one transport endpoint the agent is reachable at.
type AgentInterface struct {
	// URL is the endpoint URL for this interface (required).
	URL string `json:"url"`
	// ProtocolBinding is the protocol binding at this URL. Open-form per the spec;
	// the core bindings are "JSONRPC", "GRPC", "HTTP+JSON".
	ProtocolBinding string `json:"protocolBinding"`
	// Tenant is an opaque routing identifier when multiple agents share an endpoint.
	Tenant string `json:"tenant,omitempty"`
	// ProtocolVersion is the A2A protocol version this interface exposes, e.g. "1.0".
	ProtocolVersion string `json:"protocolVersion"`
}

// AgentProvider identifies the organization running an agent.
type AgentProvider struct {
	// URL is the provider's website or relevant documentation (required).
	URL string `json:"url"`
	// Organization is the provider's name (required).
	Organization string `json:"organization"`
}

// AgentCapabilities advertises supported protocol features.
type AgentCapabilities struct {
	// Streaming indicates streaming-response support. Pointer so an unset capability
	// is omitted rather than serialized as an explicit false (default value).
	Streaming *bool `json:"streaming,omitempty"`
	// PushNotifications indicates async push-notification support.
	PushNotifications *bool `json:"pushNotifications,omitempty"`
	// Extensions lists supported protocol extensions. This is the spec's extension
	// point and the only place vendor metadata belongs on a v1.0.1 card.
	Extensions []AgentExtension `json:"extensions,omitempty"`
	// ExtendedAgentCard indicates an authenticated extended card is available.
	ExtendedAgentCard *bool `json:"extendedAgentCard,omitempty"`
}

// AgentExtension declares a protocol extension supported by the agent.
type AgentExtension struct {
	// URI uniquely identifies the extension (required by consumers).
	URI string `json:"uri"`
	// Description explains how the agent uses the extension.
	Description string `json:"description,omitempty"`
	// Required signals the client must comply with the extension.
	Required bool `json:"required,omitempty"`
	// Params carries extension-specific configuration (open JSON object).
	Params map[string]any `json:"params,omitempty"`
}

// AgentSkill is a distinct capability the agent advertises.
type AgentSkill struct {
	// ID uniquely identifies the skill (required).
	ID string `json:"id"`
	// Name is the skill's display name (required).
	Name string `json:"name"`
	// Description details what the skill does (required).
	Description string `json:"description"`
	// Tags describe the skill's capabilities (required, may be empty array).
	Tags []string `json:"tags"`
	// Examples are example prompts/scenarios the skill handles.
	Examples []string `json:"examples,omitempty"`
	// InputModes overrides the agent's default input modes for this skill.
	InputModes []string `json:"inputModes,omitempty"`
	// OutputModes overrides the agent's default output modes for this skill.
	OutputModes []string `json:"outputModes,omitempty"`
	// SecurityRequirements lists security schemes needed for this skill.
	SecurityRequirements []map[string][]string `json:"securityRequirements,omitempty"`
}

// AgentCardSignature is a JWS signature over an AgentCard, following the JSON
// serialization of an RFC-7515 JSON Web Signature (A2A §8.4 / §4.4.7).
type AgentCardSignature struct {
	// Protected is the base64url-encoded JWS Protected Header (required).
	Protected string `json:"protected"`
	// Signature is the base64url-encoded signature value (required).
	Signature string `json:"signature"`
	// Header carries any unprotected JWS header parameters.
	Header map[string]any `json:"header,omitempty"`
}
