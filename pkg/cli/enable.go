package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/setutil"
	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var enableLog = logger.New("cli:enable")

type toggleWorkflowsByNamesTarget struct {
	Name           string
	ID             int64  // 0 if unknown
	LockFileBase   string // e.g., dev.lock.yml
	CurrentState   string // known state or "unknown"
	HasGitHubEntry bool
}

// EnableWorkflowsByNames enables workflows by specific names, or all if no names provided
func EnableWorkflowsByNames(ctx context.Context, workflowNames []string, repoOverride string) error {
	enableLog.Printf("EnableWorkflowsByNames called: workflow_count=%d, repo=%s", len(workflowNames), repoOverride)
	return toggleWorkflowsByNames(ctx, workflowNames, true, repoOverride)
}

// DisableWorkflowsByNames disables workflows by specific names, or all if no names provided
func DisableWorkflowsByNames(ctx context.Context, workflowNames []string, repoOverride string) error {
	enableLog.Printf("DisableWorkflowsByNames called: workflow_count=%d, repo=%s", len(workflowNames), repoOverride)
	return toggleWorkflowsByNames(ctx, workflowNames, false, repoOverride)
}

// toggleWorkflowsByNames toggles workflows by specific names, or all if no names provided
func toggleWorkflowsByNames(ctx context.Context, workflowNames []string, enable bool, repoOverride string) error {
	action := toggleWorkflowsByNamesAction(enable)
	enableLog.Printf("Toggle workflows: action=%s, count=%d, repo=%s", action, len(workflowNames), repoOverride)

	// If no specific workflow names provided, enable/disable all workflows
	if len(workflowNames) == 0 {
		return toggleWorkflowsByNamesAll(ctx, action, enable, repoOverride)
	}

	// Check if gh CLI is available
	if !isGHCLIAvailable() {
		return errors.New("GitHub CLI (gh) is required but not available")
	}

	// Get the core set of workflows from markdown files in .github/workflows
	mdFiles, err := getMarkdownWorkflowFiles("")
	if err != nil {
		return fmt.Errorf("no workflow files found to %s: %w", action, err)
	}

	githubWorkflows := toggleWorkflowsByNamesFetchGitHub(repoOverride)
	targets, notFoundNames := toggleWorkflowsByNamesResolveTargets(ctx, workflowNames, mdFiles, githubWorkflows, enable)

	// Report any workflows that weren't found
	if len(notFoundNames) > 0 {
		return toggleWorkflowsByNamesNotFoundError(notFoundNames)
	}

	// If no targets after filtering, everything was already in the desired state
	if len(targets) == 0 {
		enableLog.Printf("No workflows need to be %sd - all already in desired state", action)
		fmt.Fprintf(os.Stderr, "All specified workflows are already %sd\n", action)
		return nil
	}

	enableLog.Printf("Proceeding to %s %d workflows", action, len(targets))
	// Show what will be changed
	toggleWorkflowsByNamesPrintTargets(action, targets)

	// Perform the action
	var failures []string
	for _, t := range targets {
		if ok := toggleWorkflowsByNamesRunTarget(t, action, enable, repoOverride); !ok {
			failures = append(failures, t.Name)
		}
	}

	// Return error if any workflows failed to be processed
	if len(failures) > 0 {
		if enable {
			return fmt.Errorf("failed to enable %d workflow(s): %s", len(failures), strings.Join(failures, ", "))
		} else {
			return fmt.Errorf("failed to disable %d workflow(s): %s", len(failures), strings.Join(failures, ", "))
		}
	}

	return nil
}

func toggleWorkflowsByNamesAction(enable bool) string {
	if enable {
		return "enable"
	}
	return "disable"
}

func toggleWorkflowsByNamesAll(ctx context.Context, action string, enable bool, repoOverride string) error {
	enableLog.Print("No specific workflows provided, processing all workflows")
	fmt.Fprintf(os.Stderr, "No specific workflows provided. %sing all workflows...\n", strings.ToUpper(action[:1])+action[1:])
	// Get all workflow names and process them
	mdFiles, err := getMarkdownWorkflowFiles("")
	if err != nil {
		return fmt.Errorf("no workflow files found to %s: %w", action, err)
	}
	if len(mdFiles) == 0 {
		return fmt.Errorf("no markdown workflow files found to %s", action)
	}

	// Extract all workflow names and recursively call with all workflow names
	allWorkflowNames := make([]string, 0, len(mdFiles))
	for _, file := range mdFiles {
		base := filepath.Base(file)
		allWorkflowNames = append(allWorkflowNames, normalizeWorkflowID(base))
	}
	return toggleWorkflowsByNames(ctx, allWorkflowNames, enable, repoOverride)
}

func toggleWorkflowsByNamesFetchGitHub(repoOverride string) map[string]*GitHubWorkflow {
	// Get GitHub workflows status for comparison; warn but continue if unavailable
	enableLog.Print("Fetching GitHub workflows status for comparison")
	githubWorkflows, err := fetchGitHubWorkflows(repoOverride, false)
	if err != nil {
		enableLog.Printf("Failed to fetch GitHub workflows: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Unable to fetch GitHub workflows (gh CLI may not be authenticated): %v", err)))
		githubWorkflows = make(map[string]*GitHubWorkflow)
	}
	enableLog.Printf("Retrieved %d GitHub workflows from remote", len(githubWorkflows))
	return githubWorkflows
}

func toggleWorkflowsByNamesResolveTargets(ctx context.Context, workflowNames []string, mdFiles []string, githubWorkflows map[string]*GitHubWorkflow, enable bool) ([]toggleWorkflowsByNamesTarget, []string) {
	var targets []toggleWorkflowsByNamesTarget
	var notFoundNames []string
	for _, workflowName := range workflowNames {
		t, found, include := toggleWorkflowsByNamesFindTarget(ctx, workflowName, mdFiles, githubWorkflows, enable)
		if !found {
			notFoundNames = append(notFoundNames, workflowName)
			continue
		}
		if include {
			targets = append(targets, t)
		}
	}
	return targets, notFoundNames
}

func toggleWorkflowsByNamesFindTarget(ctx context.Context, workflowName string, mdFiles []string, githubWorkflows map[string]*GitHubWorkflow, enable bool) (toggleWorkflowsByNamesTarget, bool, bool) {
	for _, file := range mdFiles {
		base := filepath.Base(file)
		name := normalizeWorkflowID(base)
		if name != workflowName {
			continue
		}
		lockFile := stringutil.MarkdownToLockFile(file)
		lockFileBase := filepath.Base(lockFile)
		githubWorkflow, exists := githubWorkflows[name]

		if enable && toggleWorkflowsByNamesNeedsCompile(ctx, file, lockFile, name, exists) {
			return toggleWorkflowsByNamesTarget{}, true, false
		}
		if exists && toggleWorkflowsByNamesAlreadyDesired(name, enable, githubWorkflow.State) {
			return toggleWorkflowsByNamesTarget{}, true, false
		}
		t := toggleWorkflowsByNamesTarget{Name: name, LockFileBase: lockFileBase, CurrentState: "unknown", HasGitHubEntry: exists}
		if exists {
			t.ID = githubWorkflow.ID
			t.CurrentState = githubWorkflow.State
		}
		return t, true, true
	}
	return toggleWorkflowsByNamesTarget{}, false, false
}

func toggleWorkflowsByNamesNeedsCompile(ctx context.Context, file, lockFile, name string, exists bool) bool {
	if _, err := os.Stat(lockFile); !os.IsNotExist(err) {
		return false
	}
	if err := compileWorkflow(ctx, file, false, false, ""); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to compile workflow %s to create lock file: %v", name, err)))
		// If we can't compile and there's no GitHub entry, skip because we can't address it
		return !exists
	}
	return false
}

func toggleWorkflowsByNamesAlreadyDesired(name string, enable bool, state string) bool {
	if enable && state == "active" {
		// Already enabled
		fmt.Fprintf(os.Stderr, "Workflow %s is already enabled\n", name)
		return true
	}
	if !enable && state == "disabled_manually" {
		// Already disabled
		fmt.Fprintf(os.Stderr, "Workflow %s is already disabled\n", name)
		return true
	}
	return false
}

func toggleWorkflowsByNamesNotFoundError(notFoundNames []string) error {
	enableLog.Printf("Workflows not found: %v", notFoundNames)
	suggestions := []string{
		fmt.Sprintf("Run '%s status' to see all available workflows", string(constants.CLIExtensionPrefix)),
		"Check for typos in the workflow names",
		"Ensure the workflows have been compiled and pushed to GitHub",
	}
	// Add fuzzy match suggestions for each not found workflow
	if len(notFoundNames) == 1 {
		similarNames := suggestWorkflowNames(notFoundNames[0])
		if len(similarNames) > 0 {
			suggestions = append([]string{fmt.Sprintf("Did you mean: %s?", strings.Join(similarNames, ", "))}, suggestions...)
		}
	}
	return errors.New(console.FormatErrorWithSuggestions(
		"workflows not found: "+strings.Join(notFoundNames, ", "),
		suggestions,
	))
}

func toggleWorkflowsByNamesPrintTargets(action string, targets []toggleWorkflowsByNamesTarget) {
	fmt.Fprintf(os.Stderr, "The following workflows will be %sd:\n", action)
	for _, t := range targets {
		fmt.Fprintf(os.Stderr, "  %s (current state: %s)\n", t.Name, t.CurrentState)
	}
}

func toggleWorkflowsByNamesRunTarget(t toggleWorkflowsByNamesTarget, action string, enable bool, repoOverride string) bool {
	cmd := toggleWorkflowsByNamesCommandForTarget(t, enable, repoOverride)
	if output, err := cmd.CombinedOutput(); err != nil {
		toggleWorkflowsByNamesPrintFailure(action, t.Name, output, err)
		return false
	}
	fmt.Fprintf(os.Stderr, "%sd workflow: %s\n", strings.ToUpper(action[:1])+action[1:], t.Name)
	return true
}

func toggleWorkflowsByNamesCommandForTarget(t toggleWorkflowsByNamesTarget, enable bool, repoOverride string) *exec.Cmd {
	if enable {
		// Prefer enabling by ID, otherwise fall back to lock file name
		if t.ID != 0 {
			return toggleWorkflowsByNamesGHCommand(repoOverride, "workflow", "enable", strconv.FormatInt(t.ID, 10))
		}
		return toggleWorkflowsByNamesGHCommand(repoOverride, "workflow", "enable", t.LockFileBase)
	}

	// First cancel any running workflows (by ID when available, else by lock file name)
	if t.ID != 0 {
		if err := cancelWorkflowRuns(t.ID); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to cancel runs for workflow %s: %v", t.Name, err)))
		}
	} else if err := cancelWorkflowRunsByLockFile(t.LockFileBase); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to cancel runs for workflow %s: %v", t.Name, err)))
	}
	// Prefer disabling by lock file name for reliability
	return toggleWorkflowsByNamesGHCommand(repoOverride, "workflow", "disable", t.LockFileBase)
}

func toggleWorkflowsByNamesGHCommand(repoOverride string, args ...string) *exec.Cmd {
	if repoOverride != "" {
		args = append(args, "--repo", repoOverride)
	}
	return workflow.ExecGH(args...)
}

func toggleWorkflowsByNamesPrintFailure(action, workflowName string, output []byte, err error) {
	if len(output) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to %s workflow %s: %v", action, workflowName, err)))
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to %s workflow %s: %v\n%s", action, workflowName, err, string(output))))
	// Provide clearer hint on common permission issues
	outStr := strings.ToLower(string(output))
	if strings.Contains(outStr, "http 403") || strings.Contains(outStr, "resource not accessible by integration") {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Hint: Disabling/enabling workflows requires repository admin or maintainer permissions. Ensure your gh auth has write/admin access to this repo."))
	}
}

// DisableAllWorkflowsExcept disables all workflows except the specified ones
// Typically used to disable all workflows except the one being trialled
func DisableAllWorkflowsExcept(repoSlug string, exceptWorkflows []string, verbose bool) error {
	enableLog.Printf("Disabling all workflows except: count=%d, repo=%s", len(exceptWorkflows), repoSlug)
	workflowsDir := constants.GetWorkflowDir()

	allYAMLFiles, ok := disableAllWorkflowsExceptYAMLFiles(workflowsDir, verbose)
	if !ok {
		return nil
	}

	// Create a set of workflows to keep enabled
	keepEnabled := disableAllWorkflowsExceptKeepSet(exceptWorkflows)

	// Filter to find workflows to disable
	workflowsToDisable := disableAllWorkflowsExceptFilter(allYAMLFiles, keepEnabled, verbose)
	if len(workflowsToDisable) == 0 {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No workflows to disable"))
		}
		return nil
	}

	// Show what will be disabled
	fmt.Fprintf(os.Stderr, "Disabling %d workflow(s) in cloned repository:\n", len(workflowsToDisable))
	for _, wf := range workflowsToDisable {
		fmt.Fprintf(os.Stderr, "  %s\n", wf)
	}

	// Disable each workflow
	failures := disableAllWorkflowsExceptDisable(workflowsToDisable, repoSlug, verbose)

	if len(failures) > 0 {
		return fmt.Errorf("failed to disable %d workflow(s): %s", len(failures), strings.Join(failures, ", "))
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Disabled %d workflow(s)", len(workflowsToDisable))))
	return nil
}

func disableAllWorkflowsExceptYAMLFiles(workflowsDir string, verbose bool) ([]string, bool) {
	// Check if workflows directory exists
	if _, err := os.Stat(workflowsDir); os.IsNotExist(err) {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No .github/workflows directory found, nothing to disable"))
		}
		return nil, false
	}

	// Get all .yml and .yaml files
	ymlFiles, _ := filepath.Glob(filepath.Join(workflowsDir, "*.yml"))
	yamlFiles, _ := filepath.Glob(filepath.Join(workflowsDir, "*.yaml"))
	allYAMLFiles := append(ymlFiles, yamlFiles...)

	enableLog.Printf("Found %d YAML workflow files", len(allYAMLFiles))
	if len(allYAMLFiles) == 0 {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No YAML workflow files found"))
		}
		return nil, false
	}
	return allYAMLFiles, true
}

func disableAllWorkflowsExceptKeepSet(exceptWorkflows []string) map[string]struct{} {
	keepEnabled := make(map[string]struct{})
	for _, workflowName := range exceptWorkflows {
		// Add both .md and .lock.yml variants
		keepEnabled[workflowName+".md"] = struct{}{}
		keepEnabled[workflowName+".lock.yml"] = struct{}{}
		keepEnabled[workflowName] = struct{}{} // In case the full filename is provided
	}
	return keepEnabled
}

func disableAllWorkflowsExceptFilter(allYAMLFiles []string, keepEnabled map[string]struct{}, verbose bool) []string {
	var workflowsToDisable []string
	for _, yamlFile := range allYAMLFiles {
		base := filepath.Base(yamlFile)
		if setutil.Contains(keepEnabled, base) {
			if verbose {
				fmt.Fprintf(os.Stderr, "Keeping enabled: %s\n", base)
			}
			continue
		}

		nameWithoutExt := strings.TrimSuffix(base, filepath.Ext(base))
		if setutil.Contains(keepEnabled, nameWithoutExt) {
			if verbose {
				fmt.Fprintf(os.Stderr, "Keeping enabled: %s\n", base)
			}
			continue
		}
		workflowsToDisable = append(workflowsToDisable, base)
	}
	return workflowsToDisable
}

func disableAllWorkflowsExceptDisable(workflowsToDisable []string, repoSlug string, verbose bool) []string {
	var failures []string
	for _, wf := range workflowsToDisable {
		args := []string{"workflow", "disable", wf}
		if repoSlug != "" {
			args = append(args, "--repo", repoSlug)
		}

		cmd := workflow.ExecGH(args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to disable workflow %s: %v\n%s", wf, err, string(output))))
			}
			failures = append(failures, wf)
		} else if verbose {
			fmt.Fprintf(os.Stderr, "Disabled workflow: %s\n", wf)
		}
	}
	return failures
}
