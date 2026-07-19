package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var runWorkflowTrackingLog = logger.New("cli:run_workflow_tracking")

// WorkflowRunInfo contains information about a workflow run
type WorkflowRunInfo struct {
	URL        string
	DatabaseID int64
	Status     string
	Conclusion string
	CreatedAt  time.Time
}

// getLatestWorkflowRunWithRetry gets information about the most recent run of the specified workflow
// with retry logic to handle timing issues when a workflow has just been triggered
func getLatestWorkflowRunWithRetry(lockFileName string, repo string, verbose bool) (*WorkflowRunInfo, error) {
	runWorkflowTrackingLog.Printf("Getting latest workflow run: workflow=%s, repo=%s, max_retries=6", lockFileName, repo)
	const maxRetries = 6
	const initialDelay = 2 * time.Second
	const maxDelay = 10 * time.Second

	getLatestWorkflowRunWithRetryLogStart(lockFileName, repo, verbose)
	startTime := time.Now().UTC()
	runWorkflowTrackingLog.Printf("Start time for polling: %s", startTime.Format(time.RFC3339))
	spinner := getLatestWorkflowRunWithRetrySpinner(verbose)

	var lastErr error
	for attempt := range maxRetries {
		getLatestWorkflowRunWithRetryDelay(attempt, maxRetries, initialDelay, maxDelay, startTime, spinner, verbose)
		runInfo, retry, err := getLatestWorkflowRunWithRetryAttempt(lockFileName, repo, verbose, startTime, attempt, maxRetries, spinner)
		if err != nil {
			lastErr = err
			if retry {
				continue
			}
			return nil, err
		}
		if runInfo != nil {
			return runInfo, nil
		}
	}
	if spinner != nil {
		spinner.Stop()
	}
	if lastErr != nil {
		return nil, fmt.Errorf("failed to get workflow run after %d attempts: %w", maxRetries, lastErr)
	}
	return nil, fmt.Errorf("no workflow run found after %d attempts", maxRetries)
}

func getLatestWorkflowRunWithRetryLogStart(lockFileName, repo string, verbose bool) {
	if repo != "" {
		console.LogVerbose(verbose, fmt.Sprintf("Getting latest run for workflow: %s in repo: %s (with retry logic)", lockFileName, repo))
	} else {
		console.LogVerbose(verbose, fmt.Sprintf("Getting latest run for workflow: %s (with retry logic)", lockFileName))
	}
}

func getLatestWorkflowRunWithRetrySpinner(verbose bool) *console.SpinnerWrapper {
	if verbose {
		return nil
	}
	return console.NewSpinner("Waiting for workflow run to appear...")
}

func getLatestWorkflowRunWithRetryDelay(attempt, maxRetries int, initialDelay, maxDelay time.Duration, startTime time.Time, spinner *console.SpinnerWrapper, verbose bool) {
	if attempt <= 0 {
		return
	}
	delay := min(time.Duration(attempt)*initialDelay, maxDelay)
	elapsed := time.Since(startTime).Round(time.Second)
	console.LogVerbose(verbose, fmt.Sprintf("Waiting %v before retry attempt %d/%d...", delay, attempt+1, maxRetries))
	if !verbose && spinner != nil {
		if attempt == 1 {
			spinner.Start()
		}
		spinner.UpdateMessage(fmt.Sprintf("Waiting for workflow run... (attempt %d/%d, %v elapsed)", attempt+1, maxRetries, elapsed))
	}
	time.Sleep(delay)
}

func getLatestWorkflowRunWithRetryAttempt(lockFileName, repo string, verbose bool, startTime time.Time, attempt, maxRetries int, spinner *console.SpinnerWrapper) (*WorkflowRunInfo, bool, error) {
	output, err := getLatestWorkflowRunWithRetryOutput(lockFileName, repo)
	if err != nil {
		runWorkflowTrackingLog.Printf("Attempt %d/%d failed to get runs: %v", attempt+1, maxRetries, err)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Attempt %d/%d failed: %v", attempt+1, maxRetries, err)))
		}
		return nil, true, fmt.Errorf("failed to get workflow runs: %w", err)
	}
	if len(output) == 0 || string(output) == "[]" {
		runWorkflowTrackingLog.Printf("Attempt %d/%d: no runs found, output empty or []", attempt+1, maxRetries)
		console.LogVerbose(verbose, fmt.Sprintf("Attempt %d/%d: no runs found yet", attempt+1, maxRetries))
		return nil, true, errors.New("no runs found for workflow")
	}
	run, err := getLatestWorkflowRunWithRetryParse(output, verbose, attempt, maxRetries)
	if err != nil {
		return nil, true, err
	}
	return getLatestWorkflowRunWithRetrySelect(run, startTime, attempt, maxRetries, spinner, verbose)
}

func getLatestWorkflowRunWithRetryOutput(lockFileName, repo string) ([]byte, error) {
	if repo != "" {
		return workflow.ExecGH("run", "list", "--repo", repo, "--workflow", lockFileName, "--limit", "1", "--json", "url,databaseId,status,conclusion,createdAt").Output()
	}
	return workflow.ExecGH("run", "list", "--workflow", lockFileName, "--limit", "1", "--json", "url,databaseId,status,conclusion,createdAt").Output()
}

type getLatestWorkflowRunWithRetryRun struct {
	URL        string `json:"url"`
	DatabaseID int64  `json:"databaseId"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	CreatedAt  string `json:"createdAt"`
}

func getLatestWorkflowRunWithRetryParse(output []byte, verbose bool, attempt, maxRetries int) (getLatestWorkflowRunWithRetryRun, error) {
	var runs []getLatestWorkflowRunWithRetryRun
	if err := json.Unmarshal(output, &runs); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Attempt %d/%d failed to parse JSON: %v", attempt+1, maxRetries, err)))
		}
		return getLatestWorkflowRunWithRetryRun{}, fmt.Errorf("failed to parse workflow run data: %w", err)
	}
	if len(runs) == 0 {
		console.LogVerbose(verbose, fmt.Sprintf("Attempt %d/%d: no runs in parsed JSON", attempt+1, maxRetries))
		return getLatestWorkflowRunWithRetryRun{}, errors.New("no runs found")
	}
	return runs[0], nil
}

func getLatestWorkflowRunWithRetrySelect(run getLatestWorkflowRunWithRetryRun, startTime time.Time, attempt, maxRetries int, spinner *console.SpinnerWrapper, verbose bool) (*WorkflowRunInfo, bool, error) {
	createdAt := getLatestWorkflowRunWithRetryCreatedAt(run.CreatedAt, verbose)
	runInfo := &WorkflowRunInfo{URL: run.URL, DatabaseID: run.DatabaseID, Status: run.Status, Conclusion: run.Conclusion, CreatedAt: createdAt}
	if !createdAt.IsZero() && createdAt.After(startTime.Add(-30*time.Second)) {
		runWorkflowTrackingLog.Printf("Found matching run: id=%d, created_at=%s, within_tolerance=true", run.DatabaseID, createdAt.Format(time.RFC3339))
		console.LogVerbose(verbose, fmt.Sprintf("Found recent run (ID: %d) created at %v (started polling at %v)", run.DatabaseID, createdAt.Format(time.RFC3339), startTime.Format(time.RFC3339)))
		if spinner != nil {
			spinner.StopWithMessage("✓ Found workflow run")
		}
		return runInfo, false, nil
	}
	getLatestWorkflowRunWithRetryLogOldRun(run, createdAt, attempt, maxRetries, verbose)
	if attempt < 3 {
		return nil, true, errors.New("workflow run appears to be from a previous execution")
	}
	console.LogVerbose(verbose, fmt.Sprintf("Returning workflow run (ID: %d) after %d attempts (timing uncertain)", run.DatabaseID, attempt+1))
	if spinner != nil {
		spinner.StopWithMessage("✓ Found workflow run")
	}
	return runInfo, false, nil
}

func getLatestWorkflowRunWithRetryCreatedAt(value string, verbose bool) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsedTime, err := time.Parse(time.RFC3339, value)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not parse creation time '%s': %v", value, err)))
		}
		return time.Time{}
	}
	return parsedTime
}

func getLatestWorkflowRunWithRetryLogOldRun(run getLatestWorkflowRunWithRetryRun, createdAt time.Time, attempt, maxRetries int, verbose bool) {
	if createdAt.IsZero() {
		console.LogVerbose(verbose, fmt.Sprintf("Attempt %d/%d: Found run (ID: %d) but no creation timestamp available", attempt+1, maxRetries, run.DatabaseID))
		return
	}
	console.LogVerbose(verbose, fmt.Sprintf("Attempt %d/%d: Found run (ID: %d) but it was created at %v (too old)", attempt+1, maxRetries, run.DatabaseID, createdAt.Format(time.RFC3339)))
}
