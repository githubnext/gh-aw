package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/stringutil"
)

var schemaSuggestionsLog = logger.New("parser:schema_suggestions")

// Constants for suggestion limits and field generation
const (
	maxClosestMatches       = 3  // Maximum number of closest matches to find
	maxSuggestions          = 5  // Maximum number of suggestions to show
	maxAcceptedFields       = 10 // Maximum number of accepted fields to display
	maxExampleFields        = 3  // Maximum number of fields to include in example JSON
	maxPathSearchDistance   = 2  // Maximum Levenshtein distance for high-confidence path suggestions
	maxPathSuggestions      = 3  // Maximum number of path locations to suggest
	schemaTraversalMaxDepth = 15 // Maximum recursion depth when traversing schema
)

// schemaFieldLocation represents a location in the schema where a field is valid as a property.
type schemaFieldLocation struct {
	FieldName  string // the actual field name in the schema (may differ from query if fuzzy match)
	SchemaPath string // the parent schema path where this field is valid (e.g., "/on", "/safe-outputs")
	Distance   int    // Levenshtein distance from the query field name (0 = exact match)
}

// generateSchemaBasedSuggestions generates helpful suggestions based on the schema and error type.
// frontmatterContent is the raw YAML frontmatter text, used to extract the user's typed value for enum suggestions.
func generateSchemaBasedSuggestions(schemaJSON, errorMessage, jsonPath, frontmatterContent string) string {
	schemaSuggestionsLog.Printf("Generating schema suggestions: path=%s, schema_size=%d bytes", jsonPath, len(schemaJSON))
	schemaDoc, err := getParsedSchemaDoc(schemaJSON)
	if err != nil {
		schemaSuggestionsLog.Printf("Failed to parse schema JSON: %v", err)
		return ""
	}

	if suggestion, handled := enumSuggestion(errorMessage, jsonPath, frontmatterContent); handled {
		return suggestion
	}

	if suggestion := additionalPropertiesSuggestion(schemaDoc, errorMessage, jsonPath); suggestion != "" {
		return suggestion
	}

	if suggestion := rangeConstraintSuggestion(schemaDoc, errorMessage, jsonPath); suggestion != "" {
		return suggestion
	}

	if suggestion := typeMismatchSuggestion(schemaDoc, errorMessage, jsonPath); suggestion != "" {
		return suggestion
	}

	schemaSuggestionsLog.Print("No suggestions generated for error")
	return ""
}

// extractSchemaExamples navigates the schema to the given JSON path using
// navigateToSchemaPath and returns any "examples" array entries as formatted strings.
// Returns nil if the path does not exist in the schema or the field has no examples.
// The schema must have a top-level "properties" map for path navigation to succeed.
func extractSchemaExamples(schemaDoc any, jsonPath string) []string {
	schemaMap, ok := schemaDoc.(map[string]any)
	if !ok {
		return nil
	}
	target := navigateToSchemaPath(schemaMap, jsonPath)
	if target == nil {
		return nil
	}
	raw, ok := target["examples"]
	if !ok {
		return nil
	}
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return nil
	}
	examples := make([]string, 0, len(items))
	for _, item := range items {
		examples = append(examples, fmt.Sprintf("%v", item))
	}
	return examples
}

// extractAcceptedFieldsFromSchema extracts the list of accepted fields from a schema at a given JSON path
func extractAcceptedFieldsFromSchema(schemaDoc any, jsonPath string) []string {
	schemaMap, ok := schemaDoc.(map[string]any)
	if !ok {
		return nil
	}

	// Navigate to the schema section for the given path
	targetSchema := navigateToSchemaPath(schemaMap, jsonPath)
	if targetSchema == nil {
		return nil
	}

	// Extract properties from the target schema
	if properties, ok := targetSchema["properties"].(map[string]any); ok {
		var fields []string
		for fieldName := range properties {
			fields = append(fields, fieldName)
		}
		sort.Strings(fields) // Sort for consistent output
		return fields
	}

	return nil
}

// navigateToSchemaPath navigates to the appropriate schema section for a given JSON path
func navigateToSchemaPath(schema map[string]any, jsonPath string) map[string]any {
	if jsonPath == "" {
		schemaSuggestionsLog.Print("Navigating to root schema path")
		return schema // Root level
	}

	// Parse the JSON path and navigate through the schema
	schemaSuggestionsLog.Printf("Navigating schema path: %s", jsonPath)
	pathSegments := parseJSONPath(jsonPath)
	current := schema

	for _, segment := range pathSegments {
		switch segment.Type {
		case "key":
			// Navigate to properties -> key
			if properties, ok := current["properties"].(map[string]any); ok {
				if keySchema, ok := properties[segment.Value].(map[string]any); ok {
					current = resolveSchemaWithOneOf(keySchema)
				} else {
					return nil // Path not found in schema
				}
			} else {
				return nil // No properties in current schema
			}
		case "index":
			// For array indices, navigate to items schema
			if items, ok := current["items"].(map[string]any); ok {
				current = items
			} else {
				return nil // No items schema for array
			}
		}
	}

	return current
}

// resolveSchemaWithOneOf resolves a schema that may contain oneOf, choosing the object variant for suggestions
func resolveSchemaWithOneOf(schema map[string]any) map[string]any {
	// Check if this schema has oneOf
	if oneOf, ok := schema["oneOf"].([]any); ok {
		// Look for the first object type in oneOf that has properties
		for _, variant := range oneOf {
			if variantMap, ok := variant.(map[string]any); ok {
				if schemaType, ok := variantMap["type"].(string); ok && schemaType == "object" {
					if _, hasProperties := variantMap["properties"]; hasProperties {
						return variantMap
					}
				}
			}
		}
		// If no object with properties found, return the first variant
		if len(oneOf) > 0 {
			if firstVariant, ok := oneOf[0].(map[string]any); ok {
				return firstVariant
			}
		}
	}

	return schema
}

// generateFieldSuggestions creates a helpful suggestion message for invalid field names
func generateFieldSuggestions(invalidProps, acceptedFields []string) string {
	if len(acceptedFields) == 0 || len(invalidProps) == 0 {
		return ""
	}

	var suggestion strings.Builder

	// Find closest matches using Levenshtein distance
	var suggestions []string
	for _, invalidProp := range invalidProps {
		closest := FindClosestMatches(invalidProp, acceptedFields, maxClosestMatches)
		suggestions = append(suggestions, closest...)
	}

	// Remove duplicates
	uniqueSuggestions := sliceutil.Deduplicate(suggestions)

	// Generate appropriate message based on suggestions found
	if len(uniqueSuggestions) > 0 {
		if len(invalidProps) == 1 && len(uniqueSuggestions) == 1 {
			// Single typo, single suggestion
			suggestion.WriteString("Did you mean '")
			suggestion.WriteString(uniqueSuggestions[0])
			suggestion.WriteString("'?")
		} else {
			// Multiple typos or multiple suggestions
			suggestion.WriteString("Did you mean: ")
			if len(uniqueSuggestions) <= maxSuggestions {
				suggestion.WriteString(strings.Join(uniqueSuggestions, ", "))
			} else {
				suggestion.WriteString(strings.Join(uniqueSuggestions[:maxSuggestions], ", "))
				suggestion.WriteString(", ...")
			}
		}
	} else {
		// No close matches found - show all valid fields
		suggestion.WriteString("Valid fields are: ")
		if len(acceptedFields) <= maxAcceptedFields {
			suggestion.WriteString(strings.Join(acceptedFields, ", "))
		} else {
			suggestion.WriteString(strings.Join(acceptedFields[:maxAcceptedFields], ", "))
			suggestion.WriteString(", ...")
		}
	}

	return suggestion.String()
}

// FindClosestMatches finds the closest matching strings using Levenshtein distance.
// It returns up to maxResults matches that have a Levenshtein distance of 3 or less.
// Results are sorted by distance (closest first), then alphabetically for ties.
func FindClosestMatches(target string, candidates []string, maxResults int) []string {
	schemaSuggestionsLog.Printf("Finding closest matches for '%s' from %d candidates", target, len(candidates))
	results := stringutil.FindClosestMatches(target, candidates, maxResults)
	schemaSuggestionsLog.Printf("Found %d closest matches", len(results))
	return results
}

// LevenshteinDistance computes the Levenshtein distance between two strings.
// This is the minimum number of single-character edits (insertions, deletions, or substitutions)
// required to change one string into the other.
func LevenshteinDistance(a, b string) int {
	return stringutil.LevenshteinDistance(a, b)
}

// generateExampleJSONForPath generates an example JSON object for a specific schema path
func generateExampleJSONForPath(schemaDoc any, jsonPath string) string {
	schemaMap, ok := schemaDoc.(map[string]any)
	if !ok {
		return ""
	}

	// Navigate to the target schema
	targetSchema := navigateToSchemaPath(schemaMap, jsonPath)
	if targetSchema == nil {
		return ""
	}

	// Generate example based on schema type
	example := generateExampleFromSchema(targetSchema)
	if example == nil {
		return ""
	}

	// Convert to JSON string
	exampleJSON, err := json.Marshal(example)
	if err != nil {
		return ""
	}

	return string(exampleJSON)
}

// generateExampleFromSchema generates an example value based on a JSON schema
func generateExampleFromSchema(schema map[string]any) any {
	schemaType, ok := schema["type"].(string)
	if !ok {
		schemaType = inferSchemaType(schema)
		if schemaType == "" {
			return nil
		}
	}

	switch schemaType {
	case "string":
		return generateStringExample(schema)
	case "number", "integer":
		return generateNumberExample(schema)
	case "boolean":
		return true
	case "array":
		return generateArrayExample(schema)
	case "object":
		return generateObjectExample(schema)
	}

	return nil
}

func enumSuggestion(errorMessage, jsonPath, frontmatterContent string) (string, bool) {
	if !strings.Contains(strings.ToLower(errorMessage), "value must be one of") {
		return "", false
	}

	schemaSuggestionsLog.Print("Detected enum constraint violation")
	enumValues := extractEnumValuesFromError(errorMessage)
	actualPath := extractEnumConstraintPath(errorMessage, jsonPath)
	userValue := extractYAMLValueAtPath(frontmatterContent, actualPath)
	if userValue == "" || len(enumValues) == 0 {
		return "", true
	}

	closest := sliceutil.Deduplicate(FindClosestMatches(userValue, enumValues, maxClosestMatches))
	if len(closest) == 1 {
		return fmt.Sprintf("Did you mean '%s'?", closest[0]), true
	}
	if len(closest) > 1 {
		return fmt.Sprintf("Did you mean: %s?", strings.Join(closest, ", ")), true
	}
	return "", true
}

func additionalPropertiesSuggestion(schemaDoc any, errorMessage, jsonPath string) string {
	lowerError := strings.ToLower(errorMessage)
	if !strings.Contains(lowerError, "additional propert") || !strings.Contains(lowerError, "not allowed") {
		return ""
	}

	schemaSuggestionsLog.Print("Detected additional properties error")
	invalidProps := extractAdditionalPropertyNames(errorMessage)
	acceptedFields := extractAcceptedFieldsFromSchema(schemaDoc, jsonPath)
	var suggestions []string
	if len(acceptedFields) > 0 {
		schemaSuggestionsLog.Printf("Found %d accepted fields for invalid properties %v", len(acceptedFields), invalidProps)
		if s := generateFieldSuggestions(invalidProps, acceptedFields); s != "" {
			suggestions = append(suggestions, s)
		}
	}
	if s := generatePathLocationSuggestion(invalidProps, schemaDoc, jsonPath); s != "" {
		schemaSuggestionsLog.Printf("Found path location suggestion: %s", s)
		suggestions = append(suggestions, s)
	}
	return strings.Join(suggestions, ". ")
}

func rangeConstraintSuggestion(schemaDoc any, errorMessage, jsonPath string) string {
	lowerMsg := strings.ToLower(errorMessage)
	isRangeError := (strings.Contains(lowerMsg, "minimum:") || strings.Contains(lowerMsg, "maximum:")) &&
		strings.Contains(lowerMsg, "got ") && strings.Contains(lowerMsg, "want ")
	if !isRangeError {
		return ""
	}

	schemaSuggestionsLog.Print("Detected range constraint violation, looking for schema examples")
	if examples := extractSchemaExamples(schemaDoc, jsonPath); len(examples) > 0 {
		schemaSuggestionsLog.Printf("Found %d schema examples for %s", len(examples), jsonPath)
		return "Example values: " + strings.Join(examples, ", ")
	}
	return ""
}

func typeMismatchSuggestion(schemaDoc any, errorMessage, jsonPath string) string {
	lowerMsg := strings.ToLower(errorMessage)
	if !strings.Contains(lowerMsg, "got ") || !strings.Contains(lowerMsg, "want ") {
		return ""
	}

	schemaSuggestionsLog.Print("Detected type mismatch error")
	example := generateExampleJSONForPath(schemaDoc, jsonPath)
	if example == "" {
		return ""
	}

	schemaSuggestionsLog.Printf("Generated example JSON: length=%d bytes", len(example))
	return "Expected format: " + example
}

func inferSchemaType(schema map[string]any) string {
	if _, hasProperties := schema["properties"]; hasProperties {
		return "object"
	}
	if _, hasItems := schema["items"]; hasItems {
		return "array"
	}
	return ""
}

func generateStringExample(schema map[string]any) any {
	if enum, ok := schema["enum"].([]any); ok && len(enum) > 0 {
		if str, ok := enum[0].(string); ok {
			return str
		}
	}
	return "string"
}

func generateNumberExample(schema map[string]any) any {
	if examples, ok := schema["examples"].([]any); ok && len(examples) > 0 {
		return examples[0]
	}
	if defaultVal, ok := schema["default"]; ok {
		return defaultVal
	}
	return 42
}

func generateArrayExample(schema map[string]any) any {
	items, ok := schema["items"].(map[string]any)
	if !ok {
		return []any{}
	}
	itemExample := generateExampleFromSchema(items)
	if itemExample == nil {
		return []any{}
	}
	return []any{itemExample}
}

func generateObjectExample(schema map[string]any) any {
	result := make(map[string]any)
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return result
	}

	requiredFields := collectRequiredFields(schema)
	count := 0
	count = addObjectExamples(result, properties, requiredFields, true, count)
	addObjectExamples(result, properties, requiredFields, false, count)
	return result
}

func collectRequiredFields(schema map[string]any) map[string]struct {
} {
	requiredFields := make(map[string]struct {
	})
	required, ok := schema["required"].([]any)
	if !ok {
		return requiredFields
	}
	for _, field := range required {
		if fieldName, ok := field.(string); ok {
			requiredFields[fieldName] = struct {
			}{}
		}
	}
	return requiredFields
}

func addObjectExamples(result map[string]any, properties map[string]any, requiredFields map[string]struct {
}, includeRequired bool, count int) int {
	for propName, propSchema := range properties {
		if hasStringKey(requiredFields, propName) != includeRequired || count >= maxExampleFields {
			continue
		}
		propSchemaMap, ok := propSchema.(map[string]any)
		if !ok {
			continue
		}
		result[propName] = generateExampleFromSchema(propSchemaMap)
		count++
	}
	return count
}

// enumValuePattern matches single-quoted values in enum error messages like "value must be one of 'a', 'b', 'c'"
var enumValuePattern = regexp.MustCompile(`'([^']+)'`)

// extractEnumValuesFromError extracts the list of valid enum values from an error message
// like "value must be one of 'claude', 'codex', 'copilot', 'gemini'".
func extractEnumValuesFromError(errorMessage string) []string {
	matches := enumValuePattern.FindAllStringSubmatch(errorMessage, -1)
	var values []string
	for _, match := range matches {
		if len(match) >= 2 {
			values = append(values, match[1])
		}
	}
	return values
}

// extractYAMLValueAtPath extracts the scalar value at a JSON path from raw YAML frontmatter.
// Supports top-level paths ("/field") and two-level nested paths ("/parent/child").
// Deeper paths return an empty string.
func extractYAMLValueAtPath(yamlContent, jsonPath string) string {
	if yamlContent == "" || jsonPath == "" {
		return ""
	}
	segments := strings.SplitN(strings.TrimPrefix(jsonPath, "/"), "/", 3)
	switch len(segments) {
	case 1:
		return extractTopLevelYAMLValue(yamlContent, segments[0])
	case 2:
		return extractNestedYAMLValue(yamlContent, segments[0], segments[1])
	default:
		return ""
	}
}

// extractTopLevelYAMLValue extracts the scalar value of a top-level key from raw YAML.
// Uses horizontal-only whitespace between the colon and value to avoid matching multi-line blocks.
// Only keys at column 0 (no indentation) are matched, preventing false matches against
// nested keys with the same name.
func extractTopLevelYAMLValue(yamlContent, fieldName string) string {
	escapedField := regexp.QuoteMeta(fieldName)

	// Try single-quoted value: field: 'value'  (anchored to column 0, no leading whitespace)
	reSingle := regexp.MustCompile(`(?m)^` + escapedField + `[ \t]*:[ \t]*'([^'\n]+)'`)
	if match := reSingle.FindStringSubmatch(yamlContent); len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	// Try double-quoted value: field: "value"
	reDouble := regexp.MustCompile(`(?m)^` + escapedField + `[ \t]*:[ \t]*"([^"\n]+)"`)
	if match := reDouble.FindStringSubmatch(yamlContent); len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	// Try unquoted value: field: value
	reUnquoted := regexp.MustCompile(`(?m)^` + escapedField + `[ \t]*:[ \t]*([^'"\n#][^\n#]*?)(?:[ \t]*#.*)?$`)
	if match := reUnquoted.FindStringSubmatch(yamlContent); len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

// extractNestedYAMLValue extracts the scalar value of a direct child key under a parent key in raw YAML.
// It finds the parent key's block (by indentation), determines the direct-child indent level from
// the first non-blank line inside the block, and only matches keys at that exact indent level.
// This prevents false matches against grandchildren that share the same key name.
func extractNestedYAMLValue(yamlContent, parentKey, childKey string) string {
	lines := strings.Split(yamlContent, "\n")

	escapedParent := regexp.QuoteMeta(parentKey)
	parentPattern := regexp.MustCompile(`^(\s*)` + escapedParent + `[ \t]*:`)
	escapedChild := regexp.QuoteMeta(childKey)

	parentIndent := -1
	childIndent := -1 // indent of direct children (set on first non-blank line inside the block)
	inParentBlock := false

	for _, line := range lines {
		if !inParentBlock {
			if match := parentPattern.FindStringSubmatch(line); match != nil {
				parentIndent = len(match[1])
				inParentBlock = true
			}
			continue
		}

		// Inside parent block: skip blank lines
		if strings.TrimSpace(line) == "" {
			continue
		}
		lineIndent := len(line) - len(strings.TrimLeft(line, " \t"))

		// Left parent block if indentation returned to parent level or less
		if lineIndent <= parentIndent {
			break
		}

		// Establish the direct-child indentation from the first non-blank child line
		if childIndent == -1 {
			childIndent = lineIndent
		}

		// Only match keys at the direct-child indent level (not grandchildren deeper)
		if lineIndent != childIndent {
			continue
		}

		// Try to match child key with its value (single-quoted, double-quoted, unquoted).
		childPrefix := `^\s+` + escapedChild + `[ \t]*:[ \t]*`
		reSingle := regexp.MustCompile(childPrefix + `'([^'\n]+)'`)
		if match := reSingle.FindStringSubmatch(line); len(match) >= 2 {
			return strings.TrimSpace(match[1])
		}
		reDouble := regexp.MustCompile(childPrefix + `"([^"\n]+)"`)
		if match := reDouble.FindStringSubmatch(line); len(match) >= 2 {
			return strings.TrimSpace(match[1])
		}
		reUnquoted := regexp.MustCompile(childPrefix + `([^'"\n#][^\n#]*?)(?:[ \t]*#.*)?$`)
		if match := reUnquoted.FindStringSubmatch(line); len(match) >= 2 {
			return strings.TrimSpace(match[1])
		}
	}

	return ""
}

// extractEnumConstraintPath finds the JSON path of an enum constraint violation in an error message.
// For simple errors like "value must be one of 'a', 'b'", it returns the provided fallbackPath.
// For oneOf errors that contain a nested sub-path such as:
//
//	"- at '/permissions/contents': value must be one of 'read', 'write', 'none'"
//
// it extracts "/permissions/contents" as the actual constraint path.
var enumConstraintPathPattern = regexp.MustCompile(`at '(/[^']+)':\s*value must be one of`)

func extractEnumConstraintPath(errorMessage, fallbackPath string) string {
	if match := enumConstraintPathPattern.FindStringSubmatch(errorMessage); len(match) >= 2 {
		return match[1]
	}
	return fallbackPath
}

// collectSchemaPropertyPaths recursively collects all (fieldName, parentPath) pairs from a JSON schema document.
// It traverses properties, oneOf/anyOf/allOf, and items to build a complete picture of valid fields across the schema.
func collectSchemaPropertyPaths(schemaDoc any, currentPath string, depth int) []schemaFieldLocation {
	if depth > schemaTraversalMaxDepth {
		return nil
	}

	schemaMap, ok := schemaDoc.(map[string]any)
	if !ok {
		return nil
	}

	var results []schemaFieldLocation

	// Collect fields from properties and recurse into each property's schema
	if properties, ok := schemaMap["properties"].(map[string]any); ok {
		for fieldName, fieldSchema := range properties {
			results = append(results, schemaFieldLocation{FieldName: fieldName, SchemaPath: currentPath})
			sub := collectSchemaPropertyPaths(fieldSchema, currentPath+"/"+fieldName, depth+1)
			results = append(results, sub...)
		}
	}

	// Recurse into oneOf/anyOf/allOf variants (schema composition keywords)
	for _, keyword := range []string{"oneOf", "anyOf", "allOf"} {
		if variants, ok := schemaMap[keyword].([]any); ok {
			for _, variant := range variants {
				sub := collectSchemaPropertyPaths(variant, currentPath, depth+1)
				results = append(results, sub...)
			}
		}
	}

	// Recurse into items for array schemas
	if items, ok := schemaMap["items"].(map[string]any); ok {
		sub := collectSchemaPropertyPaths(items, currentPath, depth+1)
		results = append(results, sub...)
	}

	return results
}

// findFieldLocationsInSchema searches the entire schema for where the given field name is valid as a property.
// It first attempts an exact match, then falls back to fuzzy matching with a high-confidence distance threshold.
// The currentPath is excluded so we never suggest the same location that triggered the error.
func findFieldLocationsInSchema(schemaDoc any, targetField, currentPath string) []schemaFieldLocation {
	allLocations := collectSchemaPropertyPaths(schemaDoc, "", 0)
	targetLower := strings.ToLower(targetField)

	seen := make(map[string]struct {
	})

	// Collect exact matches first
	var exactMatches []schemaFieldLocation
	for _, loc := range allLocations {
		if loc.SchemaPath == currentPath {
			continue
		}
		key := loc.FieldName + "|" + loc.SchemaPath
		if hasStringKey(seen, key) {
			continue
		}
		seen[key] = struct {
		}{}

		if strings.EqualFold(loc.FieldName, targetField) {
			loc.Distance = 0
			exactMatches = append(exactMatches, loc)
		}
	}

	if len(exactMatches) > 0 {
		schemaSuggestionsLog.Printf("Found %d exact schema locations for field '%s'", len(exactMatches), targetField)
		return exactMatches
	}

	// Fall back to fuzzy matching with a stricter distance threshold for high confidence
	seenFuzzy := make(map[string]struct {
	})
	var fuzzyMatches []schemaFieldLocation
	for _, loc := range allLocations {
		if loc.SchemaPath == currentPath {
			continue
		}
		key := loc.FieldName + "|" + loc.SchemaPath
		if hasStringKey(seenFuzzy, key) {
			continue
		}
		seenFuzzy[key] = struct {
		}{}

		dist := LevenshteinDistance(targetLower, strings.ToLower(loc.FieldName))
		if dist > 0 && dist <= maxPathSearchDistance {
			loc.Distance = dist
			fuzzyMatches = append(fuzzyMatches, loc)
		}
	}

	// Sort fuzzy matches by distance (ascending), then path for stable output
	slices.SortFunc(fuzzyMatches, func(a, b schemaFieldLocation) int {
		if a.Distance != b.Distance {
			if a.Distance < b.Distance {
				return -1
			}
			return 1
		}
		switch {
		case a.SchemaPath < b.SchemaPath:
			return -1
		case a.SchemaPath > b.SchemaPath:
			return 1
		default:
			return 0
		}
	})

	schemaSuggestionsLog.Printf("Found %d fuzzy schema locations for field '%s'", len(fuzzyMatches), targetField)
	return fuzzyMatches
}

// formatSchemaPathForDisplay converts a JSON schema path to a human-readable string.
// e.g., "/on" → "on", "" → the root level
func formatSchemaPathForDisplay(schemaPath string) string {
	if schemaPath == "" {
		return "the root level"
	}
	return strings.TrimPrefix(schemaPath, "/")
}

// generatePathLocationSuggestion generates a suggestion message indicating where invalid fields
// belong in the schema. It searches the entire schema for each field and suggests the correct path.
func generatePathLocationSuggestion(invalidProps []string, schemaDoc any, currentPath string) string {
	if len(invalidProps) == 0 {
		return ""
	}

	var parts []string
	for _, prop := range invalidProps {
		locations := findFieldLocationsInSchema(schemaDoc, prop, currentPath)
		if len(locations) == 0 {
			continue
		}

		// Limit to the top N locations
		if len(locations) > maxPathSuggestions {
			locations = locations[:maxPathSuggestions]
		}

		// Collect unique path display names; track the actual field name for fuzzy matches
		actualFieldName := locations[0].FieldName
		var pathNames []string
		seenPaths := make(map[string]struct {
		})
		for _, loc := range locations {
			display := "'" + formatSchemaPathForDisplay(loc.SchemaPath) + "'"
			if !hasStringKey(seenPaths, display) {
				seenPaths[display] = struct {
				}{}
				pathNames = append(pathNames, display)
			}
		}
		if len(pathNames) == 0 {
			continue
		}

		var msg strings.Builder
		if !strings.EqualFold(actualFieldName, prop) {
			// Fuzzy match — tell the user the actual field name and where it belongs
			msg.WriteString("Did you mean '")
			msg.WriteString(actualFieldName)
			msg.WriteString("'? It belongs under ")
		} else {
			// Exact match — the field exists in the schema but in a different location
			msg.WriteString("'")
			msg.WriteString(prop)
			msg.WriteString("' belongs under ")
		}

		if len(pathNames) == 1 {
			msg.WriteString(pathNames[0])
		} else {
			last := pathNames[len(pathNames)-1]
			msg.WriteString(strings.Join(pathNames[:len(pathNames)-1], ", "))
			msg.WriteString(" or ")
			msg.WriteString(last)
		}

		parts = append(parts, msg.String())
	}

	return strings.Join(parts, ". ")
}
