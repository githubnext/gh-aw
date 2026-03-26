//go:build !integration

package gitutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
