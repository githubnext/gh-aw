package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

// NewTrialCommand creates the trial command
func NewTrialCommand(verbose bool, validateEngine func(string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trial <workflow> -r <source-org/source-repo>",
		Short: "Trial an agentic workflow from a source repository against the current target repository",
		Long: `Trial an agentic workflow from a source repository against the current target repository.

This command creates a temporary private repository in your GitHub space, installs the specified
workflow from the source repository, and runs it in "trial mode" to capture safe outputs without
making actual changes to the target repository.

Examples:
  ` + constants.CLIExtensionPrefix + ` trial weekly-research -r githubnext/agentics
  ` + constants.CLIExtensionPrefix + ` trial daily-backlog-burner -r githubnext/agentics
  ` + constants.CLIExtensionPrefix + ` trial my-workflow -r organization/repository --delete-repo
  ` + constants.CLIExtensionPrefix + ` trial my-workflow -r organization/repository --quiet

The workflow must support workflow_dispatch trigger to be used in trial mode.
The trial repository will be created as private and kept by default unless --delete-repo is specified.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			workflowName := args[0]
			sourceRepo, _ := cmd.Flags().GetString("repository")
			deleteRepo, _ := cmd.Flags().GetBool("delete-repo")
			quiet, _ := cmd.Flags().GetBool("quiet")
			timeout, _ := cmd.Flags().GetInt("timeout")

			if sourceRepo == "" {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage("source repository is required (-r flag)"))
				os.Exit(1)
			}

			if err := RunWorkflowTrial(workflowName, sourceRepo, deleteRepo, quiet, timeout, verbose); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		},
	}

	// Add flags
	cmd.Flags().StringP("repository", "r", "", "Source repository containing the workflow (required)")
	cmd.Flags().Bool("delete-repo", false, "Delete the trial repository after completion (default: keep)")
	cmd.Flags().BoolP("quiet", "q", false, "Skip confirmation prompts")
	cmd.Flags().Int("timeout", 30, "Timeout in minutes for workflow execution (default: 30)")

	// Mark the repository flag as required
	if err := cmd.MarkFlagRequired("repository"); err != nil {
		// This should never happen in practice, but we need to handle the error for linting
		panic(fmt.Sprintf("Failed to mark repository flag as required: %v", err))
	}

	return cmd
}

// RunWorkflowTrial executes the main logic for trialing a workflow
func RunWorkflowTrial(workflowName, sourceRepo string, deleteRepo, quiet bool, timeoutMinutes int, verbose bool) error {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting trial of workflow '%s' from '%s'", workflowName, sourceRepo)))

	// Step 0: Determine current target repository
	targetRepo, err := getCurrentRepositoryInfo()
	if err != nil {
		return fmt.Errorf("failed to determine target repository: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Target repository: %s", targetRepo)))

	// Step 1: Use fixed trial repository name
	trialRepoName := "gh-aw-trial"
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Trial repository: %s", trialRepoName)))

	// Step 1.5: Show confirmation unless quiet mode
	if !quiet {
		if err := showTrialConfirmation(workflowName, sourceRepo, targetRepo, trialRepoName, deleteRepo); err != nil {
			return err
		}
	}

	// Step 2: Create or reuse trial repository
	if err := ensureTrialRepository(trialRepoName, verbose); err != nil {
		return fmt.Errorf("failed to ensure trial repository: %w", err)
	}

	// Set up cleanup if requested
	if deleteRepo {
		defer func() {
			if err := cleanupTrialRepository(trialRepoName, verbose); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to cleanup trial repository: %v", err)))
			}
		}()
	}

	// Step 3: Clone trial repository to local temp directory
	tempDir, err := cloneTrialRepository(trialRepoName, verbose)
	if err != nil {
		return fmt.Errorf("failed to clone trial repository: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to cleanup local temp directory: %v", err)))
		}
	}()

	// Step 4: Install workflow with trial mode compilation
	if err := installWorkflowInTrialMode(tempDir, workflowName, sourceRepo, targetRepo, trialRepoName, verbose); err != nil {
		return fmt.Errorf("failed to install workflow in trial mode: %w", err)
	}

	// Step 5: Add user's PAT as repository secret
	if err := addGitHubTokenSecret(trialRepoName, verbose); err != nil {
		return fmt.Errorf("failed to add GitHub token secret: %w", err)
	}

	// Step 6: Run the workflow and wait for completion
	runID, err := triggerWorkflowRun(trialRepoName, workflowName, verbose)
	if err != nil {
		return fmt.Errorf("failed to trigger workflow run: %w", err)
	}

	// Get username for workflow run URL
	username, err := getCurrentGitHubUsername()
	if err == nil {
		workflowRunURL := fmt.Sprintf("https://github.com/%s/%s/actions/runs/%s", username, trialRepoName, runID)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow run started with ID: %s (%s)", runID, workflowRunURL)))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow run started with ID: %s", runID)))
	}

	// Step 7: Wait for workflow completion
	if err := waitForWorkflowCompletion(trialRepoName, runID, timeoutMinutes, verbose); err != nil {
		// Clean up secrets even if workflow failed
		if cleanupErr := cleanupTrialSecrets(trialRepoName, verbose); cleanupErr != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to cleanup secrets: %v", cleanupErr)))
		}
		return fmt.Errorf("workflow execution failed or timed out: %w", err)
	}

	// Step 8: Download and display safe outputs
	if err := downloadAndDisplaySafeOutputs(trialRepoName, runID, verbose); err != nil {
		return fmt.Errorf("failed to download and display safe outputs: %w", err)
	}

	// Step 9: Clean up secrets
	if err := cleanupTrialSecrets(trialRepoName, verbose); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to cleanup secrets: %v", err)))
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Trial completed successfully"))

	if deleteRepo {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Trial repository will be cleaned up"))
	} else {
		// Get username for display purposes
		username, err := getCurrentGitHubUsername()
		if err == nil {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Trial repository preserved: https://github.com/%s/%s", username, trialRepoName)))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Trial repository preserved: %s", trialRepoName)))
		}
	}

	return nil
}

// getCurrentRepositoryInfo determines the current repository from git remote
func getCurrentRepositoryInfo() (string, error) {
	// Get git remote URL
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git remote origin: %w", err)
	}

	remoteURL := strings.TrimSpace(string(output))

	// Parse GitHub repository from remote URL
	// Handle both SSH and HTTPS formats
	var repoPath string

	// SSH format: git@github.com:owner/repo.git
	if strings.HasPrefix(remoteURL, "git@github.com:") {
		repoPath = strings.TrimPrefix(remoteURL, "git@github.com:")
	} else if strings.Contains(remoteURL, "github.com/") {
		// HTTPS format: https://github.com/owner/repo.git
		parts := strings.Split(remoteURL, "github.com/")
		if len(parts) >= 2 {
			repoPath = parts[1]
		}
	} else {
		return "", fmt.Errorf("remote URL does not appear to be a GitHub repository: %s", remoteURL)
	}

	// Remove .git suffix if present
	repoPath = strings.TrimSuffix(repoPath, ".git")

	// Validate format (should be owner/repo)
	parts := strings.Split(repoPath, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid repository format: %s", repoPath)
	}

	return repoPath, nil
}

// getCurrentGitHubUsername gets the current GitHub username from gh CLI
func getCurrentGitHubUsername() (string, error) {
	cmd := exec.Command("gh", "api", "user", "--jq", ".login")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub username: %w", err)
	}

	username := strings.TrimSpace(string(output))
	if username == "" {
		return "", fmt.Errorf("GitHub username is empty")
	}

	return username, nil
}

// showTrialConfirmation displays a confirmation prompt to the user
func showTrialConfirmation(workflowName, sourceRepo, targetRepo, trialRepoName string, deleteRepo bool) error {
	// Get username for trial repo URL display
	username, err := getCurrentGitHubUsername()
	if err != nil {
		return fmt.Errorf("failed to get GitHub username: %w", err)
	}

	trialRepoURL := fmt.Sprintf("https://github.com/%s/%s", username, trialRepoName)

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("=== Trial Execution Plan ==="))
	fmt.Fprintf(os.Stderr, console.FormatInfoMessage("Workflow: %s\n"), workflowName)
	fmt.Fprintf(os.Stderr, console.FormatInfoMessage("Source Repository: %s\n"), sourceRepo)
	fmt.Fprintf(os.Stderr, console.FormatInfoMessage("Target Repository: %s\n"), targetRepo)
	fmt.Fprintf(os.Stderr, console.FormatInfoMessage("Trial Repository: %s (%s)\n"), trialRepoName, trialRepoURL)

	if deleteRepo {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Repository Cleanup: Trial repository will be deleted after completion"))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Repository Cleanup: Trial repository will be preserved"))
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(""))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("This will:"))
	fmt.Fprintf(os.Stderr, console.FormatInfoMessage("1. Create a private trial repository at %s\n"), trialRepoURL)
	fmt.Fprintf(os.Stderr, console.FormatInfoMessage("2. Install and compile the specified workflow in trial mode against %s\n"), targetRepo)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("3. Execute the workflow and collect any safe outputs"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("4. Display the results from the workflow execution"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("5. Clean up API key secrets from the trial repository"))
	if deleteRepo {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("6. Delete the trial repository"))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("6. Preserve the trial repository for inspection"))
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(""))

	fmt.Fprint(os.Stderr, console.FormatPromptMessage("Do you want to continue? [y/N]: "))

	var response string
	_, err = fmt.Scanln(&response)
	if err != nil {
		// Handle EOF or other input errors as cancellation
		response = "n"
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" && response != "yes" {
		return fmt.Errorf("trial cancelled by user")
	}

	return nil
}

// ensureTrialRepository creates a trial repository if it doesn't exist, or reuses existing one
func ensureTrialRepository(repoName string, verbose bool) error {
	// Get current GitHub username
	username, err := getCurrentGitHubUsername()
	if err != nil {
		return fmt.Errorf("failed to get GitHub username: %w", err)
	}

	// Check if repository already exists
	fullRepoName := fmt.Sprintf("%s/%s", username, repoName)
	cmd := exec.Command("gh", "repo", "view", fullRepoName)
	if err := cmd.Run(); err == nil {
		// Repository exists, reuse it
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Reusing existing trial repository: %s", repoName)))
		}
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Using existing trial repository: https://github.com/%s/%s", username, repoName)))
		return nil
	}

	// Repository doesn't exist, create it
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Creating private trial repository: %s", repoName)))
	}

	// Use gh CLI to create private repo with initial README
	cmd = exec.Command("gh", "repo", "create", repoName, "--private", "--add-readme", "--description", "GitHub Agentic Workflows trial repository")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("failed to create trial repository: %w (output: %s)", err, string(output))
	}

	// Show trial repository creation message with URL
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Created trial repository: https://github.com/%s/%s", username, repoName)))

	// Give GitHub a moment to fully initialize the repository
	time.Sleep(2 * time.Second)

	return nil
}

func cleanupTrialRepository(repoName string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Cleaning up trial repository: %s", repoName)))
	}

	// Get current GitHub username
	username, err := getCurrentGitHubUsername()
	if err != nil {
		return fmt.Errorf("failed to get GitHub username for cleanup: %w", err)
	}

	// Use gh CLI to delete the repository with proper username/repo format
	fullRepoName := fmt.Sprintf("%s/%s", username, repoName)
	cmd := exec.Command("gh", "repo", "delete", fullRepoName, "--yes")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("failed to delete trial repository: %w (output: %s)", err, string(output))
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Deleted trial repository: %s", repoName)))
	}

	return nil
}

func cloneTrialRepository(repoName string, verbose bool) (string, error) {
	// Get current GitHub username
	username, err := getCurrentGitHubUsername()
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub username: %w", err)
	}

	// Create temporary directory
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("gh-aw-trial-%x", time.Now().UnixNano()))

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Cloning trial repository to: %s", tempDir)))
	}

	// Clone the repository with proper username/repo format
	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", username, repoName)
	cmd := exec.Command("git", "clone", repoURL, tempDir)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("failed to clone trial repository %s: %w (output: %s)", repoURL, err, string(output))
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Cloned trial repository to: %s", tempDir)))
	}

	return tempDir, nil
}

func installWorkflowInTrialMode(tempDir, workflowName, sourceRepo, targetRepo, trialRepoName string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Installing workflow '%s' from '%s' in trial mode", workflowName, sourceRepo)))
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		return fmt.Errorf("failed to change to temp directory: %w", err)
	}

	// Install the source repository as a package
	if err := InstallPackage(sourceRepo, true, verbose); err != nil {
		return fmt.Errorf("failed to install source repository: %w", err)
	}

	// Add the workflow from the installed package
	if err := AddWorkflows([]string{workflowName}, 1, verbose, "", sourceRepo, "", true, false); err != nil {
		return fmt.Errorf("failed to add workflow: %w", err)
	}

	// Now we need to modify the workflow for trial mode
	if err := modifyWorkflowForTrialMode(tempDir, workflowName, targetRepo, verbose); err != nil {
		return fmt.Errorf("failed to modify workflow for trial mode: %w", err)
	}

	// Compile the workflow with trial modifications
	workflowDataList, err := CompileWorkflows([]string{}, verbose, "", true, false, "", false, false, false, true)
	if err != nil {
		return fmt.Errorf("failed to compile workflow: %w", err)
	}

	// Determine required engine secret from workflow data
	if err := determineEngineSecret(workflowDataList, workflowName, trialRepoName, verbose); err != nil {
		return fmt.Errorf("failed to determine engine secret: %w", err)
	}

	// Commit and push the changes
	if err := commitAndPushWorkflow(tempDir, workflowName, verbose); err != nil {
		return fmt.Errorf("failed to commit and push workflow: %w", err)
	}

	return nil
}

func addGitHubTokenSecret(repoName string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Adding GitHub token as repository secret"))
	}

	// Get the current auth token using the proper helper
	token, err := parser.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("failed to get GitHub auth token: %w", err)
	}

	// Get current GitHub username
	username, err := getCurrentGitHubUsername()
	if err != nil {
		return fmt.Errorf("failed to get GitHub username: %w", err)
	}

	// Construct full repository name
	fullRepoName := fmt.Sprintf("%s/%s", username, repoName)

	// Add the token as a repository secret
	cmd := exec.Command("gh", "secret", "set", "GH_AW_GITHUB_TOKEN", "--repo", fullRepoName, "--body", token)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("failed to set repository secret: %w (output: %s)", err, string(output))
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Added GH_AW_GITHUB_TOKEN secret to trial repository"))
	}

	return nil
}

func triggerWorkflowRun(repoName, workflowName string, verbose bool) (string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Triggering workflow run for: %s", workflowName)))
	}

	// Get current GitHub username
	username, err := getCurrentGitHubUsername()
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub username: %w", err)
	}

	// Construct full repository name
	fullRepoName := fmt.Sprintf("%s/%s", username, repoName)

	// Trigger workflow using gh CLI
	lockFileName := fmt.Sprintf("%s.lock.yml", workflowName)
	cmd := exec.Command("gh", "workflow", "run", lockFileName, "--repo", fullRepoName)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("failed to trigger workflow run: %w (output: %s)", err, string(output))
	}

	// Get the most recent run ID for this workflow
	time.Sleep(2 * time.Second) // Wait for the run to be created

	cmd = exec.Command("gh", "run", "list", "--repo", fullRepoName, "--workflow", lockFileName, "--limit", "1", "--json", "databaseId")
	output, err = cmd.Output()

	if err != nil {
		return "", fmt.Errorf("failed to get workflow run ID: %w", err)
	}

	// Parse the JSON to extract run ID - simple regex since we know the format
	runIDRegex := regexp.MustCompile(`"databaseId":(\d+)`)
	matches := runIDRegex.FindStringSubmatch(string(output))

	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract run ID from output: %s", string(output))
	}

	runID := matches[1]

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Workflow run started with ID: %s", runID)))
	}

	return runID, nil
}

func waitForWorkflowCompletion(repoName, runID string, timeoutMinutes int, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Waiting for workflow completion (timeout: %d minutes)", timeoutMinutes)))
	}

	// Get current GitHub username
	username, err := getCurrentGitHubUsername()
	if err != nil {
		return fmt.Errorf("failed to get GitHub username: %w", err)
	}

	// Construct full repository name
	fullRepoName := fmt.Sprintf("%s/%s", username, repoName)

	timeout := time.Duration(timeoutMinutes) * time.Minute
	start := time.Now()

	for {
		// Check if timeout exceeded
		if time.Since(start) > timeout {
			return fmt.Errorf("workflow execution timed out after %d minutes", timeoutMinutes)
		}

		// Check workflow status
		cmd := exec.Command("gh", "run", "view", runID, "--repo", fullRepoName, "--json", "status,conclusion")
		output, err := cmd.Output()

		if err != nil {
			return fmt.Errorf("failed to check workflow status: %w", err)
		}

		status := string(output)

		// Check if completed
		if strings.Contains(status, `"status":"completed"`) {
			if strings.Contains(status, `"conclusion":"success"`) {
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Workflow completed successfully"))
				}
				return nil
			} else if strings.Contains(status, `"conclusion":"failure"`) {
				return fmt.Errorf("workflow failed")
			} else if strings.Contains(status, `"conclusion":"cancelled"`) {
				return fmt.Errorf("workflow was cancelled")
			} else {
				return fmt.Errorf("workflow completed with unknown conclusion")
			}
		}

		// Still running, wait before checking again
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatProgressMessage("Workflow still running..."))
		}
		time.Sleep(10 * time.Second)
	}
}

func downloadAndDisplaySafeOutputs(repoName, runID string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Downloading and displaying safe outputs"))
	}

	// Get current GitHub username
	username, err := getCurrentGitHubUsername()
	if err != nil {
		return fmt.Errorf("failed to get GitHub username: %w", err)
	}

	// Construct full repository name
	fullRepoName := fmt.Sprintf("%s/%s", username, repoName)

	// Download artifacts using gh CLI
	tempArtifactDir := filepath.Join(os.TempDir(), fmt.Sprintf("gh-aw-artifacts-%s", runID))
	defer os.RemoveAll(tempArtifactDir)

	cmd := exec.Command("gh", "run", "download", runID, "--repo", fullRepoName, "--dir", tempArtifactDir)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Artifacts might not exist for some workflows - this is okay
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("No artifacts found: %s", string(output))))
		}
		fmt.Println(console.FormatInfoMessage("No safe outputs artifacts were generated by this workflow"))
		return nil
	}

	// Look for agent_output.json in the downloaded artifacts
	var agentOutputPath string
	err = filepath.Walk(tempArtifactDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "agent_output.json" {
			agentOutputPath = path
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to search for agent outputs: %w", err)
	}

	if agentOutputPath == "" {
		fmt.Println(console.FormatInfoMessage("No safe outputs were generated by this workflow"))
		return nil
	}

	// Read and display the safe outputs
	content, err := os.ReadFile(agentOutputPath)
	if err != nil {
		return fmt.Errorf("failed to read agent outputs: %w", err)
	}

	fmt.Println(console.FormatSuccessMessage("=== Safe Outputs Generated by Workflow ==="))
	fmt.Println(string(content))
	fmt.Println(console.FormatSuccessMessage("=== End of Safe Outputs ==="))

	return nil
}

// determineEngineSecret determines and sets the appropriate engine secret based on workflow configuration
func determineEngineSecret(workflowDataList []*workflow.WorkflowData, workflowName, trialRepoName string, verbose bool) error {
	var engineType string

	// Generate the expected processed workflow name from the filename
	// Convert hyphens to spaces and capitalize each word
	processedWorkflowName := strings.ReplaceAll(workflowName, "-", " ")
	words := strings.Fields(processedWorkflowName)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	processedWorkflowName = strings.Join(words, " ")

	// Find the matching workflow and determine its engine
	for _, workflowData := range workflowDataList {
		// Check both the original filename-based name and the processed display name
		if workflowData.Name == workflowName || workflowData.Name == processedWorkflowName {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Found matching workflow: %s", workflowData.Name)))
			}
			// Check if engine is specified in the AI field (legacy)
			if workflowData.AI != "" {
				engineType = workflowData.AI
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Found engine in AI field: %s", engineType)))
				}
				break
			}
			// Check if engine is specified in the EngineConfig
			if workflowData.EngineConfig != nil && workflowData.EngineConfig.ID != "" {
				engineType = workflowData.EngineConfig.ID
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Found engine in EngineConfig: %s", engineType)))
				}
				break
			}
		}
	}

	// Default to copilot if no engine specified
	if engineType == "" {
		engineType = "copilot"
	}

	// Set the appropriate secret based on engine type
	switch engineType {
	case "claude":
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Setting ANTHROPIC_API_KEY secret for Claude engine"))
		}
		return addEngineSecret("ANTHROPIC_API_KEY", "anthropic_api_key", trialRepoName, verbose)
	case "codex", "openai":
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Setting OPENAI_API_KEY secret for OpenAI engine"))
		}
		return addEngineSecret("OPENAI_API_KEY", "openai_api_key", trialRepoName, verbose)
	case "copilot":
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Setting COPILOT_CLI_TOKEN secret for Copilot engine"))
		}
		return addEngineSecret("COPILOT_CLI_TOKEN", "copilot_cli_token", trialRepoName, verbose)
	default:
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Unknown engine type '%s', defaulting to Copilot", engineType)))
		}
		return addEngineSecret("COPILOT_CLI_TOKEN", "copilot_cli_token", trialRepoName, verbose)
	}
}

// addEngineSecret adds an engine-specific secret to the repository
func addEngineSecret(secretName, userSecretName, trialRepoName string, verbose bool) error {
	// First try to get the secret from environment variables
	secretValue := os.Getenv(secretName)
	if secretValue == "" {
		// Try common alternative environment variable names
		switch secretName {
		case "ANTHROPIC_API_KEY":
			secretValue = os.Getenv("ANTHROPIC_KEY")
		case "OPENAI_API_KEY":
			secretValue = os.Getenv("OPENAI_KEY")
		case "COPILOT_CLI_TOKEN":
			// Use the proper GitHub token helper that handles both env vars and gh CLI
			var err error
			secretValue, err = parser.GetGitHubToken()
			if err != nil {
				return fmt.Errorf("failed to get GitHub token for COPILOT_CLI_TOKEN: %w", err)
			}
		}
	}

	if secretValue == "" {
		return fmt.Errorf("environment variable %s not found. Please set it before running the trial", secretName)
	}

	// Get current GitHub username
	username, err := getCurrentGitHubUsername()
	if err != nil {
		return fmt.Errorf("failed to get GitHub username: %w", err)
	}

	// Construct full repository name
	fullRepoName := fmt.Sprintf("%s/%s", username, trialRepoName)

	// Add the secret to the repository
	addSecretCmd := exec.Command("gh", "secret", "set", secretName, "--repo", fullRepoName, "--body", secretValue)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Running: gh secret set %s --repo %s --body <redacted>", secretName, fullRepoName)))
	}

	if output, err := addSecretCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add %s secret: %w\nOutput: %s", secretName, err, string(output))
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully added %s secret", secretName)))
	}

	return nil
}

// modifyWorkflowForTrialMode modifies the workflow to work in trial mode
func modifyWorkflowForTrialMode(tempDir, workflowName, targetRepo string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Modifying workflow for trial mode"))
	}

	// Find the workflow markdown file
	workflowPath := filepath.Join(tempDir, ".github", "workflows", fmt.Sprintf("%s.md", workflowName))

	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Replace repository references in the content
	modifiedContent := string(content)

	// Replace github.repository references to point to target repo
	modifiedContent = strings.ReplaceAll(modifiedContent, "${{ github.repository }}", targetRepo)

	// Also replace any hardcoded checkout actions to use the target repo
	checkoutPattern := regexp.MustCompile(`uses: actions/checkout@[^\s]*`)
	modifiedContent = checkoutPattern.ReplaceAllStringFunc(modifiedContent, func(match string) string {
		return fmt.Sprintf("%s\n        with:\n          repository: %s", match, targetRepo)
	})

	// Write the modified content back
	if err := os.WriteFile(workflowPath, []byte(modifiedContent), 0644); err != nil {
		return fmt.Errorf("failed to write modified workflow: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Workflow modified for trial mode"))
	}

	return nil
}

// commitAndPushWorkflow commits and pushes the workflow changes
func commitAndPushWorkflow(tempDir, workflowName string, verbose bool) error {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Committing workflow and lock files to trial repository"))

	// Add all changes
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add changes: %w (output: %s)", err, string(output))
	}

	// Check if there are any changes to commit
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = tempDir
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	// If no changes, skip commit and push
	if len(strings.TrimSpace(string(statusOutput))) == 0 {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No changes detected, skipping commit"))
		}
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Workflow and lock files are up to date in trial repository"))
		return nil
	}

	// Commit changes
	commitMsg := fmt.Sprintf("Add trial workflow: %s and compiled lock files", workflowName)
	cmd = exec.Command("git", "commit", "-m", commitMsg)
	cmd.Dir = tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit changes: %w (output: %s)", err, string(output))
	}

	// Push to main
	cmd = exec.Command("git", "push", "origin", "main")
	cmd.Dir = tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push changes: %w (output: %s)", err, string(output))
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Workflow and lock files committed and pushed to trial repository"))

	return nil
}

// cleanupTrialSecrets removes API key secrets from the trial repository for security
func cleanupTrialSecrets(repoName string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Cleaning up API key secrets from trial repository"))
	}

	// Get current GitHub username
	username, err := getCurrentGitHubUsername()
	if err != nil {
		return fmt.Errorf("failed to get GitHub username: %w", err)
	}

	// Construct full repository name
	fullRepoName := fmt.Sprintf("%s/%s", username, repoName)

	// List of API key secrets to remove (keep GH_AW_GITHUB_TOKEN as it's needed for repository operations)
	secretsToRemove := []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "COPILOT_CLI_TOKEN"}

	for _, secretName := range secretsToRemove {
		cmd := exec.Command("gh", "secret", "delete", secretName, "--repo", fullRepoName)
		if output, err := cmd.CombinedOutput(); err != nil {
			// It's okay if the secret doesn't exist, just log in verbose mode
			if verbose && !strings.Contains(string(output), "Not Found") {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Could not delete secret %s: %s", secretName, string(output))))
			}
		} else if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Deleted secret: %s", secretName)))
		}
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("API key secrets cleaned up from trial repository"))
	}

	return nil
}
