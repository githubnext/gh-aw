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
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

var trialLog = logger.New("cli:trial_command")

// WorkflowTrialResult represents the result of running a single workflow trial
type WorkflowTrialResult struct {
	WorkflowName string                 `json:"workflow_name"`
	RunID        string                 `json:"run_id"`
	SafeOutputs  map[string]interface{} `json:"safe_outputs"`
	//AgentStdioLogs      []string               `json:"agent_stdio_logs,omitempty"`
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

// TrialSecretTracker tracks which secrets were added during a trial for cleanup
type TrialSecretTracker struct {
	RepoSlug     string          `json:"repo_slug"`
	AddedSecrets map[string]bool `json:"added_secrets"` // secrets that were successfully added by trial
}

// NewTrialSecretTracker creates a new secret tracker for a repository
func NewTrialSecretTracker(repoSlug string) *TrialSecretTracker {
	return &TrialSecretTracker{
		RepoSlug:     repoSlug,
		AddedSecrets: make(map[string]bool),
	}
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

Repository mode examples:
  ` + constants.CLIExtensionPrefix + ` trial githubnext/agentics/my-workflow --logical-repo myorg/myrepo  # Simulate running against myorg/myrepo
  ` + constants.CLIExtensionPrefix + ` trial githubnext/agentics/my-workflow --clone-repo myorg/myrepo   # Clone myorg/myrepo contents into host

Repeat and cleanup examples:
  ` + constants.CLIExtensionPrefix + ` trial githubnext/agentics/my-workflow --repeat 3                # Run 3 times total
  ` + constants.CLIExtensionPrefix + ` trial githubnext/agentics/my-workflow --delete-host-repo-after  # Delete repo after completion
  ` + constants.CLIExtensionPrefix + ` trial githubnext/agentics/my-workflow --quiet --host-repo my-trial # Custom host repo

Auto-merge examples:
  ` + constants.CLIExtensionPrefix + ` trial githubnext/agentics/my-workflow --auto-merge-prs          # Auto-merge any PRs created during trial

Advanced examples:
  ` + constants.CLIExtensionPrefix + ` trial githubnext/agentics/my-workflow --host-repo . # Use current repo as host
  ` + constants.CLIExtensionPrefix + ` trial ./local-workflow.md --clone-repo upstream/repo --repeat 2

Repository modes:
- Default: Simulate execution against current repository (using --logical-repo semantics)
- --logical-repo: Simulate execution against specified repository (github.repository points to that repo)
- --clone-repo: Clone specified repository contents into host repository (no simulation)

All workflows must support workflow_dispatch trigger to be used in trial mode.
The host repository will be created as private and kept by default unless --delete-host-repo-after is specified.
Trial results are saved both locally (in trials/ directory) and in the host repository for future reference.`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			workflowSpecs := args
			logicalRepoSpec, _ := cmd.Flags().GetString("logical-repo")
			cloneRepoSpec, _ := cmd.Flags().GetString("clone-repo")
			hostRepoSpec, _ := cmd.Flags().GetString("host-repo")
			deleteHostRepo, _ := cmd.Flags().GetBool("delete-host-repo-after")
			forceDeleteHostRepo, _ := cmd.Flags().GetBool("force-delete-host-repo-before")
			yes, _ := cmd.Flags().GetBool("yes")
			timeout, _ := cmd.Flags().GetInt("timeout")
			triggerContext, _ := cmd.Flags().GetString("trigger-context")
			repeatCount, _ := cmd.Flags().GetInt("repeat")
			autoMergePRs, _ := cmd.Flags().GetBool("auto-merge-prs")
			engineOverride, _ := cmd.Flags().GetString("engine")
			appendText, _ := cmd.Flags().GetString("append")
			verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")

			if err := validateEngine(engineOverride); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}

			if err := RunWorkflowTrials(workflowSpecs, logicalRepoSpec, cloneRepoSpec, hostRepoSpec, deleteHostRepo, forceDeleteHostRepo, yes, timeout, triggerContext, repeatCount, autoMergePRs, engineOverride, appendText, verbose); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
				os.Exit(1)
			}
		},
	}

	// Add flags
	cmd.Flags().StringP("logical-repo", "s", "", "The repo we're simulating the execution for, as if the workflow was installed in that repo (defaults to current repository)")
	cmd.Flags().String("clone-repo", "", "Alternative to --logical-repo: clone the contents of the specified repo into the host repo instead of using logical repository simulation")

	// Get current username for default trial repo description
	username, _ := getCurrentGitHubUsername()
	defaultHostRepo := "gh-aw-trial"
	if username != "" {
		defaultHostRepo = fmt.Sprintf("%s/gh-aw-trial", username)
	}

	cmd.Flags().String("host-repo", "", fmt.Sprintf("Custom host repository slug (defaults to '%s'). Use '.' for current repository", defaultHostRepo))
	cmd.Flags().Bool("delete-host-repo-after", false, "Delete the host repository after completion (default: keep)")
	cmd.Flags().Bool("force-delete-host-repo-before", false, "Force delete the host repository before creation, if it exists before creating it")
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompts")
	cmd.Flags().Int("timeout", 30, "Timeout in minutes for workflow execution (default: 30)")
	cmd.Flags().String("trigger-context", "", "Trigger context URL (e.g., GitHub issue URL) for issue-triggered workflows")
	cmd.Flags().Int("repeat", 0, "Number of times to repeat running workflows (0 = run once)")
	cmd.Flags().Bool("auto-merge-prs", false, "Auto-merge any pull requests created during the trial (requires --clone-repo)")
	cmd.Flags().StringP("engine", "a", "", "Override AI engine (claude, codex, copilot, custom)")
	cmd.Flags().String("append", "", "Append extra content to the end of agentic workflow on installation")

	return cmd
}

// RunWorkflowTrials executes the main logic for trialing one or more workflows
func RunWorkflowTrials(workflowSpecs []string, logicalRepoSpec string, cloneRepoSpec string, hostRepoSpec string, deleteHostRepo, forceDeleteHostRepo, quiet bool, timeoutMinutes int, triggerContext string, repeatCount int, autoMergePRs bool, engineOverride string, appendText string, verbose bool) error {
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
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting trial of workflow '%s' from '%s'", parsedSpecs[0].WorkflowName, parsedSpecs[0].RepoSlug)))
	} else {
		workflowNames := make([]string, len(parsedSpecs))
		for i, spec := range parsedSpecs {
			workflowNames[i] = spec.WorkflowName
		}
		joinedNames := strings.Join(workflowNames, ", ")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Starting trial of %d workflows (%s)", len(parsedSpecs), joinedNames)))
	}

	// Step 0: Validate mutually exclusive flags and determine workflow mode
	if logicalRepoSpec != "" && cloneRepoSpec != "" {
		return fmt.Errorf("--logical-repo and --clone-repo are mutually exclusive, please specify only one")
	}

	var logicalRepoSlug string
	var cloneRepoSlug string
	var cloneRepoVersion string

	if cloneRepoSpec != "" {
		// Use clone-repo mode: clone the specified repo contents into host repo
		cloneRepo, err := parseRepoSpec(cloneRepoSpec)
		if err != nil {
			return fmt.Errorf("invalid --clone-repo specification '%s': %w", cloneRepoSpec, err)
		}

		cloneRepoSlug = cloneRepo.RepoSlug
		cloneRepoVersion = cloneRepo.Version
		logicalRepoSlug = "" // Empty string means skip logical repo simulation
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Clone mode: Will clone contents from %s into host repository", cloneRepoSlug)))
	} else if logicalRepoSpec != "" {
		// Use logical-repo mode: simulate the workflow running against the specified repo
		logicalRepo, err := parseRepoSpec(logicalRepoSpec)

		if err != nil {
			return fmt.Errorf("invalid --logical-repo specification '%s': %w", logicalRepoSpec, err)
		}

		logicalRepoSlug = logicalRepo.RepoSlug
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Target repository (specified): %s", logicalRepoSlug)))
	} else {
		// Fall back to current repository for logical-repo mode
		var err error
		logicalRepoSlug, err = getCurrentRepositoryInfo()
		if err != nil {
			return fmt.Errorf("failed to determine simulated host repository: %w", err)
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Target repository (current): %s", logicalRepoSlug)))
	}

	// Step 1: Determine host repository slug
	var hostRepoSlug string
	if hostRepoSpec != "" {

		hostRepo, err := parseRepoSpec(hostRepoSpec)

		if err != nil {
			return fmt.Errorf("invalid --host-repo specification '%s': %w", hostRepoSpec, err)
		}
		hostRepoSlug = hostRepo.RepoSlug
		trialLog.Printf("Using specified host repository: %s", hostRepoSlug)
	} else {
		// Use default trial repo with current username
		username, err := getCurrentGitHubUsername()
		if err != nil {
			return fmt.Errorf("failed to get GitHub username for default trial repo: %w", err)
		}
		hostRepoSlug = fmt.Sprintf("%s/gh-aw-trial", username)
		trialLog.Printf("Using default host repository: %s", hostRepoSlug)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Host repository (default): %s", hostRepoSlug)))
	}

	// Step 1.5: Show confirmation unless quiet mode
	if !quiet {
		if err := showTrialConfirmation(parsedSpecs, logicalRepoSlug, cloneRepoSlug, hostRepoSlug, deleteHostRepo); err != nil {
			return err
		}
	}

	// Step 2: Create or reuse host repository
	trialLog.Printf("Ensuring trial repository exists: %s", hostRepoSlug)
	if err := ensureTrialRepository(hostRepoSlug, cloneRepoSlug, forceDeleteHostRepo, verbose); err != nil {
		return fmt.Errorf("failed to ensure host repository: %w", err)
	}

	// Step 2.5: Create secret tracker
	secretTracker := NewTrialSecretTracker(hostRepoSlug)
	trialLog.Print("Created secret tracker for trial")

	// Set up secret cleanup to always run on exit
	defer func() {
		if err := cleanupTrialSecrets(hostRepoSlug, secretTracker, verbose); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to cleanup secrets: %v", err)))
		}
	}()

	// Set up cleanup if requested
	if deleteHostRepo {
		defer func() {
			if err := cleanupTrialRepository(hostRepoSlug, verbose); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to cleanup host repository: %v", err)))
			}
		}()
	}

	// Step 2.7: Clone source repository contents if in clone-repo mode
	if cloneRepoSlug != "" {
		if err := cloneRepoContentsIntoHost(cloneRepoSlug, cloneRepoVersion, hostRepoSlug, verbose); err != nil {
			return fmt.Errorf("failed to clone repository contents: %w", err)
		}
	}

	// Function to run all trials once
	runAllTrials := func() error {
		// Generate a unique datetime-ID for this trial session
		dateTimeID := fmt.Sprintf("%s-%d", time.Now().Format("20060102-150405"), time.Now().UnixNano()%1000000)
		trialLog.Printf("Starting trial run: dateTimeID=%s", dateTimeID)

		// Step 3: Clone host repository to local temp directory
		trialLog.Printf("Cloning trial host repository: %s", hostRepoSlug)
		tempDir, err := cloneTrialHostRepository(hostRepoSlug, verbose)
		if err != nil {
			return fmt.Errorf("failed to clone host repository: %w", err)
		}
		trialLog.Printf("Cloned repository to: %s", tempDir)
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
			if err := installWorkflowInTrialMode(tempDir, parsedSpec, logicalRepoSlug, cloneRepoSlug, hostRepoSlug, secretTracker, engineOverride, appendText, verbose); err != nil {
				return fmt.Errorf("failed to install workflow '%s' in trial mode: %w", parsedSpec.WorkflowName, err)
			}

			// Add user's PAT as repository secret (only once)
			if i == 0 {
				if err := addGitHubTokenSecret(hostRepoSlug, secretTracker, verbose); err != nil {
					return fmt.Errorf("failed to add GitHub token secret: %w", err)
				}
			}

			// Run the workflow and wait for completion (with trigger context if provided)
			runID, err := triggerWorkflowRun(hostRepoSlug, parsedSpec.WorkflowName, triggerContext, verbose)
			if err != nil {
				return fmt.Errorf("failed to trigger workflow run for '%s': %w", parsedSpec.WorkflowName, err)
			}

			// Generate workflow run URL
			workflowRunURL := fmt.Sprintf("https://github.com/%s/actions/runs/%s", hostRepoSlug, runID)
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow run started with ID: %s (%s)", runID, workflowRunURL)))

			// Wait for workflow completion
			if err := WaitForWorkflowCompletion(hostRepoSlug, runID, timeoutMinutes, verbose); err != nil {
				return fmt.Errorf("workflow '%s' execution failed or timed out: %w", parsedSpec.WorkflowName, err)
			}

			// Auto-merge PRs if requested
			if autoMergePRs {
				if err := AutoMergePullRequestsLegacy(hostRepoSlug, verbose); err != nil {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to auto-merge pull requests: %v", err)))
				}
			}

			// Download and process all artifacts
			artifacts, err := downloadAllArtifacts(hostRepoSlug, runID, verbose)
			if err != nil {
				return fmt.Errorf("failed to download artifacts for '%s': %w", parsedSpec.WorkflowName, err)
			}

			// Save individual workflow results
			result := WorkflowTrialResult{
				WorkflowName: parsedSpec.WorkflowName,
				RunID:        runID,
				SafeOutputs:  artifacts.SafeOutputs,
				//AgentStdioLogs:      artifacts.AgentStdioLogs,
				AgenticRunInfo:      artifacts.AgenticRunInfo,
				AdditionalArtifacts: artifacts.AdditionalArtifacts,
				Timestamp:           time.Now(),
			}
			workflowResults = append(workflowResults, result)

			// Save individual trial file
			sanitizedTargetRepo := sanitizeRepoSlugForFilename(logicalRepoSlug)
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
			// if len(artifacts.AgentStdioLogs) > 0 {
			// 	fmt.Println(console.FormatInfoMessage(fmt.Sprintf("=== Agent Stdio Logs Available from %s (%d files) ===", parsedSpec.WorkflowName, len(artifacts.AgentStdioLogs))))
			// }
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
			sanitizedTargetRepo := sanitizeRepoSlugForFilename(logicalRepoSlug)
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
		if err := copyTrialResultsToHostRepo(tempDir, dateTimeID, workflowNames, logicalRepoSlug, verbose); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to copy trial results to repository: %v", err)))
		}

		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("All trials completed successfully"))
		return nil
	}

	// Execute trials with optional repeat functionality
	return ExecuteWithRepeat(RepeatOptions{
		RepeatCount:   repeatCount,
		RepeatMessage: "Repeating trial run",
		ExecuteFunc:   runAllTrials,
		CleanupFunc: func() {
			if deleteHostRepo {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Host repository will be cleaned up"))
			} else {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Host repository preserved: https://github.com/%s", hostRepoSlug)))
			}
		},
		UseStderr: true,
	})

}

// getCurrentRepositoryInfo determines the current repository from the gh CLI (cached)
// This is a wrapper around GetCurrentRepoSlug for backward compatibility
func getCurrentRepositoryInfo() (string, error) {
	return GetCurrentRepoSlug()
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
func showTrialConfirmation(parsedSpecs []*WorkflowSpec, logicalRepoSlug, cloneRepoSlug, hostRepoSlug string, deleteHostRepo bool) error {
	hostRepoSlugURL := fmt.Sprintf("https://github.com/%s", hostRepoSlug)

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("=== Trial Execution Plan ==="))
	if len(parsedSpecs) == 1 {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessage("Workflow: %s (from %s)\n"), parsedSpecs[0].WorkflowName, parsedSpecs[0].RepoSlug)
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Workflows:"))
		for _, spec := range parsedSpecs {
			fmt.Fprintf(os.Stderr, console.FormatInfoMessage("  - %s (from %s)\n"), spec.WorkflowName, spec.RepoSlug)
		}
	}

	// Display target repository info based on mode
	if cloneRepoSlug != "" {
		// Clone-repo mode
		fmt.Fprintf(os.Stderr, console.FormatInfoMessage("Source Repository: %s (will be cloned)\n"), cloneRepoSlug)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Mode: Clone repository contents into host repository"))
	} else {
		// Logical-repo mode
		fmt.Fprintf(os.Stderr, console.FormatInfoMessage("Target Repository: %s (simulated)\n"), logicalRepoSlug)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Mode: Simulate execution against target repository"))
	}

	fmt.Fprintf(os.Stderr, console.FormatInfoMessage("Trial Repository: %s (%s)\n"), hostRepoSlug, hostRepoSlugURL)

	if deleteHostRepo {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Repository Cleanup: Host repository will be deleted after completion"))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Repository Cleanup: Host repository will be preserved"))
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(""))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("This will:"))
	fmt.Fprintf(os.Stderr, console.FormatInfoMessage("1. Create a private host repository at %s\n"), hostRepoSlugURL)

	if cloneRepoSlug != "" {
		// Clone-repo mode steps
		fmt.Fprintf(os.Stderr, console.FormatInfoMessage("2. Clone contents from %s into the host repository\n"), cloneRepoSlug)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("3. Install and compile the specified workflows in the host repository"))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("4. Execute each workflow and collect any safe outputs"))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("5. Display the results from each workflow execution"))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("6. Clean up API key secrets from the host repository"))
	} else {
		// Logical-repo mode steps
		fmt.Fprintf(os.Stderr, console.FormatInfoMessage("2. Install and compile the specified workflows in trial mode against %s\n"), logicalRepoSlug)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("3. Execute each workflow and collect any safe outputs"))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("4. Display the results from each workflow execution"))
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("5. Clean up API key secrets from the host repository"))
	}

	// Final step (delete/preserve repository)
	stepNum := "6"
	if cloneRepoSlug != "" {
		stepNum = "7" // Clone mode has one extra step
	}

	if deleteHostRepo {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessage("%s. Delete the host repository\n"), stepNum)
	} else {
		fmt.Fprintf(os.Stderr, console.FormatInfoMessage("%s. Preserve the host repository for inspection\n"), stepNum)
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
// For clone-repo mode, reusing an existing host repository is not allowed
// If forceDeleteHostRepo is true, deletes the repository if it exists before creating it
func ensureTrialRepository(repoSlug string, cloneRepoSlug string, forceDeleteHostRepo bool, verbose bool) error {
	parts := strings.Split(repoSlug, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository slug format: %s (expected user/repo)", repoSlug)
	}

	// Check if repository already exists
	cmd := exec.Command("gh", "repo", "view", repoSlug)
	if err := cmd.Run(); err == nil {
		// Repository exists - determine what to do
		if forceDeleteHostRepo {
			// Force delete mode: delete the existing repository first
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Force deleting existing host repository: %s", repoSlug)))
			}

			deleteCmd := exec.Command("gh", "repo", "delete", repoSlug, "--yes")
			if deleteOutput, deleteErr := deleteCmd.CombinedOutput(); deleteErr != nil {
				return fmt.Errorf("failed to force delete existing host repository %s: %w (output: %s)", repoSlug, deleteErr, string(deleteOutput))
			}

			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Force deleted existing host repository: %s", repoSlug)))

			// Continue to create the repository below
		} else if cloneRepoSlug != "" {
			// Clone-repo mode: reusing existing repository is not allowed (unless force delete is used)
			return fmt.Errorf("host repository %s already exists, but reusing existing repositories is not allowed in clone-repo mode. Please specify a different --host-repo, use --force-delete-host-repo-before, or delete the existing repository", repoSlug)
		} else {
			// Logical-repo mode: reusing is allowed
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Reusing existing host repository: %s", repoSlug)))
			}
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Using existing host repository: https://github.com/%s", repoSlug)))
			return nil
		}
	}

	// Repository doesn't exist, create it
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Creating private host repository: %s", repoSlug)))
	}

	// Use gh CLI to create private repo with initial README using full OWNER/REPO format
	cmd = exec.Command("gh", "repo", "create", repoSlug, "--private", "--add-readme", "--description", "GitHub Agentic Workflows host repository")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("failed to create host repository: %w (output: %s)", err, string(output))
	}

	// Show host repository creation message with URL
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Created host repository: https://github.com/%s", repoSlug)))

	// Prompt user to enable GitHub Actions permissions
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(""))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("IMPORTANT: You must enable GitHub Actions permissions for the repository."))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("1. Go to: https://github.com/%s/settings/actions", repoSlug)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("2. Under 'Workflow permissions', select 'Allow GitHub Actions to create and approve pull requests'"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("3. Click 'Save'"))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(""))

	// Wait for user confirmation
	fmt.Fprint(os.Stderr, console.FormatPromptMessage("Press Enter after you have enabled these permissions..."))
	var userInput string
	_, _ = fmt.Scanln(&userInput) // Ignore error (user pressed Enter without typing anything)
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Continuing with trial setup"))

	// Enable discussions in the repository as most workflows use them
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Enabling discussions in repository: %s", repoSlug)))
	}

	discussionsCmd := exec.Command("gh", "repo", "edit", repoSlug, "--enable-discussions")
	if discussionsOutput, discussionsErr := discussionsCmd.CombinedOutput(); discussionsErr != nil {
		// Non-fatal error, just warn
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to enable discussions: %v (output: %s)", discussionsErr, string(discussionsOutput))))
	} else if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Enabled discussions in host repository"))
	}

	// Give GitHub a moment to fully initialize the repository
	time.Sleep(2 * time.Second)

	return nil
}

func cleanupTrialRepository(repoSlug string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Cleaning up host repository: %s", repoSlug)))
	}

	// Use gh CLI to delete the repository with proper username/repo format
	cmd := exec.Command("gh", "repo", "delete", repoSlug, "--yes")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("failed to delete host repository: %w (output: %s)", err, string(output))
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Deleted host repository: %s", repoSlug)))
	}

	return nil
}

func cloneTrialHostRepository(repoSlug string, verbose bool) (string, error) {
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
func installWorkflowInTrialMode(tempDir string, parsedSpec *WorkflowSpec, logicalRepoSlug, cloneRepoSlug, hostRepoSlug string, secretTracker *TrialSecretTracker, engineOverride string, appendText string, verbose bool) error {
	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		return fmt.Errorf("failed to change to temp directory: %w", err)
	}

	// Check if this is a local workflow
	if strings.HasPrefix(parsedSpec.WorkflowPath, "./") {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Installing local workflow '%s' from '%s' in trial mode", parsedSpec.WorkflowName, parsedSpec.WorkflowPath)))
		}

		// For local workflows, copy the file directly from the filesystem
		if err := installLocalWorkflowInTrialMode(originalDir, tempDir, parsedSpec, appendText, verbose); err != nil {
			return fmt.Errorf("failed to install local workflow: %w", err)
		}
	} else {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Installing workflow '%s' from '%s' in trial mode", parsedSpec.WorkflowName, parsedSpec.RepoSlug)))
		}

		// Install the source repository as a package
		if err := InstallPackage(parsedSpec.RepoSlug, verbose); err != nil {
			return fmt.Errorf("failed to install source repository: %w", err)
		}

		// Add the workflow from the installed package
		if err := AddWorkflows([]string{parsedSpec.String()}, 1, verbose, "", "", true, appendText, false); err != nil {
			return fmt.Errorf("failed to add workflow: %w", err)
		}
	}

	// Now we need to modify the workflow for trial mode
	if err := modifyWorkflowForTrialMode(tempDir, parsedSpec.WorkflowName, logicalRepoSlug, verbose); err != nil {
		return fmt.Errorf("failed to modify workflow for trial mode: %w", err)
	}

	// Compile the workflow with trial modifications
	config := CompileConfig{
		MarkdownFiles:        []string{".github/workflows/" + parsedSpec.WorkflowName + ".md"},
		Verbose:              verbose,
		EngineOverride:       engineOverride,
		Validate:             true,
		Watch:                false,
		WorkflowDir:          "",
		SkipInstructions:     false,
		NoEmit:               false,
		Purge:                false,
		TrialMode:            (cloneRepoSlug == ""), // Enable trial mode in compiler if in clone-repo mode
		TrialLogicalRepoSlug: logicalRepoSlug,
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
	if err := determineAndAddEngineSecret(workflowData, hostRepoSlug, secretTracker, engineOverride, verbose); err != nil {
		return fmt.Errorf("failed to determine engine secret: %w", err)
	}

	// Commit and push the changes
	if err := commitAndPushWorkflow(tempDir, parsedSpec.WorkflowName, verbose); err != nil {
		return fmt.Errorf("failed to commit and push workflow: %w", err)
	}

	return nil
}

// installLocalWorkflowInTrialMode installs a local workflow file for trial mode
func installLocalWorkflowInTrialMode(originalDir, tempDir string, parsedSpec *WorkflowSpec, appendText string, verbose bool) error {
	// Construct the source path (relative to original directory)
	sourcePath := filepath.Join(originalDir, parsedSpec.WorkflowPath)

	// Check if the source file exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("local workflow file does not exist: %s", sourcePath)
	}

	// Create the workflows directory in the temp directory
	workflowsDir := filepath.Join(tempDir, constants.GetWorkflowDir())
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflows directory: %w", err)
	}

	// Construct the destination path
	destPath := filepath.Join(workflowsDir, parsedSpec.WorkflowName+".md")

	// Read the source file
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read local workflow file: %w", err)
	}

	// Append text if provided
	if appendText != "" {
		contentStr := string(content)
		// Ensure we have a newline before appending
		if !strings.HasSuffix(contentStr, "\n") {
			contentStr += "\n"
		}
		contentStr += "\n" + appendText
		content = []byte(contentStr)
	}

	// Write the content to the destination
	if err := os.WriteFile(destPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write workflow to destination: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Copied local workflow from %s to %s", sourcePath, destPath)))
	}

	return nil
}

func addGitHubTokenSecret(repoSlug string, tracker *TrialSecretTracker, verbose bool) error {
	secretName := "GH_AW_GITHUB_TOKEN"

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Adding GitHub token as repository secret"))
	}

	// Check if secret already exists by trying to list secrets
	listCmd := exec.Command("gh", "secret", "list", "--repo", repoSlug)
	listOutput, listErr := listCmd.CombinedOutput()
	secretExists := listErr == nil && strings.Contains(string(listOutput), secretName)

	// Skip if secret already exists
	if secretExists {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Secret %s already exists, skipping", secretName)))
		}
		return nil
	}

	// Get the current auth token using the proper helper
	token, err := parser.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("failed to get GitHub auth token: %w", err)
	}

	// Add the token as a repository secret
	setCmd := exec.Command("gh", "secret", "set", secretName, "--repo", repoSlug, "--body", token)
	output, err := setCmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("failed to set repository secret: %w (output: %s)", err, string(output))
	}

	// Mark as successfully added
	tracker.AddedSecrets[secretName] = true

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Added %s secret to host repository", secretName)))
	}

	return nil
}

func triggerWorkflowRun(repoSlug, workflowName string, triggerContext string, verbose bool) (string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Triggering workflow run for: %s", workflowName)))
	}

	// Trigger workflow using gh CLI
	lockFileName := fmt.Sprintf("%s.lock.yml", workflowName)

	// Build the command args
	args := []string{"workflow", "run", lockFileName, "--repo", repoSlug}

	// If trigger context is provided, extract issue number and add it as input
	if triggerContext != "" {
		issueNumber := parseIssueSpec(triggerContext)
		if issueNumber != "" {
			args = append(args, "--field", fmt.Sprintf("issue_number=%s", issueNumber))
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Using issue number %s from trigger context", issueNumber)))
			}
		} else if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Could not extract issue number from trigger context, running without inputs"))
		}
	}

	cmd := exec.Command("gh", args...)
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

// parseIssueSpec extracts the issue number from various formats
// Supports:
// - GitHub issue URLs: https://github.com/owner/repo/issues/123
// - Issue references: #123
// - Plain numbers: 123
func parseIssueSpec(input string) string {
	input = strings.TrimSpace(input)

	// First try to match GitHub issue URLs
	urlRegex := regexp.MustCompile(`https://github\.com/[^/]+/[^/]+/issues/(\d+)`)
	if matches := urlRegex.FindStringSubmatch(input); len(matches) >= 2 {
		return matches[1]
	}

	// Try to match issue references like #123
	refRegex := regexp.MustCompile(`^#(\d+)$`)
	if matches := refRegex.FindStringSubmatch(input); len(matches) >= 2 {
		return matches[1]
	}

	// Try to match plain numbers like 123
	numberRegex := regexp.MustCompile(`^\d+$`)
	if numberRegex.MatchString(input) {
		return input
	}

	return ""
}

// determineAndAddEngineSecret determines and sets the appropriate engine secret based on workflow configuration with tracking
func determineAndAddEngineSecret(workflowData *workflow.WorkflowData, hostRepoSlug string, tracker *TrialSecretTracker, engineOverride string, verbose bool) error {
	var engineType string

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Determining required engine secret for workflow"))
	}

	// Debug: Always show what engine override we received
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("DEBUG: engineOverride parameter = '%s'", engineOverride)))

	// Use engine override if provided
	if engineOverride != "" {
		engineType = engineOverride
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Using engine override: %s", engineType)))
	} else {
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
		return addEngineSecret("ANTHROPIC_API_KEY", hostRepoSlug, tracker, verbose)
	case "codex", "openai":
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Setting OPENAI_API_KEY secret for OpenAI engine"))
		}
		return addEngineSecret("OPENAI_API_KEY", hostRepoSlug, tracker, verbose)
	case "copilot":
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Setting COPILOT_CLI_TOKEN secret for Copilot engine"))
		}
		return addEngineSecret("COPILOT_CLI_TOKEN", hostRepoSlug, tracker, verbose)
	default:
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Unknown engine type '%s', defaulting to Copilot", engineType)))
		}
		return addEngineSecret("COPILOT_CLI_TOKEN", hostRepoSlug, tracker, verbose)
	}
}

// addEngineSecret adds an engine-specific secret to the repository with tracking
func addEngineSecret(secretName, hostRepoSlug string, tracker *TrialSecretTracker, verbose bool) error {
	// Check if secret already exists by trying to list secrets
	listCmd := exec.Command("gh", "secret", "list", "--repo", hostRepoSlug)
	listOutput, listErr := listCmd.CombinedOutput()
	secretExists := listErr == nil && strings.Contains(string(listOutput), secretName)

	// Skip if secret already exists
	if secretExists {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Secret %s already exists, skipping", secretName)))
		}
		return nil
	}

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
	repoSlug := hostRepoSlug

	// Add the secret to the repository
	addSecretCmd := exec.Command("gh", "secret", "set", secretName, "--repo", repoSlug, "--body", secretValue)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Running: gh secret set %s --repo %s --body <redacted>", secretName, repoSlug)))
	}

	if output, err := addSecretCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add %s secret: %w\nOutput: %s", secretName, err, string(output))
	}

	// Mark as successfully added
	tracker.AddedSecrets[secretName] = true

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully added %s secret", secretName)))
	}

	return nil
}

// modifyWorkflowForTrialMode modifies the workflow to work in trial mode
func modifyWorkflowForTrialMode(tempDir, workflowName, logicalRepoSlug string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Modifying workflow for trial mode"))
	}

	// Find the workflow markdown file
	workflowPath := filepath.Join(tempDir, constants.GetWorkflowDir(), fmt.Sprintf("%s.md", workflowName))

	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Replace repository references in the content only if logicalRepoSlug is not empty
	modifiedContent := string(content)

	if logicalRepoSlug != "" {
		// Replace github.repository references to point to simulated host repo
		modifiedContent = strings.ReplaceAll(modifiedContent, "${{ github.repository }}", logicalRepoSlug)

		// Also replace any hardcoded checkout actions to use the simulated host repo
		// Split content into lines to preserve indentation
		lines := strings.Split(modifiedContent, "\n")
		checkoutPattern := regexp.MustCompile(`^(\s*)(uses: actions/checkout@[^\s]*)(.*)$`)

		var newLines []string
		for _, line := range lines {
			if matches := checkoutPattern.FindStringSubmatch(line); len(matches) >= 3 {
				indentation := matches[1]
				usesLine := matches[2]
				remainder := matches[3]

				// Add the original uses line
				newLines = append(newLines, fmt.Sprintf("%s%s%s", indentation, usesLine, remainder))
				// Add the with clause at the same indentation level as uses
				newLines = append(newLines, fmt.Sprintf("%swith:", indentation))
				newLines = append(newLines, fmt.Sprintf("%s  repository: %s", indentation, logicalRepoSlug))
			} else {
				newLines = append(newLines, line)
			}
		}

		modifiedContent = strings.Join(newLines, "\n")
	} else if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Skipping repository simulation modifications (using clone-repo mode)"))
	}

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

// cleanupTrialSecrets removes API key secrets from the host repository for security, based on tracking information
func cleanupTrialSecrets(repoSlug string, tracker *TrialSecretTracker, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Cleaning up API key secrets from host repository"))
	}

	secretsDeleted := 0
	// Only delete secrets that were actually added by this trial command
	for secretName := range tracker.AddedSecrets {
		cmd := exec.Command("gh", "secret", "delete", secretName, "--repo", repoSlug)
		if output, err := cmd.CombinedOutput(); err != nil {
			// It's okay if the secret doesn't exist, just log in verbose mode
			if verbose && !strings.Contains(string(output), "Not Found") {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Could not delete secret %s: %s", secretName, string(output))))
			}
		} else {
			secretsDeleted++
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Deleted secret: %s", secretName)))
			}
		}
	}

	if verbose {
		if secretsDeleted > 0 {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("API key secrets cleaned up from host repository (%d deleted)", secretsDeleted)))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("No secrets needed cleanup (none were added by trial)"))
		}
	}

	return nil
}

// TrialArtifacts represents all artifacts downloaded from a workflow run
type TrialArtifacts struct {
	SafeOutputs map[string]interface{} `json:"safe_outputs"`
	//AgentStdioLogs      []string               `json:"agent_stdio_logs,omitempty"`
	AgenticRunInfo      map[string]interface{} `json:"agentic_run_info,omitempty"`
	AdditionalArtifacts map[string]interface{} `json:"additional_artifacts,omitempty"`
}

// downloadAllArtifacts downloads and parses all available artifacts from a workflow run
func downloadAllArtifacts(hostRepoSlug, runID string, verbose bool) (*TrialArtifacts, error) {
	// Use the repository slug directly (should already be in user/repo format)
	repoSlug := hostRepoSlug

	// Create temp directory for artifact download
	tempDir, err := os.MkdirTemp("", "trial-artifacts-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Download all artifacts for this run
	cmd := exec.Command("gh", "run", "download", runID, "--repo", repoSlug, "--dir", tempDir)
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

		// case strings.Contains(relPath, "agent") && strings.HasSuffix(path, ".log"):
		// 	// Collect agent stdio logs
		// 	if logContent := readTextArtifact(path, verbose); logContent != "" {
		// 		artifacts.AgentStdioLogs = append(artifacts.AgentStdioLogs, logContent)
		// 	}

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

// copyTrialResultsToHostRepo copies trial result files to the host repository and commits them
func copyTrialResultsToHostRepo(tempDir, dateTimeID string, workflowNames []string, logicalRepoSlug string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Copying trial results to host repository"))
	}

	// Create trials directory in the host repository
	trialsDir := filepath.Join(tempDir, "trials")
	if err := os.MkdirAll(trialsDir, 0755); err != nil {
		return fmt.Errorf("failed to create trials directory in repository: %w", err)
	}

	// Copy individual workflow result files
	sanitizedTargetRepo := sanitizeRepoSlugForFilename(logicalRepoSlug)
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
	if repoSlug == "" {
		return "clone-mode"
	}
	return strings.ReplaceAll(repoSlug, "/", "-")
}

// copyFile copies a file from source to destination
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

// cloneRepoContentsIntoHost clones the contents of the source repo into the host repo
// Uses a simplified approach with force push since host repo is freshly created
func cloneRepoContentsIntoHost(cloneRepoSlug string, cloneRepoVersion string, hostRepoSlug string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Cloning contents from %s into host repository %s", cloneRepoSlug, hostRepoSlug)))
	}

	// Save the original working directory to restore it later
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	defer os.Chdir(originalDir)

	// Create temporary directory to clone the source repo
	tempCloneDir := filepath.Join(os.TempDir(), fmt.Sprintf("gh-aw-clone-%x", time.Now().UnixNano()))
	defer os.RemoveAll(tempCloneDir)

	// Clone the source repository
	cloneURL := fmt.Sprintf("https://github.com/%s.git", cloneRepoSlug)
	cmd := exec.Command("git", "clone", cloneURL, tempCloneDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone source repository %s: %w (output: %s)", cloneURL, err, string(output))
	}

	// Change to the cloned repository directory
	if err := os.Chdir(tempCloneDir); err != nil {
		return fmt.Errorf("failed to change to clone directory: %w", err)
	}

	// If a version/tag/SHA is specified, checkout that ref
	if cloneRepoVersion != "" {
		cmd = exec.Command("git", "checkout", cloneRepoVersion)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to checkout ref '%s': %w (output: %s)", cloneRepoVersion, err, string(output))
		}
	}

	// Add the host repository as a new remote
	hostURL := fmt.Sprintf("https://github.com/%s.git", hostRepoSlug)
	cmd = exec.Command("git", "remote", "add", "host", hostURL)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add host remote: %w (output: %s)", err, string(output))
	}

	// Force push the current branch to the host repository's main branch
	cmd = exec.Command("git", "push", "--force", "host", "HEAD:main")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to force push to host repository: %w (output: %s)", err, string(output))
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully pushed contents from %s to %s", cloneRepoSlug, hostRepoSlug)))
	}

	return nil
}
