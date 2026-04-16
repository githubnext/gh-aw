package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/timeutil"
)

// renderCreatedItemsTable renders the list of items created in GitHub by safe output handlers
// as a table with clickable URLs for easy auditing.
func renderCreatedItemsTable(items []CreatedItemReport) {
	auditReportLog.Printf("Rendering created items table with %d item(s)", len(items))
	config := console.TableConfig{
		Headers: []string{"Type", "Repo", "Number", "Temp ID", "URL"},
		Rows:    make([][]string, 0, len(items)),
	}

	for _, item := range items {
		numberStr := ""
		if item.Number > 0 {
			numberStr = strconv.Itoa(item.Number)
		}

		row := []string{
			item.Type,
			item.Repo,
			numberStr,
			item.TemporaryID,
			item.URL,
		}
		config.Rows = append(config.Rows, row)
	}

	fmt.Fprint(os.Stderr, console.RenderTable(config))
	fmt.Fprintln(os.Stderr)
}

// renderKeyFindings renders key findings with colored severity indicators
func renderKeyFindings(findings []Finding) {
	auditReportLog.Printf("Rendering key findings: total=%d", len(findings))
	// Group findings by severity for better presentation
	critical := sliceutil.Filter(findings, func(f Finding) bool { return f.Severity == "critical" })
	high := sliceutil.Filter(findings, func(f Finding) bool { return f.Severity == "high" })
	medium := sliceutil.Filter(findings, func(f Finding) bool { return f.Severity == "medium" })
	low := sliceutil.Filter(findings, func(f Finding) bool { return f.Severity == "low" })
	info := sliceutil.Filter(findings, func(f Finding) bool {
		return f.Severity != "critical" && f.Severity != "high" && f.Severity != "medium" && f.Severity != "low"
	})

	// Render critical findings first
	for _, finding := range critical {
		fmt.Fprintf(os.Stderr, "  🔴 %s [%s]\n", console.FormatErrorMessage(finding.Title), finding.Category)
		fmt.Fprintf(os.Stderr, "     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Fprintf(os.Stderr, "     Impact: %s\n", finding.Impact)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Then high severity
	for _, finding := range high {
		fmt.Fprintf(os.Stderr, "  🟠 %s [%s]\n", console.FormatWarningMessage(finding.Title), finding.Category)
		fmt.Fprintf(os.Stderr, "     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Fprintf(os.Stderr, "     Impact: %s\n", finding.Impact)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Medium severity
	for _, finding := range medium {
		fmt.Fprintf(os.Stderr, "  🟡 %s [%s]\n", finding.Title, finding.Category)
		fmt.Fprintf(os.Stderr, "     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Fprintf(os.Stderr, "     Impact: %s\n", finding.Impact)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Low severity
	for _, finding := range low {
		fmt.Fprintf(os.Stderr, "  ℹ️  %s [%s]\n", finding.Title, finding.Category)
		fmt.Fprintf(os.Stderr, "     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Fprintf(os.Stderr, "     Impact: %s\n", finding.Impact)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Info findings
	for _, finding := range info {
		fmt.Fprintf(os.Stderr, "  ✅ %s [%s]\n", console.FormatSuccessMessage(finding.Title), finding.Category)
		fmt.Fprintf(os.Stderr, "     %s\n", finding.Description)
		if finding.Impact != "" {
			fmt.Fprintf(os.Stderr, "     Impact: %s\n", finding.Impact)
		}
		fmt.Fprintln(os.Stderr)
	}
}

// renderRecommendations renders actionable recommendations
func renderRecommendations(recommendations []Recommendation) {
	auditReportLog.Printf("Rendering recommendations: total=%d", len(recommendations))
	// Group by priority
	high := sliceutil.Filter(recommendations, func(r Recommendation) bool { return r.Priority == "high" })
	medium := sliceutil.Filter(recommendations, func(r Recommendation) bool { return r.Priority == "medium" })
	low := sliceutil.Filter(recommendations, func(r Recommendation) bool { return r.Priority != "high" && r.Priority != "medium" })

	// Render high priority first
	for i, rec := range high {
		fmt.Fprintf(os.Stderr, "  %d. [HIGH] %s\n", i+1, console.FormatWarningMessage(rec.Action))
		fmt.Fprintf(os.Stderr, "     Reason: %s\n", rec.Reason)
		if rec.Example != "" {
			fmt.Fprintf(os.Stderr, "     Example: %s\n", rec.Example)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Medium priority
	startIdx := len(high) + 1
	for i, rec := range medium {
		fmt.Fprintf(os.Stderr, "  %d. [MEDIUM] %s\n", startIdx+i, rec.Action)
		fmt.Fprintf(os.Stderr, "     Reason: %s\n", rec.Reason)
		if rec.Example != "" {
			fmt.Fprintf(os.Stderr, "     Example: %s\n", rec.Example)
		}
		fmt.Fprintln(os.Stderr)
	}

	// Low priority
	startIdx += len(medium)
	for i, rec := range low {
		fmt.Fprintf(os.Stderr, "  %d. [LOW] %s\n", startIdx+i, rec.Action)
		fmt.Fprintf(os.Stderr, "     Reason: %s\n", rec.Reason)
		if rec.Example != "" {
			fmt.Fprintf(os.Stderr, "     Example: %s\n", rec.Example)
		}
		fmt.Fprintln(os.Stderr)
	}
}

// renderSafeOutputSummary renders safe output summary with type breakdown
func renderSafeOutputSummary(summary *SafeOutputSummary) {
	if summary == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "  Total Items:       %d\n", summary.TotalItems)
	fmt.Fprintf(os.Stderr, "  Summary:           %s\n", summary.Summary)
	fmt.Fprintln(os.Stderr)

	// Type breakdown table
	if len(summary.TypeDetails) > 0 {
		config := console.TableConfig{
			Headers: []string{"Type", "Count"},
			Rows:    make([][]string, 0, len(summary.TypeDetails)),
		}
		for _, detail := range summary.TypeDetails {
			row := []string{detail.Type, strconv.Itoa(detail.Count)}
			config.Rows = append(config.Rows, row)
		}
		fmt.Fprint(os.Stderr, console.RenderTable(config))
		fmt.Fprintln(os.Stderr)
	}
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

// renderErrorsAndWarnings renders the errors and warnings section
func renderErrorsAndWarnings(errors []ErrorInfo, warnings []ErrorInfo) {
	if len(errors) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Errors (%d):", len(errors))))
		for _, err := range errors {
			if err.File != "" && err.Line > 0 {
				fmt.Fprintf(os.Stderr, "    %s:%d: %s\n", filepath.Base(err.File), err.Line, err.Message)
			} else {
				fmt.Fprintf(os.Stderr, "    %s\n", err.Message)
			}
		}
		fmt.Fprintln(os.Stderr)
	}

	if len(warnings) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warnings (%d):", len(warnings))))
		for _, warn := range warnings {
			if warn.File != "" && warn.Line > 0 {
				fmt.Fprintf(os.Stderr, "    %s:%d: %s\n", filepath.Base(warn.File), warn.Line, warn.Message)
			} else {
				fmt.Fprintf(os.Stderr, "    %s\n", warn.Message)
			}
		}
		fmt.Fprintln(os.Stderr)
	}
}
