package cli

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/repoutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var repoLog = logger.New("cli:repo")

// repoSlugCacheState holds the cached repository slug and protects it with a mutex.
// Using a mutex-guarded struct instead of sync.Once avoids the data race that arises
// when resetting sync.Once via struct assignment (= sync.Once{}) after first use.
type repoSlugCacheState struct {
	mu     sync.Mutex
	result string
	err    error
	done   bool
}

// Global cache for current repository info
var currentRepoSlugCache repoSlugCacheState

// ClearCurrentRepoSlugCache clears the current repository slug cache.
// This is useful for testing or when repository context might have changed.
func ClearCurrentRepoSlugCache() {
	currentRepoSlugCache.mu.Lock()
	defer currentRepoSlugCache.mu.Unlock()
	currentRepoSlugCache.result = ""
	currentRepoSlugCache.err = nil
	currentRepoSlugCache.done = false
}

// getCurrentRepoSlugUncached gets the current repository slug (owner/repo) using gh CLI (uncached)
// Falls back to git remote parsing if gh CLI is not available
func getCurrentRepoSlugUncached() (string, error) {
	repoLog.Print("Fetching current repository slug")

	// Try gh CLI first (most reliable)
	repoLog.Print("Attempting to get repository slug via gh CLI")
	output, err := workflow.RunGH("Fetching repository info...", "repo", "view", "--json", "owner,name", "--jq", ".owner.login + \"/\" + .name")
	if err == nil {
		repoSlug := strings.TrimSpace(string(output))
		if repoSlug != "" {
			// Validate format (should be owner/repo)
			parts := strings.Split(repoSlug, "/")
			if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
				repoLog.Printf("Successfully got repository slug via gh CLI: %s", repoSlug)
				return repoSlug, nil
			}
		}
	}

	// Fallback to git remote parsing if gh CLI is not available or fails
	repoLog.Print("gh CLI failed, falling back to git remote parsing")
	gitCmd := exec.Command("git", "remote", "get-url", "origin")
	gitOutput, err := gitCmd.Output()
	if err != nil {
		repoLog.Printf("Failed to get git remote URL: %v", err)
		return "", fmt.Errorf("failed to get current repository (gh CLI and git remote both failed): %w", err)
	}

	remoteURL := strings.TrimSpace(string(gitOutput))
	repoLog.Printf("Parsing git remote URL: %s", remoteURL)

	// Parse GitHub repository from remote URL
	// Handle both SSH and HTTPS formats
	var repoPath string

	// SSH format: git@github.com:owner/repo.git
	if after, ok := strings.CutPrefix(remoteURL, "git@github.com:"); ok {
		repoPath = after
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
		repoLog.Printf("Invalid repository format: %s", repoPath)
		return "", fmt.Errorf("invalid repository format: %s. Expected format: owner/repo. Example: github/gh-aw", repoPath)
	}

	repoLog.Printf("Successfully parsed repository slug from git remote: %s", repoPath)
	return repoPath, nil
}

// GetCurrentRepoSlug gets the current repository slug with caching.
// This is the recommended function to use for repository access across the codebase.
func GetCurrentRepoSlug() (string, error) {
	currentRepoSlugCache.mu.Lock()
	if !currentRepoSlugCache.done {
		currentRepoSlugCache.result, currentRepoSlugCache.err = getCurrentRepoSlugUncached()
		currentRepoSlugCache.done = true
	}
	result := currentRepoSlugCache.result
	err := currentRepoSlugCache.err
	currentRepoSlugCache.mu.Unlock()

	if err != nil {
		return "", err
	}

	repoLog.Printf("Using cached repository slug: %s", result)
	return result, nil
}

// SplitRepoSlug wraps repoutil.SplitRepoSlug for backward compatibility.
// It splits a repository slug (owner/repo) into owner and repo parts.
// New code should use repoutil.SplitRepoSlug directly.
func SplitRepoSlug(slug string) (owner, repo string, err error) {
	return repoutil.SplitRepoSlug(slug)
}
