package gitutil

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var log = logger.New("gitutil:gitutil")

// IsRateLimitError checks if an error message indicates a GitHub API rate limit error.
// This is used to detect transient failures caused by hitting the GitHub API rate limit
// (HTTP 403 "API rate limit exceeded" or HTTP 429 responses).
func IsRateLimitError(errMsg string) bool {
	lowerMsg := strings.ToLower(errMsg)
	return strings.Contains(lowerMsg, "api rate limit exceeded") ||
		strings.Contains(lowerMsg, "rate limit exceeded") ||
		strings.Contains(lowerMsg, "secondary rate limit")
}

// IsAuthError checks if an error message indicates an authentication issue.
// This is used to detect when GitHub API calls fail due to missing or invalid credentials.
func IsAuthError(errMsg string) bool {
	log.Printf("Checking if error is auth-related: %s", errMsg)
	lowerMsg := strings.ToLower(errMsg)
	isAuth := strings.Contains(lowerMsg, "gh_token") ||
		strings.Contains(lowerMsg, "github_token") ||
		strings.Contains(lowerMsg, "authentication") ||
		strings.Contains(lowerMsg, "not logged into") ||
		strings.Contains(lowerMsg, "unauthorized") ||
		strings.Contains(lowerMsg, "forbidden") ||
		strings.Contains(lowerMsg, "permission denied") ||
		strings.Contains(lowerMsg, "saml enforcement")
	if isAuth {
		log.Print("Detected authentication error")
	}
	return isAuth
}

// IsHexString checks if a string contains only hexadecimal characters.
// This is used to validate Git commit SHAs and other hexadecimal identifiers.
func IsHexString(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// ExtractBaseRepo extracts the base repository (owner/repo) from a repository path
// that may include subfolders.
// For "actions/checkout" -> "actions/checkout"
// For "github/codeql-action/upload-sarif" -> "github/codeql-action"
func ExtractBaseRepo(repoPath string) string {
	parts := strings.Split(repoPath, "/")
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return repoPath
}

// FindGitRoot finds the root directory of the git repository.
// Returns an error if not in a git repository or if the git command fails.
func FindGitRoot() (string, error) {
	log.Print("Finding git root directory")
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to find git root: %v", err)
		return "", fmt.Errorf("not in a git repository or git command failed: %w", err)
	}
	gitRoot := strings.TrimSpace(string(output))
	log.Printf("Found git root: %s", gitRoot)
	return gitRoot, nil
}

// ReadFileFromHEAD returns the content of filePath as recorded in the most recent
// git commit (HEAD). This is used in safe update mode so that the manifest baseline
// comes from the committed version of the lock file, not from the working-tree copy,
// preventing a local agent from tampering with the file to bypass enforcement.
//
// filePath may be absolute or relative; it is resolved to an absolute path before
// computing its path relative to the git root.
//
// Returns an error when:
//   - the current directory is not inside a git repository
//   - the file does not exist in HEAD (e.g. it has never been committed)
//   - the git command fails for another reason
func ReadFileFromHEAD(filePath string) (string, error) {
	gitRoot, err := FindGitRoot()
	if err != nil {
		return "", fmt.Errorf("cannot read %q from git HEAD: %w", filePath, err)
	}

	return ReadFileFromHEADWithRoot(filePath, gitRoot)
}

// ReadFileFromHEADWithRoot is like ReadFileFromHEAD but accepts a pre-computed git
// repository root, avoiding the subprocess overhead of calling FindGitRoot().
// Use this when the caller already knows the git root (e.g. from a cached value).
func ReadFileFromHEADWithRoot(filePath, gitRoot string) (string, error) {
	if gitRoot == "" {
		return "", fmt.Errorf("gitRoot must not be empty when reading %q from HEAD", filePath)
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("cannot resolve absolute path for %q: %w", filePath, err)
	}

	// git show requires the path to be relative to the repository root and to use
	// forward slashes even on Windows.
	relPath, err := filepath.Rel(gitRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("cannot compute path of %q relative to git root %q: %w", absPath, gitRoot, err)
	}

	// Reject paths that escape the repository (e.g. "../secret").
	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("path %q is outside the git repository root %q", filePath, gitRoot)
	}

	relPath = filepath.ToSlash(relPath)

	log.Printf("Reading %q from git HEAD (relative path: %s)", filePath, relPath)

	cmd := exec.Command("git", "-C", gitRoot, "show", "HEAD:"+relPath)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("File %q not found in HEAD commit: %v", filePath, err)
		return "", fmt.Errorf("file %q not found in HEAD commit: %w", filePath, err)
	}
	return string(output), nil
}
