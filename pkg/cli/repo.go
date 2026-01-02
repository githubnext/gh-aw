package cli

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/repoutil"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

var repoLog = logger.New("cli:repo")

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
	repoLog.Print("Fetching current repository slug")

	// Try gh CLI first (most reliable)
	repoLog.Print("Attempting to get repository slug via gh CLI")
	cmd := workflow.ExecGH("repo", "view", "--json", "owner,name", "--jq", ".owner.login + \"/\" + .name")
	output, err := cmd.Output()
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
	cmd = exec.Command("git", "remote", "get-url", "origin")
	output, err = cmd.Output()
	if err != nil {
		repoLog.Printf("Failed to get git remote URL: %v", err)
		return "", fmt.Errorf("failed to get current repository (gh CLI and git remote both failed): %w", err)
	}

	remoteURL := strings.TrimSpace(string(output))
	repoLog.Printf("Parsing git remote URL: %s", remoteURL)

	// Parse GitHub repository from remote URL using repoutil
	owner, repo, err := repoutil.ParseGitHubRepoURL(remoteURL)
	if err != nil {
		repoLog.Printf("Failed to parse git remote URL: %v", err)
		return "", fmt.Errorf("failed to parse git remote URL: %w. Expected format: owner/repo. Example: githubnext/gh-aw", err)
	}

	repoSlug := owner + "/" + repo
	repoLog.Printf("Successfully parsed repository slug from git remote: %s", repoSlug)
	return repoSlug, nil
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

	repoLog.Printf("Using cached repository slug: %s", currentRepoSlugResult)
	return currentRepoSlugResult, nil
}

// SplitRepoSlug wraps repoutil.SplitRepoSlug for backward compatibility.
// It splits a repository slug (owner/repo) into owner and repo parts.
// New code should use repoutil.SplitRepoSlug directly.
func SplitRepoSlug(slug string) (owner, repo string, err error) {
	return repoutil.SplitRepoSlug(slug)
}
