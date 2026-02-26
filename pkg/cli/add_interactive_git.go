package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/workflow"
)

// applyChanges creates the PR, merges it, and adds the secret
func (c *AddInteractiveConfig) applyChanges(ctx context.Context, workflowFiles, initFiles []string, secretName, secretValue string) error {
	addInteractiveLog.Print("Applying changes")

	fmt.Fprintln(os.Stderr, "")

	// Add the workflow using existing implementation with --create-pull-request
	// Pass the resolved workflows to avoid re-fetching them
	// Pass Quiet=true to suppress detailed output (already shown earlier in interactive mode)
	// This returns the result including PR number and HasWorkflowDispatch
	opts := AddOptions{
		Verbose:                c.Verbose,
		Quiet:                  true,
		EngineOverride:         c.EngineOverride,
		Name:                   "",
		Force:                  false,
		AppendText:             "",
		CreatePR:               true,
		Push:                   false,
		NoGitattributes:        c.NoGitattributes,
		WorkflowDir:            c.WorkflowDir,
		NoStopAfter:            c.NoStopAfter,
		StopAfter:              c.StopAfter,
		DisableSecurityScanner: false,
	}
	result, err := AddResolvedWorkflows(c.WorkflowSpecs, c.resolvedWorkflows, opts)
	if err != nil {
		return fmt.Errorf("failed to add workflow: %w", err)
	}
	c.addResult = result

	// Step 8b: Auto-merge the PR
	if result.PRNumber == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Could not determine PR number"))
		fmt.Fprintln(os.Stderr, "Please merge the PR manually from the GitHub web interface.")
	} else {
		if err := c.mergePullRequest(result.PRNumber); err != nil {
			// Check if already merged
			if strings.Contains(err.Error(), "already merged") || strings.Contains(err.Error(), "MERGED") {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Merged pull request "+result.PRURL))
			} else {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to merge PR: %v", err)))
				fmt.Fprintln(os.Stderr, "Please merge the PR manually from the GitHub web interface.")

				// Ask user whether to continue or stop
				continueAfterMerge := true
				mergeForm := huh.NewForm(
					huh.NewGroup(
						huh.NewConfirm().
							Title("Would you like to continue?").
							Description("Select 'Yes' once you have merged the PR, or 'No' to stop here").
							Affirmative("Yes, continue (PR is merged)").
							Negative("No, stop for now").
							Value(&continueAfterMerge),
					),
				).WithAccessible(console.IsAccessibleMode())

				if mergeFormErr := mergeForm.Run(); mergeFormErr != nil {
					return fmt.Errorf("failed to get user input: %w", mergeFormErr)
				}

				if !continueAfterMerge {
					fmt.Fprintln(os.Stderr, "")
					fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Stopped. You can continue later by merging the PR and running the workflow manually."))
					return errors.New("user chose to stop after merge failure")
				}
			}
		} else {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Merged pull request "+result.PRURL))
		}
	}

	// Step 8c: Add the secret (skip if no secret configured or already exists in repository)
	if secretName == "" {
		// No secret to configure (e.g., user doesn't have write access to the repository)
	} else if secretValue == "" {
		// Secret already exists in repo, nothing to do
		if c.Verbose {
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Secret '%s' already configured", secretName)))
		}
	} else {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, console.FormatProgressMessage(fmt.Sprintf("Adding secret '%s' to repository...", secretName)))

		if err := c.addRepositorySecret(secretName, secretValue); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Failed to add secret: %v", err)))
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Please add the secret manually:")
			fmt.Fprintln(os.Stderr, "  1. Go to your repository Settings → Secrets and variables → Actions")
			fmt.Fprintf(os.Stderr, "  2. Click 'New repository secret' and add '%s'\n", secretName)
			return fmt.Errorf("failed to add secret: %w", err)
		}

		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Secret '%s' added", secretName)))
	}

	// Step 8d: Update local branch with merged changes from GitHub
	if err := c.updateLocalBranch(); err != nil {
		// Non-fatal - warn but continue, workflow can still run on GitHub
		addInteractiveLog.Printf("Failed to update local branch: %v", err)
		if c.Verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not update local branch: %v", err)))
		}
	}

	return nil
}

// updateLocalBranch fetches and pulls the latest changes from GitHub after PR merge
func (c *AddInteractiveConfig) updateLocalBranch() error {
	addInteractiveLog.Print("Updating local branch with merged changes")

	// Get the default branch name using gh
	output, err := workflow.RunGHCombined("Getting default branch...", "repo", "view", "--repo", c.RepoOverride, "--json", "defaultBranchRef", "--jq", ".defaultBranchRef.name")
	defaultBranch := "main"
	if err == nil {
		defaultBranch = strings.TrimSpace(string(output))
	}
	addInteractiveLog.Printf("Default branch: %s", defaultBranch)

	// Fetch the latest changes from origin
	if c.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatProgressMessage("Fetching latest changes from GitHub..."))
	}

	// Use git fetch followed by git pull
	fetchCmd := exec.Command("git", "fetch", "origin", defaultBranch)
	fetchOutput, err := fetchCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch failed: %w (output: %s)", err, string(fetchOutput))
	}

	pullCmd := exec.Command("git", "pull", "origin", defaultBranch)
	pullOutput, err := pullCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %w (output: %s)", err, string(pullOutput))
	}

	if c.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Local branch updated with merged changes"))
	}

	return nil
}

// checkCleanWorkingDirectory verifies the working directory has no uncommitted changes.
// This is checked early in the interactive flow to avoid failing later during PR creation.
func (c *AddInteractiveConfig) checkCleanWorkingDirectory() error {
	addInteractiveLog.Print("Checking working directory is clean")

	if err := checkCleanWorkingDirectory(c.Verbose); err != nil {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage("Working directory is not clean."))
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "The add wizard creates a pull request which requires a clean working directory.")
		fmt.Fprintln(os.Stderr, "Please commit or stash your changes first:")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, console.FormatCommandMessage("  git stash        # Temporarily stash changes"))
		fmt.Fprintln(os.Stderr, console.FormatCommandMessage("  git add -A && git commit -m 'wip'  # Commit changes"))
		fmt.Fprintln(os.Stderr, "")
		return errors.New("working directory is not clean")
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Working directory is clean"))
	return nil
}

// mergePullRequest merges the specified PR
func (c *AddInteractiveConfig) mergePullRequest(prNumber int) error {
	output, err := workflow.RunGHCombined("Merging pull request...", "pr", "merge", strconv.Itoa(prNumber), "--repo", c.RepoOverride, "--merge")
	if err != nil {
		return fmt.Errorf("merge failed: %w (output: %s)", err, string(output))
	}
	return nil
}
