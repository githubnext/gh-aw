package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var forecastRunLog = logger.New("cli:forecast_run")

// forecastPeriodDays maps period names to the number of days in a projection window.
var forecastPeriodDays = map[string]int{
	"week":  7,
	"month": 30,
}

// RunForecast is the entry point for the forecast command.
func RunForecast(config ForecastConfig) error {
	forecastRunLog.Printf("Running forecast: workflows=%v, days=%d, period=%s, eval=%v", config.WorkflowIDs, config.Days, config.Period, config.EvalMode)
	ctx, stop, err := runForecastContext(config)
	if err != nil {
		return err
	}
	defer stop()

	// Emit experimental warning so users know this command is not yet stable.
	// Per R-IMPL-040: the warning MUST NOT be emitted when --json is specified,
	// as JSON callers are assumed to be automated pipelines that handle warnings separately.
	runForecastExperimentalWarning(config)

	// Validate period.
	periodDays, err := runForecastValidate(&config)
	if err != nil {
		return err
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
	startDate, validationStartDate, validationEndDate := runForecastDates(config, now, periodDays)

	runForecastPrintStart(workflowIDs, config)
	spinner := runForecastSpinner(config)

	results := make([]ForecastWorkflowResult, 0, len(workflowIDs))
	if err := runForecastWorkflows(runForecastWorkflowsParams{
		Ctx:                 ctx,
		WorkflowIDs:         workflowIDs,
		StartDate:           startDate,
		ValidationStartDate: validationStartDate,
		ValidationEndDate:   validationEndDate,
		Config:              config,
		PeriodDays:          periodDays,
		Spinner:             spinner,
		Now:                 now,
		Results:             &results,
	}); err != nil {
		return err
	}

	if !config.Verbose {
		spinner.Stop()
	}

	// Sort results by Monte Carlo P50 (or point estimate when MC unavailable) descending.
	runForecastSortResults(results)

	return runForecastRender(results, config, now)
}

func runForecastContext(config ForecastConfig) (context.Context, context.CancelFunc, error) {
	if config.TimeoutMinutes < 0 {
		return nil, nil, fmt.Errorf("invalid timeout value: %d; must be >= 0", config.TimeoutMinutes)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	if config.TimeoutMinutes == 0 {
		return ctx, stop, nil
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(config.TimeoutMinutes)*time.Minute)
	return timeoutCtx, func() {
		cancel()
		stop()
	}, nil
}

func runForecastValidate(config *ForecastConfig) (int, error) {
	periodDays, ok := forecastPeriodDays[config.Period]
	if !ok {
		return 0, fmt.Errorf("invalid period %q: must be 'week' or 'month'", config.Period)
	}
	if config.Days != 7 && config.Days != 30 {
		return 0, fmt.Errorf("invalid days value: %d; must be 7 or 30", config.Days)
	}
	if config.SampleSize <= 0 {
		config.SampleSize = 100
	}
	return periodDays, nil
}

func runForecastExperimentalWarning(config ForecastConfig) {
	if !config.JSONOutput {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("forecast is an experimental command and may change without notice"))
	}
}

func runForecastPrintStart(workflowIDs []string, config ForecastConfig) {
	if !config.Verbose && !config.JSONOutput {
		label := fmt.Sprintf("Forecasting %d workflow(s) using %d-day history → projecting per %s",
			len(workflowIDs), config.Days, config.Period)
		fmt.Fprintf(os.Stderr, "%s\n", console.FormatInfoMessage(label))
	}
}

func runForecastSpinner(config ForecastConfig) *console.SpinnerWrapper {
	spinner := console.NewSpinner("Sampling workflow run history…")
	if !config.Verbose {
		spinner.Start()
	}
	return spinner
}

func runForecastDates(config ForecastConfig, now time.Time, periodDays int) (string, string, string) {
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
		startDate = anchor.AddDate(0, 0, -config.Days).Format("2006-01-02")
	}
	return startDate, validationStartDate, validationEndDate
}

type runForecastWorkflowsParams struct {
	Ctx                 context.Context
	WorkflowIDs         []string
	StartDate           string
	ValidationStartDate string
	ValidationEndDate   string
	Config              ForecastConfig
	PeriodDays          int
	Spinner             *console.SpinnerWrapper
	Now                 time.Time
	Results             *[]ForecastWorkflowResult
}

func runForecastWorkflows(p runForecastWorkflowsParams) error {
	for _, wfID := range p.WorkflowIDs {
		if err := runForecastCheckContext(p.Ctx, p.Config, p.Spinner, *p.Results, p.Now); err != nil {
			return err
		}
		if !p.Config.Verbose {
			p.Spinner.UpdateMessage(fmt.Sprintf("Sampling %s…", wfID))
		}
		result, err := forecastWorkflow(p.Ctx, wfID, p.StartDate, p.Config, p.PeriodDays)
		if err != nil {
			if handledErr := runForecastHandleWorkflowError(err, wfID, p.Config, p.Spinner, *p.Results, p.Now); handledErr != nil {
				return handledErr
			}
			continue
		}
		if p.Config.EvalMode {
			result.Evaluation = evaluateForecast(p.Ctx, wfID, result, p.ValidationStartDate, p.ValidationEndDate, p.Config)
		}
		*p.Results = append(*p.Results, result)
	}
	return nil
}

func runForecastCheckContext(ctx context.Context, config ForecastConfig, spinner *console.SpinnerWrapper, results []ForecastWorkflowResult, now time.Time) error {
	if err := ctx.Err(); err != nil {
		if !config.Verbose {
			spinner.Stop()
		}
		emitPartialForecastResults(results, config, now)
		return normalizeForecastRunError(err, config)
	}
	return nil
}

func runForecastHandleWorkflowError(err error, wfID string, config ForecastConfig, spinner *console.SpinnerWrapper, results []ForecastWorkflowResult, now time.Time) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		if !config.Verbose {
			spinner.Stop()
		}
		emitPartialForecastResults(results, config, now)
		return normalizeForecastRunError(err, config)
	}
	if !config.Verbose {
		spinner.Stop()
	}
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping %s: %v", wfID, err)))
	if !config.Verbose {
		spinner.Start()
	}
	return nil
}

func runForecastSortResults(results []ForecastWorkflowResult) {
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
}

func runForecastRender(results []ForecastWorkflowResult, config ForecastConfig, now time.Time) error {
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
