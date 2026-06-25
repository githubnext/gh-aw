package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var runUpgradeForTargetRepoFn = runUpgradeForTargetRepo
var searchOrgAnyWorkflowReposFn = searchOrgAnyWorkflowRepos
var createIssueForUpgradeOrgRepoFn = createIssueForUpgradeOrgRepo

// runUpgradeForOrg runs the upgrade command across all repositories in an
// organization that have agentic workflow files. Without --create-pull-request
// or --create-issue it prints a dry-run preview; with --create-pull-request it
// checks out each repository, runs the upgrade, and opens a pull request; with
// --create-issue it opens a GitHub issue in each repository.
func runUpgradeForOrg(ctx context.Context, org string, repoGlobs []string, opts upgradeOptions, createPR bool, createIssue bool, verbose bool) error {
	if strings.TrimSpace(org) == "" {
		return errors.New("--org cannot be empty")
	}
	if err := validateRepoGlobs(repoGlobs); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Discovering repositories in "+org+" with agentic workflows..."))
	repoPaths, err := searchOrgAnyWorkflowReposFn(ctx, org, verbose)
	if err != nil {
		return err
	}

	if len(repoPaths) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No repositories found with agentic workflows"))
		return nil
	}

	repos := filterOrgRepos(repoPaths, repoGlobs)
	if len(repos) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No repositories matched the requested --repos filters"))
		return nil
	}

	if !createPR && !createIssue {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Dry-run preview of upgrade pull requests:"))
		for _, repo := range repos {
			fmt.Fprintf(os.Stderr, "- %s\n", repo)
		}
		return nil
	}

	if createIssue {
		for _, repo := range repos {
			if err := waitForOrgRateLimitFn(ctx, "core", verbose); err != nil && verbose {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Continuing after rate limit check failure for %s: %v", repo, err)))
			}
			if err := createIssueForUpgradeOrgRepoFn(ctx, repo, verbose); err != nil {
				return err
			}
		}
		return nil
	}

	for _, repo := range repos {
		if err := waitForOrgRateLimitFn(ctx, "core", verbose); err != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Continuing after rate limit check failure for %s: %v", repo, err)))
		}
		if err := runUpgradeForTargetRepoFn(ctx, repo, opts, verbose); err != nil {
			return err
		}
	}

	return nil
}

// runUpgradeForTargetRepo checks out repo to a temporary directory, runs the
// upgrade command inside it, and opens a pull request with the resulting changes.
func runUpgradeForTargetRepo(ctx context.Context, repo string, opts upgradeOptions, verbose bool) error {
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return fmt.Errorf("--org requires running inside a git repository: %w", err)
	}

	updatesDir, err := ensureUpdateTargetRepoGitignore(gitRoot)
	if err != nil {
		return err
	}

	checkoutDir := filepath.Join(updatesDir, sanitizeRepoPath(repo))
	if err := shallowCloneTargetRepo(ctx, repo, checkoutDir); err != nil {
		return err
	}

	// Extend sparse checkout to include .github/skills; upgrade also updates
	// the dispatcher skill (ensureAgenticWorkflowsDispatcher) and needs that path present.
	sparseAddCmd := exec.CommandContext(ctx, "git", "-C", checkoutDir, "sparse-checkout", "add", ".github/skills")
	if output, err := sparseAddCmd.CombinedOutput(); err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return fmt.Errorf("failed to extend sparse checkout for %s: %w", repo, err)
		}
		return fmt.Errorf("failed to extend sparse checkout for %s: %w: %s", repo, err, trimmed)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Checked out "+repo+" at "+checkoutDir))
	}

	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to read current directory: %w", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(checkoutDir); err != nil {
		return fmt.Errorf("failed to change directory to checkout %s: %w", checkoutDir, err)
	}

	if err := PreflightCheckForCreatePR(verbose); err != nil {
		return err
	}

	// Override fields that must be adjusted for a remote-repo upgrade.
	// workflowDir is intentionally reset: --dir is a local-machine concept and
	// must not be forwarded to remote repos where that path may not exist.
	opts.ctx = ctx
	opts.skipExtensionUpgrade = true
	opts.verbose = verbose
	opts.workflowDir = ""

	if err := runUpgradeCommand(opts); err != nil {
		return err
	}

	// Skip PR creation when the upgrade produced no changes (e.g. repo is already up to date).
	changed, err := hasPendingChanges()
	if err != nil {
		return err
	}
	if !changed {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Skipping PR for "+repo+": already up to date"))
		}
		return nil
	}

	prBody := "This PR upgrades agentic workflows by applying the latest codemods, " +
		"updating GitHub Actions versions, and recompiling all workflows."
	_, err = CreatePRWithChanges("upgrade-agentic-workflows", "chore: upgrade agentic workflows",
		"Upgrade agentic workflows", prBody, verbose)
	return err
}

// searchOrgAnyWorkflowRepos searches an organization's repositories for any
// agentic workflow markdown files in .github/workflows, returning a sorted
// deduplicated slice of "owner/repo" strings. README.md is excluded so that
// repos containing only documentation are not treated as having workflows.
func searchOrgAnyWorkflowRepos(ctx context.Context, org string, verbose bool) ([]string, error) {
	query := fmt.Sprintf(`org:%s path:.github/workflows extension:md NOT filename:README`, org)
	return searchOrgReposByQuery(ctx, query, verbose)
}

// createIssueForUpgradeOrgRepo opens a GitHub issue in the target repository
// to notify maintainers that agentic workflow upgrades are available. The issue
// prompts them to run gh aw upgrade locally to apply codemods, update action
// versions, and recompile workflows. It is idempotent: if an open issue with the
// same title already exists it logs a skip message instead of creating a duplicate.
func createIssueForUpgradeOrgRepo(ctx context.Context, repo string, verbose bool) error {
	title := "Upgrade agentic workflows"

	// Idempotency guard: skip if an open issue with this title already exists.
	// Parse the response in Go rather than embedding the title in a jq expression
	// to avoid any quoting or injection issues.
	existsOutput, err := workflow.RunGHContext(ctx, "Checking for existing upgrade issue...",
		"api",
		fmt.Sprintf("/repos/%s/issues?state=open&per_page=100", repo),
	)
	if err == nil {
		var issues []struct {
			Title string `json:"title"`
		}
		if jsonErr := json.Unmarshal(existsOutput, &issues); jsonErr == nil {
			for _, issue := range issues {
				if issue.Title == title {
					if verbose {
						fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Skipping "+repo+": upgrade issue already exists"))
					}
					return nil
				}
			}
		}
	}

	body := "Agentic workflow files detected in this repository may have upgrades available.\n\n" +
		"Run `gh aw upgrade` to apply the latest codemods, update GitHub Actions versions, and recompile all workflows.\n\n" +
		"Review the upgrade output and any generated changes before committing to ensure there are no unexpected modifications.\n"

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Creating upgrade issue in "+repo+"..."))
	}

	endpoint := fmt.Sprintf("/repos/%s/issues", repo)
	_, err = workflow.RunGHContext(ctx, "Creating issue...",
		"api",
		"--method", "POST",
		endpoint,
		"-f", "title="+title,
		"-f", "body="+body,
	)
	if err != nil {
		return fmt.Errorf("failed to create issue in %s: %w", repo, err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created issue in "+repo))
	return nil
}
