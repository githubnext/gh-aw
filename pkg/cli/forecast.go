package cli

// This file implements the `forecast` command, which samples a workflow's recent
// GitHub Actions run history and projects forward effective token usage (including
// Monte Carlo probability distributions) on a per-week or per-month basis.
//
// Workflow metadata (trigger types, concurrency, experiments) is read from the
// workflow's Markdown frontmatter so that projections account for how often the
// workflow is actually expected to fire and how many concurrent runs it supports.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
)

var forecastRunLog = logger.New("cli:forecast_run")

// forecastPeriodDays maps period names to the number of days in a projection window.
var forecastPeriodDays = map[string]int{
	"week":  7,
	"month": 30,
}

const (
	forecastRateLimitMaxAttempts = 3
	forecastRateLimitBaseBackoff = 2 * time.Second
)

var (
	forecastFetchGitHubWorkflows      = fetchGitHubWorkflows
	forecastListWorkflowRunsPaginated = listWorkflowRunsWithPagination
	forecastRateLimitSleep            = func(ctx context.Context, delay time.Duration) error {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		select {
		case <-timer.C:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
)

// ForecastWorkflowResult contains the projected metrics for a single workflow.
type ForecastWorkflowResult struct {
	// WorkflowID is the short identifier of the workflow (basename without .md).
	WorkflowID string `json:"workflow_id"`
	// Period is the projection window ("week" or "month").
	Period string `json:"period"`
	// SampledRuns is the number of completed runs used to derive per-run averages.
	SampledRuns int `json:"sampled_runs"`
	// HistoryDays is the number of calendar days covered by the sampled runs.
	HistoryDays int `json:"history_days"`

	// Observed run frequency (derived from sampled run history).
	ObservedRunsPerPeriod float64 `json:"observed_runs_per_period"`

	// SuccessRate is the fraction of sampled runs that completed successfully (0–1).
	SuccessRate float64 `json:"success_rate"`

	// Average per-run metrics (from completed runs).
	AvgEffectiveTokens int     `json:"avg_effective_tokens"`
	AvgDurationSeconds float64 `json:"avg_duration_seconds"`

	// Projected totals for the period.
	ProjectedEffectiveTokens int `json:"projected_effective_tokens"`

	// MonteCarlo contains the probability distribution of projected effective-token
	// counts derived from a Monte Carlo simulation (10 000 trials).
	// Nil when no completed runs were available.
	MonteCarlo *ForecastMonteCarloSummary `json:"monte_carlo,omitempty"`

	// Trigger information derived from frontmatter.
	ActiveTriggers []string `json:"active_triggers"`
	// ConcurrencyLimit is the workflow-level concurrency limit (0 = unlimited).
	ConcurrencyLimit int `json:"concurrency_limit"`

	// ExperimentVariants contains per-variant forecasts when the workflow defines A/B
	// experiments.  Nil when no experiments are present.
	ExperimentVariants []ForecastVariantResult `json:"experiment_variants,omitempty"`

	// Evaluation contains backtesting quality metrics when --eval is set.
	// Nil in normal forecast mode.
	Evaluation *ForecastEvaluation `json:"evaluation,omitempty"`
}

// ForecastVariantResult contains projected metrics split by A/B experiment variant.
type ForecastVariantResult struct {
	ExperimentName string  `json:"experiment_name"`
	Variant        string  `json:"variant"`
	RunCount       int     `json:"run_count"`
	Fraction       float64 `json:"fraction"`
}

// ForecastEvaluation contains the quality metrics for a backtested forecast.
// It is populated only when --eval is set.  The training window ends one
// projection period before now; the validation window is the most recent period.
type ForecastEvaluation struct {
	// TrainingStartDate is the ISO-8601 date the training window began.
	TrainingStartDate string `json:"training_start_date"`
	// TrainingEndDate is the ISO-8601 date the training window ended
	// (= the start of the validation window).
	TrainingEndDate string `json:"training_end_date"`
	// ValidationEndDate is the ISO-8601 date the validation window ended (= today).
	ValidationEndDate string `json:"validation_end_date"`

	// ActualRuns is the number of completed runs observed in the validation window.
	ActualRuns int `json:"actual_runs"`
	// ActualEffectiveTokens is the total effective-token count actually consumed
	// in the validation window.
	ActualEffectiveTokens int `json:"actual_effective_tokens"`

	// P50ErrorAbs is the signed difference (actual − P50 forecast) in effective tokens.
	// Positive = actual was higher than forecast; negative = forecast over-estimated.
	P50ErrorAbs int `json:"p50_error_abs"`
	// P50ErrorPct is P50ErrorAbs as a percentage of the P50 forecast.
	// NaN-safe: 0 when P50 is 0.
	P50ErrorPct float64 `json:"p50_error_pct"`
	// InCI is true when ActualEffectiveTokens fell within the P10–P90 confidence
	// interval.  A well-calibrated model should be in-CI ~80% of the time.
	InCI bool `json:"in_ci"`
}

// ForecastResult is the top-level output of the forecast command.
type ForecastResult struct {
	Period    string                   `json:"period"`
	AsOf      string                   `json:"as_of"`
	EvalMode  bool                     `json:"eval_mode,omitempty"`
	Workflows []ForecastWorkflowResult `json:"workflows"`
}

// RunForecast is the entry point for the forecast command.
func RunForecast(config ForecastConfig) error {
	forecastRunLog.Printf("Running forecast: workflows=%v, days=%d, period=%s, eval=%v", config.WorkflowIDs, config.Days, config.Period, config.EvalMode)
	if config.TimeoutMinutes < 0 {
		return fmt.Errorf("invalid timeout value: %d; must be >= 0", config.TimeoutMinutes)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if config.TimeoutMinutes > 0 {
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(config.TimeoutMinutes)*time.Minute)
		defer cancel()
		ctx = timeoutCtx
	}

	// Emit experimental warning so users know this command is not yet stable.
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage("forecast is an experimental command and may change without notice"))

	// Validate period.
	periodDays, ok := forecastPeriodDays[config.Period]
	if !ok {
		return fmt.Errorf("invalid period %q: must be 'week' or 'month'", config.Period)
	}
	if config.Days != 7 && config.Days != 30 {
		return fmt.Errorf("invalid days value: %d; must be 7 or 30", config.Days)
	}
	if config.SampleSize <= 0 {
		config.SampleSize = 100
	}

	// Resolve the list of workflow IDs to forecast.
	workflowIDs, err := resolveForecastWorkflows(ctx, config)
	if err != nil {
		return normalizeForecastRunError(err, config)
	}
	if len(workflowIDs) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No agentic workflows found to forecast"))
		return nil
	}

	now := time.Now()

	// In eval mode, shift the entire date range back by one period so we can
	// compare the forecast against the actual runs in the most recent period.
	//
	//  ┌──────────────────────────────────────────────────────────────────┐
	//  │  [anchor - days ... anchor]  training  │  [anchor ... now]  val  │
	//  └──────────────────────────────────────────────────────────────────┘
	//   anchor = now - periodDays
	//
	// Normal mode: startDate = now - days (no anchor shift).
	var anchor time.Time
	var validationStartDate, validationEndDate string
	if config.EvalMode {
		anchor = now.AddDate(0, 0, -periodDays)
		validationStartDate = anchor.Format("2006-01-02")
		validationEndDate = now.Format("2006-01-02")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
			fmt.Sprintf("Eval mode: training window ends %s; validation window %s → %s",
				anchor.Format("2006-01-02"), validationStartDate, validationEndDate)))
	}

	startDate := now.AddDate(0, 0, -config.Days).Format("2006-01-02")
	if config.EvalMode {
		// Training window ends at the anchor, not now.
		startDate = anchor.AddDate(0, 0, -config.Days).Format("2006-01-02")
	}

	if !config.Verbose && !config.JSONOutput {
		label := fmt.Sprintf("Forecasting %d workflow(s) using %d-day history → projecting per %s",
			len(workflowIDs), config.Days, config.Period)
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(label))
	}

	spinner := console.NewSpinner("Sampling workflow run history…")
	if !config.Verbose {
		spinner.Start()
	}

	results := make([]ForecastWorkflowResult, 0, len(workflowIDs))
	for _, wfID := range workflowIDs {
		if err := ctx.Err(); err != nil {
			if !config.Verbose {
				spinner.Stop()
			}
			return normalizeForecastRunError(err, config)
		}
		if !config.Verbose {
			spinner.UpdateMessage(fmt.Sprintf("Sampling %s…", wfID))
		}

		// forecastWorkflow uses the shifted startDate; in eval mode we also pass the
		// anchor so the function knows where the training window ends.
		result, err := forecastWorkflow(ctx, wfID, startDate, config, periodDays)
		if err != nil {
			// context.Canceled typically indicates user interruption (Ctrl-C), while
			// context.DeadlineExceeded indicates the configured forecast timeout.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				if !config.Verbose {
					spinner.Stop()
				}
				return normalizeForecastRunError(err, config)
			}
			if !config.Verbose {
				spinner.Stop()
			}
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
				fmt.Sprintf("Skipping %s: %v", wfID, err)))
			if !config.Verbose {
				spinner.Start()
			}
			continue
		}

		// In eval mode, fetch the validation-window runs and attach evaluation metrics.
		if config.EvalMode {
			result.Evaluation = evaluateForecast(ctx, wfID, result, validationStartDate, validationEndDate, config)
		}

		results = append(results, result)
	}

	if !config.Verbose {
		spinner.Stop()
	}

	// Sort results by Monte Carlo P50 (or point estimate when MC unavailable) descending.
	sort.Slice(results, func(i, j int) bool {
		pi := results[i].ProjectedEffectiveTokens
		if mc := results[i].MonteCarlo; mc != nil {
			pi = mc.P50ProjectedEffectiveTokens
		}
		pj := results[j].ProjectedEffectiveTokens
		if mc := results[j].MonteCarlo; mc != nil {
			pj = mc.P50ProjectedEffectiveTokens
		}
		return pi > pj
	})

	output := ForecastResult{
		Period:    config.Period,
		AsOf:      now.UTC().Format(time.RFC3339),
		EvalMode:  config.EvalMode,
		Workflows: results,
	}

	if config.JSONOutput {
		return renderForecastJSON(output)
	}
	return renderForecastTable(output, config)
}

func normalizeForecastRunError(err error, config ForecastConfig) error {
	if config.TimeoutMinutes > 0 && errors.Is(err, context.DeadlineExceeded) {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(
			fmt.Sprintf("Forecast computation timed out after %d minute(s).", config.TimeoutMinutes),
		))
		return &ExitCodeError{Code: 124}
	}
	return err
}

// resolveForecastWorkflows returns the ordered list of workflow IDs to forecast.
// When WorkflowIDs is empty, all agentic workflow IDs in the repository are returned.
// When RepoOverride is set, workflows are discovered via the GitHub API instead of local files.
func resolveForecastWorkflows(ctx context.Context, config ForecastConfig) ([]string, error) {
	if config.RepoOverride != "" {
		return resolveForecastWorkflowsFromRemote(ctx, config.WorkflowIDs, config.RepoOverride, config.Verbose)
	}

	if len(config.WorkflowIDs) > 0 {
		// Resolve each provided ID to a canonical lock-file workflow name.
		resolved := make([]string, 0, len(config.WorkflowIDs))
		for _, id := range config.WorkflowIDs {
			name, err := workflow.FindWorkflowName(id)
			if err != nil {
				return nil, fmt.Errorf("workflow %q not found: %w", id, err)
			}
			resolved = append(resolved, name)
		}
		return resolved, nil
	}

	// No explicit IDs: discover all agentic workflows from .lock.yml files.
	names, err := getAgenticWorkflowNames(config.Verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to discover agentic workflows: %w", err)
	}
	return names, nil
}

// resolveForecastWorkflowsFromRemote resolves workflow names for a remote repository using
// the GitHub API. When ids is empty, all workflows in the remote repository are returned.
// When ids are provided, each is matched (case-insensitively) against remote workflow names
// and file-path basenames.
func resolveForecastWorkflowsFromRemote(ctx context.Context, ids []string, repoOverride string, verbose bool) ([]string, error) {
	githubWorkflows, err := fetchWorkflowsWithBackoff(ctx, ids, repoOverride, verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows in %s: %w", repoOverride, err)
	}

	if len(ids) == 0 {
		// Return display names for all workflows in the remote repo.
		names := make([]string, 0, len(githubWorkflows))
		for _, wf := range githubWorkflows {
			names = append(names, wf.Name)
		}
		sort.Strings(names)
		return names, nil
	}

	// Match each provided ID against the remote workflow list.
	resolved := make([]string, 0, len(ids))
	for _, id := range ids {
		matched := matchRemoteWorkflowName(id, githubWorkflows)
		if matched == "" {
			return nil, fmt.Errorf("workflow %q not found in %s", id, repoOverride)
		}
		resolved = append(resolved, matched)
	}
	return resolved, nil
}

func forecastRateLimitBackoffDuration(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	return time.Duration(attempt) * forecastRateLimitBaseBackoff
}

func fetchWorkflowsWithBackoff(ctx context.Context, ids []string, repoOverride string, verbose bool) (map[string]*GitHubWorkflow, error) {
	var lastErr error

	for attempt := 1; attempt <= forecastRateLimitMaxAttempts; attempt++ {
		githubWorkflows, err := forecastFetchGitHubWorkflows(repoOverride, verbose)
		if err == nil {
			return githubWorkflows, nil
		}
		if !gitutil.IsRateLimitError(err.Error()) {
			return nil, err
		}

		lastErr = err
		if attempt == forecastRateLimitMaxAttempts {
			break
		}

		backoff := forecastRateLimitBackoffDuration(attempt)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
			fmt.Sprintf("GitHub API rate limit hit while discovering workflows in %s; backing off for %s before retry %d/%d",
				repoOverride, backoff, attempt+1, forecastRateLimitMaxAttempts)))
		if err := forecastRateLimitSleep(ctx, backoff); err != nil {
			return nil, err
		}
	}

	if len(ids) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
			fmt.Sprintf("GitHub API rate limit exhausted while discovering workflows in %s; continuing with caller-supplied workflow IDs as partial results",
				repoOverride)))

		partialWorkflows := make(map[string]*GitHubWorkflow, len(ids))
		for _, id := range ids {
			partialWorkflows[id] = &GitHubWorkflow{Name: id, Path: id, State: "unknown"}
		}
		return partialWorkflows, nil
	}

	return nil, fmt.Errorf("GitHub API rate limit exhausted after %d attempts: %w", forecastRateLimitMaxAttempts, lastErr)
}

func listRunsWithBackoff(ctx context.Context, opts ListWorkflowRunsOptions, workflowID string) ([]WorkflowRun, int, error) {
	var lastErr error
	opts.Context = ctx

	for attempt := 1; attempt <= forecastRateLimitMaxAttempts; attempt++ {
		runs, total, err := forecastListWorkflowRunsPaginated(opts)
		if err == nil {
			return runs, total, nil
		}
		if !gitutil.IsRateLimitError(err.Error()) {
			return nil, 0, err
		}

		lastErr = err
		if attempt == forecastRateLimitMaxAttempts {
			break
		}

		backoff := forecastRateLimitBackoffDuration(attempt)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
			fmt.Sprintf("GitHub API rate limit hit while sampling %s; backing off for %s before retry %d/%d",
				workflowID, backoff, attempt+1, forecastRateLimitMaxAttempts)))
		if err := forecastRateLimitSleep(ctx, backoff); err != nil {
			return nil, 0, err
		}
	}

	return nil, 0, lastErr
}

// matchRemoteWorkflowName returns the display name of the workflow in the remote map that
// best matches id. Matching is tried against the file-based key (e.g. "ci-doctor") and the
// display name (e.g. "CI Failure Doctor"), both case-insensitively. Returns "" on no match.
func matchRemoteWorkflowName(id string, workflows map[string]*GitHubWorkflow) string {
	lowerID := strings.ToLower(id)
	for key, wf := range workflows {
		if strings.ToLower(key) == lowerID || strings.ToLower(wf.Name) == lowerID {
			return wf.Name
		}
	}
	return ""
}

// forecastWorkflow computes a ForecastWorkflowResult for a single workflow.
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

	// Determine the API name used to filter workflow runs (prefer lock file name).
	apiName := workflowName
	if lockFile, err := workflow.GetWorkflowLockFileName(workflowName); err == nil {
		apiName = lockFile
	}

	// Fetch completed runs from the history window.
	opts := ListWorkflowRunsOptions{
		WorkflowName: apiName,
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
			// Enrich with ET from a locally-cached run summary when available.
			// gh run list does not return token-usage fields; they are only stored in
			// the aw_info.json artifacts downloaded by `gh aw logs`.  Loading the cached
			// RunSummary avoids re-downloading artifacts while still providing accurate
			// ET observations for runs that have already been processed locally.
			if r.EffectiveTokens == 0 {
				r.EffectiveTokens = loadCachedEffectiveTokens(r.DatabaseID, config.Verbose)
			}
			completed = append(completed, r)
		}
	}
	result.SampledRuns = len(completed)

	if len(completed) == 0 {
		forecastRunLog.Printf("No completed runs found for %s in last %d days", workflowName, config.Days)
		return result, nil
	}

	// Compute per-run averages.
	var totalET int
	var totalDurSec float64
	successCount := 0
	etObservations := make([]int, 0, len(completed))

	for _, r := range completed {
		totalET += r.EffectiveTokens
		totalDurSec += r.Duration.Seconds()
		etObservations = append(etObservations, r.EffectiveTokens)
		if r.Conclusion == "success" {
			successCount++
		}
	}

	n := len(completed)
	result.AvgEffectiveTokens = totalET / n
	result.AvgDurationSeconds = totalDurSec / float64(n)
	result.SuccessRate = float64(successCount) / float64(n)

	// Compute observed run frequency: runs per calendar day over the history window,
	// scaled to the projection period.
	result.ObservedRunsPerPeriod = float64(n) / float64(config.Days) * float64(periodDays)

	// Projected token usage (point estimate using simple means).
	result.ProjectedEffectiveTokens = int(math.Round(result.ObservedRunsPerPeriod * float64(result.AvgEffectiveTokens)))

	// Monte Carlo simulation: model run-count (Poisson), per-run token usage
	// (bootstrap), and per-run success (Bernoulli) to produce P10/P50/P90 ranges.
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // non-cryptographic simulation RNG
	result.MonteCarlo = runMonteCarlo(etObservations, successCount, result.ObservedRunsPerPeriod, rng)

	// Populate experiment variant fractions from run history when metadata has variants.
	result.ExperimentVariants = computeVariantFractions(result.ExperimentVariants, completed)

	return result, nil
}

// workflowMeta holds parsed metadata from a workflow's Markdown frontmatter.
type workflowMeta struct {
	activeTriggers   []string
	concurrencyLimit int
	variants         []ForecastVariantResult
}

// loadWorkflowMeta reads the workflow's Markdown file and extracts frontmatter metadata.
// Errors are non-fatal; a partial result is returned on failure.
func loadWorkflowMeta(workflowName string, verbose bool) workflowMeta {
	meta := workflowMeta{}

	// Try to find the Markdown source file.
	mdFile := findMarkdownFileForWorkflow(workflowName)
	if mdFile == "" {
		forecastRunLog.Printf("Markdown file not found for workflow %q", workflowName)
		return meta
	}

	content, err := os.ReadFile(mdFile)
	if err != nil {
		forecastRunLog.Printf("Failed to read Markdown file %q: %v", mdFile, err)
		return meta
	}

	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil || result.Frontmatter == nil {
		forecastRunLog.Printf("Failed to parse frontmatter for %q: %v", workflowName, err)
		return meta
	}

	cfg, err := workflow.ParseFrontmatterConfig(result.Frontmatter)
	if err != nil || cfg == nil {
		forecastRunLog.Printf("Failed to build FrontmatterConfig for %q: %v", workflowName, err)
		return meta
	}

	// Collect active trigger names.
	meta.activeTriggers = extractTriggerNames(cfg)

	// Concurrency limit: read the `cancel-in-progress` or derive from the concurrency map.
	meta.concurrencyLimit = extractConcurrencyLimit(cfg)

	// Collect experiment variant names (counts come from run history later).
	meta.variants = extractExperimentVariantStubs(cfg)

	return meta
}

// findMarkdownFileForWorkflow tries to locate the .md source file for a workflow.
func findMarkdownFileForWorkflow(workflowName string) string {
	// workflowName might be a display name like "CI Doctor" or a lock file like "ci-doctor.lock.yml".
	// Try to reverse-engineer the md file path.
	candidates := []string{
		fmt.Sprintf(".github/workflows/%s.md", workflowName),
	}
	// Strip known suffixes.
	for _, sfx := range []string{".lock.yml", ".yml", ".yaml"} {
		if base, ok := strings.CutSuffix(workflowName, sfx); ok {
			// Also strip ".lock" from lock files.
			base, _ = strings.CutSuffix(base, ".lock")
			candidates = append(candidates, fmt.Sprintf(".github/workflows/%s.md", base))
		}
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// extractTriggerNames returns the list of active trigger event names from a workflow config.
func extractTriggerNames(cfg *workflow.FrontmatterConfig) []string {
	if cfg.On == nil {
		return nil
	}
	names := make([]string, 0, len(cfg.On))
	for k := range cfg.On {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// extractConcurrencyLimit returns the workflow-level concurrency limit.
// Returns 0 when unlimited (no concurrency config) and 1 when concurrency is configured
// (either via cancel-in-progress or a concurrency group, since GitHub Actions queues at
// most one pending run when a concurrency group is set).
func extractConcurrencyLimit(cfg *workflow.FrontmatterConfig) int {
	if cfg.Concurrency == nil {
		return 0
	}
	// When concurrency is configured with cancel-in-progress: true, effective concurrency = 1.
	if v, ok := cfg.Concurrency["cancel-in-progress"]; ok {
		if b, _ := v.(bool); b {
			return 1
		}
	}
	// When there's a concurrency group without cancel-in-progress, runs queue up; treat as 1
	// active at a time by convention (GitHub Actions queues at most one pending run).
	if _, hasGroup := cfg.Concurrency["group"]; hasGroup {
		return 1
	}
	return 0
}

// extractExperimentVariantStubs extracts experiment variant metadata from frontmatter.
// Run counts are not yet known at this stage; they are populated from run history later.
func extractExperimentVariantStubs(cfg *workflow.FrontmatterConfig) []ForecastVariantResult {
	if len(cfg.ExperimentConfigs) == 0 {
		return nil
	}
	stubs := make([]ForecastVariantResult, 0)
	for expName, expCfg := range cfg.ExperimentConfigs {
		if expCfg == nil {
			continue
		}
		for _, variant := range expCfg.Variants {
			stubs = append(stubs, ForecastVariantResult{
				ExperimentName: expName,
				Variant:        variant,
			})
		}
	}
	sort.Slice(stubs, func(i, j int) bool {
		if stubs[i].ExperimentName != stubs[j].ExperimentName {
			return stubs[i].ExperimentName < stubs[j].ExperimentName
		}
		return stubs[i].Variant < stubs[j].Variant
	})
	return stubs
}

// computeVariantFractions populates run counts and fractions on the variant stubs
// by examining the DisplayTitle of sampled runs (gh-aw encodes the variant there).
// When no stubs are present (workflow has no experiments), returns nil.
func computeVariantFractions(stubs []ForecastVariantResult, runs []WorkflowRun) []ForecastVariantResult {
	if len(stubs) == 0 {
		return nil
	}

	total := len(runs)
	if total == 0 {
		return stubs
	}

	// Count how many run titles contain each variant name.
	for i, stub := range stubs {
		count := 0
		for _, r := range runs {
			if strings.Contains(r.DisplayTitle, stub.Variant) {
				count++
			}
		}
		stubs[i].RunCount = count
		stubs[i].Fraction = float64(count) / float64(total)
	}
	return stubs
}

// extractWorkflowIDFromName returns the short workflow ID from a display/lock name.
func extractWorkflowIDFromName(name string) string {
	for _, sfx := range []string{".lock.yml", ".yml", ".yaml"} {
		if base, ok := strings.CutSuffix(name, sfx); ok {
			base, _ = strings.CutSuffix(base, ".lock")
			name = base
		}
	}
	return name
}

// loadCachedEffectiveTokens looks up a locally-cached RunSummary for the given
// run ID and returns the TotalEffectiveTokens from its TokenUsage summary.
// Returns 0 when no cache exists or the cache does not contain token data.
// This avoids re-downloading aw_info.json artifacts for runs already processed by
// `gh aw logs` while still providing accurate ET observations for the simulation.
//
// Cache location: <defaultLogsOutputDir>/run-<runID>/run_summary.json
// (defaultLogsOutputDir is ".github/aw/logs" — defined in logs_models.go)
func loadCachedEffectiveTokens(runID int64, verbose bool) int {
	dir := filepath.Join(defaultLogsOutputDir, fmt.Sprintf("run-%d", runID))
	summary, ok := loadRunSummary(dir, verbose)
	if !ok || summary == nil {
		return 0
	}
	if summary.TokenUsage != nil && summary.TokenUsage.TotalEffectiveTokens > 0 {
		return summary.TokenUsage.TotalEffectiveTokens
	}
	// Fallback: legacy run summaries (written before TokenUsage was a separate
	// field) may have stored the computed ET directly on the Run struct.
	if summary.Run.EffectiveTokens > 0 {
		return summary.Run.EffectiveTokens
	}
	return 0
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
		if r.EffectiveTokens == 0 {
			r.EffectiveTokens = loadCachedEffectiveTokens(r.DatabaseID, config.Verbose)
		}
		eval.ActualRuns++
		eval.ActualEffectiveTokens += r.EffectiveTokens
	}

	// Compute error metrics against P50 (falls back to point estimate).
	p50 := forecast.ProjectedEffectiveTokens
	p10 := forecast.ProjectedEffectiveTokens
	p90 := forecast.ProjectedEffectiveTokens
	if mc := forecast.MonteCarlo; mc != nil {
		p50 = mc.P50ProjectedEffectiveTokens
		p10 = mc.P10ProjectedEffectiveTokens
		p90 = mc.P90ProjectedEffectiveTokens
	}

	eval.P50ErrorAbs = eval.ActualEffectiveTokens - p50
	if p50 > 0 {
		eval.P50ErrorPct = float64(eval.P50ErrorAbs) / float64(p50) * 100
	}
	eval.InCI = eval.ActualEffectiveTokens >= p10 && eval.ActualEffectiveTokens <= p90

	return eval
}

// ── Rendering ───────────────────────────────────────────────────────────────

// renderForecastJSON outputs the forecast result as pretty-printed JSON.
func renderForecastJSON(output ForecastResult) error {
	b, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal forecast JSON: %w", err)
	}
	fmt.Fprintln(os.Stdout, string(b))
	return nil
}

// forecastTableRow is a flattened struct used for console table rendering.
type forecastTableRow struct {
	Workflow           string `json:"workflow"                console:"header:Workflow"`
	Runs               int    `json:"runs"                    console:"header:Sampled Runs"`
	SuccessRate        string `json:"success_rate"            console:"header:Success Rate"`
	AvgEffectiveTokens string `json:"avg_effective_tokens"    console:"header:Avg ET"`
	ProjectedTokens    string `json:"projected_tokens"        console:"header:Proj. ET (P50)"`
	ETRange            string `json:"et_range"                console:"header:80% CI (P10–P90)"`
	Triggers           string `json:"triggers"                console:"header:Triggers"`
}

// renderForecastTable renders the forecast result as a human-readable table.
func renderForecastTable(output ForecastResult, config ForecastConfig) error {
	periodLabel := strings.ToUpper(output.Period[:1]) + output.Period[1:]
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
		fmt.Sprintf("Workflow Forecast — per %s (based on last %d days of history)", periodLabel, config.Days)))
	fmt.Fprintln(os.Stderr, "")

	anyUnreliable := false
	rows := make([]forecastTableRow, 0, len(output.Workflows))
	for _, wf := range output.Workflows {
		// Use Monte Carlo P50 as the primary ET estimate when available.
		projETStr := formatForecastTokens(wf.ProjectedEffectiveTokens)
		etRangeStr := "-"
		unreliableMark := ""
		if mc := wf.MonteCarlo; mc != nil {
			projETStr = formatForecastTokens(mc.P50ProjectedEffectiveTokens)
			if mc.P10ProjectedEffectiveTokens == 0 && mc.P90ProjectedEffectiveTokens == 0 {
				etRangeStr = "-"
			} else {
				etRangeStr = fmt.Sprintf("%s–%s",
					formatForecastTokens(mc.P10ProjectedEffectiveTokens),
					formatForecastTokens(mc.P90ProjectedEffectiveTokens))
			}
			if !mc.IsReliable {
				anyUnreliable = true
				unreliableMark = "*"
			}
		}
		row := forecastTableRow{
			Workflow:           wf.WorkflowID + unreliableMark,
			Runs:               wf.SampledRuns,
			SuccessRate:        formatForecastPercent(wf.SuccessRate, wf.SampledRuns > 0),
			AvgEffectiveTokens: formatForecastTokens(wf.AvgEffectiveTokens),
			ProjectedTokens:    projETStr,
			ETRange:            etRangeStr,
			Triggers:           formatTriggerList(wf.ActiveTriggers),
		}
		rows = append(rows, row)
	}

	fmt.Fprint(os.Stderr, console.RenderStruct(rows))
	fmt.Fprintln(os.Stderr, "")

	// Show experiment variant details when present.
	for _, wf := range output.Workflows {
		if len(wf.ExperimentVariants) > 0 {
			printVariantBreakdown(wf)
		}
	}

	// Show backtesting evaluation table in --eval mode.
	if output.EvalMode {
		printEvalBreakdown(output.Workflows)
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
		fmt.Sprintf("P50 = median; 80%% CI = P10–P90 from %d-trial Monte Carlo simulation (Gamma–Poisson model accounts for rate estimation uncertainty).", monteCarloIterations)))
	if anyUnreliable {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
			fmt.Sprintf("* Fewer than %d sampled runs — confidence intervals may be unreliable.", minObservationsForReliableForecast)))
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
		fmt.Sprintf("Run '%s forecast --json' for full output.", string(constants.CLIExtensionPrefix))))
	return nil
}

// printEvalBreakdown renders the backtesting comparison table.
func printEvalBreakdown(workflows []ForecastWorkflowResult) {
	type evalRow struct {
		Workflow    string `json:"workflow"       console:"header:Workflow"`
		ActualRuns  int    `json:"actual_runs"    console:"header:Actual Runs"`
		ActualET    string `json:"actual_et"      console:"header:Actual ET"`
		ForecastP50 string `json:"forecast_p50"   console:"header:Forecast P50"`
		ErrorAbs    string `json:"error_abs"      console:"header:Error (abs)"`
		ErrorPct    string `json:"error_pct"      console:"header:Error %"`
		InCI        string `json:"in_ci"          console:"header:In 80% CI?"`
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Backtesting evaluation (actual vs forecasted):"))
	var rows []evalRow
	for _, wf := range workflows {
		ev := wf.Evaluation
		if ev == nil {
			continue
		}
		p50 := wf.ProjectedEffectiveTokens
		if mc := wf.MonteCarlo; mc != nil {
			p50 = mc.P50ProjectedEffectiveTokens
		}
		inCI := "No"
		if ev.InCI {
			inCI = "Yes ✓"
		}
		rows = append(rows, evalRow{
			Workflow:    wf.WorkflowID,
			ActualRuns:  ev.ActualRuns,
			ActualET:    formatForecastTokens(ev.ActualEffectiveTokens),
			ForecastP50: formatForecastTokens(p50),
			ErrorAbs:    formatForecastSignedTokens(ev.P50ErrorAbs),
			ErrorPct:    fmt.Sprintf("%.1f%%", ev.P50ErrorPct),
			InCI:        inCI,
		})
	}
	fmt.Fprint(os.Stderr, console.RenderStruct(rows))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
		"Training window ended at the forecast anchor; validation window is the following projection period."))
}

func printVariantBreakdown(wf ForecastWorkflowResult) {
	type variantRow struct {
		Experiment string `json:"experiment" console:"header:Experiment"`
		Variant    string `json:"variant"    console:"header:Variant"`
		Runs       int    `json:"runs"       console:"header:Runs"`
		Fraction   string `json:"fraction"   console:"header:Fraction"`
	}

	fmt.Fprintf(os.Stderr, "  Experiment variants for %s:\n", wf.WorkflowID)
	varRows := make([]variantRow, 0, len(wf.ExperimentVariants))
	for _, v := range wf.ExperimentVariants {
		varRows = append(varRows, variantRow{
			Experiment: v.ExperimentName,
			Variant:    v.Variant,
			Runs:       v.RunCount,
			Fraction:   formatForecastPercent(v.Fraction, wf.SampledRuns > 0),
		})
	}
	fmt.Fprint(os.Stderr, console.RenderStruct(varRows))
	fmt.Fprintln(os.Stderr, "")
}

// ── Format helpers ───────────────────────────────────────────────────────────

// formatForecastPercent formats v as a percentage string.
// hasData must be false when the underlying sample is empty (no runs), in which
// case "N/A" is returned; otherwise the value (including 0%) is formatted.
func formatForecastPercent(v float64, hasData bool) string {
	if !hasData {
		return "N/A"
	}
	return fmt.Sprintf("%.0f%%", v*100)
}

func formatForecastTokens(n int) string {
	if n == 0 {
		return "-"
	}
	if n < 1000 {
		return strconv.Itoa(n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.2fM", float64(n)/1_000_000)
}

// formatForecastSignedTokens formats a signed integer token count, preserving
// the sign so callers can display positive/negative deltas (e.g., error abs).
func formatForecastSignedTokens(n int) string {
	if n == 0 {
		return "0"
	}
	sign := ""
	v := n
	if n < 0 {
		sign = "-"
		v = -n
	}
	return sign + formatForecastTokens(v)
}

func formatTriggerList(triggers []string) string {
	if len(triggers) == 0 {
		return "-"
	}
	if len(triggers) <= 3 {
		return strings.Join(triggers, ", ")
	}
	return strings.Join(triggers[:3], ", ") + fmt.Sprintf(" +%d", len(triggers)-3)
}
