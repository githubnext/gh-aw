package cli

import (
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

var experimentsLog = logger.New("cli:experiments_command")

type ExperimentsListConfig struct {
	RepoOverride string
	JSONOutput   bool
}

// ExperimentsAnalyzeConfig holds configuration for the experiments analyze subcommand.
type ExperimentsAnalyzeConfig struct {
	ExperimentName string
	RepoOverride   string
	JSONOutput     bool
}

// NewExperimentsCommand creates the experiments command with its subcommands.
func NewExperimentsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "experiments",
		Short: "List and analyze experiment workflow branches in the repository",
		Long: `List and analyze experiment workflow branches in the repository.

Experiments are tracked via git branches with the "experiments/" prefix (e.g.,
experiments/my-workflow). Each branch stores a state.jsonl or state.json file
written by the workflow's pick_experiment step, containing variant counts and
run history.

Available subcommands:
  - list    - List all experiment workflow branches (default)
  - analyze - Analyze a specific experiment workflow in detail`,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` experiments                        # List all experiments (default)
  ` + string(constants.CLIExtensionPrefix) + ` experiments list                   # List all experiments
  ` + string(constants.CLIExtensionPrefix) + ` experiments list --json            # Output in JSON format
  ` + string(constants.CLIExtensionPrefix) + ` experiments analyze my-workflow    # Analyze experiments/my-workflow
  ` + string(constants.CLIExtensionPrefix) + ` experiments analyze my-workflow --json  # Analyze in JSON format`,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			repoOverride, _ := cmd.Flags().GetString("repo")
			return RunExperimentsList(ExperimentsListConfig{
				RepoOverride: repoOverride,
				JSONOutput:   jsonOutput,
			})
		},
	}

	addJSONFlag(cmd)
	addRepoFlag(cmd)

	cmd.AddCommand(NewExperimentsListSubcommand())
	cmd.AddCommand(NewExperimentsAnalyzeSubcommand())

	return cmd
}

// NewExperimentsListSubcommand creates the experiments list subcommand.
func NewExperimentsListSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all experiment workflow branches",
		Long: `List all experiment workflow branches in the repository.

Reads the state.jsonl/state.json file from each experiments/* branch and shows a summary
of each workflow's A/B experiments: number of experiments defined, total runs,
and timestamp of the most recent run.`,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` experiments list                             # List all experiments
  ` + string(constants.CLIExtensionPrefix) + ` experiments list --json                      # Output in JSON format
  ` + string(constants.CLIExtensionPrefix) + ` experiments list --repo owner/repo           # List from a specific repository`,
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			repoOverride, _ := cmd.Flags().GetString("repo")
			return RunExperimentsList(ExperimentsListConfig{
				RepoOverride: repoOverride,
				JSONOutput:   jsonOutput,
			})
		},
	}

	addJSONFlag(cmd)
	addRepoFlag(cmd)

	return cmd
}

// NewExperimentsAnalyzeSubcommand creates the experiments analyze subcommand.
func NewExperimentsAnalyzeSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze <experiment>",
		Short: "Analyze a specific experiment workflow in detail",
		Long: `Analyze a specific experiment workflow in detail.

The experiment argument is the workflow ID (branch name without the "experiments/"
prefix, e.g., "my-workflow" for the "experiments/my-workflow" branch).

Reads the state.jsonl/state.json file from the branch and shows per-variant counts, total
runs, and the most recent run assignments.`,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` experiments analyze my-workflow              # Analyze experiments/my-workflow
  ` + string(constants.CLIExtensionPrefix) + ` experiments analyze my-workflow --json       # Output in JSON format
  ` + string(constants.CLIExtensionPrefix) + ` experiments analyze my-workflow --repo owner/repo  # Analyze in a specific repository`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jsonOutput, _ := cmd.Flags().GetBool("json")
			repoOverride, _ := cmd.Flags().GetString("repo")
			return RunExperimentsAnalyze(ExperimentsAnalyzeConfig{
				ExperimentName: args[0],
				RepoOverride:   repoOverride,
				JSONOutput:     jsonOutput,
			})
		},
	}

	addJSONFlag(cmd)
	addRepoFlag(cmd)

	return cmd
}

// RunExperimentsList lists all experiment branches.
func RunExperimentsList(config ExperimentsListConfig) error {
	experimentsLog.Printf("Listing experiments: repo=%s, json=%v", config.RepoOverride, config.JSONOutput)

	var experiments []ExperimentInfo
	var err error

	if config.RepoOverride != "" {
		experiments, err = fetchRemoteExperiments(config.RepoOverride)
	} else {
		experiments, err = fetchLocalExperiments()
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
		return nil
	}

	if config.JSONOutput {
		jsonBytes, err := marshalIndentJSONOrWrap(experiments, "experiments list")
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(jsonBytes))
		return nil
	}

	if len(experiments) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No experiment workflow branches found (branches matching experiments/* pattern)."))
		return nil
	}

	count := len(experiments)
	if count == 1 {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Found 1 experiment workflow"))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Found %d experiment workflows", count)))
	}
	fmt.Fprint(os.Stderr, console.RenderStruct(experiments))

	return nil
}

// RunExperimentsAnalyze analyzes a specific experiment branch.
func RunExperimentsAnalyze(config ExperimentsAnalyzeConfig) error {
	experimentsLog.Printf("Analyzing experiment: name=%s, repo=%s, json=%v",
		config.ExperimentName, config.RepoOverride, config.JSONOutput)

	branchName := experimentsBranchPrefix + config.ExperimentName

	// Load experiment configs and evals from the workflow frontmatter to enrich the statistical
	// output with hypothesis text, analysis_type, min_samples, guardrail thresholds, and resolved
	// eval metric questions.
	// Config loading is best-effort: failures are silently ignored and analysis falls back to
	// defaults (min_samples=20, equal expected proportions, no hypothesis displayed).
	// This ensures the command remains functional even when the workflow .md file is absent
	// (e.g., when analysing experiments from a remote repository without the workflow checked out).
	var frontmatterResult experimentFrontmatterResult
	if config.RepoOverride != "" {
		frontmatterResult = loadRemoteExperimentConfigs(config.RepoOverride, config.ExperimentName)
	} else {
		frontmatterResult = loadLocalExperimentConfigs(config.ExperimentName)
	}
	experimentsLog.Printf("Loaded %d experiment config(s) for %s", len(frontmatterResult.ExperimentConfigs), config.ExperimentName)

	var details *ExperimentDetails
	var err error

	if config.RepoOverride != "" {
		details, err = fetchRemoteExperimentDetails(config.RepoOverride, branchName, config.ExperimentName)
	} else {
		details, err = fetchLocalExperimentDetails(branchName, config.ExperimentName)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
		return nil
	}

	var metricEvalResults map[string]MetricEvalResults
	if config.RepoOverride != "" {
		metricEvalResults = loadRemoteMetricEvalResults(config.RepoOverride, details.WorkflowID)
	} else {
		metricEvalResults = loadLocalMetricEvalResults(details.WorkflowID)
	}

	// Compute statistical analyses for each named experiment.
	details.Analyses = computeExperimentAnalyses(
		details.Experiments,
		frontmatterResult.ExperimentConfigs,
		frontmatterResult.Evals,
		metricEvalResults,
	)

	if config.JSONOutput {
		jsonBytes, err := marshalIndentJSONOrWrap(details, "experiment details")
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(jsonBytes))
		return nil
	}

	printExperimentDetails(details)
	return nil
}

// computeExperimentAnalyses computes statistical analyses for all named experiments.
// configs maps experiment names to their configuration; values may be nil.
// evals provides the eval definitions for resolving eval-backed metric references; may be nil.
func computeExperimentAnalyses(
	experiments []ExperimentVariantStats,
	configs map[string]*workflow.ExperimentConfig,
	evals *workflow.EvalsConfig,
	metricEvalResults map[string]MetricEvalResults,
) []ExperimentAnalysis {
	if len(experiments) == 0 {
		return nil
	}
	analyses := make([]ExperimentAnalysis, 0, len(experiments))
	for _, exp := range experiments {
		var cfg *workflow.ExperimentConfig
		if configs != nil {
			cfg = configs[exp.Name]
		}
		analyses = append(analyses, computeExperimentAnalysis(exp, cfg, evals, metricEvalResults))
	}
	return analyses
}
