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
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

var auditLog = logger.New("cli:audit")

// NewAuditCommand creates the audit command
func NewAuditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit <run-id-or-url>",
		Short: "Audit a workflow run and generate a detailed report",
		Long: `Audit a single workflow run by downloading artifacts and logs, detecting errors,
analyzing MCP tool usage, and generating a concise Markdown report suitable for AI agents.

This command accepts:
- A numeric run ID (e.g., 1234567890)
- A GitHub Actions run URL (e.g., https://github.com/owner/repo/actions/runs/1234567890)
- A GitHub Actions job URL (e.g., https://github.com/owner/repo/actions/runs/1234567890/job/9876543210)
- A GitHub Actions job URL with step (e.g., https://github.com/owner/repo/actions/runs/1234567890/job/9876543210#step:7:1)
- A GitHub workflow run URL (e.g., https://github.com/owner/repo/runs/1234567890)
- GitHub Enterprise URLs (e.g., https://github.example.com/owner/repo/actions/runs/1234567890)

When a job URL is provided:
- If a step number is included (#step:7:1), extracts that specific step's output
- If no step number, finds and extracts the first failing step's output
- Saves job logs to the output directory

Examples:
  ` + string(constants.CLIExtensionPrefix) + ` audit 1234567890     # Audit run with ID 1234567890
  ` + string(constants.CLIExtensionPrefix) + ` audit https://github.com/owner/repo/actions/runs/1234567890  # Audit from run URL
  ` + string(constants.CLIExtensionPrefix) + ` audit https://github.com/owner/repo/actions/runs/1234567890/job/9876543210  # Audit job and extract first failing step
  ` + string(constants.CLIExtensionPrefix) + ` audit https://github.com/owner/repo/actions/runs/1234567890/job/9876543210#step:7:1  # Extract step 7 output
  ` + string(constants.CLIExtensionPrefix) + ` audit https://github.com/owner/repo/runs/1234567890  # Audit from workflow run URL
  ` + string(constants.CLIExtensionPrefix) + ` audit https://github.example.com/owner/repo/actions/runs/1234567890  # Audit from GitHub Enterprise
  ` + string(constants.CLIExtensionPrefix) + ` audit 1234567890 -o ./audit-reports  # Custom output directory
  ` + string(constants.CLIExtensionPrefix) + ` audit 1234567890 -v  # Verbose output
  ` + string(constants.CLIExtensionPrefix) + ` audit 1234567890 --parse  # Parse agent logs and firewall logs, generating log.md and firewall.md
  ` + string(constants.CLIExtensionPrefix) + ` audit 1234567890 --repo owner/repo  # Audit run from a specific repository`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runIDOrURL := args[0]

			// Parse run information from input (either numeric ID or URL)
			// Use extended parsing to capture job ID and step information
			components, err := parser.ParseRunURLExtended(runIDOrURL)
			if err != nil {
				return err
			}

			outputDir, _ := cmd.Flags().GetString("output")
			verbose, _ := cmd.Flags().GetBool("verbose")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			parse, _ := cmd.Flags().GetBool("parse")
			repoFlag, _ := cmd.Flags().GetString("repo")
			artifacts, _ := cmd.Flags().GetStringSlice("artifacts")

			// If --repo is provided and owner/repo were not parsed from a URL, apply them
			if repoFlag != "" && components.Owner == "" {
				parts := strings.SplitN(repoFlag, "/", 2)
				if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
					return fmt.Errorf("invalid repository format '%s': expected 'owner/repo'", repoFlag)
				}
				components.Owner = parts[0]
				components.Repo = parts[1]
			}

			return AuditWorkflowRun(
				cmd.Context(),
				components.Number,
				components.Owner,
				components.Repo,
				components.Host,
				outputDir,
				verbose,
				parse,
				jsonOutput,
				components.JobID,
				components.StepNumber,
				artifacts,
			)
		},
	}

	// Add flags to audit command
	addOutputFlag(cmd, defaultLogsOutputDir)
	addJSONFlag(cmd)
	addRepoFlag(cmd)
	cmd.Flags().Bool("parse", false, "Run JavaScript parsers on agent logs and firewall logs, writing Markdown to log.md and firewall.md")
	cmd.Flags().StringSlice("artifacts", nil, "Artifact sets to download (default: all). Valid sets: "+strings.Join(ValidArtifactSetNames(), ", "))

	// Register completions for audit command
	RegisterDirFlagCompletion(cmd, "output")

	// Add subcommands
	cmd.AddCommand(NewAuditDiffSubcommand())

	return cmd
}

// isPermissionError checks if an error is related to permissions/authentication
func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "authentication required") ||
		strings.Contains(errStr, "exit status 4") ||
		strings.Contains(errStr, "GitHub CLI authentication") ||
		strings.Contains(errStr, "permission") ||
		strings.Contains(errStr, "GH_TOKEN")
}

// AuditWorkflowRun audits a single workflow run and generates a report
// If jobID is provided (>0), focuses audit on that specific job
// If stepNumber is provided (>0), extracts output for that specific step
func AuditWorkflowRun(ctx context.Context, runID int64, owner, repo, hostname string, outputDir string, verbose bool, parse bool, jsonOutput bool, jobID int64, stepNumber int, artifactSets []string) error {
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
		return auditJobRun(runID, jobID, stepNumber, owner, repo, hostname, runOutputDir, verbose, jsonOutput)
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
		return renderAuditReport(ctx, processedRun, summary.Metrics, summary.MCPToolUsage, runOutputDir, owner, repo, hostname, verbose, parse, jsonOutput)
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

	// Extract metrics from logs
	metrics, err := extractLogMetrics(runOutputDir, verbose, run.WorkflowPath)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract metrics: %v", err)))
		}
		metrics = LogMetrics{}
	}

	// Update run with metrics
	run.TokenUsage = metrics.TokenUsage
	run.EstimatedCost = metrics.EstimatedCost
	run.Turns = metrics.Turns
	run.ErrorCount = 0
	run.WarningCount = 0
	run.LogsPath = runOutputDir

	// Calculate duration
	if !run.StartedAt.IsZero() && !run.UpdatedAt.IsZero() {
		run.Duration = run.UpdatedAt.Sub(run.StartedAt)
	}

	// Add failed jobs to error count
	if failedJobCount, err := fetchJobStatuses(run.DatabaseID, verbose); err == nil {
		run.ErrorCount += failedJobCount
		if verbose && failedJobCount > 0 {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Added %d failed jobs to error count", failedJobCount)))
		}
	}

	// Fetch detailed job information including durations
	jobDetails, err := fetchJobDetails(run.DatabaseID, verbose)
	if err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch job details: %v", err)))
	}

	// Extract missing tools
	missingTools, err := extractMissingToolsFromRun(runOutputDir, run, verbose)
	if err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract missing tools: %v", err)))
	}

	// Extract missing data
	missingData, err := extractMissingDataFromRun(runOutputDir, run, verbose)
	if err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract missing data: %v", err)))
	}

	// Extract noops
	noops, noopErr := extractNoopsFromRun(runOutputDir, run, verbose)
	if noopErr != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract noops: %v", noopErr)))
	}

	// Extract MCP failures
	mcpFailures, err := extractMCPFailuresFromRun(runOutputDir, run, verbose)
	if err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract MCP failures: %v", err)))
	}

	// Analyze access logs if available
	accessAnalysis, err := analyzeAccessLogs(runOutputDir, verbose)
	if err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze access logs: %v", err)))
	}

	// Analyze firewall/gateway data only when the agent artifact was downloaded.
	// Firewall audit logs are now included in the unified agent artifact.
	// Skip silently when the artifact was intentionally excluded from the filter to
	// avoid spurious "not found" warnings in verbose mode.
	hasFirewallArtifact := artifactMatchesFilter(constants.AgentArtifactName, artifactFilter)

	// Analyze firewall logs if available
	var firewallAnalysis *FirewallAnalysis
	var policyAnalysis *PolicyAnalysis
	var mcpToolUsage *MCPToolUsageData
	var tokenUsageSummary *TokenUsageSummary
	if hasFirewallArtifact {
		firewallAnalysis, err = analyzeFirewallLogs(runOutputDir, verbose)
		if err != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze firewall logs: %v", err)))
		}

		// Supplement firewall analysis with blocked domains extracted directly from
		// agent-stdio.log (e.g., Codex CLI emits "--allow-domains <domain>" warnings
		// when the sandbox firewall denies a network request).
		if agentLogFirewall := extractFirewallFromAgentLog(runOutputDir, verbose); agentLogFirewall != nil {
			if firewallAnalysis == nil {
				firewallAnalysis = agentLogFirewall
			} else {
				firewallAnalysis.AddMetrics(agentLogFirewall)
			}
		}

		// Analyze firewall policy artifacts if available (policy-manifest.json + audit.jsonl)
		policyAnalysis, err = analyzeFirewallPolicy(runOutputDir, verbose)
		if err != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze firewall policy: %v", err)))
		}

		// Extract MCP tool usage data from gateway logs
		mcpToolUsage, err = extractMCPToolUsageData(runOutputDir, verbose)
		if err != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract MCP tool usage: %v", err)))
		}

		// Analyze token usage from firewall proxy logs
		tokenUsageSummary, err = analyzeTokenUsage(runOutputDir, verbose)
		if err != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze token usage: %v", err)))
		}
	}

	// Analyze redacted domains if available
	redactedDomainsAnalysis, err := analyzeRedactedDomains(runOutputDir, verbose)
	if err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze redacted domains: %v", err)))
	}

	// Analyze GitHub API rate limit consumption from github_rate_limits.jsonl
	rateLimitUsage, err := analyzeGitHubRateLimits(runOutputDir, verbose)
	if err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze GitHub rate limit usage: %v", err)))
	}

	// List all artifacts
	artifacts, err := listArtifacts(runOutputDir)
	if err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to list artifacts: %v", err)))
	}

	currentCreatedItems := extractCreatedItemsFromManifest(runOutputDir)
	run.SafeItemsCount = len(currentCreatedItems)

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

	return renderAuditReport(ctx, processedRun, metrics, mcpToolUsage, runOutputDir, owner, repo, hostname, verbose, parse, jsonOutput)
}

// renderAuditReport builds and renders the audit report from a fully-populated processedRun.
// It is called both when serving from a cached run summary and after a fresh processing pass,
// ensuring that the two paths produce identical output.
func renderAuditReport(ctx context.Context, processedRun ProcessedRun, metrics LogMetrics, mcpToolUsage *MCPToolUsageData, runOutputDir string, owner, repo, hostname string, verbose bool, parse bool, jsonOutput bool) error {
	runID := processedRun.Run.DatabaseID

	currentCreatedItems := extractCreatedItemsFromManifest(runOutputDir)
	processedRun.Run.SafeItemsCount = len(currentCreatedItems)

	currentSnapshot := buildAuditComparisonSnapshot(processedRun, currentCreatedItems)
	comparison := buildAuditComparisonForRun(ctx, processedRun, currentSnapshot, runOutputDir, owner, repo, hostname, verbose)

	// Build structured audit data
	auditData := buildAuditData(processedRun, metrics, mcpToolUsage)
	auditData.Comparison = comparison

	// Render output based on format preference
	if jsonOutput {
		if err := renderJSON(auditData); err != nil {
			return fmt.Errorf("failed to render JSON output: %w", err)
		}
	} else {
		renderConsole(auditData, runOutputDir)
	}

	// Display gateway metrics if available
	if gatewayMetrics, err := parseGatewayLogs(runOutputDir, verbose); err == nil {
		if metricsOutput := renderGatewayMetricsTable(gatewayMetrics, verbose); metricsOutput != "" {
			fmt.Fprint(os.Stderr, metricsOutput)
		}
	}

	// Conditionally attempt to render agentic log (similar to `logs --parse`) if --parse flag is set
	// This creates a log.md file in the run directory for a rich, human-readable agent session summary.
	// We intentionally do not fail the audit on parse errors; they are reported as warnings.
	if parse {
		awInfoPath := filepath.Join(runOutputDir, "aw_info.json")
		if engine := extractEngineFromAwInfo(awInfoPath, verbose); engine != nil { // reuse existing helper in same package
			if err := parseAgentLog(runOutputDir, engine, verbose); err != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse agent log for run %d: %v", runID, err)))
				}
			} else {
				// Always show success message for parsing, not just in verbose mode
				logMdPath := filepath.Join(runOutputDir, "log.md")
				if _, err := os.Stat(logMdPath); err == nil {
					fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Parsed log for run %d → %s", runID, logMdPath)))
				}
			}
		} else if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No engine detected (aw_info.json missing or invalid); skipping agent log rendering"))
		}

		// Also parse firewall logs if they exist
		if err := parseFirewallLogs(runOutputDir, verbose); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse firewall logs for run %d: %v", runID, err)))
			}
		} else {
			// Show success message if firewall.md was created
			firewallMdPath := filepath.Join(runOutputDir, "firewall.md")
			if _, err := os.Stat(firewallMdPath); err == nil {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Parsed firewall logs for run %d → %s", runID, firewallMdPath)))
			}
		}
	}

	// Display logs location (only for console output)
	if !jsonOutput {
		absOutputDir, _ := filepath.Abs(runOutputDir)
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Audit complete. Logs saved to "+absOutputDir))
	}

	return nil
}

// auditJobRun performs a targeted audit of a specific job within a workflow run
// If stepNumber > 0, focuses on extracting output for that specific step
func auditJobRun(runID int64, jobID int64, stepNumber int, owner, repo, hostname string, outputDir string, verbose bool, jsonOutput bool) error {
	// Auto-detect GHES host from git remote if hostname is not provided
	if hostname == "" {
		hostname = getHostFromOriginRemote()
		if hostname != "github.com" {
			auditLog.Printf("Auto-detected GHES host from git remote: %s", hostname)
		}
	}

	auditLog.Printf("Starting job-specific audit: runID=%d, jobID=%d, stepNumber=%d, hostname=%s", runID, jobID, stepNumber, hostname)

	// Create output directory for job-specific artifacts
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Fetch job logs using gh CLI.
	// Use GH_HOST env var instead of --hostname (which is only valid for gh api, not gh run view).
	args := []string{"run", "view"}

	// Add repository flag if specified
	if owner != "" && repo != "" {
		args = append(args, "-R", fmt.Sprintf("%s/%s", owner, repo))
	}

	args = append(args, "--job", strconv.FormatInt(jobID, 10), "--log")

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Fetching logs for job %d...", jobID)))
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Executing: gh "+strings.Join(args, " ")))
	}

	cmd := workflow.ExecGH(args...)
	workflow.SetGHHostEnv(cmd, hostname)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to fetch job logs: %w\nOutput: %s", err, string(output))
	}

	jobLogContent := string(output)

	// Save full job log
	jobLogPath := filepath.Join(outputDir, fmt.Sprintf("job-%d.log", jobID))
	if err := os.WriteFile(jobLogPath, []byte(jobLogContent), 0600); err != nil {
		return fmt.Errorf("failed to write job log: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Job log saved to "+jobLogPath))
	}

	// If step number is specified, extract that step's output
	if stepNumber > 0 {
		stepOutput, err := extractStepOutput(jobLogContent, stepNumber)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not extract step %d output: %v", stepNumber, err)))
			}
		} else {
			stepLogPath := filepath.Join(outputDir, fmt.Sprintf("job-%d-step-%d.log", jobID, stepNumber))
			if err := os.WriteFile(stepLogPath, []byte(stepOutput), 0600); err != nil {
				return fmt.Errorf("failed to write step log: %w", err)
			}
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Step %d output saved to %s", stepNumber, stepLogPath)))
			}
		}
	} else {
		// No step specified, find and extract first failing step
		failingStepNum, failingStepOutput := findFirstFailingStep(jobLogContent)
		if failingStepNum > 0 {
			stepLogPath := filepath.Join(outputDir, fmt.Sprintf("job-%d-step-%d-failed.log", jobID, failingStepNum))
			if err := os.WriteFile(stepLogPath, []byte(failingStepOutput), 0600); err != nil {
				return fmt.Errorf("failed to write failing step log: %w", err)
			}
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("First failing step %d output saved to %s", failingStepNum, stepLogPath)))
			}
		} else if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No failing steps found in job"))
		}
	}

	// Display summary
	if !jsonOutput {
		absOutputDir, _ := filepath.Abs(outputDir)
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Job audit complete. Logs saved to "+absOutputDir))

		// Display file locations
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("\nDownloaded files:"))
		fmt.Fprintf(os.Stderr, "  - %s (full job log)\n", jobLogPath)

		if stepNumber > 0 {
			stepLogPath := filepath.Join(outputDir, fmt.Sprintf("job-%d-step-%d.log", jobID, stepNumber))
			if _, err := os.Stat(stepLogPath); err == nil {
				fmt.Fprintf(os.Stderr, "  - %s (step %d output)\n", stepLogPath, stepNumber)
			}
		} else {
			failingStepPath := filepath.Join(outputDir, fmt.Sprintf("job-%d-step-*-failed.log", jobID))
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
		// GitHub CLI / API may surface the 404 in several forms depending on version.
		outputStr := string(output)
		if strings.Contains(outputStr, "Not Found") ||
			strings.Contains(outputStr, "404") ||
			strings.Contains(outputStr, "not found") ||
			strings.Contains(outputStr, "Could not resolve") ||
			strings.Contains(err.Error(), "404") {
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
