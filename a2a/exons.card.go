// Package a2a provides Google A2A protocol Agent Card types for agent discovery
// and orchestration. Types in this package are pure data structures with JSON tags —
// compilation logic lives in the root exons package.
//
// This is a pure metadata representation — no template execution or network
// communication occurs within this package.
package a2a

// AgentCard represents a Google A2A protocol Agent Card (v0.3).
// An Agent Card describes an agent's capabilities, skills, and communication modes
// for discovery and orchestration on A2A networks.
//
// Thread safety: AgentCard instances are safe for concurrent read access once constructed.
// Use value copies for concurrent mutation.
type AgentCard struct {
	// Name is the agent's display name (required)
	Name string `json:"name"`
	// Description of the agent's purpose
	Description string `json:"description,omitempty"`
	// URL is the agent's service endpoint (required)
	URL string `json:"url"`
	// Version of the agent implementation
	Version string `json:"version,omitempty"`
	// ProtocolVersion is the A2A protocol version (defaults to "0.3.0")
	ProtocolVersion string `json:"protocolVersion"`
	// Provider identifies the organization running the agent
	Provider *Provider `json:"provider,omitempty"`
	// Capabilities advertises what the agent supports
	Capabilities *Capabilities `json:"capabilities,omitempty"`
	// Skills lists the agent's advertised capabilities
	Skills []Skill `json:"skills,omitempty"`
	// DefaultInputModes lists accepted input MIME types (e.g., "text/plain", "application/json")
	DefaultInputModes []string `json:"defaultInputModes,omitempty"`
	// DefaultOutputModes lists produced output MIME types
	DefaultOutputModes []string `json:"defaultOutputModes,omitempty"`
	// SecuritySchemes defines inbound authentication methods.
	// Uses map[string]any because A2A security schemes follow the OpenAPI spec
	// and vary widely per scheme type (bearer, apiKey, oauth2, etc.).
	SecuritySchemes map[string]any `json:"securitySchemes,omitempty"`
	// Security references required security schemes
	Security []map[string][]string `json:"security,omitempty"`
	// Metadata contains additional key-value pairs.
	// Uses map[string]any because A2A metadata is an open-ended extension point
	// with no fixed schema — values can be strings, numbers, arrays, or nested objects.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Provider identifies the organization running an agent.
type Provider struct {
	// Organization is the provider's name
	Organization string `json:"organization"`
	// URL is the provider's website
	URL string `json:"url,omitempty"`
}

// Capabilities advertises what protocol features the agent supports.
type Capabilities struct {
	// Streaming indicates if the agent supports streaming responses
	Streaming bool `json:"streaming,omitempty"`
	// PushNotifications indicates if the agent supports push notifications
	PushNotifications bool `json:"pushNotifications,omitempty"`
}

// Skill represents a capability the agent advertises to other agents.
type Skill struct {
	// ID is the unique identifier for this skill
	ID string `json:"id"`
	// Name is the display name
	Name string `json:"name"`
	// Description explains what the skill does
	Description string `json:"description,omitempty"`
	// Tags for categorization
	Tags []string `json:"tags,omitempty"`
	// InputModes overrides the agent's default input modes for this skill
	InputModes []string `json:"inputModes,omitempty"`
	// OutputModes overrides the agent's default output modes for this skill
	OutputModes []string `json:"outputModes,omitempty"`
}
