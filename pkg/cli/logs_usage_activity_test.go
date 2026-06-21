//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadUsageActivitySummary(t *testing.T) {
	t.Parallel()

	runDir := t.TempDir()
	summaryPath := filepath.Join(runDir, "usage", "activity", "summary.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(summaryPath), 0o755), "should create usage activity directory")
	require.NoError(t, os.WriteFile(summaryPath, []byte(`{
		"schema":"`+usageActivitySummarySchema+`",
		"firewall":{"total_requests":10,"allowed_requests":8,"blocked_requests":2},
		"session":{"turns":7},
		"gateway":{"total_calls":5,"failed_calls":1}
	}`), 0o644), "should write usage activity summary")

	summary, err := loadUsageActivitySummary(runDir)
	require.NoError(t, err, "loadUsageActivitySummary should parse the primary usage path")
	require.NotNil(t, summary, "summary should not be nil")
	require.NotNil(t, summary.Firewall, "firewall section should be present")
	assert.Equal(t, 10, summary.Firewall.TotalRequests, "firewall total_requests should be parsed from JSON")
	require.NotNil(t, summary.Session, "session section should be present")
	assert.Equal(t, 7, summary.Session.Turns, "session turns should be parsed from JSON")
	require.NotNil(t, summary.Gateway, "gateway section should be present")
	assert.Equal(t, 5, summary.Gateway.TotalCalls, "gateway total_calls should be parsed from JSON")
}

func TestApplyUsageActivitySummaryToResult(t *testing.T) {
	t.Parallel()

	result := DownloadResult{}
	summary := &usageActivitySummary{
		Session: &usageActivitySession{Turns: 4},
		Firewall: &usageActivityFirewall{
			TotalRequests:   12,
			AllowedRequests: 9,
			BlockedRequests: 3,
		},
		Gateway: &usageActivityGateway{
			TotalCalls:  6,
			FailedCalls: 2,
			Servers: []usageActivityGatewayServer{
				{ServerName: "github", ToolCallCount: 5, FailedCalls: 2},
				{ServerName: "playwright", ToolCallCount: 1, FailedCalls: 0},
			},
		},
	}

	applyUsageActivitySummaryToResult(summary, &result, true)

	assert.Equal(t, 4, result.Run.Turns, "turns should be backfilled when detailed session artifacts are absent")
	require.NotNil(t, result.FirewallAnalysis, "firewall summary should be backfilled")
	assert.Equal(t, 12, result.FirewallAnalysis.TotalRequests, "firewall total requests should be copied from the summary")
	assert.Equal(t, 3, result.FirewallAnalysis.BlockedRequests, "firewall blocked requests should be copied from the summary")
	require.NotNil(t, result.MCPToolUsage, "gateway summary should be backfilled")
	assert.Empty(t, result.MCPToolUsage.Summary, "usage-summary backfill should preserve empty summary rows instead of null")
	assert.Empty(t, result.MCPToolUsage.ToolCalls, "usage-summary backfill should preserve empty tool call rows instead of null")
	require.Len(t, result.MCPToolUsage.Servers, 2, "gateway servers should be copied from the summary")
	assert.Equal(t, "github", result.MCPToolUsage.Servers[0].ServerName, "server names should be preserved")
	assert.Equal(t, 5, result.MCPToolUsage.Servers[0].ToolCallCount, "tool call counts should be preserved")
	assert.Equal(t, 2, result.MCPToolUsage.Servers[0].ErrorCount, "failed call counts should map to server error counts")
}

func TestLoadUsageActivitySummaryFallbackPath(t *testing.T) {
	t.Parallel()

	runDir := t.TempDir()
	fallbackPath := filepath.Join(runDir, "activity", "summary.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(fallbackPath), 0o755), "should create fallback activity directory")
	require.NoError(t, os.WriteFile(fallbackPath, []byte(`{"schema":"`+usageActivitySummarySchema+`","session":{"turns":3}}`), 0o644), "should write fallback activity summary")

	summary, err := loadUsageActivitySummary(runDir)
	require.NoError(t, err, "fallback activity summary should load without error")
	require.NotNil(t, summary, "summary should be loaded from the fallback path")
	require.NotNil(t, summary.Session, "session section should be present in the fallback summary")
	assert.Equal(t, 3, summary.Session.Turns, "session turns should be loaded from the fallback path")
}

func TestLoadUsageActivitySummaryNoFile(t *testing.T) {
	t.Parallel()

	summary, err := loadUsageActivitySummary(t.TempDir())
	require.NoError(t, err, "missing activity summary should not be treated as an error")
	assert.Nil(t, summary, "missing activity summary should return nil")
}

func TestLoadUsageActivitySummaryMalformedPrimaryFallsBack(t *testing.T) {
	t.Parallel()

	runDir := t.TempDir()
	primaryPath := filepath.Join(runDir, "usage", "activity", "summary.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(primaryPath), 0o755), "should create primary activity directory")
	require.NoError(t, os.WriteFile(primaryPath, []byte(`{not valid json`), 0o644), "should write malformed primary summary")

	fallbackPath := filepath.Join(runDir, "activity", "summary.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(fallbackPath), 0o755), "should create fallback activity directory")
	require.NoError(t, os.WriteFile(fallbackPath, []byte(`{"schema":"`+usageActivitySummarySchema+`","session":{"turns":5}}`), 0o644), "should write valid fallback summary")

	summary, err := loadUsageActivitySummary(runDir)
	require.NoError(t, err, "valid fallback summary should be used when the primary summary is malformed")
	require.NotNil(t, summary, "fallback summary should be returned")
	require.NotNil(t, summary.Session, "session section should be present after fallback")
	assert.Equal(t, 5, summary.Session.Turns, "fallback session turns should be preserved")
}

func TestLoadUsageActivitySummaryRejectsUnsupportedSchema(t *testing.T) {
	t.Parallel()

	runDir := t.TempDir()
	summaryPath := filepath.Join(runDir, "usage", "activity", "summary.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(summaryPath), 0o755), "should create usage activity directory")
	require.NoError(t, os.WriteFile(summaryPath, []byte(`{"schema":"usage-activity-summary/v2"}`), 0o644), "should write unsupported schema summary")

	summary, err := loadUsageActivitySummary(runDir)
	require.Error(t, err, "unsupported activity summary schema should return an error")
	assert.Nil(t, summary, "unsupported schema should not be returned")
	assert.Contains(t, err.Error(), "unsupported usage activity summary schema", "schema validation error should explain the mismatch")
}

func TestApplyUsageActivitySummaryDoesNotOverwriteExistingData(t *testing.T) {
	t.Parallel()

	existingFirewall := &FirewallAnalysis{TotalRequests: 100}
	existingMCP := &MCPToolUsageData{
		Summary:   []MCPToolSummary{},
		ToolCalls: []MCPToolCall{},
	}
	result := DownloadResult{
		Run:              WorkflowRun{Turns: 9},
		FirewallAnalysis: existingFirewall,
		MCPToolUsage:     existingMCP,
	}
	summary := &usageActivitySummary{
		Session: &usageActivitySession{Turns: 4},
		Firewall: &usageActivityFirewall{
			TotalRequests:   12,
			AllowedRequests: 9,
			BlockedRequests: 3,
		},
		Gateway: &usageActivityGateway{
			Servers: []usageActivityGatewayServer{{ServerName: "github", ToolCallCount: 5, FailedCalls: 2}},
		},
	}

	applyUsageActivitySummaryToResult(summary, &result, false)

	assert.Equal(t, 9, result.Run.Turns, "existing turns must not be overwritten when detailed artifacts are available")
	assert.Same(t, existingFirewall, result.FirewallAnalysis, "existing firewall analysis must not be replaced")
	assert.Same(t, existingMCP, result.MCPToolUsage, "existing MCP tool usage must not be replaced")
}
