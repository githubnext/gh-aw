package cli

import (
	"sort"
	"time"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stats"
)

var auditCrossRunLog = logger.New("cli:audit_cross_run")

// mcpErrorRateThreshold is the error-rate above which an MCP server is flagged as unreliable.
const mcpErrorRateThreshold = 0.10

// mcpConnectionRateThreshold is the minimum fraction of runs a server must appear in
// before it is flagged as unreliable due to low connectivity.
const mcpConnectionRateThreshold = 0.75

// spikeDetectionMultiplier is the ratio above which a run's cost or token usage is
// flagged as a spike relative to the cross-run average (e.g., 2.0 → >2x avg).
const spikeDetectionMultiplier = 2.0

// CrossRunAuditReport represents aggregated audit data across multiple workflow runs.
// It includes firewall analysis, metrics trends, MCP server health, and error trends.
type CrossRunAuditReport struct {
	RunsAnalyzed    int                       `json:"runs_analyzed"`
	RunsWithData    int                       `json:"runs_with_data"`
	RunsWithoutData int                       `json:"runs_without_data"`
	Summary         CrossRunSummary           `json:"summary"`
	MetricsTrend    MetricsTrendData          `json:"metrics_trend"`
	MCPHealth       []MCPServerCrossRunHealth `json:"mcp_health,omitempty"`
	ErrorTrend      ErrorTrendData            `json:"error_trend"`
	DomainInventory []DomainInventoryEntry    `json:"domain_inventory"`
	PerRunBreakdown []PerRunFirewallBreakdown `json:"per_run_breakdown"`
	Drain3Insights  []ObservabilityInsight    `json:"drain3_insights,omitempty"`
}

// CrossRunSummary provides top-level statistics across all analyzed runs.
type CrossRunSummary struct {
	TotalRequests   int     `json:"total_requests"`
	TotalAllowed    int     `json:"total_allowed"`
	TotalBlocked    int     `json:"total_blocked"`
	OverallDenyRate float64 `json:"overall_deny_rate"` // 0.0–1.0
	UniqueDomains   int     `json:"unique_domains"`
}

// MetricsTrendData contains aggregated cost, token, turn, and duration statistics
// across multiple runs, with spike detection for anomalous runs.
//
// Token counts (MinTokens, MaxTokens, AvgTokens) are stored as int to preserve
// integer semantics consistent with the source data; MedianTokens and StdDevTokens
// use float64 because statistical measures of integer quantities can be fractional.
//
// Duration fields only aggregate runs where timing data was recorded (duration > 0),
// so the duration statistics may cover fewer runs than the cost/token/turn statistics.
// All stddev fields use the sample standard deviation (Bessel's correction).
type MetricsTrendData struct {
	TotalCost    float64 `json:"total_cost"`
	AvgCost      float64 `json:"avg_cost"`
	MedianCost   float64 `json:"median_cost"`
	StdDevCost   float64 `json:"stddev_cost"`
	MinCost      float64 `json:"min_cost"`
	MaxCost      float64 `json:"max_cost"`
	TotalTokens  int     `json:"total_tokens"`
	AvgTokens    int     `json:"avg_tokens"`
	MedianTokens float64 `json:"median_tokens"` // float64: median of integer counts can be fractional
	StdDevTokens float64 `json:"stddev_tokens"` // float64: stddev is always fractional
	MinTokens    int     `json:"min_tokens"`
	MaxTokens    int     `json:"max_tokens"`
	TotalTurns   int     `json:"total_turns"`
	AvgTurns     float64 `json:"avg_turns"`
	MedianTurns  float64 `json:"median_turns"`
	StdDevTurns  float64 `json:"stddev_turns"`
	MaxTurns     int     `json:"max_turns"`
	// Duration statistics (stored as nanoseconds for JSON portability).
	// Only runs with duration > 0 contribute; runs without timing data are excluded.
	AvgDurationNs    int64   `json:"avg_duration_ns"`
	MedianDurationNs int64   `json:"median_duration_ns"`
	StdDevDurationNs int64   `json:"stddev_duration_ns"`
	MinDurationNs    int64   `json:"min_duration_ns"`
	MaxDurationNs    int64   `json:"max_duration_ns"`
	CostSpikes       []int64 `json:"cost_spikes,omitempty"`  // Run IDs with cost > 2x avg
	TokenSpikes      []int64 `json:"token_spikes,omitempty"` // Run IDs with tokens > 2x avg
	RunsWithCost     int     `json:"runs_with_cost"`         // Runs that reported non-zero cost
}

// MCPServerCrossRunHealth describes the health of a single MCP server across runs.
type MCPServerCrossRunHealth struct {
	ServerName    string  `json:"server_name"`
	RunsConnected int     `json:"runs_connected"` // Runs where server was used (appeared in tool usage)
	TotalRuns     int     `json:"total_runs"`
	TotalCalls    int     `json:"total_calls"`
	TotalErrors   int     `json:"total_errors"`
	ErrorRate     float64 `json:"error_rate"` // 0.0–1.0
	Unreliable    bool    `json:"unreliable"` // True if error_rate > 0.10 or connected < 75% of runs
}

// ErrorTrendData summarizes error and warning patterns across runs.
type ErrorTrendData struct {
	RunsWithErrors   int     `json:"runs_with_errors"`
	TotalErrors      int     `json:"total_errors"`
	AvgErrorsPerRun  float64 `json:"avg_errors_per_run"`
	RunsWithWarnings int     `json:"runs_with_warnings"`
	TotalWarnings    int     `json:"total_warnings"`
}

// DomainInventoryEntry describes a single domain seen across multiple runs.
type DomainInventoryEntry struct {
	Domain        string            `json:"domain"`
	SeenInRuns    int               `json:"seen_in_runs"`
	TotalAllowed  int               `json:"total_allowed"`
	TotalBlocked  int               `json:"total_blocked"`
	OverallStatus string            `json:"overall_status"` // "allowed", "denied", "mixed"
	PerRunStatus  []DomainRunStatus `json:"per_run_status"`
}

// DomainRunStatus records the status of a domain in a single run.
type DomainRunStatus struct {
	RunID   int64  `json:"run_id"`
	Status  string `json:"status"` // "allowed", "denied", "mixed", "absent"
	Allowed int    `json:"allowed"`
	Blocked int    `json:"blocked"`
}

// PerRunFirewallBreakdown is a summary row for a single run within the cross-run report.
// It extends the firewall view with cost, token, turn, and MCP error information.
type PerRunFirewallBreakdown struct {
	RunID         int64         `json:"run_id"`
	WorkflowName  string        `json:"workflow_name"`
	Conclusion    string        `json:"conclusion"`
	Duration      time.Duration `json:"duration_ns"` // Wall-clock duration of the run
	TotalRequests int           `json:"total_requests"`
	Allowed       int           `json:"allowed"`
	Blocked       int           `json:"blocked"`
	DenyRate      float64       `json:"deny_rate"` // 0.0–1.0
	UniqueDomains int           `json:"unique_domains"`
	Cost          float64       `json:"cost"`
	Tokens        int           `json:"tokens"`
	Turns         int           `json:"turns"`
	MCPErrors     int           `json:"mcp_errors"`
	ErrorCount    int           `json:"error_count"`
	HasData       bool          `json:"has_data"`
	CostSpike     bool          `json:"cost_spike,omitempty"`  // True if cost > 2x avg
	TokenSpike    bool          `json:"token_spike,omitempty"` // True if tokens > 2x avg
}

// crossRunInput bundles per-run data needed for aggregation.
type crossRunInput struct {
	RunID            int64
	WorkflowName     string
	Conclusion       string
	Duration         time.Duration
	FirewallAnalysis *FirewallAnalysis
	Metrics          LogMetrics
	MCPToolUsage     *MCPToolUsageData
	MCPFailures      []MCPFailureReport
	ErrorCount       int
}

// buildCrossRunAuditReport aggregates data from multiple runs into a CrossRunAuditReport.
func buildCrossRunAuditReport(inputs []crossRunInput) *CrossRunAuditReport {
	auditCrossRunLog.Printf("Building cross-run audit report: %d inputs", len(inputs))

	report := &CrossRunAuditReport{
		RunsAnalyzed: len(inputs),
	}

	// Aggregate per-domain data across all runs
	type domainAgg struct {
		totalAllowed int
		totalBlocked int
		perRun       []DomainRunStatus
	}
	domainMap := make(map[string]*domainAgg)

	// Ordered list of run IDs for deterministic per-run status
	runIDs := make([]int64, 0, len(inputs))
	for _, in := range inputs {
		runIDs = append(runIDs, in.RunID)
	}

	// --- Phase 1: build per-run breakdowns and collect raw values for aggregation ---

	var metricsRows []metricsRawRow

	// MCP server aggregation: server name → aggregate across runs
	type mcpServerAgg struct {
		totalCalls  int
		totalErrors int
		runsSeen    map[int64]bool
	}
	mcpServerMap := make(map[string]*mcpServerAgg)

	for _, in := range inputs {
		breakdown := PerRunFirewallBreakdown{
			RunID:        in.RunID,
			WorkflowName: in.WorkflowName,
			Conclusion:   in.Conclusion,
			Duration:     in.Duration,
			Cost:         in.Metrics.EstimatedCost,
			Tokens:       in.Metrics.TokenUsage,
			Turns:        in.Metrics.Turns,
			ErrorCount:   in.ErrorCount,
		}

		// Count MCP errors for this run
		breakdown.MCPErrors = len(in.MCPFailures)
		if in.MCPToolUsage != nil {
			for _, srv := range in.MCPToolUsage.Servers {
				breakdown.MCPErrors += srv.ErrorCount

				// Aggregate server-level stats
				agg, ok := mcpServerMap[srv.ServerName]
				if !ok {
					agg = &mcpServerAgg{runsSeen: make(map[int64]bool)}
					mcpServerMap[srv.ServerName] = agg
				}
				agg.totalCalls += srv.ToolCallCount
				agg.totalErrors += srv.ErrorCount
				agg.runsSeen[in.RunID] = true
			}
		}

		if in.FirewallAnalysis != nil {
			report.RunsWithData++
			breakdown.HasData = true
			breakdown.TotalRequests = in.FirewallAnalysis.TotalRequests
			breakdown.Allowed = in.FirewallAnalysis.AllowedRequests
			breakdown.Blocked = in.FirewallAnalysis.BlockedRequests
			if breakdown.TotalRequests > 0 {
				breakdown.DenyRate = float64(breakdown.Blocked) / float64(breakdown.TotalRequests)
			}
			breakdown.UniqueDomains = len(in.FirewallAnalysis.RequestsByDomain)

			report.Summary.TotalRequests += breakdown.TotalRequests
			report.Summary.TotalAllowed += breakdown.Allowed
			report.Summary.TotalBlocked += breakdown.Blocked

			for domain, stats := range in.FirewallAnalysis.RequestsByDomain {
				agg, exists := domainMap[domain]
				if !exists {
					agg = &domainAgg{}
					domainMap[domain] = agg
				}
				agg.totalAllowed += stats.Allowed
				agg.totalBlocked += stats.Blocked
				agg.perRun = append(agg.perRun, DomainRunStatus{
					RunID:   in.RunID,
					Status:  classifyFirewallDomainStatus(stats),
					Allowed: stats.Allowed,
					Blocked: stats.Blocked,
				})
			}
		} else {
			report.RunsWithoutData++
		}

		// Collect metrics for trend aggregation
		metricsRows = append(metricsRows, metricsRawRow{
			runID:    in.RunID,
			cost:     in.Metrics.EstimatedCost,
			tokens:   in.Metrics.TokenUsage,
			turns:    in.Metrics.Turns,
			duration: in.Duration,
		})

		// Error trend
		if in.ErrorCount > 0 {
			report.ErrorTrend.RunsWithErrors++
			report.ErrorTrend.TotalErrors += in.ErrorCount
		}

		report.PerRunBreakdown = append(report.PerRunBreakdown, breakdown)
	}

	// --- Phase 2: compute overall firewall summary ---
	if report.Summary.TotalRequests > 0 {
		report.Summary.OverallDenyRate = float64(report.Summary.TotalBlocked) / float64(report.Summary.TotalRequests)
	}

	// --- Phase 3: compute metrics trends ---
	report.MetricsTrend = buildMetricsTrend(metricsRows)

	// Mark cost/token spikes on per-run breakdowns
	spikeRunIDs := make(map[int64]bool, len(report.MetricsTrend.CostSpikes))
	for _, rid := range report.MetricsTrend.CostSpikes {
		spikeRunIDs[rid] = true
	}
	tokenSpikeRunIDs := make(map[int64]bool, len(report.MetricsTrend.TokenSpikes))
	for _, rid := range report.MetricsTrend.TokenSpikes {
		tokenSpikeRunIDs[rid] = true
	}
	for i := range report.PerRunBreakdown {
		if spikeRunIDs[report.PerRunBreakdown[i].RunID] {
			report.PerRunBreakdown[i].CostSpike = true
		}
		if tokenSpikeRunIDs[report.PerRunBreakdown[i].RunID] {
			report.PerRunBreakdown[i].TokenSpike = true
		}
	}

	// --- Phase 4: compute MCP health ---
	if len(mcpServerMap) > 0 {
		sortedServers := make([]string, 0, len(mcpServerMap))
		for name := range mcpServerMap {
			sortedServers = append(sortedServers, name)
		}
		sort.Strings(sortedServers)

		for _, name := range sortedServers {
			agg := mcpServerMap[name]
			connected := len(agg.runsSeen)
			var errorRate float64
			if agg.totalCalls > 0 {
				errorRate = float64(agg.totalErrors) / float64(agg.totalCalls)
			}
			unreliable := errorRate > mcpErrorRateThreshold || (len(inputs) > 0 && float64(connected)/float64(len(inputs)) < mcpConnectionRateThreshold)
			report.MCPHealth = append(report.MCPHealth, MCPServerCrossRunHealth{
				ServerName:    name,
				RunsConnected: connected,
				TotalRuns:     len(inputs),
				TotalCalls:    agg.totalCalls,
				TotalErrors:   agg.totalErrors,
				ErrorRate:     errorRate,
				Unreliable:    unreliable,
			})
		}
	}

	// --- Phase 5: compute error trend averages ---
	if len(inputs) > 0 {
		report.ErrorTrend.AvgErrorsPerRun = float64(report.ErrorTrend.TotalErrors) / float64(len(inputs))
	}

	// --- Phase 6: build domain inventory sorted by domain name ---
	sortedDomains := make([]string, 0, len(domainMap))
	for domain := range domainMap {
		sortedDomains = append(sortedDomains, domain)
	}
	sort.Strings(sortedDomains)

	report.Summary.UniqueDomains = len(sortedDomains)

	for _, domain := range sortedDomains {
		agg := domainMap[domain]
		presentRuns := make(map[int64]bool, len(agg.perRun))
		for _, prs := range agg.perRun {
			presentRuns[prs.RunID] = true
		}

		// Build full per-run status including "absent" for runs without this domain
		fullPerRun := make([]DomainRunStatus, 0, len(runIDs))
		for _, rid := range runIDs {
			if presentRuns[rid] {
				for _, prs := range agg.perRun {
					if prs.RunID == rid {
						fullPerRun = append(fullPerRun, prs)
						break
					}
				}
			} else {
				fullPerRun = append(fullPerRun, DomainRunStatus{
					RunID:  rid,
					Status: "absent",
				})
			}
		}

		entry := DomainInventoryEntry{
			Domain:        domain,
			SeenInRuns:    len(agg.perRun),
			TotalAllowed:  agg.totalAllowed,
			TotalBlocked:  agg.totalBlocked,
			OverallStatus: classifyFirewallDomainStatus(DomainRequestStats{Allowed: agg.totalAllowed, Blocked: agg.totalBlocked}),
			PerRunStatus:  fullPerRun,
		}
		report.DomainInventory = append(report.DomainInventory, entry)
	}

	auditCrossRunLog.Printf("Cross-run audit report built: runs=%d, with_data=%d, unique_domains=%d, mcp_servers=%d",
		report.RunsAnalyzed, report.RunsWithData, report.Summary.UniqueDomains, len(report.MCPHealth))

	// --- Phase 7: drain3 multi-run pattern analysis ---
	report.Drain3Insights = buildDrain3InsightsFromCrossRunInputs(inputs)

	return report
}

// metricsRawRow holds per-run raw metric values for aggregation.
type metricsRawRow struct {
	runID    int64
	cost     float64
	tokens   int
	turns    int
	duration time.Duration
}

// buildMetricsTrend computes aggregate metrics (min/max/avg/median/stddev/total, spike
// detection) from a slice of per-run raw metric rows.  Mean and variance are computed
// using Welford's online algorithm via StatVar for numerical stability.
func buildMetricsTrend(rows []metricsRawRow) MetricsTrendData {
	auditCrossRunLog.Printf("Building metrics trend from %d rows", len(rows))
	if len(rows) == 0 {
		return MetricsTrendData{}
	}

	var costStats, tokenStats, turnStats, durationStats stats.StatVar

	trend := MetricsTrendData{}
	for _, r := range rows {
		trend.TotalCost += r.cost
		trend.TotalTokens += r.tokens
		trend.TotalTurns += r.turns

		if r.cost > 0 {
			trend.RunsWithCost++
		}
		if r.turns > trend.MaxTurns {
			trend.MaxTurns = r.turns
		}

		costStats.Add(r.cost)
		tokenStats.Add(float64(r.tokens))
		turnStats.Add(float64(r.turns))
		// Only include runs where duration was measured to avoid pulling the
		// statistics toward zero for runs without timing data.
		if r.duration > 0 {
			durationStats.Add(float64(r.duration))
		}
	}

	if costStats.Count() > 0 {
		trend.AvgCost = costStats.Mean()
		trend.MedianCost = costStats.Median()
		trend.StdDevCost = costStats.SampleStdDev()
		trend.MinCost = costStats.Min()
		trend.MaxCost = costStats.Max()
	}
	if tokenStats.Count() > 0 {
		trend.AvgTokens = int(tokenStats.Mean())
		trend.MedianTokens = tokenStats.Median()
		trend.StdDevTokens = tokenStats.SampleStdDev()
		trend.MinTokens = int(tokenStats.Min())
		trend.MaxTokens = int(tokenStats.Max())
	}
	if turnStats.Count() > 0 {
		trend.AvgTurns = turnStats.Mean()
		trend.MedianTurns = turnStats.Median()
		trend.StdDevTurns = turnStats.SampleStdDev()
	}
	if durationStats.Count() > 0 {
		trend.AvgDurationNs = int64(durationStats.Mean())
		trend.MedianDurationNs = int64(durationStats.Median())
		trend.StdDevDurationNs = int64(durationStats.SampleStdDev())
		trend.MinDurationNs = int64(durationStats.Min())
		trend.MaxDurationNs = int64(durationStats.Max())
	}

	// Spike detection: > spikeDetectionMultiplier × average
	if trend.AvgCost > 0 {
		for _, r := range rows {
			if r.cost > spikeDetectionMultiplier*trend.AvgCost {
				trend.CostSpikes = append(trend.CostSpikes, r.runID)
			}
		}
	}
	if trend.AvgTokens > 0 {
		for _, r := range rows {
			if r.tokens > int(spikeDetectionMultiplier*float64(trend.AvgTokens)) {
				trend.TokenSpikes = append(trend.TokenSpikes, r.runID)
			}
		}
	}

	auditCrossRunLog.Printf("Metrics trend computed: avg_cost=%.4f, avg_tokens=%d, avg_turns=%.1f, cost_spikes=%d, token_spikes=%d",
		trend.AvgCost, trend.AvgTokens, trend.AvgTurns, len(trend.CostSpikes), len(trend.TokenSpikes))

	return trend
}

// buildDrain3InsightsFromCrossRunInputs converts cross-run inputs to ProcessedRuns and
// delegates to the shared multi-run drain3 analysis function.
// Returns nil if inputs is empty or if no events could be extracted.
func buildDrain3InsightsFromCrossRunInputs(inputs []crossRunInput) []ObservabilityInsight {
	if len(inputs) == 0 {
		return nil
	}
	runs := make([]ProcessedRun, 0, len(inputs))
	for _, in := range inputs {
		pr := ProcessedRun{
			Run: WorkflowRun{
				DatabaseID:    in.RunID,
				WorkflowName:  in.WorkflowName,
				Conclusion:    in.Conclusion,
				Duration:      in.Duration,
				Turns:         in.Metrics.Turns,
				TokenUsage:    in.Metrics.TokenUsage,
				EstimatedCost: in.Metrics.EstimatedCost,
				ErrorCount:    in.ErrorCount,
			},
			MCPFailures: in.MCPFailures,
		}
		runs = append(runs, pr)
	}
	return buildDrain3InsightsMultiRun(runs)
}
