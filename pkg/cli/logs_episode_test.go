//go:build !integration

package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildEpisodeDataIncludesToolCalls(t *testing.T) {
	runs := []RunData{
		{
			DatabaseID:   101,
			WorkflowName: "my-workflow",
			Status:       "completed",
			Conclusion:   "success",
			TokenUsage:   1000,
			CreatedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID:   101,
				WorkflowName: "my-workflow",
			},
			MCPToolUsage: &MCPToolUsageData{
				ToolCalls: []MCPToolCall{
					{
						ServerName: "github",
						ToolName:   "get_file_contents",
						InputSize:  400,
						OutputSize: 9200,
						Duration:   "350ms",
						Status:     "success",
					},
					{
						ServerName: "github",
						ToolName:   "create_pull_request",
						InputSize:  200,
						OutputSize: 3000,
						Duration:   "600ms",
						Status:     "error",
						Error:      "403 Resource not accessible by integration",
					},
				},
			},
		},
	}

	episodes, _ := buildEpisodeData(runs, processedRuns)
	require.Len(t, episodes, 1, "expected one episode")

	ep := episodes[0]
	require.Len(t, ep.ToolCalls, 2, "expected two tool calls in episode")

	// First tool call: success
	tc0 := ep.ToolCalls[0]
	assert.Equal(t, "get_file_contents", tc0.Tool, "tool name should match")
	assert.Equal(t, "github", tc0.Server, "server name should match")
	assert.Equal(t, (400+9200)/CharsPerToken, tc0.Tokens, "tokens should be estimated from sizes")
	assert.InDelta(t, float64(350), tc0.DurationMS, 0.001, "duration_ms should be 350")
	assert.Equal(t, "success", tc0.Status, "status should match")
	assert.Empty(t, tc0.Error, "no error expected")

	// Second tool call: error
	tc1 := ep.ToolCalls[1]
	assert.Equal(t, "create_pull_request", tc1.Tool, "tool name should match")
	assert.Equal(t, "github", tc1.Server, "server name should match")
	assert.Equal(t, (200+3000)/CharsPerToken, tc1.Tokens, "tokens should be estimated from sizes")
	assert.InDelta(t, float64(600), tc1.DurationMS, 0.001, "duration_ms should be 600")
	assert.Equal(t, "error", tc1.Status, "status should match")
	assert.Equal(t, "403 Resource not accessible by integration", tc1.Error, "error message should match")
}

func TestBuildEpisodeDataNoToolCallsWhenMCPUsageAbsent(t *testing.T) {
	runs := []RunData{
		{
			DatabaseID:   200,
			WorkflowName: "no-mcp-workflow",
			Status:       "completed",
			CreatedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{
				DatabaseID:   200,
				WorkflowName: "no-mcp-workflow",
			},
			MCPToolUsage: nil, // no MCP tool usage
		},
	}

	episodes, _ := buildEpisodeData(runs, processedRuns)
	require.Len(t, episodes, 1, "expected one episode")

	ep := episodes[0]
	assert.Empty(t, ep.ToolCalls, "tool_calls should be absent when no MCP usage data")
}

func TestBuildEpisodeDataAggregatesToolCallsAcrossRuns(t *testing.T) {
	// Two runs belonging to the same episode (via dispatch)
	workflowCallID := "dispatch:wc-42"
	runs := []RunData{
		{
			DatabaseID:   301,
			WorkflowName: "orchestrator",
			Status:       "completed",
			CreatedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			AwContext: &AwContext{
				WorkflowCallID: "wc-42",
			},
		},
		{
			DatabaseID:   302,
			WorkflowName: "worker",
			Status:       "completed",
			CreatedAt:    time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC),
			AwContext: &AwContext{
				WorkflowCallID: "wc-42",
			},
		},
	}
	processedRuns := []ProcessedRun{
		{
			Run: WorkflowRun{DatabaseID: 301, WorkflowName: "orchestrator"},
			MCPToolUsage: &MCPToolUsageData{
				ToolCalls: []MCPToolCall{
					{
						ServerName: "github",
						ToolName:   "search_code",
						InputSize:  100,
						OutputSize: 500,
						Duration:   "200ms",
						Status:     "success",
					},
				},
			},
		},
		{
			Run: WorkflowRun{DatabaseID: 302, WorkflowName: "worker"},
			MCPToolUsage: &MCPToolUsageData{
				ToolCalls: []MCPToolCall{
					{
						ServerName: "github",
						ToolName:   "create_issue",
						InputSize:  50,
						OutputSize: 200,
						Duration:   "400ms",
						Status:     "success",
					},
				},
			},
		},
	}

	episodes, _ := buildEpisodeData(runs, processedRuns)
	require.Len(t, episodes, 1, "expected one merged episode from two dispatch runs")

	ep := episodes[0]
	assert.Equal(t, workflowCallID, ep.EpisodeID, "episode id should reflect dispatch call id")
	assert.Len(t, ep.ToolCalls, 2, "tool_calls should include calls from both runs")
}

func TestMCPToolCallToEpisodeToolCall(t *testing.T) {
	tests := []struct {
		name           string
		input          MCPToolCall
		expectedTool   string
		expectedServer string
		expectedTokens int
		expectedDurMS  float64
		expectedStatus string
		expectedError  string
	}{
		{
			name: "success call with duration",
			input: MCPToolCall{
				ServerName: "github",
				ToolName:   "list_issues",
				InputSize:  400,
				OutputSize: 1200,
				Duration:   "250ms",
				Status:     "success",
			},
			expectedTool:   "list_issues",
			expectedServer: "github",
			expectedTokens: (400 + 1200) / CharsPerToken,
			expectedDurMS:  250,
			expectedStatus: "success",
		},
		{
			name: "error call with error message",
			input: MCPToolCall{
				ServerName: "playwright",
				ToolName:   "navigate",
				InputSize:  100,
				OutputSize: 0,
				Duration:   "1s",
				Status:     "error",
				Error:      "timeout",
			},
			expectedTool:   "navigate",
			expectedServer: "playwright",
			expectedTokens: 100 / CharsPerToken,
			expectedDurMS:  1000,
			expectedStatus: "error",
			expectedError:  "timeout",
		},
		{
			name: "call without duration",
			input: MCPToolCall{
				ServerName: "github",
				ToolName:   "get_repo",
				InputSize:  200,
				OutputSize: 800,
				Duration:   "",
				Status:     "success",
			},
			expectedTool:   "get_repo",
			expectedServer: "github",
			expectedTokens: (200 + 800) / CharsPerToken,
			expectedDurMS:  0,
			expectedStatus: "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mcpToolCallToEpisodeToolCall(tt.input)
			assert.Equal(t, tt.expectedTool, got.Tool, "Tool should match")
			assert.Equal(t, tt.expectedServer, got.Server, "Server should match")
			assert.Equal(t, tt.expectedTokens, got.Tokens, "Tokens should be estimated from sizes")
			assert.InDelta(t, tt.expectedDurMS, got.DurationMS, 0.001, "DurationMS should match")
			assert.Equal(t, tt.expectedStatus, got.Status, "Status should match")
			assert.Equal(t, tt.expectedError, got.Error, "Error should match")
		})
	}
}
