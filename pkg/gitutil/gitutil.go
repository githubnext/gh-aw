package gitutil

import (
	"fmt"
	"os/exec"
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
