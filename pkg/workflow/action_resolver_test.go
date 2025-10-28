package workflow

import (
	"strings"
	"testing"
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
	tmpDir := t.TempDir()
	cache := NewActionCache(tmpDir)
	resolver := NewActionResolver(cache)

	// Manually add an entry to the cache with a valid SHA
	validSHA := "08c6903cd8c0fde910a37f88322edcfb5dd907a8"
	cache.Set("actions/checkout", "v5", validSHA)

	// Resolve should return cached value without making API call
	sha, err := resolver.ResolveSHA("actions/checkout", "v5")
	if err != nil {
		t.Errorf("Expected no error for cached entry, got: %v", err)
	}
	if sha != validSHA {
		t.Errorf("Expected SHA '%s', got '%s'", validSHA, sha)
	}
}

// Note: Testing the actual GitHub API resolution requires network access
// and is tested in integration tests or with network-dependent test tags

func TestValidateSHA(t *testing.T) {
	tests := []struct {
		name        string
		sha         string
		repo        string
		version     string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "valid SHA",
			sha:         "08c6903cd8c0fde910a37f88322edcfb5dd907a8",
			repo:        "actions/checkout",
			version:     "v5",
			shouldError: false,
		},
		{
			name:        "invalid SHA - too short",
			sha:         "08c6903cd8c0fde910a37f88322edcfb5dd907a",
			repo:        "test/action",
			version:     "v1",
			shouldError: true,
			errorMsg:    "expected 40 characters",
		},
		{
			name:        "invalid SHA - too long",
			sha:         "08c6903cd8c0fde910a37f88322edcfb5dd907a8a",
			repo:        "test/action",
			version:     "v1",
			shouldError: true,
			errorMsg:    "expected 40 characters",
		},
		{
			name:        "invalid SHA - non-hex characters",
			sha:         "gggggggggggggggggggggggggggggggggggggggg",
			repo:        "test/action",
			version:     "v1",
			shouldError: true,
			errorMsg:    "must be 40 hex characters",
		},
		{
			name:        "invalid SHA - empty",
			sha:         "",
			repo:        "test/action",
			version:     "v1",
			shouldError: true,
			errorMsg:    "empty SHA",
		},
		{
			name:        "invalid SHA - special characters",
			sha:         "08c6903cd8c0fde910a37f88322edcfb5dd907@8",
			repo:        "test/action",
			version:     "v1",
			shouldError: true,
			errorMsg:    "must be 40 hex characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateSHA(tt.sha, tt.repo, tt.version)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				if result != "" {
					t.Errorf("Expected empty result on error, got %q", result)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if result != tt.sha {
					t.Errorf("Expected SHA %q, got %q", tt.sha, result)
				}
			}
		})
	}
}
