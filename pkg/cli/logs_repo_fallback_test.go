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
	// Ensure the relevant env vars are clean before each sub-test.
	t.Cleanup(func() {
		os.Unsetenv("GITHUB_REPOSITORY")
		os.Unsetenv("GH_REPO")
	})

	t.Run("explicit repoOverride is returned unchanged", func(t *testing.T) {
		os.Unsetenv("GITHUB_REPOSITORY")
		os.Unsetenv("GH_REPO")

		result, cleanup := applyRepoOverrideFallback("explicit/repo")
		defer cleanup()

		assert.Equal(t, "explicit/repo", result, "explicit repoOverride should be returned as-is")
		_, ghRepoSet := os.LookupEnv("GH_REPO")
		assert.False(t, ghRepoSet, "GH_REPO should not be set when repoOverride is already explicit")
	})

	t.Run("empty repoOverride falls back to GITHUB_REPOSITORY", func(t *testing.T) {
		require.NoError(t, os.Setenv("GITHUB_REPOSITORY", "owner/repo"), "should set GITHUB_REPOSITORY")
		os.Unsetenv("GH_REPO")

		result, cleanup := applyRepoOverrideFallback("")
		defer cleanup()

		assert.Equal(t, "owner/repo", result, "repoOverride should be populated from GITHUB_REPOSITORY")
		assert.Equal(t, "owner/repo", os.Getenv("GH_REPO"), "GH_REPO should be set to GITHUB_REPOSITORY value")
		cleanup()
		_, ghRepoSet := os.LookupEnv("GH_REPO")
		assert.False(t, ghRepoSet, "cleanup should unset GH_REPO")
	})

	t.Run("pre-existing GH_REPO is respected and not overwritten", func(t *testing.T) {
		require.NoError(t, os.Setenv("GITHUB_REPOSITORY", "owner/repo"), "should set GITHUB_REPOSITORY")
		require.NoError(t, os.Setenv("GH_REPO", "existing/repo"), "should set GH_REPO")

		result, cleanup := applyRepoOverrideFallback("")
		defer cleanup()

		assert.Equal(t, "owner/repo", result, "repoOverride should still be populated from GITHUB_REPOSITORY")
		assert.Equal(t, "existing/repo", os.Getenv("GH_REPO"), "pre-existing GH_REPO should not be overwritten")
		cleanup()
		assert.Equal(t, "existing/repo", os.Getenv("GH_REPO"), "cleanup should not remove a pre-existing GH_REPO")
	})

	t.Run("empty repoOverride with no GITHUB_REPOSITORY returns empty", func(t *testing.T) {
		os.Unsetenv("GITHUB_REPOSITORY")
		os.Unsetenv("GH_REPO")

		result, cleanup := applyRepoOverrideFallback("")
		defer cleanup()

		assert.Empty(t, result, "repoOverride should remain empty when GITHUB_REPOSITORY is not set")
		_, ghRepoSet := os.LookupEnv("GH_REPO")
		assert.False(t, ghRepoSet, "GH_REPO should not be set when GITHUB_REPOSITORY is absent")
	})
}
