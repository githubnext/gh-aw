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
	fmt.Println("# Audit Report — Cross-Run Analysis")
	fmt.Println()

	// Executive summary
	fmt.Println("## Executive Summary")
	fmt.Println()
	fmt.Printf("| Metric | Value |\n")
	fmt.Printf("|--------|-------|\n")
	fmt.Printf("| Runs analyzed | %d |\n", report.RunsAnalyzed)
	fmt.Printf("| Runs with firewall data | %d |\n", report.RunsWithData)
	fmt.Printf("| Runs without firewall data | %d |\n", report.RunsWithoutData)
	fmt.Printf("| Total requests | %d |\n", report.Summary.TotalRequests)
	fmt.Printf("| Allowed requests | %d |\n", report.Summary.TotalAllowed)
	fmt.Printf("| Blocked requests | %d |\n", report.Summary.TotalBlocked)
	fmt.Printf("| Overall denial rate | %.1f%% |\n", report.Summary.OverallDenyRate*100)
	fmt.Printf("| Unique domains | %d |\n", report.Summary.UniqueDomains)
	fmt.Println()

	// Metrics trends
	mt := report.MetricsTrend
	if mt.RunsWithCost > 0 || mt.TotalTokens > 0 {
		fmt.Println("## Metrics Trends")
		fmt.Println()
		if mt.RunsWithCost > 0 {
			fmt.Printf("**Cost Trend** (%d runs with cost data)\n\n", mt.RunsWithCost)
			fmt.Printf("| Total | Avg/run | Min | Max |\n")
			fmt.Printf("|-------|---------|-----|-----|\n")
			fmt.Printf("| $%.4f | $%.4f | $%.4f | $%.4f |\n", mt.TotalCost, mt.AvgCost, mt.MinCost, mt.MaxCost)
			if len(mt.CostSpikes) > 0 {
				fmt.Printf("\n⚠ Cost spikes (>2x avg) in runs: %s\n", formatRunIDs(mt.CostSpikes))
			}
			fmt.Println()
		}
		if mt.TotalTokens > 0 {
			fmt.Printf("**Token Trend**\n\n")
			fmt.Printf("| Total | Avg/run | Min | Max |\n")
			fmt.Printf("|-------|---------|-----|-----|\n")
			fmt.Printf("| %d | %d | %d | %d |\n", mt.TotalTokens, mt.AvgTokens, mt.MinTokens, mt.MaxTokens)
			if len(mt.TokenSpikes) > 0 {
				fmt.Printf("\n⚠ Token spikes (>2x avg) in runs: %s\n", formatRunIDs(mt.TokenSpikes))
			}
			fmt.Println()
		}
		if mt.TotalTurns > 0 {
			fmt.Printf("**Turn Trend**\n\n")
			fmt.Printf("| Total | Avg/run | Max |\n")
			fmt.Printf("|-------|---------|-----|\n")
			fmt.Printf("| %d | %.1f | %d |\n", mt.TotalTurns, mt.AvgTurns, mt.MaxTurns)
			fmt.Println()
		}
		if mt.AvgDurationNs > 0 {
			fmt.Printf("**Duration Trend**\n\n")
			fmt.Printf("| Avg | Min | Max |\n")
			fmt.Printf("|-----|-----|-----|\n")
			fmt.Printf("| %s | %s | %s |\n",
				timeutil.FormatDurationNs(mt.AvgDurationNs),
				timeutil.FormatDurationNs(mt.MinDurationNs),
				timeutil.FormatDurationNs(mt.MaxDurationNs))
			fmt.Println()
		}
	}

	// MCP health
	if len(report.MCPHealth) > 0 {
		fmt.Printf("## MCP Server Health (%d runs)\n\n", report.RunsAnalyzed)
		fmt.Printf("| Server | Connected | Error Rate | Total Calls | Errors | Status |\n")
		fmt.Printf("|--------|-----------|------------|-------------|--------|--------|\n")
		for _, h := range report.MCPHealth {
			status := "✅ ok"
			if h.Unreliable {
				status = "⚠ unreliable"
			}
			fmt.Printf("| `%s` | %d/%d | %.1f%% | %d | %d | %s |\n",
				h.ServerName, h.RunsConnected, h.TotalRuns,
				h.ErrorRate*100, h.TotalCalls, h.TotalErrors, status)
		}
		fmt.Println()
	}

	// Error trend
	et := report.ErrorTrend
	if et.TotalErrors > 0 || et.TotalWarnings > 0 {
		fmt.Println("## Error Trend")
		fmt.Println()
		fmt.Printf("| Metric | Value |\n")
		fmt.Printf("|--------|-------|\n")
		fmt.Printf("| Runs with errors | %d/%d (%.0f%%) |\n",
			et.RunsWithErrors, report.RunsAnalyzed,
			safePercent(et.RunsWithErrors, report.RunsAnalyzed))
		fmt.Printf("| Total errors | %d |\n", et.TotalErrors)
		fmt.Printf("| Avg errors/run | %.2f |\n", et.AvgErrorsPerRun)
		if et.TotalWarnings > 0 {
			fmt.Printf("| Runs with warnings | %d/%d |\n", et.RunsWithWarnings, report.RunsAnalyzed)
			fmt.Printf("| Total warnings | %d |\n", et.TotalWarnings)
		}
		fmt.Println()
	}

	// Domain inventory
	if len(report.DomainInventory) > 0 {
		fmt.Println("## Domain Inventory")
		fmt.Println()
		fmt.Printf("| Domain | Status | Seen In | Allowed | Blocked |\n")
		fmt.Printf("|--------|--------|---------|---------|--------|\n")
		for _, entry := range report.DomainInventory {
			icon := statusEmoji(entry.OverallStatus)
			fmt.Printf("| `%s` | %s %s | %d/%d runs | %d | %d |\n",
				entry.Domain, icon, entry.OverallStatus, entry.SeenInRuns, report.RunsAnalyzed,
				entry.TotalAllowed, entry.TotalBlocked)
		}
		fmt.Println()
	}

	// Drain3 insights
	if len(report.Drain3Insights) > 0 {
		fmt.Println("## Agent Event Pattern Analysis")
		fmt.Println()
		for _, insight := range report.Drain3Insights {
			severityIcon := "ℹ"
			switch insight.Severity {
			case "high":
				severityIcon = "🔴"
			case "medium":
				severityIcon = "🟠"
			case "low":
				severityIcon = "🟡"
			}
			fmt.Printf("### %s %s\n\n", severityIcon, insight.Title)
			fmt.Printf("**Category:** %s | **Severity:** %s\n\n", insight.Category, insight.Severity)
			fmt.Printf("%s\n\n", insight.Summary)
			if insight.Evidence != "" {
				fmt.Printf("_Evidence:_ `%s`\n\n", insight.Evidence)
			}
		}
	}

	// Per-run breakdown
	if len(report.PerRunBreakdown) > 0 {
		fmt.Println("## Per-Run Breakdown")
		fmt.Println()
		fmt.Printf("| Run ID | Workflow | Conclusion | Duration | Firewall | Cost | Tokens | Turns | MCP Err | Errors |\n")
		fmt.Printf("|--------|----------|------------|----------|----------|------|--------|-------|---------|--------|\n")
		for _, run := range report.PerRunBreakdown {
			firewallCol := "—"
			if run.HasData {
				firewallCol = fmt.Sprintf("%d/%d", run.Allowed, run.Blocked)
			}
			costStr := "—"
			if run.Cost > 0 {
				costStr = fmt.Sprintf("$%.4f", run.Cost)
				if run.CostSpike {
					costStr += " ⚠"
				}
			}
			tokenStr := "—"
			if run.Tokens > 0 {
				tokenStr = formatTokens(run.Tokens)
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
			fmt.Printf("| %d | %s | %s | %s | %s | %s | %s | %s | %d | %d |\n",
				run.RunID, run.WorkflowName, run.Conclusion, durStr,
				firewallCol, costStr, tokenStr, turnsStr,
				run.MCPErrors, run.ErrorCount)
		}
		fmt.Println()
	}
}

// renderCrossRunReportPretty outputs the cross-run report as formatted console output to stderr.
func renderCrossRunReportPretty(report *CrossRunAuditReport) {
	crossRunRenderLog.Printf("Rendering cross-run report as pretty output: runs_analyzed=%d, runs_with_data=%d, deny_rate=%.1f%%",
		report.RunsAnalyzed, report.RunsWithData, report.Summary.OverallDenyRate*100)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Audit Report — Cross-Run Analysis"))
	fmt.Fprintln(os.Stderr)

	// Executive summary
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Executive Summary"))
	fmt.Fprintf(os.Stderr, "  Runs analyzed:              %d\n", report.RunsAnalyzed)
	fmt.Fprintf(os.Stderr, "  Runs with firewall data:    %d\n", report.RunsWithData)
	fmt.Fprintf(os.Stderr, "  Runs without firewall data: %d\n", report.RunsWithoutData)
	fmt.Fprintf(os.Stderr, "  Total requests:             %d\n", report.Summary.TotalRequests)
	fmt.Fprintf(os.Stderr, "  Allowed / Blocked:          %d / %d\n", report.Summary.TotalAllowed, report.Summary.TotalBlocked)
	fmt.Fprintf(os.Stderr, "  Overall denial rate:        %.1f%%\n", report.Summary.OverallDenyRate*100)
	fmt.Fprintf(os.Stderr, "  Unique domains:             %d\n", report.Summary.UniqueDomains)
	fmt.Fprintln(os.Stderr)

	// Metrics trends
	mt := report.MetricsTrend
	if mt.RunsWithCost > 0 || mt.TotalTokens > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Metrics Trends"))
		if mt.RunsWithCost > 0 {
			spikeNote := ""
			if len(mt.CostSpikes) > 0 {
				spikeNote = fmt.Sprintf("  ⚠ Cost spikes in runs: %s\n", formatRunIDs(mt.CostSpikes))
			}
			fmt.Fprintf(os.Stderr, "  Cost:     total=$%.4f  avg=$%.4f/run  min=$%.4f  max=$%.4f\n%s",
				mt.TotalCost, mt.AvgCost, mt.MinCost, mt.MaxCost, spikeNote)
		}
		if mt.TotalTokens > 0 {
			spikeNote := ""
			if len(mt.TokenSpikes) > 0 {
				spikeNote = fmt.Sprintf("  ⚠ Token spikes in runs: %s\n", formatRunIDs(mt.TokenSpikes))
			}
			fmt.Fprintf(os.Stderr, "  Tokens:   total=%s  avg=%s/run  min=%s  max=%s\n%s",
				formatTokens(mt.TotalTokens), formatTokens(mt.AvgTokens),
				formatTokens(mt.MinTokens), formatTokens(mt.MaxTokens), spikeNote)
		}
		if mt.TotalTurns > 0 {
			fmt.Fprintf(os.Stderr, "  Turns:    total=%d  avg=%.1f/run  max=%d\n",
				mt.TotalTurns, mt.AvgTurns, mt.MaxTurns)
		}
		if mt.AvgDurationNs > 0 {
			fmt.Fprintf(os.Stderr, "  Duration: avg=%s  min=%s  max=%s\n",
				timeutil.FormatDurationNs(mt.AvgDurationNs),
				timeutil.FormatDurationNs(mt.MinDurationNs),
				timeutil.FormatDurationNs(mt.MaxDurationNs))
		}
		fmt.Fprintln(os.Stderr)
	}

	// MCP health
	if len(report.MCPHealth) > 0 {
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

	// Error trend
	et := report.ErrorTrend
	if et.TotalErrors > 0 || et.TotalWarnings > 0 {
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

	// Domain inventory
	if len(report.DomainInventory) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Domain Inventory (%d domains)", len(report.DomainInventory))))
		for _, entry := range report.DomainInventory {
			icon := statusEmoji(entry.OverallStatus)
			fmt.Fprintf(os.Stderr, "  %s %-45s  %s  seen=%d/%d  allowed=%d  blocked=%d\n",
				icon, entry.Domain, entry.OverallStatus, entry.SeenInRuns, report.RunsAnalyzed,
				entry.TotalAllowed, entry.TotalBlocked)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Drain3 insights
	if len(report.Drain3Insights) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Agent Event Pattern Analysis (%d insights)", len(report.Drain3Insights))))
		for _, insight := range report.Drain3Insights {
			severityIcon := "ℹ"
			switch insight.Severity {
			case "high":
				severityIcon = "🔴"
			case "medium":
				severityIcon = "🟠"
			case "low":
				severityIcon = "🟡"
			}
			fmt.Fprintf(os.Stderr, "  %s [%s/%s] %s\n", severityIcon, insight.Category, insight.Severity, insight.Title)
			fmt.Fprintf(os.Stderr, "     %s\n", insight.Summary)
			if insight.Evidence != "" {
				fmt.Fprintf(os.Stderr, "     evidence: %s\n", insight.Evidence)
			}
		}
		fmt.Fprintln(os.Stderr)
	}

	// Per-run breakdown
	if len(report.PerRunBreakdown) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Per-Run Breakdown"))
		for _, run := range report.PerRunBreakdown {
			costStr := ""
			if run.Cost > 0 {
				costStr = fmt.Sprintf("  cost=$%.4f", run.Cost)
				if run.CostSpike {
					costStr += "⚠"
				}
			}
			tokenStr := ""
			if run.Tokens > 0 {
				tokenStr = "  tokens=" + formatTokens(run.Tokens)
				if run.TokenSpike {
					tokenStr += "⚠"
				}
			}
			durStr := ""
			if run.Duration > 0 {
				durStr = "  dur=" + timeutil.FormatDurationNs(int64(run.Duration))
			}
			if !run.HasData {
				fmt.Fprintf(os.Stderr, "  Run #%-12d  %-30s  %-10s  (no firewall data)%s%s%s  mcp_errors=%d  errors=%d\n",
					run.RunID, stringutil.Truncate(run.WorkflowName, 30), run.Conclusion,
					durStr, costStr, tokenStr, run.MCPErrors, run.ErrorCount)
				continue
			}
			fmt.Fprintf(os.Stderr, "  Run #%-12d  %-30s  %-10s  requests=%d  allowed=%d  blocked=%d  deny=%.1f%%  domains=%d%s%s%s  turns=%d  mcp_errors=%d  errors=%d\n",
				run.RunID, stringutil.Truncate(run.WorkflowName, 30), run.Conclusion,
				run.TotalRequests, run.Allowed, run.Blocked,
				run.DenyRate*100, run.UniqueDomains,
				durStr, costStr, tokenStr,
				run.Turns, run.MCPErrors, run.ErrorCount)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Final status
	if report.RunsWithData == 0 && len(report.MCPHealth) == 0 && report.MetricsTrend.TotalTokens == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No data found in any of the analyzed runs."))
	} else {
		parts := []string{
			fmt.Sprintf("%d runs analyzed", report.RunsAnalyzed),
		}
		if report.Summary.UniqueDomains > 0 {
			parts = append(parts, fmt.Sprintf("%d unique domains", report.Summary.UniqueDomains))
			parts = append(parts, fmt.Sprintf("%.1f%% overall denial rate", report.Summary.OverallDenyRate*100))
		}
		if report.MetricsTrend.RunsWithCost > 0 {
			parts = append(parts, fmt.Sprintf("total cost $%.4f", report.MetricsTrend.TotalCost))
		}
		if len(report.MetricsTrend.CostSpikes) > 0 {
			parts = append(parts, fmt.Sprintf("%d cost spike(s)", len(report.MetricsTrend.CostSpikes)))
		}
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Report complete: "+strings.Join(parts, ", ")))
	}
}

// formatRunIDs formats a slice of run IDs as a comma-separated string.
func formatRunIDs(ids []int64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("#%d", id)
	}
	return strings.Join(parts, ", ")
}

// safePercent returns percentage of part/total, returning 0 when total is 0.
func safePercent(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}
