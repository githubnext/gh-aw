package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var actionCacheLog = logger.New("workflow:action_cache")

const (
	// CacheFileName is the name of the cache file in .github/aw/
	CacheFileName = "actions-lock.json"
)

// ActionCacheEntry represents a cached action pin resolution
type ActionCacheEntry struct {
	Repo    string `json:"repo"`
	Version string `json:"version"`
	SHA     string `json:"sha"`
}

// ActionCache manages cached action pin resolutions
type ActionCache struct {
	Entries map[string]ActionCacheEntry `json:"entries"` // key: "repo@version"
	path    string
}

// NewActionCache creates a new action cache instance
func NewActionCache(repoRoot string) *ActionCache {
	cachePath := filepath.Join(repoRoot, ".github", "aw", CacheFileName)
	actionCacheLog.Printf("Creating action cache with path: %s", cachePath)
	return &ActionCache{
		Entries: make(map[string]ActionCacheEntry),
		path:    cachePath,
	}
}

// Load loads the cache from disk
func (c *ActionCache) Load() error {
	actionCacheLog.Printf("Loading action cache from: %s", c.path)
	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Cache file doesn't exist yet, that's OK
			actionCacheLog.Print("Cache file does not exist, starting with empty cache")
			return nil
		}
		actionCacheLog.Printf("Failed to read cache file: %v", err)
		return err
	}

	if err := json.Unmarshal(data, c); err != nil {
		actionCacheLog.Printf("Failed to unmarshal cache data: %v", err)
		return err
	}

	actionCacheLog.Printf("Successfully loaded cache with %d entries", len(c.Entries))
	return nil
}

// Save saves the cache to disk
func (c *ActionCache) Save() error {
	actionCacheLog.Printf("Saving action cache to: %s with %d entries", c.path, len(c.Entries))

	// Ensure directory exists
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		actionCacheLog.Printf("Failed to create cache directory: %v", err)
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		actionCacheLog.Printf("Failed to marshal cache data: %v", err)
		return err
	}

	// Add trailing newline for prettier compliance
	data = append(data, '\n')

	if err := os.WriteFile(c.path, data, 0644); err != nil {
		actionCacheLog.Printf("Failed to write cache file: %v", err)
		return err
	}

	actionCacheLog.Print("Successfully saved action cache")
	return nil
}

// Get retrieves a cached entry if it exists
func (c *ActionCache) Get(repo, version string) (string, bool) {
	key := repo + "@" + version
	entry, exists := c.Entries[key]
	if !exists {
		return "", false
	}

	return entry.SHA, true
}

// Set stores a new cache entry
func (c *ActionCache) Set(repo, version, sha string) {
	key := repo + "@" + version
	c.Entries[key] = ActionCacheEntry{
		Repo:    repo,
		Version: version,
		SHA:     sha,
	}
}

// GetCachePath returns the path to the cache file
func (c *ActionCache) GetCachePath() string {
	return c.path
}
