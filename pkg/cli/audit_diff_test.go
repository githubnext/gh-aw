//go:build !integration

package cli

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeFirewallDiff_NewDomains(t *testing.T) {
	run1 := &FirewallAnalysis{
		TotalRequests:   5,
		AllowedRequests: 5,
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443": {Allowed: 5, Blocked: 0},
		},
	}
	run2 := &FirewallAnalysis{
		TotalRequests:   20,
		AllowedRequests: 17,
		BlockedRequests: 3,
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443":        {Allowed: 5, Blocked: 0},
			"registry.npmjs.org:443":    {Allowed: 15, Blocked: 0},
			"telemetry.example.com:443": {Allowed: 0, Blocked: 2},
		},
	}

	diff := computeFirewallDiff(100, 200, run1, run2)

	assert.Equal(t, int64(100), diff.Run1ID, "Run1ID should match")
	assert.Equal(t, int64(200), diff.Run2ID, "Run2ID should match")
	assert.Len(t, diff.NewDomains, 2, "Should have 2 new domains")
	assert.Empty(t, diff.RemovedDomains, "Should have no removed domains")
	assert.Empty(t, diff.StatusChanges, "Should have no status changes")

	// Check new domains are sorted
	assert.Equal(t, "registry.npmjs.org:443", diff.NewDomains[0].Domain, "First new domain should be registry.npmjs.org")
	assert.Equal(t, "new", diff.NewDomains[0].Status, "Status should be 'new'")
	assert.Equal(t, "allowed", diff.NewDomains[0].Run2Status, "Registry should be allowed")
	assert.False(t, diff.NewDomains[0].IsAnomaly, "Allowed new domain should not be anomaly")

	assert.Equal(t, "telemetry.example.com:443", diff.NewDomains[1].Domain, "Second new domain should be telemetry.example.com")
	assert.Equal(t, "denied", diff.NewDomains[1].Run2Status, "Telemetry should be denied")
	assert.True(t, diff.NewDomains[1].IsAnomaly, "New denied domain should be anomaly")
	assert.Equal(t, "new denied domain", diff.NewDomains[1].AnomalyNote, "Anomaly note should explain the issue")

	// Check summary
	assert.Equal(t, 2, diff.Summary.NewDomainCount, "Summary should show 2 new domains")
	assert.True(t, diff.Summary.HasAnomalies, "Should have anomalies")
	assert.Equal(t, 1, diff.Summary.AnomalyCount, "Should have 1 anomaly")
}

func TestComputeFirewallDiff_RemovedDomains(t *testing.T) {
	run1 := &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443":       {Allowed: 5, Blocked: 0},
			"old-api.internal.com:443": {Allowed: 8, Blocked: 0},
		},
	}
	run2 := &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443": {Allowed: 5, Blocked: 0},
		},
	}

	diff := computeFirewallDiff(100, 200, run1, run2)

	assert.Len(t, diff.RemovedDomains, 1, "Should have 1 removed domain")
	assert.Equal(t, "old-api.internal.com:443", diff.RemovedDomains[0].Domain, "Removed domain should be old-api.internal.com")
	assert.Equal(t, "removed", diff.RemovedDomains[0].Status, "Status should be 'removed'")
	assert.Equal(t, "allowed", diff.RemovedDomains[0].Run1Status, "Domain was allowed in run 1")
	assert.Equal(t, 8, diff.RemovedDomains[0].Run1Allowed, "Domain had 8 allowed requests")
	assert.Equal(t, 1, diff.Summary.RemovedDomainCount, "Summary should show 1 removed domain")
}

func TestComputeFirewallDiff_StatusChanges(t *testing.T) {
	run1 := &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"staging.api.com:443":    {Allowed: 10, Blocked: 0},
			"legacy.service.com:443": {Allowed: 0, Blocked: 5},
		},
	}
	run2 := &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"staging.api.com:443":    {Allowed: 0, Blocked: 3},
			"legacy.service.com:443": {Allowed: 7, Blocked: 0},
		},
	}

	diff := computeFirewallDiff(100, 200, run1, run2)

	assert.Len(t, diff.StatusChanges, 2, "Should have 2 status changes")

	// legacy.service.com: denied → allowed (anomaly: previously denied, now allowed)
	legacyEntry := findDiffEntry(diff.StatusChanges, "legacy.service.com:443")
	require.NotNil(t, legacyEntry, "Should find legacy.service.com in status changes")
	assert.Equal(t, "denied", legacyEntry.Run1Status, "Was denied in run 1")
	assert.Equal(t, "allowed", legacyEntry.Run2Status, "Now allowed in run 2")
	assert.True(t, legacyEntry.IsAnomaly, "Should be flagged as anomaly")
	assert.Equal(t, "previously denied, now allowed", legacyEntry.AnomalyNote, "Anomaly note should explain the flip")

	// staging.api.com: allowed → denied (anomaly)
	stagingEntry := findDiffEntry(diff.StatusChanges, "staging.api.com:443")
	require.NotNil(t, stagingEntry, "Should find staging.api.com in status changes")
	assert.Equal(t, "allowed", stagingEntry.Run1Status, "Was allowed in run 1")
	assert.Equal(t, "denied", stagingEntry.Run2Status, "Now denied in run 2")
	assert.True(t, stagingEntry.IsAnomaly, "Should be flagged as anomaly")

	assert.Equal(t, 2, diff.Summary.StatusChangeCount, "Summary should show 2 status changes")
	assert.True(t, diff.Summary.HasAnomalies, "Should have anomalies")
}

func TestComputeFirewallDiff_VolumeChanges(t *testing.T) {
	run1 := &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443":  {Allowed: 23, Blocked: 0},
			"cdn.example.com:443": {Allowed: 50, Blocked: 0},
		},
	}
	run2 := &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443":  {Allowed: 89, Blocked: 0},
			"cdn.example.com:443": {Allowed: 55, Blocked: 0},
		},
	}

	diff := computeFirewallDiff(100, 200, run1, run2)

	// api.github.com: 23 → 89 = +287% (over 100% threshold)
	assert.Len(t, diff.VolumeChanges, 1, "Should have 1 volume change (api.github.com, not cdn)")
	assert.Equal(t, "api.github.com:443", diff.VolumeChanges[0].Domain, "Volume change should be for api.github.com")
	assert.Equal(t, "+287%", diff.VolumeChanges[0].VolumeChange, "Volume change should be +287%")

	// cdn.example.com: 50 → 55 = +10% (under threshold, not flagged)
	assert.Equal(t, 1, diff.Summary.VolumeChangeCount, "Summary should show 1 volume change")
	assert.False(t, diff.Summary.HasAnomalies, "Volume changes alone should not create anomalies")
}

func TestComputeFirewallDiff_BothNil(t *testing.T) {
	diff := computeFirewallDiff(100, 200, nil, nil)

	assert.Empty(t, diff.NewDomains, "Should have no new domains")
	assert.Empty(t, diff.RemovedDomains, "Should have no removed domains")
	assert.Empty(t, diff.StatusChanges, "Should have no status changes")
	assert.Empty(t, diff.VolumeChanges, "Should have no volume changes")
	assert.False(t, diff.Summary.HasAnomalies, "Should have no anomalies")
}

func TestComputeFirewallDiff_Run1Nil(t *testing.T) {
	run2 := &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443": {Allowed: 5, Blocked: 0},
		},
	}

	diff := computeFirewallDiff(100, 200, nil, run2)

	assert.Len(t, diff.NewDomains, 1, "All run2 domains should be new")
	assert.Equal(t, "api.github.com:443", diff.NewDomains[0].Domain, "New domain should be api.github.com")
}

func TestComputeFirewallDiff_Run2Nil(t *testing.T) {
	run1 := &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443": {Allowed: 5, Blocked: 0},
		},
	}

	diff := computeFirewallDiff(100, 200, run1, nil)

	assert.Len(t, diff.RemovedDomains, 1, "All run1 domains should be removed")
	assert.Equal(t, "api.github.com:443", diff.RemovedDomains[0].Domain, "Removed domain should be api.github.com")
}

func TestComputeFirewallDiff_NoChanges(t *testing.T) {
	stats := map[string]DomainRequestStats{
		"api.github.com:443": {Allowed: 5, Blocked: 0},
	}
	run1 := &FirewallAnalysis{RequestsByDomain: stats}
	run2 := &FirewallAnalysis{RequestsByDomain: stats}

	diff := computeFirewallDiff(100, 200, run1, run2)

	assert.Empty(t, diff.NewDomains, "Should have no new domains")
	assert.Empty(t, diff.RemovedDomains, "Should have no removed domains")
	assert.Empty(t, diff.StatusChanges, "Should have no status changes")
	assert.Empty(t, diff.VolumeChanges, "Should have no volume changes")
}

func TestComputeFirewallDiff_CompleteScenario(t *testing.T) {
	run1 := &FirewallAnalysis{
		TotalRequests:   46,
		AllowedRequests: 38,
		BlockedRequests: 8,
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443":       {Allowed: 23, Blocked: 0},
			"old-api.internal.com:443": {Allowed: 8, Blocked: 0},
			"staging.api.com:443":      {Allowed: 7, Blocked: 0},
			"blocked.example.com:443":  {Allowed: 0, Blocked: 8},
		},
	}
	run2 := &FirewallAnalysis{
		TotalRequests:   108,
		AllowedRequests: 106,
		BlockedRequests: 2,
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443":        {Allowed: 89, Blocked: 0},
			"registry.npmjs.org:443":    {Allowed: 15, Blocked: 0},
			"telemetry.example.com:443": {Allowed: 0, Blocked: 2},
			"staging.api.com:443":       {Allowed: 0, Blocked: 0}, // no requests (edge case)
			"blocked.example.com:443":   {Allowed: 0, Blocked: 0}, // no longer any requests (edge case)
		},
	}

	diff := computeFirewallDiff(12345, 12346, run1, run2)

	// Verify new domains
	assert.Len(t, diff.NewDomains, 2, "Should have 2 new domains")

	// Verify removed domains
	assert.Len(t, diff.RemovedDomains, 1, "Should have 1 removed domain (old-api.internal.com)")

	// api.github.com: 23 → 89 = +287%
	assert.GreaterOrEqual(t, len(diff.VolumeChanges), 1, "Should have at least 1 volume change")
}

func TestDomainStatus(t *testing.T) {
	tests := []struct {
		name     string
		stats    DomainRequestStats
		expected string
	}{
		{name: "allowed only", stats: DomainRequestStats{Allowed: 5, Blocked: 0}, expected: "allowed"},
		{name: "denied only", stats: DomainRequestStats{Allowed: 0, Blocked: 3}, expected: "denied"},
		{name: "mixed", stats: DomainRequestStats{Allowed: 2, Blocked: 1}, expected: "mixed"},
		{name: "zero requests", stats: DomainRequestStats{Allowed: 0, Blocked: 0}, expected: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyFirewallDomainStatus(tt.stats)
			assert.Equal(t, tt.expected, result, "Domain status should match")
		})
	}
}

func TestFormatVolumeChange(t *testing.T) {
	tests := []struct {
		name     string
		total1   int
		total2   int
		expected string
	}{
		{name: "increase 287%", total1: 23, total2: 89, expected: "+287%"},
		{name: "decrease 50%", total1: 100, total2: 50, expected: "-50%"},
		{name: "double", total1: 10, total2: 20, expected: "+100%"},
		{name: "from zero", total1: 0, total2: 10, expected: "+∞"},
		{name: "no change", total1: 10, total2: 10, expected: "+0%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatVolumeChange(tt.total1, tt.total2)
			assert.Equal(t, tt.expected, result, "Volume change format should match")
		})
	}
}

func TestFirewallDiffJSONSerialization(t *testing.T) {
	diff := computeFirewallDiff(100, 200, &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443": {Allowed: 5, Blocked: 0},
		},
	}, &FirewallAnalysis{
		RequestsByDomain: map[string]DomainRequestStats{
			"api.github.com:443":  {Allowed: 5, Blocked: 0},
			"new.example.com:443": {Allowed: 3, Blocked: 0},
		},
	})

	data, err := json.MarshalIndent(diff, "", "  ")
	require.NoError(t, err, "Should serialize diff to JSON")

	var parsed FirewallDiff
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err, "Should deserialize diff from JSON")

	assert.Equal(t, int64(100), parsed.Run1ID, "Run1ID should survive serialization")
	assert.Equal(t, int64(200), parsed.Run2ID, "Run2ID should survive serialization")
	assert.Len(t, parsed.NewDomains, 1, "Should have 1 new domain after deserialization")
	assert.Equal(t, "new.example.com:443", parsed.NewDomains[0].Domain, "New domain should match")
}

func TestStatusEmoji(t *testing.T) {
	assert.Equal(t, "✅", firewallStatusEmoji("allowed"), "Allowed should show checkmark")
	assert.Equal(t, "❌", firewallStatusEmoji("denied"), "Denied should show X")
	assert.Equal(t, "⚠️", firewallStatusEmoji("mixed"), "Mixed should show warning")
	assert.Equal(t, "❓", firewallStatusEmoji("unknown"), "Unknown should show question mark")
	assert.Equal(t, "❓", firewallStatusEmoji(""), "Empty should show question mark")
}

func TestIsEmptyDiff(t *testing.T) {
	emptyDiff := &FirewallDiff{}
	assert.True(t, isEmptyFirewallDiff(emptyDiff), "Empty diff should be detected")

	nonEmptyDiff := &FirewallDiff{
		NewDomains: []DomainDiffEntry{{Domain: "test.com"}},
	}
	assert.False(t, isEmptyFirewallDiff(nonEmptyDiff), "Non-empty diff should not be detected as empty")
}

// findDiffEntry is a test helper to find a domain in a list of diff entries
func findDiffEntry(entries []DomainDiffEntry, domain string) *DomainDiffEntry {
	for i := range entries {
		if entries[i].Domain == domain {
			return &entries[i]
		}
	}
	return nil
}

// findMCPToolDiffEntry is a test helper to find a tool entry by server and tool name
func findMCPToolDiffEntry(entries []MCPToolDiffEntry, serverName, toolName string) *MCPToolDiffEntry {
	for i := range entries {
		if entries[i].ServerName == serverName && entries[i].ToolName == toolName {
			return &entries[i]
		}
	}
	return nil
}

func TestComputeMCPToolsDiff_NewTools(t *testing.T) {
	run1 := &MCPToolUsageData{
		Summary: []MCPToolSummary{
			{ServerName: "github", ToolName: "issue_read", CallCount: 5, ErrorCount: 0},
		},
	}
	run2 := &MCPToolUsageData{
		Summary: []MCPToolSummary{
			{ServerName: "github", ToolName: "issue_read", CallCount: 5, ErrorCount: 0},
			{ServerName: "github", ToolName: "create_issue", CallCount: 3, ErrorCount: 0},
			{ServerName: "playwright", ToolName: "screenshot", CallCount: 2, ErrorCount: 1},
		},
	}

	diff := computeMCPToolsDiff(run1, run2)

	assert.Len(t, diff.NewTools, 2, "Should have 2 new tools")
	assert.Empty(t, diff.RemovedTools, "Should have no removed tools")
	assert.Empty(t, diff.ChangedTools, "Should have no changed tools")

	createIssue := findMCPToolDiffEntry(diff.NewTools, "github", "create_issue")
	require.NotNil(t, createIssue, "Should find create_issue in new tools")
	assert.Equal(t, "new", createIssue.Status, "Status should be 'new'")
	assert.Equal(t, 3, createIssue.Run2CallCount, "Call count should be 3")
	assert.False(t, createIssue.IsAnomaly, "No-error new tool should not be anomaly")

	screenshot := findMCPToolDiffEntry(diff.NewTools, "playwright", "screenshot")
	require.NotNil(t, screenshot, "Should find screenshot in new tools")
	assert.True(t, screenshot.IsAnomaly, "New tool with errors should be anomaly")
	assert.Equal(t, "new tool with errors", screenshot.AnomalyNote, "Anomaly note should explain errors")
	assert.Equal(t, 1, screenshot.Run2ErrorCount, "Error count should be 1")

	assert.Equal(t, 2, diff.Summary.NewToolCount, "Summary should show 2 new tools")
	assert.True(t, diff.Summary.HasAnomalies, "Should have anomalies")
	assert.Equal(t, 1, diff.Summary.AnomalyCount, "Should have 1 anomaly")
}

func TestComputeMCPToolsDiff_RemovedTools(t *testing.T) {
	run1 := &MCPToolUsageData{
		Summary: []MCPToolSummary{
			{ServerName: "github", ToolName: "issue_read", CallCount: 10, ErrorCount: 0},
			{ServerName: "github", ToolName: "search_repos", CallCount: 4, ErrorCount: 0},
		},
	}
	run2 := &MCPToolUsageData{
		Summary: []MCPToolSummary{
			{ServerName: "github", ToolName: "issue_read", CallCount: 8, ErrorCount: 0},
		},
	}

	diff := computeMCPToolsDiff(run1, run2)

	assert.Len(t, diff.RemovedTools, 1, "Should have 1 removed tool")
	assert.Equal(t, "search_repos", diff.RemovedTools[0].ToolName, "Removed tool should be search_repos")
	assert.Equal(t, "removed", diff.RemovedTools[0].Status, "Status should be 'removed'")
	assert.Equal(t, 4, diff.RemovedTools[0].Run1CallCount, "Should preserve run1 call count")
	assert.Equal(t, 1, diff.Summary.RemovedToolCount, "Summary should show 1 removed tool")
}

func TestComputeMCPToolsDiff_ChangedTools(t *testing.T) {
	run1 := &MCPToolUsageData{
		Summary: []MCPToolSummary{
			{ServerName: "github", ToolName: "issue_read", CallCount: 5, ErrorCount: 0},
			{ServerName: "github", ToolName: "create_pr", CallCount: 2, ErrorCount: 1},
		},
	}
	run2 := &MCPToolUsageData{
		Summary: []MCPToolSummary{
			{ServerName: "github", ToolName: "issue_read", CallCount: 10, ErrorCount: 0},
			{ServerName: "github", ToolName: "create_pr", CallCount: 2, ErrorCount: 3},
		},
	}

	diff := computeMCPToolsDiff(run1, run2)

	assert.Len(t, diff.ChangedTools, 2, "Should have 2 changed tools")

	issueRead := findMCPToolDiffEntry(diff.ChangedTools, "github", "issue_read")
	require.NotNil(t, issueRead, "Should find issue_read in changed tools")
	assert.Equal(t, "changed", issueRead.Status, "Status should be 'changed'")
	assert.Equal(t, 5, issueRead.Run1CallCount, "Run1 call count should be 5")
	assert.Equal(t, 10, issueRead.Run2CallCount, "Run2 call count should be 10")
	assert.Equal(t, "+5", issueRead.CallCountChange, "Call count change should be +5")
	assert.False(t, issueRead.IsAnomaly, "No error increase should not be anomaly")

	createPR := findMCPToolDiffEntry(diff.ChangedTools, "github", "create_pr")
	require.NotNil(t, createPR, "Should find create_pr in changed tools")
	assert.True(t, createPR.IsAnomaly, "Increased error count should be anomaly")
	assert.Equal(t, "error count increased", createPR.AnomalyNote, "Anomaly note should explain error increase")
	assert.Equal(t, 1, createPR.Run1ErrorCount, "Run1 error count should be 1")
	assert.Equal(t, 3, createPR.Run2ErrorCount, "Run2 error count should be 3")

	assert.Equal(t, 2, diff.Summary.ChangedToolCount, "Summary should show 2 changed tools")
	assert.True(t, diff.Summary.HasAnomalies, "Should have anomalies")
	assert.Equal(t, 1, diff.Summary.AnomalyCount, "Should have 1 anomaly")
}

func TestComputeMCPToolsDiff_BothNil(t *testing.T) {
	diff := computeMCPToolsDiff(nil, nil)

	assert.Empty(t, diff.NewTools, "Should have no new tools")
	assert.Empty(t, diff.RemovedTools, "Should have no removed tools")
	assert.Empty(t, diff.ChangedTools, "Should have no changed tools")
	assert.False(t, diff.Summary.HasAnomalies, "Should have no anomalies")
}

func TestComputeMCPToolsDiff_NoChanges(t *testing.T) {
	toolSummary := []MCPToolSummary{
		{ServerName: "github", ToolName: "issue_read", CallCount: 5, ErrorCount: 0},
	}
	run1 := &MCPToolUsageData{Summary: toolSummary}
	run2 := &MCPToolUsageData{Summary: toolSummary}

	diff := computeMCPToolsDiff(run1, run2)

	assert.Empty(t, diff.NewTools, "Should have no new tools")
	assert.Empty(t, diff.RemovedTools, "Should have no removed tools")
	assert.Empty(t, diff.ChangedTools, "Should have no changed tools")
}

func TestComputeMCPToolsDiff_SortedOutput(t *testing.T) {
	run1 := &MCPToolUsageData{}
	run2 := &MCPToolUsageData{
		Summary: []MCPToolSummary{
			{ServerName: "z-server", ToolName: "tool", CallCount: 1},
			{ServerName: "a-server", ToolName: "tool", CallCount: 1},
			{ServerName: "m-server", ToolName: "tool", CallCount: 1},
		},
	}

	diff := computeMCPToolsDiff(run1, run2)

	require.Len(t, diff.NewTools, 3, "Should have 3 new tools")
	assert.Equal(t, "a-server", diff.NewTools[0].ServerName, "First tool should be a-server (sorted)")
	assert.Equal(t, "m-server", diff.NewTools[1].ServerName, "Second tool should be m-server (sorted)")
	assert.Equal(t, "z-server", diff.NewTools[2].ServerName, "Third tool should be z-server (sorted)")
}

func TestComputeRunMetricsDiff_WithData(t *testing.T) {
	summary1 := &RunSummary{
		RunID: 100,
		Run: WorkflowRun{
			TokenUsage: 5000,
			Duration:   10 * time.Minute,
			Turns:      8,
		},
	}
	summary2 := &RunSummary{
		RunID: 200,
		Run: WorkflowRun{
			TokenUsage: 7500,
			Duration:   15 * time.Minute,
			Turns:      12,
		},
	}

	diff := computeRunMetricsDiff(summary1, summary2)

	require.NotNil(t, diff, "Should produce metrics diff when data is available")
	assert.Equal(t, 5000, diff.Run1TokenUsage, "Run1 token usage should be 5000")
	assert.Equal(t, 7500, diff.Run2TokenUsage, "Run2 token usage should be 7500")
	assert.Equal(t, "+50%", diff.TokenUsageChange, "Token usage should increase by 50%")

	assert.Equal(t, "10m0s", diff.Run1Duration, "Run1 duration should be 10m0s")
	assert.Equal(t, "15m0s", diff.Run2Duration, "Run2 duration should be 15m0s")
	assert.Equal(t, "+5m0s", diff.DurationChange, "Duration should increase by 5m0s")

	assert.Equal(t, 8, diff.Run1Turns, "Run1 turns should be 8")
	assert.Equal(t, 12, diff.Run2Turns, "Run2 turns should be 12")
	assert.Equal(t, 4, diff.TurnsChange, "Turns change should be +4")
}

func TestComputeRunMetricsDiff_NegativeChange(t *testing.T) {
	summary1 := &RunSummary{
		Run: WorkflowRun{
			TokenUsage: 8000,
			Duration:   20 * time.Minute,
			Turns:      15,
		},
	}
	summary2 := &RunSummary{
		Run: WorkflowRun{
			TokenUsage: 4000,
			Duration:   12 * time.Minute,
			Turns:      10,
		},
	}

	diff := computeRunMetricsDiff(summary1, summary2)

	require.NotNil(t, diff, "Should produce metrics diff")
	assert.Equal(t, "-50%", diff.TokenUsageChange, "Token usage should decrease by 50%")
	assert.Equal(t, "-8m0s", diff.DurationChange, "Duration should decrease by 8m0s")
	assert.Equal(t, -5, diff.TurnsChange, "Turns change should be -5")
}

func TestComputeRunMetricsDiff_BothNil(t *testing.T) {
	diff := computeRunMetricsDiff(nil, nil)
	assert.Nil(t, diff, "Should return nil when both summaries are nil")
}

func TestComputeRunMetricsDiff_AllZero(t *testing.T) {
	summary1 := &RunSummary{Run: WorkflowRun{}}
	summary2 := &RunSummary{Run: WorkflowRun{}}

	diff := computeRunMetricsDiff(summary1, summary2)
	assert.Nil(t, diff, "Should return nil when all metrics are zero")
}

func TestComputeAuditDiff_CombinesAllSections(t *testing.T) {
	summary1 := &RunSummary{
		RunID: 100,
		FirewallAnalysis: &FirewallAnalysis{
			RequestsByDomain: map[string]DomainRequestStats{
				"api.github.com:443": {Allowed: 5, Blocked: 0},
			},
		},
		MCPToolUsage: &MCPToolUsageData{
			Summary: []MCPToolSummary{
				{ServerName: "github", ToolName: "issue_read", CallCount: 3, ErrorCount: 0},
			},
		},
		Run: WorkflowRun{TokenUsage: 2000, Turns: 5},
	}
	summary2 := &RunSummary{
		RunID: 200,
		FirewallAnalysis: &FirewallAnalysis{
			RequestsByDomain: map[string]DomainRequestStats{
				"api.github.com:443":  {Allowed: 5, Blocked: 0},
				"new.example.com:443": {Allowed: 3, Blocked: 0},
			},
		},
		MCPToolUsage: &MCPToolUsageData{
			Summary: []MCPToolSummary{
				{ServerName: "github", ToolName: "issue_read", CallCount: 7, ErrorCount: 0},
				{ServerName: "github", ToolName: "create_issue", CallCount: 2, ErrorCount: 0},
			},
		},
		Run: WorkflowRun{TokenUsage: 3000, Turns: 8},
	}

	diff := computeAuditDiff(100, 200, summary1, summary2)

	assert.Equal(t, int64(100), diff.Run1ID, "Run1ID should match")
	assert.Equal(t, int64(200), diff.Run2ID, "Run2ID should match")

	require.NotNil(t, diff.FirewallDiff, "Should have firewall diff")
	assert.Len(t, diff.FirewallDiff.NewDomains, 1, "Should have 1 new domain")

	require.NotNil(t, diff.MCPToolsDiff, "Should have MCP tools diff")
	assert.Len(t, diff.MCPToolsDiff.NewTools, 1, "Should have 1 new tool")
	assert.Len(t, diff.MCPToolsDiff.ChangedTools, 1, "Should have 1 changed tool")

	require.NotNil(t, diff.RunMetricsDiff, "Should have run metrics diff")
	assert.Equal(t, 2000, diff.RunMetricsDiff.Run1TokenUsage, "Run1 token usage should match")
	assert.Equal(t, 3000, diff.RunMetricsDiff.Run2TokenUsage, "Run2 token usage should match")
}

func TestComputeAuditDiff_NilSummaries(t *testing.T) {
	diff := computeAuditDiff(100, 200, nil, nil)

	assert.Equal(t, int64(100), diff.Run1ID, "Run1ID should be set even with nil summaries")
	assert.NotNil(t, diff.FirewallDiff, "FirewallDiff should be non-nil (empty)")
	assert.Nil(t, diff.MCPToolsDiff, "MCPToolsDiff should be nil when no MCP data")
	assert.Nil(t, diff.RunMetricsDiff, "RunMetricsDiff should be nil when no metrics data")
	assert.True(t, isEmptyAuditDiff(diff), "Diff with nil summaries should be empty")
}

func TestAuditDiffJSONSerialization(t *testing.T) {
	summary1 := &RunSummary{
		FirewallAnalysis: &FirewallAnalysis{
			RequestsByDomain: map[string]DomainRequestStats{
				"api.github.com:443": {Allowed: 5},
			},
		},
		MCPToolUsage: &MCPToolUsageData{
			Summary: []MCPToolSummary{
				{ServerName: "github", ToolName: "issue_read", CallCount: 3},
			},
		},
		Run: WorkflowRun{TokenUsage: 1000, Turns: 4},
	}
	summary2 := &RunSummary{
		FirewallAnalysis: &FirewallAnalysis{
			RequestsByDomain: map[string]DomainRequestStats{
				"api.github.com:443":  {Allowed: 5},
				"new.example.com:443": {Allowed: 2},
			},
		},
		MCPToolUsage: &MCPToolUsageData{
			Summary: []MCPToolSummary{
				{ServerName: "github", ToolName: "issue_read", CallCount: 5},
			},
		},
		Run: WorkflowRun{TokenUsage: 1500, Turns: 6},
	}

	diff := computeAuditDiff(100, 200, summary1, summary2)

	data, err := json.MarshalIndent(diff, "", "  ")
	require.NoError(t, err, "Should serialize AuditDiff to JSON")

	var parsed AuditDiff
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err, "Should deserialize AuditDiff from JSON")

	assert.Equal(t, int64(100), parsed.Run1ID, "Run1ID should survive serialization")
	assert.Equal(t, int64(200), parsed.Run2ID, "Run2ID should survive serialization")
	require.NotNil(t, parsed.FirewallDiff, "FirewallDiff should survive serialization")
	assert.Len(t, parsed.FirewallDiff.NewDomains, 1, "New domains should survive serialization")
	require.NotNil(t, parsed.MCPToolsDiff, "MCPToolsDiff should survive serialization")
	require.NotNil(t, parsed.RunMetricsDiff, "RunMetricsDiff should survive serialization")
	assert.Equal(t, 1000, parsed.RunMetricsDiff.Run1TokenUsage, "Token usage should survive serialization")
}

func TestFormatCountChange(t *testing.T) {
	tests := []struct {
		name     string
		count1   int
		count2   int
		expected string
	}{
		{name: "increase", count1: 3, count2: 8, expected: "+5"},
		{name: "decrease", count1: 10, count2: 3, expected: "-7"},
		{name: "no change", count1: 5, count2: 5, expected: "+0"},
		{name: "from zero", count1: 0, count2: 4, expected: "+4"},
		{name: "to zero", count1: 6, count2: 0, expected: "-6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCountChange(tt.count1, tt.count2)
			assert.Equal(t, tt.expected, result, "Count change format should match")
		})
	}
}

func TestIsEmptyMCPToolsDiff(t *testing.T) {
	assert.True(t, isEmptyMCPToolsDiff(&MCPToolsDiff{}), "Empty MCPToolsDiff should be detected")
	assert.False(t, isEmptyMCPToolsDiff(&MCPToolsDiff{
		NewTools: []MCPToolDiffEntry{{ToolName: "test"}},
	}), "Non-empty MCPToolsDiff should not be detected as empty")
}

func TestIsEmptyAuditDiff(t *testing.T) {
	assert.True(t, isEmptyAuditDiff(&AuditDiff{}), "Empty AuditDiff should be detected")
	assert.True(t, isEmptyAuditDiff(&AuditDiff{
		FirewallDiff: &FirewallDiff{},
		MCPToolsDiff: &MCPToolsDiff{},
	}), "AuditDiff with empty sub-diffs should be detected as empty")
	assert.False(t, isEmptyAuditDiff(&AuditDiff{
		MCPToolsDiff: &MCPToolsDiff{
			NewTools: []MCPToolDiffEntry{{ToolName: "test"}},
		},
	}), "AuditDiff with MCP changes should not be empty")
	assert.False(t, isEmptyAuditDiff(&AuditDiff{
		RunMetricsDiff: &RunMetricsDiff{Run1TokenUsage: 100},
	}), "AuditDiff with metrics diff should not be empty")
}

func TestComputeTokenUsageDiff_BothNil(t *testing.T) {
	diff := computeTokenUsageDiff(nil, nil)
	assert.Nil(t, diff, "Should return nil when both summaries are nil")
}

func TestComputeTokenUsageDiff_WithData(t *testing.T) {
	tu1 := &TokenUsageSummary{
		TotalInputTokens:      10000,
		TotalOutputTokens:     2000,
		TotalCacheReadTokens:  5000,
		TotalCacheWriteTokens: 1000,
		TotalEffectiveTokens:  8000,
		TotalRequests:         10,
		CacheEfficiency:       0.333,
	}
	tu2 := &TokenUsageSummary{
		TotalInputTokens:      15000,
		TotalOutputTokens:     3000,
		TotalCacheReadTokens:  7000,
		TotalCacheWriteTokens: 800,
		TotalEffectiveTokens:  12000,
		TotalRequests:         14,
		CacheEfficiency:       0.318,
	}

	diff := computeTokenUsageDiff(tu1, tu2)

	require.NotNil(t, diff, "Should produce token usage diff when data is available")
	assert.Equal(t, 10000, diff.Run1InputTokens, "Run1 input tokens should be 10000")
	assert.Equal(t, 15000, diff.Run2InputTokens, "Run2 input tokens should be 15000")
	assert.Equal(t, "+50%", diff.InputTokensChange, "Input tokens should increase by 50%")

	assert.Equal(t, 2000, diff.Run1OutputTokens, "Run1 output tokens should be 2000")
	assert.Equal(t, 3000, diff.Run2OutputTokens, "Run2 output tokens should be 3000")
	assert.Equal(t, "+50%", diff.OutputTokensChange, "Output tokens should increase by 50%")

	assert.Equal(t, 5000, diff.Run1CacheReadTokens, "Run1 cache read tokens should be 5000")
	assert.Equal(t, 7000, diff.Run2CacheReadTokens, "Run2 cache read tokens should be 7000")
	assert.Equal(t, "+40%", diff.CacheReadTokensChange, "Cache read tokens should increase by 40%")

	assert.Equal(t, 1000, diff.Run1CacheWriteTokens, "Run1 cache write tokens should be 1000")
	assert.Equal(t, 800, diff.Run2CacheWriteTokens, "Run2 cache write tokens should be 800")
	assert.Equal(t, "-20%", diff.CacheWriteTokensChange, "Cache write tokens should decrease by 20%")

	assert.Equal(t, 8000, diff.Run1EffectiveTokens, "Run1 effective tokens should be 8000")
	assert.Equal(t, 12000, diff.Run2EffectiveTokens, "Run2 effective tokens should be 12000")
	assert.Equal(t, "+50%", diff.EffectiveTokensChange, "Effective tokens should increase by 50%")

	assert.Equal(t, 10, diff.Run1TotalRequests, "Run1 requests should be 10")
	assert.Equal(t, 14, diff.Run2TotalRequests, "Run2 requests should be 14")
	assert.Equal(t, "+4", diff.RequestsDelta, "Requests delta should be +4")

	assert.InDelta(t, 0.333, diff.Run1CacheEfficiency, 0.001, "Run1 cache efficiency should match")
	assert.InDelta(t, 0.318, diff.Run2CacheEfficiency, 0.001, "Run2 cache efficiency should match")
	assert.Equal(t, "-1.5pp", diff.CacheEfficiencyChange, "Cache efficiency change should be -1.5pp")
}

func TestComputeTokenUsageDiff_Run1Nil(t *testing.T) {
	tu2 := &TokenUsageSummary{
		TotalInputTokens:  5000,
		TotalOutputTokens: 1000,
		TotalRequests:     5,
	}

	diff := computeTokenUsageDiff(nil, tu2)

	require.NotNil(t, diff, "Should produce diff when run2 has data")
	assert.Equal(t, 0, diff.Run1InputTokens, "Run1 input tokens should be 0 when nil")
	assert.Equal(t, 5000, diff.Run2InputTokens, "Run2 input tokens should be 5000")
	assert.Equal(t, "+∞", diff.InputTokensChange, "Input change should be +∞ from zero")
}

func TestComputeTokenUsageDiff_Run2Nil(t *testing.T) {
	tu1 := &TokenUsageSummary{
		TotalInputTokens:  5000,
		TotalOutputTokens: 1000,
	}

	diff := computeTokenUsageDiff(tu1, nil)

	require.NotNil(t, diff, "Should produce diff when run1 has data")
	assert.Equal(t, 5000, diff.Run1InputTokens, "Run1 input tokens should be 5000")
	assert.Equal(t, 0, diff.Run2InputTokens, "Run2 input tokens should be 0 when nil")
	assert.Equal(t, "-100%", diff.InputTokensChange, "Input change should be -100%")
}

func TestComputeRunMetricsDiff_WithTokenUsageDetails(t *testing.T) {
	summary1 := &RunSummary{
		RunID: 100,
		Run:   WorkflowRun{Duration: 5 * time.Minute, Turns: 4},
		TokenUsage: &TokenUsageSummary{
			TotalInputTokens:     8000,
			TotalOutputTokens:    1500,
			TotalEffectiveTokens: 6000,
			TotalRequests:        8,
			CacheEfficiency:      0.25,
		},
	}
	summary2 := &RunSummary{
		RunID: 200,
		Run:   WorkflowRun{Duration: 7 * time.Minute, Turns: 6},
		TokenUsage: &TokenUsageSummary{
			TotalInputTokens:     12000,
			TotalOutputTokens:    2000,
			TotalEffectiveTokens: 9000,
			TotalRequests:        11,
			CacheEfficiency:      0.30,
		},
	}

	diff := computeRunMetricsDiff(summary1, summary2)

	require.NotNil(t, diff, "Should produce metrics diff")
	require.NotNil(t, diff.TokenUsageDetails, "Should populate TokenUsageDetails from RunSummary.TokenUsage")

	assert.Equal(t, 8000, diff.TokenUsageDetails.Run1InputTokens, "Run1 input tokens should be 8000")
	assert.Equal(t, 12000, diff.TokenUsageDetails.Run2InputTokens, "Run2 input tokens should be 12000")
	assert.Equal(t, "+50%", diff.TokenUsageDetails.InputTokensChange, "Input tokens change should be +50%")

	assert.Equal(t, 6000, diff.TokenUsageDetails.Run1EffectiveTokens, "Run1 effective tokens should be 6000")
	assert.Equal(t, 9000, diff.TokenUsageDetails.Run2EffectiveTokens, "Run2 effective tokens should be 9000")
	assert.Equal(t, "+50%", diff.TokenUsageDetails.EffectiveTokensChange, "Effective tokens change should be +50%")
}

func TestComputeRunMetricsDiff_TokenUsageDetailsAloneNotNil(t *testing.T) {
	// Verify that detailed token usage data alone (without Run.TokenUsage set)
	// still produces a non-nil RunMetricsDiff
	summary1 := &RunSummary{
		Run: WorkflowRun{},
		TokenUsage: &TokenUsageSummary{
			TotalInputTokens: 5000,
			TotalRequests:    5,
		},
	}
	summary2 := &RunSummary{
		Run: WorkflowRun{},
		TokenUsage: &TokenUsageSummary{
			TotalInputTokens: 8000,
			TotalRequests:    7,
		},
	}

	diff := computeRunMetricsDiff(summary1, summary2)

	require.NotNil(t, diff, "Should produce non-nil diff when only TokenUsage data is present")
	require.NotNil(t, diff.TokenUsageDetails, "Should have TokenUsageDetails")
	assert.Equal(t, 5000, diff.TokenUsageDetails.Run1InputTokens, "Run1 input tokens should be 5000")
	assert.Equal(t, 8000, diff.TokenUsageDetails.Run2InputTokens, "Run2 input tokens should be 8000")
}

func TestComputeAuditDiff_MultipleRuns(t *testing.T) {
	base := &RunSummary{
		RunID: 100,
		FirewallAnalysis: &FirewallAnalysis{
			RequestsByDomain: map[string]DomainRequestStats{
				"api.github.com:443": {Allowed: 5, Blocked: 0},
			},
		},
		MCPToolUsage: &MCPToolUsageData{
			Summary: []MCPToolSummary{
				{ServerName: "github", ToolName: "issue_read", CallCount: 3, ErrorCount: 0},
			},
		},
		Run: WorkflowRun{Turns: 5},
		TokenUsage: &TokenUsageSummary{
			TotalInputTokens:  10000,
			TotalOutputTokens: 2000,
			TotalRequests:     10,
		},
	}

	compare1 := &RunSummary{
		RunID: 200,
		FirewallAnalysis: &FirewallAnalysis{
			RequestsByDomain: map[string]DomainRequestStats{
				"api.github.com:443":   {Allowed: 5, Blocked: 0},
				"new1.example.com:443": {Allowed: 3, Blocked: 0},
			},
		},
		MCPToolUsage: &MCPToolUsageData{
			Summary: []MCPToolSummary{
				{ServerName: "github", ToolName: "issue_read", CallCount: 5, ErrorCount: 0},
			},
		},
		Run: WorkflowRun{Turns: 7},
		TokenUsage: &TokenUsageSummary{
			TotalInputTokens:  15000,
			TotalOutputTokens: 3000,
			TotalRequests:     12,
		},
	}

	compare2 := &RunSummary{
		RunID: 300,
		FirewallAnalysis: &FirewallAnalysis{
			RequestsByDomain: map[string]DomainRequestStats{
				"api.github.com:443":   {Allowed: 5, Blocked: 0},
				"new2.example.com:443": {Allowed: 1, Blocked: 2},
			},
		},
		Run: WorkflowRun{Turns: 4},
		TokenUsage: &TokenUsageSummary{
			TotalInputTokens:  8000,
			TotalOutputTokens: 1500,
			TotalRequests:     8,
		},
	}

	// Compute two diffs from the same base
	diff1 := computeAuditDiff(base.RunID, compare1.RunID, base, compare1)
	diff2 := computeAuditDiff(base.RunID, compare2.RunID, base, compare2)

	// Diff 1: base vs compare1
	assert.Equal(t, int64(100), diff1.Run1ID, "Diff1 Run1ID should be base")
	assert.Equal(t, int64(200), diff1.Run2ID, "Diff1 Run2ID should be compare1")
	require.NotNil(t, diff1.FirewallDiff, "Diff1 should have firewall diff")
	assert.Len(t, diff1.FirewallDiff.NewDomains, 1, "Diff1 should have 1 new domain")
	assert.Equal(t, "new1.example.com:443", diff1.FirewallDiff.NewDomains[0].Domain, "Diff1 new domain should be new1")
	require.NotNil(t, diff1.RunMetricsDiff, "Diff1 should have run metrics diff")
	require.NotNil(t, diff1.RunMetricsDiff.TokenUsageDetails, "Diff1 should have token usage details")
	assert.Equal(t, "+50%", diff1.RunMetricsDiff.TokenUsageDetails.InputTokensChange, "Diff1 input tokens should increase by 50%")

	// Diff 2: base vs compare2
	assert.Equal(t, int64(100), diff2.Run1ID, "Diff2 Run1ID should be base")
	assert.Equal(t, int64(300), diff2.Run2ID, "Diff2 Run2ID should be compare2")
	require.NotNil(t, diff2.FirewallDiff, "Diff2 should have firewall diff")
	assert.Len(t, diff2.FirewallDiff.NewDomains, 1, "Diff2 should have 1 new domain")
	assert.Equal(t, "new2.example.com:443", diff2.FirewallDiff.NewDomains[0].Domain, "Diff2 new domain should be new2")
	assert.True(t, diff2.FirewallDiff.NewDomains[0].IsAnomaly, "Diff2 new domain should be anomaly (blocked)")
	require.NotNil(t, diff2.RunMetricsDiff, "Diff2 should have run metrics diff")
	require.NotNil(t, diff2.RunMetricsDiff.TokenUsageDetails, "Diff2 should have token usage details")
	assert.Equal(t, "-20%", diff2.RunMetricsDiff.TokenUsageDetails.InputTokensChange, "Diff2 input tokens should decrease by 20%")

	// The two diffs should be independent (no shared state)
	assert.NotEqual(t, diff1.Run2ID, diff2.Run2ID, "The two diffs should have different Run2IDs")
}

func TestComputeGitHubRateLimitDiff_BothNil(t *testing.T) {
	diff := computeGitHubRateLimitDiff(nil, nil)
	assert.Nil(t, diff, "both nil should return nil")
}

func TestComputeGitHubRateLimitDiff_WithData(t *testing.T) {
	rl1 := &GitHubRateLimitUsage{
		TotalRequestsMade: 40,
		CoreConsumed:      40,
		CoreRemaining:     4960,
		CoreLimit:         5000,
	}
	rl2 := &GitHubRateLimitUsage{
		TotalRequestsMade: 60,
		CoreConsumed:      60,
		CoreRemaining:     4940,
		CoreLimit:         5000,
	}

	diff := computeGitHubRateLimitDiff(rl1, rl2)
	require.NotNil(t, diff, "diff should not be nil")

	assert.Equal(t, 40, diff.Run1TotalAPICalls, "Run1 total API calls should be 40")
	assert.Equal(t, 60, diff.Run2TotalAPICalls, "Run2 total API calls should be 60")
	assert.Equal(t, "+50%", diff.APICallsChange, "API calls should increase by 50%")
	assert.Equal(t, 40, diff.Run1CoreConsumed, "Run1 core consumed should be 40")
	assert.Equal(t, 60, diff.Run2CoreConsumed, "Run2 core consumed should be 60")
	assert.Equal(t, "+50%", diff.CoreConsumedChange, "Core consumed should increase by 50%")
	assert.Equal(t, 4960, diff.Run1CoreRemaining, "Run1 core remaining should be 4960")
	assert.Equal(t, 4940, diff.Run2CoreRemaining, "Run2 core remaining should be 4940")
	assert.Equal(t, 5000, diff.Run1CoreLimit, "Run1 core limit should be 5000")
	assert.Equal(t, 5000, diff.Run2CoreLimit, "Run2 core limit should be 5000")
}

func TestComputeGitHubRateLimitDiff_Run1Nil(t *testing.T) {
	rl2 := &GitHubRateLimitUsage{
		TotalRequestsMade: 30,
		CoreConsumed:      30,
		CoreRemaining:     4970,
		CoreLimit:         5000,
	}

	diff := computeGitHubRateLimitDiff(nil, rl2)
	require.NotNil(t, diff, "diff should not be nil when run2 has data")
	assert.Equal(t, 0, diff.Run1TotalAPICalls, "Run1 total API calls should be 0")
	assert.Equal(t, 30, diff.Run2TotalAPICalls, "Run2 total API calls should be 30")
}

func TestComputeRunMetricsDiff_WithRateLimitData(t *testing.T) {
	summary1 := &RunSummary{
		Run: WorkflowRun{
			TokenUsage: 1000,
			Duration:   2 * time.Minute,
			Turns:      5,
		},
		GitHubRateLimitUsage: &GitHubRateLimitUsage{
			TotalRequestsMade: 40,
			CoreConsumed:      40,
			CoreRemaining:     4960,
			CoreLimit:         5000,
		},
	}
	summary2 := &RunSummary{
		Run: WorkflowRun{
			TokenUsage: 1200,
			Duration:   3 * time.Minute,
			Turns:      6,
		},
		GitHubRateLimitUsage: &GitHubRateLimitUsage{
			TotalRequestsMade: 60,
			CoreConsumed:      60,
			CoreRemaining:     4940,
			CoreLimit:         5000,
		},
	}

	diff := computeRunMetricsDiff(summary1, summary2)
	require.NotNil(t, diff, "diff should not be nil")
	require.NotNil(t, diff.GitHubRateLimitDetails, "GitHubRateLimitDetails should be populated")
	assert.Equal(t, 40, diff.GitHubRateLimitDetails.Run1TotalAPICalls, "Run1 API calls should be 40")
	assert.Equal(t, 60, diff.GitHubRateLimitDetails.Run2TotalAPICalls, "Run2 API calls should be 60")
	assert.Equal(t, "+50%", diff.GitHubRateLimitDetails.APICallsChange, "API calls should increase by 50%")
}

func TestComputeRunMetricsDiff_RateLimitAloneNotNil(t *testing.T) {
	// RunMetricsDiff should be non-nil when only rate limit data is present
	summary1 := &RunSummary{
		GitHubRateLimitUsage: &GitHubRateLimitUsage{
			TotalRequestsMade: 10,
		},
	}
	summary2 := &RunSummary{
		GitHubRateLimitUsage: &GitHubRateLimitUsage{
			TotalRequestsMade: 15,
		},
	}

	diff := computeRunMetricsDiff(summary1, summary2)
	require.NotNil(t, diff, "diff should not be nil when rate limit data is present")
	require.NotNil(t, diff.GitHubRateLimitDetails, "GitHubRateLimitDetails should be populated")
}
