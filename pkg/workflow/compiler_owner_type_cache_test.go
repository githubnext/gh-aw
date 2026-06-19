//go:build !integration

package workflow

import (
	"testing"
)

// TestRepositoryOwnerIsIndividualUser_CacheHit verifies that repositoryOwnerIsIndividualUser
// returns the cached owner type without making a network call when the owner is already
// in the cache. This ensures the owner-type check is performed at most once per repo
// during a single compilation run.
func TestRepositoryOwnerIsIndividualUser_CacheHit(t *testing.T) {
	tests := []struct {
		name           string
		cachedType     string
		expectedResult bool
	}{
		{
			name:           "cached User type returns true",
			cachedType:     "User",
			expectedResult: true,
		},
		{
			name:           "cached Organization type returns false",
			cachedType:     "Organization",
			expectedResult: false,
		},
		{
			name:           "cached empty string (API error) returns false",
			cachedType:     "",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler()
			c.SetRepositorySlug("myowner/myrepo")

			// Pre-populate the cache so no network call is made.
			// If the cache is not consulted, RunGH would be called without a real gh binary
			// available in unit tests, causing the function to return false even for the
			// "User" case — the test would then fail on that case.
			c.ownerTypeCache["myowner"] = tt.cachedType

			got := c.repositoryOwnerIsIndividualUser()
			if got != tt.expectedResult {
				t.Errorf("repositoryOwnerIsIndividualUser() = %v, want %v (cached owner type %q)",
					got, tt.expectedResult, tt.cachedType)
			}
		})
	}
}

// TestRepositoryOwnerIsIndividualUser_MalformedSlug verifies that the function
// returns false immediately when repositorySlug is missing or malformed, without
// consulting the cache or making a network call.
func TestRepositoryOwnerIsIndividualUser_MalformedSlug(t *testing.T) {
	tests := []struct {
		name string
		slug string
	}{
		{name: "empty slug", slug: ""},
		{name: "no slash", slug: "owneronly"},
		{name: "trailing slash only", slug: "/"},
		{name: "missing owner", slug: "/repo"},
		{name: "missing repo", slug: "owner/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler()
			c.SetRepositorySlug(tt.slug)

			got := c.repositoryOwnerIsIndividualUser()
			if got {
				t.Errorf("repositoryOwnerIsIndividualUser() = true for malformed slug %q, want false", tt.slug)
			}
		})
	}
}

// TestRepositoryOwnerIsIndividualUser_CacheSharedAcrossCompilations verifies that
// the owner-type cache persists across multiple calls on the same Compiler instance,
// which represents multiple workflow files compiled in a single `gh aw compile` run.
func TestRepositoryOwnerIsIndividualUser_CacheSharedAcrossCompilations(t *testing.T) {
	c := NewCompiler()
	c.SetRepositorySlug("myorg/repo-a")

	// Seed the cache as if a previous call already resolved the owner type.
	c.ownerTypeCache["myorg"] = "Organization"

	// Simulate compiling a second workflow in the same repo (different repo name, same owner).
	c.SetRepositorySlugIfUnlocked("myorg/repo-b")

	// The cache entry for "myorg" must be reused — no network call is made.
	got := c.repositoryOwnerIsIndividualUser()
	if got {
		t.Error("repositoryOwnerIsIndividualUser() = true for Organization owner, want false")
	}

	// The cache should still hold the original value (not been cleared or overwritten).
	if val := c.ownerTypeCache["myorg"]; val != "Organization" {
		t.Errorf("ownerTypeCache[myorg] = %q after second call, want %q", val, "Organization")
	}
}

// TestRepositoryOwnerIsIndividualUser_CacheInitializedByNewCompiler verifies that
// NewCompiler initializes ownerTypeCache so callers never encounter a nil map panic.
func TestRepositoryOwnerIsIndividualUser_CacheInitializedByNewCompiler(t *testing.T) {
	c := NewCompiler()
	if c.ownerTypeCache == nil {
		t.Fatal("NewCompiler() left ownerTypeCache nil; expected an initialized map")
	}
}
