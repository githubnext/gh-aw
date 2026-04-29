// This file provides command-line interface functionality for gh-aw.
// This file (logs_orchestrator.go) contains the main orchestration logic for downloading
// and processing workflow logs from GitHub Actions.
//
// Key responsibilities:
//   - Coordinating the main download workflow (DownloadWorkflowLogs)
//   - Managing pagination and iteration through workflow runs
//   - Applying filters (engine, firewall, staged, etc.)
//   - Building and rendering output (console, JSON, tool graphs)

package cli

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/envutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
)

var logsOrchestratorLog = logger.New("cli:logs_orchestrator")

// It reads from the GH_AW_MAX_CONCURRENT_DOWNLOADS environment variable if set,
// validates the value is between 1 and 100, and falls back to the default if invalid.
func getMaxConcurrentDownloads() int {
	return envutil.GetIntFromEnv("GH_AW_MAX_CONCURRENT_DOWNLOADS", MaxConcurrentDownloads, 1, 100, logsOrchestratorLog)
}

// DownloadWorkflowLogs downloads and analyzes workflow logs with metrics
func DownloadWorkflowLogs(ctx context.Context, workflowName string, count int, startDate, endDate, outputDir, engine, ref string, beforeRunID, afterRunID int64, repoOverride string, verbose bool, toolGraph bool, noStaged bool, firewallOnly bool, noFirewall bool, parse bool, jsonOutput bool, timeout int, summaryFile string, safeOutputType string, filteredIntegrity bool, train bool, format string, artifactSets []string, after string) error {
	logsOrchestratorLog.Printf("Starting workflow log download: workflow=%s, count=%d, startDate=%s, endDate=%s, outputDir=%s, summaryFile=%s, safeOutputType=%s, filteredIntegrity=%v, train=%v, format=%s, artifactSets=%v, after=%s", workflowName, count, startDate, endDate, outputDir, summaryFile, safeOutputType, filteredIntegrity, train, format, artifactSets, after)

	// Validate and resolve artifact sets into a concrete filter (list of artifact base names).
	if err := ValidateArtifactSets(artifactSets); err != nil {
		return err
	}
	artifactFilter := ResolveArtifactFilter(artifactSets)
	if len(artifactFilter) > 0 {
		logsOrchestratorLog.Printf("Artifact filter active: %v", artifactFilter)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Artifact filter: downloading only "+strings.Join(artifactFilter, ", ")))
		}
	}

	// Ensure .github/aw/logs/.gitignore exists on every invocation
	if err := ensureLogsGitignore(); err != nil {
		// Log but don't fail - this is not critical for downloading logs
		logsOrchestratorLog.Printf("Failed to ensure logs .gitignore: %v", err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to ensure .github/aw/logs/.gitignore: %v", err)))
		}
	}

	// Check context cancellation at the start
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
	}

	// Clean up cached run folders older than the --after cutoff, if specified.
	// Runs after the context check so a cancelled context never triggers disk scanning.
	if after != "" {
		cutoff, parseErr := parseCleanupCutoff(after)
		if parseErr != nil {
			return parseErr
		}
		logsOrchestratorLog.Printf("Cleaning up run folders older than %s (cutoff: %s)", after, cutoff.Format(time.RFC3339))
		removed, cleanErr := cleanupOldRunFolders(outputDir, cutoff, verbose)
		if cleanErr != nil {
			// Non-fatal: log but continue with download
			logsOrchestratorLog.Printf("Failed to clean up old run folders: %v", cleanErr)
			if !jsonOutput {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to clean up old run folders: %v", cleanErr)))
			}
		} else if removed > 0 {
			if !jsonOutput {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Removed %d cached run folder(s) older than %s", removed, after)))
			}
		} else if verbose && !jsonOutput {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("No cached run folders older than %s found", after)))
		}
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Fetching workflow runs from GitHub Actions..."))
	}

	// Start timeout timer if specified
	var startTime time.Time
	var timeoutReached bool
	if timeout > 0 {
		startTime = time.Now()
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Timeout set to %d minutes", timeout)))
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
		// Check context cancellation
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
			return ctx.Err()
		default:
		}

		// Check timeout if specified
		if timeout > 0 {
			elapsed := time.Since(startTime).Seconds()
			if elapsed >= float64(timeout)*60 {
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

		// Query the GitHub API rate limit before each iteration (except the first)
		// and wait as needed.  This replaces the static cooldown sleep: the helper
		// always sleeps at least APICallCooldown but will also block until the
		// reset window when the remaining budget is nearly exhausted.
		if iteration > 0 {
			if rlErr := checkAndWaitForRateLimit(verbose); rlErr != nil {
				logsOrchestratorLog.Printf("Rate limit check failed (using static cooldown): %v", rlErr)
			}
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

		runs, totalFetched, err := listWorkflowRunsWithPagination(ListWorkflowRunsOptions{
			WorkflowName:   workflowName,
			Limit:          batchSize,
			StartDate:      startDate,
			EndDate:        endDate,
			BeforeDate:     beforeDate,
			Ref:            ref,
			BeforeRunID:    beforeRunID,
			AfterRunID:     afterRunID,
			RepoOverride:   repoOverride,
			ProcessedCount: len(processedRuns),
			TargetCount:    count,
			Verbose:        verbose,
		})
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
			chunkSize := min(max(remainingNeeded*3, remainingNeeded), len(runsRemaining))

			chunk := runsRemaining[:chunkSize]
			runsRemaining = runsRemaining[chunkSize:]

			downloadResults := downloadRunArtifactsConcurrent(ctx, chunk, outputDir, verbose, remainingNeeded, repoOverride, artifactFilter)

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
				if engine != "" || noStaged || firewallOnly || noFirewall {
					awInfo, awInfoErr = parseAwInfo(awInfoPath, verbose)
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
						logsOrchestratorLog.Printf("Skipping run %d: engine filter=%s, no match detected", result.Run.DatabaseID, engine)
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
						logsOrchestratorLog.Printf("Skipping run %d: staged workflow filtered by --no-staged", result.Run.DatabaseID)
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
						logsOrchestratorLog.Printf("Skipping run %d: no firewall detected, filtered by --firewall", result.Run.DatabaseID)
						if verbose {
							fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: workflow does not use firewall (filtered by --firewall)", result.Run.DatabaseID)))
						}
						continue
					}
					if noFirewall && hasFirewall {
						logsOrchestratorLog.Printf("Skipping run %d: firewall detected, filtered by --no-firewall", result.Run.DatabaseID)
						if verbose {
							fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: workflow uses firewall (filtered by --no-firewall)", result.Run.DatabaseID)))
						}
						continue
					}
				}

				// Apply safe output type filtering if --safe-output flag is specified
				if safeOutputType != "" {
					hasSafeOutputType, checkErr := runContainsSafeOutputType(result.LogsPath, safeOutputType, verbose)
					if checkErr != nil && verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to check safe output type for run %d: %v", result.Run.DatabaseID, checkErr)))
					}

					if !hasSafeOutputType {
						if verbose {
							fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: no '%s' safe output messages found", result.Run.DatabaseID, safeOutputType)))
						}
						continue
					}
				}

				// Apply filtered-integrity filtering if --filtered-integrity flag is specified
				if filteredIntegrity {
					hasFiltered, checkErr := runHasDifcFilteredItems(result.LogsPath, verbose)
					if checkErr != nil {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to check DIFC filtered items for run %d: %v", result.Run.DatabaseID, checkErr)))
						continue
					}

					if !hasFiltered {
						logsOrchestratorLog.Printf("Skipping run %d: no DIFC filtered items found", result.Run.DatabaseID)
						if verbose {
							fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: no DIFC integrity-filtered items found in gateway logs", result.Run.DatabaseID)))
						}
						continue
					}
				}

				// Update run with metrics and path
				run := result.Run
				run.TokenUsage = result.Metrics.TokenUsage
				run.EstimatedCost = result.Metrics.EstimatedCost
				run.Turns = result.Metrics.Turns
				run.AvgTimeBetweenTurns = result.Metrics.AvgTimeBetweenTurns
				run.ErrorCount = 0
				run.WarningCount = 0
				run.LogsPath = result.LogsPath

				// Propagate effective tokens from cached firewall proxy summary when available
				if result.TokenUsage != nil && result.TokenUsage.TotalEffectiveTokens > 0 {
					run.EffectiveTokens = result.TokenUsage.TotalEffectiveTokens
				}

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
					// Estimate billable Actions minutes from wall-clock time.
					// GitHub Actions bills per minute, rounded up per job.
					run.ActionMinutes = math.Ceil(run.Duration.Minutes())
				}

				processedRun := ProcessedRun{
					Run:                     run,
					AwContext:               result.AwContext,
					TaskDomain:              result.TaskDomain,
					BehaviorFingerprint:     result.BehaviorFingerprint,
					AgenticAssessments:      result.AgenticAssessments,
					AccessAnalysis:          result.AccessAnalysis,
					FirewallAnalysis:        result.FirewallAnalysis,
					RedactedDomainsAnalysis: result.RedactedDomainsAnalysis,
					MissingTools:            result.MissingTools,
					MissingData:             result.MissingData,
					Noops:                   result.Noops,
					MCPFailures:             result.MCPFailures,
					MCPToolUsage:            result.MCPToolUsage,
					TokenUsage:              result.TokenUsage,
					GitHubRateLimitUsage:    result.GitHubRateLimitUsage,
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
		// When JSON output is requested, output JSON first to stdout before any stderr messages
		// This prevents stderr messages from corrupting JSON when both streams are redirected together
		if jsonOutput {
			logsData := buildLogsData([]ProcessedRun{}, outputDir, nil)
			if err := renderLogsJSON(logsData); err != nil {
				return fmt.Errorf("failed to render JSON output: %w", err)
			}
		}
		// Now print warning messages to stderr after JSON output (if any) is complete
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

	return renderLogsOutput(processedRuns, outputDir, summaryFile, format, jsonOutput, toolGraph, train, continuation, verbose)
}

// renderLogsOutput finalizes processedRuns and renders them in the appropriate output
// format: JSON, console metrics table, or cross-run audit report (pretty/markdown).
// continuation is optional and only set when a timeout was reached during a paginated download.
func renderLogsOutput(processedRuns []ProcessedRun, outputDir, summaryFile, format string, jsonOutput, toolGraph, train bool, continuation *ContinuationData, verbose bool) error {
	// Update MissingToolCount, MissingDataCount, and NoopCount in runs
	for i := range processedRuns {
		processedRuns[i].Run.MissingToolCount = len(processedRuns[i].MissingTools)
		processedRuns[i].Run.MissingDataCount = len(processedRuns[i].MissingData)
		processedRuns[i].Run.NoopCount = len(processedRuns[i].Noops)
	}

	// Build structured logs data
	logsOrchestratorLog.Printf("Building logs data from %d processed runs (continuation=%t)", len(processedRuns), continuation != nil)
	logsData := buildLogsData(processedRuns, outputDir, continuation)

	// Write summary file if requested (default behavior unless disabled with empty string)
	if summaryFile != "" {
		summaryPath := filepath.Join(outputDir, summaryFile)
		if err := writeSummaryFile(summaryPath, logsData, verbose); err != nil {
			return fmt.Errorf("failed to write summary file: %w", err)
		}
	}

	// Train drain3 weights if requested.
	if train {
		if err := TrainDrain3Weights(processedRuns, outputDir, verbose); err != nil {
			return fmt.Errorf("log pattern training: %w", err)
		}
	}

	// Render output based on format preference.
	// When --format markdown or --format pretty is specified, generate a cross-run audit report
	// instead of the default metrics table.
	if format == "markdown" || format == "pretty" {
		inputs := make([]crossRunInput, 0, len(processedRuns))
		for _, pr := range processedRuns {
			inputs = append(inputs, crossRunInput{
				RunID:            pr.Run.DatabaseID,
				WorkflowName:     pr.Run.WorkflowName,
				Conclusion:       pr.Run.Conclusion,
				Duration:         pr.Run.Duration,
				FirewallAnalysis: pr.FirewallAnalysis,
				Metrics: LogMetrics{
					TokenUsage:    pr.Run.TokenUsage,
					EstimatedCost: pr.Run.EstimatedCost,
					Turns:         pr.Run.Turns,
				},
				MCPToolUsage: pr.MCPToolUsage,
				MCPFailures:  pr.MCPFailures,
				ErrorCount:   pr.Run.ErrorCount,
			})
		}
		report := buildCrossRunAuditReport(inputs)
		if jsonOutput {
			return renderCrossRunReportJSON(report)
		}
		if format == "pretty" {
			renderCrossRunReportPretty(report)
			return nil
		}
		renderCrossRunReportMarkdown(report)
		return nil
	}

	if jsonOutput {
		if err := renderLogsJSON(logsData); err != nil {
			return fmt.Errorf("failed to render JSON output: %w", err)
		}
	} else {
		renderLogsConsole(logsData)

		// Display aggregated gateway metrics if any runs have gateway.jsonl files
		displayAggregatedGatewayMetrics(processedRuns, outputDir, verbose)

		// Generate tool sequence graph if requested (console output only)
		if toolGraph {
			generateToolGraph(processedRuns, verbose)
		}
	}

	return nil
}

// DownloadWorkflowLogsFromStdin fetches and processes workflow run logs for runs
// provided as IDs or URLs, bypassing the GitHub API run-discovery step.
// This is used when the --stdin flag is passed to the logs command.
func DownloadWorkflowLogsFromStdin(ctx context.Context, runURLs []string, outputDir, engine, repoOverride string, verbose, toolGraph, noStaged, firewallOnly, noFirewall bool, parse, jsonOutput bool, timeout int, summaryFile, safeOutputType string, filteredIntegrity, train bool, format string, artifactSets []string) error {
	logsOrchestratorLog.Printf("Starting stdin log download: runs=%d, outputDir=%s", len(runURLs), outputDir)

	if err := ValidateArtifactSets(artifactSets); err != nil {
		return err
	}
	artifactFilter := ResolveArtifactFilter(artifactSets)
	if len(artifactFilter) > 0 {
		logsOrchestratorLog.Printf("Artifact filter active: %v", artifactFilter)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Artifact filter: downloading only "+strings.Join(artifactFilter, ", ")))
		}
	}

	if err := ensureLogsGitignore(); err != nil {
		logsOrchestratorLog.Printf("Failed to ensure logs .gitignore: %v", err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to ensure .github/aw/logs/.gitignore: %v", err)))
		}
	}

	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
	}

	if len(runURLs) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No run IDs or URLs provided on stdin"))
		return nil
	}

	// Parse owner/repo (and optional GHES host) from --repo override if provided.
	// Accepted formats: "owner/repo" or "HOST/owner/repo".
	var hostOverride, ownerOverride, repoNameOverride string
	if repoOverride != "" {
		parts := strings.SplitN(repoOverride, "/", 3)
		switch len(parts) {
		case 3: // HOST/owner/repo
			if parts[0] == "" || parts[1] == "" || parts[2] == "" {
				return fmt.Errorf("invalid repository format '%s': expected '[HOST/]owner/repo'", repoOverride)
			}
			hostOverride, ownerOverride, repoNameOverride = parts[0], parts[1], parts[2]
		case 2: // owner/repo
			if parts[0] == "" || parts[1] == "" {
				return fmt.Errorf("invalid repository format '%s': expected '[HOST/]owner/repo'", repoOverride)
			}
			ownerOverride, repoNameOverride = parts[0], parts[1]
		default:
			return fmt.Errorf("invalid repository format '%s': expected '[HOST/]owner/repo'", repoOverride)
		}
	}

	// Start timeout timer if specified
	var startTime time.Time
	if timeout > 0 {
		startTime = time.Now()
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Timeout set to %d minutes", timeout)))
		}
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Fetching metadata for %d runs from stdin...", len(runURLs))))
	}

	// Build WorkflowRun objects by fetching metadata for each provided URL
	var runs []WorkflowRun
	for _, rawURL := range runURLs {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
			return ctx.Err()
		default:
		}

		if timeout > 0 && time.Since(startTime).Seconds() >= float64(timeout)*60 {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Timeout reached before all run metadata could be fetched"))
			break
		}

		components, err := parser.ParseRunURLExtended(rawURL)
		if err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping invalid run %q: %v", rawURL, err)))
			continue
		}

		// Prefer owner/repo embedded in the URL; fall back to --repo override.
		// If neither source provides owner, the run cannot be fetched — return an
		// actionable error rather than silently continuing with a broken API call.
		owner := components.Owner
		repo := components.Repo
		host := components.Host
		if owner == "" {
			owner = ownerOverride
			repo = repoNameOverride
			if host == "" {
				host = hostOverride
			}
		}
		if owner == "" {
			return fmt.Errorf("run %q does not include repository information; pass --repo owner/repo or provide full run URLs", rawURL)
		}

		run, err := fetchWorkflowRunMetadata(ctx, components.Number, owner, repo, host, verbose)
		if err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping run %d: failed to fetch metadata: %v", components.Number, err)))
			continue
		}
		runs = append(runs, run)
	}

	if len(runs) == 0 {
		if jsonOutput {
			logsData := buildLogsData([]ProcessedRun{}, outputDir, nil)
			if err := renderLogsJSON(logsData); err != nil {
				return fmt.Errorf("failed to render JSON output: %w", err)
			}
		}
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No valid runs could be loaded from stdin"))
		return nil
	}

	// Download artifacts for all runs concurrently
	downloadResults := downloadRunArtifactsConcurrent(ctx, runs, outputDir, verbose, len(runs), repoOverride, artifactFilter)

	// Process download results applying the same filters as DownloadWorkflowLogs
	var processedRuns []ProcessedRun
	for _, result := range downloadResults {
		if result.Skipped {
			if verbose && result.Error != nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping run %d: %v", result.Run.DatabaseID, result.Error)))
			}
			continue
		}

		if result.Error != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to download artifacts for run %d: %v", result.Run.DatabaseID, result.Error)))
			continue
		}

		awInfoPath := filepath.Join(result.LogsPath, "aw_info.json")
		var awInfo *AwInfo
		var awInfoErr error
		if engine != "" || noStaged || firewallOnly || noFirewall {
			awInfo, awInfoErr = parseAwInfo(awInfoPath, verbose)
		}

		if engine != "" {
			detectedEngine := extractEngineFromAwInfo(awInfoPath, verbose)
			var engineMatches bool
			if detectedEngine != nil {
				registry := workflow.GetGlobalEngineRegistry()
				for _, supportedEngine := range constants.AgenticEngines {
					if testEngine, err := registry.GetEngine(supportedEngine); err == nil && testEngine == detectedEngine {
						engineMatches = (supportedEngine == engine)
						break
					}
				}
			}
			if !engineMatches {
				logsOrchestratorLog.Printf("Skipping run %d: engine filter=%s, no match detected", result.Run.DatabaseID, engine)
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: engine does not match filter '%s'", result.Run.DatabaseID, engine)))
				}
				continue
			}
		}

		if noStaged {
			var isStaged bool
			if awInfoErr == nil && awInfo != nil {
				isStaged = awInfo.Staged
			}
			if isStaged {
				logsOrchestratorLog.Printf("Skipping run %d: staged workflow filtered by --no-staged", result.Run.DatabaseID)
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: workflow is staged (filtered by --no-staged)", result.Run.DatabaseID)))
				}
				continue
			}
		}

		if firewallOnly || noFirewall {
			var hasFirewall bool
			if awInfoErr == nil && awInfo != nil {
				hasFirewall = awInfo.Steps.Firewall != ""
			}
			if firewallOnly && !hasFirewall {
				logsOrchestratorLog.Printf("Skipping run %d: no firewall detected, filtered by --firewall", result.Run.DatabaseID)
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: workflow does not use firewall (filtered by --firewall)", result.Run.DatabaseID)))
				}
				continue
			}
			if noFirewall && hasFirewall {
				logsOrchestratorLog.Printf("Skipping run %d: firewall detected, filtered by --no-firewall", result.Run.DatabaseID)
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: workflow uses firewall (filtered by --no-firewall)", result.Run.DatabaseID)))
				}
				continue
			}
		}

		if safeOutputType != "" {
			hasSafeOutputType, checkErr := runContainsSafeOutputType(result.LogsPath, safeOutputType, verbose)
			if checkErr != nil && verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to check safe output type for run %d: %v", result.Run.DatabaseID, checkErr)))
			}
			if !hasSafeOutputType {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: no '%s' safe output messages found", result.Run.DatabaseID, safeOutputType)))
				}
				continue
			}
		}

		if filteredIntegrity {
			hasFiltered, checkErr := runHasDifcFilteredItems(result.LogsPath, verbose)
			if checkErr != nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to check DIFC filtered items for run %d: %v", result.Run.DatabaseID, checkErr)))
				continue
			}
			if !hasFiltered {
				logsOrchestratorLog.Printf("Skipping run %d: no DIFC filtered items found", result.Run.DatabaseID)
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: no DIFC integrity-filtered items found in gateway logs", result.Run.DatabaseID)))
				}
				continue
			}
		}

		run := result.Run
		run.TokenUsage = result.Metrics.TokenUsage
		run.EstimatedCost = result.Metrics.EstimatedCost
		run.Turns = result.Metrics.Turns
		run.AvgTimeBetweenTurns = result.Metrics.AvgTimeBetweenTurns
		run.ErrorCount = 0
		run.WarningCount = 0
		run.LogsPath = result.LogsPath

		if result.TokenUsage != nil && result.TokenUsage.TotalEffectiveTokens > 0 {
			run.EffectiveTokens = result.TokenUsage.TotalEffectiveTokens
		}
		if failedJobCount, err := fetchJobStatuses(run.DatabaseID, verbose); err == nil {
			run.ErrorCount += failedJobCount
		}
		if !run.StartedAt.IsZero() && !run.UpdatedAt.IsZero() {
			run.Duration = run.UpdatedAt.Sub(run.StartedAt)
			run.ActionMinutes = math.Ceil(run.Duration.Minutes())
		}

		processedRun := ProcessedRun{
			Run:                     run,
			AwContext:               result.AwContext,
			TaskDomain:              result.TaskDomain,
			BehaviorFingerprint:     result.BehaviorFingerprint,
			AgenticAssessments:      result.AgenticAssessments,
			AccessAnalysis:          result.AccessAnalysis,
			FirewallAnalysis:        result.FirewallAnalysis,
			RedactedDomainsAnalysis: result.RedactedDomainsAnalysis,
			MissingTools:            result.MissingTools,
			MissingData:             result.MissingData,
			Noops:                   result.Noops,
			MCPFailures:             result.MCPFailures,
			MCPToolUsage:            result.MCPToolUsage,
			TokenUsage:              result.TokenUsage,
			GitHubRateLimitUsage:    result.GitHubRateLimitUsage,
			JobDetails:              result.JobDetails,
		}
		processedRuns = append(processedRuns, processedRun)

		if parse {
			detectedEngine := extractEngineFromAwInfo(awInfoPath, verbose)
			if err := parseAgentLog(result.LogsPath, detectedEngine, verbose); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse log for run %d: %v", run.DatabaseID, err)))
			} else {
				logMdPath := filepath.Join(result.LogsPath, "log.md")
				if _, err := os.Stat(logMdPath); err == nil {
					fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Parsed log for run %d → %s", run.DatabaseID, logMdPath)))
				}
			}
			if err := parseFirewallLogs(result.LogsPath, verbose); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse firewall logs for run %d: %v", run.DatabaseID, err)))
			} else {
				firewallMdPath := filepath.Join(result.LogsPath, "firewall.md")
				if _, err := os.Stat(firewallMdPath); err == nil {
					fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Parsed firewall logs for run %d → %s", run.DatabaseID, firewallMdPath)))
				}
			}
		}
	}

	if len(processedRuns) == 0 {
		if jsonOutput {
			logsData := buildLogsData([]ProcessedRun{}, outputDir, nil)
			if err := renderLogsJSON(logsData); err != nil {
				return fmt.Errorf("failed to render JSON output: %w", err)
			}
		}
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No workflow runs with artifacts found matching the specified criteria"))
		return nil
	}

	return renderLogsOutput(processedRuns, outputDir, summaryFile, format, jsonOutput, toolGraph, train, nil, verbose)
}
