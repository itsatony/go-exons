package exons

import "regexp"

// validateRegexPattern checks if a string is a valid regular expression.
func validateRegexPattern(pattern string) error {
	_, err := regexp.Compile(pattern)
	return err
}

// MemorySpec configures agent memory behavior.
// go-exons stores this declaration; memory implementation is the consumer's job.
type MemorySpec struct {
	// Scope is the primary namespace for memory isolation.
	Scope string `yaml:"scope,omitempty" json:"scope,omitempty"`

	// AutoRecall hints that the runtime should recall relevant memories before each turn.
	AutoRecall *bool `yaml:"auto_recall,omitempty" json:"auto_recall,omitempty"`

	// AutoRecord hints that the runtime should record important facts after each turn.
	AutoRecord *bool `yaml:"auto_record,omitempty" json:"auto_record,omitempty"`

	// ReadScopes lists additional scopes to read from (read-only).
	ReadScopes []string `yaml:"read_scopes,omitempty" json:"read_scopes,omitempty"`
}

// HasMemory returns true if a scope is configured.
func (m *MemorySpec) HasMemory() bool {
	if m == nil {
		return false
	}
	return m.Scope != ""
}

// GetAutoRecall returns the auto_recall value and whether it was explicitly set.
func (m *MemorySpec) GetAutoRecall() (bool, bool) {
	if m == nil || m.AutoRecall == nil {
		return false, false
	}
	return *m.AutoRecall, true
}

// GetAutoRecord returns the auto_record value and whether it was explicitly set.
func (m *MemorySpec) GetAutoRecord() (bool, bool) {
	if m == nil || m.AutoRecord == nil {
		return false, false
	}
	return *m.AutoRecord, true
}

// Validate checks the MemorySpec for invalid field values.
// Scope and ReadScopes entries, if set, must match the slug pattern.
func (m *MemorySpec) Validate() error {
	if m == nil {
		return nil
	}
	if m.Scope != "" && !specSlugRegex.MatchString(m.Scope) {
		return NewMetadataValidationError(ErrMsgMemoryInvalidScope, m.Scope)
	}
	for _, rs := range m.ReadScopes {
		if !specSlugRegex.MatchString(rs) {
			return NewMetadataValidationError(ErrMsgMemoryReadScopeInvalid, rs)
		}
	}
	return nil
}

// Clone creates a deep copy of the MemorySpec.
func (m *MemorySpec) Clone() *MemorySpec {
	if m == nil {
		return nil
	}
	clone := &MemorySpec{
		Scope: m.Scope,
	}
	if m.AutoRecall != nil {
		v := *m.AutoRecall
		clone.AutoRecall = &v
	}
	if m.AutoRecord != nil {
		v := *m.AutoRecord
		clone.AutoRecord = &v
	}
	if m.ReadScopes != nil {
		clone.ReadScopes = make([]string, len(m.ReadScopes))
		copy(clone.ReadScopes, m.ReadScopes)
	}
	return clone
}

// DispatchSpec configures multi-agent routing rules.
// go-exons stores this declaration; orchestration is the consumer's job.
type DispatchSpec struct {
	// TriggerKeywords are terms that suggest this agent should handle a task.
	TriggerKeywords []string `yaml:"trigger_keywords,omitempty" json:"trigger_keywords,omitempty"`

	// TriggerDescription explains when to route tasks to this agent.
	TriggerDescription string `yaml:"trigger_description,omitempty" json:"trigger_description,omitempty"`

	// CostLimitUSD is the maximum cost budget for a delegated task.
	// Nil means no limit is set; zero means zero budget.
	CostLimitUSD *float64 `yaml:"cost_limit_usd,omitempty" json:"cost_limit_usd,omitempty"`
}

// HasTriggers returns true if any dispatch triggers are configured.
func (d *DispatchSpec) HasTriggers() bool {
	if d == nil {
		return false
	}
	return len(d.TriggerKeywords) > 0 || d.TriggerDescription != ""
}

// Validate checks the DispatchSpec for invalid field values.
// CostLimitUSD must be between 0 and 1000. TriggerKeywords entries must be non-empty.
func (d *DispatchSpec) Validate() error {
	if d == nil {
		return nil
	}
	if d.CostLimitUSD != nil && (*d.CostLimitUSD < 0 || *d.CostLimitUSD > DispatchCostLimitMax) {
		return NewMetadataValidationError(ErrMsgDispatchCostLimit, "")
	}
	for _, kw := range d.TriggerKeywords {
		if kw == "" {
			return NewMetadataValidationError(ErrMsgDispatchKeywordEmpty, "")
		}
	}
	return nil
}

// GetCostLimitUSD returns the cost limit and whether it was explicitly set.
func (d *DispatchSpec) GetCostLimitUSD() (float64, bool) {
	if d == nil || d.CostLimitUSD == nil {
		return 0, false
	}
	return *d.CostLimitUSD, true
}

// Clone creates a deep copy of the DispatchSpec.
func (d *DispatchSpec) Clone() *DispatchSpec {
	if d == nil {
		return nil
	}
	clone := &DispatchSpec{
		TriggerDescription: d.TriggerDescription,
	}
	if d.CostLimitUSD != nil {
		v := *d.CostLimitUSD
		clone.CostLimitUSD = &v
	}
	if d.TriggerKeywords != nil {
		clone.TriggerKeywords = make([]string, len(d.TriggerKeywords))
		copy(clone.TriggerKeywords, d.TriggerKeywords)
	}
	return clone
}

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

// Validate checks the VerificationCase for required fields and constraints.
// Name is required and must match slug pattern. Either Expect or Ref must be set,
// but not both. TimeoutSeconds, if set, must be between 1 and 600.
// Expect must have at least one assertion. OutputMatchesRegex must be valid regex.
func (vc *VerificationCase) Validate() error {
	if vc.Name == "" {
		return NewMetadataValidationError(ErrMsgVerifyNameRequired, "")
	}
	if !specSlugRegex.MatchString(vc.Name) {
		return NewMetadataValidationError(ErrMsgVerifyNameInvalid, vc.Name)
	}

	// Ref and Expect are mutually exclusive
	if vc.Ref != "" && vc.Expect != nil {
		return NewMetadataValidationError(ErrMsgVerifyRefAndExpect, vc.Name)
	}

	// Must have Expect or Ref
	if vc.Ref == "" && vc.Expect == nil {
		return NewMetadataValidationError(ErrMsgVerifyNoAssertions, vc.Name)
	}

	// Validate expect has at least one assertion
	if vc.Expect != nil {
		if !verificationExpectHasAssertion(vc.Expect) {
			return NewMetadataValidationError(ErrMsgVerifyNoAssertions, vc.Name)
		}
		// Validate regex compiles
		if vc.Expect.OutputMatchesRegex != "" {
			if err := validateRegexPattern(vc.Expect.OutputMatchesRegex); err != nil {
				return NewMetadataValidationError(ErrMsgVerifyRegexInvalid, vc.Name)
			}
		}
	}

	// Validate timeout
	if vc.TimeoutSeconds != 0 && (vc.TimeoutSeconds < VerifyTimeoutMin || vc.TimeoutSeconds > VerifyTimeoutMax) {
		return NewMetadataValidationError(ErrMsgVerifyTimeout, vc.Name)
	}

	return nil
}

// verificationExpectHasAssertion returns true if at least one assertion field is set.
func verificationExpectHasAssertion(ve *VerificationExpect) bool {
	return len(ve.ToolCalls) > 0 || len(ve.ToolCallsAbsent) > 0 ||
		ve.OutputContains != "" || ve.OutputNotContains != "" || ve.OutputMatchesRegex != ""
}

// Clone creates a deep copy of the VerificationCase.
func (vc *VerificationCase) Clone() VerificationCase {
	clone := VerificationCase{
		Name:           vc.Name,
		Description:    vc.Description,
		Prompt:         vc.Prompt,
		Ref:            vc.Ref,
		TimeoutSeconds: vc.TimeoutSeconds,
	}
	if vc.Tags != nil {
		clone.Tags = make([]string, len(vc.Tags))
		copy(clone.Tags, vc.Tags)
	}
	if vc.Input != nil {
		clone.Input = deepCopyMap(vc.Input)
	}
	if vc.Expect != nil {
		clone.Expect = vc.Expect.Clone()
	}
	return clone
}

// VerificationExpect defines assertions for a verification case.
type VerificationExpect struct {
	ToolCalls          []string `yaml:"tool_calls,omitempty" json:"tool_calls,omitempty"`
	ToolCallsAbsent    []string `yaml:"tool_calls_absent,omitempty" json:"tool_calls_absent,omitempty"`
	OutputContains     string   `yaml:"output_contains,omitempty" json:"output_contains,omitempty"`
	OutputNotContains  string   `yaml:"output_not_contains,omitempty" json:"output_not_contains,omitempty"`
	OutputMatchesRegex string   `yaml:"output_matches_regex,omitempty" json:"output_matches_regex,omitempty"`
}

// Clone creates a deep copy of the VerificationExpect.
func (ve *VerificationExpect) Clone() *VerificationExpect {
	if ve == nil {
		return nil
	}
	clone := &VerificationExpect{
		OutputContains:     ve.OutputContains,
		OutputNotContains:  ve.OutputNotContains,
		OutputMatchesRegex: ve.OutputMatchesRegex,
	}
	if ve.ToolCalls != nil {
		clone.ToolCalls = make([]string, len(ve.ToolCalls))
		copy(clone.ToolCalls, ve.ToolCalls)
	}
	if ve.ToolCallsAbsent != nil {
		clone.ToolCallsAbsent = make([]string, len(ve.ToolCallsAbsent))
		copy(clone.ToolCallsAbsent, ve.ToolCallsAbsent)
	}
	return clone
}

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

// Validate checks the RegistrySpec for invalid field values.
// Namespace, if set, must match slug pattern. Origin must be internal, external, or unknown.
func (r *RegistrySpec) Validate() error {
	if r == nil {
		return nil
	}
	if r.Namespace != "" && !specSlugRegex.MatchString(r.Namespace) {
		return NewMetadataValidationError(ErrMsgRegistryNamespace, r.Namespace)
	}
	if r.Origin != "" {
		switch r.Origin {
		case OriginInternal, OriginExternal, OriginUnknown:
			// valid
		default:
			return NewMetadataValidationError(ErrMsgRegistryOrigin, r.Origin)
		}
	}
	return nil
}

// Clone creates a deep copy of the RegistrySpec.
func (r *RegistrySpec) Clone() *RegistrySpec {
	if r == nil {
		return nil
	}
	return &RegistrySpec{
		Namespace: r.Namespace,
		Origin:    r.Origin,
		Version:   r.Version,
	}
}

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

// Validate checks the SafetyConfig for invalid field values.
// Guardrails, if set, must be "enabled" or "disabled".
func (sc *SafetyConfig) Validate() error {
	if sc == nil {
		return nil
	}
	if sc.Guardrails != "" {
		switch sc.Guardrails {
		case GuardrailsEnabled, GuardrailsDisabled:
			// valid
		default:
			return NewMetadataValidationError(ErrMsgSafetyGuardrails, sc.Guardrails)
		}
	}
	return nil
}

// Clone creates a deep copy of the SafetyConfig.
func (sc *SafetyConfig) Clone() *SafetyConfig {
	if sc == nil {
		return nil
	}
	clone := &SafetyConfig{
		Guardrails: sc.Guardrails,
	}
	if sc.RequireConfirmationFor != nil {
		clone.RequireConfirmationFor = make([]string, len(sc.RequireConfirmationFor))
		copy(clone.RequireConfirmationFor, sc.RequireConfirmationFor)
	}
	if sc.DenyTools != nil {
		clone.DenyTools = make([]string, len(sc.DenyTools))
		copy(clone.DenyTools, sc.DenyTools)
	}
	return clone
}

// Origin constants for RegistrySpec.
const (
	OriginInternal = "internal"
	OriginExternal = "external"
	OriginUnknown  = "unknown"
)

// Guardrails constants for SafetyConfig.
const (
	GuardrailsEnabled  = "enabled"
	GuardrailsDisabled = "disabled"
)

// Metadata validation limits.
const (
	DispatchCostLimitMax = 1000.0
	VerifyTimeoutMin     = 1
	VerifyTimeoutMax     = 600
)
