package cli

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var runLog = logger.New("cli:run_command")

// RunWorkflowOnGitHub runs an agentic workflow on GitHub Actions
func RunWorkflowOnGitHub(ctx context.Context, workflowIdOrName string, enable bool, engineOverride string, repoOverride string, refOverride string, autoMergePRs bool, pushSecrets bool, waitForCompletion bool, inputs []string, verbose bool) error {
	runLog.Printf("Starting workflow run: workflow=%s, enable=%v, engineOverride=%s, repo=%s, ref=%s, wait=%v, inputs=%v", workflowIdOrName, enable, engineOverride, repoOverride, refOverride, waitForCompletion, inputs)

	// Check context cancellation at the start
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
	}

	if workflowIdOrName == "" {
		return fmt.Errorf("workflow name or ID is required")
	}

	// Validate input format early before attempting workflow validation
	for _, input := range inputs {
		if !strings.Contains(input, "=") {
			return fmt.Errorf("invalid input format '%s': expected key=value", input)
		}
		// Check that key (before '=') is not empty
		parts := strings.SplitN(input, "=", 2)
		if len(parts[0]) == 0 {
			return fmt.Errorf("invalid input format '%s': key cannot be empty", input)
		}
	}

	if verbose {
		fmt.Printf("Running workflow on GitHub Actions: %s\n", workflowIdOrName)
	}

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

		// Validate workflow inputs
		if err := validateWorkflowInputs(workflowFile, inputs); err != nil {
			return fmt.Errorf("%w", err)
		}

		// Check if the workflow file has local modifications
		if status, err := checkWorkflowFileStatus(workflowFile); err == nil && status != nil {
			var warnings []string

			if status.IsModified {
				warnings = append(warnings, "The workflow file has unstaged changes")
			}
			if status.IsStaged {
				warnings = append(warnings, "The workflow file has staged changes")
			}
			if status.HasUnpushedCommits {
				warnings = append(warnings, "The workflow file has unpushed commits")
			}

			if len(warnings) > 0 {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(strings.Join(warnings, ", ")))
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("These changes will not be reflected in the GitHub Actions run"))
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Consider pushing your changes before running the workflow"))
			}
		}
	}

	// Handle --enable flag logic: check workflow state and enable if needed
	var wasDisabled bool
	var workflowID int64
	if enable {
		// Get current workflow status
		wf, err := getWorkflowStatus(workflowIdOrName, repoOverride, verbose)
		if err != nil {
			if verbose {
				fmt.Printf("Warning: Could not check workflow status: %v\n", err)
			}
		}

		// If we successfully got workflow status, check if it needs enabling
		if err == nil {
			workflowID = wf.ID
			if wf.State == "disabled_manually" {
				wasDisabled = true
				runLog.Printf("Workflow %s is disabled, temporarily enabling for this run (id=%d)", workflowIdOrName, wf.ID)
				if verbose {
					fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Workflow '%s' is disabled, enabling it temporarily...", workflowIdOrName)))
				}
				// Enable the workflow
				enableArgs := []string{"workflow", "enable", strconv.FormatInt(wf.ID, 10)}
				if repoOverride != "" {
					enableArgs = append(enableArgs, "--repo", repoOverride)
				}
				cmd := workflow.ExecGH(enableArgs...)
				if err := cmd.Run(); err != nil {
					runLog.Printf("Failed to enable workflow %s: %v", workflowIdOrName, err)
					return fmt.Errorf("failed to enable workflow '%s': %w", workflowIdOrName, err)
				}
				fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Enabled workflow: %s", workflowIdOrName)))
			} else {
				runLog.Printf("Workflow %s is already enabled (state=%s)", workflowIdOrName, wf.State)
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
			return fmt.Errorf("failed to find workflow in local .github/workflows: %w", err)
		}

		// For local workflows, use the simple filename
		filename := strings.TrimSuffix(filepath.Base(workflowIdOrName), ".md")
		lockFileName = filename + ".lock.yml"

		// Check if the lock file exists in .github/workflows
		lockFilePath = filepath.Join(".github/workflows", lockFileName)
		if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
			runLog.Printf("Lock file not found: %s (workflow must be compiled first)", lockFilePath)
			suggestions := []string{
				fmt.Sprintf("Run '%s compile' to compile all workflows", string(constants.CLIExtensionPrefix)),
				fmt.Sprintf("Run '%s compile %s' to compile this specific workflow", string(constants.CLIExtensionPrefix), filename),
			}
			return errors.New(console.FormatErrorWithSuggestions(
				fmt.Sprintf("workflow lock file '%s' not found in .github/workflows", lockFileName),
				suggestions,
			))
		}
		runLog.Printf("Found lock file: %s", lockFilePath)
	}

	// Recompile workflow if engine override is provided (only for local workflows)
	if engineOverride != "" && repoOverride == "" {
		if verbose {
			fmt.Printf("Recompiling workflow with engine override: %s\n", engineOverride)
		}

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
		if _, err := CompileWorkflows(ctx, config); err != nil {
			return fmt.Errorf("failed to recompile workflow with engine override: %w", err)
		}

		if verbose {
			fmt.Printf("Successfully recompiled workflow with engine: %s\n", engineOverride)
		}
	} else if engineOverride != "" && repoOverride != "" {
		if verbose {
			fmt.Printf("Note: Engine override ignored for remote repository workflows\n")
		}
	}

	if verbose {
		fmt.Printf("Using lock file: %s\n", lockFileName)
	}

	// Handle secret pushing if requested
	var secretTracker *TrialSecretTracker
	if pushSecrets {
		// Determine target repository
		var targetRepo string
		if repoOverride != "" {
			targetRepo = repoOverride
		} else {
			// Get current repository slug
			currentRepo, err := GetCurrentRepoSlug()
			if err != nil {
				return fmt.Errorf("failed to determine current repository for secret handling: %w", err)
			}
			targetRepo = currentRepo
		}

		secretTracker = NewTrialSecretTracker(targetRepo)
		runLog.Printf("Created secret tracker for repository: %s", targetRepo)

		// Set up secret cleanup to always run on exit
		defer func() {
			if err := cleanupTrialSecrets(targetRepo, secretTracker, verbose); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to cleanup secrets: %v", err)))
			}
		}()

		// Add GitHub token secret
		if err := addGitHubTokenSecret(targetRepo, secretTracker, verbose); err != nil {
			return fmt.Errorf("failed to add GitHub token secret: %w", err)
		}

		// Determine and add engine secrets
		if repoOverride == "" && lockFilePath != "" {
			// For local workflows, read and parse the workflow to determine engine requirements
			workflowMarkdownPath := strings.TrimSuffix(lockFilePath, ".lock.yml") + ".md"
			config := CompileConfig{
				MarkdownFiles:        []string{workflowMarkdownPath},
				Verbose:              false, // Don't be verbose during secret determination
				EngineOverride:       engineOverride,
				Validate:             false,
				Watch:                false,
				WorkflowDir:          "",
				SkipInstructions:     true,
				NoEmit:               true, // Don't emit files, just compile for analysis
				Purge:                false,
				TrialMode:            false,
				TrialLogicalRepoSlug: "",
				Strict:               false,
			}
			workflowDataList, err := CompileWorkflows(ctx, config)
			if err == nil && len(workflowDataList) == 1 {
				workflowData := workflowDataList[0]
				if err := determineAndAddEngineSecret(workflowData.EngineConfig, targetRepo, secretTracker, engineOverride, verbose); err != nil {
					// Log warning but don't fail - the workflow might still run without secrets
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to determine engine secret: %v", err)))
					}
				}
			} else if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Failed to compile workflow for secret determination - continuing without engine secrets"))
			}
		} else if repoOverride != "" {
			// For remote workflows, we can't analyze the workflow file, so create a minimal EngineConfig
			// with engine information and reuse the existing determineAndAddEngineSecret function
			var engineType string
			if engineOverride != "" {
				engineType = engineOverride
			} else {
				engineType = "copilot" // Default engine
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Using default Copilot engine for remote workflow secret handling"))
				}
			}

			// Create minimal EngineConfig with engine information
			engineConfig := &workflow.EngineConfig{
				ID: engineType,
			}

			if err := determineAndAddEngineSecret(engineConfig, targetRepo, secretTracker, engineOverride, verbose); err != nil {
				// Log warning but don't fail - the workflow might still run without secrets
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to determine engine secret for remote workflow: %v", err)))
				}
			}
		}
	}

	// Build the gh workflow run command with optional repo and ref overrides
	args := []string{"workflow", "run", lockFileName}
	if repoOverride != "" {
		args = append(args, "--repo", repoOverride)
	}

	// Determine the ref to use (branch/tag)
	// If refOverride is specified, use it; otherwise for local workflows, use current branch
	ref := refOverride
	if ref == "" && repoOverride == "" {
		// For local workflows without explicit ref, use the current branch
		if currentBranch, err := getCurrentBranch(); err == nil {
			ref = currentBranch
			runLog.Printf("Using current branch for workflow run: %s", ref)
		} else if verbose {
			fmt.Printf("Note: Could not determine current branch: %v\n", err)
		}
	}
	if ref != "" {
		args = append(args, "--ref", ref)
	}

	// Add workflow inputs if provided
	if len(inputs) > 0 {
		for _, input := range inputs {
			// Add as raw field flag to gh workflow run
			args = append(args, "-f", input)
		}
	}

	// Record the start time for auto-merge PR filtering
	workflowStartTime := time.Now()

	// Execute gh workflow run command and capture output
	cmd := workflow.ExecGH(args...)

	if verbose {
		var cmdParts []string
		cmdParts = append(cmdParts, "gh workflow run", lockFileName)
		if repoOverride != "" {
			cmdParts = append(cmdParts, "--repo", repoOverride)
		}
		if ref != "" {
			cmdParts = append(cmdParts, "--ref", ref)
		}
		if len(inputs) > 0 {
			for _, input := range inputs {
				cmdParts = append(cmdParts, "-f", input)
			}
		}
		fmt.Printf("Executing: %s\n", strings.Join(cmdParts, " "))
	}

	// Capture both stdout and stderr
	stdout, err := cmd.Output()
	if err != nil {
		// If there's an error, try to get stderr for better error reporting
		var stderrOutput string
		if exitError, ok := err.(*exec.ExitError); ok {
			stderrOutput = string(exitError.Stderr)
			fmt.Fprintf(os.Stderr, "%s", exitError.Stderr)
		}

		// Restore workflow state if it was disabled and we enabled it (even on error)
		if enable && wasDisabled && workflowID != 0 {
			restoreWorkflowState(workflowIdOrName, workflowID, repoOverride, verbose)
		}

		// Check if this is a permission error in a codespace
		errorMsg := err.Error() + " " + stderrOutput
		if isRunningInCodespace() && is403PermissionError(errorMsg) {
			// Show specialized error message for codespace users
			fmt.Fprint(os.Stderr, getCodespacePermissionErrorMessage())
			return fmt.Errorf("failed to run workflow on GitHub Actions: permission denied (403)")
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

		// Suggest audit command for analysis
		fmt.Printf("\n💡 To analyze this run, use: %s audit %d\n", string(constants.CLIExtensionPrefix), runInfo.DatabaseID)
	} else if verbose && runErr != nil {
		fmt.Printf("Note: Could not get workflow run URL: %v\n", runErr)
	}

	// Wait for workflow completion if requested (for --repeat or --auto-merge-prs)
	if waitForCompletion || autoMergePRs {
		if runErr != nil {
			if autoMergePRs {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not get workflow run information for auto-merge: %v", runErr)))
			} else if waitForCompletion {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not get workflow run information: %v", runErr)))
			}
		} else {
			// Determine target repository: use repo override if provided, otherwise get current repo
			targetRepo := repoOverride
			if targetRepo == "" {
				if currentRepo, err := GetCurrentRepoSlug(); err != nil {
					if autoMergePRs {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not determine target repository for auto-merge: %v", err)))
					}
					targetRepo = ""
				} else {
					targetRepo = currentRepo
				}
			}

			if targetRepo != "" {
				// Wait for workflow completion
				if autoMergePRs {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Auto-merge PRs enabled - waiting for workflow completion..."))
				} else {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Waiting for workflow completion..."))
				}

				runIDStr := fmt.Sprintf("%d", runInfo.DatabaseID)
				if err := WaitForWorkflowCompletion(targetRepo, runIDStr, 30, verbose); err != nil {
					if autoMergePRs {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Workflow did not complete successfully, skipping auto-merge: %v", err)))
					} else {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Workflow did not complete successfully: %v", err)))
					}
				} else {
					// Auto-merge PRs if requested and workflow completed successfully
					if autoMergePRs {
						if err := AutoMergePullRequestsCreatedAfter(targetRepo, workflowStartTime, verbose); err != nil {
							fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to auto-merge pull requests: %v", err)))
						}
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
func RunWorkflowsOnGitHub(ctx context.Context, workflowNames []string, repeatCount int, enable bool, engineOverride string, repoOverride string, refOverride string, autoMergePRs bool, pushSecrets bool, inputs []string, verbose bool) error {
	if len(workflowNames) == 0 {
		return fmt.Errorf("at least one workflow name or ID is required")
	}

	// Check context cancellation at the start
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
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

		// Wait for completion when using --repeat to ensure workflows finish before next iteration
		waitForCompletion := repeatCount > 0

		for i, workflowName := range workflowNames {
			// Check for cancellation before each workflow
			select {
			case <-ctx.Done():
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
				return ctx.Err()
			default:
			}

			if len(workflowNames) > 1 {
				if msg := console.FormatProgressMessage(fmt.Sprintf("Running workflow %d/%d: %s", i+1, len(workflowNames), workflowName)); msg != "" {
					fmt.Println(msg)
				}
			}

			if err := RunWorkflowOnGitHub(ctx, workflowName, enable, engineOverride, repoOverride, refOverride, autoMergePRs, pushSecrets, waitForCompletion, inputs, verbose); err != nil {
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

// getWorkflowInputs extracts workflow_dispatch inputs from the workflow markdown file
func getWorkflowInputs(markdownPath string) (map[string]*workflow.InputDefinition, error) {
	// Read the file
	contentBytes, err := os.ReadFile(markdownPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	content := string(contentBytes)

	// Extract frontmatter
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract frontmatter: %w", err)
	}

	// Check if 'on' section is present
	onSection, exists := result.Frontmatter["on"]
	if !exists {
		return nil, nil
	}

	// Convert to map if possible
	onMap, ok := onSection.(map[string]any)
	if !ok {
		return nil, nil
	}

	// Get workflow_dispatch section
	workflowDispatch, exists := onMap["workflow_dispatch"]
	if !exists {
		return nil, nil
	}

	// Convert to map
	workflowDispatchMap, ok := workflowDispatch.(map[string]any)
	if !ok {
		// workflow_dispatch might be null/empty
		return nil, nil
	}

	// Get inputs section
	inputsSection, exists := workflowDispatchMap["inputs"]
	if !exists {
		return nil, nil
	}

	// Convert to map
	inputsMap, ok := inputsSection.(map[string]any)
	if !ok {
		return nil, nil
	}

	// Parse input definitions
	return workflow.ParseInputDefinitions(inputsMap), nil
}

// validateWorkflowInputs validates that required inputs are provided and checks for typos.
//
// This validation function is co-located with the run command implementation because:
//   - It's specific to the workflow run operation
//   - It's only called during workflow dispatch
//   - It provides immediate feedback before triggering the workflow
//
// The function validates:
//   - All required inputs are provided
//   - Provided input names match defined inputs (typo detection)
//   - Suggestions for misspelled input names
//
// This follows the principle that domain-specific validation belongs in domain files.
func validateWorkflowInputs(markdownPath string, providedInputs []string) error {
	// Extract workflow inputs
	workflowInputs, err := getWorkflowInputs(markdownPath)
	if err != nil {
		// Don't fail validation if we can't extract inputs
		runLog.Printf("Failed to extract workflow inputs: %v", err)
		return nil
	}

	// If no inputs are defined, no validation needed
	if len(workflowInputs) == 0 {
		return nil
	}

	// Parse provided inputs into a map
	providedInputsMap := make(map[string]string)
	for _, input := range providedInputs {
		parts := strings.SplitN(input, "=", 2)
		if len(parts) == 2 {
			providedInputsMap[parts[0]] = parts[1]
		}
	}

	// Check for required inputs that are missing
	var missingInputs []string
	for inputName, inputDef := range workflowInputs {
		if inputDef.Required {
			if _, exists := providedInputsMap[inputName]; !exists {
				missingInputs = append(missingInputs, inputName)
			}
		}
	}

	// Check for typos in provided input names
	var typos []string
	var suggestions []string
	validInputNames := make([]string, 0, len(workflowInputs))
	for inputName := range workflowInputs {
		validInputNames = append(validInputNames, inputName)
	}

	for providedName := range providedInputsMap {
		// Check if this is a valid input name
		if _, exists := workflowInputs[providedName]; !exists {
			// Find closest matches
			matches := parser.FindClosestMatches(providedName, validInputNames, 3)
			if len(matches) > 0 {
				typos = append(typos, providedName)
				suggestions = append(suggestions, fmt.Sprintf("'%s' -> did you mean '%s'?", providedName, strings.Join(matches, "', '")))
			} else {
				typos = append(typos, providedName)
				suggestions = append(suggestions, fmt.Sprintf("'%s' is not a valid input name", providedName))
			}
		}
	}

	// Build error message if there are validation errors
	if len(missingInputs) > 0 || len(typos) > 0 {
		var errorParts []string

		if len(missingInputs) > 0 {
			errorParts = append(errorParts, fmt.Sprintf("Missing required input(s): %s", strings.Join(missingInputs, ", ")))
		}

		if len(typos) > 0 {
			errorParts = append(errorParts, fmt.Sprintf("Invalid input name(s):\n  %s", strings.Join(suggestions, "\n  ")))
		}

		// Add helpful information about valid inputs
		if len(workflowInputs) > 0 {
			var inputDescriptions []string
			for name, def := range workflowInputs {
				required := ""
				if def.Required {
					required = " (required)"
				}
				desc := ""
				if def.Description != "" {
					desc = fmt.Sprintf(": %s", def.Description)
				}
				inputDescriptions = append(inputDescriptions, fmt.Sprintf("  %s%s%s", name, required, desc))
			}
			errorParts = append(errorParts, fmt.Sprintf("\nValid inputs:\n%s", strings.Join(inputDescriptions, "\n")))
		}

		return fmt.Errorf("%s", strings.Join(errorParts, "\n\n"))
	}

	return nil
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

	// Create spinner outside the loop so we can update it
	var spinner *console.SpinnerWrapper
	if !verbose {
		spinner = console.NewSpinner("Waiting for workflow run to appear...")
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff, capped at maxDelay
			delay := time.Duration(attempt) * initialDelay
			if delay > maxDelay {
				delay = maxDelay
			}

			// Calculate elapsed time since start
			elapsed := time.Since(startTime).Round(time.Second)

			if verbose {
				fmt.Printf("Waiting %v before retry attempt %d/%d...\n", delay, attempt+1, maxRetries)
			} else {
				// Show spinner starting from second attempt to avoid flickering
				if attempt == 1 {
					spinner.Start()
				}
				// Update spinner with progress information
				spinner.UpdateMessage(fmt.Sprintf("Waiting for workflow run... (attempt %d/%d, %v elapsed)", attempt+1, maxRetries, elapsed))
			}
			time.Sleep(delay)
		}

		// Build command with optional repo parameter
		var cmd *exec.Cmd
		if repo != "" {
			cmd = workflow.ExecGH("run", "list", "--repo", repo, "--workflow", lockFileName, "--limit", "1", "--json", "url,databaseId,status,conclusion,createdAt")
		} else {
			cmd = workflow.ExecGH("run", "list", "--workflow", lockFileName, "--limit", "1", "--json", "url,databaseId,status,conclusion,createdAt")
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
			if spinner != nil {
				spinner.StopWithMessage("✓ Found workflow run")
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
		if spinner != nil {
			spinner.StopWithMessage("✓ Found workflow run")
		}
		return runInfo, nil
	}

	// Stop spinner on failure
	if spinner != nil {
		spinner.Stop()
	}

	// If we exhausted all retries, return the last error
	if lastErr != nil {
		return nil, fmt.Errorf("failed to get workflow run after %d attempts: %w", maxRetries, lastErr)
	}

	return nil, fmt.Errorf("no workflow run found after %d attempts", maxRetries)
}

// validateRemoteWorkflow checks if a workflow exists in a remote repository and can be triggered.
//
// This validation function is co-located with the run command implementation because:
//   - It's specific to remote workflow execution
//   - It's only called when running workflows in remote repositories
//   - It provides early validation before attempting workflow dispatch
//
// The function validates:
//   - The specified repository exists and is accessible
//   - The workflow file exists in the repository
//   - The workflow can be triggered via GitHub Actions API
//
// This follows the principle that domain-specific validation belongs in domain files.
func validateRemoteWorkflow(workflowName string, repoOverride string, verbose bool) error {
	if repoOverride == "" {
		return fmt.Errorf("repository must be specified for remote workflow validation")
	}

	// Add .lock.yml extension if not present
	lockFileName := workflowName
	if !strings.HasSuffix(lockFileName, ".lock.yml") {
		lockFileName += ".lock.yml"
	}

	if verbose {
		fmt.Printf("Checking if workflow '%s' exists in repository '%s'...\n", lockFileName, repoOverride)
	}

	// Use gh CLI to list workflows in the target repository
	cmd := workflow.ExecGH("workflow", "list", "--repo", repoOverride, "--json", "name,path,state")
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
	for _, wf := range workflows {
		if strings.HasSuffix(wf.Path, lockFileName) {
			if verbose {
				fmt.Printf("Found workflow '%s' in repository (path: %s, state: %s)\n",
					wf.Name, wf.Path, wf.State)
			}
			return nil
		}
	}

	suggestions := []string{
		"Check if the workflow has been pushed to the remote repository",
		"Verify the workflow file exists in the repository's .github/workflows directory",
		fmt.Sprintf("Run '%s status' to see available workflows", string(constants.CLIExtensionPrefix)),
	}
	return errors.New(console.FormatErrorWithSuggestions(
		fmt.Sprintf("workflow '%s' not found in repository '%s'", lockFileName, repoOverride),
		suggestions,
	))
}
