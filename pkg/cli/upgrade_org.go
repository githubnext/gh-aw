package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
)

var runUpgradeForTargetRepoFn = runUpgradeForTargetRepo
var searchOrgAnyWorkflowReposFn = searchOrgAnyWorkflowRepos
var createIssueForUpgradeOrgRepoFn = createIssueForUpgradeOrgRepo

// runUpgradeForOrg runs the upgrade command across all repositories in an
// organization that have agentic workflow files. Without --create-pull-request
// or --create-issue it prints a dry-run preview; with --create-pull-request it
// checks out each repository, runs the upgrade, and opens a pull request; with
// --create-issue it opens a GitHub issue in each repository.
//
// The function delegates to runCommandForOrg, which provides shared logic for
// organization discovery, rate-limit handling, graceful cancellation, result
// sorting, and per-repo error recovery.
func runUpgradeForOrg(ctx context.Context, org string, repoGlobs []string, opts upgradeOptions, createPR bool, createIssue bool, verbose bool) error {
	return runCommandForOrg(ctx, org, repoGlobs, orgRunCallbacks{
		SearchFn: searchOrgAnyWorkflowReposFn,
		// ScanFn is nil: all discovered repos are upgrade candidates and no
		// per-repo API scan is required to determine that.
		ScanFn: nil,
		ReportFn: func(results []orgRepoPreview, applying bool) {
			if applying {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Repositories with agentic workflows (%d):", len(results))))
			} else {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Dry-run preview of upgrade pull requests:"))
			}
			for _, r := range results {
				fmt.Fprintf(os.Stderr, "- %s\n", r.Repo)
			}
		},
		ApplyFn: func(ctx context.Context, preview orgRepoPreview, v bool) error {
			return runUpgradeForTargetRepoFn(ctx, preview.Repo, opts, v)
		},
		IssueFn: func(ctx context.Context, preview orgRepoPreview, v bool) error {
			return createIssueForUpgradeOrgRepoFn(ctx, preview.Repo, v)
		},
		DiscoveringMsg:  "Discovering repositories in " + org + " with agentic workflows...",
		NoReposMsg:      "No repositories found with agentic workflows",
		ApplyLabel:      "Upgrading",
		IssueLabel:      "Creating issue in",
		AllFailApplyMsg: "failed to upgrade any repository",
		AllFailIssueMsg: "failed to create issues in any repository",
	}, createPR, createIssue, verbose)
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

	releaseTag, releaseURL := getGhawReleaseInfo()
	xmlMarker := buildOrgXMLMarker(ghawUpgradeMarkerPrefix, releaseTag)

	// Close any stale upgrade PRs in the target repo before creating the new one.
	closeExistingOrgPRsByMarker(ctx, repo, ghawUpgradeMarkerPrefix, verbose)

	var releaseLine string
	if releaseURL != "" {
		releaseLine = fmt.Sprintf("\n[View gh-aw release %s](%s)\n", releaseTag, releaseURL)
	}
	prBody := "This PR upgrades agentic workflows by applying the latest codemods, " +
		"updating GitHub Actions versions, and recompiling all workflows." +
		releaseLine + "\n" + xmlMarker

	prURL, err := CreatePRWithChanges("upgrade-agentic-workflows", "chore: upgrade agentic workflows",
		"Upgrade agentic workflows", prBody, verbose)
	if err != nil {
		return err
	}

	if prURL != "" {
		addLabelToOrgPR(ctx, prURL, agenticWorkflowsLabel, verbose)
	}
	return nil
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
// to notify maintainers that agentic workflow upgrades are available. Any
// previously-open issues carrying the gh-aw-upgrade XML marker are closed first
// so that only the most recent notification remains.
func createIssueForUpgradeOrgRepo(ctx context.Context, repo string, verbose bool) error {
	title := "[aw] Upgrade available"

	releaseTag, releaseURL := getGhawReleaseInfo()
	xmlMarker := buildOrgXMLMarker(ghawUpgradeMarkerPrefix, releaseTag)

	// Close stale upgrade issues before creating the new one.
	closeExistingOrgIssuesByMarker(ctx, repo, ghawUpgradeMarkerPrefix, verbose)

	var releaseSection string
	if releaseURL != "" {
		releaseSection = fmt.Sprintf("\n[View gh-aw release %s](%s)\n", releaseTag, releaseURL)
	}

	body := "Agentic workflow files detected in this repository may have upgrades available.\n\n" +
		"Run `gh aw upgrade` to apply the latest codemods, update GitHub Actions versions, and recompile all workflows.\n\n" +
		"Review the upgrade output and any generated changes before committing to ensure there are no unexpected modifications.\n" +
		releaseSection + "\n" +
		"### How to execute\n\n" +
		"- **Assign to agent**: Assign this issue to Copilot to automatically apply the upgrade\n" +
		"- **Via @copilot comment**: Add a comment `@copilot upgrade agentic workflows` on this issue\n" +
		"- **Via CLI**: Run `gh aw upgrade` in your local checkout\n\n" +
		xmlMarker + "\n"

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Creating upgrade issue in "+repo+"..."))
	}

	if err := createOrgIssue(ctx, repo, title, body, agenticWorkflowsLabel); err != nil {
		return fmt.Errorf("failed to create issue in %s: %w", repo, err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Created issue in "+repo))
	return nil
}
