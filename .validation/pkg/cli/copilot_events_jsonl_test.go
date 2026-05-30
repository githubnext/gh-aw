//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFindEventsJSONLFile verifies that findEventsJSONLFile can locate an events.jsonl
// file both at the canonical copilot-session-state path and via recursive search.
func TestFindEventsJSONLFile(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(string) error
		expectFound    bool
		expectContains string // substring the found path must contain (optional)
	}{
		{
			name: "finds events.jsonl at canonical session-state path",
			setupFunc: func(dir string) error {
				sessionDir := filepath.Join(dir, "sandbox", "agent", "logs",
					"copilot-session-state", "be25adf7-5860-40ac-bfb6-2eb178a0f848")
				if err := os.MkdirAll(sessionDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(sessionDir, "events.jsonl"), []byte("{}"), 0644)
			},
			expectFound:    true,
			expectContains: "events.jsonl",
		},
		{
			name: "finds events.jsonl via recursive fallback",
			setupFunc: func(dir string) error {
				nestedDir := filepath.Join(dir, "some", "nested", "dir")
				if err := os.MkdirAll(nestedDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(nestedDir, "events.jsonl"), []byte("{}"), 0644)
			},
			expectFound:    true,
			expectContains: "events.jsonl",
		},
		{
			name: "returns empty string when events.jsonl is absent",
			setupFunc: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "other.log"), []byte("log content"), 0644)
			},
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, tt.setupFunc(dir), "setup should succeed")

			result := findEventsJSONLFile(dir)
			if tt.expectFound {
				assert.NotEmpty(t, result, "should find events.jsonl")
				if tt.expectContains != "" {
					assert.Contains(t, result, tt.expectContains, "path should contain expected substring")
				}
			} else {
				assert.Empty(t, result, "should not find events.jsonl")
			}
		})
	}
}

// realFormatEventsLine builds an events.jsonl line using the actual Copilot CLI format:
// each event has a top-level type, a nested data object, an id, and a timestamp.
func realFormatEventsLine(eventType string, dataJSON string) string {
	return `{"type":"` + eventType + `","data":` + dataJSON + `,"id":"test-id","timestamp":"2026-04-02T04:00:00.000Z"}`
}

// TestParseEventsJSONLFile verifies that parseEventsJSONLFile correctly extracts
// turns, tool calls, tool sequences, and token counts from events.jsonl using
// the real Copilot CLI format (nested data object, tool.execution_start events).
func TestParseEventsJSONLFile(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		wantTurns     int
		wantToolCalls []string // tool names that must appear in ToolCalls
		wantTokens    int      // expected TokenUsage
		wantSequences int      // number of tool sequences expected
		wantErr       bool
	}{
		{
			name: "full session with real format",
			content: realFormatEventsLine("session.start", `{"sessionId":"accf8264","copilotVersion":"1.0.15"}`) + "\n" +
				realFormatEventsLine("session.model_change", `{"newModel":"claude-sonnet-4.6"}`) + "\n" +
				realFormatEventsLine("user.message", `{"content":"Do the task","agentMode":"autopilot"}`) + "\n" +
				realFormatEventsLine("tool.execution_start", `{"toolCallId":"tc1","toolName":"bash","arguments":{"command":"ls"}}`) + "\n" +
				realFormatEventsLine("tool.execution_complete", `{"toolCallId":"tc1","success":true,"model":"claude-sonnet-4.6"}`) + "\n" +
				realFormatEventsLine("tool.execution_start", `{"toolCallId":"tc2","toolName":"read_file","arguments":{"path":"main.go"}}`) + "\n" +
				realFormatEventsLine("tool.execution_complete", `{"toolCallId":"tc2","success":true,"model":"claude-sonnet-4.6"}`) + "\n" +
				realFormatEventsLine("user.message", `{"content":"Now verify","agentMode":"autopilot"}`) + "\n" +
				realFormatEventsLine("tool.execution_start", `{"toolCallId":"tc3","toolName":"bash","arguments":{"command":"go test"}}`) + "\n" +
				realFormatEventsLine("session.shutdown", `{"shutdownType":"routine","totalPremiumRequests":2,"modelMetrics":{"claude-sonnet-4.6":{"requests":{"count":14,"cost":2},"usage":{"inputTokens":799195,"outputTokens":6148,"cacheReadTokens":721116,"cacheWriteTokens":0}}}}`) + "\n",
			wantTurns:     2,
			wantToolCalls: []string{"bash", "read_file"},
			wantTokens:    805343, // 799195 + 6148
			wantSequences: 2,
		},
		{
			name: "session with one turn and shutdown with modelMetrics",
			content: realFormatEventsLine("user.message", `{"content":"Do something"}`) + "\n" +
				realFormatEventsLine("tool.execution_start", `{"toolCallId":"tc1","toolName":"mcpscripts-gh","arguments":{}}`) + "\n" +
				realFormatEventsLine("session.shutdown", `{"shutdownType":"routine","totalPremiumRequests":3,"modelMetrics":{"claude-haiku-4.5":{"requests":{"count":1,"cost":0},"usage":{"inputTokens":5442,"outputTokens":457,"cacheReadTokens":0,"cacheWriteTokens":0}}}}`) + "\n",
			wantTurns:     1,
			wantToolCalls: []string{"mcpscripts-gh"},
			wantTokens:    5899, // 5442 + 457
			wantSequences: 1,
		},
		{
			name: "shutdown falls back to totalPremiumRequests when modelMetrics absent",
			content: realFormatEventsLine("user.message", `{"content":"Hello"}`) + "\n" +
				realFormatEventsLine("session.shutdown", `{"shutdownType":"routine","totalPremiumRequests":5}`) + "\n",
			wantTurns:  1,
			wantTokens: 5, // falls back to totalPremiumRequests
		},
		{
			name: "repeated tool calls aggregated by name",
			content: realFormatEventsLine("user.message", `{"content":"Run tests"}`) + "\n" +
				realFormatEventsLine("tool.execution_start", `{"toolCallId":"tc1","toolName":"bash","arguments":{}}`) + "\n" +
				realFormatEventsLine("tool.execution_start", `{"toolCallId":"tc2","toolName":"bash","arguments":{}}`) + "\n" +
				realFormatEventsLine("session.shutdown", `{"shutdownType":"routine","totalPremiumRequests":1,"modelMetrics":{"m":{"usage":{"inputTokens":100,"outputTokens":10}}}}`) + "\n",
			wantTurns:     1,
			wantToolCalls: []string{"bash"},
			wantTokens:    110,
			wantSequences: 1,
		},
		{
			name:    "empty file returns error",
			content: "",
			wantErr: true,
		},
		{
			name:    "file with no recognizable events returns error",
			content: "not json at all\njust plain text\n",
			wantErr: true,
		},
		{
			name: "malformed lines are skipped gracefully",
			content: realFormatEventsLine("user.message", `{"content":"Hello"}`) + "\n" +
				"{invalid json}\n" +
				realFormatEventsLine("session.shutdown", `{"totalPremiumRequests":2,"modelMetrics":{"m":{"usage":{"inputTokens":100,"outputTokens":20}}}}`) + "\n",
			wantTurns:  1,
			wantTokens: 120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			eventsPath := filepath.Join(dir, "events.jsonl")
			require.NoError(t, os.WriteFile(eventsPath, []byte(tt.content), 0644))

			metrics, err := parseEventsJSONLFile(eventsPath, false)

			if tt.wantErr {
				assert.Error(t, err, "expected an error")
				return
			}

			require.NoError(t, err, "should parse without error")
			assert.Equal(t, tt.wantTurns, metrics.Turns, "turns should match")
			assert.Equal(t, tt.wantTokens, metrics.TokenUsage, "token usage should match")

			if tt.wantSequences > 0 {
				assert.Len(t, metrics.ToolSequences, tt.wantSequences, "tool sequence count should match")
			}

			// Verify each expected tool name appears in ToolCalls
			toolNames := make(map[string]bool)
			for _, tc := range metrics.ToolCalls {
				toolNames[tc.Name] = true
			}
			for _, expectedTool := range tt.wantToolCalls {
				assert.True(t, toolNames[expectedTool], "tool %q should be in ToolCalls", expectedTool)
			}
		})
	}
}

// TestParseEventsJSONLFile_RealArtifact validates the parser against known metrics
// from the actual artifact in run 23883588837 (accf8264 session).
// Expected: 2 turns, 811242 total tokens (799195+6148 from claude-sonnet + 5442+457 from claude-haiku).
func TestParseEventsJSONLFile_RealArtifact(t *testing.T) {
	content := realFormatEventsLine("session.start", `{"sessionId":"accf8264","copilotVersion":"1.0.15"}`) + "\n" +
		realFormatEventsLine("user.message", `{"content":"task prompt"}`) + "\n" +
		realFormatEventsLine("tool.execution_start", `{"toolCallId":"t1","toolName":"report_intent","arguments":{}}`) + "\n" +
		realFormatEventsLine("tool.execution_start", `{"toolCallId":"t2","toolName":"github-list_pull_requests","arguments":{}}`) + "\n" +
		realFormatEventsLine("tool.execution_start", `{"toolCallId":"t3","toolName":"mcpscripts-gh","arguments":{}}`) + "\n" +
		realFormatEventsLine("tool.execution_start", `{"toolCallId":"t4","toolName":"web_fetch","arguments":{}}`) + "\n" +
		realFormatEventsLine("tool.execution_start", `{"toolCallId":"t5","toolName":"bash","arguments":{}}`) + "\n" +
		realFormatEventsLine("tool.execution_start", `{"toolCallId":"t6","toolName":"mcpscripts-github_discussion_query","arguments":{}}`) + "\n" +
		realFormatEventsLine("tool.execution_start", `{"toolCallId":"t7","toolName":"playwright-browser_navigate","arguments":{}}`) + "\n" +
		realFormatEventsLine("tool.execution_start", `{"toolCallId":"t8","toolName":"task","arguments":{}}`) + "\n" +
		realFormatEventsLine("tool.execution_start", `{"toolCallId":"t9","toolName":"safeoutputs-add_comment","arguments":{}}`) + "\n" +
		realFormatEventsLine("user.message", `{"content":"second turn"}`) + "\n" +
		realFormatEventsLine("tool.execution_start", `{"toolCallId":"t10","toolName":"bash","arguments":{}}`) + "\n" +
		realFormatEventsLine("session.shutdown", `{"shutdownType":"routine","totalPremiumRequests":2,"modelMetrics":{"claude-sonnet-4.6":{"requests":{"count":14,"cost":2},"usage":{"inputTokens":799195,"outputTokens":6148,"cacheReadTokens":721116,"cacheWriteTokens":0}},"claude-haiku-4.5":{"requests":{"count":1,"cost":0},"usage":{"inputTokens":5442,"outputTokens":457,"cacheReadTokens":0,"cacheWriteTokens":0}}}}`) + "\n"

	dir := t.TempDir()
	eventsPath := filepath.Join(dir, "events.jsonl")
	require.NoError(t, os.WriteFile(eventsPath, []byte(content), 0644))

	metrics, err := parseEventsJSONLFile(eventsPath, false)
	require.NoError(t, err, "should parse without error")

	assert.Equal(t, 2, metrics.Turns, "should detect 2 turns")
	// (799195+6148) + (5442+457) = 805343 + 5899 = 811242
	assert.Equal(t, 811242, metrics.TokenUsage, "should sum tokens from both models")

	// Should have 2 sequences (one per user.message)
	assert.Len(t, metrics.ToolSequences, 2, "should have 2 tool sequences")

	// bash appears in both turns
	toolCounts := make(map[string]int)
	for _, tc := range metrics.ToolCalls {
		toolCounts[tc.Name] = tc.CallCount
	}
	assert.Equal(t, 2, toolCounts["bash"], "bash should be called twice (once per turn)")
	assert.Equal(t, 1, toolCounts["report_intent"], "report_intent called once")
}

// TestExtractLogMetrics_EventsJSONLPriority verifies that extractLogMetrics uses
// events.jsonl as the primary source when it is present, and falls back to log
// file parsing when it is absent.
func TestExtractLogMetrics_EventsJSONLPriority(t *testing.T) {
	t.Run("uses events.jsonl when present", func(t *testing.T) {
		dir := t.TempDir()

		// Write aw_info.json so the engine is detected
		require.NoError(t, os.WriteFile(filepath.Join(dir, "aw_info.json"),
			[]byte(`{"engine_id":"copilot"}`), 0644))

		// Create events.jsonl in the canonical location
		sessionDir := filepath.Join(dir, "sandbox", "agent", "logs",
			"copilot-session-state", "session-uuid-123")
		require.NoError(t, os.MkdirAll(sessionDir, 0755))
		eventsContent :=
			realFormatEventsLine("session.start", `{"sessionId":"s1","copilotVersion":"1.0.0"}`) + "\n" +
				realFormatEventsLine("user.message", `{"content":"Do something"}`) + "\n" +
				realFormatEventsLine("tool.execution_start", `{"toolCallId":"tc1","toolName":"bash","arguments":{}}`) + "\n" +
				realFormatEventsLine("session.shutdown", `{"shutdownType":"routine","totalPremiumRequests":7,"modelMetrics":{"m":{"usage":{"inputTokens":100,"outputTokens":20}}}}`) + "\n"
		require.NoError(t, os.WriteFile(filepath.Join(sessionDir, "events.jsonl"),
			[]byte(eventsContent), 0644))

		// Also create a .log file that would give different metrics if parsed
		require.NoError(t, os.WriteFile(filepath.Join(dir, "agent-stdio.log"),
			[]byte("some log content without metrics"), 0644))

		metrics, err := extractLogMetrics(dir, false)
		require.NoError(t, err, "extractLogMetrics should not error")

		// Should use events.jsonl values: turns=1, tokenUsage=120 (100+20 from modelMetrics)
		assert.Equal(t, 1, metrics.Turns, "turns should come from events.jsonl")
		assert.Equal(t, 120, metrics.TokenUsage, "token usage should come from modelMetrics in events.jsonl")
	})

	t.Run("falls back to log file walk when events.jsonl absent", func(t *testing.T) {
		dir := t.TempDir()

		// Write aw_info.json for copilot engine
		require.NoError(t, os.WriteFile(filepath.Join(dir, "aw_info.json"),
			[]byte(`{"engine_id":"copilot"}`), 0644))

		// No events.jsonl – only a log file with parseable content
		logContent := `2025-09-26T11:13:11.798Z [DEBUG] Starting Copilot CLI
2025-09-26T11:13:12.575Z [DEBUG] data:
2025-09-26T11:13:12.575Z [DEBUG] {
2025-09-26T11:13:12.575Z [DEBUG]   "choices": [{"message": {"role": "assistant", "content": null, "tool_calls": []}}],
2025-09-26T11:13:12.575Z [DEBUG]   "usage": {"prompt_tokens": 1000, "completion_tokens": 50, "total_tokens": 1050}
2025-09-26T11:13:12.575Z [DEBUG] }
`
		require.NoError(t, os.WriteFile(filepath.Join(dir, "process-123.log"),
			[]byte(logContent), 0644))

		metrics, err := extractLogMetrics(dir, false)
		require.NoError(t, err, "extractLogMetrics should not error")

		// Should fall back to log parsing and find at least 1 turn
		assert.GreaterOrEqual(t, metrics.Turns, 1, "should detect at least one turn from log file")
	})
}

// TestParseEventsJSONLFile_TBT verifies that Time Between Turns is computed
// correctly from per-turn timestamps embedded in user.message events.
func TestParseEventsJSONLFile_TBT(t *testing.T) {
	// Helper that produces an events.jsonl line with a specific RFC3339 timestamp.
	lineWithTimestamp := func(eventType, dataJSON, timestamp string) string {
		return `{"type":"` + eventType + `","data":` + dataJSON + `,"id":"test-id","timestamp":"` + timestamp + `"}`
	}

	t.Run("computes avg and max TBT from multi-turn timestamps", func(t *testing.T) {
		// Turn 1 at T+0s, Turn 2 at T+30s, Turn 3 at T+90s → intervals: 30s, 60s → avg 45s, max 60s
		content := lineWithTimestamp("session.start", `{"sessionId":"s1","copilotVersion":"1.0.0"}`, "2026-01-01T10:00:00Z") + "\n" +
			lineWithTimestamp("user.message", `{"content":"turn 1"}`, "2026-01-01T10:00:00Z") + "\n" +
			lineWithTimestamp("user.message", `{"content":"turn 2"}`, "2026-01-01T10:00:30Z") + "\n" +
			lineWithTimestamp("user.message", `{"content":"turn 3"}`, "2026-01-01T10:01:30Z") + "\n" +
			lineWithTimestamp("session.shutdown", `{"shutdownType":"routine","totalPremiumRequests":3,"modelMetrics":{"m":{"usage":{"inputTokens":100,"outputTokens":10}}}}`, "2026-01-01T10:02:00Z") + "\n"

		dir := t.TempDir()
		eventsPath := filepath.Join(dir, "events.jsonl")
		require.NoError(t, os.WriteFile(eventsPath, []byte(content), 0644))

		metrics, err := parseEventsJSONLFile(eventsPath, false)
		require.NoError(t, err, "should parse without error")

		assert.Equal(t, 3, metrics.Turns, "should detect 3 turns")
		assert.Equal(t, 45*time.Second, metrics.AvgTimeBetweenTurns, "avg TBT should be 45s")
		assert.Equal(t, 60*time.Second, metrics.MaxTimeBetweenTurns, "max TBT should be 60s")
	})

	t.Run("no TBT when only one turn", func(t *testing.T) {
		content := lineWithTimestamp("user.message", `{"content":"single turn"}`, "2026-01-01T10:00:00Z") + "\n" +
			lineWithTimestamp("session.shutdown", `{"shutdownType":"routine","totalPremiumRequests":1,"modelMetrics":{"m":{"usage":{"inputTokens":50,"outputTokens":5}}}}`, "2026-01-01T10:01:00Z") + "\n"

		dir := t.TempDir()
		eventsPath := filepath.Join(dir, "events.jsonl")
		require.NoError(t, os.WriteFile(eventsPath, []byte(content), 0644))

		metrics, err := parseEventsJSONLFile(eventsPath, false)
		require.NoError(t, err, "should parse without error")

		assert.Equal(t, 1, metrics.Turns, "should detect 1 turn")
		assert.Zero(t, metrics.AvgTimeBetweenTurns, "no TBT with a single turn")
		assert.Zero(t, metrics.MaxTimeBetweenTurns, "no TBT with a single turn")
	})

	t.Run("no TBT when all timestamps are identical", func(t *testing.T) {
		// Use realFormatEventsLine which always uses the same timestamp — no intervals.
		content := realFormatEventsLine("user.message", `{"content":"turn 1"}`) + "\n" +
			realFormatEventsLine("user.message", `{"content":"turn 2"}`) + "\n" +
			realFormatEventsLine("session.shutdown", `{"shutdownType":"routine","totalPremiumRequests":2,"modelMetrics":{"m":{"usage":{"inputTokens":50,"outputTokens":5}}}}`) + "\n"

		dir := t.TempDir()
		eventsPath := filepath.Join(dir, "events.jsonl")
		require.NoError(t, os.WriteFile(eventsPath, []byte(content), 0644))

		metrics, err := parseEventsJSONLFile(eventsPath, false)
		require.NoError(t, err, "should parse without error")

		assert.Equal(t, 2, metrics.Turns, "should detect 2 turns")
		// All user.message events share the same timestamp → interval is 0 → TBT stays zero.
		assert.Zero(t, metrics.AvgTimeBetweenTurns, "TBT should be zero when all timestamps are identical")
	})
}
