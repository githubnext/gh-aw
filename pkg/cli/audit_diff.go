package cli

import (
	"context"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
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

	// Handle nil cases
	run1Stats := make(map[string]DomainRequestStats)
	run2Stats := make(map[string]DomainRequestStats)

	if run1 != nil {
		run1Stats = run1.RequestsByDomain
	}
	if run2 != nil {
		run2Stats = run2.RequestsByDomain
	}

	// If both are nil/empty, return empty diff
	if len(run1Stats) == 0 && len(run2Stats) == 0 {
		return diff
	}

	// Collect all domains
	allDomains := make(map[string]bool)
	for domain := range run1Stats {
		allDomains[domain] = true
	}
	for domain := range run2Stats {
		allDomains[domain] = true
	}

	// Sorted domain list for deterministic output
	sortedDomains := make([]string, 0, len(allDomains))
	for domain := range allDomains {
		sortedDomains = append(sortedDomains, domain)
	}
	sort.Strings(sortedDomains)

	anomalyCount := 0

	for _, domain := range sortedDomains {
		stats1, inRun1 := run1Stats[domain]
		stats2, inRun2 := run2Stats[domain]

		if !inRun1 && inRun2 {
			// New domain in run 2
			entry := DomainDiffEntry{
				Domain:      domain,
				Status:      "new",
				Run2Allowed: stats2.Allowed,
				Run2Blocked: stats2.Blocked,
				Run2Status:  classifyFirewallDomainStatus(stats2),
			}
			// Anomaly: new denied domain
			if stats2.Blocked > 0 {
				entry.IsAnomaly = true
				entry.AnomalyNote = "new denied domain"
				anomalyCount++
			}
			diff.NewDomains = append(diff.NewDomains, entry)
		} else if inRun1 && !inRun2 {
			// Removed domain
			entry := DomainDiffEntry{
				Domain:      domain,
				Status:      "removed",
				Run1Allowed: stats1.Allowed,
				Run1Blocked: stats1.Blocked,
				Run1Status:  classifyFirewallDomainStatus(stats1),
			}
			diff.RemovedDomains = append(diff.RemovedDomains, entry)
		} else {
			// Domain exists in both runs - check for changes
			status1 := classifyFirewallDomainStatus(stats1)
			status2 := classifyFirewallDomainStatus(stats2)

			if status1 != status2 {
				// Status changed
				entry := DomainDiffEntry{
					Domain:      domain,
					Status:      "status_changed",
					Run1Allowed: stats1.Allowed,
					Run1Blocked: stats1.Blocked,
					Run2Allowed: stats2.Allowed,
					Run2Blocked: stats2.Blocked,
					Run1Status:  status1,
					Run2Status:  status2,
				}
				// Anomaly: previously denied, now allowed
				if status1 == "denied" && status2 == "allowed" {
					entry.IsAnomaly = true
					entry.AnomalyNote = "previously denied, now allowed"
					anomalyCount++
				}
				// Anomaly: previously allowed, now denied
				if status1 == "allowed" && status2 == "denied" {
					entry.IsAnomaly = true
					entry.AnomalyNote = "previously allowed, now denied"
					anomalyCount++
				}
				diff.StatusChanges = append(diff.StatusChanges, entry)
			} else {
				// Check for significant volume changes (>100% threshold)
				total1 := stats1.Allowed + stats1.Blocked
				total2 := stats2.Allowed + stats2.Blocked

				if total1 > 0 {
					pctChange := (float64(total2-total1) / float64(total1)) * 100
					if math.Abs(pctChange) > volumeChangeThresholdPercent {
						entry := DomainDiffEntry{
							Domain:       domain,
							Status:       "volume_changed",
							Run1Allowed:  stats1.Allowed,
							Run1Blocked:  stats1.Blocked,
							Run2Allowed:  stats2.Allowed,
							Run2Blocked:  stats2.Blocked,
							Run1Status:   status1,
							Run2Status:   status2,
							VolumeChange: formatVolumeChange(total1, total2),
						}
						diff.VolumeChanges = append(diff.VolumeChanges, entry)
					}
				}
			}
		}
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

// formatVolumeChange formats the volume change as a human-readable string
func formatVolumeChange(total1, total2 int) string {
	if total1 == 0 {
		return "+∞"
	}
	pctChange := (float64(total2-total1) / float64(total1)) * 100
	if pctChange >= 0 {
		return "+" + formatPercent(pctChange)
	}
	return formatPercent(pctChange)
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
	Run1EffectiveTokens    int     `json:"run1_effective_tokens"`
	Run2EffectiveTokens    int     `json:"run2_effective_tokens"`
	EffectiveTokensChange  string  `json:"effective_tokens_change,omitempty"`
	Run1TotalRequests      int     `json:"run1_total_requests"`
	Run2TotalRequests      int     `json:"run2_total_requests"`
	RequestsDelta          string  `json:"requests_delta,omitempty"` // Absolute request-count delta, e.g. "+4"
	Run1CacheEfficiency    float64 `json:"run1_cache_efficiency"`
	Run2CacheEfficiency    float64 `json:"run2_cache_efficiency"`
	CacheEfficiencyChange  string  `json:"cache_efficiency_change,omitempty"` // Percentage-point delta, e.g. "+1.5pp"
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
	TokenUsageDetails      *TokenUsageDiff      `json:"token_usage_details,omitempty"`       // Detailed breakdown from firewall proxy
	GitHubRateLimitDetails *GitHubRateLimitDiff `json:"github_rate_limit_details,omitempty"` // GitHub API quota consumption diff
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

	allKeys := make(map[string]bool)
	for k := range run1Tools {
		allKeys[k] = true
	}
	for k := range run2Tools {
		allKeys[k] = true
	}

	sortedKeys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	diff := &MCPToolsDiff{}
	anomalyCount := 0

	for _, key := range sortedKeys {
		s1, inRun1 := run1Tools[key]
		s2, inRun2 := run2Tools[key]

		if !inRun1 && inRun2 {
			entry := MCPToolDiffEntry{
				ServerName:     s2.ServerName,
				ToolName:       s2.ToolName,
				Status:         "new",
				Run2CallCount:  s2.CallCount,
				Run2ErrorCount: s2.ErrorCount,
			}
			if s2.ErrorCount > 0 {
				entry.IsAnomaly = true
				entry.AnomalyNote = "new tool with errors"
				anomalyCount++
			}
			diff.NewTools = append(diff.NewTools, entry)
		} else if inRun1 && !inRun2 {
			diff.RemovedTools = append(diff.RemovedTools, MCPToolDiffEntry{
				ServerName:     s1.ServerName,
				ToolName:       s1.ToolName,
				Status:         "removed",
				Run1CallCount:  s1.CallCount,
				Run1ErrorCount: s1.ErrorCount,
			})
		} else if s1.CallCount != s2.CallCount || s1.ErrorCount != s2.ErrorCount {
			entry := MCPToolDiffEntry{
				ServerName:      s1.ServerName,
				ToolName:        s1.ToolName,
				Status:          "changed",
				Run1CallCount:   s1.CallCount,
				Run2CallCount:   s2.CallCount,
				Run1ErrorCount:  s1.ErrorCount,
				Run2ErrorCount:  s2.ErrorCount,
				CallCountChange: formatCountChange(s1.CallCount, s2.CallCount),
			}
			if s2.ErrorCount > s1.ErrorCount {
				entry.IsAnomaly = true
				entry.AnomalyNote = "error count increased"
				anomalyCount++
			}
			diff.ChangedTools = append(diff.ChangedTools, entry)
		}
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

// computeRunMetricsDiff computes the diff of run-level metrics between two runs.
// Returns nil if no meaningful metrics data is available.
func computeRunMetricsDiff(summary1, summary2 *RunSummary) *RunMetricsDiff {
	var run1Tokens, run2Tokens int
	var run1Duration, run2Duration time.Duration
	var run1Turns, run2Turns int
	var tu1, tu2 *TokenUsageSummary
	var rl1, rl2 *GitHubRateLimitUsage

	if summary1 != nil {
		run1Tokens = summary1.Run.TokenUsage
		run1Duration = summary1.Run.Duration
		run1Turns = summary1.Run.Turns
		tu1 = summary1.TokenUsage
		rl1 = summary1.GitHubRateLimitUsage
	}
	if summary2 != nil {
		run2Tokens = summary2.Run.TokenUsage
		run2Duration = summary2.Run.Duration
		run2Turns = summary2.Run.Turns
		tu2 = summary2.TokenUsage
		rl2 = summary2.GitHubRateLimitUsage
	}

	// Skip if there is no meaningful data
	hasTokenDetails := tu1 != nil || tu2 != nil
	hasRateLimitDetails := rl1 != nil || rl2 != nil
	if run1Tokens == 0 && run2Tokens == 0 && run1Duration == 0 && run2Duration == 0 && run1Turns == 0 && run2Turns == 0 && !hasTokenDetails && !hasRateLimitDetails {
		return nil
	}

	diff := &RunMetricsDiff{
		Run1TokenUsage: run1Tokens,
		Run2TokenUsage: run2Tokens,
		Run1Turns:      run1Turns,
		Run2Turns:      run2Turns,
		TurnsChange:    run2Turns - run1Turns,
	}

	if run1Tokens > 0 || run2Tokens > 0 {
		diff.TokenUsageChange = formatVolumeChange(run1Tokens, run2Tokens)
	}

	if run1Duration > 0 {
		diff.Run1Duration = run1Duration.Round(time.Second).String()
	}
	if run2Duration > 0 {
		diff.Run2Duration = run2Duration.Round(time.Second).String()
	}
	if run1Duration > 0 && run2Duration > 0 {
		delta := run2Duration - run1Duration
		if delta >= 0 {
			diff.DurationChange = "+" + delta.Round(time.Second).String()
		} else {
			diff.DurationChange = delta.Round(time.Second).String()
		}
	}

	diff.TokenUsageDetails = computeTokenUsageDiff(tu1, tu2)
	diff.GitHubRateLimitDetails = computeGitHubRateLimitDiff(rl1, rl2)

	return diff
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

	var (
		run1Input, run2Input           int
		run1Output, run2Output         int
		run1CacheRead, run2CacheRead   int
		run1CacheWrite, run2CacheWrite int
		run1Effective, run2Effective   int
		run1Requests, run2Requests     int
		run1CacheEff, run2CacheEff     float64
	)

	if tu1 != nil {
		run1Input = tu1.TotalInputTokens
		run1Output = tu1.TotalOutputTokens
		run1CacheRead = tu1.TotalCacheReadTokens
		run1CacheWrite = tu1.TotalCacheWriteTokens
		run1Effective = tu1.TotalEffectiveTokens
		run1Requests = tu1.TotalRequests
		run1CacheEff = tu1.CacheEfficiency
	}
	if tu2 != nil {
		run2Input = tu2.TotalInputTokens
		run2Output = tu2.TotalOutputTokens
		run2CacheRead = tu2.TotalCacheReadTokens
		run2CacheWrite = tu2.TotalCacheWriteTokens
		run2Effective = tu2.TotalEffectiveTokens
		run2Requests = tu2.TotalRequests
		run2CacheEff = tu2.CacheEfficiency
	}

	diff := &TokenUsageDiff{
		Run1InputTokens:      run1Input,
		Run2InputTokens:      run2Input,
		Run1OutputTokens:     run1Output,
		Run2OutputTokens:     run2Output,
		Run1CacheReadTokens:  run1CacheRead,
		Run2CacheReadTokens:  run2CacheRead,
		Run1CacheWriteTokens: run1CacheWrite,
		Run2CacheWriteTokens: run2CacheWrite,
		Run1EffectiveTokens:  run1Effective,
		Run2EffectiveTokens:  run2Effective,
		Run1TotalRequests:    run1Requests,
		Run2TotalRequests:    run2Requests,
		Run1CacheEfficiency:  run1CacheEff,
		Run2CacheEfficiency:  run2CacheEff,
	}

	if run1Input > 0 || run2Input > 0 {
		diff.InputTokensChange = formatVolumeChange(run1Input, run2Input)
	}
	if run1Output > 0 || run2Output > 0 {
		diff.OutputTokensChange = formatVolumeChange(run1Output, run2Output)
	}
	if run1CacheRead > 0 || run2CacheRead > 0 {
		diff.CacheReadTokensChange = formatVolumeChange(run1CacheRead, run2CacheRead)
	}
	if run1CacheWrite > 0 || run2CacheWrite > 0 {
		diff.CacheWriteTokensChange = formatVolumeChange(run1CacheWrite, run2CacheWrite)
	}
	if run1Effective > 0 || run2Effective > 0 {
		diff.EffectiveTokensChange = formatVolumeChange(run1Effective, run2Effective)
	}
	if run1Requests > 0 || run2Requests > 0 {
		diff.RequestsDelta = formatCountChange(run1Requests, run2Requests)
	}
	if run1CacheEff > 0 || run2CacheEff > 0 {
		diff.CacheEfficiencyChange = formatPercentagePointChange(run1CacheEff, run2CacheEff)
	}

	return diff
}

// formatPercentagePointChange formats the change between two ratio values (0.0-1.0) as a
// percentage-point delta (e.g. "+1.5pp", "-2.3pp")
func formatPercentagePointChange(ratio1, ratio2 float64) string {
	delta := (ratio2 - ratio1) * 100
	if delta >= 0 {
		return fmt.Sprintf("+%.1fpp", delta)
	}
	return fmt.Sprintf("%.1fpp", delta)
}

// formatCountChange formats the absolute change in a count value (e.g. "+3", "-1")
func formatCountChange(count1, count2 int) string {
	delta := count2 - count1
	if delta >= 0 {
		return fmt.Sprintf("+%d", delta)
	}
	return strconv.Itoa(delta)
}

// loadRunSummaryForDiff loads or builds a RunSummary for a given run for use in diffing.
// It first tries to load from a cached RunSummary (which includes MCP tool usage and run
// metrics); otherwise it downloads artifacts and analyzes firewall logs, returning a partial
// summary with only FirewallAnalysis populated.
// artifactFilter restricts which artifacts are downloaded; nil means download all.
func loadRunSummaryForDiff(ctx context.Context, runID int64, outputDir string, owner, repo, hostname string, verbose bool, artifactFilter []string) (*RunSummary, error) {
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
	if err := downloadRunArtifacts(ctx, runID, runOutputDir, verbose, owner, repo, hostname, artifactFilter); err != nil {
		if !errors.Is(err, ErrNoArtifacts) {
			return nil, fmt.Errorf("failed to download artifacts for run %d: %w", runID, err)
		}
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
