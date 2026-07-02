//go:build !js && !wasm

package parser

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var remoteLog = logger.New("parser:remote_fetch")

// publicAPIClient is a shared HTTP client used for unauthenticated GitHub API
// fallback calls. It carries a timeout to prevent indefinite hangs on slow or
// unresponsive hosts.
var publicAPIClient = &http.Client{Timeout: constants.DefaultHTTPClientTimeout}

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

func createRESTClientForHost(host string) (*api.RESTClient, error) {
	opts := api.ClientOptions{Timeout: constants.DefaultHTTPClientTimeout}
	if host != "" {
		opts.Host = host
	}
	return api.NewRESTClient(opts)
}

func buildContentsAPIPath(owner, repo, path, ref string) string {
	pathSegments := strings.Split(path, "/")
	for i := range pathSegments {
		pathSegments[i] = url.PathEscape(pathSegments[i])
	}
	return fmt.Sprintf(
		"repos/%s/%s/contents/%s?ref=%s",
		owner,
		repo,
		strings.Join(pathSegments, "/"),
		url.QueryEscape(ref),
	)
}

func fetchRemoteFileContent(client *api.RESTClient, owner, repo, path, ref string, fileContent any) error {
	return client.Get(buildContentsAPIPath(owner, repo, path, ref), fileContent)
}
