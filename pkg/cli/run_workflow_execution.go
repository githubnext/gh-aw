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
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var executionLog = logger.New("cli:run_workflow_execution")

// workflowCompletionWaitTimeoutMinutes matches the GitHub Actions maximum job runtime.
const workflowCompletionWaitTimeoutMinutes = 6 * 60

// betweenWorkflowsDelay paces sequential workflow triggers to avoid overwhelming the GitHub API.
const betweenWorkflowsDelay = 1 * time.Second

// RunOptions contains all configuration options for running workflows
type RunOptions struct {
	Enable            bool     // Enable the workflow if it's disabled
	EngineOverride    string   // Override AI engine
	RepoOverride      string   // Target repository (owner/repo format)
	RefOverride       string   // Branch or tag name
	AutoMergePRs      bool     // Auto-merge PRs created during execution
	Push              bool     // Commit and push workflow files before running
	WaitForCompletion bool     // Wait for workflow completion
	RepeatCount       int      // Number of times to repeat (0 = run once)
	Inputs            []string // Workflow inputs in key=value format
	Verbose           bool     // Enable verbose output
	DryRun            bool     // Validate without actually triggering
	JSON              bool     // Output results in JSON format
	Approve           bool     // Approve safe update changes during compilation
}

// WorkflowRunResult contains the result of a single workflow run trigger for JSON output
type WorkflowRunResult struct {
	Workflow string `json:"workflow"`
	LockFile string `json:"lock_file"`
	Status   string `json:"status"` // "triggered", "dry_run", "error"
	RunID    int64  `json:"run_id,omitempty"`
	RunURL   string `json:"run_url,omitempty"`
	Error    string `json:"error,omitempty"`
}

// RunWorkflowOnGitHub runs an agentic workflow on GitHub Actions
func RunWorkflowOnGitHub(ctx context.Context, workflowIdOrName string, opts RunOptions) error {
	executionLog.Printf("Starting workflow run: workflow=%s, enable=%v, engineOverride=%s, repo=%s, ref=%s, push=%v, wait=%v, inputs=%v", workflowIdOrName, opts.Enable, opts.EngineOverride, opts.RepoOverride, opts.RefOverride, opts.Push, opts.WaitForCompletion, opts.Inputs)
	if err := runWorkflowOnGitHubValidateStart(ctx, workflowIdOrName, opts); err != nil {
		return err
	}
	if err := runWorkflowOnGitHubValidateWorkflow(workflowIdOrName, opts); err != nil {
		return err
	}

	wasDisabled, workflowID, err := runWorkflowOnGitHubEnableWorkflow(workflowIdOrName, opts)
	if err != nil {
		return err
	}
	lockFileName, lockFilePath, err := runWorkflowOnGitHubResolveLockFile(workflowIdOrName, opts)
	if err != nil {
		return err
	}
	if err := runWorkflowOnGitHubPrepareLocal(ctx, workflowIdOrName, lockFileName, lockFilePath, opts); err != nil {
		return err
	}
	args, ref := runWorkflowOnGitHubBuildArgs(lockFileName, opts)
	workflowStartTime := time.Now()

	if opts.DryRun {
		return runWorkflowOnGitHubDryRun(workflowIdOrName, lockFileName, ref, opts, wasDisabled, workflowID)
	}
	runInfo, runErr, err := runWorkflowOnGitHubTrigger(ctx, workflowIdOrName, lockFileName, args, ref, opts, wasDisabled, workflowID)
	if err != nil {
		return err
	}
	if err := runWorkflowOnGitHubAfterTrigger(ctx, workflowIdOrName, runInfo, runErr, workflowStartTime, opts); err != nil {
		return err
	}
	if opts.Enable && wasDisabled && workflowID != 0 {
		restoreWorkflowState(workflowIdOrName, workflowID, opts.RepoOverride, opts.Verbose)
	}
	return nil
}

func runWorkflowOnGitHubValidateStart(ctx context.Context, workflowIdOrName string, opts RunOptions) error {
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
	}
	if workflowIdOrName == "" {
		return errors.New("workflow name or ID is required")
	}
	for _, input := range opts.Inputs {
		if !strings.Contains(input, "=") {
			return fmt.Errorf("invalid input format '%s': expected key=value", input)
		}
		parts := strings.SplitN(input, "=", 2)
		if parts[0] == "" {
			return fmt.Errorf("invalid input format '%s': key cannot be empty", input)
		}
	}
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Running workflow on GitHub Actions: "+workflowIdOrName))
	}
	if !isGHCLIAvailable() {
		return errors.New("GitHub CLI (gh) is required but not available")
	}
	return nil
}

func runWorkflowOnGitHubValidateWorkflow(workflowIdOrName string, opts RunOptions) error {
	if opts.RepoOverride != "" {
		executionLog.Printf("Validating remote workflow: %s in repo %s", workflowIdOrName, opts.RepoOverride)
		if err := validateRemoteWorkflow(workflowIdOrName, opts.RepoOverride, opts.Verbose); err != nil {
			return fmt.Errorf("failed to validate remote workflow: %w", err)
		}
		return nil
	}
	executionLog.Printf("Validating local workflow: %s", workflowIdOrName)
	workflowFile, err := resolveWorkflowFile(workflowIdOrName, opts.Verbose)
	if err != nil {
		return err
	}
	runnable, err := IsRunnable(workflowFile)
	if err != nil {
		return fmt.Errorf("failed to check if workflow %s is runnable: %w", workflowFile, err)
	}
	if !runnable {
		return fmt.Errorf("workflow '%s' cannot be run on GitHub Actions - it must have 'workflow_dispatch' trigger", workflowIdOrName)
	}
	executionLog.Printf("Workflow is runnable: %s", workflowFile)
	if err := validateWorkflowInputs(workflowFile, opts.Inputs); err != nil {
		return fmt.Errorf("%w", err)
	}
	runWorkflowOnGitHubWarnLocalStatus(workflowFile)
	return nil
}

func runWorkflowOnGitHubWarnLocalStatus(workflowFile string) {
	status, err := checkWorkflowFileStatus(workflowFile)
	if err != nil || status == nil {
		return
	}
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

func runWorkflowOnGitHubEnableWorkflow(workflowIdOrName string, opts RunOptions) (bool, int64, error) {
	if !opts.Enable {
		return false, 0, nil
	}
	wf, err := getWorkflowStatus(workflowIdOrName, opts.RepoOverride, opts.Verbose)
	if err != nil {
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not check workflow status: %v", err)))
		}
		return false, 0, nil
	}
	if wf.State != "disabled_manually" {
		executionLog.Printf("Workflow %s is already enabled (state=%s)", workflowIdOrName, wf.State)
		return false, wf.ID, nil
	}
	executionLog.Printf("Workflow %s is disabled, temporarily enabling for this run (id=%d)", workflowIdOrName, wf.ID)
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow '%s' is disabled, enabling it temporarily...", workflowIdOrName)))
	}
	enableArgs := []string{"workflow", "enable", strconv.FormatInt(wf.ID, 10)}
	if opts.RepoOverride != "" {
		enableArgs = append(enableArgs, "--repo", opts.RepoOverride)
	}
	if err := workflow.ExecGH(enableArgs...).Run(); err != nil {
		executionLog.Printf("Failed to enable workflow %s: %v", workflowIdOrName, err)
		return false, 0, fmt.Errorf("failed to enable workflow '%s': %w", workflowIdOrName, err)
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Enabled workflow: "+workflowIdOrName))
	return true, wf.ID, nil
}

func runWorkflowOnGitHubResolveLockFile(workflowIdOrName string, opts RunOptions) (string, string, error) {
	normalizedID := normalizeWorkflowID(workflowIdOrName)
	lockFileName := normalizedID + ".lock.yml"
	var lockFilePath string
	if opts.RepoOverride == "" {
		workflowsDir := getWorkflowsDir()
		_, _, err := readWorkflowFile(normalizedID+".md", workflowsDir)
		if err != nil {
			return "", "", fmt.Errorf("failed to find workflow in local %s: %w", workflowsDir, err)
		}
		lockFilePath = filepath.Join(constants.GetWorkflowDir(), lockFileName)
		if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
			executionLog.Printf("Lock file not found: %s (workflow must be compiled first)", lockFilePath)
			suggestions := []string{
				fmt.Sprintf("Run '%s compile' to compile all workflows", string(constants.CLIExtensionPrefix)),
				fmt.Sprintf("Run '%s compile %s' to compile this specific workflow", string(constants.CLIExtensionPrefix), normalizedID),
			}
			errMsg := console.FormatErrorWithSuggestions(fmt.Sprintf("workflow lock file '%s' not found in %s", lockFileName, constants.GetWorkflowDir()), suggestions)
			return "", "", errors.New(errMsg)
		}
		executionLog.Printf("Found lock file: %s", lockFilePath)
	}
	return lockFileName, lockFilePath, nil
}

func runWorkflowOnGitHubPrepareLocal(ctx context.Context, workflowIdOrName, lockFileName, lockFilePath string, opts RunOptions) error {
	if opts.EngineOverride != "" && opts.RepoOverride == "" {
		if err := runWorkflowOnGitHubRecompileWithEngine(ctx, lockFilePath, opts); err != nil {
			return err
		}
	} else if opts.EngineOverride != "" && opts.RepoOverride != "" && opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Note: Engine override ignored for remote repository workflows"))
	}
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Using lock file: "+lockFileName))
	}
	runWorkflowOnGitHubWarnLockStatus(workflowIdOrName, lockFilePath, opts)
	if opts.Push {
		return runWorkflowOnGitHubPushFiles(ctx, workflowIdOrName, lockFilePath, opts)
	}
	return nil
}

func runWorkflowOnGitHubRecompileWithEngine(ctx context.Context, lockFilePath string, opts RunOptions) error {
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Recompiling workflow with engine override: "+opts.EngineOverride))
	}
	workflowMarkdownPath := stringutil.LockFileToMarkdown(lockFilePath)
	config := CompileConfig{MarkdownFiles: []string{workflowMarkdownPath}, Verbose: opts.Verbose, EngineOverride: opts.EngineOverride, Validate: true, Strict: false}
	if _, err := CompileWorkflows(ctx, config); err != nil {
		return fmt.Errorf("failed to recompile workflow with engine override: %w", err)
	}
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Successfully recompiled workflow with engine: "+opts.EngineOverride))
	}
	return nil
}

func runWorkflowOnGitHubWarnLockStatus(workflowIdOrName, lockFilePath string, opts RunOptions) {
	if opts.Push || opts.RepoOverride != "" {
		return
	}
	workflowMarkdownPath := stringutil.LockFileToMarkdown(lockFilePath)
	if status, err := checkLockFileStatus(workflowMarkdownPath); err == nil {
		if status.Missing {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Lock file is missing"))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Run 'gh aw run %s --push' to automatically compile and push the lock file", workflowIdOrName)))
		} else if status.Outdated {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Lock file is outdated (workflow file is newer)"))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Run 'gh aw run %s --push' to automatically compile and push the lock file", workflowIdOrName)))
		}
	}
}

func runWorkflowOnGitHubPushFiles(ctx context.Context, workflowIdOrName, lockFilePath string, opts RunOptions) error {
	if opts.RepoOverride != "" {
		return errors.New("--push flag is only supported for local workflows, not remote repositories")
	}
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Collecting workflow files for push..."))
	}
	workflowMarkdownPath := stringutil.LockFileToMarkdown(lockFilePath)
	files, err := collectWorkflowFiles(ctx, workflowMarkdownPath, opts.Verbose, opts.Approve)
	if err != nil {
		return fmt.Errorf("failed to collect workflow files: %w", err)
	}
	if err := pushWorkflowFiles(ctx, workflowIdOrName, files, opts.RefOverride, opts.Verbose); err != nil {
		return fmt.Errorf("failed to push workflow files: %w", err)
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully pushed %d file(s) for workflow %s", len(files), workflowIdOrName)))
	return nil
}

func runWorkflowOnGitHubBuildArgs(lockFileName string, opts RunOptions) ([]string, string) {
	args := []string{"workflow", "run", lockFileName}
	if opts.RepoOverride != "" {
		args = append(args, "--repo", opts.RepoOverride)
	}
	ref := opts.RefOverride
	if ref == "" && opts.RepoOverride == "" {
		if currentBranch, err := getCurrentBranch(); err == nil {
			ref = currentBranch
			executionLog.Printf("Using current branch for workflow run: %s", ref)
		} else if opts.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Note: Could not determine current branch: %v", err)))
		}
	}
	if ref != "" {
		args = append(args, "--ref", ref)
	}
	for _, input := range opts.Inputs {
		args = append(args, "-f", input)
	}
	return args, ref
}

func runWorkflowOnGitHubDryRun(workflowIdOrName, lockFileName, ref string, opts RunOptions, wasDisabled bool, workflowID int64) error {
	if opts.Verbose {
		cmdParts := runWorkflowOnGitHubCommandParts(lockFileName, ref, opts)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Dry run mode - command that would be executed:"))
		fmt.Fprintln(os.Stderr, console.FormatCommandMessage(strings.Join(cmdParts, " ")))
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Validation passed for workflow: %s (dry run - not executed)", lockFileName)))
	if opts.Enable && wasDisabled && workflowID != 0 {
		restoreWorkflowState(workflowIdOrName, workflowID, opts.RepoOverride, opts.Verbose)
	}
	return nil
}

func runWorkflowOnGitHubCommandParts(lockFileName, ref string, opts RunOptions) []string {
	cmdParts := []string{"gh workflow run", lockFileName}
	if opts.RepoOverride != "" {
		cmdParts = append(cmdParts, "--repo", opts.RepoOverride)
	}
	if ref != "" {
		cmdParts = append(cmdParts, "--ref", ref)
	}
	for _, input := range opts.Inputs {
		cmdParts = append(cmdParts, "-f", input)
	}
	return cmdParts
}

func runWorkflowOnGitHubTrigger(ctx context.Context, workflowIdOrName, lockFileName string, args []string, ref string, opts RunOptions, wasDisabled bool, workflowID int64) (*WorkflowRunInfo, error, error) {
	cmd := workflow.ExecGHContext(ctx, args...)
	if opts.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatCommandMessage(strings.Join(runWorkflowOnGitHubCommandParts(lockFileName, ref, opts), " ")))
	}
	stdout, err := cmd.Output()
	if err != nil {
		return nil, nil, runWorkflowOnGitHubHandleTriggerError(err, workflowIdOrName, opts, wasDisabled, workflowID)
	}
	output := strings.TrimSpace(string(stdout))
	if output != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(output))
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Successfully triggered workflow: "+lockFileName))
	executionLog.Printf("Workflow triggered successfully: %s", lockFileName)
	return runWorkflowOnGitHubRunInfo(lockFileName, opts, output)
}

func runWorkflowOnGitHubHandleTriggerError(err error, workflowIdOrName string, opts RunOptions, wasDisabled bool, workflowID int64) error {
	var stderrOutput string
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		stderrOutput = string(exitError.Stderr)
		fmt.Fprintf(os.Stderr, "%s", exitError.Stderr)
	}
	if opts.Enable && wasDisabled && workflowID != 0 {
		restoreWorkflowState(workflowIdOrName, workflowID, opts.RepoOverride, opts.Verbose)
	}
	errorMsg := err.Error() + " " + stderrOutput
	if isRunningInCodespace() && is403PermissionError(errorMsg) {
		fmt.Fprint(os.Stderr, getCodespacePermissionErrorMessage())
		return errors.New("failed to run workflow on GitHub Actions: permission denied (403)")
	}
	return fmt.Errorf("failed to run workflow on GitHub Actions: %w", err)
}

func runWorkflowOnGitHubRunInfo(lockFileName string, opts RunOptions, output string) (*WorkflowRunInfo, error, error) {
	var runInfo *WorkflowRunInfo
	var runErr error
	if parsedRunInfo := parseRunInfoFromOutput(output); parsedRunInfo != nil {
		executionLog.Printf("Parsed run info from gh output: id=%d, url=%s", parsedRunInfo.DatabaseID, parsedRunInfo.URL)
		runInfo = parsedRunInfo
	} else {
		runInfo, runErr = getLatestWorkflowRunWithRetry(lockFileName, opts.RepoOverride, opts.Verbose)
	}
	return runInfo, runErr, nil
}

func runWorkflowOnGitHubAfterTrigger(ctx context.Context, workflowIdOrName string, runInfo *WorkflowRunInfo, runErr error, workflowStartTime time.Time, opts RunOptions) error {
	if runErr == nil && runInfo.URL != "" {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("🔗 View workflow run: "+runInfo.URL))
		executionLog.Printf("Workflow run URL: %s (ID: %d)", runInfo.URL, runInfo.DatabaseID)
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("💡 To analyze this run, use: %s audit %d", string(constants.CLIExtensionPrefix), runInfo.DatabaseID)))
	} else if opts.Verbose && runErr != nil {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Note: Could not get workflow run URL: %v", runErr)))
	}
	if opts.WaitForCompletion || opts.AutoMergePRs {
		return runWorkflowOnGitHubWaitAfterTrigger(ctx, runInfo, runErr, workflowStartTime, opts)
	}
	return nil
}

func runWorkflowOnGitHubWaitAfterTrigger(ctx context.Context, runInfo *WorkflowRunInfo, runErr error, workflowStartTime time.Time, opts RunOptions) error {
	if runErr != nil {
		if opts.AutoMergePRs {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not get workflow run information for auto-merge: %v", runErr)))
		} else if opts.WaitForCompletion {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not get workflow run information: %v", runErr)))
		}
		return nil
	}
	targetRepo := runWorkflowOnGitHubTargetRepo(opts)
	if targetRepo == "" {
		return nil
	}
	if opts.AutoMergePRs {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Auto-merge PRs enabled - waiting for workflow completion..."))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Waiting for workflow completion..."))
	}
	runIDStr := strconv.FormatInt(runInfo.DatabaseID, 10)
	if err := WaitForWorkflowCompletion(ctx, targetRepo, runIDStr, workflowCompletionWaitTimeoutMinutes, opts.Verbose); err != nil {
		if ctx.Err() != nil || errors.Is(err, ErrInterrupted) {
			return err
		}
		runWorkflowOnGitHubPrintCompletionWarning(err, opts)
	} else if opts.AutoMergePRs {
		if err := AutoMergePullRequestsCreatedAfter(targetRepo, workflowStartTime, opts.Verbose); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to auto-merge pull requests: %v", err)))
		}
	}
	return nil
}

func runWorkflowOnGitHubTargetRepo(opts RunOptions) string {
	targetRepo := opts.RepoOverride
	if targetRepo != "" {
		return targetRepo
	}
	currentRepo, err := GetCurrentRepoSlug()
	if err != nil {
		if opts.AutoMergePRs {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not determine target repository for auto-merge: %v", err)))
		}
		return ""
	}
	return currentRepo
}

func runWorkflowOnGitHubPrintCompletionWarning(err error, opts RunOptions) {
	if opts.AutoMergePRs {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Workflow did not complete successfully, skipping auto-merge: %v", err)))
	} else {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Workflow did not complete successfully: %v", err)))
	}
}

// RunWorkflowsOnGitHub runs multiple agentic workflows on GitHub Actions, optionally repeating a specified number of times
func RunWorkflowsOnGitHub(ctx context.Context, workflowNames []string, opts RunOptions) error {
	if len(workflowNames) == 0 {
		return errors.New("at least one workflow name or ID is required")
	}
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
	}
	if err := runWorkflowsOnGitHubValidate(workflowNames, opts); err != nil {
		return err
	}
	runAllWorkflows := func() error {
		return runWorkflowsOnGitHubRunAll(ctx, workflowNames, opts)
	}
	if opts.JSON {
		runAllWorkflows = runWorkflowsOnGitHubJSONWrapper(runAllWorkflows, workflowNames, opts)
	}
	return ExecuteWithRepeat(RepeatOptions{
		Ctx:           ctx,
		RepeatCount:   opts.RepeatCount,
		RepeatMessage: "Repeating workflow run",
		ExecuteFunc:   runAllWorkflows,
		UseStderr:     false,
	})
}

func runWorkflowsOnGitHubValidate(workflowNames []string, opts RunOptions) error {
	for _, workflowName := range workflowNames {
		if workflowName == "" {
			return errors.New("workflow name cannot be empty")
		}
		if opts.RepoOverride != "" {
			if err := validateRemoteWorkflow(workflowName, opts.RepoOverride, opts.Verbose); err != nil {
				return fmt.Errorf("failed to validate remote workflow '%s': %w", workflowName, err)
			}
			continue
		}
		if err := runWorkflowsOnGitHubValidateLocal(workflowName, opts); err != nil {
			return err
		}
	}
	return nil
}

func runWorkflowsOnGitHubValidateLocal(workflowName string, opts RunOptions) error {
	workflowFile, err := resolveWorkflowFile(workflowName, opts.Verbose)
	if err != nil {
		return err
	}
	runnable, err := IsRunnable(workflowFile)
	if err != nil {
		return fmt.Errorf("failed to check if workflow '%s' is runnable: %w", workflowName, err)
	}
	if !runnable {
		return fmt.Errorf("workflow '%s' cannot be run on GitHub Actions - it must have 'workflow_dispatch' trigger", workflowName)
	}
	return nil
}

func runWorkflowsOnGitHubRunAll(ctx context.Context, workflowNames []string, opts RunOptions) error {
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Running %d workflow(s)...", len(workflowNames))))
	for i, workflowName := range workflowNames {
		if err := runWorkflowsOnGitHubRunOne(ctx, workflowNames, workflowName, i, opts); err != nil {
			return err
		}
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Successfully triggered %d workflow(s)", len(workflowNames))))
	return nil
}

func runWorkflowsOnGitHubRunOne(ctx context.Context, workflowNames []string, workflowName string, i int, opts RunOptions) error {
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	default:
	}
	if len(workflowNames) > 1 {
		fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Running workflow %d/%d: %s", i+1, len(workflowNames), workflowName)))
	}
	workflowOpts := opts
	if opts.RepeatCount > 0 {
		workflowOpts.WaitForCompletion = true
	}
	if err := RunWorkflowOnGitHub(ctx, workflowName, workflowOpts); err != nil {
		return fmt.Errorf("failed to run workflow '%s': %w", workflowName, err)
	}
	return runWorkflowsOnGitHubDelay(ctx, workflowNames, i)
}

func runWorkflowsOnGitHubDelay(ctx context.Context, workflowNames []string, i int) error {
	if i >= len(workflowNames)-1 {
		return nil
	}
	timer := time.NewTimer(betweenWorkflowsDelay)
	select {
	case <-ctx.Done():
		timer.Stop()
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Operation cancelled"))
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func runWorkflowsOnGitHubJSONWrapper(runAllWorkflows func() error, workflowNames []string, opts RunOptions) func() error {
	return func() error {
		results := runWorkflowsOnGitHubJSONResults(workflowNames, opts)
		runErr := runAllWorkflows()
		if runErr != nil {
			for i := range results {
				results[i].Status = "error"
				results[i].Error = runErr.Error()
			}
		}
		jsonBytes, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Fprintln(os.Stdout, string(jsonBytes))
		return runErr
	}
}

func runWorkflowsOnGitHubJSONResults(workflowNames []string, opts RunOptions) []WorkflowRunResult {
	var results []WorkflowRunResult
	for _, workflowName := range workflowNames {
		normalizedID := normalizeWorkflowID(workflowName)
		status := "triggered"
		if opts.DryRun {
			status = "dry_run"
		}
		results = append(results, WorkflowRunResult{Workflow: normalizedID, LockFile: normalizedID + ".lock.yml", Status: status})
	}
	return results
}

// runInfoURLRegexp matches GitHub Actions run URLs of the form:
// https://{host}/{owner}/{repo}/actions/runs/{run_id}
// Supports both public GitHub (github.com) and GitHub Enterprise Server deployments.
var runInfoURLRegexp = regexp.MustCompile(`https://[^/\s]+/[^/\s]+/[^/\s]+/actions/runs/(\d+)`)

// parseRunInfoFromOutput tries to extract workflow run information from the
// output of `gh workflow run` (v2.87+), which now returns the run URL.
// Returns nil if the run URL cannot be found in the output.
func parseRunInfoFromOutput(output string) *WorkflowRunInfo {
	matches := runInfoURLRegexp.FindStringSubmatch(output)
	if len(matches) < 2 {
		return nil
	}
	runID, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return nil
	}
	return &WorkflowRunInfo{
		URL:        matches[0],
		DatabaseID: runID,
	}
}
