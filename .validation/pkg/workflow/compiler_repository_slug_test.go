//go:build !integration

package workflow

import (
	"testing"
)

// TestRepositorySlugLock verifies that LockRepositorySlug prevents
// SetRepositorySlugIfUnlocked from overwriting the slug.
func TestRepositorySlugLock(t *testing.T) {
	c := NewCompiler()

	// Initially not locked; slug can be set freely.
	c.SetRepositorySlug("github/gh-aw")
	if got := c.GetRepositorySlug(); got != "github/gh-aw" {
		t.Fatalf("expected github/gh-aw, got %q", got)
	}
	if c.IsRepositorySlugLocked() {
		t.Fatal("slug should not be locked before LockRepositorySlug is called")
	}

	// Lock the slug.
	c.LockRepositorySlug()
	if !c.IsRepositorySlugLocked() {
		t.Fatal("slug should be locked after LockRepositorySlug is called")
	}

	// SetRepositorySlugIfUnlocked must not overwrite a locked slug.
	c.SetRepositorySlugIfUnlocked("trask/gh-aw")
	if got := c.GetRepositorySlug(); got != "github/gh-aw" {
		t.Fatalf("locked slug should not have changed; expected github/gh-aw, got %q", got)
	}

	// SetRepositorySlug (unconditional) still works regardless of lock.
	c.SetRepositorySlug("other/repo")
	if got := c.GetRepositorySlug(); got != "other/repo" {
		t.Fatalf("SetRepositorySlug should always overwrite; expected other/repo, got %q", got)
	}
}

// TestSetRepositorySlugIfUnlocked_WhenUnlocked verifies that when the slug is not
// locked, SetRepositorySlugIfUnlocked updates it normally.
func TestSetRepositorySlugIfUnlocked_WhenUnlocked(t *testing.T) {
	c := NewCompiler()

	c.SetRepositorySlugIfUnlocked("owner/repo")
	if got := c.GetRepositorySlug(); got != "owner/repo" {
		t.Fatalf("expected owner/repo, got %q", got)
	}

	// A second call also updates when still unlocked.
	c.SetRepositorySlugIfUnlocked("other/repo")
	if got := c.GetRepositorySlug(); got != "other/repo" {
		t.Fatalf("expected other/repo, got %q", got)
	}
}

// TestRepositorySlugLock_NewCompilerDefaultsUnlocked verifies that a freshly
// created Compiler starts with an unlocked slug.
func TestRepositorySlugLock_NewCompilerDefaultsUnlocked(t *testing.T) {
	c := NewCompiler()
	if c.IsRepositorySlugLocked() {
		t.Fatal("new compiler should start with slug unlocked")
	}
	if got := c.GetRepositorySlug(); got != "" {
		t.Fatalf("new compiler should have empty slug, got %q", got)
	}
}
