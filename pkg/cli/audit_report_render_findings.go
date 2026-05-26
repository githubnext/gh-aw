package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/timeutil"
)

// renderTokenUsage displays token usage data from the firewall proxy
func renderTokenUsage(summary *TokenUsageSummary) {
	fmt.Fprintf(os.Stderr, "  Tokens:     %s input, %s output, %s cache read, %s cache write\n",
		console.FormatNumber(summary.TotalInputTokens),
		console.FormatNumber(summary.TotalOutputTokens),
		console.FormatNumber(summary.TotalCacheReadTokens),
		console.FormatNumber(summary.TotalCacheWriteTokens))
	fmt.Fprintf(os.Stderr, "  Requests:   %d (avg %s)\n",
		summary.TotalRequests, timeutil.FormatDurationMs(summary.AvgDurationMs()))
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
