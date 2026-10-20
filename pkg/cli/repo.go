package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/repoutil"
	"github.com/github/gh-aw/pkg/syncutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var repoLog = logger.New("cli:repo")

// Global cache for current repository info
var currentRepoSlugCache syncutil.OnceLoader[string]

// getCurrentRepoSlugUncached prefers the local origin over a hosted lookup,
// while preserving gh's explicit GH_REPO override.
func getCurrentRepoSlugUncached() (string, error) {
	repoLog.Print("Fetching current repository slug")

	if os.Getenv("GH_REPO") == "" { //nolint:osgetenvlibrary // Preserve gh's explicit repository selection at the CLI boundary.
		if repoSlug, err := getCurrentRepoSlugFromGit(); err == nil {
			return repoSlug, nil
		} else {
			repoLog.Printf("Local repository lookup unavailable: %v", err)
		}
	}

	repoLog.Print("Attempting to get repository slug via gh CLI")
	output, err := workflow.RunGH("Fetching repository info...", "repo", "view", "--json", "owner,name", "--jq", ".owner.login + \"/\" + .name")
	if err == nil {
		repoSlug := strings.TrimSpace(string(output))
		if repoSlug != "" {
			// Validate format (should be owner/repo)
			if _, _, err := repoutil.SplitRepoSlug(repoSlug); err == nil {
				repoLog.Printf("Successfully got repository slug via gh CLI: %s", repoSlug)
				return repoSlug, nil
			}
		}
	}

	if err != nil {
		return "", fmt.Errorf("failed to get current repository via gh CLI: %w", err)
	}
	return "", errors.New("failed to get current repository: gh CLI returned an invalid repository slug")
}

func getCurrentRepoSlugFromGit() (string, error) {
	gitCmd := exec.Command("git", "remote", "get-url", "origin")
	gitOutput, err := gitCmd.Output()
	if err != nil {
		repoLog.Printf("Failed to get git remote URL: %v", err)
		return "", fmt.Errorf("failed to get local Git origin: %w", err)
	}

	remoteURL := strings.TrimSpace(string(gitOutput))
	// Delegate to the shared helper which supports both HTTPS and SSH formats,
	// including GitHub Enterprise hosts configured via getGitHubHost().
	repoPath := parseGitHubRepoSlugFromURL(remoteURL)
	if repoPath == "" {
		return "", errors.New("local origin does not identify a GitHub repository")
	}

	// Validate format (should be owner/repo)
	if _, _, err := repoutil.SplitRepoSlug(repoPath); err != nil {
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
