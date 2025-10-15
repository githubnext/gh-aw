package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/parser"
)

// RunWorkflowOnGitHub runs an agentic workflow on GitHub Actions
func RunWorkflowOnGitHub(workflowIdOrName string, enable bool, verbose bool) error {
	if workflowIdOrName == "" {
		return fmt.Errorf("workflow name or ID is required")
	}

	if verbose {
		fmt.Printf("Running workflow on GitHub Actions: %s\n", workflowIdOrName)
	}

	// Check if gh CLI is available
	if !isGHCLIAvailable() {
		return fmt.Errorf("GitHub CLI (gh) is required but not available")
	}

	// Try to resolve the workflow file path to find the corresponding .lock.yml file
	workflowFile, err := resolveWorkflowFile(workflowIdOrName, verbose)
	if err != nil {
		return fmt.Errorf("failed to resolve workflow: %w", err)
	}

	// Check if the workflow is runnable (has workflow_dispatch trigger)
	runnable, err := IsRunnable(workflowFile)
	if err != nil {
		return fmt.Errorf("failed to check if workflow %s is runnable: %w", workflowFile, err)
	}

	if !runnable {
		return fmt.Errorf("workflow '%s' cannot be run on GitHub Actions - it must have 'workflow_dispatch' trigger", workflowIdOrName)
	}

	// Handle --enable flag logic: check workflow state and enable if needed
	var wasDisabled bool
	var workflowID int64
	if enable {
		// Get current workflow status
		workflow, err := getWorkflowStatus(workflowIdOrName, verbose)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Could not check workflow status: %v\n", err)
			}
		}

		// If we successfully got workflow status, check if it needs enabling
		if err == nil {
			workflowID = workflow.ID
			if workflow.State == "disabled_manually" {
				wasDisabled = true
				if verbose {
					fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Workflow '%s' is disabled, enabling it temporarily...", workflowIdOrName)))
				}
				// Enable the workflow
				cmd := exec.Command("gh", "workflow", "enable", strconv.FormatInt(workflow.ID, 10))
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("failed to enable workflow '%s': %w", workflowIdOrName, err)
				}
				fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Enabled workflow: %s", workflowIdOrName)))
			}
		}
	}

	// Determine the lock file name based on the workflow source
	var lockFileName string

	// Check that the workflow exists locally before trying to run it
	workflowsDir := getWorkflowsDir()

	_, _, err = readWorkflowFile(workflowIdOrName+".md", workflowsDir)
	if err != nil {
		return fmt.Errorf("failed to find workflow in local .github/workflows or components: %w", err)
	}

	// For local workflows, use the simple filename
	filename := strings.TrimSuffix(filepath.Base(workflowIdOrName), ".md")
	lockFileName = filename + ".lock.yml"

	// Check if the lock file exists in .github/workflows
	lockFilePath := filepath.Join(".github/workflows", lockFileName)
	if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
		return fmt.Errorf("workflow lock file '%s' not found in .github/workflows - run '"+constants.CLIExtensionPrefix+" compile' first", lockFileName)
	}

	if verbose {
		fmt.Printf("Using lock file: %s\n", lockFileName)
	}

	// Execute gh workflow run command and capture output
	cmd := exec.Command("gh", "workflow", "run", lockFileName)

	if verbose {
		fmt.Printf("Executing: gh workflow run %s\n", lockFileName)
	}

	// Capture both stdout and stderr
	stdout, err := cmd.Output()
	if err != nil {
		// If there's an error, try to get stderr for better error reporting
		if exitError, ok := err.(*exec.ExitError); ok {
			fmt.Fprintf(os.Stderr, "%s", exitError.Stderr)
		}

		// Restore workflow state if it was disabled and we enabled it (even on error)
		if enable && wasDisabled && workflowID != 0 {
			restoreWorkflowState(workflowIdOrName, workflowID, verbose)
		}

		return fmt.Errorf("failed to run workflow on GitHub Actions: %w", err)
	}

	// Display the output from gh workflow run
	output := strings.TrimSpace(string(stdout))
	if output != "" {
		fmt.Println(output)
	}

	fmt.Printf("Successfully triggered workflow: %s\n", lockFileName)

	// Try to get the latest run for this workflow to show a direct link
	// Add a delay to allow GitHub Actions time to register the new workflow run
	if runInfo, err := getLatestWorkflowRunWithRetry(lockFileName, "", verbose); err == nil && runInfo.URL != "" {
		fmt.Printf("\n🔗 View workflow run: %s\n", runInfo.URL)
	} else if verbose && err != nil {
		fmt.Printf("Note: Could not get workflow run URL: %v\n", err)
	}

	// Restore workflow state if it was disabled and we enabled it
	if enable && wasDisabled && workflowID != 0 {
		restoreWorkflowState(workflowIdOrName, workflowID, verbose)
	}

	return nil
}

// RunWorkflowsOnGitHub runs multiple agentic workflows on GitHub Actions, optionally repeating at intervals
func RunWorkflowsOnGitHub(workflowNames []string, repeatSeconds int, enable bool, verbose bool) error {
	if len(workflowNames) == 0 {
		return fmt.Errorf("at least one workflow name or ID is required")
	}

	// Validate all workflows exist and are runnable before starting
	for _, workflowName := range workflowNames {
		if workflowName == "" {
			return fmt.Errorf("workflow name cannot be empty")
		}

		// Check if workflow exists and is runnable
		workflowFile, err := resolveWorkflowFile(workflowName, verbose)
		if err != nil {
			return fmt.Errorf("failed to resolve workflow '%s': %w", workflowName, err)
		}

		runnable, err := IsRunnable(workflowFile)
		if err != nil {
			return fmt.Errorf("failed to check if workflow '%s' is runnable: %w", workflowName, err)
		}

		if !runnable {
			return fmt.Errorf("workflow '%s' cannot be run on GitHub Actions - it must have 'workflow_dispatch' trigger", workflowName)
		}
	}

	// Function to run all workflows once
	runAllWorkflows := func() error {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Running %d workflow(s)...", len(workflowNames))))

		for i, workflowName := range workflowNames {
			if len(workflowNames) > 1 {
				fmt.Println(console.FormatProgressMessage(fmt.Sprintf("Running workflow %d/%d: %s", i+1, len(workflowNames), workflowName)))
			}

			if err := RunWorkflowOnGitHub(workflowName, enable, verbose); err != nil {
				return fmt.Errorf("failed to run workflow '%s': %w", workflowName, err)
			}

			// Add a small delay between workflows to avoid overwhelming GitHub API
			if i < len(workflowNames)-1 {
				time.Sleep(1 * time.Second)
			}
		}

		fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Successfully triggered %d workflow(s)", len(workflowNames))))
		return nil
	}

	// Execute workflows with optional repeat functionality
	return ExecuteWithRepeat(RepeatOptions{
		RepeatSeconds: repeatSeconds,
		RepeatMessage: "Repeating workflow run at %s",
		ExecuteFunc:   runAllWorkflows,
		UseStderr:     false, // Use stdout for run command
	})
}

// IsRunnable checks if a workflow can be run (has schedule or workflow_dispatch trigger)
func IsRunnable(markdownPath string) (bool, error) {
	// Read the file
	contentBytes, err := os.ReadFile(markdownPath)
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}
	content := string(contentBytes)

	// Extract frontmatter
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return false, fmt.Errorf("failed to extract frontmatter: %w", err)
	}

	// Check if 'on' section is present
	onSection, exists := result.Frontmatter["on"]
	if !exists {
		// If no 'on' section, it defaults to runnable triggers (schedule, workflow_dispatch)
		return true, nil
	}

	// Convert to string to analyze
	onStr := fmt.Sprintf("%v", onSection)
	onStrLower := strings.ToLower(onStr)

	// Check for schedule or workflow_dispatch triggers
	hasSchedule := strings.Contains(onStrLower, "schedule") || strings.Contains(onStrLower, "cron")
	hasWorkflowDispatch := strings.Contains(onStrLower, "workflow_dispatch")

	return hasSchedule || hasWorkflowDispatch, nil
}

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
	const maxRetries = 6
	const initialDelay = 2 * time.Second
	const maxDelay = 10 * time.Second

	if verbose {
		if repo != "" {
			fmt.Printf("Getting latest run for workflow: %s in repo: %s (with retry logic)\n", lockFileName, repo)
		} else {
			fmt.Printf("Getting latest run for workflow: %s (with retry logic)\n", lockFileName)
		}
	}

	// Capture the current time before we start polling
	// This helps us identify runs that were created after the workflow was triggered
	startTime := time.Now().UTC()

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff, capped at maxDelay
			delay := time.Duration(attempt) * initialDelay
			if delay > maxDelay {
				delay = maxDelay
			}

			if verbose {
				fmt.Printf("Waiting %v before retry attempt %d/%d...\n", delay, attempt+1, maxRetries)
			} else if attempt == 1 {
				// Show spinner only starting from second attempt to avoid flickering
				spinner := console.NewSpinner("Waiting for workflow run to appear...")
				spinner.Start()
				time.Sleep(delay)
				spinner.Stop()
				continue
			}
			time.Sleep(delay)
		}

		// Build command with optional repo parameter
		var cmd *exec.Cmd
		if repo != "" {
			cmd = exec.Command("gh", "run", "list", "--repo", repo, "--workflow", lockFileName, "--limit", "1", "--json", "url,databaseId,status,conclusion,createdAt")
		} else {
			cmd = exec.Command("gh", "run", "list", "--workflow", lockFileName, "--limit", "1", "--json", "url,databaseId,status,conclusion,createdAt")
		}

		output, err := cmd.Output()
		if err != nil {
			lastErr = fmt.Errorf("failed to get workflow runs: %w", err)
			if verbose {
				fmt.Printf("Attempt %d/%d failed: %v\n", attempt+1, maxRetries, err)
			}
			continue
		}

		if len(output) == 0 || string(output) == "[]" {
			lastErr = fmt.Errorf("no runs found for workflow")
			if verbose {
				fmt.Printf("Attempt %d/%d: no runs found yet\n", attempt+1, maxRetries)
			}
			continue
		}

		// Parse the JSON output
		var runs []struct {
			URL        string `json:"url"`
			DatabaseID int64  `json:"databaseId"`
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
			CreatedAt  string `json:"createdAt"`
		}

		if err := json.Unmarshal(output, &runs); err != nil {
			lastErr = fmt.Errorf("failed to parse workflow run data: %w", err)
			if verbose {
				fmt.Printf("Attempt %d/%d failed to parse JSON: %v\n", attempt+1, maxRetries, err)
			}
			continue
		}

		if len(runs) == 0 {
			lastErr = fmt.Errorf("no runs found")
			if verbose {
				fmt.Printf("Attempt %d/%d: no runs in parsed JSON\n", attempt+1, maxRetries)
			}
			continue
		}

		run := runs[0]

		// Parse the creation timestamp
		var createdAt time.Time
		if run.CreatedAt != "" {
			if parsedTime, err := time.Parse(time.RFC3339, run.CreatedAt); err == nil {
				createdAt = parsedTime
			} else if verbose {
				fmt.Printf("Warning: Could not parse creation time '%s': %v\n", run.CreatedAt, err)
			}
		}

		runInfo := &WorkflowRunInfo{
			URL:        run.URL,
			DatabaseID: run.DatabaseID,
			Status:     run.Status,
			Conclusion: run.Conclusion,
			CreatedAt:  createdAt,
		}

		// If we found a run and it was created after we started (within 30 seconds tolerance),
		// it's likely the run we just triggered
		if !createdAt.IsZero() && createdAt.After(startTime.Add(-30*time.Second)) {
			if verbose {
				fmt.Printf("Found recent run (ID: %d) created at %v (started polling at %v)\n",
					run.DatabaseID, createdAt.Format(time.RFC3339), startTime.Format(time.RFC3339))
			}
			return runInfo, nil
		}

		if verbose {
			if createdAt.IsZero() {
				fmt.Printf("Attempt %d/%d: Found run (ID: %d) but no creation timestamp available\n", attempt+1, maxRetries, run.DatabaseID)
			} else {
				fmt.Printf("Attempt %d/%d: Found run (ID: %d) but it was created at %v (too old)\n",
					attempt+1, maxRetries, run.DatabaseID, createdAt.Format(time.RFC3339))
			}
		}

		// For the first few attempts, if we have a run but it's too old, keep trying
		if attempt < 3 {
			lastErr = fmt.Errorf("workflow run appears to be from a previous execution")
			continue
		}

		// For later attempts, return what we found even if timing is uncertain
		if verbose {
			fmt.Printf("Returning workflow run (ID: %d) after %d attempts (timing uncertain)\n", run.DatabaseID, attempt+1)
		}
		return runInfo, nil
	}

	// If we exhausted all retries, return the last error
	if lastErr != nil {
		return nil, fmt.Errorf("failed to get workflow run after %d attempts: %w", maxRetries, lastErr)
	}

	return nil, fmt.Errorf("no workflow run found after %d attempts", maxRetries)
}
