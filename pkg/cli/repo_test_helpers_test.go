//go:build integration || !integration

package cli

// ClearCurrentRepoSlugCache clears the current repository slug cache.
// This is useful for testing when repository context might have changed.
func ClearCurrentRepoSlugCache() {
	currentRepoSlugCache.mu.Lock()
	defer currentRepoSlugCache.mu.Unlock()
	currentRepoSlugCache.result = ""
	currentRepoSlugCache.err = nil
	currentRepoSlugCache.done = false
}
