//go:build !js && !wasm

package parser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/errorutil"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
)

var remoteLog = logger.New("parser:remote_fetch")

// gitListCloneCache is a process-lifetime cache of shallow clones used by
// git-based directory listing fallbacks to avoid repeated clone operations for
// the same repository/ref tuple. Entries are not explicitly cleaned up because
// the CLI process is short-lived and temporary directories are OS-managed.
var gitListCloneCache = struct {
	mu   sync.Mutex
	dirs map[string]string
}{
	dirs: make(map[string]string),
}

func getOrCreateListRepoClone(owner, repo, ref, host string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", errors.New("git fallback requires a non-empty ref")
	}

	githubHost := GetGitHubHostForRepo(owner, repo)
	if host != "" {
		githubHost = stringutil.NormalizeGitHubHostURL(host)
	}
	repoURL := fmt.Sprintf("%s/%s/%s.git", githubHost, owner, repo)
	cacheKey := fmt.Sprintf("%s|%s|%s|%s", githubHost, owner, repo, ref)

	if cloneDir, found := func() (string, bool) {
		gitListCloneCache.mu.Lock()
		defer gitListCloneCache.mu.Unlock()
		if cloneDir, ok := gitListCloneCache.dirs[cacheKey]; ok {
			if stat, err := os.Stat(filepath.Join(cloneDir, ".git")); err == nil && stat.IsDir() {
				return cloneDir, true
			}
			delete(gitListCloneCache.dirs, cacheKey)
		}
		return "", false
	}(); found {
		return cloneDir, nil
	}

	tmpDir, err := os.MkdirTemp("", "gh-aw-list-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	cloneCmd := exec.Command("git", "clone", "--depth", "1", "--branch", ref, "--single-branch", "--filter=blob:none", "--no-checkout", repoURL, tmpDir)
	cloneOutput, err := cloneCmd.CombinedOutput()
	if err != nil {
		if cleanupErr := os.RemoveAll(tmpDir); cleanupErr != nil {
			remoteLog.Printf("Failed to clean up temp directory %q: %v", tmpDir, cleanupErr)
		}
		remoteLog.Printf("Failed to clone repository: %s", string(cloneOutput))
		return "", fmt.Errorf("failed to clone repository for %s/%s@%s: %w", owner, repo, ref, err)
	}

	existingDir, found := func() (string, bool) {
		gitListCloneCache.mu.Lock()
		defer gitListCloneCache.mu.Unlock()
		if existingDir, ok := gitListCloneCache.dirs[cacheKey]; ok {
			if stat, statErr := os.Stat(filepath.Join(existingDir, ".git")); statErr == nil && stat.IsDir() {
				return existingDir, true
			}
		}
		gitListCloneCache.dirs[cacheKey] = tmpDir
		return "", false
	}()
	if found {
		if cleanupErr := os.RemoveAll(tmpDir); cleanupErr != nil {
			remoteLog.Printf("Failed to clean up duplicate clone %q: %v", tmpDir, cleanupErr)
		}
		return existingDir, nil
	}
	return tmpDir, nil
}

// isUnderWorkflowsDirectory checks if a file path is a top-level workflow file (not in shared subdirectory)
func isUnderWorkflowsDirectory(filePath string) bool {
	// Normalize the path to use forward slashes
	normalizedPath := filepath.ToSlash(filePath)

	// Check if the path contains .github/workflows/
	if !strings.Contains(normalizedPath, constants.WorkflowsDirSlash) {
		return false
	}

	// Extract the part after .github/workflows/
	parts := strings.Split(normalizedPath, constants.WorkflowsDirSlash)
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
	return strings.Contains(normalizedPath, constants.AgentsDir) && strings.HasSuffix(strings.ToLower(normalizedPath), ".md")
}

// isRepositoryImport checks if an import spec is a repository-only import (no file path)
// Format: owner/repo@ref or owner/repo (downloads entire .github folder, no agent extraction)
func isRepositoryImport(importPath string) bool {
	// Remove section reference if present
	cleanPath := importPath
	if before, _, ok := strings.Cut(importPath, "#"); ok {
		cleanPath = before
	}

	// Remove ref if present to check the path structure
	pathWithoutRef := cleanPath
	if before, _, ok := strings.Cut(cleanPath, "@"); ok {
		pathWithoutRef = before
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

	if builtinPath, handled, err := resolveBuiltinIncludePath(filePath); handled {
		return builtinPath, err
	}

	if isWorkflowSpec(filePath) {
		remoteLog.Printf("Detected workflowspec format: %s", filePath)
		return downloadIncludeFromWorkflowSpec(filePath, cache)
	}

	remoteLog.Printf("Using local file resolution for: %s", filePath)
	resolveBase, securityBase, normalizedFilePath := computeIncludeResolveAndSecurityBases(filePath, baseDir)
	return resolveAndValidateLocalIncludePath(normalizedFilePath, resolveBase, securityBase)
}

func resolveBuiltinIncludePath(filePath string) (string, bool, error) {
	if !strings.HasPrefix(filePath, BuiltinPathPrefix) {
		return "", false, nil
	}
	if !BuiltinVirtualFileExists(filePath) {
		return "", true, fmt.Errorf("builtin file not found: %s", filePath)
	}
	remoteLog.Printf("Resolved builtin path: %s", filePath)
	return filePath, true, nil
}

func findGitHubFolder(baseDir string) string {
	githubFolder := baseDir
	for !strings.HasSuffix(githubFolder, ".github") {
		parent := filepath.Dir(githubFolder)
		if parent == githubFolder || parent == "." || parent == "/" {
			githubFolder = baseDir
			break
		}
		githubFolder = parent
	}
	return githubFolder
}

func computeIncludeResolveAndSecurityBases(filePath, baseDir string) (string, string, string) {
	githubFolder := findGitHubFolder(baseDir)
	resolveBase := baseDir
	securityBase := githubFolder
	normalizedFilePath := filePath
	if strings.HasSuffix(githubFolder, ".github") {
		repoRoot := filepath.Dir(githubFolder)
		filePathSlash := filepath.ToSlash(filePath)
		if strings.HasPrefix(filePathSlash, constants.GithubDir) {
			resolveBase = repoRoot
		} else if stripped, ok := strings.CutPrefix(filePathSlash, "/"); ok {
			if !strings.HasPrefix(stripped, constants.GithubDir) && !strings.HasPrefix(stripped, ".agents/") {
				return "", "", filePath
			}
			normalizedFilePath = filepath.FromSlash(stripped)
			resolveBase = repoRoot
			if strings.HasPrefix(stripped, ".agents/") {
				securityBase = filepath.Join(repoRoot, ".agents")
			} else {
				securityBase = githubFolder
			}
		}
	}
	return resolveBase, securityBase, normalizedFilePath
}

func resolveAndValidateLocalIncludePath(filePath, resolveBase, securityBase string) (string, error) {
	if stripped, ok := strings.CutPrefix(filepath.ToSlash(filePath), "/"); ok {
		if !strings.HasPrefix(stripped, constants.GithubDir) && !strings.HasPrefix(stripped, ".agents/") {
			remoteLog.Printf("Security: Path not within .github or .agents: %s", filePath)
			return "", fmt.Errorf("security: path %s must be within .github or .agents folder", filePath)
		}
	}
	fullPath := filepath.Join(resolveBase, filePath)
	normalizedSecurityBase := filepath.Clean(securityBase)
	normalizedFullPath := filepath.Clean(fullPath)
	relativePath, err := filepath.Rel(normalizedSecurityBase, normalizedFullPath)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) || filepath.IsAbs(relativePath) {
		allowedFolder := filepath.Base(normalizedSecurityBase)
		remoteLog.Printf("Security: Path escapes allowed folder: %s (resolves to: %s)", filePath, relativePath)
		return "", fmt.Errorf("security: path %s must be within %s folder (resolves to: %s)", filePath, allowedFolder, relativePath)
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		remoteLog.Printf("Local file not found: %s", fullPath)
		// Return a simple error that will be wrapped with source location by the caller
		return "", fmt.Errorf("file not found: %s", fullPath)
	}
	remoteLog.Printf("Resolved to local file: %s", fullPath)
	return fullPath, nil
}

// IsWorkflowSpec checks if a path looks like a workflowspec (owner/repo/path[@ref]).
func IsWorkflowSpec(path string) bool {
	// Remove section reference if present
	cleanPath := path
	if before, _, ok := strings.Cut(path, "#"); ok {
		cleanPath = before
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

	// Preserve legacy behavior expected by parser tests: URL-like paths are
	// currently treated as workflowspecs because downstream parsing supports
	// repository/path extraction from slash-delimited remote references.
	if strings.Contains(cleanPath, "://") {
		return true
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

	// Safe indexing: len(parts) >= 3 is guaranteed above.
	owner := parts[0]
	repo := parts[1]
	if owner == "" || repo == "" {
		return false
	}

	return true
}

func isWorkflowSpec(path string) bool {
	return IsWorkflowSpec(path)
}

// downloadIncludeFromWorkflowSpec downloads an include file from GitHub using workflowspec
// It first checks the cache, and only downloads if not cached
func downloadIncludeFromWorkflowSpec(spec string, cache *ImportCache) (string, error) {
	remoteLog.Printf("Downloading from workflowspec: %s", spec)
	owner, repo, filePath, ref, err := parseWorkflowSpecParts(spec)
	if err != nil {
		return "", err
	}
	remoteLog.Printf("Parsed workflowspec: owner=%s, repo=%s, file=%s, ref=%s", owner, repo, filePath, ref)

	sha := resolveWorkflowSpecSHAForCache(owner, repo, ref, cache)
	if cache != nil && sha != "" {
		if cachedPath, found := cache.Get(owner, repo, filePath, sha); found {
			remoteLog.Printf("Using cached import: %s/%s/%s@%s (SHA: %s)", owner, repo, filePath, ref, sha)
			return cachedPath, nil
		}
	}

	remoteLog.Printf("Fetching file from GitHub: %s/%s/%s@%s", owner, repo, filePath, ref)
	content, err := downloadFileFromGitHub(owner, repo, filePath, ref)
	if err != nil {
		return "", fmt.Errorf("failed to download include from %s: %w", spec, err)
	}
	remoteLog.Printf("Successfully downloaded file: size=%d bytes", len(content))

	if cache != nil && sha != "" {
		cachedPath, err := cache.Set(owner, repo, filePath, sha, content)
		if err != nil {
			remoteLog.Printf("Failed to cache import: %v", err)
		} else {
			remoteLog.Printf("Successfully cached download at: %s", cachedPath)
			return cachedPath, nil
		}
	}
	return writeDownloadedIncludeToTempFile(content)
}

func parseWorkflowSpecParts(spec string) (string, string, string, string, error) {
	cleanSpec := spec
	if before, _, ok := strings.Cut(spec, "#"); ok {
		cleanSpec = before
	}
	parts := strings.SplitN(cleanSpec, "@", 2)
	pathPart := parts[0]
	ref := "main"
	if len(parts) == 2 {
		ref = parts[1]
	} else {
		remoteLog.Print("No ref specified, defaulting to 'main'")
	}
	slashParts := strings.Split(pathPart, "/")
	if len(slashParts) < 3 {
		remoteLog.Printf("Invalid workflowspec format: %s", spec)
		return "", "", "", "", errors.New("invalid workflowspec: must be owner/repo/path[@ref]")
	}
	return slashParts[0], slashParts[1], strings.Join(slashParts[2:], "/"), ref, nil
}

func resolveWorkflowSpecSHAForCache(owner, repo, ref string, cache *ImportCache) string {
	if cache == nil {
		return ""
	}
	resolvedSHA, err := resolveRefToSHA(owner, repo, ref, "")
	if err != nil {
		remoteLog.Printf("Failed to resolve ref to SHA, will skip cache: %v", err)
		return ""
	}
	return resolvedSHA
}

func writeDownloadedIncludeToTempFile(content []byte) (string, error) {
	tempFile, err := os.CreateTemp("", "gh-aw-include-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	cleanupOnError := true
	fileClosed := false
	defer func() {
		if cleanupOnError {
			if !fileClosed {
				if closeErr := tempFile.Close(); closeErr != nil {
					remoteLog.Printf("Warning: failed to close temp file during deferred cleanup: %v", closeErr)
				}
			}
			if rmErr := os.Remove(tempFile.Name()); rmErr != nil && !os.IsNotExist(rmErr) {
				remoteLog.Printf("Warning: failed to remove temp file %s: %v", tempFile.Name(), rmErr)
			}
		}
	}()
	if _, err := tempFile.Write(content); err != nil {
		if closeErr := tempFile.Close(); closeErr != nil {
			remoteLog.Printf("Warning: failed to close temp file during cleanup: %v", closeErr)
		}
		fileClosed = true
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		fileClosed = true
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}
	cleanupOnError = false
	fileClosed = true
	return tempFile.Name(), nil
}

// resolveRefToSHAViaGit resolves a git ref to SHA using git ls-remote
// This is a fallback for when GitHub API authentication fails
func resolveRefToSHAViaGit(owner, repo, ref, host string) (string, error) {
	remoteLog.Printf("Attempting git ls-remote fallback for ref resolution: %s/%s@%s", owner, repo, ref)

	var githubHost string
	if host != "" {
		githubHost = "https://" + host
	} else {
		githubHost = GetGitHubHostForRepo(owner, repo)
	}
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
	if len(lines) == 0 || lines[0] == "" {
		return "", fmt.Errorf("no matching ref found for %s", ref)
	}

	// Extract SHA from the first line
	parts := strings.Fields(lines[0])
	if len(parts) < 1 {
		return "", errors.New("invalid git ls-remote output format")
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
func resolveRefToSHA(owner, repo, ref, host string) (string, error) {
	// If ref is already a full SHA (40 hex characters), return it as-is
	if len(ref) == 40 && gitutil.IsHexString(ref) {
		return ref, nil
	}

	// Use gh CLI to get the commit SHA for the ref
	// This works for branches, tags, and short SHAs
	// Using go-gh to properly handle enterprise GitHub instances via GH_HOST
	apiPath := buildCommitLookupAPIPath(owner, repo, ref)
	var args []string
	if host != "" {
		args = []string{"api", "--hostname", host, apiPath, "--jq", ".sha"}
	} else {
		args = []string{"api", apiPath, "--jq", ".sha"}
	}

	stdout, stderr, err := gh.Exec(args...)

	if err != nil {
		outputStr := stderr.String()
		if gitutil.IsAuthError(outputStr) {
			remoteLog.Printf("GitHub API authentication failed, attempting git ls-remote fallback for %s/%s@%s", owner, repo, ref)
			// Try fallback using git ls-remote for public repositories
			sha, gitErr := resolveRefToSHAViaGit(owner, repo, ref, host)
			if gitErr != nil {
				// If git fallback also fails, return both errors
				return "", fmt.Errorf("failed to resolve ref via GitHub API (auth error) and git ls-remote: API error: %w, Git error: %w", err, gitErr)
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

// buildCommitLookupAPIPath returns the GitHub commits API path for a ref,
// URL-escaping the ref segment so branch names containing slashes are valid.
func buildCommitLookupAPIPath(owner, repo, ref string) string {
	return fmt.Sprintf("/repos/%s/%s/commits/%s", owner, repo, url.PathEscape(ref))
}

// downloadFileViaGit downloads a file from a Git repository using git commands
// This is a fallback for when GitHub API authentication fails
func downloadFileViaGit(ctx context.Context, owner, repo, path, ref, host string) ([]byte, error) {
	remoteLog.Printf("Attempting git fallback for %s/%s/%s@%s", owner, repo, path, ref)

	// First, try via raw.githubusercontent.com — no auth required for public repos and
	// no dependency on git being installed.
	// Only attempt raw URL for github.com repos (not GHE) since raw.githubusercontent.com
	// only serves public GitHub content.
	if host == "" || host == "github.com" {
		content, rawErr := downloadFileViaRawURL(ctx, owner, repo, path, ref)
		if rawErr == nil {
			return content, nil
		}
		remoteLog.Printf("Raw URL download failed for %s/%s/%s@%s, trying git archive: %v", owner, repo, path, ref, rawErr)
	}

	// Use git archive to get the file content without cloning
	// This works for public repositories without authentication
	var githubHost string
	if host != "" {
		githubHost = "https://" + host
	} else {
		githubHost = GetGitHubHostForRepo(owner, repo)
	}
	repoURL := fmt.Sprintf("%s/%s/%s.git", githubHost, owner, repo)

	// git archive command: git archive --remote=<repo> <ref> <path>
	// #nosec G204 -- repoURL, ref, and path are from workflow import configuration authored by the
	// developer; exec.Command with separate args (not shell execution) prevents shell injection.
	cmd := exec.Command("git", "archive", "--remote="+repoURL, ref, path)
	archiveOutput, err := cmd.Output()
	if err != nil {
		// If git archive fails, try with git clone + git show as a fallback
		return downloadFileViaGitClone(owner, repo, path, ref, host)
	}

	// Extract the file from the tar archive using Go's archive/tar (cross-platform)
	content, err := fileutil.ExtractFileFromTar(archiveOutput, path)
	if err != nil {
		return nil, fmt.Errorf("failed to extract file from git archive: %w", err)
	}

	remoteLog.Printf("Successfully downloaded file via git archive: %s/%s/%s@%s", owner, repo, path, ref)
	return content, nil
}

// downloadFileViaRawURL fetches a file using the raw.githubusercontent.com URL.
// This requires no authentication for public repositories and no git installation.
func downloadFileViaRawURL(ctx context.Context, owner, repo, filePath, ref string) ([]byte, error) {
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, ref, filePath)
	remoteLog.Printf("Attempting raw URL download: %s", rawURL)

	// Use a client with a timeout to prevent indefinite hangs on slow/unresponsive hosts.
	rawClient := &http.Client{Timeout: constants.DefaultHTTPClientTimeout}

	// #nosec G107 -- rawURL is constructed from workflow import configuration authored by
	// the developer; the owner, repo, filePath, and ref are user-supplied workflow spec fields.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("raw URL request failed for %s: %w", rawURL, err)
	}
	resp, err := rawClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("raw URL request failed for %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("raw URL returned HTTP %d for %s", resp.StatusCode, rawURL)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read raw URL response body for %s: %w", rawURL, err)
	}

	remoteLog.Printf("Successfully downloaded file via raw URL: %s", rawURL)
	return content, nil
}

// downloadFileViaGitClone downloads a file by shallow cloning the repository
// This is used as a fallback when git archive doesn't work
func downloadFileViaGitClone(owner, repo, path, ref, host string) ([]byte, error) {
	remoteLog.Printf("Attempting git clone fallback for %s/%s/%s@%s", owner, repo, path, ref)

	// Create a temporary directory for the shallow clone
	tmpDir, err := os.MkdirTemp("", "gh-aw-git-clone-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	var githubHost string
	if host != "" {
		githubHost = "https://" + host
	} else {
		githubHost = GetGitHubHostForRepo(owner, repo)
	}
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
	if err := fileutil.ValidatePathWithinBase(tmpDir, filePath); err != nil {
		return nil, fmt.Errorf("refusing to read file outside clone directory: %w", err)
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file from cloned repository: %w", err)
	}

	remoteLog.Printf("Successfully downloaded file via git clone: %s/%s/%s@%s", owner, repo, path, ref)
	return content, nil
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
// The caller must provide a REST client (already authenticated for the correct host).
func resolveRemoteSymlinks(client *api.RESTClient, owner, repo, filePath, ref string) (string, error) {
	parts := strings.Split(filePath, "/")
	if len(parts) <= 1 {
		return "", fmt.Errorf("no directory components to resolve in path: %s", filePath)
	}

	if client == nil {
		return "", fmt.Errorf("no REST client available for symlink resolution of %s/%s/%s@%s", owner, repo, filePath, ref)
	}

	remoteLog.Printf("Attempting symlink resolution for %s/%s/%s@%s (%d path components)", owner, repo, filePath, ref, len(parts))

	for i := 1; i < len(parts); i++ {
		dirPath := strings.Join(parts[:i], "/")
		resolvedPath, found, err := resolveRemoteSymlinkComponent(client, owner, repo, filePath, ref, parts, i, dirPath)
		if err != nil {
			return "", err
		}
		if found {
			return resolvedPath, nil
		}
	}

	remoteLog.Printf("No symlinks found after checking all %d directory components of %s", len(parts)-1, filePath)
	return "", fmt.Errorf("no symlinks found in path: %s", filePath)
}

func resolveRemoteSymlinkComponent(
	client *api.RESTClient,
	owner, repo, filePath, ref string,
	parts []string,
	index int,
	dirPath string,
) (string, bool, error) {
	target, isSymlink, err := checkRemoteSymlink(client, owner, repo, dirPath, ref)
	if err != nil {
		if errorutil.IsNotFoundError(err) {
			remoteLog.Printf("Path component %s returned 404, skipping", dirPath)
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to check path component %s for symlinks: %w", dirPath, err)
	}
	if !isSymlink {
		return "", false, nil
	}
	parentDir := ""
	if index > 1 {
		parentDir = strings.Join(parts[:index-1], "/")
	}
	resolvedBase, err := resolveAndValidateRemoteSymlinkBase(parentDir, target, dirPath)
	if err != nil {
		return "", false, err
	}
	remaining := strings.Join(parts[index:], "/")
	resolvedPath := resolvedBase + "/" + remaining
	remoteLog.Printf("Resolved symlink in remote path: %s -> %s (full: %s -> %s)", dirPath, target, filePath, resolvedPath)
	return resolvedPath, true, nil
}

func resolveAndValidateRemoteSymlinkBase(parentDir, target, dirPath string) (string, error) {
	remoteLog.Printf("Resolving symlink: component=%s target=%s parentDir=%s", dirPath, target, parentDir)
	resolvedBase := pathpkg.Clean(target)
	if parentDir != "" {
		resolvedBase = pathpkg.Clean(pathpkg.Join(parentDir, target))
	}
	remoteLog.Printf("Resolved base after path.Clean: %s", resolvedBase)
	if resolvedBase == "" || resolvedBase == "." || pathpkg.IsAbs(resolvedBase) || strings.HasPrefix(resolvedBase, "..") {
		remoteLog.Printf("Rejecting resolved base %q (escapes repository root)", resolvedBase)
		return "", fmt.Errorf("symlink target %q at %s resolves outside repository root: %s", target, dirPath, resolvedBase)
	}
	return resolvedBase, nil
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
	return downloadFileFromGitHubWithDepth(owner, repo, path, ref, 0, "")
}

// DownloadFileFromGitHubForHost downloads a file from a GitHub repository using the GitHub API,
// targeting a specific GitHub host. Use this when the target repository is on a different host
// than the one configured via GH_HOST (e.g., fetching from github.com while GH_HOST is a GHE instance).
// host is the hostname without scheme (e.g., "github.com", "myorg.ghe.com").
// An empty host uses the default configured host (GH_HOST or github.com).
func DownloadFileFromGitHubForHost(owner, repo, path, ref, host string) ([]byte, error) {
	return downloadFileFromGitHubWithDepth(owner, repo, path, ref, 0, host)
}

// ResolveRefToSHAForHost resolves a git ref to its full commit SHA on a specific GitHub host.
// Use this when the target repository is on a different host than the one configured via GH_HOST.
// host is the hostname without scheme (e.g., "github.com", "myorg.ghe.com").
// An empty host uses the default configured host (GH_HOST or github.com).
func ResolveRefToSHAForHost(owner, repo, ref, host string) (string, error) {
	return resolveRefToSHA(owner, repo, ref, host)
}

func downloadFileFromGitHub(owner, repo, path, ref string) ([]byte, error) {
	return downloadFileFromGitHubWithDepth(owner, repo, path, ref, 0, "")
}

func downloadFileFromGitHubWithDepth(owner, repo, path, ref string, symlinkDepth int, host string) ([]byte, error) {
	client, err := createRESTClientForHost(host)
	if err != nil {
		if gitutil.IsAuthError(err.Error()) {
			remoteLog.Printf("REST client creation failed due to auth error, attempting git fallback for %s/%s/%s@%s: %v", owner, repo, path, ref, err)
			content, gitErr := downloadFileViaGit(context.Background(), owner, repo, path, ref, host)
			if gitErr != nil {
				remoteLog.Printf("Git fallback also failed for %s/%s/%s@%s: %v", owner, repo, path, ref, gitErr)
				return nil, fmt.Errorf("failed to fetch file content: %w", err)
			}
			return content, nil
		}
		return nil, fmt.Errorf("failed to create REST client: %w", err)
	}

	var fileContent struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
		Name     string `json:"name"`
	}

	err = fetchRemoteFileContent(client, owner, repo, path, ref, &fileContent)
	if err != nil {
		if gitutil.IsAuthError(err.Error()) {
			remoteLog.Printf("GitHub API authentication failed, attempting git fallback for %s/%s/%s@%s", owner, repo, path, ref)
			content, gitErr := downloadFileViaGit(context.Background(), owner, repo, path, ref, host)
			if gitErr != nil {
				return nil, fmt.Errorf("failed to fetch file content via GitHub API (auth error) and git fallback: API error: %w, Git error: %w", err, gitErr)
			}
			return content, nil
		}

		if errorutil.IsNotFoundError(err) && symlinkDepth < constants.MaxSymlinkDepth {
			if content, handled, resolveErr := retryDownloadViaResolvedSymlink(client, owner, repo, path, ref, symlinkDepth, host); handled {
				return content, resolveErr
			}
		}

		return nil, fmt.Errorf("failed to fetch file content from %s/%s/%s@%s: %w", owner, repo, path, ref, err)
	}

	if fileContent.Content == "" {
		return nil, fmt.Errorf("empty content returned from GitHub API for %s/%s/%s@%s", owner, repo, path, ref)
	}

	content, err := base64.StdEncoding.DecodeString(fileContent.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 content: %w", err)
	}

	return content, nil
}

func createRESTClientForHost(host string) (*api.RESTClient, error) {
	if host != "" {
		return api.NewRESTClient(api.ClientOptions{Host: host})
	}
	return api.DefaultRESTClient()
}

func fetchRemoteFileContent(client *api.RESTClient, owner, repo, path, ref string, fileContent any) error {
	return client.Get(fmt.Sprintf("repos/%s/%s/contents/%s?ref=%s", owner, repo, path, ref), fileContent)
}

func retryDownloadViaResolvedSymlink(
	client *api.RESTClient,
	owner, repo, path, ref string,
	symlinkDepth int,
	host string,
) ([]byte, bool, error) {
	remoteLog.Printf("File not found at %s/%s/%s@%s, checking for symlinks in path (depth: %d)", owner, repo, path, ref, symlinkDepth)
	resolvedPath, resolveErr := resolveRemoteSymlinks(client, owner, repo, path, ref)
	if resolveErr == nil && resolvedPath != path {
		remoteLog.Printf("Retrying download with symlink-resolved path: %s -> %s", path, resolvedPath)
		content, err := downloadFileFromGitHubWithDepth(owner, repo, resolvedPath, ref, symlinkDepth+1, host)
		return content, true, err
	}
	return nil, false, nil
}

// ListWorkflowFiles lists workflow files from a remote GitHub repository
// Returns a list of .md files in the specified directory (excluding subdirectories)
func ListWorkflowFiles(owner, repo, ref, workflowPath string) ([]string, error) {
	return listWorkflowFilesForHost(owner, repo, ref, workflowPath, "")
}

// ListWorkflowFilesForHost lists workflow files from a remote GitHub repository on an explicit host.
// Use this when the target repository is on a different host than the one configured via GH_HOST.
func ListWorkflowFilesForHost(owner, repo, ref, workflowPath, host string) ([]string, error) {
	return listWorkflowFilesForHost(owner, repo, ref, workflowPath, host)
}

func listWorkflowFilesForHost(owner, repo, ref, workflowPath, host string) ([]string, error) {
	remoteLog.Printf("Listing workflow files for %s/%s@%s (path: %s)", owner, repo, ref, workflowPath)

	// Create REST client
	var (
		client *api.RESTClient
		err    error
	)
	if host != "" {
		client, err = api.NewRESTClient(api.ClientOptions{Host: host})
	} else {
		client, err = api.DefaultRESTClient()
	}
	if err != nil {
		remoteLog.Printf("Failed to create REST client, attempting git fallback: %v", err)
		return listWorkflowFilesViaGitForHost(owner, repo, ref, workflowPath, host)
	}

	// Define response struct for GitHub contents API (array of file objects)
	var contents []struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Type string `json:"type"`
	}

	// Fetch directory contents from GitHub API
	endpoint := fmt.Sprintf("repos/%s/%s/contents/%s?ref=%s", owner, repo, workflowPath, ref)
	err = client.Get(endpoint, &contents)
	if err != nil {
		errStr := err.Error()

		// Check if this is an authentication error
		if gitutil.IsAuthError(errStr) {
			remoteLog.Printf("GitHub API authentication failed, attempting git fallback for %s/%s@%s", owner, repo, ref)
			// Try fallback using git commands for public repositories
			files, gitErr := listWorkflowFilesViaGitForHost(owner, repo, ref, workflowPath, host)
			if gitErr != nil {
				// If git fallback also fails, return both errors
				return nil, fmt.Errorf("failed to list workflow files via GitHub API (auth error) and git fallback: API error: %w, Git error: %w", err, gitErr)
			}
			return files, nil
		}

		return nil, fmt.Errorf("failed to list workflow files from %s/%s@%s (path: %s): %w", owner, repo, ref, workflowPath, err)
	}

	// Filter to only .md files (not in subdirectories)
	var workflowFiles []string
	for _, item := range contents {
		if item.Type == "file" && strings.HasSuffix(strings.ToLower(item.Name), ".md") {
			workflowFiles = append(workflowFiles, item.Path)
		}
	}

	remoteLog.Printf("Found %d workflow files in %s/%s@%s (path: %s)", len(workflowFiles), owner, repo, ref, workflowPath)
	return workflowFiles, nil
}

// ListDirAllFilesForHost lists all files (any extension) that are direct children of
// the given directory in a remote GitHub repository. Subdirectories and their contents
// are not included. This is used for skill file discovery.
func ListDirAllFilesForHost(owner, repo, ref, dirPath, host string) ([]string, error) {
	return listDirAllFilesForHost(owner, repo, ref, dirPath, host)
}

func listDirAllFilesForHost(owner, repo, ref, dirPath, host string) ([]string, error) {
	remoteLog.Printf("Listing all files in dir for %s/%s@%s (path: %s)", owner, repo, ref, dirPath)

	var (
		client *api.RESTClient
		err    error
	)
	if host != "" {
		client, err = api.NewRESTClient(api.ClientOptions{Host: host})
	} else {
		client, err = api.DefaultRESTClient()
	}
	if err != nil {
		remoteLog.Printf("Failed to create REST client, attempting git fallback: %v", err)
		return listDirAllFilesViaGitForHost(owner, repo, ref, dirPath, host)
	}

	var contents []struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Type string `json:"type"`
	}

	endpoint := fmt.Sprintf("repos/%s/%s/contents/%s?ref=%s", owner, repo, dirPath, ref)
	err = client.Get(endpoint, &contents)
	if err != nil {
		errStr := err.Error()
		if gitutil.IsAuthError(errStr) {
			remoteLog.Printf("GitHub API auth failed, attempting git fallback for %s/%s@%s", owner, repo, ref)
			files, gitErr := listDirAllFilesViaGitForHost(owner, repo, ref, dirPath, host)
			if gitErr != nil {
				return nil, fmt.Errorf("failed to list dir files via API (auth error) and git fallback: API error: %w, Git error: %w", err, gitErr)
			}
			return files, nil
		}
		return nil, fmt.Errorf("failed to list dir files from %s/%s@%s (path: %s): %w", owner, repo, ref, dirPath, err)
	}

	var files []string
	for _, item := range contents {
		if item.Type == "file" {
			files = append(files, item.Path)
		}
	}

	remoteLog.Printf("Found %d files in dir %s/%s@%s (path: %s)", len(files), owner, repo, ref, dirPath)
	return files, nil
}

func listDirAllFilesViaGitForHost(owner, repo, ref, dirPath, host string) ([]string, error) {
	remoteLog.Printf("Git fallback for listing all dir files: %s/%s@%s (path: %s)", owner, repo, ref, dirPath)

	tmpDir, err := getOrCreateListRepoClone(owner, repo, ref, host)
	if err != nil {
		return nil, err
	}

	lsTreeCmd := exec.Command("git", "-C", tmpDir, "ls-tree", "-r", "--name-only", "HEAD", dirPath+"/")
	lsTreeOutput, err := lsTreeCmd.CombinedOutput()
	if err != nil {
		remoteLog.Printf("Failed to list dir files: %s", string(lsTreeOutput))
		return nil, fmt.Errorf("failed to list dir files: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(lsTreeOutput)), "\n")
	var files []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Only include direct children (no additional path separator after dirPath/)
		afterDirPath := strings.TrimPrefix(line, dirPath+"/")
		if !strings.Contains(afterDirPath, "/") && afterDirPath != "" {
			files = append(files, line)
		}
	}

	remoteLog.Printf("Found %d files in dir via git for %s/%s@%s (path: %s)", len(files), owner, repo, ref, dirPath)
	return files, nil
}

// ListDirAllFilesRecursivelyForHost lists all files (any extension) that are under the
// given directory in a remote GitHub repository, including files in subdirectories at any
// depth. This is used for copying entire skill folders.
func ListDirAllFilesRecursivelyForHost(owner, repo, ref, dirPath, host string) ([]string, error) {
	return listDirAllFilesRecursivelyForHost(owner, repo, ref, dirPath, host)
}

func listDirAllFilesRecursivelyForHost(owner, repo, ref, dirPath, host string) ([]string, error) {
	remoteLog.Printf("Listing all files recursively in dir for %s/%s@%s (path: %s)", owner, repo, ref, dirPath)

	var (
		client *api.RESTClient
		err    error
	)
	if host != "" {
		client, err = api.NewRESTClient(api.ClientOptions{Host: host})
	} else {
		client, err = api.DefaultRESTClient()
	}
	if err != nil {
		remoteLog.Printf("Failed to create REST client, attempting git fallback: %v", err)
		return listDirAllFilesRecursivelyViaGitForHost(owner, repo, ref, dirPath, host)
	}

	files, err := listContentsRecursively(client, owner, repo, ref, dirPath)
	if err != nil {
		errStr := err.Error()
		if gitutil.IsAuthError(errStr) {
			remoteLog.Printf("GitHub API auth failed, attempting git fallback for %s/%s@%s", owner, repo, ref)
			gitFiles, gitErr := listDirAllFilesRecursivelyViaGitForHost(owner, repo, ref, dirPath, host)
			if gitErr != nil {
				return nil, fmt.Errorf("failed to list dir files recursively via API (auth error) and git fallback: API error: %w, Git error: %w", err, gitErr)
			}
			return gitFiles, nil
		}
		return nil, err
	}

	remoteLog.Printf("Found %d files recursively in dir %s/%s@%s (path: %s)", len(files), owner, repo, ref, dirPath)
	return files, nil
}

// listContentsRecursively uses the GitHub Contents API to recursively enumerate all
// files under dirPath. Each subdirectory triggers an additional API call.
func listContentsRecursively(client *api.RESTClient, owner, repo, ref, dirPath string) ([]string, error) {
	const maxSkillDirRecursionDepth = 10
	return listContentsRecursivelyWithDepth(client, owner, repo, ref, dirPath, 0, maxSkillDirRecursionDepth)
}

func listContentsRecursivelyWithDepth(client *api.RESTClient, owner, repo, ref, dirPath string, depth, maxDepth int) ([]string, error) {
	if depth > maxDepth {
		return nil, fmt.Errorf("maximum skill directory recursion depth exceeded at %q (max depth: %d)", dirPath, maxDepth)
	}

	var contents []struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Type string `json:"type"`
	}

	endpoint := fmt.Sprintf("repos/%s/%s/contents/%s?ref=%s", owner, repo, dirPath, ref)
	if err := client.Get(endpoint, &contents); err != nil {
		return nil, fmt.Errorf("failed to list dir files from %s/%s (path: %s): %w", owner, repo, dirPath, err)
	}

	var files []string
	for _, item := range contents {
		switch item.Type {
		case "file":
			files = append(files, item.Path)
		case "dir":
			subFiles, err := listContentsRecursivelyWithDepth(client, owner, repo, ref, item.Path, depth+1, maxDepth)
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
		}
	}
	return files, nil
}

func listDirAllFilesRecursivelyViaGitForHost(owner, repo, ref, dirPath, host string) ([]string, error) {
	remoteLog.Printf("Git fallback for listing all dir files recursively: %s/%s@%s (path: %s)", owner, repo, ref, dirPath)

	tmpDir, err := getOrCreateListRepoClone(owner, repo, ref, host)
	if err != nil {
		return nil, err
	}

	// Normalise dirPath so it never has a trailing slash before we append one.
	cleanDirPath := strings.TrimRight(dirPath, "/")
	lsTreeCmd := exec.Command("git", "-C", tmpDir, "ls-tree", "-r", "--name-only", "HEAD", cleanDirPath+"/")
	lsTreeOutput, err := lsTreeCmd.CombinedOutput()
	if err != nil {
		remoteLog.Printf("Failed to list dir files recursively: %s", string(lsTreeOutput))
		return nil, fmt.Errorf("failed to list dir files recursively: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(lsTreeOutput)), "\n")
	var files []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// git ls-tree already scopes results to dirPrefix; include every non-empty line.
		files = append(files, line)
	}

	remoteLog.Printf("Found %d files recursively in dir via git for %s/%s@%s (path: %s)", len(files), owner, repo, ref, dirPath)
	return files, nil
}

// ListDirSubdirsForHost lists subdirectory paths that are direct children of the given
// directory in a remote GitHub repository. This is used for auto-discovering skill dirs.
func ListDirSubdirsForHost(owner, repo, ref, dirPath, host string) ([]string, error) {
	return listDirSubdirsForHost(owner, repo, ref, dirPath, host)
}

func listDirSubdirsForHost(owner, repo, ref, dirPath, host string) ([]string, error) {
	remoteLog.Printf("Listing subdirs in %s/%s@%s (path: %s)", owner, repo, ref, dirPath)

	var (
		client *api.RESTClient
		err    error
	)
	if host != "" {
		client, err = api.NewRESTClient(api.ClientOptions{Host: host})
	} else {
		client, err = api.DefaultRESTClient()
	}
	if err != nil {
		remoteLog.Printf("Failed to create REST client, attempting git fallback: %v", err)
		return listDirSubdirsViaGitForHost(owner, repo, ref, dirPath, host)
	}

	var contents []struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Type string `json:"type"`
	}

	endpoint := fmt.Sprintf("repos/%s/%s/contents/%s?ref=%s", owner, repo, dirPath, ref)
	err = client.Get(endpoint, &contents)
	if err != nil {
		errStr := err.Error()
		if gitutil.IsAuthError(errStr) {
			remoteLog.Printf("GitHub API auth failed, attempting git fallback for %s/%s@%s", owner, repo, ref)
			dirs, gitErr := listDirSubdirsViaGitForHost(owner, repo, ref, dirPath, host)
			if gitErr != nil {
				return nil, fmt.Errorf("failed to list subdirs via API (auth error) and git fallback: API error: %w, Git error: %w", err, gitErr)
			}
			return dirs, nil
		}
		return nil, fmt.Errorf("failed to list subdirs from %s/%s@%s (path: %s): %w", owner, repo, ref, dirPath, err)
	}

	var dirs []string
	for _, item := range contents {
		if item.Type == "dir" {
			dirs = append(dirs, item.Path)
		}
	}

	remoteLog.Printf("Found %d subdirs in %s/%s@%s (path: %s)", len(dirs), owner, repo, ref, dirPath)
	return dirs, nil
}

func listDirSubdirsViaGitForHost(owner, repo, ref, dirPath, host string) ([]string, error) {
	remoteLog.Printf("Git fallback for listing subdirs: %s/%s@%s (path: %s)", owner, repo, ref, dirPath)

	tmpDir, err := getOrCreateListRepoClone(owner, repo, ref, host)
	if err != nil {
		return nil, err
	}

	// Use ls-tree -d to list only direct subdirectory entries.
	lsTreeDirsCmd := exec.Command("git", "-C", tmpDir, "ls-tree", "--name-only", "-d", "HEAD", dirPath+"/")
	lsTreeDirsOutput, err := lsTreeDirsCmd.CombinedOutput()
	if err != nil {
		remoteLog.Printf("Failed to list tree subdirs: %s", string(lsTreeDirsOutput))
		return nil, fmt.Errorf("failed to list subdirs: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(lsTreeDirsOutput)), "\n")
	var dirs []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		afterDirPath := strings.TrimPrefix(line, dirPath+"/")
		if !strings.Contains(afterDirPath, "/") && afterDirPath != "" {
			dirs = append(dirs, line)
		}
	}

	remoteLog.Printf("Found %d subdirs via git for %s/%s@%s (path: %s)", len(dirs), owner, repo, ref, dirPath)
	return dirs, nil
}

func listWorkflowFilesViaGitForHost(owner, repo, ref, workflowPath, host string) ([]string, error) {
	remoteLog.Printf("Attempting git fallback for listing workflow files: %s/%s@%s (path: %s)", owner, repo, ref, workflowPath)

	githubHost := GetGitHubHostForRepo(owner, repo)
	if host != "" {
		githubHost = stringutil.NormalizeGitHubHostURL(host)
	}
	repoURL := fmt.Sprintf("%s/%s/%s.git", githubHost, owner, repo)

	// Create a temporary directory for minimal clone
	tmpDir, err := os.MkdirTemp("", "gh-aw-list-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Do a minimal clone using filter=blob:none for faster cloning (metadata only, no blobs)
	// Use --depth=1 for shallow clone and --no-checkout to skip checkout initially
	cloneCmd := exec.Command("git", "clone", "--depth", "1", "--branch", ref, "--single-branch", "--filter=blob:none", "--no-checkout", repoURL, tmpDir)
	cloneOutput, err := cloneCmd.CombinedOutput()
	if err != nil {
		remoteLog.Printf("Failed to clone repository: %s", string(cloneOutput))
		return nil, fmt.Errorf("failed to clone repository for %s/%s@%s: %w", owner, repo, ref, err)
	}

	// Use git ls-tree to list files in the specified workflows directory
	lsTreeCmd := exec.Command("git", "-C", tmpDir, "ls-tree", "-r", "--name-only", "HEAD", workflowPath+"/")
	lsTreeOutput, err := lsTreeCmd.CombinedOutput()
	if err != nil {
		remoteLog.Printf("Failed to list files: %s", string(lsTreeOutput))
		return nil, fmt.Errorf("failed to list workflow files: %w", err)
	}

	// Parse output and filter for .md files (not in subdirectories)
	lines := strings.Split(strings.TrimSpace(string(lsTreeOutput)), "\n")
	var workflowFiles []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Only include .md files directly in the workflow path (not in subdirectories)
		if strings.HasSuffix(strings.ToLower(line), ".md") {
			// Check if it's a top-level file (no additional slashes after workflowPath/)
			afterWorkflowPath := strings.TrimPrefix(line, workflowPath+"/")
			if !strings.Contains(afterWorkflowPath, "/") {
				workflowFiles = append(workflowFiles, line)
			}
		}
	}

	remoteLog.Printf("Found %d workflow files via git for %s/%s@%s (path: %s)", len(workflowFiles), owner, repo, ref, workflowPath)
	return workflowFiles, nil
}
