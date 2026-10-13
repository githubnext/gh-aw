//go:build !integration

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const steeringNoticesTokenUsageJSONL = `{"timestamp":"2026-10-10T10:00:00Z","event":"token_usage","request_id":"req-1","purpose":"agent","provider":"copilot","model":"claude-sonnet-5","path":"/v1/messages","status":200,"input_tokens":10,"output_tokens":5,"aic":1.5}
{"timestamp":"2026-10-10T10:00:01Z","event":"token_usage","request_id":"req-2","purpose":"agent","provider":"copilot","model":"claude-sonnet-5","path":"/v1/messages","status":200,"input_tokens":10,"output_tokens":5,"aic":1.5,"steering":{"type":"ai_credit","threshold":80}}
{"timestamp":"2026-10-10T10:00:02Z","event":"token_usage","request_id":"req-3","purpose":"agent","provider":"copilot","model":"claude-sonnet-5","path":"/v1/messages","status":200,"input_tokens":10,"output_tokens":5,"aic":1.5,"steering":{"type":"timeout","threshold":90}}
`

func steeringNoticesSessionEvent(t *testing.T, phase, requestID string, steering any) string {
	t.Helper()
	data := map[string]any{
		"provider":  "copilot",
		"model":     "claude-sonnet-5",
		"path":      "/v1/messages",
		"requestId": requestID,
		"status":    200,
		"aic":       1.5,
		"usage":     map[string]any{"inputTokens": 10, "outputTokens": 5},
	}
	if steering != nil {
		data["steering"] = steering
	}
	path := "sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl"
	if phase == "detection" {
		path = "threat-detection/" + path
	}
	event := map[string]any{
		"type":       "firewall.token_usage",
		"timestamp":  "2026-10-10T10:00:00Z",
		"data":       data,
		"provenance": map[string]any{"component": "firewall", "phase": phase, "path": path},
	}
	encoded, err := json.Marshal(event)
	require.NoError(t, err)
	return string(encoded)
}

func writeSteeringNoticesRun(t *testing.T, sessionLines []string) string {
	t.Helper()
	runDir := testutil.TempDir(t, "steering-notices")
	logsDir := filepath.Join(runDir, "sandbox", "firewall", "logs", "api-proxy-logs")
	require.NoError(t, os.MkdirAll(logsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(logsDir, "token-usage.jsonl"), []byte(steeringNoticesTokenUsageJSONL), 0o644))
	if sessionLines != nil {
		usageDir := filepath.Join(runDir, "usage")
		require.NoError(t, os.MkdirAll(usageDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(usageDir, "aw_session.jsonl"), []byte(strings.Join(sessionLines, "\n")+"\n"), 0o644))
	}
	return runDir
}

func TestAuditSteeringFromUnifiedSession(t *testing.T) {
	runDir := writeSteeringNoticesRun(t, []string{
		steeringNoticesSessionEvent(t, "agent", "req-2", map[string]any{"type": "ai_credit", "threshold": 80}),
		`{"type":"firewall.steering","timestamp":"2026-10-10T10:00:01Z","data":{"event":"ai_credit_steering","message":"[AWF AI CREDIT WARNING] budget","requestId":"req-2","threshold":80},"provenance":{"component":"firewall","phase":"agent"}}`,
		`{"type":"firewall.steering","timestamp":"2026-10-10T10:00:02Z","data":{"event":"timeout_steering","message":"[AWF TIME WARNING] time","requestId":"det-1","threshold":90},"provenance":{"component":"firewall","phase":"detection"}}`,
		`{"type":"firewall.steering","data":{"event":"model_steering","message":"other steering"},"provenance":{"component":"firewall","phase":"agent"}}`,
		`{"type":"firewall.steering","data":{"event":"token_steering","message":"[AWF TOKEN WARNING] ignored"},"provenance":{"component":"agent","phase":"agent"}}`,
	})
	summary, err := analyzeTokenUsage(runDir, false)
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, 3, summary.TotalSteeringEvents)
	assert.Equal(t, map[string]int{"ai_credit_steering": 1, "timeout_steering": 1, "model_steering": 1}, summary.SteeringEventCounts)
	require.Len(t, summary.SteeringNotices, 1)

	events, err := extractGatewaySteeringEvents(runDir)
	require.NoError(t, err)
	assert.Equal(t, []GatewaySteeringEvent{
		{Type: aiCreditSteeringEventName, Message: "[AWF AI CREDIT WARNING] budget", Timestamp: "2026-10-10T10:00:01Z", RequestID: "req-2", Threshold: 80},
		{Type: timeoutSteeringEventName, Message: "[AWF TIME WARNING] time", Timestamp: "2026-10-10T10:00:02Z", RequestID: "det-1", Threshold: 90},
	}, events)
	audit, _ := buildLocalAuditData(ProcessedRun{
		Run:                   WorkflowRun{DatabaseID: 7, WorkflowName: "wf", LogsPath: runDir},
		TokenUsage:            summary,
		GatewaySteeringEvents: events,
	}, LogMetrics{}, nil)
	assert.Equal(t, events, audit.GatewaySteeringEvents)
	assert.Equal(t, 3, audit.FirewallTokenUsage.TotalSteeringEvents)

	rawRunDir := writeSteeringNoticesRun(t, []string{
		`{"type":"firewall.steering","data":{"event":"timeout_steering","message":"[AWF TIME WARNING] session"},"provenance":{"component":"firewall","phase":"agent"}}`,
	})
	rawPath := filepath.Join(rawRunDir, "sandbox", "firewall", "logs", "api-proxy-logs", "events.jsonl")
	require.NoError(t, os.WriteFile(rawPath, []byte(`{"event":"token_steering","message":"[AWF TOKEN WARNING] raw"}`+"\n"), 0o644))
	summary, err = analyzeTokenUsage(rawRunDir, false)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"token_steering": 1}, summary.SteeringEventCounts, "raw and unified copies must not be double-counted")
}

func TestAnalyzeTokenUsageSteeringNotices(t *testing.T) {
	t.Run("lists notices from unified firewall.token_usage events", func(t *testing.T) {
		runDir := writeSteeringNoticesRun(t, []string{
			steeringNoticesSessionEvent(t, "agent", "req-1", nil),
			steeringNoticesSessionEvent(t, "agent", "req-2", map[string]any{"type": "ai_credit", "threshold": 80}),
			steeringNoticesSessionEvent(t, "agent", "req-3", map[string]any{"type": "timeout", "threshold": 90}),
			steeringNoticesSessionEvent(t, "detection", "det-1", map[string]any{"type": "token", "threshold": 95}),
		})

		summary, err := analyzeTokenUsage(runDir, false)
		require.NoError(t, err)
		require.NotNil(t, summary)
		assert.Equal(t, []SteeringNotice{
			{Type: "ai_credit", Threshold: 80, RequestID: "req-2", Phase: "agent"},
			{Type: "timeout", Threshold: 90, RequestID: "req-3", Phase: "agent"},
			{Type: "token", Threshold: 95, RequestID: "det-1", Phase: "detection"},
		}, summary.SteeringNotices)
		assert.Equal(t, 3, summary.TotalRequests, "detection events should not change agent token usage")
	})

	t.Run("no steering fields produce no notices", func(t *testing.T) {
		runDir := writeSteeringNoticesRun(t, []string{
			steeringNoticesSessionEvent(t, "agent", "req-1", nil),
			steeringNoticesSessionEvent(t, "agent", "req-2", nil),
			steeringNoticesSessionEvent(t, "agent", "req-3", nil),
		})

		summary, err := analyzeTokenUsage(runDir, false)
		require.NoError(t, err)
		require.NotNil(t, summary)
		assert.Nil(t, summary.SteeringNotices)
		encoded, err := json.Marshal(summary)
		require.NoError(t, err)
		assert.NotContains(t, string(encoded), "steering_notices")
	})

	t.Run("runs without a unified session list no notices", func(t *testing.T) {
		runDir := writeSteeringNoticesRun(t, nil)

		summary, err := analyzeTokenUsage(runDir, false)
		require.NoError(t, err)
		require.NotNil(t, summary)
		assert.Nil(t, summary.SteeringNotices, "notices are read only from the unified session")
	})

	t.Run("ignores malformed or unknown steering values", func(t *testing.T) {
		runDir := writeSteeringNoticesRun(t, []string{
			steeringNoticesSessionEvent(t, "agent", "req-1", "ai_credit"),
			steeringNoticesSessionEvent(t, "agent", "req-2", map[string]any{"type": "unknown", "threshold": 80}),
			steeringNoticesSessionEvent(t, "agent", "req-3", map[string]any{"type": "token", "threshold": "high"}),
		})

		summary, err := analyzeTokenUsage(runDir, false)
		require.NoError(t, err)
		require.NotNil(t, summary)
		assert.Nil(t, summary.SteeringNotices)
		assert.Equal(t, 3, summary.TotalRequests, "malformed steering values must not drop token usage")
	})

	t.Run("rejects thresholds that are not exactly 80, 90, 95 or 99", func(t *testing.T) {
		runDir := writeSteeringNoticesRun(t, []string{
			steeringNoticesSessionEvent(t, "agent", "req-1", map[string]any{"type": "token", "threshold": 79.6}),
			steeringNoticesSessionEvent(t, "agent", "req-2", map[string]any{"type": "timeout", "threshold": 94.6}),
			steeringNoticesSessionEvent(t, "agent", "req-3", map[string]any{"type": "ai_credit", "threshold": 100}),
			steeringNoticesSessionEvent(t, "agent", "req-4", map[string]any{"type": "token", "threshold": "90"}),
		})

		summary, err := analyzeTokenUsage(runDir, false)
		require.NoError(t, err)
		require.NotNil(t, summary)
		assert.Nil(t, summary.SteeringNotices)
		assert.False(t, summary.SteeringNoticesUnavailable)
	})

	t.Run("truncated unified session marks notices unavailable", func(t *testing.T) {
		valid := steeringNoticesSessionEvent(t, "agent", "req-2", map[string]any{"type": "ai_credit", "threshold": 80})
		truncated := steeringNoticesSessionEvent(t, "agent", "req-3", map[string]any{"type": "timeout", "threshold": 90})
		runDir := writeSteeringNoticesRun(t, []string{valid, truncated[:len(truncated)/2]})

		summary, err := analyzeTokenUsage(runDir, false)
		require.NoError(t, err)
		require.NotNil(t, summary)
		assert.Nil(t, summary.SteeringNotices)
		assert.True(t, summary.SteeringNoticesUnavailable)
		require.NotEmpty(t, summary.Warnings)
		assert.Contains(t, strings.Join(summary.Warnings, "\n"), "steering notices unavailable")

		processed := ProcessedRun{
			Run:        WorkflowRun{DatabaseID: 9, WorkflowName: "wf", LogsPath: runDir},
			TokenUsage: summary,
		}
		audit, _ := buildLocalAuditData(processed, LogMetrics{}, nil)
		assert.True(t, audit.SteeringNoticesUnavailable)
		logs := buildLogsData([]ProcessedRun{processed}, t.TempDir(), nil)
		require.Len(t, logs.Runs, 1)
		assert.True(t, logs.Runs[0].SteeringNoticesUnavailable)
		compact := compactLogsData(logs)
		encoded, err := json.Marshal(compact.Runs[0])
		require.NoError(t, err)
		assert.Contains(t, string(encoded), `"steering_notices_unavailable":true`)
	})
}

func TestSteeringThresholdDecodesExactValuesOnly(t *testing.T) {
	tests := []struct {
		raw  string
		want steeringThreshold
	}{
		{`80`, 80}, {`90`, 90}, {`95`, 95}, {`99`, 99}, {`90.0`, 90},
		{`79.6`, 0}, {`94.6`, 0}, {`100`, 0}, {`50`, 0}, {`"90"`, 0}, {`null`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			var got steeringThreshold
			require.NoError(t, json.Unmarshal([]byte(tt.raw), &got))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSteeringNoticesInAuditAndLogs(t *testing.T) {
	notices := []SteeringNotice{
		{Type: "ai_credit", Threshold: 80, RequestID: "req-2", Phase: "agent"},
		{Type: "timeout", Threshold: 90, RequestID: "req-3", Phase: "agent"},
	}
	processed := ProcessedRun{
		Run:        WorkflowRun{DatabaseID: 7, WorkflowName: "wf", LogsPath: t.TempDir()},
		TokenUsage: &TokenUsageSummary{SteeringNotices: notices},
	}

	audit, _ := buildLocalAuditData(processed, LogMetrics{}, nil)
	assert.Equal(t, notices, audit.SteeringNotices)

	data := buildLogsData([]ProcessedRun{processed}, t.TempDir(), nil)
	require.Len(t, data.Runs, 1)
	assert.Equal(t, notices, data.Runs[0].SteeringNotices)
	compact := compactLogsData(data)
	require.Len(t, compact.Runs, 1)
	assert.Nil(t, compact.Runs[0].TokenUsageSummary)
	assert.Equal(t, notices, compact.Runs[0].SteeringNotices, "compact logs output should keep delivered notices")

	encoded, err := json.Marshal(data.Runs[0])
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"steering_notices":[{"type":"ai_credit","threshold":80,"request_id":"req-2","phase":"agent"}`)

	empty := ProcessedRun{Run: WorkflowRun{DatabaseID: 8, WorkflowName: "wf", LogsPath: t.TempDir()}, TokenUsage: &TokenUsageSummary{}}
	emptyAudit, _ := buildLocalAuditData(empty, LogMetrics{}, nil)
	assert.Nil(t, emptyAudit.SteeringNotices)
	emptyLogs := buildLogsData([]ProcessedRun{empty}, t.TempDir(), nil)
	encoded, err = json.Marshal(emptyLogs.Runs[0])
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "steering_notices")
}

func TestRenderConsoleSteeringNotices(t *testing.T) {
	output := captureStderrForGuardPolicyReportTest(func() {
		renderConsoleSteeringNotices([]SteeringNotice{
			{Type: "ai_credit", Threshold: 80, RequestID: "req-2", Phase: "agent"},
			{Type: "token", Threshold: 95, RequestID: "det-1", Phase: "detection"},
		}, false)
	})
	assert.Contains(t, output, "steering_notices:")
	assert.Contains(t, output, "ai_credit 80% request_id=req-2\n")
	assert.Contains(t, output, "token 95% request_id=det-1 phase=detection")

	assert.Empty(t, captureStderrForGuardPolicyReportTest(func() { renderConsoleSteeringNotices(nil, false) }))
	assert.Contains(t, captureStderrForGuardPolicyReportTest(func() { renderConsoleSteeringNotices(nil, true) }), "steering_notices: unavailable")
}

func TestGatewaySteeringEventsIncludeAICreditAndThreshold(t *testing.T) {
	runDir := testutil.TempDir(t, "steering-event-log")
	logsDir := filepath.Join(runDir, "sandbox", "firewall", "logs", "api-proxy-logs")
	require.NoError(t, os.MkdirAll(logsDir, 0o755))
	t.Cleanup(func() { apiProxySteeringLogCache.Delete(filepath.Clean(runDir)) })
	events := strings.Join([]string{
		`{"timestamp":"2026-10-10T10:00:00Z","event":"ai_credit_steering","request_id":"req-2","provider":"copilot","threshold":80,"message":"[AWF AI CREDIT WARNING] 80% of the AI-credit budget used"}`,
		`{"timestamp":"2026-10-10T10:00:01Z","event":"timeout_steering","request_id":"req-3","threshold":90,"message":"[AWF TIME WARNING] 90% of the runtime used"}`,
		`{"timestamp":"2026-10-10T10:00:02Z","event":"token_steering","message":"[AWF TOKEN WARNING] budget nearly exhausted"}`,
		`{"timestamp":"2026-10-10T10:00:03Z","event":"ai_credit_steering","message":"not an AWF warning"}`,
	}, "\n") + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(logsDir, "events.jsonl"), []byte(events), 0o644))

	got, err := extractGatewaySteeringEvents(runDir)
	require.NoError(t, err)
	assert.Equal(t, []GatewaySteeringEvent{
		{Type: "ai_credit_steering", Message: "[AWF AI CREDIT WARNING] 80% of the AI-credit budget used", Timestamp: "2026-10-10T10:00:00Z", Threshold: 80, RequestID: "req-2"},
		{Type: "timeout_steering", Message: "[AWF TIME WARNING] 90% of the runtime used", Timestamp: "2026-10-10T10:00:01Z", Threshold: 90, RequestID: "req-3"},
		{Type: "token_steering", Message: "[AWF TOKEN WARNING] budget nearly exhausted", Timestamp: "2026-10-10T10:00:02Z"},
	}, got)

	encoded, err := json.Marshal(got[2])
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"token_steering","message":"[AWF TOKEN WARNING] budget nearly exhausted","timestamp":"2026-10-10T10:00:02Z"}`, string(encoded), "events without threshold or request id should serialize as before")
}

func TestSteeringEntryToTimelineEventThreshold(t *testing.T) {
	event, ok := steeringEntryToTimelineEvent(proxyEventsEntry{Event: "ai_credit_steering", Message: "[AWF AI CREDIT WARNING] 90%", Threshold: 90})
	require.True(t, ok)
	assert.Equal(t, "credit 90%", event.Status)

	event, ok = steeringEntryToTimelineEvent(proxyEventsEntry{Event: "token_steering", Message: "[AWF TOKEN WARNING] budget"})
	require.True(t, ok)
	assert.Equal(t, "token", event.Status, "events without a threshold keep their existing status")
}
