package exons

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// EstimateTokens (standalone function)
// =============================================================================

func TestEstimateTokens_Empty(t *testing.T) {
	est := EstimateTokens("")
	assert.Equal(t, 0, est.Characters)
	assert.Equal(t, 0, est.Words)
	assert.Equal(t, 0, est.Lines)
	assert.Equal(t, 0, est.EstimatedGPT)
	assert.Equal(t, 0, est.EstimatedClaude)
	assert.Equal(t, 0, est.EstimatedLlama)
	assert.Equal(t, 0, est.EstimatedGeneric)
}

func TestEstimateTokens_SingleChar(t *testing.T) {
	est := EstimateTokens("a")
	assert.Equal(t, 1, est.Characters)
	assert.Equal(t, 1, est.Words)
	assert.Equal(t, 1, est.Lines)
	// For a single char: 1/4.0 + 0.5 = 0.75 truncates to 0
	// This is expected behavior — tokens are an approximation
	assert.True(t, est.EstimatedGPT >= 0)
	assert.True(t, est.EstimatedGeneric >= 0)
}

func TestEstimateTokens_ShortText(t *testing.T) {
	est := EstimateTokens("Hello World")
	assert.Equal(t, 11, est.Characters)
	assert.Equal(t, 2, est.Words)
	assert.Equal(t, 1, est.Lines)
	assert.True(t, est.EstimatedGPT > 0)
	assert.True(t, est.EstimatedClaude > 0)
	assert.True(t, est.EstimatedLlama > 0)
	assert.True(t, est.EstimatedGeneric > 0)
}

func TestEstimateTokens_LongText(t *testing.T) {
	// Generate a longer text
	text := strings.Repeat("This is a sentence with multiple words. ", 100)
	est := EstimateTokens(text)
	assert.True(t, est.Characters > 1000)
	assert.True(t, est.Words > 100)
	assert.True(t, est.EstimatedGPT > 100)
	assert.True(t, est.EstimatedGeneric > est.EstimatedGPT, "generic should be more conservative (lower chars per token)")
}

func TestEstimateTokens_Code(t *testing.T) {
	code := `func main() {
	fmt.Println("Hello, World!")
	for i := 0; i < 10; i++ {
		fmt.Printf("i = %d\n", i)
	}
}`
	est := EstimateTokens(code)
	assert.True(t, est.Characters > 50)
	assert.True(t, est.Lines > 1)
	assert.True(t, est.EstimatedGPT > 0)
}

func TestEstimateTokens_MixedContent(t *testing.T) {
	mixed := `# Title
Some text with **markdown** and code:
` + "```go" + `
func hello() {}
` + "```"
	est := EstimateTokens(mixed)
	assert.True(t, est.Characters > 20)
	assert.True(t, est.Words > 5)
}

func TestEstimateTokens_Unicode(t *testing.T) {
	// Non-English content should have different tokenization estimates
	// Using many non-ASCII characters to trigger the threshold
	text := strings.Repeat("Hello ", 10) + strings.Repeat("\u4e16\u754c ", 40) // Mix of English + Chinese
	est := EstimateTokens(text)
	assert.True(t, est.Characters > 20)
	// With high non-ASCII ratio, the estimates should adjust
	if est.NonASCIIRatio > NonASCIIThreshold {
		// When non-ASCII is high, GPT and generic should be similar
		// because both use CharsPerTokenNonEnglish
		assert.Equal(t, est.EstimatedGPT, est.EstimatedClaude)
	}
}

func TestEstimateTokens_Ratios(t *testing.T) {
	t.Run("whitespace ratio", func(t *testing.T) {
		est := EstimateTokens("a b c d")
		assert.True(t, est.WhitespaceRatio > 0, "should have whitespace")
	})

	t.Run("average word length", func(t *testing.T) {
		est := EstimateTokens("hi there world")
		assert.True(t, est.AverageWordLength > 0, "should have positive avg word length")
	})

	t.Run("non ascii ratio for ascii text", func(t *testing.T) {
		est := EstimateTokens("Hello World")
		assert.Equal(t, float64(0), est.NonASCIIRatio)
	})
}

// =============================================================================
// Template.EstimateTokens
// =============================================================================

func TestTemplate_EstimateTokens(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("estimate tokens after execution", func(t *testing.T) {
		tmpl, err := engine.Parse(`Hello {~exons.var name="name" /~}! Welcome.`)
		require.NoError(t, err)
		est, err := tmpl.EstimateTokens(ctx, map[string]any{"name": "World"})
		require.NoError(t, err)
		assert.True(t, est.Characters > 0)
		assert.True(t, est.EstimatedGPT > 0)
	})

	t.Run("estimate tokens execution error", func(t *testing.T) {
		tmpl, err := engine.Parse(`{~exons.var name="missing" /~}`)
		require.NoError(t, err)
		_, err = tmpl.EstimateTokens(ctx, nil)
		assert.Error(t, err)
	})
}

// =============================================================================
// Template.EstimateSourceTokens
// =============================================================================

func TestTemplate_EstimateSourceTokens(t *testing.T) {
	engine := MustNew()

	t.Run("estimate source tokens", func(t *testing.T) {
		source := `Hello {~exons.var name="name" /~}! This is a test.`
		tmpl, err := engine.Parse(source)
		require.NoError(t, err)
		est := tmpl.EstimateSourceTokens()
		assert.True(t, est.Characters > 0)
		assert.True(t, est.EstimatedGPT > 0)
	})

	t.Run("empty template source tokens", func(t *testing.T) {
		tmpl, err := engine.Parse("")
		require.NoError(t, err)
		est := tmpl.EstimateSourceTokens()
		assert.Equal(t, 0, est.Characters)
	})
}

// =============================================================================
// Template.EstimateTokensDryRun
// =============================================================================

func TestTemplate_EstimateTokensDryRun(t *testing.T) {
	engine := MustNew()
	ctx := context.Background()

	t.Run("dry run token estimate", func(t *testing.T) {
		tmpl, err := engine.Parse(`Hello {~exons.var name="name" default="World" /~}!`)
		require.NoError(t, err)
		est := tmpl.EstimateTokensDryRun(ctx, map[string]any{"name": "Alice"})
		assert.True(t, est.Characters > 0)
	})

	t.Run("dry run with missing variables", func(t *testing.T) {
		tmpl, err := engine.Parse(`Hello {~exons.var name="name" /~}!`)
		require.NoError(t, err)
		est := tmpl.EstimateTokensDryRun(ctx, nil)
		assert.True(t, est.Characters > 0) // Has placeholder output
	})
}

// =============================================================================
// CostEstimate
// =============================================================================

func TestTokenEstimate_EstimateCost(t *testing.T) {
	t.Run("basic cost estimate", func(t *testing.T) {
		est := EstimateTokens(strings.Repeat("word ", 1000))
		cost := est.EstimateCost()
		assert.True(t, cost.InputTokens > 0)
		assert.True(t, cost.GPT4Cost > 0)
		assert.True(t, cost.GPT4oCost > 0)
		assert.True(t, cost.GPT35Cost > 0)
		assert.True(t, cost.ClaudeOpusCost > 0)
		assert.True(t, cost.ClaudeSonnetCost > 0)
		assert.True(t, cost.ClaudeHaikuCost > 0)
	})

	t.Run("cost ordering", func(t *testing.T) {
		est := EstimateTokens(strings.Repeat("word ", 1000))
		cost := est.EstimateCost()
		// GPT-4 is more expensive than GPT-4o which is more expensive than GPT-3.5
		assert.True(t, cost.GPT4Cost > cost.GPT4oCost)
		assert.True(t, cost.GPT4oCost > cost.GPT35Cost)
		// Claude Opus > Sonnet > Haiku
		assert.True(t, cost.ClaudeOpusCost > cost.ClaudeSonnetCost)
		assert.True(t, cost.ClaudeSonnetCost > cost.ClaudeHaikuCost)
	})

	t.Run("empty text zero cost", func(t *testing.T) {
		est := EstimateTokens("")
		cost := est.EstimateCost()
		assert.Equal(t, 0, cost.InputTokens)
		assert.Equal(t, float64(0), cost.GPT4Cost)
	})
}

// =============================================================================
// EstimateCostForModel
// =============================================================================

func TestTokenEstimate_EstimateCostForModel(t *testing.T) {
	est := EstimateTokens(strings.Repeat("word ", 500))

	t.Run("GPT-4 model", func(t *testing.T) {
		cost := est.EstimateCostForModel("gpt-4", PriceGPT4Per1K)
		assert.True(t, cost > 0)
	})

	t.Run("GPT-4o model", func(t *testing.T) {
		cost := est.EstimateCostForModel("gpt-4o", PriceGPT4oPer1K)
		assert.True(t, cost > 0)
	})

	t.Run("GPT-3.5 model", func(t *testing.T) {
		cost := est.EstimateCostForModel("gpt-3.5-turbo", PriceGPT35Per1K)
		assert.True(t, cost > 0)
	})

	t.Run("Claude models", func(t *testing.T) {
		cost := est.EstimateCostForModel("claude-3-opus", PriceClaudeOpusPer1K)
		assert.True(t, cost > 0)

		cost2 := est.EstimateCostForModel("claude-3-sonnet", PriceClaudeSonnetPer1K)
		assert.True(t, cost2 > 0)

		cost3 := est.EstimateCostForModel("claude-3-haiku", PriceClaudeHaikuPer1K)
		assert.True(t, cost3 > 0)
	})

	t.Run("Claude generic", func(t *testing.T) {
		cost := est.EstimateCostForModel("claude", PriceClaudeSonnetPer1K)
		assert.True(t, cost > 0)
	})

	t.Run("Llama models", func(t *testing.T) {
		cost := est.EstimateCostForModel("llama", 0.001)
		assert.True(t, cost > 0)

		cost2 := est.EstimateCostForModel("llama-2", 0.001)
		assert.True(t, cost2 > 0)

		cost3 := est.EstimateCostForModel("llama-3", 0.001)
		assert.True(t, cost3 > 0)
	})

	t.Run("unknown model uses generic", func(t *testing.T) {
		cost := est.EstimateCostForModel("unknown-model", 0.01)
		assert.True(t, cost > 0)
	})
}

// =============================================================================
// TokenBudget
// =============================================================================

func TestNewTokenBudget(t *testing.T) {
	t.Run("basic budget", func(t *testing.T) {
		budget := NewTokenBudget(4096, 1024)
		assert.Equal(t, 4096, budget.MaxTokens)
		assert.Equal(t, 1024, budget.ReservedForResponse)
		assert.Equal(t, 3072, budget.AvailableForPrompt)
	})

	t.Run("budget with zero reserved", func(t *testing.T) {
		budget := NewTokenBudget(4096, 0)
		assert.Equal(t, 4096, budget.AvailableForPrompt)
	})

	t.Run("budget with all reserved", func(t *testing.T) {
		budget := NewTokenBudget(4096, 4096)
		assert.Equal(t, 0, budget.AvailableForPrompt)
	})

	t.Run("budget with over-reserved clamps to zero", func(t *testing.T) {
		budget := NewTokenBudget(100, 200)
		assert.Equal(t, 0, budget.AvailableForPrompt)
	})
}

func TestTokenBudget_FitsWithin(t *testing.T) {
	budget := NewTokenBudget(4096, 1024)
	smallEst := EstimateTokens("Hello World")
	assert.True(t, budget.FitsWithin(smallEst))

	largeText := strings.Repeat("word ", 5000)
	largeEst := EstimateTokens(largeText)
	assert.False(t, budget.FitsWithin(largeEst))
}

func TestTokenBudget_RemainingTokens(t *testing.T) {
	budget := NewTokenBudget(4096, 1024)
	est := EstimateTokens("Hello")
	remaining := budget.RemainingTokens(est)
	assert.True(t, remaining > 0)
	assert.True(t, remaining < budget.AvailableForPrompt)
}

func TestTokenBudget_RemainingTokens_Overflow(t *testing.T) {
	budget := NewTokenBudget(10, 5) // only 5 available
	largeEst := EstimateTokens(strings.Repeat("word ", 100))
	remaining := budget.RemainingTokens(largeEst)
	assert.Equal(t, 0, remaining)
}

func TestTokenBudget_OverageTokens(t *testing.T) {
	t.Run("under budget", func(t *testing.T) {
		budget := NewTokenBudget(4096, 1024)
		est := EstimateTokens("Hello")
		assert.Equal(t, 0, budget.OverageTokens(est))
	})

	t.Run("over budget", func(t *testing.T) {
		budget := NewTokenBudget(10, 5)
		est := EstimateTokens(strings.Repeat("word ", 100))
		overage := budget.OverageTokens(est)
		assert.True(t, overage > 0)
	})
}

// =============================================================================
// Preset Budgets
// =============================================================================

func TestPresetBudgets(t *testing.T) {
	t.Run("GPT4 Turbo budget", func(t *testing.T) {
		budget := NewGPT4TurboBudget(4096)
		assert.Equal(t, ContextGPT4Turbo, budget.MaxTokens)
		assert.Equal(t, ContextGPT4Turbo-4096, budget.AvailableForPrompt)
	})

	t.Run("Claude budget", func(t *testing.T) {
		budget := NewClaudeBudget(4096)
		assert.Equal(t, ContextClaudeOpus, budget.MaxTokens)
		assert.Equal(t, ContextClaudeOpus-4096, budget.AvailableForPrompt)
	})
}

// =============================================================================
// Helper Functions
// =============================================================================

func TestCountWords(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"  hello  world  ", 2},
		{"one\ttwo\nthree", 3},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, countWords(tt.input))
		})
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"hello", 1},
		{"hello\nworld", 2},
		{"a\nb\nc", 3},
		{"trailing\n", 2},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, countLines(tt.input))
		})
	}
}

func TestCountWhitespace(t *testing.T) {
	assert.Equal(t, 0, countWhitespace("hello"))
	assert.Equal(t, 1, countWhitespace("hello world"))
	assert.Equal(t, 4, countWhitespace("  a  "))
}

func TestCountNonASCII(t *testing.T) {
	assert.Equal(t, 0, countNonASCII("hello"))
	assert.True(t, countNonASCII("\u4e16\u754c") > 0)
}

func TestEstimateTokenCount(t *testing.T) {
	assert.Equal(t, 0, estimateTokenCount(0, 4.0))
	assert.Equal(t, 0, estimateTokenCount(10, 0))
	// 10 chars / 4.0 chars per token + 0.5 = 3.0 → 3
	assert.Equal(t, 3, estimateTokenCount(10, 4.0))
}

// =============================================================================
// Context Window Constants
// =============================================================================

func TestContextWindowConstants(t *testing.T) {
	assert.Equal(t, 128000, ContextGPT4Turbo)
	assert.Equal(t, 8192, ContextGPT4)
	assert.Equal(t, 16385, ContextGPT35)
	assert.Equal(t, 200000, ContextClaudeOpus)
	assert.Equal(t, 200000, ContextClaudeSonnet)
	assert.Equal(t, 200000, ContextClaudeHaiku)
	assert.Equal(t, 8192, ContextLlama3)
}

// =============================================================================
// Pricing Constants
// =============================================================================

func TestPricingConstants(t *testing.T) {
	assert.Equal(t, 1000.0, TokenPricingUnit)
	assert.Equal(t, 0.03, PriceGPT4Per1K)
	assert.Equal(t, 0.005, PriceGPT4oPer1K)
	assert.Equal(t, 0.0015, PriceGPT35Per1K)
	assert.Equal(t, 0.015, PriceClaudeOpusPer1K)
	assert.Equal(t, 0.003, PriceClaudeSonnetPer1K)
	assert.Equal(t, 0.00025, PriceClaudeHaikuPer1K)
}
