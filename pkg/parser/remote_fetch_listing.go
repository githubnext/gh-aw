//go:build !js && !wasm

package parser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/stringutil"
)

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

	client, err := createRESTClientForHost(host)
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
	endpoint := buildContentsAPIPath(owner, repo, workflowPath, ref)
	err = client.Get(endpoint, &contents)
	if err != nil {
		errStr := err.Error()

		// Check if this is an authentication error
		if gitutil.IsAuthError(errStr) {
			remoteLog.Printf("GitHub API authentication failed, attempting git fallback for %s/%s@%s", owner, repo, ref)
			// Try fallback using git commands for public repositories
			files, gitErr := listWorkflowFilesViaGitForHost(owner, repo, ref, workflowPath, host)
			if gitErr != nil {
				if host == "" || host == "github.com" {
					remoteLog.Printf("Git fallback also failed, attempting unauthenticated API for %s/%s@%s", owner, repo, ref)
					return listWorkflowFilesViaPublicAPI(context.Background(), owner, repo, ref, workflowPath)
				}
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

	client, err := createRESTClientForHost(host)
	if err != nil {
		remoteLog.Printf("Failed to create REST client, attempting git fallback: %v", err)
		return listDirAllFilesViaGitForHost(owner, repo, ref, dirPath, host)
	}

	var contents []struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Type string `json:"type"`
	}

	endpoint := buildContentsAPIPath(owner, repo, dirPath, ref)
	err = client.Get(endpoint, &contents)
	if err != nil {
		errStr := err.Error()
		if gitutil.IsAuthError(errStr) {
			remoteLog.Printf("GitHub API auth failed, attempting git fallback for %s/%s@%s", owner, repo, ref)
			files, gitErr := listDirAllFilesViaGitForHost(owner, repo, ref, dirPath, host)
			if gitErr != nil {
				if host == "" || host == "github.com" {
					remoteLog.Printf("Git fallback also failed, attempting unauthenticated API for %s/%s@%s", owner, repo, ref)
					return listDirAllFilesViaPublicAPI(context.Background(), owner, repo, ref, dirPath)
				}
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

// listDirAllFilesViaPublicAPI lists files in a directory using an unauthenticated
// call to the public GitHub API. Used as a last-resort fallback when both
// authenticated API and git clone fail.
func listDirAllFilesViaPublicAPI(ctx context.Context, owner, repo, ref, dirPath string) ([]string, error) {
	remoteLog.Printf("Attempting unauthenticated public API for listing dir files: %s/%s@%s (path: %s)", owner, repo, ref, dirPath)
	body, err := fetchPublicGitHubContentsAPI(ctx, owner, repo, dirPath, ref)
	if err != nil {
		return nil, fmt.Errorf("unauthenticated public API also failed for %s/%s@%s (path: %s): %w", owner, repo, ref, dirPath, err)
	}

	var contents []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &contents); err != nil {
		return nil, fmt.Errorf("failed to parse public API response: %w", err)
	}

	var files []string
	for _, item := range contents {
		if item.Type == "file" {
			files = append(files, item.Path)
		}
	}
	remoteLog.Printf("Found %d files via public API for %s/%s@%s (path: %s)", len(files), owner, repo, ref, dirPath)
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

	client, err := createRESTClientForHost(host)
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
				// No public API fallback for recursive listing — would require
				// multiple unauthenticated calls and is unlikely to stay within
				// the 60 req/hour rate limit. Surface both errors.
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

	endpoint := buildContentsAPIPath(owner, repo, dirPath, ref)
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

// fetchPublicGitHubContentsAPI makes an unauthenticated GET request to the
// GitHub public REST API contents endpoint. This is used as a last-resort
// fallback when the current token (e.g. an enterprise SAML-enforced token)
// cannot access cross-organization public repositories and git clone also
// fails. Unauthenticated requests are subject to a lower rate limit
// (60 req/hour) but are sufficient for the handful of calls during update.
func fetchPublicGitHubContentsAPI(ctx context.Context, owner, repo, path, ref string) ([]byte, error) {
	// Encode each path segment independently so that '/' separators are
	// preserved — url.PathEscape would turn them into '%2F', breaking nested
	// paths like '.github/workflows/shared/foo.md'.
	segments := strings.Split(path, "/")
	encodedSegments := make([]string, len(segments))
	for i, s := range segments {
		encodedSegments[i] = url.PathEscape(s)
	}
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		owner, repo, strings.Join(encodedSegments, "/"), url.QueryEscape(ref))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := publicAPIClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

// ListDirSubdirsForHost lists subdirectory paths that are direct children of the given
// directory in a remote GitHub repository. This is used for auto-discovering skill dirs.
func ListDirSubdirsForHost(owner, repo, ref, dirPath, host string) ([]string, error) {
	return listDirSubdirsForHost(owner, repo, ref, dirPath, host)
}

func listDirSubdirsForHost(owner, repo, ref, dirPath, host string) ([]string, error) {
	remoteLog.Printf("Listing subdirs in %s/%s@%s (path: %s)", owner, repo, ref, dirPath)

	client, err := createRESTClientForHost(host)
	if err != nil {
		remoteLog.Printf("Failed to create REST client, attempting git fallback: %v", err)
		return listDirSubdirsViaGitForHost(owner, repo, ref, dirPath, host)
	}

	var contents []struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Type string `json:"type"`
	}

	endpoint := buildContentsAPIPath(owner, repo, dirPath, ref)
	err = client.Get(endpoint, &contents)
	if err != nil {
		errStr := err.Error()
		if gitutil.IsAuthError(errStr) {
			remoteLog.Printf("GitHub API auth failed, attempting git fallback for %s/%s@%s", owner, repo, ref)
			dirs, gitErr := listDirSubdirsViaGitForHost(owner, repo, ref, dirPath, host)
			if gitErr != nil {
				if host == "" || host == "github.com" {
					remoteLog.Printf("Git fallback also failed, attempting unauthenticated API for %s/%s@%s", owner, repo, ref)
					return listDirSubdirsViaPublicAPI(context.Background(), owner, repo, ref, dirPath)
				}
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

// listDirSubdirsViaPublicAPI lists subdirectories using an unauthenticated call
// to the public GitHub API. Used as a last-resort fallback when both
// authenticated API and git clone fail (e.g. enterprise SAML tokens).
func listDirSubdirsViaPublicAPI(ctx context.Context, owner, repo, ref, dirPath string) ([]string, error) {
	remoteLog.Printf("Attempting unauthenticated public API for listing subdirs: %s/%s@%s (path: %s)", owner, repo, ref, dirPath)
	body, err := fetchPublicGitHubContentsAPI(ctx, owner, repo, dirPath, ref)
	if err != nil {
		return nil, fmt.Errorf("unauthenticated public API also failed for %s/%s@%s (path: %s): %w", owner, repo, ref, dirPath, err)
	}

	var contents []struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &contents); err != nil {
		return nil, fmt.Errorf("failed to parse public API response: %w", err)
	}

	var dirs []string
	for _, item := range contents {
		if item.Type == "dir" {
			dirs = append(dirs, item.Path)
		}
	}
	remoteLog.Printf("Found %d subdirs via public API for %s/%s@%s (path: %s)", len(dirs), owner, repo, ref, dirPath)
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

// listWorkflowFilesViaPublicAPI lists workflow .md files using an unauthenticated
// call to the public GitHub API. Used as a last-resort fallback when both
// authenticated API and git clone fail.
func listWorkflowFilesViaPublicAPI(ctx context.Context, owner, repo, ref, workflowPath string) ([]string, error) {
	remoteLog.Printf("Attempting unauthenticated public API for listing workflow files: %s/%s@%s (path: %s)", owner, repo, ref, workflowPath)
	body, err := fetchPublicGitHubContentsAPI(ctx, owner, repo, workflowPath, ref)
	if err != nil {
		return nil, fmt.Errorf("unauthenticated public API also failed for %s/%s@%s (path: %s): %w", owner, repo, ref, workflowPath, err)
	}

	var contents []struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &contents); err != nil {
		return nil, fmt.Errorf("failed to parse public API response: %w", err)
	}

	var workflowFiles []string
	for _, item := range contents {
		if item.Type == "file" && strings.HasSuffix(strings.ToLower(item.Name), ".md") {
			workflowFiles = append(workflowFiles, item.Path)
		}
	}
	remoteLog.Printf("Found %d workflow files via public API for %s/%s@%s (path: %s)", len(workflowFiles), owner, repo, ref, workflowPath)
	return workflowFiles, nil
}
