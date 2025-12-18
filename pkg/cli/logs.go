package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/sourcegraph/conc/pool"
	"github.com/spf13/cobra"
)

var logsLog = logger.New("cli:logs")

// NewLogsCommand creates the logs command
func NewLogsCommand() *cobra.Command {
	logsCmd := &cobra.Command{
		Use:   "logs [workflow-id]",
		Short: "Download and analyze agentic workflow logs with aggregated metrics",
		Long: `Download workflow run logs and artifacts from GitHub Actions for agentic workflows.

This command fetches workflow runs, downloads their artifacts, and extracts them into
organized folders named by run ID. It also provides an overview table with aggregate
metrics including duration, token usage, and cost information.

Downloaded artifacts include:
- aw_info.json: Engine configuration and workflow metadata
- safe_output.jsonl: Agent's final output content (available when non-empty)
- agent_output/: Agent logs directory (if the workflow produced logs)
- agent-stdio.log: Agent standard output/error logs
- aw.patch: Git patch of changes made during execution
- workflow-logs/: GitHub Actions workflow run logs (job logs organized in subdirectory)

` + WorkflowIDExplanation + `

Examples:
  ` + constants.CLIExtensionPrefix + ` logs                           # Download logs for all workflows
  ` + constants.CLIExtensionPrefix + ` logs weekly-research           # Download logs for specific workflow
  ` + constants.CLIExtensionPrefix + ` logs weekly-research.md        # Download logs (alternative format)
  ` + constants.CLIExtensionPrefix + ` logs -c 10                     # Download last 10 matching runs
  ` + constants.CLIExtensionPrefix + ` logs --start-date 2024-01-01   # Download all runs after date
  ` + constants.CLIExtensionPrefix + ` logs --end-date 2024-01-31     # Download all runs before date
  ` + constants.CLIExtensionPrefix + ` logs --start-date -1w          # Download all runs from last week
  ` + constants.CLIExtensionPrefix + ` logs --start-date -1w -c 5     # Download all runs from last week, show up to 5
  ` + constants.CLIExtensionPrefix + ` logs --end-date -1d            # Download all runs until yesterday
  ` + constants.CLIExtensionPrefix + ` logs --start-date -1mo         # Download all runs from last month
  ` + constants.CLIExtensionPrefix + ` logs --engine claude           # Filter logs by claude engine
  ` + constants.CLIExtensionPrefix + ` logs --engine codex            # Filter logs by codex engine
  ` + constants.CLIExtensionPrefix + ` logs --engine copilot          # Filter logs by copilot engine
  ` + constants.CLIExtensionPrefix + ` logs --firewall                # Filter logs with firewall enabled
  ` + constants.CLIExtensionPrefix + ` logs --no-firewall             # Filter logs without firewall
  ` + constants.CLIExtensionPrefix + ` logs -o ./my-logs              # Custom output directory
  ` + constants.CLIExtensionPrefix + ` logs --ref main                # Filter logs by branch or tag
  ` + constants.CLIExtensionPrefix + ` logs --ref feature-xyz         # Filter logs by feature branch
  ` + constants.CLIExtensionPrefix + ` logs --after-run-id 1000       # Filter runs after run ID 1000
  ` + constants.CLIExtensionPrefix + ` logs --before-run-id 2000      # Filter runs before run ID 2000
  ` + constants.CLIExtensionPrefix + ` logs --after-run-id 1000 --before-run-id 2000  # Filter runs in range
  ` + constants.CLIExtensionPrefix + ` logs --tool-graph              # Generate Mermaid tool sequence graph
  ` + constants.CLIExtensionPrefix + ` logs --parse                   # Parse logs and generate Markdown reports
  ` + constants.CLIExtensionPrefix + ` logs --json                    # Output metrics in JSON format
  ` + constants.CLIExtensionPrefix + ` logs --parse --json            # Generate both Markdown and JSON
  ` + constants.CLIExtensionPrefix + ` logs weekly-research --repo owner/repo  # Download logs from specific repository`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var workflowName string
			if len(args) > 0 && args[0] != "" {
				// Convert workflow ID to GitHub Actions workflow name
				// First try to resolve as a workflow ID
				resolvedName, err := workflow.ResolveWorkflowName(args[0])
				if err != nil {
					// If that fails, check if it's already a GitHub Actions workflow name
					// by checking if any .lock.yml files have this as their name
					agenticWorkflowNames, nameErr := getAgenticWorkflowNames(false)
					if nameErr == nil && contains(agenticWorkflowNames, args[0]) {
						// It's already a valid GitHub Actions workflow name
						workflowName = args[0]
					} else {
						// Neither workflow ID nor valid GitHub Actions workflow name
						suggestions := []string{
							fmt.Sprintf("Run '%s status' to see all available workflows", constants.CLIExtensionPrefix),
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
				} else {
					workflowName = resolvedName
				}
			}

			count, _ := cmd.Flags().GetInt("count")
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
			campaignOnly, _ := cmd.Flags().GetBool("campaign")

			// Resolve relative dates to absolute dates for GitHub CLI
			now := time.Now()
			if startDate != "" {
				resolvedStartDate, err := workflow.ResolveRelativeDate(startDate, now)
				if err != nil {
					return fmt.Errorf("invalid start-date format '%s': %v", startDate, err)
				}
				startDate = resolvedStartDate
			}
			if endDate != "" {
				resolvedEndDate, err := workflow.ResolveRelativeDate(endDate, now)
				if err != nil {
					return fmt.Errorf("invalid end-date format '%s': %v", endDate, err)
				}
				endDate = resolvedEndDate
			}

			// Validate engine parameter using the engine registry
			if engine != "" {
				registry := workflow.GetGlobalEngineRegistry()
				if !registry.IsValidEngine(engine) {
					supportedEngines := registry.GetSupportedEngines()
					return fmt.Errorf("invalid engine value '%s'. Must be one of: %s", engine, strings.Join(supportedEngines, ", "))
				}
			}

			return DownloadWorkflowLogs(workflowName, count, startDate, endDate, outputDir, engine, ref, beforeRunID, afterRunID, repoOverride, verbose, toolGraph, noStaged, firewallOnly, noFirewall, parse, jsonOutput, timeout, campaignOnly)
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
	logsCmd.Flags().Bool("no-staged", false, "Filter out staged workflow runs (exclude runs with staged: true in aw_info.json)")
	logsCmd.Flags().Bool("firewall", false, "Filter to only runs with firewall enabled")
	logsCmd.Flags().Bool("no-firewall", false, "Filter to only runs without firewall enabled")
	logsCmd.Flags().Bool("campaign", false, "Filter to only campaign orchestrator workflows")
	logsCmd.Flags().Bool("parse", false, "Run JavaScript parsers on agent logs and firewall logs, writing Markdown to log.md and firewall.md")
	addJSONFlag(logsCmd)
	logsCmd.Flags().Int("timeout", 0, "Download timeout in seconds (0 = no timeout)")
	logsCmd.MarkFlagsMutuallyExclusive("firewall", "no-firewall")

	// Register completions for logs command
	logsCmd.ValidArgsFunction = CompleteWorkflowNames
	RegisterEngineFlagCompletion(logsCmd)
	RegisterDirFlagCompletion(logsCmd, "output")

	return logsCmd
}

// DownloadWorkflowLogs downloads and analyzes workflow logs with metrics
func DownloadWorkflowLogs(workflowName string, count int, startDate, endDate, outputDir, engine, ref string, beforeRunID, afterRunID int64, repoOverride string, verbose bool, toolGraph bool, noStaged bool, firewallOnly bool, noFirewall bool, parse bool, jsonOutput bool, timeout int, campaignOnly bool) error {
	logsLog.Printf("Starting workflow log download: workflow=%s, count=%d, startDate=%s, endDate=%s, outputDir=%s, campaignOnly=%v", workflowName, count, startDate, endDate, outputDir, campaignOnly)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Fetching workflow runs from GitHub Actions..."))
	}

	// Start timeout timer if specified
	var startTime time.Time
	var timeoutReached bool
	if timeout > 0 {
		startTime = time.Now()
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Timeout set to %d seconds", timeout)))
		}
	}

	var processedRuns []ProcessedRun
	var beforeDate string
	iteration := 0

	// Determine if we should fetch all runs (when date filters are specified) or limit by count
	// When date filters are specified, we fetch all runs within that range and apply count to final output
	// When no date filters, we fetch up to 'count' runs with artifacts (old behavior for backward compatibility)
	fetchAllInRange := startDate != "" || endDate != ""

	// Iterative algorithm: keep fetching runs until we have enough or exhaust available runs
	for iteration < MaxIterations {
		// Check timeout if specified
		if timeout > 0 {
			elapsed := time.Since(startTime).Seconds()
			if elapsed >= float64(timeout) {
				timeoutReached = true
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Timeout reached after %.1f seconds, stopping download", elapsed)))
				}
				break
			}
		}

		// Stop if we've collected enough processed runs
		if len(processedRuns) >= count {
			break
		}

		iteration++

		if verbose && iteration > 1 {
			if fetchAllInRange {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Iteration %d: Fetching more runs in date range...", iteration)))
			} else {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Iteration %d: Need %d more runs with artifacts, fetching more...", iteration, count-len(processedRuns))))
			}
		}

		// Fetch a batch of runs
		batchSize := BatchSize
		if workflowName == "" {
			// When searching for all agentic workflows, use a larger batch size
			// since there may be many CI runs interspersed with agentic runs
			batchSize = BatchSizeForAllWorkflows
		}

		// When not fetching all in range, optimize batch size based on how many we still need
		if !fetchAllInRange && count-len(processedRuns) < batchSize {
			// If we need fewer runs than the batch size, request exactly what we need
			// but add some buffer since many runs might not have artifacts
			needed := count - len(processedRuns)
			batchSize = needed * 3 // Request 3x what we need to account for runs without artifacts
			if workflowName == "" && batchSize < BatchSizeForAllWorkflows {
				// For all-workflows search, maintain a minimum batch size
				batchSize = BatchSizeForAllWorkflows
			}
			if batchSize > BatchSizeForAllWorkflows {
				batchSize = BatchSizeForAllWorkflows
			}
		}

		runs, totalFetched, err := listWorkflowRunsWithPagination(workflowName, batchSize, startDate, endDate, beforeDate, ref, beforeRunID, afterRunID, repoOverride, len(processedRuns), count, verbose)
		if err != nil {
			return err
		}

		if len(runs) == 0 {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No more workflow runs found, stopping iteration"))
			}
			break
		}

		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d workflow runs in batch %d", len(runs), iteration)))
		}

		// Process runs in chunks so cache hits can satisfy the count without
		// forcing us to scan the entire batch.
		batchProcessed := 0
		runsRemaining := runs
		for len(runsRemaining) > 0 && len(processedRuns) < count {
			remainingNeeded := count - len(processedRuns)
			if remainingNeeded <= 0 {
				break
			}

			// Process slightly more than we need to account for skips due to filters.
			chunkSize := remainingNeeded * 3
			if chunkSize < remainingNeeded {
				chunkSize = remainingNeeded
			}
			if chunkSize > len(runsRemaining) {
				chunkSize = len(runsRemaining)
			}

			chunk := runsRemaining[:chunkSize]
			runsRemaining = runsRemaining[chunkSize:]

			downloadResults := downloadRunArtifactsConcurrent(chunk, outputDir, verbose, remainingNeeded)

			for _, result := range downloadResults {
				if result.Skipped {
					if verbose {
						if result.Error != nil {
							fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping run %d: %v", result.Run.DatabaseID, result.Error)))
						}
					}
					continue
				}

				if result.Error != nil {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to download artifacts for run %d: %v", result.Run.DatabaseID, result.Error)))
					continue
				}

				// Parse aw_info.json once for all filters that need it (optimization)
				var awInfo *AwInfo
				var awInfoErr error
				awInfoPath := filepath.Join(result.LogsPath, "aw_info.json")

				// Only parse if we need it for any filter
				if engine != "" || noStaged || firewallOnly || noFirewall || campaignOnly {
					awInfo, awInfoErr = parseAwInfo(awInfoPath, verbose)
				}

				// Apply campaign filtering if --campaign flag is specified
				if campaignOnly {
					// Campaign orchestrator workflows end with .campaign.g.lock.yml
					isCampaign := strings.HasSuffix(result.Run.WorkflowName, " Campaign Orchestrator") ||
						strings.Contains(result.Run.WorkflowPath, ".campaign.g.lock.yml")

					if !isCampaign {
						if verbose {
							fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: not a campaign orchestrator workflow", result.Run.DatabaseID)))
						}
						continue
					}
				}

				// Apply engine filtering if specified
				if engine != "" {
					// Check if the run's engine matches the filter
					detectedEngine := extractEngineFromAwInfo(awInfoPath, verbose)

					var engineMatches bool
					if detectedEngine != nil {
						// Get the engine ID to compare with the filter
						registry := workflow.GetGlobalEngineRegistry()
						for _, supportedEngine := range constants.AgenticEngines {
							if testEngine, err := registry.GetEngine(supportedEngine); err == nil && testEngine == detectedEngine {
								engineMatches = (supportedEngine == engine)
								break
							}
						}
					}

					if !engineMatches {
						if verbose {
							engineName := "unknown"
							if detectedEngine != nil {
								// Try to get a readable name for the detected engine
								registry := workflow.GetGlobalEngineRegistry()
								for _, supportedEngine := range constants.AgenticEngines {
									if testEngine, err := registry.GetEngine(supportedEngine); err == nil && testEngine == detectedEngine {
										engineName = supportedEngine
										break
									}
								}
							}
							fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: engine '%s' does not match filter '%s'", result.Run.DatabaseID, engineName, engine)))
						}
						continue
					}
				}

				// Apply staged filtering if --no-staged flag is specified
				if noStaged {
					var isStaged bool
					if awInfoErr == nil && awInfo != nil {
						isStaged = awInfo.Staged
					}

					if isStaged {
						if verbose {
							fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: workflow is staged (filtered out by --no-staged)", result.Run.DatabaseID)))
						}
						continue
					}
				}

				// Apply firewall filtering if --firewall or --no-firewall flag is specified
				if firewallOnly || noFirewall {
					var hasFirewall bool
					if awInfoErr == nil && awInfo != nil {
						// Firewall is enabled if steps.firewall is non-empty (e.g., "squid")
						hasFirewall = awInfo.Steps.Firewall != ""
					}

					// Check if the run matches the filter
					if firewallOnly && !hasFirewall {
						if verbose {
							fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: workflow does not use firewall (filtered by --firewall)", result.Run.DatabaseID)))
						}
						continue
					}
					if noFirewall && hasFirewall {
						if verbose {
							fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: workflow uses firewall (filtered by --no-firewall)", result.Run.DatabaseID)))
						}
						continue
					}
				}

				// Update run with metrics and path
				run := result.Run
				run.TokenUsage = result.Metrics.TokenUsage
				run.EstimatedCost = result.Metrics.EstimatedCost
				run.Turns = result.Metrics.Turns
				run.ErrorCount = workflow.CountErrors(result.Metrics.Errors)
				run.WarningCount = workflow.CountWarnings(result.Metrics.Errors)
				run.LogsPath = result.LogsPath

				// Add failed jobs to error count
				if failedJobCount, err := fetchJobStatuses(run.DatabaseID, verbose); err == nil {
					run.ErrorCount += failedJobCount
					if verbose && failedJobCount > 0 {
						fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Added %d failed jobs to error count for run %d", failedJobCount, run.DatabaseID)))
					}
				}

				// Always use GitHub API timestamps for duration calculation
				if !run.StartedAt.IsZero() && !run.UpdatedAt.IsZero() {
					run.Duration = run.UpdatedAt.Sub(run.StartedAt)
				}

				processedRun := ProcessedRun{
					Run:                     run,
					AccessAnalysis:          result.AccessAnalysis,
					FirewallAnalysis:        result.FirewallAnalysis,
					RedactedDomainsAnalysis: result.RedactedDomainsAnalysis,
					MissingTools:            result.MissingTools,
					Noops:                   result.Noops,
					MCPFailures:             result.MCPFailures,
					JobDetails:              result.JobDetails,
				}
				processedRuns = append(processedRuns, processedRun)
				batchProcessed++

				// If --parse flag is set, parse the agent log and write to log.md
				if parse {
					// Get the engine from aw_info.json
					awInfoPath := filepath.Join(result.LogsPath, "aw_info.json")
					detectedEngine := extractEngineFromAwInfo(awInfoPath, verbose)

					if err := parseAgentLog(result.LogsPath, detectedEngine, verbose); err != nil {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse log for run %d: %v", run.DatabaseID, err)))
					} else {
						// Always show success message for parsing, not just in verbose mode
						logMdPath := filepath.Join(result.LogsPath, "log.md")
						if _, err := os.Stat(logMdPath); err == nil {
							fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Parsed log for run %d → %s", run.DatabaseID, logMdPath)))
						}
					}

					// Also parse firewall logs if they exist
					if err := parseFirewallLogs(result.LogsPath, verbose); err != nil {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse firewall logs for run %d: %v", run.DatabaseID, err)))
					} else {
						// Show success message if firewall.md was created
						firewallMdPath := filepath.Join(result.LogsPath, "firewall.md")
						if _, err := os.Stat(firewallMdPath); err == nil {
							fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Parsed firewall logs for run %d → %s", run.DatabaseID, firewallMdPath)))
						}
					}
				}

				// Stop processing this batch once we've collected enough runs.
				if len(processedRuns) >= count {
					break
				}
			}
		}

		if verbose {
			if fetchAllInRange {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Processed %d runs with artifacts in batch %d (total: %d)", batchProcessed, iteration, len(processedRuns))))
			} else {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Processed %d runs with artifacts in batch %d (total: %d/%d)", batchProcessed, iteration, len(processedRuns), count)))
			}
		}

		// Prepare for next iteration: set beforeDate to the oldest processed run from this batch
		if len(runs) > 0 && len(runsRemaining) == 0 {
			oldestRun := runs[len(runs)-1] // runs are typically ordered by creation date descending
			beforeDate = oldestRun.CreatedAt.Format(time.RFC3339)
		}

		// If we got fewer runs than requested in this batch, we've likely hit the end
		// IMPORTANT: Use totalFetched (API response size before filtering) not len(runs) (after filtering)
		// to detect end. When workflowName is empty, runs are filtered to only agentic workflows,
		// so len(runs) may be much smaller than totalFetched even when more data is available from GitHub.
		// Example: API returns 250 total runs, but only 5 are agentic workflows after filtering.
		//   Old buggy logic: len(runs)=5 < batchSize=250, stop iteration (WRONG - misses more agentic workflows!)
		//   Fixed logic: totalFetched=250 < batchSize=250 is false, continue iteration (CORRECT)
		if totalFetched < batchSize {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Received fewer runs than requested, likely reached end of available runs"))
			}
			break
		}
	}

	// Check if we hit the maximum iterations limit
	if iteration >= MaxIterations {
		if fetchAllInRange {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Reached maximum iterations (%d), collected %d runs with artifacts", MaxIterations, len(processedRuns))))
		} else if len(processedRuns) < count {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Reached maximum iterations (%d), collected %d runs with artifacts out of %d requested", MaxIterations, len(processedRuns), count)))
		}
	}

	// Report if timeout was reached
	if timeoutReached && len(processedRuns) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Timeout reached, returning %d processed runs", len(processedRuns))))
	}

	if len(processedRuns) == 0 {
		if timeoutReached {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Timeout reached before any runs could be downloaded"))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No workflow runs with artifacts found matching the specified criteria"))
		}
		return nil
	}

	// Apply count limit to final results (truncate to count if we fetched more)
	if len(processedRuns) > count {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Limiting output to %d most recent runs (fetched %d total)", count, len(processedRuns))))
		}
		processedRuns = processedRuns[:count]
	}

	// Update MissingToolCount and NoopCount in runs
	for i := range processedRuns {
		processedRuns[i].Run.MissingToolCount = len(processedRuns[i].MissingTools)
		processedRuns[i].Run.NoopCount = len(processedRuns[i].Noops)
	}

	// Build continuation data if timeout was reached and there are processed runs
	var continuation *ContinuationData
	if timeoutReached && len(processedRuns) > 0 {
		// Get the oldest run ID from processed runs to use as before_run_id for continuation
		oldestRunID := processedRuns[len(processedRuns)-1].Run.DatabaseID

		continuation = &ContinuationData{
			Message:      "Timeout reached. Use these parameters to continue fetching more logs.",
			WorkflowName: workflowName,
			Count:        count,
			StartDate:    startDate,
			EndDate:      endDate,
			Engine:       engine,
			Branch:       ref,
			AfterRunID:   afterRunID,
			BeforeRunID:  oldestRunID, // Continue from where we left off
			Timeout:      timeout,
		}
	}

	// Build structured logs data
	logsData := buildLogsData(processedRuns, outputDir, continuation)

	// Render output based on format preference
	if jsonOutput {
		if err := renderLogsJSON(logsData); err != nil {
			return fmt.Errorf("failed to render JSON output: %w", err)
		}
	} else {
		renderLogsConsole(logsData)

		// Generate tool sequence graph if requested (console output only)
		if toolGraph {
			generateToolGraph(processedRuns, verbose)
		}
	}

	return nil
}

// downloadRunArtifactsConcurrent downloads artifacts for multiple workflow runs concurrently
func downloadRunArtifactsConcurrent(runs []WorkflowRun, outputDir string, verbose bool, maxRuns int) []DownloadResult {
	logsLog.Printf("Starting concurrent artifact download: runs=%d, outputDir=%s, maxRuns=%d", len(runs), outputDir, maxRuns)
	if len(runs) == 0 {
		return []DownloadResult{}
	}

	// Process all runs in the batch to account for caching and filtering
	// The maxRuns parameter indicates how many successful results we need, but we may need to
	// process more runs to account for:
	// 1. Cached runs that may fail filters (engine, firewall, etc.)
	// 2. Runs that may be skipped due to errors
	// 3. Runs without artifacts
	//
	// By processing all runs in the batch, we ensure that the count parameter correctly
	// reflects the number of matching logs (both downloaded and cached), not just attempts.
	actualRuns := runs

	totalRuns := len(actualRuns)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Processing %d runs in parallel...", totalRuns)))
	}

	// Create spinner for progress updates (only in non-verbose mode)
	var spinner *console.SpinnerWrapper
	if !verbose {
		spinner = console.NewSpinner(fmt.Sprintf("Downloading artifacts... (0/%d completed)", totalRuns))
		spinner.Start()
	}

	// Use atomic counter for thread-safe progress tracking
	var completedCount int64

	// Use conc pool for controlled concurrency with results
	p := pool.NewWithResults[DownloadResult]().WithMaxGoroutines(MaxConcurrentDownloads)

	// Process each run concurrently
	for _, run := range actualRuns {
		run := run // capture loop variable
		p.Go(func() DownloadResult {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Processing run %d (%s)...", run.DatabaseID, run.Status)))
			}

			// Download artifacts and logs for this run
			runOutputDir := filepath.Join(outputDir, fmt.Sprintf("run-%d", run.DatabaseID))

			// Try to load cached summary first
			if summary, ok := loadRunSummary(runOutputDir, verbose); ok {
				// Valid cached summary exists, use it directly
				result := DownloadResult{
					Run:                     summary.Run,
					Metrics:                 summary.Metrics,
					AccessAnalysis:          summary.AccessAnalysis,
					FirewallAnalysis:        summary.FirewallAnalysis,
					RedactedDomainsAnalysis: summary.RedactedDomainsAnalysis,
					MissingTools:            summary.MissingTools,
					Noops:                   summary.Noops,
					MCPFailures:             summary.MCPFailures,
					JobDetails:              summary.JobDetails,
					LogsPath:                runOutputDir,
					Cached:                  true, // Mark as cached
				}
				// Update progress counter
				completed := atomic.AddInt64(&completedCount, 1)
				if spinner != nil {
					spinner.UpdateMessage(fmt.Sprintf("Downloading artifacts... (%d/%d completed)", completed, totalRuns))
				}
				return result
			}

			// No cached summary or version mismatch - download and process
			err := downloadRunArtifacts(run.DatabaseID, runOutputDir, verbose)

			result := DownloadResult{
				Run:      run,
				LogsPath: runOutputDir,
			}

			if err != nil {
				// Check if this is a "no artifacts" case
				if errors.Is(err, ErrNoArtifacts) {
					// For runs with important conclusions (timed_out, failure, cancelled),
					// still process them even without artifacts to show the failure in reports
					if isFailureConclusion(run.Conclusion) {
						// Don't skip - we want these to appear in the report
						// Just use empty metrics
						result.Metrics = LogMetrics{}

						// Try to fetch job details to get error count
						if failedJobCount, jobErr := fetchJobStatuses(run.DatabaseID, verbose); jobErr == nil {
							run.ErrorCount = failedJobCount
						}
					} else {
						// For other runs (success, neutral, etc.) without artifacts, skip them
						result.Skipped = true
						result.Error = err
					}
				} else {
					result.Error = err
				}
			} else {
				// Extract metrics from logs
				metrics, metricsErr := extractLogMetrics(runOutputDir, verbose)
				if metricsErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract metrics for run %d: %v", run.DatabaseID, metricsErr)))
					}
					// Don't fail the whole download for metrics errors
					metrics = LogMetrics{}
				}
				result.Metrics = metrics

				// Analyze access logs if available
				accessAnalysis, accessErr := analyzeAccessLogs(runOutputDir, verbose)
				if accessErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze access logs for run %d: %v", run.DatabaseID, accessErr)))
					}
				}
				result.AccessAnalysis = accessAnalysis

				// Analyze firewall logs if available
				firewallAnalysis, firewallErr := analyzeFirewallLogs(runOutputDir, verbose)
				if firewallErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze firewall logs for run %d: %v", run.DatabaseID, firewallErr)))
					}
				}
				result.FirewallAnalysis = firewallAnalysis

				// Analyze redacted domains if available
				redactedDomainsAnalysis, redactedErr := analyzeRedactedDomains(runOutputDir, verbose)
				if redactedErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze redacted domains for run %d: %v", run.DatabaseID, redactedErr)))
					}
				}
				result.RedactedDomainsAnalysis = redactedDomainsAnalysis

				// Extract missing tools if available
				missingTools, missingErr := extractMissingToolsFromRun(runOutputDir, run, verbose)
				if missingErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract missing tools for run %d: %v", run.DatabaseID, missingErr)))
					}
				}
				result.MissingTools = missingTools

				// Extract noops if available
				noops, noopErr := extractNoopsFromRun(runOutputDir, run, verbose)
				if noopErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract noops for run %d: %v", run.DatabaseID, noopErr)))
					}
				}
				result.Noops = noops

				// Extract MCP failures if available
				mcpFailures, mcpErr := extractMCPFailuresFromRun(runOutputDir, run, verbose)
				if mcpErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract MCP failures for run %d: %v", run.DatabaseID, mcpErr)))
					}
				}
				result.MCPFailures = mcpFailures

				// Fetch job details for the summary
				jobDetails, jobErr := fetchJobDetails(run.DatabaseID, verbose)
				if jobErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to fetch job details for run %d: %v", run.DatabaseID, jobErr)))
					}
				}

				// List all artifacts
				artifacts, listErr := listArtifacts(runOutputDir)
				if listErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to list artifacts for run %d: %v", run.DatabaseID, listErr)))
					}
				}

				// Create and save run summary
				summary := &RunSummary{
					CLIVersion:              GetVersion(),
					RunID:                   run.DatabaseID,
					ProcessedAt:             time.Now(),
					Run:                     run,
					Metrics:                 metrics,
					AccessAnalysis:          accessAnalysis,
					FirewallAnalysis:        firewallAnalysis,
					RedactedDomainsAnalysis: redactedDomainsAnalysis,
					MissingTools:            missingTools,
					Noops:                   noops,
					MCPFailures:             mcpFailures,
					ArtifactsList:           artifacts,
					JobDetails:              jobDetails,
				}

				if saveErr := saveRunSummary(runOutputDir, summary, verbose); saveErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to save run summary for run %d: %v", run.DatabaseID, saveErr)))
					}
				}
			}

			// Update progress counter for completed downloads
			completed := atomic.AddInt64(&completedCount, 1)
			if spinner != nil {
				spinner.UpdateMessage(fmt.Sprintf("Downloading artifacts... (%d/%d completed)", completed, totalRuns))
			}

			return result
		})
	}

	// Wait for all downloads to complete and collect results
	results := p.Wait()

	// Stop spinner with final success message
	if spinner != nil {
		successCount := 0
		for _, result := range results {
			// Count as successful if: no error AND not skipped
			// This includes both newly downloaded and cached runs
			if result.Error == nil && !result.Skipped {
				successCount++
			}
		}
		spinner.StopWithMessage(fmt.Sprintf("✓ Downloaded artifacts (%d/%d successful)", successCount, totalRuns))
	}

	if verbose {
		successCount := 0
		for _, result := range results {
			if result.Error == nil && !result.Skipped {
				successCount++
			}
		}
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Completed parallel processing: %d successful, %d total", successCount, len(results))))
	}

	return results
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
