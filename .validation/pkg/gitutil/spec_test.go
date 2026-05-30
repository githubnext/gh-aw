//go:build !integration

package gitutil_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/gitutil"
)

// TestSpec_PublicAPI_IsRateLimitError validates the documented behavior of
// IsRateLimitError as described in the package README.md.
//
// Specification: Returns true when errMsg indicates a GitHub API rate-limit
// error (case-insensitive match against "api rate limit exceeded",
// "rate limit exceeded", or "secondary rate limit").
func TestSpec_PublicAPI_IsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{
			name:     "documented phrase 'api rate limit exceeded' returns true",
			errMsg:   "403: API rate limit exceeded",
			expected: true,
		},
		{
			name:     "documented phrase 'rate limit exceeded' returns true",
			errMsg:   "rate limit exceeded for user ID 123",
			expected: true,
		},
		{
			name:     "documented phrase 'secondary rate limit' returns true",
			errMsg:   "secondary rate limit triggered",
			expected: true,
		},
		{
			name:     "case-insensitive match returns true (documented as case-insensitive)",
			errMsg:   "API RATE LIMIT EXCEEDED",
			expected: true,
		},
		{
			name:     "unrelated error message returns false",
			errMsg:   "404: not found",
			expected: false,
		},
		{
			name:     "empty string returns false",
			errMsg:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gitutil.IsRateLimitError(tt.errMsg)
			assert.Equal(t, tt.expected, result,
				"IsRateLimitError(%q) should match documented behavior", tt.errMsg)
		})
	}
}

// TestSpec_PublicAPI_IsAuthError validates the documented behavior of
// IsAuthError as described in the package README.md.
//
// Specification: Returns true when errMsg indicates an authentication or
// authorization failure (GH_TOKEN, GITHUB_TOKEN, unauthorized, forbidden,
// SAML enforcement, etc.).
func TestSpec_PublicAPI_IsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{
			name:     "GH_TOKEN reference returns true",
			errMsg:   "GH_TOKEN is invalid or expired",
			expected: true,
		},
		{
			name:     "GITHUB_TOKEN reference returns true",
			errMsg:   "GITHUB_TOKEN: authentication failed",
			expected: true,
		},
		{
			name:     "unauthorized returns true",
			errMsg:   "401: unauthorized",
			expected: true,
		},
		{
			name:     "forbidden returns true",
			errMsg:   "403: forbidden",
			expected: true,
		},
		{
			name:     "SAML enforcement message returns true (documented)",
			errMsg:   "Resource protected by organization SAML enforcement",
			expected: true,
		},
		{
			name:     "unrelated error returns false",
			errMsg:   "404: not found",
			expected: false,
		},
		{
			name:     "empty string returns false",
			errMsg:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gitutil.IsAuthError(tt.errMsg)
			assert.Equal(t, tt.expected, result,
				"IsAuthError(%q) should match documented behavior", tt.errMsg)
		})
	}
}

// TestSpec_PublicAPI_IsHexString validates the documented behavior of
// IsHexString as described in the package README.md.
//
// Specification: Returns true if s consists entirely of hexadecimal characters
// (0–9, a–f, A–F). Returns false for the empty string.
func TestSpec_PublicAPI_IsHexString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "lowercase hex digits returns true",
			input:    "abcdef0123456789",
			expected: true,
		},
		{
			name:     "uppercase hex digits returns true",
			input:    "ABCDEF0123456789",
			expected: true,
		},
		{
			name:     "mixed case hex digits returns true",
			input:    "AbCdEf01",
			expected: true,
		},
		{
			name:     "numeric only returns true",
			input:    "123456",
			expected: true,
		},
		{
			name:     "non-hex character returns false",
			input:    "abcg",
			expected: false,
		},
		{
			name:     "empty string returns false (documented edge case)",
			input:    "",
			expected: false,
		},
		{
			name:     "string with space returns false",
			input:    "abc def",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gitutil.IsHexString(tt.input)
			assert.Equal(t, tt.expected, result,
				"IsHexString(%q) should match documented behavior", tt.input)
		})
	}
}

// TestSpec_PublicAPI_ExtractBaseRepo validates the documented behavior of
// ExtractBaseRepo as described in the package README.md.
//
// Specification: Extracts the owner/repo portion from an action path that may
// include a sub-folder.
//
// Documented examples:
//
//	gitutil.ExtractBaseRepo("actions/checkout")                   → "actions/checkout"
//	gitutil.ExtractBaseRepo("github/codeql-action/upload-sarif") → "github/codeql-action"
func TestSpec_PublicAPI_ExtractBaseRepo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "two-segment path returns as-is (documented example)",
			input:    "actions/checkout",
			expected: "actions/checkout",
		},
		{
			name:     "three-segment path strips sub-folder (documented example)",
			input:    "github/codeql-action/upload-sarif",
			expected: "github/codeql-action",
		},
		{
			name:     "four-segment path returns owner/repo only",
			input:    "owner/repo/sub/path",
			expected: "owner/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gitutil.ExtractBaseRepo(tt.input)
			assert.Equal(t, tt.expected, result,
				"ExtractBaseRepo(%q) should extract owner/repo portion", tt.input)
		})
	}
}

// TestSpec_PublicAPI_IsValidFullSHA validates the documented behavior of
// IsValidFullSHA as described in the package README.md.
//
// Specification: Returns true if s is a valid 40-character lowercase hexadecimal
// SHA (the standard Git commit SHA format). Use this for strict SHA validation
// when the full 40-character form is required.
func TestSpec_PublicAPI_IsValidFullSHA(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "40-character lowercase hex returns true",
			input:    "da39a3ee5e6b4b0d3255bfef95601890afd80709",
			expected: true,
		},
		{
			name:     "40-character with uppercase hex returns false (must be lowercase)",
			input:    "DA39A3EE5E6B4B0D3255BFEF95601890AFD80709",
			expected: false,
		},
		{
			name:     "39 characters returns false (too short)",
			input:    "da39a3ee5e6b4b0d3255bfef95601890afd807",
			expected: false,
		},
		{
			name:     "41 characters returns false (too long)",
			input:    "da39a3ee5e6b4b0d3255bfef95601890afd807091",
			expected: false,
		},
		{
			name:     "empty string returns false",
			input:    "",
			expected: false,
		},
		{
			name:     "non-hex character in 40-char string returns false",
			input:    "za39a3ee5e6b4b0d3255bfef95601890afd80709",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gitutil.IsValidFullSHA(tt.input)
			assert.Equal(t, tt.expected, result,
				"IsValidFullSHA(%q) should match documented behavior", tt.input)
		})
	}
}

// TestSpec_PublicAPI_FindGitRoot validates the documented behavior of
// FindGitRoot as described in the package README.md.
//
// Specification: Returns the absolute path of the root directory of the current
// Git repository using pure Go filesystem traversal (no `git` subprocess);
// starts from the current working directory.
func TestSpec_PublicAPI_FindGitRoot(t *testing.T) {
	t.Run("returns non-empty absolute path when in git repository", func(t *testing.T) {
		root, err := gitutil.FindGitRoot()
		require.NoError(t, err, "FindGitRoot should not error when inside a git repository")
		assert.NotEmpty(t, root, "FindGitRoot should return a non-empty path")
		assert.True(t, filepath.IsAbs(root),
			"FindGitRoot should return an absolute path, got %q", root)
	})
}

// TestSpec_PublicAPI_FindGitRootFrom validates the documented behavior of
// FindGitRootFrom as described in the package README.md.
//
// Specification: Like FindGitRoot but starts from startDir; traverses upward
// looking for a .git directory or worktree marker file (a `.git` file starting
// with `gitdir:`).
func TestSpec_PublicAPI_FindGitRootFrom(t *testing.T) {
	t.Run("returns absolute repository root when startDir is inside a repo", func(t *testing.T) {
		// The current working directory of this test is inside the gh-aw
		// repository, so any subdirectory inside it should resolve to the
		// repository root.
		repoRoot, err := gitutil.FindGitRoot()
		require.NoError(t, err, "FindGitRoot should succeed inside the gh-aw repository")

		root, err := gitutil.FindGitRootFrom(repoRoot)
		require.NoError(t, err, "FindGitRootFrom should succeed when startDir is inside a repository")
		assert.NotEmpty(t, root, "FindGitRootFrom should return a non-empty path")
		assert.True(t, filepath.IsAbs(root),
			"FindGitRootFrom should return an absolute path, got %q", root)
	})

	t.Run("traverses upward from a subdirectory to locate the repository root", func(t *testing.T) {
		repoRoot, err := gitutil.FindGitRoot()
		require.NoError(t, err, "FindGitRoot should succeed inside the gh-aw repository")

		// Start from a subdirectory of the repo; FindGitRootFrom should walk
		// upward and land on the same root.
		fromSub, err := gitutil.FindGitRootFrom(filepath.Join(repoRoot, "pkg", "gitutil"))
		require.NoError(t, err, "FindGitRootFrom should succeed from a subdirectory")
		assert.Equal(t, repoRoot, fromSub,
			"FindGitRootFrom from a subdirectory should return the same root as FindGitRoot")
	})

	t.Run("returns error when startDir is not inside a git repository", func(t *testing.T) {
		// A directory created outside any git repository should produce an error.
		isolated := t.TempDir()
		_, err := gitutil.FindGitRootFrom(isolated)
		assert.Error(t, err,
			"FindGitRootFrom should return an error when startDir is not inside a git repository")
	})
}

// TestSpec_PublicAPI_ReadFileFromHEAD validates the documented behavior of
// ReadFileFromHEAD as described in the package README.md.
//
// Specification: Reads a file's content from the HEAD commit without touching
// the working tree; rejects paths that escape the repository.
func TestSpec_PublicAPI_ReadFileFromHEAD(t *testing.T) {
	root, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skip("not inside a git repository, skipping ReadFileFromHEAD tests")
	}

	t.Run("reads known file from HEAD without error", func(t *testing.T) {
		content, err := gitutil.ReadFileFromHEAD(filepath.Join(root, "go.mod"), root)
		require.NoError(t, err, "ReadFileFromHEAD should read go.mod without error")
		assert.NotEmpty(t, content, "content of go.mod should not be empty")
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := gitutil.ReadFileFromHEAD("this-file-does-not-exist-xyzzy.txt", root)
		assert.Error(t, err, "ReadFileFromHEAD should return error for non-existent file")
	})

	t.Run("rejects path with .. traversal", func(t *testing.T) {
		// Specification: "The function rejects paths that escape the repository
		// (i.e. paths containing .. after resolution)."
		outsidePath := filepath.Join(root, "..", "outside.txt")
		_, err := gitutil.ReadFileFromHEAD(outsidePath, root)
		require.Error(t, err, "ReadFileFromHEAD should reject path-traversal attempts")
		assert.Contains(t, err.Error(), "outside the git repository root")
	})

	t.Run("returns error when gitRoot is empty", func(t *testing.T) {
		// Specification: gitRoot must be the repository root (from FindGitRoot)
		_, err := gitutil.ReadFileFromHEAD("go.mod", "")
		assert.Error(t, err, "ReadFileFromHEAD should return error when gitRoot is empty")
	})
}
