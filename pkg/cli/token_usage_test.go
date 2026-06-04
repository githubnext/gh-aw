//go:build !integration

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTokenUsageFile(t *testing.T) {
	t.Run("valid single entry", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "token-usage")
		filePath := filepath.Join(tmpDir, "token-usage.jsonl")

		content := `{"timestamp":"2026-04-01T17:56:38.042Z","request_id":"abc-123","provider":"anthropic","model":"claude-sonnet-4-6","path":"/v1/messages","status":200,"streaming":true,"input_tokens":100,"output_tokens":200,"cache_read_tokens":5000,"cache_write_tokens":3000,"duration_ms":2500,"response_bytes":1500}`
		require.NoError(t, os.WriteFile(filePath, []byte(content+"\n"), 0o644), "should write test file")

		summary, err := parseTokenUsageFile(filePath, nil)
		require.NoError(t, err, "should parse without error")
		require.NotNil(t, summary, "should return non-nil summary")

		assert.Equal(t, 100, summary.TotalInputTokens, "input tokens")
		assert.Equal(t, 200, summary.TotalOutputTokens, "output tokens")
		assert.Equal(t, 5000, summary.TotalCacheReadTokens, "cache read tokens")
		assert.Equal(t, 3000, summary.TotalCacheWriteTokens, "cache write tokens")
		assert.Equal(t, 1, summary.TotalRequests, "total requests")
		assert.Equal(t, 2500, summary.TotalDurationMs, "total duration ms")
		assert.Equal(t, 1500, summary.TotalResponseBytes, "total response bytes")

		// Check by-model breakdown
		require.Contains(t, summary.ByModel, "claude-sonnet-4-6", "should have model entry")
		model := summary.ByModel["claude-sonnet-4-6"]
		assert.Equal(t, "anthropic", model.Provider, "model provider")
		assert.Equal(t, 100, model.InputTokens, "model input tokens")
		assert.Equal(t, 200, model.OutputTokens, "model output tokens")
	})

	t.Run("multiple entries with multiple models", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "token-usage")
		filePath := filepath.Join(tmpDir, "token-usage.jsonl")

		content := `{"timestamp":"2026-04-01T17:56:38.042Z","request_id":"1","provider":"anthropic","model":"claude-sonnet-4-6","path":"/v1/messages","status":200,"streaming":true,"input_tokens":3,"output_tokens":414,"cache_read_tokens":14044,"cache_write_tokens":26035,"duration_ms":6383,"response_bytes":2843}
{"timestamp":"2026-04-01T17:57:00.000Z","request_id":"2","provider":"anthropic","model":"claude-sonnet-4-6","path":"/v1/messages","status":200,"streaming":true,"input_tokens":3,"output_tokens":450,"cache_read_tokens":40984,"cache_write_tokens":0,"duration_ms":4000,"response_bytes":3000}
{"timestamp":"2026-04-01T17:58:00.000Z","request_id":"3","provider":"anthropic","model":"claude-haiku-4-5","path":"/v1/messages","status":200,"streaming":false,"input_tokens":769,"output_tokens":86,"cache_read_tokens":0,"cache_write_tokens":0,"duration_ms":700,"response_bytes":500}`
		require.NoError(t, os.WriteFile(filePath, []byte(content+"\n"), 0o644), "should write test file")

		summary, err := parseTokenUsageFile(filePath, nil)
		require.NoError(t, err, "should parse without error")
		require.NotNil(t, summary, "should return non-nil summary")

		assert.Equal(t, 775, summary.TotalInputTokens, "total input tokens")
		assert.Equal(t, 950, summary.TotalOutputTokens, "total output tokens")
		assert.Equal(t, 55028, summary.TotalCacheReadTokens, "total cache read tokens")
		assert.Equal(t, 26035, summary.TotalCacheWriteTokens, "total cache write tokens")
		assert.Equal(t, 3, summary.TotalRequests, "total requests")
		assert.Equal(t, 11083, summary.TotalDurationMs, "total duration ms")

		// Check by-model
		require.Len(t, summary.ByModel, 2, "should have 2 models")
		assert.Equal(t, 2, summary.ByModel["claude-sonnet-4-6"].Requests, "sonnet requests")
		assert.Equal(t, 1, summary.ByModel["claude-haiku-4-5"].Requests, "haiku requests")

		assert.InDelta(t, 0.0, summary.CacheEfficiency, 0.001, "cache efficiency is not computed from raw token counts")
	})

	t.Run("extracts ambient context from first chronological invocation", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "token-usage")
		filePath := filepath.Join(tmpDir, "token-usage.jsonl")

		content := `{"timestamp":"2026-04-01T17:58:00.000Z","request_id":"2","provider":"anthropic","model":"claude-sonnet-4-6","path":"/v1/messages","status":200,"streaming":true,"input_tokens":12,"output_tokens":10,"cache_read_tokens":99,"cache_write_tokens":0,"duration_ms":4000,"response_bytes":3000}
{"timestamp":"2026-04-01T17:56:00.000Z","request_id":"1","provider":"anthropic","model":"claude-sonnet-4-6","path":"/v1/messages","status":200,"streaming":true,"input_tokens":7,"output_tokens":5,"cache_read_tokens":3,"cache_write_tokens":0,"duration_ms":1000,"response_bytes":500}`
		require.NoError(t, os.WriteFile(filePath, []byte(content+"\n"), 0o644), "should write test file")

		summary, err := parseTokenUsageFile(filePath, nil)
		require.NoError(t, err, "should parse without error")
		require.NotNil(t, summary, "should return non-nil summary")
		require.NotNil(t, summary.AmbientContext, "ambient context should be present")
		assert.Equal(t, 7, summary.AmbientContext.InputTokens, "ambient input tokens should come from first invocation")
		assert.Equal(t, 3, summary.AmbientContext.CachedTokens, "ambient cached tokens should come from first invocation")
		assert.Equal(t, 10, summary.AmbientContext.EffectiveTokens, "ambient effective tokens should be input + cached")
	})

	t.Run("ambient context defaults cached tokens to zero when absent", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "token-usage")
		filePath := filepath.Join(tmpDir, "token-usage.jsonl")

		content := `{"timestamp":"2026-04-01T17:56:00.000Z","request_id":"1","provider":"anthropic","model":"claude-sonnet-4-6","path":"/v1/messages","status":200,"streaming":true,"input_tokens":11,"output_tokens":5,"duration_ms":1000,"response_bytes":500}`
		require.NoError(t, os.WriteFile(filePath, []byte(content+"\n"), 0o644), "should write test file")

		summary, err := parseTokenUsageFile(filePath, nil)
		require.NoError(t, err, "should parse without error")
		require.NotNil(t, summary, "should return non-nil summary")
		require.NotNil(t, summary.AmbientContext, "ambient context should be present")
		assert.Equal(t, 11, summary.AmbientContext.InputTokens, "ambient input tokens should match")
		assert.Equal(t, 0, summary.AmbientContext.CachedTokens, "missing cached tokens should default to zero")
		assert.Equal(t, 11, summary.AmbientContext.EffectiveTokens, "ambient effective tokens should fall back to input only")
	})

	t.Run("empty file returns nil", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "token-usage")
		filePath := filepath.Join(tmpDir, "token-usage.jsonl")
		require.NoError(t, os.WriteFile(filePath, []byte(""), 0o644))

		summary, err := parseTokenUsageFile(filePath, nil)
		require.NoError(t, err, "should not error on empty file")
		assert.Nil(t, summary, "should return nil for empty file")
	})

	t.Run("file with only blank lines returns nil", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "token-usage")
		filePath := filepath.Join(tmpDir, "token-usage.jsonl")
		require.NoError(t, os.WriteFile(filePath, []byte("\n\n\n"), 0o644))

		summary, err := parseTokenUsageFile(filePath, nil)
		require.NoError(t, err, "should not error on blank-only file")
		assert.Nil(t, summary, "should return nil for blank-only file")
	})

	t.Run("skips invalid JSON lines", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "token-usage")
		filePath := filepath.Join(tmpDir, "token-usage.jsonl")

		content := `not json
{"timestamp":"2026-04-01T17:56:38.042Z","request_id":"1","provider":"anthropic","model":"claude-sonnet-4-6","path":"/v1/messages","status":200,"streaming":true,"input_tokens":100,"output_tokens":200,"cache_read_tokens":0,"cache_write_tokens":0,"duration_ms":1000,"response_bytes":500}
also not json`
		require.NoError(t, os.WriteFile(filePath, []byte(content+"\n"), 0o644))

		summary, err := parseTokenUsageFile(filePath, nil)
		require.NoError(t, err, "should not error on mixed content")
		require.NotNil(t, summary, "should return summary from valid lines")
		assert.Equal(t, 1, summary.TotalRequests, "should count only valid entries")
		assert.Equal(t, 100, summary.TotalInputTokens, "input tokens from valid entry")
	})

	t.Run("file not found returns error", func(t *testing.T) {
		_, err := parseTokenUsageFile("/nonexistent/path/token-usage.jsonl", nil)
		assert.Error(t, err, "should error on missing file")
	})

	t.Run("entry with empty model uses unknown", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "token-usage")
		filePath := filepath.Join(tmpDir, "token-usage.jsonl")

		content := `{"timestamp":"2026-04-01T17:56:38.042Z","request_id":"1","provider":"anthropic","model":"","path":"/v1/messages","status":200,"streaming":true,"input_tokens":50,"output_tokens":25,"cache_read_tokens":0,"cache_write_tokens":0,"duration_ms":500,"response_bytes":200}`
		require.NoError(t, os.WriteFile(filePath, []byte(content+"\n"), 0o644))

		summary, err := parseTokenUsageFile(filePath, nil)
		require.NoError(t, err, "should parse without error")
		require.NotNil(t, summary, "should return non-nil summary")
		require.Contains(t, summary.ByModel, "unknown", "should use 'unknown' for empty model")
	})
}

func TestFindTokenUsageFile(t *testing.T) {
	t.Run("finds in sandbox/firewall/logs path", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "find-token-usage")
		logsDir := filepath.Join(tmpDir, "sandbox", "firewall", "logs", "api-proxy-logs")
		require.NoError(t, os.MkdirAll(logsDir, 0o755))
		tokenFile := filepath.Join(logsDir, "token-usage.jsonl")
		require.NoError(t, os.WriteFile(tokenFile, []byte(`{"input_tokens":1}`+"\n"), 0o644))

		result := findTokenUsageFile(tmpDir)
		assert.Equal(t, tokenFile, result, "should find file in primary path")
	})

	t.Run("finds in firewall-audit-logs directory", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "find-token-usage")
		logsDir := filepath.Join(tmpDir, "firewall-audit-logs", "api-proxy-logs")
		require.NoError(t, os.MkdirAll(logsDir, 0o755))
		tokenFile := filepath.Join(logsDir, "token-usage.jsonl")
		require.NoError(t, os.WriteFile(tokenFile, []byte(`{"input_tokens":1}`+"\n"), 0o644))

		result := findTokenUsageFile(tmpDir)
		assert.Equal(t, tokenFile, result, "should find file in firewall-audit-logs")
	})

	t.Run("returns empty string when not found", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "find-token-usage")
		result := findTokenUsageFile(tmpDir)
		assert.Empty(t, result, "should return empty string when file not found")
	})
}

func TestTokenUsageSummaryMethods(t *testing.T) {
	t.Run("TotalTokens", func(t *testing.T) {
		summary := &TokenUsageSummary{
			TotalInputTokens:      100,
			TotalOutputTokens:     200,
			TotalCacheReadTokens:  5000,
			TotalCacheWriteTokens: 3000,
		}
		assert.Equal(t, 8300, summary.TotalTokens(), "total tokens should be sum of all types")
	})

	t.Run("AvgDurationMs", func(t *testing.T) {
		summary := &TokenUsageSummary{
			TotalDurationMs: 10000,
			TotalRequests:   4,
		}
		assert.Equal(t, 2500, summary.AvgDurationMs(), "avg duration should be total/requests")
	})

	t.Run("AvgDurationMs with zero requests", func(t *testing.T) {
		summary := &TokenUsageSummary{
			TotalDurationMs: 10000,
			TotalRequests:   0,
		}
		assert.Equal(t, 0, summary.AvgDurationMs(), "avg duration should be 0 for zero requests")
	})

	t.Run("ModelRows sorted by total tokens", func(t *testing.T) {
		summary := &TokenUsageSummary{
			ByModel: map[string]*ModelTokenUsage{
				"small-model": {
					Provider:    "provider-a",
					InputTokens: 10,
					Requests:    1,
					DurationMs:  100,
				},
				"large-model": {
					Provider:         "provider-b",
					InputTokens:      100,
					OutputTokens:     200,
					CacheReadTokens:  5000,
					CacheWriteTokens: 3000,
					Requests:         5,
					DurationMs:       5000,
				},
			},
		}

		rows := summary.ModelRows()
		require.Len(t, rows, 2, "should have 2 model rows")
		assert.Equal(t, "large-model", rows[0].Model, "first row should be model with most tokens")
		assert.Equal(t, "small-model", rows[1].Model, "second row should be model with fewer tokens")
		assert.Equal(t, "1.0s", rows[0].AvgDuration, "avg duration for large model")
	})
}

func TestFormatDurationMs(t *testing.T) {
	tests := []struct {
		ms       int
		expected string
	}{
		{0, "0ms"},
		{500, "500ms"},
		{999, "999ms"},
		{1000, "1.0s"},
		{1500, "1.5s"},
		{6383, "6.4s"},
		{59999, "60.0s"},
		{60000, "1m0s"},
		{90000, "1m30s"},
		{125000, "2m5s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, timeutil.FormatDurationMs(tt.ms), "FormatDurationMs(%d)", tt.ms)
		})
	}
}

func TestAnalyzeTokenUsage(t *testing.T) {
	t.Run("returns nil when no file found", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "analyze-token-usage")
		summary, err := analyzeTokenUsage(tmpDir, false)
		require.NoError(t, err, "should not error when file not found")
		assert.Nil(t, summary, "should return nil when no file found")
	})

	t.Run("parses file from sandbox path", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "analyze-token-usage")
		logsDir := filepath.Join(tmpDir, "sandbox", "firewall", "logs", "api-proxy-logs")
		require.NoError(t, os.MkdirAll(logsDir, 0o755))
		tokenFile := filepath.Join(logsDir, "token-usage.jsonl")
		content := `{"timestamp":"2026-04-01T17:56:38.042Z","request_id":"1","provider":"anthropic","model":"claude-sonnet-4-6","path":"/v1/messages","status":200,"streaming":true,"input_tokens":100,"output_tokens":200,"cache_read_tokens":5000,"cache_write_tokens":3000,"duration_ms":2500,"response_bytes":1500}`
		require.NoError(t, os.WriteFile(tokenFile, []byte(content+"\n"), 0o644))

		summary, err := analyzeTokenUsage(tmpDir, false)
		require.NoError(t, err, "should parse without error")
		require.NotNil(t, summary, "should return summary")
		assert.Equal(t, 1, summary.TotalRequests, "should have 1 request")
		assert.Equal(t, 100, summary.TotalInputTokens, "should have correct input tokens")
	})

	t.Run("falls back to agent_usage.json when token-usage.jsonl is missing", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "analyze-agent-usage")
		agentUsageFile := filepath.Join(tmpDir, "agent_usage.json")
		content := `{"input_tokens":5944,"output_tokens":8698,"cache_read_tokens":1170605,"cache_write_tokens":86049,"effective_tokens":243846}`
		require.NoError(t, os.WriteFile(agentUsageFile, []byte(content), 0o644))

		summary, err := analyzeTokenUsage(tmpDir, false)
		require.NoError(t, err, "should parse agent_usage.json without error")
		require.NotNil(t, summary, "should return summary from agent_usage.json")
		assert.Equal(t, 5944, summary.TotalInputTokens, "input tokens should match agent usage")
		assert.Equal(t, 8698, summary.TotalOutputTokens, "output tokens should match agent usage")
		assert.Equal(t, 237902, summary.TotalEffectiveTokens, "effective tokens should be recomputed from raw usage, not copied from stored ET")
		assert.Equal(t, 1, summary.TotalRequests, "agent usage fallback should synthesize one request")
	})

	t.Run("recomputes ET from raw usage even when agent_usage has stale effective_tokens", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "analyze-agent-usage-recompute")
		agentUsageFile := filepath.Join(tmpDir, "agent_usage.json")
		content := `{"model":"unknown","input_tokens":10,"output_tokens":5,"cache_read_tokens":0,"cache_write_tokens":0,"effective_tokens":9999}`
		require.NoError(t, os.WriteFile(agentUsageFile, []byte(content), 0o644))

		summary, err := analyzeTokenUsage(tmpDir, false)
		require.NoError(t, err, "should parse agent_usage.json without error")
		require.NotNil(t, summary, "should return summary from agent_usage.json")
		assert.Equal(t, 30, summary.TotalEffectiveTokens, "ET should be recomputed from raw usage")
		require.Contains(t, summary.ByModel, "unknown")
		assert.Equal(t, 30, summary.ByModel["unknown"].EffectiveTokens, "per-model ET should ignore stored effective_tokens when raw exists")
	})

	t.Run("unknown model falls back to multiplier 1.0 when recomputing ET", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "analyze-agent-usage-unknown-model")
		awInfoFile := filepath.Join(tmpDir, "aw_info.json")
		awInfoContent := `{"token_weights":{"multipliers":{"known-model":5}}}`
		require.NoError(t, os.WriteFile(awInfoFile, []byte(awInfoContent), 0o644))

		agentUsageFile := filepath.Join(tmpDir, "agent_usage.json")
		agentUsageContent := `{"model":"mystery-model","input_tokens":10,"output_tokens":5,"cache_read_tokens":0,"cache_write_tokens":0}`
		require.NoError(t, os.WriteFile(agentUsageFile, []byte(agentUsageContent), 0o644))

		summary, err := analyzeTokenUsage(tmpDir, false)
		require.NoError(t, err, "should parse agent_usage.json with unknown model")
		require.NotNil(t, summary, "should return summary from agent_usage.json")
		assert.Equal(t, 30, summary.TotalEffectiveTokens, "unknown model should use 1.0 multiplier fallback")
		require.Contains(t, summary.ByModel, "mystery-model")
		assert.Equal(t, 30, summary.ByModel["mystery-model"].EffectiveTokens, "per-model ET should use fallback multiplier")
	})

	t.Run("applies custom weights from aw_info when agent_usage effective_tokens is missing", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "analyze-agent-usage-custom-weights")
		awInfoFile := filepath.Join(tmpDir, "aw_info.json")
		awInfoContent := `{"token_weights":{"multipliers":{"unknown":2}}}`
		require.NoError(t, os.WriteFile(awInfoFile, []byte(awInfoContent), 0o644))

		agentUsageFile := filepath.Join(tmpDir, "agent_usage.json")
		agentUsageContent := `{"input_tokens":10,"output_tokens":5,"cache_read_tokens":0,"cache_write_tokens":0}`
		require.NoError(t, os.WriteFile(agentUsageFile, []byte(agentUsageContent), 0o644))

		summary, err := analyzeTokenUsage(tmpDir, false)
		require.NoError(t, err, "should parse agent_usage.json with custom weights")
		require.NotNil(t, summary, "should return summary from agent_usage.json")
		assert.Equal(t, 60, summary.TotalEffectiveTokens, "custom multiplier should be applied to computed effective tokens")
		require.Contains(t, summary.ByModel, "unknown", "unknown model bucket should be present")
		assert.Equal(t, 60, summary.ByModel["unknown"].EffectiveTokens, "per-model effective tokens should use custom weights")
	})

	t.Run("records requested sub-agent models and mismatch when token logs do not show requested model", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "analyze-subagent-model-mismatch")
		logsDir := filepath.Join(tmpDir, "sandbox", "firewall", "logs", "api-proxy-logs")
		require.NoError(t, os.MkdirAll(logsDir, 0o755))
		tokenFile := filepath.Join(logsDir, "token-usage.jsonl")
		tokenContent := `{"timestamp":"2026-04-01T17:56:38.042Z","request_id":"1","provider":"anthropic","model":"claude-sonnet-4-6","path":"/v1/messages","status":200,"streaming":true,"input_tokens":100,"output_tokens":200,"cache_read_tokens":0,"cache_write_tokens":0,"duration_ms":2500,"response_bytes":1500}`
		require.NoError(t, os.WriteFile(tokenFile, []byte(tokenContent+"\n"), 0o644))

		agentLogContent := `● Agent-alpha(claude-haiku-4.5) Get model name
● Agent-beta(claude-haiku-4.5) Get model name
● Agent-gamma(claude-haiku-4.5) Get model name`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "agent-stdio.log"), []byte(agentLogContent), 0o644))

		summary, err := analyzeTokenUsage(tmpDir, false)
		require.NoError(t, err)
		require.NotNil(t, summary)
		require.Len(t, summary.SubagentModelRequests, 3)
		require.Len(t, summary.SubagentModelActuals, 1)
		assert.Equal(t, 3, summary.MismatchCount)
		assert.Equal(t, "claude-sonnet-4-6", summary.SubagentModelActuals[0].Model)
		require.Contains(t, summary.Warnings, subagentStdioWarning)

		for _, req := range summary.SubagentModelRequests {
			assert.Equal(t, "claude-haiku-4.5", req.RequestedModel)
			assert.Equal(t, 1, req.InvocationCount)
			assert.Equal(t, "claude-sonnet-4-6", req.EffectiveModel)
			assert.Equal(t, modelMismatchReasonModelNotObserved, req.ReasonCode)
		}
	})

	t.Run("records token-usage-missing reason when sub-agent model request is present but no model actuals exist", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "analyze-subagent-model-token-missing")
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "agent_usage.json"), []byte(`{}`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "agent-stdio.log"), []byte(`● Agent-alpha(claude-haiku-4.5) Get model name`), 0o644))

		summary, err := analyzeTokenUsage(tmpDir, false)
		require.NoError(t, err)
		require.NotNil(t, summary)
		require.Len(t, summary.SubagentModelRequests, 1)
		assert.Empty(t, summary.SubagentModelActuals)
		assert.Equal(t, 1, summary.MismatchCount)
		assert.Equal(t, modelMismatchReasonTokenUsageMissing, summary.SubagentModelRequests[0].ReasonCode)
		assert.Empty(t, summary.SubagentModelRequests[0].EffectiveModel)
		require.Contains(t, summary.Warnings, subagentStdioWarning)
	})

	t.Run("captures alias-based sub-agent model requests used by workflow subagents", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "analyze-subagent-model-alias")
		logsDir := filepath.Join(tmpDir, "sandbox", "firewall", "logs", "api-proxy-logs")
		require.NoError(t, os.MkdirAll(logsDir, 0o755))
		tokenFile := filepath.Join(logsDir, "token-usage.jsonl")
		tokenContent := `{"timestamp":"2026-04-01T17:56:38.042Z","request_id":"1","provider":"openai","model":"gpt-5-mini","path":"/v1/messages","status":200,"streaming":true,"input_tokens":100,"output_tokens":200,"cache_read_tokens":0,"cache_write_tokens":0,"duration_ms":2500,"response_bytes":1500}`
		require.NoError(t, os.WriteFile(tokenFile, []byte(tokenContent+"\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "agent-stdio.log"), []byte(`● workflow-characterizer(small) Classify`), 0o644))

		summary, err := analyzeTokenUsage(tmpDir, false)
		require.NoError(t, err)
		require.NotNil(t, summary)
		require.Len(t, summary.SubagentModelRequests, 1)
		assert.Equal(t, "small", summary.SubagentModelRequests[0].RequestedModel)
		assert.Equal(t, "gpt-5-mini", summary.SubagentModelRequests[0].EffectiveModel)
		assert.Equal(t, modelMismatchReasonModelNotObserved, summary.SubagentModelRequests[0].ReasonCode)
		assert.Equal(t, 1, summary.MismatchCount)
		require.Contains(t, summary.Warnings, subagentStdioWarning)
	})
}

func TestCorrelateToolCallsWithTokenDelta(t *testing.T) {
	t.Run("assigns delta to tool calls bracketed by API calls", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "token-delta")
		filePath := filepath.Join(tmpDir, "token-usage.jsonl")
		// Two API calls; tool call happens between them.
		// ET for first entry (model "unknown", default weights, m=1):
		//   1.0*1000 + 4.0*50 = 1200
		// ET for second entry:
		//   1.0*1500 + 4.0*80 = 1820
		// Expected delta = 1820 - 1200 = 620
		content := `{"timestamp":"2026-05-19T21:10:00.000Z","model":"unknown","provider":"test","input_tokens":1000,"output_tokens":50,"cache_read_tokens":0,"cache_write_tokens":0}
{"timestamp":"2026-05-19T21:10:10.000Z","model":"unknown","provider":"test","input_tokens":1500,"output_tokens":80,"cache_read_tokens":0,"cache_write_tokens":0}`
		require.NoError(t, os.WriteFile(filePath, []byte(content+"\n"), 0o644))

		toolCalls := []MCPToolCall{
			{
				Timestamp:  "2026-05-19T21:10:05.000Z",
				ServerName: "test-server",
				ToolName:   "test-tool",
			},
		}
		result := correlateToolCallsWithTokenDelta(toolCalls, filePath)
		require.Len(t, result, 1)
		assert.Equal(t, 620, result[0].EffectiveTokenDelta, "expected delta = ET(next) - ET(prev)")
	})

	t.Run("leaves delta zero when tool call has no preceding API call", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "token-delta-no-prev")
		filePath := filepath.Join(tmpDir, "token-usage.jsonl")
		content := `{"timestamp":"2026-05-19T21:10:10.000Z","model":"unknown","provider":"test","input_tokens":1000,"output_tokens":50,"cache_read_tokens":0,"cache_write_tokens":0}`
		require.NoError(t, os.WriteFile(filePath, []byte(content+"\n"), 0o644))

		toolCalls := []MCPToolCall{
			{
				Timestamp:  "2026-05-19T21:10:05.000Z", // before the only API call
				ServerName: "test-server",
				ToolName:   "test-tool",
			},
		}
		result := correlateToolCallsWithTokenDelta(toolCalls, filePath)
		require.Len(t, result, 1)
		assert.Equal(t, 0, result[0].EffectiveTokenDelta, "no delta when no preceding API call")
	})

	t.Run("leaves delta zero when tool call has no following API call", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "token-delta-no-next")
		filePath := filepath.Join(tmpDir, "token-usage.jsonl")
		content := `{"timestamp":"2026-05-19T21:10:00.000Z","model":"unknown","provider":"test","input_tokens":1000,"output_tokens":50,"cache_read_tokens":0,"cache_write_tokens":0}`
		require.NoError(t, os.WriteFile(filePath, []byte(content+"\n"), 0o644))

		toolCalls := []MCPToolCall{
			{
				Timestamp:  "2026-05-19T21:10:05.000Z", // after the only API call
				ServerName: "test-server",
				ToolName:   "test-tool",
			},
		}
		result := correlateToolCallsWithTokenDelta(toolCalls, filePath)
		require.Len(t, result, 1)
		assert.Equal(t, 0, result[0].EffectiveTokenDelta, "no delta when no following API call")
	})

	t.Run("handles empty token usage file path", func(t *testing.T) {
		toolCalls := []MCPToolCall{{Timestamp: "2026-05-19T21:10:05.000Z", ToolName: "t"}}
		result := correlateToolCallsWithTokenDelta(toolCalls, "")
		require.Len(t, result, 1)
		assert.Equal(t, 0, result[0].EffectiveTokenDelta, "no delta with empty file path")
	})

	t.Run("assigns correct deltas to multiple sequential tool calls", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "token-delta-multi")
		filePath := filepath.Join(tmpDir, "token-usage.jsonl")
		// Three API calls, two tool calls between consecutive pairs.
		content := `{"timestamp":"2026-05-19T21:10:00.000Z","model":"unknown","provider":"test","input_tokens":1000,"output_tokens":50,"cache_read_tokens":0,"cache_write_tokens":0}
{"timestamp":"2026-05-19T21:10:10.000Z","model":"unknown","provider":"test","input_tokens":1500,"output_tokens":80,"cache_read_tokens":0,"cache_write_tokens":0}
{"timestamp":"2026-05-19T21:10:20.000Z","model":"unknown","provider":"test","input_tokens":2000,"output_tokens":100,"cache_read_tokens":0,"cache_write_tokens":0}`
		require.NoError(t, os.WriteFile(filePath, []byte(content+"\n"), 0o644))
		// ET[0] = 1000 + 4*50 = 1200
		// ET[1] = 1500 + 4*80 = 1820  → delta1 = 620
		// ET[2] = 2000 + 4*100 = 2400 → delta2 = 580

		toolCalls := []MCPToolCall{
			{Timestamp: "2026-05-19T21:10:05.000Z", ServerName: "s", ToolName: "tool-a"},
			{Timestamp: "2026-05-19T21:10:15.000Z", ServerName: "s", ToolName: "tool-b"},
		}
		result := correlateToolCallsWithTokenDelta(toolCalls, filePath)
		require.Len(t, result, 2)
		assert.Equal(t, 620, result[0].EffectiveTokenDelta, "delta for tool-a")
		assert.Equal(t, 580, result[1].EffectiveTokenDelta, "delta for tool-b")
	})
}

func TestCacheEfficiency(t *testing.T) {
	t.Run("remains zero to avoid transforming raw token counts", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "cache-eff")
		filePath := filepath.Join(tmpDir, "token-usage.jsonl")
		content := `{"provider":"anthropic","model":"sonnet","input_tokens":100,"output_tokens":50,"cache_read_tokens":9900,"cache_write_tokens":0,"duration_ms":100}`
		require.NoError(t, os.WriteFile(filePath, []byte(content+"\n"), 0o644))

		summary, err := parseTokenUsageFile(filePath, nil)
		require.NoError(t, err)
		require.NotNil(t, summary)
		assert.InDelta(t, 0.0, summary.CacheEfficiency, 0.001, "cache efficiency should remain unset")
	})
}

func TestModelTokenUsageReasoningTokensJSONRoundTrip(t *testing.T) {
	original := ModelTokenUsage{
		Provider:        "anthropic",
		InputTokens:     10,
		OutputTokens:    20,
		ReasoningTokens: 30,
	}

	raw, err := json.Marshal(original)
	require.NoError(t, err)
	var encoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &encoded))
	assert.EqualValues(t, 30, encoded["reasoning_tokens"], "reasoning tokens should be persisted for ET recomputation")

	var decoded ModelTokenUsage
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, 30, decoded.ReasoningTokens, "reasoning tokens should survive JSON round-trip")
}
