package genspec

// DispatchSpec configures multi-agent routing rules.
// go-exons stores this declaration; orchestration is the consumer's job.
type DispatchSpec struct {
	// TriggerKeywords are terms that suggest this agent should handle a task.
	TriggerKeywords []string `yaml:"trigger_keywords,omitempty" json:"trigger_keywords,omitempty"`

	// TriggerDescription explains when to route tasks to this agent.
	TriggerDescription string `yaml:"trigger_description,omitempty" json:"trigger_description,omitempty"`

	// CostLimitUSD is the maximum cost budget for a delegated task.
	CostLimitUSD float64 `yaml:"cost_limit_usd,omitempty" json:"cost_limit_usd,omitempty"`
}

// HasTriggers returns true if any dispatch triggers are configured.
func (d *DispatchSpec) HasTriggers() bool {
	if d == nil {
		return false
	}
	return len(d.TriggerKeywords) > 0 || d.TriggerDescription != ""
}
