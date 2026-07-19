package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

var logsTSVLog = logger.New("cli:logs_format_tsv")

func formatTSVSummaryTokens(totalTokens int) string {
	if totalTokens == 0 {
		return "n/a"
	}
	return strconv.Itoa(totalTokens)
}

// renderLogsTSV outputs the logs data as tab-separated values for maximum token efficiency.
// This format is ~24x more compact than JSON, making it ideal for agentic consumption
// where LLM context window tokens are the primary constraint.
//
// Output format:
//
//	Line 1: Summary line (total_runs, total_duration, total_tokens, total_turns, total_errors)
//	Line 2: Column headers
//	Lines 3+: One line per run with tab-separated fields
func renderLogsTSV(data LogsData) {
	logsTSVLog.Printf("Rendering %d runs as TSV", data.Summary.TotalRuns)

	s := data.Summary
	// Summary line with key aggregates
	fmt.Fprintf(os.Stdout, "# %d runs | %s duration | %s tokens | %d turns | %d errors\n",
		s.TotalRuns, s.TotalDuration, formatTSVSummaryTokens(s.TotalTokens), s.TotalTurns, s.TotalErrors)

	if len(data.Runs) == 0 {
		return
	}

	// Header
	headers := []string{
		"run_id", "workflow", "engine", "status", "duration",
		"tokens", "aic", "turns", "errors",
		"event", "branch", "created_at", "classification", "url",
	}
	fmt.Fprintln(os.Stdout, strings.Join(headers, "\t"))

	for _, r := range data.Runs {
		fmt.Fprintln(os.Stdout, strings.Join(renderLogsTSVFields(r), "\t"))
	}

	renderLogsTSVAppendices(data)
}

func renderLogsTSVFields(r RunData) []string {
	conclusion, duration, classification := renderLogsTSVRunStrings(r)
	url := r.URL
	if idx := strings.Index(url, "/actions/runs/"); idx > 0 {
		url = url[idx:]
	}
	return []string{
		strconv.FormatInt(r.RunID, 10),
		r.WorkflowName,
		r.EngineID,
		conclusion,
		duration,
		strconv.Itoa(r.TokenUsage),
		fmt.Sprintf("%.3f", r.AIC),
		strconv.Itoa(r.Turns),
		strconv.Itoa(r.ErrorCount),
		r.Event,
		r.Branch,
		r.CreatedAt.Format("2006-01-02T15:04"),
		classification,
		url,
	}
}

func renderLogsTSVRunStrings(r RunData) (string, string, string) {
	conclusion := r.Conclusion
	if conclusion == "" {
		conclusion = r.Status
	}
	duration := r.Duration
	if duration == "" {
		duration = "-"
	}
	classification := r.Classification
	if classification == "" {
		classification = "-"
	}
	return conclusion, duration, classification
}

func renderLogsTSVAppendices(data LogsData) {
	if len(data.Observability) > 0 {
		fmt.Fprintln(os.Stdout, "# insights:")
		for _, obs := range data.Observability {
			fmt.Fprintf(os.Stdout, "# [%s] %s: %s\n", obs.Severity, obs.Title, obs.Summary)
		}
	}
	if data.FirewallLog != nil && data.FirewallLog.TotalRequests > 0 {
		fmt.Fprintf(os.Stdout, "# firewall: %d requests (%d allowed, %d blocked)\n",
			data.FirewallLog.TotalRequests, data.FirewallLog.AllowedRequests, data.FirewallLog.BlockedRequests)
	}
	if len(data.Summary.EngineCounts) > 1 {
		parts := make([]string, 0, len(data.Summary.EngineCounts))
		for engine, count := range data.Summary.EngineCounts {
			parts = append(parts, fmt.Sprintf("%s:%d", engine, count))
		}
		fmt.Fprintf(os.Stdout, "# engines: %s\n", strings.Join(parts, " "))
	}
}

// renderLogsTSVVerbose outputs a more detailed TSV with additional columns for audit use.
func renderLogsTSVVerbose(data LogsData) {
	logsTSVLog.Printf("Rendering %d runs as verbose TSV", data.Summary.TotalRuns)

	s := data.Summary
	fmt.Fprintf(os.Stdout, "# %d runs | %s duration | %s tokens | %d turns | %d errors | %d missing_tools | %d github_api_calls\n",
		s.TotalRuns, s.TotalDuration, formatTSVSummaryTokens(s.TotalTokens), s.TotalTurns, s.TotalErrors, s.TotalMissingTools, s.TotalGitHubAPICalls)

	if len(data.Runs) == 0 {
		return
	}

	headers := []string{
		"run_id", "workflow", "engine", "status", "duration",
		"tokens", "aic", "turns", "errors",
		"warnings", "missing_tools", "missing_data", "github_api",
		"event", "branch", "actor", "created_at", "tbt",
		"classification", "action_min", "display_title", "url",
	}
	fmt.Fprintln(os.Stdout, strings.Join(headers, "\t"))

	for _, r := range data.Runs {
		fmt.Fprintln(os.Stdout, strings.Join(renderLogsTSVVerboseFields(r), "\t"))
	}

	renderLogsTSVVerboseAppendices(data)
}

func renderLogsTSVVerboseFields(r RunData) []string {
	conclusion, duration, classification := renderLogsTSVRunStrings(r)
	tbt := r.AvgTimeBetweenTurns
	if tbt == "" {
		tbt = "-"
	}
	displayTitle := stringutil.Truncate(r.DisplayTitle, 50)
	return []string{
		strconv.FormatInt(r.RunID, 10),
		r.WorkflowName,
		r.EngineID,
		conclusion,
		duration,
		strconv.Itoa(r.TokenUsage),
		fmt.Sprintf("%.3f", r.AIC),
		strconv.Itoa(r.Turns),
		strconv.Itoa(r.ErrorCount),
		strconv.Itoa(r.WarningCount),
		strconv.Itoa(r.MissingToolCount),
		strconv.Itoa(r.MissingDataCount),
		strconv.Itoa(r.GitHubAPICalls),
		r.Event,
		r.Branch,
		r.Actor,
		r.CreatedAt.Format("2006-01-02T15:04"),
		tbt,
		classification,
		fmt.Sprintf("%.0f", r.ActionMinutes),
		displayTitle,
		r.URL,
	}
}

func renderLogsTSVVerboseAppendices(data LogsData) {
	if len(data.Observability) == 0 {
		return
	}
	fmt.Fprintln(os.Stdout, "# insights:")
	for _, obs := range data.Observability {
		fmt.Fprintf(os.Stdout, "# [%s] %s: %s\n", obs.Severity, obs.Title, obs.Summary)
	}
}
