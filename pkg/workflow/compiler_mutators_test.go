//go:build !integration

package workflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompilerMutators(t *testing.T) {
	c := NewCompiler()

	c.SetSkipValidation(false)
	assert.False(t, c.skipValidation)

	ctx := context.WithValue(context.Background(), struct{ k string }{"k"}, "v")
	c.SetContext(ctx)
	assert.Equal(t, ctx, c.ctx)

	c.SetNoEmit(true)
	assert.True(t, c.noEmit)

	c.SetStrictMode(true)
	assert.True(t, c.strictMode)

	c.SetActionTag("v1")
	assert.Equal(t, "v1", c.GetActionTag())

	c.SetActionMode(ActionModeRelease)
	assert.Equal(t, ActionModeRelease, c.GetActionMode())

	c.IncrementWarningCount()
	assert.Equal(t, 1, c.GetWarningCount())
	c.ResetWarningCount()
	assert.Equal(t, 0, c.GetWarningCount())

	c.AddSafeUpdateWarning("warning")
	assert.Equal(t, []string{"warning"}, c.GetSafeUpdateWarnings())
}

func TestCompilerRepositorySlugLocking(t *testing.T) {
	c := NewCompiler()

	c.SetRepositorySlug("owner/repo")
	assert.Equal(t, "owner/repo", c.GetRepositorySlug())
	assert.False(t, c.IsRepositorySlugLocked())

	c.SetRepositorySlugIfUnlocked("other/repo")
	assert.Equal(t, "other/repo", c.GetRepositorySlug())

	c.LockRepositorySlug()
	assert.True(t, c.IsRepositorySlugLocked())

	c.SetRepositorySlugIfUnlocked("ignored/repo")
	assert.Equal(t, "other/repo", c.GetRepositorySlug())
}

func TestCompilerSharedActionCacheAndResolver(t *testing.T) {
	c := NewCompiler()

	cache := c.GetSharedActionCache()
	require.NotNil(t, cache)
	resolver := c.GetSharedActionResolver()
	require.NotNil(t, resolver)

	// Subsequent calls reuse the same shared instances
	assert.Same(t, cache, c.GetSharedActionCache())
	assert.Same(t, resolver, c.GetSharedActionResolver())
}

func TestCompilerSetPriorManifestsNilResetsMap(t *testing.T) {
	c := NewCompiler()

	c.SetPriorManifests(nil)
	assert.NotNil(t, c.priorManifests)
	assert.Empty(t, c.priorManifests)
}
