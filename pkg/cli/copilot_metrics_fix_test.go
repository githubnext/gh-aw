//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractLogMetricsExcludesWorkflowLogsDir is a regression test for the
// double-counting issue reported in the "Audit shows inconsistent metrics on
// repeated calls for same run" issue.
//
// Background:
// downloadWorkflowRunLogs (called during artifact download) places GitHub Actions
// step-output files under workflow-logs/.  These files capture the runner's combined
// stdout/stderr for each step, which means they contain a copy of everything the
// agent wrote to stdout — including the same token-usage JSON blocks that are already
// in agent-stdio.log / agent.log from the dedicated agent artifact.
//
// Because the log-file walk in extractLogMetrics previously did NOT skip
// workflow-logs/, any .log or *log*.txt file found there was parsed and its
// TokenUsage was ADDED to metrics.TokenUsage (the walk uses +=).  With ~12 such
// copies the total ballooned to ≈4.7M instead of the correct ≈381k.
//
// The fix adds an explicit filepath.SkipDir return when the walk visits a directory
// named "workflow-logs", so only the agent artifact files are counted.
func TestExtractLogMetricsExcludesWorkflowLogsDir(t *testing.T) {
	tempDir := t.TempDir()

	// Simulate a Copilot-CLI run directory
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "aw_info.json"), []byte(`{"engine_id":"copilot"}`), 0600))

	// The single JSON data block that represents one LLM API call with 1000 tokens.
	oneTurn := `2025-09-26T11:00:00Z [DEBUG] data:
2025-09-26T11:00:00Z [DEBUG] {
2025-09-26T11:00:00Z [DEBUG]   "choices": [{"message": {"role": "assistant", "tool_calls": []}}],
2025-09-26T11:00:00Z [DEBUG]   "usage": {"prompt_tokens": 900, "completion_tokens": 100, "total_tokens": 1000}
2025-09-26T11:00:00Z [DEBUG] }
2025-09-26T11:00:01Z [DEBUG] Workflow done`

	// Primary agent log — the "source of truth" artifact
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "agent.log"), []byte(oneTurn), 0600))

	// Simulate workflow-logs/ as produced by downloadWorkflowRunLogs.
	// Two step-output files: one .log and one *log*.txt (both would have matched
	// the old filter), both containing identical token data.
	wfLogsDir := filepath.Join(tempDir, "workflow-logs", "agent")
	require.NoError(t, os.MkdirAll(wfLogsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(wfLogsDir, "runner.log"), []byte(oneTurn), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(wfLogsDir, "2_Run log step.txt"), []byte(oneTurn), 0644))

	metrics, err := extractLogMetrics(tempDir, false)
	require.NoError(t, err)

	// Without the fix, metrics.TokenUsage would be 3000 (1000 * 3 files).
	// With the fix, workflow-logs/ is skipped and only agent.log is counted.
	assert.Equal(t, 1000, metrics.TokenUsage,
		"TokenUsage must not include workflow-logs/ files (expected 1000, not %d)", metrics.TokenUsage)
	assert.Equal(t, 1, metrics.Turns,
		"Turns must not be inflated by workflow-logs/ copies (expected 1, not %d)", metrics.Turns)
}

// TestCopilotDebugLogTurnsExtraction verifies that Turns are correctly counted from
// [DEBUG] data: blocks in the Copilot CLI debug log format.
//
// This is a regression test for the bug where Turns was always 0 because the parser
// counted "User:"/"Human:"/"Query:" patterns that do not appear in Copilot CLI debug logs.
// The fix counts each "[DEBUG] data:" block as one API response (one turn).
func TestCopilotDebugLogTurnsExtraction(t *testing.T) {
	tempDir := t.TempDir()

	awInfoContent := `{"engine_id": "copilot"}`
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "aw_info.json"), []byte(awInfoContent), 0644))

	// Debug log with 2 API responses (2 data blocks) and "Executing tool: bash" between them.
	// The JSON blocks have empty tool_calls (common in production Copilot CLI logs).
	logContent := `2025-09-26T11:13:11.798Z [DEBUG] Starting Copilot CLI: 0.0.400
2025-09-26T11:13:12.575Z [DEBUG] data:
2025-09-26T11:13:12.575Z [DEBUG] {
2025-09-26T11:13:12.575Z [DEBUG]   "choices": [{
2025-09-26T11:13:12.575Z [DEBUG]     "message": {
2025-09-26T11:13:12.575Z [DEBUG]       "role": "assistant",
2025-09-26T11:13:12.575Z [DEBUG]       "content": null,
2025-09-26T11:13:12.575Z [DEBUG]       "tool_calls": []
2025-09-26T11:13:12.575Z [DEBUG]     }
2025-09-26T11:13:12.575Z [DEBUG]   }],
2025-09-26T11:13:12.575Z [DEBUG]   "usage": {
2025-09-26T11:13:12.575Z [DEBUG]     "prompt_tokens": 1000,
2025-09-26T11:13:12.575Z [DEBUG]     "completion_tokens": 50,
2025-09-26T11:13:12.575Z [DEBUG]     "total_tokens": 1050
2025-09-26T11:13:12.575Z [DEBUG]   }
2025-09-26T11:13:12.575Z [DEBUG] }
2025-09-26T11:13:13.000Z [DEBUG] Executing tool: bash
2025-09-26T11:13:13.500Z [DEBUG] Tool execution completed
2025-09-26T11:13:14.000Z [DEBUG] data:
2025-09-26T11:13:14.000Z [DEBUG] {
2025-09-26T11:13:14.000Z [DEBUG]   "choices": [{
2025-09-26T11:13:14.000Z [DEBUG]     "message": {
2025-09-26T11:13:14.000Z [DEBUG]       "role": "assistant",
2025-09-26T11:13:14.000Z [DEBUG]       "content": "Task done.",
2025-09-26T11:13:14.000Z [DEBUG]       "tool_calls": []
2025-09-26T11:13:14.000Z [DEBUG]     }
2025-09-26T11:13:14.000Z [DEBUG]   }],
2025-09-26T11:13:14.000Z [DEBUG]   "usage": {
2025-09-26T11:13:14.000Z [DEBUG]     "prompt_tokens": 1200,
2025-09-26T11:13:14.000Z [DEBUG]     "completion_tokens": 20,
2025-09-26T11:13:14.000Z [DEBUG]     "total_tokens": 1220
2025-09-26T11:13:14.000Z [DEBUG]   }
2025-09-26T11:13:14.000Z [DEBUG] }
2025-09-26T11:13:14.500Z [DEBUG] Workflow completed`

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "agent.log"), []byte(logContent), 0644))

	metrics, err := extractLogMetrics(tempDir, false)
	require.NoError(t, err)

	// Tokens should be accumulated from both data blocks
	assert.Equal(t, 2270, metrics.TokenUsage, "Should accumulate tokens from both data blocks: 1050 + 1220 = 2270")

	// Turns should be 2 (one per [DEBUG] data: block)
	assert.Equal(t, 2, metrics.Turns,
		"Turns should count [DEBUG] data: blocks; got %d", metrics.Turns)

	// Tool calls should be extracted from "Executing tool: bash" line
	assert.NotEmpty(t, metrics.ToolCalls, "ToolCalls should not be null when 'Executing tool:' is present")
	if len(metrics.ToolCalls) > 0 {
		found := false
		for _, tc := range metrics.ToolCalls {
			if tc.Name == "bash" {
				found = true
				assert.Equal(t, 1, tc.CallCount, "bash should have been called once")
			}
		}
		assert.True(t, found, "Expected to find 'bash' in tool calls")
	}
}

// TestCopilotDebugLogMultipleToolCalls verifies that multiple "Executing tool:" lines
// produce correct call counts in ToolCalls.
func TestCopilotDebugLogMultipleToolCalls(t *testing.T) {
	tempDir := t.TempDir()

	awInfoContent := `{"engine_id": "copilot"}`
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "aw_info.json"), []byte(awInfoContent), 0644))

	// Three API response blocks with different tool executions
	logContent := `2025-09-26T11:13:11.798Z [DEBUG] Starting Copilot CLI: 0.0.400
2025-09-26T11:13:12.000Z [DEBUG] data:
2025-09-26T11:13:12.000Z [DEBUG] {
2025-09-26T11:13:12.000Z [DEBUG]   "choices": [{"message": {"tool_calls": []}}],
2025-09-26T11:13:12.000Z [DEBUG]   "usage": {"prompt_tokens": 500, "completion_tokens": 10, "total_tokens": 510}
2025-09-26T11:13:12.000Z [DEBUG] }
2025-09-26T11:13:12.500Z [DEBUG] Executing tool: bash
2025-09-26T11:13:13.000Z [DEBUG] data:
2025-09-26T11:13:13.000Z [DEBUG] {
2025-09-26T11:13:13.000Z [DEBUG]   "choices": [{"message": {"tool_calls": []}}],
2025-09-26T11:13:13.000Z [DEBUG]   "usage": {"prompt_tokens": 600, "completion_tokens": 10, "total_tokens": 610}
2025-09-26T11:13:13.000Z [DEBUG] }
2025-09-26T11:13:13.500Z [DEBUG] Executing tool: bash
2025-09-26T11:13:14.000Z [DEBUG] data:
2025-09-26T11:13:14.000Z [DEBUG] {
2025-09-26T11:13:14.000Z [DEBUG]   "choices": [{"message": {"tool_calls": []}}],
2025-09-26T11:13:14.000Z [DEBUG]   "usage": {"prompt_tokens": 700, "completion_tokens": 20, "total_tokens": 720}
2025-09-26T11:13:14.000Z [DEBUG] }
2025-09-26T11:13:14.500Z [DEBUG] Executing tool: mcp_github
2025-09-26T11:13:15.000Z [DEBUG] Workflow done`

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "agent.log"), []byte(logContent), 0644))

	metrics, err := extractLogMetrics(tempDir, false)
	require.NoError(t, err)

	// Turns: 3 data blocks
	assert.Equal(t, 3, metrics.Turns, "Should count 3 turns from 3 data blocks")

	// Tool calls: bash x2, mcp_github x1
	assert.NotEmpty(t, metrics.ToolCalls, "ToolCalls should not be null")

	toolCounts := make(map[string]int)
	for _, tc := range metrics.ToolCalls {
		toolCounts[tc.Name] = tc.CallCount
	}
	assert.Equal(t, 2, toolCounts["bash"], "bash should have 2 calls")
	assert.Equal(t, 1, toolCounts["mcp_github"], "mcp_github should have 1 call")
}
