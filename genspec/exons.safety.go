package genspec

// SafetyConfig defines runtime safety guardrails.
// go-exons stores this declaration; enforcement is the consumer's job.
type SafetyConfig struct {
	// Guardrails enables or disables runtime safety checks.
	// Must be "enabled" or "disabled".
	Guardrails string `yaml:"guardrails,omitempty" json:"guardrails,omitempty"`

	// RequireConfirmationFor lists tool names that need user confirmation.
	RequireConfirmationFor []string `yaml:"require_confirmation_for,omitempty" json:"require_confirmation_for,omitempty"`

	// DenyTools lists tools that this agent must never invoke.
	DenyTools []string `yaml:"deny_tools,omitempty" json:"deny_tools,omitempty"`
}

// Guardrails constants.
const (
	GuardrailsEnabled  = "enabled"
	GuardrailsDisabled = "disabled"
)
