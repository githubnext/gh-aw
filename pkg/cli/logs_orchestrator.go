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
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/envutil"
	"github.com/github/gh-aw/pkg/fileutil"
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
	workflowName := opts.WorkflowName
	count := opts.Count
	startDate := opts.StartDate
	endDate := opts.EndDate
	outputDir := opts.OutputDir
	engine := opts.Engine
	ref := opts.Ref
	beforeRunID := opts.BeforeRunID
	afterRunID := opts.AfterRunID
	repoOverride := opts.RepoOverride
	verbose := opts.Verbose
	toolGraph := opts.ToolGraph
	noStaged := opts.NoStaged
	firewallOnly := opts.FirewallOnly
	noFirewall := opts.NoFirewall
	parse := opts.Parse
	jsonOutput := opts.JSONOutput
	timeoutMinutes := opts.TimeoutMinutes
	summaryFile := opts.SummaryFile
	safeOutputType := opts.SafeOutputType
	filteredIntegrity := opts.FilteredIntegrity
	evalsOnly := opts.EvalsOnly
	train := opts.Train
	format := opts.Format
	artifactSets := opts.ArtifactSets
	after := opts.After

	logsOrchestratorLog.Printf("Starting workflow log download: workflow=%s, count=%d, startDate=%s, endDate=%s, outputDir=%s, summaryFile=%s, safeOutputType=%s, filteredIntegrity=%v, evalsOnly=%v, train=%v, format=%s, artifactSets=%v, after=%s", workflowName, count, startDate, endDate, outputDir, summaryFile, safeOutputType, filteredIntegrity, evalsOnly, train, format, artifactSets, after)

	// Validate and resolve artifact sets into a concrete filter (list of artifact base names).
	if err := ValidateArtifactSets(artifactSets); err != nil {
		return err
	}
	artifactFilter := ResolveArtifactFilter(artifactSets)
	if len(artifactFilter) > 0 {
		logsOrchestratorLog.Printf("Artifact filter active: %v", artifactFilter)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Artifact filter: downloading only "+strings.Join(artifactFilter, ", ")))
		}
	}

	// Ensure .github/aw/logs/.gitignore exists on every invocation
	if err := ensureLogsGitignore(); err != nil {
		// Log but don't fail - this is not critical for downloading logs
		logsOrchestratorLog.Printf("Failed to ensure logs .gitignore: %v", err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to ensure .github/aw/logs/.gitignore: %v", err)))
		}
	}

	// Check context cancellation at the start
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
	}

	// Clean up cached run folders older than the --after cutoff, if specified.
	// Runs after the context check so a cancelled context never triggers disk scanning.
	if after != "" {
		cutoff, parseErr := parseCleanupCutoff(after)
		if parseErr != nil {
			return parseErr
		}
		logsOrchestratorLog.Printf("Cleaning up run folders older than %s (cutoff: %s)", after, cutoff.Format(time.RFC3339))
		removed, cleanErr := cleanupOldRunFolders(outputDir, cutoff, verbose)
		if cleanErr != nil {
			// Non-fatal: log but continue with download
			logsOrchestratorLog.Printf("Failed to clean up old run folders: %v", cleanErr)
			if !jsonOutput {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to clean up old run folders: %v", cleanErr)))
			}
		} else if removed > 0 {
			if !jsonOutput {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Removed %d cached run folder(s) older than %s", removed, after)))
			}
		} else if verbose && !jsonOutput {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("No cached run folders older than %s found", after)))
		}
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Fetching workflow runs from GitHub Actions..."))
	}

	// activeCtx is ctx extended with a deadline when timeoutMinutes > 0.
	// Using a named variable avoids reassigning the ctx parameter and makes it
	// explicit that a derived context governs all downstream downloads.
	activeCtx := ctx
	var startTime time.Time
	var timeoutReached bool
	if timeoutMinutes > 0 {
		startTime = time.Now()
		var timeoutCancel context.CancelFunc
		activeCtx, timeoutCancel = context.WithTimeout(ctx, time.Duration(timeoutMinutes)*time.Minute)
		defer timeoutCancel()
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Timeout set to %d minutes", timeoutMinutes)))
		}
	}

	var processedRuns []ProcessedRun
	var beforeDate string
	iteration := 0

	// Determine if we should fetch all runs in the date window or limit iteratively by count.
	// In fetchAllInRange mode (when date filters are specified), count acts as a page size:
	// the loop stops once count runs are collected and a continuation cursor is emitted so
	// callers can page through the full date window.
	// Without date filters, we fetch up to count runs with artifacts and stop (old behavior
	// for backward compatibility).
	fetchAllInRange := startDate != "" || endDate != ""

	// countLimitReached is set when the loop exits because len(processedRuns) >= count in
	// fetchAllInRange mode.  It signals that more runs may be available in the date window
	// and drives continuation-data generation so callers can page through the full range.
	var countLimitReached bool

	filters := runFilterOpts{
		engine:            engine,
		noStaged:          noStaged,
		firewallOnly:      firewallOnly,
		noFirewall:        noFirewall,
		safeOutputType:    safeOutputType,
		filteredIntegrity: filteredIntegrity,
		evalsOnly:         evalsOnly,
	}

	// Iterative algorithm: keep fetching runs until we have enough or exhaust available runs
outerLoop:
	for iteration < MaxIterations {
		// Check context cancellation or timeout deadline
		select {
		case <-activeCtx.Done():
			if isDeadlineExceeded(activeCtx) {
				// Our own timeout context expired — treat this as a graceful stop,
				// not a hard error.  break outerLoop falls through to renderLogsOutput
				// which outputs whatever processedRuns were collected before the deadline.
				timeoutReached = true
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Timeout reached, stopping download"))
				}
			} else {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
				return activeCtx.Err()
			}
			break outerLoop
		default:
		}

		// Check timeout if specified
		if timeoutMinutes > 0 {
			elapsed := time.Since(startTime).Seconds()
			if elapsed >= float64(timeoutMinutes)*60 {
				timeoutReached = true
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Timeout reached after %.1f seconds, stopping download", elapsed)))
				}
				break
			}
		}

		// Stop if we've collected enough processed runs.
		// In fetchAllInRange mode, record that we hit the count limit so the caller
		// can paginate to retrieve runs that fall outside this batch.
		if len(processedRuns) >= count {
			if fetchAllInRange {
				countLimitReached = true
			}
			break
		}

		// Query the GitHub API rate limit before each iteration (except the first)
		// and wait as needed.  This replaces the static cooldown sleep: the helper
		// always sleeps at least APICallCooldown but will also block until the
		// reset window when the remaining budget is nearly exhausted.
		if iteration > 0 {
			if rlErr := checkAndWaitForRateLimit(activeCtx, verbose); rlErr != nil {
				if errors.Is(rlErr, context.Canceled) || errors.Is(rlErr, context.DeadlineExceeded) {
					// Context was cancelled or timed out during the rate-limit wait.
					// Use continue (not break) so the top-of-loop activeCtx.Done() path
					// preserves Canceled vs DeadlineExceeded behavior.
					continue
				}
				logsOrchestratorLog.Printf("Rate limit check failed (using static cooldown): %v", rlErr)
			}
		}

		iteration++

		if verbose && iteration > 1 {
			if fetchAllInRange {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Iteration %d: Fetching more runs in date range...", iteration)))
			} else {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Iteration %d: Need %d more runs with artifacts, fetching more...", iteration, count-len(processedRuns))))
			}
		}

		// Fetch a batch of runs
		batchSize := BatchSize
		if workflowName == "" {
			// When searching for all agentic workflows, use a larger batch size
			// since there may be many CI runs interspersed with agentic runs
			batchSize = BatchSizeForAllWorkflows
		}

		// When not fetching all in range, optimize batch size based on how many we still need
		if !fetchAllInRange && count-len(processedRuns) < batchSize {
			// If we need fewer runs than the batch size, request exactly what we need
			// but add some buffer since many runs might not have artifacts
			needed := count - len(processedRuns)
			batchSize = needed * 3 // Request 3x what we need to account for runs without artifacts
			if workflowName == "" && batchSize < BatchSizeForAllWorkflows {
				// For all-workflows search, maintain a minimum batch size
				batchSize = BatchSizeForAllWorkflows
			}
			if batchSize > BatchSizeForAllWorkflows {
				batchSize = BatchSizeForAllWorkflows
			}
		}

		var oldestFetchedCreatedAt time.Time
		runs, totalFetched, err := listWorkflowRunsWithPagination(ListWorkflowRunsOptions{
			WorkflowName:           workflowName,
			Limit:                  batchSize,
			StartDate:              startDate,
			EndDate:                endDate,
			BeforeDate:             beforeDate,
			Ref:                    ref,
			BeforeRunID:            beforeRunID,
			AfterRunID:             afterRunID,
			RepoOverride:           repoOverride,
			OldestFetchedCreatedAt: &oldestFetchedCreatedAt,
			ProcessedCount:         len(processedRuns),
			TargetCount:            count,
			Verbose:                verbose,
		})
		if err != nil {
			return err
		}

		if len(runs) == 0 {
			if shouldStopPagination(totalFetched, batchSize) {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No more workflow runs found, stopping iteration"))
				}
				break
			}

			cursor, ok := selectPaginationCursorDate(nil, oldestFetchedCreatedAt)
			if !ok {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Workflow batch filtered to zero runs but no pagination cursor was found, stopping iteration"))
				}
				break
			}

			beforeDate = cursor
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Batch filtered to zero runs; advancing pagination cursor and continuing"))
			}
			continue
		}

		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d workflow runs in batch %d", len(runs), iteration)))
		}

		// Process runs in chunks so cache hits can satisfy the count without
		// forcing us to scan the entire batch.
		batchProcessed := 0
		runsRemaining := runs
	innerLoop:
		for len(runsRemaining) > 0 && len(processedRuns) < count {
			remainingNeeded := count - len(processedRuns)
			if remainingNeeded <= 0 {
				break
			}

			// Check context/timeout before starting each new chunk so we stop
			// promptly when the deadline fires between individual chunk downloads.
			select {
			case <-activeCtx.Done():
				if isDeadlineExceeded(activeCtx) {
					timeoutReached = true
				}
				break innerLoop
			default:
			}

			// Process slightly more than we need to account for skips due to filters.
			chunkSize := min(max(remainingNeeded*3, remainingNeeded), len(runsRemaining))

			chunk := runsRemaining[:chunkSize]
			runsRemaining = runsRemaining[chunkSize:]

			downloadResults := downloadRunArtifactsConcurrent(activeCtx, chunk, outputDir, verbose, remainingNeeded, repoOverride, artifactFilter)

			for _, result := range downloadResults {
				if result.Skipped {
					if verbose {
						if result.Error != nil {
							fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping run %d: %v", result.Run.DatabaseID, result.Error)))
						}
					}
					continue
				}

				if result.Error != nil {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to download artifacts for run %d: %v", result.Run.DatabaseID, result.Error)))
					continue
				}

				if applyRunFilters(activeCtx, result, filters, verbose) {
					continue
				}

				processedRun := buildProcessedRun(result, verbose, true)

				// If --parse flag is set, parse the agent log and write to log.md
				if parse {
					// Get the engine from aw_info.json
					awInfoPath := filepath.Join(result.LogsPath, "aw_info.json")
					detectedEngine := extractEngineFromAwInfo(awInfoPath, verbose)

					if err := parseAgentLog(result.LogsPath, detectedEngine, verbose); err != nil {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse log for run %d: %v", processedRun.Run.DatabaseID, err)))
					} else {
						// Always show success message for parsing, not just in verbose mode
						logMdPath := filepath.Join(result.LogsPath, "log.md")
						if fileutil.FileExists(logMdPath) {
							fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Parsed log for run %d → %s", processedRun.Run.DatabaseID, logMdPath)))
						}
					}

					// Also parse firewall logs if they exist
					if err := parseFirewallLogs(result.LogsPath, verbose); err != nil {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse firewall logs for run %d: %v", processedRun.Run.DatabaseID, err)))
					} else {
						// Show success message if firewall.md was created
						firewallMdPath := filepath.Join(result.LogsPath, "firewall.md")
						if fileutil.FileExists(firewallMdPath) {
							fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Parsed firewall logs for run %d → %s", processedRun.Run.DatabaseID, firewallMdPath)))
						}
					}
				}

				processedRuns = append(processedRuns, processedRun)
				batchProcessed++

				// Stop processing this batch once we've collected enough runs.
				if len(processedRuns) >= count {
					break
				}
			}
		}

		if verbose {
			if fetchAllInRange {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Processed %d runs with artifacts in batch %d (total: %d)", batchProcessed, iteration, len(processedRuns))))
			} else {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Processed %d runs with artifacts in batch %d (total: %d/%d)", batchProcessed, iteration, len(processedRuns), count)))
			}
		}

		// Prepare for next iteration: set beforeDate to the oldest run from the raw API batch.
		// This guarantees pagination moves forward even when filtered runs are sparse.
		if len(runsRemaining) == 0 {
			if cursor, ok := selectPaginationCursorDate(runs, oldestFetchedCreatedAt); ok {
				beforeDate = cursor
			}
		}

		// If we got fewer runs than requested in this batch, we've likely hit the end
		// IMPORTANT: Use totalFetched (API response size before filtering) not len(runs) (after filtering)
		// to detect end. When workflowName is empty, runs are filtered to only agentic workflows,
		// so len(runs) may be much smaller than totalFetched even when more data is available from GitHub.
		// Example: API returns 250 total runs, but only 5 are agentic workflows after filtering.
		//   Old buggy logic: len(runs)=5 < batchSize=250, stop iteration (WRONG - misses more agentic workflows!)
		//   Fixed logic: totalFetched=250 < batchSize=250 is false, continue iteration (CORRECT)
		if shouldStopPagination(totalFetched, batchSize) {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Received fewer runs than requested, likely reached end of available runs"))
			}
			break
		}
	}

	// Check if we hit the maximum iterations limit
	if iteration >= MaxIterations {
		if fetchAllInRange {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Reached maximum iterations (%d), collected %d runs with artifacts", MaxIterations, len(processedRuns))))
		} else if len(processedRuns) < count {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Reached maximum iterations (%d), collected %d runs with artifacts out of %d requested", MaxIterations, len(processedRuns), count)))
		}
	}

	// Report if timeout was reached
	if timeoutReached && len(processedRuns) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Timeout reached, returning %d processed runs", len(processedRuns))))
	}

	if len(processedRuns) == 0 {
		// When JSON output is requested, output JSON first to stdout before any stderr messages
		// This prevents stderr messages from corrupting JSON when both streams are redirected together
		if jsonOutput {
			logsData := buildLogsData([]ProcessedRun{}, outputDir, nil)
			logsData.Message = noRunsMessage(startDate, timeoutReached)
			if err := renderLogsJSON(logsData, verbose); err != nil {
				return fmt.Errorf("failed to render JSON output: %w", err)
			}
		}
		// Now print warning messages to stderr after JSON output (if any) is complete
		if timeoutReached {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Timeout reached before any runs could be downloaded"))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No workflow runs with artifacts found matching the specified criteria"))
		}
		return nil
	}

	// Apply count limit to final results (truncate to count if we fetched more)
	if len(processedRuns) > count {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Limiting output to %d most recent runs (fetched %d total)", count, len(processedRuns))))
		}
		processedRuns = processedRuns[:count]
	}

	// Build continuation data if timeout was reached and there are processed runs,
	// OR if a date-range fetch hit the count limit (more runs may exist in the window).
	continuation := buildContinuationIfNeeded(processedRuns, timeoutReached, countLimitReached, continuationOptions{
		workflowName:   workflowName,
		startDate:      startDate,
		endDate:        endDate,
		engine:         engine,
		branch:         ref,
		afterRunID:     afterRunID,
		count:          count,
		timeoutMinutes: timeoutMinutes,
	})

	return renderLogsOutput(processedRuns, renderLogsOutputOptions{
		outputDir:      outputDir,
		summaryFile:    summaryFile,
		format:         format,
		reportFile:     opts.ReportFile,
		jsonOutput:     jsonOutput,
		toolGraph:      toolGraph,
		train:          train,
		continuation:   continuation,
		verbose:        verbose,
		artifactFilter: artifactFilter,
	})
}
