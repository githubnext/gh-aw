package cli

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var healthLog = logger.New("cli:health")

// healthListWorkflowRuns is the run-listing entry point used by fetchWorkflowRuns.
// It is a variable so that pagination behaviour can be exercised in tests.
var healthListWorkflowRuns = listHealthWorkflowRuns

// HealthConfig holds configuration for health command execution
type HealthConfig struct {
	WorkflowName string
	Days         int
	Threshold    float64
	Verbose      bool
	JSONOutput   bool
	RepoOverride string
	LogsDir      string
	collection   *HealthCollection
}

// NewHealthCommand creates the health command
func NewHealthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health [workflow]",
		Short: "Display workflow health metrics and success rates",
		Long: `Display workflow health metrics, success rates, and execution trends.

Shows health metrics for workflows including:
- Current-revision success/failure rates from recent runs
- Revision-aware recovery evidence and shared failure clusters
- Trend indicators (↑ improving, → stable, ↓ degrading)
- Average execution duration
- Warnings when success rate drops below threshold

When called without a workflow name, displays summary for all workflows.
When called with a specific workflow name, displays detailed metrics for that workflow.

` + WorkflowIDExplanation,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` health                       # Current status of all workflows
  ` + string(constants.CLIExtensionPrefix) + ` health issue-monster         # Detailed metrics for specific workflow
  ` + string(constants.CLIExtensionPrefix) + ` health --days 30             # Look back at most 30 days
  ` + string(constants.CLIExtensionPrefix) + ` health --threshold 90        # Warn if below 90% success rate
  ` + string(constants.CLIExtensionPrefix) + ` health --json                # Output in JSON format
  ` + string(constants.CLIExtensionPrefix) + ` health issue-monster --days 90  # Current status; at most 90 days`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			days, _ := cmd.Flags().GetInt("days")
			threshold, _ := cmd.Flags().GetFloat64("threshold")
			verbose, _ := cmd.Flags().GetBool("verbose")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			repoOverride, _ := cmd.Flags().GetString("repo")
			logsDir, _ := cmd.Flags().GetString("logs-dir")

			var workflowName string
			if len(args) > 0 {
				workflowName = args[0]
			}

			config := HealthConfig{
				WorkflowName: workflowName,
				Days:         days,
				Threshold:    threshold,
				Verbose:      verbose,
				JSONOutput:   jsonOutput,
				RepoOverride: repoOverride,
				LogsDir:      logsDir,
			}

			return RunHealth(config)
		},
	}

	addHealthFlags(cmd)

	// Register completions
	cmd.ValidArgsFunction = CompleteWorkflowNames

	return cmd
}

func addHealthFlags(cmd *cobra.Command) {
	cmd.Flags().Int("days", 7, "Maximum evidence horizon in days (7, 30, or 90)")
	cmd.Flags().Float64("threshold", 80.0, "Success rate threshold for warnings (percentage)")
	cmd.Flags().String("logs-dir", defaultLogsOutputDir, "Existing logs cache for failure clustering (no artifact downloads)")
	addRepoFlag(cmd)
	addJSONFlag(cmd)
}

// RunHealth executes the health command with the given configuration
func RunHealth(config HealthConfig) error {
	healthLog.Printf("Running health check: workflow=%s, days=%d, threshold=%.1f", config.WorkflowName, config.Days, config.Threshold)

	// Validate days parameter
	if config.Days != 7 && config.Days != 30 && config.Days != 90 {
		return fmt.Errorf("invalid days value: %d. Must be 7, 30, or 90", config.Days)
	}

	// Calculate start date
	startDate := time.Now().AddDate(0, 0, -config.Days).Format(time.RFC3339)

	if config.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Fetching workflow runs since "+startDate))
	}

	// Fetch workflow runs from GitHub
	config.collection = &HealthCollection{Host: normalizedGitHubAPIHost(logsRateLimitHost(config.RepoOverride))}
	if err := discoverHealthWorkflows(&config); err != nil {
		return err
	}
	runs, err := fetchWorkflowRuns("", startDate, config.RepoOverride, config.Verbose, config.collection)
	if err != nil {
		return fmt.Errorf("failed to fetch workflow runs: %w", err)
	}

	if len(runs) == 0 && len(config.collection.workflows) == 0 && !config.collection.Partial {
		if config.WorkflowName != "" {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No runs found for workflow '%s' in the last %d days", config.WorkflowName, config.Days)))
			// When JSON output is requested for a specific workflow, still output a valid
			// zero-run JSON structure so callers can parse the result programmatically.
			if config.JSONOutput {
				return displayDetailedHealth(runs, config)
			}
		} else {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No workflow runs found in the last %d days", config.Days)))
		}
		return nil
	}

	if config.WorkflowName != "" {
		// Detailed view for specific workflow
		return displayDetailedHealth(runs, config)
	}

	// Summary view for all workflows
	return displayHealthSummary(runs, config)
}

// displayHealthSummary displays a summary of health metrics for all workflows
func displayHealthSummary(runs []WorkflowRun, config HealthConfig) error {
	healthLog.Printf("Displaying health summary: %d runs", len(runs))

	workflowHealths := calculateHealthWorkflowReports(runs, config)

	// Sort by success rate ascending (lowest first to highlight issues)
	slices.SortFunc(workflowHealths, func(a, b WorkflowHealth) int {
		return cmp.Or(cmp.Compare(a.SuccessRate, b.SuccessRate), cmp.Compare(a.WorkflowPath, b.WorkflowPath), cmp.Compare(a.WorkflowName, b.WorkflowName))
	})

	// Calculate summary (intentional-failure workflows excluded from rollup counts)
	summary := CalculateHealthSummary(workflowHealths, fmt.Sprintf("Current status (up to %d days)", config.Days), config.Threshold)
	summary.Collection = config.collection
	summary.FailureClusters, summary.FailureInvestigation = buildHealthFailureClusters(runs, config)

	// Output results
	if config.JSONOutput {
		return outputHealthJSON(summary)
	}

	return outputHealthTable(summary, config.Threshold)
}

// displayDetailedHealth displays detailed health metrics for a specific workflow
func displayDetailedHealth(runs []WorkflowRun, config HealthConfig) error {
	healthLog.Printf("Displaying detailed health: workflow=%s, %d runs", config.WorkflowName, len(runs))

	// Calculate health metrics
	health := calculateCurrentWorkflowHealth(config.WorkflowName, runs, config)
	health.Collection = config.collection
	health.FailureClusters, health.FailureInvestigation = buildHealthFailureClusters(runs, config)

	// Output results
	if config.JSONOutput {
		jsonBytes, err := marshalIndentJSONOrWrap(health, "workflow health report")
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(jsonBytes))
		return nil
	}

	// Display header message
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow Health: %s (Current status; up to %d days)", config.WorkflowName, config.Days)))
	fmt.Fprintln(os.Stderr, "")

	// Create detailed view
	type DetailedHealth struct {
		Metric string `console:"header:Metric"`
		Value  string `console:"header:Value"`
	}

	details := []DetailedHealth{
		{"Status", health.Status},
		{"Total Runs", strconv.Itoa(health.TotalRuns)},
		{"Successful", strconv.Itoa(health.SuccessCount)},
		{"Failed", strconv.Itoa(health.FailureCount)},
		{"Success Rate", health.DisplayRate},
		{"Trend", health.Trend},
		{"Avg Duration", health.DisplayDur},
		{"Avg Tokens", health.DisplayTokens},
	}

	fmt.Fprint(os.Stderr, console.RenderStruct(details))
	fmt.Fprintln(os.Stderr, "")

	// Display warning if below threshold
	if health.BelowThresh {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Success rate (%.1f%%) is below threshold (%.1f%%)", health.SuccessRate, config.Threshold)))
	} else if health.Status == "healthy" || health.Status == "recovery_observed" {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Success rate (%.1f%%) is above threshold (%.1f%%)", health.SuccessRate, config.Threshold)))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Current revision has insufficient evidence for a healthy status"))
	}

	if err := renderHealthCollection(os.Stderr, health.Collection); err != nil {
		return err
	}
	if err := renderHealthRevisions(os.Stderr, health.Revisions); err != nil {
		return err
	}
	return renderFailureReportToWriter(os.Stderr, health.FailureClusters, health.FailureInvestigation, "", config.Verbose)
}

// outputHealthJSON outputs health summary in JSON format
func outputHealthJSON(summary HealthSummary) error {
	jsonBytes, err := marshalIndentJSONOrWrap(summary, "health summary")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(jsonBytes))
	return nil
}

// outputHealthTable outputs health summary as a formatted table
func outputHealthTable(summary HealthSummary, threshold float64) error {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow Health Summary (%s)", summary.Period)))
	fmt.Fprintln(os.Stderr, "")

	// Render table
	fmt.Fprint(os.Stderr, console.RenderStruct(summary.Workflows))
	fmt.Fprintln(os.Stderr, "")

	// Display summary message
	if summary.BelowThreshold > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("%d workflow(s) below %.0f%% success threshold", summary.BelowThreshold, threshold)))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Run '%s health <workflow-name>' for details", string(constants.CLIExtensionPrefix))))
	} else if summary.InsufficientWorkflows == 0 && (summary.Collection == nil || !summary.Collection.Partial) {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("All evaluated workflows above %.0f%% success threshold", threshold)))
	} else if summary.InsufficientWorkflows > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("%d workflows have insufficient current-revision evidence", summary.InsufficientWorkflows)))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Health collection is partial; see collection limits"))
	}

	if err := renderHealthCollection(os.Stderr, summary.Collection); err != nil {
		return err
	}
	for _, health := range summary.Workflows {
		if err := renderHealthRevisions(os.Stderr, health.Revisions); err != nil {
			return err
		}
	}
	return renderFailureReportToWriter(os.Stderr, summary.FailureClusters, summary.FailureInvestigation, "", false)
}
