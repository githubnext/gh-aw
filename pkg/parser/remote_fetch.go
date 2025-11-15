package parser

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var remoteLog = logger.New("parser:remote_fetch")

// isUnderWorkflowsDirectory checks if a file path is a top-level workflow file (not in shared subdirectory)
func isUnderWorkflowsDirectory(filePath string) bool {
	// Normalize the path to use forward slashes
	normalizedPath := filepath.ToSlash(filePath)

	// Check if the path contains .github/workflows/
	if !strings.Contains(normalizedPath, ".github/workflows/") {
		return false
	}

	// Extract the part after .github/workflows/
	parts := strings.Split(normalizedPath, ".github/workflows/")
	if len(parts) < 2 {
		return false
	}

	afterWorkflows := parts[1]

	// Check if there are any slashes after .github/workflows/ (indicating subdirectory)
	// If there are, it's in a subdirectory like "shared/" and should not be treated as a workflow file
	return !strings.Contains(afterWorkflows, "/")
}

// resolveIncludePath resolves include path based on workflowspec format or relative path
func resolveIncludePath(filePath, baseDir string, cache *ImportCache) (string, error) {
	// Check if this is a workflowspec (contains owner/repo/path format)
	// Format: owner/repo/path@ref or owner/repo/path@ref#section
	if isWorkflowSpec(filePath) {
		// Download from GitHub using workflowspec (with cache support)
		return downloadIncludeFromWorkflowSpec(filePath, cache)
	}

	// Regular path, resolve relative to base directory
	fullPath := filepath.Join(baseDir, filePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", fullPath)
	}
	return fullPath, nil
}

// isWorkflowSpec checks if a path looks like a workflowspec (owner/repo/path[@ref])
func isWorkflowSpec(path string) bool {
	// Remove section reference if present
	cleanPath := path
	if idx := strings.Index(path, "#"); idx != -1 {
		cleanPath = path[:idx]
	}

	// Remove ref if present
	if idx := strings.Index(cleanPath, "@"); idx != -1 {
		cleanPath = cleanPath[:idx]
	}

	// Check if it has at least 3 parts (owner/repo/path)
	parts := strings.Split(cleanPath, "/")
	if len(parts) < 3 {
		return false
	}

	// Reject paths that start with "." (local paths like .github/workflows/...)
	if strings.HasPrefix(cleanPath, ".") {
		return false
	}

	// Reject paths that start with "shared/" (local shared files)
	if strings.HasPrefix(cleanPath, "shared/") {
		return false
	}

	// Reject absolute paths
	if strings.HasPrefix(cleanPath, "/") {
		return false
	}

	return true
}

// downloadIncludeFromWorkflowSpec downloads an include file from GitHub using workflowspec
// It first checks the cache, and only downloads if not cached
func downloadIncludeFromWorkflowSpec(spec string, cache *ImportCache) (string, error) {
	// Parse the workflowspec
	// Format: owner/repo/path@ref or owner/repo/path@ref#section

	// Remove section reference if present
	cleanSpec := spec
	if idx := strings.Index(spec, "#"); idx != -1 {
		cleanSpec = spec[:idx]
	}

	// Split on @ to get path and ref
	parts := strings.SplitN(cleanSpec, "@", 2)
	pathPart := parts[0]
	var ref string
	if len(parts) == 2 {
		ref = parts[1]
	} else {
		ref = "main" // default to main branch
	}

	// Parse path: owner/repo/path/to/file.md
	slashParts := strings.Split(pathPart, "/")
	if len(slashParts) < 3 {
		return "", fmt.Errorf("invalid workflowspec: must be owner/repo/path[@ref]")
	}

	owner := slashParts[0]
	repo := slashParts[1]
	filePath := strings.Join(slashParts[2:], "/")

	// Resolve ref to SHA for cache lookup
	var sha string
	if cache != nil {
		// Only resolve SHA if we're using the cache
		resolvedSHA, err := resolveRefToSHA(owner, repo, ref)
		if err != nil {
			// If the error is an authentication error, propagate it immediately
			lowerErr := strings.ToLower(err.Error())
			if strings.Contains(lowerErr, "auth") || strings.Contains(lowerErr, "unauthoriz") || strings.Contains(lowerErr, "forbidden") || strings.Contains(lowerErr, "token") || strings.Contains(lowerErr, "permission denied") {
				return "", fmt.Errorf("failed to resolve ref to SHA due to authentication error: %w", err)
			}
			remoteLog.Printf("Failed to resolve ref to SHA, will skip cache: %v", err)
			// Continue without caching if SHA resolution fails
		} else {
			sha = resolvedSHA
			// Check cache using SHA
			if cachedPath, found := cache.Get(owner, repo, filePath, sha); found {
				remoteLog.Printf("Using cached import: %s/%s/%s@%s (SHA: %s)", owner, repo, filePath, ref, sha)
				return cachedPath, nil
			}
		}
	}

	// Download the file content from GitHub
	content, err := downloadFileFromGitHub(owner, repo, filePath, ref)
	if err != nil {
		return "", fmt.Errorf("failed to download include from %s: %w", spec, err)
	}

	// If cache is available and we have a SHA, store in cache
	if cache != nil && sha != "" {
		cachedPath, err := cache.Set(owner, repo, filePath, sha, content)
		if err != nil {
			remoteLog.Printf("Failed to cache import: %v", err)
			// Don't fail the compilation, fall back to temp file
		} else {
			return cachedPath, nil
		}
	}

	// Fallback: Create a temporary file to store the downloaded content
	tempFile, err := os.CreateTemp("", "gh-aw-include-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := tempFile.Write(content); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return tempFile.Name(), nil
}

// resolveRefToSHA resolves a git ref (branch, tag, or SHA) to its commit SHA
func resolveRefToSHA(owner, repo, ref string) (string, error) {
	// If ref is already a full SHA (40 hex characters), return it as-is
	if len(ref) == 40 && isHexString(ref) {
		return ref, nil
	}

	// Use gh CLI to get the commit SHA for the ref
	// This works for branches, tags, and short SHAs
	cmd := exec.Command("gh", "api", fmt.Sprintf("/repos/%s/%s/commits/%s", owner, repo, ref), "--jq", ".sha")

	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		if strings.Contains(outputStr, "GH_TOKEN") || strings.Contains(outputStr, "authentication") || strings.Contains(outputStr, "not logged into") {
			return "", fmt.Errorf("failed to resolve ref to SHA: GitHub authentication required. Please run 'gh auth login' or set GH_TOKEN/GITHUB_TOKEN environment variable: %w", err)
		}
		return "", fmt.Errorf("failed to resolve ref %s to SHA for %s/%s: %s: %w", ref, owner, repo, strings.TrimSpace(outputStr), err)
	}

	sha := strings.TrimSpace(string(output))
	if sha == "" {
		return "", fmt.Errorf("empty SHA returned for ref %s in %s/%s", ref, owner, repo)
	}

	// Validate it's a valid SHA (40 hex characters)
	if len(sha) != 40 || !isHexString(sha) {
		return "", fmt.Errorf("invalid SHA format returned: %s", sha)
	}

	return sha, nil
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

func downloadFileFromGitHub(owner, repo, path, ref string) ([]byte, error) {
	// Use go-gh/v2 to download the file
	stdout, stderr, err := gh.Exec("api", fmt.Sprintf("/repos/%s/%s/contents/%s?ref=%s", owner, repo, path, ref), "--jq", ".content")

	if err != nil {
		// Check if this is an authentication error
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "GH_TOKEN") || strings.Contains(stderrStr, "authentication") || strings.Contains(stderrStr, "not logged into") {
			return nil, fmt.Errorf("failed to fetch file content: GitHub authentication required. Please run 'gh auth login' or set GH_TOKEN/GITHUB_TOKEN environment variable: %w", err)
		}
		return nil, fmt.Errorf("failed to fetch file content from %s/%s/%s@%s: %s: %w", owner, repo, path, ref, strings.TrimSpace(stderrStr), err)
	}

	// The content is base64 encoded, decode it
	contentBase64 := strings.TrimSpace(stdout.String())
	if contentBase64 == "" {
		return nil, fmt.Errorf("empty content returned from GitHub API for %s/%s/%s@%s", owner, repo, path, ref)
	}

	decodeCmd := exec.Command("base64", "-d")
	decodeCmd.Stdin = strings.NewReader(contentBase64)
	content, err := decodeCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 content: %w", err)
	}

	return content, nil
}
