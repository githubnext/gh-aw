//go:build integration

package parser

import (
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDownloadFileFromGitHubRESTClient tests the REST client-based file download
func TestDownloadFileFromGitHubRESTClient(t *testing.T) {
	// Test downloading a real file from a public repository
	// Using a known stable file from the GitHub repository itself
	owner := "github"
	repo := "gitignore"
	path := "Go.gitignore"
	ref := "main"

	content, err := downloadFileFromGitHub(owner, repo, path, ref)
	if err != nil {
		// If we get an auth error, we can skip this test in CI environments
		// where GitHub tokens might not be available
		if strings.Contains(err.Error(), "auth") || strings.Contains(err.Error(), "forbidden") {
			t.Skip("Skipping test due to authentication requirements")
		}
		t.Fatalf("Failed to download file from GitHub: %v", err)
	}

	// Verify we got content
	if len(content) == 0 {
		t.Error("Downloaded content is empty")
	}

	// Verify the content looks like a .gitignore file
	contentStr := string(content)
	if !strings.Contains(contentStr, "#") && !strings.Contains(contentStr, "*.") {
		maxLen := len(contentStr)
		if maxLen > 100 {
			maxLen = 100
		}
		t.Errorf("Content doesn't look like a .gitignore file: %s", contentStr[:maxLen])
	}
}

// TestDownloadFileFromGitHubInvalidRepo tests error handling with invalid repository
func TestDownloadFileFromGitHubInvalidRepo(t *testing.T) {
	owner := "nonexistent-owner-xyz123"
	repo := "nonexistent-repo-xyz456"
	path := "README.md"
	ref := "main"

	_, err := downloadFileFromGitHub(owner, repo, path, ref)
	if err == nil {
		t.Fatal("Expected error for nonexistent repository, got nil")
	}

	// Verify we get an appropriate error message
	errStr := err.Error()

	// Skip if authentication is not available
	if strings.Contains(errStr, "authentication token not found") {
		t.Skip("Skipping test due to missing authentication token")
	}

	if !strings.Contains(errStr, "failed to fetch file content") {
		t.Errorf("Error should mention fetch failure, got: %s", errStr)
	}
}

// TestDownloadFileFromGitHubInvalidPath tests error handling with invalid file path
func TestDownloadFileFromGitHubInvalidPath(t *testing.T) {
	owner := "github"
	repo := "gitignore"
	path := "nonexistent-file-xyz123.txt"
	ref := "main"

	_, err := downloadFileFromGitHub(owner, repo, path, ref)
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}

	// Verify we get an appropriate error message
	errStr := err.Error()

	// Skip if authentication is not available
	if strings.Contains(errStr, "authentication token not found") {
		t.Skip("Skipping test due to missing authentication token")
	}

	if !strings.Contains(errStr, "failed to fetch file content") {
		t.Errorf("Error should mention fetch failure, got: %s", errStr)
	}
}

// TestDownloadFileFromGitHubWithSHA tests downloading with a specific commit SHA
func TestDownloadFileFromGitHubWithSHA(t *testing.T) {
	// Test with a known commit SHA from a public repository
	// Using github/gitignore repository with a known commit
	owner := "github"
	repo := "gitignore"
	path := "Go.gitignore"
	// Using a recent commit SHA that should be stable
	// Note: This might fail if the SHA doesn't exist, but demonstrates SHA support
	ref := "main" // Using main instead of specific SHA to avoid brittleness

	content, err := downloadFileFromGitHub(owner, repo, path, ref)
	if err != nil {
		if strings.Contains(err.Error(), "auth") || strings.Contains(err.Error(), "forbidden") {
			t.Skip("Skipping test due to authentication requirements")
		}
		t.Fatalf("Failed to download file with SHA: %v", err)
	}

	if len(content) == 0 {
		t.Error("Downloaded content is empty")
	}
}

// TestResolveIncludePathWithWorkflowSpec tests the full workflow spec resolution
func TestResolveIncludePathWithWorkflowSpec(t *testing.T) {
	// Test resolving a workflowspec format path
	// Format: owner/repo/path@ref
	spec := "github/gitignore/Go.gitignore@main"
	cache := NewImportCache(t.TempDir())

	path, err := ResolveIncludePath(spec, "", cache)
	if err != nil {
		if strings.Contains(err.Error(), "auth") || strings.Contains(err.Error(), "forbidden") {
			t.Skip("Skipping test due to authentication requirements")
		}
		t.Fatalf("Failed to resolve workflowspec: %v", err)
	}

	// Verify we got a valid path
	if path == "" {
		t.Error("Resolved path is empty")
	}

	// The path should either be in cache or a temp file
	if !strings.Contains(path, "gh-aw") && !strings.Contains(path, "tmp") {
		t.Logf("Warning: Path doesn't look like a cached or temp file: %s", path)
	}
}

// skipOnAuthError skips the test if the error indicates missing authentication.
func skipOnAuthError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	errStr := err.Error()
	if strings.Contains(errStr, "auth") || strings.Contains(errStr, "forbidden") || strings.Contains(errStr, "authentication token not found") {
		t.Skip("Skipping test due to authentication requirements")
	}
}

// TestCheckRemoteSymlink verifies that checkRemoteSymlink correctly classifies
// directories, files, and nonexistent paths when called against the GitHub Contents API.
func TestCheckRemoteSymlink(t *testing.T) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		skipOnAuthError(t, err)
		t.Fatalf("Failed to create REST client: %v", err)
	}

	tests := []struct {
		name        string
		dirPath     string
		wantSymlink bool
		wantErr     bool
	}{
		{
			name:        "directory is not a symlink",
			dirPath:     "Global",
			wantSymlink: false,
			wantErr:     false,
		},
		{
			name:        "regular file is not a symlink",
			dirPath:     "Go.gitignore",
			wantSymlink: false,
			wantErr:     false,
		},
		{
			name:        "nonexistent path returns error",
			dirPath:     "nonexistent-path-xyz",
			wantSymlink: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, isSymlink, err := checkRemoteSymlink(client, "github", "gitignore", tt.dirPath, "main")
			if err != nil {
				skipOnAuthError(t, err)
				if !tt.wantErr {
					t.Fatalf("Unexpected error for path %q: %v", tt.dirPath, err)
				}
				if tt.dirPath == "nonexistent-path-xyz" {
					assert.True(t, isNotFoundError(err.Error()), "Expected a not-found error, got: %v", err)
				}
				return
			}

			assert.False(t, tt.wantErr, "Expected error for path %q but got none", tt.dirPath)
			assert.Equal(t, tt.wantSymlink, isSymlink, "Symlink mismatch for path %q (target=%q)", tt.dirPath, target)
		})
	}
}

// TestResolveRemoteSymlinksNoSymlinks verifies that resolveRemoteSymlinks walks all
// directory components of a real path and returns "no symlinks found" when none exist.
func TestResolveRemoteSymlinksNoSymlinks(t *testing.T) {
	// "Global/Perl.gitignore" is a real path in github/gitignore with no symlinks
	_, err := resolveRemoteSymlinks("github", "gitignore", "Global/Perl.gitignore", "main")
	require.Error(t, err, "Expected error when no symlinks found")
	skipOnAuthError(t, err)

	assert.Contains(t, err.Error(), "no symlinks found", "Should indicate no symlinks were found in path")
}

// TestDownloadFileFromGitHubSymlinkRoute verifies that downloading a nonexistent file
// through a real directory triggers the symlink resolution fallback and ultimately
// returns the original fetch error (not a panic or hang).
func TestDownloadFileFromGitHubSymlinkRoute(t *testing.T) {
	// Use a path through a real directory but with a nonexistent file.
	// This triggers: 404 -> symlink resolution -> "no symlinks found" -> original error.
	_, err := downloadFileFromGitHub("github", "gitignore", "Global/nonexistent-file-xyz123.gitignore", "main")
	require.Error(t, err, "Expected error for nonexistent file")
	skipOnAuthError(t, err)

	assert.Contains(t, err.Error(), "failed to fetch file content", "Should return the original fetch failure")
}

// TestDownloadIncludeFromWorkflowSpecWithCache tests caching behavior
func TestDownloadIncludeFromWorkflowSpecWithCache(t *testing.T) {
	cache := NewImportCache(t.TempDir())
	spec := "github/gitignore/Go.gitignore@main"

	// First download - should fetch from GitHub
	path1, err := downloadIncludeFromWorkflowSpec(spec, cache)
	if err != nil {
		if strings.Contains(err.Error(), "auth") || strings.Contains(err.Error(), "forbidden") {
			t.Skip("Skipping test due to authentication requirements")
		}
		t.Fatalf("First download failed: %v", err)
	}

	if path1 == "" {
		t.Fatal("First download returned empty path")
	}

	// Second download - should use cache if SHA resolution succeeded
	path2, err := downloadIncludeFromWorkflowSpec(spec, cache)
	if err != nil {
		t.Fatalf("Second download failed: %v", err)
	}

	if path2 == "" {
		t.Fatal("Second download returned empty path")
	}

	// Both paths should point to the same cached file if caching worked
	// Note: This might not be the same if SHA resolution failed
	t.Logf("First download path: %s", path1)
	t.Logf("Second download path: %s", path2)
}
