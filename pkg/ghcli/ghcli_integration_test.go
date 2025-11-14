package ghcli

import (
	"os"
	"testing"
)

// TestExecGHIntegration verifies that ExecGH properly sets up the gh CLI command
// with correct token resolution from environment variables.
// This is an integration test that validates the fix for remote import compilation.
func TestExecGHIntegration(t *testing.T) {
	// Save original environment
	origGHToken := os.Getenv("GH_TOKEN")
	origGitHubToken := os.Getenv("GITHUB_TOKEN")

	// Clean up after test
	defer func() {
		if origGHToken != "" {
			os.Setenv("GH_TOKEN", origGHToken)
		} else {
			os.Unsetenv("GH_TOKEN")
		}
		if origGitHubToken != "" {
			os.Setenv("GITHUB_TOKEN", origGitHubToken)
		} else {
			os.Unsetenv("GITHUB_TOKEN")
		}
	}()

	// Test scenario: GITHUB_TOKEN is set but GH_TOKEN is not
	// This is the common case in GitHub Actions where compilation fails with remote imports
	os.Unsetenv("GH_TOKEN")
	os.Setenv("GITHUB_TOKEN", "test-github-token-123")

	cmd := ExecGH("api", "/user")

	// Verify that the command has GH_TOKEN set in its environment
	found := false
	expectedEnv := "GH_TOKEN=test-github-token-123"
	if cmd.Env != nil {
		for _, env := range cmd.Env {
			if env == expectedEnv {
				found = true
				break
			}
		}
	}

	if !found {
		t.Errorf("Expected GH_TOKEN to be set in command environment from GITHUB_TOKEN, but it was not")
		t.Logf("Command Env: %v", cmd.Env)
	}

	// Verify command args are correct
	expectedArgs := []string{"gh", "api", "/user"}
	if len(cmd.Args) != len(expectedArgs) {
		t.Errorf("Expected args %v, got: %v", expectedArgs, cmd.Args)
		return
	}
	for i, arg := range expectedArgs {
		if cmd.Args[i] != arg {
			t.Errorf("Expected arg[%d] to be %q, got %q", i, arg, cmd.Args[i])
		}
	}
}
