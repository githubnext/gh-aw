package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestActionCache(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	cache := NewActionCache(tmpDir)

	// Test setting and getting
	cache.Set("actions/checkout", "v5", "abc123")

	sha, found := cache.Get("actions/checkout", "v5")
	if !found {
		t.Error("Expected to find cached entry")
	}
	if sha != "abc123" {
		t.Errorf("Expected SHA 'abc123', got '%s'", sha)
	}

	// Test cache miss
	_, found = cache.Get("actions/unknown", "v1")
	if found {
		t.Error("Expected cache miss for unknown action")
	}
}

func TestActionCacheSaveLoad(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Create and populate cache
	cache1 := NewActionCache(tmpDir)
	cache1.Set("actions/checkout", "v5", "abc123")
	cache1.Set("actions/setup-node", "v4", "def456")

	// Save to disk
	err := cache1.Save()
	if err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Verify file exists
	cachePath := filepath.Join(tmpDir, ".github", "aw", CacheFileName)
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatalf("Cache file was not created at %s", cachePath)
	}

	// Load into new cache instance
	cache2 := NewActionCache(tmpDir)
	err = cache2.Load()
	if err != nil {
		t.Fatalf("Failed to load cache: %v", err)
	}

	// Verify entries were loaded
	sha, found := cache2.Get("actions/checkout", "v5")
	if !found || sha != "abc123" {
		t.Errorf("Expected to find actions/checkout@v5 with SHA 'abc123', got '%s' (found=%v)", sha, found)
	}

	sha, found = cache2.Get("actions/setup-node", "v4")
	if !found || sha != "def456" {
		t.Errorf("Expected to find actions/setup-node@v6 with SHA 'def456', got '%s' (found=%v)", sha, found)
	}
}

func TestActionCacheLoadNonExistent(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	cache := NewActionCache(tmpDir)

	// Try to load non-existent cache - should not error
	err := cache.Load()
	if err != nil {
		t.Errorf("Loading non-existent cache should not error, got: %v", err)
	}

	// Cache should be empty
	if len(cache.Entries) != 0 {
		t.Errorf("Expected empty cache, got %d entries", len(cache.Entries))
	}
}

func TestActionCacheGetCachePath(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewActionCache(tmpDir)

	expectedPath := filepath.Join(tmpDir, ".github", "aw", CacheFileName)
	if cache.GetCachePath() != expectedPath {
		t.Errorf("Expected cache path '%s', got '%s'", expectedPath, cache.GetCachePath())
	}
}

func TestActionCacheTrailingNewline(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Create and populate cache
	cache := NewActionCache(tmpDir)
	cache.Set("actions/checkout", "v5", "abc123")

	// Save to disk
	err := cache.Save()
	if err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Read the file and check for trailing newline
	cachePath := filepath.Join(tmpDir, ".github", "aw", CacheFileName)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}

	// Verify file ends with newline (prettier compliance)
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Error("Cache file should end with a trailing newline for prettier compliance")
	}
}
