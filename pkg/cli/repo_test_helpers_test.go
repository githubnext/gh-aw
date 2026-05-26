//go:build integration || !integration

package cli

import "github.com/github/gh-aw/pkg/syncutil"

// ClearCurrentRepoSlugCache clears the current repository slug cache.
// This is useful for testing when repository context might have changed.
func ClearCurrentRepoSlugCache() {
	currentRepoSlugCache = syncutil.OnceLoader[string]{}
}
