package cli

import (
	"context"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/sliceutil"
)

var auditDiffLog = logger.New("cli:audit_diff")

// volumeChangeThresholdPercent is the minimum percentage increase to flag as a volume change.
// >100% increase means the request count more than doubled.
const volumeChangeThresholdPercent = 100.0

// DomainDiffEntry represents the diff for a single domain between two runs
type DomainDiffEntry struct {
	Domain       string `json:"domain"`
	Status       string `json:"status"`                  // "new", "removed", "status_changed", "volume_changed"
	Run1Allowed  int    `json:"run1_allowed"`            // Allowed requests in run 1
	Run1Blocked  int    `json:"run1_blocked"`            // Blocked requests in run 1
	Run2Allowed  int    `json:"run2_allowed"`            // Allowed requests in run 2
	Run2Blocked  int    `json:"run2_blocked"`            // Blocked requests in run 2
	Run1Status   string `json:"run1_status,omitempty"`   // "allowed", "denied", or "" for new domains
	Run2Status   string `json:"run2_status,omitempty"`   // "allowed", "denied", or "" for removed domains
	VolumeChange string `json:"volume_change,omitempty"` // e.g. "+287%" or "-50%"
	IsAnomaly    bool   `json:"is_anomaly,omitempty"`    // Flagged as anomalous (new denied, status flip to allowed)
	AnomalyNote  string `json:"anomaly_note,omitempty"`  // Human-readable anomaly explanation
}

// FirewallDiff represents the complete diff between two runs' firewall behavior
type FirewallDiff struct {
	Run1ID         int64               `json:"run1_id"`
	Run2ID         int64               `json:"run2_id"`
	NewDomains     []DomainDiffEntry   `json:"new_domains,omitempty"`
	RemovedDomains []DomainDiffEntry   `json:"removed_domains,omitempty"`
	StatusChanges  []DomainDiffEntry   `json:"status_changes,omitempty"`
	VolumeChanges  []DomainDiffEntry   `json:"volume_changes,omitempty"`
	Summary        FirewallDiffSummary `json:"summary"`
}

// FirewallDiffSummary provides a quick overview of the diff
type FirewallDiffSummary struct {
	NewDomainCount     int  `json:"new_domain_count"`
	RemovedDomainCount int  `json:"removed_domain_count"`
	StatusChangeCount  int  `json:"status_change_count"`
	VolumeChangeCount  int  `json:"volume_change_count"`
	HasAnomalies       bool `json:"has_anomalies"`
	AnomalyCount       int  `json:"anomaly_count"`
}

// computeFirewallDiff computes the diff between two FirewallAnalysis results.
// run1 is the "before" (baseline) and run2 is the "after" (comparison target).
// Either analysis may be nil, indicating no firewall data for that run.
func computeFirewallDiff(run1ID, run2ID int64, run1, run2 *FirewallAnalysis) *FirewallDiff {
	auditDiffLog.Printf("Computing firewall diff: run1=%d, run2=%d", run1ID, run2ID)
	diff := &FirewallDiff{
		Run1ID: run1ID,
		Run2ID: run2ID,
	}

	run1Stats, run2Stats := computeFirewallDiffStats(run1, run2)

	// If both are nil/empty, return empty diff
	if len(run1Stats) == 0 && len(run2Stats) == 0 {
		return diff
	}

	anomalyCount := 0
	for _, domain := range computeFirewallDiffSortedDomains(run1Stats, run2Stats) {
		stats1, inRun1 := run1Stats[domain]
		stats2, inRun2 := run2Stats[domain]
		anomalyCount += computeFirewallDiffDomain(diff, domain, stats1, stats2, inRun1, inRun2)
	}

	diff.Summary = FirewallDiffSummary{
		NewDomainCount:     len(diff.NewDomains),
		RemovedDomainCount: len(diff.RemovedDomains),
		StatusChangeCount:  len(diff.StatusChanges),
		VolumeChangeCount:  len(diff.VolumeChanges),
		HasAnomalies:       anomalyCount > 0,
		AnomalyCount:       anomalyCount,
	}

	auditDiffLog.Printf("Firewall diff complete: new=%d, removed=%d, status_changes=%d, volume_changes=%d, anomalies=%d",
		len(diff.NewDomains), len(diff.RemovedDomains), len(diff.StatusChanges), len(diff.VolumeChanges), anomalyCount)
	return diff
}

func computeFirewallDiffStats(run1, run2 *FirewallAnalysis) (map[string]DomainRequestStats, map[string]DomainRequestStats) {
	run1Stats := make(map[string]DomainRequestStats)
	run2Stats := make(map[string]DomainRequestStats)
	if run1 != nil {
		run1Stats = run1.RequestsByDomain
	}
	if run2 != nil {
		run2Stats = run2.RequestsByDomain
	}
	return run1Stats, run2Stats
}

func computeFirewallDiffSortedDomains(run1Stats, run2Stats map[string]DomainRequestStats) []string {
	allDomains := make(map[string]struct{})
	for domain := range run1Stats {
		allDomains[domain] = struct{}{}
	}
	for domain := range run2Stats {
		allDomains[domain] = struct{}{}
	}
	return sliceutil.SortedKeys(allDomains)
}

func computeFirewallDiffDomain(diff *FirewallDiff, domain string, stats1, stats2 DomainRequestStats, inRun1, inRun2 bool) int {
	switch {
	case !inRun1 && inRun2:
		entry := computeFirewallDiffNewDomain(domain, stats2)
		diff.NewDomains = append(diff.NewDomains, entry)
		if entry.IsAnomaly {
			return 1
		}
	case inRun1 && !inRun2:
		entry := computeFirewallDiffRemovedDomain(domain, stats1)
		diff.RemovedDomains = append(diff.RemovedDomains, entry)
		if entry.IsAnomaly {
			return 1
		}
	default:
		return computeFirewallDiffExistingDomain(diff, domain, stats1, stats2)
	}
	return 0
}

func computeFirewallDiffNewDomain(domain string, stats2 DomainRequestStats) DomainDiffEntry {
	entry := DomainDiffEntry{
		Domain:      domain,
		Status:      "new",
		Run2Allowed: stats2.Allowed,
		Run2Blocked: stats2.Blocked,
		Run2Status:  classifyFirewallDomainStatus(stats2),
	}
	if stats2.Blocked > 0 {
		entry.IsAnomaly = true
		entry.AnomalyNote = "new denied domain"
	}
	return entry
}

func computeFirewallDiffRemovedDomain(domain string, stats1 DomainRequestStats) DomainDiffEntry {
	entry := DomainDiffEntry{
		Domain:      domain,
		Status:      "removed",
		Run1Allowed: stats1.Allowed,
		Run1Blocked: stats1.Blocked,
		Run1Status:  classifyFirewallDomainStatus(stats1),
	}
	// Anomaly: the removed domain was denied in the base run. This indicates a
	// transient firewall block that prevented the agent from reaching an MCP server.
	if stats1.Blocked > 0 {
		entry.IsAnomaly = true
		entry.AnomalyNote = "denied in base run — absent from comparison run"
	}
	return entry
}

func computeFirewallDiffExistingDomain(diff *FirewallDiff, domain string, stats1, stats2 DomainRequestStats) int {
	status1 := classifyFirewallDomainStatus(stats1)
	status2 := classifyFirewallDomainStatus(stats2)
	if status1 != status2 {
		entry := computeFirewallDiffStatusChange(domain, stats1, stats2, status1, status2)
		diff.StatusChanges = append(diff.StatusChanges, entry)
		if entry.IsAnomaly {
			return 1
		}
		return 0
	}
	if entry, ok := computeFirewallDiffVolumeChange(domain, stats1, stats2, status1, status2); ok {
		diff.VolumeChanges = append(diff.VolumeChanges, entry)
	}
	return 0
}

func computeFirewallDiffStatusChange(domain string, stats1, stats2 DomainRequestStats, status1, status2 string) DomainDiffEntry {
	entry := DomainDiffEntry{
		Domain: domain, Status: "status_changed", Run1Allowed: stats1.Allowed,
		Run1Blocked: stats1.Blocked, Run2Allowed: stats2.Allowed, Run2Blocked: stats2.Blocked,
		Run1Status: status1, Run2Status: status2,
	}
	if status1 == "denied" && status2 == "allowed" {
		entry.IsAnomaly = true
		entry.AnomalyNote = "previously denied, now allowed"
	}
	if status1 == "allowed" && status2 == "denied" {
		entry.IsAnomaly = true
		entry.AnomalyNote = "previously allowed, now denied"
	}
	return entry
}

func computeFirewallDiffVolumeChange(domain string, stats1, stats2 DomainRequestStats, status1, status2 string) (DomainDiffEntry, bool) {
	total1 := stats1.Allowed + stats1.Blocked
	total2 := stats2.Allowed + stats2.Blocked
	if total1 <= 0 {
		return DomainDiffEntry{}, false
	}
	pctChange := (float64(total2-total1) / float64(total1)) * 100
	if math.Abs(pctChange) <= volumeChangeThresholdPercent {
		return DomainDiffEntry{}, false
	}
	return DomainDiffEntry{
		Domain: domain, Status: "volume_changed", Run1Allowed: stats1.Allowed,
		Run1Blocked: stats1.Blocked, Run2Allowed: stats2.Allowed, Run2Blocked: stats2.Blocked,
		Run1Status: status1, Run2Status: status2, VolumeChange: formatVolumeChange(total1, total2),
	}, true
}

// classifyFirewallDomainStatus returns "allowed", "denied", or "mixed" based on request stats
func classifyFirewallDomainStatus(stats DomainRequestStats) string {
	if stats.Allowed > 0 && stats.Blocked == 0 {
		return "allowed"
	}
	if stats.Blocked > 0 && stats.Allowed == 0 {
		return "denied"
	}
	if stats.Allowed > 0 && stats.Blocked > 0 {
		return "mixed"
	}
	return "unknown"
}

// MCPToolDiffEntry represents the diff for a single MCP tool between two runs
type MCPToolDiffEntry struct {
	ServerName      string `json:"server_name"`
	ToolName        string `json:"tool_name"`
	Status          string `json:"status"`                    // "new", "removed", "changed"
	Run1CallCount   int    `json:"run1_call_count,omitempty"` // Call count in run 1
	Run2CallCount   int    `json:"run2_call_count,omitempty"` // Call count in run 2
	Run1ErrorCount  int    `json:"run1_error_count,omitempty"`
	Run2ErrorCount  int    `json:"run2_error_count,omitempty"`
	CallCountChange string `json:"call_count_change,omitempty"` // e.g. "+2", "-3"
	IsAnomaly       bool   `json:"is_anomaly,omitempty"`
	AnomalyNote     string `json:"anomaly_note,omitempty"`
}

// MCPToolsDiff represents the complete diff of MCP tool invocations between two runs
type MCPToolsDiff struct {
	NewTools     []MCPToolDiffEntry  `json:"new_tools,omitempty"`
	RemovedTools []MCPToolDiffEntry  `json:"removed_tools,omitempty"`
	ChangedTools []MCPToolDiffEntry  `json:"changed_tools,omitempty"`
	Summary      MCPToolsDiffSummary `json:"summary"`
}

// MCPToolsDiffSummary provides a quick overview of MCP tool changes
type MCPToolsDiffSummary struct {
	NewToolCount     int  `json:"new_tool_count"`
	RemovedToolCount int  `json:"removed_tool_count"`
	ChangedToolCount int  `json:"changed_tool_count"`
	HasAnomalies     bool `json:"has_anomalies"`
	AnomalyCount     int  `json:"anomaly_count"`
}

// TokenUsageDiff represents the detailed diff of token usage between two runs,
// based on the firewall proxy token-usage.jsonl data from RunSummary.TokenUsage.
type TokenUsageDiff struct {
	Run1InputTokens        int     `json:"run1_input_tokens"`
	Run2InputTokens        int     `json:"run2_input_tokens"`
	InputTokensChange      string  `json:"input_tokens_change,omitempty"`
	Run1OutputTokens       int     `json:"run1_output_tokens"`
	Run2OutputTokens       int     `json:"run2_output_tokens"`
	OutputTokensChange     string  `json:"output_tokens_change,omitempty"`
	Run1CacheReadTokens    int     `json:"run1_cache_read_tokens"`
	Run2CacheReadTokens    int     `json:"run2_cache_read_tokens"`
	CacheReadTokensChange  string  `json:"cache_read_tokens_change,omitempty"`
	Run1CacheWriteTokens   int     `json:"run1_cache_write_tokens"`
	Run2CacheWriteTokens   int     `json:"run2_cache_write_tokens"`
	CacheWriteTokensChange string  `json:"cache_write_tokens_change,omitempty"`
	Run1AIC                float64 `json:"run1_aic,omitempty"`
	Run2AIC                float64 `json:"run2_aic,omitempty"`
	AICChange              string  `json:"aic_change,omitempty"`
	Run1TotalRequests      int     `json:"run1_total_requests"`
	Run2TotalRequests      int     `json:"run2_total_requests"`
	RequestsDelta          string  `json:"requests_delta,omitempty"` // Absolute request-count delta, e.g. "+4"
	Run1CacheEfficiency    float64 `json:"run1_cache_efficiency"`
	Run2CacheEfficiency    float64 `json:"run2_cache_efficiency"`
	CacheEfficiencyChange  string  `json:"cache_efficiency_change,omitempty"` // Percentage-point delta, e.g. "+1.5pp"
}

// ToolCallDiffEntry represents the diff for a single engine-level tool between two runs.
// Tool data comes from RunSummary.Metrics.ToolCalls (LogMetrics.ToolCalls).
type ToolCallDiffEntry struct {
	Name              string `json:"name"`
	Status            string `json:"status"`                         // "new", "removed", "changed", "unchanged"
	Run1CallCount     int    `json:"run1_call_count"`                // Call count in run 1 (0 if new)
	Run2CallCount     int    `json:"run2_call_count"`                // Call count in run 2 (0 if removed)
	CallCountChange   string `json:"call_count_change,omitempty"`    // e.g. "+3", "-1"
	Run1MaxInputSize  int    `json:"run1_max_input_size,omitempty"`  // Max input size (tokens) seen in run 1
	Run2MaxInputSize  int    `json:"run2_max_input_size,omitempty"`  // Max input size (tokens) seen in run 2
	Run1MaxOutputSize int    `json:"run1_max_output_size,omitempty"` // Max output size (tokens) seen in run 1
	Run2MaxOutputSize int    `json:"run2_max_output_size,omitempty"` // Max output size (tokens) seen in run 2
}

// BashCommandsDiff tracks bash-specific tool call differences between two runs.
// It aggregates calls to the generic "bash" / "Bash" tool and per-command "bash_*" entries
// (the latter are generated by the Codex engine log parser which records each unique shell command).
type BashCommandsDiff struct {
	Run1TotalCalls   int                 `json:"run1_total_calls"`
	Run2TotalCalls   int                 `json:"run2_total_calls"`
	TotalCallsChange string              `json:"total_calls_change,omitempty"` // e.g. "+5", "-2"
	Commands         []ToolCallDiffEntry `json:"commands,omitempty"`           // per-command breakdown (from bash_* names)
}

// ToolCallsDiffSummary provides a quick overview of engine-level tool call changes
type ToolCallsDiffSummary struct {
	NewToolCount     int `json:"new_tool_count"`
	RemovedToolCount int `json:"removed_tool_count"`
	ChangedToolCount int `json:"changed_tool_count"`
	Run1TotalCalls   int `json:"run1_total_calls"` // Total across all tools in run 1
	Run2TotalCalls   int `json:"run2_total_calls"` // Total across all tools in run 2
}

// ToolCallsDiff represents the diff of engine-level tool invocations between two runs.
// It uses data from RunSummary.Metrics.ToolCalls (LogMetrics.ToolCalls) which is populated
// by engine-specific log parsers (Claude, Codex, Copilot).
type ToolCallsDiff struct {
	NewTools     []ToolCallDiffEntry  `json:"new_tools,omitempty"`     // Tools only in run 2
	RemovedTools []ToolCallDiffEntry  `json:"removed_tools,omitempty"` // Tools only in run 1
	ChangedTools []ToolCallDiffEntry  `json:"changed_tools,omitempty"` // Tools with changed call counts
	AllTools     []ToolCallDiffEntry  `json:"all_tools,omitempty"`     // Complete view of all tools across both runs
	BashDiff     *BashCommandsDiff    `json:"bash_diff,omitempty"`     // Bash-specific analysis
	Summary      ToolCallsDiffSummary `json:"summary"`
}

// RunMetricsDiff represents the diff of run-level metrics (token usage, duration, turns) between two runs
type RunMetricsDiff struct {
	Run1TokenUsage         int                  `json:"run1_token_usage"`
	Run2TokenUsage         int                  `json:"run2_token_usage"`
	TokenUsageChange       string               `json:"token_usage_change,omitempty"` // e.g. "+15%", "-5%"
	Run1Duration           string               `json:"run1_duration,omitempty"`
	Run2Duration           string               `json:"run2_duration,omitempty"`
	DurationChange         string               `json:"duration_change,omitempty"` // e.g. "+2m30s", "-1m"
	Run1Turns              int                  `json:"run1_turns,omitempty"`
	Run2Turns              int                  `json:"run2_turns,omitempty"`
	TurnsChange            int                  `json:"turns_change,omitempty"`
	Run1TokensPerTurn      int                  `json:"run1_tokens_per_turn,omitempty"`      // Avg token usage per turn in run 1
	Run2TokensPerTurn      int                  `json:"run2_tokens_per_turn,omitempty"`      // Avg token usage per turn in run 2
	TokensPerTurnChange    string               `json:"tokens_per_turn_change,omitempty"`    // e.g. "+20%", "-10%"
	TokenUsageDetails      *TokenUsageDiff      `json:"token_usage_details,omitempty"`       // Detailed breakdown from firewall proxy
	GitHubRateLimitDetails *GitHubRateLimitDiff `json:"github_rate_limit_details,omitempty"` // GitHub API quota consumption diff
	ToolCallsDiff          *ToolCallsDiff       `json:"tool_calls_diff,omitempty"`           // Engine-level tool call diff
}

// GitHubRateLimitDiff represents the diff of GitHub API quota consumption between two runs.
// It is populated from the github_rate_limits.jsonl artifact (GitHubRateLimitUsage).
type GitHubRateLimitDiff struct {
	Run1TotalAPICalls  int    `json:"run1_total_api_calls"`
	Run2TotalAPICalls  int    `json:"run2_total_api_calls"`
	APICallsChange     string `json:"api_calls_change,omitempty"` // e.g. "+20%", "-5%"
	Run1CoreConsumed   int    `json:"run1_core_consumed,omitempty"`
	Run2CoreConsumed   int    `json:"run2_core_consumed,omitempty"`
	CoreConsumedChange string `json:"core_consumed_change,omitempty"` // e.g. "+10%", "-3%"
	Run1CoreRemaining  int    `json:"run1_core_remaining,omitempty"`
	Run2CoreRemaining  int    `json:"run2_core_remaining,omitempty"`
	Run1CoreLimit      int    `json:"run1_core_limit,omitempty"`
	Run2CoreLimit      int    `json:"run2_core_limit,omitempty"`
}

// AuditDiff is the top-level diff combining firewall behavior, MCP tool invocations,
// and run-level metrics between two workflow runs.
type AuditDiff struct {
	Run1ID         int64           `json:"run1_id"`
	Run2ID         int64           `json:"run2_id"`
	FirewallDiff   *FirewallDiff   `json:"firewall_diff,omitempty"`
	MCPToolsDiff   *MCPToolsDiff   `json:"mcp_tools_diff,omitempty"`
	RunMetricsDiff *RunMetricsDiff `json:"run_metrics_diff,omitempty"`
}

// computeAuditDiff produces a full AuditDiff combining firewall, MCP tool, and run metrics diffs.
func computeAuditDiff(run1ID, run2ID int64, summary1, summary2 *RunSummary) *AuditDiff {
	auditDiffLog.Printf("Computing full audit diff: run1=%d, run2=%d", run1ID, run2ID)
	diff := &AuditDiff{
		Run1ID: run1ID,
		Run2ID: run2ID,
	}

	var fw1, fw2 *FirewallAnalysis
	if summary1 != nil {
		fw1 = summary1.FirewallAnalysis
	}
	if summary2 != nil {
		fw2 = summary2.FirewallAnalysis
	}
	diff.FirewallDiff = computeFirewallDiff(run1ID, run2ID, fw1, fw2)

	var mcp1, mcp2 *MCPToolUsageData
	if summary1 != nil {
		mcp1 = summary1.MCPToolUsage
	}
	if summary2 != nil {
		mcp2 = summary2.MCPToolUsage
	}
	if mcp1 != nil || mcp2 != nil {
		diff.MCPToolsDiff = computeMCPToolsDiff(mcp1, mcp2)
	}

	metricsDiff := computeRunMetricsDiff(summary1, summary2)
	if metricsDiff != nil {
		diff.RunMetricsDiff = metricsDiff
	}

	return diff
}

// mcpToolKey returns a unique key for an MCP tool given its server and tool name.
func mcpToolKey(serverName, toolName string) string {
	return serverName + ":" + toolName
}

// computeMCPToolsDiff computes the diff between two runs' MCP tool usage.
// run1 is the "before" (baseline) and run2 is the "after" (comparison target).
func computeMCPToolsDiff(run1, run2 *MCPToolUsageData) *MCPToolsDiff {
	run1Count, run2Count := 0, 0
	if run1 != nil {
		run1Count = len(run1.Summary)
	}
	if run2 != nil {
		run2Count = len(run2.Summary)
	}
	auditDiffLog.Printf("Computing MCP tools diff: run1_tools=%d, run2_tools=%d", run1Count, run2Count)
	run1Tools, run2Tools := computeMCPToolsDiffMaps(run1, run2)

	diff := &MCPToolsDiff{}
	anomalyCount := 0

	for _, key := range computeMCPToolsDiffSortedKeys(run1Tools, run2Tools) {
		s1, inRun1 := run1Tools[key]
		s2, inRun2 := run2Tools[key]
		anomalyCount += computeMCPToolsDiffEntry(diff, s1, s2, inRun1, inRun2)
	}

	diff.Summary = MCPToolsDiffSummary{
		NewToolCount:     len(diff.NewTools),
		RemovedToolCount: len(diff.RemovedTools),
		ChangedToolCount: len(diff.ChangedTools),
		HasAnomalies:     anomalyCount > 0,
		AnomalyCount:     anomalyCount,
	}

	return diff
}

func computeMCPToolsDiffMaps(run1, run2 *MCPToolUsageData) (map[string]MCPToolSummary, map[string]MCPToolSummary) {
	run1Tools := make(map[string]MCPToolSummary)
	run2Tools := make(map[string]MCPToolSummary)
	if run1 != nil {
		for _, s := range run1.Summary {
			run1Tools[mcpToolKey(s.ServerName, s.ToolName)] = s
		}
	}
	if run2 != nil {
		for _, s := range run2.Summary {
			run2Tools[mcpToolKey(s.ServerName, s.ToolName)] = s
		}
	}
	return run1Tools, run2Tools
}

func computeMCPToolsDiffSortedKeys(run1Tools, run2Tools map[string]MCPToolSummary) []string {
	allKeys := make(map[string]struct{})
	for k := range run1Tools {
		allKeys[k] = struct{}{}
	}
	for k := range run2Tools {
		allKeys[k] = struct{}{}
	}
	return sliceutil.SortedKeys(allKeys)
}

func computeMCPToolsDiffEntry(diff *MCPToolsDiff, s1, s2 MCPToolSummary, inRun1, inRun2 bool) int {
	switch {
	case !inRun1 && inRun2:
		entry := MCPToolDiffEntry{ServerName: s2.ServerName, ToolName: s2.ToolName, Status: "new", Run2CallCount: s2.CallCount, Run2ErrorCount: s2.ErrorCount}
		if s2.ErrorCount > 0 {
			entry.IsAnomaly = true
			entry.AnomalyNote = "new tool with errors"
			diff.NewTools = append(diff.NewTools, entry)
			return 1
		}
		diff.NewTools = append(diff.NewTools, entry)
	case inRun1 && !inRun2:
		diff.RemovedTools = append(diff.RemovedTools, MCPToolDiffEntry{ServerName: s1.ServerName, ToolName: s1.ToolName, Status: "removed", Run1CallCount: s1.CallCount, Run1ErrorCount: s1.ErrorCount})
	case s1.CallCount != s2.CallCount || s1.ErrorCount != s2.ErrorCount:
		entry := MCPToolDiffEntry{ServerName: s1.ServerName, ToolName: s1.ToolName, Status: "changed", Run1CallCount: s1.CallCount, Run2CallCount: s2.CallCount, Run1ErrorCount: s1.ErrorCount, Run2ErrorCount: s2.ErrorCount, CallCountChange: formatCountChange(s1.CallCount, s2.CallCount)}
		if s2.ErrorCount > s1.ErrorCount {
			entry.IsAnomaly = true
			entry.AnomalyNote = "error count increased"
			diff.ChangedTools = append(diff.ChangedTools, entry)
			return 1
		}
		diff.ChangedTools = append(diff.ChangedTools, entry)
	}
	return 0
}

// computeRunMetricsDiff computes the diff of run-level metrics between two runs.
// Returns nil if no meaningful metrics data is available.
func computeRunMetricsDiff(summary1, summary2 *RunSummary) *RunMetricsDiff {
	run1 := computeRunMetricsDiffValues(summary1)
	run2 := computeRunMetricsDiffValues(summary2)

	// Skip if there is no meaningful data
	hasTokenDetails := run1.tokenUsage != nil || run2.tokenUsage != nil
	hasRateLimitDetails := run1.rateLimit != nil || run2.rateLimit != nil
	if !computeRunMetricsDiffHasData(run1, run2, hasTokenDetails, hasRateLimitDetails) {
		return nil
	}

	diff := &RunMetricsDiff{
		Run1TokenUsage: run1.tokens,
		Run2TokenUsage: run2.tokens,
		Run1Turns:      run1.turns,
		Run2Turns:      run2.turns,
		TurnsChange:    run2.turns - run1.turns,
	}

	computeRunMetricsDiffPopulateTokenAndDuration(diff, run1, run2)

	// Compute tokens per turn using engine-level token usage.
	run1PerTurn := run1.tokens
	run2PerTurn := run2.tokens
	if run1.turns > 0 {
		diff.Run1TokensPerTurn = run1PerTurn / run1.turns
	}
	if run2.turns > 0 {
		diff.Run2TokensPerTurn = run2PerTurn / run2.turns
	}
	if diff.Run1TokensPerTurn > 0 || diff.Run2TokensPerTurn > 0 {
		diff.TokensPerTurnChange = formatVolumeChange(diff.Run1TokensPerTurn, diff.Run2TokensPerTurn)
	}

	diff.TokenUsageDetails = computeTokenUsageDiff(run1.tokenUsage, run2.tokenUsage)
	diff.GitHubRateLimitDetails = computeGitHubRateLimitDiff(run1.rateLimit, run2.rateLimit)
	diff.ToolCallsDiff = computeToolCallsDiff(run1.metrics, run2.metrics)

	auditDiffLog.Printf("Run metrics diff: tokens %d->%d, turns %d->%d, has_token_details=%t, has_rate_limit_details=%t", run1.tokens, run2.tokens, run1.turns, run2.turns, hasTokenDetails, hasRateLimitDetails)
	return diff
}

type computeRunMetricsDiffSummaryValues struct {
	tokens     int
	duration   time.Duration
	turns      int
	tokenUsage *TokenUsageSummary
	rateLimit  *GitHubRateLimitUsage
	metrics    *LogMetrics
}

func computeRunMetricsDiffValues(summary *RunSummary) computeRunMetricsDiffSummaryValues {
	var values computeRunMetricsDiffSummaryValues
	if summary == nil {
		return values
	}
	values.tokens = summary.Run.TokenUsage
	values.duration = summary.Run.Duration
	// Run.Turns may be zero on cached-summary paths; Metrics.Turns is authoritative.
	values.turns = summary.Run.Turns
	if values.turns == 0 && summary.Metrics.Turns > 0 {
		values.turns = summary.Metrics.Turns
	}
	values.tokenUsage = summary.TokenUsage
	values.rateLimit = summary.GitHubRateLimitUsage
	values.metrics = &summary.Metrics
	return values
}

func computeRunMetricsDiffHasData(run1, run2 computeRunMetricsDiffSummaryValues, hasTokenDetails, hasRateLimitDetails bool) bool {
	return run1.tokens != 0 || run2.tokens != 0 || run1.duration != 0 || run2.duration != 0 ||
		run1.turns != 0 || run2.turns != 0 || hasTokenDetails || hasRateLimitDetails
}

func computeRunMetricsDiffPopulateTokenAndDuration(diff *RunMetricsDiff, run1, run2 computeRunMetricsDiffSummaryValues) {
	if run1.tokens > 0 || run2.tokens > 0 {
		diff.TokenUsageChange = formatVolumeChange(run1.tokens, run2.tokens)
	}
	if run1.duration > 0 {
		diff.Run1Duration = run1.duration.Round(time.Second).String()
	}
	if run2.duration > 0 {
		diff.Run2Duration = run2.duration.Round(time.Second).String()
	}
	if run1.duration > 0 && run2.duration > 0 {
		delta := run2.duration - run1.duration
		if delta >= 0 {
			diff.DurationChange = "+" + delta.Round(time.Second).String()
		} else {
			diff.DurationChange = delta.Round(time.Second).String()
		}
	}
}

// isBashTool returns true if the tool name represents a bash/shell invocation.
// It matches the generic "bash" / "Bash" tool names used by most engines and the
// per-command "bash_*" entries generated by the Codex log parser.
func isBashTool(name string) bool {
	lower := strings.ToLower(name)
	return lower == "bash" || strings.HasPrefix(lower, "bash_") //nolint:tolowerequalfold
}

// computeToolCallsDiff diffs engine-level tool calls from two LogMetrics values.
// Returns nil when both metrics have no tool call data.
func computeToolCallsDiff(m1, m2 *LogMetrics) *ToolCallsDiff {
	run1Tools := computeToolCallsDiffMap(m1)
	run2Tools := computeToolCallsDiffMap(m2)

	if len(run1Tools) == 0 && len(run2Tools) == 0 {
		return nil
	}

	diff := &ToolCallsDiff{}
	var run1Total, run2Total int
	// Collect bash tools during the main iteration to avoid a second traversal in computeBashCommandsDiff.
	bashRun1 := make(map[string]ToolCallInfo)
	bashRun2 := make(map[string]ToolCallInfo)

	for _, name := range computeToolCallsDiffSortedNames(run1Tools, run2Tools) {
		tc1, inRun1 := run1Tools[name]
		tc2, inRun2 := run2Tools[name]

		run1Total, run2Total = computeToolCallsDiffTrackBash(computeToolCallsDiffTrackBashParams{
			Name:      name,
			TC1:       tc1,
			TC2:       tc2,
			InRun1:    inRun1,
			InRun2:    inRun2,
			BashRun1:  bashRun1,
			BashRun2:  bashRun2,
			Run1Total: run1Total,
			Run2Total: run2Total,
		})
		entry := computeToolCallsDiffEntry(name, tc1, tc2, inRun1, inRun2)
		computeToolCallsDiffAppend(diff, entry)
		diff.AllTools = append(diff.AllTools, entry)
	}

	diff.BashDiff = computeBashCommandsDiff(bashRun1, bashRun2)
	diff.Summary = ToolCallsDiffSummary{
		NewToolCount:     len(diff.NewTools),
		RemovedToolCount: len(diff.RemovedTools),
		ChangedToolCount: len(diff.ChangedTools),
		Run1TotalCalls:   run1Total,
		Run2TotalCalls:   run2Total,
	}

	auditDiffLog.Printf("Tool calls diff: new=%d, removed=%d, changed=%d, run1_total=%d, run2_total=%d",
		len(diff.NewTools), len(diff.RemovedTools), len(diff.ChangedTools), run1Total, run2Total)
	return diff
}

func computeToolCallsDiffMap(metrics *LogMetrics) map[string]ToolCallInfo {
	tools := make(map[string]ToolCallInfo)
	if metrics == nil {
		return tools
	}
	for _, tc := range metrics.ToolCalls {
		computeToolCallsDiffAggregate(tools, tc)
	}
	return tools
}

func computeToolCallsDiffAggregate(tools map[string]ToolCallInfo, tc ToolCallInfo) {
	// Merges a tool call entry, summing call counts and taking max size fields.
	if existing, ok := tools[tc.Name]; ok {
		existing.CallCount += tc.CallCount
		if tc.MaxInputSize > existing.MaxInputSize {
			existing.MaxInputSize = tc.MaxInputSize
		}
		if tc.MaxOutputSize > existing.MaxOutputSize {
			existing.MaxOutputSize = tc.MaxOutputSize
		}
		if tc.MaxDuration > existing.MaxDuration {
			existing.MaxDuration = tc.MaxDuration
		}
		tools[tc.Name] = existing
		return
	}
	tools[tc.Name] = tc
}

func computeToolCallsDiffSortedNames(run1Tools, run2Tools map[string]ToolCallInfo) []string {
	allNames := make(map[string]struct{})
	for k := range run1Tools {
		allNames[k] = struct{}{}
	}
	for k := range run2Tools {
		allNames[k] = struct{}{}
	}
	return sliceutil.SortedKeys(allNames)
}

type computeToolCallsDiffTrackBashParams struct {
	Name      string
	TC1       ToolCallInfo
	TC2       ToolCallInfo
	InRun1    bool
	InRun2    bool
	BashRun1  map[string]ToolCallInfo
	BashRun2  map[string]ToolCallInfo
	Run1Total int
	Run2Total int
}

func computeToolCallsDiffTrackBash(p computeToolCallsDiffTrackBashParams) (int, int) {
	run1Total := p.Run1Total
	run2Total := p.Run2Total
	if p.InRun1 {
		run1Total += p.TC1.CallCount
		if isBashTool(p.Name) {
			p.BashRun1[p.Name] = p.TC1
		}
	}
	if p.InRun2 {
		run2Total += p.TC2.CallCount
		if isBashTool(p.Name) {
			p.BashRun2[p.Name] = p.TC2
		}
	}
	return run1Total, run2Total
}

func computeToolCallsDiffEntry(name string, tc1, tc2 ToolCallInfo, inRun1, inRun2 bool) ToolCallDiffEntry {
	switch {
	case !inRun1 && inRun2:
		return ToolCallDiffEntry{Name: name, Status: "new", Run2CallCount: tc2.CallCount, Run2MaxInputSize: tc2.MaxInputSize, Run2MaxOutputSize: tc2.MaxOutputSize}
	case inRun1 && !inRun2:
		return ToolCallDiffEntry{Name: name, Status: "removed", Run1CallCount: tc1.CallCount, Run1MaxInputSize: tc1.MaxInputSize, Run1MaxOutputSize: tc1.MaxOutputSize}
	case tc1.CallCount != tc2.CallCount:
		return ToolCallDiffEntry{Name: name, Status: "changed", Run1CallCount: tc1.CallCount, Run2CallCount: tc2.CallCount, CallCountChange: formatCountChange(tc1.CallCount, tc2.CallCount), Run1MaxInputSize: tc1.MaxInputSize, Run2MaxInputSize: tc2.MaxInputSize, Run1MaxOutputSize: tc1.MaxOutputSize, Run2MaxOutputSize: tc2.MaxOutputSize}
	default:
		return ToolCallDiffEntry{Name: name, Status: "unchanged", Run1CallCount: tc1.CallCount, Run2CallCount: tc2.CallCount, Run1MaxInputSize: tc1.MaxInputSize, Run2MaxInputSize: tc2.MaxInputSize, Run1MaxOutputSize: tc1.MaxOutputSize, Run2MaxOutputSize: tc2.MaxOutputSize}
	}
}

func computeToolCallsDiffAppend(diff *ToolCallsDiff, entry ToolCallDiffEntry) {
	switch entry.Status {
	case "new":
		diff.NewTools = append(diff.NewTools, entry)
	case "removed":
		diff.RemovedTools = append(diff.RemovedTools, entry)
	case "changed":
		diff.ChangedTools = append(diff.ChangedTools, entry)
	}
}

// computeBashCommandsDiff builds bash-specific analysis from pre-filtered bash tool call maps.
// The maps should contain only bash-related entries (generic "bash"/"Bash" and per-command "bash_*").
// Returns nil when no bash tool calls are present in either map.
func computeBashCommandsDiff(run1Tools, run2Tools map[string]ToolCallInfo) *BashCommandsDiff {
	allNames := make(map[string]struct{})
	for k := range run1Tools {
		allNames[k] = struct{}{}
	}
	for k := range run2Tools {
		allNames[k] = struct{}{}
	}

	if len(allNames) == 0 {
		return nil
	}

	sortedNames := sliceutil.SortedKeys(allNames)

	bashDiff := &BashCommandsDiff{}
	for _, name := range sortedNames {
		tc1 := run1Tools[name]
		tc2 := run2Tools[name]
		bashDiff.Run1TotalCalls += tc1.CallCount
		bashDiff.Run2TotalCalls += tc2.CallCount

		var status string
		switch {
		case tc1.CallCount == 0 && tc2.CallCount > 0:
			status = "new"
		case tc1.CallCount > 0 && tc2.CallCount == 0:
			status = "removed"
		case tc1.CallCount != tc2.CallCount:
			status = "changed"
		default:
			status = "unchanged"
		}

		cmd := ToolCallDiffEntry{
			Name:              name,
			Status:            status,
			Run1CallCount:     tc1.CallCount,
			Run2CallCount:     tc2.CallCount,
			Run1MaxInputSize:  tc1.MaxInputSize,
			Run2MaxInputSize:  tc2.MaxInputSize,
			Run1MaxOutputSize: tc1.MaxOutputSize,
			Run2MaxOutputSize: tc2.MaxOutputSize,
		}
		if tc1.CallCount != tc2.CallCount {
			cmd.CallCountChange = formatCountChange(tc1.CallCount, tc2.CallCount)
		}
		bashDiff.Commands = append(bashDiff.Commands, cmd)
	}

	if bashDiff.Run1TotalCalls > 0 || bashDiff.Run2TotalCalls > 0 {
		bashDiff.TotalCallsChange = formatCountChange(bashDiff.Run1TotalCalls, bashDiff.Run2TotalCalls)
	}

	return bashDiff
}

// computeGitHubRateLimitDiff computes the diff of GitHub API quota consumption between two
// runs using the GitHubRateLimitUsage data from RunSummary.GitHubRateLimitUsage.
// Returns nil when both summaries are nil.
func computeGitHubRateLimitDiff(rl1, rl2 *GitHubRateLimitUsage) *GitHubRateLimitDiff {
	if rl1 == nil && rl2 == nil {
		return nil
	}

	var run1Calls, run2Calls int
	var run1CoreConsumed, run2CoreConsumed int
	var run1CoreRemaining, run2CoreRemaining int
	var run1CoreLimit, run2CoreLimit int

	if rl1 != nil {
		run1Calls = rl1.TotalRequestsMade
		run1CoreConsumed = rl1.CoreConsumed
		run1CoreRemaining = rl1.CoreRemaining
		run1CoreLimit = rl1.CoreLimit
	}
	if rl2 != nil {
		run2Calls = rl2.TotalRequestsMade
		run2CoreConsumed = rl2.CoreConsumed
		run2CoreRemaining = rl2.CoreRemaining
		run2CoreLimit = rl2.CoreLimit
	}

	diff := &GitHubRateLimitDiff{
		Run1TotalAPICalls: run1Calls,
		Run2TotalAPICalls: run2Calls,
		Run1CoreConsumed:  run1CoreConsumed,
		Run2CoreConsumed:  run2CoreConsumed,
		Run1CoreRemaining: run1CoreRemaining,
		Run2CoreRemaining: run2CoreRemaining,
		Run1CoreLimit:     run1CoreLimit,
		Run2CoreLimit:     run2CoreLimit,
	}

	if run1Calls > 0 || run2Calls > 0 {
		diff.APICallsChange = formatVolumeChange(run1Calls, run2Calls)
	}
	if run1CoreConsumed > 0 || run2CoreConsumed > 0 {
		diff.CoreConsumedChange = formatVolumeChange(run1CoreConsumed, run2CoreConsumed)
	}

	return diff
}

// computeTokenUsageDiff computes a detailed diff of token usage between two runs using
// the firewall proxy token-usage.jsonl data (TokenUsageSummary). Returns nil when both
// summaries are nil.
func computeTokenUsageDiff(tu1, tu2 *TokenUsageSummary) *TokenUsageDiff {
	if tu1 == nil && tu2 == nil {
		return nil
	}

	run1 := computeTokenUsageDiffValues(tu1)
	run2 := computeTokenUsageDiffValues(tu2)

	diff := &TokenUsageDiff{
		Run1InputTokens: run1.input, Run2InputTokens: run2.input,
		Run1OutputTokens: run1.output, Run2OutputTokens: run2.output,
		Run1CacheReadTokens: run1.cacheRead, Run2CacheReadTokens: run2.cacheRead,
		Run1CacheWriteTokens: run1.cacheWrite, Run2CacheWriteTokens: run2.cacheWrite,
		Run1AIC: run1.aic, Run2AIC: run2.aic,
		Run1TotalRequests: run1.requests, Run2TotalRequests: run2.requests,
		Run1CacheEfficiency: run1.cacheEff, Run2CacheEfficiency: run2.cacheEff,
	}
	computeTokenUsageDiffChanges(diff, run1, run2)

	return diff
}

type computeTokenUsageDiffSummaryValues struct {
	input, output, cacheRead, cacheWrite int
	aic, cacheEff                        float64
	requests                             int
}

func computeTokenUsageDiffValues(tu *TokenUsageSummary) computeTokenUsageDiffSummaryValues {
	if tu == nil {
		return computeTokenUsageDiffSummaryValues{}
	}
	return computeTokenUsageDiffSummaryValues{
		input: tu.TotalInputTokens, output: tu.TotalOutputTokens,
		cacheRead: tu.TotalCacheReadTokens, cacheWrite: tu.TotalCacheWriteTokens,
		aic: tu.TotalAIC, requests: tu.TotalRequests, cacheEff: tu.CacheEfficiency,
	}
}

func computeTokenUsageDiffChanges(diff *TokenUsageDiff, run1, run2 computeTokenUsageDiffSummaryValues) {
	if run1.input > 0 || run2.input > 0 {
		diff.InputTokensChange = formatVolumeChange(run1.input, run2.input)
	}
	if run1.output > 0 || run2.output > 0 {
		diff.OutputTokensChange = formatVolumeChange(run1.output, run2.output)
	}
	if run1.cacheRead > 0 || run2.cacheRead > 0 {
		diff.CacheReadTokensChange = formatVolumeChange(run1.cacheRead, run2.cacheRead)
	}
	if run1.cacheWrite > 0 || run2.cacheWrite > 0 {
		diff.CacheWriteTokensChange = formatVolumeChange(run1.cacheWrite, run2.cacheWrite)
	}
	if run1.aic > 0 || run2.aic > 0 {
		diff.AICChange = formatFloatDelta(run1.aic, run2.aic)
	}
	if run1.requests > 0 || run2.requests > 0 {
		diff.RequestsDelta = formatCountChange(run1.requests, run2.requests)
	}
	if run1.cacheEff > 0 || run2.cacheEff > 0 {
		diff.CacheEfficiencyChange = formatPercentagePointChange(run1.cacheEff, run2.cacheEff)
	}
}

// loadRunSummaryForDiff loads or builds a RunSummary for a given run for use in diffing.
// It first tries to load from a cached RunSummary (which includes MCP tool usage and run
// metrics); otherwise it downloads artifacts and analyzes firewall logs, returning a partial
// summary with only FirewallAnalysis populated.
// artifactFilter restricts which artifacts are downloaded; nil means download all.
func loadRunSummaryForDiff(ctx context.Context, runID int64, outputDir string, owner, repo, hostname string, verbose bool, artifactFilter []string) (*RunSummary, error) {
	auditDiffLog.Printf("Loading run summary for diff: run_id=%d, owner=%q, repo=%q, artifact_filter=%v", runID, owner, repo, artifactFilter)
	runOutputDir := filepath.Join(outputDir, fmt.Sprintf("run-%d", runID))
	if absDir, err := filepath.Abs(runOutputDir); err == nil {
		runOutputDir = absDir
	}

	// Try cached summary first (full data including MCP tool usage, token usage, etc.)
	if summary, ok := loadRunSummary(runOutputDir, verbose); ok {
		auditDiffLog.Printf("Using cached run summary for run %d", runID)
		return summary, nil
	}

	// Download artifacts if needed
	if err := downloadRunArtifacts(ctx, downloadArtifactsOptions{runID: runID, outputDir: runOutputDir, verbose: verbose, owner: owner, repo: repo, hostname: hostname, artifactFilter: artifactFilter}); err != nil {
		if !errors.Is(err, ErrNoArtifacts) {
			auditDiffLog.Printf("Failed to download artifacts for run %d: %v", runID, err)
			return nil, fmt.Errorf("failed to download artifacts for run %d: %w", runID, err)
		}
		auditDiffLog.Printf("No artifacts found for run %d, proceeding with partial summary", runID)
	}

	// Analyze firewall logs only when the agent artifact was included in the filter.
	// Firewall audit logs are now included in the unified agent artifact.
	// Skip silently when the artifact was intentionally excluded to avoid spurious warnings.
	var analysis *FirewallAnalysis
	if artifactMatchesFilter(constants.AgentArtifactName, artifactFilter) {
		var err error
		analysis, err = analyzeFirewallLogs(runOutputDir, verbose)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze firewall logs for run %d: %w", runID, err)
		}
	}

	// Analyze GitHub API rate limit consumption
	rateLimitUsage, err := analyzeGitHubRateLimits(runOutputDir, verbose)
	if err != nil {
		auditDiffLog.Printf("Failed to analyze GitHub rate limits for run %d: %v", runID, err)
		// Non-fatal: proceed without rate limit data
	}

	return &RunSummary{
		RunID:                runID,
		FirewallAnalysis:     analysis,
		GitHubRateLimitUsage: rateLimitUsage,
	}, nil
}
