//go:build !integration

package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockWorkflowReleasesAPI stubs runWorkflowReleasesAPIFn for the duration of a test.
func mockWorkflowReleasesAPI(t *testing.T, mockFn func(context.Context, string) ([]byte, error)) {
	t.Helper()
	orig := runWorkflowReleasesAPIFn
	t.Cleanup(func() { runWorkflowReleasesAPIFn = orig })
	runWorkflowReleasesAPIFn = mockFn
}

// TestResolveLatestRelease_PrereleaseTagsSkipped verifies that prerelease tags are
// not selected as the upgrade target even when they have a higher base version than
// the latest stable release. Per semver rules, v1.1.0-beta.1 > v1.0.0, so without
// explicit filtering a prerelease could be picked incorrectly.
func TestResolveLatestRelease_PrereleaseTagsSkipped(t *testing.T) {
	mockWorkflowReleasesAPI(t, func(_ context.Context, _ string) ([]byte, error) {
		return []byte("v1.1.0-beta.1\nv1.0.0"), nil
	})

	result, err := resolveLatestRelease(context.Background(), "owner/repo", "v1.0.0", true, false)
	require.NoError(t, err, "should not error when stable release exists")
	assert.Equal(t, "v1.0.0", result, "should select latest stable release, not prerelease")
}

// TestResolveLatestRelease_PrereleaseSkippedWhenCurrentVersionInvalid verifies that when
// the current version is not a valid semantic version, the highest stable release by
// semver is returned rather than the first item in the list (which could be a prerelease
// or an older release listed first by the API).
func TestResolveLatestRelease_PrereleaseSkippedWhenCurrentVersionInvalid(t *testing.T) {
	mockWorkflowReleasesAPI(t, func(_ context.Context, _ string) ([]byte, error) {
		// Prerelease appears first, and older stable release appears before newer one.
		return []byte("v2.0.0-rc.1\nv1.3.0\nv1.5.0"), nil
	})

	result, err := resolveLatestRelease(context.Background(), "owner/repo", "not-a-version", true, false)
	require.NoError(t, err, "should not error when stable release exists")
	assert.Equal(t, "v1.5.0", result, "should skip prerelease and return highest stable release by semver")
}

// TestResolveLatestRelease_ErrorWhenOnlyPrereleasesExist verifies that an error is
// returned when the releases list contains only prerelease versions.
func TestResolveLatestRelease_ErrorWhenOnlyPrereleasesExist(t *testing.T) {
	mockWorkflowReleasesAPI(t, func(_ context.Context, _ string) ([]byte, error) {
		return []byte("v2.0.0-beta.1\nv1.0.0-rc.1"), nil
	})

	_, err := resolveLatestRelease(context.Background(), "owner/repo", "v1.0.0", true, false)
	assert.Error(t, err, "should error when no stable releases exist")
}

// TestResolveLatestRelease_StableReleaseSelected verifies that stable releases are
// correctly selected when there are no prereleases.
func TestResolveLatestRelease_StableReleaseSelected(t *testing.T) {
	mockWorkflowReleasesAPI(t, func(_ context.Context, _ string) ([]byte, error) {
		return []byte("v1.2.0\nv1.1.0\nv1.0.0"), nil
	})

	result, err := resolveLatestRelease(context.Background(), "owner/repo", "v1.0.0", false, false)
	require.NoError(t, err, "should not error when stable releases exist")
	assert.Equal(t, "v1.2.0", result, "should select highest compatible stable release")
}

// TestResolveLatestRelease_MixedPrereleaseAndStable verifies correct selection when
// releases include both prerelease and stable versions across major versions.
func TestResolveLatestRelease_MixedPrereleaseAndStable(t *testing.T) {
	mockWorkflowReleasesAPI(t, func(_ context.Context, _ string) ([]byte, error) {
		return []byte("v2.0.0-alpha.1\nv1.3.0\nv1.2.0-rc.1\nv1.1.0"), nil
	})

	// Without allowMajor, should stay on v1.x and skip prereleases.
	result, err := resolveLatestRelease(context.Background(), "owner/repo", "v1.1.0", false, false)
	require.NoError(t, err, "should not error when stable v1.x releases exist")
	assert.Equal(t, "v1.3.0", result, "should select latest stable v1.x release, skipping prereleases")
}
