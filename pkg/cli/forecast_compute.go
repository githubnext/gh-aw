package cli

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var (
	forecastLoadCachedRunAIC = loadCachedRunAIC
	// forecastDownloadRunArtifacts uses a forecast-specific implementation that downloads
	// only the usage artifact and skips workflow run log downloads (not needed for AIC computation).
	forecastDownloadRunArtifacts = forecastDownloadUsageArtifact
	// Forecast only needs TotalAIC; avoid effective-token computation/logging in this path.
	forecastAnalyzeTokenUsage = analyzeTokenUsageAICOnly
)

func forecastWorkflow(ctx context.Context, workflowName, startDate string, config ForecastConfig, periodDays int) (ForecastWorkflowResult, error) {
	result := ForecastWorkflowResult{
		WorkflowID:  extractWorkflowIDFromName(workflowName),
		Period:      config.Period,
		HistoryDays: config.Days,
	}

	// Load frontmatter metadata (triggers, concurrency, experiments).
	meta := loadWorkflowMeta(workflowName, config.Verbose)
	result.ActiveTriggers = meta.activeTriggers
	result.ConcurrencyLimit = meta.concurrencyLimit
	result.ExperimentVariants = meta.variants
	result.Engines = meta.engines

	// Determine the API name used to filter workflow runs (prefer lock file name).
	apiName := workflowName
	if lockFile, err := workflow.GetWorkflowLockFileName(workflowName); err == nil {
		apiName = lockFile
	}

	// Fetch completed runs from the history window.
	opts := ListWorkflowRunsOptions{
		WorkflowName: apiName,
		Status:       "completed",
		StartDate:    startDate,
		Limit:        config.SampleSize,
		TargetCount:  config.SampleSize,
		RepoOverride: config.RepoOverride,
		Verbose:      config.Verbose,
	}

	runs, _, err := listRunsWithBackoff(ctx, opts, result.WorkflowID)
	if err != nil {
		if gitutil.IsRateLimitError(err.Error()) {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
				fmt.Sprintf("Skipping %s: GitHub API rate limit exceeded", result.WorkflowID)))
			return result, nil
		}
		return result, err
	}

	// Only use completed runs for metric computation.
	completed := make([]WorkflowRun, 0, len(runs))
	for _, r := range runs {
		if isCompletedNonSkippedRun(r) {
			// Compute Duration from StartedAt/UpdatedAt when not already set (gh run list
			// does not populate the Duration field; health_command uses the same approach).
			if r.Duration == 0 && !r.StartedAt.IsZero() && !r.UpdatedAt.IsZero() {
				r.Duration = r.UpdatedAt.Sub(r.StartedAt)
			}
			completed = append(completed, r)
		}
	}
	if len(completed) == 0 {
		forecastRunLog.Printf("No completed runs found for %s in last %d days", workflowName, config.Days)
		return result, nil
	}

	// Compute per-run averages and collect individual run samples.
	var totalAIC float64
	var totalDurSec float64
	successCount := 0
	aicObservations := make([]int, 0, len(completed))
	samples := make([]ForecastRunSample, 0, len(completed))

	for _, r := range completed {
		runAIC := forecastLoadCachedRunAIC(ctx, r.DatabaseID, config.Verbose)
		if runAIC <= 0 {
			forecastRunLog.Printf("Skipping run %d for %s: AIC=%.3f treated as missing data", r.DatabaseID, workflowName, runAIC)
			continue
		}
		if result.WorkflowPath == "" && r.WorkflowPath != "" {
			result.WorkflowPath = r.WorkflowPath
		}
		totalAIC += runAIC
		totalDurSec += r.Duration.Seconds()
		// Monte Carlo currently samples integer observations; keep milli-AIC precision
		// so sub-1 AIC runs are represented without losing granularity.
		aicObservations = append(aicObservations, int(math.Round(runAIC*1000)))
		if r.Conclusion == "success" {
			successCount++
		}
		sample := ForecastRunSample{RunID: r.DatabaseID, AIC: roundForecastAIC(runAIC)}
		if !r.StartedAt.IsZero() {
			sample.Date = r.StartedAt.Format("2006-01-02")
		}
		if r.URL != "" {
			sample.RunURL = r.URL
		}
		samples = append(samples, sample)
	}
	result.RunSamples = samples
	if result.WorkflowPath == "" {
		for _, r := range completed {
			if r.WorkflowPath != "" {
				result.WorkflowPath = r.WorkflowPath
				break
			}
		}
	}

	n := len(aicObservations)
	result.SampledRuns = n
	if n == 0 {
		forecastRunLog.Printf("No non-zero AIC run samples found for %s in last %d days", workflowName, config.Days)
		return result, nil
	}

	result.AvgAIC = roundForecastAIC(totalAIC / float64(n))
	result.AvgDurationSeconds = totalDurSec / float64(n)
	result.SuccessRate = float64(successCount) / float64(n)

	// Compute P50 and P95 of individual run AIC (per-run percentiles, not period totals).
	sortedAIC := make([]int, len(aicObservations))
	copy(sortedAIC, aicObservations)
	sort.Ints(sortedAIC)
	result.P50AIC = roundForecastAIC(float64(percentileInt(sortedAIC, 50)) / 1000)
	result.P95AIC = roundForecastAIC(float64(percentileInt(sortedAIC, 95)) / 1000)

	// Compute observed run frequency: runs per calendar day over the history window,
	// scaled to the projection period.
	observedRunsPerDay := float64(n) / float64(config.Days)
	result.ObservedRunsPerPeriod = observedRunsPerDay * float64(periodDays)

	// Point estimates for weekly (7-day) and monthly (30-day) projections.
	weeklyRuns := observedRunsPerDay * 7
	monthlyRuns := observedRunsPerDay * 30
	result.WeeklyProjectedAIC = roundForecastAIC(weeklyRuns * result.AvgAIC)
	result.MonthlyProjectedAIC = roundForecastAIC(monthlyRuns * result.AvgAIC)

	// Projected token usage (point estimate using simple means) for the configured period.
	result.ProjectedAIC = roundForecastAIC(result.ObservedRunsPerPeriod * result.AvgAIC)

	// Monte Carlo simulation: model run-count (Poisson), per-run token usage
	// (bootstrap), and per-run success (Bernoulli) to produce P10/P50/P90 ranges.
	// Two independent RNGs ensure the weekly and monthly simulations are uncorrelated.
	seed := time.Now().UnixNano()
	rng := rand.New(rand.NewSource(seed))      //nolint:gosec // non-cryptographic simulation RNG
	rng2 := rand.New(rand.NewSource(seed + 1)) //nolint:gosec
	rng3 := rand.New(rand.NewSource(seed + 2)) //nolint:gosec
	result.MonteCarlo = runMonteCarlo(aicObservations, successCount, result.ObservedRunsPerPeriod, rng)
	result.WeeklyMonteCarlo = runMonteCarlo(aicObservations, successCount, weeklyRuns, rng2)
	result.MonthlyMonteCarlo = runMonteCarlo(aicObservations, successCount, monthlyRuns, rng3)

	// Populate experiment variant fractions from run history when metadata has variants.
	result.ExperimentVariants = computeVariantFractions(result.ExperimentVariants, completed)

	return result, nil
}

// loadCachedRunAIC looks up a locally-cached RunSummary for the given
// run ID and returns the TotalAIC from its TokenUsage summary.
// Returns 0 when no cache exists or the cache does not contain AIC data.
// This avoids re-downloading aw_info.json artifacts for runs already processed by
// `gh aw logs` while still providing accurate AIC observations for the simulation.
//
// Cache location: <defaultLogsOutputDir>/run-<runID>/run_summary.json
// (defaultLogsOutputDir is ".github/aw/logs" — defined in logs_models.go)
func loadCachedRunAIC(ctx context.Context, runID int64, verbose bool) float64 {
	dir := filepath.Join(defaultLogsOutputDir, fmt.Sprintf("run-%d", runID))
	summary, ok := loadRunSummary(dir, verbose)
	if ok && summary != nil && summary.TokenUsage != nil && summary.TokenUsage.TotalAIC > 0 {
		forecastRunLog.Printf("AIC cache hit for run %d: aic=%.3f (from run_summary.json)", runID, summary.TokenUsage.TotalAIC)
		return summary.TokenUsage.TotalAIC
	}
	if ok && summary != nil && summary.TokenUsage != nil && summary.TokenUsage.TotalAIC <= 0 {
		forecastRunLog.Printf("AIC cache stale/empty for run %d: cached_total_aic=%.3f, token_file_recompute_required=true", runID, summary.TokenUsage.TotalAIC)
	}

	forecastRunLog.Printf("AIC cache miss for run %d; downloading usage artifact to %s", runID, dir)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Downloading usage artifact for run %d…", runID)))
	}

	tryDownload := func(filter []string) error {
		return forecastDownloadRunArtifacts(ctx, runID, dir, verbose, "", "", "", filter)
	}
	usageFilter := []string{"usage"}
	if err := tryDownload(usageFilter); err != nil {
		if errors.Is(err, ErrNoArtifacts) {
			forecastRunLog.Printf("No usage artifact for run %d; AIC will be 0", runID)
			return 0
		} else if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			forecastRunLog.Printf("Usage artifact download for run %d interrupted: %v", runID, err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Usage artifact download for run %d interrupted: %v", runID, err)))
			}
			return 0
		} else {
			forecastRunLog.Printf("Failed to download usage artifact for run %d: %v", runID, err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Failed to download usage artifact for run %d: %v", runID, err)))
			}
			return 0
		}
	}

	tokenUsage, err := forecastAnalyzeTokenUsage(dir, verbose)
	if err != nil || tokenUsage == nil || tokenUsage.TotalAIC <= 0 {
		forecastRunLog.Printf("No AIC data in usage artifact for run %d (err=%v, tokenUsage=%v)", runID, err, tokenUsage)
		return 0
	}
	forecastRunLog.Printf("AIC from usage artifact for run %d: aic=%.3f", runID, tokenUsage.TotalAIC)
	return tokenUsage.TotalAIC
}

// forecastDownloadUsageArtifact is a forecast-specific replacement for
// downloadRunArtifacts. Unlike the general-purpose downloader, it:
//   - Downloads only artifacts matching artifactFilter (typically ["usage"]).
//   - Skips workflow run log downloads entirely — logs are not needed for
//     AIC computation and downloading them wastes time when forecasting
//     many runs.
//   - Returns ErrNoArtifacts immediately when no matching artifact is found
//     rather than falling back to log diagnostics.
//
// It is referenced by forecastDownloadRunArtifacts so that tests can substitute
// a mock implementation without modifying the general artifact download path.
func forecastDownloadUsageArtifact(ctx context.Context, runID int64, outputDir string, verbose bool, owner, repo, hostname string, artifactFilter []string) error {
	forecastRunLog.Printf("Downloading usage artifact: run_id=%d, output_dir=%s, filter=%v", runID, outputDir, artifactFilter)
	shouldLogProgress := IsRunningInCI() || verbose

	// Check if the requested artifacts are already on disk (cache hit from actions/cache restore).
	if fileutil.DirExists(outputDir) && !fileutil.IsDirEmpty(outputDir) {
		missing := findMissingFilterEntries(artifactFilter, outputDir)
		if len(missing) == 0 {
			forecastRunLog.Printf("Usage artifact already on disk for run %d, skipping download", runID)
			if shouldLogProgress {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
					fmt.Sprintf("Usage artifact already present for run %d, skipping download", runID)))
			}
			return nil
		}
		forecastRunLog.Printf("Usage artifact partially missing for run %d: %v; downloading missing entries", runID, missing)
		artifactFilter = missing
	}

	if err := os.MkdirAll(outputDir, constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create output directory for run %d: %w", runID, err)
	}

	// List available artifacts for the run to find which match the filter.
	artifactNames, listErr := listRunArtifactNames(ctx, runID, owner, repo, hostname, verbose)
	if listErr != nil {
		forecastRunLog.Printf("Failed to list artifacts for run %d: %v", runID, listErr)
		if fileutil.IsDirEmpty(outputDir) {
			_ = os.RemoveAll(outputDir)
		}
		return fmt.Errorf("failed to list artifacts for run %d: %w", runID, listErr)
	}

	var downloadableNames []string
	for _, name := range artifactNames {
		if !isDockerBuildArtifact(name) && artifactMatchesFilter(name, artifactFilter) {
			downloadableNames = append(downloadableNames, name)
		}
	}

	forecastRunLog.Printf("Run %d: listed artifacts=%v, filter=%v, downloadable=%v", runID, artifactNames, artifactFilter, downloadableNames)

	if len(downloadableNames) == 0 {
		// No usage artifact — clean up empty directory and report.
		if fileutil.IsDirEmpty(outputDir) {
			_ = os.RemoveAll(outputDir)
		}
		return ErrNoArtifacts
	}

	if shouldLogProgress {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
			fmt.Sprintf("Downloading usage artifact(s) for run %d: %v", runID, downloadableNames)))
	}

	artifactOpts := downloadArtifactsOptions{runID: runID, outputDir: outputDir, verbose: verbose, owner: owner, repo: repo, hostname: hostname}
	if err := downloadArtifactsByName(ctx, artifactOpts, downloadableNames); err != nil {
		return fmt.Errorf("failed to download usage artifact for run %d: %w", runID, err)
	}

	if fileutil.IsDirEmpty(outputDir) {
		return ErrNoArtifacts
	}

	forecastRunLog.Printf("Downloaded usage artifact for run %d to %s", runID, outputDir)
	return nil
}

// emitPartialForecastResults outputs whatever workflow results have been collected so
// far when the forecast computation is interrupted (timeout or user cancellation).
// Partial results are only meaningful when at least one workflow has been fully
// processed; the function is a no-op when results is empty so callers do not need to
// guard against it.
func emitPartialForecastResults(results []ForecastWorkflowResult, config ForecastConfig, now time.Time) {
	if len(results) == 0 {
		return
	}
	forecastRunLog.Printf("Emitting %d partial forecast result(s) before early exit", len(results))
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
		fmt.Sprintf("Forecast interrupted; emitting partial results for %d workflow(s) processed so far.", len(results))))

	// Sort partial results by Monte Carlo P50 descending (mirrors the full-results sort).
	slices.SortFunc(results, func(a, b ForecastWorkflowResult) int {
		pi := a.ProjectedAIC
		if mc := a.MonteCarlo; mc != nil {
			pi = mc.P50ProjectedAIC
		}
		pj := b.ProjectedAIC
		if mc := b.MonteCarlo; mc != nil {
			pj = mc.P50ProjectedAIC
		}
		if pi > pj {
			return -1
		}
		if pi < pj {
			return 1
		}
		return 0
	})

	output := ForecastResult{
		Period:    config.Period,
		AsOf:      now.UTC().Format(time.RFC3339),
		EvalMode:  config.EvalMode,
		Workflows: results,
	}
	if config.JSONOutput {
		_ = renderForecastJSON(output)
	} else {
		_ = renderForecastTable(output, config)
	}
}

func isCompletedNonSkippedRun(r WorkflowRun) bool {
	return r.Status == "completed" && r.Conclusion != "skipped"
}

// evaluateForecast fetches actual completed runs in the validation window and
// returns a ForecastEvaluation comparing them against the Monte Carlo forecast.
//
// validationStartDate / validationEndDate are ISO-8601 strings bracketing the
// period that was forecast (= one projection period immediately before now).
// Actual runs are fetched with the same pagination helper used for training,
// but with the validation date range.
func evaluateForecast(ctx context.Context, workflowName string, forecast ForecastWorkflowResult, validationStartDate, validationEndDate string, config ForecastConfig) *ForecastEvaluation {
	// Compute the actual ISO-8601 training start date by subtracting HistoryDays
	// from the validation start (= anchor).
	var trainingStartDate string
	if t, err := time.Parse("2006-01-02", validationStartDate); err == nil {
		trainingStartDate = t.AddDate(0, 0, -forecast.HistoryDays).Format("2006-01-02")
	} else {
		trainingStartDate = validationStartDate
	}
	eval := &ForecastEvaluation{
		TrainingStartDate: trainingStartDate,
		TrainingEndDate:   validationStartDate,
		ValidationEndDate: validationEndDate,
	}

	// Determine the API name used to filter workflow runs.
	apiName := workflowName
	if lockFile, err := workflow.GetWorkflowLockFileName(workflowName); err == nil {
		apiName = lockFile
	}

	// Fetch completed runs in the validation window.
	opts := ListWorkflowRunsOptions{
		WorkflowName: apiName,
		Status:       "completed",
		StartDate:    validationStartDate,
		Limit:        config.SampleSize,
		TargetCount:  config.SampleSize,
		RepoOverride: config.RepoOverride,
		Verbose:      config.Verbose,
	}
	opts.Context = ctx
	runs, _, err := listWorkflowRunsWithPagination(opts)
	if err != nil {
		forecastRunLog.Printf("Eval: failed to fetch validation runs for %s: %v", workflowName, err)
		return eval
	}

	// Filter to completed runs that fall within the validation window.
	validationEnd := time.Now()
	validationStart, _ := time.Parse("2006-01-02", validationStartDate)
	for _, r := range runs {
		if !isCompletedNonSkippedRun(r) {
			continue
		}
		// Skip runs with no timestamp — we cannot verify they belong to the
		// validation window, so including them would introduce undefined bias.
		if r.StartedAt.IsZero() {
			continue
		}
		if r.StartedAt.Before(validationStart) || r.StartedAt.After(validationEnd) {
			continue
		}
		eval.ActualRuns++
		eval.ActualAIC += forecastLoadCachedRunAIC(ctx, r.DatabaseID, config.Verbose)
	}

	// Compute error metrics against P50 (falls back to point estimate).
	p50 := forecast.ProjectedAIC
	p10 := forecast.ProjectedAIC
	p90 := forecast.ProjectedAIC
	if mc := forecast.MonteCarlo; mc != nil {
		p50 = mc.P50ProjectedAIC
		p10 = mc.P10ProjectedAIC
		p90 = mc.P90ProjectedAIC
	}

	eval.P50ErrorAbs = eval.ActualAIC - p50
	if p50 > 0 {
		eval.P50ErrorPct = eval.P50ErrorAbs / p50 * 100
	}
	eval.InCI = eval.ActualAIC >= p10 && eval.ActualAIC <= p90

	return eval
}
