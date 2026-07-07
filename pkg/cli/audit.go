package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/spf13/cobra"
)

var auditLog = logger.New("cli:audit")

// AuditOptions contains shared options for audit and audit-diff execution.
type AuditOptions struct {
	Owner            string
	Repo             string
	Hostname         string
	OutputDir        string
	Verbose          bool
	Parse            bool
	JSONOutput       bool
	JobID            int64
	StepNumber       int
	Format           string
	ArtifactSets     []string
	ExperimentFilter string
	VariantFilter    string
}

var auditCommandLong = `Audit one or more workflow runs by downloading artifacts and logs, detecting errors,
analyzing MCP tool usage, and generating a concise report suitable for AI agents.

When a single run is provided, generates a detailed Markdown report for that run.
When two or more runs are provided, the first is used as the base (reference) and the
remaining runs are compared against it, producing a diff report.

Each argument accepts:
- A numeric run ID (e.g., 1234567890)
- A GitHub Actions run URL (e.g., https://github.com/owner/repo/actions/runs/1234567890)
- A GitHub Actions job URL (e.g., https://github.com/owner/repo/actions/runs/1234567890/job/9876543210)
- A GitHub Actions job URL with step (e.g., https://github.com/owner/repo/actions/runs/1234567890/job/9876543210#step:7:1)
- A GitHub workflow run URL (e.g., https://github.com/owner/repo/runs/1234567890)
- GitHub Enterprise URLs (e.g., https://github.example.com/owner/repo/actions/runs/1234567890)

When a job URL is provided (single-run mode only):
- If a step number is included (#step:7:1), extracts that specific step's output
- If no step number, finds and extracts the first failing step's output
- Saves job logs to the output directory`

var auditCommandExample = `  ` + string(constants.CLIExtensionPrefix) + ` audit 1234567890                    # Audit run with ID 1234567890
  ` + string(constants.CLIExtensionPrefix) + ` audit https://github.com/owner/repo/actions/runs/1234567890  # Audit from run URL
  ` + string(constants.CLIExtensionPrefix) + ` audit https://github.com/owner/repo/actions/runs/1234567890/job/9876543210  # Audit job and extract first failing step
  ` + string(constants.CLIExtensionPrefix) + ` audit https://github.com/owner/repo/actions/runs/1234567890/job/9876543210#step:7:1  # Extract step 7 output
  ` + string(constants.CLIExtensionPrefix) + ` audit https://github.com/owner/repo/runs/1234567890  # Audit from workflow run URL
  ` + string(constants.CLIExtensionPrefix) + ` audit https://github.example.com/owner/repo/actions/runs/1234567890  # Audit from GitHub Enterprise
  ` + string(constants.CLIExtensionPrefix) + ` audit 1234567890 -o ./audit-reports # Custom output directory
  ` + string(constants.CLIExtensionPrefix) + ` audit 1234567890 -v                 # Verbose output
  ` + string(constants.CLIExtensionPrefix) + ` audit 1234567890 --parse            # Parse agent logs and firewall logs, generating log.md and firewall.md
  ` + string(constants.CLIExtensionPrefix) + ` audit 1234567890 --repo owner/repo  # Audit run from a specific repository
  ` + string(constants.CLIExtensionPrefix) + ` audit 1234567890 1234567891         # Diff two runs (base vs comparison)
  ` + string(constants.CLIExtensionPrefix) + ` audit 1234567890 1234567891 1234567892  # Diff base against multiple runs
  ` + string(constants.CLIExtensionPrefix) + ` audit 1234567890 1234567891 --format markdown  # Markdown diff output for PR comments`

type auditCommandOptions struct {
	outputDir        string
	verbose          bool
	jsonOutput       bool
	parse            bool
	repoFlag         string
	format           string
	artifacts        []string
	stdin            bool
	experimentFilter string
	variantFilter    string
}

// NewAuditCommand creates the audit command
func NewAuditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "audit <run-id-or-url> [run-id-or-url]...",
		Short:   "Audit workflow runs and generate detailed reports",
		Long:    auditCommandLong,
		Example: auditCommandExample,
		Args:    cobra.ArbitraryArgs,
		RunE:    runAuditCommand,
	}
	registerAuditCommandFlags(cmd)
	cmd.AddCommand(NewAuditDiffSubcommand())
	return cmd
}

func registerAuditCommandFlags(cmd *cobra.Command) {
	addOutputFlag(cmd, defaultLogsOutputDir)
	addJSONFlag(cmd)
	addRepoFlag(cmd)
	cmd.Flags().Bool("parse", false, "Run JavaScript parsers on agent logs and firewall logs, writing Markdown to log.md and firewall.md")
	cmd.Flags().String("format", "pretty", "Diff output format for multi-run mode: pretty, markdown")
	cmd.Flags().StringSlice("artifacts", nil, "Artifact sets to download (default: all, because auditing requires comprehensive artifacts for analysis). Valid sets: "+strings.Join(ValidArtifactSetNames(), ", "))
	cmd.Flags().Bool("stdin", false, "Read workflow run IDs or URLs from stdin (one per line) instead of positional arguments")
	cmd.Flags().String("experiment", "", "Filter to runs that include this experiment name")
	cmd.Flags().String("variant", "", "Filter to runs with a specific variant value (requires --experiment)")
	RegisterDirFlagCompletion(cmd, "output")
}

func runAuditCommand(cmd *cobra.Command, args []string) error {
	opts, err := getAuditCommandOptions(cmd)
	if err != nil {
		return err
	}
	args, handled, err := resolveAuditCommandArgs(args, opts.stdin)
	if err != nil || handled {
		return err
	}
	if len(args) == 1 {
		return runAuditSingle(cmd.Context(), args[0], opts)
	}
	return runAuditMulti(cmd.Context(), args, opts.repoFlag, opts.outputDir, opts.verbose, opts.jsonOutput, opts.format, opts.artifacts)
}

func getAuditCommandOptions(cmd *cobra.Command) (auditCommandOptions, error) {
	opts := auditCommandOptions{}
	opts.outputDir, _ = cmd.Flags().GetString("output")
	opts.verbose, _ = cmd.Flags().GetBool("verbose")
	opts.jsonOutput, _ = cmd.Flags().GetBool("json")
	opts.parse, _ = cmd.Flags().GetBool("parse")
	opts.repoFlag, _ = cmd.Flags().GetString("repo")
	opts.format, _ = cmd.Flags().GetString("format")
	opts.artifacts, _ = cmd.Flags().GetStringSlice("artifacts")
	opts.stdin, _ = cmd.Flags().GetBool("stdin")
	opts.experimentFilter, _ = cmd.Flags().GetString("experiment")
	opts.variantFilter, _ = cmd.Flags().GetString("variant")
	if opts.variantFilter != "" && opts.experimentFilter == "" {
		return auditCommandOptions{}, errors.New(console.FormatErrorWithSuggestions(
			"--variant requires --experiment to be specified",
			[]string{"Add --experiment <name> to filter by experiment name alongside --variant"},
		))
	}
	return opts, nil
}

func resolveAuditCommandArgs(args []string, stdin bool) ([]string, bool, error) {
	if stdin {
		if len(args) > 0 {
			return nil, false, errors.New(console.FormatErrorWithSuggestions(
				"positional arguments are not allowed with --stdin",
				[]string{"Remove the run ID arguments, or omit --stdin to use positional arguments"},
			))
		}
		stdinURLs, err := readRunIDsFromStdin(os.Stdin)
		if err != nil {
			return nil, false, fmt.Errorf("failed to read run IDs from stdin: %w", err)
		}
		if len(stdinURLs) == 0 {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No run IDs or URLs provided on stdin"))
			return nil, true, nil
		}
		args = stdinURLs
	}
	if len(args) == 0 {
		return nil, false, errors.New(console.FormatErrorWithSuggestions(
			"at least one run ID or URL is required",
			[]string{
				"Provide a run ID or URL as a positional argument",
				"Use --stdin to read run IDs from stdin (one per line)",
			},
		))
	}
	return args, false, nil
}

func runAuditSingle(ctx context.Context, runIDOrURL string, opts auditCommandOptions) error {
	components, err := parser.ParseRunURLExtended(runIDOrURL)
	if err != nil {
		return err
	}
	if err := applyAuditRepoFlag(opts.repoFlag, components); err != nil {
		return err
	}
	return AuditWorkflowRun(ctx, components.Number, AuditOptions{
		Owner:            components.Owner,
		Repo:             components.Repo,
		Hostname:         components.Host,
		OutputDir:        opts.outputDir,
		Verbose:          opts.verbose,
		Parse:            opts.parse,
		JSONOutput:       opts.jsonOutput,
		JobID:            components.JobID,
		StepNumber:       components.StepNumber,
		ArtifactSets:     opts.artifacts,
		ExperimentFilter: opts.experimentFilter,
		VariantFilter:    opts.variantFilter,
	})
}

func applyAuditRepoFlag(repoFlag string, components *parser.GitHubURLComponents) error {
	if repoFlag == "" || components.Owner != "" {
		return nil
	}
	parts := strings.SplitN(repoFlag, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid repository format '%s': expected 'owner/repo'", repoFlag)
	}
	components.Owner = parts[0]
	components.Repo = parts[1]
	return nil
}

// runAuditMulti handles the multi-run diff mode for the audit command.
// The first argument is the base run; remaining arguments are comparison runs.
// Each argument may be a numeric run ID, a GitHub Actions run URL, or a job/step
// URL — job and step specificity is silently normalized to the parent run ID.
func runAuditMulti(ctx context.Context, args []string, repoFlag, outputDir string, verbose, jsonOutput bool, format string, artifacts []string) error {
	// Parse base run (job/step URLs are accepted; only the run number is used)
	baseComponents, err := parser.ParseRunURLExtended(args[0])
	if err != nil {
		return fmt.Errorf("invalid base run %q: %w", args[0], err)
	}

	// Resolve owner/repo/hostname from --repo flag or base URL
	if err := applyAuditRepoFlag(repoFlag, baseComponents); err != nil {
		return err
	}
	owner := baseComponents.Owner
	repo := baseComponents.Repo
	hostname := baseComponents.Host

	// Parse comparison run IDs (job/step URLs are accepted; only the run number is used)
	seen := make(map[int64]bool)
	compareRunIDs := make([]int64, 0, len(args)-1)
	for _, arg := range args[1:] {
		c, err := parser.ParseRunURLExtended(arg)
		if err != nil {
			return fmt.Errorf("invalid comparison run %q: %w", arg, err)
		}
		if c.Number == baseComponents.Number {
			return fmt.Errorf("comparison run ID %d is the same as the base run ID: cannot diff a run against itself", c.Number)
		}
		if seen[c.Number] {
			return fmt.Errorf("duplicate comparison run ID %d: each run ID must appear only once", c.Number)
		}
		seen[c.Number] = true
		compareRunIDs = append(compareRunIDs, c.Number)
	}

	return RunAuditDiff(ctx, baseComponents.Number, compareRunIDs, AuditOptions{
		Owner:        owner,
		Repo:         repo,
		Hostname:     hostname,
		OutputDir:    outputDir,
		Verbose:      verbose,
		JSONOutput:   jsonOutput,
		Format:       format,
		ArtifactSets: artifacts,
	})
}

// isPermissionErrorStr checks if a string contains any known permission/authentication error marker.
// This is the canonical union of all auth-error substrings used across the codebase; update here
// rather than adding new inline strings.Contains checks in callers.
//
//nolint:errstringmatch // gh auth and gh api permission failures are intentionally classified from gh CLI text here.
func isPermissionErrorStr(s string) bool {
	return strings.Contains(s, "authentication required") ||
		strings.Contains(s, "exit status 4") ||
		strings.Contains(s, "GitHub CLI authentication") ||
		strings.Contains(s, "permission") ||
		strings.Contains(s, "GH_TOKEN") ||
		strings.Contains(s, "not logged into any GitHub hosts") ||
		strings.Contains(s, "To use GitHub CLI in a GitHub Actions workflow") ||
		strings.Contains(s, "gh auth login")
}

// isPermissionError checks if an error is related to permissions/authentication.
func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	return isPermissionErrorStr(err.Error())
}

type auditRunConfig struct {
	runID            int64
	owner            string
	repo             string
	hostname         string
	outputDir        string
	verbose          bool
	parse            bool
	jsonOutput       bool
	jobID            int64
	stepNumber       int
	artifactFilter   []string
	experimentFilter string
	variantFilter    string
}

type auditAnalysisResults struct {
	metrics                 LogMetrics
	failedJobCount          int
	jobDetails              []JobInfoWithDuration
	missingTools            []MissingToolReport
	missingData             []MissingDataReport
	noops                   []NoopReport
	mcpFailures             []MCPFailureReport
	accessAnalysis          *DomainAnalysis
	firewallAnalysis        *FirewallAnalysis
	policyAnalysis          *PolicyAnalysis
	mcpToolUsage            *MCPToolUsageData
	tokenUsageSummary       *TokenUsageSummary
	redactedDomainsAnalysis *RedactedDomainsAnalysis
	rateLimitUsage          *GitHubRateLimitUsage
	artifacts               []string
	safeItemsCount          int
}

// AuditWorkflowRun audits a single workflow run and generates a report
// If jobID is provided (>0), focuses audit on that specific job
// If stepNumber is provided (>0), extracts output for that specific step
// If experimentFilter is non-empty, the run is skipped when its experiment artifact does
// not contain an assignment for that experiment name. If variantFilter is also non-empty,
// the assigned variant must equal variantFilter.
func AuditWorkflowRun(ctx context.Context, runID int64, opts AuditOptions) error {
	cfg, err := newAuditRunConfig(runID, opts)
	if err != nil {
		return err
	}
	if err := ensureAuditNotCancelled(ctx); err != nil {
		return err
	}
	announceAuditRun(cfg)
	if cfg.jobID > 0 {
		return auditJobRun(cfg.jobOptions())
	}
	if done, err := renderCachedAuditIfAvailable(ctx, cfg); done {
		return err
	}
	run, err := prepareAuditWorkflowRun(ctx, cfg)
	if err != nil {
		return err
	}
	results := collectAuditAnalysisResults(run, cfg.outputDir, cfg.verbose, artifactMatchesFilter(constants.AgentArtifactName, cfg.artifactFilter))
	run = applyAuditMetrics(run, results)
	processedRun := buildProcessedAuditRun(run, results)
	saveAuditRunSummary(cfg.outputDir, run, processedRun, results, cfg.verbose)
	if shouldSkipAuditRun(cfg.runID, cfg.outputDir, cfg.experimentFilter, cfg.variantFilter) {
		return nil
	}
	return renderAuditReport(ctx, processedRun, results.metrics, results.mcpToolUsage, cfg.auditOptions())
}

func newAuditRunConfig(runID int64, opts AuditOptions) (auditRunConfig, error) {
	if err := ValidateArtifactSets(opts.ArtifactSets); err != nil {
		return auditRunConfig{}, err
	}
	return auditRunConfig{
		runID:            runID,
		owner:            opts.Owner,
		repo:             opts.Repo,
		hostname:         resolveAuditHostname(opts.Hostname),
		outputDir:        resolveAuditOutputDir(opts.OutputDir, runID),
		verbose:          opts.Verbose,
		parse:            opts.Parse,
		jsonOutput:       opts.JSONOutput,
		jobID:            opts.JobID,
		stepNumber:       opts.StepNumber,
		artifactFilter:   ResolveArtifactFilter(opts.ArtifactSets),
		experimentFilter: opts.ExperimentFilter,
		variantFilter:    opts.VariantFilter,
	}, nil
}

func resolveAuditHostname(hostname string) string {
	if hostname == "" {
		hostname = getHostFromOriginRemote()
		if hostname != "github.com" {
			auditLog.Printf("Auto-detected GHES host from git remote: %s", hostname)
		}
	}
	return hostname
}

func resolveAuditOutputDir(outputDir string, runID int64) string {
	runOutputDir := filepath.Join(outputDir, fmt.Sprintf("run-%d", runID))
	if absDir, err := filepath.Abs(runOutputDir); err == nil {
		return absDir
	} else {
		auditLog.Printf("Failed to resolve absolute path for output directory %q: %v", runOutputDir, err)
	}
	return runOutputDir
}

func ensureAuditNotCancelled(ctx context.Context) error {
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
		return nil
	}
}

func announceAuditRun(cfg auditRunConfig) {
	auditLog.Printf("Starting audit for workflow run: runID=%d, owner=%s, repo=%s, hostname=%s, jobID=%d, stepNumber=%d", cfg.runID, cfg.owner, cfg.repo, cfg.hostname, cfg.jobID, cfg.stepNumber)
	if len(cfg.artifactFilter) > 0 {
		auditLog.Printf("Artifact filter active: %v", cfg.artifactFilter)
		if cfg.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Artifact filter: downloading only "+strings.Join(cfg.artifactFilter, ", ")))
		}
	}
	if !cfg.verbose {
		return
	}
	if cfg.jobID > 0 && cfg.stepNumber > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Auditing workflow run %d, job %d, step %d...", cfg.runID, cfg.jobID, cfg.stepNumber)))
		return
	}
	if cfg.jobID > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Auditing workflow run %d, job %d...", cfg.runID, cfg.jobID)))
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Auditing workflow run %d...", cfg.runID)))
}

func (cfg auditRunConfig) jobOptions() auditJobRunOptions {
	return auditJobRunOptions{
		runID:      cfg.runID,
		jobID:      cfg.jobID,
		stepNumber: cfg.stepNumber,
		owner:      cfg.owner,
		repo:       cfg.repo,
		hostname:   cfg.hostname,
		outputDir:  cfg.outputDir,
		verbose:    cfg.verbose,
		jsonOutput: cfg.jsonOutput,
	}
}

func (cfg auditRunConfig) auditOptions() AuditOptions {
	return AuditOptions{
		Owner:      cfg.owner,
		Repo:       cfg.repo,
		Hostname:   cfg.hostname,
		OutputDir:  cfg.outputDir,
		Verbose:    cfg.verbose,
		Parse:      cfg.parse,
		JSONOutput: cfg.jsonOutput,
	}
}

func renderCachedAuditIfAvailable(ctx context.Context, cfg auditRunConfig) (bool, error) {
	summary, ok := loadRunSummary(cfg.outputDir, cfg.verbose)
	if !ok {
		return false, nil
	}
	auditLog.Printf("Using cached run summary for run %d (processed at %s)", cfg.runID, summary.ProcessedAt.Format(time.RFC3339))
	if cfg.verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Using cached run summary for run %d (processed at %s)", cfg.runID, summary.ProcessedAt.Format(time.RFC3339))))
	}
	if shouldSkipAuditRun(cfg.runID, cfg.outputDir, cfg.experimentFilter, cfg.variantFilter) {
		return true, nil
	}
	processedRun := processedRunFromSummary(summary, cfg.outputDir)
	return true, renderAuditReport(ctx, processedRun, summary.Metrics, summary.MCPToolUsage, cfg.auditOptions())
}

func processedRunFromSummary(summary *RunSummary, runOutputDir string) ProcessedRun {
	processedRun := ProcessedRun{
		Run:                     summary.Run,
		AwContext:               summary.AwContext,
		TaskDomain:              summary.TaskDomain,
		BehaviorFingerprint:     summary.BehaviorFingerprint,
		AgenticAssessments:      summary.AgenticAssessments,
		AccessAnalysis:          summary.AccessAnalysis,
		FirewallAnalysis:        summary.FirewallAnalysis,
		PolicyAnalysis:          summary.PolicyAnalysis,
		RedactedDomainsAnalysis: summary.RedactedDomainsAnalysis,
		MissingTools:            summary.MissingTools,
		MissingData:             summary.MissingData,
		Noops:                   summary.Noops,
		MCPFailures:             summary.MCPFailures,
		TokenUsage:              summary.TokenUsage,
		GitHubRateLimitUsage:    summary.GitHubRateLimitUsage,
		JobDetails:              summary.JobDetails,
	}
	// Run.Turns may be zero on cached-summary paths where the RunSummary was
	// serialised before the run completed. Metrics.Turns is populated from log
	// parsing and is authoritative; backfill here so that audit comparison deltas
	// are computed from an accurate value.
	if processedRun.Run.Turns == 0 && summary.Metrics.Turns > 0 {
		processedRun.Run.Turns = summary.Metrics.Turns
	}
	processedRun.Run.LogsPath = runOutputDir
	return processedRun
}

func shouldSkipAuditRun(runID int64, runOutputDir, experimentFilter, variantFilter string) bool {
	if experimentFilter == "" {
		return false
	}
	expData := extractExperimentData(runOutputDir)
	if experimentMatchesFilter(expData, experimentFilter, variantFilter) {
		return false
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(formatExperimentSkipMessage(runID, experimentFilter, variantFilter)))
	return true
}

func prepareAuditWorkflowRun(ctx context.Context, cfg auditRunConfig) (WorkflowRun, error) {
	run, hasLocalCache, useLocalCache, err := fetchAuditRunWithCache(ctx, cfg)
	if err != nil {
		return WorkflowRun{}, err
	}
	if !useLocalCache {
		useLocalCache, err = downloadAuditArtifactsIfNeeded(ctx, cfg, run, hasLocalCache)
		if err != nil {
			return WorkflowRun{}, err
		}
	}
	return prepareRunForAnalysis(run, cfg, useLocalCache), nil
}

func fetchAuditRunWithCache(ctx context.Context, cfg auditRunConfig) (WorkflowRun, bool, bool, error) {
	hasLocalCache := fileutil.DirExists(cfg.outputDir) && !fileutil.IsDirEmpty(cfg.outputDir)
	run, err := fetchWorkflowRunMetadata(ctx, cfg.runID, cfg.owner, cfg.repo, cfg.hostname, cfg.verbose)
	if err == nil {
		return run, hasLocalCache, false, nil
	}
	if !isPermissionError(err) {
		return WorkflowRun{}, false, false, err
	}
	if !hasLocalCache {
		return WorkflowRun{}, false, false, cacheRecoveryError(
			"GitHub API access denied and no local cache found.", cfg.runID, cfg.outputDir, err,
		)
	}
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage("GitHub API access denied, but found locally cached artifacts. Processing cached data..."))
	return run, hasLocalCache, true, nil
}

func downloadAuditArtifactsIfNeeded(ctx context.Context, cfg auditRunConfig, run WorkflowRun, hasLocalCache bool) (bool, error) {
	if cfg.verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Run: %s (Status: %s, Conclusion: %s)", run.WorkflowName, run.Status, run.Conclusion)))
	}
	auditLog.Printf("Downloading artifacts for run %d", cfg.runID)
	err := downloadRunArtifacts(ctx, cfg.runID, cfg.outputDir, cfg.verbose, cfg.owner, cfg.repo, cfg.hostname, cfg.artifactFilter)
	if err == nil || errors.Is(err, ErrNoArtifacts) {
		if errors.Is(err, ErrNoArtifacts) {
			auditLog.Printf("No artifacts found for run %d", cfg.runID)
			if cfg.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No artifacts attached to this run. Proceeding with metadata-only audit."))
			}
		}
		return false, nil
	}
	if isPermissionError(err) && hasLocalCache {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Artifact download failed due to permissions, but found locally cached artifacts. Processing cached data..."))
		return true, nil
	}
	if isPermissionError(err) {
		return false, cacheRecoveryError("failed to download artifacts due to permissions and no local cache found.", cfg.runID, cfg.outputDir, err)
	}
	return false, fmt.Errorf("failed to download artifacts: %w", err)
}

func cacheRecoveryError(message string, runID int64, runOutputDir string, err error) error {
	return fmt.Errorf(message+"\n\n"+
		"To download artifacts, use the GitHub MCP server:\n\n"+
		"1. Use the github-mcp-server tool 'download_workflow_run_artifacts' with:\n"+
		"   - run_id: %d\n"+
		"   - output_directory: %s\n\n"+
		"2. After downloading, run this audit command again to analyze the cached artifacts.\n\n"+
		"Original error: %v", runID, runOutputDir, err)
}

func prepareRunForAnalysis(run WorkflowRun, cfg auditRunConfig, useLocalCache bool) WorkflowRun {
	if useLocalCache && run.DatabaseID == 0 {
		run = WorkflowRun{
			DatabaseID:   cfg.runID,
			WorkflowName: fmt.Sprintf("Workflow Run %d", cfg.runID),
			Status:       "unknown",
			LogsPath:     cfg.outputDir,
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Using locally cached artifacts without metadata. Some report details may be unavailable."))
	}
	run.LogsPath = cfg.outputDir
	if !run.StartedAt.IsZero() && !run.UpdatedAt.IsZero() {
		run.Duration = run.UpdatedAt.Sub(run.StartedAt)
	}
	return run
}

func collectAuditAnalysisResults(run WorkflowRun, runOutputDir string, verbose bool, hasFirewallArtifact bool) auditAnalysisResults {
	results := auditAnalysisResults{}
	var wg sync.WaitGroup
	launchCoreAuditAnalyses(&wg, &results, run, runOutputDir, verbose)
	if hasFirewallArtifact {
		launchFirewallAuditAnalyses(&wg, &results, runOutputDir, verbose)
	}
	launchSupplementalAuditAnalyses(&wg, &results, runOutputDir, verbose)
	wg.Wait()
	return results
}

func launchCoreAuditAnalyses(wg *sync.WaitGroup, results *auditAnalysisResults, run WorkflowRun, runOutputDir string, verbose bool) {
	// Resolve experiment assignment once so all goroutines reuse the same values
	// rather than each reading state.json independently.
	expName, expVariant, _ := firstExperimentAssignment(extractExperimentData(runOutputDir))

	launchMetricsAnalysis(wg, results, runOutputDir, verbose, run.WorkflowPath)
	launchJobDetailsAnalysis(wg, results, run.DatabaseID, verbose)
	runAuditAnalysis(wg, verbose, "extractMissingToolsFromRun", "Failed to extract missing tools", func(v []MissingToolReport) {
		results.missingTools = v
	}, func() ([]MissingToolReport, error) {
		return extractMissingToolsFromRun(runOutputDir, run, verbose, expName, expVariant)
	})
	runAuditAnalysis(wg, verbose, "extractMissingDataFromRun", "Failed to extract missing data", func(v []MissingDataReport) {
		results.missingData = v
	}, func() ([]MissingDataReport, error) {
		return extractMissingDataFromRun(runOutputDir, run, verbose, expName, expVariant)
	})
	runAuditAnalysis(wg, verbose, "extractNoopsFromRun", "Failed to extract noops", func(v []NoopReport) {
		results.noops = v
	}, func() ([]NoopReport, error) {
		return extractNoopsFromRun(runOutputDir, run, verbose, expName, expVariant)
	})
	runAuditAnalysis(wg, verbose, "extractMCPFailuresFromRun", "Failed to extract MCP failures", func(v []MCPFailureReport) {
		results.mcpFailures = v
	}, func() ([]MCPFailureReport, error) {
		return extractMCPFailuresFromRun(runOutputDir, run, verbose, expName, expVariant)
	})
	runAuditAnalysis(wg, verbose, "analyzeAccessLogs", "Failed to analyze access logs", func(v *DomainAnalysis) {
		results.accessAnalysis = v
	}, func() (*DomainAnalysis, error) {
		return analyzeAccessLogs(runOutputDir, verbose)
	})
}

func launchMetricsAnalysis(wg *sync.WaitGroup, results *auditAnalysisResults, runOutputDir string, verbose bool, workflowPath string) {
	wg.Go(func() {
		metrics, err := extractLogMetrics(runOutputDir, verbose, workflowPath)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract metrics: %v", err)))
			}
			results.metrics = LogMetrics{}
			return
		}
		results.metrics = metrics
	})
}

func launchJobDetailsAnalysis(wg *sync.WaitGroup, results *auditAnalysisResults, runID int64, verbose bool) {
	wg.Go(func() {
		jobDetails, failedJobCount, err := fetchJobDetailsWithCounts(runID, verbose)
		if err != nil {
			auditLog.Printf("fetchJobDetailsWithCounts failed: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch job details: %v", err)))
			}
			return
		}
		results.jobDetails = jobDetails
		results.failedJobCount = failedJobCount
	})
}

func launchFirewallAuditAnalyses(wg *sync.WaitGroup, results *auditAnalysisResults, runOutputDir string, verbose bool) {
	launchFirewallAnalysis(wg, results, runOutputDir, verbose)
	runAuditAnalysis(wg, verbose, "analyzeFirewallPolicy", "Failed to analyze firewall policy", func(v *PolicyAnalysis) {
		results.policyAnalysis = v
	}, func() (*PolicyAnalysis, error) {
		return analyzeFirewallPolicy(runOutputDir, verbose)
	})
	runAuditAnalysis(wg, verbose, "extractMCPToolUsageData", "Failed to extract MCP tool usage", func(v *MCPToolUsageData) {
		results.mcpToolUsage = v
	}, func() (*MCPToolUsageData, error) {
		return extractMCPToolUsageData(runOutputDir, verbose)
	})
	runAuditAnalysis(wg, verbose, "analyzeTokenUsage", "Failed to analyze token usage", func(v *TokenUsageSummary) {
		results.tokenUsageSummary = v
	}, func() (*TokenUsageSummary, error) {
		return analyzeTokenUsage(runOutputDir, verbose)
	})
}

func launchFirewallAnalysis(wg *sync.WaitGroup, results *auditAnalysisResults, runOutputDir string, verbose bool) {
	wg.Go(func() {
		firewallAnalysis, err := analyzeFirewallLogs(runOutputDir, verbose)
		if err != nil {
			auditLog.Printf("analyzeFirewallLogs failed: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze firewall logs: %v", err)))
			}
		}
		if agentLogFirewall := extractFirewallFromAgentLog(runOutputDir, verbose); agentLogFirewall != nil {
			if firewallAnalysis == nil {
				firewallAnalysis = agentLogFirewall
			} else {
				firewallAnalysis.AddMetrics(agentLogFirewall)
			}
		}
		results.firewallAnalysis = firewallAnalysis
	})
}

func launchSupplementalAuditAnalyses(wg *sync.WaitGroup, results *auditAnalysisResults, runOutputDir string, verbose bool) {
	runAuditAnalysis(wg, verbose, "analyzeRedactedDomains", "Failed to analyze redacted domains", func(v *RedactedDomainsAnalysis) {
		results.redactedDomainsAnalysis = v
	}, func() (*RedactedDomainsAnalysis, error) {
		return analyzeRedactedDomains(runOutputDir, verbose)
	})
	runAuditAnalysis(wg, verbose, "analyzeGitHubRateLimits", "Failed to analyze GitHub rate limit usage", func(v *GitHubRateLimitUsage) {
		results.rateLimitUsage = v
	}, func() (*GitHubRateLimitUsage, error) {
		return analyzeGitHubRateLimits(runOutputDir, verbose)
	})
	runAuditAnalysis(wg, verbose, "listArtifacts", "Failed to list artifacts", func(v []string) {
		results.artifacts = v
	}, func() ([]string, error) {
		return listArtifacts(runOutputDir)
	})
	wg.Go(func() {
		results.safeItemsCount = len(extractCreatedItemsFromManifest(runOutputDir))
	})
}

func runAuditAnalysis[T any](wg *sync.WaitGroup, verbose bool, name, warning string, setter func(T), fn func() (T, error)) {
	wg.Go(func() {
		value, err := fn()
		if err != nil {
			auditLog.Printf("%s failed: %v", name, err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("%s: %v", warning, err)))
			}
			return
		}
		setter(value)
	})
}

func applyAuditMetrics(run WorkflowRun, results auditAnalysisResults) WorkflowRun {
	run.TokenUsage = results.metrics.TokenUsage
	run.Turns = results.metrics.Turns
	run.ErrorCount = results.failedJobCount
	if run.Conclusion == "failure" && run.ErrorCount == 0 {
		run.ErrorCount = 1
	}
	run.WarningCount = 0
	run.SafeItemsCount = results.safeItemsCount
	return run
}

func buildProcessedAuditRun(run WorkflowRun, results auditAnalysisResults) ProcessedRun {
	processedRun := ProcessedRun{
		Run:                     run,
		FirewallAnalysis:        results.firewallAnalysis,
		PolicyAnalysis:          results.policyAnalysis,
		RedactedDomainsAnalysis: results.redactedDomainsAnalysis,
		MissingTools:            results.missingTools,
		MissingData:             results.missingData,
		Noops:                   results.noops,
		MCPFailures:             results.mcpFailures,
		TokenUsage:              results.tokenUsageSummary,
		GitHubRateLimitUsage:    results.rateLimitUsage,
		JobDetails:              results.jobDetails,
	}
	awContext, _, _, taskDomain, behaviorFingerprint, agenticAssessments := deriveRunAgenticAnalysis(processedRun, results.metrics)
	processedRun.AwContext = awContext
	processedRun.TaskDomain = taskDomain
	processedRun.BehaviorFingerprint = behaviorFingerprint
	processedRun.AgenticAssessments = agenticAssessments
	return processedRun
}

func saveAuditRunSummary(runOutputDir string, run WorkflowRun, processedRun ProcessedRun, results auditAnalysisResults, verbose bool) {
	summary := buildAuditRunSummary(run, processedRun, results)
	if err := saveRunSummary(runOutputDir, summary, verbose); err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to save run summary: %v", err)))
	}
}

func buildAuditRunSummary(run WorkflowRun, processedRun ProcessedRun, results auditAnalysisResults) *RunSummary {
	return &RunSummary{
		CLIVersion:              GetVersion(),
		RunID:                   run.DatabaseID,
		ProcessedAt:             time.Now(),
		Run:                     run,
		Metrics:                 results.metrics,
		AwContext:               processedRun.AwContext,
		TaskDomain:              processedRun.TaskDomain,
		BehaviorFingerprint:     processedRun.BehaviorFingerprint,
		AgenticAssessments:      processedRun.AgenticAssessments,
		AccessAnalysis:          results.accessAnalysis,
		FirewallAnalysis:        results.firewallAnalysis,
		PolicyAnalysis:          results.policyAnalysis,
		RedactedDomainsAnalysis: results.redactedDomainsAnalysis,
		MissingTools:            results.missingTools,
		MissingData:             results.missingData,
		Noops:                   results.noops,
		MCPFailures:             results.mcpFailures,
		MCPToolUsage:            results.mcpToolUsage,
		TokenUsage:              results.tokenUsageSummary,
		GitHubRateLimitUsage:    results.rateLimitUsage,
		ArtifactsList:           results.artifacts,
		JobDetails:              results.jobDetails,
	}
}

// renderAuditReport builds and renders the audit report from a fully-populated processedRun.
// It is called both when serving from a cached run summary and after a fresh processing pass,
// ensuring that the two paths produce identical output.
func renderAuditReport(ctx context.Context, processedRun ProcessedRun, metrics LogMetrics, mcpToolUsage *MCPToolUsageData, opts AuditOptions) error {
	runID := processedRun.Run.DatabaseID
	runOutputDir := opts.OutputDir
	processedRun.Run.SafeItemsCount = len(extractCreatedItemsFromManifest(runOutputDir))
	auditData := buildRenderedAuditData(ctx, processedRun, metrics, mcpToolUsage, runOutputDir, opts)
	if err := renderAuditOutput(auditData, runOutputDir, opts.JSONOutput, opts.Verbose); err != nil {
		return err
	}
	renderAuditGatewayMetrics(runOutputDir, opts.Verbose)
	renderAuditUnifiedTimeline(runOutputDir, opts.Verbose)
	parseAuditLogsIfRequested(runID, runOutputDir, opts)
	renderAuditCompletion(runOutputDir, opts.JSONOutput)
	return nil
}

func buildRenderedAuditData(ctx context.Context, processedRun ProcessedRun, metrics LogMetrics, mcpToolUsage *MCPToolUsageData, runOutputDir string, opts AuditOptions) AuditData {
	currentCreatedItems := extractCreatedItemsFromManifest(runOutputDir)
	currentSnapshot := buildAuditComparisonSnapshot(processedRun, currentCreatedItems)
	comparison := buildAuditComparisonForRun(ctx, processedRun, currentSnapshot, runOutputDir, opts.Owner, opts.Repo, opts.Hostname, opts.Verbose)
	auditData := buildAuditData(processedRun, metrics, mcpToolUsage)
	auditData.Comparison = comparison
	return auditData
}

func renderAuditOutput(auditData AuditData, runOutputDir string, jsonOutput, verbose bool) error {
	if jsonOutput {
		if err := renderJSON(auditData); err != nil {
			return fmt.Errorf("failed to render JSON output: %w", err)
		}
		return nil
	}
	renderConsole(auditData, runOutputDir)
	if verbose {
		auditLog.Printf("Rendered console audit report for %s", runOutputDir)
	}
	return nil
}

func renderAuditGatewayMetrics(runOutputDir string, verbose bool) {
	gatewayMetrics, err := parseGatewayLogs(runOutputDir, verbose)
	if err != nil {
		return
	}
	if metricsOutput := renderGatewayMetricsTable(gatewayMetrics, verbose); metricsOutput != "" {
		fmt.Fprint(os.Stderr, metricsOutput)
	}
}

// renderAuditUnifiedTimeline builds the unified event timeline from the run output
// directory (combining MCP Gateway, AWF firewall, and agent events) and writes the
// rendered table to stderr.  It is a no-op when no events can be collected.
func renderAuditUnifiedTimeline(runOutputDir string, verbose bool) {
	events, err := BuildUnifiedTimeline(runOutputDir, verbose)
	if err != nil {
		auditLog.Printf("BuildUnifiedTimeline error for %s: %v", runOutputDir, err)
		return
	}
	if output := renderUnifiedTimeline(events); output != "" {
		fmt.Fprint(os.Stderr, output)
	}
}

func parseAuditLogsIfRequested(runID int64, runOutputDir string, opts AuditOptions) {
	if !opts.Parse {
		return
	}
	parseAgentLogIfRequested(runID, runOutputDir, opts.Verbose)
	parseFirewallLogsIfRequested(runID, runOutputDir, opts.Verbose)
}

func parseAgentLogIfRequested(runID int64, runOutputDir string, verbose bool) {
	awInfoPath := filepath.Join(runOutputDir, "aw_info.json")
	engine := extractEngineFromAwInfo(awInfoPath, verbose)
	if engine == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No engine detected (aw_info.json missing or invalid); skipping agent log rendering"))
		}
		return
	}
	if err := parseAgentLog(runOutputDir, engine, verbose); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse agent log for run %d: %v", runID, err)))
		}
		return
	}
	logMdPath := filepath.Join(runOutputDir, "log.md")
	if fileutil.FileExists(logMdPath) {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Parsed log for run %d → %s", runID, logMdPath)))
	}
}

func parseFirewallLogsIfRequested(runID int64, runOutputDir string, verbose bool) {
	if err := parseFirewallLogs(runOutputDir, verbose); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse firewall logs for run %d: %v", runID, err)))
		}
		return
	}
	firewallMdPath := filepath.Join(runOutputDir, "firewall.md")
	if fileutil.FileExists(firewallMdPath) {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Parsed firewall logs for run %d → %s", runID, firewallMdPath)))
	}
}

func renderAuditCompletion(runOutputDir string, jsonOutput bool) {
	if jsonOutput {
		return
	}
	absOutputDir, _ := filepath.Abs(runOutputDir)
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Audit complete. Logs saved to "+absOutputDir))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Tip: use --artifacts to select specific artifact sets (agent, firewall, mcp, activation, detection, etc.)"))
}
