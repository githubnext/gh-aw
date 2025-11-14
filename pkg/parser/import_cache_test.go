package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportCache(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "import-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a new cache
	cache := NewImportCache(tempDir)

	// Test cache is empty initially
	if len(cache.Entries) != 0 {
		t.Errorf("Expected empty cache, got %d entries", len(cache.Entries))
	}

	// Test Set and Get
	testContent := []byte("# Test Workflow\n\nTest content")
	owner := "testowner"
	repo := "testrepo"
	path := "workflows/test.md"
	ref := "main"
	sha := "abc123"

	cachedPath, err := cache.Set(owner, repo, path, ref, sha, testContent)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(cachedPath); os.IsNotExist(err) {
		t.Errorf("Cache file was not created: %s", cachedPath)
	}

	// Verify content
	content, err := os.ReadFile(cachedPath)
	if err != nil {
		t.Fatalf("Failed to read cached file: %v", err)
	}
	if string(content) != string(testContent) {
		t.Errorf("Content mismatch. Expected %q, got %q", testContent, content)
	}

	// Test Get
	retrievedPath, found := cache.Get(owner, repo, path, ref)
	if !found {
		t.Error("Cache entry not found after Set")
	}
	if retrievedPath != cachedPath {
		t.Errorf("Path mismatch. Expected %s, got %s", cachedPath, retrievedPath)
	}

	// Test Save and Load
	if err := cache.Save(); err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Create new cache instance and load
	cache2 := NewImportCache(tempDir)
	if err := cache2.Load(); err != nil {
		t.Fatalf("Failed to load cache: %v", err)
	}

	// Verify loaded cache has the entry
	if len(cache2.Entries) != 1 {
		t.Errorf("Expected 1 entry in loaded cache, got %d", len(cache2.Entries))
	}

	retrievedPath2, found := cache2.Get(owner, repo, path, ref)
	if !found {
		t.Error("Cache entry not found after Load")
	}
	if retrievedPath2 != cachedPath {
		t.Errorf("Path mismatch after load. Expected %s, got %s", cachedPath, retrievedPath2)
	}
}

func TestImportCacheDirectory(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "import-cache-dir-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache := NewImportCache(tempDir)

	// Test cache directory path
	expectedDir := filepath.Join(tempDir, ImportCacheDir)
	if cache.GetCacheDir() != expectedDir {
		t.Errorf("Cache dir mismatch. Expected %s, got %s", expectedDir, cache.GetCacheDir())
	}

	// Create a cache entry to trigger directory creation
	testContent := []byte("test")
	_, err = cache.Set("owner", "repo", "test.md", "main", "sha1", testContent)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("Cache directory was not created: %s", expectedDir)
	}
}

func TestImportCacheMissingFile(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "import-cache-missing-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache := NewImportCache(tempDir)

	// Add entry to cache
	testContent := []byte("test")
	cachedPath, err := cache.Set("owner", "repo", "test.md", "main", "sha1", testContent)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Delete the cached file
	if err := os.Remove(cachedPath); err != nil {
		t.Fatalf("Failed to remove cached file: %v", err)
	}

	// Try to get the entry - should return not found since file is missing
	_, found := cache.Get("owner", "repo", "test.md", "main")
	if found {
		t.Error("Expected cache miss for deleted file, but got hit")
	}
}

func TestImportCacheEmptyLoad(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "import-cache-empty-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache := NewImportCache(tempDir)

	// Load from empty directory should not fail
	if err := cache.Load(); err != nil {
		t.Errorf("Load from empty directory should not fail: %v", err)
	}

	// Cache should still be empty
	if len(cache.Entries) != 0 {
		t.Errorf("Expected empty cache after loading from empty dir, got %d entries", len(cache.Entries))
	}
}
