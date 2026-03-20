package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{"empty strings", "", "", 0},
		{"empty a", "", "hello", 5},
		{"empty b", "hello", "", 5},
		{"identical", "hello", "hello", 0},
		{"one char diff", "hello", "hallo", 1},
		{"completely different", "abc", "xyz", 3},
		{"insertion", "test", "tests", 1},
		{"deletion", "tests", "test", 1},
		{"substitution", "test", "tent", 1},
		{"case sensitive", "Hello", "hello", 1},
		{"longer strings", "username", "userName", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := levenshteinDistance(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindSimilarStrings(t *testing.T) {
	t.Run("finds similar strings", func(t *testing.T) {
		candidates := []string{"name", "names", "named", "game", "fame", "completely_different"}
		result := FindSimilarStrings("nam", candidates, 3)

		assert.Contains(t, result, "name")
		assert.Contains(t, result, "names")
		assert.LessOrEqual(t, len(result), 3)
	})

	t.Run("returns empty for no matches", func(t *testing.T) {
		candidates := []string{"xyz", "abc", "def"}
		result := FindSimilarStrings("username", candidates, 3)

		assert.Empty(t, result)
	})

	t.Run("respects maxSuggestions", func(t *testing.T) {
		candidates := []string{"name", "names", "named", "nam", "namex"}
		result := FindSimilarStrings("name", candidates, 2)

		assert.LessOrEqual(t, len(result), 2)
	})

	t.Run("empty candidates", func(t *testing.T) {
		result := FindSimilarStrings("name", nil, 3)
		assert.Empty(t, result)
	})

	t.Run("zero maxSuggestions", func(t *testing.T) {
		candidates := []string{"name", "names"}
		result := FindSimilarStrings("name", candidates, 0)
		assert.Empty(t, result)
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		candidates := []string{"UserName", "USERNAME", "username"}
		result := FindSimilarStrings("userName", candidates, 3)

		// Should find all three due to case-insensitive matching
		assert.NotEmpty(t, result)
	})

	t.Run("sorts by similarity", func(t *testing.T) {
		candidates := []string{"names", "nam", "name", "namex", "namexyz"}
		result := FindSimilarStrings("name", candidates, 5)

		// "name" should be first (distance 0)
		if len(result) > 0 {
			assert.Equal(t, "name", result[0])
		}
	})
}

func TestFormatSuggestions(t *testing.T) {
	tests := []struct {
		name        string
		suggestions []string
		expected    string
	}{
		{"empty", nil, ""},
		{"one suggestion", []string{"name"}, ". Did you mean 'name'?"},
		{"two suggestions", []string{"name", "names"}, ". Did you mean 'name' or 'names'?"},
		{"three suggestions", []string{"name", "names", "named"}, ". Did you mean 'name', 'names' or 'named'?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSuggestions(tt.suggestions)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractPathPrefix(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"simple name", "name", "name"},
		{"dot path", "user.name", "user"},
		{"deep path", "user.profile.name", "user"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPathPrefix(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, c  int
		expected int
	}{
		{1, 2, 3, 1},
		{3, 2, 1, 1},
		{2, 1, 3, 1},
		{5, 5, 5, 5},
		{0, 1, 2, 0},
		{-1, 0, 1, -1},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b, tt.c)
		assert.Equal(t, tt.expected, result)
	}
}

func TestFormatAvailableKeys(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		maxKeys  int
		expected string
	}{
		{"empty keys", nil, 5, ""},
		{"empty slice", []string{}, 5, ""},
		{"one key", []string{"name"}, 5, ". Available keys: 'name'"},
		{"two keys", []string{"name", "email"}, 5, ". Available keys: 'name', 'email'"},
		{"three keys", []string{"name", "email", "age"}, 5, ". Available keys: 'name', 'email', 'age'"},
		{"five keys", []string{"a", "b", "c", "d", "e"}, 5, ". Available keys: 'a', 'b', 'c', 'd', 'e'"},
		{"more than max", []string{"a", "b", "c", "d", "e", "f"}, 5, ". Available keys: 'a', 'b', 'c', 'd', 'e' (1 more)"},
		{"many more than max", []string{"a", "b", "c", "d", "e", "f", "g", "h"}, 3, ". Available keys: 'a', 'b', 'c' (5 more)"},
		{"zero maxKeys defaults to 5", []string{"a", "b", "c", "d", "e", "f"}, 0, ". Available keys: 'a', 'b', 'c', 'd', 'e' (1 more)"},
		{"negative maxKeys defaults to 5", []string{"a", "b", "c"}, -1, ". Available keys: 'a', 'b', 'c'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatAvailableKeys(tt.keys, tt.maxKeys)
			assert.Equal(t, tt.expected, result)
		})
	}
}
