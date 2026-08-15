//go:build !integration

package cli

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/semverutil"
)

func TestMajorVersionPreference(t *testing.T) {
	// Test that the version selection logic prefers major-only versions (v8)
	// over full semantic versions (v8.0.0) when they are semantically equal.
	// This follows GitHub Actions best practice of using major version tags.

	tests := []struct {
		name              string
		releases          []string
		currentVersion    string
		allowMajor        bool
		expectedVersion   string
		expectedPreferred string // The version that should be preferred (v8 over v8.0.0.0)
	}{
		{
			name:              "prefer v8 over v8.0.0",
			releases:          []string{"v8.0.0", "v8", "v7.0.0"},
			currentVersion:    "v8",
			allowMajor:        false,
			expectedVersion:   "v8",
			expectedPreferred: "v8",
		},
		{
			name:              "prefer v6 over v6.0.0",
			releases:          []string{"v6.0.0", "v6", "v5.0.0"},
			currentVersion:    "v6",
			allowMajor:        false,
			expectedVersion:   "v6",
			expectedPreferred: "v6",
		},
		{
			name:              "prefer v8 over v8.0.0.0 (four-part version)",
			releases:          []string{"v8.0.0.0", "v8"},
			currentVersion:    "v8",
			allowMajor:        false,
			expectedVersion:   "v8",
			expectedPreferred: "v8",
		},
		{
			name:              "prefer newest when versions differ",
			releases:          []string{"v8.1.0", "v8.0.0", "v8"},
			currentVersion:    "v8",
			allowMajor:        false,
			expectedVersion:   "v8.1.0",
			expectedPreferred: "v8.1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentVer := parseVersion(tt.currentVersion)
			if currentVer == nil {
				t.Fatalf("Failed to parse current version: %s", tt.currentVersion)
			}

			var latestCompatible string
			var latestCompatibleVersion *semverutil.SemanticVersion

			for _, release := range tt.releases {
				releaseVer := parseVersion(release)
				if releaseVer == nil {
					continue
				}

				// Check if compatible based on major version
				if !tt.allowMajor && releaseVer.Major != currentVer.Major {
					continue
				}

				// Check if this is newer than what we have
				if latestCompatibleVersion == nil || releaseVer.IsNewer(latestCompatibleVersion) {
					latestCompatible = release
					latestCompatibleVersion = releaseVer
				} else if !releaseVer.IsNewer(latestCompatibleVersion) &&
					releaseVer.Major == latestCompatibleVersion.Major &&
					releaseVer.Minor == latestCompatibleVersion.Minor &&
					releaseVer.Patch == latestCompatibleVersion.Patch {
					// If versions are equal, prefer the less precise one (e.g., "v8" over "v8.0.0")
					if !releaseVer.IsPreciseVersion() && latestCompatibleVersion.IsPreciseVersion() {
						latestCompatible = release
						latestCompatibleVersion = releaseVer
					}
				}
			}

			if latestCompatible != tt.expectedVersion {
				t.Errorf("Selected version = %q, want %q", latestCompatible, tt.expectedVersion)
			}

			// Verify that the selected version is the preferred one (less precise when equal)
			if latestCompatible != tt.expectedPreferred {
				t.Errorf("Preferred version = %q, want %q (should prefer less precise version)", latestCompatible, tt.expectedPreferred)
			}
		})
	}
}
func TestGetLatestActionRelease_FallsBackToGitWhenNoReleases(t *testing.T) {
	deps := newTestActionUpdateDeps()

	// Simulate the GitHub Releases API returning an empty list (no releases published).
	deps.runGHReleasesAPI = func(_ context.Context, baseRepo string) ([]byte, error) {
		return []byte(""), nil
	}

	gitFnCalled := false
	deps.getLatestReleaseViaGit = func(_ context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
		gitFnCalled = true
		return "v1.2.3", "abc1234567890123456789012345678901234567", nil
	}

	version, sha, err := getLatestActionReleaseWithDeps(context.Background(), deps, "github/gh-aw-actions/setup", "v1", false, false)
	if err != nil {
		t.Fatalf("expected no error when git fallback succeeds, got: %v", err)
	}
	if version != "v1.2.3" {
		t.Errorf("version = %q, want %q", version, "v1.2.3")
	}
	if sha != "abc1234567890123456789012345678901234567" {
		t.Errorf("sha = %q, want %q", sha, "abc1234567890123456789012345678901234567")
	}
	if !gitFnCalled {
		t.Error("expected getLatestActionReleaseViaGitFn to be called as fallback, but it was not")
	}
}
func TestParseActionTagRefs_PrefersPeeledCommitSHA(t *testing.T) {
	output := strings.Join([]string{
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa refs/tags/v1",
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb refs/tags/v1^{}",
		"cccccccccccccccccccccccccccccccccccccccc refs/tags/v2",
	}, "\n")

	releases, tagToSHA := parseActionTagRefs(output)

	if len(releases) != 2 || releases[0] != "v1" || releases[1] != "v2" {
		t.Fatalf("releases = %v, want [v1 v2]", releases)
	}
	if got := tagToSHA["v1"]; got != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Errorf("v1 SHA = %q, want peeled commit SHA", got)
	}
	if got := tagToSHA["v2"]; got != "cccccccccccccccccccccccccccccccccccccccc" {
		t.Errorf("v2 SHA = %q, want lightweight tag SHA", got)
	}
}

// TestGetLatestActionRelease_FallbackReturnsErrorWhenBothFail verifies that when the
// GitHub Releases API returns an empty list and the git fallback also fails, the
// function returns an error rather than silently succeeding.
func TestGetLatestActionRelease_FallbackReturnsErrorWhenBothFail(t *testing.T) {
	deps := newTestActionUpdateDeps()

	// Simulate the GitHub Releases API returning an empty list.
	deps.runGHReleasesAPI = func(_ context.Context, baseRepo string) ([]byte, error) {
		return []byte(""), nil
	}

	// Simulate the git fallback also finding nothing.
	deps.getLatestReleaseViaGit = func(_ context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
		return "", "", errors.New("no releases found")
	}

	_, _, err := getLatestActionReleaseWithDeps(context.Background(), deps, "github/gh-aw-actions/setup", "v1", false, false)
	if err == nil {
		t.Fatal("expected error when both releases API and git fallback fail, got nil")
	}
}

// TestGetLatestActionRelease_PrereleaseTagsSkipped verifies that prerelease tags are
// not selected as the upgrade target even when they have a higher base version than
// the latest stable release.  Per semver rules, v1.1.0-beta.1 > v1.0.0 (base version
// comparison), so without explicit filtering a prerelease could be picked incorrectly.
func TestGetLatestActionRelease_PrereleaseTagsSkipped(t *testing.T) {
	deps := newTestActionUpdateDeps()

	// Return a stable release alongside a higher-versioned prerelease.
	deps.runGHReleasesAPI = func(_ context.Context, baseRepo string) ([]byte, error) {
		return []byte("v1.0.0\nv1.1.0-beta.1"), nil
	}

	deps.getActionSHAForTag = func(_ context.Context, repo, tag string) (string, error) {
		return "stablesha1234567890123456789012345678901", nil
	}

	version, _, err := getLatestActionReleaseWithDeps(context.Background(), deps, "actions/checkout", "v1.0.0", true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "v1.0.0" {
		t.Errorf("version = %q, want %q (prerelease v1.1.0-beta.1 should be skipped)", version, "v1.0.0")
	}
}

// TestFindCooledDownActionVersion_SelectsOlderCooledRelease verifies that when the
// latest release is still within the cooldown window, findCooledDownActionVersion
// falls back to the newest older release that has passed the cooldown period.
func TestFindCooledDownActionVersion_SelectsOlderCooledRelease(t *testing.T) {
	deps := newTestActionUpdateDeps()

	// v1.321.0 is the newest (in cooldown); v1.320.0 has cooled down.
	deps.runGHReleasesAPI = func(_ context.Context, _ string) ([]byte, error) {
		return []byte("v1.321.0\nv1.320.0\nv1.300.0"), nil
	}
	deps.checkCoolDown = func(_ context.Context, repo, tag string, cd time.Duration) coolDownCheckResult {
		switch tag {
		case "v1.321.0":
			return checkReleaseCoolDownWithDate(repo, tag, time.Now().Add(-1*24*time.Hour), cd)
		default:
			return coolDownCheckResult{}
		}
	}
	const wantSHA = "cooled1234567890123456789012345678901234"
	deps.getActionSHAForTag = func(_ context.Context, _, tag string) (string, error) {
		if tag == "v1.320.0" {
			return wantSHA, nil
		}
		return "", errors.New("unexpected tag: " + tag)
	}

	version, sha, err := findCooledDownActionVersion(context.Background(), deps, "ruby/setup-ruby", "v1.300.0", true, false, 7*24*time.Hour, "")
	if err != nil {
		t.Fatalf("findCooledDownActionVersion() error = %v", err)
	}
	if version != "v1.320.0" {
		t.Errorf("version = %q, want %q", version, "v1.320.0")
	}
	if sha != wantSHA {
		t.Errorf("sha = %q, want %q", sha, wantSHA)
	}
}

// TestFindCooledDownActionVersion_AllInCooldownReturnsEmpty verifies that
// findCooledDownActionVersion returns empty strings when all candidate releases
// are still within the cooldown window.
func TestFindCooledDownActionVersion_AllInCooldownReturnsEmpty(t *testing.T) {
	deps := newTestActionUpdateDeps()
	deps.runGHReleasesAPI = func(_ context.Context, _ string) ([]byte, error) {
		return []byte("v1.321.0\nv1.320.0"), nil
	}
	// All releases too new.
	deps.checkCoolDown = func(_ context.Context, repo, tag string, cd time.Duration) coolDownCheckResult {
		return checkReleaseCoolDownWithDate(repo, tag, time.Now().Add(-1*24*time.Hour), cd)
	}
	deps.getActionSHAForTag = func(_ context.Context, _, _ string) (string, error) {
		return "somesha1234567890123456789012345678901234", nil
	}

	version, sha, err := findCooledDownActionVersion(context.Background(), deps, "ruby/setup-ruby", "v1.319.0", true, false, 7*24*time.Hour, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "" || sha != "" {
		t.Errorf("expected empty result when all releases in cooldown, got version=%q sha=%q", version, sha)
	}
}

// TestFindCooledDownActionVersion_EmptySHASkipped verifies that a release whose SHA
// resolves to an empty string is silently skipped (never returned as a valid candidate).
func TestFindCooledDownActionVersion_EmptySHASkipped(t *testing.T) {
	deps := newTestActionUpdateDeps()
	deps.runGHReleasesAPI = func(_ context.Context, _ string) ([]byte, error) {
		return []byte("v1.321.0\nv1.320.0"), nil
	}
	deps.checkCoolDown = func(_ context.Context, _ string, _ string, _ time.Duration) coolDownCheckResult {
		return coolDownCheckResult{} // not in cooldown
	}
	// SHA API returns empty for all tags.
	deps.getActionSHAForTag = func(_ context.Context, _, _ string) (string, error) {
		return "", nil // empty SHA
	}

	version, sha, err := findCooledDownActionVersion(context.Background(), deps, "docker/login-action", "v1.319.0", true, false, 7*24*time.Hour, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "" || sha != "" {
		t.Errorf("expected empty result when SHA is empty, got version=%q sha=%q", version, sha)
	}
}

// TestUpdateActions_CooldownFallbackToOlderRelease verifies that updateActions
// falls back to an older cooled-down release instead of skipping the update
// entirely when the newest candidate is still in cooldown.
func TestFindCooledDownActionVersion_SHAErrorFallsToNext(t *testing.T) {
	deps := newTestActionUpdateDeps()
	deps.runGHReleasesAPI = func(_ context.Context, _ string) ([]byte, error) {
		return []byte("v1.322.0\nv1.321.0\nv1.320.0"), nil
	}
	deps.checkCoolDown = func(_ context.Context, _, _ string, _ time.Duration) coolDownCheckResult {
		return coolDownCheckResult{} // all candidates cooled down
	}
	deps.getActionSHAForTag = func(_ context.Context, _, tag string) (string, error) {
		if tag == "v1.322.0" {
			return "", errors.New("not found")
		}
		if tag == "v1.321.0" {
			return "sha321_12345678901234567890123456789012", nil
		}
		return "", errors.New("unexpected tag: " + tag)
	}

	version, sha, err := findCooledDownActionVersion(context.Background(), deps, "ruby/setup-ruby", "v1.320.0", true, false, 7*24*time.Hour, "")
	if err != nil {
		t.Fatalf("findCooledDownActionVersion() error = %v", err)
	}
	if version != "v1.321.0" {
		t.Errorf("version = %q, want %q", version, "v1.321.0")
	}
	if sha != "sha321_12345678901234567890123456789012" {
		t.Errorf("sha = %q, want %q", sha, "sha321_12345678901234567890123456789012")
	}
}
