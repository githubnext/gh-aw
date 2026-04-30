// This file provides command-line interface functionality for gh-aw.
// This file (logs_command.go) contains the CLI command definition for the logs command.
//
// Key responsibilities:
//   - Defining the Cobra command structure and flags for gh aw logs
//   - Parsing command-line arguments and flags
//   - Validating inputs (workflow names, dates, engine parameters)
//   - Delegating execution to the orchestrator (DownloadWorkflowLogs)

package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

var logsCommandLog = logger.New("cli:logs_command")

// NewLogsCommand creates the logs command
func NewLogsCommand() *cobra.Command {
	logsCmd := &cobra.Command{
		Use:   "logs [workflow]",
		Short: "Download and analyze agentic workflow logs with aggregated metrics",
		Long: `Download workflow run logs and artifacts from GitHub Actions for agentic workflows.

This command fetches workflow runs, downloads their artifacts, and extracts them into
organized folders named by run ID. It also provides an overview table with aggregate
metrics including duration, token usage, and cost information.

Downloaded artifacts include:
- Workflow metadata: Engine configuration and run metadata
- safe_output.jsonl: Agent's final output content (available when non-empty)
- agent_output/: Agent logs directory (if the workflow produced logs)
- agent-stdio.log: Agent standard output/error logs
- aw.patch: Git patch of changes made during execution (legacy; see aw-{branch}.patch)
- aw-{branch}.patch: Git patch of changes for each branch (one file per PR/push)
- workflow-logs/: GitHub Actions workflow run logs (job logs organized in subdirectory)
- summary.json: Complete metrics and run data for all downloaded runs

` + WorkflowIDExplanation + `

Examples:
  # Basic usage
  ` + string(constants.CLIExtensionPrefix) + ` logs                           # Download logs for all workflows
  ` + string(constants.CLIExtensionPrefix) + ` logs weekly-research           # Download logs for specific workflow
  ` + string(constants.CLIExtensionPrefix) + ` logs weekly-research.md        # Download logs (alternative format)
  ` + string(constants.CLIExtensionPrefix) + ` logs -c 10                     # Download last 10 matching runs

  # Date filtering
  ` + string(constants.CLIExtensionPrefix) + ` logs --start-date 2024-01-01   # Download all runs after date
  ` + string(constants.CLIExtensionPrefix) + ` logs --end-date 2024-01-31     # Download all runs before date
  ` + string(constants.CLIExtensionPrefix) + ` logs --start-date -1w          # Download up to 10 runs from last week
  ` + string(constants.CLIExtensionPrefix) + ` logs --start-date -1w -c 5     # Download up to 5 runs from last week
  ` + string(constants.CLIExtensionPrefix) + ` logs --end-date -1d            # Download all runs until yesterday
  ` + string(constants.CLIExtensionPrefix) + ` logs --start-date -1mo         # Download all runs from last month

  # Content filtering
  ` + string(constants.CLIExtensionPrefix) + ` logs --engine claude           # Filter logs by claude engine
  ` + string(constants.CLIExtensionPrefix) + ` logs --engine codex            # Filter logs by codex engine
  ` + string(constants.CLIExtensionPrefix) + ` logs --engine copilot          # Filter logs by copilot engine
  ` + string(constants.CLIExtensionPrefix) + ` logs --firewall                # Filter logs with firewall enabled
  ` + string(constants.CLIExtensionPrefix) + ` logs --no-firewall             # Filter logs without firewall
  ` + string(constants.CLIExtensionPrefix) + ` logs --safe-output missing-tool     # Filter logs with missing-tool messages
  ` + string(constants.CLIExtensionPrefix) + ` logs --safe-output missing-data     # Filter logs with missing-data messages
  ` + string(constants.CLIExtensionPrefix) + ` logs --safe-output create-issue     # Filter logs with create-issue messages
  ` + string(constants.CLIExtensionPrefix) + ` logs --safe-output noop             # Filter logs with noop messages
  ` + string(constants.CLIExtensionPrefix) + ` logs --safe-output report-incomplete # Filter logs with report-incomplete messages
  ` + string(constants.CLIExtensionPrefix) + ` logs --ref main                # Filter logs by branch or tag
  ` + string(constants.CLIExtensionPrefix) + ` logs --ref feature-xyz         # Filter logs by feature branch
  ` + string(constants.CLIExtensionPrefix) + ` logs --filtered-integrity      # Filter logs containing items that were filtered by gateway integrity checks
  ` + string(constants.CLIExtensionPrefix) + ` logs --no-staged               # Exclude staged workflow runs from results

  # Run ID range filtering
  ` + string(constants.CLIExtensionPrefix) + ` logs --after-run-id 1000       # Filter runs after run ID 1000
  ` + string(constants.CLIExtensionPrefix) + ` logs --before-run-id 2000      # Filter runs before run ID 2000
  ` + string(constants.CLIExtensionPrefix) + ` logs --after-run-id 1000 --before-run-id 2000  # Filter runs in range

  # Output options
  ` + string(constants.CLIExtensionPrefix) + ` logs -o ./my-logs              # Custom output directory
  ` + string(constants.CLIExtensionPrefix) + ` logs --tool-graph              # Generate Mermaid tool sequence graph
  ` + string(constants.CLIExtensionPrefix) + ` logs --parse                   # Parse logs and generate Markdown reports
  ` + string(constants.CLIExtensionPrefix) + ` logs --json                    # Output metrics in JSON format
  ` + string(constants.CLIExtensionPrefix) + ` logs --parse --json            # Generate both Markdown and JSON
  ` + string(constants.CLIExtensionPrefix) + ` logs --format markdown         # Generate cross-run security audit report in Markdown
  ` + string(constants.CLIExtensionPrefix) + ` logs --format pretty           # Generate cross-run security audit report in console format
  ` + string(constants.CLIExtensionPrefix) + ` logs weekly-research --format markdown --last 10  # Cross-run report for last 10 runs
  ` + string(constants.CLIExtensionPrefix) + ` logs --train                   # Train log pattern weights from last 10 runs
  ` + string(constants.CLIExtensionPrefix) + ` logs my-workflow --train -c 50 # Train log pattern weights from up to 50 runs of a specific workflow

  # Cross-repository
  ` + string(constants.CLIExtensionPrefix) + ` logs weekly-research --repo owner/repo  # Download logs from specific repository

  # Cache maintenance
  ` + string(constants.CLIExtensionPrefix) + ` logs --after -1w                # Delete cached run folders older than 1 week
  ` + string(constants.CLIExtensionPrefix) + ` logs --after -30d               # Delete cached run folders older than 30 days
  ` + string(constants.CLIExtensionPrefix) + ` logs --after -1mo               # Delete cached run folders older than 1 month
  ` + string(constants.CLIExtensionPrefix) + ` logs --after 2024-01-01         # Delete cached run folders from before 2024-01-01`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logsCommandLog.Printf("Starting logs command: args=%d", len(args))

			stdin, _ := cmd.Flags().GetBool("stdin")

			// When --stdin is provided, read run IDs/URLs from stdin and bypass GitHub API discovery.
			if stdin {
				if len(args) > 0 {
					return errors.New(console.FormatErrorWithSuggestions(
						"positional arguments are not allowed with --stdin",
						[]string{"Remove the workflow name argument, or omit --stdin to use the normal discovery mode"},
					))
				}
				logsCommandLog.Printf("Reading run IDs from stdin")
				runURLs, err := readRunIDsFromStdin(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read run IDs from stdin: %w", err)
				}

				outputDir, _ := cmd.Flags().GetString("output")
				engine, _ := cmd.Flags().GetString("engine")
				repoOverride, _ := cmd.Flags().GetString("repo")
				verbose, _ := cmd.Flags().GetBool("verbose")
				toolGraph, _ := cmd.Flags().GetBool("tool-graph")
				noStaged, _ := cmd.Flags().GetBool("no-staged")
				firewallOnly, _ := cmd.Flags().GetBool("firewall")
				noFirewall, _ := cmd.Flags().GetBool("no-firewall")
				parse, _ := cmd.Flags().GetBool("parse")
				jsonOutput, _ := cmd.Flags().GetBool("json")
				timeout, _ := cmd.Flags().GetInt("timeout")
				summaryFile, _ := cmd.Flags().GetString("summary-file")
				safeOutputType, _ := cmd.Flags().GetString("safe-output")
				filteredIntegrity, _ := cmd.Flags().GetBool("filtered-integrity")
				train, _ := cmd.Flags().GetBool("train")
				format, _ := cmd.Flags().GetString("format")
				artifacts, _ := cmd.Flags().GetStringSlice("artifacts")

				if engine != "" {
					logsCommandLog.Printf("Validating engine parameter: %s", engine)
					registry := workflow.GetGlobalEngineRegistry()
					if !registry.IsValidEngine(engine) {
						supportedEngines := registry.GetSupportedEngines()
						return fmt.Errorf("invalid engine value '%s'. Must be one of: %s", engine, strings.Join(supportedEngines, ", "))
					}
				}

				return DownloadWorkflowLogsFromStdin(cmd.Context(), runURLs, outputDir, engine, repoOverride, verbose, toolGraph, noStaged, firewallOnly, noFirewall, parse, jsonOutput, timeout, summaryFile, safeOutputType, filteredIntegrity, train, format, artifacts)
			}

			var workflowName string
			if len(args) > 0 && args[0] != "" {
				logsCommandLog.Printf("Resolving workflow name from argument: %s", args[0])

				// Use flexible workflow name matching (workflow ID or display name)
				resolvedName, err := workflow.FindWorkflowName(args[0])
				if err != nil {
					// Workflow not found - provide suggestions
					suggestions := []string{
						fmt.Sprintf("Run '%s status' to see all available workflows", string(constants.CLIExtensionPrefix)),
						"Check for typos in the workflow name",
						"Use the workflow ID (e.g., 'test-claude') or GitHub Actions workflow name (e.g., 'Test Claude')",
					}

					// Add fuzzy match suggestions
					similarNames := suggestWorkflowNames(args[0])
					if len(similarNames) > 0 {
						suggestions = append([]string{fmt.Sprintf("Did you mean: %s?", strings.Join(similarNames, ", "))}, suggestions...)
					}

					return errors.New(console.FormatErrorWithSuggestions(
						fmt.Sprintf("workflow '%s' not found", args[0]),
						suggestions,
					))
				}
				workflowName = resolvedName
			}

			count, _ := cmd.Flags().GetInt("count")
			// --last is an alias for --count (for compatibility with users of `audit report --last`)
			if last, _ := cmd.Flags().GetInt("last"); last > 0 {
				count = last
			}
			startDate, _ := cmd.Flags().GetString("start-date")
			endDate, _ := cmd.Flags().GetString("end-date")
			outputDir, _ := cmd.Flags().GetString("output")
			engine, _ := cmd.Flags().GetString("engine")
			ref, _ := cmd.Flags().GetString("ref")
			beforeRunID, _ := cmd.Flags().GetInt64("before-run-id")
			afterRunID, _ := cmd.Flags().GetInt64("after-run-id")
			verbose, _ := cmd.Flags().GetBool("verbose")
			toolGraph, _ := cmd.Flags().GetBool("tool-graph")
			noStaged, _ := cmd.Flags().GetBool("no-staged")
			firewallOnly, _ := cmd.Flags().GetBool("firewall")
			noFirewall, _ := cmd.Flags().GetBool("no-firewall")
			parse, _ := cmd.Flags().GetBool("parse")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			timeout, _ := cmd.Flags().GetInt("timeout")
			repoOverride, _ := cmd.Flags().GetString("repo")
			summaryFile, _ := cmd.Flags().GetString("summary-file")
			safeOutputType, _ := cmd.Flags().GetString("safe-output")
			filteredIntegrity, _ := cmd.Flags().GetBool("filtered-integrity")
			train, _ := cmd.Flags().GetBool("train")
			format, _ := cmd.Flags().GetString("format")
			artifacts, _ := cmd.Flags().GetStringSlice("artifacts")
			after, _ := cmd.Flags().GetString("after")

			// Resolve relative dates to absolute dates for GitHub CLI
			now := time.Now()
			if startDate != "" {
				logsCommandLog.Printf("Resolving start date: %s", startDate)
				resolvedStartDate, err := workflow.ResolveRelativeDate(startDate, now)
				if err != nil {
					return fmt.Errorf("invalid start-date format '%s': %w", startDate, err)
				}
				startDate = resolvedStartDate
				logsCommandLog.Printf("Resolved start date to: %s", startDate)
			}
			if endDate != "" {
				logsCommandLog.Printf("Resolving end date: %s", endDate)
				resolvedEndDate, err := workflow.ResolveRelativeDate(endDate, now)
				if err != nil {
					return fmt.Errorf("invalid end-date format '%s': %w", endDate, err)
				}
				endDate = resolvedEndDate
				logsCommandLog.Printf("Resolved end date to: %s", endDate)
			}

			// Validate engine parameter using the engine registry
			if engine != "" {
				logsCommandLog.Printf("Validating engine parameter: %s", engine)
				registry := workflow.GetGlobalEngineRegistry()
				if !registry.IsValidEngine(engine) {
					supportedEngines := registry.GetSupportedEngines()
					return fmt.Errorf("invalid engine value '%s'. Must be one of: %s", engine, strings.Join(supportedEngines, ", "))
				}
			}

			logsCommandLog.Printf("Executing logs download: workflow=%s, count=%d, engine=%s, train=%v, after=%s", workflowName, count, engine, train, after)

			return DownloadWorkflowLogs(cmd.Context(), workflowName, count, startDate, endDate, outputDir, engine, ref, beforeRunID, afterRunID, repoOverride, verbose, toolGraph, noStaged, firewallOnly, noFirewall, parse, jsonOutput, timeout, summaryFile, safeOutputType, filteredIntegrity, train, format, artifacts, after)
		},
	}

	// Add flags to logs command
	logsCmd.Flags().IntP("count", "c", 10, "Maximum number of matching workflow runs to return (after applying filters)")
	logsCmd.Flags().String("start-date", "", "Filter runs created after this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)")
	logsCmd.Flags().String("end-date", "", "Filter runs created before this date (YYYY-MM-DD or delta like -1d, -1w, -1mo)")
	addOutputFlag(logsCmd, defaultLogsOutputDir)
	addEngineFilterFlag(logsCmd)
	logsCmd.Flags().String("ref", "", "Filter runs by branch or tag name (e.g., main, v1.0.0)")
	logsCmd.Flags().Int64("before-run-id", 0, "Filter runs with database ID before this value (exclusive)")
	logsCmd.Flags().Int64("after-run-id", 0, "Filter runs with database ID after this value (exclusive)")
	addRepoFlag(logsCmd)
	logsCmd.Flags().Bool("tool-graph", false, "Generate Mermaid tool sequence graph from agent logs")
	logsCmd.Flags().Bool("no-staged", false, "Exclude workflow runs that executed in staged mode (safe outputs previewed but not applied)")
	logsCmd.Flags().Bool("firewall", false, "Filter to only runs with firewall enabled")
	logsCmd.Flags().Bool("no-firewall", false, "Filter to only runs without firewall enabled")
	logsCmd.Flags().String("safe-output", "", "Filter to runs containing a specific safe output type (e.g., create-issue, missing-tool, missing-data, noop, report-incomplete)")
	logsCmd.Flags().Bool("filtered-integrity", false, "Filter to runs containing items that were filtered by gateway integrity checks")
	logsCmd.Flags().Bool("parse", false, "Run JavaScript parsers on agent logs and firewall logs, writing Markdown to log.md and firewall.md")
	addJSONFlag(logsCmd)
	logsCmd.Flags().Int("timeout", 0, "Download timeout in minutes (0 = no timeout)")
	logsCmd.Flags().String("summary-file", "summary.json", "Path to write the summary JSON file relative to output directory (use empty string to disable)")
	logsCmd.Flags().Bool("train", false, "Analyze log patterns across downloaded runs and save pattern weights to drain3_weights.json in the output directory")
	logsCmd.Flags().String("format", "", "Output format for cross-run audit report: pretty, markdown (generates security audit report instead of default metrics table)")
	logsCmd.Flags().Int("last", 0, "Alias for --count: number of recent runs to download")
	logsCmd.Flags().StringSlice("artifacts", nil, "Artifact sets to download (default: all). Valid sets: "+strings.Join(ValidArtifactSetNames(), ", "))
	logsCmd.Flags().String("after", "", "Remove locally cached run folders created before this date (cache cleanup). Use deltas like -1w or -1mo, or an absolute date YYYY-MM-DD. For example, --after -1w removes folders older than 1 week.")
	logsCmd.Flags().Bool("stdin", false, "Read workflow run IDs or URLs from stdin (one per line) instead of discovering runs via the GitHub API")
	logsCmd.MarkFlagsMutuallyExclusive("firewall", "no-firewall")

	// Register completions for logs command
	logsCmd.ValidArgsFunction = CompleteWorkflowNames
	RegisterEngineFlagCompletion(logsCmd)
	RegisterDirFlagCompletion(logsCmd, "output")

	return logsCmd
}

// flattenSingleFileArtifacts applies the artifact unfold rule to downloaded artifacts
// Unfold rule: If an artifact download folder contains a single file, move the file to root and delete the folder
// This simplifies artifact access by removing unnecessary nesting for single-file artifacts

// downloadWorkflowRunLogs downloads and unzips workflow run logs using GitHub API

// unzipFile extracts a zip file to a destination directory

// extractZipFile extracts a single file from a zip archive

// loadRunSummary attempts to load a run summary from disk
// Returns the summary and a boolean indicating if it was successfully loaded and is valid
// displayToolCallReport displays a table of tool usage statistics across all runs
// ExtractLogMetricsFromRun extracts log metrics from a processed run's log directory

// findAgentOutputFile searches for a file named agent_output.json within the logDir tree.
// Returns the first path found (depth-first) and a boolean indicating success.

// findAgentLogFile searches for agent logs within the logDir.
// It uses engine.GetLogFileForParsing() to determine which log file to use:
//   - If GetLogFileForParsing() returns a non-empty value that doesn't point to agent-stdio.log,
//     look for files in the "agent_output" artifact directory
//   - Otherwise, look for the "agent-stdio.log" artifact file
//
// Returns the first path found and a boolean indicating success.

// fileExists checks if a file exists

// copyFileSimple copies a file from src to dst using buffered IO.

// dirExists checks if a directory exists

// isDirEmpty checks if a directory is empty

// extractMissingToolsFromRun extracts missing tool reports from a workflow run's artifacts

// extractMCPFailuresFromRun extracts MCP server failure reports from a workflow run's logs

// extractMCPFailuresFromLogFile parses a single log file for MCP server failures

// MCPFailureSummary aggregates MCP server failures across runs
// displayMCPFailuresAnalysis displays a summary of MCP server failures across all runs
// parseAgentLog runs the JavaScript log parser on agent logs and writes markdown to log.md

// parseFirewallLogs runs the JavaScript firewall log parser and writes markdown to firewall.md
