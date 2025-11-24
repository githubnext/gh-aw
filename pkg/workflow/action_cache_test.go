package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestActionCache(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := testutil.TempDir(t, "test-*")

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
	tmpDir := testutil.TempDir(t, "test-*")

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
	tmpDir := testutil.TempDir(t, "test-*")

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
	tmpDir := testutil.TempDir(t, "test-*")
	cache := NewActionCache(tmpDir)

	expectedPath := filepath.Join(tmpDir, ".github", "aw", CacheFileName)
	if cache.GetCachePath() != expectedPath {
		t.Errorf("Expected cache path '%s', got '%s'", expectedPath, cache.GetCachePath())
	}
}

func TestActionCacheTrailingNewline(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := testutil.TempDir(t, "test-*")

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

func TestActionCacheSortedEntries(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := testutil.TempDir(t, "test-*")

	// Create cache and add entries in non-alphabetical order
	cache := NewActionCache(tmpDir)
	cache.Set("zzz/last-action", "v1", "sha111")
	cache.Set("actions/checkout", "v5", "sha222")
	cache.Set("mmm/middle-action", "v2", "sha333")
	cache.Set("actions/setup-node", "v4", "sha444")
	cache.Set("aaa/first-action", "v3", "sha555")

	// Save to disk
	err := cache.Save()
	if err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Read the file content
	cachePath := filepath.Join(tmpDir, ".github", "aw", CacheFileName)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}

	content := string(data)

	// Verify that entries appear in alphabetical order by checking their positions
	entries := []string{
		"aaa/first-action@v3",
		"actions/checkout@v5",
		"actions/setup-node@v4",
		"mmm/middle-action@v2",
		"zzz/last-action@v1",
	}

	lastPos := -1
	for _, entry := range entries {
		pos := indexOf(content, entry)
		if pos == -1 {
			t.Errorf("Entry %s not found in cache file", entry)
			continue
		}
		if pos < lastPos {
			t.Errorf("Entry %s appears before previous entry (not sorted)", entry)
		}
		lastPos = pos
	}

	// Also verify the file is valid JSON
	var loadedCache ActionCache
	err = json.Unmarshal(data, &loadedCache)
	if err != nil {
		t.Fatalf("Saved cache is not valid JSON: %v", err)
	}

	// Verify all entries are present
	if len(loadedCache.Entries) != 5 {
		t.Errorf("Expected 5 entries, got %d", len(loadedCache.Entries))
	}
}

// indexOf returns the index of substr in s, or -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestActionCacheEmptySaveDoesNotCreateFile(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := testutil.TempDir(t, "test-*")

	// Create empty cache
	cache := NewActionCache(tmpDir)

	// Save empty cache
	err := cache.Save()
	if err != nil {
		t.Fatalf("Failed to save empty cache: %v", err)
	}

	// Verify file does NOT exist
	cachePath := filepath.Join(tmpDir, ".github", "aw", CacheFileName)
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Error("Empty cache should not create a file")
	}
}

func TestActionCacheEmptySaveDeletesExistingFile(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := testutil.TempDir(t, "test-*")

	// Create cache with entries and save
	cache := NewActionCache(tmpDir)
	cache.Set("actions/checkout", "v5", "abc123")
	err := cache.Save()
	if err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Verify file exists
	cachePath := filepath.Join(tmpDir, ".github", "aw", CacheFileName)
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("Cache file should exist after saving with entries")
	}

	// Clear cache and save again
	cache.Entries = make(map[string]ActionCacheEntry)
	err = cache.Save()
	if err != nil {
		t.Fatalf("Failed to save empty cache: %v", err)
	}

	// Verify file is now deleted
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Error("Empty cache should delete existing file")
	}
}
