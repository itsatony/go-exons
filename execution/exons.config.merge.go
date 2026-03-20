package execution

// Merge creates a new Config that merges other into the receiver.
// The other config's non-nil/non-zero values override the receiver's values (more-specific wins).
// Returns a new config; neither the receiver nor other is modified.
//
// This implements 3-layer precedence for agent compilation:
//
//	agent definition (base) -> skill override (resolved) -> runtime override (SkillRef.Execution)
//
// Each layer is merged left-to-right: base.Merge(skillOverride).Merge(runtimeOverride).
// For each field, the rightmost non-nil/non-zero value wins.
func (e *Config) Merge(other *Config) *Config {
	if e == nil && other == nil {
		return nil
	}
	if e == nil {
		return other.Clone()
	}
	if other == nil {
		return e.Clone()
	}

	result := e.Clone()

	// Scalar overrides
	if other.Provider != "" {
		result.Provider = other.Provider
	}
	if other.Model != "" {
		result.Model = other.Model
	}

	// Pointer overrides
	result.Temperature = coalesceFloat64Ptr(other.Temperature, result.Temperature)
	result.MaxTokens = coalesceIntPtr(other.MaxTokens, result.MaxTokens)
	result.TopP = coalesceFloat64Ptr(other.TopP, result.TopP)
	result.TopK = coalesceIntPtr(other.TopK, result.TopK)

	if len(other.StopSequences) > 0 {
		result.StopSequences = make([]string, len(other.StopSequences))
		copy(result.StopSequences, other.StopSequences)
	}

	result.MinP = coalesceFloat64Ptr(other.MinP, result.MinP)
	result.RepetitionPenalty = coalesceFloat64Ptr(other.RepetitionPenalty, result.RepetitionPenalty)
	result.Seed = coalesceIntPtr(other.Seed, result.Seed)
	result.Logprobs = coalesceIntPtr(other.Logprobs, result.Logprobs)

	if len(other.StopTokenIDs) > 0 {
		result.StopTokenIDs = make([]int, len(other.StopTokenIDs))
		copy(result.StopTokenIDs, other.StopTokenIDs)
	}
	if len(other.LogitBias) > 0 {
		result.LogitBias = make(map[string]float64, len(other.LogitBias))
		for k, v := range other.LogitBias {
			result.LogitBias[k] = v
		}
	}

	// LLM parameter alignment fields
	result.FrequencyPenalty = coalesceFloat64Ptr(other.FrequencyPenalty, result.FrequencyPenalty)
	result.PresencePenalty = coalesceFloat64Ptr(other.PresencePenalty, result.PresencePenalty)
	result.N = coalesceIntPtr(other.N, result.N)
	result.MaxCompletionTokens = coalesceIntPtr(other.MaxCompletionTokens, result.MaxCompletionTokens)
	if other.ReasoningEffort != "" {
		result.ReasoningEffort = other.ReasoningEffort
	}
	result.TopA = coalesceFloat64Ptr(other.TopA, result.TopA)
	if other.User != "" {
		result.User = other.User
	}
	if other.ServiceTier != "" {
		result.ServiceTier = other.ServiceTier
	}
	result.Store = coalesceBoolPtr(other.Store, result.Store)

	if other.Thinking != nil {
		result.Thinking = &ThinkingConfig{Enabled: other.Thinking.Enabled}
		if other.Thinking.BudgetTokens != nil {
			bt := *other.Thinking.BudgetTokens
			result.Thinking.BudgetTokens = &bt
		}
	}

	if other.ResponseFormat != nil {
		result.ResponseFormat = cloneResponseFormat(other.ResponseFormat)
	}
	if other.GuidedDecoding != nil {
		result.GuidedDecoding = cloneGuidedDecoding(other.GuidedDecoding)
	}

	// Media fields
	if other.Modality != "" {
		result.Modality = other.Modality
	}
	if other.Image != nil {
		result.Image = other.Image.Clone()
	}
	if other.Audio != nil {
		result.Audio = other.Audio.Clone()
	}
	if other.Embedding != nil {
		result.Embedding = other.Embedding.Clone()
	}
	if other.Streaming != nil {
		result.Streaming = other.Streaming.Clone()
	}
	if other.Async != nil {
		result.Async = other.Async.Clone()
	}

	// Merge provider options (other wins on conflict)
	if len(other.ProviderOptions) > 0 {
		if result.ProviderOptions == nil {
			result.ProviderOptions = make(map[string]any, len(other.ProviderOptions))
		}
		for k, v := range other.ProviderOptions {
			result.ProviderOptions[k] = v
		}
	}

	return result
}

// coalesceFloat64Ptr returns the first non-nil pointer.
func coalesceFloat64Ptr(a, b *float64) *float64 {
	if a != nil {
		v := *a
		return &v
	}
	return b
}

// coalesceIntPtr returns the first non-nil pointer.
func coalesceIntPtr(a, b *int) *int {
	if a != nil {
		v := *a
		return &v
	}
	return b
}

// coalesceBoolPtr returns the first non-nil pointer.
func coalesceBoolPtr(a, b *bool) *bool {
	if a != nil {
		v := *a
		return &v
	}
	return b
}
