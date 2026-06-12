package cli

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/syncutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var repoLog = logger.New("cli:repo")

// Global cache for current repository info
var currentRepoSlugCache syncutil.OnceLoader[string]

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

	// Delegate to the shared helper which supports both HTTPS and SSH formats,
	// including GitHub Enterprise hosts configured via getGitHubHost().
	repoPath := parseGitHubRepoSlugFromURL(remoteURL)
	if repoPath == "" {
		return "", fmt.Errorf("remote URL does not appear to be a GitHub repository: %s", remoteURL)
	}

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
	result, err := currentRepoSlugCache.Get(getCurrentRepoSlugUncached)

	if err != nil {
		return "", err
	}

	repoLog.Printf("Using cached repository slug: %s", result)
	return result, nil
}
