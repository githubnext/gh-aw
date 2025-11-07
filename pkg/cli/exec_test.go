package cli

import (
	"os"
	"strings"
	"testing"
)

func TestGhExecOrFallback(t *testing.T) {
	tests := []struct {
		name         string
		ghToken      string
		fallbackCmd  string
		fallbackArgs []string
		fallbackEnv  []string
		expectError  bool
		description  string
	}{
		{
			name:         "uses git when GH_TOKEN not set",
			ghToken:      "",
			fallbackCmd:  "echo",
			fallbackArgs: []string{"fallback executed"},
			fallbackEnv:  nil,
			expectError:  false,
			description:  "should use fallback command when GH_TOKEN is not set",
		},
		{
			name:         "uses fallback with custom env",
			ghToken:      "",
			fallbackCmd:  "sh",
			fallbackArgs: []string{"-c", "echo $TEST_VAR"},
			fallbackEnv:  []string{"TEST_VAR=test_value"},
			expectError:  false,
			description:  "should pass custom environment variables to fallback command",
		},
		{
			name:         "fallback command failure",
			ghToken:      "",
			fallbackCmd:  "false", // command that always fails
			fallbackArgs: []string{},
			fallbackEnv:  nil,
			expectError:  true,
			description:  "should return error when fallback command fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set or unset GH_TOKEN based on test case
			originalToken := os.Getenv("GH_TOKEN")
			defer func() {
				if originalToken != "" {
					os.Setenv("GH_TOKEN", originalToken)
				} else {
					os.Unsetenv("GH_TOKEN")
				}
			}()

			if tt.ghToken != "" {
				os.Setenv("GH_TOKEN", tt.ghToken)
			} else {
				os.Unsetenv("GH_TOKEN")
			}

			stdout, _, err := ghExecOrFallback(tt.fallbackCmd, tt.fallbackArgs, tt.fallbackEnv)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for test '%s', got nil", tt.description)
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for test '%s': %v", tt.description, err)
			}

			// For successful fallback tests, verify output
			if !tt.expectError && tt.fallbackCmd == "echo" {
				if !strings.Contains(stdout, "fallback executed") {
					t.Errorf("Expected stdout to contain 'fallback executed', got: %s", stdout)
				}
			}

			// For env test, verify environment variable was passed
			if !tt.expectError && tt.fallbackCmd == "sh" && len(tt.fallbackEnv) > 0 {
				if !strings.Contains(stdout, "test_value") {
					t.Errorf("Expected stdout to contain 'test_value', got: %s", stdout)
				}
			}

			// With separated stdout/stderr, we don't expect both to be populated
			// This is a change from the previous CombinedOutput behavior
		})
	}
}

func TestGhExecOrFallbackWithGHToken(t *testing.T) {
	// This test verifies behavior when GH_TOKEN is set
	// Note: We can't easily test actual gh.Exec without a real token,
	// so we test that the function attempts to use gh CLI

	originalToken := os.Getenv("GH_TOKEN")
	defer func() {
		if originalToken != "" {
			os.Setenv("GH_TOKEN", originalToken)
		} else {
			os.Unsetenv("GH_TOKEN")
		}
	}()

	// Set a dummy token
	os.Setenv("GH_TOKEN", "dummy_token_for_test")

	// This will likely fail since we don't have a valid token,
	// but we're testing that it attempts gh.Exec path
	_, _, err := ghExecOrFallback(
		"echo",
		[]string{"fallback"},
		nil,
	)

	// We expect an error because gh.Exec will fail with invalid token/nonexistent repo
	// The important part is that it tried the gh.Exec path
	if err == nil {
		// If it succeeded, it means it used the fallback, which is wrong
		t.Error("Expected function to attempt gh.Exec with GH_TOKEN set")
	}
}

func TestGhExecOrFallbackIntegration(t *testing.T) {
	// Integration test: verify the function works end-to-end without GH_TOKEN
	originalToken := os.Getenv("GH_TOKEN")
	defer func() {
		if originalToken != "" {
			os.Setenv("GH_TOKEN", originalToken)
		} else {
			os.Unsetenv("GH_TOKEN")
		}
	}()

	os.Unsetenv("GH_TOKEN")

	// Use a simple command that we know will work
	stdout, _, err := ghExecOrFallback(
		"echo",
		[]string{"integration test output"},
		nil,
	)

	if err != nil {
		t.Errorf("Unexpected error in integration test: %v", err)
	}

	if !strings.Contains(stdout, "integration test output") {
		t.Errorf("Expected output to contain 'integration test output', got: %s", stdout)
	}
}

func TestExtractRepoSlug(t *testing.T) {
	tests := []struct {
		name         string
		repoURL      string
		githubHost   string
		expectedSlug string
	}{
		{
			name:         "standard GitHub URL",
			repoURL:      "https://github.com/owner/repo",
			githubHost:   "",
			expectedSlug: "owner/repo",
		},
		{
			name:         "GitHub URL with .git suffix",
			repoURL:      "https://github.com/owner/repo.git",
			githubHost:   "",
			expectedSlug: "owner/repo",
		},
		{
			name:         "enterprise GitHub URL",
			repoURL:      "https://github.enterprise.com/owner/repo",
			githubHost:   "https://github.enterprise.com",
			expectedSlug: "owner/repo",
		},
		{
			name:         "enterprise GitHub URL with .git",
			repoURL:      "https://github.enterprise.com/owner/repo.git",
			githubHost:   "https://github.enterprise.com",
			expectedSlug: "owner/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env
			originalHost := os.Getenv("GITHUB_SERVER_URL")
			defer func() {
				if originalHost != "" {
					os.Setenv("GITHUB_SERVER_URL", originalHost)
				} else {
					os.Unsetenv("GITHUB_SERVER_URL")
				}
			}()

			if tt.githubHost != "" {
				os.Setenv("GITHUB_SERVER_URL", tt.githubHost)
			} else {
				os.Unsetenv("GITHUB_SERVER_URL")
			}

			slug := extractRepoSlug(tt.repoURL)
			if slug != tt.expectedSlug {
				t.Errorf("Expected slug '%s', got '%s'", tt.expectedSlug, slug)
			}
		})
	}
}

func TestInferGhArgs(t *testing.T) {
	tests := []struct {
		name           string
		fallbackCmd    string
		fallbackArgs   []string
		expectedGhArgs []string
	}{
		{
			name:           "git clone simple",
			fallbackCmd:    "git",
			fallbackArgs:   []string{"clone", "https://github.com/owner/repo", "/tmp/dir"},
			expectedGhArgs: []string{"repo", "clone", "owner/repo", "/tmp/dir"},
		},
		{
			name:           "git clone with depth",
			fallbackCmd:    "git",
			fallbackArgs:   []string{"clone", "--depth", "1", "https://github.com/owner/repo", "/tmp/dir"},
			expectedGhArgs: []string{"repo", "clone", "owner/repo", "/tmp/dir", "--", "--depth", "1"},
		},
		{
			name:           "git clone with branch",
			fallbackCmd:    "git",
			fallbackArgs:   []string{"clone", "--depth", "1", "https://github.com/owner/repo", "/tmp/dir", "--branch", "main"},
			expectedGhArgs: []string{"repo", "clone", "owner/repo", "/tmp/dir", "--", "--depth", "1", "--branch", "main"},
		},
		{
			name:           "git checkout",
			fallbackCmd:    "git",
			fallbackArgs:   []string{"-C", "/tmp/dir", "checkout", "abc123"},
			expectedGhArgs: []string{"exec", "--", "git", "-C", "/tmp/dir", "checkout", "abc123"},
		},
		{
			name:           "non-git command",
			fallbackCmd:    "echo",
			fallbackArgs:   []string{"hello"},
			expectedGhArgs: []string{"exec", "--", "echo", "hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ghArgs := inferGhArgs(tt.fallbackCmd, tt.fallbackArgs)
			if len(ghArgs) != len(tt.expectedGhArgs) {
				t.Errorf("Expected %d args, got %d: %v", len(tt.expectedGhArgs), len(ghArgs), ghArgs)
				return
			}
			for i, arg := range ghArgs {
				if arg != tt.expectedGhArgs[i] {
					t.Errorf("Arg %d: expected '%s', got '%s'", i, tt.expectedGhArgs[i], arg)
				}
			}
		})
	}
}
