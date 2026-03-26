package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

// NewAuditDiffSubcommand creates the audit diff subcommand
func NewAuditDiffSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff <run-id-1> <run-id-2>",
		Short: "Compare workflow run behavior across two workflow runs",
		Long: `Compare workflow run behavior between two workflow runs to detect policy regressions,
new unauthorized domains, behavioral drift, and changes in MCP tool usage or run metrics.

This command downloads artifacts for both runs (using cached data when available),
analyzes their data, and produces a diff showing:
- New domains that appeared in the second run
- Removed domains that were in the first run but not the second
- Status changes (domains that flipped between allowed and denied)
- Volume changes (significant request count changes, >100% threshold)
- Anomaly flags (new denied domains, previously-denied now allowed)
- MCP tool invocation changes (new/removed tools, call count and error count diffs)
- Run metrics comparison (token usage, duration, turns) when cached data is available

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346                     # Compare two runs
  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346 --format markdown    # Markdown output for PR comments
  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346 --json               # JSON for CI integration
  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346 --repo owner/repo    # Specify repository`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			runID1, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid run ID %q: must be a numeric run ID", args[0])
			}
			runID2, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid run ID %q: must be a numeric run ID", args[1])
			}

			outputDir, _ := cmd.Flags().GetString("output")
			verbose, _ := cmd.Flags().GetBool("verbose")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			format, _ := cmd.Flags().GetString("format")
			repoFlag, _ := cmd.Flags().GetString("repo")

			var owner, repo, hostname string
			if repoFlag != "" {
				parts := strings.SplitN(repoFlag, "/", 2)
				if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
					return fmt.Errorf("invalid repository format '%s': expected 'owner/repo'", repoFlag)
				}
				owner = parts[0]
				repo = parts[1]
			}

			return RunAuditDiff(cmd.Context(), runID1, runID2, owner, repo, hostname, outputDir, verbose, jsonOutput, format)
		},
	}

	addOutputFlag(cmd, defaultLogsOutputDir)
	addJSONFlag(cmd)
	addRepoFlag(cmd)
	cmd.Flags().String("format", "pretty", "Output format: pretty, markdown, json")

	return cmd
}

// RunAuditDiff compares behavior between two workflow runs
func RunAuditDiff(ctx context.Context, runID1, runID2 int64, owner, repo, hostname, outputDir string, verbose, jsonOutput bool, format string) error {
	auditDiffLog.Printf("Starting audit diff: run1=%d, run2=%d", runID1, runID2)

	// Auto-detect GHES host from git remote if hostname is not provided
	if hostname == "" {
		hostname = getHostFromOriginRemote()
		if hostname != "github.com" {
			auditDiffLog.Printf("Auto-detected GHES host from git remote: %s", hostname)
		}
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Comparing workflow runs: Run #%d → Run #%d", runID1, runID2)))

	// Load run summaries for both runs
	fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Loading data for run %d...", runID1)))
	summary1, err := loadRunSummaryForDiff(runID1, outputDir, owner, repo, hostname, verbose)
	if err != nil {
		return fmt.Errorf("failed to load data for run %d: %w", runID1, err)
	}

	// Check context cancellation between downloads
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
	}

	fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Loading data for run %d...", runID2)))
	summary2, err := loadRunSummaryForDiff(runID2, outputDir, owner, repo, hostname, verbose)
	if err != nil {
		return fmt.Errorf("failed to load data for run %d: %w", runID2, err)
	}

	// Warn if no firewall data found
	fw1 := summary1.FirewallAnalysis
	fw2 := summary2.FirewallAnalysis
	if fw1 == nil && fw2 == nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No firewall data found in either run. Both runs may predate firewall logging."))
	} else {
		if fw1 == nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No firewall data found for run %d (older run may lack firewall logs)", runID1)))
		}
		if fw2 == nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No firewall data found for run %d", runID2)))
		}
	}

	// Compute the full diff
	diff := computeAuditDiff(runID1, runID2, summary1, summary2)

	// Render output
	if jsonOutput || format == "json" {
		return renderAuditDiffJSON(diff)
	}

	if format == "markdown" {
		renderAuditDiffMarkdown(diff)
		return nil
	}

	// Default: pretty console output
	renderAuditDiffPretty(diff)
	return nil
}
