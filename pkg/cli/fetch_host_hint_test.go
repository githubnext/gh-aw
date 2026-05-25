//go:build !integration

package cli

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostResolutionHintForNotFound(t *testing.T) {
	t.Run("suggests full github URL for GHE shorthand 404", func(t *testing.T) {
		t.Setenv("GITHUB_SERVER_URL", "")
		t.Setenv("GITHUB_ENTERPRISE_HOST", "ghe.example.com")
		t.Setenv("GITHUB_HOST", "")
		t.Setenv("GH_HOST", "")

		hint, ok := hostResolutionHintForNotFound(
			"githubnext", "agentics", "main", "workflows/daily-repo-status.md", "", errors.New("gh: Not Found (HTTP 404)"),
		)

		require.True(t, ok, "expected hint for GHE shorthand 404")
		assert.Contains(t, hint, "resolved on ghe.example.com", "hint should mention resolved host")
		assert.Contains(t, hint, "https://github.com/githubnext/agentics/blob/main/workflows/daily-repo-status.md", "hint should include full github.com workflow URL")
	})

	t.Run("does not suggest for github.com host", func(t *testing.T) {
		hint, ok := hostResolutionHintForNotFound(
			"githubnext", "agentics", "main", "workflows/daily-repo-status.md", "https://github.com", errors.New("gh: Not Found (HTTP 404)"),
		)
		assert.False(t, ok, "expected no hint for github.com host")
		assert.Empty(t, hint, "expected empty hint for github.com host")
	})

	t.Run("does not suggest for explicit non-github host URL", func(t *testing.T) {
		hint, ok := hostResolutionHintForNotFound(
			"githubnext", "agentics", "main", "workflows/daily-repo-status.md", "https://ghe.example.com/org/repo", errors.New("gh: Not Found (HTTP 404)"),
		)
		assert.False(t, ok, "expected no hint for explicit host")
		assert.Empty(t, hint, "expected empty hint for explicit host")
	})

	t.Run("does not suggest for nil or non-404 errors", func(t *testing.T) {
		hint, ok := hostResolutionHintForNotFound(
			"githubnext", "agentics", "main", "workflows/daily-repo-status.md", "", nil,
		)
		assert.False(t, ok, "expected no hint for nil error")
		assert.Empty(t, hint, "expected empty hint for nil error")

		t.Setenv("GITHUB_SERVER_URL", "")
		t.Setenv("GITHUB_ENTERPRISE_HOST", "ghe.example.com")
		t.Setenv("GITHUB_HOST", "")
		t.Setenv("GH_HOST", "")
		hint, ok = hostResolutionHintForNotFound(
			"githubnext", "agentics", "main", "/workflows/daily-repo-status.md", "", errors.New("gh: forbidden (HTTP 403)"),
		)
		assert.False(t, ok, "expected no hint for non-404 error")
		assert.Empty(t, hint, "expected empty hint for non-404 error")
	})

	t.Run("normalizes leading slash in workflow path", func(t *testing.T) {
		t.Setenv("GITHUB_SERVER_URL", "")
		t.Setenv("GITHUB_ENTERPRISE_HOST", "ghe.example.com")
		t.Setenv("GITHUB_HOST", "")
		t.Setenv("GH_HOST", "")

		hint, ok := hostResolutionHintForNotFound(
			"githubnext", "agentics", "main", "/workflows/daily-repo-status.md", "", errors.New("HTTP 404"),
		)
		require.True(t, ok, "expected hint for 404 with normalized path")
		assert.NotContains(t, hint, "blob/main//workflows", "expected normalized path without double slash")
	})
}
