package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
)

var runUpgradeForTargetRepoFn = runUpgradeForTargetRepo
var searchOrgAnyWorkflowReposFn = searchOrgAnyWorkflowRepos

// runUpgradeForOrg runs the upgrade command across all repositories in an
// organization that have agentic workflow files. Without --create-pull-request
// it prints a dry-run preview; with --create-pull-request it checks out each
// repository, runs the upgrade, and opens a pull request.
func runUpgradeForOrg(ctx context.Context, org string, repoGlobs []string, opts upgradeOptions, createPR bool, verbose bool) error {
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

	if !createPR {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Dry-run preview of upgrade pull requests:"))
		for _, repo := range repos {
			fmt.Fprintf(os.Stderr, "- %s\n", repo)
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
	opts.ctx = ctx
	opts.skipExtensionUpgrade = true
	opts.verbose = verbose

	if err := runUpgradeCommand(opts); err != nil {
		return err
	}

	prBody := "This PR upgrades agentic workflows by applying the latest codemods, " +
		"updating GitHub Actions versions, and recompiling all workflows."
	_, err = CreatePRWithChanges("upgrade-agentic-workflows", "chore: upgrade agentic workflows",
		"Upgrade agentic workflows", prBody, verbose)
	return err
}

// searchOrgAnyWorkflowRepos searches an organization's repositories for any
// agentic workflow markdown files in .github/workflows, returning a sorted
// deduplicated slice of "owner/repo" strings.
func searchOrgAnyWorkflowRepos(ctx context.Context, org string, verbose bool) ([]string, error) {
	query := fmt.Sprintf(`org:%s path:.github/workflows extension:md`, org)
	return searchOrgReposByQuery(ctx, query, verbose)
}
