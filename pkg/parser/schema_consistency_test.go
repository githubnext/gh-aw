package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestSchemaDefaultsMatchCode verifies that default values documented in the schema
// match the actual default values used in the code. This prevents documentation drift
// where the schema says one thing but the code does another.
//
// This test specifically checks safe-outputs configuration defaults that are critical
// for users to understand expected behavior when fields are omitted.
func TestSchemaDefaultsMatchCode(t *testing.T) {
	// Parse the main workflow schema
	var schemaDoc map[string]any
	if err := json.Unmarshal([]byte(mainWorkflowSchema), &schemaDoc); err != nil {
		t.Fatalf("Failed to parse main workflow schema: %v", err)
	}

	// Navigate to safe-outputs properties
	safeOutputsProps, err := navigateToSafeOutputsProperties(schemaDoc)
	if err != nil {
		t.Fatalf("Failed to navigate to safe-outputs properties: %v", err)
	}

	// Test cases for known default value inconsistencies
	tests := []struct {
		name          string
		fieldPath     string // Path in schema (e.g., "create-pull-request-review-comment.max")
		schemaDefault int    // Expected default value according to schema description
		codeDefault   int    // Actual default value used in code
		codeLocation  string // Where in code this default is set (for reference)
	}{
		{
			name:          "PR review comments default mismatch",
			fieldPath:     "create-pull-request-review-comment.max",
			schemaDefault: 1,  // Schema says "default: 1"
			codeDefault:   10, // Code uses 10 (safe_outputs_config.go:675)
			codeLocation:  "pkg/workflow/safe_outputs_config.go:675",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract the schema default from description
			schemaDefault, err := extractDefaultFromSchemaDescription(safeOutputsProps, tt.fieldPath)
			if err != nil {
				t.Fatalf("Failed to extract schema default for %s: %v", tt.fieldPath, err)
			}

			// Verify schema documents the expected default
			if schemaDefault != tt.schemaDefault {
				t.Errorf("Schema default mismatch: expected schema to document default=%d, got %d",
					tt.schemaDefault, schemaDefault)
			}

			// The real test: schema default should match code default
			if schemaDefault != tt.codeDefault {
				t.Errorf("Schema-code default value mismatch for %s:\n"+
					"  Schema documents default: %d\n"+
					"  Code uses default: %d (at %s)\n"+
					"  These values must match to avoid user confusion.",
					tt.fieldPath, schemaDefault, tt.codeDefault, tt.codeLocation)
			}
		})
	}
}

// TestTimeDeltaConstraintsDocumented verifies that validation constraints enforced in code
// are properly documented in the schema. This ensures users can discover limits before
// runtime validation errors.
//
// Specifically checks time delta maximum values enforced in time_delta.go parsing.
func TestTimeDeltaConstraintsDocumented(t *testing.T) {
	// These are the actual limits enforced in pkg/workflow/time_delta.go
	codeConstraints := map[string]int{
		"months":  12,     // line 142
		"weeks":   52,     // line 145
		"days":    365,    // line 148
		"hours":   8760,   // line 151 (365 * 24)
		"minutes": 525600, // line 154 (365 * 24 * 60)
	}

	// Parse the main workflow schema
	var schemaDoc map[string]any
	if err := json.Unmarshal([]byte(mainWorkflowSchema), &schemaDoc); err != nil {
		t.Fatalf("Failed to parse main workflow schema: %v", err)
	}

	// Check if time delta constraints are documented anywhere in the schema
	// Time deltas are used in various places like stop-after, schedule triggers, etc.
	schemaJSON := string(mainWorkflowSchema)

	for unit, maxValue := range codeConstraints {
		t.Run(fmt.Sprintf("max_%s_documented", unit), func(t *testing.T) {
			// Look for the constraint in schema descriptions or constraints
			// We're checking if the maximum value appears anywhere in the schema
			pattern := fmt.Sprintf(`%d\s*%s`, maxValue, unit)
			matched, _ := regexp.MatchString(pattern, schemaJSON)

			if !matched {
				// Also check for the unit name in validation contexts
				// The schema might document these limits in property descriptions
				hasUnitMention := strings.Contains(schemaJSON, unit)
				hasMaxValueMention := strings.Contains(schemaJSON, fmt.Sprintf("%d", maxValue))

				if !hasUnitMention || !hasMaxValueMention {
					t.Logf("WARNING: Time delta maximum for %s (%d) may not be documented in schema.\n"+
						"  Code enforces: maximum of %d %s (pkg/workflow/time_delta.go)\n"+
						"  Schema should document this limit so users know constraints before validation errors.",
						unit, maxValue, maxValue, unit)
				}
			}
		})
	}
}

// TestAllSafeOutputFieldsHaveSchemaDefinitions verifies that all safe output types
// used in the code have corresponding definitions in the schema. This prevents cases
// where code references fields that don't exist in the schema.
//
// This is a structural consistency check to ensure schema and code stay in sync.
func TestAllSafeOutputFieldsHaveSchemaDefinitions(t *testing.T) {
	// Parse the main workflow schema
	var schemaDoc map[string]any
	if err := json.Unmarshal([]byte(mainWorkflowSchema), &schemaDoc); err != nil {
		t.Fatalf("Failed to parse main workflow schema: %v", err)
	}

	// Get safe output type keys from schema
	schemaKeys, err := GetSafeOutputTypeKeys()
	if err != nil {
		t.Fatalf("Failed to get safe output type keys from schema: %v", err)
	}

	// Convert to map for easy lookup
	schemaKeysMap := make(map[string]bool)
	for _, key := range schemaKeys {
		schemaKeysMap[key] = true
	}

	// These are the safe output types that should be in the schema
	// Based on HasSafeOutputsEnabled and similar code in safe_outputs_config.go
	// Note: Schema names may differ slightly from code (e.g., "upload-assets" plural vs "upload_asset" singular in code)
	expectedFields := []string{
		"create-issue",
		"create-agent-task",
		"create-discussion",
		"close-discussion",
		"close-issue",
		"close-pull-request",
		"add-comment",
		"create-pull-request",
		"create-pull-request-review-comment",
		"create-code-scanning-alert",
		"add-labels",
		"add-reviewer",
		"assign-milestone",
		"assign-to-agent",
		"assign-to-user",
		"update-issue",
		"update-pull-request",
		"push-to-pull-request-branch",
		"upload-assets", // Note: Schema uses plural "upload-assets", code uses "upload_asset"
		"missing-tool",
		"noop", // Note: Schema uses "noop" (no dash), code references it as "noop"
		"link-sub-issue",
		"hide-comment",
	}

	for _, field := range expectedFields {
		t.Run(fmt.Sprintf("field_%s_in_schema", field), func(t *testing.T) {
			if !schemaKeysMap[field] {
				t.Errorf("Safe output field '%s' is referenced in code but missing from schema.\n"+
					"  Code references this field in pkg/workflow/safe_outputs_config.go\n"+
					"  Schema must include definition for this field.",
					field)
			}
		})
	}
}

// TestEngineConfigFieldsMatchSchema verifies that engine configuration fields
// used in the code match what's defined in the schema. This ensures no undocumented
// or orphaned fields exist.
func TestEngineConfigFieldsMatchSchema(t *testing.T) {
	// Parse the main workflow schema
	var schemaDoc map[string]any
	if err := json.Unmarshal([]byte(mainWorkflowSchema), &schemaDoc); err != nil {
		t.Fatalf("Failed to parse main workflow schema: %v", err)
	}

	// Navigate to engine_config definition
	defs, ok := schemaDoc["$defs"].(map[string]any)
	if !ok {
		t.Fatal("Schema missing $defs section")
	}

	engineConfig, ok := defs["engine_config"].(map[string]any)
	if !ok {
		t.Fatal("Schema missing engine_config definition")
	}

	// Get the object variant from oneOf (engine can be string or object)
	oneOf, ok := engineConfig["oneOf"].([]any)
	if !ok || len(oneOf) < 2 {
		t.Fatal("engine_config oneOf structure unexpected")
	}

	// The second variant should be the object type
	engineObject, ok := oneOf[1].(map[string]any)
	if !ok {
		t.Fatal("engine_config object variant not found")
	}

	properties, ok := engineObject["properties"].(map[string]any)
	if !ok {
		t.Fatal("engine_config properties not found")
	}

	// These are the fields that EngineConfig struct has (from pkg/workflow/engine.go)
	codeFields := []string{
		"id",
		"version",
		"model",
		"max-turns",
		"concurrency",
		"user-agent",
		"env",
		"steps",
		"error_patterns", // Note: schema uses underscore, not dash
		"config",
		"args",
		// Note: Firewall is handled separately and not a direct schema field
	}

	for _, field := range codeFields {
		t.Run(fmt.Sprintf("field_%s_in_schema", field), func(t *testing.T) {
			if _, exists := properties[field]; !exists {
				t.Errorf("Engine config field '%s' is used in code but missing from schema.\n"+
					"  Code defines this field in pkg/workflow/engine.go (EngineConfig struct)\n"+
					"  Schema must include definition for this field.",
					field)
			}
		})
	}

	// Also check the reverse: fields in schema should be used in code
	// This catches orphaned schema fields
	for field := range properties {
		t.Run(fmt.Sprintf("schema_field_%s_used_in_code", field), func(t *testing.T) {
			found := false
			for _, codeField := range codeFields {
				if field == codeField {
					found = true
					break
				}
			}
			if !found {
				t.Logf("INFO: Schema has engine config field '%s' that may not be used in EngineConfig struct.\n"+
					"  This might be intentional (e.g., handled differently) or an orphaned field.",
					field)
			}
		})
	}
}

// Helper functions

// navigateToSafeOutputsProperties navigates to the safe-outputs properties section in schema
func navigateToSafeOutputsProperties(schemaDoc map[string]any) (map[string]any, error) {
	props, ok := schemaDoc["properties"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema missing properties")
	}

	safeOutputs, ok := props["safe-outputs"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema missing safe-outputs")
	}

	safeOutputsProps, ok := safeOutputs["properties"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema missing safe-outputs properties")
	}

	return safeOutputsProps, nil
}

// extractDefaultFromSchemaDescription extracts the default value from a field's description
// in the safe-outputs section. Handles both direct properties and oneOf variants.
func extractDefaultFromSchemaDescription(safeOutputsProps map[string]any, fieldPath string) (int, error) {
	// For nested paths like "create-pull-request-review-comment.max"
	parts := strings.Split(fieldPath, ".")

	// Get the top-level field
	fieldName := parts[0]
	field, ok := safeOutputsProps[fieldName]
	if !ok {
		return 0, fmt.Errorf("field %s not found in safe-outputs", fieldName)
	}

	fieldMap, ok := field.(map[string]any)
	if !ok {
		return 0, fmt.Errorf("field %s is not an object", fieldName)
	}

	// Handle oneOf (field can be null or object)
	var targetProperties map[string]any
	if oneOf, hasOneOf := fieldMap["oneOf"].([]any); hasOneOf {
		// Find the object variant
		for _, variant := range oneOf {
			if variantMap, ok := variant.(map[string]any); ok {
				if variantType, hasType := variantMap["type"].(string); hasType && variantType == "object" {
					if props, hasProps := variantMap["properties"].(map[string]any); hasProps {
						targetProperties = props
						break
					}
				}
			}
		}
		if targetProperties == nil {
			return 0, fmt.Errorf("no object variant found in oneOf for %s", fieldName)
		}
	} else if props, hasProps := fieldMap["properties"].(map[string]any); hasProps {
		targetProperties = props
	} else {
		return 0, fmt.Errorf("field %s has no properties", fieldName)
	}

	// If there's a nested path (e.g., "max" in "create-pull-request-review-comment.max")
	if len(parts) > 1 {
		propertyName := parts[1]
		property, ok := targetProperties[propertyName]
		if !ok {
			return 0, fmt.Errorf("property %s not found in %s", propertyName, fieldName)
		}

		propertyMap, ok := property.(map[string]any)
		if !ok {
			return 0, fmt.Errorf("property %s is not an object", propertyName)
		}

		// Extract default from description
		if desc, hasDesc := propertyMap["description"].(string); hasDesc {
			return extractDefaultValueFromDescription(desc)
		}

		return 0, fmt.Errorf("no description found for %s.%s", fieldName, propertyName)
	}

	// For top-level field, check its description
	if desc, hasDesc := fieldMap["description"].(string); hasDesc {
		return extractDefaultValueFromDescription(desc)
	}

	return 0, fmt.Errorf("no description found for %s", fieldName)
}

// extractDefaultValueFromDescription parses a description string to extract the default value
// Example: "Maximum number of review comments to create (default: 1)" -> 1
func extractDefaultValueFromDescription(description string) (int, error) {
	// Look for patterns like "(default: 1)", "(default: 10)", etc.
	re := regexp.MustCompile(`\(default:\s*(\d+)\)`)
	matches := re.FindStringSubmatch(description)

	if len(matches) < 2 {
		return 0, fmt.Errorf("no default value found in description: %s", description)
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("failed to parse default value '%s': %w", matches[1], err)
	}

	return value, nil
}
