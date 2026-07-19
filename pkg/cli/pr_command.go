package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/constants"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/repoutil"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/spf13/cobra"
)

var prLog = logger.New("cli:pr_command")

// PRInfo represents the details of a pull request
type PRInfo struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	State       string `json:"state"`
	HeadSHA     string `json:"headSHA"`
	BaseBranch  string `json:"baseBranch"`
	HeadBranch  string `json:"headBranch"`
	SourceRepo  string `json:"sourceRepo"`
	TargetRepo  string `json:"targetRepo"`
	AuthorLogin string `json:"authorLogin"`
}

// NewPRCommand creates the main pr command with subcommands
func NewPRCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "Pull request utilities",
		Long: `Pull request management utilities for transferring PRs between repositories.

This command provides a tool for transferring pull requests from one repository
to another, including the code changes, title, and body. This is useful for
migrating work from trial repositories to production repositories.

Available subcommands:
  - transfer - Transfer a pull request to another repository`,
		Example: `  gh aw pr transfer https://github.com/trial/repo/pull/234
  gh aw pr transfer https://github.com/source/repo/pull/123 --repo owner/target
  gh aw pr transfer https://github.com/gh-aw-trial/repo/pull/5 --repo owner/prod-repo`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(NewPRTransferSubcommand())

	return cmd
}

// NewPRTransferSubcommand creates the pr transfer subcommand
func NewPRTransferSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer <pr-url>",
		Short: "Transfer a pull request to another repository",
		Long: `Transfer a pull request from one repository to another.

This command fetches the pull request details, applies the changes as a single commit,
and creates a new pull request in the target repository with the same title and body.

The target repository defaults to the current repository unless --repo is specified.

The command will:
1. Fetch the PR details (title, body, changes)
2. Apply changes as a single squashed commit
3. Create a new PR in the target repository
4. Copy the original title and body`,
		Example: `  gh aw pr transfer https://github.com/owner/repo/pull/234
  gh aw pr transfer https://github.com/owner/repo/pull/234 --repo owner/target-repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prURL := args[0]
			targetRepo, _ := cmd.Flags().GetString("repo")
			verbose, _ := cmd.Flags().GetBool("verbose")

			if err := transferPR(prURL, targetRepo, verbose); err != nil {
				return err
			}
			return nil
		},
	}

	addRepoFlag(cmd)

	return cmd
}

// checkRepositoryAccess checks if the current user has write access to the target repository
func checkRepositoryAccess(owner, repo string) (bool, error) {
	prLog.Printf("Checking repository access: %s/%s", owner, repo)

	// Get current user
	output, err := workflow.RunGH("Fetching user info...", "api", "/user", "--jq", ".login")
	if err != nil {
		prLog.Printf("Failed to get current user: %s", err)
		return false, fmt.Errorf("failed to get current user: %w", err)
	}
	username := strings.TrimSpace(string(output))
	prLog.Printf("Current user: %s", username)

	// Check user's permission level for the repository
	output, err = workflow.RunGH("Checking repository permissions...", "api", fmt.Sprintf("/repos/%s/%s/collaborators/%s/permission", owner, repo, username))
	if err != nil {
		// If we get an error, it likely means we don't have access or the repo doesn't exist
		prLog.Print("Repository access denied or repository not found")
		return false, nil
	}

	var permissionInfo struct {
		Permission string `json:"permission"`
	}

	if err := json.Unmarshal(output, &permissionInfo); err != nil {
		return false, fmt.Errorf("failed to parse permission info: %w", err)
	}

	// Check if user has write, maintain, or admin access
	permission := permissionInfo.Permission
	hasWriteAccess := permission == "write" || permission == "maintain" || permission == "admin"
	prLog.Printf("User permission level: %s, has write access: %v", permission, hasWriteAccess)

	return hasWriteAccess, nil
}

// createForkIfNeeded creates a fork of the target repository and returns the fork repo name
func createForkIfNeeded(targetOwner, targetRepo string, verbose bool) (forkOwner, forkRepo string, err error) {
	// Get current user
	output, err := workflow.RunGH("Fetching user info...", "api", "/user", "--jq", ".login")
	if err != nil {
		return "", "", fmt.Errorf("failed to get current user: %w", err)
	}
	currentUser := strings.TrimSpace(string(output))

	// Check if fork already exists
	forkRepoSpec := fmt.Sprintf("%s/%s", currentUser, targetRepo)
	checkCmd := workflow.ExecGH("repo", "view", forkRepoSpec, "--json", "name")
	if checkCmd.Run() == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Fork already exists: "+forkRepoSpec))
		}
		return currentUser, targetRepo, nil
	}

	// Create fork
	_, err = workflow.RunGH(fmt.Sprintf("Creating fork of %s/%s...", targetOwner, targetRepo), "repo", "fork", fmt.Sprintf("%s/%s", targetOwner, targetRepo), "--clone=false")
	if err != nil {
		return "", "", fmt.Errorf("failed to create fork: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Successfully created fork: "+forkRepoSpec))
	}

	return currentUser, targetRepo, nil
}

// fetchPRInfo fetches detailed information about a pull request
func fetchPRInfo(owner, repo string, prNumber int) (*PRInfo, error) {
	prLog.Printf("Fetching PR info: %s/%s#%d", owner, repo, prNumber)

	// Fetch PR details using gh API
	output, err := workflow.RunGH("Fetching pull request info...", "api", fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, prNumber),
		"--jq", `{
			number: .number,
			title: .title,
			body: .body,
			state: .state,
			headSHA: .head.sha,
			baseBranch: .base.ref,
			headBranch: .head.ref,
			sourceRepo: .head.repo.full_name,
			targetRepo: .base.repo.full_name,
			authorLogin: .user.login
		}`)
	if err != nil {
		prLog.Printf("Failed to fetch PR info: %s", err)
		return nil, fmt.Errorf("failed to fetch PR info: %w", err)
	}

	var prInfo PRInfo
	if err := json.Unmarshal(output, &prInfo); err != nil {
		return nil, fmt.Errorf("failed to parse PR info: %w", err)
	}

	prLog.Printf("Fetched PR #%d: state=%s, author=%s", prInfo.Number, prInfo.State, prInfo.AuthorLogin)
	return &prInfo, nil
}

// createPatchFromPR creates a git patch from the PR changes using gh pr diff
func createPatchFromPR(sourceOwner, sourceRepo string, prInfo *PRInfo, verbose bool) (string, error) {
	// Create a temporary directory for the patch
	tempDir, err := os.MkdirTemp("", "gh-aw-pr-transfer-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	patchFile := filepath.Join(tempDir, "pr.patch")

	// Use gh pr diff command directly - this is the most reliable method
	diffContent, err := workflow.RunGH("Fetching pull request diff...", "pr", "diff", strconv.Itoa(prInfo.Number), "--repo", fmt.Sprintf("%s/%s", sourceOwner, sourceRepo))
	if err != nil {
		return "", fmt.Errorf("failed to get PR diff: %w", err)
	}

	if len(diffContent) == 0 {
		return "", errors.New("PR diff is empty")
	}

	// Create proper mailbox format patch that git am expects
	var patchBuilder strings.Builder

	// Required mailbox format headers for git am
	fmt.Fprintf(&patchBuilder, "From %s Mon Sep 17 00:00:00 2001\n", prInfo.HeadSHA)
	fmt.Fprintf(&patchBuilder, "From: %s <%s@users.noreply.github.com>\n", prInfo.AuthorLogin, prInfo.AuthorLogin)
	fmt.Fprintf(&patchBuilder, "Date: %s\n", time.Now().Format(time.RFC1123))
	fmt.Fprintf(&patchBuilder, "Subject: [PATCH] %s\n", prInfo.Title)
	patchBuilder.WriteString("\n")

	if prInfo.Body != "" {
		fmt.Fprintf(&patchBuilder, "%s\n", prInfo.Body)
		patchBuilder.WriteString("\n")
	}

	fmt.Fprintf(&patchBuilder, "Original-PR: %s#%d\n", prInfo.SourceRepo, prInfo.Number)
	fmt.Fprintf(&patchBuilder, "Original-Author: %s\n", prInfo.AuthorLogin)
	patchBuilder.WriteString("---\n")

	// Add the actual diff content
	patchBuilder.Write(diffContent)

	if err := os.WriteFile(patchFile, []byte(patchBuilder.String()), constants.FilePermPublic); err != nil {
		return "", fmt.Errorf("failed to write patch file: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Successfully created patch using gh pr diff"))
	}

	return patchFile, nil
}

// applyPatchToRepo applies a patch to the target repository and returns the branch name
func applyPatchToRepo(patchFile string, prInfo *PRInfo, targetOwner, targetRepo string, verbose bool) (string, error) {
	// Get current branch to restore later
	currentBranch, err := getCurrentBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	_, err = applyPatchToRepoCheckoutDefault(targetOwner, targetRepo, verbose)
	if err != nil {
		return "", err
	}

	// Create a new branch for the transfer based on the updated default branch
	branchName := fmt.Sprintf("transfer-pr-%d-%d", prInfo.Number, time.Now().Unix())
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Creating branch: "+branchName))
	}

	if err := createAndSwitchBranch(branchName, verbose); err != nil {
		return "", fmt.Errorf("failed to create new branch: %w", err)
	}

	// Apply the patch
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Applying patch..."))
		applyPatchToRepoVerbosePatchInfo(patchFile)
	}

	// Check if patch looks like a mailbox format (starts with "From ")
	patchContent, err := os.ReadFile(patchFile)
	if err != nil {
		return "", fmt.Errorf("failed to read patch file: %w", err)
	}

	var appliedWithAm bool
	isMailboxFormat := strings.HasPrefix(string(patchContent), "From ")

	if isMailboxFormat {
		appliedWithAm = applyPatchToRepoApplyMailbox(patchFile, verbose)
	}

	if !appliedWithAm {
		if err := applyPatchToRepoApplyDiff(patchFile, currentBranch, branchName, verbose); err != nil {
			return "", err
		}
	}

	// If we didn't use git am, we need to stage and commit manually
	if !appliedWithAm {
		if err := applyPatchToRepoCommit(prInfo); err != nil {
			return "", err
		}
	} else if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Applied patch using git am (includes commit)"))
	}

	return branchName, nil
}

func applyPatchToRepoCheckoutDefault(targetOwner string, targetRepo string, verbose bool) (string, error) {
	defaultBranchOutput, err := workflow.RunGH("Fetching default branch...", "api", fmt.Sprintf("/repos/%s/%s", targetOwner, targetRepo), "--jq", ".default_branch")
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}
	defaultBranch := strings.TrimSpace(string(defaultBranchOutput))
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Checking out and updating %s branch...", defaultBranch)))
	}
	cmd := exec.Command("git", "checkout", defaultBranch)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to checkout default branch %s: %w", defaultBranch, err)
	}
	cmd = exec.Command("git", "pull", "origin", defaultBranch)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to pull latest %s: %w", defaultBranch, err)
	}
	return defaultBranch, nil
}

func applyPatchToRepoVerbosePatchInfo(patchFile string) {
	patchContent, err := os.ReadFile(patchFile)
	if err != nil {
		return
	}
	lines := strings.Split(string(patchContent), "\n")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Patch file has %d lines", len(lines))))
	if len(lines) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("First line: "+lines[0]))
	}
}

func applyPatchToRepoApplyMailbox(patchFile string, verbose bool) bool {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Applying mailbox format patch with git am..."))
	}
	cmd := exec.Command("git", "am", patchFile)
	if err := cmd.Run(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Successfully applied patch with git am"))
		}
		return true
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("git am failed, trying git apply..."))
	}
	_ = exec.Command("git", "am", "--abort").Run()
	return false
}

func applyPatchToRepoApplyDiff(patchFile string, currentBranch string, branchName string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Applying patch with git apply..."))
	}
	cmd := exec.Command("git", "apply", "--3way", patchFile)
	if err := cmd.Run(); err != nil {
		return applyPatchToRepoApplyDiffFallback(patchFile, currentBranch, branchName, verbose)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Successfully applied patch with git apply"))
	}
	return nil
}

func applyPatchToRepoApplyDiffFallback(patchFile string, currentBranch string, branchName string, verbose bool) error {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("3-way merge failed, trying with whitespace options..."))
	}
	cmd := exec.Command("git", "apply", "--ignore-space-change", "--ignore-whitespace", patchFile)
	if err := cmd.Run(); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Standard apply failed, trying with --reject to see what failed..."))
			rejectCmd := exec.Command("git", "apply", "--reject", patchFile)
			rejectOutput, _ := rejectCmd.CombinedOutput()
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Patch rejection details:"))
			fmt.Fprintln(os.Stderr, string(rejectOutput))
		}
		_ = exec.Command("git", "checkout", currentBranch).Run()
		_ = exec.Command("git", "branch", "-D", branchName).Run()
		return fmt.Errorf("failed to apply patch: %w. You may need to resolve conflicts manually", err)
	}
	return nil
}

func applyPatchToRepoCommit(prInfo *PRInfo) error {
	cmd := exec.Command("git", "add", ".")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}
	commitMsg := fmt.Sprintf("Transfer PR #%d from %s\n\n%s", prInfo.Number, prInfo.SourceRepo, prInfo.Title)
	if prInfo.Body != "" {
		commitMsg += "\n\n" + prInfo.Body
	}
	commitMsg += fmt.Sprintf("\n\nOriginal-PR: %s#%d", prInfo.SourceRepo, prInfo.Number)
	commitMsg += "\nOriginal-Author: " + prInfo.AuthorLogin
	cmd = exec.Command("git", "commit", "-m", commitMsg)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	return nil
}

// createTransferPR creates a new PR in the target repository
func createTransferPR(targetOwner, targetRepo string, prInfo *PRInfo, branchName string, verbose bool) error {
	// Check if user has write access to target repository
	hasWriteAccess, err := checkRepositoryAccess(targetOwner, targetRepo)
	if err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not check repository access: %v", err)))
	}

	var forkOwner, forkRepo string
	var needsFork bool

	if !hasWriteAccess {
		forkOwner, forkRepo, err = createTransferPRFork(targetOwner, targetRepo, verbose)
		if err != nil {
			return err
		}
		needsFork = true
	}

	// Push the branch
	if err := createTransferPRPush(branchName, needsFork, verbose); err != nil {
		return err
	}

	// Create PR body with original info
	prBody := createTransferPRBody(prInfo)

	// Create the PR
	repoFlag := fmt.Sprintf("%s/%s", targetOwner, targetRepo)
	headRef := createTransferPRHeadRef(forkOwner, branchName, needsFork)

	output, err := workflow.RunGH("Creating pull request...", "pr", "create",
		"--repo", repoFlag,
		"--title", prInfo.Title,
		"--body", prBody,
		"--head", headRef)
	if err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("PR created successfully!"))
	if needsFork {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("PR created from fork %s/%s to %s/%s", forkOwner, forkRepo, targetOwner, targetRepo)))
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("URL: "+strings.TrimSpace(string(output))))

	return nil
}

func createTransferPRFork(targetOwner string, targetRepo string, verbose bool) (string, string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No write access to target repository, using fork workflow..."))
	}
	forkOwner, forkRepo, err := createForkIfNeeded(targetOwner, targetRepo, verbose)
	if err != nil {
		return "", "", fmt.Errorf("failed to create fork: %w", err)
	}
	if err := createTransferPRForkRemote(forkOwner, forkRepo, verbose); err != nil {
		return "", "", err
	}
	if err := createTransferPRUpstreamRemote(targetOwner, targetRepo, verbose); err != nil {
		return "", "", err
	}
	return forkOwner, forkRepo, nil
}

func createTransferPRForkRemote(forkOwner string, forkRepo string, verbose bool) error {
	remoteName := "fork"
	githubHost := getGitHubHost()
	forkRepoURL := fmt.Sprintf("%s/%s/%s.git", githubHost, forkOwner, forkRepo)
	checkRemoteCmd := exec.Command("git", "remote", "get-url", remoteName)
	if checkRemoteCmd.Run() == nil {
		return nil
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Adding fork remote: "+forkRepoURL))
	}
	addRemoteCmd := exec.Command("git", "remote", "add", remoteName, forkRepoURL)
	if err := addRemoteCmd.Run(); err != nil {
		return fmt.Errorf("failed to add fork remote: %w", err)
	}
	return nil
}

func createTransferPRUpstreamRemote(targetOwner string, targetRepo string, verbose bool) error {
	upstreamRemote := "upstream"
	targetRepoURL := fmt.Sprintf("https://github.com/%s/%s.git", targetOwner, targetRepo)
	checkUpstreamCmd := exec.Command("git", "remote", "get-url", upstreamRemote)
	upstreamOutput, err := checkUpstreamCmd.Output()
	if err == nil && strings.TrimSpace(string(upstreamOutput)) == targetRepoURL {
		return nil
	}
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Adding upstream remote: "+targetRepoURL))
		}
		addUpstreamCmd := exec.Command("git", "remote", "add", upstreamRemote, targetRepoURL)
		if err := addUpstreamCmd.Run(); err != nil {
			return fmt.Errorf("failed to add upstream remote: %w", err)
		}
		return nil
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Updating upstream remote: "+targetRepoURL))
	}
	setUpstreamCmd := exec.Command("git", "remote", "set-url", upstreamRemote, targetRepoURL)
	if err := setUpstreamCmd.Run(); err != nil {
		return fmt.Errorf("failed to update upstream remote: %w", err)
	}
	return nil
}

func createTransferPRPush(branchName string, needsFork bool, verbose bool) error {
	if verbose {
		if needsFork {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Pushing branch to fork..."))
		} else {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Pushing branch to remote..."))
		}
	}
	remote := "origin"
	if needsFork {
		remote = "fork"
	}
	pushCmd := exec.Command("git", "push", "-u", remote, branchName)
	if err := pushCmd.Run(); err != nil {
		if needsFork {
			return fmt.Errorf("failed to push branch to fork: %w", err)
		}
		return fmt.Errorf("failed to push branch: %w", err)
	}
	return nil
}

func createTransferPRBody(prInfo *PRInfo) string {
	prBody := prInfo.Body
	if prBody != "" {
		prBody += "\n\n---\n\n"
	}
	prBody += fmt.Sprintf("**Transferred from:** %s#%d\n", prInfo.SourceRepo, prInfo.Number)
	prBody += "**Original Author:** @" + prInfo.AuthorLogin
	return prBody
}

func createTransferPRHeadRef(forkOwner string, branchName string, needsFork bool) string {
	if needsFork {
		return fmt.Sprintf("%s:%s", forkOwner, branchName)
	}
	return branchName
}

// transferPR is the main function that orchestrates the PR transfer
func transferPR(prURL, targetRepo string, verbose bool) error {
	prLog.Printf("Starting PR transfer: url=%s, targetRepo=%s", prURL, targetRepo)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Starting PR transfer..."))
	}

	// Parse PR URL
	sourceOwner, sourceRepoName, prNumber, err := parser.ParsePRURL(prURL)
	if err != nil {
		prLog.Printf("Failed to parse PR URL: %s", err)
		return err
	}
	prLog.Printf("Parsed source: %s/%s#%d", sourceOwner, sourceRepoName, prNumber)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Source: %s/%s PR #%d", sourceOwner, sourceRepoName, prNumber)))
	}

	targetOwner, targetRepoName, err := transferPRTarget(targetRepo)
	if err != nil {
		return err
	}

	prLog.Printf("Determined target repository: %s/%s", targetOwner, targetRepoName)

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Target: %s/%s", targetOwner, targetRepoName)))
	}

	// Check if source and target are the same
	if sourceOwner == targetOwner && sourceRepoName == targetRepoName {
		prLog.Print("Source and target repositories are the same - aborting")
		return errors.New("source and target repositories cannot be the same")
	}

	// Ensure we're in the correct git repository
	workingDir, needsCleanup, err := transferPRWorkingDir(targetRepo, targetOwner, targetRepoName, verbose)
	if err != nil {
		return err
	}

	// Cleanup function
	defer func() {
		transferPRCleanup(workingDir, needsCleanup, verbose)
	}()

	return transferPRCreate(targetOwner, targetRepoName, sourceOwner, sourceRepoName, prNumber, verbose)
}

func transferPRTarget(targetRepo string) (string, string, error) {
	if targetRepo != "" {
		repoSpec, err := parseRepoSpec(targetRepo)
		if err != nil {
			return "", "", fmt.Errorf("invalid target repository format: %w", err)
		}
		parts := strings.SplitN(repoSpec.RepoSlug, "/", 2)
		if len(parts) != 2 {
			return "", "", errors.New("invalid target repository format, expected: owner/repo")
		}
		return parts[0], parts[1], nil
	}
	slug, err := GetCurrentRepoSlug()
	if err != nil {
		return "", "", fmt.Errorf("failed to determine target repository: %w", err)
	}
	owner, repo, err := repoutil.SplitRepoSlug(slug)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse target repository: %w", err)
	}
	return owner, repo, nil
}

func transferPRWorkingDir(targetRepo string, targetOwner string, targetRepoName string, verbose bool) (string, bool, error) {
	if targetRepo == "" {
		if !isGitRepo() {
			return "", false, errors.New("not in a git repository")
		}
		return ".", false, nil
	}
	if isGitRepo() {
		slug, err := GetCurrentRepoSlug()
		if err == nil {
			currentOwner, currentRepoName, err := repoutil.SplitRepoSlug(slug)
			if err == nil && currentOwner == targetOwner && currentRepoName == targetRepoName {
				return ".", false, nil
			}
		}
	}
	tempDir, err := transferPRCloneTarget(targetOwner, targetRepoName, verbose)
	if err != nil {
		return "", false, err
	}
	return tempDir, true, nil
}

func transferPRCloneTarget(targetOwner string, targetRepoName string, verbose bool) (string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Cloning target repository %s/%s...", targetOwner, targetRepoName)))
	}
	tempDir, err := os.MkdirTemp("", "gh-aw-pr-transfer-repo-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory for repo: %w", err)
	}
	cloneCmd := workflow.ExecGH("repo", "clone", fmt.Sprintf("%s/%s", targetOwner, targetRepoName), tempDir)
	if err := cloneCmd.Run(); err != nil {
		transferPRCleanup(tempDir, true, verbose)
		return "", fmt.Errorf("failed to clone target repository: %w", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		transferPRCleanup(tempDir, true, verbose)
		return "", fmt.Errorf("failed to change to cloned repository directory: %w", err)
	}
	return tempDir, nil
}

func transferPRCleanup(workingDir string, needsCleanup bool, verbose bool) {
	if needsCleanup && workingDir != "" {
		if err := os.RemoveAll(workingDir); err != nil && verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("failed to clean up temporary directory %s: %v", workingDir, err)))
		}
	}
}

func transferPRCreate(targetOwner string, targetRepoName string, sourceOwner string, sourceRepoName string, prNumber int, verbose bool) error {
	// Fetch PR information
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Fetching PR details..."))
	}

	prInfo, err := fetchPRInfo(sourceOwner, sourceRepoName, prNumber)
	if err != nil {
		return err
	}

	if prInfo.State != "open" && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Warning: PR is in '%s' state", prInfo.State)))
	}

	// Create patch from PR
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Creating patch from PR changes..."))
	}

	patchFile, err := createPatchFromPR(sourceOwner, sourceRepoName, prInfo, verbose)
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(patchFile)) // Clean up temp directory

	// Apply patch to target repository
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Applying changes to target repository..."))
	}

	branchName, err := applyPatchToRepo(patchFile, prInfo, targetOwner, targetRepoName, verbose)
	if err != nil {
		return err
	}

	// Create PR in target repository
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Creating new PR in target repository..."))
	}

	if err := createTransferPR(targetOwner, targetRepoName, prInfo, branchName, verbose); err != nil {
		return err
	}

	return nil
}

// createPR creates a pull request using GitHub CLI and returns the PR number
func createPR(branchName, title, body string, verbose bool) (int, string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatProgressMessage("Creating PR: "+title))
	}

	// Detect the GitHub host from the git remote so that GitHub Enterprise Server
	// repositories are targeted correctly instead of defaulting to github.com.
	remoteHost := getHostFromOriginRemote()

	// Get the current repository info to ensure PR is created in the correct repo.
	// Use GH_HOST env var instead of --hostname (which is only valid for gh api, not gh repo view).
	repoOutput, err := workflow.RunGHWithHost("Fetching repository info...", remoteHost, "repo", "view", "--json", "owner,name")
	if err != nil {
		return 0, "", fmt.Errorf("failed to get current repository info: %w", err)
	}

	var repoInfo struct {
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	}

	if err := json.Unmarshal(repoOutput, &repoInfo); err != nil {
		return 0, "", fmt.Errorf("failed to parse repository info: %w", err)
	}

	repoSpec := fmt.Sprintf("%s/%s", repoInfo.Owner.Login, repoInfo.Name)

	// Build gh pr create args. Explicitly specifying --repo ensures the PR is created in the
	// current repo (not an upstream fork). Use GH_HOST env var instead of --hostname
	// (which is only valid for gh api, not gh pr create).
	prCreateArgs := []string{"pr", "create", "--repo", repoSpec, "--title", title, "--body", body, "--head", branchName}
	output, err := workflow.RunGHWithHost("Creating pull request...", remoteHost, prCreateArgs...)
	if err != nil {
		// Try to get stderr for better error reporting
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return 0, "", fmt.Errorf("failed to create PR: %w\nOutput: %s\nError: %s", err, string(output), string(exitError.Stderr))
		}
		return 0, "", fmt.Errorf("failed to create PR: %w", err)
	}

	prURL := strings.TrimSpace(string(output))

	// Parse PR number from URL (e.g., https://github.com/owner/repo/pull/123)
	prNumber := 0
	parts := strings.Split(prURL, "/")
	if len(parts) > 0 {
		if num, parseErr := strconv.Atoi(parts[len(parts)-1]); parseErr == nil {
			prNumber = num
		}
	}

	return prNumber, prURL, nil
}
