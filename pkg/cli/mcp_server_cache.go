package cli

import (
	"sync"
	"time"
)

// mcpCacheStore provides thread-safe caching for actor permissions and repository lookups.
// All exported methods are safe for concurrent use.
type mcpCacheStore struct {
	mu            sync.RWMutex
	permissions   map[string]*permissionEntry
	permissionTTL time.Duration
	repo          *repoEntry
	repoTTL       time.Duration
}

type permissionEntry struct {
	permission string
	timestamp  time.Time
}

type repoEntry struct {
	repository string
	timestamp  time.Time
}

func newMCPCacheStore() *mcpCacheStore {
	return &mcpCacheStore{
		permissions:   make(map[string]*permissionEntry),
		permissionTTL: 1 * time.Hour,
		repoTTL:       1 * time.Hour,
	}
}

// GetPermission returns the cached permission for the given actor and repo, or ("", false) on cache miss.
func (c *mcpCacheStore) GetPermission(actor, repo string) (string, bool) {
	cacheKey := actor + ":" + repo
	c.mu.RLock()
	entry, ok := c.permissions[cacheKey]
	if ok && time.Since(entry.timestamp) < c.permissionTTL {
		perm := entry.permission
		c.mu.RUnlock()
		return perm, true
	}
	c.mu.RUnlock()
	if ok {
		// Expired — remove it
		c.mu.Lock()
		delete(c.permissions, cacheKey)
		c.mu.Unlock()
	}
	return "", false
}

// SetPermission stores a permission in the cache.
func (c *mcpCacheStore) SetPermission(actor, repo, permission string) {
	cacheKey := actor + ":" + repo
	c.mu.Lock()
	c.permissions[cacheKey] = &permissionEntry{
		permission: permission,
		timestamp:  time.Now(),
	}
	c.mu.Unlock()
}

// GetRepo returns the cached repository name, or ("", false) on cache miss.
func (c *mcpCacheStore) GetRepo() (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.repo != nil && time.Since(c.repo.timestamp) < c.repoTTL {
		return c.repo.repository, true
	}
	return "", false
}

// SetRepo stores a repository name in the cache.
func (c *mcpCacheStore) SetRepo(repository string) {
	c.mu.Lock()
	c.repo = &repoEntry{
		repository: repository,
		timestamp:  time.Now(),
	}
	c.mu.Unlock()
}

var mcpCache = newMCPCacheStore()
