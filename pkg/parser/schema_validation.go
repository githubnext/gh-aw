package parser

import (
"fmt"

"github.com/githubnext/gh-aw/pkg/logger"
)

var schemaValidationLog = logger.New("parser:schema_validation")

// ValidateMainWorkflowFrontmatterWithSchema validates main workflow frontmatter using JSON schema
func ValidateMainWorkflowFrontmatterWithSchema(frontmatter map[string]any) error {
	schemaValidationLog.Print("Validating main workflow frontmatter with schema")

	// Filter out ignored fields before validation
	filtered := filterIgnoredFields(frontmatter)

	// First run custom validation for command trigger conflicts (provides better error messages)
	if err := validateCommandTriggerConflicts(filtered); err != nil {
		schemaValidationLog.Printf("Command trigger validation failed: %v", err)
		return err
	}

	// Then run the standard schema validation
	if err := validateWithSchema(filtered, mainWorkflowSchema, "main workflow file"); err != nil {
		schemaValidationLog.Printf("Schema validation failed for main workflow: %v", err)
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
	schemaValidationLog.Print("Validating included file frontmatter with schema")

	// Filter out ignored fields before validation
	filtered := filterIgnoredFields(frontmatter)

	// First run the standard schema validation
	if err := validateWithSchema(filtered, includedFileSchema, "included file"); err != nil {
		schemaValidationLog.Printf("Schema validation failed for included file: %v", err)
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
	schemaValidationLog.Printf("Validating MCP configuration for tool: %s", toolName)
	return validateWithSchema(mcpConfig, mcpConfigSchema, fmt.Sprintf("MCP configuration for tool '%s'", toolName))
}
