//go:build !integration

package cli

import (
	"os/exec"
	"strings"
	"testing"
)

func TestTrialRepositoryURLHelpers(t *testing.T) {
	tests := []struct {
		name               string
		serverURL          string
		enterpriseHost     string
		githubHost         string
		ghHost             string
		repoSlug           string
		expectedRepoURL    string
		expectedGitURL     string
		expectedActionsURL string
	}{
		{
			name:               "defaults to github.com",
			repoSlug:           "owner/repo",
			expectedRepoURL:    "https://github.com/owner/repo",
			expectedGitURL:     "https://github.com/owner/repo.git",
			expectedActionsURL: "https://github.com/owner/repo/settings/actions",
		},
		{
			name:               "uses GH_HOST for trial repository URLs",
			ghHost:             "example.ghe.com",
			repoSlug:           "owner/repo",
			expectedRepoURL:    "https://example.ghe.com/owner/repo",
			expectedGitURL:     "https://example.ghe.com/owner/repo.git",
			expectedActionsURL: "https://example.ghe.com/owner/repo/settings/actions",
		},
		{
			name:               "GITHUB_SERVER_URL takes precedence over GH_HOST",
			serverURL:          "https://server.ghe.com/",
			ghHost:             "example.ghe.com",
			repoSlug:           "owner/repo",
			expectedRepoURL:    "https://server.ghe.com/owner/repo",
			expectedGitURL:     "https://server.ghe.com/owner/repo.git",
			expectedActionsURL: "https://server.ghe.com/owner/repo/settings/actions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_SERVER_URL", tt.serverURL)
			t.Setenv("GITHUB_ENTERPRISE_HOST", tt.enterpriseHost)
			t.Setenv("GITHUB_HOST", tt.githubHost)
			t.Setenv("GH_HOST", tt.ghHost)

			if got := trialRepositoryURL(tt.repoSlug); got != tt.expectedRepoURL {
				t.Fatalf("trialRepositoryURL() = %q, want %q", got, tt.expectedRepoURL)
			}
			if got := trialRepositoryGitURL(tt.repoSlug); got != tt.expectedGitURL {
				t.Fatalf("trialRepositoryGitURL() = %q, want %q", got, tt.expectedGitURL)
			}
			if got := trialRepositoryActionsSettingsURL(tt.repoSlug); got != tt.expectedActionsURL {
				t.Fatalf("trialRepositoryActionsSettingsURL() = %q, want %q", got, tt.expectedActionsURL)
			}
		})
	}
}

func TestGetCurrentBranchIn(t *testing.T) {
	// initRepo creates a minimal git repo in dir with the given branch name.
	initRepo := func(t *testing.T, dir, branch string) {
		t.Helper()
		run := func(args ...string) {
			t.Helper()
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Dir = dir
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("command %v failed: %v (output: %s)", args, err, out)
			}
		}
		run("git", "init")
		run("git", "config", "user.email", "test@example.com")
		run("git", "config", "user.name", "Test")
		run("git", "symbolic-ref", "HEAD", "refs/heads/"+branch)
		run("git", "commit", "--allow-empty", "-m", "init")
	}

	t.Run("returns main for a repo using main", func(t *testing.T) {
		dir := t.TempDir()
		initRepo(t, dir, "main")
		got, err := getCurrentBranchIn(dir)
		if err != nil {
			t.Fatalf("getCurrentBranchIn() unexpected error: %v", err)
		}
		if got != "main" {
			t.Fatalf("getCurrentBranchIn() = %q, want %q", got, "main")
		}
	})

	t.Run("returns master for a repo using master", func(t *testing.T) {
		dir := t.TempDir()
		initRepo(t, dir, "master")
		got, err := getCurrentBranchIn(dir)
		if err != nil {
			t.Fatalf("getCurrentBranchIn() unexpected error: %v", err)
		}
		if got != "master" {
			t.Fatalf("getCurrentBranchIn() = %q, want %q", got, "master")
		}
	})

	t.Run("returns custom branch name", func(t *testing.T) {
		dir := t.TempDir()
		initRepo(t, dir, "trunk")
		got, err := getCurrentBranchIn(dir)
		if err != nil {
			t.Fatalf("getCurrentBranchIn() unexpected error: %v", err)
		}
		if got != "trunk" {
			t.Fatalf("getCurrentBranchIn() = %q, want %q", got, "trunk")
		}
	})

	t.Run("returns error for non-git directory", func(t *testing.T) {
		dir := t.TempDir()
		_, err := getCurrentBranchIn(dir)
		if err == nil {
			t.Fatal("getCurrentBranchIn() expected an error for non-git directory, got nil")
		}
	})

	t.Run("returns error in detached HEAD state", func(t *testing.T) {
		dir := t.TempDir()
		initRepo(t, dir, "main")
		// Detach HEAD by checking out the commit hash directly.
		cmd := exec.Command("git", "rev-parse", "HEAD")
		cmd.Dir = dir
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("git rev-parse HEAD failed: %v", err)
		}
		hash := strings.TrimSpace(string(out))
		detach := exec.Command("git", "checkout", hash)
		detach.Dir = dir
		if out, err := detach.CombinedOutput(); err != nil {
			t.Fatalf("git checkout %s failed: %v (output: %s)", hash, err, out)
		}
		_, err = getCurrentBranchIn(dir)
		if err == nil {
			t.Fatal("getCurrentBranchIn() expected an error in detached HEAD state, got nil")
		}
	})
}
