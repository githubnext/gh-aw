package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var downloadLog = logger.New("cli:download_workflow")

// isAuthError checks if an error message indicates an authentication issue
func isAuthError(errMsg string) bool {
	lowerMsg := strings.ToLower(errMsg)
	return strings.Contains(lowerMsg, "gh_token") ||
		strings.Contains(lowerMsg, "github_token") ||
		strings.Contains(lowerMsg, "authentication") ||
		strings.Contains(lowerMsg, "not logged into") ||
		strings.Contains(lowerMsg, "unauthorized") ||
		strings.Contains(lowerMsg, "forbidden") ||
		strings.Contains(lowerMsg, "permission denied")
}

// isHexString checks if a string contains only hexadecimal characters
func isHexString(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// resolveLatestReleaseViaGit finds the latest release using git ls-remote
func resolveLatestReleaseViaGit(repo, currentRef string, allowMajor, verbose bool) (string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching latest release for %s via git ls-remote (current: %s, allow major: %v)", repo, currentRef, allowMajor)))
	}

	repoURL := fmt.Sprintf("https://github.com/%s.git", repo)

	// List all tags
	cmd := exec.Command("git", "ls-remote", "--tags", repoURL)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to fetch releases via git ls-remote: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var releases []string

	for _, line := range lines {
		// Parse: "<sha> refs/tags/<tag>"
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			tagRef := parts[1]
			// Skip ^{} annotations (they point to the commit object)
			if strings.HasSuffix(tagRef, "^{}") {
				continue
			}
			tag := strings.TrimPrefix(tagRef, "refs/tags/")
			releases = append(releases, tag)
		}
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("no releases found")
	}

	// Parse current version
	currentVersion := parseVersion(currentRef)
	if currentVersion == nil {
		// If current ref is not a valid version, just return the first release
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Current ref is not a valid version, using first release: %s (via git)", releases[0])))
		}
		return releases[0], nil
	}

	// Find the latest compatible release
	var latestCompatible string
	var latestCompatibleVersion *semanticVersion

	for _, release := range releases {
		releaseVersion := parseVersion(release)
		if releaseVersion == nil {
			continue
		}

		// Check if compatible based on major version
		if !allowMajor && releaseVersion.major != currentVersion.major {
			continue
		}

		// Check if this is newer than what we have
		if latestCompatibleVersion == nil || releaseVersion.isNewer(latestCompatibleVersion) {
			latestCompatible = release
			latestCompatibleVersion = releaseVersion
		}
	}

	if latestCompatible == "" {
		return "", fmt.Errorf("no compatible release found")
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Latest compatible release: %s (via git)", latestCompatible)))
	}

	return latestCompatible, nil
}

// isBranchRefViaGit checks if a ref is a branch using git ls-remote
func isBranchRefViaGit(repo, ref string) (bool, error) {
	downloadLog.Printf("Attempting git ls-remote to check if ref is branch: %s@%s", repo, ref)

	repoURL := fmt.Sprintf("https://github.com/%s.git", repo)

	// List all branches and check if ref matches
	cmd := exec.Command("git", "ls-remote", "--heads", repoURL)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to list branches via git ls-remote: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		// Format: <sha> refs/heads/<branch>
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			branchRef := parts[1]
			branchName := strings.TrimPrefix(branchRef, "refs/heads/")
			if branchName == ref {
				downloadLog.Printf("Found branch via git ls-remote: %s", ref)
				return true, nil
			}
		}
	}

	return false, nil
}

// isBranchRef checks if a ref is a branch in the repository
func isBranchRef(repo, ref string) (bool, error) {
	// Use gh CLI to list branches
	cmd := workflow.ExecGH("api", fmt.Sprintf("/repos/%s/branches", repo), "--jq", ".[].name")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if this is an authentication error
		outputStr := string(output)
		if isAuthError(outputStr) || isAuthError(err.Error()) {
			downloadLog.Printf("GitHub API authentication failed, attempting git ls-remote fallback")
			// Try fallback using git ls-remote
			isBranch, gitErr := isBranchRefViaGit(repo, ref)
			if gitErr != nil {
				return false, fmt.Errorf("failed to check branch via GitHub API and git: API error: %w, Git error: %v", err, gitErr)
			}
			return isBranch, nil
		}
		return false, err
	}

	branches := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, branch := range branches {
		if branch == ref {
			return true, nil
		}
	}

	return false, nil
}

// resolveBranchHeadViaGit gets the latest commit SHA for a branch using git ls-remote
func resolveBranchHeadViaGit(repo, branch string, verbose bool) (string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching latest commit for branch %s in %s via git ls-remote", branch, repo)))
	}

	repoURL := fmt.Sprintf("https://github.com/%s.git", repo)

	// Get the SHA for the specific branch
	cmd := exec.Command("git", "ls-remote", repoURL, fmt.Sprintf("refs/heads/%s", branch))
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to fetch branch info via git ls-remote: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || len(lines[0]) == 0 {
		return "", fmt.Errorf("branch %s not found", branch)
	}

	// Parse the output: "<sha> refs/heads/<branch>"
	parts := strings.Fields(lines[0])
	if len(parts) < 1 {
		return "", fmt.Errorf("invalid git ls-remote output")
	}

	sha := parts[0]
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Latest commit on %s: %s (via git)", branch, sha)))
	}

	return sha, nil
}

// resolveBranchHead gets the latest commit SHA for a branch
func resolveBranchHead(repo, branch string, verbose bool) (string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching latest commit for branch %s in %s", branch, repo)))
	}

	// Use gh CLI to get branch info
	cmd := workflow.ExecGH("api", fmt.Sprintf("/repos/%s/branches/%s", repo, branch), "--jq", ".commit.sha")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if this is an authentication error
		outputStr := string(output)
		if isAuthError(outputStr) || isAuthError(err.Error()) {
			downloadLog.Printf("GitHub API authentication failed, attempting git ls-remote fallback")
			// Try fallback using git ls-remote
			sha, gitErr := resolveBranchHeadViaGit(repo, branch, verbose)
			if gitErr != nil {
				return "", fmt.Errorf("failed to fetch branch info via GitHub API and git: API error: %w, Git error: %v", err, gitErr)
			}
			return sha, nil
		}
		return "", fmt.Errorf("failed to fetch branch info: %w", err)
	}

	sha := strings.TrimSpace(string(output))
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Latest commit on %s: %s", branch, sha)))
	}

	return sha, nil
}

// resolveDefaultBranchHeadViaGit gets the latest commit SHA for the default branch using git ls-remote
func resolveDefaultBranchHeadViaGit(repo string, verbose bool) (string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching default branch for %s via git ls-remote", repo)))
	}

	repoURL := fmt.Sprintf("https://github.com/%s.git", repo)

	// Get HEAD to find default branch
	cmd := exec.Command("git", "ls-remote", "--symref", repoURL, "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to fetch repository info via git ls-remote: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return "", fmt.Errorf("unexpected git ls-remote output format")
	}

	// First line is: "ref: refs/heads/<branch> HEAD"
	// Second line is: "<sha> HEAD"
	var defaultBranch string
	var sha string

	for _, line := range lines {
		if strings.HasPrefix(line, "ref:") {
			// Parse: "ref: refs/heads/<branch> HEAD"
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				refPath := parts[1]
				defaultBranch = strings.TrimPrefix(refPath, "refs/heads/")
			}
		} else {
			// Parse: "<sha> HEAD"
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				sha = parts[0]
			}
		}
	}

	if defaultBranch == "" || sha == "" {
		return "", fmt.Errorf("failed to parse default branch or SHA from git ls-remote output")
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Default branch: %s (via git)", defaultBranch)))
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Latest commit on %s: %s (via git)", defaultBranch, sha)))
	}

	return sha, nil
}

// resolveDefaultBranchHead gets the latest commit SHA for the default branch
func resolveDefaultBranchHead(repo string, verbose bool) (string, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching default branch for %s", repo)))
	}

	// First get the default branch name
	cmd := workflow.ExecGH("api", fmt.Sprintf("/repos/%s", repo), "--jq", ".default_branch")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if this is an authentication error
		outputStr := string(output)
		if isAuthError(outputStr) || isAuthError(err.Error()) {
			downloadLog.Printf("GitHub API authentication failed, attempting git ls-remote fallback")
			// Try fallback using git ls-remote to get HEAD
			sha, gitErr := resolveDefaultBranchHeadViaGit(repo, verbose)
			if gitErr != nil {
				return "", fmt.Errorf("failed to fetch repository info via GitHub API and git: API error: %w, Git error: %v", err, gitErr)
			}
			return sha, nil
		}
		return "", fmt.Errorf("failed to fetch repository info: %w", err)
	}

	defaultBranch := strings.TrimSpace(string(output))
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Default branch: %s", defaultBranch)))
	}

	return resolveBranchHead(repo, defaultBranch, verbose)
}

// downloadWorkflowContentViaGit downloads a workflow file using git archive
func downloadWorkflowContentViaGit(repo, path, ref string, verbose bool) ([]byte, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching %s/%s@%s via git", repo, path, ref)))
	}

	downloadLog.Printf("Attempting git fallback for downloading workflow content: %s/%s@%s", repo, path, ref)

	// Use git archive to get the file content without cloning
	repoURL := fmt.Sprintf("https://github.com/%s.git", repo)

	// git archive command: git archive --remote=<repo> <ref> <path>
	cmd := exec.Command("git", "archive", "--remote="+repoURL, ref, path)
	archiveOutput, err := cmd.Output()
	if err != nil {
		// If git archive fails, try with git clone + read file as a fallback
		return downloadWorkflowContentViaGitClone(repo, path, ref, verbose)
	}

	// Extract the file from the tar archive
	tarCmd := exec.Command("tar", "-xO", path)
	tarCmd.Stdin = strings.NewReader(string(archiveOutput))
	content, err := tarCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to extract file from git archive: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Successfully fetched via git archive"))
	}

	return content, nil
}

// downloadWorkflowContentViaGitClone downloads a workflow file by shallow cloning
func downloadWorkflowContentViaGitClone(repo, path, ref string, verbose bool) ([]byte, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching %s/%s@%s via git clone", repo, path, ref)))
	}

	downloadLog.Printf("Attempting git clone fallback for downloading workflow content: %s/%s@%s", repo, path, ref)

	// Create a temporary directory for the shallow clone
	tmpDir, err := os.MkdirTemp("", "gh-aw-git-clone-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	repoURL := fmt.Sprintf("https://github.com/%s.git", repo)

	// Check if ref is a SHA (40 hex characters)
	isSHA := len(ref) == 40 && isHexString(ref)

	var cloneCmd *exec.Cmd
	if isSHA {
		// For SHA refs, we need to clone without --branch and then checkout the specific commit
		// Clone with minimal depth and no branch specified
		cloneCmd = exec.Command("git", "clone", "--depth", "1", "--no-single-branch", repoURL, tmpDir)
		if _, err := cloneCmd.CombinedOutput(); err != nil {
			// Try without --no-single-branch
			cloneCmd = exec.Command("git", "clone", repoURL, tmpDir)
			if output, err := cloneCmd.CombinedOutput(); err != nil {
				return nil, fmt.Errorf("failed to clone repository: %w\nOutput: %s", err, string(output))
			}
		}

		// Now checkout the specific commit
		checkoutCmd := exec.Command("git", "-C", tmpDir, "checkout", ref)
		if output, err := checkoutCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("failed to checkout commit %s: %w\nOutput: %s", ref, err, string(output))
		}
	} else {
		// For branch/tag refs, use --branch flag
		cloneCmd = exec.Command("git", "clone", "--depth", "1", "--branch", ref, repoURL, tmpDir)
		if output, err := cloneCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("failed to clone repository: %w\nOutput: %s", err, string(output))
		}
	}

	// Read the file
	filePath := filepath.Join(tmpDir, path)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file from cloned repository: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Successfully fetched via git clone"))
	}

	return content, nil
}

// downloadWorkflowContent downloads the content of a workflow file from GitHub
func downloadWorkflowContent(repo, path, ref string, verbose bool) ([]byte, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching %s/%s@%s", repo, path, ref)))
	}

	// Use gh CLI to download the file
	cmd := workflow.ExecGH("api", fmt.Sprintf("/repos/%s/contents/%s?ref=%s", repo, path, ref), "--jq", ".content")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if this is an authentication error
		outputStr := string(output)
		if isAuthError(outputStr) || isAuthError(err.Error()) {
			downloadLog.Printf("GitHub API authentication failed, attempting git fallback for %s/%s@%s", repo, path, ref)
			// Try fallback using git commands
			content, gitErr := downloadWorkflowContentViaGit(repo, path, ref, verbose)
			if gitErr != nil {
				return nil, fmt.Errorf("failed to fetch file content via GitHub API and git: API error: %w, Git error: %v", err, gitErr)
			}
			return content, nil
		}
		return nil, fmt.Errorf("failed to fetch file content: %w", err)
	}

	// The content is base64 encoded, decode it
	contentBase64 := strings.TrimSpace(string(output))
	cmd = exec.Command("base64", "-d")
	cmd.Stdin = strings.NewReader(contentBase64)
	content, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to decode file content: %w", err)
	}

	return content, nil
}
