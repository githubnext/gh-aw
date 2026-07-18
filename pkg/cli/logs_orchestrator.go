// This file provides command-line interface functionality for gh-aw.
// This file (logs_orchestrator.go) contains the main orchestration logic for downloading
// and processing workflow logs from GitHub Actions.
//
// Key responsibilities:
//   - Coordinating the main download workflow (DownloadWorkflowLogs)
//   - Managing pagination and iteration through workflow runs
//   - Applying filters (engine, firewall, staged, etc.)
//   - Building and rendering output (console, JSON, tool graphs)

package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/github/gh-aw/pkg/envutil"
	"github.com/github/gh-aw/pkg/logger"
)

var logsOrchestratorLog = logger.New("cli:logs_orchestrator")

// isDeadlineExceeded reports whether ctx.Err() is context.DeadlineExceeded,
// returning false for any other error (including nil).  It is used to
// distinguish our own timeout cancellation (graceful partial results) from a
// user-initiated cancellation or other error.
func isDeadlineExceeded(ctx context.Context) bool {
	// errors.Is handles nil gracefully (returns false), so no nil check needed.
	return errors.Is(ctx.Err(), context.DeadlineExceeded)
}

// applyMetricsTurnsToRun sets run.Turns from metrics when a log-derived count is
// available. It deliberately does NOT overwrite when metrics.Turns is zero so that
// a backfilled value from applyUsageActivitySummaryToResult (session.turns) is
// preserved for usage-only artifact downloads where events.jsonl/.log are absent.
func applyMetricsTurnsToRun(run *WorkflowRun, metrics LogMetrics) {
	if metrics.Turns > 0 {
		run.Turns = metrics.Turns
	}
}

// noRunsMessage returns a human-readable explanation for why zero workflow runs
// were returned.  It inspects the startDate filter and the timeoutReached flag
// so callers receive actionable guidance instead of a silent empty result.
//
// Priority order (timeout is checked first because it is the most definitive
// cause — the date filter may still be valid but no data was collected):
//  1. Timeout – the download was cut short before any run was collected.
//  2. Future start date – GitHub cannot have runs in the future.
//  3. Start date older than GitHubActionsRetentionDays – beyond GitHub's default retention window.
//  4. Generic fallback for any other combination of filters.
func noRunsMessage(startDate string, timeoutReached bool) string {
	if timeoutReached {
		return "No runs found. Timeout reached before any runs could be downloaded."
	}
	if startDate != "" {
		if t, err := parseFilterDate(startDate); err == nil {
			now := time.Now()
			if t.After(now) {
				return fmt.Sprintf("No runs found. The start_date %q is in the future.", startDate)
			}
			// GitHub Actions retains logs for GitHubActionsRetentionDays by default.
			if t.Before(now.AddDate(0, 0, -GitHubActionsRetentionDays)) {
				return fmt.Sprintf("No runs found. Data may not be available beyond the %d-day retention period.", GitHubActionsRetentionDays)
			}
		}
	}
	return "No runs found matching the specified criteria."
}

// parseFilterDate tries to parse a date or datetime string in the formats used
// by the logs command's --start-date / --end-date flags after date resolution.
// Both plain dates ("2006-01-02") and RFC 3339 timestamps are accepted.
func parseFilterDate(s string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date %q", s)
}

// It reads from the GH_AW_MAX_CONCURRENT_DOWNLOADS environment variable if set,
// validates the value is between 1 and 100, and falls back to the default if invalid.
func getMaxConcurrentDownloads() int {
	return envutil.GetIntFromEnv("GH_AW_MAX_CONCURRENT_DOWNLOADS", MaxConcurrentDownloads, 1, 100, logsOrchestratorLog)
}

func shouldStopPagination(totalFetched, batchSize int) bool {
	return totalFetched < batchSize
}

func selectPaginationCursorDate(filteredRuns []WorkflowRun, oldestFetchedCreatedAt time.Time) (string, bool) {
	if !oldestFetchedCreatedAt.IsZero() {
		return oldestFetchedCreatedAt.Format(time.RFC3339), true
	}
	if len(filteredRuns) == 0 {
		return "", false
	}
	return filteredRuns[len(filteredRuns)-1].CreatedAt.Format(time.RFC3339), true
}

// buildContinuationIfNeeded returns a ContinuationData cursor when more runs may
// be available after this batch, or nil if the full result set was collected.
//
// A continuation is emitted in two cases:
//   - timeoutReached: the caller's timeout expired mid-download; runs beyond the
//     deadline were not fetched and may still exist.
//   - countLimitReached: in fetchAllInRange mode the count cap was hit before the
//     date window was exhausted; the next page starts just before the oldest run
//     returned in this batch.
func buildContinuationIfNeeded(
	processedRuns []ProcessedRun,
	timeoutReached, countLimitReached bool,
	opts continuationOptions,
) *ContinuationData {
	if len(processedRuns) == 0 || (!timeoutReached && !countLimitReached) {
		return nil
	}
	// Use the oldest processed run as the before_run_id cursor for the next page.
	oldestRunID := processedRuns[len(processedRuns)-1].Run.DatabaseID
	logsOrchestratorLog.Printf("Building continuation cursor: before_run_id=%d, timeoutReached=%v, countLimitReached=%v", oldestRunID, timeoutReached, countLimitReached)
	message := "Timeout reached. Use these parameters to continue fetching more logs."
	if countLimitReached {
		// In fetchAllInRange mode the date window may contain more runs than count.
		message = "Count limit reached. Use these parameters to continue fetching more logs from the same date range."
	}
	return &ContinuationData{
		Message:      message,
		WorkflowName: opts.workflowName,
		Count:        opts.count,
		StartDate:    opts.startDate,
		EndDate:      opts.endDate,
		Engine:       opts.engine,
		Branch:       opts.branch,
		AfterRunID:   opts.afterRunID,
		BeforeRunID:  oldestRunID,
		Timeout:      opts.timeoutMinutes,
	}
}

// DownloadWorkflowLogs downloads and analyzes workflow logs with metrics
func DownloadWorkflowLogs(ctx context.Context, opts LogsDownloadOptions) error {
	logsOrchestratorLog.Printf("Downloading workflow logs: workflow=%q, count=%d, outputDir=%q", opts.WorkflowName, opts.Count, opts.OutputDir)
	runtime, err := prepareLogsDownload(ctx, opts)
	if err != nil {
		return err
	}
	defer cancelLogsDownload(runtime.timeoutCancel)

	processedRuns, timeoutReached, countLimitReached, err := collectProcessedWorkflowRuns(runtime, opts)
	if err != nil {
		return err
	}
	if handled, err := handleEmptyProcessedRuns(processedRuns, opts, timeoutReached); handled || err != nil {
		logsOrchestratorLog.Printf("No processed runs to render (timeoutReached=%v, err=%v)", timeoutReached, err)
		return err
	}

	processedRuns = limitProcessedRuns(processedRuns, opts.Count, opts.Verbose)
	logsOrchestratorLog.Printf("Collected %d processed runs (timeoutReached=%v, countLimitReached=%v)", len(processedRuns), timeoutReached, countLimitReached)
	continuation := buildContinuationIfNeeded(processedRuns, timeoutReached, countLimitReached, continuationOptions{
		workflowName:   opts.WorkflowName,
		startDate:      opts.StartDate,
		endDate:        opts.EndDate,
		engine:         opts.Engine,
		branch:         opts.Ref,
		afterRunID:     opts.AfterRunID,
		count:          opts.Count,
		timeoutMinutes: opts.TimeoutMinutes,
	})

	return renderLogsOutput(processedRuns, renderLogsOutputOptions{
		outputDir:      opts.OutputDir,
		summaryFile:    opts.SummaryFile,
		format:         opts.Format,
		reportFile:     opts.ReportFile,
		jsonOutput:     opts.JSONOutput,
		toolGraph:      opts.ToolGraph,
		train:          opts.Train,
		continuation:   continuation,
		verbose:        opts.Verbose,
		artifactFilter: runtime.artifactFilter,
	})
}
