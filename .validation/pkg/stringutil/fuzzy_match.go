// Package stringutil provides utility functions for working with strings.
package stringutil

import (
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var fuzzyMatchLog = logger.New("stringutil:fuzzy_match")

// FindClosestMatches finds the closest matching strings using Levenshtein distance.
// It returns up to maxResults matches that have a Levenshtein distance of 3 or less.
// Results are sorted by distance (closest first), then alphabetically for ties.
//
// This function is useful for "Did you mean?" suggestions when a user provides
// an unrecognized value (e.g., a typo in an engine name or event type).
func FindClosestMatches(target string, candidates []string, maxResults int) []string {
	fuzzyMatchLog.Printf("FindClosestMatches: target=%q, candidates=%d, maxResults=%d", target, len(candidates), maxResults)
	type match struct {
		value    string
		distance int
	}

	const maxDistance = 3 // Maximum acceptable Levenshtein distance

	var matches []match
	targetLower := strings.ToLower(target)

	for _, candidate := range candidates {
		candidateLower := strings.ToLower(candidate)

		// Skip exact matches
		if targetLower == candidateLower {
			continue
		}

		distance := LevenshteinDistance(targetLower, candidateLower)

		// Only include if distance is within acceptable range
		if distance <= maxDistance {
			matches = append(matches, match{value: candidate, distance: distance})
		}
	}

	// Sort by distance (lower is better), then alphabetically for ties
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].distance != matches[j].distance {
			return matches[i].distance < matches[j].distance
		}
		return matches[i].value < matches[j].value
	})

	// Return top matches
	var results []string
	for i := 0; i < len(matches) && i < maxResults; i++ {
		results = append(results, matches[i].value)
	}

	fuzzyMatchLog.Printf("FindClosestMatches: returning %d match(es) within distance %d", len(results), maxDistance)
	return results
}

// LevenshteinDistance computes the Levenshtein distance between two strings.
// This is the minimum number of single-character edits (insertions, deletions, or substitutions)
// required to change one string into the other.
func LevenshteinDistance(a, b string) int {
	aLen := len(a)
	bLen := len(b)

	// Early exit for empty strings
	if aLen == 0 {
		return bLen
	}
	if bLen == 0 {
		return aLen
	}

	// Create a 2D matrix for dynamic programming
	// We only need the previous row, so we can optimize space
	previousRow := make([]int, bLen+1)
	currentRow := make([]int, bLen+1)

	// Initialize the first row (distance from empty string)
	for i := 0; i <= bLen; i++ {
		previousRow[i] = i
	}

	// Calculate distances for each character in string a
	for i := 1; i <= aLen; i++ {
		currentRow[0] = i // Distance from empty string

		for j := 1; j <= bLen; j++ {
			// Cost of substitution (0 if characters match, 1 otherwise)
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}

			// Minimum of:
			// - Deletion: previousRow[j] + 1
			// - Insertion: currentRow[j-1] + 1
			// - Substitution: previousRow[j-1] + cost
			deletion := previousRow[j] + 1
			insertion := currentRow[j-1] + 1
			substitution := previousRow[j-1] + cost

			currentRow[j] = min(deletion, min(insertion, substitution))
		}

		// Swap rows for next iteration
		previousRow, currentRow = currentRow, previousRow
	}

	return previousRow[bLen]
}
