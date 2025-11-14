package parser

import (
	"os"
	"path/filepath"
	"testing"
)

// TestImportCacheIntegration tests the cache with the full import flow
func TestImportCacheIntegration(t *testing.T) {
	// Create temp directories for testing
	tempDir, err := os.MkdirTemp("", "import-cache-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a cache
	cache := NewImportCache(tempDir)

	// Simulate a workflow file that imports from another repo
	workflowContent := `---
imports:
  - testowner/testrepo/workflows/shared.md@main
---

# Test Workflow

Use shared configuration.
`

	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Simulate a remote file being cached
	sharedContent := []byte(`---
tools:
  edit:
---

# Shared Configuration

This is shared configuration.
`)

	// Cache the "remote" file
	cachedPath, err := cache.Set("testowner", "testrepo", "workflows/shared.md", "main", "abc123", sharedContent)
	if err != nil {
		t.Fatalf("Failed to cache file: %v", err)
	}

	// Save the cache manifest
	if err := cache.Save(); err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Verify the manifest file was created
	manifestPath := filepath.Join(tempDir, ImportCacheDir, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Errorf("Manifest file was not created: %s", manifestPath)
	}

	// Verify cache can retrieve the file
	retrievedPath, found := cache.Get("testowner", "testrepo", "workflows/shared.md", "main")
	if !found {
		t.Error("Failed to retrieve cached file")
	}
	if retrievedPath != cachedPath {
		t.Errorf("Retrieved path mismatch. Expected %s, got %s", cachedPath, retrievedPath)
	}

	// Verify the cached file contains correct content
	content, err := os.ReadFile(retrievedPath)
	if err != nil {
		t.Fatalf("Failed to read cached file: %v", err)
	}
	if string(content) != string(sharedContent) {
		t.Errorf("Content mismatch. Expected %q, got %q", sharedContent, content)
	}

	// Test loading from a new cache instance (simulating offline scenario)
	cache2 := NewImportCache(tempDir)
	if err := cache2.Load(); err != nil {
		t.Fatalf("Failed to load cache in new instance: %v", err)
	}

	// Verify we can still retrieve the file
	retrievedPath2, found := cache2.Get("testowner", "testrepo", "workflows/shared.md", "main")
	if !found {
		t.Error("Failed to retrieve cached file from loaded cache")
	}
	if retrievedPath2 != cachedPath {
		t.Errorf("Retrieved path mismatch after load. Expected %s, got %s", cachedPath, retrievedPath2)
	}
}

// TestImportCacheMultipleFiles tests caching multiple files from different repos
func TestImportCacheMultipleFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "import-cache-multi-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache := NewImportCache(tempDir)

	// Cache multiple files
	files := []struct {
		owner   string
		repo    string
		path    string
		ref     string
		sha     string
		content string
	}{
		{"owner1", "repo1", "workflows/a.md", "main", "sha1", "Content A"},
		{"owner1", "repo1", "workflows/b.md", "v1.0", "sha2", "Content B"},
		{"owner2", "repo2", "config/c.md", "main", "sha3", "Content C"},
	}

	for _, f := range files {
		_, err := cache.Set(f.owner, f.repo, f.path, f.ref, f.sha, []byte(f.content))
		if err != nil {
			t.Fatalf("Failed to cache file %s/%s/%s@%s: %v", f.owner, f.repo, f.path, f.ref, err)
		}
	}

	// Save cache
	if err := cache.Save(); err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Verify all files are retrievable
	for _, f := range files {
		path, found := cache.Get(f.owner, f.repo, f.path, f.ref)
		if !found {
			t.Errorf("Failed to retrieve cached file %s/%s/%s@%s", f.owner, f.repo, f.path, f.ref)
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read cached file: %v", err)
			continue
		}

		if string(content) != f.content {
			t.Errorf("Content mismatch for %s/%s/%s@%s. Expected %q, got %q",
				f.owner, f.repo, f.path, f.ref, f.content, string(content))
		}
	}

	// Load from new cache instance and verify
	cache2 := NewImportCache(tempDir)
	if err := cache2.Load(); err != nil {
		t.Fatalf("Failed to load cache: %v", err)
	}

	if len(cache2.Entries) != len(files) {
		t.Errorf("Expected %d entries in loaded cache, got %d", len(files), len(cache2.Entries))
	}
}
