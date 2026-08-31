package cli

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var operationalValueReportLog = logger.New("cli:graders_operational_value_report")

const defaultOperationalValueReportOutputDir = "reports/operational-value"

var operationalValueReportLoadEvaluator = loadOperationalValueReportEvaluator

type OperationalValueReportConfig struct {
	Workflow     string
	RepoOverride string
	Until        string
	OutputDir    string
	CacheDir     string
	Refresh      bool
	JSONOutput   bool
}

type operationalValueReportCommandOutput struct {
	Report    operationalValueReport              `json:"report"`
	Artifacts operationalValueReportArtifactPaths `json:"artifacts"`
	CacheDir  string                              `json:"cacheDir"`
}

func newGradersOperationalValueReportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report <workflow>",
		Short: "Build a complete operational-value history report",
		Long: `Build a complete operational-value history from every completed workflow run since adoption.

The current frozen evaluator is replayed against runs that predate grader artifacts.
Mature observations are cached in digest-scoped UTC weekly files. The command writes
a JSON report, an SVG timeline, and a Markdown report with the frozen evidence contract.`,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` graders operational-value report daily-file-diet
  ` + string(constants.CLIExtensionPrefix) + ` graders operational-value report daily-file-diet --until 2026-08-31T00:00:00Z
  ` + string(constants.CLIExtensionPrefix) + ` graders operational-value report daily-file-diet --refresh --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			until, _ := cmd.Flags().GetString("until")
			outputDir, _ := cmd.Flags().GetString("output")
			cacheDir, _ := cmd.Flags().GetString("cache-dir")
			repoOverride, _ := cmd.Flags().GetString("repo")
			refresh, _ := cmd.Flags().GetBool("refresh")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			return RunOperationalValueReport(cmd.Context(), OperationalValueReportConfig{
				Workflow: args[0], RepoOverride: repoOverride, Until: until,
				OutputDir: outputDir, CacheDir: cacheDir, Refresh: refresh, JSONOutput: jsonOutput,
			})
		},
	}
	cmd.Flags().String("until", "", "UTC endpoint for evidence collection (defaults to now)")
	cmd.Flags().String("cache-dir", "", "Weekly observation cache directory (defaults to the user cache)")
	cmd.Flags().Bool("refresh", false, "Re-evaluate observations instead of reading weekly cache entries")
	addOutputFlag(cmd, defaultOperationalValueReportOutputDir)
	addRepoFlag(cmd)
	addJSONFlag(cmd)
	cmd.ValidArgsFunction = CompleteWorkflowNames
	return cmd
}

func RunOperationalValueReport(ctx context.Context, config OperationalValueReportConfig) error {
	evidenceAt, err := operationalValueReportEvaluatorEvidenceTime(config.Until, time.Now())
	if err != nil {
		return err
	}
	repoSlug, _, evaluatorHost, err := resolveOperationalValueRegradeRepo(config.RepoOverride)
	if err != nil {
		return err
	}
	evaluator, err := operationalValueReportLoadEvaluator(ctx, config.Workflow, evaluatorHost)
	if err != nil {
		return err
	}
	if !strings.EqualFold(repoSlug, evaluator.Definition.Repository) {
		return fmt.Errorf("operational-value evaluator repository %q does not match report repository %q", evaluator.Definition.Repository, repoSlug)
	}
	startAt, err := parseOperationalValueTimestamp(evaluator.Definition.Adoption.AdoptedAt, "adoption.adoptedAt")
	if err != nil {
		return err
	}
	if !evidenceAt.After(startAt) {
		return fmt.Errorf("report endpoint %s must follow workflow adoption %s", evidenceAt.Format(time.RFC3339), startAt.Format(time.RFC3339))
	}
	hostname := ""
	if parsedHost, parseErr := url.Parse(evaluatorHost); parseErr == nil {
		hostname = parsedHost.Hostname()
	}
	workflowFile := strings.TrimSuffix(filepath.Base(evaluator.Definition.SourcePath), ".md") + ".lock.yml"
	operationalValueReportLog.Printf("Listing report runs: repository=%s workflow=%s start=%s end=%s", repoSlug, workflowFile, startAt, evidenceAt)
	runs, err := operationalValueReportListRuns(ctx, repoSlug, hostname, workflowFile, startAt, evidenceAt)
	if err != nil {
		return err
	}
	cacheDir := config.CacheDir
	if cacheDir == "" {
		cacheDir, err = defaultOperationalValueReportCacheRoot()
		if err != nil {
			return err
		}
	}
	observations, stats, err := backfillOperationalValueReportObservations(ctx, evaluator, runs, evidenceAt, cacheDir, evaluatorHost, config.Refresh)
	if err != nil {
		return err
	}
	report := buildOperationalValueReport(evaluator, observations, evidenceAt, stats)
	outputDir := config.OutputDir
	if outputDir == "" {
		outputDir = defaultOperationalValueReportOutputDir
	}
	paths, err := writeOperationalValueReportArtifacts(report, outputDir)
	if err != nil {
		return err
	}
	if config.JSONOutput {
		output := operationalValueReportCommandOutput{Report: report, Artifacts: paths, CacheDir: cacheDir}
		data, err := marshalIndentJSONOrWrap(output, "operational-value report command output")
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(data))
		return nil
	}
	fmt.Fprintln(os.Stdout, console.FormatSuccessMessage(fmt.Sprintf("Built operational-value report for %s from %d workflow runs", evaluator.Definition.WorkflowName, len(runs))))
	fmt.Fprintf(os.Stdout, "Markdown: %s\nSVG: %s\nJSON: %s\nWeekly cache: %s\n", paths.Markdown, paths.SVG, paths.JSON, cacheDir)
	return nil
}
