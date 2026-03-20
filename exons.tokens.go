package exons

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TokenEstimate provides estimated token counts for various LLM providers.
// These are approximations based on typical tokenizer behavior and should be
// used for cost estimation and planning, not as exact values.
type TokenEstimate struct {
	// Characters is the raw character count (UTF-8 runes)
	Characters int

	// Words is the word count (whitespace-separated)
	Words int

	// Lines is the line count
	Lines int

	// EstimatedGPT is the estimated token count for GPT models (GPT-3.5, GPT-4)
	// Based on ~4 characters per token for English text
	EstimatedGPT int

	// EstimatedClaude is the estimated token count for Claude models
	// Based on ~4 characters per token for English text
	EstimatedClaude int

	// EstimatedLlama is the estimated token count for Llama models
	// Based on ~3.5 characters per token
	EstimatedLlama int

	// EstimatedGeneric is a conservative generic estimate
	// Uses ~3 characters per token for safety margin
	EstimatedGeneric int

	// WhitespaceRatio is the ratio of whitespace to total characters
	WhitespaceRatio float64

	// AverageWordLength is the average word length in characters
	AverageWordLength float64

	// NonASCIIRatio is the ratio of non-ASCII characters.
	// Higher values suggest non-English content which typically has different tokenization.
	NonASCIIRatio float64
}

// Token estimation constants — average characters per token for each model family.
const (
	// CharsPerTokenGPT is the average characters per token for GPT models
	CharsPerTokenGPT = 4.0

	// CharsPerTokenClaude is the average characters per token for Claude models
	CharsPerTokenClaude = 4.0

	// CharsPerTokenLlama is the average characters per token for Llama models
	CharsPerTokenLlama = 3.5

	// CharsPerTokenGeneric is a conservative estimate for unknown models
	CharsPerTokenGeneric = 3.0

	// CharsPerTokenNonEnglish is the average for non-English content.
	// Non-English typically uses more tokens per character.
	CharsPerTokenNonEnglish = 2.0
)

// NonASCIIThreshold is the ratio above which content is considered non-English
// for tokenization estimation purposes.
const NonASCIIThreshold = 0.3

// TokenRoundingOffset is added before truncation to round up for safety.
const TokenRoundingOffset = 0.5

// EstimateTokens estimates token counts for a given text string.
// This is useful for cost estimation before sending prompts to LLM providers.
func EstimateTokens(text string) *TokenEstimate {
	if text == "" {
		return &TokenEstimate{}
	}

	// Count basic metrics
	charCount := utf8.RuneCountInString(text)
	words := countWords(text)
	lines := countLines(text)
	whitespace := countWhitespace(text)
	nonASCII := countNonASCII(text)

	// Calculate ratios
	whitespaceRatio := float64(whitespace) / float64(charCount)
	nonASCIIRatio := float64(nonASCII) / float64(charCount)
	avgWordLen := 0.0
	if words > 0 {
		avgWordLen = float64(charCount-whitespace) / float64(words)
	}

	// Adjust token estimate based on content type
	// Non-English content typically uses more tokens
	gptMultiplier := CharsPerTokenGPT
	claudeMultiplier := CharsPerTokenClaude
	llamaMultiplier := CharsPerTokenLlama
	genericMultiplier := CharsPerTokenGeneric

	if nonASCIIRatio > NonASCIIThreshold {
		// Significant non-ASCII content - adjust for non-English tokenization
		gptMultiplier = CharsPerTokenNonEnglish
		claudeMultiplier = CharsPerTokenNonEnglish
		llamaMultiplier = CharsPerTokenNonEnglish
		genericMultiplier = CharsPerTokenNonEnglish
	}

	return &TokenEstimate{
		Characters:        charCount,
		Words:             words,
		Lines:             lines,
		EstimatedGPT:      estimateTokenCount(charCount, gptMultiplier),
		EstimatedClaude:   estimateTokenCount(charCount, claudeMultiplier),
		EstimatedLlama:    estimateTokenCount(charCount, llamaMultiplier),
		EstimatedGeneric:  estimateTokenCount(charCount, genericMultiplier),
		WhitespaceRatio:   whitespaceRatio,
		AverageWordLength: avgWordLen,
		NonASCIIRatio:     nonASCIIRatio,
	}
}

// EstimateTokens estimates token count for a template after execution.
// This executes the template and estimates tokens on the rendered output.
func (t *Template) EstimateTokens(ctx context.Context, data map[string]any) (*TokenEstimate, error) {
	result, err := t.Execute(ctx, data)
	if err != nil {
		return nil, err
	}
	return EstimateTokens(result), nil
}

// EstimateTokensDryRun estimates token counts without full execution.
// Uses dry-run output which includes placeholders for dynamic content.
// Useful for getting a rough estimate without executing resolvers.
func (t *Template) EstimateTokensDryRun(ctx context.Context, data map[string]any) *TokenEstimate {
	result := t.DryRun(ctx, data)
	return EstimateTokens(result.Output)
}

// EstimateSourceTokens estimates token count for the raw template source.
// Useful for understanding template complexity before execution.
func (t *Template) EstimateSourceTokens() *TokenEstimate {
	return EstimateTokens(t.Source())
}

// CostEstimate provides estimated costs for various LLM providers.
type CostEstimate struct {
	// InputTokens is the estimated input token count
	InputTokens int

	// GPT4Cost is estimated cost in USD for GPT-4 input
	// Based on $0.03 per 1K tokens (as of 2024)
	GPT4Cost float64

	// GPT4oCost is estimated cost in USD for GPT-4o input
	// Based on $0.005 per 1K tokens (as of 2024)
	GPT4oCost float64

	// GPT35Cost is estimated cost in USD for GPT-3.5-turbo input
	// Based on $0.0015 per 1K tokens (as of 2024)
	GPT35Cost float64

	// ClaudeOpusCost is estimated cost in USD for Claude Opus input
	// Based on $0.015 per 1K tokens (as of 2024)
	ClaudeOpusCost float64

	// ClaudeSonnetCost is estimated cost in USD for Claude Sonnet input
	// Based on $0.003 per 1K tokens (as of 2024)
	ClaudeSonnetCost float64

	// ClaudeHaikuCost is estimated cost in USD for Claude Haiku input
	// Based on $0.00025 per 1K tokens (as of 2024)
	ClaudeHaikuCost float64
}

// Pricing constants (per 1K tokens, input only, as of late 2024).
const (
	TokenPricingUnit       = 1000.0 // tokens per pricing unit (1K)
	PriceGPT4Per1K         = 0.03
	PriceGPT4oPer1K        = 0.005
	PriceGPT35Per1K        = 0.0015
	PriceClaudeOpusPer1K   = 0.015
	PriceClaudeSonnetPer1K = 0.003
	PriceClaudeHaikuPer1K  = 0.00025
)

// EstimateCost calculates estimated costs for various LLM providers.
func (e *TokenEstimate) EstimateCost() *CostEstimate {
	inputTokens := e.EstimatedGeneric // Use conservative estimate

	return &CostEstimate{
		InputTokens:      inputTokens,
		GPT4Cost:         float64(inputTokens) / TokenPricingUnit * PriceGPT4Per1K,
		GPT4oCost:        float64(inputTokens) / TokenPricingUnit * PriceGPT4oPer1K,
		GPT35Cost:        float64(inputTokens) / TokenPricingUnit * PriceGPT35Per1K,
		ClaudeOpusCost:   float64(inputTokens) / TokenPricingUnit * PriceClaudeOpusPer1K,
		ClaudeSonnetCost: float64(inputTokens) / TokenPricingUnit * PriceClaudeSonnetPer1K,
		ClaudeHaikuCost:  float64(inputTokens) / TokenPricingUnit * PriceClaudeHaikuPer1K,
	}
}

// Model name constants for cost estimation.
const (
	modelGPT4       = "gpt-4"
	modelGPT4Turbo  = "gpt-4-turbo"
	modelGPT4o      = "gpt-4o"
	modelGPT4oMini  = "gpt-4o-mini"
	modelGPT35Turbo = "gpt-3.5-turbo"
	modelGPT35      = "gpt-3.5"
	modelClaudeOpus = "claude-3-opus"
	modelClaudeSon  = "claude-3-sonnet"
	modelClaudeHaik = "claude-3-haiku"
	modelClaude     = "claude"
	modelLlama      = "llama"
	modelLlama2     = "llama-2"
	modelLlama3     = "llama-3"
)

// EstimateCostForModel returns estimated cost for a specific model.
func (e *TokenEstimate) EstimateCostForModel(model string, pricePerThousand float64) float64 {
	var tokens int
	switch strings.ToLower(model) {
	case modelGPT4, modelGPT4Turbo, modelGPT4o, modelGPT4oMini:
		tokens = e.EstimatedGPT
	case modelGPT35Turbo, modelGPT35:
		tokens = e.EstimatedGPT
	case modelClaudeOpus, modelClaudeSon, modelClaudeHaik, modelClaude:
		tokens = e.EstimatedClaude
	case modelLlama, modelLlama2, modelLlama3:
		tokens = e.EstimatedLlama
	default:
		tokens = e.EstimatedGeneric
	}
	return float64(tokens) / TokenPricingUnit * pricePerThousand
}

// Helper functions

func estimateTokenCount(charCount int, charsPerToken float64) int {
	if charCount == 0 || charsPerToken == 0 {
		return 0
	}
	estimate := float64(charCount) / charsPerToken
	// Round up for safety
	return int(estimate + TokenRoundingOffset)
}

func countWords(text string) int {
	words := 0
	inWord := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			if inWord {
				words++
				inWord = false
			}
		} else {
			inWord = true
		}
	}
	if inWord {
		words++
	}
	return words
}

func countLines(text string) int {
	if text == "" {
		return 0
	}
	lines := 1
	for _, r := range text {
		if r == '\n' {
			lines++
		}
	}
	return lines
}

func countWhitespace(text string) int {
	count := 0
	for _, r := range text {
		if unicode.IsSpace(r) {
			count++
		}
	}
	return count
}

// NonASCIICodePoint is the threshold above which a rune is considered non-ASCII.
const NonASCIICodePoint = 127

func countNonASCII(text string) int {
	count := 0
	for _, r := range text {
		if r > NonASCIICodePoint {
			count++
		}
	}
	return count
}

// TokenBudget helps manage token limits for prompts.
type TokenBudget struct {
	// MaxTokens is the maximum tokens allowed
	MaxTokens int

	// ReservedForResponse is tokens reserved for model response
	ReservedForResponse int

	// AvailableForPrompt is tokens available for the prompt
	AvailableForPrompt int
}

// NewTokenBudget creates a new token budget calculator.
func NewTokenBudget(maxTokens, reservedForResponse int) *TokenBudget {
	available := maxTokens - reservedForResponse
	if available < 0 {
		available = 0
	}
	return &TokenBudget{
		MaxTokens:           maxTokens,
		ReservedForResponse: reservedForResponse,
		AvailableForPrompt:  available,
	}
}

// Common context window sizes.
const (
	ContextGPT4Turbo    = 128000
	ContextGPT4         = 8192
	ContextGPT35        = 16385
	ContextClaudeOpus   = 200000
	ContextClaudeSonnet = 200000
	ContextClaudeHaiku  = 200000
	ContextLlama3       = 8192
)

// NewGPT4TurboBudget creates a budget for GPT-4 Turbo.
func NewGPT4TurboBudget(reservedForResponse int) *TokenBudget {
	return NewTokenBudget(ContextGPT4Turbo, reservedForResponse)
}

// NewClaudeBudget creates a budget for Claude models.
func NewClaudeBudget(reservedForResponse int) *TokenBudget {
	return NewTokenBudget(ContextClaudeOpus, reservedForResponse)
}

// FitsWithin checks if the estimated tokens fit within the budget.
func (b *TokenBudget) FitsWithin(estimate *TokenEstimate) bool {
	return estimate.EstimatedGeneric <= b.AvailableForPrompt
}

// RemainingTokens returns how many tokens remain after the estimate.
func (b *TokenBudget) RemainingTokens(estimate *TokenEstimate) int {
	remaining := b.AvailableForPrompt - estimate.EstimatedGeneric
	if remaining < 0 {
		return 0
	}
	return remaining
}

// OverageTokens returns how many tokens over budget, or 0 if within budget.
func (b *TokenBudget) OverageTokens(estimate *TokenEstimate) int {
	overage := estimate.EstimatedGeneric - b.AvailableForPrompt
	if overage < 0 {
		return 0
	}
	return overage
}
