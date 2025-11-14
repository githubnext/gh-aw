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
// It checks the filesystem directly for the expected cache file
func (c *ImportCache) Get(owner, repo, path, ref string) (string, bool) {
	// Use content-based approach: try to find cached file for this import
	// Cache path: .github/aw/imports/owner/repo/ref/sanitized_path.md
	sanitizedPath := strings.ReplaceAll(path, "/", "_")
	relativeCachePath := filepath.Join(ImportCacheDir, owner, repo, ref, sanitizedPath)
	fullCachePath := filepath.Join(c.baseDir, relativeCachePath)

	// Check if the cached file exists
	if _, err := os.Stat(fullCachePath); os.IsNotExist(err) {
		importCacheLog.Printf("Cache miss: %s/%s/%s@%s", owner, repo, path, ref)
		return "", false
	}

	importCacheLog.Printf("Cache hit: %s/%s/%s@%s -> %s", owner, repo, path, ref, fullCachePath)
	return fullCachePath, true
}

// Set stores a new cache entry by saving the content to the cache directory
func (c *ImportCache) Set(owner, repo, path, ref, sha string, content []byte) (string, error) {
	// Use ref (not sha) in path to match Get() lookup
	// This allows the same ref to be cached consistently
	sanitizedPath := strings.ReplaceAll(path, "/", "_")
	relativeCachePath := filepath.Join(ImportCacheDir, owner, repo, ref, sanitizedPath)
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

	importCacheLog.Printf("Cached import: %s/%s/%s@%s -> %s", owner, repo, path, ref, fullCachePath)
	return fullCachePath, nil
}

// GetCacheDir returns the base cache directory path
func (c *ImportCache) GetCacheDir() string {
	return filepath.Join(c.baseDir, ImportCacheDir)
}
