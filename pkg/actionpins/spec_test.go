//go:build !integration

package actionpins_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/github/gh-aw/pkg/actionpins"
)

// TestSpec_PublicAPI_FormatReference validates the documented format "repo@sha # version".
func TestSpec_PublicAPI_FormatReference(t *testing.T) {
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
			result := actionpins.FormatReference(tt.repo, tt.sha, tt.version)
			assert.Equal(t, tt.expected, result, "FormatReference(%q, %q, %q) should match spec format", tt.repo, tt.sha, tt.version)
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

// TestSpec_PublicAPI_GetActionPins validates that GetActionPins returns a non-nil slice.
func TestSpec_PublicAPI_GetActionPins(t *testing.T) {
	pins := actionpins.GetActionPins()
	assert.NotNil(t, pins, "GetActionPins should return non-nil slice of all loaded pins")
}

// TestSpec_PublicAPI_GetActionPinsByRepo validates GetActionPinsByRepo for known and unknown repos.
func TestSpec_PublicAPI_GetActionPinsByRepo(t *testing.T) {
	t.Run("returns no pins for unknown repository", func(t *testing.T) {
		// SPEC_MISMATCH: spec implies a non-nil slice but implementation returns nil from map lookup.
		pins := actionpins.GetActionPinsByRepo("does-not-exist/unknown-action-xyzzy")
		assert.Empty(t, pins, "should return empty result for unknown repo")
	})

	t.Run("returns pins for a known repository when embedded data is loaded", func(t *testing.T) {
		all := actionpins.GetActionPins()
		if len(all) == 0 {
			t.Skip("no embedded pin data available")
		}
		known := all[0].Repo
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
		all := actionpins.GetActionPins()
		if len(all) == 0 {
			t.Skip("no embedded pin data available")
		}
		known := all[0].Repo
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
		assert.NoError(t, err, "implementation returns no error even in strict mode for unknown pin")
		assert.Empty(t, result, "strict mode should return empty reference for unknown pin")
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

// TestSpec_DesignDecision_FormatConsistency validates that FormatReference and FormatCacheKey
// produce outputs consistent with the spec: cacheKey = "repo@version", ref = "repo@sha # version".
func TestSpec_DesignDecision_FormatConsistency(t *testing.T) {
	repo := "actions/checkout"
	version := "v4"
	sha := "deadbeef"

	cacheKey := actionpins.FormatCacheKey(repo, version)
	reference := actionpins.FormatReference(repo, sha, version)

	assert.True(t, strings.HasPrefix(cacheKey, repo+"@"), "cache key should be repo@version")
	assert.True(t, strings.HasPrefix(reference, repo+"@"), "reference should start with repo@sha")
	assert.Contains(t, cacheKey, version, "cache key should contain version")
	assert.Contains(t, reference, sha, "reference should contain sha")
	assert.Contains(t, reference, version, "reference should contain version comment")
}
