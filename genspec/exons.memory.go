package genspec

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
