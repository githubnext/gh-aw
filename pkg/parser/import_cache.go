package parser

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var importCacheLog = logger.New("parser:import_cache")

const (
	// ImportCacheDir is the directory where cached imports are stored
	ImportCacheDir = ".github/aw/imports"
)

// ImportCacheEntry represents a cached import file
type ImportCacheEntry struct {
	Owner     string `json:"owner"`      // GitHub owner (user or org)
	Repo      string `json:"repo"`       // Repository name
	Path      string `json:"path"`       // File path within repository
	Ref       string `json:"ref"`        // Git ref (branch, tag, or SHA)
	SHA       string `json:"sha"`        // Commit SHA for the cached content
	CachePath string `json:"cache_path"` // Path to cached file
}

// ImportCache manages cached imported workflow files
type ImportCache struct {
	Entries map[string]ImportCacheEntry `json:"entries"` // key: "owner/repo/path@ref"
	baseDir string                      // Base directory for cache (typically repo root)
}

// NewImportCache creates a new import cache instance
func NewImportCache(repoRoot string) *ImportCache {
	importCacheLog.Printf("Creating import cache with base dir: %s", repoRoot)
	return &ImportCache{
		Entries: make(map[string]ImportCacheEntry),
		baseDir: repoRoot,
	}
}

// Load loads the cache from the manifest file
func (c *ImportCache) Load() error {
	manifestPath := filepath.Join(c.baseDir, ImportCacheDir, "manifest.json")
	importCacheLog.Printf("Loading import cache from: %s", manifestPath)

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Cache manifest doesn't exist yet, that's OK
			importCacheLog.Print("Cache manifest does not exist, starting with empty cache")
			return nil
		}
		importCacheLog.Printf("Failed to read cache manifest: %v", err)
		return err
	}

	if err := json.Unmarshal(data, c); err != nil {
		importCacheLog.Printf("Failed to unmarshal cache data: %v", err)
		return err
	}

	importCacheLog.Printf("Successfully loaded cache with %d entries", len(c.Entries))
	return nil
}

// Save saves the cache manifest to disk with sorted entries
func (c *ImportCache) Save() error {
	manifestPath := filepath.Join(c.baseDir, ImportCacheDir, "manifest.json")
	importCacheLog.Printf("Saving import cache to: %s with %d entries", manifestPath, len(c.Entries))

	// Ensure directory exists
	dir := filepath.Dir(manifestPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		importCacheLog.Printf("Failed to create cache directory: %v", err)
		return err
	}

	// Marshal with sorted entries
	data, err := c.marshalSorted()
	if err != nil {
		importCacheLog.Printf("Failed to marshal cache data: %v", err)
		return err
	}

	// Add trailing newline for prettier compliance
	data = append(data, '\n')

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		importCacheLog.Printf("Failed to write cache manifest: %v", err)
		return err
	}

	importCacheLog.Print("Successfully saved import cache")
	return nil
}

// marshalSorted marshals the cache with entries sorted by key
func (c *ImportCache) marshalSorted() ([]byte, error) {
	// Extract and sort the keys
	keys := make([]string, 0, len(c.Entries))
	for key := range c.Entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Manually construct JSON with sorted keys
	var result []byte
	result = append(result, []byte("{\n  \"entries\": {\n")...)

	for i, key := range keys {
		entry := c.Entries[key]

		// Marshal the entry
		entryJSON, err := json.MarshalIndent(entry, "    ", "  ")
		if err != nil {
			return nil, err
		}

		// Add the key and entry
		result = append(result, []byte("    \""+key+"\": ")...)
		result = append(result, entryJSON...)

		// Add comma if not the last entry
		if i < len(keys)-1 {
			result = append(result, ',')
		}
		result = append(result, '\n')
	}

	result = append(result, []byte("  }\n}")...)
	return result, nil
}

// Get retrieves a cached file path if it exists
func (c *ImportCache) Get(owner, repo, path, ref string) (string, bool) {
	key := fmt.Sprintf("%s/%s/%s@%s", owner, repo, path, ref)
	entry, exists := c.Entries[key]
	if !exists {
		return "", false
	}

	// Verify the cached file still exists
	fullPath := filepath.Join(c.baseDir, entry.CachePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		importCacheLog.Printf("Cached file not found: %s", fullPath)
		return "", false
	}

	importCacheLog.Printf("Cache hit: %s -> %s", key, fullPath)
	return fullPath, true
}

// Set stores a new cache entry and saves the content to the cache directory
func (c *ImportCache) Set(owner, repo, path, ref, sha string, content []byte) (string, error) {
	key := fmt.Sprintf("%s/%s/%s@%s", owner, repo, path, ref)

	// Create cache path: .github/aw/imports/owner/repo/sha/path
	// Use content hash for SHA if not provided
	if sha == "" {
		hash := sha256.Sum256(content)
		sha = hex.EncodeToString(hash[:])[:12] // Use first 12 chars
	}

	// Sanitize path for filesystem
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

	// Store entry in manifest
	c.Entries[key] = ImportCacheEntry{
		Owner:     owner,
		Repo:      repo,
		Path:      path,
		Ref:       ref,
		SHA:       sha,
		CachePath: relativeCachePath,
	}

	importCacheLog.Printf("Cached import: %s -> %s", key, fullCachePath)
	return fullCachePath, nil
}

// GetCacheDir returns the base cache directory path
func (c *ImportCache) GetCacheDir() string {
	return filepath.Join(c.baseDir, ImportCacheDir)
}
