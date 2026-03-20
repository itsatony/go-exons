package genspec

// VerificationCase is a declarative test specification for an agent.
// go-exons validates these; running them is the consumer's job.
type VerificationCase struct {
	// Name is the unique identifier for this verification (slug pattern).
	Name string `yaml:"name" json:"name"`

	// Description explains what this verification checks.
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Tags categorize this verification case.
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`

	// Input provides template variables for the verification.
	Input map[string]any `yaml:"input,omitempty" json:"input,omitempty"`

	// Prompt is the user message to send to the agent.
	Prompt string `yaml:"prompt,omitempty" json:"prompt,omitempty"`

	// Expect defines assertions on the agent's response.
	Expect *VerificationExpect `yaml:"expect,omitempty" json:"expect,omitempty"`

	// Ref points to an external verification definition (mutually exclusive with Expect).
	Ref string `yaml:"ref,omitempty" json:"ref,omitempty"`

	// TimeoutSeconds is the maximum time for this verification.
	TimeoutSeconds int `yaml:"timeout_seconds,omitempty" json:"timeout_seconds,omitempty"`
}

// VerificationExpect defines assertions for a verification case.
type VerificationExpect struct {
	ToolCalls          []string `yaml:"tool_calls,omitempty" json:"tool_calls,omitempty"`
	ToolCallsAbsent    []string `yaml:"tool_calls_absent,omitempty" json:"tool_calls_absent,omitempty"`
	OutputContains     string   `yaml:"output_contains,omitempty" json:"output_contains,omitempty"`
	OutputNotContains  string   `yaml:"output_not_contains,omitempty" json:"output_not_contains,omitempty"`
	OutputMatchesRegex string   `yaml:"output_matches_regex,omitempty" json:"output_matches_regex,omitempty"`
}
