package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/timeutil"
)

var crossRunRenderLog = logger.New("cli:audit_cross_run_render")

// renderCrossRunReportJSON outputs the cross-run report as JSON to stdout.
func renderCrossRunReportJSON(report *CrossRunAuditReport) error {
	crossRunRenderLog.Printf("Rendering cross-run report as JSON: runs_analyzed=%d, domains=%d", report.RunsAnalyzed, len(report.DomainInventory))
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// renderCrossRunReportMarkdown outputs the cross-run report as Markdown to stdout.
func renderCrossRunReportMarkdown(report *CrossRunAuditReport) {
	crossRunRenderLog.Printf("Rendering cross-run report as markdown: runs_analyzed=%d, domains=%d", report.RunsAnalyzed, len(report.DomainInventory))
	fmt.Fprintln(os.Stdout, "# Audit Report — Cross-Run Analysis")
	fmt.Fprintln(os.Stdout)

	renderMarkdownExecutiveSummary(report)
	renderMarkdownMetricsTrend(report.MetricsTrend)
	renderMarkdownMCPHealth(report)
	renderMarkdownErrorTrend(report)
	renderMarkdownDomainInventory(report)
	renderMarkdownDrain3Insights(report.Drain3Insights)
	renderMarkdownPerRunBreakdown(report.PerRunBreakdown)
}

func renderMarkdownExecutiveSummary(report *CrossRunAuditReport) {
	fmt.Fprintln(os.Stdout, "## Executive Summary")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "| Metric | Value |\n")
	fmt.Fprintf(os.Stdout, "|--------|-------|\n")
	fmt.Fprintf(os.Stdout, "| Runs analyzed | %d |\n", report.RunsAnalyzed)
	fmt.Fprintf(os.Stdout, "| Runs with firewall data | %d |\n", report.RunsWithData)
	fmt.Fprintf(os.Stdout, "| Runs without firewall data | %d |\n", report.RunsWithoutData)
	fmt.Fprintf(os.Stdout, "| Total requests | %d |\n", report.Summary.TotalRequests)
	fmt.Fprintf(os.Stdout, "| Allowed requests | %d |\n", report.Summary.TotalAllowed)
	fmt.Fprintf(os.Stdout, "| Blocked requests | %d |\n", report.Summary.TotalBlocked)
	fmt.Fprintf(os.Stdout, "| Overall denial rate | %.1f%% |\n", report.Summary.OverallDenyRate*100)
	fmt.Fprintf(os.Stdout, "| Unique domains | %d |\n", report.Summary.UniqueDomains)
	fmt.Fprintln(os.Stdout)
}

func renderMarkdownMetricsTrend(mt MetricsTrendData) {
	if mt.TotalTokens == 0 && mt.TotalTurns == 0 && mt.AvgDurationNs == 0 {
		return
	}

	fmt.Fprintln(os.Stdout, "## Metrics Trends")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "| Metric | Total | Avg/run | Min | Max | Spikes |\n")
	fmt.Fprintf(os.Stdout, "|--------|-------|---------|-----|-----|--------|\n")
	if mt.TotalTokens > 0 {
		spikes := "—"
		if len(mt.TokenSpikes) > 0 {
			spikes = "⚠ " + formatRunIDs(mt.TokenSpikes)
		}
		fmt.Fprintf(os.Stdout, "| Token Trend | %d | %d | %d | %d | %s |\n",
			mt.TotalTokens, mt.AvgTokens, mt.MinTokens, mt.MaxTokens, spikes)
	}
	if mt.TotalTurns > 0 {
		fmt.Fprintf(os.Stdout, "| Turns | %d | %.1f | — | %d | — |\n",
			mt.TotalTurns, mt.AvgTurns, mt.MaxTurns)
	}
	if mt.AvgDurationNs > 0 {
		fmt.Fprintf(os.Stdout, "| Duration | — | %s | %s | %s | — |\n",
			timeutil.FormatDurationNs(mt.AvgDurationNs),
			timeutil.FormatDurationNs(mt.MinDurationNs),
			timeutil.FormatDurationNs(mt.MaxDurationNs))
	}
	fmt.Fprintln(os.Stdout)
}

func renderMarkdownMCPHealth(report *CrossRunAuditReport) {
	if len(report.MCPHealth) == 0 {
		return
	}
	fmt.Fprintf(os.Stdout, "## MCP Server Health (%d runs)\n\n", report.RunsAnalyzed)
	fmt.Fprintf(os.Stdout, "| Server | Connected | Error Rate | Total Calls | Errors | Status |\n")
	fmt.Fprintf(os.Stdout, "|--------|-----------|------------|-------------|--------|--------|\n")
	for _, h := range report.MCPHealth {
		status := "✅ ok"
		if h.Unreliable {
			status = "⚠ unreliable"
		}
		fmt.Fprintf(os.Stdout, "| `%s` | %d/%d | %.1f%% | %d | %d | %s |\n",
			h.ServerName, h.RunsConnected, h.TotalRuns,
			h.ErrorRate*100, h.TotalCalls, h.TotalErrors, status)
	}
	fmt.Fprintln(os.Stdout)
}

func renderMarkdownErrorTrend(report *CrossRunAuditReport) {
	et := report.ErrorTrend
	if et.TotalErrors == 0 && et.TotalWarnings == 0 {
		return
	}
	fmt.Fprintln(os.Stdout, "## Error Trend")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "| Metric | Value |\n")
	fmt.Fprintf(os.Stdout, "|--------|-------|\n")
	fmt.Fprintf(os.Stdout, "| Runs with errors | %d/%d (%.0f%%) |\n",
		et.RunsWithErrors, report.RunsAnalyzed,
		safePercent(et.RunsWithErrors, report.RunsAnalyzed))
	fmt.Fprintf(os.Stdout, "| Total errors | %d |\n", et.TotalErrors)
	fmt.Fprintf(os.Stdout, "| Avg errors/run | %.2f |\n", et.AvgErrorsPerRun)
	if et.TotalWarnings > 0 {
		fmt.Fprintf(os.Stdout, "| Runs with warnings | %d/%d |\n", et.RunsWithWarnings, report.RunsAnalyzed)
		fmt.Fprintf(os.Stdout, "| Total warnings | %d |\n", et.TotalWarnings)
	}
	fmt.Fprintln(os.Stdout)
}

func renderMarkdownDomainInventory(report *CrossRunAuditReport) {
	if len(report.DomainInventory) == 0 {
		return
	}
	fmt.Fprintln(os.Stdout, "## Domain Inventory")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "| Domain | Status | Seen In | Allowed | Blocked |\n")
	fmt.Fprintf(os.Stdout, "|--------|--------|---------|---------|--------|\n")
	for _, entry := range report.DomainInventory {
		fmt.Fprintf(os.Stdout, "| `%s` | %s %s | %d/%d runs | %d | %d |\n",
			entry.Domain, firewallStatusEmoji(entry.OverallStatus), entry.OverallStatus,
			entry.SeenInRuns, report.RunsAnalyzed, entry.TotalAllowed, entry.TotalBlocked)
	}
	fmt.Fprintln(os.Stdout)
}

func renderMarkdownDrain3Insights(insights []ObservabilityInsight) {
	if len(insights) == 0 {
		return
	}
	crossRunRenderLog.Printf("Rendering markdown drain3 insights: count=%d", len(insights))
	fmt.Fprintln(os.Stdout, "## Agent Event Pattern Analysis")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "| Severity | Category | Title | Summary |\n")
	fmt.Fprintf(os.Stdout, "|----------|----------|-------|--------|\n")
	for _, insight := range insights {
		summary := insight.Summary
		if insight.Evidence != "" {
			summary += " (" + insight.Evidence + ")"
		}
		fmt.Fprintf(os.Stdout, "| %s %s | %s | %s | %s |\n",
			renderSeverityIcon(insight.Severity), insight.Severity, insight.Category, insight.Title, summary)
	}
	fmt.Fprintln(os.Stdout)
}

func renderMarkdownPerRunBreakdown(runs []PerRunFirewallBreakdown) {
	if len(runs) == 0 {
		return
	}
	fmt.Fprintln(os.Stdout, "## Per-Run Breakdown")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "| Run ID | Workflow | Conclusion | Duration | Firewall | Tokens | Turns | MCP Err | Errors |\n")
	fmt.Fprintf(os.Stdout, "|--------|----------|------------|----------|----------|--------|-------|---------|--------|\n")
	for _, run := range runs {
		firewallCol, tokenStr, turnsStr, durStr := markdownPerRunFields(run)
		fmt.Fprintf(os.Stdout, "| %d | %s | %s | %s | %s | %s | %s | %d | %d |\n",
			run.RunID, run.WorkflowName, run.Conclusion, durStr,
			firewallCol, tokenStr, turnsStr,
			run.MCPErrors, run.ErrorCount)
	}
	fmt.Fprintln(os.Stdout)
}

func markdownPerRunFields(run PerRunFirewallBreakdown) (string, string, string, string) {
	firewallCol := "—"
	if run.HasData {
		firewallCol = fmt.Sprintf("%d/%d", run.Allowed, run.Blocked)
	}
	tokenStr := "—"
	if run.Tokens > 0 {
		tokenStr = console.FormatTokens(run.Tokens)
		if run.TokenSpike {
			tokenStr += " ⚠"
		}
	}
	turnsStr := "—"
	if run.Turns > 0 {
		turnsStr = strconv.Itoa(run.Turns)
	}
	durStr := "—"
	if run.Duration > 0 {
		durStr = timeutil.FormatDurationNs(int64(run.Duration))
	}
	return firewallCol, tokenStr, turnsStr, durStr
}

// renderCrossRunReportPretty outputs the cross-run report as formatted console output to stderr.
func renderCrossRunReportPretty(report *CrossRunAuditReport) {
	crossRunRenderLog.Printf("Rendering cross-run report as pretty output: runs_analyzed=%d, runs_with_data=%d, deny_rate=%.1f%%",
		report.RunsAnalyzed, report.RunsWithData, report.Summary.OverallDenyRate*100)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Audit Report — Cross-Run Analysis"))
	fmt.Fprintln(os.Stderr)

	renderPrettyExecutiveSummary(report)
	renderPrettyMetricsTrend(report.MetricsTrend)
	renderPrettyMCPHealth(report)
	renderPrettyErrorTrend(report)
	renderPrettyDomainInventory(report)
	renderPrettyDrain3Insights(report.Drain3Insights)
	renderPrettyPerRunBreakdown(report.PerRunBreakdown)
	renderPrettyFinalStatus(report)
}

func renderPrettyExecutiveSummary(report *CrossRunAuditReport) {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Executive Summary"))
	fmt.Fprintf(os.Stderr, "  Runs analyzed:              %d\n", report.RunsAnalyzed)
	fmt.Fprintf(os.Stderr, "  Runs with firewall data:    %d\n", report.RunsWithData)
	fmt.Fprintf(os.Stderr, "  Runs without firewall data: %d\n", report.RunsWithoutData)
	fmt.Fprintf(os.Stderr, "  Total requests:             %d\n", report.Summary.TotalRequests)
	fmt.Fprintf(os.Stderr, "  Allowed / Blocked:          %d / %d\n", report.Summary.TotalAllowed, report.Summary.TotalBlocked)
	fmt.Fprintf(os.Stderr, "  Overall denial rate:        %.1f%%\n", report.Summary.OverallDenyRate*100)
	fmt.Fprintf(os.Stderr, "  Unique domains:             %d\n", report.Summary.UniqueDomains)
	fmt.Fprintln(os.Stderr)
}

func renderPrettyMetricsTrend(mt MetricsTrendData) {
	if mt.TotalTokens == 0 && mt.TotalTurns == 0 && mt.AvgDurationNs == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Metrics Trends"))
	renderPrettyTokenTrend(mt)
	renderPrettyTurnTrend(mt)
	renderPrettyDurationTrend(mt)
	fmt.Fprintln(os.Stderr)
}

func renderPrettyTokenTrend(mt MetricsTrendData) {
	if mt.TotalTokens == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "  Tokens:   total=%s  avg=%s/run  min=%s  max=%s\n%s",
		console.FormatTokens(mt.TotalTokens), console.FormatTokens(mt.AvgTokens),
		console.FormatTokens(mt.MinTokens), console.FormatTokens(mt.MaxTokens), prettySpikeNote("Token", mt.TokenSpikes))
}

func renderPrettyTurnTrend(mt MetricsTrendData) {
	if mt.TotalTurns > 0 {
		fmt.Fprintf(os.Stderr, "  Turns:    total=%d  avg=%.1f/run  max=%d\n", mt.TotalTurns, mt.AvgTurns, mt.MaxTurns)
	}
}

func renderPrettyDurationTrend(mt MetricsTrendData) {
	if mt.AvgDurationNs > 0 {
		fmt.Fprintf(os.Stderr, "  Duration: avg=%s  min=%s  max=%s\n",
			timeutil.FormatDurationNs(mt.AvgDurationNs),
			timeutil.FormatDurationNs(mt.MinDurationNs),
			timeutil.FormatDurationNs(mt.MaxDurationNs))
	}
}

func prettySpikeNote(label string, runIDs []int64) string {
	if len(runIDs) == 0 {
		return ""
	}
	return fmt.Sprintf("  ⚠ %s spikes in runs: %s\n", label, formatRunIDs(runIDs))
}

func renderPrettyMCPHealth(report *CrossRunAuditReport) {
	if len(report.MCPHealth) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("MCP Server Health (%d runs)", report.RunsAnalyzed)))
	for _, h := range report.MCPHealth {
		statusIcon := "✅"
		if h.Unreliable {
			statusIcon = "⚠"
		}
		fmt.Fprintf(os.Stderr, "  %s %-30s  connected=%d/%d  calls=%d  errors=%d  error_rate=%.1f%%\n",
			statusIcon, h.ServerName, h.RunsConnected, h.TotalRuns,
			h.TotalCalls, h.TotalErrors, h.ErrorRate*100)
	}
	fmt.Fprintln(os.Stderr)
}

func renderPrettyErrorTrend(report *CrossRunAuditReport) {
	et := report.ErrorTrend
	if et.TotalErrors == 0 && et.TotalWarnings == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Error Trend"))
	fmt.Fprintf(os.Stderr, "  Runs with errors:  %d/%d (%.0f%%)\n",
		et.RunsWithErrors, report.RunsAnalyzed,
		safePercent(et.RunsWithErrors, report.RunsAnalyzed))
	fmt.Fprintf(os.Stderr, "  Total errors:      %d (avg=%.2f/run)\n", et.TotalErrors, et.AvgErrorsPerRun)
	if et.TotalWarnings > 0 {
		fmt.Fprintf(os.Stderr, "  Total warnings:    %d (%d runs)\n", et.TotalWarnings, et.RunsWithWarnings)
	}
	fmt.Fprintln(os.Stderr)
}

func renderPrettyDomainInventory(report *CrossRunAuditReport) {
	if len(report.DomainInventory) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Domain Inventory (%d domains)", len(report.DomainInventory))))
	for _, entry := range report.DomainInventory {
		fmt.Fprintf(os.Stderr, "  %s %-45s  %s  seen=%d/%d  allowed=%d  blocked=%d\n",
			firewallStatusEmoji(entry.OverallStatus), entry.Domain, entry.OverallStatus,
			entry.SeenInRuns, report.RunsAnalyzed, entry.TotalAllowed, entry.TotalBlocked)
	}
	fmt.Fprintln(os.Stderr)
}

func renderPrettyDrain3Insights(insights []ObservabilityInsight) {
	if len(insights) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Agent Event Pattern Analysis (%d insights)", len(insights))))
	for _, insight := range insights {
		fmt.Fprintf(os.Stderr, "  %s [%s/%s] %s\n", renderSeverityIcon(insight.Severity), insight.Category, insight.Severity, insight.Title)
		fmt.Fprintf(os.Stderr, "     %s\n", insight.Summary)
		if insight.Evidence != "" {
			fmt.Fprintf(os.Stderr, "     evidence: %s\n", insight.Evidence)
		}
	}
	fmt.Fprintln(os.Stderr)
}

func renderSeverityIcon(severity string) string {
	switch severity {
	case "high":
		return "🔴"
	case "medium":
		return "🟠"
	case "low":
		return "🟡"
	default:
		return "ℹ"
	}
}

func renderPrettyPerRunBreakdown(runs []PerRunFirewallBreakdown) {
	if len(runs) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Per-Run Breakdown"))
	for _, run := range runs {
		fmt.Fprintln(os.Stderr, prettyPerRunLine(run))
	}
	fmt.Fprintln(os.Stderr)
}

func prettyPerRunLine(run PerRunFirewallBreakdown) string {
	prefix := fmt.Sprintf("  Run #%-12d  %-30s  %-10s", run.RunID, stringutil.Truncate(run.WorkflowName, 30), run.Conclusion)
	optional := prettyPerRunOptionalFields(run)
	if !run.HasData {
		return fmt.Sprintf("%s  (no firewall data)%s  mcp_errors=%d  errors=%d", prefix, optional, run.MCPErrors, run.ErrorCount)
	}
	return fmt.Sprintf("%s  requests=%d  allowed=%d  blocked=%d  deny=%.1f%%  domains=%d%s  turns=%d  mcp_errors=%d  errors=%d",
		prefix, run.TotalRequests, run.Allowed, run.Blocked, run.DenyRate*100,
		run.UniqueDomains, optional, run.Turns, run.MCPErrors, run.ErrorCount)
}

func prettyPerRunOptionalFields(run PerRunFirewallBreakdown) string {
	parts := strings.Builder{}
	if run.Duration > 0 {
		parts.WriteString("  dur=")
		parts.WriteString(timeutil.FormatDurationNs(int64(run.Duration)))
	}
	if run.Tokens > 0 {
		parts.WriteString("  tokens=")
		parts.WriteString(console.FormatTokens(run.Tokens))
		if run.TokenSpike {
			parts.WriteString("⚠")
		}
	}
	return parts.String()
}

func renderPrettyFinalStatus(report *CrossRunAuditReport) {
	if report.RunsWithData == 0 && len(report.MCPHealth) == 0 && report.MetricsTrend.TotalTokens == 0 {
		crossRunRenderLog.Printf("No data found in any analyzed runs: runs_analyzed=%d", report.RunsAnalyzed)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No data found in any of the analyzed runs."))
		return
	}

	parts := []string{fmt.Sprintf("%d runs analyzed", report.RunsAnalyzed)}
	if report.Summary.UniqueDomains > 0 {
		parts = append(parts, fmt.Sprintf("%d unique domains", report.Summary.UniqueDomains))
		parts = append(parts, fmt.Sprintf("%.1f%% overall denial rate", report.Summary.OverallDenyRate*100))
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Report complete: "+strings.Join(parts, ", ")))
}

// formatRunIDs formats a slice of run IDs as a comma-separated string.
func formatRunIDs(ids []int64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("#%d", id)
	}
	return strings.Join(parts, ", ")
}
