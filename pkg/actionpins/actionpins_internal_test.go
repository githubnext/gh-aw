//go:build !integration

package actionpins

import (
	"context"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildByRepoIndex_GroupsByRepoAndSortsDescending(t *testing.T) {
	pins := []ActionPin{
		{Repo: "actions/checkout", Version: "v4.0.0", SHA: "sha-v4"},
		{Repo: "actions/checkout", Version: "v5.0.0", SHA: "sha-v5"},
		{Repo: "actions/setup-go", Version: "v5.1.0", SHA: "sha-go-v5-1"},
		{Repo: "actions/setup-go", Version: "v5.0.0", SHA: "sha-go-v5-0"},
	}

	byRepo := buildByRepoIndex(pins)

	require.Len(t, byRepo["actions/checkout"], 2, "Expected checkout pins to be grouped")
	require.Equal(t, "v5.0.0", byRepo["actions/checkout"][0].Version, "Expected v5.0.0 as newest checkout pin")
	require.Equal(t, "v4.0.0", byRepo["actions/checkout"][1].Version, "Expected v4.0.0 as second checkout pin")

	require.Len(t, byRepo["actions/setup-go"], 2, "Expected setup-go pins to be grouped")
	require.Equal(t, "v5.1.0", byRepo["actions/setup-go"][0].Version, "Expected v5.1.0 as newest setup-go pin")
	require.Equal(t, "v5.0.0", byRepo["actions/setup-go"][1].Version, "Expected v5.0.0 as second setup-go pin")
}

func TestCountPinKeyMismatches_ReturnsOnlyVersionMismatches(t *testing.T) {
	t.Run("returns zero for empty entries", func(t *testing.T) {
		assert.Zero(t, countPinKeyMismatches(map[string]ActionPin{}), "Expected empty input to produce zero mismatches")
	})

	t.Run("returns zero when all key versions match", func(t *testing.T) {
		entries := map[string]ActionPin{
			"actions/checkout@v5": {Repo: "actions/checkout", Version: "v5", SHA: "sha-1"},
			"actions/setup-go@v4": {Repo: "actions/setup-go", Version: "v4", SHA: "sha-2"},
		}
		assert.Zero(t, countPinKeyMismatches(entries), "Expected zero mismatches when key versions match pin versions")
	})

	t.Run("ignores invalid keys without version suffix", func(t *testing.T) {
		entries := map[string]ActionPin{
			"invalid-key": {Repo: "actions/cache", Version: "v4", SHA: "sha-3"},
		}
		assert.Zero(t, countPinKeyMismatches(entries), "Expected invalid keys without @version to be ignored")
	})

	t.Run("counts only true key-version mismatches", func(t *testing.T) {
		entries := map[string]ActionPin{
			"actions/checkout@v5": {Repo: "actions/checkout", Version: "v5", SHA: "sha-1"},
			"actions/setup-go@v5": {Repo: "actions/setup-go", Version: "v4", SHA: "sha-2"},
			"invalid-key":         {Repo: "actions/cache", Version: "v4", SHA: "sha-3"},
		}
		assert.Equal(t, 1, countPinKeyMismatches(entries), "Expected only one key/version mismatch to be counted")
	})
}

func TestInitWarnings_InitializesAndPreservesMap(t *testing.T) {
	t.Run("initializes nil warnings map", func(t *testing.T) {
		ctx := &PinContext{}

		initWarnings(ctx)

		require.NotNil(t, ctx.Warnings, "Expected warnings map to be initialized")
		assert.Empty(t, ctx.Warnings, "Expected initialized warnings map to be empty")
	})

	t.Run("preserves existing warnings map", func(t *testing.T) {
		existing := map[string]struct{}{"actions/checkout@v5": {}}
		ctx := &PinContext{Warnings: make(map[string]bool, len(existing))}
		for warning := range existing {
			ctx.Warnings[warning] = true
		}

		initWarnings(ctx)

		require.NotNil(t, ctx.Warnings, "Expected warnings map to remain initialized")
		assert.Len(t, ctx.Warnings, len(existing), "Expected existing warnings entries to be preserved")
		for warning := range existing {
			assert.True(t, ctx.Warnings[warning], "Expected warning %q to be preserved", warning)
		}
	})
}

func TestFormatPinnedActionWithResolution_ConsistentVersionComment(t *testing.T) {
	tests := []struct {
		name            string
		repo            string
		sha             string
		sourceVersion   string
		resolvedVersion string
		expected        string
	}{
		{
			name:            "shows only source version when resolvedVersion is empty",
			repo:            "actions/checkout",
			sha:             "abc123",
			sourceVersion:   "v4",
			resolvedVersion: "",
			expected:        "actions/checkout@abc123 # v4",
		},
		{
			name:            "shows only version when source equals resolved",
			repo:            "actions/checkout",
			sha:             "abc123",
			sourceVersion:   "v4.1.2",
			resolvedVersion: "v4.1.2",
			expected:        "actions/checkout@abc123 # v4.1.2",
		},
		{
			name:            "shows both versions when source differs from resolved",
			repo:            "actions/checkout",
			sha:             "abc123",
			sourceVersion:   "v4",
			resolvedVersion: "v4.1.2",
			expected:        "actions/checkout@abc123 # v4.1.2 (source v4)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPinnedActionWithResolution(tt.repo, tt.sha, tt.sourceVersion, tt.resolvedVersion)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindCompatiblePin_SemverFallback(t *testing.T) {
	pins := []ActionPin{
		{Repo: "actions/checkout", Version: "v5.2.0", SHA: "sha-v5-2"},
		{Repo: "actions/checkout", Version: "v5.0.0", SHA: "sha-v5-0"},
		{Repo: "actions/checkout", Version: "v4.9.9", SHA: "sha-v4-9"},
	}

	tests := []struct {
		name          string
		version       string
		wantFound     bool
		wantVersion   string
		availablePins []ActionPin
	}{
		{
			name:          "exact-major",
			version:       "v5",
			wantFound:     true,
			wantVersion:   "v5.2.0",
			availablePins: pins,
		},
		{
			// major-compatible-match: requesting v5.0.0 from a list that contains only v5.0.0 and
			// v4.9.9 returns v5.0.0 — the function uses major-compatible matching, not exact-version
			// matching. The only v5.x present happens to be v5.0.0, so the result looks exact.
			name:        "major-compatible-match",
			version:     "v5.0.0",
			wantFound:   true,
			wantVersion: "v5.0.0",
			availablePins: []ActionPin{
				{Repo: "actions/checkout", Version: "v5.0.0", SHA: "sha-v5-0"},
				{Repo: "actions/checkout", Version: "v4.9.9", SHA: "sha-v4-9"},
			},
		},
		{
			// first-compatible-not-exact: requesting v5.0.0 from [v5.2.0, v5.0.0] returns v5.2.0
			// because findCompatiblePin returns the first major-compatible match, not the exact one.
			// This is a consequence of the pre-sorted (descending) slice contract; see
			// returns-first-compatible-not-highest for the complementary case with an unsorted list.
			name:          "first-compatible-not-exact",
			version:       "v5.0.0",
			wantFound:     true,
			wantVersion:   "v5.2.0",
			availablePins: pins,
		},
		{
			// returns-first-compatible-not-highest: list order determines the result.
			// With an unsorted list [v5.0.0, v5.2.0], requesting v5 returns v5.0.0 (the first
			// major-compatible entry), not v5.2.0. Callers (e.g. resolveNonStrictHardcodedPin) must
			// supply a pre-sorted (descending) slice via GetActionPinsByRepo to get the highest pin.
			name:        "returns-first-compatible-not-highest",
			version:     "v5",
			wantFound:   true,
			wantVersion: "v5.0.0",
			availablePins: []ActionPin{
				{Repo: "actions/checkout", Version: "v5.0.0", SHA: "sha-low"},
				{Repo: "actions/checkout", Version: "v5.2.0", SHA: "sha-high"},
			},
		},
		{
			// minor-version-constraint: IsCompatible performs major-only comparison, so
			// requesting v5.1 from [v5.2.0, v5.0.0] returns v5.2.0 (same major, first match).
			name:        "minor-version-constraint",
			version:     "v5.1",
			wantFound:   true,
			wantVersion: "v5.2.0",
			availablePins: []ActionPin{
				{Repo: "actions/checkout", Version: "v5.2.0", SHA: "sha-a"},
				{Repo: "actions/checkout", Version: "v5.0.0", SHA: "sha-b"},
			},
		},
		{
			name:          "no-match",
			version:       "v6",
			wantFound:     false,
			availablePins: pins,
		},
		{
			name:          "empty-version",
			version:       "",
			wantFound:     false,
			availablePins: pins,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pin, found := findCompatiblePin(tt.availablePins, tt.version)
			assert.Equal(t, tt.wantFound, found)
			if tt.wantFound {
				assert.Equal(t, tt.wantVersion, pin.Version)
			} else {
				assert.Equal(t, ActionPin{}, pin, "expected zero-value ActionPin when not found")
			}
		})
	}
}

func TestFindVersionBySHA_ReturnsVersionForKnownSHA(t *testing.T) {
	t.Run("returns version for a known SHA in embedded data", func(t *testing.T) {
		pins := GetActionPinsByRepo("actions/checkout")
		require.NotEmpty(t, pins, "prerequisite: embedded pins must exist for actions/checkout")

		knownPin := pins[0]
		version := findVersionBySHA("actions/checkout", knownPin.SHA)
		assert.Equal(t, knownPin.Version, version, "should return the version for a known SHA")
	})

	t.Run("returns empty string for unknown SHA", func(t *testing.T) {
		version := findVersionBySHA("actions/checkout", "0000000000000000000000000000000000000000")
		assert.Empty(t, version, "should return empty string for unknown SHA")
	})

	t.Run("returns empty string for unknown repo", func(t *testing.T) {
		version := findVersionBySHA("does-not-exist/unknown", "abc123")
		assert.Empty(t, version, "should return empty string for unknown repo")
	})
}

func TestGetContainerPin_ReturnsPinnedImage(t *testing.T) {
	pin, ok := GetContainerPin("node:lts-alpine")
	require.True(t, ok, "Expected embedded container pin for node:lts-alpine")
	assert.Equal(t, "node:lts-alpine", pin.Image, "Expected image name to match key")
	assert.NotEmpty(t, pin.Digest, "Expected digest to be populated")
	assert.Contains(t, pin.PinnedImage, "@sha256:", "Expected pinned image to include digest")
}

func TestGetContainerPin_MCPGatewayVersionsArePinned(t *testing.T) {
	tests := []struct {
		name   string
		image  string
		digest string
	}{
		{
			name:   "v0.3.6",
			image:  "ghcr.io/github/gh-aw-mcpg:v0.3.6",
			digest: "sha256:2bb8eef86006a4c5963c55616a9c51c32f27bfdecb023b8aa6f91f6718d9171c",
		},
		{
			name:   "v0.3.9",
			image:  "ghcr.io/github/gh-aw-mcpg:v0.3.9",
			digest: "sha256:64828b42a4482f58fab16509d7f8f495a6d97c972a98a68aff20543531ac0388",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pin, ok := GetContainerPin(tt.image)
			require.True(t, ok, "Expected embedded container pin for %s", tt.image)
			assert.Equal(t, tt.image, pin.Image, "Expected image name to match key")
			assert.Equal(t, tt.digest, pin.Digest, "Expected digest to match for %s", tt.image)
			assert.Equal(t, tt.image+"@"+tt.digest, pin.PinnedImage, "Expected pinned image to include digest for %s", tt.image)
		})
	}
}

func TestGetContainerPin_DefaultMCPImagesArePinned(t *testing.T) {
	images := []string{
		constants.DefaultMCPGatewayContainer + ":" + string(constants.DefaultMCPGatewayVersion),
		"ghcr.io/github/github-mcp-server:" + string(constants.DefaultGitHubMCPServerVersion),
	}

	for _, image := range images {
		pin, ok := GetContainerPin(image)
		require.True(t, ok, "Expected embedded container pin for %s", image)
		assert.Equal(t, image, pin.Image, "Expected image name to match key")
		assert.NotEmpty(t, pin.Digest, "Expected digest to be populated for %s", image)
		assert.Equal(t, image+"@"+pin.Digest, pin.PinnedImage, "Expected pinned image to include digest for %s", image)
	}
}

type countingResolver struct {
	called int
}

func (r *countingResolver) ResolveSHA(_ context.Context, _, _ string) (string, error) {
	r.called++
	return "", nil
}

func TestResolveActionPinDynamically_SkipsForSHAInput(t *testing.T) {
	resolver := &countingResolver{}
	ctx := &PinContext{Resolver: resolver}

	result, ok := resolveActionPinDynamically(
		"actions/checkout",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		true,
		ctx,
	)

	assert.False(t, ok, "Expected no dynamic resolution for SHA input")
	assert.Empty(t, result, "Expected empty result when dynamic resolution is skipped")
	assert.Zero(t, resolver.called, "Expected resolver not to be called for SHA input")
}

func TestResolveActionPinFromHardcodedPins_StrictModeNoFallback(t *testing.T) {
	ctx := &PinContext{StrictMode: true, Warnings: make(map[string]bool)}

	result, ok := resolveActionPinFromHardcodedPins("actions/checkout", "v999", false, ctx)

	assert.False(t, ok, "Expected strict mode not to fall back to any other hardcoded version")
	assert.Empty(t, result, "Expected no pinned result in strict mode without exact match")
}

func TestResolveExactHardcodedPin_BySHA(t *testing.T) {
	pins := []ActionPin{{Repo: "actions/checkout", Version: "v5.0.0", SHA: "sha-v5"}}

	result, ok := resolveExactHardcodedPin("actions/checkout", "sha-v5", true, pins)

	require.True(t, ok, "Expected exact SHA match to resolve")
	assert.Contains(t, result, "sha-v5", "Expected result to include matched SHA")
}

func TestResolveExactHardcodedPin_ByVersion(t *testing.T) {
	pins := []ActionPin{{Repo: "actions/checkout", Version: "v5.0.0", SHA: "sha-v5"}}

	result, ok := resolveExactHardcodedPin("actions/checkout", "v5.0.0", false, pins)

	require.True(t, ok, "Expected exact version match to resolve")
	assert.Contains(t, result, "sha-v5", "Expected result to include matched SHA")
	assert.Contains(t, result, "v5.0.0", "Expected result to include matched version")
}

func TestResolveExactHardcodedPin_NoMatch(t *testing.T) {
	pins := []ActionPin{{Repo: "actions/checkout", Version: "v5.0.0", SHA: "sha-v5"}}

	result, ok := resolveExactHardcodedPin("actions/checkout", "v4.0.0", false, pins)

	assert.False(t, ok, "Expected no resolution when version does not match and input is not SHA")
	assert.Empty(t, result, "Expected empty result when no exact match is found")
}

func TestResolveExactHardcodedPin_VersionTakesPrecedenceOverSHA(t *testing.T) {
	// When isAlreadySHA=false, only the version-match path runs; the SHA-match loop is
	// skipped entirely. This test uses a pin whose Version and SHA fields are identical
	// to make the path selection explicit: the version loop matches and returns before
	// the SHA loop would ever execute.
	pins := []ActionPin{{Repo: "actions/checkout", Version: "sha-v5", SHA: "sha-v5"}}
	result, ok := resolveExactHardcodedPin("actions/checkout", "sha-v5", false, pins)
	require.True(t, ok, "Expected version-match path to resolve when isAlreadySHA=false")
	assert.Contains(t, result, "sha-v5", "Expected result to include the matched pin's SHA/version")
}

func TestResolveNonStrictHardcodedPin_SelectsHighestCompatible(t *testing.T) {
	pins := []ActionPin{
		{Repo: "actions/checkout", Version: "v5.2.0", SHA: "sha-v5-2"},
		{Repo: "actions/checkout", Version: "v5.0.0", SHA: "sha-v5-0"},
		{Repo: "actions/checkout", Version: "v4.9.9", SHA: "sha-v4-9"},
	}

	t.Run("uses semver-compatible pin when available", func(t *testing.T) {
		ctx := &PinContext{Warnings: make(map[string]bool)}
		var result string

		stderrOutput := testutil.CaptureStderr(t, func() {
			result = resolveNonStrictHardcodedPin("actions/checkout", "v5", pins, ctx)
		})

		assert.Contains(t, result, "sha-v5-2", "Expected highest semver-compatible pin to be selected")
		assert.True(t, ctx.Warnings["actions/checkout@v5"], "Expected warning cache key to be recorded")
		// Assert on the stable unstyled message fragment to avoid ANSI-code fragility.
		assert.Contains(t, stderrOutput, "using hardcoded pin for actions/checkout@v5.2.0", "Expected warning to be emitted for compatible fallback")
	})

	t.Run("deduplicates warning on compatible path", func(t *testing.T) {
		ctx := &PinContext{Warnings: make(map[string]bool)}

		stderrOutput := testutil.CaptureStderr(t, func() {
			resolveNonStrictHardcodedPin("actions/checkout", "v5", pins, ctx)
			resolveNonStrictHardcodedPin("actions/checkout", "v5", pins, ctx)
		})

		assert.Equal(t, 1, strings.Count(stderrOutput, "using hardcoded pin for actions/checkout@v5.2.0"),
			"Expected warning emitted exactly once for repeated compatible resolution")
		assert.Len(t, ctx.Warnings, 1)
	})
}

func TestResolveNonStrictHardcodedPin_FallsBackToHighestWhenNoCompatible(t *testing.T) {
	pins := []ActionPin{
		{Repo: "actions/checkout", Version: "v5.2.0", SHA: "sha-v5-2"},
		{Repo: "actions/checkout", Version: "v5.0.0", SHA: "sha-v5-0"},
		{Repo: "actions/checkout", Version: "v4.9.9", SHA: "sha-v4-9"},
	}

	ctx := &PinContext{Warnings: make(map[string]bool)}
	var first, second string

	stderrOutput := testutil.CaptureStderr(t, func() {
		first = resolveNonStrictHardcodedPin("actions/checkout", "v9", pins, ctx)
		second = resolveNonStrictHardcodedPin("actions/checkout", "v9", pins, ctx)
	})

	assert.Contains(t, first, "sha-v5-2", "Expected fallback to highest available pin when no compatible version exists")
	assert.Contains(t, second, "sha-v5-2", "Expected consistent fallback result on repeated calls")
	assert.True(t, ctx.Warnings["actions/checkout@v9"], "Expected warning cache key to be recorded")
	assert.Len(t, ctx.Warnings, 1, "Expected warning deduplication for repeated resolution attempts")
	// Assert on the stable unstyled message fragment to avoid ANSI-code fragility.
	assert.Equal(t, 1, strings.Count(stderrOutput, "using hardcoded pin for actions/checkout@v5.2.0"), "Expected warning to be emitted exactly once for repeated fallback resolution")
}

func TestResolveActionPinFromHardcodedPins_SkipHardcodedFallback(t *testing.T) {
	t.Run("returns false immediately when SkipHardcodedFallback is set", func(t *testing.T) {
		ctx := &PinContext{SkipHardcodedFallback: true, Warnings: make(map[string]bool)}

		// actions/checkout has hardcoded pins, but SkipHardcodedFallback should prevent use
		result, ok := resolveActionPinFromHardcodedPins("actions/checkout", "v4", false, ctx)

		assert.False(t, ok, "Expected SkipHardcodedFallback to prevent hardcoded pin lookup")
		assert.Empty(t, result, "Expected no pinned result when SkipHardcodedFallback is set")
	})

	t.Run("allows hardcoded pins when SkipHardcodedFallback is not set", func(t *testing.T) {
		ctx := &PinContext{SkipHardcodedFallback: false, Warnings: make(map[string]bool)}

		// actions/checkout has hardcoded pins and should resolve
		result, ok := resolveActionPinFromHardcodedPins("actions/checkout", "v4", false, ctx)

		assert.True(t, ok, "Expected hardcoded pins to be consulted when SkipHardcodedFallback is false")
		assert.NotEmpty(t, result, "Expected a pinned result when SkipHardcodedFallback is not set")
	})
}

func TestApplyActionPinMapping_NoMapping(t *testing.T) {
	ctx := &PinContext{Warnings: make(map[string]bool)}

	repo, version := applyActionPinMapping("actions/checkout", "v4", ctx)

	assert.Equal(t, "actions/checkout", repo, "repo should be unchanged when no mapping exists")
	assert.Equal(t, "v4", version, "version should be unchanged when no mapping exists")
}

func TestApplyActionPinMapping_AppliesMapping(t *testing.T) {
	ctx := &PinContext{
		Warnings: make(map[string]bool),
		Mappings: map[string]string{
			"actions/checkout@v4": "acme-corp/checkout@v4",
		},
	}

	repo, version := applyActionPinMapping("actions/checkout", "v4", ctx)

	assert.Equal(t, "acme-corp/checkout", repo, "repo should be replaced by mapping")
	assert.Equal(t, "v4", version, "version should be replaced by mapping")
}

func TestApplyActionPinMapping_OnlyMatchesExact(t *testing.T) {
	ctx := &PinContext{
		Warnings: make(map[string]bool),
		Mappings: map[string]string{
			"actions/checkout@v4": "acme-corp/checkout@v4",
		},
	}

	// Different version — should not match.
	repo, version := applyActionPinMapping("actions/checkout", "v5", ctx)

	assert.Equal(t, "actions/checkout", repo, "repo should be unchanged when version does not match")
	assert.Equal(t, "v5", version, "version should be unchanged when version does not match")
}

func TestApplyActionPinMapping_DeduplicatesInfoMessage(t *testing.T) {
	ctx := &PinContext{
		Warnings: make(map[string]bool),
		Mappings: map[string]string{
			"actions/checkout@v4": "acme-corp/checkout@v4",
		},
	}

	applyActionPinMapping("actions/checkout", "v4", ctx)
	applyActionPinMapping("actions/checkout", "v4", ctx)

	notifyKey := "map:actions/checkout@v4"
	assert.True(t, ctx.Warnings[notifyKey], "Expected notification key to be marked as seen")
	// Count should be exactly one entry (deduplication).
	mapNotifications := 0
	for k := range ctx.Warnings {
		if strings.HasPrefix(k, "map:") {
			mapNotifications++
		}
	}
	assert.Equal(t, 1, mapNotifications, "Expected exactly one deduplicated mapping notification")
}

func TestApplyActionPinMapping_InvalidMappingValueSkipped(t *testing.T) {
	ctx := &PinContext{
		Warnings: make(map[string]bool),
		Mappings: map[string]string{
			"actions/checkout@v4": "no-at-sign", // invalid: missing at sign and version
		},
	}

	repo, version := applyActionPinMapping("actions/checkout", "v4", ctx)

	// Invalid value should be silently skipped.
	assert.Equal(t, "actions/checkout", repo, "repo should be unchanged for invalid mapping value")
	assert.Equal(t, "v4", version, "version should be unchanged for invalid mapping value")
}

func TestApplyActionPinMapping_TargetWithoutRefSeparatorSkipped(t *testing.T) {
	ctx := &PinContext{
		Warnings: make(map[string]bool),
		Mappings: map[string]string{
			"actions/checkout@v4": "acme-corp/checkout", // invalid: missing @ separator
		},
	}

	repo, version := applyActionPinMapping("actions/checkout", "v4", ctx)

	assert.Equal(t, "actions/checkout", repo, "repo should be unchanged when mapping target has no @ separator")
	assert.Equal(t, "v4", version, "version should be unchanged when mapping target has no @ separator")
	assert.NotContains(t, ctx.Warnings, "map:actions/checkout@v4", "invalid mapping target should not set mapping warning key")
}
