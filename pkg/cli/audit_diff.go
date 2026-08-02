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

// DiffEntryBase holds common anomaly-flagging fields shared by all diff entry types.
type DiffEntryBase struct {
	Status      string `json:"status"`
	IsAnomaly   bool   `json:"is_anomaly,omitempty"`   // Flagged as anomalous
	AnomalyNote string `json:"anomaly_note,omitempty"` // Human-readable anomaly explanation
}

// DomainDiffEntry represents the diff for a single domain between two runs
type DomainDiffEntry struct {
	Domain string `json:"domain"`
	DiffEntryBase
	Run1Allowed  int    `json:"run1_allowed"`            // Allowed requests in run 1
	Run1Blocked  int    `json:"run1_blocked"`            // Blocked requests in run 1
	Run2Allowed  int    `json:"run2_allowed"`            // Allowed requests in run 2
	Run2Blocked  int    `json:"run2_blocked"`            // Blocked requests in run 2
	Run1Status   string `json:"run1_status,omitempty"`   // "allowed", "denied", or "" for new domains
	Run2Status   string `json:"run2_status,omitempty"`   // "allowed", "denied", or "" for removed domains
	VolumeChange string `json:"volume_change,omitempty"` // e.g. "+287%" or "-50%"
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
	ctx := &firewallDiffContext{diff: &FirewallDiff{Run1ID: run1ID, Run2ID: run2ID}}
	run1Stats := firewallRequestStats(run1)
	run2Stats := firewallRequestStats(run2)
	if len(run1Stats) == 0 && len(run2Stats) == 0 {
		return ctx.diff
	}
	for _, domain := range collectFirewallDomains(run1Stats, run2Stats) {
		stats1, inRun1 := run1Stats[domain]
		stats2, inRun2 := run2Stats[domain]
		processFirewallDomain(ctx, domain, stats1, inRun1, stats2, inRun2)
	}
	return finalizeFirewallDiff(ctx)
}

type firewallDiffContext struct {
	diff         *FirewallDiff
	anomalyCount int
}

func firewallRequestStats(analysis *FirewallAnalysis) map[string]DomainRequestStats {
	if analysis == nil {
		return map[string]DomainRequestStats{}
	}
	return analysis.RequestsByDomain
}

func collectFirewallDomains(run1Stats, run2Stats map[string]DomainRequestStats) []string {
	allDomains := make(map[string]struct{})
	for domain := range run1Stats {
		allDomains[domain] = struct{}{}
	}
	for domain := range run2Stats {
		allDomains[domain] = struct{}{}
	}
	return sliceutil.SortedKeys(allDomains)
}

func processFirewallDomain(ctx *firewallDiffContext, domain string, stats1 DomainRequestStats, inRun1 bool, stats2 DomainRequestStats, inRun2 bool) {
	switch {
	case !inRun1 && inRun2:
		ctx.diff.NewDomains = append(ctx.diff.NewDomains, buildNewFirewallDomainEntry(ctx, domain, stats2))
	case inRun1 && !inRun2:
		ctx.diff.RemovedDomains = append(ctx.diff.RemovedDomains, buildRemovedFirewallDomainEntry(ctx, domain, stats1))
	default:
		processSharedFirewallDomain(ctx, domain, stats1, stats2)
	}
}

func buildNewFirewallDomainEntry(ctx *firewallDiffContext, domain string, stats DomainRequestStats) DomainDiffEntry {
	entry := DomainDiffEntry{
		Domain:        domain,
		DiffEntryBase: DiffEntryBase{Status: "new"},
		Run2Allowed:   stats.Allowed,
		Run2Blocked:   stats.Blocked,
		Run2Status:    classifyFirewallDomainStatus(stats),
	}
	if stats.Blocked > 0 {
		entry.IsAnomaly = true
		entry.AnomalyNote = "new denied domain"
		ctx.anomalyCount++
	}
	return entry
}

func buildRemovedFirewallDomainEntry(ctx *firewallDiffContext, domain string, stats DomainRequestStats) DomainDiffEntry {
	entry := DomainDiffEntry{
		Domain:        domain,
		DiffEntryBase: DiffEntryBase{Status: "removed"},
		Run1Allowed:   stats.Allowed,
		Run1Blocked:   stats.Blocked,
		Run1Status:    classifyFirewallDomainStatus(stats),
	}
	if stats.Blocked > 0 {
		entry.IsAnomaly = true
		entry.AnomalyNote = "denied in base run — absent from comparison run"
		ctx.anomalyCount++
	}
	return entry
}

func processSharedFirewallDomain(ctx *firewallDiffContext, domain string, stats1, stats2 DomainRequestStats) {
	status1 := classifyFirewallDomainStatus(stats1)
	status2 := classifyFirewallDomainStatus(stats2)
	if maybeAppendFirewallStatusChange(ctx, domain, stats1, stats2, status1, status2) {
		return
	}
	maybeAppendFirewallVolumeChange(ctx, domain, stats1, stats2, status1, status2)
}

func maybeAppendFirewallStatusChange(ctx *firewallDiffContext, domain string, stats1, stats2 DomainRequestStats, status1, status2 string) bool {
	if status1 == status2 {
		return false
	}
	entry := DomainDiffEntry{
		Domain:        domain,
		DiffEntryBase: DiffEntryBase{Status: "status_changed"},
		Run1Allowed:   stats1.Allowed,
		Run1Blocked:   stats1.Blocked,
		Run2Allowed:   stats2.Allowed,
		Run2Blocked:   stats2.Blocked,
		Run1Status:    status1,
		Run2Status:    status2,
	}
	markFirewallStatusChangeAnomaly(ctx, &entry, status1, status2)
	ctx.diff.StatusChanges = append(ctx.diff.StatusChanges, entry)
	return true
}

func markFirewallStatusChangeAnomaly(ctx *firewallDiffContext, entry *DomainDiffEntry, status1, status2 string) {
	switch {
	case status1 == "denied" && status2 == "allowed":
		entry.IsAnomaly = true
		entry.AnomalyNote = "previously denied, now allowed"
		ctx.anomalyCount++
	case status1 == "allowed" && status2 == "denied":
		entry.IsAnomaly = true
		entry.AnomalyNote = "previously allowed, now denied"
		ctx.anomalyCount++
	}
}

func maybeAppendFirewallVolumeChange(ctx *firewallDiffContext, domain string, stats1, stats2 DomainRequestStats, status1, status2 string) {
	total1 := stats1.Allowed + stats1.Blocked
	total2 := stats2.Allowed + stats2.Blocked
	if total1 == 0 {
		return
	}
	pctChange := (float64(total2-total1) / float64(total1)) * 100
	if math.Abs(pctChange) <= volumeChangeThresholdPercent {
		return
	}
	ctx.diff.VolumeChanges = append(ctx.diff.VolumeChanges, DomainDiffEntry{
		Domain:        domain,
		DiffEntryBase: DiffEntryBase{Status: "volume_changed"},
		Run1Allowed:   stats1.Allowed,
		Run1Blocked:   stats1.Blocked,
		Run2Allowed:   stats2.Allowed,
		Run2Blocked:   stats2.Blocked,
		Run1Status:    status1,
		Run2Status:    status2,
		VolumeChange:  formatVolumeChange(total1, total2),
	})
}

func finalizeFirewallDiff(ctx *firewallDiffContext) *FirewallDiff {
	ctx.diff.Summary = FirewallDiffSummary{
		NewDomainCount:     len(ctx.diff.NewDomains),
		RemovedDomainCount: len(ctx.diff.RemovedDomains),
		StatusChangeCount:  len(ctx.diff.StatusChanges),
		VolumeChangeCount:  len(ctx.diff.VolumeChanges),
		HasAnomalies:       ctx.anomalyCount > 0,
		AnomalyCount:       ctx.anomalyCount,
	}
	auditDiffLog.Printf("Firewall diff complete: new=%d, removed=%d, status_changes=%d, volume_changes=%d, anomalies=%d",
		len(ctx.diff.NewDomains), len(ctx.diff.RemovedDomains), len(ctx.diff.StatusChanges), len(ctx.diff.VolumeChanges), ctx.anomalyCount)
	return ctx.diff
}

// classifyFirewallDomainStatus returns "allowed", "denied", or "mixed" based on request stats
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
	ServerName string `json:"server_name"`
	ToolName   string `json:"tool_name"`
	DiffEntryBase
	Run1CallCount   int    `json:"run1_call_count,omitempty"` // Call count in run 1
	Run2CallCount   int    `json:"run2_call_count,omitempty"` // Call count in run 2
	Run1ErrorCount  int    `json:"run1_error_count,omitempty"`
	Run2ErrorCount  int    `json:"run2_error_count,omitempty"`
	CallCountChange string `json:"call_count_change,omitempty"` // e.g. "+2", "-3"
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
	Name string `json:"name"`
	DiffEntryBase
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
	run1Tools := mcpToolSummaryMap(run1)
	run2Tools := mcpToolSummaryMap(run2)
	auditDiffLog.Printf("Computing MCP tools diff: run1_tools=%d, run2_tools=%d", len(run1Tools), len(run2Tools))
	ctx := &mcpToolsDiffContext{diff: &MCPToolsDiff{}}
	for _, key := range collectMCPToolKeys(run1Tools, run2Tools) {
		processMCPToolDiffKey(ctx, key, run1Tools, run2Tools)
	}
	ctx.diff.Summary = MCPToolsDiffSummary{
		NewToolCount:     len(ctx.diff.NewTools),
		RemovedToolCount: len(ctx.diff.RemovedTools),
		ChangedToolCount: len(ctx.diff.ChangedTools),
		HasAnomalies:     ctx.anomalyCount > 0,
		AnomalyCount:     ctx.anomalyCount,
	}
	return ctx.diff
}

type mcpToolsDiffContext struct {
	diff         *MCPToolsDiff
	anomalyCount int
}

func mcpToolSummaryMap(run *MCPToolUsageData) map[string]MCPToolSummary {
	tools := make(map[string]MCPToolSummary)
	if run == nil {
		return tools
	}
	for _, summary := range run.Summary {
		tools[mcpToolKey(summary.ServerName, summary.ToolName)] = summary
	}
	return tools
}

func collectMCPToolKeys(run1Tools, run2Tools map[string]MCPToolSummary) []string {
	allKeys := make(map[string]struct{})
	for key := range run1Tools {
		allKeys[key] = struct{}{}
	}
	for key := range run2Tools {
		allKeys[key] = struct{}{}
	}
	return sliceutil.SortedKeys(allKeys)
}

func processMCPToolDiffKey(ctx *mcpToolsDiffContext, key string, run1Tools, run2Tools map[string]MCPToolSummary) {
	s1, inRun1 := run1Tools[key]
	s2, inRun2 := run2Tools[key]
	switch {
	case !inRun1 && inRun2:
		ctx.diff.NewTools = append(ctx.diff.NewTools, buildNewMCPToolDiffEntry(ctx, s2))
	case inRun1 && !inRun2:
		ctx.diff.RemovedTools = append(ctx.diff.RemovedTools, buildRemovedMCPToolDiffEntry(s1))
	case s1.CallCount != s2.CallCount || s1.ErrorCount != s2.ErrorCount:
		ctx.diff.ChangedTools = append(ctx.diff.ChangedTools, buildChangedMCPToolDiffEntry(ctx, s1, s2))
	}
}

func buildNewMCPToolDiffEntry(ctx *mcpToolsDiffContext, summary MCPToolSummary) MCPToolDiffEntry {
	entry := MCPToolDiffEntry{
		ServerName:     summary.ServerName,
		ToolName:       summary.ToolName,
		DiffEntryBase:  DiffEntryBase{Status: "new"},
		Run2CallCount:  summary.CallCount,
		Run2ErrorCount: summary.ErrorCount,
	}
	if summary.ErrorCount > 0 {
		entry.IsAnomaly = true
		entry.AnomalyNote = "new tool with errors"
		ctx.anomalyCount++
	}
	return entry
}

func buildRemovedMCPToolDiffEntry(summary MCPToolSummary) MCPToolDiffEntry {
	return MCPToolDiffEntry{
		ServerName:     summary.ServerName,
		ToolName:       summary.ToolName,
		DiffEntryBase:  DiffEntryBase{Status: "removed"},
		Run1CallCount:  summary.CallCount,
		Run1ErrorCount: summary.ErrorCount,
	}
}

func buildChangedMCPToolDiffEntry(ctx *mcpToolsDiffContext, before, after MCPToolSummary) MCPToolDiffEntry {
	entry := MCPToolDiffEntry{
		ServerName:      before.ServerName,
		ToolName:        before.ToolName,
		DiffEntryBase:   DiffEntryBase{Status: "changed"},
		Run1CallCount:   before.CallCount,
		Run2CallCount:   after.CallCount,
		Run1ErrorCount:  before.ErrorCount,
		Run2ErrorCount:  after.ErrorCount,
		CallCountChange: formatCountChange(before.CallCount, after.CallCount),
	}
	if after.ErrorCount > before.ErrorCount {
		entry.IsAnomaly = true
		entry.AnomalyNote = "error count increased"
		ctx.anomalyCount++
	}
	return entry
}

// computeRunMetricsDiff computes the diff of run-level metrics between two runs.
// computeRunMetricsDiff computes the diff of run-level metrics between two runs.
// Returns nil if no meaningful metrics data is available.
func computeRunMetricsDiff(summary1, summary2 *RunSummary) *RunMetricsDiff {
	inputs := extractRunMetricsInputs(summary1, summary2)
	if !hasRunMetricsData(inputs) {
		return nil
	}
	diff := newRunMetricsDiff(inputs)
	applyRunMetricsTokenUsage(diff)
	applyRunMetricsDuration(diff, inputs)
	applyRunMetricsTokensPerTurn(diff)
	diff.TokenUsageDetails = computeTokenUsageDiff(inputs.tokenUsage1, inputs.tokenUsage2)
	diff.GitHubRateLimitDetails = computeGitHubRateLimitDiff(inputs.rateLimit1, inputs.rateLimit2)
	diff.ToolCallsDiff = computeToolCallsDiff(inputs.metrics1, inputs.metrics2)
	auditDiffLog.Printf("Run metrics diff: tokens %d->%d, turns %d->%d, has_token_details=%t, has_rate_limit_details=%t",
		inputs.run1Tokens, inputs.run2Tokens, inputs.run1Turns, inputs.run2Turns, inputs.tokenUsage1 != nil || inputs.tokenUsage2 != nil, inputs.rateLimit1 != nil || inputs.rateLimit2 != nil)
	return diff
}

type runMetricsInputs struct {
	run1Tokens   int
	run2Tokens   int
	run1Turns    int
	run2Turns    int
	run1Duration time.Duration
	run2Duration time.Duration
	tokenUsage1  *TokenUsageSummary
	tokenUsage2  *TokenUsageSummary
	rateLimit1   *GitHubRateLimitUsage
	rateLimit2   *GitHubRateLimitUsage
	metrics1     *LogMetrics
	metrics2     *LogMetrics
}

func extractRunMetricsInputs(summary1, summary2 *RunSummary) runMetricsInputs {
	inputs := runMetricsInputs{}
	if summary1 != nil {
		inputs.run1Tokens = summary1.Run.TokenUsage
		inputs.run1Duration = summary1.Run.Duration
		inputs.run1Turns = runSummaryTurnCount(summary1)
		inputs.tokenUsage1 = summary1.TokenUsage
		inputs.rateLimit1 = summary1.GitHubRateLimitUsage
		inputs.metrics1 = &summary1.Metrics
	}
	if summary2 != nil {
		inputs.run2Tokens = summary2.Run.TokenUsage
		inputs.run2Duration = summary2.Run.Duration
		inputs.run2Turns = runSummaryTurnCount(summary2)
		inputs.tokenUsage2 = summary2.TokenUsage
		inputs.rateLimit2 = summary2.GitHubRateLimitUsage
		inputs.metrics2 = &summary2.Metrics
	}
	return inputs
}

func runSummaryTurnCount(summary *RunSummary) int {
	turns := summary.Run.Turns
	if turns == 0 && summary.Metrics.Turns > 0 {
		return summary.Metrics.Turns
	}
	return turns
}

func hasRunMetricsData(inputs runMetricsInputs) bool {
	return !(inputs.run1Tokens == 0 && inputs.run2Tokens == 0 &&
		inputs.run1Duration == 0 && inputs.run2Duration == 0 &&
		inputs.run1Turns == 0 && inputs.run2Turns == 0 &&
		inputs.tokenUsage1 == nil && inputs.tokenUsage2 == nil &&
		inputs.rateLimit1 == nil && inputs.rateLimit2 == nil)
}

func newRunMetricsDiff(inputs runMetricsInputs) *RunMetricsDiff {
	return &RunMetricsDiff{
		Run1TokenUsage: inputs.run1Tokens,
		Run2TokenUsage: inputs.run2Tokens,
		Run1Turns:      inputs.run1Turns,
		Run2Turns:      inputs.run2Turns,
		TurnsChange:    inputs.run2Turns - inputs.run1Turns,
	}
}

func applyRunMetricsTokenUsage(diff *RunMetricsDiff) {
	if diff.Run1TokenUsage > 0 || diff.Run2TokenUsage > 0 {
		diff.TokenUsageChange = formatVolumeChange(diff.Run1TokenUsage, diff.Run2TokenUsage)
	}
}

func applyRunMetricsDuration(diff *RunMetricsDiff, inputs runMetricsInputs) {
	if inputs.run1Duration > 0 {
		diff.Run1Duration = inputs.run1Duration.Round(time.Second).String()
	}
	if inputs.run2Duration > 0 {
		diff.Run2Duration = inputs.run2Duration.Round(time.Second).String()
	}
	if inputs.run1Duration == 0 || inputs.run2Duration == 0 {
		return
	}
	delta := inputs.run2Duration - inputs.run1Duration
	if delta >= 0 {
		diff.DurationChange = "+" + delta.Round(time.Second).String()
		return
	}
	diff.DurationChange = delta.Round(time.Second).String()
}

func applyRunMetricsTokensPerTurn(diff *RunMetricsDiff) {
	if diff.Run1Turns > 0 {
		diff.Run1TokensPerTurn = diff.Run1TokenUsage / diff.Run1Turns
	}
	if diff.Run2Turns > 0 {
		diff.Run2TokensPerTurn = diff.Run2TokenUsage / diff.Run2Turns
	}
	if diff.Run1TokensPerTurn > 0 || diff.Run2TokensPerTurn > 0 {
		diff.TokensPerTurnChange = formatVolumeChange(diff.Run1TokensPerTurn, diff.Run2TokensPerTurn)
	}
}

// isBashTool returns true if the tool name represents a bash/shell invocation.
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
	run1Tools := aggregateToolCallMap(m1)
	run2Tools := aggregateToolCallMap(m2)
	if len(run1Tools) == 0 && len(run2Tools) == 0 {
		return nil
	}
	state := newToolCallDiffState(run1Tools, run2Tools)
	for _, name := range collectToolCallNames(run1Tools, run2Tools) {
		processToolCallDiffEntry(state, name, run1Tools, run2Tools)
	}
	finalizeToolCallDiffState(state)
	auditDiffLog.Printf("Tool calls diff: new=%d, removed=%d, changed=%d, run1_total=%d, run2_total=%d",
		len(state.diff.NewTools), len(state.diff.RemovedTools), len(state.diff.ChangedTools), state.run1Total, state.run2Total)
	return state.diff
}

type toolCallDiffState struct {
	diff      *ToolCallsDiff
	run1Total int
	run2Total int
	bashRun1  map[string]ToolCallInfo
	bashRun2  map[string]ToolCallInfo
}

func aggregateToolCallMap(metrics *LogMetrics) map[string]ToolCallInfo {
	tools := make(map[string]ToolCallInfo)
	if metrics == nil {
		return tools
	}
	for _, call := range metrics.ToolCalls {
		aggregateToolCallInfo(tools, call)
	}
	return tools
}

func aggregateToolCallInfo(tools map[string]ToolCallInfo, call ToolCallInfo) {
	if existing, ok := tools[call.Name]; ok {
		existing.CallCount += call.CallCount
		if call.MaxInputSize > existing.MaxInputSize {
			existing.MaxInputSize = call.MaxInputSize
		}
		if call.MaxOutputSize > existing.MaxOutputSize {
			existing.MaxOutputSize = call.MaxOutputSize
		}
		if call.MaxDuration > existing.MaxDuration {
			existing.MaxDuration = call.MaxDuration
		}
		tools[call.Name] = existing
		return
	}
	tools[call.Name] = call
}

func newToolCallDiffState(run1Tools, run2Tools map[string]ToolCallInfo) *toolCallDiffState {
	return &toolCallDiffState{
		diff:     &ToolCallsDiff{},
		bashRun1: make(map[string]ToolCallInfo),
		bashRun2: make(map[string]ToolCallInfo),
	}
}

func collectToolCallNames(run1Tools, run2Tools map[string]ToolCallInfo) []string {
	allNames := make(map[string]struct{})
	for name := range run1Tools {
		allNames[name] = struct{}{}
	}
	for name := range run2Tools {
		allNames[name] = struct{}{}
	}
	return sliceutil.SortedKeys(allNames)
}

func processToolCallDiffEntry(state *toolCallDiffState, name string, run1Tools, run2Tools map[string]ToolCallInfo) {
	tc1, inRun1 := run1Tools[name]
	tc2, inRun2 := run2Tools[name]
	trackToolCallTotals(state, name, tc1, inRun1, tc2, inRun2)
	entry := buildToolCallDiffEntry(name, tc1, inRun1, tc2, inRun2)
	appendToolCallDiffEntry(state.diff, entry)
	state.diff.AllTools = append(state.diff.AllTools, entry)
}

func trackToolCallTotals(state *toolCallDiffState, name string, tc1 ToolCallInfo, inRun1 bool, tc2 ToolCallInfo, inRun2 bool) {
	if inRun1 {
		state.run1Total += tc1.CallCount
		if isBashTool(name) {
			state.bashRun1[name] = tc1
		}
	}
	if inRun2 {
		state.run2Total += tc2.CallCount
		if isBashTool(name) {
			state.bashRun2[name] = tc2
		}
	}
}

func buildToolCallDiffEntry(name string, tc1 ToolCallInfo, inRun1 bool, tc2 ToolCallInfo, inRun2 bool) ToolCallDiffEntry {
	entry := ToolCallDiffEntry{Name: name}
	switch {
	case !inRun1 && inRun2:
		entry.DiffEntryBase = DiffEntryBase{Status: "new"}
		entry.Run2CallCount = tc2.CallCount
		entry.Run2MaxInputSize = tc2.MaxInputSize
		entry.Run2MaxOutputSize = tc2.MaxOutputSize
	case inRun1 && !inRun2:
		entry.DiffEntryBase = DiffEntryBase{Status: "removed"}
		entry.Run1CallCount = tc1.CallCount
		entry.Run1MaxInputSize = tc1.MaxInputSize
		entry.Run1MaxOutputSize = tc1.MaxOutputSize
	case tc1.CallCount != tc2.CallCount:
		entry = buildChangedToolCallDiffEntry(name, tc1, tc2)
	default:
		entry = buildUnchangedToolCallDiffEntry(name, tc1, tc2)
	}
	return entry
}

func buildChangedToolCallDiffEntry(name string, before, after ToolCallInfo) ToolCallDiffEntry {
	return ToolCallDiffEntry{
		Name:              name,
		DiffEntryBase:     DiffEntryBase{Status: "changed"},
		Run1CallCount:     before.CallCount,
		Run2CallCount:     after.CallCount,
		CallCountChange:   formatCountChange(before.CallCount, after.CallCount),
		Run1MaxInputSize:  before.MaxInputSize,
		Run2MaxInputSize:  after.MaxInputSize,
		Run1MaxOutputSize: before.MaxOutputSize,
		Run2MaxOutputSize: after.MaxOutputSize,
	}
}

func buildUnchangedToolCallDiffEntry(name string, before, after ToolCallInfo) ToolCallDiffEntry {
	return ToolCallDiffEntry{
		Name:              name,
		DiffEntryBase:     DiffEntryBase{Status: "unchanged"},
		Run1CallCount:     before.CallCount,
		Run2CallCount:     after.CallCount,
		Run1MaxInputSize:  before.MaxInputSize,
		Run2MaxInputSize:  after.MaxInputSize,
		Run1MaxOutputSize: before.MaxOutputSize,
		Run2MaxOutputSize: after.MaxOutputSize,
	}
}

func appendToolCallDiffEntry(diff *ToolCallsDiff, entry ToolCallDiffEntry) {
	switch entry.Status {
	case "new":
		diff.NewTools = append(diff.NewTools, entry)
	case "removed":
		diff.RemovedTools = append(diff.RemovedTools, entry)
	case "changed":
		diff.ChangedTools = append(diff.ChangedTools, entry)
	}
}

func finalizeToolCallDiffState(state *toolCallDiffState) {
	state.diff.BashDiff = computeBashCommandsDiff(state.bashRun1, state.bashRun2)
	state.diff.Summary = ToolCallsDiffSummary{
		NewToolCount:     len(state.diff.NewTools),
		RemovedToolCount: len(state.diff.RemovedTools),
		ChangedToolCount: len(state.diff.ChangedTools),
		Run1TotalCalls:   state.run1Total,
		Run2TotalCalls:   state.run2Total,
	}
}

// computeBashCommandsDiff builds bash-specific analysis from pre-filtered bash tool call maps.
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
			DiffEntryBase:     DiffEntryBase{Status: status},
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
	inputs := extractTokenUsageInputs(tu1, tu2)
	diff := &TokenUsageDiff{
		Run1InputTokens:      inputs.run1Input,
		Run2InputTokens:      inputs.run2Input,
		Run1OutputTokens:     inputs.run1Output,
		Run2OutputTokens:     inputs.run2Output,
		Run1CacheReadTokens:  inputs.run1CacheRead,
		Run2CacheReadTokens:  inputs.run2CacheRead,
		Run1CacheWriteTokens: inputs.run1CacheWrite,
		Run2CacheWriteTokens: inputs.run2CacheWrite,
		Run1AIC:              inputs.run1AIC,
		Run2AIC:              inputs.run2AIC,
		Run1TotalRequests:    inputs.run1Requests,
		Run2TotalRequests:    inputs.run2Requests,
		Run1CacheEfficiency:  inputs.run1CacheEff,
		Run2CacheEfficiency:  inputs.run2CacheEff,
	}
	applyTokenUsageChanges(diff)
	return diff
}

type tokenUsageInputs struct {
	run1Input      int
	run2Input      int
	run1Output     int
	run2Output     int
	run1CacheRead  int
	run2CacheRead  int
	run1CacheWrite int
	run2CacheWrite int
	run1AIC        float64
	run2AIC        float64
	run1Requests   int
	run2Requests   int
	run1CacheEff   float64
	run2CacheEff   float64
}

func extractTokenUsageInputs(tu1, tu2 *TokenUsageSummary) tokenUsageInputs {
	inputs := tokenUsageInputs{}
	if tu1 != nil {
		inputs.run1Input = tu1.TotalInputTokens
		inputs.run1Output = tu1.TotalOutputTokens
		inputs.run1CacheRead = tu1.TotalCacheReadTokens
		inputs.run1CacheWrite = tu1.TotalCacheWriteTokens
		inputs.run1AIC = tu1.TotalAIC
		inputs.run1Requests = tu1.TotalRequests
		inputs.run1CacheEff = tu1.CacheEfficiency
	}
	if tu2 != nil {
		inputs.run2Input = tu2.TotalInputTokens
		inputs.run2Output = tu2.TotalOutputTokens
		inputs.run2CacheRead = tu2.TotalCacheReadTokens
		inputs.run2CacheWrite = tu2.TotalCacheWriteTokens
		inputs.run2AIC = tu2.TotalAIC
		inputs.run2Requests = tu2.TotalRequests
		inputs.run2CacheEff = tu2.CacheEfficiency
	}
	return inputs
}

func applyTokenUsageChanges(diff *TokenUsageDiff) {
	if diff.Run1InputTokens > 0 || diff.Run2InputTokens > 0 {
		diff.InputTokensChange = formatVolumeChange(diff.Run1InputTokens, diff.Run2InputTokens)
	}
	if diff.Run1OutputTokens > 0 || diff.Run2OutputTokens > 0 {
		diff.OutputTokensChange = formatVolumeChange(diff.Run1OutputTokens, diff.Run2OutputTokens)
	}
	if diff.Run1CacheReadTokens > 0 || diff.Run2CacheReadTokens > 0 {
		diff.CacheReadTokensChange = formatVolumeChange(diff.Run1CacheReadTokens, diff.Run2CacheReadTokens)
	}
	if diff.Run1CacheWriteTokens > 0 || diff.Run2CacheWriteTokens > 0 {
		diff.CacheWriteTokensChange = formatVolumeChange(diff.Run1CacheWriteTokens, diff.Run2CacheWriteTokens)
	}
	if diff.Run1AIC > 0 || diff.Run2AIC > 0 {
		diff.AICChange = formatFloatDelta(diff.Run1AIC, diff.Run2AIC)
	}
	if diff.Run1TotalRequests > 0 || diff.Run2TotalRequests > 0 {
		diff.RequestsDelta = formatCountChange(diff.Run1TotalRequests, diff.Run2TotalRequests)
	}
	if diff.Run1CacheEfficiency > 0 || diff.Run2CacheEfficiency > 0 {
		diff.CacheEfficiencyChange = formatPercentagePointChange(diff.Run1CacheEfficiency, diff.Run2CacheEfficiency)
	}
}

// loadRunSummaryForDiff loads or builds a RunSummary for a given run for use in diffing.
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
