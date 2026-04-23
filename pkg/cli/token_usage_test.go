//go:build !integration

package cli

import (
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

		// Check cache efficiency
		expectedEfficiency := float64(55028) / float64(775+55028)
		assert.InDelta(t, expectedEfficiency, summary.CacheEfficiency, 0.001, "cache efficiency")
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
		assert.Equal(t, 243846, summary.TotalEffectiveTokens, "effective tokens should match agent usage")
		assert.Equal(t, 1, summary.TotalRequests, "agent usage fallback should synthesize one request")
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
}

func TestCacheEfficiency(t *testing.T) {
	t.Run("zero when no cache reads", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "cache-eff")
		filePath := filepath.Join(tmpDir, "token-usage.jsonl")
		content := `{"provider":"anthropic","model":"sonnet","input_tokens":100,"output_tokens":50,"cache_read_tokens":0,"cache_write_tokens":0,"duration_ms":100}`
		require.NoError(t, os.WriteFile(filePath, []byte(content+"\n"), 0o644))

		summary, err := parseTokenUsageFile(filePath, nil)
		require.NoError(t, err)
		require.NotNil(t, summary)
		assert.InDelta(t, 0.0, summary.CacheEfficiency, 0.001, "cache efficiency should be 0 with no cache reads")
	})

	t.Run("high efficiency with mostly cache reads", func(t *testing.T) {
		tmpDir := testutil.TempDir(t, "cache-eff")
		filePath := filepath.Join(tmpDir, "token-usage.jsonl")
		content := `{"provider":"anthropic","model":"sonnet","input_tokens":100,"output_tokens":50,"cache_read_tokens":9900,"cache_write_tokens":0,"duration_ms":100}`
		require.NoError(t, os.WriteFile(filePath, []byte(content+"\n"), 0o644))

		summary, err := parseTokenUsageFile(filePath, nil)
		require.NoError(t, err)
		require.NotNil(t, summary)
		assert.InDelta(t, 0.99, summary.CacheEfficiency, 0.001, "cache efficiency should be ~99%")
	})
}
