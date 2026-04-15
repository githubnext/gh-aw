//go:build !integration

package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApplyRepoOverrideFallback verifies the repository-override fallback logic
// used to avoid the git dependency in agent sandbox environments.
func TestApplyRepoOverrideFallback(t *testing.T) {
	t.Cleanup(func() {
		os.Unsetenv("GITHUB_REPOSITORY")
	})

	t.Run("explicit repoOverride is returned unchanged", func(t *testing.T) {
		os.Unsetenv("GITHUB_REPOSITORY")

		result := applyRepoOverrideFallback("explicit/repo")

		assert.Equal(t, "explicit/repo", result, "explicit repoOverride should be returned as-is")
	})

	t.Run("empty repoOverride falls back to GITHUB_REPOSITORY", func(t *testing.T) {
		require.NoError(t, os.Setenv("GITHUB_REPOSITORY", "owner/repo"), "should set GITHUB_REPOSITORY")

		result := applyRepoOverrideFallback("")

		assert.Equal(t, "owner/repo", result, "repoOverride should be populated from GITHUB_REPOSITORY")
	})

	t.Run("empty repoOverride with no GITHUB_REPOSITORY returns empty", func(t *testing.T) {
		os.Unsetenv("GITHUB_REPOSITORY")

		result := applyRepoOverrideFallback("")

		assert.Empty(t, result, "repoOverride should remain empty when GITHUB_REPOSITORY is not set")
	})
}
