//go:build !integration

package cli

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/semverutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"
)

func TestActionKeyVersionConsistency(t *testing.T) {
	// This test ensures that when an action is updated, the key in the map
	// is updated to match the new version, preventing key/version mismatches
	// that would cause version comments to change on each build.

	// Simulate the actions-lock.json structure using ActionCache
	tmpDir := testutil.TempDir(t, "test-*")
	cache := workflow.NewActionCache(tmpDir)
	cache.Set("actions/checkout", "v5.0.0", "oldsha1234567890123456789012345678901234")

	// Simulate an update to a newer version
	oldVersion := "v5.0.0"
	latestVersion := "v5.0.1"
	latestSHA := "newsha1234567890123456789012345678901234"

	// Apply the update logic from UpdateActions: delete old key, set new entry
	cache.Delete("actions/checkout", oldVersion)
	cache.Set("actions/checkout", latestVersion, latestSHA)

	oldKey := "actions/checkout@v5.0.0"
	newKey := "actions/checkout@v5.0.1"

	// Verify the old key is gone
	if _, exists := cache.Entries[oldKey]; exists {
		t.Errorf("Old key %q should have been deleted", oldKey)
	}

	// Verify the new key exists
	updatedEntry, exists := cache.Entries[newKey]
	if !exists {
		t.Errorf("New key %q should exist", newKey)
	}

	// Verify the entry has the correct version
	if updatedEntry.Version != latestVersion {
		t.Errorf("Entry version = %q, want %q", updatedEntry.Version, latestVersion)
	}

	// Most importantly: verify key and version field match
	keyVersion := newKey[len("actions/checkout@"):]
	if keyVersion != updatedEntry.Version {
		t.Errorf("Key version %q does not match entry version %q", keyVersion, updatedEntry.Version)
	}
}

func TestActionKeyVersionConsistencyInJSON(t *testing.T) {
	// This test ensures that when actions-lock.json is saved to disk and reloaded,
	// there are no key/version mismatches between the map key and the entry's Version field.

	tmpDir := testutil.TempDir(t, "test-*")
	cache := workflow.NewActionCache(tmpDir)
	cache.Set("actions/checkout", "v5.0.1", "93cb6efe18208431cddfb8368fd83d5badbf9bfd")
	cache.Set("actions/setup-node", "v6.1.0", "395ad3262231945c25e8478fd5baf05154b1d79f")

	// Save to disk and reload to exercise the JSON round-trip.
	if err := cache.Save(); err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}
	reloaded := workflow.NewActionCache(tmpDir)
	if err := reloaded.Load(); err != nil {
		t.Fatalf("Failed to reload cache: %v", err)
	}

	// Verify all entries have matching key and version after a round-trip.
	for key, entry := range reloaded.Entries {
		// Extract version from key (format: "repo@version")
		atIndex := len(key)
		for i := len(key) - 1; i >= 0; i-- {
			if key[i] == '@' {
				atIndex = i
				break
			}
		}

		if atIndex < len(key) {
			keyVersion := key[atIndex+1:]
			if keyVersion != entry.Version {
				t.Errorf("Key %q has version in key %q but entry version is %q - this mismatch causes version comments to change on each build",
					key, keyVersion, entry.Version)
			}
		}
	}
}

// TestUpdateActions_SafeOutputsInputsPreserved verifies that cached inputs and descriptions
// for safe-outputs.actions entries are preserved in actions-lock.json when other (unrelated)
// actions are updated. Previously, actionsLockEntry lacked Inputs/ActionDescription fields,
// causing them to be silently dropped whenever the file was rewritten.
func TestUpdateActions_SafeOutputsInputsPreserved(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	// Stub the release-fetch function so no network calls are made.
	// actions/checkout gets a bump; owner/my-safe-action is already at latest.
	orig := getLatestActionReleaseFn
	defer func() { getLatestActionReleaseFn = orig }()
	getLatestActionReleaseFn = func(_ context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
		switch repo {
		case "actions/checkout":
			return "v5", "newcheckoutsha1234567890123456789012345", nil
		case "owner/my-safe-action":
			// Same version/SHA → no update needed
			return "v1", "mysafesha12345678901234567890123456789012", nil
		default:
			return currentVersion, "", nil
		}
	}

	// Build actions-lock.json with a regular action and a safe-outputs action (with cached inputs).
	cache := workflow.NewActionCache(tmpDir)
	cache.Set("actions/checkout", "v4", "oldcheckoutsha234567890123456789012345678")
	cache.Set("owner/my-safe-action", "v1", "mysafesha12345678901234567890123456789012")
	cache.SetInputs("owner/my-safe-action", "v1", map[string]*workflow.ActionYAMLInput{
		"foo": {Description: "Foo input", Required: true},
	})
	if err := cache.Save(); err != nil {
		t.Fatalf("failed to save initial cache: %v", err)
	}

	// Run UpdateActions from tmpDir
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	if err := UpdateActions(context.Background(), false, false, false); err != nil {
		t.Fatalf("UpdateActions() error = %v", err)
	}

	// Reload the saved cache and verify safe-outputs inputs were preserved.
	saved := workflow.NewActionCache(tmpDir)
	if err := saved.Load(); err != nil {
		t.Fatalf("failed to reload cache: %v", err)
	}

	// actions/checkout should have been updated to v5
	checkoutEntry, ok := saved.Entries["actions/checkout@v5"]
	if !ok {
		t.Error("expected actions/checkout@v5 entry after update")
	} else if checkoutEntry.SHA != "newcheckoutsha1234567890123456789012345" {
		t.Errorf("actions/checkout SHA = %q, want newcheckoutsha...", checkoutEntry.SHA)
	}

	// safe-outputs action inputs must still be present
	safeEntry, ok := saved.Entries["owner/my-safe-action@v1"]
	if !ok {
		t.Fatal("expected owner/my-safe-action@v1 entry to be present after update")
	}
	if safeEntry.Inputs == nil {
		t.Error("safe-outputs action inputs were lost after update (expected to be preserved)")
	} else if _, hasFoo := safeEntry.Inputs["foo"]; !hasFoo {
		t.Errorf("safe-outputs action inputs missing 'foo' key; got %v", safeEntry.Inputs)
	}
}

func TestExtractBaseRepo(t *testing.T) {
	tests := []struct {
		name       string
		actionPath string
		want       string
	}{
		{
			name:       "action without subfolder",
			actionPath: "actions/checkout",
			want:       "actions/checkout",
		},
		{
			name:       "action with one subfolder",
			actionPath: "actions/cache/restore",
			want:       "actions/cache",
		},
		{
			name:       "action with multiple subfolders",
			actionPath: "github/codeql-action/upload-sarif",
			want:       "github/codeql-action",
		},
		{
			name:       "action with deeply nested subfolders",
			actionPath: "owner/repo/path/to/action",
			want:       "owner/repo",
		},
		{
			name:       "action with only owner",
			actionPath: "owner",
			want:       "owner",
		},
		{
			name:       "empty string",
			actionPath: "",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gitutil.ExtractBaseRepo(tt.actionPath)
			if got != tt.want {
				t.Errorf("gitutil.ExtractBaseRepo(%q) = %q, want %q", tt.actionPath, got, tt.want)
			}
		})
	}
}

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

func TestIsCoreAction(t *testing.T) {
	tests := []struct {
		name string
		repo string
		want bool
	}{
		{"actions/checkout is core", "actions/checkout", true},
		{"actions/setup-go is core", "actions/setup-go", true},
		{"actions/cache/restore is core", "actions/cache/restore", true},
		{"github/codeql-action is not core", "github/codeql-action", false},
		{"docker/login-action is not core", "docker/login-action", false},
		{"super-linter/super-linter is not core", "super-linter/super-linter", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCoreAction(tt.repo)
			if got != tt.want {
				t.Errorf("isCoreAction(%q) = %v, want %v", tt.repo, got, tt.want)
			}
		})
	}
}

func TestUpdateActionRefsInContent_NonCoreActionsUnchanged(t *testing.T) {
	// When allowMajor=false (--disable-release-bump), non-actions/* org references
	// should not be modified because they are not core actions.
	input := `steps:
  - uses: docker/login-action@v3
  - uses: github/codeql-action/upload-sarif@v3
  - run: echo hello`

	cache := make(map[string]latestReleaseResult)
	changed, newContent, err := updateActionRefsInContent(context.Background(), input, cache, false, false)
	if err != nil {
		t.Fatalf("updateActionRefsInContent() error = %v", err)
	}
	if changed {
		t.Errorf("updateActionRefsInContent() changed = true, want false for non-actions/* refs with allowMajor=false")
	}
	if newContent != input {
		t.Errorf("updateActionRefsInContent() modified content for non-actions/* refs\nGot: %s\nWant: %s", newContent, input)
	}
}

func TestUpdateActionRefsInContent_NoActionRefs(t *testing.T) {
	input := `description: Test workflow
steps:
  - run: echo hello
  - run: echo world`

	cache := make(map[string]latestReleaseResult)
	changed, _, err := updateActionRefsInContent(context.Background(), input, cache, true, false)
	if err != nil {
		t.Fatalf("updateActionRefsInContent() error = %v", err)
	}
	if changed {
		t.Errorf("updateActionRefsInContent() changed = true, want false for content with no action refs")
	}
}

func TestUpdateActionRefsInContent_VersionTagReplacement(t *testing.T) {
	// Stub getLatestActionReleaseFn so the test doesn't hit the network
	orig := getLatestActionReleaseFn
	defer func() { getLatestActionReleaseFn = orig }()

	getLatestActionReleaseFn = func(_ context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
		switch repo {
		case "actions/checkout":
			return "v6", "de0fac2e4500dabe0009e67214ff5f5447ce83dd", nil
		case "actions/setup-go":
			return "v6", "4b73464bb391a5985ede5d7fd8a6c0c9c59c4c4e", nil
		default:
			return currentVersion, "", nil
		}
	}

	input := `steps:
  - uses: actions/checkout@v4
  - uses: actions/setup-go@v5
  - run: echo hello`

	want := `steps:
  - uses: actions/checkout@v6
  - uses: actions/setup-go@v6
  - run: echo hello`

	cache := make(map[string]latestReleaseResult)
	changed, got, err := updateActionRefsInContent(context.Background(), input, cache, true, false)
	if err != nil {
		t.Fatalf("updateActionRefsInContent() error = %v", err)
	}
	if !changed {
		t.Error("updateActionRefsInContent() changed = false, want true")
	}
	if got != want {
		t.Errorf("updateActionRefsInContent() output mismatch\nGot:\n%s\nWant:\n%s", got, want)
	}
}

func TestUpdateActionRefsInContent_SHAPinnedReplacement(t *testing.T) {
	// Stub getLatestActionReleaseFn so the test doesn't hit the network
	orig := getLatestActionReleaseFn
	defer func() { getLatestActionReleaseFn = orig }()

	newSHA := "de0fac2e4500dabe0009e67214ff5f5447ce83dd"
	getLatestActionReleaseFn = func(_ context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
		return "v6.0.2", newSHA, nil
	}

	oldSHA := "11bd71901bbe5b1630ceea73d27597364c9af683"
	input := "        uses: actions/checkout@" + oldSHA + " # v5.0.0"
	want := "        uses: actions/checkout@" + newSHA + "  # v6.0.2"

	cache := make(map[string]latestReleaseResult)
	changed, got, err := updateActionRefsInContent(context.Background(), input, cache, true, false)
	if err != nil {
		t.Fatalf("updateActionRefsInContent() error = %v", err)
	}
	if !changed {
		t.Error("updateActionRefsInContent() changed = false, want true")
	}
	if got != want {
		t.Errorf("updateActionRefsInContent() output mismatch\nGot:  %s\nWant: %s", got, want)
	}
}

func TestUpdateActionRefsInContent_CacheReusedAcrossLines(t *testing.T) {
	// Verify that the cache prevents duplicate calls to getLatestActionReleaseFn
	orig := getLatestActionReleaseFn
	defer func() { getLatestActionReleaseFn = orig }()

	callCount := 0
	getLatestActionReleaseFn = func(_ context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
		callCount++
		return "v8", "ed597411d8f9245be5a6f5b7f5d52e63b7e62e96", nil
	}

	// Two lines referencing the same repo@version: should resolve via cache after first call
	input := `steps:
  - uses: actions/github-script@v7
  - uses: actions/github-script@v7`

	cache := make(map[string]latestReleaseResult)
	changed, _, err := updateActionRefsInContent(context.Background(), input, cache, true, false)
	if err != nil {
		t.Fatalf("updateActionRefsInContent() error = %v", err)
	}
	if !changed {
		t.Error("updateActionRefsInContent() changed = false, want true")
	}
	if callCount != 1 {
		t.Errorf("getLatestActionReleaseFn called %d times, want 1 (cache should prevent second call)", callCount)
	}
}

func TestUpdateActionRefsInContent_AllOrgsUpdatedWhenAllowMajor(t *testing.T) {
	// With allowMajor=true (default behaviour), non-actions/* org references should
	// also be updated to the latest major version.
	orig := getLatestActionReleaseFn
	defer func() { getLatestActionReleaseFn = orig }()

	getLatestActionReleaseFn = func(_ context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
		switch repo {
		case "docker/login-action":
			return "v4", "newsha11234567890123456789012345678901234", nil
		case "github/codeql-action":
			return "v4", "newsha21234567890123456789012345678901234", nil
		default:
			return currentVersion, "", nil
		}
	}

	input := `steps:
  - uses: docker/login-action@v3
  - uses: github/codeql-action@v3`

	want := `steps:
  - uses: docker/login-action@v4
  - uses: github/codeql-action@v4`

	cache := make(map[string]latestReleaseResult)
	changed, got, err := updateActionRefsInContent(context.Background(), input, cache, true, false)
	if err != nil {
		t.Fatalf("updateActionRefsInContent() error = %v", err)
	}
	if !changed {
		t.Error("updateActionRefsInContent() changed = false, want true")
	}
	if got != want {
		t.Errorf("updateActionRefsInContent() output mismatch\nGot:\n%s\nWant:\n%s", got, want)
	}
}

// TestGetLatestActionRelease_FallsBackToGitWhenNoReleases verifies that when the GitHub
// Releases API returns an empty list, getLatestActionRelease falls back to the git
// ls-remote tag scan (getLatestActionReleaseViaGitFn) rather than returning an error.
func TestGetLatestActionRelease_FallsBackToGitWhenNoReleases(t *testing.T) {
	origAPIfn := runGHReleasesAPIFn
	origGitFn := getLatestActionReleaseViaGitFn
	defer func() {
		runGHReleasesAPIFn = origAPIfn
		getLatestActionReleaseViaGitFn = origGitFn
	}()

	// Simulate the GitHub Releases API returning an empty list (no releases published).
	runGHReleasesAPIFn = func(_ context.Context, baseRepo string) ([]byte, error) {
		return []byte(""), nil
	}

	gitFnCalled := false
	getLatestActionReleaseViaGitFn = func(_ context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
		gitFnCalled = true
		return "v1.2.3", "abc1234567890123456789012345678901234567", nil
	}

	version, sha, err := getLatestActionRelease(context.Background(), "github/gh-aw-actions/setup", "v1", false, false)
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

// TestGetLatestActionRelease_FallbackReturnsErrorWhenBothFail verifies that when the
// GitHub Releases API returns an empty list and the git fallback also fails, the
// function returns an error rather than silently succeeding.
func TestGetLatestActionRelease_FallbackReturnsErrorWhenBothFail(t *testing.T) {
	origAPIfn := runGHReleasesAPIFn
	origGitFn := getLatestActionReleaseViaGitFn
	defer func() {
		runGHReleasesAPIFn = origAPIfn
		getLatestActionReleaseViaGitFn = origGitFn
	}()

	// Simulate the GitHub Releases API returning an empty list.
	runGHReleasesAPIFn = func(_ context.Context, baseRepo string) ([]byte, error) {
		return []byte(""), nil
	}

	// Simulate the git fallback also finding nothing.
	getLatestActionReleaseViaGitFn = func(_ context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
		return "", "", errors.New("no releases found")
	}

	_, _, err := getLatestActionRelease(context.Background(), "github/gh-aw-actions/setup", "v1", false, false)
	if err == nil {
		t.Fatal("expected error when both releases API and git fallback fail, got nil")
	}
}

// TestGetLatestActionRelease_PrereleaseTagsSkipped verifies that prerelease tags are
// not selected as the upgrade target even when they have a higher base version than
// the latest stable release.  Per semver rules, v1.1.0-beta.1 > v1.0.0 (base version
// comparison), so without explicit filtering a prerelease could be picked incorrectly.
func TestGetLatestActionRelease_PrereleaseTagsSkipped(t *testing.T) {
	origAPIfn := runGHReleasesAPIFn
	origSHAfn := getActionSHAForTagFn
	defer func() {
		runGHReleasesAPIFn = origAPIfn
		getActionSHAForTagFn = origSHAfn
	}()

	// Return a stable release alongside a higher-versioned prerelease.
	runGHReleasesAPIFn = func(_ context.Context, baseRepo string) ([]byte, error) {
		return []byte("v1.0.0\nv1.1.0-beta.1"), nil
	}

	getActionSHAForTagFn = func(_ context.Context, repo, tag string) (string, error) {
		return "stablesha1234567890123456789012345678901", nil
	}

	version, _, err := getLatestActionRelease(context.Background(), "actions/checkout", "v1.0.0", true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "v1.0.0" {
		t.Errorf("version = %q, want %q (prerelease v1.1.0-beta.1 should be skipped)", version, "v1.0.0")
	}
}

// TestUpdateActions_GhAwNativeActionCappedAtCLIVersion verifies that gh-aw native actions
// (github/gh-aw-actions/* and github/gh-aw/actions/*) are never updated to a version newer
// than the currently running CLI. This prevents users on an older CLI version from having
// their gh-aw actions pinned to a newer (possibly incompatible or pre-release) version.
func TestUpdateActions_GhAwNativeActionCappedAtCLIVersion(t *testing.T) {
	// Set the running CLI version to v0.68.3
	origVersion := GetVersion()
	SetVersionInfo("v0.68.3")
	defer SetVersionInfo(origVersion)

	// Stub getLatestActionReleaseFn to return a newer version (v0.68.7) simulating
	// the scenario where a newer release exists but the CLI is still at v0.68.3.
	origReleaseFn := getLatestActionReleaseFn
	defer func() { getLatestActionReleaseFn = origReleaseFn }()
	getLatestActionReleaseFn = func(_ context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
		switch repo {
		case "github/gh-aw-actions/setup":
			return "v0.68.7", "newersha1234567890123456789012345678901234", nil
		case "github/gh-aw/actions/setup":
			return "v0.68.7", "newersha1234567890123456789012345678901234", nil
		default:
			return currentVersion, "defaultsha12345678901234567890123456789012", nil
		}
	}

	// Stub getActionSHAForTagFn to return a SHA for the CLI version tag (v0.68.3).
	origSHAfn := getActionSHAForTagFn
	defer func() { getActionSHAForTagFn = origSHAfn }()
	const cliVersionSHA = "cliversha12345678901234567890123456789012"
	getActionSHAForTagFn = func(_ context.Context, repo, tag string) (string, error) {
		if tag == "v0.68.3" {
			return cliVersionSHA, nil
		}
		return "othersha12345678901234567890123456789012", nil
	}

	tmpDir := testutil.TempDir(t, "test-*")
	cache := workflow.NewActionCache(tmpDir)
	cache.Set("github/gh-aw-actions/setup", "v0.68.1", "oldsha1234567890123456789012345678901234a")
	cache.Set("github/gh-aw/actions/setup", "v0.68.1", "oldsha1234567890123456789012345678901234b")
	if err := cache.Save(); err != nil {
		t.Fatalf("failed to save initial cache: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	if err := UpdateActions(context.Background(), false, false, false); err != nil {
		t.Fatalf("UpdateActions() error = %v", err)
	}

	saved := workflow.NewActionCache(tmpDir)
	if err := saved.Load(); err != nil {
		t.Fatalf("failed to reload cache: %v", err)
	}

	// Both gh-aw native actions must have been updated to the CLI version (v0.68.3),
	// not the latest release (v0.68.7).
	for _, repo := range []string{"github/gh-aw-actions/setup", "github/gh-aw/actions/setup"} {
		expectedKey := repo + "@v0.68.3"
		entry, ok := saved.Entries[expectedKey]
		if !ok {
			t.Errorf("expected entry %q in actions-lock.json (capped at CLI version), got entries: %v", expectedKey, savedEntryKeys(saved))
			continue
		}
		if entry.SHA != cliVersionSHA {
			t.Errorf("%s SHA = %q, want CLI-version SHA %q", repo, entry.SHA, cliVersionSHA)
		}
		// The newer version must NOT appear.
		newerKey := repo + "@v0.68.7"
		if _, found := saved.Entries[newerKey]; found {
			t.Errorf("found unexpected entry %q (gh-aw native action must not exceed CLI version)", newerKey)
		}
	}
}

// TestIsGhAwNativeAction verifies that isGhAwNativeAction correctly identifies gh-aw
// native action repos and excludes non-native repos.
func TestIsGhAwNativeAction(t *testing.T) {
	tests := []struct {
		repo string
		want bool
	}{
		{"github/gh-aw-actions/setup", true},
		{"github/gh-aw/actions/setup", true},
		{"github/gh-aw/actions/setup-cli", true},
		{"actions/checkout", false},
		{"actions/setup-node", false},
		{"docker/login-action", false},
		{"github/codeql-action/upload-sarif", false},
	}
	for _, tt := range tests {
		t.Run(tt.repo, func(t *testing.T) {
			got := isGhAwNativeAction(tt.repo)
			if got != tt.want {
				t.Errorf("isGhAwNativeAction(%q) = %v, want %v", tt.repo, got, tt.want)
			}
		})
	}
}

// savedEntryKeys returns the map keys of a loaded ActionCache for error messages.
func savedEntryKeys(cache *workflow.ActionCache) []string {
	keys := make([]string, 0, len(cache.Entries))
	for k := range cache.Entries {
		keys = append(keys, k)
	}
	return keys
}
