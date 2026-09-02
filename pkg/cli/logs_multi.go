package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/stringutil"
)

type logsWorkflowTarget struct {
	workflowName string
	repoOverride string
}

type logsTargetResult struct {
	target logsWorkflowTarget
	result workflowLogsResult
	err    error
}

var collectWorkflowLogsForTarget = collectWorkflowLogs

// DownloadWorkflowLogsForTargets downloads several workflow reports concurrently
// and renders one combined report. Each target gets an isolated output directory
// so run IDs from different repositories cannot collide in the local cache.
func DownloadWorkflowLogsForTargets(
	ctx context.Context,
	opts LogsDownloadOptions,
	targets []logsWorkflowTarget,
	initialErrors []error,
) error {
	if len(targets) == 0 {
		return errors.Join(initialErrors...)
	}
	if err := ensureLogsGitignoreWithWarning(opts.Verbose); err != nil {
		return err
	}

	results := collectLogsTargets(ctx, opts, targets)
	processedRuns, continuations, timeoutReached, allErrors := mergeLogsTargetResults(results, initialErrors)
	for _, err := range allErrors {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Skipping workflow target: "+err.Error()))
	}
	if len(processedRuns) == 0 {
		if len(allErrors) > 0 {
			return errors.Join(allErrors...)
		}
		_, err := handleEmptyProcessedRuns(nil, opts, timeoutReached)
		return err
	}

	slices.SortStableFunc(processedRuns, func(a, b ProcessedRun) int {
		return b.Run.CreatedAt.Compare(a.Run.CreatedAt)
	})
	artifactFilter, err := resolveLogsArtifactFilter(opts.ArtifactSets, opts.Verbose)
	if err != nil {
		return err
	}
	return renderLogsOutput(processedRuns, renderLogsOutputOptions{
		outputDir:      opts.OutputDir,
		summaryFile:    opts.SummaryFile,
		format:         opts.Format,
		reportFile:     opts.ReportFile,
		jsonOutput:     opts.JSONOutput,
		toolGraph:      opts.ToolGraph,
		train:          opts.Train,
		verbose:        opts.Verbose,
		artifactFilter: artifactFilter,
		startDate:      opts.StartDate,
		endDate:        opts.EndDate,
		checkStaleness: true,
		suppressRender: opts.SuppressRender,
		continuations:  continuations,
	})
}

func collectLogsTargets(ctx context.Context, opts LogsDownloadOptions, targets []logsWorkflowTarget) <-chan logsTargetResult {
	results := make(chan logsTargetResult, len(targets))
	var wg sync.WaitGroup
	workerCount := min(len(targets), getMaxConcurrentWorkflowDownloads())
	sem := make(chan struct{}, workerCount)
	perTargetDownloads := max(1, getMaxConcurrentDownloads()/workerCount)
	for _, target := range targets {
		wg.Go(func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					results <- logsTargetResult{target: target, err: fmt.Errorf("workflow collector panicked: %v", recovered)}
				}
			}()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results <- logsTargetResult{target: target, err: ctx.Err()}
				return
			}

			targetOpts := opts
			targetOpts.WorkflowName = target.workflowName
			targetOpts.RepoOverride = target.repoOverride
			targetOpts.OutputDir = logsTargetOutputDir(opts.OutputDir, target)
			targetOpts.SummaryFile = ""
			targetOpts.Train = false
			targetOpts.SuppressRender = true
			targetOpts.skipEnsureGitignore = true
			targetOpts.rateLimitFirstRequest = true
			targetOpts.maxConcurrentDownloads = perTargetDownloads
			result, err := collectWorkflowLogsForTarget(ctx, targetOpts)
			results <- logsTargetResult{target: target, result: result, err: err}
		})
	}
	wg.Wait()
	close(results)
	return results
}

func mergeLogsTargetResults(
	results <-chan logsTargetResult,
	initialErrors []error,
) ([]ProcessedRun, []WorkflowContinuation, bool, []error) {
	allErrors := append([]error(nil), initialErrors...)
	var processedRuns []ProcessedRun
	timeoutReached := false
	var continuations []WorkflowContinuation
	for targetResult := range results {
		if targetResult.err != nil {
			allErrors = append(allErrors, fmt.Errorf("%s: %w", targetResult.target.displayName(), targetResult.err))
			continue
		}
		processedRuns = append(processedRuns, targetResult.result.processedRuns...)
		timeoutReached = timeoutReached || targetResult.result.timeoutReached
		if targetResult.result.continuation != nil {
			continuations = append(continuations, WorkflowContinuation{
				Repository:       targetResult.target.repoOverride,
				ContinuationData: *targetResult.result.continuation,
			})
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
				"Partial results for workflow target "+targetResult.target.displayName()+"; continuation parameters were written to the report",
			))
		}
	}
	return processedRuns, continuations, timeoutReached, allErrors
}

func (t logsWorkflowTarget) displayName() string {
	if t.repoOverride == "" {
		return t.workflowName
	}
	return filepath.Join(t.repoOverride, t.workflowName)
}

func logsTargetOutputDir(root string, target logsWorkflowTarget) string {
	workflowDir := stringutil.SanitizeForFilename(target.workflowName)
	if target.repoOverride == "" {
		return filepath.Join(root, workflowDir)
	}
	return filepath.Join(root, stringutil.SanitizeForFilename(target.repoOverride), workflowDir)
}

func getMaxConcurrentWorkflowDownloads() int {
	const maxConcurrentWorkflows = 4
	return min(maxConcurrentWorkflows, getMaxConcurrentDownloads())
}
