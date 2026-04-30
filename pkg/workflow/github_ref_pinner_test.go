//go:build !integration

package workflow

import (
	"errors"
	"testing"

	actionpins "github.com/github/gh-aw/pkg/actionpins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLatestTagResolver implements both SHAResolver and latestTagResolver so it
// can be assigned to PinContext.Resolver to exercise the full fallback path.
// latestTagCalled records whether ResolveLatestTag was invoked.
type mockLatestTagResolver struct {
	latestTag       string
	latestSHA       string
	latestErr       error
	latestTagCalled bool
}

func (m *mockLatestTagResolver) ResolveSHA(_, version string) (string, error) {
	if version == m.latestTag && m.latestSHA != "" {
		return m.latestSHA, nil
	}
	return "", errors.New("not found")
}

func (m *mockLatestTagResolver) ResolveLatestTag(_ string) (tag, sha string, err error) {
	m.latestTagCalled = true
	if m.latestErr != nil {
		return "", "", m.latestErr
	}
	return m.latestTag, m.latestSHA, nil
}

func TestPinGitHubRef_FallsBackToLatestTag(t *testing.T) {
	// Use a resolver that knows about a "latest" release for a repo that has
	// no embedded action pins (so ResolveLatestActionPin returns "").
	resolver := &mockLatestTagResolver{
		latestTag: "v2.3.0",
		latestSHA: "aabbccddeeff00112233445566778899aabbccdd",
	}
	pinner := &pinContextGitHubRefPinner{
		ctx: &actionpins.PinContext{
			Resolver:        resolver,
			AllowActionRefs: true,
			Warnings:        make(map[string]bool),
		},
	}

	result := pinner.PinGitHubRef("microsoft/apm-sample-package")
	assert.Equal(t, "microsoft/apm-sample-package@aabbccddeeff00112233445566778899aabbccdd # v2.3.0", result,
		"should produce pinned reference via latest-tag fallback")
}

func TestPinGitHubRef_EmbeddedPinSkipsFallback(t *testing.T) {
	// actions-ecosystem/action-add-labels is present in the embedded action_pins.json.
	// ResolveLatestActionPin should succeed via the hardcoded pins, so
	// ResolveLatestTag must NOT be called.
	resolver := &mockLatestTagResolver{
		latestTag: "v99.0.0",
		latestSHA: "9999999999999999999999999999999999999999",
	}
	pinner := &pinContextGitHubRefPinner{
		ctx: &actionpins.PinContext{
			Resolver:        resolver,
			AllowActionRefs: true,
			Warnings:        make(map[string]bool),
		},
	}

	result := pinner.PinGitHubRef("actions-ecosystem/action-add-labels")
	// The result is the embedded pinned reference, not the mock's v99 tag.
	assert.Contains(t, result, "actions-ecosystem/action-add-labels@",
		"should return a pinned reference from embedded pins")
	assert.NotContains(t, result, "v99.0.0",
		"should not use the latest-tag fallback when embedded pins are available")
	assert.False(t, resolver.latestTagCalled,
		"ResolveLatestTag must not be called when embedded pins are available")
}

func TestPinGitHubRef_FallbackPreservesSubpath(t *testing.T) {
	resolver := &mockLatestTagResolver{
		latestTag: "v1.0.0",
		latestSHA: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
	}
	pinner := &pinContextGitHubRefPinner{
		ctx: &actionpins.PinContext{
			Resolver:        resolver,
			AllowActionRefs: true,
			Warnings:        make(map[string]bool),
		},
	}

	result := pinner.PinGitHubRef("owner/repo/skills/review")
	assert.Equal(t,
		"owner/repo/skills/review@deadbeefdeadbeefdeadbeefdeadbeefdeadbeef # v1.0.0",
		result,
		"subpath should be preserved in the pinned reference",
	)
}

func TestPinGitHubRef_FallbackReturnsOriginalOnError(t *testing.T) {
	resolver := &mockLatestTagResolver{
		latestErr: errors.New("no releases found"),
	}
	pinner := &pinContextGitHubRefPinner{
		ctx: &actionpins.PinContext{
			Resolver:        resolver,
			AllowActionRefs: true,
			Warnings:        make(map[string]bool),
		},
	}

	result := pinner.PinGitHubRef("microsoft/no-releases")
	assert.Equal(t, "microsoft/no-releases", result,
		"should return original value when latest-tag resolution fails")
}

func TestPinGitHubRef_ExplicitRefSkipsFallback(t *testing.T) {
	// When an explicit ref is provided, ResolveActionPin is used; the latest-tag
	// fallback must not be triggered even if the resolver supports it.
	resolver := &mockLatestTagResolver{
		latestTag: "v99.0.0",
		latestSHA: "9999999999999999999999999999999999999999",
	}
	pinner := &pinContextGitHubRefPinner{
		ctx: &actionpins.PinContext{
			Resolver:        resolver,
			AllowActionRefs: true,
			Warnings:        make(map[string]bool),
		},
	}

	// "owner/repo@v1.0.0" has an explicit ref — resolution goes through
	// ResolveActionPin (which will fail here because the SHA resolver returns
	// "not found" for "v1.0.0"), so the original value is returned unchanged.
	result := pinner.PinGitHubRef("owner/repo@v1.0.0")
	require.Equal(t, "owner/repo@v1.0.0", result,
		"should return original value when explicit-ref resolution fails, not invoke latest-tag fallback")
	assert.False(t, resolver.latestTagCalled,
		"ResolveLatestTag must not be called for explicit-ref inputs")
}

func TestPinGitHubRef_NoResolverReturnsOriginal(t *testing.T) {
	pinner := &pinContextGitHubRefPinner{
		ctx: &actionpins.PinContext{
			Resolver:        nil,
			AllowActionRefs: true,
			Warnings:        make(map[string]bool),
		},
	}

	result := pinner.PinGitHubRef("microsoft/apm-sample-package")
	assert.Equal(t, "microsoft/apm-sample-package", result,
		"should return original value when no resolver is set")
}
