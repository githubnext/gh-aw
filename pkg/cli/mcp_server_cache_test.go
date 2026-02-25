//go:build !integration

package cli

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestMCPCacheStore_ConcurrentPermissionAccess(t *testing.T) {
	cache := newMCPCacheStore()
	cache.permissionTTL = 50 * time.Millisecond

	// Pre-populate
	for i := range 5 {
		cache.SetPermission(fmt.Sprintf("actor%d", i), "owner/repo", "write")
	}

	const numGoroutines = 20
	const numIterations = 100

	var wg sync.WaitGroup

	for range numGoroutines {
		wg.Go(func() {
			for i := range numIterations {
				actor := fmt.Sprintf("actor%d", i%10)
				cache.GetPermission(actor, "owner/repo")
				cache.SetPermission(actor, "owner/repo", "write")
			}
		})
	}

	wg.Wait()
}

func TestMCPCacheStore_ConcurrentRepoAccess(t *testing.T) {
	cache := newMCPCacheStore()
	cache.repoTTL = 50 * time.Millisecond

	const numGoroutines = 20
	const numIterations = 100

	var wg sync.WaitGroup

	for g := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for range numIterations {
				cache.GetRepo()
				cache.SetRepo(fmt.Sprintf("owner/repo-%d", id))
			}
		}(g)
	}

	wg.Wait()
}

func TestMCPCacheStore_PermissionExpiry(t *testing.T) {
	cache := newMCPCacheStore()
	cache.permissionTTL = 10 * time.Millisecond

	cache.SetPermission("actor", "owner/repo", "admin")

	// Should hit cache
	perm, ok := cache.GetPermission("actor", "owner/repo")
	if !ok || perm != "admin" {
		t.Errorf("GetPermission() = (%q, %v), want (\"admin\", true)", perm, ok)
	}

	// Wait for expiry
	time.Sleep(20 * time.Millisecond)

	// Should miss cache
	_, ok = cache.GetPermission("actor", "owner/repo")
	if ok {
		t.Error("GetPermission() should return false after TTL expiry")
	}
}

func TestMCPCacheStore_RepoExpiry(t *testing.T) {
	cache := newMCPCacheStore()
	cache.repoTTL = 10 * time.Millisecond

	cache.SetRepo("owner/repo")

	// Should hit cache
	repo, ok := cache.GetRepo()
	if !ok || repo != "owner/repo" {
		t.Errorf("GetRepo() = (%q, %v), want (\"owner/repo\", true)", repo, ok)
	}

	// Wait for expiry
	time.Sleep(20 * time.Millisecond)

	// Should miss cache
	_, ok = cache.GetRepo()
	if ok {
		t.Error("GetRepo() should return false after TTL expiry")
	}
}
