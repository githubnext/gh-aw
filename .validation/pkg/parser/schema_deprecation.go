package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/github/gh-aw/pkg/logger"
)

var schemaDeprecationLog = logger.New("parser:schema_deprecation")

// DeprecatedField represents a deprecated field with its replacement information
type DeprecatedField struct {
	Name               string // The deprecated field name (leaf key only)
	Path               string // Full dot-separated path, e.g. "tools.grep" (empty for top-level fields)
	Replacement        string // The recommended replacement field name
	Description        string // Description from the schema
	DeprecationMessage string // x-deprecation-message from the schema (preferred user-facing text)
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
		schemaDeprecationLog.Print("Getting deprecated fields from main workflow schema")
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
		schemaDeprecationLog.Printf("Found %d deprecated fields in main workflow schema", len(fields))
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
	schemaDeprecationLog.Printf("Checking frontmatter for deprecated fields: %d fields to check", len(deprecatedFields))
	var found []DeprecatedField

	for _, deprecatedField := range deprecatedFields {
		if _, exists := frontmatter[deprecatedField.Name]; exists {
			schemaDeprecationLog.Printf("Found deprecated field: %s (replacement: %s)", deprecatedField.Name, deprecatedField.Replacement)
			found = append(found, deprecatedField)
		}
	}

	schemaDeprecationLog.Printf("Deprecated field check complete: found %d of %d fields in frontmatter", len(found), len(deprecatedFields))
	return found
}

// ---- Deep (nested) schema walker ------------------------------------------------

// deprecatedFieldsDeepCache caches the result of the deep schema walk so the
// expensive 414 KB JSON unmarshal is only performed once per process lifetime.
var (
	deprecatedFieldsDeepOnce  sync.Once
	deprecatedFieldsDeepCache []DeprecatedField
	deprecatedFieldsDeepErr   error
)

// GetMainWorkflowDeprecatedFieldsDeep returns deprecated fields from the entire
// schema hierarchy (nested properties and oneOf variants) as dot-separated paths
// (e.g. "tools.grep", "tools.github.repos").
// The result is cached after the first call so the schema is only parsed once.
// Callers must not modify the returned slice.
func GetMainWorkflowDeprecatedFieldsDeep() ([]DeprecatedField, error) {
	deprecatedFieldsDeepOnce.Do(func() {
		schemaDeprecationLog.Print("Getting deep deprecated fields from main workflow schema")
		var schemaDoc map[string]any
		if err := json.Unmarshal([]byte(mainWorkflowSchema), &schemaDoc); err != nil {
			deprecatedFieldsDeepErr = fmt.Errorf("failed to parse main workflow schema: %w", err)
			return
		}
		var fields []DeprecatedField
		collectDeprecatedDeep(schemaDoc, "", &fields)
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Path < fields[j].Path
		})
		deprecatedFieldsDeepCache = fields
		schemaDeprecationLog.Printf("Found %d deprecated fields (deep) in main workflow schema", len(fields))
	})
	return deprecatedFieldsDeepCache, deprecatedFieldsDeepErr
}

// collectDeprecatedDeep recursively walks a schema node and appends any
// deprecated leaf properties to results.
//
// parentPath is the dot-joined path to the current schema node's parent
// (empty for the root schema).  The function looks at the node's "properties"
// map (and "oneOf" / "anyOf" / "allOf" variants that may add more properties
// at the same level) to find fields marked deprecated:true.
func collectDeprecatedDeep(schemaNode map[string]any, parentPath string, results *[]DeprecatedField) {
	properties, ok := schemaNode["properties"].(map[string]any)
	if !ok {
		return
	}

	for fieldName, fieldSchema := range properties {
		fieldSchemaMap, ok := fieldSchema.(map[string]any)
		if !ok {
			continue
		}

		path := fieldName
		if parentPath != "" {
			path = parentPath + "." + fieldName
		}

		if isDeprecated, ok := fieldSchemaMap["deprecated"].(bool); ok && isDeprecated {
			description := ""
			if desc, ok := fieldSchemaMap["description"].(string); ok {
				description = desc
			}
			deprecationMsg := ""
			if msg, ok := fieldSchemaMap["x-deprecation-message"].(string); ok {
				deprecationMsg = msg
			}
			replacement := extractReplacementFromDescription(description)

			*results = append(*results, DeprecatedField{
				Name:               fieldName,
				Path:               path,
				Replacement:        replacement,
				Description:        description,
				DeprecationMessage: deprecationMsg,
			})
			// Do not recurse further into a deprecated field — its children
			// are implicitly deprecated through the parent.
			continue
		}

		// Recurse into nested properties.
		collectDeprecatedDeep(fieldSchemaMap, path, results)

		// Also recurse into oneOf / anyOf / allOf variants: these can introduce
		// additional properties at the same level (e.g. tools.github has an
		// oneOf with an object variant that owns toolset, repos, etc.).
		for _, keyword := range []string{"oneOf", "anyOf", "allOf"} {
			if variants, ok := fieldSchemaMap[keyword].([]any); ok {
				for _, v := range variants {
					if vm, ok := v.(map[string]any); ok {
						collectDeprecatedDeep(vm, path, results)
					}
				}
			}
		}
	}
}

// FindDeprecatedFieldsInFrontmatterDeep checks the full (possibly nested)
// frontmatter map for deprecated fields identified by their dot-separated paths.
// It returns every DeprecatedField whose path resolves to an existing key in the
// frontmatter (e.g. "tools.grep" matches frontmatter["tools"]["grep"]).
func FindDeprecatedFieldsInFrontmatterDeep(frontmatter map[string]any, deprecatedFields []DeprecatedField) []DeprecatedField {
	schemaDeprecationLog.Printf("Deep-checking frontmatter for deprecated fields: %d fields to check", len(deprecatedFields))
	var found []DeprecatedField

	for _, f := range deprecatedFields {
		lookupPath := f.Path
		if lookupPath == "" {
			lookupPath = f.Name
		}
		segments := strings.Split(lookupPath, ".")
		if fieldExistsAtPath(frontmatter, segments) {
			schemaDeprecationLog.Printf("Found deprecated field at path %q", lookupPath)
			found = append(found, f)
		}
	}

	schemaDeprecationLog.Printf("Deep deprecated field check complete: found %d", len(found))
	return found
}

// fieldExistsAtPath reports whether the nested key path described by segments
// exists in m.  An empty segments slice always returns false.
func fieldExistsAtPath(m map[string]any, segments []string) bool {
	if len(segments) == 0 {
		return false
	}
	value, exists := m[segments[0]]
	if !exists {
		return false
	}
	if len(segments) == 1 {
		return true
	}
	nested, ok := value.(map[string]any)
	if !ok {
		return false
	}
	return fieldExistsAtPath(nested, segments[1:])
}
