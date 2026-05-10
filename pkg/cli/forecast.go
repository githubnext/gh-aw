package cli

// This file implements the `forecast` command, which samples a workflow's recent
// GitHub Actions run history and projects forward token usage, cost, and yield on
// a per-week or per-month basis.
//
// Workflow metadata (trigger types, concurrency, experiments) is read from the
// workflow's Markdown frontmatter so that projections account for how often the
// workflow is actually expected to fire and how many concurrent runs it supports.

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
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

// costPerEffectiveToken is the approximate USD cost per effective token.
// This mirrors the value used elsewhere in the codebase (e.g. health metrics).
const costPerEffectiveToken = 0.000015

// ForecastEpisodeSummary contains episode-level aggregate metrics derived from
// run history without downloading artifacts.  Episodes are reconstructed from the
// fields available in the GitHub Actions run list (event type, head SHA, branch).
// Dispatch and workflow_call linkages that require aw_info.json are not available
// in this lightweight analysis, so the episode count is a lower-bound estimate.
type ForecastEpisodeSummary struct {
	// SampledEpisodes is the number of distinct episodes detected in the sampled
	// run history.  Each "episode" represents one logical task execution, which may
	// span multiple runs when a workflow dispatches sub-workflows.
	SampledEpisodes int `json:"sampled_episodes"`
	// RunsPerEpisode is the average number of runs per episode (SampledRuns /
	// SampledEpisodes).  Values > 1 indicate orchestrator-style workflows that
	// dispatch multiple sub-workflows per task.
	RunsPerEpisode float64 `json:"runs_per_episode"`
	// AvgEffectiveTokensPerEpisode is the mean effective-token count per episode.
	AvgEffectiveTokensPerEpisode int `json:"avg_effective_tokens_per_episode"`
	// ObservedEpisodesPerPeriod is the projected number of episodes in the forecast
	// period, scaled from the observed episode frequency.
	ObservedEpisodesPerPeriod float64 `json:"observed_episodes_per_period"`
	// ProjectedCostPerEpisode is the projected USD cost per episode
	// (AvgEffectiveTokensPerEpisode × costPerEffectiveToken).
	ProjectedCostPerEpisode float64 `json:"projected_cost_per_episode"`
}

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
	// Yield is the effective throughput: success rate × observed runs per period.
	Yield float64 `json:"yield"`

	// Average per-run metrics (from completed runs).
	AvgEffectiveTokens int     `json:"avg_effective_tokens"`
	AvgDurationSeconds float64 `json:"avg_duration_seconds"`

	// Projected totals for the period.
	ProjectedEffectiveTokens int     `json:"projected_effective_tokens"`
	ProjectedCostUSD         float64 `json:"projected_cost_usd"`

	// EpisodeAnalysis contains episode-level metrics derived from the sampled runs.
	// Nil when no completed runs were available to analyze.
	EpisodeAnalysis *ForecastEpisodeSummary `json:"episode_analysis,omitempty"`

	// MonteCarlo contains the probability distribution of projected costs and
	// effective-token counts derived from a Monte Carlo simulation (10 000 trials).
	// Nil when no completed runs were available.
	MonteCarlo *ForecastMonteCarloSummary `json:"monte_carlo,omitempty"`

	// Trigger information derived from frontmatter.
	ActiveTriggers []string `json:"active_triggers"`
	// ConcurrencyLimit is the workflow-level concurrency limit (0 = unlimited).
	ConcurrencyLimit int `json:"concurrency_limit"`

	// ExperimentVariants contains per-variant forecasts when the workflow defines A/B
	// experiments.  Nil when no experiments are present.
	ExperimentVariants []ForecastVariantResult `json:"experiment_variants,omitempty"`
}

// ForecastVariantResult contains projected metrics split by A/B experiment variant.
type ForecastVariantResult struct {
	ExperimentName string  `json:"experiment_name"`
	Variant        string  `json:"variant"`
	RunCount       int     `json:"run_count"`
	Fraction       float64 `json:"fraction"`
}

// ForecastResult is the top-level output of the forecast command.
type ForecastResult struct {
	Period    string                   `json:"period"`
	AsOf      string                   `json:"as_of"`
	Workflows []ForecastWorkflowResult `json:"workflows"`
}

// RunForecast is the entry point for the forecast command.
func RunForecast(config ForecastConfig) error {
	forecastRunLog.Printf("Running forecast: workflows=%v, days=%d, period=%s", config.WorkflowIDs, config.Days, config.Period)

	// Validate period.
	periodDays, ok := forecastPeriodDays[config.Period]
	if !ok {
		return fmt.Errorf("invalid period %q: must be 'week' or 'month'", config.Period)
	}
	if config.Days != 7 && config.Days != 30 && config.Days != 90 {
		return fmt.Errorf("invalid days value: %d; must be 7, 30, or 90", config.Days)
	}
	if config.SampleSize <= 0 {
		config.SampleSize = 100
	}

	// Resolve the list of workflow IDs to forecast.
	workflowIDs, err := resolveForecastWorkflows(config)
	if err != nil {
		return err
	}
	if len(workflowIDs) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No agentic workflows found to forecast"))
		return nil
	}

	startDate := time.Now().AddDate(0, 0, -config.Days).Format("2006-01-02")
	if !config.Verbose && !config.JSONOutput {
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(
			fmt.Sprintf("Forecasting %d workflow(s) using %d-day history → projecting per %s",
				len(workflowIDs), config.Days, config.Period)))
	}

	spinner := console.NewSpinner("Sampling workflow run history…")
	if !config.Verbose {
		spinner.Start()
	}

	results := make([]ForecastWorkflowResult, 0, len(workflowIDs))
	for _, wfID := range workflowIDs {
		if !config.Verbose {
			spinner.UpdateMessage(fmt.Sprintf("Sampling %s…", wfID))
		}

		result, err := forecastWorkflow(wfID, startDate, config, periodDays)
		if err != nil {
			if !config.Verbose {
				spinner.Stop()
			}
			return fmt.Errorf("forecast failed for workflow %q: %w", wfID, err)
		}
		results = append(results, result)
	}

	if !config.Verbose {
		spinner.Stop()
	}

	// Sort results by projected effective tokens descending for easy comparison.
	sort.Slice(results, func(i, j int) bool {
		return results[i].ProjectedEffectiveTokens > results[j].ProjectedEffectiveTokens
	})

	output := ForecastResult{
		Period:    config.Period,
		AsOf:      time.Now().UTC().Format(time.RFC3339),
		Workflows: results,
	}

	if config.JSONOutput {
		return renderForecastJSON(output)
	}
	return renderForecastTable(output, config)
}

// resolveForecastWorkflows returns the ordered list of workflow IDs to forecast.
// When WorkflowIDs is empty, all agentic workflow IDs in the repository are returned.
// When RepoOverride is set, workflows are discovered via the GitHub API instead of local files.
func resolveForecastWorkflows(config ForecastConfig) ([]string, error) {
	if config.RepoOverride != "" {
		return resolveForecastWorkflowsFromRemote(config.WorkflowIDs, config.RepoOverride, config.Verbose)
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
func resolveForecastWorkflowsFromRemote(ids []string, repoOverride string, verbose bool) ([]string, error) {
	githubWorkflows, err := fetchGitHubWorkflows(repoOverride, verbose)
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
func forecastWorkflow(workflowName, startDate string, config ForecastConfig, periodDays int) (ForecastWorkflowResult, error) {
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
		RepoOverride: config.RepoOverride,
		Verbose:      config.Verbose,
	}

	runs, _, err := listWorkflowRunsWithPagination(opts)
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
		if r.Status == "completed" {
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

	// Effective throughput (yield) accounts for the success rate.
	result.Yield = result.ObservedRunsPerPeriod * result.SuccessRate

	// Projected token usage and cost (point estimate using simple means).
	result.ProjectedEffectiveTokens = int(math.Round(result.ObservedRunsPerPeriod * float64(result.AvgEffectiveTokens)))
	result.ProjectedCostUSD = float64(result.ProjectedEffectiveTokens) * costPerEffectiveToken

	// Monte Carlo simulation: model run-count (Poisson), per-run token usage
	// (bootstrap), and per-run success (Bernoulli) to produce P10/P50/P90 ranges.
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // non-cryptographic simulation RNG
	result.MonteCarlo = runMonteCarlo(etObservations, successCount, result.ObservedRunsPerPeriod, rng)

	// Populate experiment variant fractions from run history when metadata has variants.
	result.ExperimentVariants = computeVariantFractions(result.ExperimentVariants, completed)

	// Build lightweight episode analysis from the completed runs using the fields
	// available in the GitHub Actions run list (no artifact download required).
	result.EpisodeAnalysis = buildForecastEpisodeSummary(completed, config.Days, periodDays)

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

// workflowRunToRunData converts a WorkflowRun (sourced from the GitHub Actions API)
// to a RunData using the fields available without artifact downloads.  Fields that
// require aw_info.json (AwContext, Repository, Ref, SHA, Actor, RunAttempt, …) are
// left as zero values; the episode engine degrades gracefully when they are absent.
func workflowRunToRunData(r WorkflowRun) RunData {
	return RunData{
		RunID:           r.DatabaseID,
		Number:          r.Number,
		WorkflowName:    r.WorkflowName,
		WorkflowPath:    r.WorkflowPath,
		Status:          r.Status,
		Conclusion:      r.Conclusion,
		URL:             r.URL,
		Event:           r.Event,
		Branch:          r.HeadBranch,
		HeadSHA:         r.HeadSha,
		DisplayTitle:    r.DisplayTitle,
		CreatedAt:       r.CreatedAt,
		StartedAt:       r.StartedAt,
		UpdatedAt:       r.UpdatedAt,
		TokenUsage:      r.TokenUsage,
		EffectiveTokens: r.EffectiveTokens,
		EstimatedCost:   r.EstimatedCost,
	}
}

// buildForecastEpisodeSummary derives episode-level metrics from a slice of
// completed WorkflowRun objects using the lightweight episode engine.  Returns nil
// when no runs are provided.
//
// Because only GitHub API fields are available (no aw_info.json artifacts), the
// episode engine can link runs via workflow_run event SHA/branch matching but
// cannot detect dispatch or workflow_call lineage.  The resulting episode count is
// therefore a lower-bound estimate for orchestrator-style workflows.
func buildForecastEpisodeSummary(runs []WorkflowRun, historyDays, periodDays int) *ForecastEpisodeSummary {
	if len(runs) == 0 {
		return nil
	}

	runData := make([]RunData, 0, len(runs))
	for _, r := range runs {
		runData = append(runData, workflowRunToRunData(r))
	}

	// buildEpisodeData returns (episodes, edges); edges are not needed for
	// the lightweight forecast summary so they are intentionally discarded.
	episodes, _ := buildEpisodeData(runData, nil)
	numEpisodes := len(episodes)
	if numEpisodes == 0 {
		return nil
	}

	var totalEpisodeET int
	for _, ep := range episodes {
		totalEpisodeET += ep.TotalEffectiveTokens
	}

	avgETPerEpisode := totalEpisodeET / numEpisodes
	runsPerEpisode := float64(len(runs)) / float64(numEpisodes)
	observedEpisodesPerPeriod := float64(numEpisodes) / float64(historyDays) * float64(periodDays)
	projectedCostPerEpisode := float64(avgETPerEpisode) * costPerEffectiveToken

	return &ForecastEpisodeSummary{
		SampledEpisodes:              numEpisodes,
		RunsPerEpisode:               runsPerEpisode,
		AvgEffectiveTokensPerEpisode: avgETPerEpisode,
		ObservedEpisodesPerPeriod:    observedEpisodesPerPeriod,
		ProjectedCostPerEpisode:      projectedCostPerEpisode,
	}
}

// ── Rendering ───────────────────────────────────────────────────────────────

// renderForecastJSON outputs the forecast result as pretty-printed JSON.
func renderForecastJSON(output ForecastResult) error {
	b, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal forecast JSON: %w", err)
	}
	fmt.Println(string(b))
	return nil
}

// forecastTableRow is a flattened struct used for console table rendering.
type forecastTableRow struct {
	Workflow           string `json:"workflow"                console:"header:Workflow"`
	Runs               int    `json:"runs"                    console:"header:Sampled Runs"`
	SuccessRate        string `json:"success_rate"            console:"header:Success Rate"`
	Yield              string `json:"yield"                   console:"header:Yield/Period"`
	AvgEffectiveTokens string `json:"avg_effective_tokens"    console:"header:Avg ET"`
	ProjectedTokens    string `json:"projected_tokens"        console:"header:Proj. ET (P50)"`
	ProjectedCost      string `json:"projected_cost"          console:"header:Proj. Cost (P50)"`
	CostRange          string `json:"cost_range"              console:"header:80% CI (P10–P90)"`
	Triggers           string `json:"triggers"                console:"header:Triggers"`
}

// renderForecastTable renders the forecast result as a human-readable table.
func renderForecastTable(output ForecastResult, config ForecastConfig) error {
	periodLabel := strings.ToUpper(output.Period[:1]) + output.Period[1:]
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
		fmt.Sprintf("Workflow Forecast — per %s (based on last %d days of history)", periodLabel, config.Days)))
	fmt.Fprintln(os.Stderr, "")

	rows := make([]forecastTableRow, 0, len(output.Workflows))
	for _, wf := range output.Workflows {
		// Use Monte Carlo P50 as the primary cost/ET estimate when available.
		projETStr := formatForecastTokens(wf.ProjectedEffectiveTokens)
		projCostStr := fmt.Sprintf("$%.3f", wf.ProjectedCostUSD)
		ciStr := "-"
		if mc := wf.MonteCarlo; mc != nil {
			projETStr = formatForecastTokens(mc.P50ProjectedEffectiveTokens)
			projCostStr = fmt.Sprintf("$%.3f", mc.P50ProjectedCostUSD)
			ciStr = fmt.Sprintf("$%.3f–$%.3f", mc.P10ProjectedCostUSD, mc.P90ProjectedCostUSD)
		}
		row := forecastTableRow{
			Workflow:           wf.WorkflowID,
			Runs:               wf.SampledRuns,
			SuccessRate:        formatForecastPercent(wf.SuccessRate),
			Yield:              fmt.Sprintf("%.1f", wf.Yield),
			AvgEffectiveTokens: formatForecastTokens(wf.AvgEffectiveTokens),
			ProjectedTokens:    projETStr,
			ProjectedCost:      projCostStr,
			CostRange:          ciStr,
			Triggers:           formatTriggerList(wf.ActiveTriggers),
		}
		rows = append(rows, row)
	}

	fmt.Fprint(os.Stderr, console.RenderStruct(rows))
	fmt.Fprintln(os.Stderr, "")

	// Show episode analysis when any workflow has multi-run episodes.
	anyMultiRunEpisodes := false
	for _, wf := range output.Workflows {
		if wf.EpisodeAnalysis != nil && wf.EpisodeAnalysis.RunsPerEpisode > 1.0 {
			anyMultiRunEpisodes = true
			break
		}
	}
	if anyMultiRunEpisodes {
		printEpisodeBreakdown(output.Workflows)
	}

	// Show experiment variant details when present.
	for _, wf := range output.Workflows {
		if len(wf.ExperimentVariants) > 0 {
			printVariantBreakdown(wf)
		}
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
		fmt.Sprintf("P50 = median; 80%% CI = P10–P90 from %d-trial Monte Carlo simulation.", monteCarloIterations)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
		fmt.Sprintf("Run '%s forecast --json' for full output. Costs use %.0e USD/ET.",
			string(constants.CLIExtensionPrefix), costPerEffectiveToken)))
	return nil
}

// printEpisodeBreakdown renders per-episode projected cost for workflows that have
// multi-run episodes (i.e. orchestrator-style workflows dispatching sub-workflows).
func printEpisodeBreakdown(workflows []ForecastWorkflowResult) {
	type episodeRow struct {
		Workflow          string `json:"workflow"             console:"header:Workflow"`
		Episodes          int    `json:"episodes"             console:"header:Episodes"`
		RunsPerEpisode    string `json:"runs_per_episode"     console:"header:Runs/Episode"`
		AvgETPerEpisode   string `json:"avg_et_per_episode"   console:"header:Avg ET/Episode"`
		EpisodeCostPerPrd string `json:"episode_cost_per_prd" console:"header:Proj. $/Episode"`
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Episode analysis (runs grouped by logical task):"))
	epRows := make([]episodeRow, 0, len(workflows))
	for _, wf := range workflows {
		ep := wf.EpisodeAnalysis
		if ep == nil {
			continue
		}
		epRows = append(epRows, episodeRow{
			Workflow:          wf.WorkflowID,
			Episodes:          ep.SampledEpisodes,
			RunsPerEpisode:    fmt.Sprintf("%.1f", ep.RunsPerEpisode),
			AvgETPerEpisode:   formatForecastTokens(ep.AvgEffectiveTokensPerEpisode),
			EpisodeCostPerPrd: fmt.Sprintf("$%.3f", ep.ProjectedCostPerEpisode),
		})
	}
	fmt.Fprint(os.Stderr, console.RenderStruct(epRows))
	fmt.Fprintln(os.Stderr, "")
}

// printVariantBreakdown renders a small per-variant table for a workflow.
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
			Fraction:   formatForecastPercent(v.Fraction),
		})
	}
	fmt.Fprint(os.Stderr, console.RenderStruct(varRows))
	fmt.Fprintln(os.Stderr, "")
}

// ── Format helpers ───────────────────────────────────────────────────────────

func formatForecastPercent(v float64) string {
	if v == 0 {
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

func formatTriggerList(triggers []string) string {
	if len(triggers) == 0 {
		return "-"
	}
	if len(triggers) <= 3 {
		return strings.Join(triggers, ", ")
	}
	return strings.Join(triggers[:3], ", ") + fmt.Sprintf(" +%d", len(triggers)-3)
}
