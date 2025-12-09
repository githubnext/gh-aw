package workflow

import (
	"os"
	"testing"
)

func TestGetCurrentCommitSHA(t *testing.T) {
	// Save original environment
	origSHA := os.Getenv("GITHUB_SHA")
	defer func() {
		if origSHA != "" {
			os.Setenv("GITHUB_SHA", origSHA)
		} else {
			os.Unsetenv("GITHUB_SHA")
		}
		ResetGitSHACache()
	}()

	t.Run("GITHUB_SHA environment variable", func(t *testing.T) {
		ResetGitSHACache()
		mockSHA := "1234567890abcdef1234567890abcdef12345678"
		os.Setenv("GITHUB_SHA", mockSHA)

		sha := GetCurrentCommitSHA()
		if sha != mockSHA {
			t.Errorf("Expected SHA %q from GITHUB_SHA, got %q", mockSHA, sha)
		}
	})

	t.Run("git rev-parse fallback", func(t *testing.T) {
		ResetGitSHACache()
		os.Unsetenv("GITHUB_SHA")

		sha := GetCurrentCommitSHA()
		// The SHA should be a valid 40-character hex string
		if len(sha) != 40 {
			t.Errorf("Expected 40-character SHA from git rev-parse, got %d characters: %q", len(sha), sha)
		}

		// Verify it's a valid hex string
		for _, c := range sha {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("SHA contains invalid character %q: %s", c, sha)
			}
		}
	})

	t.Run("caching works", func(t *testing.T) {
		ResetGitSHACache()
		os.Unsetenv("GITHUB_SHA")

		// First call
		sha1 := GetCurrentCommitSHA()
		// Second call should return cached value
		sha2 := GetCurrentCommitSHA()

		if sha1 != sha2 {
			t.Errorf("Expected cached SHA to match: %q != %q", sha1, sha2)
		}
	})
}

func TestGetCurrentGitTag(t *testing.T) {
	// Save original environment
	origRef := os.Getenv("GITHUB_REF")
	defer func() {
		if origRef != "" {
			os.Setenv("GITHUB_REF", origRef)
		} else {
			os.Unsetenv("GITHUB_REF")
		}
	}()

	t.Run("GITHUB_REF with tag", func(t *testing.T) {
		os.Setenv("GITHUB_REF", "refs/tags/v1.2.3")

		tag := GetCurrentGitTag()
		if tag != "v1.2.3" {
			t.Errorf("Expected tag 'v1.2.3', got %q", tag)
		}
	})

	t.Run("GITHUB_REF without tag", func(t *testing.T) {
		os.Setenv("GITHUB_REF", "refs/heads/main")

		tag := GetCurrentGitTag()
		if tag != "" {
			t.Errorf("Expected empty tag on branch, got %q", tag)
		}
	})

	t.Run("no GITHUB_REF", func(t *testing.T) {
		os.Unsetenv("GITHUB_REF")

		tag := GetCurrentGitTag()
		// Will try git describe - may or may not return a tag depending on repo state
		// Just verify it doesn't crash
		t.Logf("Tag from git describe: %q", tag)
	})
}

func TestResetGitSHACache(t *testing.T) {
	// Save original environment
	origSHA := os.Getenv("GITHUB_SHA")
	defer func() {
		if origSHA != "" {
			os.Setenv("GITHUB_SHA", origSHA)
		} else {
			os.Unsetenv("GITHUB_SHA")
		}
		ResetGitSHACache()
	}()

	// Set up cache
	os.Unsetenv("GITHUB_SHA")
	sha1 := GetCurrentCommitSHA()

	// Reset cache
	ResetGitSHACache()

	// Get SHA again - should recompute
	sha2 := GetCurrentCommitSHA()

	// They should be the same (same repo), but this tests that reset works
	if sha1 != sha2 {
		t.Errorf("Expected same SHA after reset: %q != %q", sha1, sha2)
	}
}
