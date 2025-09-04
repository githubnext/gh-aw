package parser

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed schemas/main_workflow_schema.json
var mainWorkflowSchema string

//go:embed schemas/included_file_schema.json
var includedFileSchema string

//go:embed schemas/mcp_config_schema.json
var mcpConfigSchema string

// ValidateMainWorkflowFrontmatterWithSchema validates main workflow frontmatter using JSON schema
func ValidateMainWorkflowFrontmatterWithSchema(frontmatter map[string]any) error {
	// First run the standard schema validation
	if err := validateWithSchema(frontmatter, mainWorkflowSchema, "main workflow file"); err != nil {
		return err
	}

	// Then run custom validation for engine-specific rules
	return validateEngineSpecificRules(frontmatter)
}

// ValidateMainWorkflowFrontmatterWithSchemaAndLocation validates main workflow frontmatter with file location info
func ValidateMainWorkflowFrontmatterWithSchemaAndLocation(frontmatter map[string]any, filePath string) error {
	// First run the standard schema validation with location
	if err := validateWithSchemaAndLocation(frontmatter, mainWorkflowSchema, "main workflow file", filePath); err != nil {
		return err
	}

	// Then run custom validation for engine-specific rules
	return validateEngineSpecificRules(frontmatter)
}

// ValidateIncludedFileFrontmatterWithSchema validates included file frontmatter using JSON schema
func ValidateIncludedFileFrontmatterWithSchema(frontmatter map[string]any) error {
	return validateWithSchema(frontmatter, includedFileSchema, "included file")
}

// ValidateIncludedFileFrontmatterWithSchemaAndLocation validates included file frontmatter with file location info
func ValidateIncludedFileFrontmatterWithSchemaAndLocation(frontmatter map[string]any, filePath string) error {
	return validateWithSchemaAndLocation(frontmatter, includedFileSchema, "included file", filePath)
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

				// Create a compiler error with precise location information
				compilerErr := console.CompilerError{
					Position: console.ErrorPosition{
						File:   filePath,
						Line:   adjustedLine,
						Column: location.Column,
					},
					Type:    "error",
					Message: primaryPath.Message,
					Context: contextLines,
					Hint:    "Check the YAML frontmatter against the schema requirements",
				}

				// Format and return the error
				formattedErr := console.FormatError(compilerErr)
				return errors.New(formattedErr)
			}
		}

		// Fallback: Create a compiler error with basic location information
		compilerErr := console.CompilerError{
			Position: console.ErrorPosition{
				File:   filePath,
				Line:   frontmatterStart,
				Column: 1,
			},
			Type:    "error",
			Message: errorMsg,
			Context: contextLines,
			Hint:    "Check the YAML frontmatter against the schema requirements",
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

	// Check if codex engine has permissions configured
	if engineID == "codex" {
		if _, hasPermissions := engineMap["permissions"]; hasPermissions {
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
