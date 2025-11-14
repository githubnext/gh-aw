package cli

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var cacheLog = logger.New("cli:compile_cache")

// CompilationCache tracks workflow file hashes to enable incremental compilation
type CompilationCache struct {
	Hashes map[string]string `json:"hashes"` // workflow path -> SHA256 hash
	path   string
}

// LoadCompilationCache loads the cache from disk, or creates a new empty cache
func LoadCompilationCache(workflowDir string) (*CompilationCache, error) {
	gitRoot, err := findGitRoot()
	if err != nil {
		// If not in git repo, use current directory
		gitRoot = "."
	}

	cachePath := filepath.Join(gitRoot, ".gh-aw-cache")
	cacheLog.Printf("Loading compilation cache from %s", cachePath)

	cache := &CompilationCache{
		Hashes: make(map[string]string),
		path:   cachePath,
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			cacheLog.Print("Cache file does not exist, starting with empty cache")
			return cache, nil
		}
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	if err := json.Unmarshal(data, cache); err != nil {
		cacheLog.Printf("Failed to unmarshal cache, starting fresh: %v", err)
		// Return empty cache instead of error (corrupted cache is recoverable)
		return cache, nil
	}

	cacheLog.Printf("Loaded cache with %d entries", len(cache.Hashes))
	return cache, nil
}

// Save writes the cache to disk
func (c *CompilationCache) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(c.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	cacheLog.Printf("Saved cache with %d entries to %s", len(c.Hashes), c.path)
	return nil
}

// ComputeHash calculates SHA256 hash of a file
func ComputeHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// NeedsRecompile checks if a workflow file needs recompilation
func (c *CompilationCache) NeedsRecompile(filePath string) (bool, error) {
	hash, err := ComputeHash(filePath)
	if err != nil {
		return true, err // If we can't compute hash, assume it needs recompilation
	}

	oldHash, exists := c.Hashes[filePath]
	if !exists {
		cacheLog.Printf("File %s not in cache, needs compilation", filepath.Base(filePath))
		return true, nil
	}

	if oldHash != hash {
		cacheLog.Printf("File %s hash changed, needs recompilation", filepath.Base(filePath))
		return true, nil
	}

	cacheLog.Printf("File %s unchanged, skipping compilation", filepath.Base(filePath))
	return false, nil
}

// UpdateHash updates the hash for a workflow file
func (c *CompilationCache) UpdateHash(filePath string) error {
	hash, err := ComputeHash(filePath)
	if err != nil {
		return err
	}

	c.Hashes[filePath] = hash
	cacheLog.Printf("Updated hash for %s", filepath.Base(filePath))
	return nil
}

// RemoveHash removes a workflow from the cache
func (c *CompilationCache) RemoveHash(filePath string) {
	delete(c.Hashes, filePath)
	cacheLog.Printf("Removed hash for %s", filepath.Base(filePath))
}
