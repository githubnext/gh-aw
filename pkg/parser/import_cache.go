package parser

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var importCacheLog = logger.New("parser:import_cache")

const (
	// ImportCacheDir is the directory where cached imports are stored
	ImportCacheDir = ".github/aw/imports"
)

// ImportCache manages cached imported workflow files
type ImportCache struct {
	baseDir string // Base directory for cache (typically repo root)
}

// NewImportCache creates a new import cache instance
func NewImportCache(repoRoot string) *ImportCache {
	importCacheLog.Printf("Creating import cache with base dir: %s", repoRoot)
	return &ImportCache{
		baseDir: repoRoot,
	}
}

// Get retrieves a cached file path if it exists
// sha parameter should be the resolved commit SHA
func (c *ImportCache) Get(owner, repo, path, sha string) (string, bool) {
	// Use SHA-based approach: cache files are stored by commit SHA
	// Cache path: .github/aw/imports/owner/repo/sha/sanitized_path.md
	sanitizedPath := strings.ReplaceAll(path, "/", "_")
	relativeCachePath := filepath.Join(ImportCacheDir, owner, repo, sha, sanitizedPath)
	fullCachePath := filepath.Join(c.baseDir, relativeCachePath)

	// Check if the cached file exists
	if _, err := os.Stat(fullCachePath); os.IsNotExist(err) {
		importCacheLog.Printf("Cache miss: %s/%s/%s@%s", owner, repo, path, sha)
		return "", false
	}

	importCacheLog.Printf("Cache hit: %s/%s/%s@%s -> %s", owner, repo, path, sha, fullCachePath)
	return fullCachePath, true
}

// Set stores a new cache entry by saving the content to the cache directory
// sha parameter should be the resolved commit SHA
func (c *ImportCache) Set(owner, repo, path, sha, _ string, content []byte) (string, error) {
	// Use SHA in path for consistent caching
	// This ensures that different refs pointing to the same commit reuse the same cache
	sanitizedPath := strings.ReplaceAll(path, "/", "_")
	relativeCachePath := filepath.Join(ImportCacheDir, owner, repo, sha, sanitizedPath)
	fullCachePath := filepath.Join(c.baseDir, relativeCachePath)

	// Ensure directory exists
	dir := filepath.Dir(fullCachePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		importCacheLog.Printf("Failed to create cache directory: %v", err)
		return "", err
	}

	// Write content to cache file
	if err := os.WriteFile(fullCachePath, content, 0644); err != nil {
		importCacheLog.Printf("Failed to write cache file: %v", err)
		return "", err
	}

	importCacheLog.Printf("Cached import: %s/%s/%s@%s -> %s", owner, repo, path, sha, fullCachePath)
	return fullCachePath, nil
}

// GetCacheDir returns the base cache directory path
func (c *ImportCache) GetCacheDir() string {
	return filepath.Join(c.baseDir, ImportCacheDir)
}
