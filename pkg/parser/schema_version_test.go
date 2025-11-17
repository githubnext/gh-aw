package parser

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGetMainWorkflowSchemaVersion(t *testing.T) {
	version, err := GetMainWorkflowSchemaVersion()
	if err != nil {
		t.Fatalf("failed to get main workflow schema version: %v", err)
	}

	// Verify version is set
	if version.Version == "" {
		t.Error("expected non-empty version, got empty string")
	}

	// Verify version format (should be semver-like)
	if version.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", version.Version)
	}

	// Verify ID is set
	if version.ID == "" {
		t.Error("expected non-empty $id, got empty string")
	}

	// Verify ID format
	expectedID := "https://github.com/githubnext/gh-aw/schemas/workflow/v1.0.0"
	if version.ID != expectedID {
		t.Errorf("expected $id '%s', got '%s'", expectedID, version.ID)
	}
}

func TestGetIncludedFileSchemaVersion(t *testing.T) {
	version, err := GetIncludedFileSchemaVersion()
	if err != nil {
		t.Fatalf("failed to get included file schema version: %v", err)
	}

	// Verify version is set
	if version.Version == "" {
		t.Error("expected non-empty version, got empty string")
	}

	// Verify version format (should be semver-like)
	if version.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", version.Version)
	}

	// Verify ID is set
	if version.ID == "" {
		t.Error("expected non-empty $id, got empty string")
	}

	// Verify ID format
	expectedID := "https://github.com/githubnext/gh-aw/schemas/included-file/v1.0.0"
	if version.ID != expectedID {
		t.Errorf("expected $id '%s', got '%s'", expectedID, version.ID)
	}
}

func TestGetMCPConfigSchemaVersion(t *testing.T) {
	version, err := GetMCPConfigSchemaVersion()
	if err != nil {
		t.Fatalf("failed to get MCP config schema version: %v", err)
	}

	// Verify version is set
	if version.Version == "" {
		t.Error("expected non-empty version, got empty string")
	}

	// Verify version format (should be semver-like)
	if version.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", version.Version)
	}

	// Verify ID is set
	if version.ID == "" {
		t.Error("expected non-empty $id, got empty string")
	}

	// Verify ID format
	expectedID := "https://github.com/githubnext/gh-aw/schemas/mcp-config/v1.0.0"
	if version.ID != expectedID {
		t.Errorf("expected $id '%s', got '%s'", expectedID, version.ID)
	}
}

func TestSchemaVersionEmbedding(t *testing.T) {
	// Test that the schema strings are properly embedded and contain version fields
	tests := []struct {
		name       string
		schemaJSON string
		schemaName string
	}{
		{
			name:       "main workflow schema",
			schemaJSON: mainWorkflowSchema,
			schemaName: "main_workflow_schema.json",
		},
		{
			name:       "included file schema",
			schemaJSON: includedFileSchema,
			schemaName: "included_file_schema.json",
		},
		{
			name:       "MCP config schema",
			schemaJSON: mcpConfigSchema,
			schemaName: "mcp_config_schema.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the schema string is not empty
			if tt.schemaJSON == "" {
				t.Errorf("schema %s is empty", tt.schemaName)
			}

			// Parse the schema JSON
			var schemaDoc map[string]any
			if err := json.Unmarshal([]byte(tt.schemaJSON), &schemaDoc); err != nil {
				t.Fatalf("failed to parse %s: %v", tt.schemaName, err)
			}

			// Verify version field exists
			version, hasVersion := schemaDoc["version"]
			if !hasVersion {
				t.Errorf("schema %s missing 'version' field", tt.schemaName)
			}

			// Verify version is a string
			versionStr, ok := version.(string)
			if !ok {
				t.Errorf("schema %s 'version' field is not a string, got %T", tt.schemaName, version)
			}

			// Verify version is not empty
			if versionStr == "" {
				t.Errorf("schema %s 'version' field is empty", tt.schemaName)
			}

			// Verify $id field exists
			id, hasID := schemaDoc["$id"]
			if !hasID {
				t.Errorf("schema %s missing '$id' field", tt.schemaName)
			}

			// Verify $id is a string
			idStr, ok := id.(string)
			if !ok {
				t.Errorf("schema %s '$id' field is not a string, got %T", tt.schemaName, id)
			}

			// Verify $id is not empty
			if idStr == "" {
				t.Errorf("schema %s '$id' field is empty", tt.schemaName)
			}

			// Verify $id contains version
			if versionStr != "" && !strings.Contains(idStr, versionStr) {
				t.Errorf("schema %s '$id' field '%s' does not contain version '%s'", tt.schemaName, idStr, versionStr)
			}
		})
	}
}

func TestGetSchemaVersion(t *testing.T) {
	tests := []struct {
		name        string
		schemaJSON  string
		wantVersion string
		wantID      string
		wantErr     bool
	}{
		{
			name: "valid schema with version and id",
			schemaJSON: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"$id": "https://example.com/schema/v1.0.0",
				"version": "1.0.0",
				"type": "object"
			}`,
			wantVersion: "1.0.0",
			wantID:      "https://example.com/schema/v1.0.0",
			wantErr:     false,
		},
		{
			name: "schema with version only",
			schemaJSON: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"version": "2.0.0",
				"type": "object"
			}`,
			wantVersion: "2.0.0",
			wantID:      "",
			wantErr:     false,
		},
		{
			name: "schema with id only",
			schemaJSON: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"$id": "https://example.com/schema/v3.0.0",
				"type": "object"
			}`,
			wantVersion: "",
			wantID:      "https://example.com/schema/v3.0.0",
			wantErr:     false,
		},
		{
			name: "schema without version or id",
			schemaJSON: `{
				"$schema": "http://json-schema.org/draft-07/schema#",
				"type": "object"
			}`,
			wantVersion: "",
			wantID:      "",
			wantErr:     false,
		},
		{
			name:        "invalid JSON",
			schemaJSON:  `{invalid json`,
			wantVersion: "",
			wantID:      "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := getSchemaVersion(tt.schemaJSON)

			// Check error expectation
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check version
			if version.Version != tt.wantVersion {
				t.Errorf("expected version '%s', got '%s'", tt.wantVersion, version.Version)
			}

			// Check ID
			if version.ID != tt.wantID {
				t.Errorf("expected $id '%s', got '%s'", tt.wantID, version.ID)
			}
		})
	}
}

func TestSchemaVersionsConsistency(t *testing.T) {
	// Get all schema versions
	mainVersion, err := GetMainWorkflowSchemaVersion()
	if err != nil {
		t.Fatalf("failed to get main workflow schema version: %v", err)
	}

	includedVersion, err := GetIncludedFileSchemaVersion()
	if err != nil {
		t.Fatalf("failed to get included file schema version: %v", err)
	}

	mcpVersion, err := GetMCPConfigSchemaVersion()
	if err != nil {
		t.Fatalf("failed to get MCP config schema version: %v", err)
	}

	// All schemas should have the same version (1.0.0 initially)
	// This test ensures consistency across all schema files
	if mainVersion.Version != "1.0.0" {
		t.Errorf("main workflow schema version should be '1.0.0', got '%s'", mainVersion.Version)
	}

	if includedVersion.Version != "1.0.0" {
		t.Errorf("included file schema version should be '1.0.0', got '%s'", includedVersion.Version)
	}

	if mcpVersion.Version != "1.0.0" {
		t.Errorf("MCP config schema version should be '1.0.0', got '%s'", mcpVersion.Version)
	}

	// Verify all IDs are unique and properly formatted
	ids := map[string]string{
		"main workflow": mainVersion.ID,
		"included file": includedVersion.ID,
		"MCP config":    mcpVersion.ID,
	}

	seenIDs := make(map[string]string)
	for name, id := range ids {
		if id == "" {
			t.Errorf("%s schema has empty $id", name)
			continue
		}

		if prevName, seen := seenIDs[id]; seen {
			t.Errorf("duplicate $id '%s' found in %s and %s schemas", id, name, prevName)
		}
		seenIDs[id] = name

		// Verify ID starts with expected base URL
		expectedPrefix := "https://github.com/githubnext/gh-aw/schemas/"
		if !strings.Contains(id, expectedPrefix) {
			t.Errorf("%s schema $id '%s' does not contain expected prefix '%s'", name, id, expectedPrefix)
		}
	}
}
