package internal

import "strings"

// KeyLister is an optional interface that ContextAccessor implementations
// can implement to support "did you mean?" suggestions.
type KeyLister interface {
	// Keys returns a list of all top-level keys in the context.
	Keys() []string
}

// FindSimilarStrings finds strings from candidates that are similar to target.
// Returns up to maxSuggestions suggestions, sorted by similarity (closest first).
// Uses Levenshtein distance with a maximum distance threshold.
func FindSimilarStrings(target string, candidates []string, maxSuggestions int) []string {
	if len(candidates) == 0 || maxSuggestions <= 0 {
		return nil
	}

	// Maximum distance to consider a candidate as similar
	maxDistance := len(target) / 2
	if maxDistance < 2 {
		maxDistance = 2
	}

	type scored struct {
		str      string
		distance int
	}

	var similar []scored
	targetLower := strings.ToLower(target)

	for _, candidate := range candidates {
		// Calculate Levenshtein distance
		dist := levenshteinDistance(targetLower, strings.ToLower(candidate))

		// Only consider if within threshold
		if dist <= maxDistance {
			similar = append(similar, scored{str: candidate, distance: dist})
		}
	}

	// Sort by distance (ascending)
	for i := 0; i < len(similar)-1; i++ {
		for j := i + 1; j < len(similar); j++ {
			if similar[j].distance < similar[i].distance {
				similar[i], similar[j] = similar[j], similar[i]
			}
		}
	}

	// Return up to maxSuggestions
	result := make([]string, 0, maxSuggestions)
	for i := 0; i < len(similar) && i < maxSuggestions; i++ {
		result = append(result, similar[i].str)
	}

	return result
}

// levenshteinDistance calculates the Levenshtein distance between two strings.
// This is the minimum number of single-character edits (insertions, deletions,
// or substitutions) required to change one string into the other.
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create a matrix of distances
	// We only need two rows at a time to save memory
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)

	// Initialize first row
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}

	// Fill in the rest of the matrix
	for i := 1; i <= len(a); i++ {
		curr[0] = i

		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}

			// Minimum of deletion, insertion, and substitution
			curr[j] = min(
				prev[j]+1,      // deletion
				curr[j-1]+1,    // insertion
				prev[j-1]+cost, // substitution
			)
		}

		// Swap rows
		prev, curr = curr, prev
	}

	return prev[len(b)]
}

// min returns the minimum of three integers.
func min(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}

// FormatSuggestions formats a list of suggestions as a human-readable string.
// Example output: "Did you mean: 'name', 'names', or 'named'?"
func FormatSuggestions(suggestions []string) string {
	if len(suggestions) == 0 {
		return ""
	}

	if len(suggestions) == 1 {
		return ". Did you mean '" + suggestions[0] + "'?"
	}

	var sb strings.Builder
	sb.WriteString(". Did you mean ")

	for i, s := range suggestions {
		if i > 0 {
			if i == len(suggestions)-1 {
				sb.WriteString(" or ")
			} else {
				sb.WriteString(", ")
			}
		}
		sb.WriteByte('\'')
		sb.WriteString(s)
		sb.WriteByte('\'')
	}

	sb.WriteByte('?')
	return sb.String()
}

// ExtractPathPrefix extracts the prefix of a dot-notation path.
// For "user.profile.name", it returns "user" (the first segment).
// For "name" (no dots), it returns "name".
func ExtractPathPrefix(path string) string {
	idx := strings.Index(path, ".")
	if idx == -1 {
		return path
	}
	return path[:idx]
}

// FormatAvailableKeys formats a list of available keys as a human-readable string.
// Limits output to maxKeys to avoid very long error messages.
// Example output: "Available keys: 'name', 'email', 'age' (3 more)"
func FormatAvailableKeys(keys []string, maxKeys int) string {
	if len(keys) == 0 {
		return ""
	}

	if maxKeys <= 0 {
		maxKeys = 5
	}

	var sb strings.Builder
	sb.WriteString(". Available keys: ")

	displayCount := len(keys)
	if displayCount > maxKeys {
		displayCount = maxKeys
	}

	for i := 0; i < displayCount; i++ {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteByte('\'')
		sb.WriteString(keys[i])
		sb.WriteByte('\'')
	}

	remaining := len(keys) - displayCount
	if remaining > 0 {
		sb.WriteString(" (")
		sb.WriteString(intToString(remaining))
		sb.WriteString(" more)")
	}

	return sb.String()
}

// intToString converts an int to a string.
// Using strings.Itoa here caused an import cycle, so we inline it.
func intToString(i int) string {
	if i == 0 {
		return "0"
	}

	var neg bool
	if i < 0 {
		neg = true
		i = -i
	}

	// Max int is about 19 digits
	buf := make([]byte, 20)
	pos := len(buf)

	for i > 0 {
		pos--
		buf[pos] = byte(i%10) + '0'
		i /= 10
	}

	if neg {
		pos--
		buf[pos] = '-'
	}

	return string(buf[pos:])
}
