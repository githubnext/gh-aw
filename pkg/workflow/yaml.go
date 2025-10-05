package workflow

import (
	"regexp"
	"sort"

	"github.com/goccy/go-yaml"
)

// UnquoteYAMLKey removes quotes from a YAML key at the start of a line.
// This is necessary because yaml.Marshal adds quotes around reserved words like "on".
// The function only replaces the quoted key if it appears at the start of a line
// (optionally preceded by whitespace) to avoid replacing quoted strings in values.
func UnquoteYAMLKey(yamlStr string, key string) string {
	// Create a regex pattern that matches the quoted key at the start of a line
	// Pattern: (start of line or newline) + (optional whitespace) + quoted key + colon
	pattern := `(^|\n)([ \t]*)"` + regexp.QuoteMeta(key) + `":`

	// Replacement: keep the line start and whitespace, but remove quotes from the key
	// Need to use ReplaceAllStringFunc to properly construct the replacement
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllStringFunc(yamlStr, func(match string) string {
		// Find the submatch groups
		submatches := re.FindStringSubmatch(match)
		if len(submatches) >= 3 {
			// submatches[0] is the full match
			// submatches[1] is the line start (^ or \n)
			// submatches[2] is the whitespace
			return submatches[1] + submatches[2] + key + ":"
		}
		return match
	})
}

// MarshalWithFieldOrder marshals a map to YAML with fields in a specific order.
// Priority fields are emitted first in the order specified, then remaining fields alphabetically.
// This is used to ensure GitHub Actions workflow fields appear in a conventional order.
func MarshalWithFieldOrder(data map[string]any, priorityFields []string) ([]byte, error) {
	orderedData := OrderMapFields(data, priorityFields)
	// Marshal the ordered data with proper options for GitHub Actions
	return yaml.MarshalWithOptions(orderedData,
		yaml.Indent(2),                        // Use 2-space indentation
		yaml.UseLiteralStyleIfMultiline(true), // Use literal block scalars for multiline strings
	)
}

// OrderMapFields converts a map to yaml.MapSlice with fields in a specific order.
// Priority fields are emitted first in the order specified, then remaining fields alphabetically.
// This is a helper function that can be used when you need the MapSlice directly.
func OrderMapFields(data map[string]any, priorityFields []string) yaml.MapSlice {
	var orderedData yaml.MapSlice

	// First, add priority fields in the specified order
	for _, fieldName := range priorityFields {
		if value, exists := data[fieldName]; exists {
			orderedData = append(orderedData, yaml.MapItem{Key: fieldName, Value: value})
		}
	}

	// Then add remaining fields in alphabetical order
	var remainingKeys []string
	for key := range data {
		// Skip if it's already been added as a priority field
		isPriority := false
		for _, priorityField := range priorityFields {
			if key == priorityField {
				isPriority = true
				break
			}
		}
		if !isPriority {
			remainingKeys = append(remainingKeys, key)
		}
	}

	// Sort remaining keys alphabetically
	sort.Strings(remainingKeys)

	// Add remaining fields to the ordered map
	for _, key := range remainingKeys {
		orderedData = append(orderedData, yaml.MapItem{Key: key, Value: data[key]})
	}

	return orderedData
}
