// This file provides command-line interface functionality for gh-aw.
// This file (audit_report_render_performance.go) contains console rendering functions
// for performance metrics, token usage, and GitHub rate limit sections of the audit report.

package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/timeutil"
)

// renderPerformanceMetrics renders performance metrics
func renderPerformanceMetrics(metrics *PerformanceMetrics) {
	auditReportLog.Printf("Rendering performance metrics: tokens_per_min=%.1f, cost_efficiency=%s, most_used_tool=%s",
		metrics.TokensPerMinute, metrics.CostEfficiency, metrics.MostUsedTool)
	if metrics.TokensPerMinute > 0 {
		fmt.Fprintf(os.Stderr, "  Tokens per Minute: %.1f\n", metrics.TokensPerMinute)
	}

	if metrics.CostEfficiency != "" {
		efficiencyDisplay := metrics.CostEfficiency
		switch metrics.CostEfficiency {
		case "excellent", "good":
			efficiencyDisplay = console.FormatSuccessMessage(metrics.CostEfficiency)
		case "moderate":
			efficiencyDisplay = console.FormatWarningMessage(metrics.CostEfficiency)
		case "poor":
			efficiencyDisplay = console.FormatErrorMessage(metrics.CostEfficiency)
		}
		fmt.Fprintf(os.Stderr, "  Cost Efficiency: %s\n", efficiencyDisplay)
	}

	if metrics.AvgToolDuration != "" {
		fmt.Fprintf(os.Stderr, "  Average Tool Duration: %s\n", metrics.AvgToolDuration)
	}

	if metrics.MostUsedTool != "" {
		fmt.Fprintf(os.Stderr, "  Most Used Tool: %s\n", metrics.MostUsedTool)
	}

	if metrics.NetworkRequests > 0 {
		fmt.Fprintf(os.Stderr, "  Network Requests: %d\n", metrics.NetworkRequests)
	}

	fmt.Fprintln(os.Stderr)
}

// renderTokenUsage displays token usage data from the firewall proxy
func renderTokenUsage(summary *TokenUsageSummary) {
	totalTokens := summary.TotalTokens()
	cacheTokens := summary.TotalCacheReadTokens + summary.TotalCacheWriteTokens

	fmt.Fprintf(os.Stderr, "  Total:      %s tokens (%s input, %s output, %s cache)\n",
		console.FormatNumber(totalTokens),
		console.FormatNumber(summary.TotalInputTokens),
		console.FormatNumber(summary.TotalOutputTokens),
		console.FormatNumber(cacheTokens))
	fmt.Fprintf(os.Stderr, "  Requests:   %d (avg %s)\n",
		summary.TotalRequests, timeutil.FormatDurationMs(summary.AvgDurationMs()))
	if summary.CacheEfficiency > 0 {
		fmt.Fprintf(os.Stderr, "  Cache hit:  %.1f%%\n", summary.CacheEfficiency*100)
	}
	fmt.Fprintln(os.Stderr)

	rows := summary.ModelRows()
	if len(rows) > 0 {
		config := console.TableConfig{
			Headers: []string{"Model", "Provider", "Input", "Output", "Cache Read", "Cache Write", "Requests", "Avg Duration"},
			Rows:    make([][]string, 0, len(rows)),
		}
		for _, row := range rows {
			config.Rows = append(config.Rows, []string{
				row.Model,
				row.Provider,
				console.FormatNumber(row.InputTokens),
				console.FormatNumber(row.OutputTokens),
				console.FormatNumber(row.CacheReadTokens),
				console.FormatNumber(row.CacheWriteTokens),
				strconv.Itoa(row.Requests),
				row.AvgDuration,
			})
		}
		fmt.Fprint(os.Stderr, console.RenderTable(config))
		fmt.Fprintln(os.Stderr)
	}
}

// renderGitHubRateLimitUsage displays GitHub API quota consumption for the run.
func renderGitHubRateLimitUsage(usage *GitHubRateLimitUsage) {
	if usage == nil {
		return
	}

	// Summary line
	summary := "Total GitHub API calls: " + console.FormatNumber(usage.TotalRequestsMade)
	if usage.CoreLimit > 0 {
		summary += fmt.Sprintf("  |  Core quota consumed: %s / %s  (remaining: %s)",
			console.FormatNumber(usage.CoreConsumed),
			console.FormatNumber(usage.CoreLimit),
			console.FormatNumber(usage.CoreRemaining),
		)
	}
	fmt.Fprintf(os.Stderr, "  %s\n\n", summary)

	// Per-resource breakdown table (only when there are multiple resources or non-core resources)
	rows := usage.ResourceRows()
	if len(rows) == 0 {
		return
	}
	cfg := console.TableConfig{
		Headers: []string{"Resource", "API Calls", "Quota Consumed", "Remaining", "Limit"},
		Rows:    make([][]string, 0, len(rows)),
	}
	for _, row := range rows {
		cfg.Rows = append(cfg.Rows, []string{
			row.Resource,
			console.FormatNumber(row.RequestsMade),
			console.FormatNumber(row.QuotaConsumed),
			console.FormatNumber(row.FinalRemaining),
			console.FormatNumber(row.Limit),
		})
	}
	fmt.Fprint(os.Stderr, console.RenderTable(cfg))
	fmt.Fprintln(os.Stderr)
}
