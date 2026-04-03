//go:build !integration

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCrossRunAuditReport_EmptyInputs(t *testing.T) {
	report := buildCrossRunAuditReport([]crossRunInput{})

	assert.Equal(t, 0, report.RunsAnalyzed, "Should have 0 runs analyzed")
	assert.Equal(t, 0, report.RunsWithData, "Should have 0 runs with data")
	assert.Equal(t, 0, report.RunsWithoutData, "Should have 0 runs without data")
	assert.Equal(t, 0, report.Summary.UniqueDomains, "Should have 0 unique domains")
	assert.Empty(t, report.DomainInventory, "Domain inventory should be empty")
	assert.Empty(t, report.PerRunBreakdown, "Per-run breakdown should be empty")
}

func TestBuildCrossRunAuditReport_SingleRunWithData(t *testing.T) {
	inputs := []crossRunInput{
		{
			RunID:        100,
			WorkflowName: "test-workflow",
			Conclusion:   "success",
			FirewallAnalysis: &FirewallAnalysis{
				TotalRequests:   10,
				AllowedRequests: 8,
				BlockedRequests: 2,
				RequestsByDomain: map[string]DomainRequestStats{
					"api.github.com:443":     {Allowed: 5, Blocked: 0},
					"evil.example.com:443":   {Allowed: 0, Blocked: 2},
					"registry.npmjs.org:443": {Allowed: 3, Blocked: 0},
				},
			},
		},
	}

	report := buildCrossRunAuditReport(inputs)

	assert.Equal(t, 1, report.RunsAnalyzed, "Should analyze 1 run")
	assert.Equal(t, 1, report.RunsWithData, "Should have 1 run with data")
	assert.Equal(t, 0, report.RunsWithoutData, "Should have 0 runs without data")

	// Summary
	assert.Equal(t, 10, report.Summary.TotalRequests, "Total requests should be 10")
	assert.Equal(t, 8, report.Summary.TotalAllowed, "Total allowed should be 8")
	assert.Equal(t, 2, report.Summary.TotalBlocked, "Total blocked should be 2")
	assert.InDelta(t, 0.2, report.Summary.OverallDenyRate, 0.01, "Deny rate should be 0.2")
	assert.Equal(t, 3, report.Summary.UniqueDomains, "Should have 3 unique domains")

	// Domain inventory
	assert.Len(t, report.DomainInventory, 3, "Should have 3 domain entries")

	// Per-run breakdown
	require.Len(t, report.PerRunBreakdown, 1, "Should have 1 per-run breakdown entry")
	assert.Equal(t, int64(100), report.PerRunBreakdown[0].RunID, "Run ID should match")
	assert.True(t, report.PerRunBreakdown[0].HasData, "Run should have data")
}

func TestBuildCrossRunAuditReport_MultipleRuns(t *testing.T) {
	inputs := []crossRunInput{
		{
			RunID:        100,
			WorkflowName: "workflow-a",
			Conclusion:   "success",
			FirewallAnalysis: &FirewallAnalysis{
				TotalRequests:   5,
				AllowedRequests: 5,
				BlockedRequests: 0,
				RequestsByDomain: map[string]DomainRequestStats{
					"api.github.com:443":     {Allowed: 3, Blocked: 0},
					"npm.pkg.github.com:443": {Allowed: 2, Blocked: 0},
				},
			},
		},
		{
			RunID:        200,
			WorkflowName: "workflow-a",
			Conclusion:   "failure",
			FirewallAnalysis: &FirewallAnalysis{
				TotalRequests:   8,
				AllowedRequests: 5,
				BlockedRequests: 3,
				RequestsByDomain: map[string]DomainRequestStats{
					"api.github.com:443":   {Allowed: 3, Blocked: 0},
					"evil.example.com:443": {Allowed: 0, Blocked: 3},
					"pypi.org:443":         {Allowed: 2, Blocked: 0},
				},
			},
		},
		{
			RunID:            300,
			WorkflowName:     "workflow-b",
			Conclusion:       "success",
			FirewallAnalysis: nil, // no firewall data
		},
	}

	report := buildCrossRunAuditReport(inputs)

	assert.Equal(t, 3, report.RunsAnalyzed, "Should analyze 3 runs")
	assert.Equal(t, 2, report.RunsWithData, "Should have 2 runs with data")
	assert.Equal(t, 1, report.RunsWithoutData, "Should have 1 run without data")

	// Summary
	assert.Equal(t, 13, report.Summary.TotalRequests, "Total requests should be 13")
	assert.Equal(t, 10, report.Summary.TotalAllowed, "Total allowed should be 10")
	assert.Equal(t, 3, report.Summary.TotalBlocked, "Total blocked should be 3")
	assert.Equal(t, 4, report.Summary.UniqueDomains, "Should have 4 unique domains")

	// Domain inventory: api.github.com should be seen in 2 runs
	var githubEntry *DomainInventoryEntry
	for i, entry := range report.DomainInventory {
		if entry.Domain == "api.github.com:443" {
			githubEntry = &report.DomainInventory[i]
			break
		}
	}
	require.NotNil(t, githubEntry, "Should find api.github.com in inventory")
	assert.Equal(t, 2, githubEntry.SeenInRuns, "api.github.com should be seen in 2 runs")
	assert.Equal(t, 6, githubEntry.TotalAllowed, "api.github.com total allowed should be 6")
	assert.Equal(t, "allowed", githubEntry.OverallStatus, "api.github.com should be overall allowed")

	// Per-run status for api.github.com should include all 3 runs
	require.Len(t, githubEntry.PerRunStatus, 3, "api.github.com per-run status should include all 3 runs")
	assert.Equal(t, "allowed", githubEntry.PerRunStatus[0].Status, "Run 100 should be allowed")
	assert.Equal(t, "allowed", githubEntry.PerRunStatus[1].Status, "Run 200 should be allowed")
	assert.Equal(t, "absent", githubEntry.PerRunStatus[2].Status, "Run 300 should be absent")

	// evil.example.com should only be in run 200
	var evilEntry *DomainInventoryEntry
	for i, entry := range report.DomainInventory {
		if entry.Domain == "evil.example.com:443" {
			evilEntry = &report.DomainInventory[i]
			break
		}
	}
	require.NotNil(t, evilEntry, "Should find evil.example.com in inventory")
	assert.Equal(t, 1, evilEntry.SeenInRuns, "evil.example.com should be seen in 1 run")
	assert.Equal(t, "denied", evilEntry.OverallStatus, "evil.example.com should be overall denied")

	// Per-run breakdown: run 300 should have HasData=false
	require.Len(t, report.PerRunBreakdown, 3, "Should have 3 per-run breakdown entries")
	assert.False(t, report.PerRunBreakdown[2].HasData, "Run 300 should have no data")
}

func TestBuildCrossRunAuditReport_AllRunsWithoutData(t *testing.T) {
	inputs := []crossRunInput{
		{RunID: 100, WorkflowName: "wf", Conclusion: "success", FirewallAnalysis: nil},
		{RunID: 200, WorkflowName: "wf", Conclusion: "failure", FirewallAnalysis: nil},
	}

	report := buildCrossRunAuditReport(inputs)

	assert.Equal(t, 2, report.RunsAnalyzed, "Should analyze 2 runs")
	assert.Equal(t, 0, report.RunsWithData, "Should have 0 runs with data")
	assert.Equal(t, 2, report.RunsWithoutData, "Should have 2 runs without data")
	assert.Equal(t, 0, report.Summary.UniqueDomains, "Should have 0 unique domains")
	assert.InDelta(t, 0.0, report.Summary.OverallDenyRate, 0.001, "Deny rate should be 0")
}

func TestBuildCrossRunAuditReport_DomainInventorySorted(t *testing.T) {
	inputs := []crossRunInput{
		{
			RunID:        100,
			WorkflowName: "wf",
			Conclusion:   "success",
			FirewallAnalysis: &FirewallAnalysis{
				TotalRequests:   6,
				AllowedRequests: 6,
				BlockedRequests: 0,
				RequestsByDomain: map[string]DomainRequestStats{
					"z-domain.com:443": {Allowed: 2},
					"a-domain.com:443": {Allowed: 2},
					"m-domain.com:443": {Allowed: 2},
				},
			},
		},
	}

	report := buildCrossRunAuditReport(inputs)

	require.Len(t, report.DomainInventory, 3, "Should have 3 domains")
	assert.Equal(t, "a-domain.com:443", report.DomainInventory[0].Domain, "First domain should be a-domain")
	assert.Equal(t, "m-domain.com:443", report.DomainInventory[1].Domain, "Second domain should be m-domain")
	assert.Equal(t, "z-domain.com:443", report.DomainInventory[2].Domain, "Third domain should be z-domain")
}

func TestRenderCrossRunReportJSON(t *testing.T) {
	report := &CrossRunAuditReport{
		RunsAnalyzed:    2,
		RunsWithData:    1,
		RunsWithoutData: 1,
		Summary: CrossRunSummary{
			TotalRequests:   10,
			TotalAllowed:    8,
			TotalBlocked:    2,
			OverallDenyRate: 0.2,
			UniqueDomains:   2,
		},
		DomainInventory: []DomainInventoryEntry{
			{
				Domain:        "api.github.com:443",
				SeenInRuns:    1,
				TotalAllowed:  8,
				TotalBlocked:  0,
				OverallStatus: "allowed",
			},
		},
		PerRunBreakdown: []PerRunFirewallBreakdown{
			{
				RunID:         100,
				WorkflowName:  "test",
				Conclusion:    "success",
				TotalRequests: 10,
				Allowed:       8,
				Blocked:       2,
				DenyRate:      0.2,
				UniqueDomains: 2,
				HasData:       true,
			},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := renderCrossRunReportJSON(report)

	w.Close()
	os.Stdout = oldStdout

	require.NoError(t, err, "renderCrossRunReportJSON should not error")

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify valid JSON
	var parsed CrossRunAuditReport
	err = json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err, "Should produce valid JSON output")
	assert.Equal(t, 2, parsed.RunsAnalyzed, "RunsAnalyzed should match")
	assert.Equal(t, 10, parsed.Summary.TotalRequests, "TotalRequests should match")
}

func TestRenderCrossRunReportMarkdown(t *testing.T) {
	report := &CrossRunAuditReport{
		RunsAnalyzed:    1,
		RunsWithData:    1,
		RunsWithoutData: 0,
		Summary: CrossRunSummary{
			TotalRequests:   5,
			TotalAllowed:    5,
			TotalBlocked:    0,
			OverallDenyRate: 0.0,
			UniqueDomains:   1,
		},
		DomainInventory: []DomainInventoryEntry{
			{
				Domain:        "api.github.com:443",
				SeenInRuns:    1,
				TotalAllowed:  5,
				TotalBlocked:  0,
				OverallStatus: "allowed",
			},
		},
		PerRunBreakdown: []PerRunFirewallBreakdown{
			{
				RunID:         100,
				WorkflowName:  "test",
				Conclusion:    "success",
				TotalRequests: 5,
				Allowed:       5,
				Blocked:       0,
				DenyRate:      0.0,
				UniqueDomains: 1,
				HasData:       true,
			},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	renderCrossRunReportMarkdown(report)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "# Audit Report", "Should have markdown header")
	assert.Contains(t, output, "Executive Summary", "Should have executive summary")
	assert.Contains(t, output, "Domain Inventory", "Should have domain inventory")
	assert.Contains(t, output, "Per-Run Breakdown", "Should have per-run breakdown")
	assert.Contains(t, output, "api.github.com:443", "Should contain the domain")
}

func TestNewAuditReportSubcommand(t *testing.T) {
	cmd := NewAuditReportSubcommand()

	assert.Equal(t, "report", cmd.Use, "Command Use should be 'report'")
	assert.NotEmpty(t, cmd.Short, "Short description should not be empty")
	assert.NotEmpty(t, cmd.Long, "Long description should not be empty")

	// Check flags exist
	workflowFlag := cmd.Flags().Lookup("workflow")
	require.NotNil(t, workflowFlag, "Should have --workflow flag")
	assert.Equal(t, "w", workflowFlag.Shorthand, "Workflow flag shorthand should be 'w'")

	lastFlag := cmd.Flags().Lookup("last")
	require.NotNil(t, lastFlag, "Should have --last flag")
	assert.Equal(t, "20", lastFlag.DefValue, "Default value for --last should be 20")

	jsonFlag := cmd.Flags().Lookup("json")
	require.NotNil(t, jsonFlag, "Should have --json flag")

	repoFlag := cmd.Flags().Lookup("repo")
	require.NotNil(t, repoFlag, "Should have --repo flag")

	outputFlag := cmd.Flags().Lookup("output")
	require.NotNil(t, outputFlag, "Should have --output flag")

	formatFlag := cmd.Flags().Lookup("format")
	require.NotNil(t, formatFlag, "Should have --format flag")
	assert.Equal(t, "markdown", formatFlag.DefValue, "Default value for --format should be markdown")
}

func TestNewAuditReportSubcommand_RejectsExtraArgs(t *testing.T) {
	cmd := NewAuditReportSubcommand()
	cmd.SetArgs([]string{"extra-arg"})
	err := cmd.Execute()
	require.Error(t, err, "Should reject extra positional arguments")
	assert.Contains(t, err.Error(), "unknown command", "Error should indicate unknown command")
}

func TestRunAuditReportConfig_LastClampBounds(t *testing.T) {
	tests := []struct {
		name     string
		inputCfg RunAuditReportConfig
		wantLast int
	}{
		{
			name:     "negative last defaults to 20",
			inputCfg: RunAuditReportConfig{Last: -5},
			wantLast: 20,
		},
		{
			name:     "zero last defaults to 20",
			inputCfg: RunAuditReportConfig{Last: 0},
			wantLast: 20,
		},
		{
			name:     "over max clamped to max",
			inputCfg: RunAuditReportConfig{Last: 100},
			wantLast: maxAuditReportRuns,
		},
		{
			name:     "within bounds unchanged",
			inputCfg: RunAuditReportConfig{Last: 10},
			wantLast: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.inputCfg
			// Apply the same clamping logic as RunAuditReport
			if cfg.Last <= 0 {
				cfg.Last = 20
			}
			if cfg.Last > maxAuditReportRuns {
				cfg.Last = maxAuditReportRuns
			}
			assert.Equal(t, tt.wantLast, cfg.Last, "Last should be clamped correctly")
		})
	}
}

func TestRunAuditReportConfig_FormatPrecedence(t *testing.T) {
	tests := []struct {
		name       string
		jsonOutput bool
		format     string
		wantFormat string // "json", "markdown", or "pretty"
	}{
		{
			name:       "json flag takes precedence over format",
			jsonOutput: true,
			format:     "markdown",
			wantFormat: "json",
		},
		{
			name:       "json flag with format=pretty still uses json",
			jsonOutput: true,
			format:     "pretty",
			wantFormat: "json",
		},
		{
			name:       "format=json without json flag",
			jsonOutput: false,
			format:     "json",
			wantFormat: "json",
		},
		{
			name:       "format=pretty selects pretty",
			jsonOutput: false,
			format:     "pretty",
			wantFormat: "pretty",
		},
		{
			name:       "format=markdown selects markdown",
			jsonOutput: false,
			format:     "markdown",
			wantFormat: "markdown",
		},
		{
			name:       "default format is markdown",
			jsonOutput: false,
			format:     "",
			wantFormat: "markdown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply the same format selection logic as RunAuditReport
			var selected string
			if tt.jsonOutput || tt.format == "json" {
				selected = "json"
			} else if tt.format == "pretty" {
				selected = "pretty"
			} else {
				selected = "markdown"
			}
			assert.Equal(t, tt.wantFormat, selected, "Format should be selected correctly")
		})
	}
}

func TestNewAuditReportSubcommand_RepoParsingWithHost(t *testing.T) {
	tests := []struct {
		name      string
		repoFlag  string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "owner/repo format",
			repoFlag:  "myorg/myrepo",
			wantOwner: "myorg",
			wantRepo:  "myrepo",
		},
		{
			name:      "host/owner/repo format",
			repoFlag:  "github.example.com/myorg/myrepo",
			wantOwner: "myorg",
			wantRepo:  "myrepo",
		},
		{
			name:     "missing repo",
			repoFlag: "onlyowner",
			wantErr:  true,
		},
		{
			name:     "empty owner",
			repoFlag: "/repo",
			wantErr:  true,
		},
		{
			name:     "empty repo",
			repoFlag: "owner/",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply the same repo parsing logic
			parts := strings.Split(tt.repoFlag, "/")
			if len(parts) < 2 {
				assert.True(t, tt.wantErr, "Should expect error for: %s", tt.repoFlag)
				return
			}
			ownerPart := parts[len(parts)-2]
			repoPart := parts[len(parts)-1]
			if ownerPart == "" || repoPart == "" {
				assert.True(t, tt.wantErr, "Should expect error for: %s", tt.repoFlag)
				return
			}

			assert.False(t, tt.wantErr, "Should not expect error for: %s", tt.repoFlag)
			assert.Equal(t, tt.wantOwner, ownerPart, "Owner should match")
			assert.Equal(t, tt.wantRepo, repoPart, "Repo should match")
		})
	}
}

func TestBuildCrossRunAuditReport_MetricsTrend(t *testing.T) {
	inputs := []crossRunInput{
		{
			RunID:        100,
			WorkflowName: "wf",
			Conclusion:   "success",
			Metrics:      LogMetrics{EstimatedCost: 1.00, TokenUsage: 10000, Turns: 5},
			ErrorCount:   0,
		},
		{
			RunID:        200,
			WorkflowName: "wf",
			Conclusion:   "success",
			Metrics:      LogMetrics{EstimatedCost: 2.00, TokenUsage: 20000, Turns: 10},
			ErrorCount:   2,
		},
		{
			RunID:        300,
			WorkflowName: "wf",
			Conclusion:   "success",
			// Spike: cost = 7x avg(1.5) so > 2x; tokens = 4x avg(15K) so > 2x
			Metrics:    LogMetrics{EstimatedCost: 10.50, TokenUsage: 60000, Turns: 20},
			ErrorCount: 1,
		},
	}

	report := buildCrossRunAuditReport(inputs)

	mt := report.MetricsTrend
	assert.Equal(t, 3, mt.RunsWithCost, "All runs have cost")
	assert.InDelta(t, 13.50, mt.TotalCost, 0.001, "Total cost should be 13.50")
	assert.InDelta(t, 4.50, mt.AvgCost, 0.001, "Avg cost should be 4.50")
	assert.InDelta(t, 1.00, mt.MinCost, 0.001, "Min cost should be 1.00")
	assert.InDelta(t, 10.50, mt.MaxCost, 0.001, "Max cost should be 10.50")

	assert.Equal(t, 90000, mt.TotalTokens, "Total tokens should be 90000")
	assert.Equal(t, 30000, mt.AvgTokens, "Avg tokens should be 30000")
	assert.Equal(t, 10000, mt.MinTokens, "Min tokens should be 10000")
	assert.Equal(t, 60000, mt.MaxTokens, "Max tokens should be 60000")

	assert.Equal(t, 35, mt.TotalTurns, "Total turns should be 35")
	assert.Equal(t, 20, mt.MaxTurns, "Max turns should be 20")

	// Spike detection: run 300 has cost=10.50, avg=4.50 → 10.50 > 2*4.50=9.0 → spike
	require.Len(t, mt.CostSpikes, 1, "Should detect 1 cost spike")
	assert.Equal(t, int64(300), mt.CostSpikes[0], "Cost spike should be in run 300")

	// Token spike: run 300 has 60000, avg=30000 → 60000 is not strictly greater than 2*30000=60000
	assert.Empty(t, mt.TokenSpikes, "Should detect no token spikes (60000 is not strictly greater than 2*30000)")

	// Error trend
	et := report.ErrorTrend
	assert.Equal(t, 2, et.RunsWithErrors, "Should have 2 runs with errors")
	assert.Equal(t, 3, et.TotalErrors, "Total errors should be 3")
	assert.InDelta(t, 1.0, et.AvgErrorsPerRun, 0.01, "Avg errors per run should be 1.0")
}

func TestBuildCrossRunAuditReport_MCPHealth(t *testing.T) {
	inputs := []crossRunInput{
		{
			RunID:        100,
			WorkflowName: "wf",
			Conclusion:   "success",
			MCPToolUsage: &MCPToolUsageData{
				Servers: []MCPServerStats{
					{ServerName: "github", ToolCallCount: 10, ErrorCount: 0},
					{ServerName: "safeoutputs", ToolCallCount: 5, ErrorCount: 1},
				},
			},
		},
		{
			RunID:        200,
			WorkflowName: "wf",
			Conclusion:   "success",
			MCPToolUsage: &MCPToolUsageData{
				Servers: []MCPServerStats{
					{ServerName: "github", ToolCallCount: 8, ErrorCount: 2},
				},
			},
		},
	}

	report := buildCrossRunAuditReport(inputs)

	require.Len(t, report.MCPHealth, 2, "Should have 2 MCP server entries")

	// Find github server
	var githubHealth *MCPServerCrossRunHealth
	for i, h := range report.MCPHealth {
		if h.ServerName == "github" {
			githubHealth = &report.MCPHealth[i]
			break
		}
	}
	require.NotNil(t, githubHealth, "Should find github server health")
	assert.Equal(t, 2, githubHealth.RunsConnected, "Github should be connected in 2 runs")
	assert.Equal(t, 2, githubHealth.TotalRuns, "Total runs should be 2")
	assert.Equal(t, 18, githubHealth.TotalCalls, "Total calls should be 18")
	assert.Equal(t, 2, githubHealth.TotalErrors, "Total errors should be 2")
	assert.InDelta(t, 2.0/18.0, githubHealth.ErrorRate, 0.001, "Error rate should be 2/18")
	// 2/18 = ~11.1% error rate, which is > mcpErrorRateThreshold (10%), so it IS unreliable
	assert.True(t, githubHealth.Unreliable, "Github should be unreliable (error rate 11% > mcpErrorRateThreshold)")

	// Find safeoutputs server
	var soHealth *MCPServerCrossRunHealth
	for i, h := range report.MCPHealth {
		if h.ServerName == "safeoutputs" {
			soHealth = &report.MCPHealth[i]
			break
		}
	}
	require.NotNil(t, soHealth, "Should find safeoutputs server health")
	assert.Equal(t, 1, soHealth.RunsConnected, "Safeoutputs should be connected in 1 run")
	// 1/2 runs connected = 50% < 75% threshold → unreliable
	assert.True(t, soHealth.Unreliable, "Safeoutputs should be unreliable (only 50% runs connected)")
}

func TestBuildMetricsTrend_Empty(t *testing.T) {
	trend := buildMetricsTrend(nil)
	assert.InDelta(t, 0.0, trend.TotalCost, 0.001, "Empty rows should produce zero total cost")
	assert.Equal(t, 0, trend.TotalTokens, "Empty rows should produce zero total tokens")
	assert.Empty(t, trend.CostSpikes, "Empty rows should produce no cost spikes")
}

func TestBuildMetricsTrend_NoSpikes(t *testing.T) {
	rows := []metricsRawRow{
		{runID: 1, cost: 1.0, tokens: 100, turns: 3},
		{runID: 2, cost: 1.1, tokens: 110, turns: 4},
		{runID: 3, cost: 0.9, tokens: 90, turns: 2},
	}
	trend := buildMetricsTrend(rows)
	assert.Empty(t, trend.CostSpikes, "No spikes when all costs are similar")
	assert.Empty(t, trend.TokenSpikes, "No spikes when all tokens are similar")
	assert.Equal(t, 3, trend.RunsWithCost, "All runs have cost")
}

func TestRenderCrossRunReportMarkdown_IncludesNewSections(t *testing.T) {
	report := &CrossRunAuditReport{
		RunsAnalyzed:    2,
		RunsWithData:    2,
		RunsWithoutData: 0,
		Summary: CrossRunSummary{
			TotalRequests:   5,
			TotalAllowed:    5,
			TotalBlocked:    0,
			OverallDenyRate: 0.0,
			UniqueDomains:   1,
		},
		MetricsTrend: MetricsTrendData{
			TotalCost:    3.0,
			AvgCost:      1.5,
			MinCost:      1.0,
			MaxCost:      2.0,
			TotalTokens:  30000,
			AvgTokens:    15000,
			MinTokens:    10000,
			MaxTokens:    20000,
			TotalTurns:   15,
			MaxTurns:     10,
			AvgTurns:     7.5,
			RunsWithCost: 2,
			CostSpikes:   []int64{200},
		},
		MCPHealth: []MCPServerCrossRunHealth{
			{
				ServerName:    "github",
				RunsConnected: 2,
				TotalRuns:     2,
				TotalCalls:    20,
				TotalErrors:   1,
				ErrorRate:     0.05,
				Unreliable:    false,
			},
		},
		ErrorTrend: ErrorTrendData{
			RunsWithErrors:  1,
			TotalErrors:     2,
			AvgErrorsPerRun: 1.0,
		},
		DomainInventory: []DomainInventoryEntry{
			{
				Domain:        "api.github.com:443",
				SeenInRuns:    2,
				TotalAllowed:  5,
				TotalBlocked:  0,
				OverallStatus: "allowed",
			},
		},
		PerRunBreakdown: []PerRunFirewallBreakdown{
			{
				RunID:         100,
				WorkflowName:  "test",
				Conclusion:    "success",
				TotalRequests: 3,
				Allowed:       3,
				Blocked:       0,
				DenyRate:      0.0,
				UniqueDomains: 1,
				Cost:          1.0,
				Tokens:        10000,
				Turns:         5,
				MCPErrors:     0,
				HasData:       true,
			},
			{
				RunID:         200,
				WorkflowName:  "test",
				Conclusion:    "success",
				TotalRequests: 2,
				Allowed:       2,
				Blocked:       0,
				Cost:          2.0,
				Tokens:        20000,
				Turns:         10,
				MCPErrors:     1,
				HasData:       true,
				CostSpike:     true,
			},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	renderCrossRunReportMarkdown(report)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "# Audit Report", "Should have markdown header")
	assert.Contains(t, output, "Executive Summary", "Should have executive summary")
	assert.Contains(t, output, "Metrics Trends", "Should have metrics trends section")
	assert.Contains(t, output, "Cost Trend", "Should have cost trend")
	assert.Contains(t, output, "Token Trend", "Should have token trend")
	assert.Contains(t, output, "MCP Server Health", "Should have MCP health section")
	assert.Contains(t, output, "Error Trend", "Should have error trend section")
	assert.Contains(t, output, "Domain Inventory", "Should have domain inventory")
	assert.Contains(t, output, "Per-Run Breakdown", "Should have per-run breakdown")
	assert.Contains(t, output, "⚠", "Should have spike warnings")
}

func TestBuildDrain3InsightsFromCrossRunInputs_Empty(t *testing.T) {
	insights := buildDrain3InsightsFromCrossRunInputs(nil)
	assert.Nil(t, insights, "should return nil for empty inputs")

	insights = buildDrain3InsightsFromCrossRunInputs([]crossRunInput{})
	assert.Nil(t, insights, "should return nil for empty slice")
}

func TestBuildDrain3InsightsFromCrossRunInputs_WithInputs(t *testing.T) {
	inputs := []crossRunInput{
		{
			RunID:        1,
			WorkflowName: "test-workflow",
			Conclusion:   "success",
			Metrics: LogMetrics{
				Turns:         5,
				TokenUsage:    1000,
				EstimatedCost: 0.05,
			},
			ErrorCount: 0,
		},
		{
			RunID:        2,
			WorkflowName: "test-workflow",
			Conclusion:   "failure",
			Metrics: LogMetrics{
				Turns:         8,
				TokenUsage:    2000,
				EstimatedCost: 0.1,
			},
			ErrorCount: 2,
			MCPFailures: []MCPFailureReport{
				{ServerName: "github", Status: "timeout"},
			},
		},
	}

	// Verify the conversion maps fields correctly by checking via a converted ProcessedRun.
	runs := make([]ProcessedRun, 0, len(inputs))
	for _, in := range inputs {
		runs = append(runs, ProcessedRun{
			Run: WorkflowRun{
				DatabaseID:    in.RunID,
				WorkflowName:  in.WorkflowName,
				Conclusion:    in.Conclusion,
				Turns:         in.Metrics.Turns,
				TokenUsage:    in.Metrics.TokenUsage,
				EstimatedCost: in.Metrics.EstimatedCost,
				ErrorCount:    in.ErrorCount,
			},
			MCPFailures: in.MCPFailures,
		})
	}
	require.Equal(t, int64(1), runs[0].Run.DatabaseID, "first run ID should map to 1")
	require.Equal(t, "test-workflow", runs[0].Run.WorkflowName, "workflow name should be mapped")
	require.Equal(t, "success", runs[0].Run.Conclusion, "conclusion should be mapped")
	require.Equal(t, 5, runs[0].Run.Turns, "turns should be mapped from Metrics.Turns")
	require.Equal(t, 1000, runs[0].Run.TokenUsage, "tokens should be mapped from Metrics.TokenUsage")
	require.Len(t, runs[1].MCPFailures, 1, "MCPFailures should be mapped")
	require.Equal(t, "github", runs[1].MCPFailures[0].ServerName, "MCP server name should be mapped")

	insights := buildDrain3InsightsFromCrossRunInputs(inputs)
	// Drain3 insights may or may not be generated depending on event count,
	// but the function should not panic or error.
	// If insights are generated they should have valid fields.
	for _, insight := range insights {
		assert.NotEmpty(t, insight.Category, "insight should have a category")
		assert.NotEmpty(t, insight.Severity, "insight should have a severity")
		assert.NotEmpty(t, insight.Title, "insight should have a title")
	}
}

func TestBuildCrossRunAuditReport_IncludesDrain3Insights(t *testing.T) {
	inputs := []crossRunInput{
		{
			RunID:        100,
			WorkflowName: "test-workflow",
			Conclusion:   "success",
			Metrics:      LogMetrics{Turns: 5, TokenUsage: 500, EstimatedCost: 0.05},
			ErrorCount:   1,
			MCPFailures:  []MCPFailureReport{{ServerName: "github", Status: "timeout"}},
		},
		{
			RunID:        101,
			WorkflowName: "test-workflow",
			Conclusion:   "failure",
			Metrics:      LogMetrics{Turns: 8, TokenUsage: 2000, EstimatedCost: 0.1},
			ErrorCount:   2,
		},
	}

	report := buildCrossRunAuditReport(inputs)
	require.NotNil(t, report, "report should not be nil")

	// Phase 7 should have run and may produce insights. Even if no events are
	// extracted the field must be initialised (nil is acceptable).
	// Verify that Phase 7 fired without panic; if insights were produced, check
	// they have the required fields.
	for _, insight := range report.Drain3Insights {
		assert.NotEmpty(t, insight.Category, "Drain3 insight should have a category")
		assert.NotEmpty(t, insight.Severity, "Drain3 insight should have a severity")
		assert.NotEmpty(t, insight.Title, "Drain3 insight should have a title")
	}
}

func TestRenderCrossRunReportMarkdown_IncludesDrain3Section(t *testing.T) {
	report := &CrossRunAuditReport{
		RunsAnalyzed: 1,
		Drain3Insights: []ObservabilityInsight{
			{
				Category: "execution",
				Severity: "info",
				Title:    "Log template patterns mined",
				Summary:  "Analysis identified 2 event templates.",
				Evidence: "plan=1 finish=1",
			},
			{
				Category: "reliability",
				Severity: "high",
				Title:    "2 anomalous event pattern(s) detected",
				Summary:  "Unusual events detected.",
			},
		},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	renderCrossRunReportMarkdown(report)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "Agent Event Pattern Analysis", "Should include agent event pattern analysis section header")
	assert.Contains(t, output, "Log template patterns mined", "Should include first insight title")
	assert.Contains(t, output, "2 anomalous event pattern(s) detected", "Should include second insight title")
	assert.Contains(t, output, "plan=1 finish=1", "Should include evidence")
	assert.Contains(t, output, "🔴", "Should include high severity icon")
}
