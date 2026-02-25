package cli

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var downloadLog = logger.New("cli:download_workflow")

// downloadWorkflowContentViaGit downloads a workflow file using git archive
func downloadWorkflowContentViaGit(repo, path, ref string, verbose bool) ([]byte, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching %s/%s@%s via git", repo, path, ref)))
	}

	downloadLog.Printf("Attempting git fallback for downloading workflow content: %s/%s@%s", repo, path, ref)

	// Use git archive to get the file content without cloning
	githubHost := getGitHubHostForRepo(repo)
	repoURL := fmt.Sprintf("%s/%s.git", githubHost, repo)

	// git archive command: git archive --remote=<repo> <ref> <path>
	// #nosec G204 -- repoURL, ref, and path are from workflow import configuration authored by the
	// developer; exec.Command with separate args (not shell execution) prevents shell injection.
	cmd := exec.Command("git", "archive", "--remote="+repoURL, ref, path)
	archiveOutput, err := cmd.Output()
	if err != nil {
		// If git archive fails, try with git clone + read file as a fallback
		return downloadWorkflowContentViaGitClone(repo, path, ref, verbose)
	}

	// Extract the file from the tar archive using Go's archive/tar (cross-platform)
	content, err := fileutil.ExtractFileFromTar(archiveOutput, path)
	if err != nil {
		return nil, fmt.Errorf("failed to extract file from git archive: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Successfully fetched via git archive"))
	}

	return content, nil
}

// downloadWorkflowContentViaGitClone downloads a workflow file by shallow cloning with sparse checkout
func downloadWorkflowContentViaGitClone(repo, path, ref string, verbose bool) ([]byte, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching %s/%s@%s via git clone", repo, path, ref)))
	}

	downloadLog.Printf("Attempting git clone fallback for downloading workflow content: %s/%s@%s", repo, path, ref)

	// Create a temporary directory for the sparse checkout
	tmpDir, err := os.MkdirTemp("", "gh-aw-git-clone-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	githubHost := getGitHubHostForRepo(repo)
	repoURL := fmt.Sprintf("%s/%s.git", githubHost, repo)

	// Initialize git repository
	initCmd := exec.Command("git", "-C", tmpDir, "init")
	if output, err := initCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to initialize git repository: %w\nOutput: %s", err, string(output))
	}

	// Add remote
	remoteCmd := exec.Command("git", "-C", tmpDir, "remote", "add", "origin", repoURL)
	if output, err := remoteCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to add remote: %w\nOutput: %s", err, string(output))
	}

	// Enable sparse-checkout
	sparseCmd := exec.Command("git", "-C", tmpDir, "config", "core.sparseCheckout", "true")
	if output, err := sparseCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to enable sparse checkout: %w\nOutput: %s", err, string(output))
	}

	// Set sparse-checkout pattern to only include the file we need
	sparseInfoDir := filepath.Join(tmpDir, ".git", "info")
	if err := os.MkdirAll(sparseInfoDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sparse-checkout directory: %w", err)
	}

	sparseCheckoutFile := filepath.Join(sparseInfoDir, "sparse-checkout")
	// Use owner-only read/write permissions (0600) for security best practices
	if err := os.WriteFile(sparseCheckoutFile, []byte(path+"\n"), 0600); err != nil {
		return nil, fmt.Errorf("failed to write sparse-checkout file: %w", err)
	}

	// Check if ref is a SHA (40 hex characters)
	isSHA := len(ref) == 40 && gitutil.IsHexString(ref)

	if isSHA {
		// For SHA refs, fetch without specifying a ref (fetch all) then checkout the specific commit
		// Note: sparse-checkout with SHA refs may not reduce bandwidth as much as with branch refs,
		// because the server needs to send enough history to reach the specific commit.
		// However, it still limits the working directory to only the requested file.
		fetchCmd := exec.Command("git", "-C", tmpDir, "fetch", "--depth", "1", "origin", ref)
		if _, err := fetchCmd.CombinedOutput(); err != nil {
			// If fetching specific SHA fails, try fetching all branches with depth 1
			fetchCmd = exec.Command("git", "-C", tmpDir, "fetch", "--depth", "1", "origin")
			if output, err := fetchCmd.CombinedOutput(); err != nil {
				return nil, fmt.Errorf("failed to fetch repository: %w\nOutput: %s", err, string(output))
			}
		}

		// Checkout the specific commit
		checkoutCmd := exec.Command("git", "-C", tmpDir, "checkout", ref)
		if output, err := checkoutCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("failed to checkout commit %s: %w\nOutput: %s", ref, err, string(output))
		}
	} else {
		// For branch/tag refs, fetch the specific ref
		fetchCmd := exec.Command("git", "-C", tmpDir, "fetch", "--depth", "1", "origin", ref)
		if output, err := fetchCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("failed to fetch ref %s: %w\nOutput: %s", ref, err, string(output))
		}

		// Checkout FETCH_HEAD
		checkoutCmd := exec.Command("git", "-C", tmpDir, "checkout", "FETCH_HEAD")
		if output, err := checkoutCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("failed to checkout FETCH_HEAD: %w\nOutput: %s", err, string(output))
		}
	}

	// Read the file
	filePath := filepath.Join(tmpDir, path)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file from cloned repository: %w", err)
	}

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage("Successfully fetched via git sparse checkout"))
	}

	return content, nil
}

// downloadWorkflowContent downloads the content of a workflow file from GitHub
func downloadWorkflowContent(repo, path, ref string, verbose bool) ([]byte, error) {
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatVerboseMessage(fmt.Sprintf("Fetching %s/%s@%s", repo, path, ref)))
	}

	// Use gh CLI to download the file
	output, err := workflow.RunGHCombined("Downloading workflow...", "api", fmt.Sprintf("/repos/%s/contents/%s?ref=%s", repo, path, ref), "--jq", ".content")
	if err != nil {
		// Check if this is an authentication error
		outputStr := string(output)
		if gitutil.IsAuthError(outputStr) || gitutil.IsAuthError(err.Error()) {
			downloadLog.Printf("GitHub API authentication failed, attempting git fallback for %s/%s@%s", repo, path, ref)
			// Try fallback using git commands
			content, gitErr := downloadWorkflowContentViaGit(repo, path, ref, verbose)
			if gitErr != nil {
				return nil, fmt.Errorf("failed to fetch file content via GitHub API and git: API error: %w, Git error: %w", err, gitErr)
			}
			return content, nil
		}
		return nil, fmt.Errorf("failed to fetch file content: %w", err)
	}

	// The content is base64 encoded, decode it
	return decodeBase64FileContent(string(output))
}

// decodeBase64FileContent decodes base64-encoded file content returned by the GitHub API.
// The GitHub API wraps lines at 60 characters and may include surrounding whitespace,
// so both are stripped before decoding.
func decodeBase64FileContent(raw string) ([]byte, error) {
	cleaned := strings.ReplaceAll(strings.TrimSpace(raw), "\n", "")
	content, err := base64.StdEncoding.DecodeString(cleaned)
	if err != nil {
		return nil, fmt.Errorf("failed to decode file content: %w", err)
	}
	return content, nil
}
