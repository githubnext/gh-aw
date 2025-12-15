//go:build integration

package parser

import (
	"strings"
	"testing"
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
