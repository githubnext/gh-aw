package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/workflow"
)

// executeTrialRun runs one complete set of trials for all workflow specs.
// It is called (possibly multiple times) by RunWorkflowTrials via ExecuteWithRepeat.
func executeTrialRun(ctx context.Context, parsedSpecs []*WorkflowSpec, hostRepoSlug, logicalRepoSlug, cloneRepoSlug string, directTrialMode bool, opts TrialOptions) error {
	// Generate a unique datetime-ID for this trial session
	dateTimeID := fmt.Sprintf("%s-%d", time.Now().Format("20060102-150405"), time.Now().UnixNano()%1000000)
	trialLog.Printf("Starting trial run: dateTimeID=%s", dateTimeID)

	// Determine target repo slug for filenames once
	// In direct trial mode, use hostRepoSlug; otherwise use logicalRepoSlug
	targetRepoForFilename := logicalRepoSlug
	if directTrialMode {
		targetRepoForFilename = hostRepoSlug
	}

	// Step 3: Clone host repository to local temp directory
	trialLog.Printf("Cloning trial host repository: %s", hostRepoSlug)
	tempDir, err := cloneTrialHostRepository(hostRepoSlug, opts.Verbose)
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

	for _, parsedSpec := range parsedSpecs {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("=== Running trial for workflow: %s ===", parsedSpec.WorkflowName)))

		// Install workflow with trial mode compilation
		if err := installWorkflowInTrialMode(ctx, tempDir, parsedSpec, logicalRepoSlug, cloneRepoSlug, hostRepoSlug, directTrialMode, &opts); err != nil {
			return fmt.Errorf("failed to install workflow '%s' in trial mode: %w", parsedSpec.WorkflowName, err)
		}

		// Display workflow description if present
		workflowPath := filepath.Join(tempDir, ".github/workflows", parsedSpec.WorkflowName+".md")
		if description := ExtractWorkflowDescriptionFromFile(workflowPath); description != "" {
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(description))
			fmt.Fprintln(os.Stderr, "")
		}

		// Run the workflow and wait for completion (with trigger context if provided)
		runID, err := triggerWorkflowRun(hostRepoSlug, parsedSpec.WorkflowName, opts.TriggerContext, opts.Verbose)
		if err != nil {
			return fmt.Errorf("failed to trigger workflow run for '%s': %w", parsedSpec.WorkflowName, err)
		}

		// Generate workflow run URL
		githubHost := getGitHubHost()
		workflowRunURL := fmt.Sprintf("%s/%s/actions/runs/%s", githubHost, hostRepoSlug, runID)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow run started with ID: %s (%s)", runID, workflowRunURL)))

		// Wait for workflow completion
		if err := WaitForWorkflowCompletion(ctx, hostRepoSlug, runID, opts.TimeoutMinutes, opts.Verbose); err != nil {
			// If the context was canceled or its deadline was exceeded, return that directly
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			return fmt.Errorf("workflow '%s' execution failed or timed out: %w", parsedSpec.WorkflowName, err)
		}

		// Auto-merge PRs if requested
		if opts.AutoMergePRs {
			if err := AutoMergePullRequestsLegacy(hostRepoSlug, opts.Verbose); err != nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to auto-merge pull requests: %v", err)))
			}
		}

		// Download and process all artifacts
		artifacts, err := downloadAllArtifacts(hostRepoSlug, runID, opts.Verbose)
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
		sanitizedTargetRepo := stringutil.SanitizeForFilename(targetRepoForFilename)
		individualFilename := fmt.Sprintf("trials/%s-%s.%s.json", parsedSpec.WorkflowName, sanitizedTargetRepo, dateTimeID)
		if err := saveTrialResult(individualFilename, result, opts.Verbose); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to save individual trial result: %v", err)))
		}

		// Display safe outputs to stdout
		if len(artifacts.SafeOutputs) > 0 {
			outputBytes, _ := json.MarshalIndent(artifacts.SafeOutputs, "", "  ")
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("=== Safe Outputs from %s ===", parsedSpec.WorkflowName)))
			fmt.Println(string(outputBytes))
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("=== End of Safe Outputs ==="))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("=== No Safe Outputs Generated by %s ===", parsedSpec.WorkflowName)))
		}

		// Display additional artifact information if available
		// if len(artifacts.AgentStdioLogs) > 0 {
		// 	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("=== Agent Stdio Logs Available from %s (%d files) ===", parsedSpec.WorkflowName, len(artifacts.AgentStdioLogs))))
		// }
		if len(artifacts.AgenticRunInfo) > 0 {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("=== Agentic Run Information Available from %s ===", parsedSpec.WorkflowName)))
		}
		if len(artifacts.AdditionalArtifacts) > 0 {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("=== Additional Artifacts Available from %s (%d files) ===", parsedSpec.WorkflowName, len(artifacts.AdditionalArtifacts))))
		}

		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Trial completed for workflow: "+parsedSpec.WorkflowName))
	}

	// Step 6: Save combined results for multi-workflow trials
	if len(parsedSpecs) > 1 {
		workflowNames := sliceutil.Map(parsedSpecs, func(spec *WorkflowSpec) string { return spec.WorkflowName })
		workflowNamesStr := strings.Join(workflowNames, "-")
		sanitizedTargetRepo := stringutil.SanitizeForFilename(targetRepoForFilename)
		combinedFilename := fmt.Sprintf("trials/%s-%s.%s.json", workflowNamesStr, sanitizedTargetRepo, dateTimeID)
		combinedResult := CombinedTrialResult{
			WorkflowNames: workflowNames,
			Results:       workflowResults,
			Timestamp:     time.Now(),
		}
		if err := saveTrialResult(combinedFilename, combinedResult, opts.Verbose); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to save combined trial result: %v", err)))
		}
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Combined results saved to: "+combinedFilename))
	}

	// Step 6.5: Copy trial results to host repository and commit them
	workflowNames := sliceutil.Map(parsedSpecs, func(spec *WorkflowSpec) string { return spec.WorkflowName })
	if err := copyTrialResultsToHostRepo(tempDir, dateTimeID, workflowNames, targetRepoForFilename, opts.Verbose); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to copy trial results to repository: %v", err)))
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("All trials completed successfully"))
	return nil
}

func triggerWorkflowRun(repoSlug, workflowName string, triggerContext string, verbose bool) (string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Triggering workflow run for: "+workflowName))
	}

	// Trigger workflow using gh CLI
	lockFileName := workflowName + ".lock.yml"

	// Build the command args
	args := []string{"workflow", "run", lockFileName, "--repo", repoSlug}

	// If trigger context is provided, extract issue number and add it as input
	if triggerContext != "" {
		issueNumber := parseIssueSpec(triggerContext)
		if issueNumber != "" {
			args = append(args, "--field", "issue_number="+issueNumber)
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Using issue number %s from trigger context", issueNumber)))
			}
		} else if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Could not extract issue number from trigger context, running without inputs"))
		}
	}

	output, err := workflow.RunGHCombined("Triggering workflow...", args...)

	if err != nil {
		return "", fmt.Errorf("failed to trigger workflow run: %w (output: %s)", err, string(output))
	}

	// Get the most recent run ID for this workflow using shared retry logic
	runInfo, err := getLatestWorkflowRunWithRetry(lockFileName, repoSlug, verbose)
	if err != nil {
		return "", fmt.Errorf("failed to get workflow run ID: %w", err)
	}

	runID := strconv.FormatInt(runInfo.DatabaseID, 10)

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

// saveTrialResult saves a trial result to a JSON file
func saveTrialResult(filename string, result any, verbose bool) error {
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	if err := os.WriteFile(filename, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write result file: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Saved trial result to: "+filename))
	}

	return nil
}

// copyTrialResultsToHostRepo copies trial result files to the host repository and commits them
func copyTrialResultsToHostRepo(tempDir, dateTimeID string, workflowNames []string, targetRepoSlug string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Copying trial results to host repository"))
	}

	// Create trials directory in the host repository
	trialsDir := filepath.Join(tempDir, "trials")
	if err := os.MkdirAll(trialsDir, 0755); err != nil {
		return fmt.Errorf("failed to create trials directory in repository: %w", err)
	}

	// Copy individual workflow result files
	sanitizedTargetRepo := stringutil.SanitizeForFilename(targetRepoSlug)
	for _, workflowName := range workflowNames {
		sourceFile := fmt.Sprintf("trials/%s-%s.%s.json", workflowName, sanitizedTargetRepo, dateTimeID)
		destFile := filepath.Join(trialsDir, fmt.Sprintf("%s-%s.%s.json", workflowName, sanitizedTargetRepo, dateTimeID))

		if err := fileutil.CopyFile(sourceFile, destFile); err != nil {
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

		if err := fileutil.CopyFile(combinedSourceFile, combinedDestFile); err != nil {
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

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Trial results copied to repository and pushed"))

	return nil
}
