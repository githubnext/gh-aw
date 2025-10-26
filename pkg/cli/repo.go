package cli

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// Global cache for current repository info
var (
	getCurrentRepoSlugOnce sync.Once
	currentRepoSlugResult  string
	currentRepoSlugError   error
)

// ClearCurrentRepoSlugCache clears the current repository slug cache
// This is useful for testing or when repository context might have changed
func ClearCurrentRepoSlugCache() {
	getCurrentRepoSlugOnce = sync.Once{}
	currentRepoSlugResult = ""
	currentRepoSlugError = nil
}

// getCurrentRepoSlugUncached gets the current repository slug (owner/repo) using gh CLI (uncached)
// Falls back to git remote parsing if gh CLI is not available
func getCurrentRepoSlugUncached() (string, error) {
	// Try gh CLI first (most reliable)
	cmd := exec.Command("gh", "repo", "view", "--json", "owner,name", "--jq", ".owner.login + \"/\" + .name")
	output, err := cmd.Output()
	if err == nil {
		repoSlug := strings.TrimSpace(string(output))
		if repoSlug != "" {
			// Validate format (should be owner/repo)
			parts := strings.Split(repoSlug, "/")
			if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
				return repoSlug, nil
			}
		}
	}

	// Fallback to git remote parsing if gh CLI is not available or fails
	cmd = exec.Command("git", "remote", "get-url", "origin")
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current repository (gh CLI and git remote both failed): %w", err)
	}

	remoteURL := strings.TrimSpace(string(output))

	// Parse GitHub repository from remote URL
	// Handle both SSH and HTTPS formats
	var repoPath string

	// SSH format: git@github.com:owner/repo.git
	if strings.HasPrefix(remoteURL, "git@github.com:") {
		repoPath = strings.TrimPrefix(remoteURL, "git@github.com:")
	} else if strings.Contains(remoteURL, "github.com/") {
		// HTTPS format: https://github.com/owner/repo.git
		parts := strings.Split(remoteURL, "github.com/")
		if len(parts) >= 2 {
			repoPath = parts[1]
		}
	} else {
		return "", fmt.Errorf("remote URL does not appear to be a GitHub repository: %s", remoteURL)
	}

	// Remove .git suffix if present
	repoPath = strings.TrimSuffix(repoPath, ".git")

	// Validate format (should be owner/repo)
	parts := strings.Split(repoPath, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid repository format: %s", repoPath)
	}

	return repoPath, nil
}

// GetCurrentRepoSlug gets the current repository slug with caching using sync.Once
// This is the recommended function to use for repository access across the codebase
func GetCurrentRepoSlug() (string, error) {
	getCurrentRepoSlugOnce.Do(func() {
		currentRepoSlugResult, currentRepoSlugError = getCurrentRepoSlugUncached()
	})

	if currentRepoSlugError != nil {
		return "", currentRepoSlugError
	}

	return currentRepoSlugResult, nil
}
