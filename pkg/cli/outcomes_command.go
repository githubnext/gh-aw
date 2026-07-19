package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/github"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var outcomesLog = logger.New("cli:outcomes")

// NewOutcomesCommand creates the outcomes command
func NewOutcomesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "outcomes <run-id>",
		Short: "Check what happened to a workflow run's safe outputs",
		Long: `Evaluate the outcomes of safe output actions from a workflow run.

For each safe output (created issue, PR, comment, label, etc.), checks the current
state of the GitHub object to determine whether the action was accepted, rejected,
ignored, or is still pending.

This answers the question: "Did this workflow's actions actually help?"`,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` outcomes 1234567890                # Check outcomes for a specific run
  ` + string(constants.CLIExtensionPrefix) + ` outcomes 1234567890 --json         # JSON output
  ` + string(constants.CLIExtensionPrefix) + ` outcomes 1234567890 --repo o/r     # Specify repository
  ` + string(constants.CLIExtensionPrefix) + ` outcomes 1234567890 -v             # Verbose output`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			repoOverride, _ := cmd.Flags().GetString("repo")
			outputDir, _ := cmd.Flags().GetString("output")
			outcomesDir, _ := cmd.Flags().GetString("outcomes-dir")

			runID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid run ID %q: %w", args[0], err)
			}

			return RunOutcomes(OutcomesConfig{
				RunID:        runID,
				Verbose:      verbose,
				JSONOutput:   jsonOutput,
				RepoOverride: repoOverride,
				OutputDir:    outputDir,
				OutcomesDir:  outcomesDir,
			})
		},
	}

	addJSONFlag(cmd)
	addRepoFlag(cmd)
	addOutputFlag(cmd, "")
	cmd.Flags().String("outcomes-dir", "", "Write outcome JSONL to this directory for OTLP export")
	cmd.AddCommand(NewOutcomesHistorySubcommand())

	return cmd
}

// OutcomesConfig holds configuration for the outcomes command.
type OutcomesConfig struct {
	RunID        int64
	Verbose      bool
	JSONOutput   bool
	RepoOverride string
	OutputDir    string
	OutcomesDir  string
}

// OutcomesData is the structured output of the outcomes command.
type OutcomesData struct {
	RunID    int64           `json:"run_id"`
	Workflow string          `json:"workflow,omitempty"`
	Items    []OutcomeReport `json:"items"`
	Summary  OutcomeSummary  `json:"summary"`
}

// RunOutcomes executes the outcomes evaluation for a single run.
func RunOutcomes(config OutcomesConfig) error {
	outcomesLog.Printf("Evaluating outcomes for run %d", config.RunID)

	repo, err := runOutcomesResolveRepo(config.RepoOverride)
	if err != nil {
		return err
	}
	owner, repoName, hostname := runOutcomesRepoParts(repo)
	runDir := runOutcomesRunDir(config)
	summary, cached := loadRunSummary(runDir, config.Verbose)
	items, err := runOutcomesLoadItems(config, runDir, repo, owner, repoName, hostname, cached, summary)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		return runOutcomesNoItems(config)
	}

	if config.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Evaluating outcomes for %d safe output items...", len(items))))
	}

	// Run the evaluations
	mapping := github.LoadObjectiveMappingFromConfig()
	reports := EvaluateOutcomes(items, repo, mapping)
	outcomeSummary := ComputeOutcomeSummary(reports, mapping)

	// Write outcome JSONL if requested (for OTLP export or downstream processing).
	// The --outcomes-dir flag takes precedence over the GH_AW_OUTCOMES_DIR env var.
	outcomesDir := config.OutcomesDir
	if outcomesDir == "" {
		outcomesDir = lookupEnv("GH_AW_OUTCOMES_DIR")
	}
	if outcomesDir != "" {
		writeOutcomeJSONL(outcomesDir, config.RunID, reports)
	}

	workflowName := runOutcomesWorkflowName(cached, summary)

	if config.JSONOutput {
		return runOutcomesJSON(config, reports, outcomeSummary, workflowName)
	}
	runOutcomesConsole(config, reports, outcomeSummary, workflowName)
	return nil
}

func runOutcomesResolveRepo(repoOverride string) (string, error) {
	if repoOverride != "" {
		return repoOverride, nil
	}
	slug, err := GetCurrentRepoSlug()
	if err != nil {
		return "", fmt.Errorf("could not determine repository: %w", err)
	}
	return slug, nil
}

func runOutcomesRepoParts(repo string) (string, string, string) {
	var owner, repoName, hostname string
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) == 2 {
		owner = parts[0]
		repoName = parts[1]
	}
	return owner, repoName, hostname
}

func runOutcomesRunDir(config OutcomesConfig) string {
	outputDir := config.OutputDir
	if outputDir == "" {
		outputDir = defaultLogsOutputDir
	}
	return filepath.Join(outputDir, fmt.Sprintf("run-%d", config.RunID))
}

func runOutcomesLoadItems(config OutcomesConfig, runDir string, repo string, owner string, repoName string, hostname string, cached bool, summary *RunSummary) ([]CreatedItemReport, error) {
	var items []CreatedItemReport
	if cached && summary != nil {
		items = runOutcomesExtractItems(runDir, repo)
		if config.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Loaded %d safe output items from cache", len(items))))
		}
	}
	if len(items) > 0 {
		return items, nil
	}
	if config.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Downloading artifacts for run %d...", config.RunID)))
	}
	ctx := context.Background()
	err := downloadRunArtifacts(ctx, downloadArtifactsOptions{runID: config.RunID, outputDir: runDir, verbose: config.Verbose, owner: owner, repo: repoName, hostname: hostname})
	if err != nil {
		return nil, fmt.Errorf("failed to download artifacts for run %d: %w", config.RunID, err)
	}
	return runOutcomesExtractItems(runDir, repo), nil
}

func runOutcomesExtractItems(runDir string, repo string) []CreatedItemReport {
	items := extractCreatedItemsFromManifest(runDir)
	if len(items) == 0 {
		items = extractCreatedItemsFromManifest(filepath.Join(runDir, "safe-outputs-items"))
	}
	return enrichItemsFromAgentOutput(items, runDir, repo)
}

func runOutcomesNoItems(config OutcomesConfig) error {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No safe output items found for this run"))
	if config.JSONOutput {
		data := OutcomesData{RunID: config.RunID, Items: []OutcomeReport{}, Summary: OutcomeSummary{}}
		out, _ := json.MarshalIndent(data, "", "  ")
		fmt.Fprintln(os.Stdout, string(out))
	}
	return nil
}

func runOutcomesWorkflowName(cached bool, summary *RunSummary) string {
	if cached && summary != nil {
		return summary.Run.WorkflowName
	}
	return ""
}

func runOutcomesJSON(config OutcomesConfig, reports []OutcomeReport, outcomeSummary OutcomeSummary, workflowName string) error {
	data := OutcomesData{RunID: config.RunID, Workflow: workflowName, Items: reports, Summary: outcomeSummary}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Fprintln(os.Stdout, string(out))
	return nil
}

func runOutcomesConsole(config OutcomesConfig, reports []OutcomeReport, outcomeSummary OutcomeSummary, workflowName string) {
	if workflowName != "" {
		fmt.Fprintf(os.Stderr, "\n%s\n", console.FormatInfoMessage(fmt.Sprintf("Outcomes for %s (run %d)", workflowName, config.RunID)))
	} else {
		fmt.Fprintf(os.Stderr, "\n%s\n", console.FormatInfoMessage(fmt.Sprintf("Outcomes for run %d", config.RunID)))
	}
	runOutcomesConsoleItems(reports)
	runOutcomesConsoleSummary(outcomeSummary)
}

func runOutcomesConsoleItems(reports []OutcomeReport) {
	fmt.Fprintln(os.Stderr)
	for _, r := range reports {
		resultStr := string(r.Result)
		if r.Detail != "" {
			resultStr += " (" + r.Detail + ")"
		}
		numStr := ""
		if r.ObjectNumber > 0 {
			numStr = fmt.Sprintf("#%d", r.ObjectNumber)
		}
		fmt.Fprintf(os.Stderr, "  %-28s %-12s %-40s %s\n", r.Type, numStr, resultStr, runOutcomesTimeString(r.TimeToOutcomeHours))
	}
	fmt.Fprintln(os.Stderr)
}

func runOutcomesTimeString(hours float64) string {
	if hours <= 0 {
		return ""
	}
	if hours < 1 {
		return fmt.Sprintf("%.0fm", hours*60)
	}
	return fmt.Sprintf("%.1fh", hours)
}

func runOutcomesConsoleSummary(outcomeSummary OutcomeSummary) {
	resolved := outcomeSummary.Accepted + outcomeSummary.Rejected
	fmt.Fprintf(os.Stderr, "  Acceptance: %d/%d", outcomeSummary.Accepted, resolved)
	if resolved > 0 {
		fmt.Fprintf(os.Stderr, " (%.0f%%)", outcomeSummary.AcceptanceRate*100)
	}
	fmt.Fprintln(os.Stderr)
	if outcomeSummary.Accepted > 0 {
		fmt.Fprintf(os.Stderr, "  Zero-touch: %d/%d (%.0f%%)\n", outcomeSummary.ZeroTouch, outcomeSummary.Accepted, outcomeSummary.ZeroTouchRate*100)
	}
	if outcomeSummary.Rejected > 0 {
		fmt.Fprintf(os.Stderr, "  Waste: %d/%d (%.0f%%)\n", outcomeSummary.Rejected, outcomeSummary.Total, outcomeSummary.WasteRate*100)
	}
	if outcomeSummary.Pending > 0 {
		fmt.Fprintf(os.Stderr, "  Pending: %d\n", outcomeSummary.Pending)
	}
	if outcomeSummary.MedianTimeToOutcome > 0 {
		fmt.Fprintf(os.Stderr, "  Median time to outcome: %.1fh\n", outcomeSummary.MedianTimeToOutcome)
	}
	fmt.Fprintln(os.Stderr)
}
