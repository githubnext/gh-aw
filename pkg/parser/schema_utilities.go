package parser

import (
	"slices"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var schemaUtilitiesLog = logger.New("parser:schema_utilities")

// filterIgnoredFields removes ignored fields from frontmatter without warnings
// NOTE: This function is kept for backward compatibility but currently does nothing
// as all previously ignored fields (description, applyTo) are now validated by the schema
func filterIgnoredFields(frontmatter map[string]any) map[string]any {
	if frontmatter == nil {
		return nil
	}

	// Check if there are any ignored fields configured
	if len(constants.IgnoredFrontmatterFields) == 0 {
		// No fields to filter, return as-is
		return frontmatter
	}

	schemaUtilitiesLog.Printf("Filtering ignored frontmatter fields: checking %d ignored field(s) against %d frontmatter keys", len(constants.IgnoredFrontmatterFields), len(frontmatter))

	// Create a copy of the frontmatter map without ignored fields
	filtered := make(map[string]any)
	for key, value := range frontmatter {
		// Skip ignored fields
		ignored := slices.Contains(constants.IgnoredFrontmatterFields, key)
		if ignored {
			schemaUtilitiesLog.Printf("Removing ignored frontmatter field: %s", key)
		} else {
			filtered[key] = value
		}
	}

	return filtered
}
