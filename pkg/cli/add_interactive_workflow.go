package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/huh/v2"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/workflow"
)

// checkStatusAndOfferRun checks if the workflow appears in status and offers to run it
func (c *AddInteractiveConfig) checkStatusAndOfferRun(ctx context.Context) error {
	addInteractiveLog.Print("Checking workflow status and offering to run")

	// Wait a moment for GitHub to process the merge
	fmt.Fprintln(os.Stderr, "")

	workflowFound, err := c.waitForWorkflowAvailability(ctx)
	if err != nil {
		return err
	}
	if !workflowFound {
		return c.showWorkflowStatusUnavailableMessage()
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Workflow is ready"))

	if !c.canOfferWorkflowRun() {
		addInteractiveLog.Print("Workflow does not have workflow_dispatch trigger, skipping run offer")
		c.showFinalInstructions()
		return nil
	}

	if isRunningInCodespace() {
		return c.showCodespaceRunInstructions()
	}

	runNow, err := c.promptToRunWorkflowNow(ctx)
	if err != nil {
		return nil
	}
	if !runNow {
		c.showFinalInstructions()
		return nil
	}

	return c.runWorkflowAfterStatusCheck(ctx)
}

func (c *AddInteractiveConfig) waitForWorkflowAvailability(ctx context.Context) (bool, error) {
	spinner := c.startWorkflowStatusSpinner()
	if spinner != nil {
		defer spinner.Stop()
	}
	for i := range 5 {
		found, err := c.checkWorkflowAvailabilityAttempt(ctx, i)
		if err != nil || found {
			return found, err
		}
	}
	return false, nil
}

func (c *AddInteractiveConfig) startWorkflowStatusSpinner() *console.SpinnerWrapper {
	if c.Verbose {
		return nil
	}
	spinner := console.NewSpinner("Waiting for workflow to be available...")
	spinner.Start()
	return spinner
}

func (c *AddInteractiveConfig) checkWorkflowAvailabilityAttempt(ctx context.Context, attempt int) (bool, error) {
	if err := waitForWorkflowStatusRetry(ctx); err != nil {
		return false, err
	}
	workflowName := c.primaryWorkflowName()
	if workflowName == "" {
		return false, nil
	}
	if c.Verbose {
		fmt.Fprintf(os.Stderr, "Checking workflow status (attempt %d/5) for: %s\n", attempt+1, workflowName)
	}
	statuses, err := findWorkflowsByFilenamePattern(workflowName, c.RepoOverride, c.Verbose)
	return interpretWorkflowStatusCheck(statuses, err, c.Verbose)
}

func waitForWorkflowStatusRetry(ctx context.Context) error {
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func interpretWorkflowStatusCheck(statuses []WorkflowStatus, err error, verbose bool) (bool, error) {
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Status check error: %v\n", err)
		}
		return false, nil
	}
	if len(statuses) > 0 {
		if verbose {
			fmt.Fprintf(os.Stderr, "Found %d workflow(s) matching pattern\n", len(statuses))
		}
		return true, nil
	}
	if verbose {
		fmt.Fprintln(os.Stderr, "No workflows found matching pattern yet")
	}
	return false, nil
}

func (c *AddInteractiveConfig) showWorkflowStatusUnavailableMessage() error {
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Could not verify workflow status."))
	fmt.Fprintf(os.Stderr, "You can check status with: %s status\n", string(constants.CLIExtensionPrefix))
	c.showFinalInstructions()
	return nil
}

func (c *AddInteractiveConfig) canOfferWorkflowRun() bool {
	return c.addResult != nil && c.addResult.HasWorkflowDispatch
}

func (c *AddInteractiveConfig) showCodespaceRunInstructions() error {
	addInteractiveLog.Print("Running in Codespaces, skipping run offer and showing Actions link")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Running in GitHub Codespaces - please trigger the workflow manually from the Actions page"))
	fmt.Fprintf(os.Stderr, "🔗 https://github.com/%s/actions\n", c.RepoOverride)
	c.showFinalInstructions()
	return nil
}

func (c *AddInteractiveConfig) promptToRunWorkflowNow(ctx context.Context) (bool, error) {
	fmt.Fprintln(os.Stderr, "")
	runNow := true
	form := console.NewConfirmForm(
		huh.NewConfirm().
			Title("Would you like to run the workflow once now?").
			Description("This will trigger the workflow immediately").
			Affirmative("Yes, run once now").
			Negative("No, I'll run later").
			Value(&runNow),
	)
	if err := form.RunWithContext(ctx); err != nil {
		return false, err
	}
	return runNow, nil
}

func (c *AddInteractiveConfig) runWorkflowAfterStatusCheck(ctx context.Context) error {
	workflowName := c.primaryWorkflowName()
	if workflowName == "" {
		c.showFinalInstructions()
		return nil
	}

	fmt.Fprintln(os.Stderr, "")
	c.updateLocalBranchForRunOffer()
	if err := RunSpecificWorkflowInteractively(ctx, RunWorkflowOptions{
		WorkflowName:   workflowName,
		Verbose:        c.Verbose,
		EngineOverride: c.EngineOverride,
		RepoOverride:   c.RepoOverride,
	}); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to run workflow: %v", err)))
		c.showFinalInstructions()
		return nil
	}

	c.printLatestWorkflowRunURL(workflowName)
	c.showFinalInstructions()
	return nil
}

func (c *AddInteractiveConfig) updateLocalBranchForRunOffer() {
	// Pull the merged workflow files after GitHub has processed the merge to avoid a
	// race where git fetch completes before the new workflow is available locally.
	if !c.Verbose {
		fmt.Fprintln(os.Stderr, "Updating local branch (this may take a few seconds)...")
	}
	if err := c.updateLocalBranch(); err != nil {
		addInteractiveLog.Printf("Failed to update local branch: %v", err)
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not update local branch: %v", err)))
		fmt.Fprintln(os.Stderr, "You may need to switch to your repository's default branch (for example 'main') and run 'git pull' manually before running the workflow.")
	}
	if !c.Verbose {
		fmt.Fprintln(os.Stderr, "Finished updating local branch.")
	}
}

func (c *AddInteractiveConfig) printLatestWorkflowRunURL(workflowName string) {
	runInfo, err := getLatestWorkflowRunWithRetry(workflowName+".lock.yml", c.RepoOverride, c.Verbose)
	if err != nil || runInfo.URL == "" {
		return
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Workflow triggered successfully!"))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, "🔗 View workflow run: %s\n", runInfo.URL)
}

// findWorkflowsByFilenamePattern is a helper to find workflows registered in GitHub by filename pattern.
// The pattern is matched against the workflow filename (basename without extension)
func findWorkflowsByFilenamePattern(pattern, repoOverride string, verbose bool) ([]WorkflowStatus, error) {
	// This would normally call StatusWorkflows but we need just a simple check
	// For now, we'll use the gh CLI directly
	// Request 'path' field so we can match by filename, not by workflow name
	args := []string{"workflow", "list", "--json", "name,state,path"}
	if repoOverride != "" {
		args = append(args, "--repo", repoOverride)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: gh %s\n", strings.Join(args, " "))
	}

	output, err := workflow.RunGH("Checking workflow status...", args...)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "gh workflow list failed: %v\n", err)
		}
		return nil, err
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "gh workflow list output: %s\n", string(output))
		fmt.Fprintf(os.Stderr, "Looking for workflow with filename containing: %s\n", pattern)
	}

	// Check if any workflow path contains the pattern
	// The pattern is the workflow name (e.g., "daily-repo-status")
	// The path is like ".github/workflows/daily-repo-status.lock.yml"
	// We check if the path contains the pattern
	if strings.Contains(string(output), pattern+".lock.yml") || strings.Contains(string(output), pattern+".md") {
		if verbose {
			fmt.Fprintf(os.Stderr, "Workflow with filename '%s' found in workflow list\n", pattern)
		}
		return []WorkflowStatus{{WorkflowListItem: WorkflowListItem{Workflow: pattern}}}, nil
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Workflow with filename '%s' NOT found in workflow list\n", pattern)
	}
	return nil, nil
}
