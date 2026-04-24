//go:build !integration

package parser

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

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

// TestValidateMCPConfigWithSchema tests the ValidateMCPConfigWithSchema function
// which validates a single MCP server configuration against the MCP config JSON schema.
func TestValidateMCPConfigWithSchema(t *testing.T) {
	tests := []struct {
		name        string
		mcpConfig   map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "valid stdio config with command",
			mcpConfig: map[string]any{
				"type":    "stdio",
				"command": "npx",
				"args":    []any{"-y", "@modelcontextprotocol/server-filesystem"},
			},
			wantErr: false,
		},
		{
			name: "valid stdio config with container",
			mcpConfig: map[string]any{
				"type":      "stdio",
				"container": "docker.io/mcp/brave-search",
				"env": map[string]any{
					"BRAVE_API_KEY": "secret",
				},
			},
			wantErr: false,
		},
		{
			name: "valid http config",
			mcpConfig: map[string]any{
				"type": "http",
				"url":  "https://api.example.com/mcp",
				"headers": map[string]any{
					"Authorization": "Bearer token",
				},
			},
			wantErr: false,
		},
		{
			name: "valid config inferred from url requires explicit type in schema",
			mcpConfig: map[string]any{
				"type": "http",
				"url":  "http://localhost:8765",
			},
			wantErr: false,
		},
		{
			name:        "empty config fails anyOf - missing type, url, command, and container",
			mcpConfig:   map[string]any{},
			wantErr:     true,
			errContains: "missing property",
		},
		{
			name: "invalid container pattern rejected by schema",
			mcpConfig: map[string]any{
				"container": "INVALID CONTAINER NAME WITH SPACES",
			},
			wantErr:     true,
			errContains: "jsonschema validation failed",
		},
		{
			name: "invalid env key pattern rejected by schema",
			mcpConfig: map[string]any{
				"type":    "stdio",
				"command": "node",
				"env": map[string]any{
					"lowercase-key": "value",
				},
			},
			wantErr:     true,
			errContains: "jsonschema validation failed",
		},
		{
			name: "invalid mounts item pattern rejected by schema",
			mcpConfig: map[string]any{
				"type":      "stdio",
				"container": "mcp/server",
				"mounts":    []any{"invalid-mount-format"},
			},
			wantErr:     true,
			errContains: "jsonschema validation failed",
		},
		{
			name: "additional property rejected by schema",
			mcpConfig: map[string]any{
				"type":          "stdio",
				"command":       "node",
				"unknown-field": "value",
			},
			wantErr:     true,
			errContains: "jsonschema validation failed",
		},
		{
			name: "valid local type (alias for stdio)",
			mcpConfig: map[string]any{
				"type":    "local",
				"command": "node",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMCPConfigWithSchema(tt.mcpConfig)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateMCPConfigWithSchema() expected error, got nil")
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ValidateMCPConfigWithSchema() unexpected error = %v", err)
				return
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateMCPConfigWithSchema() error = %v, expected to contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestValidateMainWorkflowFrontmatterWithSchemaAndLocation_WorkflowDispatchNumberInputType(t *testing.T) {
	t.Parallel()

	frontmatter := map[string]any{
		"on": map[string]any{
			"workflow_dispatch": map[string]any{
				"inputs": map[string]any{
					"max_retries": map[string]any{
						"description": "Maximum retries",
						"type":        "number",
						"default":     3,
						"required":    false,
					},
				},
			},
		},
		"engine": "copilot",
	}

	err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(frontmatter, "/tmp/gh-aw/workflow_dispatch_number_test.md")
	if err != nil {
		t.Fatalf("expected workflow_dispatch number input type to validate, got: %v", err)
	}
}

func TestValidateMainWorkflowFrontmatterWithSchemaAndLocation_EngineDriverPattern(t *testing.T) {
	t.Parallel()

	validFrontmatter := map[string]any{
		"on": "push",
		"engine": map[string]any{
			"id":     "claude",
			"driver": "custom_driver.cjs",
		},
	}

	err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(validFrontmatter, "/tmp/gh-aw/engine-driver-valid-pattern-test.md")
	if err != nil {
		t.Fatalf("expected valid engine.driver pattern to pass schema validation, got: %v", err)
	}

	invalidFrontmatter := map[string]any{
		"on": "push",
		"engine": map[string]any{
			"id":     "claude",
			"driver": "../driver.cjs",
		},
	}

	err = ValidateMainWorkflowFrontmatterWithSchemaAndLocation(invalidFrontmatter, "/tmp/gh-aw/engine-driver-invalid-pattern-test.md")
	if err == nil {
		t.Fatal("expected invalid engine.driver pattern to fail schema validation")
	}

	invalidFlagLikeFrontmatter := map[string]any{
		"on": "push",
		"engine": map[string]any{
			"id":     "claude",
			"driver": "-driver.cjs",
		},
	}

	err = ValidateMainWorkflowFrontmatterWithSchemaAndLocation(invalidFlagLikeFrontmatter, "/tmp/gh-aw/engine-driver-invalid-flaglike-pattern-test.md")
	if err == nil {
		t.Fatal("expected flag-like engine.driver pattern to fail schema validation")
	}
}

func TestMainWorkflowSchema_WorkflowDispatchNumberTypeDocumentation(t *testing.T) {
	t.Parallel()

	schemaPath := "schemas/main_workflow_schema.json"
	schemaContent, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("failed to read schema: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(schemaContent, &schema); err != nil {
		t.Fatalf("failed to parse schema json: %v", err)
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema properties section not found")
	}
	onField, ok := properties["on"].(map[string]any)
	if !ok {
		t.Fatal("'on' field not found in schema")
	}

	onOneOf, ok := onField["oneOf"].([]any)
	if !ok {
		t.Fatal("'on.oneOf' not found in schema")
	}

	var workflowDispatchInputType map[string]any
	for _, onEntry := range onOneOf {
		onEntryMap, ok := onEntry.(map[string]any)
		if !ok {
			continue
		}
		onProps, ok := onEntryMap["properties"].(map[string]any)
		if !ok {
			continue
		}
		eventsConfig, ok := onProps["workflow_dispatch"].(map[string]any)
		if !ok {
			continue
		}
		eventsOneOf, ok := eventsConfig["oneOf"].([]any)
		if !ok {
			continue
		}

		for _, eventEntry := range eventsOneOf {
			eventEntryMap, ok := eventEntry.(map[string]any)
			if !ok {
				continue
			}
			eventProps, ok := eventEntryMap["properties"].(map[string]any)
			if !ok {
				continue
			}
			inputsField, ok := eventProps["inputs"].(map[string]any)
			if !ok {
				continue
			}
			inputDefs, ok := inputsField["additionalProperties"].(map[string]any)
			if !ok {
				continue
			}
			inputDefProps, ok := inputDefs["properties"].(map[string]any)
			if !ok {
				continue
			}
			typeField, ok := inputDefProps["type"].(map[string]any)
			if !ok {
				t.Fatal("'on.workflow_dispatch.inputs.<id>.type' field missing")
			}
			workflowDispatchInputType = typeField
			break
		}
	}

	if workflowDispatchInputType == nil {
		t.Fatal("workflow_dispatch input type schema not found")
	}

	enumVals, ok := workflowDispatchInputType["enum"].([]any)
	if !ok {
		t.Fatal("workflow_dispatch input type enum not found")
	}
	hasNumber := false
	for _, val := range enumVals {
		if val == "number" {
			hasNumber = true
			break
		}
	}
	if !hasNumber {
		t.Fatalf("workflow_dispatch input type enum should include 'number', got: %v", enumVals)
	}

	typeDescription, ok := workflowDispatchInputType["description"].(string)
	if !ok {
		t.Fatal("workflow_dispatch input type description not found")
	}
	if !strings.Contains(typeDescription, "number") {
		t.Fatalf("workflow_dispatch input type description should mention 'number', got: %q", typeDescription)
	}
}

func TestMainWorkflowSchema_CreatePullRequestAllowedBaseBranches(t *testing.T) {
	t.Parallel()

	schemaPath := "schemas/main_workflow_schema.json"
	schemaContent, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("failed to read schema: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(schemaContent, &schema); err != nil {
		t.Fatalf("failed to parse schema json: %v", err)
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema properties section not found")
	}

	safeOutputs, ok := properties["safe-outputs"].(map[string]any)
	if !ok {
		t.Fatal("'safe-outputs' field not found in schema")
	}

	safeOutputsProperties, ok := safeOutputs["properties"].(map[string]any)
	if !ok {
		t.Fatal("'safe-outputs.properties' not found in schema")
	}

	createPullRequest, ok := safeOutputsProperties["create-pull-request"].(map[string]any)
	if !ok {
		t.Fatal("'safe-outputs.create-pull-request' not found in schema")
	}

	createPullRequestOneOf, ok := createPullRequest["oneOf"].([]any)
	if !ok {
		t.Fatal("'safe-outputs.create-pull-request.oneOf' not found in schema")
	}

	var createPullRequestProperties map[string]any
	for _, candidate := range createPullRequestOneOf {
		candidateMap, ok := candidate.(map[string]any)
		if !ok {
			continue
		}

		properties, ok := candidateMap["properties"].(map[string]any)
		if !ok {
			continue
		}

		createPullRequestProperties = properties
		break
	}
	if createPullRequestProperties == nil {
		t.Fatal("'safe-outputs.create-pull-request' object schema with properties not found")
	}

	allowedBaseBranches, ok := createPullRequestProperties["allowed-base-branches"].(map[string]any)
	if !ok {
		t.Fatal("'allowed-base-branches' not found under safe-outputs.create-pull-request")
	}

	if gotType, _ := allowedBaseBranches["type"].(string); gotType != "array" {
		t.Fatalf("'allowed-base-branches' should be type array, got: %v", allowedBaseBranches["type"])
	}

	items, ok := allowedBaseBranches["items"].(map[string]any)
	if !ok {
		t.Fatal("'allowed-base-branches.items' not found in schema")
	}

	if gotItemType, _ := items["type"].(string); gotItemType != "string" {
		t.Fatalf("'allowed-base-branches.items' should be type string, got: %v", items["type"])
	}
}

func TestGetSafeOutputTypeKeys(t *testing.T) {
	keys, err := GetSafeOutputTypeKeys()
	if err != nil {
		t.Fatalf("GetSafeOutputTypeKeys() returned error: %v", err)
	}

	// Should return multiple keys
	if len(keys) == 0 {
		t.Error("GetSafeOutputTypeKeys() returned empty list")
	}

	// Should include known safe output types
	expectedKeys := []string{
		"create-issue",
		"add-comment",
		"create-discussion",
		"create-pull-request",
		"update-issue",
	}

	keySet := make(map[string]bool)
	for _, key := range keys {
		keySet[key] = true
	}

	for _, expected := range expectedKeys {
		if !keySet[expected] {
			t.Errorf("GetSafeOutputTypeKeys() missing expected key: %s", expected)
		}
	}

	// Should NOT include meta-configuration fields
	metaFields := []string{
		"allowed-domains",
		"staged",
		"env",
		"github-token",
		"github-app",
		"max-patch-size",
		"jobs",
		"runs-on",
		"messages",
		"needs",
	}

	for _, meta := range metaFields {
		if keySet[meta] {
			t.Errorf("GetSafeOutputTypeKeys() should not include meta field: %s", meta)
		}
	}

	// Keys should be sorted
	for i := 1; i < len(keys); i++ {
		if keys[i-1] > keys[i] {
			t.Errorf("GetSafeOutputTypeKeys() keys are not sorted: %s > %s", keys[i-1], keys[i])
		}
	}
}

// TestNormalizeForJSONSchema verifies that normalizeForJSONSchema correctly converts
// YAML-native integer types to float64 while leaving other types unchanged.
func TestNormalizeForJSONSchema(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any
	}{
		// Integer type conversions
		{name: "int", input: int(42), expected: float64(42)},
		{name: "int8", input: int8(8), expected: float64(8)},
		{name: "int16", input: int16(16), expected: float64(16)},
		{name: "int32", input: int32(32), expected: float64(32)},
		{name: "int64", input: int64(64), expected: float64(64)},
		{name: "int64 negative", input: int64(-5), expected: float64(-5)},

		// Unsigned integer type conversions
		{name: "uint", input: uint(42), expected: float64(42)},
		{name: "uint8", input: uint8(8), expected: float64(8)},
		{name: "uint16", input: uint16(16), expected: float64(16)},
		{name: "uint32", input: uint32(32), expected: float64(32)},
		{name: "uint64", input: uint64(64), expected: float64(64)},
		{name: "uint64 large", input: uint64(9999999999999), expected: float64(9999999999999)},

		// Float type conversions
		{name: "float32", input: float32(3.14), expected: float64(float32(3.14))},

		// Pass-through types
		{name: "float64 passthrough", input: float64(2.718), expected: float64(2.718)},
		{name: "string passthrough", input: "hello", expected: "hello"},
		{name: "bool true passthrough", input: true, expected: true},
		{name: "bool false passthrough", input: false, expected: false},
		{name: "nil passthrough", input: nil, expected: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeForJSONSchema(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeForJSONSchema(%T(%v)) = %T(%v), want %T(%v)",
					tt.input, tt.input, result, result, tt.expected, tt.expected)
			}
		})
	}
}

func TestValidateMainWorkflowFrontmatterWithSchemaAndLocation_GitHubAppClientID(t *testing.T) {
	frontmatter := map[string]any{
		"name": "Client ID validation",
		"on": map[string]any{
			"issues": map[string]any{
				"types": []any{"opened"},
			},
		},
		"github-app": map[string]any{
			"client-id":   "${{ vars.APP_ID }}",
			"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
		},
	}

	err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(frontmatter, "/tmp/gh-aw/client-id-schema-test.md")
	if err != nil {
		t.Fatalf("expected client-id in github-app to pass schema validation, got: %v", err)
	}
}

// TestNormalizeForJSONSchema_NestedMap verifies recursive normalization of maps.
func TestNormalizeForJSONSchema_NestedMap(t *testing.T) {
	input := map[string]any{
		"name":    "test",
		"count":   uint64(42),
		"offset":  int64(-3),
		"enabled": true,
		"nested": map[string]any{
			"port":  uint64(8080),
			"label": "inner",
		},
	}

	result := normalizeForJSONSchema(input)
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	if resultMap["name"] != "test" {
		t.Errorf("name: got %v, want test", resultMap["name"])
	}
	if resultMap["count"] != float64(42) {
		t.Errorf("count: got %T(%v), want float64(42)", resultMap["count"], resultMap["count"])
	}
	if resultMap["offset"] != float64(-3) {
		t.Errorf("offset: got %T(%v), want float64(-3)", resultMap["offset"], resultMap["offset"])
	}
	if resultMap["enabled"] != true {
		t.Errorf("enabled: got %v, want true", resultMap["enabled"])
	}

	nestedMap, ok := resultMap["nested"].(map[string]any)
	if !ok {
		t.Fatalf("nested: expected map[string]any, got %T", resultMap["nested"])
	}
	if nestedMap["port"] != float64(8080) {
		t.Errorf("nested.port: got %T(%v), want float64(8080)", nestedMap["port"], nestedMap["port"])
	}
	if nestedMap["label"] != "inner" {
		t.Errorf("nested.label: got %v, want inner", nestedMap["label"])
	}

	// Verify the original input is NOT mutated
	if input["count"] != uint64(42) {
		t.Errorf("original input mutated: count is %T(%v), expected uint64(42)", input["count"], input["count"])
	}
}

// TestNormalizeForJSONSchema_Slice verifies recursive normalization of slices.
func TestNormalizeForJSONSchema_Slice(t *testing.T) {
	input := []any{uint64(1), "two", int64(-3), true, nil, float64(4.5)}

	result := normalizeForJSONSchema(input)
	resultSlice, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}

	expected := []any{float64(1), "two", float64(-3), true, nil, float64(4.5)}
	if len(resultSlice) != len(expected) {
		t.Fatalf("length mismatch: got %d, want %d", len(resultSlice), len(expected))
	}
	for i, want := range expected {
		if resultSlice[i] != want {
			t.Errorf("[%d]: got %T(%v), want %T(%v)", i, resultSlice[i], resultSlice[i], want, want)
		}
	}
}

// TestNormalizeForJSONSchema_TypedSlice verifies that typed slices (e.g. []string)
// are converted to []any, since goccy/go-yaml may produce typed slices that the
// JSON schema validator does not recognize.
func TestNormalizeForJSONSchema_TypedSlice(t *testing.T) {
	input := []string{"a", "b", "c"}

	result := normalizeForJSONSchema(input)
	resultSlice, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}

	if len(resultSlice) != 3 {
		t.Fatalf("length mismatch: got %d, want 3", len(resultSlice))
	}
	for i, want := range []string{"a", "b", "c"} {
		if resultSlice[i] != want {
			t.Errorf("[%d]: got %T(%v), want %T(%v)", i, resultSlice[i], resultSlice[i], want, want)
		}
	}
}

// TestValidateWithSchema_YAMLTypedSlice verifies that validateWithSchema accepts
// typed slices (e.g. []string) that goccy/go-yaml produces for array fields.
func TestValidateWithSchema_YAMLTypedSlice(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"tags": {"type": "array", "items": {"type": "string"}},
			"name": {"type": "string"}
		},
		"additionalProperties": false
	}`

	frontmatter := map[string]any{
		"tags": []string{"v1", "v2"},
		"name": "test",
	}

	err := validateWithSchema(frontmatter, schema, "yaml typed slice")
	if err != nil {
		t.Errorf("validateWithSchema should accept typed slices, got: %v", err)
	}
}

// TestValidateMainWorkflowFrontmatterWithSchemaAndLocation_ProtectedFilesObjectForm
// verifies that the protected-files field on create-pull-request and
// push-to-pull-request-branch accepts the documented object form
// {policy, exclude} in addition to the plain string enum.
//
// This is a regression test for the bug where the schema only accepted
// "string or null" for protected-files, rejecting object-form configurations
// with "expected string or null, got object".
func TestValidateMainWorkflowFrontmatterWithSchemaAndLocation_ProtectedFilesObjectForm(t *testing.T) {
	t.Parallel()

	baseFrontmatter := func(safeOutputs map[string]any) map[string]any {
		return map[string]any{
			"on":           map[string]any{"issues": map[string]any{"types": []any{"opened"}}},
			"engine":       "copilot",
			"safe-outputs": safeOutputs,
		}
	}

	tests := []struct {
		name        string
		safeOutputs map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "create-pull-request: string form passes",
			safeOutputs: map[string]any{
				"create-pull-request": map[string]any{
					"protected-files": "fallback-to-issue",
				},
			},
			wantErr: false,
		},
		{
			name: "create-pull-request: object form with policy and exclude passes",
			safeOutputs: map[string]any{
				"create-pull-request": map[string]any{
					"protected-files": map[string]any{
						"policy":  "fallback-to-issue",
						"exclude": []any{".claude/", ".github/instructions/"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "create-pull-request: object form with only exclude passes",
			safeOutputs: map[string]any{
				"create-pull-request": map[string]any{
					"protected-files": map[string]any{
						"exclude": []any{"AGENTS.md"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "create-pull-request: object form with only policy passes",
			safeOutputs: map[string]any{
				"create-pull-request": map[string]any{
					"protected-files": map[string]any{
						"policy": "allowed",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "create-pull-request: object form with invalid extra field is rejected",
			safeOutputs: map[string]any{
				"create-pull-request": map[string]any{
					"protected-files": map[string]any{
						"policy":       "blocked",
						"unknown-prop": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "unknown-prop",
		},
		{
			name: "push-to-pull-request-branch: object form with policy and exclude passes",
			safeOutputs: map[string]any{
				"push-to-pull-request-branch": map[string]any{
					"protected-files": map[string]any{
						"policy":  "fallback-to-issue",
						"exclude": []any{"AGENTS.md", ".agents/"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "push-to-pull-request-branch: string form passes",
			safeOutputs: map[string]any{
				"push-to-pull-request-branch": map[string]any{
					"protected-files": "blocked",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			frontmatter := baseFrontmatter(tt.safeOutputs)
			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(frontmatter, "/tmp/gh-aw/protected-files-schema-test.md")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected validation error for %q, got nil", tt.name)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got: %v", tt.errContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected %q to pass schema validation, got: %v", tt.name, err)
				}
			}
		})
	}
}

// TestMainWorkflowSchema_ProtectedFilesObjectFormStructure verifies that the
// main workflow JSON schema defines protected-files as a oneOf [string, object],
// not as a oneOf [string, null] (the old broken form that caused
// "expected string or null, got object" errors).
func TestMainWorkflowSchema_ProtectedFilesObjectFormStructure(t *testing.T) {
	t.Parallel()

	schemaContent, err := os.ReadFile("schemas/main_workflow_schema.json")
	if err != nil {
		t.Fatalf("failed to read schema: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(schemaContent, &schema); err != nil {
		t.Fatalf("failed to parse schema JSON: %v", err)
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema missing 'properties'")
	}
	safeOutputsSchema, ok := properties["safe-outputs"].(map[string]any)
	if !ok {
		t.Fatal("schema missing 'properties.safe-outputs'")
	}
	safeOutputsProps, ok := safeOutputsSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema missing 'properties.safe-outputs.properties'")
	}

	for _, handlerName := range []string{"create-pull-request", "push-to-pull-request-branch"} {
		t.Run(handlerName, func(t *testing.T) {
			handlerSchema, ok := safeOutputsProps[handlerName].(map[string]any)
			if !ok {
				t.Fatalf("schema missing 'safe-outputs.%s'", handlerName)
			}
			handlerOneOf, ok := handlerSchema["oneOf"].([]any)
			if !ok {
				t.Fatalf("'safe-outputs.%s' missing oneOf", handlerName)
			}

			// Find the object branch (the one with properties)
			var objectBranchProps map[string]any
			for _, candidate := range handlerOneOf {
				c, ok := candidate.(map[string]any)
				if !ok {
					continue
				}
				if c["type"] == "object" {
					if props, ok := c["properties"].(map[string]any); ok {
						objectBranchProps = props
						break
					}
				}
			}
			if objectBranchProps == nil {
				t.Fatalf("'safe-outputs.%s' has no object branch in oneOf", handlerName)
			}

			pfSchema, ok := objectBranchProps["protected-files"].(map[string]any)
			if !ok {
				t.Fatalf("'safe-outputs.%s.properties.protected-files' not found", handlerName)
			}

			pfOneOf, ok := pfSchema["oneOf"].([]any)
			if !ok {
				t.Fatalf("'safe-outputs.%s.properties.protected-files' missing oneOf", handlerName)
			}

			var hasStringBranch, hasObjectBranch bool
			for _, branch := range pfOneOf {
				b, ok := branch.(map[string]any)
				if !ok {
					continue
				}
				switch b["type"] {
				case "string":
					hasStringBranch = true
				case "object":
					hasObjectBranch = true
				case "null":
					t.Errorf("'safe-outputs.%s.protected-files' has a null branch in its oneOf; "+
						"the object form would produce 'expected string or null, got object' errors", handlerName)
				}
			}
			if !hasStringBranch {
				t.Errorf("'safe-outputs.%s.protected-files' missing string branch in oneOf", handlerName)
			}
			if !hasObjectBranch {
				t.Errorf("'safe-outputs.%s.protected-files' missing object branch in oneOf; "+
					"the object form {policy, exclude} would fail compilation", handlerName)
			}

			// Verify the object branch has the expected sub-fields
			for _, branch := range pfOneOf {
				b, ok := branch.(map[string]any)
				if !ok || b["type"] != "object" {
					continue
				}
				objProps, ok := b["properties"].(map[string]any)
				if !ok {
					t.Fatalf("'safe-outputs.%s.protected-files' object branch missing properties", handlerName)
				}
				if _, hasPolicyField := objProps["policy"]; !hasPolicyField {
					t.Errorf("'safe-outputs.%s.protected-files' object branch missing 'policy' field", handlerName)
				}
				if _, hasExcludeField := objProps["exclude"]; !hasExcludeField {
					t.Errorf("'safe-outputs.%s.protected-files' object branch missing 'exclude' field", handlerName)
				}
				if b["additionalProperties"] != false {
					t.Errorf("'safe-outputs.%s.protected-files' object branch should have additionalProperties: false", handlerName)
				}
			}
		})
	}
}

// TestValidateWithSchema_YAMLIntegerTypes verifies that validateWithSchema accepts
// YAML-native integer types (uint64/int64) when the schema expects number/integer.
func TestValidateWithSchema_YAMLIntegerTypes(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"timeout-minutes": {"type": "integer"},
			"max-retries": {"type": "number"},
			"name": {"type": "string"}
		},
		"additionalProperties": false
	}`

	// Simulate what goccy/go-yaml produces: uint64 for positive, int64 for negative
	frontmatter := map[string]any{
		"timeout-minutes": uint64(20),
		"max-retries":     int64(3),
		"name":            "test",
	}

	err := validateWithSchema(frontmatter, schema, "yaml integer types")
	if err != nil {
		t.Errorf("validateWithSchema should accept YAML integer types, got: %v", err)
	}
}
