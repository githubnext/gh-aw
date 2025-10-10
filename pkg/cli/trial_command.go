package cli

import (
	"encoding/json"
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

// WorkflowTrialResult represents the result of running a single workflow trial
type WorkflowTrialResult struct {
	WorkflowName        string                 `json:"workflow_name"`
	RunID               string                 `json:"run_id"`
	SafeOutputs         map[string]interface{} `json:"safe_outputs"`
	AgentStdioLogs      []string               `json:"agent_stdio_logs,omitempty"`
	AgenticRunInfo      map[string]interface{} `json:"agentic_run_info,omitempty"`
	AdditionalArtifacts map[string]interface{} `json:"additional_artifacts,omitempty"`
	Timestamp           time.Time              `json:"timestamp"`
}

// CombinedTrialResult represents the combined results of multiple workflow trials
type CombinedTrialResult struct {
	WorkflowNames []string              `json:"workflow_names"`
	Results       []WorkflowTrialResult `json:"results"`
	Timestamp     time.Time             `json:"timestamp"`
}

// NewTrialCommand creates the trial command
func NewTrialCommand(validateEngine func(string) error) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trial <owner/repo/workflow1> [owner/repo/workflow2...]",
		Short: "Trial one or more agentic workflows as if they were running in a repository",
		Long: `Trial one or more agentic workflows as if they were running in a repository.

This command creates a temporary private repository in your GitHub space, installs the specified
workflow(s) from their source repositories, and runs them in "trial mode" to capture safe outputs without
making actual changes to the "simulated" host repository

Single workflow:
  ` + constants.CLIExtensionPrefix + ` trial githubnext/agentics/weekly-research
  Outputs: stdout + local trials/weekly-research.DATETIME-ID.json + trial repo trials/

Multiple workflows (for comparison):
  ` + constants.CLIExtensionPrefix + ` trial githubnext/agentics/daily-plan githubnext/agentics/weekly-research
  Outputs: stdout + local trials/ + trial repo trials/ (individual + combined results)

Workflows from different repositories:
  ` + constants.CLIExtensionPrefix + ` trial githubnext/agentics/daily-plan myorg/myrepo/custom-workflow

Other examples:
  ` + constants.CLIExtensionPrefix + ` trial githubnext/agentics/my-workflow --delete-host-repo
  ` + constants.CLIExtensionPrefix + ` trial githubnext/agentics/my-workflow --quiet --host-repo my-custom-trial

All workflows must support workflow_dispatch trigger to be used in trial mode.
The host repository will be created as private and kept by default unless --delete-host-repo is specified.
Trial results are saved both locally (in trials/ directory) and in the host repository for future reference.`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			workflowSpecs := args
			simulatedHostRepoSlug, _ := cmd.Flags().GetString("simulated-host-repo")
			trialRepo, _ := cmd.Flags().GetString("host-repo")
			deleteRepo, _ := cmd.Flags().GetBool("delete-host-repo")
			yes, _ := cmd.Flags().GetBool("yes")
			timeout, _ := cmd.Flags().GetInt("timeout")
			verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")

			if err := RunWorkflowTrials(workflowSpecs, simulatedHostRepoSlug, trialRepo, deleteRepo, yes, timeout, verbose); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		},
	}

	// Add flags
	cmd.Flags().StringP("simulated-host-repo", "s", "", "The repo we're simulating the execution for, as if the workflow was installed in that repo (defaults to current repository)")

	// Get current username for default trial repo description
	username, _ := getCurrentGitHubUsername()
	defaultHostRepo := "gh-aw-trial"
	if username != "" {
		defaultHostRepo = fmt.Sprintf("%s/gh-aw-trial", username)
	}

	cmd.Flags().String("host-repo", "", fmt.Sprintf("Custom host repository slug (defaults to '%s')", defaultHostRepo))
	cmd.Flags().Bool("delete-host-repo", false, "Delete the host repository after completion (default: keep)")
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompts")
	cmd.Flags().Int("timeout", 30, "Timeout in minutes for workflow execution (default: 30)")

	return cmd
}

// RunWorkflowTrials executes the main logic for trialing one or more workflows
func RunWorkflowTrials(workflowSpecs []string, simulatedHostRepoSlug string, trialRepo string, deleteRepo, quiet bool, timeoutMinutes int, verbose bool) error {
	// Parse all workflow specifications
	var parsedSpecs []*WorkflowSpec
	for _, spec := range workflowSpecs {
		parsedSpec, err := parseWorkflowSpec(spec)
		if err != nil {
			return fmt.Errorf("invalid workflow specification '%s': %w", spec, err)
		}
		parsedSpecs = append(parsedSpecs, parsedSpec)
	}

	if len(parsedSpecs) == 1 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting trial of workflow '%s' from '%s'", parsedSpecs[0].WorkflowName, parsedSpecs[0].Repo)))
	} else {
		workflowNames := make([]string, len(parsedSpecs))
		for i, spec := range parsedSpecs {
			workflowNames[i] = spec.WorkflowName
		}
		joinedNames := strings.Join(workflowNames, ", ")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting trial of %d workflows (%s)", len(parsedSpecs), joinedNames)))
	}

	// Generate a unique datetime-ID for this trial session
	dateTimeID := fmt.Sprintf("%s-%d", time.Now().Format("20060102-150405"), time.Now().UnixNano()%1000000)

	// Step 0: Determine simulated host repository
	var finalSimulatedRepoSlug string
	if simulatedHostRepoSlug != "" {
		// Use the provided simulated host repository
		finalSimulatedRepoSlug = simulatedHostRepoSlug
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Target repository (specified): %s", finalSimulatedRepoSlug)))
	} else {
		// Fall back to current repository
		var err error
		finalSimulatedRepoSlug, err = getCurrentRepositoryInfo()
		if err != nil {
			return fmt.Errorf("failed to determine simulated host repository: %w", err)
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Target repository (current): %s", finalSimulatedRepoSlug)))
	}

	// Step 1: Determine host repository slug
	var trialRepoSlug string
	if trialRepo != "" {
		// User provided a custom trial repo (could be just name or full slug)
		if strings.Contains(trialRepo, "/") {
			// Full slug provided (user/repo)
			trialRepoSlug = trialRepo
		} else {
			// Just repo name provided, prepend current username
			username, err := getCurrentGitHubUsername()
			if err != nil {
				return fmt.Errorf("failed to get GitHub username for trial repo: %w", err)
			}
			trialRepoSlug = fmt.Sprintf("%s/%s", username, trialRepo)
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Trial repository (custom): %s", trialRepoSlug)))
	} else {
		// Use default trial repo with current username
		username, err := getCurrentGitHubUsername()
		if err != nil {
			return fmt.Errorf("failed to get GitHub username for default trial repo: %w", err)
		}
		trialRepoSlug = fmt.Sprintf("%s/gh-aw-trial", username)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Trial repository (default): %s", trialRepoSlug)))
	}

	// Step 1.5: Show confirmation unless quiet mode
	if !quiet {
		if err := showTrialConfirmation(parsedSpecs, finalSimulatedRepoSlug, trialRepoSlug, deleteRepo); err != nil {
			return err
		}
	}

	// Step 2: Create or reuse host repository
	if err := ensureTrialRepository(trialRepoSlug, verbose); err != nil {
		return fmt.Errorf("failed to ensure host repository: %w", err)
	}

	// Set up secret cleanup to always run on exit
	defer func() {
		if err := cleanupTrialSecrets(trialRepoSlug, verbose); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to cleanup secrets: %v", err)))
		}
	}()

	// Set up cleanup if requested
	if deleteRepo {
		defer func() {
			if err := cleanupTrialRepository(trialRepoSlug, verbose); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to cleanup host repository: %v", err)))
			}
		}()
	}

	// Step 3: Clone host repository to local temp directory
	tempDir, err := cloneTrialRepository(trialRepoSlug, verbose)
	if err != nil {
		return fmt.Errorf("failed to clone host repository: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to cleanup local temp directory: %v", err)))
		}
	}()

	// Step 4: Create trials directory
	if err := os.MkdirAll("trials", 0755); err != nil {
		return fmt.Errorf("failed to create trials directory: %w", err)
	}

	// Step 5: Run trials for each workflow
	var workflowResults []WorkflowTrialResult

	for i, parsedSpec := range parsedSpecs {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("=== Running trial for workflow: %s ===", parsedSpec.WorkflowName)))

		// Install workflow with trial mode compilation
		if err := installWorkflowInTrialMode(tempDir, parsedSpec, finalSimulatedRepoSlug, trialRepoSlug, verbose); err != nil {
			return fmt.Errorf("failed to install workflow '%s' in trial mode: %w", parsedSpec.WorkflowName, err)
		}

		// Add user's PAT as repository secret (only once)
		if i == 0 {
			if err := addGitHubTokenSecret(trialRepoSlug, verbose); err != nil {
				return fmt.Errorf("failed to add GitHub token secret: %w", err)
			}
		}

		// Run the workflow and wait for completion
		runID, err := triggerWorkflowRun(trialRepoSlug, parsedSpec.WorkflowName, verbose)
		if err != nil {
			return fmt.Errorf("failed to trigger workflow run for '%s': %w", parsedSpec.WorkflowName, err)
		}

		// Generate workflow run URL
		workflowRunURL := fmt.Sprintf("https://github.com/%s/actions/runs/%s", trialRepoSlug, runID)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow run started with ID: %s (%s)", runID, workflowRunURL)))

		// Wait for workflow completion
		if err := waitForWorkflowCompletion(trialRepoSlug, runID, timeoutMinutes, verbose); err != nil {
			return fmt.Errorf("workflow '%s' execution failed or timed out: %w", parsedSpec.WorkflowName, err)
		}

		// Download and process all artifacts
		artifacts, err := downloadAllArtifacts(trialRepoSlug, runID, verbose)
		if err != nil {
			return fmt.Errorf("failed to download artifacts for '%s': %w", parsedSpec.WorkflowName, err)
		}

		// Save individual workflow results
		result := WorkflowTrialResult{
			WorkflowName:        parsedSpec.WorkflowName,
			RunID:               runID,
			SafeOutputs:         artifacts.SafeOutputs,
			AgentStdioLogs:      artifacts.AgentStdioLogs,
			AgenticRunInfo:      artifacts.AgenticRunInfo,
			AdditionalArtifacts: artifacts.AdditionalArtifacts,
			Timestamp:           time.Now(),
		}
		workflowResults = append(workflowResults, result)

		// Save individual trial file
		sanitizedTargetRepo := sanitizeRepoSlugForFilename(finalSimulatedRepoSlug)
		individualFilename := fmt.Sprintf("trials/%s-%s.%s.json", parsedSpec.WorkflowName, sanitizedTargetRepo, dateTimeID)
		if err := saveTrialResult(individualFilename, result, verbose); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to save individual trial result: %v", err)))
		}

		// Display safe outputs to stdout
		if len(artifacts.SafeOutputs) > 0 {
			outputBytes, _ := json.MarshalIndent(artifacts.SafeOutputs, "", "  ")
			fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("=== Safe Outputs from %s ===", parsedSpec.WorkflowName)))
			fmt.Println(string(outputBytes))
			fmt.Println(console.FormatSuccessMessage("=== End of Safe Outputs ==="))
		} else {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("=== No Safe Outputs Generated by %s ===", parsedSpec.WorkflowName)))
		}

		// Display additional artifact information if available
		if len(artifacts.AgentStdioLogs) > 0 {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("=== Agent Stdio Logs Available from %s (%d files) ===", parsedSpec.WorkflowName, len(artifacts.AgentStdioLogs))))
		}
		if len(artifacts.AgenticRunInfo) > 0 {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("=== Agentic Run Information Available from %s ===", parsedSpec.WorkflowName)))
		}
		if len(artifacts.AdditionalArtifacts) > 0 {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("=== Additional Artifacts Available from %s (%d files) ===", parsedSpec.WorkflowName, len(artifacts.AdditionalArtifacts))))
		}

		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Trial completed for workflow: %s", parsedSpec.WorkflowName)))
	}

	// Step 6: Save combined results for multi-workflow trials
	if len(parsedSpecs) > 1 {
		workflowNames := make([]string, len(parsedSpecs))
		for i, spec := range parsedSpecs {
			workflowNames[i] = spec.WorkflowName
		}
		workflowNamesStr := strings.Join(workflowNames, "-")
		sanitizedTargetRepo := sanitizeRepoSlugForFilename(finalSimulatedRepoSlug)
		combinedFilename := fmt.Sprintf("trials/%s-%s.%s.json", workflowNamesStr, sanitizedTargetRepo, dateTimeID)
		combinedResult := CombinedTrialResult{
			WorkflowNames: workflowNames,
			Results:       workflowResults,
			Timestamp:     time.Now(),
		}
		if err := saveTrialResult(combinedFilename, combinedResult, verbose); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to save combined trial result: %v", err)))
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Combined results saved to: %s", combinedFilename)))
	}

	// Step 6.5: Copy trial results to host repository and commit them
	workflowNames := make([]string, len(parsedSpecs))
	for i, spec := range parsedSpecs {
		workflowNames[i] = spec.WorkflowName
	}
	if err := copyTrialResultsToRepo(tempDir, dateTimeID, workflowNames, finalSimulatedRepoSlug, verbose); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to copy trial results to repository: %v", err)))
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("All trials completed successfully"))

	if deleteRepo {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Trial repository will be cleaned up"))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Trial repository preserved: https://github.com/%s", trialRepoSlug)))
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

// showTrialConfirmation displays a confirmation prompt to the user using parsed workflow specs
func showTrialConfirmation(parsedSpecs []*WorkflowSpec, simulatedHostRepoSlug, trialRepoSlug string, deleteRepo bool) error {
	trialRepoURL := fmt.Sprintf("https://github.com/%s", trialRepoSlug)

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("=== Trial Execution Plan ==="))
	if len(parsedSpecs) == 1 {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessage("Workflow: %s (from %s)\n"), parsedSpecs[0].WorkflowName, parsedSpecs[0].Repo)
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Workflows:"))
		for _, spec := range parsedSpecs {
			fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  - %s (from %s)\n"), spec.WorkflowName, spec.Repo)
		}
	}
	fmt.Fprintf(os.Stderr, console.FormatInfoMessage("Target Repository: %s\n"), simulatedHostRepoSlug)
	fmt.Fprintf(os.Stderr, console.FormatInfoMessage("Trial Repository: %s (%s)\n"), trialRepoSlug, trialRepoURL)

	if deleteRepo {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Repository Cleanup: Trial repository will be deleted after completion"))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Repository Cleanup: Trial repository will be preserved"))
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(""))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("This will:"))
	fmt.Fprintf(os.Stderr, console.FormatInfoMessage("1. Create a private host repository at %s\n"), trialRepoURL)
	fmt.Fprintf(os.Stderr, console.FormatInfoMessage("2. Install and compile the specified workflows in trial mode against %s\n"), simulatedHostRepoSlug)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("3. Execute each workflow and collect any safe outputs"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("4. Display the results from each workflow execution"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("5. Clean up API key secrets from the host repository"))
	if deleteRepo {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("6. Delete the host repository"))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("6. Preserve the host repository for inspection"))
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(""))

	fmt.Fprint(os.Stderr, console.FormatPromptMessage("Do you want to continue? [y/N]: "))

	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		response = "n" // Default to no on error
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" && response != "yes" {
		return fmt.Errorf("trial cancelled by user")
	}

	return nil
}

// ensureTrialRepository creates a host repository if it doesn't exist, or reuses existing one
func ensureTrialRepository(repoSlug string, verbose bool) error {
	// repoSlug is always in user/repo format by the time it reaches this function
	fullRepoName := repoSlug
	parts := strings.Split(repoSlug, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository slug format: %s (expected user/repo)", repoSlug)
	}

	// Check if repository already exists
	cmd := exec.Command("gh", "repo", "view", fullRepoName)
	if err := cmd.Run(); err == nil {
		// Repository exists, reuse it
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Reusing existing host repository: %s", fullRepoName)))
		}
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Using existing host repository: https://github.com/%s", fullRepoName)))
		return nil
	}

	// Repository doesn't exist, create it
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Creating private host repository: %s", fullRepoName)))
	}

	// Use gh CLI to create private repo with initial README using full OWNER/REPO format
	cmd = exec.Command("gh", "repo", "create", fullRepoName, "--private", "--add-readme", "--description", "GitHub Agentic Workflows host repository")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("failed to create host repository: %w (output: %s)", err, string(output))
	}

	// Show host repository creation message with URL
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Created host repository: https://github.com/%s", fullRepoName)))

	// Give GitHub a moment to fully initialize the repository
	time.Sleep(2 * time.Second)

	return nil
}

func cleanupTrialRepository(repoSlug string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Cleaning up host repository: %s", repoSlug)))
	}

	// repoSlug is always in user/repo format by the time it reaches this function
	fullRepoName := repoSlug

	// Use gh CLI to delete the repository with proper username/repo format
	cmd := exec.Command("gh", "repo", "delete", fullRepoName, "--yes")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("failed to delete host repository: %w (output: %s)", err, string(output))
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Deleted host repository: %s", fullRepoName)))
	}

	return nil
}

func cloneTrialRepository(repoSlug string, verbose bool) (string, error) {
	// Create temporary directory
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("gh-aw-trial-%x", time.Now().UnixNano()))

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Cloning host repository to: %s", tempDir)))
	}

	// Clone the repository using the full slug
	repoURL := fmt.Sprintf("https://github.com/%s.git", repoSlug)
	cmd := exec.Command("git", "clone", repoURL, tempDir)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("failed to clone host repository %s: %w (output: %s)", repoURL, err, string(output))
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Cloned host repository to: %s", tempDir)))
	}

	return tempDir, nil
}

// installWorkflowInTrialMode installs a workflow in trial mode using a parsed spec
func installWorkflowInTrialMode(tempDir string, parsedSpec *WorkflowSpec, simulatedHostRepoSlug, trialRepoSlug string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Installing workflow '%s' from '%s' in trial mode", parsedSpec.WorkflowName, parsedSpec.Repo)))
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
	if err := InstallPackage(parsedSpec.Repo, verbose); err != nil {
		return fmt.Errorf("failed to install source repository: %w", err)
	}

	// Add the workflow from the installed package
	if err := AddWorkflows([]string{parsedSpec.String()}, 1, verbose, "", "", true, false); err != nil {
		return fmt.Errorf("failed to add workflow: %w", err)
	}

	// Now we need to modify the workflow for trial mode
	if err := modifyWorkflowForTrialMode(tempDir, parsedSpec.WorkflowName, simulatedHostRepoSlug, verbose); err != nil {
		return fmt.Errorf("failed to modify workflow for trial mode: %w", err)
	}

	// Compile the workflow with trial modifications
	config := CompileConfig{
		MarkdownFiles:     []string{".github/workflows/" + parsedSpec.WorkflowName + ".md"},
		Verbose:           verbose,
		EngineOverride:    "",
		Validate:          true,
		Watch:             false,
		WorkflowDir:       "",
		SkipInstructions:  false,
		NoEmit:            false,
		Purge:             false,
		TrialMode:         true,
		SimulatedRepoSlug: simulatedHostRepoSlug,
	}
	workflowDataList, err := CompileWorkflows(config)
	if err != nil {
		return fmt.Errorf("failed to compile workflow: %w", err)
	}
	if len(workflowDataList) != 1 {
		return fmt.Errorf("expected one compiled workflow, got %d", len(workflowDataList))
	}
	workflowData := workflowDataList[0]

	// Determine required engine secret from workflow data
	if err := determineAndAddEngineSecret(workflowData, trialRepoSlug, verbose); err != nil {
		return fmt.Errorf("failed to determine engine secret: %w", err)
	}

	// Commit and push the changes
	if err := commitAndPushWorkflow(tempDir, parsedSpec.WorkflowName, verbose); err != nil {
		return fmt.Errorf("failed to commit and push workflow: %w", err)
	}

	return nil
}

func addGitHubTokenSecret(repoSlug string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Adding GitHub token as repository secret"))
	}

	// Get the current auth token using the proper helper
	token, err := parser.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("failed to get GitHub auth token: %w", err)
	}

	// Use the repository slug directly
	fullRepoName := repoSlug

	// Add the token as a repository secret
	cmd := exec.Command("gh", "secret", "set", "GH_AW_GITHUB_TOKEN", "--repo", fullRepoName, "--body", token)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("failed to set repository secret: %w (output: %s)", err, string(output))
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Added GH_AW_GITHUB_TOKEN secret to host repository"))
	}

	return nil
}

func triggerWorkflowRun(repoSlug, workflowName string, verbose bool) (string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Triggering workflow run for: %s", workflowName)))
	}

	// Trigger workflow using gh CLI
	lockFileName := fmt.Sprintf("%s.lock.yml", workflowName)
	cmd := exec.Command("gh", "workflow", "run", lockFileName, "--repo", repoSlug)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("failed to trigger workflow run: %w (output: %s)", err, string(output))
	}

	// Get the most recent run ID for this workflow using shared retry logic
	runInfo, err := getLatestWorkflowRunWithRetry(lockFileName, repoSlug, verbose)
	if err != nil {
		return "", fmt.Errorf("failed to get workflow run ID: %w", err)
	}

	runID := fmt.Sprintf("%d", runInfo.DatabaseID)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Workflow run started with ID: %s (status: %s)", runID, runInfo.Status)))
	}

	return runID, nil
}

func waitForWorkflowCompletion(repoSlug, runID string, timeoutMinutes int, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Waiting for workflow completion (timeout: %d minutes)", timeoutMinutes)))
	}

	// Use the repository slug directly
	fullRepoName := repoSlug

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

// determineAndAddEngineSecret determines and sets the appropriate engine secret based on workflow configuration
func determineAndAddEngineSecret(workflowData *workflow.WorkflowData, trialRepoSlug string, verbose bool) error {
	var engineType string

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Determining required engine secret for workflow"))
	}

	// Find the matching workflow and determine its engine
	// Check both the original filename-based name and the processed display name
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Found matching workflow: %s", workflowData.Name)))
	}
	// Check if engine is specified in the EngineConfig
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.ID != "" {
		engineType = workflowData.EngineConfig.ID
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Found engine in EngineConfig: %s", engineType)))
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
		return addEngineSecret("ANTHROPIC_API_KEY", trialRepoSlug, verbose)
	case "codex", "openai":
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Setting OPENAI_API_KEY secret for OpenAI engine"))
		}
		return addEngineSecret("OPENAI_API_KEY", trialRepoSlug, verbose)
	case "copilot":
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Setting COPILOT_CLI_TOKEN secret for Copilot engine"))
		}
		return addEngineSecret("COPILOT_CLI_TOKEN", trialRepoSlug, verbose)
	default:
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Unknown engine type '%s', defaulting to Copilot", engineType)))
		}
		return addEngineSecret("COPILOT_CLI_TOKEN", trialRepoSlug, verbose)
	}
}

// addEngineSecret adds an engine-specific secret to the repository
func addEngineSecret(secretName, trialRepoSlug string, verbose bool) error {
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

	// Use the repository slug directly (should already be in user/repo format)
	fullRepoName := trialRepoSlug

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
func modifyWorkflowForTrialMode(tempDir, workflowName, simulatedHostRepoSlug string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Modifying workflow for trial mode"))
	}

	// Find the workflow markdown file
	workflowPath := filepath.Join(tempDir, constants.GetWorkflowDir(), fmt.Sprintf("%s.md", workflowName))

	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Replace repository references in the content
	modifiedContent := string(content)

	// Replace github.repository references to point to simulated host repo
	modifiedContent = strings.ReplaceAll(modifiedContent, "${{ github.repository }}", simulatedHostRepoSlug)

	// Also replace any hardcoded checkout actions to use the simulated host repo
	checkoutPattern := regexp.MustCompile(`uses: actions/checkout@[^\s]*`)
	modifiedContent = checkoutPattern.ReplaceAllStringFunc(modifiedContent, func(match string) string {
		return fmt.Sprintf("%s\n        with:\n          repository: %s", match, simulatedHostRepoSlug)
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
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Committing workflow and lock files to host repository"))

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
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Workflow and lock files are up to date in host repository"))
		return nil
	}

	// Commit changes
	commitMsg := fmt.Sprintf("Add trial workflow: %s and compiled lock files", workflowName)
	cmd = exec.Command("git", "commit", "-m", commitMsg)
	cmd.Dir = tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit changes: %w (output: %s)", err, string(output))
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Pulling latest changes from main branch"))
	}
	cmd = exec.Command("git", "pull", "origin", "main")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to pull latest changes: %w (output: %s)", err, string(output))
	}

	// Push to main
	cmd = exec.Command("git", "push", "origin", "main")
	cmd.Dir = tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push changes: %w (output: %s)", err, string(output))
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Workflow and lock files committed and pushed to host repository"))

	return nil
}

// cleanupTrialSecrets removes API key secrets from the host repository for security
func cleanupTrialSecrets(repoSlug string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Cleaning up API key secrets from host repository"))
	}

	// Use the repository slug directly
	fullRepoName := repoSlug

	// List of API key secrets to remove (keep GH_AW_GITHUB_TOKEN as it's needed for repository operations)
	secretsToRemove := []string{"GH_AW_GITHUB_TOKEN", "ANTHROPIC_API_KEY", "OPENAI_API_KEY", "COPILOT_CLI_TOKEN"}

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
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("API key secrets cleaned up from host repository"))
	}

	return nil
}

// TrialArtifacts represents all artifacts downloaded from a workflow run
type TrialArtifacts struct {
	SafeOutputs         map[string]interface{} `json:"safe_outputs"`
	AgentStdioLogs      []string               `json:"agent_stdio_logs,omitempty"`
	AgenticRunInfo      map[string]interface{} `json:"agentic_run_info,omitempty"`
	AdditionalArtifacts map[string]interface{} `json:"additional_artifacts,omitempty"`
}

// downloadAllArtifacts downloads and parses all available artifacts from a workflow run
func downloadAllArtifacts(trialRepoSlug, runID string, verbose bool) (*TrialArtifacts, error) {
	// Use the repository slug directly (should already be in user/repo format)
	fullRepoName := trialRepoSlug

	// Create temp directory for artifact download
	tempDir, err := os.MkdirTemp("", "trial-artifacts-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Download all artifacts for this run
	cmd := exec.Command("gh", "run", "download", runID, "--repo", fullRepoName, "--dir", tempDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If no artifacts exist, that's okay - some workflows don't generate artifacts
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("No artifacts found for run %s: %s", runID, string(output))))
		}
		return &TrialArtifacts{}, nil
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Downloaded all artifacts for run %s to %s", runID, tempDir)))
	}

	artifacts := &TrialArtifacts{
		AdditionalArtifacts: make(map[string]interface{}),
	}

	// Walk through all downloaded artifacts
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Get relative path from temp directory
		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to get relative path for %s: %v", path, err)))
			}
			return nil
		}

		// Handle specific artifact types
		switch {
		case strings.HasSuffix(path, "agent_output.json"):
			// Parse safe outputs
			if safeOutputs := parseJSONArtifact(path, verbose); safeOutputs != nil {
				artifacts.SafeOutputs = safeOutputs
			}

		case strings.HasSuffix(path, "aw_info.json"):
			// Parse agentic run information
			if runInfo := parseJSONArtifact(path, verbose); runInfo != nil {
				artifacts.AgenticRunInfo = runInfo
			}

		case strings.Contains(relPath, "agent") && strings.HasSuffix(path, ".log"):
			// Collect agent stdio logs
			if logContent := readTextArtifact(path, verbose); logContent != "" {
				artifacts.AgentStdioLogs = append(artifacts.AgentStdioLogs, logContent)
			}

		case strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".jsonl") || strings.HasSuffix(path, ".log") || strings.HasSuffix(path, ".txt"):
			// Handle other artifacts
			if strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".jsonl") {
				if content := parseJSONArtifact(path, verbose); content != nil {
					artifacts.AdditionalArtifacts[relPath] = content
				}
			} else {
				if content := readTextArtifact(path, verbose); content != "" {
					artifacts.AdditionalArtifacts[relPath] = content
				}
			}
		}

		return nil
	})

	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Error walking artifact directory: %v", err)))
		}
	}

	return artifacts, nil
}

// parseJSONArtifact parses a JSON artifact file and returns the parsed content
func parseJSONArtifact(filePath string, verbose bool) map[string]interface{} {
	content, err := os.ReadFile(filePath)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to read JSON artifact %s: %v", filePath, err)))
		}
		return nil
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(content, &parsed); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to parse JSON artifact %s: %v", filePath, err)))
		}
		return nil
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Parsed JSON artifact: %s", filepath.Base(filePath))))
	}

	return parsed
}

// readTextArtifact reads a text artifact file and returns its content
func readTextArtifact(filePath string, verbose bool) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to read text artifact %s: %v", filePath, err)))
		}
		return ""
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Read text artifact: %s (%d bytes)", filepath.Base(filePath), len(content))))
	}

	return string(content)
}

// saveTrialResult saves a trial result to a JSON file
func saveTrialResult(filename string, result interface{}, verbose bool) error {
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	if err := os.WriteFile(filename, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write result file: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Saved trial result to: %s", filename)))
	}

	return nil
}

// copyTrialResultsToRepo copies trial result files to the host repository and commits them
func copyTrialResultsToRepo(tempDir, dateTimeID string, workflowNames []string, simulatedHostRepoSlug string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Copying trial results to host repository"))
	}

	// Create trials directory in the host repository
	trialsDir := filepath.Join(tempDir, "trials")
	if err := os.MkdirAll(trialsDir, 0755); err != nil {
		return fmt.Errorf("failed to create trials directory in repository: %w", err)
	}

	// Copy individual workflow result files
	sanitizedTargetRepo := sanitizeRepoSlugForFilename(simulatedHostRepoSlug)
	for _, workflowName := range workflowNames {
		sourceFile := fmt.Sprintf("trials/%s-%s.%s.json", workflowName, sanitizedTargetRepo, dateTimeID)
		destFile := filepath.Join(trialsDir, fmt.Sprintf("%s-%s.%s.json", workflowName, sanitizedTargetRepo, dateTimeID))

		if err := copyFile(sourceFile, destFile); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to copy %s: %v", sourceFile, err)))
			}
			continue
		}

		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Copied %s to repository", sourceFile)))
		}
	}

	// Copy combined results file if it exists (for multi-workflow trials)
	if len(workflowNames) > 1 {
		workflowNamesStr := strings.Join(workflowNames, "-")
		combinedSourceFile := fmt.Sprintf("trials/%s-%s.%s.json", workflowNamesStr, sanitizedTargetRepo, dateTimeID)
		combinedDestFile := filepath.Join(trialsDir, fmt.Sprintf("%s-%s.%s.json", workflowNamesStr, sanitizedTargetRepo, dateTimeID))

		if err := copyFile(combinedSourceFile, combinedDestFile); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to copy combined results: %v", err)))
			}
		} else if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Copied %s to repository", combinedSourceFile)))
		}
	}

	// Change to temp directory to commit the changes
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		return fmt.Errorf("failed to change to temp directory: %w", err)
	}

	// Add trial results to git
	cmd := exec.Command("git", "add", "trials/")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add trial results: %w (output: %s)", err, string(output))
	}

	// Check if there are any changes to commit
	statusCmd := exec.Command("git", "status", "--porcelain", "trials/")
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	// If no changes, skip commit and push
	if len(strings.TrimSpace(string(statusOutput))) == 0 {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No new trial results to commit"))
		}
		return nil
	}

	// Commit trial results
	commitMsg := fmt.Sprintf("Add trial results for %s (%s)", strings.Join(workflowNames, ", "), dateTimeID)
	cmd = exec.Command("git", "commit", "-m", commitMsg)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit trial results: %w (output: %s)", err, string(output))
	}

	// Pull latest changes from main before pushing to avoid conflicts
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Pulling latest changes from main branch"))
	}
	cmd = exec.Command("git", "pull", "origin", "main")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to pull latest changes: %w (output: %s)", err, string(output))
	}

	// Push to main
	cmd = exec.Command("git", "push", "origin", "main")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push trial results: %w (output: %s)", err, string(output))
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Trial results copied to repository and pushed"))

	return nil
}

// sanitizeRepoSlugForFilename converts a repository slug (owner/repo) to a filename-safe string
func sanitizeRepoSlugForFilename(repoSlug string) string {
	return strings.ReplaceAll(repoSlug, "/", "-")
}

// copyFile copies a file from source to destination
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}
