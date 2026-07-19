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
	"github.com/github/gh-aw/pkg/fileutil"
)

type logsDownloadRuntime struct {
	activeCtx       context.Context
	startTime       time.Time
	timeoutCancel   context.CancelFunc
	artifactFilter  []string
	fetchAllInRange bool
	filters         runFilterOpts
}

type workflowRunBatch struct {
	runs                   []WorkflowRun
	totalFetched           int
	batchSize              int
	oldestFetchedCreatedAt time.Time
}

type processWorkflowRunBatchOptions struct {
	count          int
	outputDir      string
	verbose        bool
	repoOverride   string
	artifactFilter []string
	evalsOnly      bool
	artifactSets   []string
	parse          bool
	filters        runFilterOpts
}

func prepareLogsDownload(ctx context.Context, opts LogsDownloadOptions) (logsDownloadRuntime, error) {
	logLogsDownloadStart(opts)
	artifactFilter, err := resolveLogsArtifactFilter(opts.ArtifactSets, opts.Verbose)
	if err != nil {
		return logsDownloadRuntime{}, err
	}
	if err := prepareLogsDownloadOutput(ctx, opts); err != nil {
		return logsDownloadRuntime{}, err
	}
	activeCtx, timeoutCancel, startTime := buildLogsDownloadContext(ctx, opts.TimeoutMinutes, opts.Verbose)
	return logsDownloadRuntime{
		activeCtx:       activeCtx,
		startTime:       startTime,
		timeoutCancel:   timeoutCancel,
		artifactFilter:  artifactFilter,
		fetchAllInRange: opts.StartDate != "" || opts.EndDate != "",
		filters: runFilterOpts{
			engine:            opts.Engine,
			noStaged:          opts.NoStaged,
			firewallOnly:      opts.FirewallOnly,
			noFirewall:        opts.NoFirewall,
			safeOutputType:    opts.SafeOutputType,
			filteredIntegrity: opts.FilteredIntegrity,
			evalsOnly:         opts.EvalsOnly,
		},
	}, nil
}

func logLogsDownloadStart(opts LogsDownloadOptions) {
	logsOrchestratorLog.Printf("Starting workflow log download: workflow=%s, count=%d, startDate=%s, endDate=%s, outputDir=%s, summaryFile=%s, safeOutputType=%s, filteredIntegrity=%v, evalsOnly=%v, train=%v, format=%s, artifactSets=%v, after=%s", opts.WorkflowName, opts.Count, opts.StartDate, opts.EndDate, opts.OutputDir, opts.SummaryFile, opts.SafeOutputType, opts.FilteredIntegrity, opts.EvalsOnly, opts.Train, opts.Format, opts.ArtifactSets, opts.After)
}

func resolveLogsArtifactFilter(artifactSets []string, verbose bool) ([]string, error) {
	if err := ValidateArtifactSets(artifactSets); err != nil {
		return nil, err
	}
	artifactFilter := ResolveArtifactFilter(artifactSets)
	if len(artifactFilter) > 0 {
		logsOrchestratorLog.Printf("Artifact filter active: %v", artifactFilter)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Artifact filter: downloading only "+strings.Join(artifactFilter, ", ")))
		}
	}
	return artifactFilter, nil
}

func prepareLogsDownloadOutput(ctx context.Context, opts LogsDownloadOptions) error {
	if err := ensureLogsGitignoreWithWarning(opts.Verbose); err != nil {
		return err
	}
	if err := checkLogsDownloadContext(ctx); err != nil {
		return err
	}
	if err := cleanupLogsOutputDir(opts); err != nil {
		return err
	}
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Fetching workflow runs from GitHub Actions..."))
	}
	return nil
}

func ensureLogsGitignoreWithWarning(verbose bool) error {
	if err := ensureLogsGitignore(); err != nil {
		logsOrchestratorLog.Printf("Failed to ensure logs .gitignore: %v", err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to ensure .github/aw/logs/.gitignore: %v", err)))
		}
	}
	return nil
}

func checkLogsDownloadContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
		return nil
	}
}

func cleanupLogsOutputDir(opts LogsDownloadOptions) error {
	if opts.After == "" {
		return nil
	}
	cutoff, err := parseCleanupCutoff(opts.After)
	if err != nil {
		return err
	}
	logsOrchestratorLog.Printf("Cleaning up run folders older than %s (cutoff: %s)", opts.After, cutoff.Format(time.RFC3339))
	removed, cleanErr := cleanupOldRunFolders(opts.OutputDir, cutoff, opts.Verbose)
	if cleanErr != nil {
		logsOrchestratorLog.Printf("Failed to clean up old run folders: %v", cleanErr)
		if !opts.JSONOutput {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to clean up old run folders: %v", cleanErr)))
		}
		return nil
	}
	if removed > 0 && !opts.JSONOutput {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Removed %d cached run folder(s) older than %s", removed, opts.After)))
	} else if removed == 0 && opts.Verbose && !opts.JSONOutput {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("No cached run folders older than %s found", opts.After)))
	}
	return nil
}

func buildLogsDownloadContext(ctx context.Context, timeoutMinutes int, verbose bool) (context.Context, context.CancelFunc, time.Time) {
	if timeoutMinutes <= 0 {
		return ctx, nil, time.Time{}
	}
	startTime := time.Now()
	activeCtx, timeoutCancel := context.WithTimeout(ctx, time.Duration(timeoutMinutes)*time.Minute)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Timeout set to %d minutes", timeoutMinutes)))
	}
	return activeCtx, timeoutCancel, startTime
}

func cancelLogsDownload(cancel context.CancelFunc) {
	if cancel != nil {
		cancel()
	}
}

func collectProcessedWorkflowRuns(runtime logsDownloadRuntime, opts LogsDownloadOptions) ([]ProcessedRun, bool, bool, error) {
	var processedRuns []ProcessedRun
	var beforeDate string
	var iteration int
	var timeoutReached, countLimitReached bool
	for iteration < MaxIterations {
		stop, timedOut, err := shouldStopLogsIteration(runtime, opts)
		if err != nil {
			return processedRuns, timeoutReached || timedOut, countLimitReached, err
		}
		if stop {
			timeoutReached = timeoutReached || timedOut
			break
		}
		if len(processedRuns) >= opts.Count {
			countLimitReached = runtime.fetchAllInRange
			break
		}
		if err := waitForLogsRateLimit(runtime.activeCtx, opts.Verbose, iteration); err != nil {
			continue
		}
		iteration++
		logLogsIterationFetch(opts, runtime.fetchAllInRange, iteration, len(processedRuns))
		batch, err := fetchWorkflowRunBatch(opts, beforeDate, len(processedRuns), runtime.fetchAllInRange)
		if err != nil {
			return nil, false, false, err
		}
		if len(batch.runs) == 0 {
			cursor, shouldContinue, shouldStop := handleEmptyWorkflowRunBatch(batch, opts.Verbose)
			if shouldStop {
				break
			}
			if shouldContinue {
				beforeDate = cursor
				continue
			}
		}
		logWorkflowRunBatchFound(batch, iteration, opts.Verbose)
		processedRuns, batchProcessed, allRunsConsumed, timedOut := processWorkflowRunBatch(runtime.activeCtx, batch, processedRuns, processWorkflowRunBatchOptions{
			count:          opts.Count,
			outputDir:      opts.OutputDir,
			verbose:        opts.Verbose,
			repoOverride:   opts.RepoOverride,
			artifactFilter: runtime.artifactFilter,
			evalsOnly:      opts.EvalsOnly,
			artifactSets:   opts.ArtifactSets,
			parse:          opts.Parse,
			filters:        runtime.filters,
		})
		timeoutReached = timeoutReached || timedOut
		logProcessedWorkflowRunBatch(opts, runtime.fetchAllInRange, iteration, batchProcessed, len(processedRuns), opts.Verbose)
		if allRunsConsumed {
			if cursor, ok := selectPaginationCursorDate(batch.runs, batch.oldestFetchedCreatedAt); ok {
				beforeDate = cursor
			}
		}
		if shouldStopAfterWorkflowRunBatch(batch, opts.Verbose) {
			break
		}
	}
	logLogsIterationLimit(runtime.fetchAllInRange, iteration, len(processedRuns), opts.Count)
	logLogsTimeoutResult(timeoutReached, len(processedRuns))
	return processedRuns, timeoutReached, countLimitReached, nil
}

func shouldStopLogsIteration(runtime logsDownloadRuntime, opts LogsDownloadOptions) (bool, bool, error) {
	select {
	case <-runtime.activeCtx.Done():
		if isDeadlineExceeded(runtime.activeCtx) {
			if opts.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Timeout reached, stopping download"))
			}
			return true, true, nil
		}
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return true, false, runtime.activeCtx.Err()
	default:
	}
	if opts.TimeoutMinutes > 0 && time.Since(runtime.startTime).Seconds() >= float64(opts.TimeoutMinutes)*60 {
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Timeout reached after %.1f seconds, stopping download", time.Since(runtime.startTime).Seconds())))
		}
		return true, true, nil
	}
	return false, false, nil
}

func waitForLogsRateLimit(ctx context.Context, verbose bool, iteration int) error {
	if iteration == 0 {
		return nil
	}
	if err := checkAndWaitForRateLimit(ctx, verbose); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		logsOrchestratorLog.Printf("Rate limit check failed (using static cooldown): %v", err)
	}
	return nil
}

func logLogsIterationFetch(opts LogsDownloadOptions, fetchAllInRange bool, iteration, processedCount int) {
	if !opts.Verbose || iteration <= 1 {
		return
	}
	if fetchAllInRange {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Iteration %d: Fetching more runs in date range...", iteration)))
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Iteration %d: Need %d more runs with artifacts, fetching more...", iteration, opts.Count-processedCount)))
}

func fetchWorkflowRunBatch(opts LogsDownloadOptions, beforeDate string, processedCount int, fetchAllInRange bool) (workflowRunBatch, error) {
	batchSize := computeLogsBatchSize(opts.WorkflowName, opts.Count, processedCount, fetchAllInRange)
	var oldestFetchedCreatedAt time.Time
	runs, totalFetched, err := listWorkflowRunsWithPagination(ListWorkflowRunsOptions{
		WorkflowName:           opts.WorkflowName,
		Limit:                  batchSize,
		StartDate:              opts.StartDate,
		EndDate:                opts.EndDate,
		BeforeDate:             beforeDate,
		Ref:                    opts.Ref,
		BeforeRunID:            opts.BeforeRunID,
		AfterRunID:             opts.AfterRunID,
		RepoOverride:           opts.RepoOverride,
		OldestFetchedCreatedAt: &oldestFetchedCreatedAt,
		ProcessedCount:         processedCount,
		TargetCount:            opts.Count,
		Verbose:                opts.Verbose,
	})
	return workflowRunBatch{runs: runs, totalFetched: totalFetched, batchSize: batchSize, oldestFetchedCreatedAt: oldestFetchedCreatedAt}, err
}

func computeLogsBatchSize(workflowName string, count, processedCount int, fetchAllInRange bool) int {
	batchSize := BatchSize
	if workflowName == "" {
		batchSize = BatchSizeForAllWorkflows
	}
	if fetchAllInRange || count-processedCount >= batchSize {
		return batchSize
	}
	needed := count - processedCount
	batchSize = needed * 3
	if workflowName == "" && batchSize < BatchSizeForAllWorkflows {
		batchSize = BatchSizeForAllWorkflows
	}
	if batchSize > BatchSizeForAllWorkflows {
		batchSize = BatchSizeForAllWorkflows
	}
	return batchSize
}

func handleEmptyWorkflowRunBatch(batch workflowRunBatch, verbose bool) (string, bool, bool) {
	if len(batch.runs) > 0 {
		return "", false, false
	}
	if shouldStopPagination(batch.totalFetched, batch.batchSize) {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No more workflow runs found, stopping iteration"))
		}
		return "", false, true
	}
	cursor, ok := selectPaginationCursorDate(nil, batch.oldestFetchedCreatedAt)
	if !ok {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Workflow batch filtered to zero runs but no pagination cursor was found, stopping iteration"))
		}
		return "", false, true
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Batch filtered to zero runs; advancing pagination cursor and continuing"))
	}
	return cursor, true, false
}

func logWorkflowRunBatchFound(batch workflowRunBatch, iteration int, verbose bool) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d workflow runs in batch %d", len(batch.runs), iteration)))
	}
}

func processWorkflowRunBatch(
	activeCtx context.Context,
	batch workflowRunBatch,
	processedRuns []ProcessedRun,
	opts processWorkflowRunBatchOptions,
) ([]ProcessedRun, int, bool, bool) {
	runsRemaining := batch.runs
	batchProcessed := 0
	for len(runsRemaining) > 0 && len(processedRuns) < opts.count {
		remainingNeeded := opts.count - len(processedRuns)
		if remainingNeeded <= 0 {
			break
		}
		if stop, timedOut := batchContextDone(activeCtx); stop {
			return processedRuns, batchProcessed, false, timedOut
		}
		chunk := nextWorkflowRunChunk(&runsRemaining, remainingNeeded)
		processedRuns, batchProcessed = appendProcessedWorkflowRuns(activeCtx, processedRuns, chunk, batchProcessed, opts)
		if len(processedRuns) >= opts.count {
			break
		}
	}
	return processedRuns, batchProcessed, len(runsRemaining) == 0, false
}

func batchContextDone(ctx context.Context) (bool, bool) {
	select {
	case <-ctx.Done():
		return true, isDeadlineExceeded(ctx)
	default:
		return false, false
	}
}

func nextWorkflowRunChunk(runsRemaining *[]WorkflowRun, remainingNeeded int) []WorkflowRun {
	chunkSize := min(max(remainingNeeded*3, remainingNeeded), len(*runsRemaining))
	chunk := (*runsRemaining)[:chunkSize]
	*runsRemaining = (*runsRemaining)[chunkSize:]
	return chunk
}

func appendProcessedWorkflowRuns(
	activeCtx context.Context,
	processedRuns []ProcessedRun,
	chunk []WorkflowRun,
	batchProcessed int,
	opts processWorkflowRunBatchOptions,
) ([]ProcessedRun, int) {
	downloadResults := downloadRunArtifactsConcurrent(activeCtx, chunk, runArtifactsConcurrentOptions{
		outputDir:      opts.outputDir,
		verbose:        opts.verbose,
		maxRuns:        opts.count - len(processedRuns),
		repoOverride:   opts.repoOverride,
		artifactFilter: opts.artifactFilter,
		evalsOnly:      opts.evalsOnly,
		artifactSets:   opts.artifactSets,
	})
	for _, result := range downloadResults {
		if shouldSkipProcessedWorkflowRun(result, opts.verbose) || applyRunFilters(activeCtx, result, opts.filters, opts.verbose) {
			continue
		}
		processedRun := buildProcessedRun(result, opts.verbose, true)
		parseWorkflowRunArtifacts(result, processedRun, opts.parse, opts.verbose)
		processedRuns = append(processedRuns, processedRun)
		batchProcessed++
		if len(processedRuns) >= opts.count {
			break
		}
	}
	return processedRuns, batchProcessed
}

func shouldSkipProcessedWorkflowRun(result DownloadResult, verbose bool) bool {
	if result.Skipped {
		if verbose && result.Error != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping run %d: %v", result.Run.DatabaseID, result.Error)))
		}
		return true
	}
	if result.Error != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to download artifacts for run %d: %v", result.Run.DatabaseID, result.Error)))
		return true
	}
	return false
}

func parseWorkflowRunArtifacts(result DownloadResult, processedRun ProcessedRun, parse, verbose bool) {
	if !parse {
		return
	}
	awInfoPath := filepath.Join(result.LogsPath, "aw_info.json")
	detectedEngine := extractEngineFromAwInfo(awInfoPath, verbose)
	if err := parseAgentLog(result.LogsPath, detectedEngine, verbose); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse log for run %d: %v", processedRun.Run.DatabaseID, err)))
	} else if logMdPath := filepath.Join(result.LogsPath, "log.md"); fileutil.FileExists(logMdPath) {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Parsed log for run %d → %s", processedRun.Run.DatabaseID, logMdPath)))
	}
	if err := parseFirewallLogs(result.LogsPath, verbose); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse firewall logs for run %d: %v", processedRun.Run.DatabaseID, err)))
	} else if firewallMdPath := filepath.Join(result.LogsPath, "firewall.md"); fileutil.FileExists(firewallMdPath) {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Parsed firewall logs for run %d → %s", processedRun.Run.DatabaseID, firewallMdPath)))
	}
}

func logProcessedWorkflowRunBatch(opts LogsDownloadOptions, fetchAllInRange bool, iteration, batchProcessed, processedCount int, verbose bool) {
	if !verbose {
		return
	}
	if fetchAllInRange {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Processed %d runs with artifacts in batch %d (total: %d)", batchProcessed, iteration, processedCount)))
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Processed %d runs with artifacts in batch %d (total: %d/%d)", batchProcessed, iteration, processedCount, opts.Count)))
}

func shouldStopAfterWorkflowRunBatch(batch workflowRunBatch, verbose bool) bool {
	if !shouldStopPagination(batch.totalFetched, batch.batchSize) {
		return false
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Received fewer runs than requested, likely reached end of available runs"))
	}
	return true
}

func logLogsIterationLimit(fetchAllInRange bool, iteration, processedCount, count int) {
	if iteration < MaxIterations {
		return
	}
	if fetchAllInRange {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Reached maximum iterations (%d), collected %d runs with artifacts", MaxIterations, processedCount)))
		return
	}
	if processedCount < count {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Reached maximum iterations (%d), collected %d runs with artifacts out of %d requested", MaxIterations, processedCount, count)))
	}
}

func logLogsTimeoutResult(timeoutReached bool, processedCount int) {
	if timeoutReached && processedCount > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Timeout reached, returning %d processed runs", processedCount)))
	}
}

func handleEmptyProcessedRuns(processedRuns []ProcessedRun, opts LogsDownloadOptions, timeoutReached bool) (bool, error) {
	if len(processedRuns) > 0 {
		return false, nil
	}
	if opts.JSONOutput {
		logsData := buildLogsData([]ProcessedRun{}, opts.OutputDir, nil)
		logsData.Message = noRunsMessage(opts.StartDate, timeoutReached)
		if err := renderLogsJSON(logsData, opts.Verbose); err != nil {
			return true, fmt.Errorf("failed to render JSON output: %w", err)
		}
	}
	if timeoutReached {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Timeout reached before any runs could be downloaded"))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No workflow runs with artifacts found matching the specified criteria"))
	}
	return true, nil
}

func limitProcessedRuns(processedRuns []ProcessedRun, count int, verbose bool) []ProcessedRun {
	if len(processedRuns) <= count {
		return processedRuns
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Limiting output to %d most recent runs (fetched %d total)", count, len(processedRuns))))
	}
	return processedRuns[:count]
}
