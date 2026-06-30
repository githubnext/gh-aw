//go:build !integration

package actionpins_test

import (
	"context"
	"errors"
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
		{
			name:     "no @ separator returns full string",
			uses:     "actions/checkout",
			expected: "actions/checkout",
		},
		{
			name:     "empty string returns empty string",
			uses:     "",
			expected: "",
		},
		{
			name:     "leading @ returns empty repo",
			uses:     "@v4",
			expected: "",
		},
		{
			name:     "multiple @ separators returns part before first @",
			uses:     "actions/checkout@v4@extra",
			expected: "actions/checkout",
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
		{
			name:     "no @ separator returns empty string",
			uses:     "actions/checkout",
			expected: "",
		},
		{
			name:     "empty string returns empty string",
			uses:     "",
			expected: "",
		},
		{
			name:     "leading @ returns version part after @",
			uses:     "@v4",
			expected: "v4",
		},
		{
			name:     "multiple @ separators returns everything after first @",
			uses:     "actions/checkout@v4@extra",
			expected: "v4@extra",
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

// TestSpec_PublicAPI_GetLatestActionPinByRepo validates GetLatestActionPinByRepo returns the latest pin.
func TestSpec_PublicAPI_GetLatestActionPinByRepo(t *testing.T) {
	t.Run("returns false for unknown repository", func(t *testing.T) {
		_, ok := actionpins.GetLatestActionPinByRepo("does-not-exist/unknown-action-xyzzy")
		assert.False(t, ok, "should return false for unknown repo")
	})

	t.Run("returns a pin for a known repository", func(t *testing.T) {
		known := "actions/checkout"
		pin, ok := actionpins.GetLatestActionPinByRepo(known)
		require.True(t, ok, "should return true for a known repo")
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

// TestSpec_PublicAPI_ResolveActionPin_NilContext validates nil context fallback to embedded pins.
func TestSpec_PublicAPI_ResolveActionPin_NilContext(t *testing.T) {
	latestPin, ok := actionpins.GetLatestActionPinByRepo("actions/checkout")
	require.True(t, ok, "expected embedded pins for actions/checkout")

	result, err := actionpins.ResolveActionPin("actions/checkout", latestPin.Version, nil)
	require.NoError(t, err, "nil ctx should still resolve from embedded pins")
	assert.Equal(t,
		actionpins.FormatPinnedActionReference("actions/checkout", latestPin.SHA, latestPin.Version),
		result,
		"nil ctx should resolve from embedded pins with correct SHA and format")
}

// TestSpec_PublicAPI_ResolveActionPin_EnforcePinned validates unresolved pin handling in enforce mode.
func TestSpec_PublicAPI_ResolveActionPin_EnforcePinned(t *testing.T) {
	t.Run("returns error when EnforcePinned=true and pin is unresolved", func(t *testing.T) {
		var failures []actionpins.ResolutionFailure
		ctx := &actionpins.PinContext{
			EnforcePinned: true,
			Warnings:      make(map[string]bool),
			RecordResolutionFailure: func(f actionpins.ResolutionFailure) {
				failures = append(failures, f)
			},
		}
		_, err := actionpins.ResolveActionPin("does-not-exist/x", "v1", ctx)
		require.Error(t, err, "enforce mode should return an error when pin is unresolved")
		assert.Contains(t, err.Error(), "unable to pin action",
			"enforce mode error should mention unable to pin action")
		require.Len(t, failures, 1, "failure should be audited when enforce mode errors")
		assert.Equal(t, actionpins.ResolutionErrorTypePinNotFound, failures[0].ErrorType,
			"unresolved pin in enforce mode should audit pin_not_found")
	})

	t.Run("AllowActionRefs downgrades to warning with no error", func(t *testing.T) {
		ctx := &actionpins.PinContext{EnforcePinned: true, AllowActionRefs: true, Warnings: make(map[string]bool)}
		result, err := actionpins.ResolveActionPin("does-not-exist/x", "v1", ctx)
		require.NoError(t, err, "AllowActionRefs should downgrade unresolved pin enforcement to warning")
		assert.Empty(t, result, "AllowActionRefs downgrade should keep unresolved result empty")
		assert.True(t, ctx.Warnings[actionpins.FormatCacheKey("does-not-exist/x", "v1")],
			"AllowActionRefs should record warning dedup key")
	})

	t.Run("dynamic resolver fails with EnforcePinned=true returns error", func(t *testing.T) {
		var failures []actionpins.ResolutionFailure
		ctx := &actionpins.PinContext{
			Resolver:      &testSHAResolver{err: errors.New("network error")},
			EnforcePinned: true,
			Warnings:      make(map[string]bool),
			RecordResolutionFailure: func(f actionpins.ResolutionFailure) {
				failures = append(failures, f)
			},
		}
		_, err := actionpins.ResolveActionPin("does-not-exist/x", "v1", ctx)
		require.Error(t, err, "EnforcePinned should return an error when dynamic resolution fails")
		assert.Contains(t, err.Error(), "unable to pin action",
			"error message should mention unable to pin action")
		require.Len(t, failures, 1, "failure should be audited when dynamic resolution fails in enforce mode")
		assert.Equal(t, actionpins.ResolutionErrorTypeDynamicResolutionFailed, failures[0].ErrorType,
			"dynamic resolver failure should classify as dynamic_resolution_failed")
	})
}

// TestSpec_PublicAPI_ResolveActionPin_SkipHardcodedFallback validates that setting
// PinContext.SkipHardcodedFallback=true prevents the embedded hardcoded pins from
// being consulted, even for a well-known action that is present in the embedded data.
func TestSpec_PublicAPI_ResolveActionPin_SkipHardcodedFallback(t *testing.T) {
	t.Run("known action with SkipHardcodedFallback=true returns empty result", func(t *testing.T) {
		// actions/checkout has entries in the embedded pins.
		// With SkipHardcodedFallback=true and no dynamic resolver, resolution should
		// fall through without producing a pinned reference.
		ctx := &actionpins.PinContext{
			SkipHardcodedFallback: true,
			Warnings:              make(map[string]bool),
		}
		result, err := actionpins.ResolveActionPin("actions/checkout", "v4", ctx)
		require.NoError(t, err, "SkipHardcodedFallback should not cause an error")
		assert.Empty(t, result, "SkipHardcodedFallback=true should prevent hardcoded pin lookup")
	})

	t.Run("known action with SkipHardcodedFallback=false resolves normally", func(t *testing.T) {
		ctx := &actionpins.PinContext{
			SkipHardcodedFallback: false,
			Warnings:              make(map[string]bool),
		}
		result, err := actionpins.ResolveActionPin("actions/checkout", "v4", ctx)
		require.NoError(t, err, "hardcoded pin lookup should succeed when SkipHardcodedFallback is false")
		assert.NotEmpty(t, result, "SkipHardcodedFallback=false should allow hardcoded pin lookup")
		assert.Contains(t, result, "actions/checkout@", "result should reference actions/checkout")
	})
}

// TestSpec_PublicAPI_ResolveLatestActionPin validates latest-version resolution behavior.
func TestSpec_PublicAPI_ResolveLatestActionPin(t *testing.T) {
	t.Run("returns latest pinned reference for known repository", func(t *testing.T) {
		known := "actions/checkout"
		latestPin, ok := actionpins.GetLatestActionPinByRepo(known)
		require.True(t, ok, "expected latest pin for known repository")

		result := actionpins.ResolveLatestActionPin(known, nil)
		expected := actionpins.FormatPinnedActionReference(known, latestPin.SHA, latestPin.Version)
		assert.Equal(t, expected, result, "should resolve latest pinned reference")
	})

	t.Run("returns empty string for unknown repository", func(t *testing.T) {
		result := actionpins.ResolveLatestActionPin("does-not-exist/x", nil)
		assert.Empty(t, result, "unknown repo should return empty pin")
	})
}

// TestSpec_Types_PinContext validates the documented PinContext type fields.
func TestSpec_Types_PinContext(t *testing.T) {
	t.Run("strict mode disables non-exact fallback", func(t *testing.T) {
		ctx := &actionpins.PinContext{StrictMode: true, Warnings: make(map[string]bool)}
		result, err := actionpins.ResolveActionPin("actions/checkout", "v999", ctx)
		require.NoError(t, err)
		assert.Empty(t, result, "strict mode must not fall back to a non-exact version")
	})

	t.Run("nil resolver enables embedded-only lookup", func(t *testing.T) {
		latestPin, ok := actionpins.GetLatestActionPinByRepo("actions/checkout")
		require.True(t, ok, "expected embedded pins for actions/checkout")

		ctx := &actionpins.PinContext{Warnings: make(map[string]bool)}
		result, err := actionpins.ResolveActionPin("actions/checkout", latestPin.Version, ctx)
		require.NoError(t, err)
		assert.Equal(t,
			actionpins.FormatPinnedActionReference("actions/checkout", latestPin.SHA, latestPin.Version),
			result,
			"nil Resolver should use embedded pins")
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

	assert.Truef(t, strings.HasPrefix(cacheKey, repo+"@"), "cache key should be repo@version, got %q", cacheKey)
	assert.Truef(t, strings.HasPrefix(reference, repo+"@"), "reference should start with repo@sha, got %q", reference)
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
		Containers: map[string]actionpins.ContainerPin{
			"ghcr.io/example/image:latest": {
				Image:       "ghcr.io/example/image:latest",
				Digest:      "sha256:def456",
				PinnedImage: "ghcr.io/example/image@sha256:def456",
			},
		},
	}
	assert.Len(t, data.Entries, 1, "ActionPinsData.Entries should hold pin entries")
	entry := data.Entries["actions/checkout@v5"]
	assert.Equal(t, "actions/checkout", entry.Repo, "entry Repo should match")
	assert.Len(t, data.Containers, 1, "ActionPinsData.Containers should hold container pins")
}

// TestSpec_PublicAPI_ResolveActionPin_EmbeddedMatch validates embedded-only pin resolution returns
// a formatted reference for a known repository. Spec: "Embedded-only lookup from bundled pin data"
func TestSpec_PublicAPI_ResolveActionPin_EmbeddedMatch(t *testing.T) {
	known := "actions/checkout"
	latestPin, ok := actionpins.GetLatestActionPinByRepo(known)
	require.True(t, ok, "prerequisite: known repo must be in embedded data")

	ctx := &actionpins.PinContext{StrictMode: false, Warnings: make(map[string]bool)}
	result, err := actionpins.ResolveActionPin(known, latestPin.Version, ctx)
	require.NoError(t, err, "embedded-only ResolveActionPin should not error for known pin")
	assert.NotEmpty(t, result, "should return non-empty pinned reference for known embedded pin")
	assert.Contains(t, result, latestPin.SHA, "resolved reference should contain the pin SHA")
}

// testSHAResolver is a fake SHAResolver used in tests.
type testSHAResolver struct {
	sha string
	err error
}

func (r *testSHAResolver) ResolveSHA(_ context.Context, _, _ string) (string, error) {
	return r.sha, r.err
}

// TestSpec_DynamicResolution_VersionCommentConsistency validates that when dynamic resolution
// succeeds and the returned SHA matches an embedded pin, the version comment includes both
// the resolved version and the source version — consistent with the embedded-fallback path.
func TestSpec_DynamicResolution_VersionCommentConsistency(t *testing.T) {
	known := "actions/checkout"
	latestPin, ok := actionpins.GetLatestActionPinByRepo(known)
	require.True(t, ok, "prerequisite: known repo must be in embedded data")

	t.Run("shows resolved version and source version when SHA matches embedded pin", func(t *testing.T) {
		// Simulate dynamic resolution returning the same SHA as the embedded pin,
		// but requested with a shorter version tag (e.g. "v4" instead of "v4.1.2").
		sourceVersion := "v4"
		ctx := &actionpins.PinContext{
			Resolver: &testSHAResolver{sha: latestPin.SHA},
			Warnings: make(map[string]bool),
		}
		result, err := actionpins.ResolveActionPin(known, sourceVersion, ctx)
		require.NoError(t, err)
		assert.Contains(t, result, latestPin.SHA, "result should contain the resolved SHA")
		assert.Contains(t, result, latestPin.Version, "result should contain the resolved version")
		assert.Contains(t, result, sourceVersion, "result should contain the source version")
	})

	t.Run("shows only source version when SHA is not in embedded pins", func(t *testing.T) {
		unknownSHA := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		sourceVersion := "v4"
		ctx := &actionpins.PinContext{
			Resolver: &testSHAResolver{sha: unknownSHA},
			Warnings: make(map[string]bool),
		}
		result, err := actionpins.ResolveActionPin(known, sourceVersion, ctx)
		require.NoError(t, err)
		assert.Contains(t, result, unknownSHA, "result should contain the resolved SHA")
		assert.Contains(t, result, sourceVersion, "result should contain the source version")
	})

	t.Run("skips version comment when version is already a SHA", func(t *testing.T) {
		ctx := &actionpins.PinContext{Warnings: make(map[string]bool)}
		result, err := actionpins.ResolveActionPin(known, latestPin.SHA, ctx)
		require.NoError(t, err)
		assert.Contains(t, result, latestPin.SHA, "result should contain the SHA")
		// Resolver is not called for SHA inputs; only version comment content matters
	})
}

// TestSpec_PublicAPI_GetContainerPin validates the documented GetContainerPin function.
// Spec: "Returns a pinned container image by its original image reference"
func TestSpec_PublicAPI_GetContainerPin(t *testing.T) {
	t.Run("returns false for unknown container image", func(t *testing.T) {
		_, ok := actionpins.GetContainerPin("does-not-exist/unknown-image:latest")
		assert.False(t, ok, "should return false for unknown container image")
	})

	t.Run("returns pinned container for known image", func(t *testing.T) {
		// "alpine:latest" is present in the embedded action_pins.json containers map.
		pin, ok := actionpins.GetContainerPin("alpine:latest")
		require.True(t, ok, "should return true for a known container image")
		assert.Equal(t, "alpine:latest", pin.Image, "ContainerPin.Image should match the queried image")
		require.NotEmpty(t, pin.Digest, "ContainerPin.Digest should be non-empty for a known image")
		assert.NotEmpty(t, pin.PinnedImage, "ContainerPin.PinnedImage should be non-empty for a known image")
		assert.Contains(t, pin.PinnedImage, pin.Digest, "PinnedImage should contain the digest")
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

// TestSpec_Constants_ResolutionErrorType validates the documented ResolutionErrorType constant values.
// Spec table: ResolutionErrorTypeDynamicResolutionFailed="dynamic_resolution_failed",
// ResolutionErrorTypePinNotFound="pin_not_found".
func TestSpec_Constants_ResolutionErrorType(t *testing.T) {
	assert.Equal(t, "dynamic_resolution_failed", string(actionpins.ResolutionErrorTypeDynamicResolutionFailed),
		"ResolutionErrorTypeDynamicResolutionFailed should equal the documented value")
	assert.Equal(t, "pin_not_found", string(actionpins.ResolutionErrorTypePinNotFound),
		"ResolutionErrorTypePinNotFound should equal the documented value")
}

// TestSpec_Types_ResolutionFailure validates the documented ResolutionFailure type structure.
// Spec: "Captures an unresolved action-ref pinning event (repo, ref, error type)".
func TestSpec_Types_ResolutionFailure(t *testing.T) {
	failure := actionpins.ResolutionFailure{
		Repo:      "unknown/action",
		Ref:       "v1",
		ErrorType: actionpins.ResolutionErrorTypePinNotFound,
	}
	assert.Equal(t, "unknown/action", failure.Repo, "ResolutionFailure.Repo field")
	assert.Equal(t, "v1", failure.Ref, "ResolutionFailure.Ref field")
	assert.Equal(t, actionpins.ResolutionErrorTypePinNotFound, failure.ErrorType, "ResolutionFailure.ErrorType field")
}

// TestSpec_PublicAPI_RecordResolutionFailure validates the documented auditing behavior:
// PinContext.RecordResolutionFailure collects ResolutionFailure events for unresolved pins,
// classified with ResolutionErrorTypePinNotFound when no usable pin is found.
// Spec section "Auditing Resolution Failures".
func TestSpec_PublicAPI_RecordResolutionFailure(t *testing.T) {
	var failures []actionpins.ResolutionFailure
	ctx := &actionpins.PinContext{
		Warnings: make(map[string]bool),
		RecordResolutionFailure: func(f actionpins.ResolutionFailure) {
			failures = append(failures, f)
		},
	}

	_, err := actionpins.ResolveActionPin("does-not-exist/unknown-action-xyzzy", "v1", ctx)
	require.NoError(t, err, "ResolveActionPin should not error even when the pin is unresolved")

	require.Len(t, failures, 1, "RecordResolutionFailure should be invoked once for an unresolved pin")
	assert.Equal(t, actionpins.ResolutionErrorTypePinNotFound, failures[0].ErrorType,
		"unresolved pin with no resolver should be classified as pin_not_found")
	assert.Equal(t, "does-not-exist/unknown-action-xyzzy", failures[0].Repo,
		"recorded failure should carry the queried repo")
	assert.Equal(t, "v1", failures[0].Ref, "recorded failure should carry the queried ref")
}

// TestSpec_PublicAPI_RecordResolutionFailure_DynamicFailed validates dynamic-resolution failure auditing.
func TestSpec_PublicAPI_RecordResolutionFailure_DynamicFailed(t *testing.T) {
	var failures []actionpins.ResolutionFailure
	ctx := &actionpins.PinContext{
		Resolver: &testSHAResolver{err: errors.New("network error")},
		Warnings: make(map[string]bool),
		RecordResolutionFailure: func(f actionpins.ResolutionFailure) {
			failures = append(failures, f)
		},
	}

	_, err := actionpins.ResolveActionPin("does-not-exist/x", "v1", ctx)
	require.NoError(t, err, "dynamic resolver failures should be audited and downgraded to unresolved pin")
	require.Len(t, failures, 1, "expected one resolution failure to be recorded")
	assert.Equal(t, actionpins.ResolutionErrorTypeDynamicResolutionFailed, failures[0].ErrorType,
		"resolver error should classify as dynamic_resolution_failed")
	assert.Equal(t, "does-not-exist/x", failures[0].Repo, "recorded failure should carry the queried repo")
	assert.Equal(t, "v1", failures[0].Ref, "recorded failure should carry the queried ref")
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

	require.NotEmpty(t, results[0], "baseline goroutine 0 should return pins")
	for i := 1; i < goroutines; i++ {
		assert.NotEmpty(t, results[i], "concurrent GetActionPinsByRepo should return pins for known repo")
		assert.Lenf(t, results[i], len(results[0]),
			"concurrent GetActionPinsByRepo should return same number of pins (goroutine %d vs 0)", i)
	}
}

// TestSpec_PublicAPI_ResolveActionPin_DynamicHappyPath validates that a dynamic resolver that
// successfully returns a SHA produces a correctly formatted pinned reference.
func TestSpec_PublicAPI_ResolveActionPin_DynamicHappyPath(t *testing.T) {
	known := "actions/checkout"
	latestPin, ok := actionpins.GetLatestActionPinByRepo(known)
	require.True(t, ok, "prerequisite: known repo must be in embedded data")

	// Resolver returns the SHA of the latest embedded pin so that the resolved
	// version comment can be verified end-to-end.
	ctx := &actionpins.PinContext{
		Resolver: &testSHAResolver{sha: latestPin.SHA},
		Warnings: make(map[string]bool),
	}
	result, err := actionpins.ResolveActionPin(known, "v4", ctx)
	require.NoError(t, err, "dynamic resolver success should not return an error")
	assert.NotEmpty(t, result, "should return a non-empty pinned reference when resolver succeeds")
	assert.Contains(t, result, latestPin.SHA, "result should contain the SHA returned by the resolver")
	assert.True(t, strings.HasPrefix(result, known+"@"),
		"result should start with repo@sha in the documented format")
}

// TestSpec_PublicAPI_ResolveLatestActionPin_NonNilContext validates that a non-nil PinContext
// is forwarded correctly to the embedded pin resolution path.
func TestSpec_PublicAPI_ResolveLatestActionPin_NonNilContext(t *testing.T) {
	known := "actions/checkout"
	latestPin, ok := actionpins.GetLatestActionPinByRepo(known)
	require.True(t, ok, "prerequisite: known repo must be in embedded data")

	ctx := &actionpins.PinContext{Warnings: make(map[string]bool)}
	result := actionpins.ResolveLatestActionPin(known, ctx)
	expected := actionpins.FormatPinnedActionReference(known, latestPin.SHA, latestPin.Version)
	assert.Equal(t, expected, result,
		"non-nil ctx without a resolver should resolve the same reference as the nil-ctx path")
}

// TestSpec_PublicAPI_RecordResolutionFailure_WarningDedup validates that repeated resolution
// failures for the same repo@version emit the warning only once (Warnings map deduplication).
func TestSpec_PublicAPI_RecordResolutionFailure_WarningDedup(t *testing.T) {
	var failures []actionpins.ResolutionFailure
	ctx := &actionpins.PinContext{
		Warnings: make(map[string]bool),
		RecordResolutionFailure: func(f actionpins.ResolutionFailure) {
			failures = append(failures, f)
		},
	}

	repo, version := "does-not-exist/x", "v1"
	cacheKey := actionpins.FormatCacheKey(repo, version)

	// First call: failure is recorded and warning key is set.
	_, err := actionpins.ResolveActionPin(repo, version, ctx)
	require.NoError(t, err)
	require.Len(t, failures, 1, "first call should record one resolution failure")
	assert.True(t, ctx.Warnings[cacheKey], "warning key should be set after first call")

	// Second call with the same args: failure is recorded again (auditing is not
	// deduplicated), but the warning is suppressed by the Warnings map.
	_, err = actionpins.ResolveActionPin(repo, version, ctx)
	require.NoError(t, err)
	assert.Len(t, failures, 2, "second call should append another resolution failure")
	assert.True(t, ctx.Warnings[cacheKey], "warning dedup key should remain set after second call")
}

// TestSpec_PublicAPI_ResolveActionPin_AppliesMapping validates that ctx.Mappings redirects
// action resolution to the mapped repository and version.
func TestSpec_PublicAPI_ResolveActionPin_AppliesMapping(t *testing.T) {
	// actions/checkout is in the embedded pins; acme-corp/checkout is not.
	// After mapping, resolution should succeed using the mapped repo's pins.
	checkoutPins := actionpins.GetActionPinsByRepo("actions/checkout")
	require.NotEmpty(t, checkoutPins, "prerequisite: embedded pins for actions/checkout must exist")

	t.Run("mapping redirects to a known pinned repo", func(t *testing.T) {
		ctx := &actionpins.PinContext{
			Warnings: make(map[string]bool),
			Mappings: map[string]string{
				"actions/setup-node@v4": "actions/checkout@v4",
			},
		}
		// Request setup-node@v4 which is mapped to checkout@v4 — both exist in embedded pins.
		result, err := actionpins.ResolveActionPin("actions/setup-node", "v4", ctx)
		require.NoError(t, err)
		assert.Contains(t, result, "actions/checkout@", "result should reference the mapped repo")
	})

	t.Run("mapping notification key is recorded in warnings", func(t *testing.T) {
		ctx := &actionpins.PinContext{
			Warnings: make(map[string]bool),
			Mappings: map[string]string{
				"actions/setup-node@v4": "actions/checkout@v4",
			},
		}
		_, _ = actionpins.ResolveActionPin("actions/setup-node", "v4", ctx)
		assert.True(t, ctx.Warnings["map:actions/setup-node@v4"], "mapping notification key should be set in warnings")
	})

	t.Run("no mapping leaves resolution unchanged", func(t *testing.T) {
		ctx := &actionpins.PinContext{Warnings: make(map[string]bool)}

		result, err := actionpins.ResolveActionPin("actions/checkout", "v4", ctx)
		require.NoError(t, err)
		assert.Contains(t, result, "actions/checkout@", "result should reference the original repo when no mapping exists")
	})
}
