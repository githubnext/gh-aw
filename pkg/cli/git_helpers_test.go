//go:build !integration

package cli

import (
	"testing"
)

func TestParseGitHubRepoSlugFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "HTTPS URL with .git",
			url:      "https://github.com/github/gh-aw.git",
			expected: "github/gh-aw",
		},
		{
			name:     "HTTPS URL without .git",
			url:      "https://github.com/github/gh-aw",
			expected: "github/gh-aw",
		},
		{
			name:     "SSH URL with .git",
			url:      "git@github.com:github/gh-aw.git",
			expected: "github/gh-aw",
		},
		{
			name:     "SSH URL without .git",
			url:      "git@github.com:github/gh-aw",
			expected: "github/gh-aw",
		},
		{
			name:     "SSH SCP-style URL without username with .git",
			url:      "github.com:github/gh-aw.git",
			expected: "github/gh-aw",
		},
		{
			name:     "SSH SCP-style URL without username without .git",
			url:      "github.com:github/gh-aw",
			expected: "github/gh-aw",
		},
		{
			name:     "SSH URL scheme with .git",
			url:      "ssh://git@github.com/github/gh-aw.git",
			expected: "github/gh-aw",
		},
		{
			name:     "SSH URL scheme without .git",
			url:      "ssh://git@github.com/github/gh-aw",
			expected: "github/gh-aw",
		},
		{
			name:     "SSH URL scheme without user info",
			url:      "ssh://github.com/github/gh-aw.git",
			expected: "github/gh-aw",
		},
		{
			name:     "Invalid URL",
			url:      "not-a-github-url",
			expected: "",
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: "",
		},
		{
			name:     "URL with subdirectory",
			url:      "https://github.com/owner/repo/subfolder",
			expected: "owner/repo/subfolder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGitHubRepoSlugFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("parseGitHubRepoSlugFromURL(%q) = %q, expected %q", tt.url, result, tt.expected)
			}
		})
	}
}

// TestParseGitHubRepoSlugFromURLGHE verifies that GHE SSH URLs with non-standard
// usernames (e.g. example@example.ghe.com:owner/repo.git) are parsed correctly
// when GITHUB_SERVER_URL is configured to point at the GHE instance.
func TestParseGitHubRepoSlugFromURLGHE(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "GHE SSH SCP-style with custom username and .git",
			url:      "example@example.ghe.com:example-org/example-repo.git",
			expected: "example-org/example-repo",
		},
		{
			name:     "GHE SSH SCP-style with custom username without .git",
			url:      "example@example.ghe.com:example-org/example-repo",
			expected: "example-org/example-repo",
		},
		{
			name:     "GHE SSH SCP-style with git username and .git",
			url:      "git@example.ghe.com:example-org/example-repo.git",
			expected: "example-org/example-repo",
		},
		{
			name:     "GHE HTTPS URL",
			url:      "https://example.ghe.com/example-org/example-repo.git",
			expected: "example-org/example-repo",
		},
		{
			name:     "GHE SSH URL scheme",
			url:      "ssh://example@example.ghe.com/example-org/example-repo.git",
			expected: "example-org/example-repo",
		},
		{
			name:     "SSH SCP-style URL for a different host should not match",
			url:      "example@other.host.com:example-org/example-repo.git",
			expected: "",
		},
	}

	t.Setenv("GITHUB_SERVER_URL", "https://example.ghe.com")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGitHubRepoSlugFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("parseGitHubRepoSlugFromURL(%q) = %q, expected %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestGetRepositorySlugFromRemote(t *testing.T) {
	// This test verifies that the function can execute without errors in a git repo
	// The actual value will depend on the repository being tested
	result := getRepositorySlugFromRemote()

	// In the gh-aw repository, we should get a non-empty slug
	// But we can't assert the exact value since it might change
	if result != "" {
		t.Logf("Repository slug: %s", result)
	} else {
		t.Log("Repository slug is empty (may be expected if not in a git repo)")
	}
}

func TestFindGitRootForPath(t *testing.T) {
	// Test with current file path
	gitRoot, err := findGitRootForPath("git_helpers_test.go")
	if err != nil {
		// This is okay if we're not in a git repository
		t.Logf("findGitRootForPath returned error: %v", err)
		return
	}

	if gitRoot == "" {
		t.Error("findGitRootForPath returned empty string without error")
	} else {
		t.Logf("Git root: %s", gitRoot)
	}
}

func TestGetRepositorySlugFromRemoteForPath(t *testing.T) {
	// Test with current file path
	slug := getRepositorySlugFromRemoteForPath("git_helpers_test.go")

	// Log the result - we can't assert exact value
	if slug != "" {
		t.Logf("Repository slug for path: %s", slug)
	} else {
		t.Log("Repository slug for path is empty (may be expected if not in a git repo)")
	}
}
