//go:build !integration

package actionpins

import (
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
