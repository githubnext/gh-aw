//go:build !integration

package cli

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestUpdateActionRefsInContent_VersionTagReplacement(t *testing.T) {
	// Stub latest release lookup so the test doesn't hit the network.
	deps := newActionUpdateDepsWithLatestRelease(func(_ context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
		switch repo {
		case "actions/checkout":
			return "v6", "de0fac2e4500dabe0009e67214ff5f5447ce83dd", nil
		case "actions/setup-go":
			return "v6", "4b73464bb391a5985ede5d7fd8a6c0c9c59c4c4e", nil
		default:
			return currentVersion, "", nil
		}
	})

	input := `steps:
  - uses: actions/checkout@v4
  - uses: actions/setup-go@v5
  - run: echo hello`

	want := `steps:
  - uses: actions/checkout@v6
  - uses: actions/setup-go@v6
  - run: echo hello`

	cache := make(map[string]latestReleaseResult)
	changed, got, err := updateActionRefsInContentWithDeps(context.Background(), deps, input, cache, make(map[string]coolDownCheckResult), true, false, 0)
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
	newSHA := "de0fac2e4500dabe0009e67214ff5f5447ce83dd"
	deps := newActionUpdateDepsWithLatestRelease(func(_ context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
		return "v6.0.2", newSHA, nil
	})

	oldSHA := "11bd71901bbe5b1630ceea73d27597364c9af683"
	input := "        uses: actions/checkout@" + oldSHA + " # v5.0.0"
	want := "        uses: actions/checkout@" + newSHA + "  # v6.0.2"

	cache := make(map[string]latestReleaseResult)
	changed, got, err := updateActionRefsInContentWithDeps(context.Background(), deps, input, cache, make(map[string]coolDownCheckResult), true, false, 0)
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
	// Verify that the cache prevents duplicate calls to latest-release resolution.
	callCount := 0
	deps := newActionUpdateDepsWithLatestRelease(func(_ context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
		callCount++
		return "v8", "ed597411d8f9245be5a6f5b7f5d52e63b7e62e96", nil
	})

	// Two lines referencing the same repo@version: should resolve via cache after first call
	input := `steps:
  - uses: actions/github-script@v7
  - uses: actions/github-script@v7`

	cache := make(map[string]latestReleaseResult)
	changed, _, err := updateActionRefsInContentWithDeps(context.Background(), deps, input, cache, make(map[string]coolDownCheckResult), true, false, 0)
	if err != nil {
		t.Fatalf("updateActionRefsInContent() error = %v", err)
	}
	if !changed {
		t.Error("updateActionRefsInContent() changed = false, want true")
	}
	if callCount != 1 {
		t.Errorf("latest release resolver called %d times, want 1 (cache should prevent second call)", callCount)
	}
}
func TestUpdateActionRefsInContent_AllOrgsUpdatedWhenAllowMajor(t *testing.T) {
	// With allowMajor=true (default behaviour), non-actions/* org references should
	// also be updated to the latest major version.
	deps := newActionUpdateDepsWithLatestRelease(func(_ context.Context, repo, currentVersion string, allowMajor, verbose bool) (string, string, error) {
		switch repo {
		case "docker/login-action":
			return "v4", "newsha11234567890123456789012345678901234", nil
		case "github/codeql-action":
			return "v4", "newsha21234567890123456789012345678901234", nil
		default:
			return currentVersion, "", nil
		}
	})

	input := `steps:
  - uses: docker/login-action@v3
  - uses: github/codeql-action@v3`

	want := `steps:
  - uses: docker/login-action@v4
  - uses: github/codeql-action@v4`

	cache := make(map[string]latestReleaseResult)
	changed, got, err := updateActionRefsInContentWithDeps(context.Background(), deps, input, cache, make(map[string]coolDownCheckResult), true, false, 0)
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
func TestUpdateSkillRefsInContentWithResolver_UpdatesStringAndObjectSkillRefs(t *testing.T) {
	oldRepoSkillSHA := "1111111111111111111111111111111111111111"
	oldPathSkillSHA := "2222222222222222222222222222222222222222"
	newRepoSkillSHA := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	newPathSkillSHA := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	input := `---
name: test
skills:
  - githubnext/skills@` + oldRepoSkillSHA + `
  - skill: githubnext/skills/review/security@` + oldPathSkillSHA + `
  - ${{ inputs.dynamic_skill }}
---
body
`

	resolver := func(_ context.Context, repo, currentRef string, allowMajor, verbose bool, coolDown time.Duration) (string, error) {
		if repo != "githubnext/skills" {
			t.Fatalf("resolver called with repo %q, want githubnext/skills", repo)
		}
		switch currentRef {
		case oldRepoSkillSHA:
			return newRepoSkillSHA, nil
		case oldPathSkillSHA:
			return newPathSkillSHA, nil
		default:
			return currentRef, nil
		}
	}

	changed, got, err := updateSkillRefsInContentWithResolver(context.Background(), input, true, false, 0, resolver)
	if err != nil {
		t.Fatalf("updateSkillRefsInContentWithResolver() error = %v", err)
	}
	if !changed {
		t.Fatal("updateSkillRefsInContentWithResolver() changed = false, want true")
	}
	if !strings.Contains(got, "githubnext/skills@"+newRepoSkillSHA) {
		t.Fatalf("updated content missing updated repo skill ref:\n%s", got)
	}
	if !strings.Contains(got, "githubnext/skills/review/security@"+newPathSkillSHA) {
		t.Fatalf("updated content missing updated path skill ref:\n%s", got)
	}
	if !strings.Contains(got, "- ${{ inputs.dynamic_skill }}") {
		t.Fatalf("updated content unexpectedly modified expression skill ref:\n%s", got)
	}
}
func TestUpdateSkillRefsInContentWithResolver_PreservesObjectAuthFields(t *testing.T) {
	oldSHA := "1111111111111111111111111111111111111111"
	newSHA := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	input := `---
name: test
skills:
  - skill: githubnext/skills/review/security@` + oldSHA + `
    github-token: ${{ secrets.SOME_TOKEN }}
---
body
`
	resolver := func(_ context.Context, repo, currentRef string, allowMajor, verbose bool, coolDown time.Duration) (string, error) {
		if repo != "githubnext/skills" {
			t.Fatalf("resolver called with repo %q, want githubnext/skills", repo)
		}
		if currentRef != oldSHA {
			t.Fatalf("resolver called with ref %q, want %q", currentRef, oldSHA)
		}
		return newSHA, nil
	}

	changed, got, err := updateSkillRefsInContentWithResolver(context.Background(), input, true, false, 0, resolver)
	if err != nil {
		t.Fatalf("updateSkillRefsInContentWithResolver() error = %v", err)
	}
	if !changed {
		t.Fatal("updateSkillRefsInContentWithResolver() changed = false, want true")
	}
	if !strings.Contains(got, "skill: githubnext/skills/review/security@"+newSHA) {
		t.Fatalf("updated content missing updated object skill ref:\n%s", got)
	}
	if !strings.Contains(got, "github-token: ${{ secrets.SOME_TOKEN }}") {
		t.Fatalf("updated content dropped github-token object field:\n%s", got)
	}
}
func TestUpdateSkillRefsInContentWithResolver_NoFrontmatterNoChange(t *testing.T) {
	input := "steps:\n  - run: echo hello\n"
	changed, got, err := updateSkillRefsInContentWithResolver(context.Background(), input, true, false, 0, resolveLatestRef)
	if err != nil {
		t.Fatalf("updateSkillRefsInContentWithResolver() error = %v", err)
	}
	if changed {
		t.Fatal("updateSkillRefsInContentWithResolver() changed = true, want false")
	}
	if got != input {
		t.Fatalf("content changed unexpectedly:\n got: %q\nwant: %q", got, input)
	}
}

// TestUpdateActionRefsInContent_CooldownFallback verifies that
// updateActionRefsInContentWithDeps falls back to an older cooled-down release
// when the newest candidate is still within the cooldown window.
func TestUpdateActionRefsInContent_CooldownFallback(t *testing.T) {
	deps := newTestActionUpdateDeps()

	// Latest version is v1.321.0 (in cooldown); fallback is v1.320.0.
	deps.getLatestRelease = func(_ context.Context, repo, _ string, _, _ bool) (string, string, error) {
		if repo == "ruby/setup-ruby" {
			return "v1.321.0", "sha321_12345678901234567890123456789012", nil
		}
		return "v1.0.0", "default_1234567890123456789012345678901234", nil
	}
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
	const fallbackSHA = "sha320_12345678901234567890123456789012"
	deps.getActionSHAForTag = func(_ context.Context, _, tag string) (string, error) {
		if tag == "v1.320.0" {
			return fallbackSHA, nil
		}
		return "sha321_12345678901234567890123456789012", nil
	}

	input := "steps:\n  - uses: ruby/setup-ruby@v1.300.0\n"

	changed, got, err := updateActionRefsInContentWithDeps(
		context.Background(), deps, input,
		make(map[string]latestReleaseResult),
		make(map[string]coolDownCheckResult),
		true, false, 7*24*time.Hour,
	)
	if err != nil {
		t.Fatalf("updateActionRefsInContentWithDeps() error = %v", err)
	}
	if !changed {
		t.Fatal("expected content to be updated, but changed = false")
	}
	if want := "steps:\n  - uses: ruby/setup-ruby@v1.320.0\n"; got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
