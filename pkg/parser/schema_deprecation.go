package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
)

// DeprecatedField represents a deprecated field with its replacement information
type DeprecatedField struct {
	Name        string // The deprecated field name
	Replacement string // The recommended replacement field name
	Description string // Description from the schema
}

// GetMainWorkflowDeprecatedFields returns a list of deprecated fields from the main workflow schema
func GetMainWorkflowDeprecatedFields() ([]DeprecatedField, error) {
	// Parse the schema JSON
	var schemaDoc map[string]any
	if err := json.Unmarshal([]byte(mainWorkflowSchema), &schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to parse main workflow schema: %w", err)
	}

	return extractDeprecatedFields(schemaDoc)
}

// extractDeprecatedFields extracts deprecated fields from a schema document
func extractDeprecatedFields(schemaDoc map[string]any) ([]DeprecatedField, error) {
	var deprecated []DeprecatedField

	// Look for properties in the schema
	properties, ok := schemaDoc["properties"].(map[string]any)
	if !ok {
		return deprecated, nil
	}

	// Check each property for deprecation
	for fieldName, fieldSchema := range properties {
		fieldSchemaMap, ok := fieldSchema.(map[string]any)
		if !ok {
			continue
		}

		// Check if the field is marked as deprecated
		if isDeprecated, ok := fieldSchemaMap["deprecated"].(bool); ok && isDeprecated {
			// Extract description to find replacement suggestion
			description := ""
			if desc, ok := fieldSchemaMap["description"].(string); ok {
				description = desc
			}

			// Try to extract replacement from description
			replacement := extractReplacementFromDescription(description)

			deprecated = append(deprecated, DeprecatedField{
				Name:        fieldName,
				Replacement: replacement,
				Description: description,
			})
		}
	}

	// Sort by field name for consistent output
	sort.Slice(deprecated, func(i, j int) bool {
		return deprecated[i].Name < deprecated[j].Name
	})

	return deprecated, nil
}

// extractReplacementFromDescription extracts the replacement field name from a description
// It looks for patterns like "Use 'field-name' instead" or "Deprecated: Use 'field-name'"
func extractReplacementFromDescription(description string) string {
	// Common patterns in deprecation messages
	patterns := []string{
		`[Uu]se '([^']+)' instead`,
		`[Uu]se "([^"]+)" instead`,
		`[Uu]se ` + "`" + `([^` + "`" + `]+)` + "`" + ` instead`,
		`[Rr]eplace(?:d)? (?:with|by) '([^']+)'`,
		`[Rr]eplace(?:d)? (?:with|by) "([^"]+)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(description); len(match) >= 2 {
			return match[1]
		}
	}

	return ""
}

// FindDeprecatedFieldsInFrontmatter checks frontmatter for deprecated fields
// Returns a list of deprecated fields that were found
func FindDeprecatedFieldsInFrontmatter(frontmatter map[string]any, deprecatedFields []DeprecatedField) []DeprecatedField {
	var found []DeprecatedField

	for _, deprecatedField := range deprecatedFields {
		if _, exists := frontmatter[deprecatedField.Name]; exists {
			found = append(found, deprecatedField)
		}
	}

	return found
}
