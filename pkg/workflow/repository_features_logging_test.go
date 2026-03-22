//go:build !integration

package workflow

import (
	"testing"
)

// TestGetRepositoryFeaturesLogOnce verifies that verbose logging only happens once per repository
func TestGetRepositoryFeaturesLogOnce(t *testing.T) {
	// This test verifies the structure but can't fully test the logging without API access
	// The key behavior is that the logged cache prevents duplicate logging

	testRepo := "owner/repo"

	// First time: LoadOrStore returns loaded=false, should log (if verbose=true and API succeeds)
	// Second time: LoadOrStore returns loaded=true, should NOT log even if verbose=true

	// We can't actually test the full flow without API access, but we can verify:
	// 1. The logged cache mechanism exists
	// 2. LoadOrStore is used correctly

	// Verify that logged cache can be used
	_, loaded := repositoryFeaturesLoggedCache.LoadOrStore(testRepo, true)
	if loaded {
		t.Error("First LoadOrStore should return loaded=false")
	}

	// Second call should return loaded=true
	_, loaded = repositoryFeaturesLoggedCache.LoadOrStore(testRepo, true)
	if !loaded {
		t.Error("Second LoadOrStore should return loaded=true")
	}
}
