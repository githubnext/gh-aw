// This file provides command-line interface functionality for gh-aw.
// This file (logs_github_api.go) contains functions for interacting with the GitHub API
// to fetch workflow runs, job statuses, and job details.
//
// Key responsibilities:
//   - Listing workflow runs with pagination
//   - Fetching job statuses and details for workflow runs
//   - Handling GitHub CLI authentication and error responses

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var logsGitHubAPILog = logger.New("cli:logs_github_api")

// buildCreatedFilter constructs a single --created filter value from the provided date
// bounds.  Using a single --created flag is required because gh run list treats --created
// as a single string flag; supplying it multiple times only keeps the last value, silently
// discarding earlier bounds (see https://github.com/cli/cli/blob/trunk/pkg/cmd/run/list/list.go).
//
// When both a lower bound (startDate) and an upper bound are present the function uses
// GitHub's range syntax ("start..end") so that both bounds are enforced in one expression.
//
// beforeDate is an exclusive upper bound used for cursor-based pagination.  Because the
// range syntax is inclusive on both ends, one second is subtracted from beforeDate so that
// the run at the cursor position is not returned again on the next page.
func buildCreatedFilter(startDate, endDate, beforeDate string) string {
	// Determine the effective inclusive upper bound.
	var upper string
	if beforeDate != "" {
		// beforeDate is exclusive (< beforeDate); convert to inclusive by subtracting 1 s.
		t, err := time.Parse(time.RFC3339, beforeDate)
		if err == nil {
			upper = t.Add(-time.Second).Format(time.RFC3339)
		} else {
			// Unparseable beforeDate: use it as-is and treat as inclusive best-effort.
			// Log a warning so the caller knows the exact exclusive bound may be missed.
			logsGitHubAPILog.Printf("buildCreatedFilter: could not parse beforeDate %q as RFC3339, using as-is: %v", beforeDate, err)
			upper = beforeDate
		}
	} else if endDate != "" {
		upper = endDate
	}

	switch {
	case startDate != "" && upper != "":
		return startDate + ".." + upper
	case startDate != "":
		return ">=" + startDate
	case beforeDate != "":
		// No startDate, but we have a pagination cursor: keep the original < form.
		return "<" + beforeDate
	case endDate != "":
		return "<=" + endDate
	default:
		return ""
	}
}

// fetchJobDetailsWithCounts fetches all job information for a workflow run in a single API
// call and returns the full detail slice together with the count of failed jobs.
// It is the single source of truth for the jobs endpoint; fetchJobDetails and
// fetchJobStatuses are thin wrappers that each return only the value they need.
func fetchJobDetailsWithCounts(runID int64, verbose bool) ([]JobInfoWithDuration, int, error) {
	logsGitHubAPILog.Printf("Fetching job details: runID=%d", runID)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching job details for run %d", runID)))
	}

	output, err := workflow.RunGHCombined("Fetching job details...", "api",
		fmt.Sprintf("repos/{owner}/{repo}/actions/runs/%d/jobs", runID),
		"--jq", ".jobs[] | {name: .name, status: .status, conclusion: .conclusion, started_at: .started_at, completed_at: .completed_at}")
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Failed to fetch job details for run %d: %v", runID, err)))
		}
		return nil, 0, err
	}

	var jobs []JobInfoWithDuration
	failedJobs := 0
	lines := strings.SplitSeq(strings.TrimSpace(string(output)), "\n")
	for line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var job JobInfo
		if err := json.Unmarshal([]byte(line), &job); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Failed to parse job info: "+line))
			}
			continue
		}

		jobWithDuration := JobInfoWithDuration{JobInfo: job}
		if !job.StartedAt.IsZero() && !job.CompletedAt.IsZero() {
			jobWithDuration.Duration = job.CompletedAt.Sub(job.StartedAt)
		}
		jobs = append(jobs, jobWithDuration)

		if isFailureConclusion(job.Conclusion) {
			failedJobs++
			logsGitHubAPILog.Printf("Found failed job: name=%s, conclusion=%s", job.Name, job.Conclusion)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Found failed job '%s' with conclusion '%s'", job.Name, job.Conclusion)))
			}
		}
	}

	logsGitHubAPILog.Printf("Job fetch complete: total=%d failed=%d", len(jobs), failedJobs)
	return jobs, failedJobs, nil
}

// fetchJobDetails gets detailed job information including durations for a workflow run.
// Errors from the underlying API call are suppressed so that callers can continue
// processing even when job data is unavailable (e.g. missing permissions).
func fetchJobDetails(runID int64, verbose bool) ([]JobInfoWithDuration, error) {
	jobs, _, err := fetchJobDetailsWithCounts(runID, verbose)
	if err != nil {
		// Don't fail the entire operation if we can't get job info
		return nil, nil
	}
	return jobs, nil
}

// fetchJobStatuses gets the count of failed jobs for a workflow run.
// Errors from the underlying API call are suppressed so that callers can continue
// processing even when job data is unavailable (e.g. missing permissions).
func fetchJobStatuses(runID int64, verbose bool) (int, error) {
	_, failedJobs, err := fetchJobDetailsWithCounts(runID, verbose)
	if err != nil {
		// Don't fail the entire operation if we can't get job info
		return 0, nil
	}
	return failedJobs, nil
}

// ListWorkflowRunsOptions holds the options for listWorkflowRunsWithPagination
type ListWorkflowRunsOptions struct {
	Context      context.Context
	WorkflowName string // filter by specific workflow (if empty, fetches all agentic workflows)
	Status       string // filter by run status/conclusion (for example: completed, success, failure)
	Limit        int    // maximum number of runs to fetch in this API call (batch size)
	StartDate    string // filter by creation date (>=); combined with EndDate/BeforeDate into a single --created range
	EndDate      string // filter by creation date (<=); combined with StartDate into a single --created range
	BeforeDate   string // exclusive upper bound used for pagination (<); combined with StartDate into a single --created range
	Ref          string // filter by branch or tag name
	BeforeRunID  int64  // filter by run database ID (< this ID)
	AfterRunID   int64  // filter by run database ID (> this ID)
	RepoOverride string // fetch from a specific repository instead of current
	// OldestFetchedCreatedAt, when set, is populated with the oldest run creation
	// timestamp returned by GitHub in this batch before any workflow/conclusion filtering.
	OldestFetchedCreatedAt *time.Time
	ProcessedCount         int  // number of runs already processed (for progress display)
	TargetCount            int  // target number of runs to fetch (for progress display)
	Verbose                bool // enable verbose logging
}

// listWorkflowRunsWithPagination fetches workflow runs from GitHub Actions using the GitHub CLI.
//
// This function retrieves workflow runs with pagination support and applies various filters
// as specified in the ListWorkflowRunsOptions.
//
// Returns:
//   - []WorkflowRun: filtered list of workflow runs
//   - int: total number of runs fetched from API before agentic workflow filtering
//   - error: any error that occurred
//
// The totalFetched count is critical for pagination - it indicates whether more data is available
// from GitHub, whereas the filtered runs count may be much smaller after filtering for agentic workflows.
//
// The limit parameter specifies the batch size for the GitHub API call (how many runs to fetch in this request),
// not the total number of matching runs the user wants to find.
//
// The processedCount and targetCount parameters are used to display progress in the spinner message.
func listWorkflowRunsWithPagination(opts ListWorkflowRunsOptions) ([]WorkflowRun, int, error) {
	logsGitHubAPILog.Printf("Listing workflow runs: workflow=%s, limit=%d, startDate=%s, endDate=%s, ref=%s", opts.WorkflowName, opts.Limit, opts.StartDate, opts.EndDate, opts.Ref)
	args := []string{"run", "list", "--json", "databaseId,number,url,status,conclusion,workflowName,createdAt,startedAt,updatedAt,event,headBranch,headSha,displayTitle"}

	// Add filters
	if opts.WorkflowName != "" {
		args = append(args, "--workflow", opts.WorkflowName)
	}
	if opts.Status != "" {
		args = append(args, "--status", opts.Status)
	}
	if opts.Limit > 0 {
		args = append(args, "--limit", strconv.Itoa(opts.Limit))
	}
	// Build a single --created filter that covers the full date range.
	// gh run list's --created flag is a single string (not a slice); passing it
	// multiple times only keeps the last value and silently drops earlier bounds.
	// buildCreatedFilter combines all bounds into one expression so that every
	// bound is honoured.
	if createdFilter := buildCreatedFilter(opts.StartDate, opts.EndDate, opts.BeforeDate); createdFilter != "" {
		args = append(args, "--created", createdFilter)
	}
	// Add ref filter (uses --branch flag which also works for tags)
	if opts.Ref != "" {
		args = append(args, "--branch", opts.Ref)
	}
	// Add repo filter
	if opts.RepoOverride != "" {
		args = append(args, "--repo", opts.RepoOverride)
	}

	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Executing: gh "+strings.Join(args, " ")))
	}

	// Start spinner for network operation
	spinnerMsg := workflowRunsSpinnerMessage(opts)
	spinner := console.NewSpinner(spinnerMsg)
	if !opts.Verbose {
		spinner.Start()
	}

	cmdCtx := opts.Context
	if cmdCtx == nil {
		cmdCtx = context.Background()
	}
	cmd := workflow.ExecGHContext(cmdCtx, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Stop spinner on error
		if !opts.Verbose {
			spinner.Stop()
		}

		// Extract detailed error information including exit code
		var exitCode int
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
			logsGitHubAPILog.Printf("gh run list command failed with exit code %d. Command: gh %v", exitCode, args)
			logsGitHubAPILog.Printf("combined output: %s", string(output))
		} else {
			logsGitHubAPILog.Printf("gh run list command failed (not ExitError): %v. Command: gh %v", err, args)
		}

		// Check for different error types with heuristics
		errMsg := err.Error()
		outputMsg := string(output)
		combinedMsg := errMsg + " " + outputMsg
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(outputMsg))
		}

		// Check for invalid field errors first (before auth errors).
		// GitHub CLI may capitalise the message differently across versions, so
		// use a case-insensitive comparison. Note that some gh versions emit
		// "Unknown JSON field: ..." (with "JSON" between "Unknown" and "field"),
		// so we also check for "unknown json field" and "unknown json" explicitly.
		combinedMsgLower := strings.ToLower(combinedMsg)
		if strings.Contains(combinedMsgLower, "invalid field") ||
			strings.Contains(combinedMsgLower, "unknown field") ||
			strings.Contains(combinedMsgLower, "unknown json field") ||
			strings.Contains(combinedMsgLower, "unknown json") ||
			strings.Contains(combinedMsgLower, "field not found") ||
			strings.Contains(combinedMsgLower, "no such field") {
			return nil, 0, fmt.Errorf("invalid field in JSON query (exit code %d): %s", exitCode, string(output))
		}

		// Check for authentication errors.
		// "exit status 1" is intentionally omitted: gh exits 1 for many non-auth
		// errors (e.g. unsupported JSON fields), so matching it caused misleading
		// "authentication required" messages for unrelated failures.
		if isPermissionErrorStr(combinedMsg) {
			return nil, 0, errors.New("GitHub CLI authentication required. Run 'gh auth login' first")
		}

		if len(output) > 0 {
			return nil, 0, fmt.Errorf("failed to list workflow runs (exit code %d): %s", exitCode, string(output))
		}
		return nil, 0, fmt.Errorf("failed to list workflow runs (exit code %d): %w", exitCode, err)
	}

	var runs []WorkflowRun
	if err := json.Unmarshal(output, &runs); err != nil {
		// Stop spinner on parse error
		if !opts.Verbose {
			spinner.Stop()
		}
		return nil, 0, fmt.Errorf("failed to parse workflow runs: %w", err)
	}

	// Stop spinner silently - don't show per-iteration messages
	if !opts.Verbose {
		spinner.Stop()
	}

	// Store the total count fetched from API before filtering
	totalFetched := len(runs)
	if opts.OldestFetchedCreatedAt != nil {
		var oldest time.Time
		if totalFetched > 0 {
			oldest = runs[totalFetched-1].CreatedAt
		}
		*opts.OldestFetchedCreatedAt = oldest
	}

	// Filter only agentic workflow runs when no specific workflow is specified
	// If a workflow name was specified, we already filtered by it in the API call
	var agenticRuns []WorkflowRun
	if opts.WorkflowName == "" {
		// No specific workflow requested, filter to only agentic workflows
		// Get the list of agentic workflow names from .lock.yml files
		agenticWorkflowNames, err := getAgenticWorkflowNames(opts.Verbose)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get agentic workflow names: %w", err)
		}

		for _, run := range runs {
			if slices.Contains(agenticWorkflowNames, run.WorkflowName) {
				agenticRuns = append(agenticRuns, run)
			}
		}
	} else {
		// Specific workflow requested, return all runs (they're already filtered by GitHub API)
		agenticRuns = runs
	}

	// Apply run ID filtering if specified
	if opts.BeforeRunID > 0 || opts.AfterRunID > 0 {
		var filteredRuns []WorkflowRun
		for _, run := range agenticRuns {
			// Apply before-run-id filter (exclusive)
			if opts.BeforeRunID > 0 && run.DatabaseID >= opts.BeforeRunID {
				continue
			}
			// Apply after-run-id filter (exclusive)
			if opts.AfterRunID > 0 && run.DatabaseID <= opts.AfterRunID {
				continue
			}
			filteredRuns = append(filteredRuns, run)
		}
		agenticRuns = filteredRuns
	}

	// Filter out skipped and cancelled runs — they carry no useful agentic data
	// and should not count toward the requested run count.
	{
		filtered := agenticRuns[:0]
		for _, run := range agenticRuns {
			if run.Conclusion == "skipped" || run.Conclusion == "cancelled" {
				continue
			}
			filtered = append(filtered, run)
		}
		agenticRuns = filtered
	}

	return agenticRuns, totalFetched, nil
}

func workflowRunsSpinnerMessage(opts ListWorkflowRunsOptions) string {
	if opts.TargetCount > 0 {
		return fmt.Sprintf("Fetching workflow runs from GitHub... (%d / %d)", opts.ProcessedCount, opts.TargetCount)
	}
	return "Fetching workflow runs from GitHub..."
}
