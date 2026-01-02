// Package repoutil provides utility functions for working with GitHub repository slugs and URLs.
package repoutil

import (
	"fmt"
	"strings"
)

// SplitRepoSlug splits a repository slug (owner/repo) into owner and repo parts.
// Returns an error if the slug format is invalid.
func SplitRepoSlug(slug string) (owner, repo string, err error) {
	parts := strings.Split(slug, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repo format: %s", slug)
	}
	return parts[0], parts[1], nil
}

// ParseGitHubRepoURL extracts the owner and repo from a GitHub repository URL.
// Handles both SSH (git@github.com:owner/repo.git) and HTTPS (https://github.com/owner/repo.git) formats.
// This function is specifically for parsing git remote URLs to extract repository slugs.
func ParseGitHubRepoURL(url string) (owner, repo string, err error) {
	var repoPath string

	// SSH format: git@github.com:owner/repo.git
	if strings.HasPrefix(url, "git@github.com:") {
		repoPath = strings.TrimPrefix(url, "git@github.com:")
	} else if strings.Contains(url, "github.com/") {
		// HTTPS format: https://github.com/owner/repo.git
		parts := strings.Split(url, "github.com/")
		if len(parts) >= 2 {
			repoPath = parts[1]
		}
	} else {
		return "", "", fmt.Errorf("URL does not appear to be a GitHub repository: %s", url)
	}

	// Remove .git suffix if present
	repoPath = strings.TrimSuffix(repoPath, ".git")

	// Split into owner/repo
	return SplitRepoSlug(repoPath)
}

// ExtractBaseRepo extracts the base repository (owner/repo) from an action path
// that may include subfolders.
// Examples:
//   - "actions/checkout" -> "actions/checkout"
//   - "actions/cache/restore" -> "actions/cache"
//   - "github/codeql-action/upload-sarif" -> "github/codeql-action"
func ExtractBaseRepo(actionPath string) string {
	parts := strings.Split(actionPath, "/")
	if len(parts) >= 2 {
		// Return owner/repo (first two segments)
		return parts[0] + "/" + parts[1]
	}
	// If less than 2 parts, return as-is (shouldn't happen in practice)
	return actionPath
}

// SanitizeForFilename converts a repository slug (owner/repo) to a filename-safe string.
// Replaces "/" with "-". Returns "clone-mode" if the slug is empty.
func SanitizeForFilename(slug string) string {
	if slug == "" {
		return "clone-mode"
	}
	return strings.ReplaceAll(slug, "/", "-")
}
