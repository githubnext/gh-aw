//go:build !js && !wasm

package parser

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
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

// isCustomAgentFile checks if a file path is a custom agent file under .github/agents/
// Custom agent files use GitHub Copilot's agent format, which differs from gh-aw workflow format.
// These files have a different schema for the 'tools' field (array vs object).
func isCustomAgentFile(filePath string) bool {
	// Normalize the path to use forward slashes
	normalizedPath := filepath.ToSlash(filePath)

	// Check if the path contains .github/agents/ and ends with .md
	return strings.Contains(normalizedPath, ".github/agents/") && strings.HasSuffix(strings.ToLower(normalizedPath), ".md")
}

// isRepositoryImport checks if an import spec is a repository-only import (no file path)
// Format: owner/repo@ref or owner/repo (downloads entire .github folder, no agent extraction)
func isRepositoryImport(importPath string) bool {
	// Remove section reference if present
	cleanPath := importPath
	if idx := strings.Index(importPath, "#"); idx != -1 {
		cleanPath = importPath[:idx]
	}

	// Remove ref if present to check the path structure
	pathWithoutRef := cleanPath
	if idx := strings.Index(cleanPath, "@"); idx != -1 {
		pathWithoutRef = cleanPath[:idx]
	}

	// Split by slash to count parts
	parts := strings.Split(pathWithoutRef, "/")

	// Repository import has exactly 2 parts: owner/repo
	// File imports have 1 part (local file) or 3+ parts (owner/repo/path/to/file)
	if len(parts) != 2 {
		return false
	}

	// Reject local paths
	if strings.HasPrefix(pathWithoutRef, ".") || strings.HasPrefix(pathWithoutRef, "/") {
		return false
	}

	// Reject paths that start with common local directory names
	if strings.HasPrefix(pathWithoutRef, "shared/") {
		return false
	}

	// Additional validation: check if it looks like a valid owner/repo format
	// GitHub identifiers can't start with numbers, must be alphanumeric with hyphens/underscores
	owner := parts[0]
	repo := parts[1]

	// Basic validation - ensure they're not empty and don't look like file extensions
	if owner == "" || repo == "" {
		return false
	}

	// Reject if repo part looks like a file extension (ends with .md, .yaml, etc.)
	if strings.Contains(repo, ".") {
		return false
	}

	return true
}

// ResolveIncludePath resolves include path based on workflowspec format or relative path
func ResolveIncludePath(filePath, baseDir string, cache *ImportCache) (string, error) {
	remoteLog.Printf("Resolving include path: file_path=%s, base_dir=%s", filePath, baseDir)

	// Check if this is a workflowspec (contains owner/repo/path format)
	// Format: owner/repo/path@ref or owner/repo/path@ref#section
	if isWorkflowSpec(filePath) {
		remoteLog.Printf("Detected workflowspec format: %s", filePath)
		// Download from GitHub using workflowspec (with cache support)
		return downloadIncludeFromWorkflowSpec(filePath, cache)
	}

	remoteLog.Printf("Using local file resolution for: %s", filePath)
	// Regular path, resolve relative to base directory
	fullPath := filepath.Join(baseDir, filePath)

	// Security check: ensure the resolved path is within the .github folder
	// baseDir should be .github or a subdirectory within it
	githubFolder := baseDir
	// Find the .github folder by traversing up from baseDir
	for !strings.HasSuffix(githubFolder, ".github") && githubFolder != "." && githubFolder != "/" {
		githubFolder = filepath.Dir(githubFolder)
		if githubFolder == "." || githubFolder == "/" {
			// If we can't find .github folder, use baseDir
			githubFolder = baseDir
			break
		}
	}

	// Normalize paths for comparison
	normalizedGithubFolder := filepath.Clean(githubFolder)
	normalizedFullPath := filepath.Clean(fullPath)

	// Check if fullPath is within githubFolder
	relativePath, err := filepath.Rel(normalizedGithubFolder, normalizedFullPath)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) || filepath.IsAbs(relativePath) {
		remoteLog.Printf("Security: Path escapes .github folder: %s (resolves to: %s)", filePath, relativePath)
		return "", fmt.Errorf("security: path %s must be within .github folder (resolves to: %s)", filePath, relativePath)
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		remoteLog.Printf("Local file not found: %s", fullPath)
		// Return a simple error that will be wrapped with source location by the caller
		return "", fmt.Errorf("file not found: %s", fullPath)
	}
	remoteLog.Printf("Resolved to local file: %s", fullPath)
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
	remoteLog.Printf("Downloading from workflowspec: %s", spec)

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
		remoteLog.Print("No ref specified, defaulting to 'main'")
	}

	// Parse path: owner/repo/path/to/file.md
	slashParts := strings.Split(pathPart, "/")
	if len(slashParts) < 3 {
		remoteLog.Printf("Invalid workflowspec format: %s", spec)
		return "", fmt.Errorf("invalid workflowspec: must be owner/repo/path[@ref]")
	}

	owner := slashParts[0]
	repo := slashParts[1]
	filePath := strings.Join(slashParts[2:], "/")
	remoteLog.Printf("Parsed workflowspec: owner=%s, repo=%s, file=%s, ref=%s", owner, repo, filePath, ref)

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
	remoteLog.Printf("Fetching file from GitHub: %s/%s/%s@%s", owner, repo, filePath, ref)
	content, err := downloadFileFromGitHub(owner, repo, filePath, ref)
	if err != nil {
		return "", fmt.Errorf("failed to download include from %s: %w", spec, err)
	}
	remoteLog.Printf("Successfully downloaded file: size=%d bytes", len(content))

	// If cache is available and we have a SHA, store in cache
	if cache != nil && sha != "" {
		cachedPath, err := cache.Set(owner, repo, filePath, sha, content)
		if err != nil {
			remoteLog.Printf("Failed to cache import: %v", err)
			// Don't fail the compilation, fall back to temp file
		} else {
			remoteLog.Printf("Successfully cached download at: %s", cachedPath)
			return cachedPath, nil
		}
	}

	// Fallback: Create a temporary file to store the downloaded content
	tempFile, err := os.CreateTemp("", "gh-aw-include-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := tempFile.Write(content); err != nil {
		// Close the temp file and clean up, logging any errors
		if closeErr := tempFile.Close(); closeErr != nil {
			remoteLog.Printf("Warning: failed to close temp file during cleanup: %v", closeErr)
		}
		if rmErr := os.Remove(tempFile.Name()); rmErr != nil {
			remoteLog.Printf("Warning: failed to remove temp file %s: %v", tempFile.Name(), rmErr)
		}
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		// Clean up temp file if close fails
		if rmErr := os.Remove(tempFile.Name()); rmErr != nil {
			remoteLog.Printf("Warning: failed to remove temp file %s: %v", tempFile.Name(), rmErr)
		}
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return tempFile.Name(), nil
}

// resolveRefToSHAViaGit resolves a git ref to SHA using git ls-remote
// This is a fallback for when GitHub API authentication fails
func resolveRefToSHAViaGit(owner, repo, ref string) (string, error) {
	remoteLog.Printf("Attempting git ls-remote fallback for ref resolution: %s/%s@%s", owner, repo, ref)

	githubHost := GetGitHubHostForRepo(owner, repo)
	repoURL := fmt.Sprintf("%s/%s/%s.git", githubHost, owner, repo)

	// Try to resolve the ref using git ls-remote
	// Format: git ls-remote <repo> <ref>
	cmd := exec.Command("git", "ls-remote", repoURL, ref)
	output, err := cmd.Output()
	if err != nil {
		// If exact ref doesn't work, try with refs/heads/ and refs/tags/ prefixes
		for _, prefix := range []string{"refs/heads/", "refs/tags/"} {
			cmd = exec.Command("git", "ls-remote", repoURL, prefix+ref)
			output, err = cmd.Output()
			if err == nil && len(output) > 0 {
				break
			}
		}

		if err != nil {
			return "", fmt.Errorf("failed to resolve ref via git ls-remote: %w", err)
		}
	}

	// Parse the output: "<sha> <ref>"
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || len(lines[0]) == 0 {
		return "", fmt.Errorf("no matching ref found for %s", ref)
	}

	// Extract SHA from the first line
	parts := strings.Fields(lines[0])
	if len(parts) < 1 {
		return "", fmt.Errorf("invalid git ls-remote output format")
	}

	sha := parts[0]

	// Validate it's a valid SHA
	if len(sha) != 40 || !gitutil.IsHexString(sha) {
		return "", fmt.Errorf("invalid SHA format from git ls-remote: %s", sha)
	}

	remoteLog.Printf("Successfully resolved ref via git ls-remote: %s/%s@%s -> %s", owner, repo, ref, sha)
	return sha, nil
}

// resolveRefToSHA resolves a git ref (branch, tag, or SHA) to its commit SHA
func resolveRefToSHA(owner, repo, ref string) (string, error) {
	// If ref is already a full SHA (40 hex characters), return it as-is
	if len(ref) == 40 && gitutil.IsHexString(ref) {
		return ref, nil
	}

	// Use gh CLI to get the commit SHA for the ref
	// This works for branches, tags, and short SHAs
	// Using go-gh to properly handle enterprise GitHub instances via GH_HOST
	stdout, stderr, err := gh.Exec("api", fmt.Sprintf("/repos/%s/%s/commits/%s", owner, repo, ref), "--jq", ".sha")

	if err != nil {
		outputStr := stderr.String()
		if gitutil.IsAuthError(outputStr) {
			remoteLog.Printf("GitHub API authentication failed, attempting git ls-remote fallback for %s/%s@%s", owner, repo, ref)
			// Try fallback using git ls-remote for public repositories
			sha, gitErr := resolveRefToSHAViaGit(owner, repo, ref)
			if gitErr != nil {
				// If git fallback also fails, return both errors
				return "", fmt.Errorf("failed to resolve ref via GitHub API (auth error) and git ls-remote: API error: %w, Git error: %v", err, gitErr)
			}
			return sha, nil
		}
		return "", fmt.Errorf("failed to resolve ref %s to SHA for %s/%s: %s: %w", ref, owner, repo, strings.TrimSpace(outputStr), err)
	}

	sha := strings.TrimSpace(stdout.String())
	if sha == "" {
		return "", fmt.Errorf("empty SHA returned for ref %s in %s/%s", ref, owner, repo)
	}

	// Validate it's a valid SHA (40 hex characters)
	if len(sha) != 40 || !gitutil.IsHexString(sha) {
		return "", fmt.Errorf("invalid SHA format returned: %s", sha)
	}

	return sha, nil
}

// downloadFileViaGit downloads a file from a Git repository using git commands
// This is a fallback for when GitHub API authentication fails
func downloadFileViaGit(owner, repo, path, ref string) ([]byte, error) {
	remoteLog.Printf("Attempting git fallback for %s/%s/%s@%s", owner, repo, path, ref)

	// Use git archive to get the file content without cloning
	// This works for public repositories without authentication
	githubHost := GetGitHubHostForRepo(owner, repo)
	repoURL := fmt.Sprintf("%s/%s/%s.git", githubHost, owner, repo)

	// git archive command: git archive --remote=<repo> <ref> <path>
	cmd := exec.Command("git", "archive", "--remote="+repoURL, ref, path)
	archiveOutput, err := cmd.Output()
	if err != nil {
		// If git archive fails, try with git clone + git show as a fallback
		return downloadFileViaGitClone(owner, repo, path, ref)
	}

	// Extract the file from the tar archive
	// git archive outputs a tar archive containing the requested file
	tarCmd := exec.Command("tar", "-xO", path)
	tarCmd.Stdin = strings.NewReader(string(archiveOutput))
	content, err := tarCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to extract file from git archive: %w", err)
	}

	remoteLog.Printf("Successfully downloaded file via git archive: %s/%s/%s@%s", owner, repo, path, ref)
	return content, nil
}

// downloadFileViaGitClone downloads a file by shallow cloning the repository
// This is used as a fallback when git archive doesn't work
func downloadFileViaGitClone(owner, repo, path, ref string) ([]byte, error) {
	remoteLog.Printf("Attempting git clone fallback for %s/%s/%s@%s", owner, repo, path, ref)

	// Create a temporary directory for the shallow clone
	tmpDir, err := os.MkdirTemp("", "gh-aw-git-clone-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	githubHost := GetGitHubHostForRepo(owner, repo)
	repoURL := fmt.Sprintf("%s/%s/%s.git", githubHost, owner, repo)

	// Check if ref is a SHA (40 hex characters)
	isSHA := len(ref) == 40 && gitutil.IsHexString(ref)

	var cloneCmd *exec.Cmd
	if isSHA {
		// For SHA refs, we need to clone without --branch and then checkout the specific commit
		// Clone with minimal depth and no branch specified
		cloneCmd = exec.Command("git", "clone", "--depth", "1", "--no-single-branch", repoURL, tmpDir)
		if output, err := cloneCmd.CombinedOutput(); err != nil {
			// Try without --no-single-branch if the first attempt fails
			remoteLog.Printf("Clone with --no-single-branch failed, trying full clone: %s", string(output))
			cloneCmd = exec.Command("git", "clone", repoURL, tmpDir)
			if output, err := cloneCmd.CombinedOutput(); err != nil {
				return nil, fmt.Errorf("failed to clone repository: %w\nOutput: %s", err, string(output))
			}
		}

		// Now checkout the specific commit
		checkoutCmd := exec.Command("git", "-C", tmpDir, "checkout", ref)
		if output, err := checkoutCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("failed to checkout commit %s: %w\nOutput: %s", ref, err, string(output))
		}
	} else {
		// For branch/tag refs, use --branch flag
		cloneCmd = exec.Command("git", "clone", "--depth", "1", "--branch", ref, repoURL, tmpDir)
		if output, err := cloneCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("failed to clone repository: %w\nOutput: %s", err, string(output))
		}
	}

	// Read the file from the cloned repository
	filePath := filepath.Join(tmpDir, path)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file from cloned repository: %w", err)
	}

	remoteLog.Printf("Successfully downloaded file via git clone: %s/%s/%s@%s", owner, repo, path, ref)
	return content, nil
}

// isNotFoundError checks if an error message indicates a 404 Not Found response
func isNotFoundError(errMsg string) bool {
	lowerMsg := strings.ToLower(errMsg)
	return strings.Contains(lowerMsg, "404") || strings.Contains(lowerMsg, "not found")
}

// checkRemoteSymlink checks if a path in a remote GitHub repository is a symlink.
// Returns the symlink target and true if it is a symlink, or empty string and false otherwise.
// A nil error with false means the path is not a symlink (e.g., it's a directory or file).
func checkRemoteSymlink(client *api.RESTClient, owner, repo, dirPath, ref string) (string, bool, error) {
	endpoint := fmt.Sprintf("repos/%s/%s/contents/%s?ref=%s", owner, repo, dirPath, ref)
	remoteLog.Printf("Checking if path component is symlink: %s/%s/%s@%s", owner, repo, dirPath, ref)

	// The Contents API returns a JSON object for files/symlinks but a JSON array for directories.
	// Decode into json.RawMessage first to distinguish these cases without error-driven control flow.
	var raw json.RawMessage
	err := client.Get(endpoint, &raw)
	if err != nil {
		remoteLog.Printf("Contents API error for %s: %v", dirPath, err)
		return "", false, err
	}

	// If the response is an array, this is a directory listing — not a symlink
	trimmed := strings.TrimSpace(string(raw))
	if len(trimmed) > 0 && trimmed[0] == '[' {
		remoteLog.Printf("Path component %s is a directory (not a symlink)", dirPath)
		return "", false, nil
	}

	// Parse the object response to check the type
	var result struct {
		Type   string `json:"type"`
		Target string `json:"target"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", false, fmt.Errorf("failed to parse contents response for %s: %w", dirPath, err)
	}

	if result.Type == "symlink" && result.Target != "" {
		remoteLog.Printf("Path component %s is a symlink -> %s", dirPath, result.Target)
		return result.Target, true, nil
	}

	remoteLog.Printf("Path component %s is type=%s (not a symlink)", dirPath, result.Type)
	return "", false, nil
}

// resolveRemoteSymlinks resolves symlinks in a remote GitHub repository path.
// The GitHub Contents API doesn't follow symlinks in path components. For example,
// if .github/workflows/shared is a symlink to ../../gh-agent-workflows/shared,
// fetching .github/workflows/shared/elastic-tools.md returns 404.
// This function walks the path components and resolves any symlinks found.
func resolveRemoteSymlinks(owner, repo, filePath, ref string) (string, error) {
	parts := strings.Split(filePath, "/")
	if len(parts) <= 1 {
		return "", fmt.Errorf("no directory components to resolve in path: %s", filePath)
	}

	remoteLog.Printf("Attempting symlink resolution for %s/%s/%s@%s (%d path components)", owner, repo, filePath, ref, len(parts))

	client, err := api.DefaultRESTClient()
	if err != nil {
		return "", fmt.Errorf("failed to create REST client: %w", err)
	}

	// Check each directory prefix (not including the final filename) to find symlinks
	for i := 1; i < len(parts); i++ {
		dirPath := strings.Join(parts[:i], "/")

		target, isSymlink, err := checkRemoteSymlink(client, owner, repo, dirPath, ref)
		if err != nil {
			// Only ignore 404s (path component doesn't exist yet at this prefix level).
			// Propagate real API failures (auth, rate limit, network) immediately.
			if isNotFoundError(err.Error()) {
				remoteLog.Printf("Path component %s returned 404, skipping", dirPath)
				continue
			}
			return "", fmt.Errorf("failed to check path component %s for symlinks: %w", dirPath, err)
		}

		if isSymlink {
			// Resolve the symlink target relative to the symlink's parent directory.
			// For example, if .github/workflows/shared is a symlink to ../../gh-agent-workflows/shared,
			// the parent is .github/workflows and the resolved base is gh-agent-workflows/shared.
			parentDir := ""
			if i > 1 {
				parentDir = strings.Join(parts[:i-1], "/")
			}

			remoteLog.Printf("Resolving symlink: component=%s target=%s parentDir=%s", dirPath, target, parentDir)

			var resolvedBase string
			if parentDir != "" {
				resolvedBase = pathpkg.Clean(pathpkg.Join(parentDir, target))
			} else {
				resolvedBase = pathpkg.Clean(target)
			}

			remoteLog.Printf("Resolved base after path.Clean: %s", resolvedBase)

			// Validate the resolved base doesn't escape the repository root
			if resolvedBase == "" || resolvedBase == "." || pathpkg.IsAbs(resolvedBase) || strings.HasPrefix(resolvedBase, "..") {
				remoteLog.Printf("Rejecting resolved base %q (escapes repository root)", resolvedBase)
				return "", fmt.Errorf("symlink target %q at %s resolves outside repository root: %s", target, dirPath, resolvedBase)
			}

			// Reconstruct the full path with the resolved symlink
			remaining := strings.Join(parts[i:], "/")
			resolvedPath := resolvedBase + "/" + remaining

			remoteLog.Printf("Resolved symlink in remote path: %s -> %s (full: %s -> %s)",
				dirPath, target, filePath, resolvedPath)

			return resolvedPath, nil
		}
	}

	remoteLog.Printf("No symlinks found after checking all %d directory components of %s", len(parts)-1, filePath)
	return "", fmt.Errorf("no symlinks found in path: %s", filePath)
}

// DownloadFileFromGitHub downloads a file from a GitHub repository using the GitHub API.
// This is the exported wrapper for downloadFileFromGitHub.
// Parameters:
// - owner: Repository owner (e.g., "github")
// - repo: Repository name (e.g., "gh-aw")
// - path: Path to the file within the repository (e.g., ".github/workflows/workflow.md")
// - ref: Git reference (branch, tag, or commit SHA)
// Returns the file content as bytes or an error if the file cannot be retrieved.
func DownloadFileFromGitHub(owner, repo, path, ref string) ([]byte, error) {
	return downloadFileFromGitHub(owner, repo, path, ref)
}

// ResolveRefToSHA resolves a git ref (branch, tag, or short SHA) to its full commit SHA.
// This is the exported wrapper for resolveRefToSHA.
// If the ref is already a 40-character hex SHA, it returns it as-is.
func ResolveRefToSHA(owner, repo, ref string) (string, error) {
	return resolveRefToSHA(owner, repo, ref)
}

func downloadFileFromGitHub(owner, repo, path, ref string) ([]byte, error) {
	return downloadFileFromGitHubWithDepth(owner, repo, path, ref, 0)
}

func downloadFileFromGitHubWithDepth(owner, repo, path, ref string, symlinkDepth int) ([]byte, error) {
	// Create REST client
	client, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client: %w", err)
	}

	// Define response struct for GitHub file content API
	var fileContent struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
		Name     string `json:"name"`
	}

	// Fetch file content from GitHub API
	err = client.Get(fmt.Sprintf("repos/%s/%s/contents/%s?ref=%s", owner, repo, path, ref), &fileContent)
	if err != nil {
		errStr := err.Error()

		// Check if this is an authentication error
		if gitutil.IsAuthError(errStr) {
			remoteLog.Printf("GitHub API authentication failed, attempting git fallback for %s/%s/%s@%s", owner, repo, path, ref)
			// Try fallback using git commands for public repositories
			content, gitErr := downloadFileViaGit(owner, repo, path, ref)
			if gitErr != nil {
				// If git fallback also fails, return both errors
				return nil, fmt.Errorf("failed to fetch file content via GitHub API (auth error) and git fallback: API error: %w, Git error: %v", err, gitErr)
			}
			return content, nil
		}

		// Check if this is a 404 — the path may traverse a symlink that the API doesn't follow
		if isNotFoundError(errStr) && symlinkDepth < constants.MaxSymlinkDepth {
			remoteLog.Printf("File not found at %s/%s/%s@%s, checking for symlinks in path (depth: %d)", owner, repo, path, ref, symlinkDepth)
			resolvedPath, resolveErr := resolveRemoteSymlinks(owner, repo, path, ref)
			if resolveErr == nil && resolvedPath != path {
				remoteLog.Printf("Retrying download with symlink-resolved path: %s -> %s", path, resolvedPath)
				return downloadFileFromGitHubWithDepth(owner, repo, resolvedPath, ref, symlinkDepth+1)
			}
		}

		return nil, fmt.Errorf("failed to fetch file content from %s/%s/%s@%s: %w", owner, repo, path, ref, err)
	}

	// Verify we have content
	if fileContent.Content == "" {
		return nil, fmt.Errorf("empty content returned from GitHub API for %s/%s/%s@%s", owner, repo, path, ref)
	}

	// Decode base64 content using native Go base64 package
	content, err := base64.StdEncoding.DecodeString(fileContent.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 content: %w", err)
	}

	return content, nil
}
