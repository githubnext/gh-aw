package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/spf13/cobra"
)

var auditReportCommandLog = logger.New("cli:audit_report")

// NewAuditReportSubcommand creates the audit report subcommand
func NewAuditReportSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate a cross-run security audit report",
		Long: `Generate a comprehensive audit report across multiple workflow runs.
Designed for security reviews, compliance checks, and feeding debugging/optimization agents.

This command:
- Fetches recent workflow runs (optionally filtered by workflow name)
- Downloads firewall artifacts in parallel
- Aggregates firewall data across runs
- Outputs an executive summary, domain inventory, and per-run breakdown

Output: Markdown by default (suitable for security reviews, piping to files,
$GITHUB_STEP_SUMMARY). Also supports JSON.

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` audit report --workflow "agent-task" --last 10  # Report on last 10 runs of a workflow
  ` + string(constants.CLIExtensionPrefix) + ` audit report                                     # Report on recent runs (default: last 20)
  ` + string(constants.CLIExtensionPrefix) + ` audit report --workflow "agent-task" --last 5 --json  # JSON for dashboards
  ` + string(constants.CLIExtensionPrefix) + ` audit report --format pretty                     # Console-formatted output
  ` + string(constants.CLIExtensionPrefix) + ` audit report --repo owner/repo --last 10          # Report on a specific repository`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowName, _ := cmd.Flags().GetString("workflow")
			last, _ := cmd.Flags().GetInt("last")
			outputDir, _ := cmd.Flags().GetString("output")
			verbose, _ := cmd.Flags().GetBool("verbose")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			format, _ := cmd.Flags().GetString("format")
			repoFlag, _ := cmd.Flags().GetString("repo")

			var owner, repo string
			if repoFlag != "" {
				parts := strings.Split(repoFlag, "/")
				if len(parts) < 2 {
					return fmt.Errorf("invalid repository format '%s': expected '[HOST/]owner/repo'", repoFlag)
				}
				ownerPart := parts[len(parts)-2]
				repoPart := parts[len(parts)-1]
				if ownerPart == "" || repoPart == "" {
					return fmt.Errorf("invalid repository format '%s': expected '[HOST/]owner/repo'", repoFlag)
				}
				owner = ownerPart
				repo = repoPart
			}

			return RunAuditReport(cmd.Context(), RunAuditReportConfig{
				WorkflowName: workflowName,
				Last:         last,
				OutputDir:    outputDir,
				Verbose:      verbose,
				JSONOutput:   jsonOutput,
				Format:       format,
				Owner:        owner,
				Repo:         repo,
			})
		},
	}

	addOutputFlag(cmd, defaultLogsOutputDir)
	addJSONFlag(cmd)
	addRepoFlag(cmd)
	cmd.Flags().StringP("workflow", "w", "", "Filter by workflow name or filename")
	cmd.Flags().Int("last", 20, "Number of recent runs to analyze (max 50)")
	cmd.Flags().String("format", "markdown", "Output format: markdown, pretty, json")

	return cmd
}

// RunAuditReportConfig holds the configuration for RunAuditReport.
type RunAuditReportConfig struct {
	WorkflowName string
	Last         int
	OutputDir    string
	Verbose      bool
	JSONOutput   bool
	Format       string
	Owner        string
	Repo         string
}

// RunAuditReport generates a cross-run firewall audit report.
func RunAuditReport(ctx context.Context, cfg RunAuditReportConfig) error {
	auditReportCommandLog.Printf("Starting audit report: workflow=%s, last=%d", cfg.WorkflowName, cfg.Last)

	// Clamp --last to bounds
	if cfg.Last <= 0 {
		cfg.Last = 20
	}
	if cfg.Last > maxAuditReportRuns {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("--last clamped to maximum of %d", maxAuditReportRuns)))
		cfg.Last = maxAuditReportRuns
	}

	// Auto-detect GHES host from git remote
	hostname := getHostFromOriginRemote()
	if hostname != "" && hostname != "github.com" {
		auditReportCommandLog.Printf("Auto-detected GHES host from git remote: %s", hostname)

		// If GH_HOST is not already set, use the detected hostname so gh CLI targets GHES
		if _, ok := os.LookupEnv("GH_HOST"); !ok {
			if err := os.Setenv("GH_HOST", hostname); err != nil {
				auditReportCommandLog.Printf("Failed to set GH_HOST from git remote: %v", err)
			} else if cfg.Verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Using GitHub host %s (from git remote)", hostname)))
			}
		}
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
	}

	// Build repo override string for API calls
	repoOverride := ""
	if cfg.Owner != "" && cfg.Repo != "" {
		repoOverride = cfg.Owner + "/" + cfg.Repo
	}

	// Fetch workflow runs
	workflowLabel := cfg.WorkflowName
	if workflowLabel == "" {
		workflowLabel = "all agentic workflows"
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Fetching last %d runs for %s...", cfg.Last, workflowLabel)))

	runs, _, err := listWorkflowRunsWithPagination(ListWorkflowRunsOptions{
		WorkflowName: cfg.WorkflowName,
		Limit:        cfg.Last,
		RepoOverride: repoOverride,
		Verbose:      cfg.Verbose,
		TargetCount:  cfg.Last,
	})
	if err != nil {
		return fmt.Errorf("failed to list workflow runs: %w", err)
	}

	if len(runs) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No workflow runs found matching the criteria."))
		return nil
	}

	// Cap to requested count
	if len(runs) > cfg.Last {
		runs = runs[:cfg.Last]
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d runs. Downloading artifacts in parallel...", len(runs))))

	// Download artifacts concurrently
	results := downloadRunArtifactsConcurrent(ctx, runs, cfg.OutputDir, cfg.Verbose, cfg.Last, repoOverride)

	// Build aggregation inputs — include skipped/error runs as HasData=false entries
	// so the report accurately counts RunsWithoutData for expired or missing artifacts
	inputs := make([]crossRunInput, 0, len(results))
	for _, r := range results {
		inputs = append(inputs, crossRunInput{
			RunID:            r.Run.DatabaseID,
			WorkflowName:     r.Run.WorkflowName,
			Conclusion:       r.Run.Conclusion,
			Duration:         r.Run.Duration,
			FirewallAnalysis: r.FirewallAnalysis,
			Metrics:          r.Metrics,
			MCPToolUsage:     r.MCPToolUsage,
			MCPFailures:      r.MCPFailures,
			ErrorCount:       r.Run.ErrorCount,
		})
	}

	if len(inputs) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No runs could be processed."))
		return nil
	}

	// Build cross-run report
	report := buildCrossRunAuditReport(inputs)

	// Render output
	if cfg.JSONOutput || cfg.Format == "json" {
		return renderCrossRunReportJSON(report)
	}

	if cfg.Format == "pretty" {
		renderCrossRunReportPretty(report)
		return nil
	}

	// Default: markdown
	renderCrossRunReportMarkdown(report)
	return nil
}
