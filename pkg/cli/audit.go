package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

var auditLog = logger.New("cli:audit")

// NewAuditCommand creates the audit command
func NewAuditCommand() *cobra.Command {
	auditCmd := &cobra.Command{
		Use:   "audit <run-id-or-url>",
		Short: "Investigate a single GitHub Actions workflow run and generate a concise report",
		Long: `Audit a single workflow run by downloading artifacts and logs, detecting errors,
analyzing MCP tool usage, and generating a concise markdown report suitable for AI agents.

This command accepts:
- A numeric run ID (e.g., 1234567890)
- A GitHub Actions run URL (e.g., https://github.com/owner/repo/actions/runs/1234567890)
- A GitHub Actions job URL (e.g., https://github.com/owner/repo/actions/runs/1234567890/job/9876543210)
- A GitHub workflow run URL (e.g., https://github.com/owner/repo/runs/1234567890)
- GitHub Enterprise URLs (e.g., https://github.example.com/owner/repo/actions/runs/1234567890)

This command:
- Downloads artifacts and logs for the specified run ID
- Detects errors and warnings in the logs
- Analyzes MCP tool usage statistics
- Extracts missing tool reports
- Generates a concise markdown report

Examples:
  ` + constants.CLIExtensionPrefix + ` audit 1234567890     # Audit run with ID 1234567890
  ` + constants.CLIExtensionPrefix + ` audit https://github.com/owner/repo/actions/runs/1234567890  # Audit from run URL
  ` + constants.CLIExtensionPrefix + ` audit https://github.com/owner/repo/actions/runs/1234567890/job/9876543210  # Audit from job URL
  ` + constants.CLIExtensionPrefix + ` audit https://github.com/owner/repo/runs/1234567890  # Audit from workflow run URL
  ` + constants.CLIExtensionPrefix + ` audit https://github.example.com/owner/repo/actions/runs/1234567890  # Audit from GitHub Enterprise
  ` + constants.CLIExtensionPrefix + ` audit 1234567890 -o ./audit-reports  # Custom output directory
  ` + constants.CLIExtensionPrefix + ` audit 1234567890 -v  # Verbose output
  ` + constants.CLIExtensionPrefix + ` audit 1234567890 --parse  # Parse agent logs and generate log.md`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runIDOrURL := args[0]

			// Parse run information from input (either numeric ID or URL)
			runInfo, err := parseRunURL(runIDOrURL)
			if err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}

			outputDir, _ := cmd.Flags().GetString("output")
			verbose, _ := cmd.Flags().GetBool("verbose")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			parse, _ := cmd.Flags().GetBool("parse")

			if err := AuditWorkflowRun(runInfo, outputDir, verbose, parse, jsonOutput); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		},
	}

	// Add flags to audit command
	auditCmd.Flags().StringP("output", "o", "./logs", "Output directory for downloaded logs and artifacts")
	auditCmd.Flags().Bool("json", false, "Output audit report as JSON instead of formatted console tables")
	auditCmd.Flags().Bool("parse", false, "Run JavaScript parser on agent logs and write markdown to log.md")

	return auditCmd
}

// RunURLInfo contains the parsed information from a workflow run URL
type RunURLInfo struct {
	RunID    int64
	Owner    string
	Repo     string
	Hostname string
}

// extractRunID extracts the run ID from either a numeric string or a GitHub Actions URL
func extractRunID(input string) (int64, error) {
	info, err := parseRunURL(input)
	if err != nil {
		return 0, err
	}
	return info.RunID, nil
}

// parseRunURL parses a run ID or URL and extracts all relevant information
func parseRunURL(input string) (RunURLInfo, error) {
	// First try to parse as a direct numeric ID
	if runID, err := strconv.ParseInt(input, 10, 64); err == nil {
		return RunURLInfo{RunID: runID}, nil
	}

	// Try to extract information from GitHub Actions URL
	// Patterns:
	// - https://github.com/owner/repo/actions/runs/12345678
	// - https://github.com/owner/repo/actions/runs/12345678/job/98765432
	// - https://github.com/owner/repo/actions/runs/12345678/attempts/2
	// - https://github.com/owner/repo/runs/12345678 (action run URL)
	// - https://github.example.com/owner/repo/actions/runs/12345678 (enterprise)

	// Pattern to match GitHub URLs with run IDs
	// Captures: hostname, owner, repo, runID
	urlPattern := regexp.MustCompile(`^https?://([^/]+)/([^/]+)/([^/]+)/(?:actions/)?runs/(\d+)`)
	matches := urlPattern.FindStringSubmatch(input)

	if len(matches) >= 5 {
		runID, err := strconv.ParseInt(matches[4], 10, 64)
		if err != nil {
			return RunURLInfo{}, fmt.Errorf("invalid run ID in URL '%s': %w", input, err)
		}
		return RunURLInfo{
			RunID:    runID,
			Hostname: matches[1],
			Owner:    matches[2],
			Repo:     matches[3],
		}, nil
	}

	return RunURLInfo{}, fmt.Errorf("invalid run ID or URL '%s': must be a numeric run ID or a GitHub URL containing '/actions/runs/{run-id}' or '/runs/{run-id}'", input)
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
func AuditWorkflowRun(runInfo RunURLInfo, outputDir string, verbose bool, parse bool, jsonOutput bool) error {
	auditLog.Printf("Starting audit for workflow run: runID=%d, owner=%s, repo=%s", runInfo.RunID, runInfo.Owner, runInfo.Repo)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Auditing workflow run %d...", runInfo.RunID)))
	}

	runOutputDir := filepath.Join(outputDir, fmt.Sprintf("run-%d", runInfo.RunID))
	auditLog.Printf("Using output directory: %s", runOutputDir)

	// Check if we have locally cached artifacts first
	hasLocalCache := dirExists(runOutputDir) && !isDirEmpty(runOutputDir)

	// Try to get run metadata from GitHub API
	run, metadataErr := fetchWorkflowRunMetadata(runInfo, verbose)
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
					"Original error: %v", runInfo.RunID, runOutputDir, metadataErr)
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
		auditLog.Printf("Downloading artifacts for run %d", runInfo.RunID)
		err := downloadRunArtifacts(runInfo.RunID, runOutputDir, verbose)
		if err != nil {
			// Gracefully handle cases where the run legitimately has no artifacts
			if errors.Is(err, ErrNoArtifacts) {
				auditLog.Printf("No artifacts found for run %d", runInfo.RunID)
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
						"Original error: %v", runInfo.RunID, runOutputDir, err)
				}
			} else {
				return fmt.Errorf("failed to download artifacts: %w", err)
			}
		}
	}

	// If using local cache without metadata, create a minimal run structure
	if useLocalCache && run.DatabaseID == 0 {
		run = WorkflowRun{
			DatabaseID:   runInfo.RunID,
			WorkflowName: fmt.Sprintf("Workflow Run %d", runInfo.RunID),
			Status:       "unknown",
			LogsPath:     runOutputDir,
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Using locally cached artifacts without metadata. Some report details may be unavailable."))
	}

	// Extract metrics from logs
	metrics, err := extractLogMetrics(runOutputDir, verbose)
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
	run.ErrorCount = workflow.CountErrors(metrics.Errors)
	run.WarningCount = workflow.CountWarnings(metrics.Errors)
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

	// List all artifacts
	artifacts, err := listArtifacts(runOutputDir)
	if err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to list artifacts: %v", err)))
	}

	// Create processed run for report generation
	processedRun := ProcessedRun{
		Run:          run,
		MissingTools: missingTools,
		MCPFailures:  mcpFailures,
		JobDetails:   jobDetails,
	}

	// Build structured audit data
	auditData := buildAuditData(processedRun, metrics)

	// Render output based on format preference
	if jsonOutput {
		if err := renderJSON(auditData); err != nil {
			return fmt.Errorf("failed to render JSON output: %w", err)
		}
	} else {
		renderConsole(auditData, runOutputDir)
	}

	// Conditionally attempt to render agentic log (similar to `logs --parse`) if --parse flag is set
	// This creates a log.md file in the run directory for a rich, human-readable agent session summary.
	// We intentionally do not fail the audit on parse errors; they are reported as warnings.
	if parse {
		awInfoPath := filepath.Join(runOutputDir, "aw_info.json")
		if engine := extractEngineFromAwInfo(awInfoPath, verbose); engine != nil { // reuse existing helper in same package
			if err := parseAgentLog(runOutputDir, engine, verbose); err != nil {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse agent log for run %d: %v", runInfo.RunID, err)))
				}
			} else {
				// Always show success message for parsing, not just in verbose mode
				logMdPath := filepath.Join(runOutputDir, "log.md")
				if _, err := os.Stat(logMdPath); err == nil {
					fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Parsed log for run %d → %s", runInfo.RunID, logMdPath)))
				}
			}
		} else if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No engine detected (aw_info.json missing or invalid); skipping agent log rendering"))
		}
	}

	// Save run summary for caching future audit runs
	summary := &RunSummary{
		CLIVersion:     GetVersion(),
		RunID:          run.DatabaseID,
		ProcessedAt:    time.Now(),
		Run:            run,
		Metrics:        metrics,
		AccessAnalysis: accessAnalysis,
		MissingTools:   missingTools,
		MCPFailures:    mcpFailures,
		ArtifactsList:  artifacts,
		JobDetails:     jobDetails,
	}

	if err := saveRunSummary(runOutputDir, summary, verbose); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to save run summary: %v", err)))
		}
	}

	// Display logs location (only for console output)
	if !jsonOutput {
		absOutputDir, _ := filepath.Abs(runOutputDir)
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Audit complete. Logs saved to %s", absOutputDir)))
	}

	return nil
}

// fetchWorkflowRunMetadata fetches metadata for a single workflow run
func fetchWorkflowRunMetadata(runInfo RunURLInfo, verbose bool) (WorkflowRun, error) {
	// Build the API endpoint
	var endpoint string
	if runInfo.Owner != "" && runInfo.Repo != "" {
		// Use explicit owner/repo from the URL
		endpoint = fmt.Sprintf("repos/%s/%s/actions/runs/%d", runInfo.Owner, runInfo.Repo, runInfo.RunID)
	} else {
		// Fall back to {owner}/{repo} placeholders for context-based resolution
		endpoint = fmt.Sprintf("repos/{owner}/{repo}/actions/runs/%d", runInfo.RunID)
	}

	args := []string{"api"}

	// Add hostname flag if specified (for GitHub Enterprise)
	if runInfo.Hostname != "" && runInfo.Hostname != "github.com" {
		args = append(args, "--hostname", runInfo.Hostname)
	}

	args = append(args,
		endpoint,
		"--jq",
		"{databaseId: .id, number: .run_number, url: .html_url, status: .status, conclusion: .conclusion, workflowName: .name, createdAt: .created_at, startedAt: .run_started_at, updatedAt: .updated_at, event: .event, headBranch: .head_branch, headSha: .head_sha, displayTitle: .display_title}",
	)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Executing: gh %s", strings.Join(args, " "))))
	}

	cmd := exec.Command("gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(string(output)))
		}
		return WorkflowRun{}, fmt.Errorf("failed to fetch run metadata: %w", err)
	}

	var run WorkflowRun
	if err := json.Unmarshal(output, &run); err != nil {
		return WorkflowRun{}, fmt.Errorf("failed to parse run metadata: %w", err)
	}

	return run, nil
}

// generateAuditReport generates a concise markdown report for AI agent consumption
func generateAuditReport(processedRun ProcessedRun, metrics LogMetrics) string {
	run := processedRun.Run
	var report strings.Builder

	report.WriteString("# Workflow Run Audit Report\n\n")

	// Basic information
	report.WriteString("## Overview\n\n")
	report.WriteString(fmt.Sprintf("- **Run ID**: %d\n", run.DatabaseID))
	report.WriteString(fmt.Sprintf("- **Workflow**: %s\n", run.WorkflowName))
	report.WriteString(fmt.Sprintf("- **Status**: %s", run.Status))
	if run.Conclusion != "" && run.Status == "completed" {
		report.WriteString(fmt.Sprintf(" (%s)", run.Conclusion))
	}
	report.WriteString("\n")
	report.WriteString(fmt.Sprintf("- **Created**: %s\n", run.CreatedAt.Format(time.RFC3339)))
	if !run.StartedAt.IsZero() {
		report.WriteString(fmt.Sprintf("- **Started**: %s\n", run.StartedAt.Format(time.RFC3339)))
	}
	if !run.UpdatedAt.IsZero() {
		report.WriteString(fmt.Sprintf("- **Updated**: %s\n", run.UpdatedAt.Format(time.RFC3339)))
	}
	if run.Duration > 0 {
		report.WriteString(fmt.Sprintf("- **Duration**: %s\n", formatDuration(run.Duration)))
	}
	report.WriteString(fmt.Sprintf("- **Event**: %s\n", run.Event))
	report.WriteString(fmt.Sprintf("- **Branch**: %s\n", run.HeadBranch))
	report.WriteString(fmt.Sprintf("- **URL**: %s\n", run.URL))
	report.WriteString("\n")

	// Metrics
	report.WriteString("## Metrics\n\n")
	if run.TokenUsage > 0 {
		report.WriteString(fmt.Sprintf("- **Token Usage**: %s\n", formatNumber(run.TokenUsage)))
	}
	if run.EstimatedCost > 0 {
		report.WriteString(fmt.Sprintf("- **Estimated Cost**: $%.3f\n", run.EstimatedCost))
	}
	if run.Turns > 0 {
		report.WriteString(fmt.Sprintf("- **Turns**: %d\n", run.Turns))
	}
	report.WriteString(fmt.Sprintf("- **Errors**: %d\n", run.ErrorCount))
	report.WriteString(fmt.Sprintf("- **Warnings**: %d\n", run.WarningCount))
	report.WriteString("\n")

	// MCP Tool Usage
	if len(metrics.ToolCalls) > 0 {
		report.WriteString("## MCP Tool Usage\n\n")

		// Aggregate tool statistics
		toolStats := make(map[string]*workflow.ToolCallInfo)
		for _, toolCall := range metrics.ToolCalls {
			displayKey := workflow.PrettifyToolName(toolCall.Name)
			if existing, exists := toolStats[displayKey]; exists {
				existing.CallCount += toolCall.CallCount
				if toolCall.MaxInputSize > existing.MaxInputSize {
					existing.MaxInputSize = toolCall.MaxInputSize
				}
				if toolCall.MaxOutputSize > existing.MaxOutputSize {
					existing.MaxOutputSize = toolCall.MaxOutputSize
				}
				if toolCall.MaxDuration > existing.MaxDuration {
					existing.MaxDuration = toolCall.MaxDuration
				}
			} else {
				toolStats[displayKey] = &workflow.ToolCallInfo{
					Name:          displayKey,
					CallCount:     toolCall.CallCount,
					MaxInputSize:  toolCall.MaxInputSize,
					MaxOutputSize: toolCall.MaxOutputSize,
					MaxDuration:   toolCall.MaxDuration,
				}
			}
		}

		// Sort tools by call count
		var toolNames []string
		for name := range toolStats {
			toolNames = append(toolNames, name)
		}

		// Display top tools
		report.WriteString("| Tool | Calls | Max Input | Max Output | Max Duration |\n")
		report.WriteString("|------|-------|-----------|------------|-------------|\n")
		for _, name := range toolNames {
			tool := toolStats[name]
			inputStr := "N/A"
			if tool.MaxInputSize > 0 {
				inputStr = formatNumber(tool.MaxInputSize)
			}
			outputStr := "N/A"
			if tool.MaxOutputSize > 0 {
				outputStr = formatNumber(tool.MaxOutputSize)
			}
			durationStr := "N/A"
			if tool.MaxDuration > 0 {
				durationStr = formatDuration(tool.MaxDuration)
			}
			report.WriteString(fmt.Sprintf("| %s | %d | %s | %s | %s |\n",
				name, tool.CallCount, inputStr, outputStr, durationStr))
		}
		report.WriteString("\n")
	}

	// MCP Failures
	if len(processedRun.MCPFailures) > 0 {
		report.WriteString("## MCP Server Failures\n\n")
		for _, failure := range processedRun.MCPFailures {
			report.WriteString(fmt.Sprintf("- **%s**: %s\n", failure.ServerName, failure.Status))
		}
		report.WriteString("\n")
	}

	// Missing Tools
	if len(processedRun.MissingTools) > 0 {
		report.WriteString("## Missing Tools\n\n")
		for _, tool := range processedRun.MissingTools {
			report.WriteString(fmt.Sprintf("### %s\n\n", tool.Tool))
			report.WriteString(fmt.Sprintf("- **Reason**: %s\n", tool.Reason))
			if tool.Alternatives != "" {
				report.WriteString(fmt.Sprintf("- **Alternatives**: %s\n", tool.Alternatives))
			}
			if tool.Timestamp != "" {
				report.WriteString(fmt.Sprintf("- **Timestamp**: %s\n", tool.Timestamp))
			}
			report.WriteString("\n")
		}
	}

	// Error Summary
	if run.ErrorCount > 0 || run.WarningCount > 0 {
		report.WriteString("## Issue Summary\n\n")
		if run.ErrorCount > 0 {
			report.WriteString(fmt.Sprintf("This run encountered **%d error(s)**. ", run.ErrorCount))
		}
		if run.WarningCount > 0 {
			report.WriteString(fmt.Sprintf("This run had **%d warning(s)**. ", run.WarningCount))
		}
		report.WriteString("\n\n")

		// Display individual errors and warnings using compiler error format
		if len(metrics.Errors) > 0 {
			report.WriteString("### Errors and Warnings\n\n")
			report.WriteString("```\n")
			for _, logErr := range metrics.Errors {
				// Create a CompilerError for formatting
				compilerErr := console.CompilerError{
					Position: console.ErrorPosition{
						File:   logErr.File,
						Line:   logErr.Line,
						Column: 1, // Default to column 1 for log errors
					},
					Type:    logErr.Type,
					Message: logErr.Message,
				}
				// Format the error using console.FormatError and add to report
				formattedErr := console.FormatError(compilerErr)
				report.WriteString(formattedErr)
			}
			report.WriteString("```\n\n")
		}
	}

	// Artifacts
	report.WriteString("## Available Artifacts\n\n")
	report.WriteString(fmt.Sprintf("Logs and artifacts are available at: `%s`\n\n", run.LogsPath))

	// Check for specific artifacts
	artifacts := []string{}
	if _, err := os.Stat(filepath.Join(run.LogsPath, "aw_info.json")); err == nil {
		artifacts = append(artifacts, "aw_info.json (engine configuration)")
	}
	if _, err := os.Stat(filepath.Join(run.LogsPath, "safe_output.jsonl")); err == nil {
		artifacts = append(artifacts, "safe_output.jsonl (safe outputs)")
	}
	if _, err := os.Stat(filepath.Join(run.LogsPath, "aw.patch")); err == nil {
		artifacts = append(artifacts, "aw.patch (code changes)")
	}
	if _, err := os.Stat(filepath.Join(run.LogsPath, constants.AgentOutputArtifactName)); err == nil {
		artifacts = append(artifacts, "agent_output.json (validated safe outputs)")
	}

	if len(artifacts) > 0 {
		for _, artifact := range artifacts {
			report.WriteString(fmt.Sprintf("- %s\n", artifact))
		}
		report.WriteString("\n")
	}

	// Enumerate all files/directories present (top-level) to show full artifact set
	entries, err := os.ReadDir(run.LogsPath)
	if err == nil && len(entries) > 0 {
		report.WriteString("### All Downloaded Entries (top-level)\n\n")
		maxList := 100
		shown := 0
		for _, e := range entries {
			if shown >= maxList {
				break
			}
			name := e.Name()
			full := filepath.Join(run.LogsPath, name)
			var sizeInfo string
			if info, statErr := os.Stat(full); statErr == nil && !info.IsDir() {
				sizeInfo = fmt.Sprintf(" (%d bytes)", info.Size())
			}
			if e.IsDir() {
				report.WriteString(fmt.Sprintf("- %s/\n", name))
			} else {
				report.WriteString(fmt.Sprintf("- %s%s\n", name, sizeInfo))
			}
			shown++
		}
		if len(entries) > maxList {
			report.WriteString(fmt.Sprintf("- ... %d more entries omitted\n", len(entries)-maxList))
		}
		report.WriteString("\n")
	} else if err == nil && len(entries) == 0 {
		report.WriteString("(No artifact or log files were present for this run)\n\n")
	}

	return report.String()
}
