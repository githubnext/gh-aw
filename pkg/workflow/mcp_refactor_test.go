//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseEngineRenderMCPConfig_DefaultNoOp(t *testing.T) {
	var yaml strings.Builder
	err := (&BaseEngine{}).RenderMCPConfig(&yaml, nil, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, yaml.String())
}

func TestDefaultJSONMCPConfigEngines_RenderMCPConfig(t *testing.T) {
	tests := []struct {
		name       string
		engine     CodingAgentEngine
		configPath string
	}{
		{
			name:       "antigravity",
			engine:     NewAntigravityEngine(),
			configPath: "${RUNNER_TEMP}/gh-aw/mcp-config/mcp-servers.json",
		},
		{
			name:       "claude",
			engine:     NewClaudeEngine(),
			configPath: "${RUNNER_TEMP}/gh-aw/mcp-config/mcp-servers.json",
		},
		{
			name:       "gemini",
			engine:     NewGeminiEngine(),
			configPath: "${RUNNER_TEMP}/gh-aw/mcp-config/mcp-servers.json",
		},
		{
			name:       "crush",
			engine:     NewCrushEngine(),
			configPath: "/tmp/gh-aw/mcp-config/mcp-servers.json",
		},
		{
			name:       "opencode",
			engine:     NewOpenCodeEngine(),
			configPath: "/tmp/gh-aw/mcp-config/mcp-servers.json",
		},
	}

	workflowData := &WorkflowData{
		Tools: map[string]any{
			"agentic-workflows": nil,
		},
	}
	mcpTools := []string{"agentic-workflows"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			err := tt.engine.RenderMCPConfig(&yaml, workflowData.Tools, mcpTools, workflowData)
			require.NoError(t, err)

			var expected strings.Builder
			err = renderDefaultJSONMCPConfig(&expected, workflowData.Tools, mcpTools, workflowData, tt.configPath)
			require.NoError(t, err)

			assert.Equal(t, normalizeHeredocDelimiters(expected.String()), normalizeHeredocDelimiters(yaml.String()))
			assert.Contains(t, yaml.String(), "agenticworkflows")
		})
	}
}

func TestSingleJSONResponseLogMetrics_Engines(t *testing.T) {
	tests := []struct {
		name   string
		engine LogParser
	}{
		{name: "antigravity", engine: NewAntigravityEngine()},
		{name: "gemini", engine: NewGeminiEngine()},
	}

	logContent := `not json
{"response":"first","stats":{"models":{"model-a":{"input_tokens":2,"output_tokens":3}},"tools":{"bash":{},"github_search_issues":{}}}}
{"response":"second","stats":{"models":{"model-b":{"input_tokens":5,"output_tokens":7}},"tools":{"bash":{},"github_search_issues":{},"web_fetch":{}}}}`

	expectedCalls := map[string]int{
		"bash":                 2,
		"github_search_issues": 2,
		"web_fetch":            1,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := tt.engine.ParseLogMetrics(logContent, true)
			assert.Equal(t, 1, metrics.Turns)
			assert.Equal(t, 17, metrics.TokenUsage)
			require.Len(t, metrics.ToolCalls, len(expectedCalls))

			actualCalls := make(map[string]int, len(metrics.ToolCalls))
			for _, toolCall := range metrics.ToolCalls {
				actualCalls[toolCall.Name] = toolCall.CallCount
			}
			assert.Equal(t, expectedCalls, actualCalls)
		})
	}
}
