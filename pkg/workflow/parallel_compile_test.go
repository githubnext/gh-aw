//go:build !integration

package workflow

import (
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

// ---------------------------------------------------------------------------
// ActionCache.Clone / MergeFrom
// ---------------------------------------------------------------------------

func TestActionCacheCloneIsIndependent(t *testing.T) {
	tmpDir := testutil.TempDir(t, "cache-clone-*")
	original := NewActionCache(tmpDir)
	original.Set("actions/checkout", "v4", "sha-original")

	clone := original.Clone()

	// Mutating the clone must not affect the original
	clone.Set("actions/setup-go", "v5", "sha-new")

	_, found := original.Get("actions/setup-go", "v5")
	if found {
		t.Error("mutation of clone should not affect the original cache")
	}

	// The original entry should still be present in the clone
	sha, found := clone.Get("actions/checkout", "v4")
	if !found {
		t.Fatal("cloned cache should contain entries from the original")
	}
	if sha != "sha-original" {
		t.Errorf("expected sha-original, got %q", sha)
	}
}

func TestActionCacheMergeFromOnlyAddsNew(t *testing.T) {
	tmpDir := testutil.TempDir(t, "cache-merge-*")

	parent := NewActionCache(tmpDir)
	parent.Set("actions/checkout", "v4", "parent-sha")

	child := parent.Clone()
	child.Set("actions/setup-go", "v5", "child-sha")
	// Mutate an existing key in the child; parent must NOT pick this up
	child.Entries["actions/checkout@v4"] = ActionCacheEntry{SHA: "child-overwritten"}

	parent.MergeFrom(child)

	// New key should be present
	sha, found := parent.Get("actions/setup-go", "v5")
	if !found {
		t.Fatal("MergeFrom should add new entries from child")
	}
	if sha != "child-sha" {
		t.Errorf("expected child-sha, got %q", sha)
	}

	// Existing key in parent must NOT be overwritten
	sha, _ = parent.Get("actions/checkout", "v4")
	if sha != "parent-sha" {
		t.Errorf("MergeFrom must not overwrite existing parent entry; got %q", sha)
	}
}

func TestActionCacheMergeFromNilIsNoop(t *testing.T) {
	tmpDir := testutil.TempDir(t, "cache-merge-nil-*")
	parent := NewActionCache(tmpDir)
	parent.Set("actions/checkout", "v4", "sha")

	// Should not panic
	parent.MergeFrom(nil)

	sha, found := parent.Get("actions/checkout", "v4")
	if !found || sha != "sha" {
		t.Error("MergeFrom(nil) should be a no-op")
	}
}

// ---------------------------------------------------------------------------
// Compiler.Fork / Merge
// ---------------------------------------------------------------------------

func newParallelTestCompiler(t *testing.T) *Compiler {
	t.Helper()
	return NewCompiler()
}

func TestCompilerForkInheritsParentConfig(t *testing.T) {
	parent := newParallelTestCompiler(t)
	parent.WarmUp()

	child := parent.Fork()
	if child == nil {
		t.Fatal("Fork() returned nil")
	}

	// Config flags copied by value – spot-check a handful
	if child.verbose != parent.verbose {
		t.Error("Fork should copy verbose flag")
	}
	if child.strictMode != parent.strictMode {
		t.Error("Fork should copy strictMode flag")
	}
}

func TestCompilerForkHasIndependentActionCache(t *testing.T) {
	parent := newParallelTestCompiler(t)
	parent.WarmUp()

	parentCache := parent.GetSharedActionCache()
	parentCache.Set("actions/checkout", "v4", "parent-sha")

	child := parent.Fork()
	childCache := child.GetSharedActionCache()

	// Child should see the pre-fork entry
	sha, found := childCache.Get("actions/checkout", "v4")
	if !found || sha != "parent-sha" {
		t.Error("child should inherit pre-fork cache entries")
	}

	// Child mutation must not affect parent
	childCache.Set("actions/setup-go", "v5", "child-sha")
	_, found = parentCache.Get("actions/setup-go", "v5")
	if found {
		t.Error("child mutation of action cache should not affect parent before Merge")
	}
}

func TestCompilerMergeAccumulatesWarningCount(t *testing.T) {
	parent := newParallelTestCompiler(t)
	parent.WarmUp()

	child1 := parent.Fork()
	child1.warningCount = 3

	child2 := parent.Fork()
	child2.warningCount = 5

	parent.Merge(child1)
	parent.Merge(child2)

	if parent.GetWarningCount() != 8 {
		t.Errorf("expected warning count 8, got %d", parent.GetWarningCount())
	}
}

func TestCompilerMergePropagateCacheEntries(t *testing.T) {
	parent := newParallelTestCompiler(t)
	parent.WarmUp()

	child := parent.Fork()
	child.GetSharedActionCache().Set("actions/new-action", "v1", "new-sha")

	parent.Merge(child)

	sha, found := parent.GetSharedActionCache().Get("actions/new-action", "v1")
	if !found {
		t.Fatal("Merge should propagate new cache entries from child to parent")
	}
	if sha != "new-sha" {
		t.Errorf("expected new-sha, got %q", sha)
	}
}

func TestCompilerMergeNilIsNoop(t *testing.T) {
	parent := newParallelTestCompiler(t)
	parent.WarmUp()

	// Should not panic
	parent.Merge(nil)
	if parent.GetWarningCount() != 0 {
		t.Error("Merge(nil) should be a no-op")
	}
}
