package parser

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var schemaValidationLog = logger.New("parser:schema_validation")

// Fields that cannot be used in shared/included workflows (only allowed in main workflows with 'on' field)
var sharedWorkflowForbiddenFields = map[string]bool{
	"on":              true, // Trigger field - only for main workflows
	"bots":            true,
	"cache":           true,
	"command":         true,
	"concurrency":     true,
	"container":       true,
	"env":             true,
	"environment":     true,
	"features":        true,
	"github-token":    true,
	"if":              true,
	"imports":         true,
	"labels":          true,
	"name":            true,
	"post-steps":      true,
	"roles":           true,
	"run-name":        true,
	"runs-on":         true,
	"sandbox":         true,
	"source":          true,
	"strict":          true,
	"timeout-minutes": true,
	"timeout_minutes": true,
	"tracker-id":      true,
}

// validateSharedWorkflowFields checks that a shared workflow doesn't contain forbidden fields
func validateSharedWorkflowFields(frontmatter map[string]any) error {
	var forbiddenFound []string

	for key := range frontmatter {
		if sharedWorkflowForbiddenFields[key] {
			forbiddenFound = append(forbiddenFound, key)
		}
	}

	if len(forbiddenFound) > 0 {
		if len(forbiddenFound) == 1 {
			return fmt.Errorf("field '%s' cannot be used in shared workflows (only allowed in main workflows with 'on' trigger)", forbiddenFound[0])
		}
		return fmt.Errorf("fields %v cannot be used in shared workflows (only allowed in main workflows with 'on' trigger)", forbiddenFound)
	}

	return nil
}

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

	// First check for forbidden fields in shared workflows
	if err := validateSharedWorkflowFields(filtered); err != nil {
		schemaValidationLog.Printf("Shared workflow field validation failed: %v", err)
		return err
	}

	// To validate shared workflows against the main schema, we temporarily add an 'on' field
	// This allows us to use the full schema validation while still enforcing the forbidden field check above
	tempFrontmatter := make(map[string]any)
	for k, v := range filtered {
		tempFrontmatter[k] = v
	}
	// Add a temporary 'on' field to satisfy the schema's required field
	tempFrontmatter["on"] = "push"

	// Validate with the main schema (which will catch unknown fields)
	if err := validateWithSchema(tempFrontmatter, mainWorkflowSchema, "included file"); err != nil {
		schemaValidationLog.Printf("Schema validation failed for included file: %v", err)
		return err
	}

	// Run custom validation for engine-specific rules
	return validateEngineSpecificRules(filtered)
}

// ValidateIncludedFileFrontmatterWithSchemaAndLocation validates included file frontmatter with file location info
func ValidateIncludedFileFrontmatterWithSchemaAndLocation(frontmatter map[string]any, filePath string) error {
	// Filter out ignored fields before validation
	filtered := filterIgnoredFields(frontmatter)

	// First check for forbidden fields in shared workflows
	if err := validateSharedWorkflowFields(filtered); err != nil {
		return err
	}

	// To validate shared workflows against the main schema, we temporarily add an 'on' field
	tempFrontmatter := make(map[string]any)
	for k, v := range filtered {
		tempFrontmatter[k] = v
	}
	// Add a temporary 'on' field to satisfy the schema's required field
	tempFrontmatter["on"] = "push"

	// Validate with the main schema (which will catch unknown fields)
	if err := validateWithSchemaAndLocation(tempFrontmatter, mainWorkflowSchema, "included file", filePath); err != nil {
		return err
	}

	// Run custom validation for engine-specific rules
	return validateEngineSpecificRules(filtered)
}

// ValidateMCPConfigWithSchema validates MCP configuration using JSON schema
func ValidateMCPConfigWithSchema(mcpConfig map[string]any, toolName string) error {
	schemaValidationLog.Printf("Validating MCP configuration for tool: %s", toolName)
	return validateWithSchema(mcpConfig, mcpConfigSchema, fmt.Sprintf("MCP configuration for tool '%s'", toolName))
}
