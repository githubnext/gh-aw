package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/timeutil"
)

// renderMCPServerHealth renders MCP server health summary
func renderMCPServerHealth(health *MCPServerHealth) {
	if health == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "  %s\n", health.Summary)
	if health.TotalRequests > 0 {
		fmt.Fprintf(os.Stderr, "  Total Requests:    %d\n", health.TotalRequests)
		fmt.Fprintf(os.Stderr, "  Total Errors:      %d\n", health.TotalErrors)
		fmt.Fprintf(os.Stderr, "  Error Rate:        %.1f%%\n", health.ErrorRate)
	}
	fmt.Fprintln(os.Stderr)

	// Server health table
	if len(health.Servers) > 0 {
		config := console.TableConfig{
			Headers: []string{"Server", "Requests", "Tool Calls", "Errors", "Error Rate", "Avg Latency", "Status"},
			Rows:    make([][]string, 0, len(health.Servers)),
		}
		for _, server := range health.Servers {
			row := []string{
				server.ServerName,
				strconv.Itoa(server.RequestCount),
				strconv.Itoa(server.ToolCalls),
				strconv.Itoa(server.ErrorCount),
				server.ErrorRateStr,
				server.AvgLatency,
				server.Status,
			}
			config.Rows = append(config.Rows, row)
		}
		fmt.Fprint(os.Stderr, console.RenderTable(config))
	}

	// Slowest tool calls
	if len(health.SlowestCalls) > 0 {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "  Slowest Tool Calls:")
		config := console.TableConfig{
			Headers: []string{"Server", "Tool", "Duration"},
			Rows:    make([][]string, 0, len(health.SlowestCalls)),
		}
		for _, call := range health.SlowestCalls {
			row := []string{call.ServerName, call.ToolName, call.Duration}
			config.Rows = append(config.Rows, row)
		}
		fmt.Fprint(os.Stderr, console.RenderTable(config))
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
