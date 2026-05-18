package cli

import (
	"sync"
	"time"

	"github.com/github/gh-aw/pkg/logger"
)

var mcpServerCacheLog = logger.New("cli:mcp_server_cache")

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

func (c *mcpCacheStore) getPermissionEntry(cacheKey string) (*permissionEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.permissions[cacheKey]
	return entry, ok
}

func (c *mcpCacheStore) deletePermissionEntryIfUnchanged(cacheKey string, entry *permissionEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if currentEntry, ok := c.permissions[cacheKey]; ok && currentEntry == entry {
		delete(c.permissions, cacheKey)
	}
}

// GetPermission returns the cached permission for the given actor and repo, or ("", false) on cache miss.
func (c *mcpCacheStore) GetPermission(actor, repo string) (string, bool) {
	cacheKey := actor + ":" + repo
	entry, ok := c.getPermissionEntry(cacheKey)
	if ok && time.Since(entry.timestamp) < c.permissionTTL {
		mcpServerCacheLog.Printf("Permission cache hit: actor=%s, repo=%s, permission=%s", actor, repo, entry.permission)
		return entry.permission, true
	}
	if ok {
		// Expired — remove it
		mcpServerCacheLog.Printf("Permission cache entry expired for actor=%s, repo=%s", actor, repo)
		c.deletePermissionEntryIfUnchanged(cacheKey, entry)
	}
	mcpServerCacheLog.Printf("Permission cache miss: actor=%s, repo=%s", actor, repo)
	return "", false
}

// SetPermission stores a permission in the cache.
func (c *mcpCacheStore) SetPermission(actor, repo, permission string) {
	cacheKey := actor + ":" + repo
	mcpServerCacheLog.Printf("Caching permission: actor=%s, repo=%s, permission=%s", actor, repo, permission)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.permissions[cacheKey] = &permissionEntry{
		permission: permission,
		timestamp:  time.Now(),
	}
}

// GetRepo returns the cached repository name, or ("", false) on cache miss.
func (c *mcpCacheStore) GetRepo() (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.repo != nil && time.Since(c.repo.timestamp) < c.repoTTL {
		mcpServerCacheLog.Printf("Repo cache hit: repository=%s", c.repo.repository)
		return c.repo.repository, true
	}
	mcpServerCacheLog.Print("Repo cache miss")
	return "", false
}

// SetRepo stores a repository name in the cache.
func (c *mcpCacheStore) SetRepo(repository string) {
	mcpServerCacheLog.Printf("Caching repository: %s", repository)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.repo = &repoEntry{
		repository: repository,
		timestamp:  time.Now(),
	}
}

var mcpCache = newMCPCacheStore()
