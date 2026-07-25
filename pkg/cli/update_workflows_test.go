//go:build !integration

package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockWorkflowUpdateDeps(mockFn func(context.Context, string) ([]byte, error)) workflowUpdateDeps {
	deps := defaultWorkflowUpdateDeps()
	deps.runReleasesAPI = mockFn
	return deps
}

// TestResolveLatestRelease_PrereleaseTagsSkipped verifies that prerelease tags are
// not selected as the upgrade target even when they have a higher base version than
// the latest stable release. Per semver rules, v1.1.0-beta.1 > v1.0.0, so without
// explicit filtering a prerelease could be picked incorrectly.
func TestResolveLatestRelease_PrereleaseTagsSkipped(t *testing.T) {
	t.Parallel()

	deps := mockWorkflowUpdateDeps(func(_ context.Context, _ string) ([]byte, error) {
		return []byte("v1.1.0-beta.1\nv1.0.0"), nil
	})

	result, err := resolveLatestReleaseWithDeps(context.Background(), deps, "owner/repo", "v1.0.0", true, false, 0)
	require.NoError(t, err, "should not error when stable release exists")
	assert.Equal(t, "v1.0.0", result, "should select latest stable release, not prerelease")
}

// TestResolveLatestRelease_PrereleaseSkippedWhenCurrentVersionInvalid verifies that when
// the current version is not a valid semantic version, the highest stable release by
// semver is returned rather than the first item in the list (which could be a prerelease
// or an older release listed first by the API).
func TestResolveLatestRelease_PrereleaseSkippedWhenCurrentVersionInvalid(t *testing.T) {
	t.Parallel()

	deps := mockWorkflowUpdateDeps(func(_ context.Context, _ string) ([]byte, error) {
		// Prerelease appears first, and older stable release appears before newer one.
		return []byte("v2.0.0-rc.1\nv1.3.0\nv1.5.0"), nil
	})

	result, err := resolveLatestReleaseWithDeps(context.Background(), deps, "owner/repo", "not-a-version", true, false, 0)
	require.NoError(t, err, "should not error when stable release exists")
	assert.Equal(t, "v1.5.0", result, "should skip prerelease and return highest stable release by semver")
}

// TestResolveLatestRelease_ErrorWhenOnlyPrereleasesExist verifies that an error is
// returned when the releases list contains only prerelease versions.
func TestResolveLatestRelease_ErrorWhenOnlyPrereleasesExist(t *testing.T) {
	t.Parallel()

	deps := mockWorkflowUpdateDeps(func(_ context.Context, _ string) ([]byte, error) {
		return []byte("v2.0.0-beta.1\nv1.0.0-rc.1"), nil
	})

	_, err := resolveLatestReleaseWithDeps(context.Background(), deps, "owner/repo", "v1.0.0", true, false, 0)
	assert.Error(t, err, "should error when no stable releases exist")
}

// TestResolveLatestRelease_StableReleaseSelected verifies that stable releases are
// correctly selected when there are no prereleases.
func TestResolveLatestRelease_StableReleaseSelected(t *testing.T) {
	t.Parallel()

	deps := mockWorkflowUpdateDeps(func(_ context.Context, _ string) ([]byte, error) {
		return []byte("v1.2.0\nv1.1.0\nv1.0.0"), nil
	})

	result, err := resolveLatestReleaseWithDeps(context.Background(), deps, "owner/repo", "v1.0.0", false, false, 0)
	require.NoError(t, err, "should not error when stable releases exist")
	assert.Equal(t, "v1.2.0", result, "should select highest compatible stable release")
}

// TestResolveLatestRelease_MixedPrereleaseAndStable verifies correct selection when
// releases include both prerelease and stable versions across major versions.
func TestResolveLatestRelease_MixedPrereleaseAndStable(t *testing.T) {
	t.Parallel()

	deps := mockWorkflowUpdateDeps(func(_ context.Context, _ string) ([]byte, error) {
		return []byte("v2.0.0-alpha.1\nv1.3.0\nv1.2.0-rc.1\nv1.1.0"), nil
	})

	// Without allowMajor, should stay on v1.x and skip prereleases.
	result, err := resolveLatestReleaseWithDeps(context.Background(), deps, "owner/repo", "v1.1.0", false, false, 0)
	require.NoError(t, err, "should not error when stable v1.x releases exist")
	assert.Equal(t, "v1.3.0", result, "should select latest stable v1.x release, skipping prereleases")
}

// TestResolveLatestRelease_CooldownFallback verifies that when the newest upgrade
// candidate is within the cooldown window the resolver falls back to the next
// older release that has already passed the cooldown period.
func TestResolveLatestRelease_CooldownFallback(t *testing.T) {
	t.Parallel()

	deps := mockWorkflowUpdateDeps(func(_ context.Context, _ string) ([]byte, error) {
		// v1.3.0 is the newest; v1.2.0 is one step older and has cooled down.
		return []byte("v1.3.0\nv1.2.0\nv1.0.0"), nil
	})
	deps.checkCoolDown = func(_ context.Context, repo, tag string, cd time.Duration) coolDownCheckResult {
		switch tag {
		case "v1.3.0":
			// Too new: published 2 days ago.
			return checkReleaseCoolDownWithDate(repo, tag, time.Now().Add(-2*24*time.Hour), cd)
		case "v1.2.0":
			// Cooled down: published 10 days ago.
			return checkReleaseCoolDownWithDate(repo, tag, time.Now().Add(-10*24*time.Hour), cd)
		default:
			return coolDownCheckResult{}
		}
	}

	result, err := resolveLatestReleaseWithDeps(context.Background(), deps, "owner/repo", "v1.0.0", false, false, 7*24*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, "v1.2.0", result, "should fall back to v1.2.0 when v1.3.0 is in cooldown")
}

// TestResolveLatestRelease_CooldownAllInWindow returns currentRef when every
// upgrade candidate is still within the cooldown window.
func TestResolveLatestRelease_CooldownAllInWindow(t *testing.T) {
	t.Parallel()

	deps := mockWorkflowUpdateDeps(func(_ context.Context, _ string) ([]byte, error) {
		return []byte("v1.2.0\nv1.1.0\nv1.0.0"), nil
	})
	// All upgrade candidates are too new.
	deps.checkCoolDown = func(_ context.Context, repo, tag string, cd time.Duration) coolDownCheckResult {
		return checkReleaseCoolDownWithDate(repo, tag, time.Now().Add(-1*24*time.Hour), cd)
	}

	result, err := resolveLatestReleaseWithDeps(context.Background(), deps, "owner/repo", "v1.0.0", false, false, 7*24*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0", result, "should return currentRef when all upgrade candidates are in cooldown")
}
