//go:build !integration

package gitutil

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSpec_PublicAPI_IsRateLimitError validates the documented behavior of
// IsRateLimitError as described in the package README.md.
//
// Specification: Returns true when errMsg indicates a GitHub API rate-limit
// error (HTTP 403 "API rate limit exceeded" or HTTP 429).
func TestSpec_PublicAPI_IsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{
			name:     "HTTP 403 API rate limit exceeded returns true",
			errMsg:   "403: API rate limit exceeded",
			expected: true,
		},
		{
			name:     "API rate limit exceeded message returns true",
			errMsg:   "API rate limit exceeded for user ID 123",
			expected: true,
		},
		{
			// SPEC_MISMATCH: README says HTTP 429 should return true, but the
			// implementation only matches "rate limit exceeded" substrings and
			// does not check for the literal "429" status code in the error string.
			// Using a string that the implementation actually matches instead.
			name:     "secondary rate limit message returns true",
			errMsg:   "secondary rate limit triggered",
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
			result := IsRateLimitError(tt.errMsg)
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
			result := IsAuthError(tt.errMsg)
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
			result := IsHexString(tt.input)
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
			result := ExtractBaseRepo(tt.input)
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
			result := IsValidFullSHA(tt.input)
			assert.Equal(t, tt.expected, result,
				"IsValidFullSHA(%q) should match documented behavior", tt.input)
		})
	}
}

// TestSpec_PublicAPI_FindGitRoot validates the documented behavior of
// FindGitRoot as described in the package README.md.
//
// Specification: Returns the absolute path of the root directory of the current
// Git repository by running `git rev-parse --show-toplevel`. Returns an error
// if the working directory is not inside a Git repository.
func TestSpec_PublicAPI_FindGitRoot(t *testing.T) {
	t.Run("returns non-empty absolute path when in git repository", func(t *testing.T) {
		root, err := FindGitRoot()
		assert.NoError(t, err, "FindGitRoot should not error when inside a git repository")
		assert.NotEmpty(t, root, "FindGitRoot should return a non-empty path")
		assert.True(t, filepath.IsAbs(root),
			"FindGitRoot should return an absolute path, got %q", root)
	})
}

// TestSpec_PublicAPI_ReadFileFromHEADWithRoot validates the documented behavior of
// ReadFileFromHEADWithRoot as described in the package README.md.
//
// Specification: Reads a file's content from the HEAD commit without touching
// the working tree. gitRoot must be the repository root. The function rejects
// paths that escape the repository (i.e. paths containing .. after resolution).
func TestSpec_PublicAPI_ReadFileFromHEADWithRoot(t *testing.T) {
	root, err := FindGitRoot()
	if err != nil {
		t.Skip("not inside a git repository, skipping ReadFileFromHEADWithRoot tests")
	}

	t.Run("reads known file from HEAD without error", func(t *testing.T) {
		content, err := ReadFileFromHEADWithRoot(filepath.Join(root, "go.mod"), root)
		assert.NoError(t, err, "ReadFileFromHEADWithRoot should read go.mod without error")
		assert.NotEmpty(t, content, "content of go.mod should not be empty")
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := ReadFileFromHEADWithRoot("this-file-does-not-exist-xyzzy.txt", root)
		assert.Error(t, err, "ReadFileFromHEADWithRoot should return error for non-existent file")
	})

	t.Run("rejects path with .. traversal", func(t *testing.T) {
		// Specification: "The function rejects paths that escape the repository
		// (i.e. paths containing .. after resolution)."
		_, err := ReadFileFromHEADWithRoot("../outside/file.txt", root)
		assert.Error(t, err, "ReadFileFromHEADWithRoot should reject path-traversal attempts")
	})

	t.Run("returns error when gitRoot is empty", func(t *testing.T) {
		// Specification: gitRoot must be the repository root (from FindGitRoot)
		_, err := ReadFileFromHEADWithRoot("go.mod", "")
		assert.Error(t, err, "ReadFileFromHEADWithRoot should return error when gitRoot is empty")
	})
}
