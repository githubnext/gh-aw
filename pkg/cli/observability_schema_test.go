//go:build !integration

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObservabilityReportSchemaIncludesLineageAndReasoning(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "schemas", "observability-report.json")
	schemaContent, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "should read observability schema")

	var schema map[string]any
	require.NoError(t, json.Unmarshal(schemaContent, &schema), "schema should parse as JSON")

	assert.Equal(t, "http://json-schema.org/draft-07/schema#", schema["$schema"], "schema should use Draft 7 for consistency with published schemas")

	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok, "root properties should exist")

	_, hasLineage := properties["lineage"]
	assert.True(t, hasLineage, "schema should include lineage section")

	_, hasReasoning := properties["reasoning"]
	assert.True(t, hasReasoning, "schema should include reasoning section")

	defs, ok := schema["$defs"].(map[string]any)
	require.True(t, ok, "schema defs should exist")

	awContextDef, ok := defs["AwContext"].(map[string]any)
	require.True(t, ok, "AwContext definition should exist")

	awContextProps, ok := awContextDef["properties"].(map[string]any)
	require.True(t, ok, "AwContext properties should exist")
	assert.Contains(t, awContextProps, "repo")
	assert.Contains(t, awContextProps, "run_id")
	assert.Contains(t, awContextProps, "workflow_id")
	assert.Contains(t, awContextProps, "workflow_call_id")
	assert.Contains(t, awContextProps, "actor")
	assert.Contains(t, awContextProps, "event_type")

	reasoningStepDef, ok := defs["ReasoningStep"].(map[string]any)
	require.True(t, ok, "ReasoningStep definition should exist")

	reasoningStepProps, ok := reasoningStepDef["properties"].(map[string]any)
	require.True(t, ok, "ReasoningStep properties should exist")
	assert.Contains(t, reasoningStepProps, "kind")
	assert.Contains(t, reasoningStepProps, "summary")
	assert.Contains(t, reasoningStepProps, "evidence")
	assert.Contains(t, reasoningStepProps, "tool_refs")

	lineageDef, ok := defs["Lineage"].(map[string]any)
	require.True(t, ok, "Lineage definition should exist")

	lineageRequired, ok := lineageDef["required"].([]any)
	require.True(t, ok, "Lineage required array should exist")
	assert.Contains(t, lineageRequired, "trace_id")
}
