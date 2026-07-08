// This file provides command-line interface functionality for gh-aw.
// This file (logs_orchestrator_filters.go) contains run-filter helpers for the
// logs orchestrator: deciding whether a downloaded run should be included in
// results and constructing a ProcessedRun from a DownloadResult.

package cli

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/github/gh-aw/pkg/console"
)

// runFilterOpts bundles the filter flags passed to applyRunFilters.
type runFilterOpts struct {
	engine            string
	noStaged          bool
	firewallOnly      bool
	noFirewall        bool
	safeOutputType    string
	filteredIntegrity bool
}

var fetchJobStatusesForProcessedRun = fetchJobStatuses

// matchEngineFilter checks whether the run recorded in awInfo matches the
// requested engine filter string.  It returns (matches, detectedEngineID).
// detectedEngineID is "" when awInfo is unavailable or carries no engine_id.
func matchEngineFilter(awInfo *AwInfo, awInfoErr error, filterEngine string) (bool, string) {
	if awInfoErr != nil || awInfo == nil || awInfo.EngineID == "" {
		return false, ""
	}
	return awInfo.EngineID == filterEngine, awInfo.EngineID
}

// applyRunFilters applies all configured run filters to a DownloadResult.
// It parses aw_info.json once (lazily) when any filter that needs it is active.
// Returns true when the run should be skipped / excluded from results.
func applyRunFilters(result DownloadResult, opts runFilterOpts, verbose bool) bool {
	// Parse aw_info.json once for all filters that need it (optimization).
	var awInfo *AwInfo
	var awInfoErr error
	if opts.engine != "" || opts.noStaged || opts.firewallOnly || opts.noFirewall {
		awInfoPath := filepath.Join(result.LogsPath, "aw_info.json")
		awInfo, awInfoErr = parseAwInfo(awInfoPath, verbose)
	}

	// Apply engine filtering if specified.
	if opts.engine != "" {
		engineMatches, detectedEngineID := matchEngineFilter(awInfo, awInfoErr, opts.engine)
		if !engineMatches {
			if detectedEngineID == "" {
				detectedEngineID = "unknown"
			}
			logsOrchestratorLog.Printf("Skipping run %d: engine filter=%s, detected=%s", result.Run.DatabaseID, opts.engine, detectedEngineID)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: engine '%s' does not match filter '%s'", result.Run.DatabaseID, detectedEngineID, opts.engine)))
			}
			return true
		}
	}

	// Apply staged filtering if --no-staged flag is specified.
	if opts.noStaged {
		var isStaged bool
		if awInfoErr == nil && awInfo != nil {
			isStaged = awInfo.Staged
		}
		if isStaged {
			logsOrchestratorLog.Printf("Skipping run %d: staged workflow filtered by --no-staged", result.Run.DatabaseID)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: workflow is staged (filtered out by --no-staged)", result.Run.DatabaseID)))
			}
			return true
		}
	}

	// Apply firewall filtering if --firewall or --no-firewall flag is specified.
	if opts.firewallOnly || opts.noFirewall {
		var hasFirewall bool
		if awInfoErr == nil && awInfo != nil {
			// Firewall is enabled if steps.firewall is non-empty (e.g. "squid").
			hasFirewall = awInfo.Steps.Firewall != ""
		}
		if opts.firewallOnly && !hasFirewall {
			logsOrchestratorLog.Printf("Skipping run %d: no firewall detected, filtered by --firewall", result.Run.DatabaseID)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: workflow does not use firewall (filtered by --firewall)", result.Run.DatabaseID)))
			}
			return true
		}
		if opts.noFirewall && hasFirewall {
			logsOrchestratorLog.Printf("Skipping run %d: firewall detected, filtered by --no-firewall", result.Run.DatabaseID)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: workflow uses firewall (filtered by --no-firewall)", result.Run.DatabaseID)))
			}
			return true
		}
	}

	// Apply safe output type filtering if --safe-output flag is specified.
	if opts.safeOutputType != "" {
		hasSafeOutputType, checkErr := runContainsSafeOutputType(result.LogsPath, opts.safeOutputType, verbose)
		if checkErr != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to check safe output type for run %d: %v", result.Run.DatabaseID, checkErr)))
		}
		if !hasSafeOutputType {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: no '%s' safe output messages found", result.Run.DatabaseID, opts.safeOutputType)))
			}
			return true
		}
	}

	// Apply filtered-integrity filtering if --filtered-integrity flag is specified.
	if opts.filteredIntegrity {
		hasFiltered, checkErr := runHasDifcFilteredItems(result.LogsPath, verbose)
		if checkErr != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to check DIFC filtered items for run %d: %v", result.Run.DatabaseID, checkErr)))
			return true
		}
		if !hasFiltered {
			logsOrchestratorLog.Printf("Skipping run %d: no DIFC filtered items found", result.Run.DatabaseID)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Skipping run %d: no DIFC integrity-filtered items found in gateway logs", result.Run.DatabaseID)))
			}
			return true
		}
	}

	return false
}

// buildProcessedRun constructs a ProcessedRun from a DownloadResult, computing
// duration, action minutes, effective tokens, and job-failure counts.
func buildProcessedRun(result DownloadResult, verbose, logFailedJobs bool) ProcessedRun {
	run := result.Run
	run.TokenUsage = result.Metrics.TokenUsage
	applyMetricsTurnsToRun(&run, result.Metrics)
	run.AvgTimeBetweenTurns = result.Metrics.AvgTimeBetweenTurns
	run.ErrorCount = 0
	run.WarningCount = 0
	run.LogsPath = result.LogsPath

	// Propagate effective tokens from cached firewall proxy summary when available.
	if result.TokenUsage != nil && result.TokenUsage.TotalEffectiveTokens > 0 {
		run.EffectiveTokens = result.TokenUsage.TotalEffectiveTokens
	}

	// Add failed jobs to error count.
	if failedJobCount, err := fetchJobStatusesForProcessedRun(run.DatabaseID, verbose); err == nil {
		run.ErrorCount += failedJobCount
		if verbose && logFailedJobs && failedJobCount > 0 {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Added %d failed jobs to error count for run %d", failedJobCount, run.DatabaseID)))
		}
	}

	// Always use GitHub API timestamps for duration calculation.
	// GitHub Actions bills per minute, rounded up per job.
	if !run.StartedAt.IsZero() && !run.UpdatedAt.IsZero() {
		run.Duration = run.UpdatedAt.Sub(run.StartedAt)
		run.ActionMinutes = math.Ceil(run.Duration.Minutes())
	}

	return ProcessedRun{
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
}
