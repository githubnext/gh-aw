package workflow

import (
	"fmt"
	"sort"
	"strings"
)

// levenshteinDistance computes the Levenshtein distance between two strings.
// This is the minimum number of single-character edits (insertions, deletions, or substitutions)
// required to change one string into the other.
func levenshteinDistance(a, b string) int {
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
			currentRow[j] = min3(
				previousRow[j]+1,      // deletion
				currentRow[j-1]+1,     // insertion
				previousRow[j-1]+cost, // substitution
			)
		}

		// Swap rows for next iteration
		previousRow, currentRow = currentRow, previousRow
	}

	return previousRow[bLen]
}

// min3 returns the minimum of three integers
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// fieldSuggestion represents a suggested field name with its distance
type fieldSuggestion struct {
	field    string
	distance int
}

// suggestFieldName returns field name suggestions based on Levenshtein distance.
// Returns suggestions with edit distance ≤ 3, sorted by distance.
// Returns empty slice if no close matches found.
func suggestFieldName(invalidField string, validFields []string) []string {
	const maxDistance = 3

	// Normalize to lowercase for comparison
	invalidLower := strings.ToLower(invalidField)

	var suggestions []fieldSuggestion

	for _, validField := range validFields {
		validLower := strings.ToLower(validField)

		// Skip exact matches (distance 0)
		if invalidLower == validLower {
			continue
		}

		distance := levenshteinDistance(invalidLower, validLower)

		// Only include suggestions within acceptable distance
		if distance <= maxDistance {
			suggestions = append(suggestions, fieldSuggestion{
				field:    validField,
				distance: distance,
			})
		}
	}

	// Sort by distance (closest first), then alphabetically for ties
	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].distance != suggestions[j].distance {
			return suggestions[i].distance < suggestions[j].distance
		}
		return suggestions[i].field < suggestions[j].field
	})

	// Convert to string slice
	result := make([]string, len(suggestions))
	for i, s := range suggestions {
		result[i] = s.field
	}

	return result
}

// enhanceSchemaValidationError enhances schema validation errors with field suggestions.
// If the error indicates an unknown property, it extracts valid field names from the schema
// and provides suggestions based on fuzzy matching.
func enhanceSchemaValidationError(err error, schema map[string]interface{}) error {
	if err == nil {
		return nil
	}

	errorMsg := err.Error()

	// Check if this is an "unknown property" error
	if !strings.Contains(strings.ToLower(errorMsg), "unknown propert") {
		return err
	}

	// Extract the invalid field name from the error message
	invalidField := extractInvalidFieldName(errorMsg)
	if invalidField == "" {
		return err
	}

	// Extract valid field names from the schema
	validFields := extractValidFieldsFromSchema(schema)
	if len(validFields) == 0 {
		return err
	}

	// Get suggestions
	suggestions := suggestFieldName(invalidField, validFields)
	if len(suggestions) == 0 {
		return err
	}

	// Enhance error message with suggestions
	var enhancedMsg strings.Builder
	enhancedMsg.WriteString(errorMsg)

	// Add suggestion based on distance
	if len(suggestions) == 1 {
		// Single suggestion - show it directly
		enhancedMsg.WriteString(fmt.Sprintf(". Did you mean '%s'?", suggestions[0]))
	} else if len(suggestions) > 1 {
		// Multiple suggestions
		firstDistance := levenshteinDistance(strings.ToLower(invalidField), strings.ToLower(suggestions[0]))
		secondDistance := levenshteinDistance(strings.ToLower(invalidField), strings.ToLower(suggestions[1]))
		
		// If first suggestion is much closer (distance ≤ 2) and second is farther, show only first
		if firstDistance <= 2 && secondDistance > firstDistance {
			enhancedMsg.WriteString(fmt.Sprintf(". Did you mean '%s'?", suggestions[0]))
		} else {
			// Multiple similar suggestions - show them all
			enhancedMsg.WriteString(". Did you mean one of: ")
			for i, suggestion := range suggestions {
				if i > 0 {
					enhancedMsg.WriteString(", ")
				}
				enhancedMsg.WriteString(fmt.Sprintf("'%s'", suggestion))
			}
			enhancedMsg.WriteString("?")
		}
	}

	return fmt.Errorf("%s", enhancedMsg.String())
}

