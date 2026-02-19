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

// GetPermission returns the cached permission for the given actor and repo, or ("", false) on cache miss.
func (c *mcpCacheStore) GetPermission(actor, repo string) (string, bool) {
	cacheKey := actor + ":" + repo
	c.mu.RLock()
	entry, ok := c.permissions[cacheKey]
	if ok && time.Since(entry.timestamp) < c.permissionTTL {
		perm := entry.permission
		c.mu.RUnlock()
		mcpServerCacheLog.Printf("Permission cache hit: actor=%s, repo=%s, permission=%s", actor, repo, perm)
		return perm, true
	}
	c.mu.RUnlock()
	if ok {
		// Expired — remove it
		mcpServerCacheLog.Printf("Permission cache entry expired for actor=%s, repo=%s", actor, repo)
		c.mu.Lock()
		delete(c.permissions, cacheKey)
		c.mu.Unlock()
	}
	mcpServerCacheLog.Printf("Permission cache miss: actor=%s, repo=%s", actor, repo)
	return "", false
}

// SetPermission stores a permission in the cache.
func (c *mcpCacheStore) SetPermission(actor, repo, permission string) {
	cacheKey := actor + ":" + repo
	mcpServerCacheLog.Printf("Caching permission: actor=%s, repo=%s, permission=%s", actor, repo, permission)
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
	c.repo = &repoEntry{
		repository: repository,
		timestamp:  time.Now(),
	}
	c.mu.Unlock()
}

var mcpCache = newMCPCacheStore()
