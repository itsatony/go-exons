package a2a

import "fmt"

// Offline conformance validation for A2A v1.0.1 Agent Cards.
//
// These rules are pinned by hand to the normative source below (there is no
// published stand-alone JSON Schema for v1.0.x; the proto's REQUIRED field_behavior
// annotations are authoritative). Keeping them here — zero-network, in the same
// package as the types — lets producers self-check a card before serving it and lets
// importers reject a malformed inbound card, without ever reaching out to a2a-protocol.org.
//
// When bumping the pinned version, update A2ASpecSource and re-derive the rules from
// the proto's REQUIRED annotations.
const A2ASpecSource = "github.com/a2aproject/A2A specification/a2a.proto @ v1.0.1"

// Violation is a single conformance problem found on a card.
type Violation struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (v Violation) String() string { return fmt.Sprintf("%s: %s", v.Field, v.Message) }

// Validate checks a card against the pinned A2A v1.0.1 required-field rules and
// returns every violation (empty slice ⇒ conformant). It validates structure, not
// signatures (use VerifySignatures for that).
func (c *AgentCard) Validate() []Violation {
	var vs []Violation
	add := func(field, msg string) { vs = append(vs, Violation{Field: field, Message: msg}) }

	if c == nil {
		return []Violation{{Field: "(card)", Message: "card is nil"}}
	}
	if c.Name == "" {
		add("name", "required and must be non-empty")
	}
	if c.Description == "" {
		add("description", "required and must be non-empty")
	}
	if c.Version == "" {
		add("version", "required and must be non-empty")
	}
	if len(c.SupportedInterfaces) == 0 {
		add("supportedInterfaces", "required and must declare at least one interface")
	}
	for i, iface := range c.SupportedInterfaces {
		if iface.URL == "" {
			add(fmt.Sprintf("supportedInterfaces[%d].url", i), "required")
		}
		if iface.ProtocolBinding == "" {
			add(fmt.Sprintf("supportedInterfaces[%d].protocolBinding", i), "required")
		}
		if iface.ProtocolVersion == "" {
			add(fmt.Sprintf("supportedInterfaces[%d].protocolVersion", i), "required")
		}
	}
	if c.Capabilities == nil {
		add("capabilities", "required")
	}
	if c.Provider != nil {
		if c.Provider.URL == "" {
			add("provider.url", "required when provider is present")
		}
		if c.Provider.Organization == "" {
			add("provider.organization", "required when provider is present")
		}
	}
	if len(c.DefaultInputModes) == 0 {
		add("defaultInputModes", "required and must be non-empty")
	}
	if len(c.DefaultOutputModes) == 0 {
		add("defaultOutputModes", "required and must be non-empty")
	}
	if len(c.Skills) == 0 {
		add("skills", "required and must declare at least one skill")
	}
	for i, s := range c.Skills {
		if s.ID == "" {
			add(fmt.Sprintf("skills[%d].id", i), "required")
		}
		if s.Name == "" {
			add(fmt.Sprintf("skills[%d].name", i), "required")
		}
		if s.Description == "" {
			add(fmt.Sprintf("skills[%d].description", i), "required")
		}
		if s.Tags == nil {
			add(fmt.Sprintf("skills[%d].tags", i), "required (may be an empty array, but must be present)")
		}
	}
	return vs
}
