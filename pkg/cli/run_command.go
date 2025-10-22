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
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
)

var runLog = logger.New("cli:run_command")

// RunWorkflowOnGitHub runs an agentic workflow on GitHub Actions
func RunWorkflowOnGitHub(workflowIdOrName string, enable bool, engineOverride string, repoOverride string, autoMergePRs bool, verbose bool) error {
	runLog.Printf("Starting workflow run: workflow=%s, enable=%v, engineOverride=%s, repo=%s", workflowIdOrName, enable, engineOverride, repoOverride)

	if workflowIdOrName == "" {
		return fmt.Errorf("workflow name or ID is required")
	}

	runLog.Printf("Running workflow on GitHub Actions: %s", workflowIdOrName)

	// Check if gh CLI is available
	if !isGHCLIAvailable() {
		return fmt.Errorf("GitHub CLI (gh) is required but not available")
	}

	// Validate workflow exists and is runnable
	if repoOverride != "" {
		runLog.Printf("Validating remote workflow: %s in repo %s", workflowIdOrName, repoOverride)
		// For remote repositories, use remote validation
		if err := validateRemoteWorkflow(workflowIdOrName, repoOverride, verbose); err != nil {
			return fmt.Errorf("failed to validate remote workflow: %w", err)
		}
		// Note: We skip local runnable check for remote workflows as we assume they are properly configured
	} else {
		runLog.Printf("Validating local workflow: %s", workflowIdOrName)
		// For local workflows, use existing local validation
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
		runLog.Printf("Workflow is runnable: %s", workflowFile)
	}

	// Handle --enable flag logic: check workflow state and enable if needed
	var wasDisabled bool
	var workflowID int64
	if enable {
		// Get current workflow status
		workflow, err := getWorkflowStatus(workflowIdOrName, repoOverride, verbose)
		if err != nil {
			runLog.Printf("Warning: Could not check workflow status: %v", err)
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
				enableArgs := []string{"workflow", "enable", strconv.FormatInt(workflow.ID, 10)}
				if repoOverride != "" {
					enableArgs = append(enableArgs, "--repo", repoOverride)
				}
				cmd := exec.Command("gh", enableArgs...)
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("failed to enable workflow '%s': %w", workflowIdOrName, err)
				}
				fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Enabled workflow: %s", workflowIdOrName)))
			}
		}
	}

	// Determine the lock file name based on the workflow source
	var lockFileName string
	var lockFilePath string

	if repoOverride != "" {
		// For remote repositories, construct lock file name directly
		filename := strings.TrimSuffix(filepath.Base(workflowIdOrName), ".md")
		lockFileName = filename + ".lock.yml"
	} else {
		// For local workflows, validate the workflow exists locally
		workflowsDir := getWorkflowsDir()

		_, _, err := readWorkflowFile(workflowIdOrName+".md", workflowsDir)
		if err != nil {
			return fmt.Errorf("failed to find workflow in local .github/workflows or components: %w", err)
		}

		// For local workflows, use the simple filename
		filename := strings.TrimSuffix(filepath.Base(workflowIdOrName), ".md")
		lockFileName = filename + ".lock.yml"

		// Check if the lock file exists in .github/workflows
		lockFilePath = filepath.Join(".github/workflows", lockFileName)
		if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
			return fmt.Errorf("workflow lock file '%s' not found in .github/workflows - run '"+constants.CLIExtensionPrefix+" compile' first", lockFileName)
		}
	}

	// Recompile workflow if engine override is provided (only for local workflows)
	if engineOverride != "" && repoOverride == "" {
		runLog.Printf("Recompiling workflow with engine override: %s", engineOverride)

		workflowMarkdownPath := strings.TrimSuffix(lockFilePath, ".lock.yml") + ".md"
		config := CompileConfig{
			MarkdownFiles:        []string{workflowMarkdownPath},
			Verbose:              verbose,
			EngineOverride:       engineOverride,
			Validate:             true,
			Watch:                false,
			WorkflowDir:          "",
			SkipInstructions:     false,
			NoEmit:               false,
			Purge:                false,
			TrialMode:            false,
			TrialLogicalRepoSlug: "",
			Strict:               false,
		}
		if _, err := CompileWorkflows(config); err != nil {
			return fmt.Errorf("failed to recompile workflow with engine override: %w", err)
		}

		runLog.Printf("Successfully recompiled workflow with engine: %s", engineOverride)
	} else if engineOverride != "" && repoOverride != "" {
		runLog.Printf("Note: Engine override ignored for remote repository workflows")
	}

	runLog.Printf("Using lock file: %s", lockFileName)

	// Build the gh workflow run command with optional repo override
	args := []string{"workflow", "run", lockFileName}
	if repoOverride != "" {
		args = append(args, "--repo", repoOverride)
	}

	// Record the start time for auto-merge PR filtering
	workflowStartTime := time.Now()

	// Execute gh workflow run command and capture output
	cmd := exec.Command("gh", args...)

	if repoOverride != "" {
		runLog.Printf("Executing: gh workflow run %s --repo %s", lockFileName, repoOverride)
	} else {
		runLog.Printf("Executing: gh workflow run %s", lockFileName)
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
			restoreWorkflowState(workflowIdOrName, workflowID, repoOverride, verbose)
		}

		return fmt.Errorf("failed to run workflow on GitHub Actions: %w", err)
	}

	// Display the output from gh workflow run
	output := strings.TrimSpace(string(stdout))
	if output != "" {
		fmt.Println(output)
	}

	fmt.Printf("Successfully triggered workflow: %s\n", lockFileName)
	runLog.Printf("Workflow triggered successfully: %s", lockFileName)

	// Try to get the latest run for this workflow to show a direct link
	// Add a delay to allow GitHub Actions time to register the new workflow run
	runInfo, runErr := getLatestWorkflowRunWithRetry(lockFileName, repoOverride, verbose)
	if runErr == nil && runInfo.URL != "" {
		fmt.Printf("\n🔗 View workflow run: %s\n", runInfo.URL)
		runLog.Printf("Workflow run URL: %s (ID: %d)", runInfo.URL, runInfo.DatabaseID)
	} else if runErr != nil {
		runLog.Printf("Note: Could not get workflow run URL: %v", runErr)
	}

	// Auto-merge PRs if requested and we have a valid run
	if autoMergePRs {
		if runErr != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not get workflow run information for auto-merge: %v", runErr)))
		} else {
			// Wait for workflow completion before attempting to auto-merge PRs
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Auto-merge PRs enabled - waiting for workflow completion..."))

			// Determine target repository: use repo override if provided, otherwise get current repo
			targetRepo := repoOverride
			if targetRepo == "" {
				if currentRepo, err := GetCurrentRepoSlug(); err != nil {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not determine target repository for auto-merge: %v", err)))
					targetRepo = ""
				} else {
					targetRepo = currentRepo
				}
			}

			if targetRepo != "" {
				runIDStr := fmt.Sprintf("%d", runInfo.DatabaseID)
				if err := WaitForWorkflowCompletion(targetRepo, runIDStr, 30, verbose); err != nil {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Workflow did not complete successfully, skipping auto-merge: %v", err)))
				} else {
					// Auto-merge PRs created after the workflow start time
					if err := AutoMergePullRequestsCreatedAfter(targetRepo, workflowStartTime, verbose); err != nil {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to auto-merge pull requests: %v", err)))
					}
				}
			}
		}
	}

	// Restore workflow state if it was disabled and we enabled it
	if enable && wasDisabled && workflowID != 0 {
		restoreWorkflowState(workflowIdOrName, workflowID, repoOverride, verbose)
	}

	return nil
}

// RunWorkflowsOnGitHub runs multiple agentic workflows on GitHub Actions, optionally repeating a specified number of times
func RunWorkflowsOnGitHub(workflowNames []string, repeatCount int, enable bool, engineOverride string, repoOverride string, autoMergePRs bool, verbose bool) error {
	if len(workflowNames) == 0 {
		return fmt.Errorf("at least one workflow name or ID is required")
	}

	// Validate all workflows exist and are runnable before starting
	for _, workflowName := range workflowNames {
		if workflowName == "" {
			return fmt.Errorf("workflow name cannot be empty")
		}

		// Validate workflow exists
		if repoOverride != "" {
			// For remote repositories, use remote validation
			if err := validateRemoteWorkflow(workflowName, repoOverride, verbose); err != nil {
				return fmt.Errorf("failed to validate remote workflow '%s': %w", workflowName, err)
			}
		} else {
			// For local workflows, use existing local validation
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
	}

	// Function to run all workflows once
	runAllWorkflows := func() error {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Running %d workflow(s)...", len(workflowNames))))

		for i, workflowName := range workflowNames {
			if len(workflowNames) > 1 {
				fmt.Println(console.FormatProgressMessage(fmt.Sprintf("Running workflow %d/%d: %s", i+1, len(workflowNames), workflowName)))
			}

			if err := RunWorkflowOnGitHub(workflowName, enable, engineOverride, repoOverride, autoMergePRs, verbose); err != nil {
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
		RepeatCount:   repeatCount,
		RepeatMessage: "Repeating workflow run",
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

	if repo != "" {
		runLog.Printf("Getting latest run for workflow: %s in repo: %s (with retry logic)", lockFileName, repo)
	} else {
		runLog.Printf("Getting latest run for workflow: %s (with retry logic)", lockFileName)
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

			runLog.Printf("Waiting %v before retry attempt %d/%d...", delay, attempt+1, maxRetries)
			if attempt == 1 {
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
			runLog.Printf("Attempt %d/%d failed: %v", attempt+1, maxRetries, err)
			continue
		}

		if len(output) == 0 || string(output) == "[]" {
			lastErr = fmt.Errorf("no runs found for workflow")
			runLog.Printf("Attempt %d/%d: no runs found yet", attempt+1, maxRetries)
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
			runLog.Printf("Attempt %d/%d failed to parse JSON: %v", attempt+1, maxRetries, err)
			continue
		}

		if len(runs) == 0 {
			lastErr = fmt.Errorf("no runs found")
			runLog.Printf("Attempt %d/%d: no runs in parsed JSON", attempt+1, maxRetries)
			continue
		}

		run := runs[0]

		// Parse the creation timestamp
		var createdAt time.Time
		if run.CreatedAt != "" {
			if parsedTime, err := time.Parse(time.RFC3339, run.CreatedAt); err == nil {
				createdAt = parsedTime
			} else {
				runLog.Printf("Warning: Could not parse creation time '%s': %v", run.CreatedAt, err)
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
			runLog.Printf("Found recent run (ID: %d) created at %v (started polling at %v)",
				run.DatabaseID, createdAt.Format(time.RFC3339), startTime.Format(time.RFC3339))
			return runInfo, nil
		}

		if createdAt.IsZero() {
			runLog.Printf("Attempt %d/%d: Found run (ID: %d) but no creation timestamp available", attempt+1, maxRetries, run.DatabaseID)
		} else {
			runLog.Printf("Attempt %d/%d: Found run (ID: %d) but it was created at %v (too old)",
				attempt+1, maxRetries, run.DatabaseID, createdAt.Format(time.RFC3339))
		}

		// For the first few attempts, if we have a run but it's too old, keep trying
		if attempt < 3 {
			lastErr = fmt.Errorf("workflow run appears to be from a previous execution")
			continue
		}

		// For later attempts, return what we found even if timing is uncertain
		runLog.Printf("Returning workflow run (ID: %d) after %d attempts (timing uncertain)", run.DatabaseID, attempt+1)
		return runInfo, nil
	}

	// If we exhausted all retries, return the last error
	if lastErr != nil {
		return nil, fmt.Errorf("failed to get workflow run after %d attempts: %w", maxRetries, lastErr)
	}

	return nil, fmt.Errorf("no workflow run found after %d attempts", maxRetries)
}

// validateRemoteWorkflow checks if a workflow exists in a remote repository and can be triggered
func validateRemoteWorkflow(workflowName string, repoOverride string, verbose bool) error {
	if repoOverride == "" {
		return fmt.Errorf("repository must be specified for remote workflow validation")
	}

	// Add .lock.yml extension if not present
	lockFileName := workflowName
	if !strings.HasSuffix(lockFileName, ".lock.yml") {
		lockFileName += ".lock.yml"
	}

	runLog.Printf("Checking if workflow '%s' exists in repository '%s'...", lockFileName, repoOverride)

	// Use gh CLI to list workflows in the target repository
	cmd := exec.Command("gh", "workflow", "list", "--repo", repoOverride, "--json", "name,path,state")
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("failed to list workflows in repository '%s': %s", repoOverride, string(exitError.Stderr))
		}
		return fmt.Errorf("failed to list workflows in repository '%s': %w", repoOverride, err)
	}

	// Parse the JSON response
	var workflows []struct {
		Name  string `json:"name"`
		Path  string `json:"path"`
		State string `json:"state"`
	}

	if err := json.Unmarshal(output, &workflows); err != nil {
		return fmt.Errorf("failed to parse workflow list response: %w", err)
	}

	// Look for the workflow by checking if the lock file path exists
	for _, workflow := range workflows {
		if strings.HasSuffix(workflow.Path, lockFileName) {
			runLog.Printf("Found workflow '%s' in repository (path: %s, state: %s)",
				workflow.Name, workflow.Path, workflow.State)
			return nil
		}
	}

	return fmt.Errorf("workflow '%s' not found in repository '%s'", lockFileName, repoOverride)
}
