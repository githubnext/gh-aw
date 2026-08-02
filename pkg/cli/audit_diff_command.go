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

// NewAuditDiffSubcommand creates the audit diff subcommand.
// Deprecated: pass multiple run IDs directly to `audit` instead (e.g. `gh aw audit <base> <compare...>`).
// This subcommand is hidden and kept for backward compatibility only.
func NewAuditDiffSubcommand() *cobra.Command {
	cmd := newAuditDiffSubcommandDefinition()
	addOutputFlag(cmd, defaultLogsOutputDir)
	addJSONFlag(cmd)
	addRepoFlag(cmd)
	cmd.Flags().String("format", "pretty", "Output format: pretty, markdown")
	cmd.Flags().StringSlice("artifacts", nil, "Artifact sets to download (default: all, because auditing requires comprehensive artifacts for analysis). Valid sets: "+strings.Join(ValidArtifactSetNames(), ", "))
	return cmd
}

func newAuditDiffSubcommandDefinition() *cobra.Command {
	return &cobra.Command{
		Use:    "diff <base-run-id> <compare-run-id>...",
		Short:  "[Deprecated] Compare workflow runs (use: gh aw audit <base> <compare...>)",
		Hidden: true,
		Long: `Deprecated: pass multiple run IDs directly to the audit command instead.

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
- Detailed token usage breakdown (input/output/cache + AI Credits) from firewall proxy`,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346                               # Compare two runs
  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346 12347 12348                   # Compare base against 3 runs
  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346 --format markdown             # Markdown output for PR comments
  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346 --json                        # JSON for CI integration
  ` + string(constants.CLIExtensionPrefix) + ` audit diff 12345 12346 --repo owner/repo             # Specify repository`,
		Args: cobra.MinimumNArgs(2),
		RunE: runAuditDiffSubcommand,
	}
}

func runAuditDiffSubcommand(cmd *cobra.Command, args []string) error {
	baseRunID, compareRunIDs, err := parseAuditDiffRunIDs(args)
	if err != nil {
		return err
	}
	opts, err := auditDiffOptionsFromFlags(cmd)
	if err != nil {
		return err
	}
	return RunAuditDiff(cmd.Context(), baseRunID, compareRunIDs, opts)
}

func parseAuditDiffRunIDs(args []string) (int64, []int64, error) {
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
		if err := validateAuditDiffCompareRunID(baseRunID, id, seen); err != nil {
			return 0, nil, err
		}
		seen[id] = true
		compareRunIDs = append(compareRunIDs, id)
	}
	return baseRunID, compareRunIDs, nil
}

func validateAuditDiffCompareRunID(baseRunID, compareRunID int64, seen map[int64]bool) error {
	if compareRunID == baseRunID {
		return fmt.Errorf("comparison run ID %d is the same as the base run ID: cannot diff a run against itself", compareRunID)
	}
	if seen[compareRunID] {
		return fmt.Errorf("duplicate comparison run ID %d: each run ID must appear only once", compareRunID)
	}
	return nil
}

func auditDiffOptionsFromFlags(cmd *cobra.Command) (AuditOptions, error) {
	outputDir, _ := cmd.Flags().GetString("output")
	verbose, _ := cmd.Flags().GetBool("verbose")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	format, _ := cmd.Flags().GetString("format")
	artifacts, _ := cmd.Flags().GetStringSlice("artifacts")
	owner, repo, err := parseAuditDiffRepoFlag(cmd)
	if err != nil {
		return AuditOptions{}, err
	}
	return AuditOptions{
		Owner:        owner,
		Repo:         repo,
		OutputDir:    outputDir,
		Verbose:      verbose,
		JSONOutput:   jsonOutput,
		Format:       format,
		ArtifactSets: artifacts,
	}, nil
}

func parseAuditDiffRepoFlag(cmd *cobra.Command) (string, string, error) {
	repoFlag, _ := cmd.Flags().GetString("repo")
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
	runtime, err := newAuditDiffRuntime(ctx, opts)
	if err != nil {
		return err
	}
	printAuditDiffStart(baseRunID, compareRunIDs)
	baseSummary, err := loadAuditDiffBaseSummary(ctx, baseRunID, runtime)
	if err != nil {
		return err
	}
	diffs, err := computeAuditDiffs(ctx, baseRunID, compareRunIDs, baseSummary, runtime)
	if err != nil {
		return err
	}
	return renderAuditDiffOutput(diffs, runtime.opts)
}

type auditDiffRuntime struct {
	opts           AuditOptions
	hostname       string
	artifactFilter []string
}

func newAuditDiffRuntime(ctx context.Context, opts AuditOptions) (*auditDiffRuntime, error) {
	auditDiffLog.Printf("Starting audit diff: base options=%+v", opts)
	if err := ValidateArtifactSets(opts.ArtifactSets); err != nil {
		return nil, err
	}
	runtime := &auditDiffRuntime{opts: opts, artifactFilter: ResolveArtifactFilter(opts.ArtifactSets)}
	printAuditDiffArtifactFilter(runtime.artifactFilter, opts.Verbose)
	runtime.hostname = resolveAuditDiffHostname(opts.Hostname)
	if err := ensureAuditDiffContext(ctx); err != nil {
		return nil, err
	}
	return runtime, nil
}

func printAuditDiffArtifactFilter(artifactFilter []string, verbose bool) {
	if len(artifactFilter) == 0 {
		return
	}
	auditDiffLog.Printf("Artifact filter active: %v", artifactFilter)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Artifact filter: downloading only "+strings.Join(artifactFilter, ", ")))
	}
}

func resolveAuditDiffHostname(hostname string) string {
	if hostname != "" {
		return hostname
	}
	hostname = getHostFromOriginRemote()
	if hostname != "github.com" {
		auditDiffLog.Printf("Auto-detected GHES host from git remote: %s", hostname)
	}
	return hostname
}

func ensureAuditDiffContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
		return nil
	}
}

func printAuditDiffStart(baseRunID int64, compareRunIDs []int64) {
	if len(compareRunIDs) == 1 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Comparing workflow runs: Run #%d → Run #%d", baseRunID, compareRunIDs[0])))
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Comparing workflow runs: Run #%d (base) vs %d comparison runs", baseRunID, len(compareRunIDs))))
}

func loadAuditDiffBaseSummary(ctx context.Context, baseRunID int64, runtime *auditDiffRuntime) (*RunSummary, error) {
	fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Loading data for base run %d...", baseRunID)))
	summary, err := loadRunSummaryForDiff(ctx, baseRunID, runtime.opts.OutputDir, runtime.opts.Owner, runtime.opts.Repo, runtime.hostname, runtime.opts.Verbose, runtime.artifactFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to load data for base run %d: %w", baseRunID, err)
	}
	return summary, nil
}

func computeAuditDiffs(ctx context.Context, baseRunID int64, compareRunIDs []int64, baseSummary *RunSummary, runtime *auditDiffRuntime) ([]*AuditDiff, error) {
	diffs := make([]*AuditDiff, 0, len(compareRunIDs))
	for _, compareRunID := range compareRunIDs {
		if err := ensureAuditDiffContext(ctx); err != nil {
			return nil, err
		}
		compareSummary, err := loadAuditDiffCompareSummary(ctx, compareRunID, runtime)
		if err != nil {
			return nil, fmt.Errorf("failed to load data for run %d: %w", compareRunID, err)
		}
		warnAboutAuditDiffFirewallData(baseRunID, compareRunID, baseSummary, compareSummary)
		diffs = append(diffs, computeAuditDiff(baseRunID, compareRunID, baseSummary, compareSummary))
	}
	return diffs, nil
}

func loadAuditDiffCompareSummary(ctx context.Context, compareRunID int64, runtime *auditDiffRuntime) (*RunSummary, error) {
	fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Loading data for run %d...", compareRunID)))
	return loadRunSummaryForDiff(ctx, compareRunID, runtime.opts.OutputDir, runtime.opts.Owner, runtime.opts.Repo, runtime.hostname, runtime.opts.Verbose, runtime.artifactFilter)
}

func warnAboutAuditDiffFirewallData(baseRunID, compareRunID int64, baseSummary, compareSummary *RunSummary) {
	fw1 := baseSummary.FirewallAnalysis
	fw2 := compareSummary.FirewallAnalysis
	switch {
	case fw1 == nil && fw2 == nil:
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No firewall data found for run pair %d→%d. Both runs may predate firewall logging.", baseRunID, compareRunID)))
	case fw1 == nil:
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No firewall data found for base run %d (older run may lack firewall logs)", baseRunID)))
	case fw2 == nil:
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No firewall data found for run %d", compareRunID)))
	}
}

func renderAuditDiffOutput(diffs []*AuditDiff, opts AuditOptions) error {
	switch {
	case opts.JSONOutput || opts.Format == "json":
		return renderAuditDiffJSON(diffs)
	case opts.Format == "markdown":
		renderAuditDiffMarkdown(diffs)
	default:
		renderAuditDiffPretty(diffs)
	}
	return nil
}
