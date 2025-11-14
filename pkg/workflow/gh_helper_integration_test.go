// go:build integration
//go:build integration

package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestExecGHWithRESTFallback_RealAPI(t *testing.T) {
	// Save original environment
	originalGHToken := os.Getenv("GH_TOKEN")
	originalGitHubToken := os.Getenv("GITHUB_TOKEN")
	defer func() {
		os.Setenv("GH_TOKEN", originalGHToken)
		os.Setenv("GITHUB_TOKEN", originalGitHubToken)
	}()

	// Clear tokens to force REST API fallback
	os.Unsetenv("GH_TOKEN")
	os.Unsetenv("GITHUB_TOKEN")

	t.Run("fallback to REST API for public repository tag", func(t *testing.T) {
		// Test with a known public repository and tag
		output, fromREST, err := ExecGHWithRESTFallback(
			"api",
			"/repos/actions/checkout/git/ref/tags/v4",
			"--jq", ".object.sha",
		)

		if err != nil {
			t.Fatalf("Expected fallback to succeed, got error: %v", err)
		}

		if !fromREST {
			t.Logf("gh CLI succeeded (gh is configured), but we expected REST fallback")
			// This is OK - if gh CLI is configured, it will succeed before trying REST
			// The important thing is that the command succeeded
		}

		// Verify we got a valid SHA (40 hex characters)
		sha := strings.TrimSpace(string(output))
		if len(sha) != 40 {
			t.Errorf("Expected 40 character SHA, got: %s (length %d)", sha, len(sha))
		}

		// Verify it's all hex characters
		for _, c := range sha {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("Expected hex SHA, got character %c in: %s", c, sha)
				break
			}
		}

		t.Logf("Successfully resolved SHA: %s (fromREST: %v)", sha, fromREST)
	})

	t.Run("fallback handles nested field extraction", func(t *testing.T) {
		// Test with a repository info endpoint
		output, fromREST, err := ExecGHWithRESTFallback(
			"api",
			"/repos/actions/checkout",
			"--jq", ".name",
		)

		if err != nil {
			t.Fatalf("Expected fallback to succeed, got error: %v", err)
		}

		name := strings.TrimSpace(string(output))
		if name != "checkout" {
			t.Errorf("Expected repository name 'checkout', got: %s", name)
		}

		t.Logf("Successfully extracted name: %s (fromREST: %v)", name, fromREST)
	})

	t.Run("fallback returns error for non-existent repository", func(t *testing.T) {
		output, fromREST, err := ExecGHWithRESTFallback(
			"api",
			"/repos/nonexistent-owner-12345/nonexistent-repo-67890/git/ref/tags/v1.0",
			"--jq", ".object.sha",
		)

		// This should fail (either gh CLI or REST API)
		if err == nil {
			t.Errorf("Expected error for non-existent repository, but got output: %s", string(output))
		}

		t.Logf("Correctly failed for non-existent repository (fromREST: %v): %v", fromREST, err)
	})
}

func TestActionResolver_WithRESTFallback(t *testing.T) {
	// Save original environment
	originalGHToken := os.Getenv("GH_TOKEN")
	originalGitHubToken := os.Getenv("GITHUB_TOKEN")
	defer func() {
		os.Setenv("GH_TOKEN", originalGHToken)
		os.Setenv("GITHUB_TOKEN", originalGitHubToken)
	}()

	// Clear tokens to force REST API fallback
	os.Unsetenv("GH_TOKEN")
	os.Unsetenv("GITHUB_TOKEN")

	// Create a temporary directory for cache
	tmpDir := t.TempDir()
	cache := NewActionCache(tmpDir)
	resolver := NewActionResolver(cache)

	t.Run("resolve action SHA using REST API fallback", func(t *testing.T) {
		// Test resolving a real action
		sha, err := resolver.ResolveSHA("actions/checkout", "v4")

		if err != nil {
			t.Fatalf("Expected resolution to succeed via REST API fallback, got error: %v", err)
		}

		// Verify we got a valid SHA
		if len(sha) != 40 {
			t.Errorf("Expected 40 character SHA, got: %s (length %d)", sha, len(sha))
		}

		t.Logf("Successfully resolved actions/checkout@v4 to SHA: %s", sha)

		// Verify caching works
		cachedSHA, found := cache.Get("actions/checkout", "v4")
		if !found {
			t.Errorf("Expected SHA to be cached after resolution")
		}
		if cachedSHA != sha {
			t.Errorf("Cached SHA %s doesn't match resolved SHA %s", cachedSHA, sha)
		}
	})

	t.Run("resolve complex action path", func(t *testing.T) {
		// Test with a complex action path (has subdirectory)
		sha, err := resolver.ResolveSHA("github/codeql-action/upload-sarif", "v3")

		if err != nil {
			t.Fatalf("Expected resolution to succeed, got error: %v", err)
		}

		if len(sha) != 40 {
			t.Errorf("Expected 40 character SHA, got: %s (length %d)", sha, len(sha))
		}

		t.Logf("Successfully resolved github/codeql-action/upload-sarif@v3 to SHA: %s", sha)
	})
}

func TestCallGitHubRESTAPI_RealEndpoint(t *testing.T) {
	t.Run("fetch repository info without authentication", func(t *testing.T) {
		// Test direct REST API call to a public repository
		output, err := callGitHubRESTAPI("/repos/actions/checkout", "")

		if err != nil {
			t.Fatalf("Expected REST API call to succeed, got error: %v", err)
		}

		// Verify we got JSON response
		if len(output) == 0 {
			t.Errorf("Expected non-empty response")
		}

		// The response should contain repository information
		if !strings.Contains(string(output), "checkout") {
			t.Errorf("Expected response to contain 'checkout', got: %s", string(output[:100]))
		}

		t.Logf("Successfully fetched repository info (%d bytes)", len(output))
	})

	t.Run("fetch with jq filter", func(t *testing.T) {
		output, err := callGitHubRESTAPI("/repos/actions/checkout", ".name")

		if err != nil {
			t.Fatalf("Expected REST API call to succeed, got error: %v", err)
		}

		name := strings.TrimSpace(string(output))
		if name != "checkout" {
			t.Errorf("Expected 'checkout', got: %s", name)
		}

		t.Logf("Successfully extracted name: %s", name)
	})

	t.Run("fetch nested field with jq filter", func(t *testing.T) {
		output, err := callGitHubRESTAPI("/repos/actions/checkout/git/ref/tags/v4", ".object.sha")

		if err != nil {
			t.Fatalf("Expected REST API call to succeed, got error: %v", err)
		}

		sha := strings.TrimSpace(string(output))
		if len(sha) != 40 {
			t.Errorf("Expected 40 character SHA, got: %s (length %d)", sha, len(sha))
		}

		t.Logf("Successfully extracted SHA: %s", sha)
	})

	t.Run("handle 404 error gracefully", func(t *testing.T) {
		_, err := callGitHubRESTAPI("/repos/nonexistent-owner/nonexistent-repo", "")

		if err == nil {
			t.Errorf("Expected error for non-existent repository")
		}

		// Verify error message contains status code
		if !strings.Contains(err.Error(), "404") {
			t.Errorf("Expected error to mention 404, got: %v", err)
		}

		t.Logf("Correctly handled 404 error: %v", err)
	})
}
