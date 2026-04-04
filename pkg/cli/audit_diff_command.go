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
		Use:   "diff <base-run-id> <compare-run-id>...",
		Short: "Compare behavior across workflow runs",
		Long: `Compare workflow run behavior between a base run and one or more comparison runs
to detect policy regressions, new unauthorized domains, behavioral drift, and changes in
MCP tool usage, token usage, or run metrics.

The first argument is the base (reference) run. All subsequent arguments are compared
against that base. This enables tracking behavioral drift across multiple runs at once.

This command downloads artifacts for all runs (using cached data when available),
analyzes their data, and produces a diff showing:
- New domains that appeared in the comparison run
- Removed domains that were in the base run but not the comparison
- Status changes (domains that flipped between allowed and denied)
- Volume changes (significant request count changes, >100% threshold)
- Anomaly flags (new denied domains, previously-denied now allowed)
- MCP tool invocation changes (new/removed tools, call count and error count diffs)
- Run metrics comparison (token usage, duration, turns) when cached data is available
- Detailed token usage breakdown (input/output/cache/effective tokens) from firewall proxy

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346                               # Compare two runs
  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346 12347 12348                   # Compare base against 3 runs
  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346 --format markdown             # Markdown output for PR comments
  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346 --json                        # JSON for CI integration
  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346 --repo owner/repo             # Specify repository`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseRunID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid base run ID %q: must be a numeric run ID", args[0])
			}

			compareRunIDs := make([]int64, 0, len(args)-1)
			seen := make(map[int64]bool)
			for _, arg := range args[1:] {
				id, err := strconv.ParseInt(arg, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid run ID %q: must be a numeric run ID", arg)
				}
				if id == baseRunID {
					return fmt.Errorf("comparison run ID %d is the same as the base run ID: cannot diff a run against itself", id)
				}
				if seen[id] {
					return fmt.Errorf("duplicate comparison run ID %d: each run ID must appear only once", id)
				}
				seen[id] = true
				compareRunIDs = append(compareRunIDs, id)
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

			return RunAuditDiff(cmd.Context(), baseRunID, compareRunIDs, owner, repo, hostname, outputDir, verbose, jsonOutput, format)
		},
	}

	addOutputFlag(cmd, defaultLogsOutputDir)
	addJSONFlag(cmd)
	addRepoFlag(cmd)
	cmd.Flags().String("format", "pretty", "Output format: pretty, markdown")

	return cmd
}

// RunAuditDiff compares behavior between a base workflow run and one or more comparison runs.
// The base run is the reference point; each comparison run is diffed against it independently.
func RunAuditDiff(ctx context.Context, baseRunID int64, compareRunIDs []int64, owner, repo, hostname, outputDir string, verbose, jsonOutput bool, format string) error {
	auditDiffLog.Printf("Starting audit diff: base=%d, compare=%v", baseRunID, compareRunIDs)

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

	if len(compareRunIDs) == 1 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Comparing workflow runs: Run #%d → Run #%d", baseRunID, compareRunIDs[0])))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Comparing workflow runs: Run #%d (base) vs %d comparison runs", baseRunID, len(compareRunIDs))))
	}

	// Load base run summary once (shared across all comparisons)
	fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Loading data for base run %d...", baseRunID)))
	baseSummary, err := loadRunSummaryForDiff(baseRunID, outputDir, owner, repo, hostname, verbose)
	if err != nil {
		return fmt.Errorf("failed to load data for base run %d: %w", baseRunID, err)
	}

	diffs := make([]*AuditDiff, 0, len(compareRunIDs))

	for _, compareRunID := range compareRunIDs {
		// Check context cancellation between downloads
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
			return ctx.Err()
		default:
		}

		fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Loading data for run %d...", compareRunID)))
		compareSummary, err := loadRunSummaryForDiff(compareRunID, outputDir, owner, repo, hostname, verbose)
		if err != nil {
			return fmt.Errorf("failed to load data for run %d: %w", compareRunID, err)
		}

		// Warn if no firewall data found for this pair
		fw1 := baseSummary.FirewallAnalysis
		fw2 := compareSummary.FirewallAnalysis
		if fw1 == nil && fw2 == nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No firewall data found for run pair %d→%d. Both runs may predate firewall logging.", baseRunID, compareRunID)))
		} else {
			if fw1 == nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No firewall data found for base run %d (older run may lack firewall logs)", baseRunID)))
			}
			if fw2 == nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No firewall data found for run %d", compareRunID)))
			}
		}

		diff := computeAuditDiff(baseRunID, compareRunID, baseSummary, compareSummary)
		diffs = append(diffs, diff)
	}

	// Render output
	if jsonOutput || format == "json" {
		return renderAuditDiffJSON(diffs)
	}

	if format == "markdown" {
		renderAuditDiffMarkdown(diffs)
		return nil
	}

	// Default: pretty console output
	renderAuditDiffPretty(diffs)
	return nil
}
