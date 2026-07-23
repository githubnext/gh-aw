package gitutil

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/logger"
)

var gitutilLog = logger.New("gitutil:gitutil")
var ErrNotGitRepository = errors.New("not in a git repository")

var fullSHARegex = regexp.MustCompile(`^[0-9a-f]{40}$`)
var gitObjectIDRegex = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

// IsRateLimitError checks if an error message indicates a GitHub API rate limit error.
// This is used to detect transient failures caused by hitting the GitHub API rate limit
// (HTTP 403 "API rate limit exceeded" or HTTP 429 responses).
func IsRateLimitError(errMsg string) bool {
	lowerMsg := strings.ToLower(errMsg)
	return strings.Contains(lowerMsg, "api rate limit exceeded") ||
		strings.Contains(lowerMsg, "rate limit exceeded") ||
		strings.Contains(lowerMsg, "secondary rate limit")
}

// IsAuthError checks if an error message indicates an authentication issue.
// This is used to detect when GitHub API calls fail due to missing or invalid credentials.
func IsAuthError(errMsg string) bool {
	gitutilLog.Printf("Checking if error is auth-related: %s", errMsg)
	lowerMsg := strings.ToLower(errMsg)
	isAuth := strings.Contains(lowerMsg, "gh_token") ||
		strings.Contains(lowerMsg, "github_token") ||
		strings.Contains(lowerMsg, "authentication") ||
		strings.Contains(lowerMsg, "not logged into") ||
		strings.Contains(lowerMsg, "unauthorized") ||
		strings.Contains(lowerMsg, "forbidden") ||
		strings.Contains(lowerMsg, "permission denied") ||
		strings.Contains(lowerMsg, "saml enforcement")
	if isAuth {
		gitutilLog.Print("Detected authentication error")
	}
	return isAuth
}

// IsHexString checks if a string contains only hexadecimal characters.
// This is used to validate Git commit SHAs and other hexadecimal identifiers.
func IsHexString(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// IsValidFullSHA checks if s is a valid 40-character lowercase hexadecimal SHA.
func IsValidFullSHA(s string) bool {
	return fullSHARegex.MatchString(s)
}

// ExtractBaseRepo extracts the base repository (owner/repo) from a repository path
// that may include subfolders.
// For "actions/checkout" -> "actions/checkout"
// For "github/codeql-action/upload-sarif" -> "github/codeql-action"
func ExtractBaseRepo(repoPath string) string {
	parts := strings.Split(repoPath, "/")
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return repoPath
}

// FindGitRoot finds the root directory of the git repository.
// Uses pure Go filesystem traversal to avoid requiring the git executable,
// which can fail when the binary runs under Rosetta 2 on macOS ARM64 or in
// environments where git is not on PATH.
// Returns an error if not in a git repository.
func FindGitRoot() (string, error) {
	gitutilLog.Print("Finding git root directory")

	dir, err := os.Getwd()
	if err != nil {
		gitutilLog.Printf("Failed to get current directory: %v", err)
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	root, err := FindGitRootFrom(dir)
	if err != nil {
		gitutilLog.Printf("Failed to find git root: %v", err)
		return "", err
	}

	gitutilLog.Printf("Found git root: %s", root)
	return root, nil
}

// FindGitRootFrom finds the root directory of the git repository starting from
// the given directory. It traverses upward until it finds a .git entry (file or
// directory) or reaches the filesystem root.
// Returns an error if not in a git repository.
func FindGitRootFrom(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path for %q: %w", startDir, err)
	}
	dir = filepath.Clean(dir)
	for {
		gitPath := filepath.Join(dir, ".git")
		info, err := os.Stat(gitPath)
		if err == nil {
			// .git exists — accept if it's a directory (normal repo) or a
			// regular file (worktree / git-submodule pointer).
			if info.IsDir() {
				return dir, nil
			}
			// Worktree marker: must be a regular file beginning with "gitdir:"
			if info.Mode().IsRegular() {
				data, readErr := os.ReadFile(gitPath)
				if readErr != nil {
					return "", fmt.Errorf("failed to read .git file at %q: %w", gitPath, readErr)
				}
				if strings.HasPrefix(strings.TrimSpace(string(data)), "gitdir:") {
					return dir, nil
				}
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			// Unexpected error (e.g. permission denied) — surface it.
			return "", fmt.Errorf("failed to stat %q: %w", gitPath, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotGitRepository
		}
		dir = parent
	}
}

// ReadFileFromHEAD reads a file from git HEAD using a pre-computed repository root.
// filePath is resolved with filepath.Abs, so relative paths are interpreted from the
// current process working directory (not gitRoot). Prefer passing an absolute path
// within gitRoot, such as filepath.Join(gitRoot, "path/to/file").
// The implementation avoids git show HEAD:path interpolation by resolving a
// literal tree entry with git ls-tree, validating the resulting blob object ID,
// and then reading the blob with git cat-file.
// Use this when the caller already knows the git root (e.g. from a cached value).
func ReadFileFromHEAD(filePath, gitRoot string) (string, error) {
	if gitRoot == "" {
		return "", fmt.Errorf("gitRoot must not be empty when reading %q from HEAD", filePath)
	}

	cleanGitRoot, err := fileutil.ValidateAbsolutePath(gitRoot)
	if err != nil {
		return "", fmt.Errorf("invalid git repository root %q: %w", gitRoot, err)
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("cannot resolve absolute path for %q: %w", filePath, err)
	}
	if err := fileutil.ValidatePathWithinBase(cleanGitRoot, absPath); err != nil {
		return "", fmt.Errorf("path %q is outside the git repository root %q", filePath, gitRoot)
	}

	// git ls-tree pathspecs require the path to be relative to the repository root
	// and to use forward slashes even on Windows.
	relPath, err := filepath.Rel(cleanGitRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("cannot compute path of %q relative to git root %q: %w", absPath, cleanGitRoot, err)
	}

	// Reject paths that escape the repository (e.g. "../secret").
	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("path %q is outside the git repository root %q", filePath, gitRoot)
	}

	relPath = filepath.ToSlash(relPath)

	gitutilLog.Printf("Reading %q from git HEAD (relative path: %s)", filePath, relPath)

	blobID, err := resolveHEADBlobID(cleanGitRoot, relPath)
	if err != nil {
		gitutilLog.Printf("File %q not found in HEAD commit: %v", filePath, err)
		return "", fmt.Errorf("file %q not found in HEAD commit: %w", filePath, err)
	}

	cmd := exec.Command("git", "-C", cleanGitRoot, "cat-file", "blob", blobID)
	output, err := cmd.Output()
	if err != nil {
		gitutilLog.Printf("File %q not found in HEAD commit: %v", filePath, err)
		return "", fmt.Errorf("file %q not found in HEAD commit: %w", filePath, err)
	}
	return string(output), nil
}

func resolveHEADBlobID(gitRoot, relPath string) (string, error) {
	pathspec := ":(literal)" + relPath
	cmd := exec.Command("git", "-C", gitRoot, "ls-tree", "-z", "--full-tree", "HEAD", "--", pathspec)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	if len(output) == 0 {
		return "", fmt.Errorf("path %q not found in HEAD", relPath)
	}

	entry := strings.TrimSuffix(string(output), "\x00")
	metadata, entryPath, found := strings.Cut(entry, "\t")
	if !found {
		return "", fmt.Errorf("unexpected git ls-tree output for %q", relPath)
	}
	fields := strings.Fields(metadata)
	if len(fields) != 3 {
		return "", fmt.Errorf("unexpected git ls-tree metadata for %q", relPath)
	}
	// relPath is already normalized to forward slashes via filepath.ToSlash above,
	// and git ls-tree also emits forward slashes on all platforms.
	if entryPath != relPath {
		return "", fmt.Errorf("path %q resolved to unexpected entry %q", relPath, entryPath)
	}
	if fields[1] != "blob" {
		return "", fmt.Errorf("path %q in HEAD is not a file", relPath)
	}
	if !isGitObjectID(fields[2]) {
		return "", fmt.Errorf("path %q in HEAD resolved to invalid object ID %q", relPath, fields[2])
	}

	return fields[2], nil
}

func isGitObjectID(s string) bool {
	return gitObjectIDRegex.MatchString(s)
}
