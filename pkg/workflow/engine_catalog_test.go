//go:build !integration

package workflow

import (
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEngineCatalog_IDs verifies that IDs() returns all engine IDs in sorted order.
func TestEngineCatalog_IDs(t *testing.T) {
	registry := NewEngineRegistry()
	catalog := NewEngineCatalog(registry)

	ids := catalog.IDs()
	require.NotEmpty(t, ids, "IDs() should return a non-empty list")

	// Verify all built-in engines are present
	expectedIDs := []string{"claude", "codex", "copilot", "gemini"}
	assert.Equal(t, expectedIDs, ids, "IDs() should return all built-in engines in sorted order")

	// Verify the list is sorted
	sorted := make([]string, len(ids))
	copy(sorted, ids)
	sort.Strings(sorted)
	assert.Equal(t, sorted, ids, "IDs() should return IDs in sorted order")
}

// TestEngineCatalog_DisplayNames verifies that DisplayNames() returns names in sorted ID order.
func TestEngineCatalog_DisplayNames(t *testing.T) {
	registry := NewEngineRegistry()
	catalog := NewEngineCatalog(registry)

	names := catalog.DisplayNames()
	require.NotEmpty(t, names, "DisplayNames() should return a non-empty list")
	assert.Len(t, names, len(catalog.IDs()), "DisplayNames() should have same length as IDs()")

	// Verify display names match expected values in sorted ID order (claude, codex, copilot, gemini)
	expectedNames := []string{"Claude Code", "Codex", "GitHub Copilot CLI", "Google Gemini CLI"}
	assert.Equal(t, expectedNames, names, "DisplayNames() should return display names in sorted ID order")
}

// TestEngineCatalog_All verifies that All() returns all definitions in sorted ID order.
func TestEngineCatalog_All(t *testing.T) {
	registry := NewEngineRegistry()
	catalog := NewEngineCatalog(registry)

	defs := catalog.All()
	require.NotEmpty(t, defs, "All() should return a non-empty list")
	assert.Len(t, defs, len(catalog.IDs()), "All() should have same length as IDs()")

	ids := catalog.IDs()
	for i, def := range defs {
		assert.Equal(t, ids[i], def.ID, "All()[%d].ID should match IDs()[%d]", i, i)
		assert.NotEmpty(t, def.DisplayName, "All()[%d].DisplayName should not be empty", i)
	}
}

// engineSchemaEnums parses the main workflow schema and extracts engine enum values
// from both the string variant and the object id property of engine_config.
func engineSchemaEnums(t *testing.T) []string {
	t.Helper()

	schemaBytes, err := os.ReadFile("../parser/schemas/main_workflow_schema.json")
	require.NoError(t, err, "should be able to read main_workflow_schema.json")

	var schema map[string]any
	require.NoError(t, json.Unmarshal(schemaBytes, &schema), "schema should be valid JSON")

	defs, ok := schema["$defs"].(map[string]any)
	require.True(t, ok, "schema should have $defs")

	engineConfig, ok := defs["engine_config"].(map[string]any)
	require.True(t, ok, "$defs should have engine_config")

	oneOf, ok := engineConfig["oneOf"].([]any)
	require.True(t, ok, "engine_config should have oneOf")

	// The first oneOf variant is the plain string enum
	for _, variant := range oneOf {
		v, ok := variant.(map[string]any)
		if !ok {
			continue
		}
		if v["type"] == "string" {
			rawEnum, ok := v["enum"].([]any)
			if !ok {
				continue
			}
			enums := make([]string, 0, len(rawEnum))
			for _, e := range rawEnum {
				if s, ok := e.(string); ok {
					enums = append(enums, s)
				}
			}
			sort.Strings(enums)
			return enums
		}
	}
	t.Fatal("could not find string enum in engine_config oneOf")
	return nil
}

// TestEngineCatalogMatchesSchema asserts that the schema engine enum values exactly
// match the catalog IDs. A failure here means the schema and catalog have drifted apart.
func TestEngineCatalogMatchesSchema(t *testing.T) {
	registry := NewEngineRegistry()
	catalog := NewEngineCatalog(registry)

	catalogIDs := catalog.IDs() // already sorted
	schemaEnums := engineSchemaEnums(t)

	assert.Equal(t, catalogIDs, schemaEnums,
		"schema engine enum must match catalog IDs exactly — run 'make build' after updating the schema")
}
