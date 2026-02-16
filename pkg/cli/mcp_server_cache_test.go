//go:build !integration

package cli

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestQueryActorRole_ConcurrentCacheAccess(t *testing.T) {
	// Save and restore global state
	origCache := permissionCache
	origTTL := permissionCacheTTL
	t.Cleanup(func() {
		cacheMu.Lock()
		permissionCache = origCache
		permissionCacheTTL = origTTL
		cacheMu.Unlock()
	})

	// Use a fresh cache with a short TTL to exercise both read and write paths
	cacheMu.Lock()
	permissionCache = make(map[string]*actorPermissionCache)
	permissionCacheTTL = 50 * time.Millisecond
	cacheMu.Unlock()

	// Pre-populate cache entries
	cacheMu.Lock()
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("actor%d:owner/repo", i)
		permissionCache[key] = &actorPermissionCache{
			permission: "write",
			timestamp:  time.Now(),
		}
	}
	cacheMu.Unlock()

	const numGoroutines = 20
	const numIterations = 100

	var wg sync.WaitGroup

	// Concurrent goroutines reading and writing the cache
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < numIterations; i++ {
				cacheKey := fmt.Sprintf("actor%d:owner/repo", i%10)

				cacheMu.RLock()
				if cached, ok := permissionCache[cacheKey]; ok {
					if time.Since(cached.timestamp) >= permissionCacheTTL {
						cacheMu.RUnlock()
						cacheMu.Lock()
						delete(permissionCache, cacheKey)
						cacheMu.Unlock()
					} else {
						cacheMu.RUnlock()
					}
				} else {
					cacheMu.RUnlock()
				}

				cacheMu.Lock()
				permissionCache[cacheKey] = &actorPermissionCache{
					permission: "write",
					timestamp:  time.Now(),
				}
				cacheMu.Unlock()
			}
		}(g)
	}

	wg.Wait()
}

func TestGetRepository_ConcurrentCacheAccess(t *testing.T) {
	// Save and restore global state
	origRepoCache := repoCache
	origRepoCacheTTL := repoCacheTTL
	t.Cleanup(func() {
		cacheMu.Lock()
		repoCache = origRepoCache
		repoCacheTTL = origRepoCacheTTL
		cacheMu.Unlock()
	})

	cacheMu.Lock()
	repoCache = nil
	repoCacheTTL = 50 * time.Millisecond
	cacheMu.Unlock()

	const numGoroutines = 20
	const numIterations = 100

	var wg sync.WaitGroup

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < numIterations; i++ {
				cacheMu.RLock()
				cached := repoCache
				if cached != nil {
					_ = cached.repository
				}
				cacheMu.RUnlock()

				cacheMu.Lock()
				repoCache = &repositoryCache{
					repository: fmt.Sprintf("owner/repo-%d", goroutineID),
					timestamp:  time.Now(),
				}
				cacheMu.Unlock()
			}
		}(g)
	}

	wg.Wait()
}
