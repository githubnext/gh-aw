// Package cli provides command-line interface functionality for gh-aw.
// This file (logs_github_api.go) contains functions for interacting with the GitHub API
// to fetch workflow runs, job statuses, and job details.
//
// Key responsibilities:
//   - Listing workflow runs with pagination
//   - Fetching job statuses and details for workflow runs
//   - Handling GitHub CLI authentication and error responses
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/sliceutil"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var logsGitHubAPILog = logger.New("cli:logs_github_api")

// fetchJobStatuses gets job information for a workflow run and counts failed jobs
func fetchJobStatuses(runID int64, verbose bool) (int, error) {
	logsGitHubAPILog.Printf("Fetching job statuses: runID=%d", runID)

	if verbose {
		fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Fetching job statuses for run %d", runID)))
	}

	cmd := workflow.ExecGH("api", fmt.Sprintf("repos/{owner}/{repo}/actions/runs/%d/jobs", runID), "--jq", ".jobs[] | {name: .name, status: .status, conclusion: .conclusion}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if verbose {
			fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Failed to fetch job statuses for run %d: %v", runID, err)))
		}
		// Don't fail the entire operation if we can't get job info
		return 0, nil
	}

	// Parse each line as a separate JSON object
	failedJobs := 0
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var job JobInfo
		if err := json.Unmarshal([]byte(line), &job); err != nil {
			if verbose {
				fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Failed to parse job info: %s", line)))
			}
			continue
		}

		// Count jobs with failure conclusions as errors
		if isFailureConclusion(job.Conclusion) {
			failedJobs++
			logsGitHubAPILog.Printf("Found failed job: name=%s, conclusion=%s", job.Name, job.Conclusion)
			if verbose {
				fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Found failed job '%s' with conclusion '%s'", job.Name, job.Conclusion)))
			}
		}
	}

	logsGitHubAPILog.Printf("Job status check complete: failedJobs=%d", failedJobs)
	return failedJobs, nil
}

// fetchJobDetails gets detailed job information including durations for a workflow run
func fetchJobDetails(runID int64, verbose bool) ([]JobInfoWithDuration, error) {
	logsGitHubAPILog.Printf("Fetching job details: runID=%d", runID)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching job details for run %d", runID)))
	}

	cmd := workflow.ExecGH("api", fmt.Sprintf("repos/{owner}/{repo}/actions/runs/%d/jobs", runID), "--jq", ".jobs[] | {name: .name, status: .status, conclusion: .conclusion, started_at: .started_at, completed_at: .completed_at}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Failed to fetch job details for run %d: %v", runID, err)))
		}
		// Don't fail the entire operation if we can't get job info
		return nil, nil
	}

	var jobs []JobInfoWithDuration
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var job JobInfo
		if err := json.Unmarshal([]byte(line), &job); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Failed to parse job info: %s", line)))
			}
			continue
		}

		jobWithDuration := JobInfoWithDuration{
			JobInfo: job,
		}

		// Calculate duration if both timestamps are available
		if !job.StartedAt.IsZero() && !job.CompletedAt.IsZero() {
			jobWithDuration.Duration = job.CompletedAt.Sub(job.StartedAt)
		}

		jobs = append(jobs, jobWithDuration)
	}

	return jobs, nil
}

// listWorkflowRunsWithPagination fetches workflow runs from GitHub Actions using the GitHub CLI.
//
// This function retrieves workflow runs with pagination support and applies various filters:
//   - workflowName: filter by specific workflow (if empty, fetches all agentic workflows)
//   - limit: maximum number of runs to fetch in this API call (batch size)
//   - startDate/endDate: filter by creation date range
//   - beforeDate: used for pagination (fetch runs created before this date)
//   - ref: filter by branch or tag name
//   - beforeRunID/afterRunID: filter by run database ID range
//   - repoOverride: fetch from a specific repository instead of current
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
func listWorkflowRunsWithPagination(workflowName string, limit int, startDate, endDate, beforeDate, ref string, beforeRunID, afterRunID int64, repoOverride string, processedCount, targetCount int, verbose bool) ([]WorkflowRun, int, error) {
	logsGitHubAPILog.Printf("Listing workflow runs: workflow=%s, limit=%d, startDate=%s, endDate=%s, ref=%s", workflowName, limit, startDate, endDate, ref)
	args := []string{"run", "list", "--json", "databaseId,number,url,status,conclusion,workflowName,createdAt,startedAt,updatedAt,event,headBranch,headSha,displayTitle"}

	// Add filters
	if workflowName != "" {
		args = append(args, "--workflow", workflowName)
	}
	if limit > 0 {
		args = append(args, "--limit", strconv.Itoa(limit))
	}
	if startDate != "" {
		args = append(args, "--created", ">="+startDate)
	}
	if endDate != "" {
		args = append(args, "--created", "<="+endDate)
	}
	// Add beforeDate filter for pagination
	if beforeDate != "" {
		args = append(args, "--created", "<"+beforeDate)
	}
	// Add ref filter (uses --branch flag which also works for tags)
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	// Add repo filter
	if repoOverride != "" {
		args = append(args, "--repo", repoOverride)
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Executing: gh %s", strings.Join(args, " "))))
	}

	// Start spinner for network operation
	spinnerMsg := fmt.Sprintf("Fetching workflow runs from GitHub... (%d / %d)", processedCount, targetCount)
	spinner := console.NewSpinner(spinnerMsg)
	if !verbose {
		spinner.Start()
	}

	cmd := workflow.ExecGH(args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Stop spinner on error
		if !verbose {
			spinner.Stop()
		}

		// Extract detailed error information including exit code
		var exitCode int
		if exitErr, ok := err.(*exec.ExitError); ok {
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
		if verbose {
			fmt.Println(console.FormatVerboseMessage(outputMsg))
		}
		
		// Check for invalid field errors first (before auth errors)
		// GitHub CLI returns these when JSON fields don't exist or are misspelled
		if strings.Contains(combinedMsg, "invalid field") ||
			strings.Contains(combinedMsg, "unknown field") ||
			strings.Contains(combinedMsg, "field not found") ||
			strings.Contains(combinedMsg, "no such field") {
			return nil, 0, fmt.Errorf("invalid field in JSON query (exit code %d): %s", exitCode, string(output))
		}
		
		// Check for authentication errors
		if strings.Contains(combinedMsg, "exit status 4") ||
			strings.Contains(combinedMsg, "exit status 1") ||
			strings.Contains(combinedMsg, "not logged into any GitHub hosts") ||
			strings.Contains(combinedMsg, "To use GitHub CLI in a GitHub Actions workflow") ||
			strings.Contains(combinedMsg, "authentication required") ||
			strings.Contains(outputMsg, "gh auth login") {
			return nil, 0, fmt.Errorf("GitHub CLI authentication required. Run 'gh auth login' first")
		}
		
		if len(output) > 0 {
			return nil, 0, fmt.Errorf("failed to list workflow runs (exit code %d): %s", exitCode, string(output))
		}
		return nil, 0, fmt.Errorf("failed to list workflow runs (exit code %d): %w", exitCode, err)
	}

	var runs []WorkflowRun
	if err := json.Unmarshal(output, &runs); err != nil {
		// Stop spinner on parse error
		if !verbose {
			spinner.Stop()
		}
		return nil, 0, fmt.Errorf("failed to parse workflow runs: %w", err)
	}

	// Stop spinner with success message
	if !verbose {
		spinner.StopWithMessage(fmt.Sprintf("✓ Fetched %d workflow runs", len(runs)))
	}

	// Store the total count fetched from API before filtering
	totalFetched := len(runs)

	// Filter only agentic workflow runs when no specific workflow is specified
	// If a workflow name was specified, we already filtered by it in the API call
	var agenticRuns []WorkflowRun
	if workflowName == "" {
		// No specific workflow requested, filter to only agentic workflows
		// Get the list of agentic workflow names from .lock.yml files
		agenticWorkflowNames, err := getAgenticWorkflowNames(verbose)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get agentic workflow names: %w", err)
		}

		for _, run := range runs {
			if sliceutil.Contains(agenticWorkflowNames, run.WorkflowName) {
				agenticRuns = append(agenticRuns, run)
			}
		}
	} else {
		// Specific workflow requested, return all runs (they're already filtered by GitHub API)
		agenticRuns = runs
	}

	// Apply run ID filtering if specified
	if beforeRunID > 0 || afterRunID > 0 {
		var filteredRuns []WorkflowRun
		for _, run := range agenticRuns {
			// Apply before-run-id filter (exclusive)
			if beforeRunID > 0 && run.DatabaseID >= beforeRunID {
				continue
			}
			// Apply after-run-id filter (exclusive)
			if afterRunID > 0 && run.DatabaseID <= afterRunID {
				continue
			}
			filteredRuns = append(filteredRuns, run)
		}
		agenticRuns = filteredRuns
	}

	return agenticRuns, totalFetched, nil
}
