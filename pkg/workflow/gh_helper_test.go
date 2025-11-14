package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestExecGH(t *testing.T) {
	tests := []struct {
		name          string
		ghToken       string
		githubToken   string
		expectGHToken bool
		expectValue   string
	}{
		{
			name:          "GH_TOKEN is set",
			ghToken:       "gh-token-123",
			githubToken:   "",
			expectGHToken: false, // Should use existing GH_TOKEN from environment
			expectValue:   "",
		},
		{
			name:          "GITHUB_TOKEN is set, GH_TOKEN is not",
			ghToken:       "",
			githubToken:   "github-token-456",
			expectGHToken: true,
			expectValue:   "github-token-456",
		},
		{
			name:          "Both GH_TOKEN and GITHUB_TOKEN are set",
			ghToken:       "gh-token-123",
			githubToken:   "github-token-456",
			expectGHToken: false, // Should prefer existing GH_TOKEN
			expectValue:   "",
		},
		{
			name:          "Neither GH_TOKEN nor GITHUB_TOKEN is set",
			ghToken:       "",
			githubToken:   "",
			expectGHToken: false,
			expectValue:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalGHToken := os.Getenv("GH_TOKEN")
			originalGitHubToken := os.Getenv("GITHUB_TOKEN")
			defer func() {
				os.Setenv("GH_TOKEN", originalGHToken)
				os.Setenv("GITHUB_TOKEN", originalGitHubToken)
			}()

			// Set up test environment
			if tt.ghToken != "" {
				os.Setenv("GH_TOKEN", tt.ghToken)
			} else {
				os.Unsetenv("GH_TOKEN")
			}
			if tt.githubToken != "" {
				os.Setenv("GITHUB_TOKEN", tt.githubToken)
			} else {
				os.Unsetenv("GITHUB_TOKEN")
			}

			// Execute the helper
			cmd := ExecGH("api", "/user")

			// Verify the command
			if cmd.Path != "gh" && !strings.HasSuffix(cmd.Path, "/gh") {
				t.Errorf("Expected command path to be 'gh', got: %s", cmd.Path)
			}

			// Verify arguments
			if len(cmd.Args) != 3 || cmd.Args[1] != "api" || cmd.Args[2] != "/user" {
				t.Errorf("Expected args [gh api /user], got: %v", cmd.Args)
			}

			// Verify environment
			if tt.expectGHToken {
				found := false
				expectedEnv := "GH_TOKEN=" + tt.expectValue
				for _, env := range cmd.Env {
					if env == expectedEnv {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected environment to contain %s, but it wasn't found", expectedEnv)
				}
			} else {
				// When GH_TOKEN is already set or neither token is set, cmd.Env should be nil (uses parent process env)
				if cmd.Env != nil {
					t.Errorf("Expected cmd.Env to be nil (inherit parent environment), got: %v", cmd.Env)
				}
			}
		})
	}
}

func TestExecGHWithMultipleArgs(t *testing.T) {
	// Save original environment
	originalGHToken := os.Getenv("GH_TOKEN")
	originalGitHubToken := os.Getenv("GITHUB_TOKEN")
	defer func() {
		os.Setenv("GH_TOKEN", originalGHToken)
		os.Setenv("GITHUB_TOKEN", originalGitHubToken)
	}()

	// Set up test environment
	os.Unsetenv("GH_TOKEN")
	os.Setenv("GITHUB_TOKEN", "test-token")

	// Test with multiple arguments
	cmd := ExecGH("api", "repos/owner/repo/git/ref/tags/v1.0", "--jq", ".object.sha")

	// Verify command
	if cmd.Path != "gh" && !strings.HasSuffix(cmd.Path, "/gh") {
		t.Errorf("Expected command path to be 'gh', got: %s", cmd.Path)
	}

	// Verify all arguments are preserved
	expectedArgs := []string{"gh", "api", "repos/owner/repo/git/ref/tags/v1.0", "--jq", ".object.sha"}
	if len(cmd.Args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d: %v", len(expectedArgs), len(cmd.Args), cmd.Args)
	}

	for i, expected := range expectedArgs {
		if i >= len(cmd.Args) || cmd.Args[i] != expected {
			t.Errorf("Arg %d: expected %s, got %s", i, expected, cmd.Args[i])
		}
	}

	// Verify environment contains GH_TOKEN
	found := false
	for _, env := range cmd.Env {
		if env == "GH_TOKEN=test-token" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected environment to contain GH_TOKEN=test-token")
	}
}

func TestApplyJQFilter(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		filter      string
		expected    string
		expectError bool
	}{
		{
			name:     "simple field extraction",
			jsonData: `{"name": "test"}`,
			filter:   ".name",
			expected: "test\n",
		},
		{
			name:     "nested field extraction",
			jsonData: `{"object": {"sha": "abc123"}}`,
			filter:   ".object.sha",
			expected: "abc123\n",
		},
		{
			name:     "number field",
			jsonData: `{"count": 42}`,
			filter:   ".count",
			expected: "42\n",
		},
		{
			name:     "boolean field",
			jsonData: `{"enabled": true}`,
			filter:   ".enabled",
			expected: "true\n",
		},
		{
			name:        "missing field",
			jsonData:    `{"name": "test"}`,
			filter:      ".missing",
			expectError: true,
		},
		{
			name:        "invalid JSON",
			jsonData:    `{invalid}`,
			filter:      ".name",
			expectError: true,
		},
		{
			name:     "complex object",
			jsonData: `{"meta": {"nested": {"value": "deep"}}}`,
			filter:   ".meta.nested.value",
			expected: "deep\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := applyJQFilter([]byte(tt.jsonData), tt.filter)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if string(result) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

func TestCallGitHubRESTAPI(t *testing.T) {
	// This is a unit test that verifies the function exists and has the right signature
	// Integration tests would require network access

	t.Run("function exists and accepts correct parameters", func(t *testing.T) {
		// Test that the function can be called (will fail due to network, but that's expected)
		_, err := callGitHubRESTAPI("/repos/actions/checkout/git/ref/tags/v4", ".object.sha")
		
		// We expect an error (network error in test environment), but we're just verifying
		// the function signature is correct
		if err == nil {
			// If somehow this succeeds (network is available), that's fine too
			t.Log("REST API call succeeded (network available)")
		} else {
			// Expected: network error or API error
			t.Logf("Expected network error in test environment: %v", err)
		}
	})
}

func TestExecGHAPIWithRESTFallback_BasicFunctionality(t *testing.T) {
	// Save original environment
	originalGHToken := os.Getenv("GH_TOKEN")
	originalGitHubToken := os.Getenv("GITHUB_TOKEN")
	defer func() {
		os.Setenv("GH_TOKEN", originalGHToken)
		os.Setenv("GITHUB_TOKEN", originalGitHubToken)
	}()

	// Clear tokens to ensure gh CLI will fail
	os.Unsetenv("GH_TOKEN")
	os.Unsetenv("GITHUB_TOKEN")

	t.Run("function accepts API path and jq filter", func(t *testing.T) {
		// The function should accept the proper parameters
		// Network will likely fail in test environment, but we're testing the signature
		_, fromREST, err := ExecGHAPIWithRESTFallback("/repos/actions/checkout", ".name")
		
		if err == nil {
			t.Logf("Command succeeded (gh is configured or network is available)")
			return
		}

		// We expect either gh CLI error or REST API error, but the function call should work
		t.Logf("Expected error in test environment (fromREST: %v): %v", fromREST, err)
	})

	t.Run("function works without jq filter", func(t *testing.T) {
		// Test with empty jq filter
		_, fromREST, err := ExecGHAPIWithRESTFallback("/repos/actions/checkout", "")
		
		if err == nil {
			t.Logf("Command succeeded (gh is configured or network is available)")
			return
		}

		// We expect either gh CLI error or REST API error
		t.Logf("Expected error in test environment (fromREST: %v): %v", fromREST, err)
	})
}
