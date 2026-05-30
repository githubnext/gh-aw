//go:build !integration

package actionpins

import (
	"context"
	"testing"

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
	assert.Equal(t, "v5.0.0", byRepo["actions/checkout"][0].Version, "Expected checkout pins sorted by newest version first")
	assert.Equal(t, "v4.0.0", byRepo["actions/checkout"][1].Version, "Expected checkout pins sorted by newest version first")

	require.Len(t, byRepo["actions/setup-go"], 2, "Expected setup-go pins to be grouped")
	assert.Equal(t, "v5.1.0", byRepo["actions/setup-go"][0].Version, "Expected setup-go pins sorted by newest version first")
	assert.Equal(t, "v5.0.0", byRepo["actions/setup-go"][1].Version, "Expected setup-go pins sorted by newest version first")
}

func TestCountPinKeyMismatches_ReturnsOnlyVersionMismatches(t *testing.T) {
	entries := map[string]ActionPin{
		"actions/checkout@v5": {Repo: "actions/checkout", Version: "v5", SHA: "sha-1"},
		"actions/setup-go@v5": {Repo: "actions/setup-go", Version: "v4", SHA: "sha-2"},
		"invalid-key":         {Repo: "actions/cache", Version: "v4", SHA: "sha-3"},
	}

	count := countPinKeyMismatches(entries)

	assert.Equal(t, 1, count, "Expected only one key/version mismatch to be counted")
}

func TestInitWarnings_InitializesAndPreservesMap(t *testing.T) {
	t.Run("initializes nil warnings map", func(t *testing.T) {
		ctx := &PinContext{}

		initWarnings(ctx)

		require.NotNil(t, ctx.Warnings, "Expected warnings map to be initialized")
		assert.Empty(t, ctx.Warnings, "Expected initialized warnings map to be empty")
	})

	t.Run("preserves existing warnings map", func(t *testing.T) {
		existing := map[string]bool{"actions/checkout@v5": true}
		ctx := &PinContext{Warnings: existing}

		initWarnings(ctx)

		require.NotNil(t, ctx.Warnings, "Expected warnings map to remain initialized")
		assert.Equal(t, existing, ctx.Warnings, "Expected existing warnings entries to be preserved")
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

func TestGetContainerPin_MCPGatewayV036IsPinned(t *testing.T) {
	const image = "ghcr.io/github/gh-aw-mcpg:v0.3.6"

	pin, ok := GetContainerPin(image)
	require.True(t, ok, "Expected embedded container pin for %s", image)
	assert.Equal(t, image, pin.Image, "Expected image name to match key")
	assert.Equal(t, "sha256:2bb8eef86006a4c5963c55616a9c51c32f27bfdecb023b8aa6f91f6718d9171c", pin.Digest, "Expected v0.3.6 digest to match")
	assert.Equal(t, image+"@sha256:2bb8eef86006a4c5963c55616a9c51c32f27bfdecb023b8aa6f91f6718d9171c", pin.PinnedImage, "Expected pinned image to include v0.3.6 digest")
}

func TestGetContainerPin_MCPGatewayV039IsPinned(t *testing.T) {
	const image = "ghcr.io/github/gh-aw-mcpg:v0.3.9"

	pin, ok := GetContainerPin(image)
	require.True(t, ok, "Expected embedded container pin for %s", image)
	assert.Equal(t, image, pin.Image, "Expected image name to match key")
	assert.Equal(t, "sha256:64828b42a4482f58fab16509d7f8f495a6d97c972a98a68aff20543531ac0388", pin.Digest, "Expected v0.3.9 digest to match")
	assert.Equal(t, image+"@sha256:64828b42a4482f58fab16509d7f8f495a6d97c972a98a68aff20543531ac0388", pin.PinnedImage, "Expected pinned image to include v0.3.9 digest")
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
	assert.Equal(t, 0, resolver.called, "Expected resolver not to be called for SHA input")
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
