package execution

// Validate checks the execution config for consistency.
// Returns nil if the config is nil or all fields are within valid ranges.
func (e *Config) Validate() error {
	if e == nil {
		return nil
	}

	// Validate temperature range if set
	if e.Temperature != nil {
		if *e.Temperature < 0.0 || *e.Temperature > 2.0 {
			return NewConfigValidationError(ErrMsgTemperatureOutOfRange)
		}
	}

	// Validate top_p range if set
	if e.TopP != nil {
		if *e.TopP < 0.0 || *e.TopP > 1.0 {
			return NewConfigValidationError(ErrMsgTopPOutOfRange)
		}
	}

	// Validate max_tokens if set
	if e.MaxTokens != nil && *e.MaxTokens <= 0 {
		return NewConfigValidationError(ErrMsgMaxTokensInvalid)
	}

	// Validate top_k if set
	if e.TopK != nil && *e.TopK < 0 {
		return NewConfigValidationError(ErrMsgTopKInvalid)
	}

	// Validate min_p range if set
	if e.MinP != nil {
		if *e.MinP < 0.0 || *e.MinP > 1.0 {
			return NewConfigValidationError(ErrMsgMinPOutOfRange)
		}
	}

	// Validate repetition_penalty if set
	if e.RepetitionPenalty != nil {
		if *e.RepetitionPenalty <= 0.0 {
			return NewConfigValidationError(ErrMsgRepetitionPenaltyOutOfRange)
		}
	}

	// Validate logprobs range if set
	if e.Logprobs != nil {
		if *e.Logprobs < 0 || *e.Logprobs > 20 {
			return NewConfigValidationError(ErrMsgLogprobsOutOfRange)
		}
	}

	// Validate stop_token_ids if set
	for _, id := range e.StopTokenIDs {
		if id < 0 {
			return NewConfigValidationError(ErrMsgStopTokenIDNegative)
		}
	}

	// Validate logit_bias values if set
	for _, v := range e.LogitBias {
		if v < -100.0 || v > 100.0 {
			return NewConfigValidationError(ErrMsgLogitBiasValueOutOfRange)
		}
	}

	// Validate frequency_penalty
	if e.FrequencyPenalty != nil {
		if *e.FrequencyPenalty < -2.0 || *e.FrequencyPenalty > 2.0 {
			return NewConfigValidationError(ErrMsgFrequencyPenaltyOutOfRange)
		}
	}

	// Validate presence_penalty
	if e.PresencePenalty != nil {
		if *e.PresencePenalty < -2.0 || *e.PresencePenalty > 2.0 {
			return NewConfigValidationError(ErrMsgPresencePenaltyOutOfRange)
		}
	}

	// Validate n
	if e.N != nil {
		if *e.N < 1 || *e.N > NMax {
			return NewConfigValidationError(ErrMsgNOutOfRange)
		}
	}

	// Validate max_completion_tokens
	if e.MaxCompletionTokens != nil && *e.MaxCompletionTokens <= 0 {
		return NewConfigValidationError(ErrMsgMaxCompletionTokensInvalid)
	}

	// Validate reasoning_effort enum
	if e.ReasoningEffort != "" && !isValidReasoningEffort(e.ReasoningEffort) {
		return NewConfigValidationError(ErrMsgReasoningEffortInvalid)
	}

	// Validate top_a
	if e.TopA != nil {
		if *e.TopA < 0.0 || *e.TopA > 1.0 {
			return NewConfigValidationError(ErrMsgTopAOutOfRange)
		}
	}

	// Validate service_tier enum
	if e.ServiceTier != "" && !isValidServiceTier(e.ServiceTier) {
		return NewConfigValidationError(ErrMsgServiceTierInvalid)
	}

	// Validate thinking config if set
	if e.Thinking != nil && e.Thinking.Enabled {
		if e.Thinking.BudgetTokens != nil && *e.Thinking.BudgetTokens <= 0 {
			return NewConfigValidationError(ErrMsgThinkingBudgetInvalid)
		}
	}

	// Validate modality if set
	if e.Modality != "" && !isValidModality(e.Modality) {
		return NewConfigValidationError(ErrMsgInvalidModality)
	}

	// Validate media configs
	if e.Image != nil {
		if err := e.Image.Validate(); err != nil {
			return err
		}
	}
	if e.Audio != nil {
		if err := e.Audio.Validate(); err != nil {
			return err
		}
	}
	if e.Embedding != nil {
		if err := e.Embedding.Validate(); err != nil {
			return err
		}
	}
	if e.Streaming != nil {
		if err := e.Streaming.Validate(); err != nil {
			return err
		}
	}
	if e.Async != nil {
		if err := e.Async.Validate(); err != nil {
			return err
		}
	}

	return nil
}
