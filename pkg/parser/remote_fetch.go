package parser

import (
	"encoding/base64"
	"encoding/json"
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
			// Try fallback without authentication
			remoteLog.Printf("gh CLI authentication failed, trying unauthenticated REST API fallback")
			return resolveRefToSHAUnauthenticated(owner, repo, ref)
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
			// Try fallback without authentication
			remoteLog.Printf("gh CLI authentication failed, trying unauthenticated REST API fallback")
			return downloadFileFromGitHubUnauthenticated(owner, repo, path, ref)
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

// resolveRefToSHAUnauthenticated resolves a git ref to SHA using unauthenticated REST API
// This is a fallback for when gh CLI authentication is not available
func resolveRefToSHAUnauthenticated(owner, repo, ref string) (string, error) {
	remoteLog.Printf("Attempting to resolve ref %s to SHA for %s/%s using unauthenticated API", ref, owner, repo)

	// Use curl to make unauthenticated request
	// -f flag makes curl fail on HTTP errors (404, 500, etc.)
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", owner, repo, ref)
	cmd := exec.Command("curl", "-s", "-f", "-H", "Accept: application/vnd.github.v3+json", url)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to resolve ref using unauthenticated API: %w", err)
	}

	// Parse JSON response
	var response struct {
		SHA     string `json:"sha"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Check for error message in response
	if response.Message != "" {
		if strings.Contains(response.Message, "Not Found") {
			return "", fmt.Errorf("ref %s not found in %s/%s", ref, owner, repo)
		}
		if strings.Contains(response.Message, "rate limit") {
			return "", fmt.Errorf("GitHub API rate limit exceeded")
		}
		return "", fmt.Errorf("GitHub API error: %s", response.Message)
	}

	// Validate it's a valid SHA (40 hex characters)
	if len(response.SHA) != 40 || !isHexString(response.SHA) {
		return "", fmt.Errorf("invalid SHA format returned: %s", response.SHA)
	}

	remoteLog.Printf("Successfully resolved ref %s to SHA %s using unauthenticated API", ref, response.SHA)
	return response.SHA, nil
}

// downloadFileFromGitHubUnauthenticated downloads a file using unauthenticated REST API
// This is a fallback for when gh CLI authentication is not available
func downloadFileFromGitHubUnauthenticated(owner, repo, path, ref string) ([]byte, error) {
	remoteLog.Printf("Attempting to download %s/%s/%s@%s using unauthenticated API", owner, repo, path, ref)

	// Use curl to make unauthenticated request
	// -f flag makes curl fail on HTTP errors (404, 500, etc.)
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", owner, repo, path, ref)
	cmd := exec.Command("curl", "-s", "-f", "-H", "Accept: application/vnd.github.v3+json", url)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file using unauthenticated API: %w", err)
	}

	// Parse JSON response
	var response struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
		Message  string `json:"message"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Check for error message in response
	if response.Message != "" {
		if strings.Contains(response.Message, "Not Found") {
			return nil, fmt.Errorf("file %s not found in %s/%s@%s", path, owner, repo, ref)
		}
		if strings.Contains(response.Message, "rate limit") {
			return nil, fmt.Errorf("GitHub API rate limit exceeded")
		}
		return nil, fmt.Errorf("GitHub API error: %s", response.Message)
	}

	// Verify encoding
	if response.Encoding != "base64" {
		return nil, fmt.Errorf("unexpected encoding: %s (expected base64)", response.Encoding)
	}

	// Remove newlines and whitespace from base64 content
	contentBase64 := strings.ReplaceAll(response.Content, "\n", "")
	contentBase64 = strings.ReplaceAll(contentBase64, " ", "")
	contentBase64 = strings.TrimSpace(contentBase64)

	if contentBase64 == "" {
		return nil, fmt.Errorf("empty content returned from GitHub API")
	}

	// Decode base64 content using Go's standard library (more portable than external base64 command)
	content, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 content: %w", err)
	}

	remoteLog.Printf("Successfully downloaded %s/%s/%s@%s using unauthenticated API (%d bytes)", owner, repo, path, ref, len(content))
	return content, nil
}
