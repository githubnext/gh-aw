//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestExtractBaseRepo(t *testing.T) {
	tests := []struct {
		name     string
		repo     string
		expected string
	}{
		{
			name:     "simple repo",
			repo:     "actions/checkout",
			expected: "actions/checkout",
		},
		{
			name:     "repo with subpath",
			repo:     "github/codeql-action/upload-sarif",
			expected: "github/codeql-action",
		},
		{
			name:     "repo with multiple subpaths",
			repo:     "owner/repo/sub/path",
			expected: "owner/repo",
		},
		{
			name:     "single part repo",
			repo:     "myrepo",
			expected: "myrepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBaseRepo(tt.repo)
			if result != tt.expected {
				t.Errorf("extractBaseRepo(%q) = %q, want %q", tt.repo, result, tt.expected)
			}
		})
	}
}

func TestActionResolverCache(t *testing.T) {
	// Create a cache and resolver
	tmpDir := testutil.TempDir(t, "test-*")
	cache := NewActionCache(tmpDir)
	resolver := NewActionResolver(cache)

	// Manually add an entry to the cache
	cache.Set("actions/checkout", "v5", "test-sha-123")

	// Resolve should return cached value without making API call
	sha, err := resolver.ResolveSHA("actions/checkout", "v5")
	if err != nil {
		t.Errorf("Expected no error for cached entry, got: %v", err)
	}
	if sha != "test-sha-123" {
		t.Errorf("Expected SHA 'test-sha-123', got '%s'", sha)
	}
}

func TestActionResolverFailedResolutionCache(t *testing.T) {
	// Create a cache and resolver
	tmpDir := testutil.TempDir(t, "test-*")
	cache := NewActionCache(tmpDir)
	resolver := NewActionResolver(cache)

	// Attempt to resolve a non-existent action
	// This will fail since we don't have a valid GitHub API connection in tests
	repo := "nonexistent/action"
	version := "v999.999.999"

	// First attempt should try to resolve
	_, err1 := resolver.ResolveSHA(repo, version)
	if err1 == nil {
		t.Error("Expected error for non-existent action on first attempt")
	}

	// Verify the failed resolution was tracked
	cacheKey := formatActionCacheKey(repo, version)
	if !resolver.failedResolutions[cacheKey] {
		t.Errorf("Expected failed resolution to be tracked for %s", cacheKey)
	}

	// Second attempt should be skipped and return error immediately
	_, err2 := resolver.ResolveSHA(repo, version)
	if err2 == nil {
		t.Error("Expected error for non-existent action on second attempt")
	}

	// Verify the error message indicates it was skipped
	expectedErrMsg := "previously failed to resolve"
	if !strings.Contains(err2.Error(), expectedErrMsg) {
		t.Errorf("Expected error message to contain %q, got: %v", expectedErrMsg, err2)
	}
}

// Note: Testing the actual GitHub API resolution requires network access
// and is tested in integration tests or with network-dependent test tags
