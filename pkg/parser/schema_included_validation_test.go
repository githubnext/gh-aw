package parser

import (
	"os"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestValidateIncludedFileFrontmatterWithSchema(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "valid frontmatter with tools only",
			frontmatter: map[string]any{
				"tools": map[string]any{"github": "test"},
			},
			wantErr: false,
		},
		{
			name:        "empty frontmatter",
			frontmatter: map[string]any{},
			wantErr:     false,
		},
		{
			name: "invalid frontmatter with on trigger",
			frontmatter: map[string]any{
				"on":    "push",
				"tools": map[string]any{"github": "test"},
			},
			wantErr:     true,
			errContains: "additional properties 'on' not allowed",
		},
		{
			name: "invalid frontmatter with multiple unexpected keys",
			frontmatter: map[string]any{
				"on":          "push",
				"permissions": "read",
				"tools":       map[string]any{"github": "test"},
			},
			wantErr:     true,
			errContains: "additional properties",
		},
		{
			name: "invalid frontmatter with only unexpected keys",
			frontmatter: map[string]any{
				"on":          "push",
				"permissions": "read",
			},
			wantErr:     true,
			errContains: "additional properties",
		},
		{
			name: "valid frontmatter with complex tools object",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []string{"list_issues", "issue_read"},
					},
					"bash": []string{"echo", "ls"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with bash as boolean true",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"bash": true,
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with bash as boolean false",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"bash": false,
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with bash as null",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"bash": nil,
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with custom MCP tool",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"myTool": map[string]any{
						"mcp": map[string]any{
							"type":    "http",
							"url":     "https://api.contoso.com",
							"headers": map[string]any{"Authorization": "Bearer token"},
						},
						"allowed": []string{"api_call1", "api_call2"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with HTTP MCP tool with underscored headers",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"datadog": map[string]any{
						"type": "http",
						"url":  "https://mcp.datadoghq.com/api/unstable/mcp-server/mcp",
						"headers": map[string]any{
							"DD_API_KEY":         "test-key",
							"DD_APPLICATION_KEY": "test-app",
							"DD_SITE":            "datadoghq.com",
						},
						"allowed": []string{"get-monitors", "get-monitor"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with cache-memory as boolean true",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"cache-memory": true,
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with cache-memory as boolean false",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"cache-memory": false,
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with cache-memory as nil",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"cache-memory": nil,
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with cache-memory as object with key",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"cache-memory": map[string]any{
						"key": "custom-memory-${{ github.workflow }}",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with cache-memory with all valid options",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"cache-memory": map[string]any{
						"key":            "custom-key",
						"retention-days": 30,
						"description":    "Test cache description",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid cache-memory with invalid retention-days (too low)",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"cache-memory": map[string]any{
						"retention-days": 0,
					},
				},
			},
			wantErr:     true,
			errContains: "got 0, want 1",
		},
		{
			name: "invalid cache-memory with invalid retention-days (too high)",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"cache-memory": map[string]any{
						"retention-days": 91,
					},
				},
			},
			wantErr:     true,
			errContains: "got 91, want 90",
		},
		{
			name: "invalid cache-memory with unsupported docker-image field",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"cache-memory": map[string]any{
						"docker-image": "custom/memory:latest",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'docker-image' not allowed",
		},
		{
			name: "invalid cache-memory with additional property",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"cache-memory": map[string]any{
						"key":            "custom-key",
						"invalid_option": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_option' not allowed",
		},
		{
			name: "valid: included file with 25 inputs (max allowed)",
			frontmatter: map[string]any{
				"inputs": map[string]any{
					"input1":  map[string]any{"description": "Input 1", "type": "string"},
					"input2":  map[string]any{"description": "Input 2", "type": "string"},
					"input3":  map[string]any{"description": "Input 3", "type": "string"},
					"input4":  map[string]any{"description": "Input 4", "type": "string"},
					"input5":  map[string]any{"description": "Input 5", "type": "string"},
					"input6":  map[string]any{"description": "Input 6", "type": "string"},
					"input7":  map[string]any{"description": "Input 7", "type": "string"},
					"input8":  map[string]any{"description": "Input 8", "type": "string"},
					"input9":  map[string]any{"description": "Input 9", "type": "string"},
					"input10": map[string]any{"description": "Input 10", "type": "string"},
					"input11": map[string]any{"description": "Input 11", "type": "string"},
					"input12": map[string]any{"description": "Input 12", "type": "string"},
					"input13": map[string]any{"description": "Input 13", "type": "string"},
					"input14": map[string]any{"description": "Input 14", "type": "string"},
					"input15": map[string]any{"description": "Input 15", "type": "string"},
					"input16": map[string]any{"description": "Input 16", "type": "string"},
					"input17": map[string]any{"description": "Input 17", "type": "string"},
					"input18": map[string]any{"description": "Input 18", "type": "string"},
					"input19": map[string]any{"description": "Input 19", "type": "string"},
					"input20": map[string]any{"description": "Input 20", "type": "string"},
					"input21": map[string]any{"description": "Input 21", "type": "string"},
					"input22": map[string]any{"description": "Input 22", "type": "string"},
					"input23": map[string]any{"description": "Input 23", "type": "string"},
					"input24": map[string]any{"description": "Input 24", "type": "string"},
					"input25": map[string]any{"description": "Input 25", "type": "string"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid: included file with 26 inputs (exceeds max)",
			frontmatter: map[string]any{
				"inputs": map[string]any{
					"input1":  map[string]any{"description": "Input 1", "type": "string"},
					"input2":  map[string]any{"description": "Input 2", "type": "string"},
					"input3":  map[string]any{"description": "Input 3", "type": "string"},
					"input4":  map[string]any{"description": "Input 4", "type": "string"},
					"input5":  map[string]any{"description": "Input 5", "type": "string"},
					"input6":  map[string]any{"description": "Input 6", "type": "string"},
					"input7":  map[string]any{"description": "Input 7", "type": "string"},
					"input8":  map[string]any{"description": "Input 8", "type": "string"},
					"input9":  map[string]any{"description": "Input 9", "type": "string"},
					"input10": map[string]any{"description": "Input 10", "type": "string"},
					"input11": map[string]any{"description": "Input 11", "type": "string"},
					"input12": map[string]any{"description": "Input 12", "type": "string"},
					"input13": map[string]any{"description": "Input 13", "type": "string"},
					"input14": map[string]any{"description": "Input 14", "type": "string"},
					"input15": map[string]any{"description": "Input 15", "type": "string"},
					"input16": map[string]any{"description": "Input 16", "type": "string"},
					"input17": map[string]any{"description": "Input 17", "type": "string"},
					"input18": map[string]any{"description": "Input 18", "type": "string"},
					"input19": map[string]any{"description": "Input 19", "type": "string"},
					"input20": map[string]any{"description": "Input 20", "type": "string"},
					"input21": map[string]any{"description": "Input 21", "type": "string"},
					"input22": map[string]any{"description": "Input 22", "type": "string"},
					"input23": map[string]any{"description": "Input 23", "type": "string"},
					"input24": map[string]any{"description": "Input 24", "type": "string"},
					"input25": map[string]any{"description": "Input 25", "type": "string"},
					"input26": map[string]any{"description": "Input 26", "type": "string"},
				},
			},
			wantErr:     true,
			errContains: "maxProperties",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIncludedFileFrontmatterWithSchema(tt.frontmatter)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateIncludedFileFrontmatterWithSchema() expected error, got nil")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ValidateIncludedFileFrontmatterWithSchema() error = %v", err)
				return
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateIncludedFileFrontmatterWithSchema() error = %v, expected to contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestValidateWithSchema(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		schema      string
		context     string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid data with simple schema",
			frontmatter: map[string]any{
				"name": "test",
			},
			schema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				},
				"additionalProperties": false
			}`,
			context: "test context",
			wantErr: false,
		},
		{
			name: "invalid data with additional property",
			frontmatter: map[string]any{
				"name":    "test",
				"invalid": "value",
			},
			schema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				},
				"additionalProperties": false
			}`,
			context:     "test context",
			wantErr:     true,
			errContains: "additional properties 'invalid' not allowed",
		},
		{
			name: "invalid schema JSON",
			frontmatter: map[string]any{
				"name": "test",
			},
			schema:      `invalid json`,
			context:     "test context",
			wantErr:     true,
			errContains: "schema validation error for test context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWithSchema(tt.frontmatter, tt.schema, tt.context)

			if tt.wantErr && err == nil {
				t.Errorf("validateWithSchema() expected error, got nil")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("validateWithSchema() error = %v", err)
				return
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateWithSchema() error = %v, expected to contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestValidateWithSchemaAndLocation_CleanedErrorMessage(t *testing.T) {
	// Test that error messages are properly cleaned of unhelpful jsonschema prefixes
	frontmatter := map[string]any{
		"on":               "push",
		"timeout_minu tes": 10, // Invalid property name with space
	}

	// Create a temporary test file
	tempFile := "/tmp/gh-aw/test_schema_validation.md"
	// Ensure the directory exists
	if err := os.MkdirAll("/tmp/gh-aw", 0755); err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	err := os.WriteFile(tempFile, []byte(`---
on: push
timeout_minu tes: 10
---

# Test workflow`), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile)

	err = ValidateMainWorkflowFrontmatterWithSchemaAndLocation(frontmatter, tempFile)

	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	errorMsg := err.Error()

	// The error message should NOT contain the unhelpful jsonschema prefixes
	if strings.Contains(errorMsg, "jsonschema validation failed") {
		t.Errorf("Error message should not contain 'jsonschema validation failed' prefix, got: %s", errorMsg)
	}

	if strings.Contains(errorMsg, "- at '': ") {
		t.Errorf("Error message should not contain '- at '':' prefix, got: %s", errorMsg)
	}

	// The error message should contain the friendly rewritten error description
	if !strings.Contains(errorMsg, "Unknown property: timeout_minu tes") {
		t.Errorf("Error message should contain the validation error, got: %s", errorMsg)
	}

	// The error message should be formatted with location information
	if !strings.Contains(errorMsg, tempFile) {
		t.Errorf("Error message should contain file path, got: %s", errorMsg)
	}
}

func TestFilterIgnoredFields(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    map[string]any
	}{
		{
			name:        "nil frontmatter",
			frontmatter: nil,
			expected:    nil,
		},
		{
			name:        "empty frontmatter",
			frontmatter: map[string]any{},
			expected:    map[string]any{},
		},
		{
			name: "frontmatter with description - no longer filtered",
			frontmatter: map[string]any{
				"description": "This is a test workflow",
				"on":          "push",
			},
			expected: map[string]any{
				"description": "This is a test workflow",
				"on":          "push",
			},
		},
		{
			name: "frontmatter with applyTo - no longer filtered",
			frontmatter: map[string]any{
				"applyTo": "some-value",
				"on":      "push",
			},
			expected: map[string]any{
				"applyTo": "some-value",
				"on":      "push",
			},
		},
		{
			name: "frontmatter with both description and applyTo - no longer filtered",
			frontmatter: map[string]any{
				"description": "This is a test workflow",
				"applyTo":     "some-value",
				"on":          "push",
				"engine":      "claude",
			},
			expected: map[string]any{
				"description": "This is a test workflow",
				"applyTo":     "some-value",
				"on":          "push",
				"engine":      "claude",
			},
		},
		{
			name: "frontmatter with only valid fields",
			frontmatter: map[string]any{
				"on":     "push",
				"engine": "claude",
			},
			expected: map[string]any{
				"on":     "push",
				"engine": "claude",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterIgnoredFields(tt.frontmatter)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d fields, got %d fields", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, ok := result[key]; !ok {
					t.Errorf("Expected field %q not found in result", key)
				} else {
					// For simple types, compare directly
					// For maps, we need to compare keys (simple check for this test)
					switch v := expectedValue.(type) {
					case map[string]any:
						if actualMap, ok := actualValue.(map[string]any); !ok {
							t.Errorf("Field %q: expected map, got %T", key, actualValue)
						} else if len(actualMap) != len(v) {
							t.Errorf("Field %q: expected map with %d keys, got %d keys", key, len(v), len(actualMap))
						}
					default:
						if actualValue != expectedValue {
							t.Errorf("Field %q: expected %v, got %v", key, expectedValue, actualValue)
						}
					}
				}
			}

			// Check that ignored fields are not present
			for _, ignoredField := range constants.IgnoredFrontmatterFields {
				if _, ok := result[ignoredField]; ok {
					t.Errorf("Ignored field %q should not be present in result", ignoredField)
				}
			}
		})
	}
}

func TestValidateMainWorkflowWithIgnoredFields(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "valid frontmatter with description field - now properly validated",
			frontmatter: map[string]any{
				"on":          "push",
				"description": "This is a test workflow description",
				"engine":      "claude",
			},
			wantErr: false,
		},
		{
			name: "invalid frontmatter with applyTo field - not allowed in main workflow",
			frontmatter: map[string]any{
				"on":      "push",
				"applyTo": "some-target",
				"engine":  "claude",
			},
			wantErr:     true,
			errContains: "applyTo",
		},
		{
			name: "valid frontmatter with description - now properly validated",
			frontmatter: map[string]any{
				"on":          "push",
				"description": "Test workflow",
				"engine":      "claude",
			},
			wantErr: false,
		},
		{
			name: "invalid frontmatter with ignored fields - other validation should still work",
			frontmatter: map[string]any{
				"on":            "push",
				"description":   "Test workflow",
				"applyTo":       "some-target",
				"invalid_field": "should-fail",
			},
			wantErr:     true,
			errContains: "invalid_field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMainWorkflowFrontmatterWithSchema(tt.frontmatter)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMainWorkflowFrontmatterWithSchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error message should contain %q, got: %v", tt.errContains, err)
				}
			}
		})
	}
}

func TestValidateIncludedFileWithIgnoredFields(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "valid included file with description field - now properly validated",
			frontmatter: map[string]any{
				"description": "This is a shared config",
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []string{"get_repository"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid included file with applyTo field - now properly validated",
			frontmatter: map[string]any{
				"applyTo": "some-target",
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []string{"get_repository"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid included file with applyTo array - now properly validated",
			frontmatter: map[string]any{
				"applyTo": []string{"**/*.py", "**/*.pyw"},
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []string{"get_repository"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid included file with both description and applyTo - now properly validated",
			frontmatter: map[string]any{
				"description": "Shared config",
				"applyTo":     "some-target",
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []string{"get_repository"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid included file with wrong type for applyTo - should fail",
			frontmatter: map[string]any{
				"applyTo": 123, // number instead of string or array
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []string{"get_repository"},
					},
				},
			},
			wantErr:     true,
			errContains: "applyTo",
		},
		{
			name: "invalid included file with wrong type for description - should fail",
			frontmatter: map[string]any{
				"description": 123, // number instead of string
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []string{"get_repository"},
					},
				},
			},
			wantErr:     true,
			errContains: "description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIncludedFileFrontmatterWithSchema(tt.frontmatter)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIncludedFileFrontmatterWithSchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error message should contain %q, got: %v", tt.errContains, err)
				}
			}
		})
	}
}

