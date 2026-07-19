package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/constants"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/sliceutil"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/workflow"
)

// issuePathPattern matches the path portion of a GitHub issue URL: /owner/repo/issues/NUMBER
var issuePathPattern = regexp.MustCompile(`^/[^/]+/[^/]+/issues/(\d+)`)

// issueRefPattern matches issue references like #123
var issueRefPattern = regexp.MustCompile(`^#(\d+)$`)

// issueNumberPattern matches plain issue numbers like 123
var issueNumberPattern = regexp.MustCompile(`^\d+$`)

// executeTrialRun runs one complete set of trials for all workflow specs.
// It is called (possibly multiple times) by RunWorkflowTrials via ExecuteWithRepeat.
func executeTrialRun(ctx context.Context, parsedSpecs []*WorkflowSpec, hostRepoSlug, logicalRepoSlug, cloneRepoSlug string, directTrialMode bool, opts TrialOptions) error {
	// Generate a unique datetime-ID for this trial session
	dateTimeID := fmt.Sprintf("%s-%d", time.Now().Format("20060102-150405"), time.Now().UnixNano()%1000000)
	trialLog.Printf("Starting trial run: dateTimeID=%s", dateTimeID)

	// Determine target repo slug for filenames once
	// In direct trial mode, use hostRepoSlug; otherwise use logicalRepoSlug
	targetRepoForFilename := executeTrialRunTargetRepoForFilename(hostRepoSlug, logicalRepoSlug, directTrialMode)

	// Step 3: Clone host repository to local temp directory
	tempDir, err := executeTrialRunCloneHostRepo(hostRepoSlug, opts.Verbose)
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to cleanup local temp directory: %v", err)))
		}
	}()

	// Step 4: Create trials directory
	if err := os.MkdirAll("trials", constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create trials directory: %w", err)
	}

	// Step 5: Run trials for each workflow
	var workflowResults []WorkflowTrialResult

	for _, parsedSpec := range parsedSpecs {
		result, err := executeTrialRunWorkflow(executeTrialRunWorkflowParams{
			Ctx:                   ctx,
			TempDir:               tempDir,
			ParsedSpec:            parsedSpec,
			HostRepoSlug:          hostRepoSlug,
			LogicalRepoSlug:       logicalRepoSlug,
			CloneRepoSlug:         cloneRepoSlug,
			DirectTrialMode:       directTrialMode,
			TargetRepoForFilename: targetRepoForFilename,
			DateTimeID:            dateTimeID,
			Opts:                  opts,
		})
		if err != nil {
			return err
		}
		workflowResults = append(workflowResults, result)
	}

	// Step 6: Save combined results for multi-workflow trials
	executeTrialRunSaveCombinedResults(parsedSpecs, workflowResults, targetRepoForFilename, dateTimeID, opts.Verbose)

	// Step 6.5: Copy trial results to host repository and commit them
	workflowNames := sliceutil.Map(parsedSpecs, func(spec *WorkflowSpec) string { return spec.WorkflowName })
	if err := copyTrialResultsToHostRepo(tempDir, dateTimeID, workflowNames, targetRepoForFilename, opts.Verbose); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to copy trial results to repository: %v", err)))
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("All trials completed successfully"))
	return nil
}

func executeTrialRunTargetRepoForFilename(hostRepoSlug, logicalRepoSlug string, directTrialMode bool) string {
	if directTrialMode {
		return hostRepoSlug
	}
	return logicalRepoSlug
}

func executeTrialRunCloneHostRepo(hostRepoSlug string, verbose bool) (string, error) {
	trialLog.Printf("Cloning trial host repository: %s", hostRepoSlug)
	tempDir, err := cloneTrialHostRepository(hostRepoSlug, verbose)
	if err != nil {
		return "", fmt.Errorf("failed to clone host repository: %w", err)
	}
	trialLog.Printf("Cloned repository to: %s", tempDir)
	return tempDir, nil
}

type executeTrialRunWorkflowParams struct {
	Ctx                   context.Context
	TempDir               string
	ParsedSpec            *WorkflowSpec
	HostRepoSlug          string
	LogicalRepoSlug       string
	CloneRepoSlug         string
	DirectTrialMode       bool
	TargetRepoForFilename string
	DateTimeID            string
	Opts                  TrialOptions
}

func executeTrialRunWorkflow(p executeTrialRunWorkflowParams) (WorkflowTrialResult, error) {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("=== Running trial for workflow: %s ===", p.ParsedSpec.WorkflowName)))

	// Install workflow with trial mode compilation
	if err := installWorkflowInTrialMode(p.Ctx, p.TempDir, p.ParsedSpec, p.LogicalRepoSlug, p.CloneRepoSlug, p.HostRepoSlug, p.DirectTrialMode, &p.Opts); err != nil {
		return WorkflowTrialResult{}, fmt.Errorf("failed to install workflow '%s' in trial mode: %w", p.ParsedSpec.WorkflowName, err)
	}
	executeTrialRunDisplayDescription(p.TempDir, p.ParsedSpec.WorkflowName)

	runID, err := executeTrialRunTriggerAndWait(p.Ctx, p.HostRepoSlug, p.ParsedSpec.WorkflowName, p.Opts)
	if err != nil {
		return WorkflowTrialResult{}, err
	}

	// Auto-merge PRs if requested
	if p.Opts.AutoMergePRs {
		if err := AutoMergePullRequestsLegacy(p.HostRepoSlug, p.Opts.Verbose); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to auto-merge pull requests: %v", err)))
		}
	}

	artifacts, err := downloadAllArtifacts(p.HostRepoSlug, runID, p.Opts.Verbose)
	if err != nil {
		return WorkflowTrialResult{}, fmt.Errorf("failed to download artifacts for '%s': %w", p.ParsedSpec.WorkflowName, err)
	}

	result := executeTrialRunResult(p.ParsedSpec.WorkflowName, runID, artifacts)
	executeTrialRunSaveIndividualResult(result, p.TargetRepoForFilename, p.DateTimeID, p.Opts.Verbose)
	executeTrialRunDisplayArtifacts(p.ParsedSpec.WorkflowName, artifacts)

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Trial completed for workflow: "+p.ParsedSpec.WorkflowName))
	return result, nil
}

func executeTrialRunDisplayDescription(tempDir, workflowName string) {
	// Display workflow description if present
	workflowPath := filepath.Join(tempDir, constants.GetWorkflowDir(), workflowName+".md")
	if description := ExtractWorkflowDescriptionFromFile(workflowPath); description != "" {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(description))
		fmt.Fprintln(os.Stderr, "")
	}
}

func executeTrialRunTriggerAndWait(ctx context.Context, hostRepoSlug, workflowName string, opts TrialOptions) (string, error) {
	// Run the workflow and wait for completion (with trigger context if provided)
	runID, err := triggerWorkflowRun(hostRepoSlug, workflowName, opts.TriggerContext, opts.Verbose)
	if err != nil {
		return "", fmt.Errorf("failed to trigger workflow run for '%s': %w", workflowName, err)
	}

	// Generate workflow run URL
	githubHost := getGitHubHost()
	workflowRunURL := fmt.Sprintf("%s/%s/actions/runs/%s", githubHost, hostRepoSlug, runID)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow run started with ID: %s (%s)", runID, workflowRunURL)))

	// Wait for workflow completion
	if err := WaitForWorkflowCompletion(ctx, hostRepoSlug, runID, opts.TimeoutMinutes, opts.Verbose); err != nil {
		// If the context was canceled or its deadline was exceeded, return that directly
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return "", err
		}
		return "", fmt.Errorf("workflow '%s' execution failed or timed out: %w", workflowName, err)
	}

	return runID, nil
}

func executeTrialRunResult(workflowName, runID string, artifacts *TrialArtifacts) WorkflowTrialResult {
	return WorkflowTrialResult{
		WorkflowName: workflowName,
		RunID:        runID,
		SafeOutputs:  artifacts.SafeOutputs,
		//AgentStdioLogs:      artifacts.AgentStdioLogs,
		AgenticRunInfo:      artifacts.AgenticRunInfo,
		AdditionalArtifacts: artifacts.AdditionalArtifacts,
		Timestamp:           time.Now(),
	}
}

func executeTrialRunSaveIndividualResult(result WorkflowTrialResult, targetRepoForFilename, dateTimeID string, verbose bool) {
	// Save individual trial file
	sanitizedTargetRepo := stringutil.SanitizeForFilename(targetRepoForFilename)
	individualFilename := fmt.Sprintf("trials/%s-%s.%s.json", result.WorkflowName, sanitizedTargetRepo, dateTimeID)
	if err := saveTrialResult(individualFilename, result, verbose); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to save individual trial result: %v", err)))
	}
}

func executeTrialRunDisplayArtifacts(workflowName string, artifacts *TrialArtifacts) {
	// Display safe outputs to stdout
	if len(artifacts.SafeOutputs) > 0 {
		outputBytes, _ := json.MarshalIndent(artifacts.SafeOutputs, "", "  ")
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("=== Safe Outputs from %s ===", workflowName)))
		fmt.Fprintln(os.Stdout, string(outputBytes))
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("=== End of Safe Outputs ==="))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("=== No Safe Outputs Generated by %s ===", workflowName)))
	}

	// Display additional artifact information if available
	// if len(artifacts.AgentStdioLogs) > 0 {
	// 	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("=== Agent Stdio Logs Available from %s (%d files) ===", workflowName, len(artifacts.AgentStdioLogs))))
	// }
	if len(artifacts.AgenticRunInfo) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("=== Agentic Run Information Available from %s ===", workflowName)))
	}
	if len(artifacts.AdditionalArtifacts) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("=== Additional Artifacts Available from %s (%d files) ===", workflowName, len(artifacts.AdditionalArtifacts))))
	}
}

func executeTrialRunSaveCombinedResults(parsedSpecs []*WorkflowSpec, workflowResults []WorkflowTrialResult, targetRepoForFilename, dateTimeID string, verbose bool) {
	if len(parsedSpecs) <= 1 {
		return
	}

	workflowNames := sliceutil.Map(parsedSpecs, func(spec *WorkflowSpec) string { return spec.WorkflowName })
	workflowNamesStr := strings.Join(workflowNames, "-")
	sanitizedTargetRepo := stringutil.SanitizeForFilename(targetRepoForFilename)
	combinedFilename := fmt.Sprintf("trials/%s-%s.%s.json", workflowNamesStr, sanitizedTargetRepo, dateTimeID)
	combinedResult := CombinedTrialResult{
		WorkflowNames: workflowNames,
		Results:       workflowResults,
		Timestamp:     time.Now(),
	}
	if err := saveTrialResult(combinedFilename, combinedResult, verbose); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to save combined trial result: %v", err)))
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Combined results saved to: "+combinedFilename))
}

func triggerWorkflowRun(repoSlug, workflowName string, triggerContext string, verbose bool) (string, error) {
	trialLog.Printf("Triggering workflow run: workflow=%s, repo=%s, hasTriggerContext=%v", workflowName, repoSlug, triggerContext != "")
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
// - GitHub issue URLs: https://github.com/owner/repo/issues/123 (public GitHub)
// - GitHub Enterprise issue URLs: https://example.ghe.com/owner/repo/issues/123 (GHES, respects GH_HOST)
// - Issue references: #123
// - Plain numbers: 123
func parseIssueSpec(input string) string {
	input = strings.TrimSpace(input)

	// First try to match GitHub issue URLs (supports public GitHub and GHES via GH_HOST).
	// Both the scheme and host are compared against the configured GitHub host so that
	// HTTP-only GHES instances and unrelated hosts are handled correctly.
	if u, err := url.Parse(input); err == nil && u.Host != "" {
		configuredHostURL, err := url.Parse(getGitHubHost())
		if err == nil && u.Scheme == configuredHostURL.Scheme && u.Host == configuredHostURL.Host {
			if matches := issuePathPattern.FindStringSubmatch(u.Path); len(matches) >= 2 {
				return matches[1]
			}
		}
	}

	// Try to match issue references like #123
	if matches := issueRefPattern.FindStringSubmatch(input); len(matches) >= 2 {
		return matches[1]
	}

	// Try to match plain numbers like 123
	if issueNumberPattern.MatchString(input) {
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

	if err := os.WriteFile(filename, jsonBytes, constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write result file: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Saved trial result to: "+filename))
	}

	return nil
}

// copyTrialResultsToHostRepo copies trial result files to the host repository and commits them
func copyTrialResultsToHostRepo(tempDir, dateTimeID string, workflowNames []string, targetRepoSlug string, verbose bool) error {
	trialLog.Printf("Copying trial results to host repo: workflows=%d, dateTimeID=%s, targetRepo=%s", len(workflowNames), dateTimeID, targetRepoSlug)
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Copying trial results to host repository"))
	}

	// Create trials directory in the host repository
	trialsDir := filepath.Join(tempDir, "trials")
	if err := os.MkdirAll(trialsDir, constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create trials directory in repository: %w", err)
	}

	copyTrialResultsToHostRepoFiles(trialsDir, dateTimeID, workflowNames, targetRepoSlug, verbose)
	if err := copyTrialResultsToHostRepoCommitAndPush(tempDir, dateTimeID, workflowNames, verbose); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Trial results copied to repository and pushed"))

	return nil
}

func copyTrialResultsToHostRepoFiles(trialsDir, dateTimeID string, workflowNames []string, targetRepoSlug string, verbose bool) {
	sanitizedTargetRepo := stringutil.SanitizeForFilename(targetRepoSlug)
	copyTrialResultsToHostRepoIndividualFiles(trialsDir, dateTimeID, workflowNames, sanitizedTargetRepo, verbose)
	copyTrialResultsToHostRepoCombinedFile(trialsDir, dateTimeID, workflowNames, sanitizedTargetRepo, verbose)
}

func copyTrialResultsToHostRepoIndividualFiles(trialsDir, dateTimeID string, workflowNames []string, sanitizedTargetRepo string, verbose bool) {
	// Copy individual workflow result files
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
}

func copyTrialResultsToHostRepoCombinedFile(trialsDir, dateTimeID string, workflowNames []string, sanitizedTargetRepo string, verbose bool) {
	// Copy combined results file if it exists (for multi-workflow trials)
	if len(workflowNames) <= 1 {
		return
	}

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

func copyTrialResultsToHostRepoCommitAndPush(tempDir, dateTimeID string, workflowNames []string, verbose bool) error {
	restoreDir, err := copyTrialResultsToHostRepoChdir(tempDir)
	if err != nil {
		return err
	}
	defer restoreDir()

	// Add trial results to git
	cmd := exec.Command("git", "add", "trials/")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add trial results: %w (output: %s)", err, string(output))
	}

	changed, err := copyTrialResultsToHostRepoHasChanges(verbose)
	if err != nil || !changed {
		return err
	}

	commitMsg := fmt.Sprintf("Add trial results for %s (%s)", strings.Join(workflowNames, ", "), dateTimeID)
	cmd = exec.Command("git", "commit", "-m", commitMsg)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit trial results: %w (output: %s)", err, string(output))
	}

	return copyTrialResultsToHostRepoPullAndPush(tempDir, verbose)
}

func copyTrialResultsToHostRepoChdir(tempDir string) (func(), error) {
	// Change to temp directory to commit the changes
	originalDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		return nil, fmt.Errorf("failed to change to temp directory: %w", err)
	}
	return func() {
		_ = os.Chdir(originalDir)
	}, nil
}

func copyTrialResultsToHostRepoHasChanges(verbose bool) (bool, error) {
	// Check if there are any changes to commit
	statusCmd := exec.Command("git", "status", "--porcelain", "trials/")
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}

	// If no changes, skip commit and push
	if strings.TrimSpace(string(statusOutput)) == "" {
		trialLog.Print("No new trial results to commit, skipping push")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No new trial results to commit"))
		}
		return false, nil
	}
	return true, nil
}

func copyTrialResultsToHostRepoPullAndPush(tempDir string, verbose bool) error {
	branch, err := getCurrentBranchIn(tempDir)
	if err != nil {
		trialLog.Printf("Failed to detect branch in %s: %v, falling back to main", tempDir, err)
		branch = "main"
	}
	// Pull latest changes before pushing to avoid conflicts
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Pulling latest changes from "+branch+" branch"))
	}
	cmd := exec.Command("git", "pull", "--rebase", "origin", branch)
	cmd.Dir = tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to pull latest changes: %w (output: %s)", err, string(output))
	}

	// Push to current branch
	cmd = exec.Command("git", "push", "origin", branch)
	cmd.Dir = tempDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push trial results: %w (output: %s)", err, string(output))
	}
	return nil
}
