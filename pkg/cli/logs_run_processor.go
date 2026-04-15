// This file provides command-line interface functionality for gh-aw.
// This file (logs_run_processor.go) contains the concurrent run artifact
// download and run filtering logic.
//
// Key responsibilities:
//   - Concurrent downloading of artifacts from multiple runs
//   - Filtering runs by safe output type, DIFC filtered items, etc.

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/sourcegraph/conc/pool"
)

// downloadRunArtifactsConcurrent downloads artifacts for multiple workflow runs concurrently
func downloadRunArtifactsConcurrent(ctx context.Context, runs []WorkflowRun, outputDir string, verbose bool, maxRuns int, repoOverride string, artifactFilter []string) []DownloadResult {
	logsOrchestratorLog.Printf("Starting concurrent artifact download: runs=%d, outputDir=%s, maxRuns=%d", len(runs), outputDir, maxRuns)
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

	// Create progress bar for tracking run processing (only in non-verbose, non-CI mode)
	// In CI environments \r is treated as a newline, producing excessive output for each update.
	var progressBar *console.ProgressBar
	if !verbose && !IsRunningInCI() {
		progressBar = console.NewProgressBar(int64(totalRuns))
		fmt.Fprintf(os.Stderr, "Processing runs: %s\r", progressBar.Update(0))
	}

	// Use atomic counter for thread-safe progress tracking
	var completedCount int64

	// Get configured max concurrent downloads (default or from environment variable)
	maxConcurrent := getMaxConcurrentDownloads()

	// Parse repoOverride into owner/repo once for cross-repo artifact download
	var dlOwner, dlRepo string
	if repoOverride != "" {
		parts := strings.SplitN(repoOverride, "/", 2)
		if len(parts) == 2 {
			dlOwner = parts[0]
			dlRepo = parts[1]
		}
	}

	// Configure concurrent download pool with bounded parallelism and context cancellation.
	// The conc pool automatically handles panic recovery and prevents goroutine leaks.
	// WithContext enables graceful cancellation via Ctrl+C.
	p := pool.NewWithResults[DownloadResult]().
		WithContext(ctx).
		WithMaxGoroutines(maxConcurrent)

	// Each download task runs concurrently with context awareness.
	// Context cancellation (e.g., via Ctrl+C) will stop all in-flight downloads gracefully.
	// Panics are automatically recovered by the pool and re-raised with full stack traces
	// after all tasks complete. This ensures one failing download doesn't break others.
	for _, run := range actualRuns {
		p.Go(func(ctx context.Context) (DownloadResult, error) {
			// Check for context cancellation before starting download
			select {
			case <-ctx.Done():
				return DownloadResult{
					Run:     run,
					Skipped: true,
					Error:   ctx.Err(),
				}, nil
			default:
			}
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
					AwContext:               summary.AwContext,
					TaskDomain:              summary.TaskDomain,
					BehaviorFingerprint:     summary.BehaviorFingerprint,
					AgenticAssessments:      summary.AgenticAssessments,
					AccessAnalysis:          summary.AccessAnalysis,
					FirewallAnalysis:        summary.FirewallAnalysis,
					RedactedDomainsAnalysis: summary.RedactedDomainsAnalysis,
					MissingTools:            summary.MissingTools,
					MissingData:             summary.MissingData,
					Noops:                   summary.Noops,
					MCPFailures:             summary.MCPFailures,
					MCPToolUsage:            summary.MCPToolUsage,
					TokenUsage:              summary.TokenUsage,
					GitHubRateLimitUsage:    summary.GitHubRateLimitUsage,
					JobDetails:              summary.JobDetails,
					LogsPath:                runOutputDir,
					Cached:                  true, // Mark as cached
				}
				// Update progress counter
				completed := atomic.AddInt64(&completedCount, 1)
				if progressBar != nil {
					fmt.Fprintf(os.Stderr, "Processing runs: %s\r", progressBar.Update(completed))
				}
				return result, nil
			}

			// No cached summary or version mismatch - download and process
			err := downloadRunArtifacts(ctx, run.DatabaseID, runOutputDir, verbose, dlOwner, dlRepo, "", artifactFilter)

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

				// Update run with metrics so fingerprint computation uses the same data
				// as the audit tool, which also derives these fields from extracted log metrics.
				result.Run.TokenUsage = metrics.TokenUsage
				result.Run.EstimatedCost = metrics.EstimatedCost
				result.Run.Turns = metrics.Turns
				result.Run.AvgTimeBetweenTurns = metrics.AvgTimeBetweenTurns
				result.Run.LogsPath = runOutputDir

				// Calculate duration and billable minutes from GitHub API timestamps.
				// This mirrors the identical computation in audit.go so that
				// processedRun.Run.Duration is consistent across both tools.
				if !result.Run.StartedAt.IsZero() && !result.Run.UpdatedAt.IsZero() {
					result.Run.Duration = result.Run.UpdatedAt.Sub(result.Run.StartedAt)
					result.Run.ActionMinutes = math.Ceil(result.Run.Duration.Minutes())
				}

				// Analyze access logs if available
				accessAnalysis, accessErr := analyzeAccessLogs(runOutputDir, verbose)
				if accessErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze access logs for run %d: %v", run.DatabaseID, accessErr)))
					}
				}
				result.AccessAnalysis = accessAnalysis

				// Analyze firewall/gateway data only when the agent artifact was downloaded.
				// Firewall audit logs are now included in the unified agent artifact.
				// Skip silently when the artifact was intentionally excluded from the filter to
				// avoid spurious "not found" warnings in verbose mode.
				hasFirewallArtifact := artifactMatchesFilter(constants.AgentArtifactName, artifactFilter)

				// Analyze firewall logs if available
				var firewallAnalysis *FirewallAnalysis
				if hasFirewallArtifact {
					var firewallErr error
					firewallAnalysis, firewallErr = analyzeFirewallLogs(runOutputDir, verbose)
					if firewallErr != nil {
						if verbose {
							fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze firewall logs for run %d: %v", run.DatabaseID, firewallErr)))
						}
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

				// Extract missing data if available
				missingData, missingDataErr := extractMissingDataFromRun(runOutputDir, run, verbose)
				if missingDataErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract missing data for run %d: %v", run.DatabaseID, missingDataErr)))
					}
				}
				result.MissingData = missingData

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

				// Extract MCP tool usage data from gateway logs if available.
				// Gated on hasFirewallArtifact since gateway.jsonl lives in the agent artifact.
				var mcpToolUsage *MCPToolUsageData
				if hasFirewallArtifact {
					var mcpToolErr error
					mcpToolUsage, mcpToolErr = extractMCPToolUsageData(runOutputDir, verbose)
					if mcpToolErr != nil {
						if verbose {
							fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to extract MCP tool usage for run %d: %v", run.DatabaseID, mcpToolErr)))
						}
					}
				}
				result.MCPToolUsage = mcpToolUsage

				// Analyze token usage from firewall proxy logs.
				// Gated on hasFirewallArtifact since token-usage.jsonl lives in the agent artifact.
				var tokenUsage *TokenUsageSummary
				if hasFirewallArtifact {
					var tokenErr error
					tokenUsage, tokenErr = analyzeTokenUsage(runOutputDir, verbose)
					if tokenErr != nil {
						if verbose {
							fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze token usage for run %d: %v", run.DatabaseID, tokenErr)))
						}
					}
				}
				result.TokenUsage = tokenUsage

				// Propagate effective tokens from the firewall proxy summary when available
				if tokenUsage != nil && tokenUsage.TotalEffectiveTokens > 0 {
					result.Run.EffectiveTokens = tokenUsage.TotalEffectiveTokens
				}

				// Analyze GitHub API rate limit consumption from github_rate_limits.jsonl
				rateLimitUsage, rlErr := analyzeGitHubRateLimits(runOutputDir, verbose)
				if rlErr != nil {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to analyze GitHub rate limit usage for run %d: %v", run.DatabaseID, rlErr)))
					}
				}
				result.GitHubRateLimitUsage = rateLimitUsage

				// Count safe output items created in GitHub (from manifest artifact)
				result.Run.SafeItemsCount = len(extractCreatedItemsFromManifest(runOutputDir))

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

				processedRun := ProcessedRun{
					Run:                     result.Run,
					AccessAnalysis:          accessAnalysis,
					FirewallAnalysis:        firewallAnalysis,
					RedactedDomainsAnalysis: redactedDomainsAnalysis,
					MissingTools:            missingTools,
					MissingData:             missingData,
					Noops:                   noops,
					MCPFailures:             mcpFailures,
					MCPToolUsage:            mcpToolUsage,
					TokenUsage:              tokenUsage,
					GitHubRateLimitUsage:    rateLimitUsage,
					JobDetails:              jobDetails,
				}
				awContext, _, _, taskDomain, behaviorFingerprint, agenticAssessments := deriveRunAgenticAnalysis(processedRun, metrics)
				result.AwContext = awContext
				result.TaskDomain = taskDomain
				result.BehaviorFingerprint = behaviorFingerprint
				result.AgenticAssessments = agenticAssessments

				// Create and save run summary
				summary := &RunSummary{
					CLIVersion:              GetVersion(),
					RunID:                   run.DatabaseID,
					ProcessedAt:             time.Now(),
					Run:                     result.Run,
					Metrics:                 metrics,
					AwContext:               result.AwContext,
					TaskDomain:              result.TaskDomain,
					BehaviorFingerprint:     result.BehaviorFingerprint,
					AgenticAssessments:      result.AgenticAssessments,
					AccessAnalysis:          accessAnalysis,
					FirewallAnalysis:        firewallAnalysis,
					RedactedDomainsAnalysis: redactedDomainsAnalysis,
					MissingTools:            missingTools,
					MissingData:             missingData,
					Noops:                   noops,
					MCPFailures:             mcpFailures,
					MCPToolUsage:            mcpToolUsage,
					TokenUsage:              tokenUsage,
					GitHubRateLimitUsage:    rateLimitUsage,
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
			if progressBar != nil {
				fmt.Fprintf(os.Stderr, "Processing runs: %s\r", progressBar.Update(completed))
			}

			return result, nil
		})
	}

	// Wait blocks until all downloads complete, context is cancelled, or panic occurs.
	// With context support, the pool guarantees:
	// - All goroutines finish gracefully on cancellation (no leaks)
	// - Panics are propagated with stack traces
	// - Partial results are returned when context is cancelled
	// - Results are collected in submission order
	results, err := p.Wait()

	// Handle context cancellation
	if err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Download interrupted: %v", err)))
	}

	// Clear progress bar silently - detailed summary shown at the end
	if progressBar != nil {
		console.ClearLine() // Clear the line
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

// runContainsSafeOutputType checks if a run's agent_output.json contains a specific safe output type
func runContainsSafeOutputType(runDir string, safeOutputType string, verbose bool) (bool, error) {
	logsOrchestratorLog.Printf("Checking run for safe output type: dir=%s, type=%s", runDir, safeOutputType)
	// Normalize the type for comparison (convert dashes to underscores)
	normalizedType := stringutil.NormalizeSafeOutputIdentifier(safeOutputType)

	// Look for agent_output.json in the run directory
	agentOutputPath := filepath.Join(runDir, constants.AgentOutputFilename)

	// Support both new flattened form and old directory form
	if stat, err := os.Stat(agentOutputPath); err != nil || stat.IsDir() {
		// Try old structure
		oldPath := filepath.Join(runDir, constants.AgentOutputArtifactName, constants.AgentOutputArtifactName)
		if _, err := os.Stat(oldPath); err == nil {
			agentOutputPath = oldPath
		} else {
			// No agent_output.json found
			return false, nil
		}
	}

	// Read the file
	content, err := os.ReadFile(agentOutputPath)
	if err != nil {
		// File doesn't exist or can't be read
		return false, nil
	}

	// Parse the JSON
	var safeOutput struct {
		Items []json.RawMessage `json:"items"`
	}

	if err := json.Unmarshal(content, &safeOutput); err != nil {
		return false, fmt.Errorf("failed to parse agent_output.json: %w", err)
	}

	// Check each item for the specified type
	for _, itemRaw := range safeOutput.Items {
		var item struct {
			Type string `json:"type"`
		}

		if err := json.Unmarshal(itemRaw, &item); err != nil {
			continue // Skip malformed items
		}

		// Normalize the item type for comparison
		normalizedItemType := stringutil.NormalizeSafeOutputIdentifier(item.Type)

		if normalizedItemType == normalizedType {
			return true, nil
		}
	}

	return false, nil
}

// runHasDifcFilteredItems checks if a run's gateway logs contain any DIFC_FILTERED events.
// It parses the gateway logs (falling back to rpc-messages.jsonl when gateway.jsonl is absent)
// and returns true when at least one DIFC integrity- or secrecy-filtered event is present.
func runHasDifcFilteredItems(runDir string, verbose bool) (bool, error) {
	logsOrchestratorLog.Printf("Checking run for DIFC filtered items: dir=%s", runDir)

	gatewayMetrics, err := parseGatewayLogs(runDir, verbose)
	if err != nil {
		// No gateway log file present — not an error for workflows without MCP
		return false, nil
	}

	if gatewayMetrics == nil {
		return false, nil
	}

	return gatewayMetrics.TotalFiltered > 0, nil
}
