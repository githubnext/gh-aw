package parser

import (
	"os"
	"strings"
	"testing"
)

func TestValidateMainWorkflowFrontmatterWithSchema(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "valid frontmatter with all allowed keys",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches": []string{"main"},
					},
					"stop-after": "2024-12-31",
				},
				"permissions":     "read",
				"run-name":        "Test Run",
				"runs-on":         "ubuntu-latest",
				"timeout_minutes": 30,
				"concurrency":     "test",
				"env":             map[string]string{"TEST": "value"},
				"if":              "true",
				"steps":           []string{"step1"},
				"engine":          "claude",
				"tools":           map[string]any{"github": "test"},
				"command":         "test-workflow",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with subset of keys",
			frontmatter: map[string]any{
				"on":     "push",
				"engine": "claude",
			},
			wantErr: false,
		},
		{
			name:        "empty frontmatter",
			frontmatter: map[string]any{},
			wantErr:     false,
		},
		{
			name: "valid engine string format - claude",
			frontmatter: map[string]any{
				"on":     "push",
				"engine": "claude",
			},
			wantErr: false,
		},
		{
			name: "valid engine string format - codex",
			frontmatter: map[string]any{
				"on":     "push",
				"engine": "codex",
			},
			wantErr: false,
		},
		{
			name: "valid engine object format - minimal",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id": "claude",
				},
			},
			wantErr: false,
		},
		{
			name: "valid engine object format - with version",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id":      "claude",
					"version": "beta",
				},
			},
			wantErr: false,
		},
		{
			name: "valid engine object format - with model",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id":    "codex",
					"model": "gpt-4o",
				},
			},
			wantErr: false,
		},
		{
			name: "valid engine object format - complete",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id":      "claude",
					"version": "beta",
					"model":   "claude-3-5-sonnet-20241022",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid engine string format",
			frontmatter: map[string]any{
				"on":     "push",
				"engine": "invalid-engine",
			},
			wantErr:     true,
			errContains: "value must be one of 'claude', 'codex'",
		},
		{
			name: "invalid engine object format - invalid id",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id": "invalid-engine",
				},
			},
			wantErr:     true,
			errContains: "value must be one of 'claude', 'codex'",
		},
		{
			name: "invalid engine object format - missing id",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"version": "beta",
					"model":   "gpt-4o",
				},
			},
			wantErr:     true,
			errContains: "missing property 'id'",
		},
		{
			name: "invalid engine object format - additional properties",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id":      "claude",
					"invalid": "property",
				},
			},
			wantErr:     true,
			errContains: "additional properties",
		},
		{
			name: "invalid frontmatter with unexpected key",
			frontmatter: map[string]any{
				"on":          "push",
				"invalid_key": "value",
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_key' not allowed",
		},
		{
			name: "invalid frontmatter with multiple unexpected keys",
			frontmatter: map[string]any{
				"on":              "push",
				"invalid_key":     "value",
				"another_invalid": "value2",
			},
			wantErr:     true,
			errContains: "additional properties",
		},
		{
			name: "invalid type for timeout_minutes",
			frontmatter: map[string]any{
				"timeout_minutes": "not-a-number",
			},
			wantErr:     true,
			errContains: "got string, want integer",
		},
		{
			name: "valid frontmatter with complex on object",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []map[string]any{
						{"cron": "0 9 * * *"},
					},
					"workflow_dispatch": map[string]any{},
				},
				"engine": "claude",
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with command trigger",
			frontmatter: map[string]any{
				"on": map[string]any{
					"command": map[string]any{
						"name": "test-command",
					},
				},
				"permissions": map[string]any{
					"issues":   "write",
					"contents": "read",
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with complex tools configuration",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"allowed": []string{"create_issue", "update_issue"},
					},
					"claude": map[string]any{
						"allowed": map[string]any{
							"WebFetch": nil,
							"Bash":     []string{"echo:*", "ls"},
						},
					},
					"customTool": map[string]any{
						"mcp": map[string]any{
							"type":    "stdio",
							"command": "my-tool",
						},
						"allowed": []string{"function1", "function2"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with detailed permissions",
			frontmatter: map[string]any{
				"permissions": map[string]any{
					"contents":      "read",
					"issues":        "write",
					"pull-requests": "read",
					"models":        "read",
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with single cache configuration",
			frontmatter: map[string]any{
				"cache": map[string]any{
					"key":          "node-modules-${{ hashFiles('package-lock.json') }}",
					"path":         "node_modules",
					"restore-keys": []string{"node-modules-"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid frontmatter with multiple cache configurations",
			frontmatter: map[string]any{
				"cache": []any{
					map[string]any{
						"key":  "cache1",
						"path": "path1",
					},
					map[string]any{
						"key":                "cache2",
						"path":               []string{"path2", "path3"},
						"restore-keys":       "restore-key",
						"fail-on-cache-miss": true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid cache configuration missing required key",
			frontmatter: map[string]any{
				"cache": map[string]any{
					"path": "node_modules",
				},
			},
			wantErr:     true,
			errContains: "missing property 'key'",
		},
		// Test cases for additional properties validation
		{
			name: "invalid permissions with additional property",
			frontmatter: map[string]any{
				"on": "push",
				"permissions": map[string]any{
					"contents":     "read",
					"invalid_perm": "write",
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_perm' not allowed",
		},
		{
			name: "invalid on trigger with additional properties",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches":     []string{"main"},
						"invalid_prop": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid schedule with additional properties",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []map[string]any{
						{
							"cron":         "0 9 * * *",
							"invalid_prop": "value",
						},
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid workflow_dispatch with additional properties",
			frontmatter: map[string]any{
				"on": map[string]any{
					"workflow_dispatch": map[string]any{
						"inputs": map[string]any{
							"test_input": map[string]any{
								"description": "Test input",
								"type":        "string",
							},
						},
						"invalid_prop": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid concurrency with additional properties",
			frontmatter: map[string]any{
				"concurrency": map[string]any{
					"group":              "test-group",
					"cancel-in-progress": true,
					"invalid_prop":       "value",
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid runs-on object with additional properties",
			frontmatter: map[string]any{
				"runs-on": map[string]any{
					"group":        "test-group",
					"labels":       []string{"ubuntu-latest"},
					"invalid_prop": "value",
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid github tools with additional properties",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"github": map[string]any{
						"allowed":      []string{"create_issue"},
						"invalid_prop": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid claude tools with additional properties",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"claude": map[string]any{
						"allowed":      []string{"WebFetch"},
						"invalid_prop": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid custom tool with additional properties",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"customTool": map[string]any{
						"allowed": []string{"function1"},
						"mcp": map[string]any{
							"type":    "stdio",
							"command": "my-tool",
						},
						"invalid_prop": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid claude configuration with additional properties",
			frontmatter: map[string]any{
				"claude": map[string]any{
					"model":        "claude-3",
					"invalid_prop": "value",
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "invalid safe-outputs configuration with additional properties",
			frontmatter: map[string]any{
				"safe-outputs": map[string]any{
					"create-issue": map[string]any{
						"title-prefix": "[ai] ",
						"invalid_prop": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "valid new GitHub Actions properties - permissions with new properties",
			frontmatter: map[string]any{
				"on": "push",
				"permissions": map[string]any{
					"contents":            "read",
					"attestations":        "write",
					"id-token":            "write",
					"packages":            "read",
					"pages":               "write",
					"repository-projects": "none",
				},
			},
			wantErr: false,
		},
		{
			name: "valid GitHub Actions defaults property",
			frontmatter: map[string]any{
				"on": "push",
				"defaults": map[string]any{
					"run": map[string]any{
						"shell":             "bash",
						"working-directory": "/app",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid defaults with additional properties",
			frontmatter: map[string]any{
				"defaults": map[string]any{
					"run": map[string]any{
						"shell":        "bash",
						"invalid_prop": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "valid strict mode true",
			frontmatter: map[string]any{
				"on":     "push",
				"strict": true,
			},
			wantErr: false,
		},
		{
			name: "valid strict mode false",
			frontmatter: map[string]any{
				"on":     "push",
				"strict": false,
			},
			wantErr: false,
		},
		{
			name: "invalid strict mode as string",
			frontmatter: map[string]any{
				"on":     "push",
				"strict": "true",
			},
			wantErr:     true,
			errContains: "want boolean",
		},
		{
			name: "valid claude engine with network permissions",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id": "claude",
					"permissions": map[string]any{
						"network": map[string]any{
							"allowed": []string{"example.com", "*.trusted.com"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid codex engine with permissions",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id": "codex",
					"permissions": map[string]any{
						"network": map[string]any{
							"allowed": []string{"example.com"},
						},
					},
				},
			},
			wantErr:     true,
			errContains: "engine permissions are not supported for codex engine",
		},
		{
			name: "valid codex engine without permissions",
			frontmatter: map[string]any{
				"on": "push",
				"engine": map[string]any{
					"id":    "codex",
					"model": "gpt-4o",
				},
			},
			wantErr: false,
		},
		{
			name: "valid codex string engine (no permissions possible)",
			frontmatter: map[string]any{
				"on":     "push",
				"engine": "codex",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMainWorkflowFrontmatterWithSchema(tt.frontmatter)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateMainWorkflowFrontmatterWithSchema() expected error, got nil")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ValidateMainWorkflowFrontmatterWithSchema() error = %v", err)
				return
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateMainWorkflowFrontmatterWithSchema() error = %v, expected to contain %v", err, tt.errContains)
				}
			}
		})
	}
}

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
						"allowed": []string{"list_issues", "get_issue"},
					},
					"claude": map[string]any{
						"allowed": map[string]any{
							"Edit":     nil,
							"WebFetch": nil,
						},
					},
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
	tempFile := "/tmp/test_schema_validation.md"
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

	// The error message should contain the actual useful error description
	if !strings.Contains(errorMsg, "additional properties 'timeout_minu tes' not allowed") {
		t.Errorf("Error message should contain the validation error, got: %s", errorMsg)
	}

	// The error message should be formatted with location information
	if !strings.Contains(errorMsg, tempFile) {
		t.Errorf("Error message should contain file path, got: %s", errorMsg)
	}
}
