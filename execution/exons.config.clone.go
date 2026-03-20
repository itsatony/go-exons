package execution

// Clone creates a deep copy of the execution config.
// Returns nil if the receiver is nil.
func (e *Config) Clone() *Config {
	if e == nil {
		return nil
	}

	clone := &Config{
		Provider: e.Provider,
		Model:    e.Model,
	}

	if e.Temperature != nil {
		t := *e.Temperature
		clone.Temperature = &t
	}
	if e.MaxTokens != nil {
		m := *e.MaxTokens
		clone.MaxTokens = &m
	}
	if e.TopP != nil {
		tp := *e.TopP
		clone.TopP = &tp
	}
	if e.TopK != nil {
		tk := *e.TopK
		clone.TopK = &tk
	}
	if e.StopSequences != nil {
		clone.StopSequences = make([]string, len(e.StopSequences))
		copy(clone.StopSequences, e.StopSequences)
	}

	if e.MinP != nil {
		v := *e.MinP
		clone.MinP = &v
	}
	if e.RepetitionPenalty != nil {
		v := *e.RepetitionPenalty
		clone.RepetitionPenalty = &v
	}
	if e.Seed != nil {
		v := *e.Seed
		clone.Seed = &v
	}
	if e.Logprobs != nil {
		v := *e.Logprobs
		clone.Logprobs = &v
	}
	if e.StopTokenIDs != nil {
		clone.StopTokenIDs = make([]int, len(e.StopTokenIDs))
		copy(clone.StopTokenIDs, e.StopTokenIDs)
	}
	if e.LogitBias != nil {
		clone.LogitBias = make(map[string]float64, len(e.LogitBias))
		for k, v := range e.LogitBias {
			clone.LogitBias[k] = v
		}
	}

	// LLM parameter alignment fields
	if e.FrequencyPenalty != nil {
		v := *e.FrequencyPenalty
		clone.FrequencyPenalty = &v
	}
	if e.PresencePenalty != nil {
		v := *e.PresencePenalty
		clone.PresencePenalty = &v
	}
	if e.N != nil {
		v := *e.N
		clone.N = &v
	}
	if e.MaxCompletionTokens != nil {
		v := *e.MaxCompletionTokens
		clone.MaxCompletionTokens = &v
	}
	clone.ReasoningEffort = e.ReasoningEffort
	if e.TopA != nil {
		v := *e.TopA
		clone.TopA = &v
	}
	clone.User = e.User
	clone.ServiceTier = e.ServiceTier
	if e.Store != nil {
		v := *e.Store
		clone.Store = &v
	}

	if e.Thinking != nil {
		clone.Thinking = &ThinkingConfig{
			Enabled: e.Thinking.Enabled,
		}
		if e.Thinking.BudgetTokens != nil {
			bt := *e.Thinking.BudgetTokens
			clone.Thinking.BudgetTokens = &bt
		}
	}

	if e.ResponseFormat != nil {
		clone.ResponseFormat = cloneResponseFormat(e.ResponseFormat)
	}
	if e.GuidedDecoding != nil {
		clone.GuidedDecoding = cloneGuidedDecoding(e.GuidedDecoding)
	}

	// Media fields
	clone.Modality = e.Modality
	if e.Image != nil {
		clone.Image = e.Image.Clone()
	}
	if e.Audio != nil {
		clone.Audio = e.Audio.Clone()
	}
	if e.Embedding != nil {
		clone.Embedding = e.Embedding.Clone()
	}
	if e.Streaming != nil {
		clone.Streaming = e.Streaming.Clone()
	}
	if e.Async != nil {
		clone.Async = e.Async.Clone()
	}

	if e.ProviderOptions != nil {
		clone.ProviderOptions = make(map[string]any, len(e.ProviderOptions))
		for k, v := range e.ProviderOptions {
			clone.ProviderOptions[k] = v
		}
	}

	return clone
}

// cloneResponseFormat creates a deep copy of ResponseFormat.
func cloneResponseFormat(rf *ResponseFormat) *ResponseFormat {
	if rf == nil {
		return nil
	}
	clone := &ResponseFormat{
		Type: rf.Type,
	}
	if rf.JSONSchema != nil {
		clone.JSONSchema = &JSONSchemaSpec{
			Name:        rf.JSONSchema.Name,
			Description: rf.JSONSchema.Description,
			Strict:      rf.JSONSchema.Strict,
		}
		if rf.JSONSchema.Schema != nil {
			clone.JSONSchema.Schema = copySchema(rf.JSONSchema.Schema)
		}
		if rf.JSONSchema.AdditionalProperties != nil {
			ap := *rf.JSONSchema.AdditionalProperties
			clone.JSONSchema.AdditionalProperties = &ap
		}
		if rf.JSONSchema.PropertyOrdering != nil {
			clone.JSONSchema.PropertyOrdering = make([]string, len(rf.JSONSchema.PropertyOrdering))
			copy(clone.JSONSchema.PropertyOrdering, rf.JSONSchema.PropertyOrdering)
		}
	}
	if rf.Enum != nil {
		clone.Enum = &EnumConstraint{
			Description: rf.Enum.Description,
		}
		if rf.Enum.Values != nil {
			clone.Enum.Values = make([]string, len(rf.Enum.Values))
			copy(clone.Enum.Values, rf.Enum.Values)
		}
	}
	return clone
}

// cloneGuidedDecoding creates a deep copy of GuidedDecoding.
func cloneGuidedDecoding(gd *GuidedDecoding) *GuidedDecoding {
	if gd == nil {
		return nil
	}
	clone := &GuidedDecoding{
		Backend:           gd.Backend,
		Regex:             gd.Regex,
		Grammar:           gd.Grammar,
		WhitespacePattern: gd.WhitespacePattern,
	}
	if gd.JSON != nil {
		clone.JSON = copySchema(gd.JSON)
	}
	if gd.Choice != nil {
		clone.Choice = make([]string, len(gd.Choice))
		copy(clone.Choice, gd.Choice)
	}
	return clone
}
