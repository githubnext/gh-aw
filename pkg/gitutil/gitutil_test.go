//go:build !integration

package gitutil

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{
			name:     "GitHub API rate limit exceeded (HTTP 403)",
			errMsg:   "gh: API rate limit exceeded for installation. If you reach out to GitHub Support for help, please include the request ID (HTTP 403)",
			expected: true,
		},
		{
			name:     "rate limit exceeded lowercase",
			errMsg:   "rate limit exceeded",
			expected: true,
		},
		{
			name:     "HTTP 403 with API rate limit message",
			errMsg:   "HTTP 403: API rate limit exceeded for installation.",
			expected: true,
		},
		{
			name:     "secondary rate limit in GitHub error message",
			errMsg:   "gh: You have exceeded a secondary rate limit",
			expected: true,
		},
		{
			name:     "authentication error is not a rate limit error",
			errMsg:   "authentication required. Run 'gh auth login' first",
			expected: false,
		},
		{
			name:     "not found error is not a rate limit error",
			errMsg:   "HTTP 404: Not Found",
			expected: false,
		},
		{
			name:     "empty string",
			errMsg:   "",
			expected: false,
		},
		{
			name:     "unrelated error message",
			errMsg:   "failed to parse workflow runs: unexpected end of JSON input",
			expected: false,
		},
		{
			name:     "mixed case",
			errMsg:   "API Rate Limit Exceeded for installation",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRateLimitError(tt.errMsg)
			assert.Equal(t, tt.expected, result, "IsRateLimitError(%q) should return %v", tt.errMsg, tt.expected)
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{
			name:     "GH_TOKEN mention",
			errMsg:   "GH_TOKEN is not set",
			expected: true,
		},
		{
			name:     "GITHUB_TOKEN mention",
			errMsg:   "GITHUB_TOKEN is missing or invalid",
			expected: true,
		},
		{
			name:     "authentication error",
			errMsg:   "authentication required",
			expected: true,
		},
		{
			name:     "not logged in",
			errMsg:   "not logged into any GitHub hosts",
			expected: true,
		},
		{
			name:     "unauthorized",
			errMsg:   "HTTP 401: Unauthorized",
			expected: true,
		},
		{
			name:     "forbidden",
			errMsg:   "HTTP 403: Forbidden",
			expected: true,
		},
		{
			name:     "permission denied",
			errMsg:   "permission denied: insufficient scope",
			expected: true,
		},
		{
			name:     "saml enforcement",
			errMsg:   "Resource protected by organization SAML enforcement",
			expected: true,
		},
		{
			name:     "rate limit error is not an auth error",
			errMsg:   "API rate limit exceeded for installation",
			expected: false,
		},
		{
			name:     "empty string",
			errMsg:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthError(tt.errMsg)
			assert.Equal(t, tt.expected, result, "IsAuthError(%q) should return %v", tt.errMsg, tt.expected)
		})
	}
}

func TestIsHexString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid lowercase hex",
			input:    "deadbeef",
			expected: true,
		},
		{
			name:     "valid uppercase hex",
			input:    "DEADBEEF",
			expected: true,
		},
		{
			name:     "valid mixed case hex",
			input:    "DeAdBeEf",
			expected: true,
		},
		{
			name:     "valid full git sha",
			input:    "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			expected: true,
		},
		{
			name:     "digits only",
			input:    "0123456789",
			expected: true,
		},
		{
			name:     "single valid char",
			input:    "a",
			expected: true,
		},
		{
			name:     "invalid char g",
			input:    "deadbeeg",
			expected: false,
		},
		{
			name:     "contains space",
			input:    "dead beef",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "non-hex word",
			input:    "xyz",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHexString(tt.input)
			assert.Equal(t, tt.expected, result, "IsHexString(%q) should return %v", tt.input, tt.expected)
		})
	}
}

func TestIsValidFullSHA(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid lowercase full SHA",
			input:    "abcdef0123456789abcdef0123456789abcdef01",
			expected: true,
		},
		{
			name:     "invalid uppercase full SHA",
			input:    "ABCDEF0123456789ABCDEF0123456789ABCDEF01",
			expected: false,
		},
		{
			name:     "invalid short SHA",
			input:    "abcdef0",
			expected: false,
		},
		{
			name:     "invalid non-hex character",
			input:    "abcdef0123456789abcdef0123456789abcdef0g",
			expected: false,
		},
		{
			name:     "invalid empty SHA",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidFullSHA(tt.input)
			assert.Equal(t, tt.expected, result, "IsValidFullSHA(%q) should return %v", tt.input, tt.expected)
		})
	}
}

func TestExtractBaseRepo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple owner/repo path",
			input:    "actions/checkout",
			expected: "actions/checkout",
		},
		{
			name:     "path with one subpath segment",
			input:    "github/codeql-action/upload-sarif",
			expected: "github/codeql-action",
		},
		{
			name:     "deep path with multiple segments",
			input:    "owner/repo/sub/dir/file",
			expected: "owner/repo",
		},
		{
			name:     "no slash returns input as-is",
			input:    "onlyone",
			expected: "onlyone",
		},
		{
			name:     "empty string returns empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractBaseRepo(tt.input)
			assert.Equal(t, tt.expected, result, "ExtractBaseRepo(%q) should return %q", tt.input, tt.expected)
		})
	}
}

func TestFindGitRoot(t *testing.T) {
	t.Run("returns non-empty path when inside a git repository", func(t *testing.T) {
		gitRoot, err := FindGitRoot()
		require.NoError(t, err, "FindGitRoot should succeed when running inside a git repository")
		assert.NotEmpty(t, gitRoot, "FindGitRoot should return a non-empty path")
	})
}

func TestReadFileFromHEADWithRoot(t *testing.T) {
	t.Run("reads a committed file with pre-computed root", func(t *testing.T) {
		gitRoot, err := FindGitRoot()
		require.NoError(t, err, "must be inside a git repository")

		content, err := ReadFileFromHEADWithRoot(filepath.Join(gitRoot, "go.mod"), gitRoot)
		require.NoError(t, err, "go.mod should be readable from HEAD with pre-computed root")
		assert.NotEmpty(t, content, "go.mod content should not be empty")
		assert.Contains(t, content, "module ", "go.mod should contain a module declaration")
	})

	t.Run("returns error for path outside git root", func(t *testing.T) {
		gitRoot, err := FindGitRoot()
		require.NoError(t, err, "must be inside a git repository")

		outsidePath := filepath.Join(t.TempDir(), "file.yml")
		_, err = ReadFileFromHEADWithRoot(outsidePath, gitRoot)
		require.Error(t, err, "should fail for a file outside the git root")
		assert.Contains(t, err.Error(), "outside the git repository root", "error should mention path is outside repo")
	})

	t.Run("returns error for empty gitRoot", func(t *testing.T) {
		_, err := ReadFileFromHEADWithRoot("some/file.yml", "")
		require.Error(t, err, "should fail when gitRoot is empty")
		assert.Contains(t, err.Error(), "gitRoot must not be empty", "error should mention empty gitRoot")
	})
}
