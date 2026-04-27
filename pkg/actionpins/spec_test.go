//go:build !integration

package actionpins_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/actionpins"
)

// TestSpec_PublicAPI_FormatPinnedActionReference validates the documented format "repo@sha # version".
func TestSpec_PublicAPI_FormatPinnedActionReference(t *testing.T) {
	tests := []struct {
		name     string
		repo     string
		sha      string
		version  string
		expected string
	}{
		{
			name:     "formats standard reference",
			repo:     "actions/checkout",
			sha:      "abc123",
			version:  "v4",
			expected: "actions/checkout@abc123 # v4",
		},
		{
			name:     "formats reference with full 40-char sha",
			repo:     "actions/setup-go",
			sha:      "cdabf2d4679a00bef48b5a7c69a9b8d0b4f6e3c9",
			version:  "v5",
			expected: "actions/setup-go@cdabf2d4679a00bef48b5a7c69a9b8d0b4f6e3c9 # v5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := actionpins.FormatPinnedActionReference(tt.repo, tt.sha, tt.version)
			assert.Equal(t, tt.expected, result, "FormatPinnedActionReference(%q, %q, %q) should match spec format", tt.repo, tt.sha, tt.version)
		})
	}
}

// TestSpec_PublicAPI_FormatCacheKey validates the documented format "repo@version".
func TestSpec_PublicAPI_FormatCacheKey(t *testing.T) {
	tests := []struct {
		name     string
		repo     string
		version  string
		expected string
	}{
		{
			name:     "formats cache key as repo@version",
			repo:     "actions/checkout",
			version:  "v4",
			expected: "actions/checkout@v4",
		},
		{
			name:     "formats cache key with full semver",
			repo:     "actions/setup-node",
			version:  "v3.0.0",
			expected: "actions/setup-node@v3.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := actionpins.FormatCacheKey(tt.repo, tt.version)
			assert.Equal(t, tt.expected, result, "FormatCacheKey(%q, %q) should match spec format", tt.repo, tt.version)
		})
	}
}

// TestSpec_PublicAPI_ExtractRepo validates extracting the repository from a uses reference.
func TestSpec_PublicAPI_ExtractRepo(t *testing.T) {
	tests := []struct {
		name     string
		uses     string
		expected string
	}{
		{
			name:     "extracts repo from tag reference",
			uses:     "actions/checkout@v4",
			expected: "actions/checkout",
		},
		{
			name:     "extracts repo from sha reference",
			uses:     "actions/setup-go@cdabf2d4679a00bef48b5a7c69a9b8d0b4f6e3c9",
			expected: "actions/setup-go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := actionpins.ExtractRepo(tt.uses)
			assert.Equal(t, tt.expected, result, "ExtractRepo(%q) should return repo part", tt.uses)
		})
	}
}

// TestSpec_PublicAPI_ExtractVersion validates extracting the version from a uses reference.
func TestSpec_PublicAPI_ExtractVersion(t *testing.T) {
	tests := []struct {
		name     string
		uses     string
		expected string
	}{
		{
			name:     "extracts tag version",
			uses:     "actions/checkout@v4",
			expected: "v4",
		},
		{
			name:     "extracts sha version",
			uses:     "actions/setup-go@abc123def456",
			expected: "abc123def456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := actionpins.ExtractVersion(tt.uses)
			assert.Equal(t, tt.expected, result, "ExtractVersion(%q) should return version part", tt.uses)
		})
	}
}

// TestSpec_PublicAPI_GetActionPinsByRepo validates GetActionPinsByRepo for known and unknown repos.
func TestSpec_PublicAPI_GetActionPinsByRepo(t *testing.T) {
	t.Run("returns no pins for unknown repository", func(t *testing.T) {
		// SPEC_MISMATCH: spec implies a non-nil slice but implementation returns nil from map lookup.
		pins := actionpins.GetActionPinsByRepo("does-not-exist/unknown-action-xyzzy")
		assert.Empty(t, pins, "should return empty result for unknown repo")
	})

	t.Run("returns pins for a known repository when embedded data is loaded", func(t *testing.T) {
		known := "actions/checkout"
		pins := actionpins.GetActionPinsByRepo(known)
		assert.NotEmpty(t, pins, "should return pins for a known repo from embedded data")
	})
}

// TestSpec_PublicAPI_GetActionPinByRepo validates GetActionPinByRepo returns the latest pin.
func TestSpec_PublicAPI_GetActionPinByRepo(t *testing.T) {
	t.Run("returns false for unknown repository", func(t *testing.T) {
		_, ok := actionpins.GetActionPinByRepo("does-not-exist/unknown-action-xyzzy")
		assert.False(t, ok, "should return false for unknown repo")
	})

	t.Run("returns a pin for a known repository", func(t *testing.T) {
		known := "actions/checkout"
		pin, ok := actionpins.GetActionPinByRepo(known)
		assert.True(t, ok, "should return true for a known repo")
		assert.Equal(t, known, pin.Repo, "returned pin should belong to the queried repo")
	})
}

// TestSpec_PublicAPI_ResolveActionPin validates resolution behavior.
// Spec: "fallback behavior controlled by PinContext.StrictMode"
func TestSpec_PublicAPI_ResolveActionPin(t *testing.T) {
	t.Run("strict mode returns empty string and no error when pin is not found", func(t *testing.T) {
		// SPEC_MISMATCH: spec implies StrictMode causes an error on missing pins, but the
		// implementation returns ("", nil) and emits a warning to stderr instead.
		ctx := &actionpins.PinContext{StrictMode: true, Warnings: make(map[string]bool)}
		result, err := actionpins.ResolveActionPin("does-not-exist/unknown-action-xyzzy", "v1", ctx)
		require.NoError(t, err, "implementation returns no error even in strict mode for unknown pin")
		assert.Empty(t, result, "strict mode should return empty reference for unknown pin")
	})
}

// TestSpec_PublicAPI_ResolveLatestActionPin validates latest-version resolution behavior.
func TestSpec_PublicAPI_ResolveLatestActionPin(t *testing.T) {
	t.Run("returns latest pinned reference for known repository", func(t *testing.T) {
		known := "actions/checkout"
		latestPin, ok := actionpins.GetActionPinByRepo(known)
		require.True(t, ok, "expected latest pin for known repository")

		result := actionpins.ResolveLatestActionPin(known, nil)
		expected := actionpins.FormatPinnedActionReference(known, latestPin.SHA, latestPin.Version)
		assert.Equal(t, expected, result, "should resolve latest pinned reference")
	})
}

// TestSpec_Types_PinContext validates the documented PinContext type fields.
func TestSpec_Types_PinContext(t *testing.T) {
	t.Run("can construct PinContext with StrictMode enabled", func(t *testing.T) {
		ctx := &actionpins.PinContext{StrictMode: true}
		assert.NotNil(t, ctx)
	})

	t.Run("can construct PinContext without resolver for embedded-only lookup", func(t *testing.T) {
		ctx := &actionpins.PinContext{}
		assert.NotNil(t, ctx)
		assert.Nil(t, ctx.Resolver, "nil Resolver enables embedded-only lookup")
	})
}

// TestSpec_DesignDecision_FormatConsistency validates that FormatPinnedActionReference and FormatCacheKey
// produce outputs consistent with the spec: cacheKey = "repo@version", ref = "repo@sha # version".
func TestSpec_DesignDecision_FormatConsistency(t *testing.T) {
	repo := "actions/checkout"
	version := "v4"
	sha := "deadbeef"

	cacheKey := actionpins.FormatCacheKey(repo, version)
	reference := actionpins.FormatPinnedActionReference(repo, sha, version)

	assert.True(t, strings.HasPrefix(cacheKey, repo+"@"), "cache key should be repo@version")
	assert.True(t, strings.HasPrefix(reference, repo+"@"), "reference should start with repo@sha")
	assert.Contains(t, cacheKey, version, "cache key should contain version")
	assert.Contains(t, reference, sha, "reference should contain sha")
	assert.Contains(t, reference, version, "reference should contain version comment")
}

// TestSpec_Types_ActionPin validates the documented ActionPin type structure.
// Spec: Repo, Version, SHA fields plus optional Inputs map.
func TestSpec_Types_ActionPin(t *testing.T) {
	pin := actionpins.ActionPin{
		Repo:    "actions/checkout",
		Version: "v5",
		SHA:     "abcdef1234567890abcdef1234567890abcdef12",
	}
	assert.Equal(t, "actions/checkout", pin.Repo, "ActionPin.Repo field")
	assert.Equal(t, "v5", pin.Version, "ActionPin.Version field")
	assert.Equal(t, "abcdef1234567890abcdef1234567890abcdef12", pin.SHA, "ActionPin.SHA field")
	assert.Nil(t, pin.Inputs, "ActionPin.Inputs should be nil when not set")
}

// TestSpec_Types_ActionYAMLInput validates the documented ActionYAMLInput type structure.
// Spec: Description, Required, Default fields.
func TestSpec_Types_ActionYAMLInput(t *testing.T) {
	input := actionpins.ActionYAMLInput{
		Description: "The branch to checkout",
		Required:    true,
		Default:     "main",
	}
	assert.Equal(t, "The branch to checkout", input.Description, "ActionYAMLInput.Description field")
	assert.True(t, input.Required, "ActionYAMLInput.Required field")
	assert.Equal(t, "main", input.Default, "ActionYAMLInput.Default field")
}

// TestSpec_Types_ActionPinsData validates the documented ActionPinsData container type.
// Spec: ActionPinsData is a JSON container used to load embedded pin entries.
func TestSpec_Types_ActionPinsData(t *testing.T) {
	data := actionpins.ActionPinsData{
		Entries: map[string]actionpins.ActionPin{
			"actions/checkout@v5": {Repo: "actions/checkout", Version: "v5", SHA: "abc123"},
		},
	}
	assert.Len(t, data.Entries, 1, "ActionPinsData.Entries should hold pin entries")
	entry := data.Entries["actions/checkout@v5"]
	assert.Equal(t, "actions/checkout", entry.Repo, "entry Repo should match")
}

// TestSpec_PublicAPI_ResolveActionPin_EmbeddedMatch validates embedded-only pin resolution returns
// a formatted reference for a known repository. Spec: "Embedded-only lookup from bundled pin data"
func TestSpec_PublicAPI_ResolveActionPin_EmbeddedMatch(t *testing.T) {
	known := "actions/checkout"
	latestPin, ok := actionpins.GetActionPinByRepo(known)
	require.True(t, ok, "prerequisite: known repo must be in embedded data")

	ctx := &actionpins.PinContext{StrictMode: false, Warnings: make(map[string]bool)}
	result, err := actionpins.ResolveActionPin(known, latestPin.Version, ctx)
	require.NoError(t, err, "embedded-only ResolveActionPin should not error for known pin")
	assert.NotEmpty(t, result, "should return non-empty pinned reference for known embedded pin")
	assert.Contains(t, result, latestPin.SHA, "resolved reference should contain the pin SHA")
}

// TestSpec_PublicAPI_GetActionPins_SPEC_MISMATCH documents a spec-implementation gap.
// SPEC_MISMATCH: The README specifies GetActionPins() []ActionPin ("Returns all loaded pins")
// but this function is not implemented. Only GetActionPinsByRepo(repo string) is available.
// Proxy validation: verify embedded data is non-empty via the available API.
func TestSpec_PublicAPI_GetActionPins_SPEC_MISMATCH(t *testing.T) {
	// SPEC_MISMATCH: GetActionPins() documented in README does not exist in the implementation.
	pins := actionpins.GetActionPinsByRepo("actions/checkout")
	assert.NotEmpty(t, pins, "embedded pin data should be non-empty (proxy for missing GetActionPins)")
}

// TestSpec_PublicAPI_GetContainerPin validates the documented GetContainerPin function.
// Spec: "Returns a pinned container image by its original image reference"
func TestSpec_PublicAPI_GetContainerPin(t *testing.T) {
	t.Run("returns false for unknown container image", func(t *testing.T) {
		_, ok := actionpins.GetContainerPin("does-not-exist/unknown-image:latest")
		assert.False(t, ok, "should return false for unknown container image")
	})
}

// TestSpec_Types_ContainerPin validates the documented ContainerPin type structure.
// Spec: Image, Digest, PinnedImage fields.
func TestSpec_Types_ContainerPin(t *testing.T) {
	pin := actionpins.ContainerPin{
		Image:       "ghcr.io/some/image:v1",
		Digest:      "sha256:abc123",
		PinnedImage: "ghcr.io/some/image@sha256:abc123",
	}
	assert.Equal(t, "ghcr.io/some/image:v1", pin.Image, "ContainerPin.Image field")
	assert.Equal(t, "sha256:abc123", pin.Digest, "ContainerPin.Digest field")
	assert.Equal(t, "ghcr.io/some/image@sha256:abc123", pin.PinnedImage, "ContainerPin.PinnedImage field")
}

// TestSpec_ThreadSafety_ConcurrentGetActionPinsByRepo validates that concurrent calls to GetActionPinsByRepo
// are safe after initialization (sync.Once guarantee from the spec).
func TestSpec_ThreadSafety_ConcurrentGetActionPinsByRepo(t *testing.T) {
	const goroutines = 10
	const repo = "actions/checkout"
	results := make([][]actionpins.ActionPin, goroutines)
	done := make(chan int, goroutines)

	for i := range goroutines {
		go func(idx int) {
			results[idx] = actionpins.GetActionPinsByRepo(repo)
			done <- idx
		}(i)
	}

	for range goroutines {
		<-done
	}

	for i := 1; i < goroutines; i++ {
		assert.NotEmpty(t, results[i], "concurrent GetActionPinsByRepo should return pins for known repo")
		assert.Len(t, results[i], len(results[0]),
			"concurrent GetActionPinsByRepo should return same number of pins (goroutine %d vs 0)", i)
	}
	assert.NotEmpty(t, results[0], "concurrent GetActionPinsByRepo should return pins for known repo")
}
