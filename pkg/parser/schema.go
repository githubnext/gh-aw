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

// ignoredFields are fields that should be silently ignored during frontmatter validation
var ignoredFields = []string{"description", "applyTo"}

// filterIgnoredFields removes ignored fields from frontmatter without warnings
func filterIgnoredFields(frontmatter map[string]any) map[string]any {
	if frontmatter == nil {
		return nil
	}

	// Create a copy of the frontmatter map without ignored fields
	filtered := make(map[string]any)
	for key, value := range frontmatter {
		// Skip ignored fields
		ignored := false
		for _, ignoredField := range ignoredFields {
			if key == ignoredField {
				ignored = true
				break
			}
		}
		if !ignored {
			filtered[key] = value
		}
	}

	return filtered
}

// ValidateMainWorkflowFrontmatterWithSchema validates main workflow frontmatter using JSON schema
func ValidateMainWorkflowFrontmatterWithSchema(frontmatter map[string]any) error {
	schemaLog.Print("Validating main workflow frontmatter with schema")

	// Filter out ignored fields before validation
	filtered := filterIgnoredFields(frontmatter)

	// First run the standard schema validation
	if err := validateWithSchema(filtered, mainWorkflowSchema, "main workflow file"); err != nil {
		schemaLog.Printf("Schema validation failed for main workflow: %v", err)
		return err
	}

	// Then run custom validation for engine-specific rules
	return validateEngineSpecificRules(filtered)
}

// ValidateMainWorkflowFrontmatterWithSchemaAndLocation validates main workflow frontmatter with file location info
func ValidateMainWorkflowFrontmatterWithSchemaAndLocation(frontmatter map[string]any, filePath string) error {
	// Filter out ignored fields before validation
	filtered := filterIgnoredFields(frontmatter)

	// First run the standard schema validation with location
	if err := validateWithSchemaAndLocation(filtered, mainWorkflowSchema, "main workflow file", filePath); err != nil {
		return err
	}

	// Then run custom validation for engine-specific rules
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
	return validateWithSchema(mcpConfig, mcpConfigSchema, fmt.Sprintf("MCP configuration for tool '%s'", toolName))
}

// validateWithSchema validates frontmatter against a JSON schema
func validateWithSchema(frontmatter map[string]any, schemaJSON, context string) error {
	// Create a new compiler
	compiler := jsonschema.NewCompiler()

	// Parse the schema JSON first
	var schemaDoc any
	if err := json.Unmarshal([]byte(schemaJSON), &schemaDoc); err != nil {
		return fmt.Errorf("schema validation error for %s: failed to parse schema JSON: %w", context, err)
	}

	// Add the schema as a resource with a temporary URL
	schemaURL := "http://contoso.com/schema.json"
	if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
		return fmt.Errorf("schema validation error for %s: failed to add schema resource: %w", context, err)
	}

	// Compile the schema
	schema, err := compiler.Compile(schemaURL)
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

// cleanJSONSchemaErrorMessage removes unhelpful prefixes from jsonschema validation errors
func cleanJSONSchemaErrorMessage(errorMsg string) string {
	// Split the error message into lines
	lines := strings.Split(errorMsg, "\n")

	var cleanedLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip the "jsonschema validation failed" line entirely
		if strings.HasPrefix(line, "jsonschema validation failed") {
			continue
		}

		// Remove the unhelpful "- at '': " prefix from error descriptions
		line = strings.TrimPrefix(line, "- at '': ")

		// Keep non-empty lines that have actual content
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}

	// Join the cleaned lines back together
	result := strings.Join(cleanedLines, "\n")

	// If we have no meaningful content left, return a generic message
	if strings.TrimSpace(result) == "" {
		return "schema validation failed"
	}

	return result
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// validateEngineSpecificRules validates engine-specific rules that are not easily expressed in JSON schema
func validateEngineSpecificRules(frontmatter map[string]any) error {
	// Check if engine is configured
	engine, ok := frontmatter["engine"]
	if !ok {
		return nil // No engine specified, nothing to validate
	}

	// Handle string format engine
	if engineStr, ok := engine.(string); ok {
		schemaLog.Printf("Validating engine-specific rules for string engine: %s", engineStr)
		// String format doesn't support permissions, so no validation needed
		_ = engineStr
		return nil
	}

	// Handle object format engine
	engineMap, ok := engine.(map[string]any)
	if !ok {
		return nil // Invalid engine format, but this should be caught by schema validation
	}

	// Check engine ID
	engineID, ok := engineMap["id"].(string)
	if !ok {
		return nil // Missing or invalid ID, but this should be caught by schema validation
	}

	schemaLog.Printf("Validating engine-specific rules for engine: %s", engineID)

	// Check if codex engine has permissions configured
	if engineID == "codex" {
		if _, hasPermissions := engineMap["permissions"]; hasPermissions {
			schemaLog.Printf("Codex engine has invalid permissions configuration")
			return errors.New("engine permissions are not supported for codex engine. Only Claude engine supports permissions configuration")
		}
	}

	return nil
}

// findFrontmatterBounds finds the start and end indices of frontmatter in file lines
// Returns: startIdx (-1 if not found), endIdx (-1 if not found), frontmatterContent
func findFrontmatterBounds(lines []string) (startIdx int, endIdx int, frontmatterContent string) {
	startIdx = -1
	endIdx = -1

	// Look for the opening "---"
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			startIdx = i
			break
		}
		// Skip empty lines and comments at the beginning
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			// Found non-empty, non-comment line before "---" - no frontmatter
			return -1, -1, ""
		}
	}

	if startIdx == -1 {
		return -1, -1, ""
	}

	// Look for the closing "---"
	for i := startIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "---" {
			endIdx = i
			break
		}
	}

	if endIdx == -1 {
		// No closing "---" found
		return -1, -1, ""
	}

	// Extract frontmatter content between the markers
	frontmatterLines := lines[startIdx+1 : endIdx]
	frontmatterContent = strings.Join(frontmatterLines, "\n")

	return startIdx, endIdx, frontmatterContent
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

	if len(invalidProps) == 1 {
		suggestion.WriteString("Did you mean one of: ")
	} else {
		suggestion.WriteString("Valid fields are: ")
	}

	// Find closest matches using simple string distance
	var suggestions []string
	for _, invalidProp := range invalidProps {
		closest := findClosestMatches(invalidProp, acceptedFields, maxClosestMatches)
		suggestions = append(suggestions, closest...)
	}

	// If we have specific suggestions, show them first
	if len(suggestions) > 0 {
		// Remove duplicates
		uniqueSuggestions := removeDuplicates(suggestions)
		if len(uniqueSuggestions) <= maxSuggestions {
			suggestion.WriteString(strings.Join(uniqueSuggestions, ", "))
		} else {
			suggestion.WriteString(strings.Join(uniqueSuggestions[:maxSuggestions], ", "))
			suggestion.WriteString(", ...")
		}
	} else {
		// Show all accepted fields if no close matches
		if len(acceptedFields) <= maxAcceptedFields {
			suggestion.WriteString(strings.Join(acceptedFields, ", "))
		} else {
			suggestion.WriteString(strings.Join(acceptedFields[:maxAcceptedFields], ", "))
			suggestion.WriteString(", ...")
		}
	}

	return suggestion.String()
}

// findClosestMatches finds the closest matching strings using simple edit distance heuristics
func findClosestMatches(target string, candidates []string, maxResults int) []string {
	type match struct {
		value string
		score int
	}

	var matches []match
	targetLower := strings.ToLower(target)

	for _, candidate := range candidates {
		candidateLower := strings.ToLower(candidate)
		score := calculateSimilarityScore(targetLower, candidateLower)

		// Only include if there's some similarity
		if score > 0 {
			matches = append(matches, match{value: candidate, score: score})
		}
	}

	// Sort by score (higher is better)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	// Return top matches
	var results []string
	for i := 0; i < len(matches) && i < maxResults; i++ {
		results = append(results, matches[i].value)
	}

	return results
}

// calculateSimilarityScore calculates a simple similarity score between two strings
func calculateSimilarityScore(a, b string) int {
	// Early exit for obviously poor matches (length difference > 2x shorter string length)
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	lengthDiff := abs(len(a) - len(b))
	if lengthDiff > minLen*2 && minLen > 0 {
		return 0
	}

	// Simple heuristics for string similarity
	score := 0

	// Bonus for substring matches
	if strings.Contains(b, a) || strings.Contains(a, b) {
		score += 10
	}

	// Bonus for common prefixes
	commonPrefix := 0
	for i := 0; i < len(a) && i < len(b) && a[i] == b[i]; i++ {
		commonPrefix++
	}
	score += commonPrefix * 2

	// Bonus for common suffixes
	commonSuffix := 0
	for i := 0; i < len(a) && i < len(b) && a[len(a)-1-i] == b[len(b)-1-i]; i++ {
		commonSuffix++
	}
	score += commonSuffix * 2

	// Penalty for length difference
	score -= lengthDiff

	return score
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

// removeDuplicates removes duplicate strings from a slice
func removeDuplicates(strings []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, str := range strings {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}

	return result
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// rewriteAdditionalPropertiesError rewrites "additional properties not allowed" errors to be more user-friendly
func rewriteAdditionalPropertiesError(message string) string {
	// Check if this is an "additional properties not allowed" error
	if strings.Contains(strings.ToLower(message), "additional propert") && strings.Contains(strings.ToLower(message), "not allowed") {
		// Extract property names from the message using regex
		re := regexp.MustCompile(`additional propert(?:y|ies) (.+?) not allowed`)
		match := re.FindStringSubmatch(message)

		if len(match) >= 2 {
			properties := match[1]
			// Clean up the property list and make it more readable
			properties = strings.ReplaceAll(properties, "'", "")

			if strings.Contains(properties, ",") {
				return fmt.Sprintf("Unknown properties: %s", properties)
			} else {
				return fmt.Sprintf("Unknown property: %s", properties)
			}
		}
	}

	return message
}
