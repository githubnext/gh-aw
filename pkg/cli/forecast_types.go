package cli

// ForecastRunSample holds the data for a single workflow run used in the forecast computation.
// Included in ForecastWorkflowResult.RunSamples so callers and issue templates can list
// the individual runs and their raw AI Credit values for human review.
type ForecastRunSample struct {
	// RunID is the GitHub Actions run ID.
	RunID int64 `json:"run_id"`
	// AIC is the AI Credit cost for this individual run.
	AIC float64 `json:"aic"`
	// Date is the ISO-8601 calendar date the run started (YYYY-MM-DD).
	// Empty when the run's start timestamp is unavailable.
	Date string `json:"date,omitempty"`
	// RunURL links to the GitHub Actions run details page.
	RunURL string `json:"run_url,omitempty"`
}

// ForecastWorkflowResult contains the projected metrics for a single workflow.
type ForecastWorkflowResult struct {
	// WorkflowID is the workflow display name with any file-extension suffix stripped
	// (e.g. ".lock.yml", ".yml"). For most workflows the display name carries no such
	// suffix, so WorkflowID equals the GitHub Actions workflow name as returned by
	// workflow.FindWorkflowName or the GitHub API Name field.
	WorkflowID string `json:"workflow_id"`
	// WorkflowPath is the workflow file path when available (e.g. ".github/workflows/ci.yml").
	WorkflowPath string `json:"workflow_path,omitempty"`
	// Engines lists engine IDs configured by the workflow frontmatter.
	Engines []string `json:"engines,omitempty"`
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
	AvgAIC             float64 `json:"avg_aic"`
	AvgDurationSeconds float64 `json:"avg_duration_seconds"`

	// P50AIC is the 50th-percentile (median) AIC of individual sampled runs.
	P50AIC float64 `json:"p50_aic_per_run"`
	// P95AIC is the 95th-percentile AIC of individual sampled runs
	// (conservative / budget-bound per-run cost estimate).
	P95AIC float64 `json:"p95_aic_per_run"`

	// Projected totals for the configured period.
	ProjectedAIC float64 `json:"projected_aic"`

	// MonteCarlo contains the probability distribution of projected AIC totals
	// for the configured period, derived from a Monte Carlo simulation (10 000 trials).
	// Nil when no completed runs were available.
	MonteCarlo *ForecastMonteCarloSummary `json:"monte_carlo,omitempty"`

	// WeeklyProjectedAIC is the point-estimate projected total AIC over a 7-day window.
	WeeklyProjectedAIC float64 `json:"weekly_projected_aic"`
	// WeeklyMonteCarlo contains the Monte Carlo distribution for the 7-day projection.
	// Nil when no completed runs were available.
	WeeklyMonteCarlo *ForecastMonteCarloSummary `json:"weekly_monte_carlo,omitempty"`

	// MonthlyProjectedAIC is the point-estimate projected total AIC over a 30-day window.
	MonthlyProjectedAIC float64 `json:"monthly_projected_aic"`
	// MonthlyMonteCarlo contains the Monte Carlo distribution for the 30-day projection.
	// Nil when no completed runs were available.
	MonthlyMonteCarlo *ForecastMonteCarloSummary `json:"monthly_monte_carlo,omitempty"`

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

	// RunSamples holds the individual per-run data used in the forecast computation.
	// Each entry records the run ID, raw AIC, and (when available) the run date.
	// Zero-AIC runs are treated as missing data and excluded.
	RunSamples []ForecastRunSample `json:"run_samples,omitempty"`
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
	// ActualAIC is the total AIC value actually consumed
	// in the validation window.
	ActualAIC float64 `json:"actual_aic"`

	// P50ErrorAbs is the signed difference (actual − P50 forecast) in AIC.
	// Positive = actual was higher than forecast; negative = forecast over-estimated.
	P50ErrorAbs float64 `json:"p50_error_abs"`
	// P50ErrorPct is P50ErrorAbs as a percentage of the P50 forecast.
	// NaN-safe: 0 when P50 is 0.
	P50ErrorPct float64 `json:"p50_error_pct"`
	// InCI is true when ActualAIC fell within the P10–P90 confidence
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

// workflowMeta holds parsed metadata from a workflow's Markdown frontmatter.
type workflowMeta struct {
	activeTriggers   []string
	concurrencyLimit int
	variants         []ForecastVariantResult
	engines          []string
}

// forecastTableRow is a flattened struct used for console table rendering.
type forecastTableRow struct {
	Workflow    string `json:"workflow"     console:"header:Workflow"`
	Engines     string `json:"engines"      console:"header:Engines"`
	Runs        int    `json:"runs"         console:"header:Runs"`
	P50PerRun   string `json:"p50_per_run"  console:"header:P50/Run"`
	P95PerRun   string `json:"p95_per_run"  console:"header:P95/Run"`
	WeeklyP50   string `json:"weekly_p50"   console:"header:Weekly (P50)"`
	MonthlyP50  string `json:"monthly_p50"  console:"header:Monthly (P50)"`
	SuccessRate string `json:"success_rate" console:"header:Success Rate"`
	Triggers    string `json:"triggers"     console:"header:Triggers"`
}
