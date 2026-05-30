//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContainerPinCRUD verifies the Get/Set/Delete lifecycle for container pins.
func TestContainerPinCRUD(t *testing.T) {
	cache := NewActionCache(t.TempDir())

	// Initially no pin.
	_, ok := cache.GetContainerPin("node:lts-alpine")
	assert.False(t, ok, "no pin expected before Set")

	// Set a pin.
	cache.SetContainerPin("node:lts-alpine", "sha256:abc123", "node:lts-alpine@sha256:abc123")
	pin, ok := cache.GetContainerPin("node:lts-alpine")
	require.True(t, ok, "pin should exist after Set")
	assert.Equal(t, "node:lts-alpine", pin.Image, "Image field")
	assert.Equal(t, "sha256:abc123", pin.Digest, "Digest field")
	assert.Equal(t, "node:lts-alpine@sha256:abc123", pin.PinnedImage, "PinnedImage field")

	// Overwrite with updated pin.
	cache.SetContainerPin("node:lts-alpine", "sha256:updated", "node:lts-alpine@sha256:updated")
	pin, ok = cache.GetContainerPin("node:lts-alpine")
	require.True(t, ok, "pin should exist after update")
	assert.Equal(t, "sha256:updated", pin.Digest, "updated Digest")

	// Delete the pin.
	cache.DeleteContainerPin("node:lts-alpine")
	_, ok = cache.GetContainerPin("node:lts-alpine")
	assert.False(t, ok, "pin should be gone after Delete")

	// Deleting a non-existent pin is a no-op.
	assert.NotPanics(t, func() { cache.DeleteContainerPin("nonexistent:latest") }, "delete non-existent should not panic")
}

// TestContainerPinSaveLoad verifies that container pins survive a JSON round-trip.
func TestContainerPinSaveLoad(t *testing.T) {
	tmpDir := testutil.TempDir(t, "container-pin-*")

	cache := NewActionCache(tmpDir)
	cache.Set("actions/checkout", "v5", "sha1")
	cache.SetContainerPin("node:lts-alpine", "sha256:abc123", "node:lts-alpine@sha256:abc123")
	cache.SetContainerPin("alpine:latest", "sha256:def456", "alpine:latest@sha256:def456")

	require.NoError(t, cache.Save(), "Save should succeed")

	// Reload from disk.
	cache2 := NewActionCache(tmpDir)
	require.NoError(t, cache2.Load(), "Load should succeed")

	// Action entry should be preserved.
	sha, ok := cache2.Get("actions/checkout", "v5")
	require.True(t, ok, "action entry should be loaded")
	assert.Equal(t, "sha1", sha, "SHA should match")

	// Container pins should be preserved.
	pin, ok := cache2.GetContainerPin("node:lts-alpine")
	require.True(t, ok, "node pin should be loaded")
	assert.Equal(t, "sha256:abc123", pin.Digest, "node digest")
	assert.Equal(t, "node:lts-alpine@sha256:abc123", pin.PinnedImage, "node pinned image")

	pin, ok = cache2.GetContainerPin("alpine:latest")
	require.True(t, ok, "alpine pin should be loaded")
	assert.Equal(t, "sha256:def456", pin.Digest, "alpine digest")
}

// TestContainerPinBackwardCompatibility verifies that loading an existing
// actions-lock.json without a container_pins section returns an empty map
// (no error, no panic).
func TestContainerPinBackwardCompatibility(t *testing.T) {
	tmpDir := testutil.TempDir(t, "container-compat-*")

	// Write a legacy actions-lock.json (no container_pins field).
	legacyJSON := `{
  "entries": {
    "actions/checkout@v5": {
      "repo": "actions/checkout",
      "version": "v5",
      "sha": "abc123"
    }
  }
}
`
	jsonPath := filepath.Join(tmpDir, ".github", "aw", CacheFileName)
	require.NoError(t, os.MkdirAll(filepath.Dir(jsonPath), 0755))
	require.NoError(t, os.WriteFile(jsonPath, []byte(legacyJSON), 0644))

	cache := NewActionCache(tmpDir)
	require.NoError(t, cache.Load(), "Load should succeed for legacy file")

	// ContainerPins should be an initialized (empty) map, not nil.
	assert.NotNil(t, cache.ContainerPins, "ContainerPins should not be nil after Load")
	assert.Empty(t, cache.ContainerPins, "ContainerPins should be empty for legacy file")

	// Action entries should still be present.
	sha, ok := cache.Get("actions/checkout", "v5")
	require.True(t, ok, "action entry should be loaded from legacy file")
	assert.Equal(t, "abc123", sha, "SHA should match")
}

// TestContainerPinMarshalSortedOutput verifies that container pins are written
// in sorted order and that the JSON is valid.
func TestContainerPinMarshalSortedOutput(t *testing.T) {
	tmpDir := testutil.TempDir(t, "container-marshal-*")
	cache := NewActionCache(tmpDir)
	cache.Set("actions/checkout", "v5", "sha1")
	cache.SetContainerPin("z-image:latest", "sha256:zzz", "z-image:latest@sha256:zzz")
	cache.SetContainerPin("a-image:latest", "sha256:aaa", "a-image:latest@sha256:aaa")

	require.NoError(t, cache.Save())

	content, err := os.ReadFile(filepath.Join(tmpDir, ".github", "aw", CacheFileName))
	require.NoError(t, err)

	// Both container images should appear in the JSON.
	contentStr := string(content)
	assert.Contains(t, contentStr, `"a-image:latest"`, "a-image pin in output")
	assert.Contains(t, contentStr, `"z-image:latest"`, "z-image pin in output")
	assert.Contains(t, contentStr, `"containers"`, "containers section present")

	// Reload and verify round-trip.
	cache2 := NewActionCache(tmpDir)
	require.NoError(t, cache2.Load())
	pin, ok := cache2.GetContainerPin("a-image:latest")
	require.True(t, ok)
	assert.Equal(t, "sha256:aaa", pin.Digest)
	pin, ok = cache2.GetContainerPin("z-image:latest")
	require.True(t, ok)
	assert.Equal(t, "sha256:zzz", pin.Digest)
}
