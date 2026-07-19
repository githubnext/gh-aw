// This file provides command-line interface functionality for gh-aw.
// This file (logs_orchestrator_stdin.go) contains DownloadWorkflowLogsFromStdin,
// the stdin-driven run-discovery and processing path used when --stdin is passed
// to the logs command.

package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/parser"
)

// DownloadWorkflowLogsFromStdin fetches and processes workflow run logs for runs
// provided as IDs or URLs, bypassing the GitHub API run-discovery step.
// This is used when the --stdin flag is passed to the logs command.
func DownloadWorkflowLogsFromStdin(ctx context.Context, opts StdinLogsOptions) error {
	logsOrchestratorLog.Printf("Starting stdin log download: runs=%d, outputDir=%s", len(opts.RunURLs), opts.OutputDir)

	artifactFilter, stop, err := downloadWorkflowLogsFromStdinPrepare(ctx, opts)
	if err != nil || stop {
		return err
	}

	repoInfo, err := downloadWorkflowLogsFromStdinRepoInfo(opts.RepoOverride)
	if err != nil {
		return err
	}
	startTime := downloadWorkflowLogsFromStdinStartTime(opts)
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Fetching metadata for %d runs from stdin...", len(opts.RunURLs))))
	}

	runs, err := downloadWorkflowLogsFromStdinFetchRuns(ctx, opts, repoInfo, startTime)
	if err != nil {
		return err
	}
	if len(runs) == 0 {
		return downloadWorkflowLogsFromStdinNoRuns(opts, "No runs found. No valid runs could be loaded from the provided input.", "No valid runs could be loaded from stdin")
	}

	downloadResults := downloadRunArtifactsConcurrent(ctx, runs, runArtifactsConcurrentOptions{outputDir: opts.OutputDir, verbose: opts.Verbose, maxRuns: len(runs), repoOverride: opts.RepoOverride, artifactFilter: artifactFilter, evalsOnly: opts.EvalsOnly, artifactSets: opts.ArtifactSets})
	processedRuns := downloadWorkflowLogsFromStdinProcessResults(ctx, downloadResults, opts)
	if len(processedRuns) == 0 {
		return downloadWorkflowLogsFromStdinNoRuns(opts, "No runs found matching the specified criteria.", "No workflow runs with artifacts found matching the specified criteria")
	}

	return renderLogsOutput(processedRuns, renderLogsOutputOptions{
		outputDir:      opts.OutputDir,
		summaryFile:    opts.SummaryFile,
		format:         opts.Format,
		reportFile:     opts.ReportFile,
		jsonOutput:     opts.JSONOutput,
		toolGraph:      opts.ToolGraph,
		train:          opts.Train,
		verbose:        opts.Verbose,
		artifactFilter: artifactFilter,
	})
}

func downloadWorkflowLogsFromStdinPrepare(ctx context.Context, opts StdinLogsOptions) ([]string, bool, error) {
	if err := ValidateArtifactSets(opts.ArtifactSets); err != nil {
		return nil, false, err
	}
	artifactFilter := ResolveArtifactFilter(opts.ArtifactSets)
	if len(artifactFilter) > 0 {
		logsOrchestratorLog.Printf("Artifact filter active: %v", artifactFilter)
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Artifact filter: downloading only "+strings.Join(artifactFilter, ", ")))
		}
	}

	if err := ensureLogsGitignore(); err != nil {
		logsOrchestratorLog.Printf("Failed to ensure logs .gitignore: %v", err)
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to ensure .github/aw/logs/.gitignore: %v", err)))
		}
	}

	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return nil, false, ctx.Err()
	default:
	}

	if len(opts.RunURLs) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("No run IDs or URLs provided on stdin"))
		return artifactFilter, true, nil
	}

	return artifactFilter, false, nil
}

type downloadWorkflowLogsFromStdinRepoOverrides struct {
	host  string
	owner string
	repo  string
}

func downloadWorkflowLogsFromStdinRepoInfo(repoOverride string) (downloadWorkflowLogsFromStdinRepoOverrides, error) {
	// Parse owner/repo (and optional GHES host) from --repo override if provided.
	// Accepted formats: "owner/repo" or "HOST/owner/repo".
	var repoInfo downloadWorkflowLogsFromStdinRepoOverrides
	if repoOverride != "" {
		parts := strings.SplitN(repoOverride, "/", 3)
		switch len(parts) {
		case 3: // HOST/owner/repo
			if parts[0] == "" || parts[1] == "" || parts[2] == "" {
				return repoInfo, fmt.Errorf("invalid repository format '%s': expected '[HOST/]owner/repo'", repoOverride)
			}
			repoInfo.host, repoInfo.owner, repoInfo.repo = parts[0], parts[1], parts[2]
		case 2: // owner/repo
			if parts[0] == "" || parts[1] == "" {
				return repoInfo, fmt.Errorf("invalid repository format '%s': expected '[HOST/]owner/repo'", repoOverride)
			}
			repoInfo.owner, repoInfo.repo = parts[0], parts[1]
		default:
			return repoInfo, fmt.Errorf("invalid repository format '%s': expected '[HOST/]owner/repo'", repoOverride)
		}
	}
	return repoInfo, nil
}

func downloadWorkflowLogsFromStdinStartTime(opts StdinLogsOptions) time.Time {
	// Start timeout timer if specified.
	var startTime time.Time
	if opts.Timeout > 0 {
		startTime = time.Now()
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Timeout set to %d minutes", opts.Timeout)))
		}
	}
	return startTime
}

func downloadWorkflowLogsFromStdinFetchRuns(ctx context.Context, opts StdinLogsOptions, repoInfo downloadWorkflowLogsFromStdinRepoOverrides, startTime time.Time) ([]WorkflowRun, error) {
	// Build WorkflowRun objects by fetching metadata for each provided URL.
	var runs []WorkflowRun
	for _, rawURL := range opts.RunURLs {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
			return runs, ctx.Err()
		default:
		}

		if opts.Timeout > 0 && time.Since(startTime).Seconds() >= float64(opts.Timeout)*60 {
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
			owner = repoInfo.owner
			repo = repoInfo.repo
			if host == "" {
				host = repoInfo.host
			}
		}
		if owner == "" {
			return runs, fmt.Errorf("run %q does not include repository information; pass --repo owner/repo or provide full run URLs", rawURL)
		}

		run, err := fetchWorkflowRunMetadata(ctx, components.Number, owner, repo, host, opts.Verbose)
		if err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping run %d: failed to fetch metadata: %v", components.Number, err)))
			continue
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func downloadWorkflowLogsFromStdinProcessResults(ctx context.Context, downloadResults []DownloadResult, opts StdinLogsOptions) []ProcessedRun {
	filters := runFilterOpts{
		engine:            opts.Engine,
		noStaged:          opts.NoStaged,
		firewallOnly:      opts.FirewallOnly,
		noFirewall:        opts.NoFirewall,
		safeOutputType:    opts.SafeOutputType,
		filteredIntegrity: opts.FilteredIntegrity,
		evalsOnly:         opts.EvalsOnly,
	}

	// Process download results applying the same filters as DownloadWorkflowLogs.
	var processedRuns []ProcessedRun
	for _, result := range downloadResults {
		processedRun, ok := downloadWorkflowLogsFromStdinProcessResult(ctx, result, filters, opts)
		if !ok {
			continue
		}
		processedRuns = append(processedRuns, processedRun)
	}
	return processedRuns
}

func downloadWorkflowLogsFromStdinProcessResult(ctx context.Context, result DownloadResult, filters runFilterOpts, opts StdinLogsOptions) (ProcessedRun, bool) {
	if result.Skipped {
		if opts.Verbose && result.Error != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Skipping run %d: %v", result.Run.DatabaseID, result.Error)))
		}
		return ProcessedRun{}, false
	}

	if result.Error != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to download artifacts for run %d: %v", result.Run.DatabaseID, result.Error)))
		return ProcessedRun{}, false
	}

	if applyRunFilters(ctx, result, filters, opts.Verbose) {
		return ProcessedRun{}, false
	}

	processedRun := buildProcessedRun(result, opts.Verbose, false)
	if opts.Parse {
		downloadWorkflowLogsFromStdinParseRun(result, processedRun, opts.Verbose)
	}
	return processedRun, true
}

func downloadWorkflowLogsFromStdinParseRun(result DownloadResult, processedRun ProcessedRun, verbose bool) {
	awInfoPath := filepath.Join(result.LogsPath, "aw_info.json")
	detectedEngine := extractEngineFromAwInfo(awInfoPath, verbose)
	if err := parseAgentLog(result.LogsPath, detectedEngine, verbose); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse log for run %d: %v", processedRun.Run.DatabaseID, err)))
	} else {
		logMdPath := filepath.Join(result.LogsPath, "log.md")
		if fileutil.FileExists(logMdPath) {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Parsed log for run %d → %s", processedRun.Run.DatabaseID, logMdPath)))
		}
	}
	if err := parseFirewallLogs(result.LogsPath, verbose); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse firewall logs for run %d: %v", processedRun.Run.DatabaseID, err)))
	} else {
		firewallMdPath := filepath.Join(result.LogsPath, "firewall.md")
		if fileutil.FileExists(firewallMdPath) {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Parsed firewall logs for run %d → %s", processedRun.Run.DatabaseID, firewallMdPath)))
		}
	}
}

func downloadWorkflowLogsFromStdinNoRuns(opts StdinLogsOptions, jsonMessage, warningMessage string) error {
	if opts.JSONOutput {
		logsData := buildLogsData([]ProcessedRun{}, opts.OutputDir, nil)
		logsData.Message = jsonMessage
		if err := renderLogsJSON(logsData, opts.Verbose); err != nil {
			return fmt.Errorf("failed to render JSON output: %w", err)
		}
	}
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warningMessage))
	return nil
}
