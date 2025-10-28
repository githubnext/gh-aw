package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	// CacheFileName is the name of the cache file in .github/aw/
	CacheFileName = "action-pins-cache.json"

	// CacheExpirationDays is the number of days before cache entries expire
	CacheExpirationDays = 7
)

// ActionCacheEntry represents a cached action pin resolution
type ActionCacheEntry struct {
	Repo      string    `json:"repo"`
	Version   string    `json:"version"`
	SHA       string    `json:"sha"`
	Timestamp time.Time `json:"timestamp"`
}

// ActionCache manages cached action pin resolutions
type ActionCache struct {
	Entries map[string]ActionCacheEntry `json:"entries"` // key: "repo@version"
	path    string
}

// NewActionCache creates a new action cache instance
func NewActionCache(repoRoot string) *ActionCache {
	cachePath := filepath.Join(repoRoot, ".github", "aw", CacheFileName)
	return &ActionCache{
		Entries: make(map[string]ActionCacheEntry),
		path:    cachePath,
	}
}

// Load loads the cache from disk
func (c *ActionCache) Load() error {
	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Cache file doesn't exist yet, that's OK
			return nil
		}
		return err
	}

	return json.Unmarshal(data, c)
}

// Save saves the cache to disk
func (c *ActionCache) Save() error {
	// Ensure directory exists
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0644)
}

// Get retrieves a cached entry if it exists and is not expired
func (c *ActionCache) Get(repo, version string) (string, bool) {
	key := repo + "@" + version
	entry, exists := c.Entries[key]
	if !exists {
		return "", false
	}

	// Check if expired
	if time.Since(entry.Timestamp) > CacheExpirationDays*24*time.Hour {
		// Entry is expired
		delete(c.Entries, key)
		return "", false
	}

	return entry.SHA, true
}

// Set stores a new cache entry
func (c *ActionCache) Set(repo, version, sha string) {
	key := repo + "@" + version
	c.Entries[key] = ActionCacheEntry{
		Repo:      repo,
		Version:   version,
		SHA:       sha,
		Timestamp: time.Now(),
	}
}

// GetCachePath returns the path to the cache file
func (c *ActionCache) GetCachePath() string {
	return c.path
}
