package parser

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var schemaLog = logger.New("parser:schema")

//go:embed schemas/main_workflow_schema.json
var mainWorkflowSchema string

//go:embed schemas/included_file_schema.json
var includedFileSchema string

//go:embed schemas/mcp_config_schema.json
var mcpConfigSchema string

// ValidateMainWorkflowFrontmatterWithSchema validates main workflow frontmatter using JSON schema
func ValidateMainWorkflowFrontmatterWithSchema(frontmatter map[string]any) error {
	schemaLog.Print("Validating main workflow frontmatter with schema")

	// Filter out ignored fields before validation
	filtered := filterIgnoredFields(frontmatter)

	// First run custom validation for command trigger conflicts (provides better error messages)
	if err := validateCommandTriggerConflicts(filtered); err != nil {
		schemaLog.Printf("Command trigger validation failed: %v", err)
		return err
	}

	// Then run the standard schema validation
	if err := validateWithSchema(filtered, mainWorkflowSchema, "main workflow file"); err != nil {
		schemaLog.Printf("Schema validation failed for main workflow: %v", err)
		return err
	}

	// Finally run other custom validation rules
	return validateEngineSpecificRules(filtered)
}

// ValidateMainWorkflowFrontmatterWithSchemaAndLocation validates main workflow frontmatter with file location info
func ValidateMainWorkflowFrontmatterWithSchemaAndLocation(frontmatter map[string]any, filePath string) error {
	// Filter out ignored fields before validation
	filtered := filterIgnoredFields(frontmatter)

	// First run custom validation for command trigger conflicts (provides better error messages)
	if err := validateCommandTriggerConflicts(filtered); err != nil {
		return err
	}

	// Then run the standard schema validation with location
	if err := validateWithSchemaAndLocation(filtered, mainWorkflowSchema, "main workflow file", filePath); err != nil {
		return err
	}

	// Finally run other custom validation rules
	return validateEngineSpecificRules(filtered)
}

// ValidateIncludedFileFrontmatterWithSchema validates included file frontmatter using JSON schema
func ValidateIncludedFileFrontmatterWithSchema(frontmatter map[string]any) error {
	schemaLog.Print("Validating included file frontmatter with schema")

	// Filter out ignored fields before validation
	filtered := filterIgnoredFields(frontmatter)

	// First run the standard schema validation
	if err := validateWithSchema(filtered, includedFileSchema, "included file"); err != nil {
		schemaLog.Printf("Schema validation failed for included file: %v", err)
		return err
	}

	// Then run custom validation for engine-specific rules
	return validateEngineSpecificRules(filtered)
}

// ValidateIncludedFileFrontmatterWithSchemaAndLocation validates included file frontmatter with file location info
func ValidateIncludedFileFrontmatterWithSchemaAndLocation(frontmatter map[string]any, filePath string) error {
	// Filter out ignored fields before validation
	filtered := filterIgnoredFields(frontmatter)

	// First run the standard schema validation with location
	if err := validateWithSchemaAndLocation(filtered, includedFileSchema, "included file", filePath); err != nil {
		return err
	}

	// Then run custom validation for engine-specific rules
	return validateEngineSpecificRules(filtered)
}

// ValidateMCPConfigWithSchema validates MCP configuration using JSON schema
func ValidateMCPConfigWithSchema(mcpConfig map[string]any, toolName string) error {
	schemaLog.Printf("Validating MCP configuration for tool: %s", toolName)
	return validateWithSchema(mcpConfig, mcpConfigSchema, fmt.Sprintf("MCP configuration for tool '%s'", toolName))
}

// validateWithSchema validates frontmatter against a JSON schema
// Cached compiled schemas to avoid recompiling on every validation
var (
	mainWorkflowSchemaOnce sync.Once
	includedFileSchemaOnce sync.Once
	mcpConfigSchemaOnce    sync.Once

	compiledMainWorkflowSchema *jsonschema.Schema
	compiledIncludedFileSchema *jsonschema.Schema
	compiledMcpConfigSchema    *jsonschema.Schema

	mainWorkflowSchemaError error
	includedFileSchemaError error
	mcpConfigSchemaError    error
)

// getCompiledMainWorkflowSchema returns the compiled main workflow schema, compiling it once and caching
func getCompiledMainWorkflowSchema() (*jsonschema.Schema, error) {
	mainWorkflowSchemaOnce.Do(func() {
		compiledMainWorkflowSchema, mainWorkflowSchemaError = compileSchema(mainWorkflowSchema, "http://contoso.com/main-workflow-schema.json")
	})
	return compiledMainWorkflowSchema, mainWorkflowSchemaError
}

// getCompiledIncludedFileSchema returns the compiled included file schema, compiling it once and caching
func getCompiledIncludedFileSchema() (*jsonschema.Schema, error) {
	includedFileSchemaOnce.Do(func() {
		compiledIncludedFileSchema, includedFileSchemaError = compileSchema(includedFileSchema, "http://contoso.com/included-file-schema.json")
	})
	return compiledIncludedFileSchema, includedFileSchemaError
}

// getCompiledMcpConfigSchema returns the compiled MCP config schema, compiling it once and caching
func getCompiledMcpConfigSchema() (*jsonschema.Schema, error) {
	mcpConfigSchemaOnce.Do(func() {
		compiledMcpConfigSchema, mcpConfigSchemaError = compileSchema(mcpConfigSchema, "http://contoso.com/mcp-config-schema.json")
	})
	return compiledMcpConfigSchema, mcpConfigSchemaError
}

// compileSchema compiles a JSON schema from a JSON string
func compileSchema(schemaJSON, schemaURL string) (*jsonschema.Schema, error) {
	schemaLog.Printf("Compiling JSON schema: %s", schemaURL)

	// Create a new compiler
	compiler := jsonschema.NewCompiler()

	// Parse the schema JSON first
	var schemaDoc any
	if err := json.Unmarshal([]byte(schemaJSON), &schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	// Add the schema as a resource
	if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}

	// Compile the schema
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	return schema, nil
}

// safeOutputMetaFields are the meta-configuration fields in safe-outputs that are NOT actual safe output types.
// These are used for configuration, not for defining safe output operations.
var safeOutputMetaFields = map[string]bool{
	"allowed-domains": true,
	"staged":          true,
	"env":             true,
	"github-token":    true,
	"app":             true,
	"max-patch-size":  true,
	"jobs":            true,
	"runs-on":         true,
	"messages":        true,
}

// GetSafeOutputTypeKeys returns the list of safe output type keys from the embedded main workflow schema.
// These are the keys under safe-outputs that define actual safe output operations (like create-issue, add-comment, etc.)
// Meta-configuration fields (like allowed-domains, staged, env, etc.) are excluded.
func GetSafeOutputTypeKeys() ([]string, error) {
	schemaLog.Print("Extracting safe output type keys from main workflow schema")

	// Parse the embedded schema JSON
	var schemaDoc map[string]any
	if err := json.Unmarshal([]byte(mainWorkflowSchema), &schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to parse main workflow schema: %w", err)
	}

	// Navigate to properties.safe-outputs.properties
	properties, ok := schemaDoc["properties"].(map[string]any)
	if !ok {
		return nil, errors.New("schema missing 'properties' field")
	}

	safeOutputs, ok := properties["safe-outputs"].(map[string]any)
	if !ok {
		return nil, errors.New("schema missing 'properties.safe-outputs' field")
	}

	safeOutputsProperties, ok := safeOutputs["properties"].(map[string]any)
	if !ok {
		return nil, errors.New("schema missing 'properties.safe-outputs.properties' field")
	}

	// Extract keys that are actual safe output types (not meta-configuration)
	var keys []string
	for key := range safeOutputsProperties {
		if !safeOutputMetaFields[key] {
			keys = append(keys, key)
		}
	}

	// Sort keys for consistent ordering
	sort.Strings(keys)

	return keys, nil
}

func validateWithSchema(frontmatter map[string]any, schemaJSON, context string) error {
	// Determine which cached schema to use based on the schemaJSON
	var schema *jsonschema.Schema
	var err error

	switch schemaJSON {
	case mainWorkflowSchema:
		schema, err = getCompiledMainWorkflowSchema()
	case includedFileSchema:
		schema, err = getCompiledIncludedFileSchema()
	case mcpConfigSchema:
		schema, err = getCompiledMcpConfigSchema()
	default:
		// Fallback for unknown schemas (shouldn't happen in normal operation)
		// Compile the schema on-the-fly
		schema, err = compileSchema(schemaJSON, "http://contoso.com/schema.json")
	}

	if err != nil {
		return fmt.Errorf("schema validation error for %s: %w", context, err)
	}

	// Convert frontmatter to JSON and back to normalize types for validation
	// Handle nil frontmatter as empty object to satisfy schema validation
	var frontmatterToValidate map[string]any
	if frontmatter == nil {
		frontmatterToValidate = make(map[string]any)
	} else {
		frontmatterToValidate = frontmatter
	}

	frontmatterJSON, err := json.Marshal(frontmatterToValidate)
	if err != nil {
		return fmt.Errorf("schema validation error for %s: failed to marshal frontmatter: %w", context, err)
	}

	var normalizedFrontmatter any
	if err := json.Unmarshal(frontmatterJSON, &normalizedFrontmatter); err != nil {
		return fmt.Errorf("schema validation error for %s: failed to unmarshal frontmatter: %w", context, err)
	}

	// Validate the normalized frontmatter
	if err := schema.Validate(normalizedFrontmatter); err != nil {
		return err
	}

	return nil
}

// validateWithSchemaAndLocation validates frontmatter against a JSON schema with location information
func validateWithSchemaAndLocation(frontmatter map[string]any, schemaJSON, context, filePath string) error {
	// First try the basic validation
	err := validateWithSchema(frontmatter, schemaJSON, context)
	if err == nil {
		return nil
	}

	// If there's an error, try to format it with precise location information
	errorMsg := err.Error()

	// Check if this is a jsonschema validation error before cleaning
	isJSONSchemaError := strings.Contains(errorMsg, "jsonschema validation failed")

	// Clean up the jsonschema error message to remove unhelpful prefixes
	if isJSONSchemaError {
		errorMsg = cleanJSONSchemaErrorMessage(errorMsg)
	}

	// Try to read the actual file content for better context
	var contextLines []string
	var frontmatterContent string
	var frontmatterStart = 2 // Default: frontmatter starts at line 2

	if filePath != "" {
		if content, readErr := os.ReadFile(filePath); readErr == nil {
			lines := strings.Split(string(content), "\n")

			// Look for frontmatter section with improved detection
			frontmatterStartIdx, frontmatterEndIdx, actualFrontmatterContent := findFrontmatterBounds(lines)

			if frontmatterStartIdx >= 0 && frontmatterEndIdx > frontmatterStartIdx {
				frontmatterContent = actualFrontmatterContent
				frontmatterStart = frontmatterStartIdx + 2 // +2 because we skip the opening "---" and use 1-based indexing

				// Use the frontmatter section plus a bit of context as context lines
				contextStart := max(0, frontmatterStartIdx)
				contextEnd := min(len(lines), frontmatterEndIdx+1)

				for i := contextStart; i < contextEnd; i++ {
					contextLines = append(contextLines, lines[i])
				}
			}
		}
	}

	// Fallback context if we couldn't read the file
	if len(contextLines) == 0 {
		contextLines = []string{"---", "# (frontmatter validation failed)", "---"}
	}

	// Try to extract precise location information from the error
	if isJSONSchemaError {
		// Extract JSON path information from the validation error
		jsonPaths := ExtractJSONPathFromValidationError(err)

		// If we have paths and frontmatter content, try to get precise locations
		if len(jsonPaths) > 0 && frontmatterContent != "" {
			// Use the first error path for the primary error location
			primaryPath := jsonPaths[0]
			location := LocateJSONPathInYAMLWithAdditionalProperties(frontmatterContent, primaryPath.Path, primaryPath.Message)

			if location.Found {
				// Adjust line number to account for frontmatter position in file
				adjustedLine := location.Line + frontmatterStart - 1

				// Create context lines around the adjusted line number in the full file
				var adjustedContextLines []string
				if filePath != "" {
					if content, readErr := os.ReadFile(filePath); readErr == nil {
						allLines := strings.Split(string(content), "\n")
						// Create context around the adjusted line (±3 lines)
						// The console formatter expects context to be centered around the error line
						contextSize := 7                                     // ±3 lines around the error
						contextStart := max(0, adjustedLine-contextSize/2-1) // -1 for 0-based indexing
						contextEnd := min(len(allLines), contextStart+contextSize)

						for i := contextStart; i < contextEnd; i++ {
							adjustedContextLines = append(adjustedContextLines, allLines[i])
						}
					}
				}

				// If we couldn't create adjusted context, fall back to frontmatter context
				if len(adjustedContextLines) == 0 {
					adjustedContextLines = contextLines
				}

				// Rewrite "additional properties not allowed" errors to be more friendly
				message := rewriteAdditionalPropertiesError(primaryPath.Message)

				// Add schema-based suggestions
				suggestions := generateSchemaBasedSuggestions(schemaJSON, primaryPath.Message, primaryPath.Path)
				if suggestions != "" {
					message = message + ". " + suggestions
				}

				// Create a compiler error with precise location information
				compilerErr := console.CompilerError{
					Position: console.ErrorPosition{
						File:   filePath,
						Line:   adjustedLine,
						Column: location.Column, // Use original column, we'll extend to word in console rendering
					},
					Type:    "error",
					Message: message,
					Context: adjustedContextLines,
					// Hints removed as per requirements
				}

				// Format and return the error
				formattedErr := console.FormatError(compilerErr)
				return errors.New(formattedErr)
			}
		}

		// Rewrite "additional properties not allowed" errors to be more friendly
		message := rewriteAdditionalPropertiesError(errorMsg)

		// Add schema-based suggestions for fallback case
		suggestions := generateSchemaBasedSuggestions(schemaJSON, errorMsg, "")
		if suggestions != "" {
			message = message + ". " + suggestions
		}

		// Fallback: Create a compiler error with basic location information
		compilerErr := console.CompilerError{
			Position: console.ErrorPosition{
				File:   filePath,
				Line:   frontmatterStart,
				Column: 1, // Use column 1 for fallback, we'll extend to word in console rendering
			},
			Type:    "error",
			Message: message,
			Context: contextLines,
			// Hints removed as per requirements
		}

		// Format and return the error
		formattedErr := console.FormatError(compilerErr)
		return errors.New(formattedErr)
	}

	// Fallback to the original error if we can't format it nicely
	return err
}

// Constants for suggestion limits and field generation
const (
	maxClosestMatches = 3  // Maximum number of closest matches to find
	maxSuggestions    = 5  // Maximum number of suggestions to show
	maxAcceptedFields = 10 // Maximum number of accepted fields to display
	maxExampleFields  = 3  // Maximum number of fields to include in example JSON
)

// generateSchemaBasedSuggestions generates helpful suggestions based on the schema and error type
func generateSchemaBasedSuggestions(schemaJSON, errorMessage, jsonPath string) string {
	// Parse the schema to extract information for suggestions
	var schemaDoc any
	if err := json.Unmarshal([]byte(schemaJSON), &schemaDoc); err != nil {
		return "" // Can't parse schema, no suggestions
	}

	// Check if this is an additional properties error
	if strings.Contains(strings.ToLower(errorMessage), "additional propert") && strings.Contains(strings.ToLower(errorMessage), "not allowed") {
		invalidProps := extractAdditionalPropertyNames(errorMessage)
		acceptedFields := extractAcceptedFieldsFromSchema(schemaDoc, jsonPath)

		if len(acceptedFields) > 0 {
			return generateFieldSuggestions(invalidProps, acceptedFields)
		}
	}

	// Check if this is a type error
	if strings.Contains(strings.ToLower(errorMessage), "got ") && strings.Contains(strings.ToLower(errorMessage), "want ") {
		example := generateExampleJSONForPath(schemaDoc, jsonPath)
		if example != "" {
			return fmt.Sprintf("Expected format: %s", example)
		}
	}

	return ""
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
		return schema // Root level
	}

	// Parse the JSON path and navigate through the schema
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
	uniqueSuggestions := removeDuplicates(suggestions)

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
		// Try to infer from other properties
		if _, hasProperties := schema["properties"]; hasProperties {
			schemaType = "object"
		} else if _, hasItems := schema["items"]; hasItems {
			schemaType = "array"
		} else {
			return nil
		}
	}

	switch schemaType {
	case "string":
		if enum, ok := schema["enum"].([]any); ok && len(enum) > 0 {
			if str, ok := enum[0].(string); ok {
				return str
			}
		}
		return "string"
	case "number", "integer":
		return 42
	case "boolean":
		return true
	case "array":
		if items, ok := schema["items"].(map[string]any); ok {
			itemExample := generateExampleFromSchema(items)
			if itemExample != nil {
				return []any{itemExample}
			}
		}
		return []any{}
	case "object":
		result := make(map[string]any)
		if properties, ok := schema["properties"].(map[string]any); ok {
			// Add required properties first
			requiredFields := make(map[string]bool)
			if required, ok := schema["required"].([]any); ok {
				for _, field := range required {
					if fieldName, ok := field.(string); ok {
						requiredFields[fieldName] = true
					}
				}
			}

			// Add a few example properties (prioritize required ones)
			count := 0

			// First, add required fields
			for propName, propSchema := range properties {
				if requiredFields[propName] && count < maxExampleFields {
					if propSchemaMap, ok := propSchema.(map[string]any); ok {
						result[propName] = generateExampleFromSchema(propSchemaMap)
						count++
					}
				}
			}

			// Then add some optional fields if we have room
			for propName, propSchema := range properties {
				if !requiredFields[propName] && count < maxExampleFields {
					if propSchemaMap, ok := propSchema.(map[string]any); ok {
						result[propName] = generateExampleFromSchema(propSchemaMap)
						count++
					}
				}
			}
		}
		return result
	}

	return nil
}

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

// GetMainWorkflowSchema returns the embedded main workflow schema JSON
func GetMainWorkflowSchema() string {
	return mainWorkflowSchema
}
