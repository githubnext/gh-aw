package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

var logsCompactLog = logger.New("cli:logs_format_compact")

// workflowIDFromPath extracts the workflow ID from a workflow path.
// e.g. ".github/workflows/smoke-antigravity.lock.yml" → "smoke-antigravity"
func workflowIDFromPath(path string) string {
	// Get the base filename
	base := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		base = path[idx+1:]
	}
	// Strip .lock.yml suffix
	base = strings.TrimSuffix(base, ".lock.yml")
	// Strip .yml/.yaml suffix (in case it's not a lock file)
	base = strings.TrimSuffix(base, ".yml")
	base = strings.TrimSuffix(base, ".yaml")
	return base
}

// workflowIDFromRun returns the workflow ID preferring the path-derived ID,
// falling back to a lowercased/hyphenated version of the display name.
func workflowIDFromRun(path, name string) string {
	if id := workflowIDFromPath(path); id != "" {
		return id
	}
	// Normalize display name to kebab-case ID
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "-")
	return id
}

// renderLogsCompact outputs maximally information-dense output optimized for agentic consumption.
// Designed for LLM context windows: minimal formatting, no decoration, structured but flat.
//
// Format sections:
//
//	[summary] key=value pairs on one line
//	[runs] aligned table with essential per-run metrics
//	[errors] one-line-per-error entries (only if errors exist)
//	[insights] observability insights (only medium/high severity)
//	[firewall] firewall summary with per-domain breakdown
//	[tools] top tool usage (only if present)
//	[mcp] MCP failures (only if present)
func renderLogsCompact(data LogsData) {
	logsCompactLog.Printf("Rendering %d runs in compact format", data.Summary.TotalRuns)

	renderLogsCompactSummary(data.Summary)

	if len(data.Runs) == 0 {
		return
	}

	renderLogsCompactRuns(data.Runs)

	renderLogsCompactSections(data)
	renderLogsCompactHint(data)
}

func renderLogsCompactSummary(s LogsSummary) {
	summaryParts := []string{
		"runs=" + strconv.Itoa(s.TotalRuns),
		"duration=" + s.TotalDuration,
		"turns=" + strconv.Itoa(s.TotalTurns),
		"errors=" + strconv.Itoa(s.TotalErrors),
	}
	if s.TotalAIC > 0 {
		summaryParts = append(summaryParts, "aic="+formatCompactAIC(s.TotalAIC))
	}
	if s.TotalTokens > 0 {
		summaryParts = append(summaryParts, "tokens="+strconv.Itoa(s.TotalTokens))
	}
	if s.TotalWarnings > 0 {
		summaryParts = append(summaryParts, "warnings="+strconv.Itoa(s.TotalWarnings))
	}
	if s.TotalMissingTools > 0 {
		summaryParts = append(summaryParts, "missing_tools="+strconv.Itoa(s.TotalMissingTools))
	}
	if s.TotalGitHubAPICalls > 0 {
		summaryParts = append(summaryParts, "github_api="+strconv.Itoa(s.TotalGitHubAPICalls))
	}
	renderLogsCompactAppendEngineCounts(&summaryParts, s.EngineCounts)
	renderLogsCompactAppendOutcome(&summaryParts, s, false)
	fmt.Fprintf(os.Stdout, "[summary] %s\n", strings.Join(summaryParts, " "))
}

func renderLogsCompactAppendEngineCounts(summaryParts *[]string, engineCounts map[string]int) {
	if len(engineCounts) == 0 {
		return
	}
	parts := make([]string, 0, len(engineCounts))
	for engine, count := range engineCounts {
		parts = append(parts, engine+":"+strconv.Itoa(count))
	}
	*summaryParts = append(*summaryParts, "engines="+strings.Join(parts, ","))
}

func renderLogsCompactAppendOutcome(summaryParts *[]string, s LogsSummary, verbose bool) {
	if s.OutcomeAccepted == 0 && s.OutcomeRejected == 0 {
		return
	}
	*summaryParts = append(*summaryParts, "accepted="+strconv.Itoa(s.OutcomeAccepted), "rejected="+strconv.Itoa(s.OutcomeRejected))
	if verbose {
		*summaryParts = append(*summaryParts, "ignored="+strconv.Itoa(s.OutcomeIgnored), "pending="+strconv.Itoa(s.OutcomePending))
	}
	if s.OutcomeAcceptanceRate > 0 {
		*summaryParts = append(*summaryParts, "acceptance="+fmt.Sprintf("%.0f%%", s.OutcomeAcceptanceRate*100))
	}
	if verbose && s.OutcomeWasteRate > 0 {
		*summaryParts = append(*summaryParts, "waste="+fmt.Sprintf("%.0f%%", s.OutcomeWasteRate*100))
	}
}

func renderLogsCompactRuns(runs []RunData) {
	fmt.Fprintln(os.Stdout, "[runs]")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RUNID\tWORKFLOW\tENGINE\tSTATUS\tDUR\tTOKENS\tAIC\tTURNS\tERR\tEVENT\tACTOR\tBRANCH")
	for _, r := range runs {
		status, dur, actor := renderLogsCompactRunFields(r)
		if status == "skipped" || status == "cancelled" {
			continue
		}
		branch := stringutil.Truncate(r.Branch, 30)
		wfID := workflowIDFromRun(r.WorkflowPath, r.WorkflowName)
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%d\t%s\t%d\t%d\t%s\t%s\t%s\n",
			r.RunID, wfID, r.EngineID, status, dur,
			r.TokenUsage, formatCompactAIC(r.AIC), r.Turns, r.ErrorCount,
			r.Event, actor, branch)
	}
	w.Flush()
}

func renderLogsCompactRunFields(r RunData) (string, string, string) {
	status := r.Conclusion
	if status == "" {
		status = r.Status
	}
	dur := r.Duration
	if dur == "" {
		dur = "-"
	}
	actor := r.Actor
	if actor == "" {
		actor = "-"
	}
	return status, dur, actor
}

func renderLogsCompactSections(data LogsData) {
	renderLogsCompactErrors(data.ErrorsAndWarnings)
	renderLogsCompactInsights(data.Observability)
	renderLogsCompactFirewall(data.FirewallLog)
	renderLogsCompactTools(data.ToolUsage)
	renderLogsCompactMCPFailures(data.MCPFailures)
	renderLogsCompactMissingTools(data.MissingTools)
	if data.LogsLocation != "" {
		fmt.Fprintf(os.Stdout, "[location] %s\n", data.LogsLocation)
	}
}

func renderLogsCompactErrors(errorsAndWarnings []ErrorSummary) {
	if len(errorsAndWarnings) == 0 {
		return
	}
	fmt.Fprintln(os.Stdout, "[errors]")
	for _, ew := range errorsAndWarnings {
		msg := stringutil.Truncate(ew.Message, 120)
		fmt.Fprintf(os.Stdout, "%s run=%d count=%d: %s\n", ew.Type, ew.RunID, ew.Count, msg)
	}
}

func renderLogsCompactInsights(observability []ObservabilityInsight) {
	if len(observability) == 0 {
		return
	}
	var hasActionable bool
	for _, obs := range observability {
		if obs.Severity != "info" {
			hasActionable = true
			break
		}
	}
	if !hasActionable {
		return
	}
	fmt.Fprintln(os.Stdout, "[insights]")
	for _, obs := range observability {
		if obs.Severity != "info" {
			fmt.Fprintf(os.Stdout, "[%s] %s: %s\n", obs.Severity, obs.Title, obs.Summary)
		}
	}
}

func renderLogsCompactFirewall(fw *FirewallLogSummary) {
	if fw == nil || fw.TotalRequests == 0 {
		return
	}
	fmt.Fprintf(os.Stdout, "[firewall] requests=%d allowed=%d blocked=%d\n", fw.TotalRequests, fw.AllowedRequests, fw.BlockedRequests)
	if len(fw.RequestsByDomain) > 0 {
		for domain, counts := range fw.RequestsByDomain {
			if counts.Blocked > 0 {
				fmt.Fprintf(os.Stdout, "  %s allowed=%d blocked=%d\n", domain, counts.Allowed, counts.Blocked)
			}
		}
	} else if len(fw.BlockedDomains) > 0 {
		fmt.Fprintf(os.Stdout, "  blocked: %s\n", strings.Join(fw.BlockedDomains, " "))
	}
}

func renderLogsCompactTools(toolUsage []ToolUsageSummary) {
	if len(toolUsage) == 0 {
		return
	}
	fmt.Fprintln(os.Stdout, "[tools]")
	limit := min(10, len(toolUsage))
	for i := range limit {
		t := toolUsage[i]
		fmt.Fprintf(os.Stdout, "%s calls=%d runs=%d\n", t.Name, t.TotalCalls, t.Runs)
	}
	if len(toolUsage) > limit {
		fmt.Fprintf(os.Stdout, "... +%d more tools\n", len(toolUsage)-limit)
	}
}

func renderLogsCompactMCPFailures(failures []MCPFailureSummary) {
	if len(failures) == 0 {
		return
	}
	fmt.Fprintln(os.Stdout, "[mcp-failures]")
	for _, f := range failures {
		fmt.Fprintf(os.Stdout, "server=%s count=%d runs=%v\n", f.ServerName, f.Count, f.RunIDs)
	}
}

func renderLogsCompactMissingTools(missingTools []MissingToolSummary) {
	if len(missingTools) == 0 {
		return
	}
	fmt.Fprintln(os.Stdout, "[missing-tools]")
	for _, mt := range missingTools {
		fmt.Fprintf(os.Stdout, "%s count=%d runs=%v\n", mt.Tool, mt.Count, mt.RunIDs)
	}
}

func renderLogsCompactHint(data LogsData) {
	hint := "use --json for full details, -v for verbose, --format console for tables"
	if data.Message != "" {
		hint = data.Message + " " + hint
	}
	fmt.Fprintf(os.Stdout, "[hint] %s\n", hint)
}

// renderLogsCompactVerbose adds extra columns and sections for deeper analysis.
func renderLogsCompactVerbose(data LogsData) {
	logsCompactLog.Printf("Rendering %d runs in verbose compact format", data.Summary.TotalRuns)

	renderLogsCompactVerboseSummary(data.Summary)

	if len(data.Runs) == 0 {
		return
	}

	renderLogsCompactVerboseRuns(data.Runs)
	renderLogsCompactVerboseSections(data)
}

func renderLogsCompactVerboseSummary(s LogsSummary) {
	summaryParts := []string{
		"runs=" + strconv.Itoa(s.TotalRuns),
		"duration=" + s.TotalDuration,
		"action_min=" + fmt.Sprintf("%.1f", s.TotalActionMinutes),
		"turns=" + strconv.Itoa(s.TotalTurns),
		"errors=" + strconv.Itoa(s.TotalErrors),
		"warnings=" + strconv.Itoa(s.TotalWarnings),
		"missing_tools=" + strconv.Itoa(s.TotalMissingTools),
		"github_api=" + strconv.Itoa(s.TotalGitHubAPICalls),
		"episodes=" + strconv.Itoa(s.TotalEpisodes),
	}
	if s.TotalAIC > 0 {
		summaryParts = append(summaryParts, "aic="+formatCompactAIC(s.TotalAIC))
	}
	renderLogsCompactAppendEngineCounts(&summaryParts, s.EngineCounts)
	renderLogsCompactAppendOutcome(&summaryParts, s, true)
	fmt.Fprintf(os.Stdout, "[summary] %s\n", strings.Join(summaryParts, " "))
}

func renderLogsCompactVerboseRuns(runs []RunData) {
	fmt.Fprintln(os.Stdout, "[runs]")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RUNID\tWORKFLOW\tENGINE\tSTATUS\tDUR\tTOKENS\tAIC\tTURNS\tERR\tWARN\tEVENT\tACTOR\tTBT\tCLASS\tCREATED\tBRANCH")
	for _, r := range runs {
		status, dur, actor := renderLogsCompactRunFields(r)
		if status == "skipped" || status == "cancelled" {
			continue
		}
		tbt := r.AvgTimeBetweenTurns
		if tbt == "" {
			tbt = "-"
		}
		classification := r.Classification
		if classification == "" {
			classification = "-"
		}
		wfID := workflowIDFromRun(r.WorkflowPath, r.WorkflowName)
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%d\t%s\t%d\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			r.RunID, wfID, r.EngineID, status, dur,
			r.TokenUsage, formatCompactAIC(r.AIC),
			r.Turns, r.ErrorCount, r.WarningCount,
			r.Event, actor, tbt, classification,
			r.CreatedAt.Format("01-02 15:04"), r.Branch)
	}
	w.Flush()
}

func renderLogsCompactVerboseSections(data LogsData) {
	renderLogsCompactVerboseErrors(data.ErrorsAndWarnings)
	renderLogsCompactVerboseInsights(data.Observability)
	renderLogsCompactVerboseFirewall(data.FirewallLog)
	renderLogsCompactVerboseTools(data.ToolUsage)
	renderLogsCompactVerboseMCPTools(data.MCPToolUsage)
	renderLogsCompactMCPFailures(data.MCPFailures)
	renderLogsCompactMissingTools(data.MissingTools)
	renderLogsCompactVerboseEpisodes(data.Episodes)
	if data.LogsLocation != "" {
		fmt.Fprintf(os.Stdout, "[location] %s\n", data.LogsLocation)
	}
}

func renderLogsCompactVerboseErrors(errorsAndWarnings []ErrorSummary) {
	if len(errorsAndWarnings) == 0 {
		return
	}
	fmt.Fprintln(os.Stdout, "[errors]")
	for _, ew := range errorsAndWarnings {
		fmt.Fprintf(os.Stdout, "%s run=%d count=%d: %s\n", ew.Type, ew.RunID, ew.Count, ew.Message)
	}
}

func renderLogsCompactVerboseInsights(observability []ObservabilityInsight) {
	if len(observability) == 0 {
		return
	}
	fmt.Fprintln(os.Stdout, "[insights]")
	for _, obs := range observability {
		fmt.Fprintf(os.Stdout, "[%s] %s: %s\n", obs.Severity, obs.Title, obs.Summary)
	}
}

func renderLogsCompactVerboseFirewall(fw *FirewallLogSummary) {
	if fw == nil || fw.TotalRequests == 0 {
		return
	}
	fmt.Fprintf(os.Stdout, "[firewall] requests=%d allowed=%d blocked=%d\n", fw.TotalRequests, fw.AllowedRequests, fw.BlockedRequests)
	if len(fw.RequestsByDomain) > 0 {
		for domain, counts := range fw.RequestsByDomain {
			fmt.Fprintf(os.Stdout, "  %s allowed=%d blocked=%d\n", domain, counts.Allowed, counts.Blocked)
		}
	}
}

func renderLogsCompactVerboseTools(toolUsage []ToolUsageSummary) {
	if len(toolUsage) == 0 {
		return
	}
	fmt.Fprintln(os.Stdout, "[tools]")
	for _, t := range toolUsage {
		fmt.Fprintf(os.Stdout, "%s calls=%d runs=%d\n", t.Name, t.TotalCalls, t.Runs)
	}
}

func renderLogsCompactVerboseMCPTools(mcpToolUsage *MCPToolUsageSummary) {
	if mcpToolUsage == nil || len(mcpToolUsage.Summary) == 0 {
		return
	}
	fmt.Fprintln(os.Stdout, "[mcp-tools]")
	for _, t := range mcpToolUsage.Summary {
		fmt.Fprintf(os.Stdout, "%s.%s calls=%d\n", t.ServerName, t.ToolName, t.CallCount)
	}
}

func renderLogsCompactVerboseEpisodes(episodes []EpisodeData) {
	if len(episodes) == 0 {
		return
	}
	fmt.Fprintln(os.Stdout, "[episodes]")
	for _, ep := range episodes {
		fmt.Fprintf(os.Stdout, "%s runs=%d conf=%s duration=%s\n", ep.Kind, ep.TotalRuns, ep.Confidence, ep.TotalDuration)
	}
}

func formatCompactAIC(value float64) string {
	if value <= 0 {
		return "-"
	}
	if value >= 1000 {
		return fmt.Sprintf("%.1fK", value/1000)
	}
	if value >= 10 {
		return fmt.Sprintf("%.1f", value)
	}
	if value >= 1 {
		return fmt.Sprintf("%.2f", value)
	}
	return fmt.Sprintf("%.3f", value)
}
