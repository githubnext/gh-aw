package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/errorutil"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/goccy/go-yaml"
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

// NewAuditCommand creates the audit command
func NewAuditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit <run-id-or-url> [run-id-or-url]...",
		Short: "Audit workflow runs and generate detailed reports",
		Long: `Audit one or more workflow runs by downloading artifacts and logs, detecting errors,
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
- Saves job logs to the output directory`,
		Example: `  ` + string(constants.CLIExtensionPrefix) + ` audit 1234567890                    # Audit run with ID 1234567890
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
  ` + string(constants.CLIExtensionPrefix) + ` audit 1234567890 1234567891 --format markdown  # Markdown diff output for PR comments`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			outputDir, _ := cmd.Flags().GetString("output")
			verbose, _ := cmd.Flags().GetBool("verbose")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			parse, _ := cmd.Flags().GetBool("parse")
			repoFlag, _ := cmd.Flags().GetString("repo")
			artifacts, _ := cmd.Flags().GetStringSlice("artifacts")
			stdin, _ := cmd.Flags().GetBool("stdin")
			experimentFilter, _ := cmd.Flags().GetString("experiment")
			variantFilter, _ := cmd.Flags().GetString("variant")

			// --variant requires --experiment to be meaningful.
			if variantFilter != "" && experimentFilter == "" {
				return errors.New(console.FormatErrorWithSuggestions(
					"--variant requires --experiment to be specified",
					[]string{"Add --experiment <name> to filter by experiment name alongside --variant"},
				))
			}

			// When --stdin is provided, read run IDs/URLs from stdin instead of positional args.
			if stdin {
				if len(args) > 0 {
					return errors.New(console.FormatErrorWithSuggestions(
						"positional arguments are not allowed with --stdin",
						[]string{"Remove the run ID arguments, or omit --stdin to use positional arguments"},
					))
				}
				stdinURLs, err := readRunIDsFromStdin(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read run IDs from stdin: %w", err)
				}
				if len(stdinURLs) == 0 {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No run IDs or URLs provided on stdin"))
					return nil
				}
				args = stdinURLs
			}

			if len(args) == 0 {
				return errors.New(console.FormatErrorWithSuggestions(
					"at least one run ID or URL is required",
					[]string{
						"Provide a run ID or URL as a positional argument",
						"Use --stdin to read run IDs from stdin (one per line)",
					},
				))
			}

			if len(args) == 1 {
				// Single run: existing audit behavior
				runIDOrURL := args[0]

				// Parse run information from input (either numeric ID or URL)
				// Use extended parsing to capture job ID and step information
				components, err := parser.ParseRunURLExtended(runIDOrURL)
				if err != nil {
					return err
				}

				// If --repo is provided and owner/repo were not parsed from a URL, apply them
				if repoFlag != "" && components.Owner == "" {
					parts := strings.SplitN(repoFlag, "/", 2)
					if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
						return fmt.Errorf("invalid repository format '%s': expected 'owner/repo'", repoFlag)
					}
					components.Owner = parts[0]
					components.Repo = parts[1]
				}

				return AuditWorkflowRun(cmd.Context(), components.Number, AuditOptions{
					Owner:            components.Owner,
					Repo:             components.Repo,
					Hostname:         components.Host,
					OutputDir:        outputDir,
					Verbose:          verbose,
					Parse:            parse,
					JSONOutput:       jsonOutput,
					JobID:            components.JobID,
					StepNumber:       components.StepNumber,
					ArtifactSets:     artifacts,
					ExperimentFilter: experimentFilter,
					VariantFilter:    variantFilter,
				})
			}

			// Multiple runs: diff mode (first is base, rest are comparisons)
			format, _ := cmd.Flags().GetString("format")
			return runAuditMulti(cmd.Context(), args, repoFlag, outputDir, verbose, jsonOutput, format, artifacts)
		},
	}

	// Add flags to audit command
	addOutputFlag(cmd, defaultLogsOutputDir)
	addJSONFlag(cmd)
	addRepoFlag(cmd)
	cmd.Flags().Bool("parse", false, "Run JavaScript parsers on agent logs and firewall logs, writing Markdown to log.md and firewall.md")
	cmd.Flags().String("format", "pretty", "Diff output format for multi-run mode: pretty, markdown")
	cmd.Flags().StringSlice("artifacts", nil, "Artifact sets to download (default: all). Valid sets: "+strings.Join(ValidArtifactSetNames(), ", "))
	cmd.Flags().Bool("stdin", false, "Read workflow run IDs or URLs from stdin (one per line) instead of positional arguments")
	cmd.Flags().String("experiment", "", "Filter to runs that include this experiment name")
	cmd.Flags().String("variant", "", "Filter to runs with a specific variant value (requires --experiment)")

	// Register completions for audit command
	RegisterDirFlagCompletion(cmd, "output")

	// Add subcommands
	cmd.AddCommand(NewAuditDiffSubcommand())

	return cmd
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
	owner := baseComponents.Owner
	repo := baseComponents.Repo
	hostname := baseComponents.Host
	if repoFlag != "" && owner == "" {
		parts := strings.SplitN(repoFlag, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid repository format '%s': expected 'owner/repo'", repoFlag)
		}
		owner = parts[0]
		repo = parts[1]
	}

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

// AuditWorkflowRun audits a single workflow run and generates a report
// If jobID is provided (>0), focuses audit on that specific job
// If stepNumber is provided (>0), extracts output for that specific step
// If experimentFilter is non-empty, the run is skipped when its experiment artifact does
// not contain an assignment for that experiment name. If variantFilter is also non-empty,
// the assigned variant must equal variantFilter.
func AuditWorkflowRun(ctx context.Context, runID int64, opts AuditOptions) error {
	owner := opts.Owner
	repo := opts.Repo
	hostname := opts.Hostname
	outputDir := opts.OutputDir
	verbose := opts.Verbose
	parse := opts.Parse
	jsonOutput := opts.JSONOutput
	jobID := opts.JobID
	stepNumber := opts.StepNumber
	artifactSets := opts.ArtifactSets
	experimentFilter := opts.ExperimentFilter
	variantFilter := opts.VariantFilter

	// Auto-detect GHES host from git remote if hostname is not provided
	if hostname == "" {
		hostname = getHostFromOriginRemote()
		if hostname != "github.com" {
			auditLog.Printf("Auto-detected GHES host from git remote: %s", hostname)
		}
	}

	auditLog.Printf("Starting audit for workflow run: runID=%d, owner=%s, repo=%s, hostname=%s, jobID=%d, stepNumber=%d", runID, owner, repo, hostname, jobID, stepNumber)

	// Validate and resolve artifact sets into a concrete filter.
	if err := ValidateArtifactSets(artifactSets); err != nil {
		return err
	}
	artifactFilter := ResolveArtifactFilter(artifactSets)
	if len(artifactFilter) > 0 {
		auditLog.Printf("Artifact filter active: %v", artifactFilter)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Artifact filter: downloading only "+strings.Join(artifactFilter, ", ")))
		}
	}

	// Check context cancellation at the start
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
	}

	if verbose {
		if jobID > 0 {
			if stepNumber > 0 {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Auditing workflow run %d, job %d, step %d...", runID, jobID, stepNumber)))
			} else {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Auditing workflow run %d, job %d...", runID, jobID)))
			}
		} else {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Auditing workflow run %d...", runID)))
		}
	}

	runOutputDir := filepath.Join(outputDir, fmt.Sprintf("run-%d", runID))
	if absDir, err := filepath.Abs(runOutputDir); err == nil {
		runOutputDir = absDir
	} else {
		auditLog.Printf("Failed to resolve absolute path for output directory %q: %v", runOutputDir, err)
	}
	auditLog.Printf("Using output directory: %s", runOutputDir)

	// If job ID is provided, handle job-specific audit
	if jobID > 0 {
		return auditJobRun(auditJobRunOptions{
			runID:      runID,
			jobID:      jobID,
			stepNumber: stepNumber,
			owner:      owner,
			repo:       repo,
			hostname:   hostname,
			outputDir:  runOutputDir,
			verbose:    verbose,
			jsonOutput: jsonOutput,
		})
	}

	// Use cached run summary when available to ensure deterministic metrics across repeated calls.
	// Re-processing the same log files can produce different results (e.g. when GitHub's API
	// returns aggregated data that differs from the locally-stored firewall logs), so we always
	// prefer the first fully-processed summary written to disk.  The cache is automatically
	// invalidated whenever the CLI version changes (see loadRunSummary).
	if summary, ok := loadRunSummary(runOutputDir, verbose); ok {
		auditLog.Printf("Using cached run summary for run %d (processed at %s)", runID, summary.ProcessedAt.Format(time.RFC3339))
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Using cached run summary for run %d (processed at %s)", runID, summary.ProcessedAt.Format(time.RFC3339))))
		}
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
		// Override the cached LogsPath with the current runOutputDir so that downstream
		// file reads (created items, aw_info, etc.) resolve correctly even if the run
		// directory has been moved or copied since the summary was first written.
		processedRun.Run.LogsPath = runOutputDir

		// Apply experiment filter before rendering when flags are active.
		if experimentFilter != "" {
			expData := extractExperimentData(runOutputDir)
			if !experimentMatchesFilter(expData, experimentFilter, variantFilter) {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
					formatExperimentSkipMessage(runID, experimentFilter, variantFilter),
				))
				return nil
			}
		}

		return renderAuditReport(ctx, processedRun, summary.Metrics, summary.MCPToolUsage, AuditOptions{
			Owner:      owner,
			Repo:       repo,
			Hostname:   hostname,
			OutputDir:  runOutputDir,
			Verbose:    verbose,
			Parse:      parse,
			JSONOutput: jsonOutput,
		})
	}

	// Check if we have locally cached artifacts first
	hasLocalCache := fileutil.DirExists(runOutputDir) && !fileutil.IsDirEmpty(runOutputDir)

	// Try to get run metadata from GitHub API
	run, metadataErr := fetchWorkflowRunMetadata(ctx, runID, owner, repo, hostname, verbose)
	var useLocalCache bool

	if metadataErr != nil {
		// Check if it's a permission error
		if isPermissionError(metadataErr) {
			if hasLocalCache {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("GitHub API access denied, but found locally cached artifacts. Processing cached data..."))
				useLocalCache = true
			} else {
				// Provide helpful message about using GitHub MCP server
				return fmt.Errorf("GitHub API access denied and no local cache found.\n\n"+
					"To download artifacts, use the GitHub MCP server:\n\n"+
					"1. Use the github-mcp-server tool 'download_workflow_run_artifacts' with:\n"+
					"   - run_id: %d\n"+
					"   - output_directory: %s\n\n"+
					"2. After downloading, run this audit command again to analyze the cached artifacts.\n\n"+
					"Original error: %v", runID, runOutputDir, metadataErr)
			}
		} else {
			return fmt.Errorf("failed to fetch run metadata: %w", metadataErr)
		}
	}

	if !useLocalCache {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Run: %s (Status: %s, Conclusion: %s)", run.WorkflowName, run.Status, run.Conclusion)))
		}

		// Download artifacts for the run
		auditLog.Printf("Downloading artifacts for run %d", runID)
		err := downloadRunArtifacts(ctx, runID, runOutputDir, verbose, owner, repo, hostname, artifactFilter)
		if err != nil {
			// Gracefully handle cases where the run legitimately has no artifacts
			if errors.Is(err, ErrNoArtifacts) {
				auditLog.Printf("No artifacts found for run %d", runID)
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No artifacts attached to this run. Proceeding with metadata-only audit."))
				}
			} else if isPermissionError(err) {
				if hasLocalCache {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Artifact download failed due to permissions, but found locally cached artifacts. Processing cached data..."))
					useLocalCache = true
				} else {
					return fmt.Errorf("failed to download artifacts due to permissions and no local cache found.\n\n"+
						"To download artifacts, use the GitHub MCP server:\n\n"+
						"1. Use the github-mcp-server tool 'download_workflow_run_artifacts' with:\n"+
						"   - run_id: %d\n"+
						"   - output_directory: %s\n\n"+
						"2. After downloading, run this audit command again to analyze the cached artifacts.\n\n"+
						"Original error: %v", runID, runOutputDir, err)
				}
			} else {
				return fmt.Errorf("failed to download artifacts: %w", err)
			}
		}
	}

	// If using local cache without metadata, create a minimal run structure
	if useLocalCache && run.DatabaseID == 0 {
		run = WorkflowRun{
			DatabaseID:   runID,
			WorkflowName: fmt.Sprintf("Workflow Run %d", runID),
			Status:       "unknown",
			LogsPath:     runOutputDir,
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Using locally cached artifacts without metadata. Some report details may be unavailable."))
	}

	// Set LogsPath early so all goroutines receive a fully-initialised run value.
	run.LogsPath = runOutputDir

	// Calculate duration before launching goroutines so it is visible to any
	// analysis function that uses run.Duration.
	if !run.StartedAt.IsZero() && !run.UpdatedAt.IsZero() {
		run.Duration = run.UpdatedAt.Sub(run.StartedAt)
	}

	// Determine whether firewall-related artifacts should be expected before
	// spawning goroutines, so the conditional goroutines can be started together
	// with all other tasks.
	hasFirewallArtifact := artifactMatchesFilter(constants.AgentArtifactName, artifactFilter)

	// Run all independent analysis tasks concurrently to reduce total audit time.
	// Each goroutine writes to its own distinct variable; the main goroutine only
	// reads those variables after wg.Wait(), so there are no data races.
	var (
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
	)
	var wg sync.WaitGroup

	// Log metrics extraction – reads agent log files; can be slow for large runs.
	wg.Go(func() {
		var err error
		metrics, err = extractLogMetrics(runOutputDir, verbose, run.WorkflowPath)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract metrics: %v", err)))
			}
			metrics = LogMetrics{}
		}
	})

	// Fetch job details and derive failedJobCount from a single API call.
	wg.Go(func() {
		var err error
		jobDetails, failedJobCount, err = fetchJobDetailsWithCounts(run.DatabaseID, verbose)
		if err != nil {
			auditLog.Printf("fetchJobDetailsWithCounts failed: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch job details: %v", err)))
			}
			return
		}
	})

	wg.Go(func() {
		var err error
		missingTools, err = extractMissingToolsFromRun(runOutputDir, run, verbose)
		if err != nil {
			auditLog.Printf("extractMissingToolsFromRun failed: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract missing tools: %v", err)))
			}
		}
	})

	wg.Go(func() {
		var err error
		missingData, err = extractMissingDataFromRun(runOutputDir, run, verbose)
		if err != nil {
			auditLog.Printf("extractMissingDataFromRun failed: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract missing data: %v", err)))
			}
		}
	})

	wg.Go(func() {
		var err error
		noops, err = extractNoopsFromRun(runOutputDir, run, verbose)
		if err != nil {
			auditLog.Printf("extractNoopsFromRun failed: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract noops: %v", err)))
			}
		}
	})

	wg.Go(func() {
		var err error
		mcpFailures, err = extractMCPFailuresFromRun(runOutputDir, run, verbose)
		if err != nil {
			auditLog.Printf("extractMCPFailuresFromRun failed: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract MCP failures: %v", err)))
			}
		}
	})

	wg.Go(func() {
		var err error
		accessAnalysis, err = analyzeAccessLogs(runOutputDir, verbose)
		if err != nil {
			auditLog.Printf("analyzeAccessLogs failed: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze access logs: %v", err)))
			}
		}
	})

	// Firewall-related tasks are conditional on the agent artifact being present.
	// analyzeFirewallLogs and extractFirewallFromAgentLog must remain sequential
	// within one goroutine because the agent-log result is merged into the
	// firewall-log result. The other three firewall tasks are fully independent.
	if hasFirewallArtifact {
		wg.Go(func() {
			fa, err := analyzeFirewallLogs(runOutputDir, verbose)
			if err != nil {
				auditLog.Printf("analyzeFirewallLogs failed: %v", err)
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze firewall logs: %v", err)))
				}
			}
			// Supplement firewall analysis with blocked domains extracted directly
			// from agent-stdio.log (e.g. Codex CLI emits "--allow-domains <domain>"
			// warnings when the sandbox firewall denies a network request).
			if agentLogFirewall := extractFirewallFromAgentLog(runOutputDir, verbose); agentLogFirewall != nil {
				if fa == nil {
					fa = agentLogFirewall
				} else {
					fa.AddMetrics(agentLogFirewall)
				}
			}
			firewallAnalysis = fa
		})

		wg.Go(func() {
			var err error
			policyAnalysis, err = analyzeFirewallPolicy(runOutputDir, verbose)
			if err != nil {
				auditLog.Printf("analyzeFirewallPolicy failed: %v", err)
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze firewall policy: %v", err)))
				}
			}
		})

		wg.Go(func() {
			var err error
			mcpToolUsage, err = extractMCPToolUsageData(runOutputDir, verbose)
			if err != nil {
				auditLog.Printf("extractMCPToolUsageData failed: %v", err)
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract MCP tool usage: %v", err)))
				}
			}
		})

		wg.Go(func() {
			var err error
			tokenUsageSummary, err = analyzeTokenUsage(runOutputDir, verbose)
			if err != nil {
				auditLog.Printf("analyzeTokenUsage failed: %v", err)
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze token usage: %v", err)))
				}
			}
		})
	}

	wg.Go(func() {
		var err error
		redactedDomainsAnalysis, err = analyzeRedactedDomains(runOutputDir, verbose)
		if err != nil {
			auditLog.Printf("analyzeRedactedDomains failed: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze redacted domains: %v", err)))
			}
		}
	})

	wg.Go(func() {
		var err error
		rateLimitUsage, err = analyzeGitHubRateLimits(runOutputDir, verbose)
		if err != nil {
			auditLog.Printf("analyzeGitHubRateLimits failed: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze GitHub rate limit usage: %v", err)))
			}
		}
	})

	wg.Go(func() {
		var err error
		artifacts, err = listArtifacts(runOutputDir)
		if err != nil {
			auditLog.Printf("listArtifacts failed: %v", err)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to list artifacts: %v", err)))
			}
		}
	})

	wg.Go(func() {
		safeItemsCount = len(extractCreatedItemsFromManifest(runOutputDir))
	})

	// Wait for all analysis tasks to complete before assembling the report.
	wg.Wait()

	// Update run with aggregated results from parallel tasks.
	run.TokenUsage = metrics.TokenUsage
	run.Turns = metrics.Turns
	run.ErrorCount = failedJobCount
	run.WarningCount = 0
	run.SafeItemsCount = safeItemsCount

	// Create processed run for report generation
	processedRun := ProcessedRun{
		Run:                     run,
		FirewallAnalysis:        firewallAnalysis,
		PolicyAnalysis:          policyAnalysis,
		RedactedDomainsAnalysis: redactedDomainsAnalysis,
		MissingTools:            missingTools,
		MissingData:             missingData,
		Noops:                   noops,
		MCPFailures:             mcpFailures,
		TokenUsage:              tokenUsageSummary,
		GitHubRateLimitUsage:    rateLimitUsage,
		JobDetails:              jobDetails,
	}
	awContext, _, _, taskDomain, behaviorFingerprint, agenticAssessments := deriveRunAgenticAnalysis(processedRun, metrics)
	processedRun.AwContext = awContext
	processedRun.TaskDomain = taskDomain
	processedRun.BehaviorFingerprint = behaviorFingerprint
	processedRun.AgenticAssessments = agenticAssessments

	// Save run summary for caching future audit runs
	summary := &RunSummary{
		CLIVersion:              GetVersion(),
		RunID:                   run.DatabaseID,
		ProcessedAt:             time.Now(),
		Run:                     run,
		Metrics:                 metrics,
		AwContext:               processedRun.AwContext,
		TaskDomain:              processedRun.TaskDomain,
		BehaviorFingerprint:     processedRun.BehaviorFingerprint,
		AgenticAssessments:      processedRun.AgenticAssessments,
		AccessAnalysis:          accessAnalysis,
		FirewallAnalysis:        firewallAnalysis,
		PolicyAnalysis:          policyAnalysis,
		RedactedDomainsAnalysis: redactedDomainsAnalysis,
		MissingTools:            missingTools,
		MissingData:             missingData,
		Noops:                   noops,
		MCPFailures:             mcpFailures,
		MCPToolUsage:            mcpToolUsage,
		TokenUsage:              tokenUsageSummary,
		GitHubRateLimitUsage:    rateLimitUsage,
		ArtifactsList:           artifacts,
		JobDetails:              jobDetails,
	}

	if err := saveRunSummary(runOutputDir, summary, verbose); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to save run summary: %v", err)))
		}
	}

	// Apply experiment filter before rendering when flags are active.
	if experimentFilter != "" {
		expData := extractExperimentData(runOutputDir)
		if !experimentMatchesFilter(expData, experimentFilter, variantFilter) {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(
				formatExperimentSkipMessage(runID, experimentFilter, variantFilter),
			))
			return nil
		}
	}

	return renderAuditReport(ctx, processedRun, metrics, mcpToolUsage, AuditOptions{
		Owner:      owner,
		Repo:       repo,
		Hostname:   hostname,
		OutputDir:  runOutputDir,
		Verbose:    verbose,
		Parse:      parse,
		JSONOutput: jsonOutput,
	})
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

// auditJobRunOptions holds parameters for auditJobRun.
type auditJobRunOptions struct {
	runID      int64
	jobID      int64
	stepNumber int
	owner      string
	repo       string
	hostname   string
	outputDir  string
	verbose    bool
	jsonOutput bool
}

// auditJobRun performs a targeted audit of a specific job within a workflow run
// If stepNumber > 0, focuses on extracting output for that specific step
func auditJobRun(opts auditJobRunOptions) error {
	// Auto-detect GHES host from git remote if hostname is not provided
	if opts.hostname == "" {
		opts.hostname = getHostFromOriginRemote()
		if opts.hostname != "github.com" {
			auditLog.Printf("Auto-detected GHES host from git remote: %s", opts.hostname)
		}
	}

	auditLog.Printf("Starting job-specific audit: runID=%d, jobID=%d, stepNumber=%d, hostname=%s", opts.runID, opts.jobID, opts.stepNumber, opts.hostname)

	// Create output directory for job-specific artifacts
	if err := os.MkdirAll(opts.outputDir, constants.DirPermSensitive); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Fetch job logs using gh CLI.
	// Use GH_HOST env var instead of --hostname (which is only valid for gh api, not gh run view).
	args := []string{"run", "view"}

	// Add repository flag if specified
	if opts.owner != "" && opts.repo != "" {
		args = append(args, "-R", fmt.Sprintf("%s/%s", opts.owner, opts.repo))
	}

	args = append(args, "--job", strconv.FormatInt(opts.jobID, 10), "--log")

	if opts.verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Fetching logs for job %d...", opts.jobID)))
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Executing: gh "+strings.Join(args, " ")))
	}

	cmd := workflow.ExecGH(args...)
	workflow.SetGHHostEnv(cmd, opts.hostname)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to fetch job logs: %w\nOutput: %s", err, string(output))
	}

	jobLogContent := string(output)

	// Save full job log
	jobLogPath := filepath.Join(opts.outputDir, fmt.Sprintf("job-%d.log", opts.jobID))
	if err := os.WriteFile(jobLogPath, []byte(jobLogContent), constants.FilePermSensitive); err != nil {
		return fmt.Errorf("failed to write job log: %w", err)
	}

	if opts.verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Job log saved to "+jobLogPath))
	}

	// If step number is specified, extract that step's output
	if opts.stepNumber > 0 {
		stepOutput, err := extractStepOutput(jobLogContent, opts.stepNumber)
		if err != nil {
			if opts.verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not extract step %d output: %v", opts.stepNumber, err)))
			}
		} else {
			stepLogPath := filepath.Join(opts.outputDir, fmt.Sprintf("job-%d-step-%d.log", opts.jobID, opts.stepNumber))
			if err := os.WriteFile(stepLogPath, []byte(stepOutput), constants.FilePermSensitive); err != nil {
				return fmt.Errorf("failed to write step log: %w", err)
			}
			if opts.verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Step %d output saved to %s", opts.stepNumber, stepLogPath)))
			}
		}
	} else {
		// No step specified, find and extract first failing step
		failingStepNum, failingStepOutput := findFirstFailingStep(jobLogContent)
		if failingStepNum > 0 {
			stepLogPath := filepath.Join(opts.outputDir, fmt.Sprintf("job-%d-step-%d-failed.log", opts.jobID, failingStepNum))
			if err := os.WriteFile(stepLogPath, []byte(failingStepOutput), constants.FilePermSensitive); err != nil {
				return fmt.Errorf("failed to write failing step log: %w", err)
			}
			if opts.verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("First failing step %d output saved to %s", failingStepNum, stepLogPath)))
			}
		} else if opts.verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No failing steps found in job"))
		}
	}

	// Display summary
	if !opts.jsonOutput {
		absOutputDir, _ := filepath.Abs(opts.outputDir)
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Job audit complete. Logs saved to "+absOutputDir))

		// Display file locations
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("\nDownloaded files:"))
		fmt.Fprintf(os.Stderr, "  - %s (full job log)\n", jobLogPath)

		if opts.stepNumber > 0 {
			stepLogPath := filepath.Join(opts.outputDir, fmt.Sprintf("job-%d-step-%d.log", opts.jobID, opts.stepNumber))
			if fileutil.FileExists(stepLogPath) {
				fmt.Fprintf(os.Stderr, "  - %s (step %d output)\n", stepLogPath, opts.stepNumber)
			}
		} else {
			failingStepPath := filepath.Join(opts.outputDir, fmt.Sprintf("job-%d-step-*-failed.log", opts.jobID))
			matches, _ := filepath.Glob(failingStepPath)
			for _, match := range matches {
				fmt.Fprintf(os.Stderr, "  - %s (first failing step)\n", match)
			}
		}
	}

	return nil
}

// extractStepOutput extracts the output of a specific step from job logs
func extractStepOutput(jobLog string, stepNumber int) (string, error) {
	auditLog.Printf("Extracting output for step %d from job logs (%d bytes)", stepNumber, len(jobLog))
	lines := strings.Split(jobLog, "\n")
	var stepOutput []string
	inStep := false
	stepPattern := "##[group]Run " // GitHub Actions step marker
	stepEndPattern := "##[endgroup]"
	currentStep := 0

	for _, line := range lines {
		// Detect step boundaries
		if strings.Contains(line, stepPattern) || strings.HasPrefix(line, fmt.Sprintf("##[group]Step %d:", stepNumber)) {
			currentStep++
			if currentStep == stepNumber {
				inStep = true
			}
		} else if strings.Contains(line, stepEndPattern) {
			if inStep {
				break // End of target step
			}
		}

		if inStep {
			stepOutput = append(stepOutput, line)
		}
	}

	if len(stepOutput) == 0 {
		auditLog.Printf("Step %d not found in job logs (scanned %d lines)", stepNumber, len(lines))
		return "", fmt.Errorf("step %d not found in job logs", stepNumber)
	}

	auditLog.Printf("Extracted %d lines for step %d", len(stepOutput), stepNumber)
	return strings.Join(stepOutput, "\n"), nil
}

// findFirstFailingStep finds the first step that failed in the job logs
func findFirstFailingStep(jobLog string) (int, string) {
	auditLog.Printf("Searching for first failing step in job logs (%d bytes)", len(jobLog))
	lines := strings.Split(jobLog, "\n")
	var stepOutput []string
	inStep := false
	currentStep := 0
	foundFailure := false

	for _, line := range lines {
		// Detect step start
		if strings.Contains(line, "##[group]") {
			if inStep && foundFailure {
				break // We found a complete failing step
			}
			inStep = true
			currentStep++
			stepOutput = []string{line}
			foundFailure = false
		} else if inStep {
			stepOutput = append(stepOutput, line)

			// Detect failure indicators
			if strings.Contains(line, "##[error]") ||
				strings.Contains(line, "Error:") ||
				strings.Contains(line, "FAILED") ||
				strings.Contains(line, "exit code") && !strings.Contains(line, "exit code 0") {
				foundFailure = true
			}
		}
	}

	if foundFailure && len(stepOutput) > 0 {
		auditLog.Printf("Found failing step %d with %d lines of output", currentStep, len(stepOutput))
		return currentStep, strings.Join(stepOutput, "\n")
	}

	auditLog.Print("No failing step found in job logs")
	return 0, ""
}

// fetchWorkflowRunMetadata fetches metadata for a single workflow run
func fetchWorkflowRunMetadata(ctx context.Context, runID int64, owner, repo, hostname string, verbose bool) (WorkflowRun, error) {
	// Build the API endpoint
	var endpoint string
	if owner != "" && repo != "" {
		// Use explicit owner/repo from the URL
		endpoint = fmt.Sprintf("repos/%s/%s/actions/runs/%d", owner, repo, runID)
	} else {
		// Fall back to {owner}/{repo} placeholders for context-based resolution
		endpoint = fmt.Sprintf("repos/{owner}/{repo}/actions/runs/%d", runID)
	}

	args := []string{"api"}

	// Add hostname flag if specified (for GitHub Enterprise)
	if hostname != "" && hostname != "github.com" {
		args = append(args, "--hostname", hostname)
	}

	args = append(args,
		endpoint,
		"--jq",
		"{databaseId: .id, number: .run_number, url: .html_url, status: .status, conclusion: .conclusion, workflowName: .name, workflowPath: .path, createdAt: .created_at, startedAt: .run_started_at, updatedAt: .updated_at, event: .event, headBranch: .head_branch, headSha: .head_sha, displayTitle: .display_title}",
	)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Executing: gh "+strings.Join(args, " ")))
	}

	output, err := workflow.RunGHCombinedContext(ctx, "Fetching run metadata...", args...)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(string(output)))
		}
		// Provide a human-readable error when the run ID doesn't exist.
		// The gh CLI may surface the 404 in the Go error (checked via errorutil.IsNotFoundError)
		// or in its combined stdout/stderr output (checked below) depending on the CLI version.
		// "Could not resolve" catches DNS failures from git clone fallbacks.
		outputStr := string(output)
		if errorutil.IsNotFoundError(err) ||
			errorutil.IsNotFoundError(errors.New(outputStr)) ||
			strings.Contains(outputStr, "Could not resolve") {
			return WorkflowRun{}, fmt.Errorf("workflow run %d not found. Please verify the run ID is correct and that you have access to the repository", runID)
		}
		return WorkflowRun{}, fmt.Errorf("failed to fetch run metadata: %w", err)
	}

	var run WorkflowRun
	if err := json.Unmarshal(output, &run); err != nil {
		return WorkflowRun{}, fmt.Errorf("failed to parse run metadata: %w", err)
	}

	// When the GitHub API returns the workflow file path as the run's name (e.g. for runs
	// that were cancelled or failed before any jobs started), resolve the actual workflow
	// display name so that audit output is consistent with 'gh aw logs'.
	if strings.HasPrefix(run.WorkflowName, ".github/") {
		if displayName := resolveWorkflowDisplayName(ctx, run.WorkflowPath, owner, repo, hostname); displayName != "" {
			auditLog.Printf("Resolved workflow display name: %q -> %q", run.WorkflowName, displayName)
			run.WorkflowName = displayName
		}
	}

	return run, nil
}

// resolveWorkflowDisplayName returns the human-readable display name for a workflow file.
// It first attempts to read the YAML file from the local filesystem (resolving the path
// relative to the git repository root so that it works from any working directory inside
// the repo); if that fails it falls back to a GitHub API call.  An empty string is
// returned on any error so that callers can gracefully keep the original value.
func resolveWorkflowDisplayName(ctx context.Context, workflowPath, owner, repo, hostname string) string {
	// Try local file first.  workflowPath is a repo-relative path like
	// ".github/workflows/foo.lock.yml", so we resolve it against the git root to
	// produce a correct absolute path regardless of the current working directory.
	if gitRoot, err := gitutil.FindGitRoot(); err == nil {
		absPath := filepath.Join(gitRoot, workflowPath)
		if content, err := os.ReadFile(absPath); err == nil {
			if name := extractWorkflowNameFromYAML(content); name != "" {
				return name
			}
		}
	}

	// Fall back to the GitHub Actions workflows API.
	filename := filepath.Base(workflowPath)
	var endpoint string
	if owner != "" && repo != "" {
		endpoint = fmt.Sprintf("repos/%s/%s/actions/workflows/%s", owner, repo, filename)
	} else {
		endpoint = "repos/{owner}/{repo}/actions/workflows/" + filename
	}

	args := []string{"api"}
	if hostname != "" && hostname != "github.com" {
		args = append(args, "--hostname", hostname)
	}
	args = append(args, endpoint, "--jq", ".name")

	out, err := workflow.RunGHCombinedContext(ctx, "Fetching workflow name...", args...)
	if err != nil {
		auditLog.Printf("Failed to fetch workflow display name for %q: %v", workflowPath, err)
		return ""
	}

	return strings.TrimSpace(string(out))
}

// extractWorkflowNameFromYAML parses a GitHub Actions workflow YAML document and
// returns the value of its top-level "name:" field.  An empty string is returned
// when the field is absent or the document cannot be parsed.
func extractWorkflowNameFromYAML(content []byte) string {
	var wf struct {
		Name string `yaml:"name"`
	}
	if err := yaml.Unmarshal(content, &wf); err != nil {
		auditLog.Printf("Failed to parse workflow YAML for name extraction (file may be malformed): %v", err)
		return ""
	}
	return wf.Name
}
