package ghcli

import (
	"os"
	"testing"
)

func TestExecGH(t *testing.T) {
	tests := []struct {
		name          string
		ghToken       string
		githubToken   string
		expectedEnv   string // Expected GH_TOKEN value in environment
		shouldHaveEnv bool
	}{
		{
			name:          "GH_TOKEN already set",
			ghToken:       "gh-token-123",
			githubToken:   "github-token-456",
			expectedEnv:   "", // Should use existing GH_TOKEN, not add to env
			shouldHaveEnv: false,
		},
		{
			name:          "GITHUB_TOKEN fallback",
			ghToken:       "",
			githubToken:   "github-token-456",
			expectedEnv:   "GH_TOKEN=github-token-456",
			shouldHaveEnv: true,
		},
		{
			name:          "No tokens set",
			ghToken:       "",
			githubToken:   "",
			expectedEnv:   "",
			shouldHaveEnv: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			// Set test environment
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

			// Create command
			cmd := ExecGH("api", "/user")

			// Verify command was created
			if cmd == nil {
				t.Error("Expected command to be created")
				return
			}

			// Verify command args
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

			// Verify environment variable handling
			if tt.shouldHaveEnv {
				found := false
				if cmd.Env != nil {
					for _, env := range cmd.Env {
						if env == tt.expectedEnv {
							found = true
							break
						}
					}
				}
				if !found {
					t.Errorf("Expected environment to contain %q, got: %v", tt.expectedEnv, cmd.Env)
				}
			} else {
				// When GH_TOKEN is already set, Env should be nil (uses parent process env)
				if tt.ghToken != "" && cmd.Env != nil {
					t.Errorf("Expected Env to be nil when GH_TOKEN is already set, got: %v", cmd.Env)
				}
			}
		})
	}
}
