package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCompilationCache_LoadAndSave(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	cacheFile := filepath.Join(tmpDir, ".gh-aw-cache")

	// Create a cache
	cache := &CompilationCache{
		Hashes: map[string]string{
			"workflow1.md": "hash1",
			"workflow2.md": "hash2",
		},
		path: cacheFile,
	}

	// Save the cache
	if err := cache.Save(); err != nil {
		t.Fatalf("Failed to save cache: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		t.Fatal("Cache file was not created")
	}

	// Load the cache
	loaded := &CompilationCache{
		Hashes: make(map[string]string),
		path:   cacheFile,
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}

	if err := json.Unmarshal(data, loaded); err != nil {
		t.Fatalf("Failed to unmarshal cache: %v", err)
	}

	// Verify loaded cache matches
	if len(loaded.Hashes) != 2 {
		t.Errorf("Expected 2 hashes, got %d", len(loaded.Hashes))
	}

	if loaded.Hashes["workflow1.md"] != "hash1" {
		t.Errorf("Hash mismatch for workflow1.md")
	}

	if loaded.Hashes["workflow2.md"] != "hash2" {
		t.Errorf("Hash mismatch for workflow2.md")
	}
}

func TestCompilationCache_NeedsRecompile(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	// Write initial content
	initialContent := "# Initial content\n"
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create cache
	cache := &CompilationCache{
		Hashes: make(map[string]string),
		path:   filepath.Join(tmpDir, ".gh-aw-cache"),
	}

	// First check - should need recompile (not in cache)
	needsCompile, err := cache.NeedsRecompile(testFile)
	if err != nil {
		t.Fatalf("NeedsRecompile failed: %v", err)
	}
	if !needsCompile {
		t.Error("Expected to need recompilation for file not in cache")
	}

	// Update hash
	if err := cache.UpdateHash(testFile); err != nil {
		t.Fatalf("UpdateHash failed: %v", err)
	}

	// Second check - should NOT need recompile (unchanged)
	needsCompile, err = cache.NeedsRecompile(testFile)
	if err != nil {
		t.Fatalf("NeedsRecompile failed: %v", err)
	}
	if needsCompile {
		t.Error("Expected NOT to need recompilation for unchanged file")
	}

	// Modify file
	modifiedContent := "# Modified content\n"
	if err := os.WriteFile(testFile, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Third check - should need recompile (changed)
	needsCompile, err = cache.NeedsRecompile(testFile)
	if err != nil {
		t.Fatalf("NeedsRecompile failed: %v", err)
	}
	if !needsCompile {
		t.Error("Expected to need recompilation for modified file")
	}
}

func TestCompilationCache_ComputeHash(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	content := "# Test content\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Compute hash twice - should be identical
	hash1, err := ComputeHash(testFile)
	if err != nil {
		t.Fatalf("ComputeHash failed: %v", err)
	}

	hash2, err := ComputeHash(testFile)
	if err != nil {
		t.Fatalf("ComputeHash failed: %v", err)
	}

	if hash1 != hash2 {
		t.Error("Hash should be deterministic")
	}

	// Modify file - hash should change
	if err := os.WriteFile(testFile, []byte("# Different content\n"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	hash3, err := ComputeHash(testFile)
	if err != nil {
		t.Fatalf("ComputeHash failed: %v", err)
	}

	if hash1 == hash3 {
		t.Error("Hash should change when content changes")
	}
}

func TestCompilationCache_RemoveHash(t *testing.T) {
	cache := &CompilationCache{
		Hashes: map[string]string{
			"workflow1.md": "hash1",
			"workflow2.md": "hash2",
		},
		path: ".gh-aw-cache",
	}

	cache.RemoveHash("workflow1.md")

	if _, exists := cache.Hashes["workflow1.md"]; exists {
		t.Error("Hash should be removed")
	}

	if _, exists := cache.Hashes["workflow2.md"]; !exists {
		t.Error("Other hashes should remain")
	}
}
