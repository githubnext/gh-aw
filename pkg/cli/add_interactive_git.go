package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"charm.land/huh/v2"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/styles"
	"github.com/github/gh-aw/pkg/workflow"
)

// createWorkflowPRAndConfigureSecret creates the PR, merges it, and adds the secret
func (c *AddInteractiveConfig) createWorkflowPRAndConfigureSecret(ctx context.Context, workflowFiles, initFiles []string, secretName, secretValue string) error {
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

	// Step 8b: Optionally merge the PR – loop until merged, confirmed-merged, or user exits
	if result.PRNumber == 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Could not determine PR number"))
		fmt.Fprintln(os.Stderr, "Please merge the PR manually from the GitHub web interface.")
	} else {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Pull request created: "+result.PRURL))
		fmt.Fprintln(os.Stderr, "")

		// mergeAction values used in the select loop
		type mergeAction string
		const (
			mergeActionAttempt   mergeAction = "attempt"
			mergeActionEditTitle mergeAction = "editTitle"
			mergeActionReview    mergeAction = "review"
			mergeActionConfirmed mergeAction = "confirmed"
			mergeActionExit      mergeAction = "exit"
		)

		mergeDone := false     // true when the PR is merged (or confirmed merged)
		mergeFailed := false   // true after an unsuccessful merge attempt
		userReviewing := false // true after the user chose "I'll review myself"

		for !mergeDone {
			// Build option list based on current state
			var options []huh.Option[mergeAction]

			options = append(options, huh.NewOption("Attempt to merge", mergeActionAttempt))

			if mergeFailed {
				options = append(options, huh.NewOption("Edit PR title and retry", mergeActionEditTitle))
			}

			if userReviewing {
				options = append(options, huh.NewOption("PR has been manually merged", mergeActionConfirmed))
			} else {
				options = append(options, huh.NewOption("I'll review/merge myself", mergeActionReview))
			}

			if userReviewing {
				options = append(options, huh.NewOption("Exit, I'm done here", mergeActionExit))
			} else {
				options = append(options, huh.NewOption("Exit", mergeActionExit))
			}

			var chosen mergeAction
			selectForm := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[mergeAction]().
						Title("What would you like to do with pull request " + result.PRURL + "?").
						Options(options...).
						Value(&chosen),
				),
			).WithTheme(styles.HuhTheme).WithAccessible(console.IsAccessibleMode())

			if selectErr := selectForm.Run(); selectErr != nil {
				return fmt.Errorf("failed to get user input: %w", selectErr)
			}

			switch chosen {
			case mergeActionAttempt:
				if mergeErr := c.mergePullRequest(result.PRNumber); mergeErr != nil {
					if strings.Contains(mergeErr.Error(), "already merged") || strings.Contains(mergeErr.Error(), "MERGED") {
						fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Merged pull request "+result.PRURL))
						mergeDone = true
					} else {
						fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to merge PR: %v", mergeErr)))
						if mergeFailed {
							fmt.Fprintln(os.Stderr, "Please merge the PR manually: "+result.PRURL)
						}
						mergeFailed = true
					}
				} else {
					fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Merged pull request "+result.PRURL))
					mergeDone = true
				}

			case mergeActionEditTitle:
				var newTitle string
				titleForm := huh.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("Enter new PR title").
							Description("Add a prefix if required, for example: feat: or fix:").
							Value(&newTitle),
					),
				).WithTheme(styles.HuhTheme).WithAccessible(console.IsAccessibleMode())
				if titleErr := titleForm.Run(); titleErr != nil {
					return fmt.Errorf("failed to get user input: %w", titleErr)
				}
				newTitle = strings.TrimSpace(newTitle)
				if newTitle == "" {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage("PR title cannot be empty, keeping current title"))
				} else if editErr := editPRTitle(result.PRNumber, newTitle, c.RepoOverride); editErr != nil {
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to update PR title: %v", editErr)))
				} else {
					fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("PR title updated to: "+newTitle))
					mergeFailed = false
				}

			case mergeActionReview:
				userReviewing = true
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Please review and merge the pull request: "+result.PRURL))
				fmt.Fprintln(os.Stderr, "")

			case mergeActionConfirmed:
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Great – continuing with the merged pull request"))
				mergeDone = true

			case mergeActionExit:
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Exiting. You can merge the pull request later: "+result.PRURL))
				return errors.New("user exited before PR was merged")
			}
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

	return nil
}

// updateLocalBranch fetches and pulls the latest changes from GitHub after PR merge.
// It switches to the default branch before pulling so that the working tree contains
// the merged workflow files, which are required when offering to run the workflow.
func (c *AddInteractiveConfig) updateLocalBranch() error {
	addInteractiveLog.Print("Updating local branch with merged changes")

	// Get the default branch name using gh
	output, err := workflow.RunGHCombined("Getting default branch...", "repo", "view", "--repo", c.RepoOverride, "--json", "defaultBranchRef", "--jq", ".defaultBranchRef.name")
	defaultBranch := ""
	if err == nil {
		defaultBranch = strings.TrimSpace(string(output))
	}

	// Fallback: query the local origin remote directly (works even when gh repo
	// view fails, e.g. forks without a default remote set).
	if defaultBranch == "" {
		addInteractiveLog.Print("gh repo view failed, trying git ls-remote to detect default branch")
		lsCmd := exec.Command("git", "ls-remote", "--symref", "origin", "HEAD")
		lsOutput, lsErr := lsCmd.CombinedOutput()
		if lsErr == nil {
			defaultBranch = parseDefaultBranchFromLsRemote(string(lsOutput))
		}
	}

	if defaultBranch == "" {
		defaultBranch = "main"
	}
	addInteractiveLog.Printf("Default branch: %s", defaultBranch)

	// Fetch the latest changes from origin
	if c.Verbose {
		fmt.Fprintln(os.Stderr, console.FormatProgressMessage("Fetching latest changes from GitHub..."))
	}

	fetchCmd := exec.Command("git", "fetch", "origin", defaultBranch)
	fetchOutput, err := fetchCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch failed: %w (output: %s)", err, string(fetchOutput))
	}

	// Switch to the default branch so the working tree contains the merged workflow
	// files. Without this, users on a feature branch won't have the files locally and
	// the subsequent "run workflow" step will fail with "workflow file not found".
	currentBranch, err := getCurrentBranch()
	if err != nil {
		addInteractiveLog.Printf("Could not determine current branch: %v", err)
		currentBranch = ""
	}

	if currentBranch != defaultBranch {
		addInteractiveLog.Printf("Switching from %q to default branch %q", currentBranch, defaultBranch)
		if err := switchBranch(defaultBranch, c.Verbose); err != nil {
			return fmt.Errorf("failed to switch to default branch %s: %w", defaultBranch, err)
		}
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

// squashMergeNotAllowedErr is the lowercase substring of the GitHub GraphQL API error
// returned when a repository does not permit squash merges. It is used to detect when
// a squash merge should be retried with a merge-commit strategy.
const squashMergeNotAllowedErr = "squash merges are not allowed"

// mergePullRequest merges the specified PR, attempting a squash merge first and
// falling back to a merge commit if squash merges are not allowed on the repository.
func (c *AddInteractiveConfig) mergePullRequest(prNumber int) error {
	prArg := strconv.Itoa(prNumber)
	squashOutput, squashErr := workflow.RunGHCombined("Merging pull request (squash)...", "pr", "merge", prArg, "--repo", c.RepoOverride, "--squash")
	if squashErr == nil {
		return nil
	}

	// If squash merges are not allowed on this repository (e.g. only merge commits or rebase
	// merges are enabled), fall back to a merge commit. The error text comes from the GitHub
	// GraphQL API and is surfaced verbatim in the gh CLI output.
	combinedText := strings.ToLower(string(squashOutput) + squashErr.Error())
	if strings.Contains(combinedText, squashMergeNotAllowedErr) {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Squash merges are not allowed on this repository, retrying with merge commit"))
		mergeOutput, mergeErr := workflow.RunGHCombined("Merging pull request...", "pr", "merge", prArg, "--repo", c.RepoOverride, "--merge")
		if mergeErr != nil {
			return fmt.Errorf("merge failed: %w (output: %s)", mergeErr, string(mergeOutput))
		}
		return nil
	}

	return fmt.Errorf("merge failed: %w (output: %s)", squashErr, string(squashOutput))
}

// editPRTitle updates the title of the specified PR via the gh CLI.
func editPRTitle(prNumber int, newTitle, repoOverride string) error {
	args := []string{"pr", "edit", strconv.Itoa(prNumber), "--title", newTitle}
	if repoOverride != "" {
		args = append(args, "--repo", repoOverride)
	}
	output, err := workflow.RunGHCombined("Updating PR title...", args...)
	if err != nil {
		return fmt.Errorf("failed to update PR title: %w (output: %s)", err, string(output))
	}
	return nil
}

// parseDefaultBranchFromLsRemote extracts the default branch name from
// the output of `git ls-remote --symref origin HEAD`.
//
// Example output:
//
//	ref: refs/heads/main	abc123
//	abc123	HEAD
//
// Returns "" if the branch cannot be determined.
func parseDefaultBranchFromLsRemote(output string) string {
	for line := range strings.SplitSeq(output, "\n") {
		if !strings.HasPrefix(line, "ref: refs/heads/") {
			continue
		}
		// line is e.g. "ref: refs/heads/main\tabc123"
		// Split on tab first to isolate the symref part from the hash.
		tabParts := strings.SplitN(line, "\t", 2)
		ref := strings.TrimPrefix(tabParts[0], "ref: refs/heads/")
		ref = strings.TrimSpace(ref)
		if ref != "" {
			return ref
		}
	}
	return ""
}
