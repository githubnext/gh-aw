//go:build !integration

package workflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFrontmatterConfigAgenticWorkflows(t *testing.T) {
	tests := []struct {
		name            string
		tools           map[string]any
		configured      bool
		enabled         bool
		allowWriteTools bool
	}{
		{name: "tools absent"},
		{name: "tool absent", tools: map[string]any{}},
		{name: "null defaults", tools: map[string]any{"agentic-workflows": nil}, configured: true, enabled: true},
		{name: "boolean enabled", tools: map[string]any{"agentic-workflows": true}, configured: true, enabled: true},
		{name: "boolean disabled", tools: map[string]any{"agentic-workflows": false}, configured: true},
		{name: "object defaults", tools: map[string]any{"agentic-workflows": map[string]any{}}, configured: true, enabled: true},
		{name: "writes enabled", tools: map[string]any{"agentic-workflows": map[string]any{"allow-write-tools": true}}, configured: true, enabled: true, allowWriteTools: true},
		{name: "writes disabled", tools: map[string]any{"agentic-workflows": map[string]any{"allow-write-tools": false}}, configured: true, enabled: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frontmatter := map[string]any{}
			if tt.tools != nil {
				frontmatter["tools"] = tt.tools
			}
			config, err := ParseFrontmatterConfig(frontmatter)
			require.NoError(t, err)
			roundTripped, err := ParseFrontmatterConfig(config.ToMap())
			require.NoError(t, err)
			for _, parsed := range []*FrontmatterConfig{config, roundTripped} {
				if tt.tools == nil {
					assert.Nil(t, parsed.Tools)
					continue
				}
				require.NotNil(t, parsed.Tools)
				if !tt.configured {
					assert.Nil(t, parsed.Tools.AgenticWorkflows)
					continue
				}
				require.NotNil(t, parsed.Tools.AgenticWorkflows)
				assert.Equal(t, tt.enabled, parsed.Tools.AgenticWorkflows.Enabled)
				assert.Equal(t, tt.allowWriteTools, parsed.Tools.AgenticWorkflows.AllowWriteTools)
			}
		})
	}
}

func TestAgenticWorkflowsToolConfigJSONTags(t *testing.T) {
	data, err := json.Marshal(&ToolsConfig{
		AgenticWorkflows: &AgenticWorkflowsToolConfig{Enabled: true, AllowWriteTools: true},
	})
	require.NoError(t, err)
	var tools map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &tools))
	require.Contains(t, tools, "agentic-workflows")
	assert.JSONEq(t, `{"allow-write-tools":true}`, string(tools["agentic-workflows"]))
}
