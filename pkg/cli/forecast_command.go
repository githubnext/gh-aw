package cli

import (
	"github.com/github/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

// ForecastConfig holds configuration for forecast command execution.
type ForecastConfig struct {
	// WorkflowIDs is the set of workflow IDs to forecast. When empty, all agentic
	// workflows in the repository are included.
	WorkflowIDs []string
	// Days is the historical window used to sample workflow runs.
	Days int
	// Period controls the aggregation granularity: "week" or "month".
	Period string
	// JSONOutput enables machine-readable JSON output.
	JSONOutput bool
	// Verbose enables verbose diagnostic output.
	Verbose bool
	// RepoOverride optionally targets a different repository.
	RepoOverride string
	// SampleSize is the maximum number of completed runs to sample per workflow.
	SampleSize int
}

// NewForecastCommand creates the forecast command.
func NewForecastCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "forecast [workflow]...",
		Short: "Forecast token usage and costs for agentic workflows",
		Long: `Forecast token usage, costs, and yield for agentic workflows by sampling
recent run history and projecting forward on a per-week or per-month basis.

The forecaster downloads a sample of recent workflow runs, computes per-run
metrics (effective tokens, cost, yield, duration), then projects those metrics
over the expected run frequency derived from the workflow's trigger configuration
and its GitHub Actions execution history.

Accounts for:
  - Active trigger types (schedule, pull_request, issues, workflow_dispatch, …)
  - Workflow-level concurrency configuration
  - A/B experiment variants (results are split per variant when present)
  - Observed run frequency from GitHub Actions history

If no workflow arguments are provided, all agentic workflows in the repository
are included and displayed side-by-side for easy comparison.

Multiple workflow IDs may be provided to compare specific workflows.

` + WorkflowIDExplanation + `

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` forecast                        # Forecast all workflows (monthly)
  ` + string(constants.CLIExtensionPrefix) + ` forecast ci-doctor              # Forecast a specific workflow
  ` + string(constants.CLIExtensionPrefix) + ` forecast ci-doctor daily-planner # Compare two workflows
  ` + string(constants.CLIExtensionPrefix) + ` forecast --period week           # Weekly projections
  ` + string(constants.CLIExtensionPrefix) + ` forecast --days 90              # Use 90-day history window
  ` + string(constants.CLIExtensionPrefix) + ` forecast --sample 50            # Sample up to 50 runs per workflow
  ` + string(constants.CLIExtensionPrefix) + ` forecast --json                 # Machine-readable JSON output
  ` + string(constants.CLIExtensionPrefix) + ` forecast --repo owner/repo      # Forecast in another repository`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			days, _ := cmd.Flags().GetInt("days")
			period, _ := cmd.Flags().GetString("period")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			verbose, _ := cmd.Flags().GetBool("verbose")
			repoOverride, _ := cmd.Flags().GetString("repo")
			sampleSize, _ := cmd.Flags().GetInt("sample")

			config := ForecastConfig{
				WorkflowIDs:  args,
				Days:         days,
				Period:       period,
				JSONOutput:   jsonOutput,
				Verbose:      verbose,
				RepoOverride: repoOverride,
				SampleSize:   sampleSize,
			}

			return RunForecast(config)
		},
	}

	cmd.Flags().Int("days", 30, "Historical window in days used to sample run history (7, 30, or 90)")
	cmd.Flags().String("period", "month", "Aggregation period for projections: week or month")
	cmd.Flags().Int("sample", 100, "Maximum number of completed runs to sample per workflow")
	addRepoFlag(cmd)
	addJSONFlag(cmd)

	cmd.ValidArgsFunction = CompleteWorkflowNames

	return cmd
}
