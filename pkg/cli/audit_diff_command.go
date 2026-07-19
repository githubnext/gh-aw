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

const auditDiffSubcommandLong = `Deprecated: pass multiple run IDs directly to the audit command instead.

  gh aw audit <base-run-id> <compare-run-id>...

Compare workflow run behavior between a base run and one or more comparison runs
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
- Detailed token usage breakdown (input/output/cache + AI Credits) from firewall proxy`

const auditDiffSubcommandExample = `  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346                               # Compare two runs
  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346 12347 12348                   # Compare base against 3 runs
  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346 --format markdown             # Markdown output for PR comments
  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346 --json                        # JSON for CI integration
  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346 --repo owner/repo             # Specify repository`

// NewAuditDiffSubcommand creates the audit diff subcommand.
// Deprecated: pass multiple run IDs directly to `audit` instead (e.g. `gh aw audit <base> <compare...>`).
// This subcommand is hidden and kept for backward compatibility only.
func NewAuditDiffSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "diff <base-run-id> <compare-run-id>...",
		Short:   "Compare behavior across workflow runs",
		Hidden:  true,
		Long:    auditDiffSubcommandLong,
		Example: auditDiffSubcommandExample,
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return newAuditDiffSubcommandRunE(cmd, args)
		},
	}

	addOutputFlag(cmd, defaultLogsOutputDir)
	addJSONFlag(cmd)
	addRepoFlag(cmd)
	cmd.Flags().String("format", "pretty", "Output format: pretty, markdown")
	cmd.Flags().StringSlice("artifacts", nil, "Artifact sets to download (default: all, because auditing requires comprehensive artifacts for analysis). Valid sets: "+strings.Join(ValidArtifactSetNames(), ", "))

	return cmd
}

func newAuditDiffSubcommandRunE(cmd *cobra.Command, args []string) error {
	baseRunID, compareRunIDs, err := newAuditDiffSubcommandParseRunIDs(args)
	if err != nil {
		return err
	}
	opts, err := newAuditDiffSubcommandOptions(cmd)
	if err != nil {
		return err
	}
	return RunAuditDiff(cmd.Context(), baseRunID, compareRunIDs, opts)
}

func newAuditDiffSubcommandParseRunIDs(args []string) (int64, []int64, error) {
	baseRunID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid base run ID %q: must be a numeric run ID", args[0])
	}

	compareRunIDs := make([]int64, 0, len(args)-1)
	seen := make(map[int64]bool)
	for _, arg := range args[1:] {
		id, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return 0, nil, fmt.Errorf("invalid run ID %q: must be a numeric run ID", arg)
		}
		if id == baseRunID {
			return 0, nil, fmt.Errorf("comparison run ID %d is the same as the base run ID: cannot diff a run against itself", id)
		}
		if seen[id] {
			return 0, nil, fmt.Errorf("duplicate comparison run ID %d: each run ID must appear only once", id)
		}
		seen[id] = true
		compareRunIDs = append(compareRunIDs, id)
	}
	return baseRunID, compareRunIDs, nil
}

func newAuditDiffSubcommandOptions(cmd *cobra.Command) (AuditOptions, error) {
	outputDir, _ := cmd.Flags().GetString("output")
	verbose, _ := cmd.Flags().GetBool("verbose")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	format, _ := cmd.Flags().GetString("format")
	repoFlag, _ := cmd.Flags().GetString("repo")
	artifacts, _ := cmd.Flags().GetStringSlice("artifacts")

	owner, repo, err := newAuditDiffSubcommandRepo(repoFlag)
	if err != nil {
		return AuditOptions{}, err
	}
	return AuditOptions{Owner: owner, Repo: repo, OutputDir: outputDir, Verbose: verbose, JSONOutput: jsonOutput, Format: format, ArtifactSets: artifacts}, nil
}

func newAuditDiffSubcommandRepo(repoFlag string) (string, string, error) {
	if repoFlag == "" {
		return "", "", nil
	}
	parts := strings.SplitN(repoFlag, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repository format '%s': expected 'owner/repo'", repoFlag)
	}
	return parts[0], parts[1], nil
}

// RunAuditDiff compares behavior between a base workflow run and one or more comparison runs.
// The base run is the reference point; each comparison run is diffed against it independently.
func RunAuditDiff(ctx context.Context, baseRunID int64, compareRunIDs []int64, opts AuditOptions) error {
	owner := opts.Owner
	repo := opts.Repo
	hostname := opts.Hostname
	outputDir := opts.OutputDir
	verbose := opts.Verbose
	format := opts.Format
	artifactSets := opts.ArtifactSets

	auditDiffLog.Printf("Starting audit diff: base=%d, compare=%v", baseRunID, compareRunIDs)

	artifactFilter, err := runAuditDiffArtifactFilter(artifactSets, verbose)
	if err != nil {
		return err
	}

	hostname = runAuditDiffHostname(hostname)

	if err := runAuditDiffCheckContext(ctx); err != nil {
		return err
	}

	if len(compareRunIDs) == 1 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Comparing workflow runs: Run #%d → Run #%d", baseRunID, compareRunIDs[0])))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Comparing workflow runs: Run #%d (base) vs %d comparison runs", baseRunID, len(compareRunIDs))))
	}

	// Load base run summary once (shared across all comparisons)
	fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Loading data for base run %d...", baseRunID)))
	baseSummary, err := loadRunSummaryForDiff(ctx, baseRunID, outputDir, owner, repo, hostname, verbose, artifactFilter)
	if err != nil {
		return fmt.Errorf("failed to load data for base run %d: %w", baseRunID, err)
	}

	diffs, err := runAuditDiffComparisons(runAuditDiffComparisonsParams{
		Ctx:            ctx,
		BaseRunID:      baseRunID,
		CompareRunIDs:  compareRunIDs,
		BaseSummary:    baseSummary,
		OutputDir:      outputDir,
		Owner:          owner,
		Repo:           repo,
		Hostname:       hostname,
		Verbose:        verbose,
		ArtifactFilter: artifactFilter,
	})
	if err != nil {
		return err
	}

	return runAuditDiffRender(diffs, opts, format)
}

func runAuditDiffArtifactFilter(artifactSets []string, verbose bool) ([]string, error) {
	// Validate and resolve artifact sets into a concrete filter.
	if err := ValidateArtifactSets(artifactSets); err != nil {
		return nil, err
	}
	artifactFilter := ResolveArtifactFilter(artifactSets)
	if len(artifactFilter) > 0 {
		auditDiffLog.Printf("Artifact filter active: %v", artifactFilter)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Artifact filter: downloading only "+strings.Join(artifactFilter, ", ")))
		}
	}
	return artifactFilter, nil
}

func runAuditDiffHostname(hostname string) string {
	// Auto-detect GHES host from git remote if hostname is not provided
	if hostname != "" {
		return hostname
	}
	hostname = getHostFromOriginRemote()
	if hostname != "github.com" {
		auditDiffLog.Printf("Auto-detected GHES host from git remote: %s", hostname)
	}
	return hostname
}

func runAuditDiffCheckContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
		return nil
	}
}

type runAuditDiffComparisonsParams struct {
	Ctx            context.Context
	BaseRunID      int64
	CompareRunIDs  []int64
	BaseSummary    *RunSummary
	OutputDir      string
	Owner          string
	Repo           string
	Hostname       string
	Verbose        bool
	ArtifactFilter []string
}

func runAuditDiffComparisons(p runAuditDiffComparisonsParams) ([]*AuditDiff, error) {
	diffs := make([]*AuditDiff, 0, len(p.CompareRunIDs))
	for _, compareRunID := range p.CompareRunIDs {
		if err := runAuditDiffCheckContext(p.Ctx); err != nil {
			return nil, err
		}
		fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Loading data for run %d...", compareRunID)))
		compareSummary, err := loadRunSummaryForDiff(p.Ctx, compareRunID, p.OutputDir, p.Owner, p.Repo, p.Hostname, p.Verbose, p.ArtifactFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to load data for run %d: %w", compareRunID, err)
		}
		runAuditDiffWarnFirewall(p.BaseRunID, compareRunID, p.BaseSummary, compareSummary)
		diffs = append(diffs, computeAuditDiff(p.BaseRunID, compareRunID, p.BaseSummary, compareSummary))
	}
	return diffs, nil
}

func runAuditDiffWarnFirewall(baseRunID, compareRunID int64, baseSummary, compareSummary *RunSummary) {
	fw1 := baseSummary.FirewallAnalysis
	fw2 := compareSummary.FirewallAnalysis
	if fw1 == nil && fw2 == nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No firewall data found for run pair %d→%d. Both runs may predate firewall logging.", baseRunID, compareRunID)))
		return
	}
	if fw1 == nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No firewall data found for base run %d (older run may lack firewall logs)", baseRunID)))
	}
	if fw2 == nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No firewall data found for run %d", compareRunID)))
	}
}

func runAuditDiffRender(diffs []*AuditDiff, opts AuditOptions, format string) error {
	if opts.JSONOutput || format == "json" {
		return renderAuditDiffJSON(diffs)
	}
	if format == "markdown" {
		renderAuditDiffMarkdown(diffs)
		return nil
	}
	renderAuditDiffPretty(diffs)
	return nil
}
