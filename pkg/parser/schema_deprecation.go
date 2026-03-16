package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"sync"
)

// DeprecatedField represents a deprecated field with its replacement information
type DeprecatedField struct {
	Name        string // The deprecated field name
	Replacement string // The recommended replacement field name
	Description string // Description from the schema
}

// deprecatedFieldsCache caches the result of parsing the main workflow schema so that
// the expensive 414KB JSON unmarshal is only performed once per process lifetime.
// Both the result and any error are cached permanently: since mainWorkflowSchema is an
// embedded compile-time constant, a parse failure is always a programming error (not
// transient), so re-parsing on subsequent calls would produce the same failure.
var (
	deprecatedFieldsOnce  sync.Once
	deprecatedFieldsCache []DeprecatedField
	deprecatedFieldsErr   error
)

// GetMainWorkflowDeprecatedFields returns a list of deprecated fields from the main workflow schema.
// The result is cached after the first call so the schema is only parsed once per process.
// Callers must not modify the returned slice.
func GetMainWorkflowDeprecatedFields() ([]DeprecatedField, error) {
	deprecatedFieldsOnce.Do(func() {
		log.Print("Getting deprecated fields from main workflow schema")
		var schemaDoc map[string]any
		if err := json.Unmarshal([]byte(mainWorkflowSchema), &schemaDoc); err != nil {
			deprecatedFieldsErr = fmt.Errorf("failed to parse main workflow schema: %w", err)
			return
		}
		fields, err := extractDeprecatedFields(schemaDoc)
		if err != nil {
			deprecatedFieldsErr = err
			return
		}
		deprecatedFieldsCache = fields
		log.Printf("Found %d deprecated fields in main workflow schema", len(fields))
	})
	return deprecatedFieldsCache, deprecatedFieldsErr
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

// replacementPatterns are pre-compiled regexes used by extractReplacementFromDescription.
// Pre-compiling avoids repeated compilation overhead when extracting replacements from
// many deprecated field descriptions.
var replacementPatterns = []*regexp.Regexp{
	regexp.MustCompile(`[Uu]se '([^']+)' instead`),
	regexp.MustCompile(`[Uu]se "([^"]+)" instead`),
	regexp.MustCompile("[Uu]se `([^`]+)` instead"),
	regexp.MustCompile(`[Rr]eplace(?:d)? (?:with|by) '([^']+)'`),
	regexp.MustCompile(`[Rr]eplace(?:d)? (?:with|by) "([^"]+)"`),
}

// extractReplacementFromDescription extracts the replacement field name from a description.
// It looks for patterns like "Use 'field-name' instead" or "Deprecated: Use 'field-name'".
func extractReplacementFromDescription(description string) string {
	for _, re := range replacementPatterns {
		if match := re.FindStringSubmatch(description); len(match) >= 2 {
			return match[1]
		}
	}

	return ""
}

// FindDeprecatedFieldsInFrontmatter checks frontmatter for deprecated fields
// Returns a list of deprecated fields that were found
func FindDeprecatedFieldsInFrontmatter(frontmatter map[string]any, deprecatedFields []DeprecatedField) []DeprecatedField {
	log.Printf("Checking frontmatter for deprecated fields: %d fields to check", len(deprecatedFields))
	var found []DeprecatedField

	for _, deprecatedField := range deprecatedFields {
		if _, exists := frontmatter[deprecatedField.Name]; exists {
			log.Printf("Found deprecated field: %s (replacement: %s)", deprecatedField.Name, deprecatedField.Replacement)
			found = append(found, deprecatedField)
		}
	}

	log.Printf("Deprecated field check complete: found %d of %d fields in frontmatter", len(found), len(deprecatedFields))
	return found
}
