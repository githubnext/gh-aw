package workflow

import (
	"strings"
	"testing"
)

func TestClaudeEngineWithToolsTimeout(t *testing.T) {
	engine := NewClaudeEngine()

	tests := []struct {
		name            string
		toolsTimeout    int
		expectedTimeout string
	}{
		{
			name:            "default timeout when not specified",
			toolsTimeout:    0,
			expectedTimeout: "MCP_TIMEOUT: \"60000\"", // 60 seconds default in milliseconds
		},
		{
			name:            "custom timeout of 30 seconds",
			toolsTimeout:    30,
			expectedTimeout: "MCP_TIMEOUT: \"30000\"", // 30 seconds in milliseconds
		},
		{
			name:            "custom timeout of 120 seconds",
			toolsTimeout:    120,
			expectedTimeout: "MCP_TIMEOUT: \"120000\"", // 120 seconds in milliseconds
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				ToolsTimeout: tt.toolsTimeout,
				Tools:        map[string]any{},
			}

			// Get execution steps
			executionSteps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
			if len(executionSteps) == 0 {
				t.Fatal("Expected at least one execution step")
			}

			// Check the execution step for MCP_TIMEOUT
			stepContent := strings.Join([]string(executionSteps[0]), "\n")
			if !strings.Contains(stepContent, tt.expectedTimeout) {
				t.Errorf("Expected '%s' in execution step, got: %s", tt.expectedTimeout, stepContent)
			}
		})
	}
}

func TestCodexEngineWithToolsTimeout(t *testing.T) {
	engine := NewCodexEngine()

	tests := []struct {
		name            string
		toolsTimeout    int
		expectedTimeout string
	}{
		{
			name:            "default timeout when not specified",
			toolsTimeout:    0,
			expectedTimeout: "tool_timeout_sec = 120", // 120 seconds default
		},
		{
			name:            "custom timeout of 30 seconds",
			toolsTimeout:    30,
			expectedTimeout: "tool_timeout_sec = 30",
		},
		{
			name:            "custom timeout of 180 seconds",
			toolsTimeout:    180,
			expectedTimeout: "tool_timeout_sec = 180",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				ToolsTimeout: tt.toolsTimeout,
				Name:         "test-workflow",
				Tools: map[string]any{
					"github": map[string]any{},
				},
			}

			// Render MCP config
			var configBuilder strings.Builder
			mcpTools := []string{"github"}
			engine.RenderMCPConfig(&configBuilder, workflowData.Tools, mcpTools, workflowData)
			configContent := configBuilder.String()

			if !strings.Contains(configContent, tt.expectedTimeout) {
				t.Errorf("Expected '%s' in MCP config, got: %s", tt.expectedTimeout, configContent)
			}
		})
	}
}

func TestExtractToolsTimeout(t *testing.T) {
	compiler := &Compiler{}

	tests := []struct {
		name            string
		tools           map[string]any
		expectedTimeout int
	}{
		{
			name:            "no timeout specified",
			tools:           map[string]any{},
			expectedTimeout: 0,
		},
		{
			name: "timeout as int",
			tools: map[string]any{
				"timeout": 45,
			},
			expectedTimeout: 45,
		},
		{
			name: "timeout as int64",
			tools: map[string]any{
				"timeout": int64(90),
			},
			expectedTimeout: 90,
		},
		{
			name: "timeout as uint",
			tools: map[string]any{
				"timeout": uint(75),
			},
			expectedTimeout: 75,
		},
		{
			name: "timeout as uint64",
			tools: map[string]any{
				"timeout": uint64(120),
			},
			expectedTimeout: 120,
		},
		{
			name: "timeout as float64",
			tools: map[string]any{
				"timeout": 60.0,
			},
			expectedTimeout: 60,
		},
		{
			name:            "nil tools",
			tools:           nil,
			expectedTimeout: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeout := compiler.extractToolsTimeout(tt.tools)
			if timeout != tt.expectedTimeout {
				t.Errorf("Expected timeout %d, got %d", tt.expectedTimeout, timeout)
			}
		})
	}
}
