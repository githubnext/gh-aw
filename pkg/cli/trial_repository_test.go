//go:build !integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
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

// TestMergeDirectory verifies the mergeDirectory helper that copies the source .github/
// folder into the trial host directory.
func TestMergeDirectory(t *testing.T) {
	t.Run("copies new files from src to dst", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir()

		if err := os.WriteFile(filepath.Join(src, "file.md"), []byte("hello"), 0644); err != nil {
			t.Fatal(err)
		}

		if err := mergeDirectory(src, dst); err != nil {
			t.Fatalf("mergeDirectory() unexpected error: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(dst, "file.md"))
		if err != nil {
			t.Fatalf("expected file.md to exist in dst: %v", err)
		}
		if string(content) != "hello" {
			t.Fatalf("expected 'hello', got %q", string(content))
		}
	})

	t.Run("does not overwrite existing files in dst", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir()

		if err := os.WriteFile(filepath.Join(src, "workflow.md"), []byte("source version"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, "workflow.md"), []byte("trial version"), 0644); err != nil {
			t.Fatal(err)
		}

		if err := mergeDirectory(src, dst); err != nil {
			t.Fatalf("mergeDirectory() unexpected error: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(dst, "workflow.md"))
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != "trial version" {
			t.Fatalf("expected dst file to be preserved, got %q", string(content))
		}
	})

	t.Run("creates nested directories and copies files", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir()

		skillDir := filepath.Join(src, "skills", "my-skill")
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("skill content"), 0644); err != nil {
			t.Fatal(err)
		}

		if err := mergeDirectory(src, dst); err != nil {
			t.Fatalf("mergeDirectory() unexpected error: %v", err)
		}

		dstSkillFile := filepath.Join(dst, "skills", "my-skill", "SKILL.md")
		content, err := os.ReadFile(dstSkillFile)
		if err != nil {
			t.Fatalf("expected SKILL.md to exist in dst: %v", err)
		}
		if string(content) != "skill content" {
			t.Fatalf("expected 'skill content', got %q", string(content))
		}
	})

	t.Run("preserves existing workflow while copying new skill files", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir()

		// Source has both a workflow file and a skill file
		workflowDir := filepath.Join(src, "workflows")
		if err := os.MkdirAll(workflowDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workflowDir, "example.md"), []byte("source workflow"), 0644); err != nil {
			t.Fatal(err)
		}
		skillDir := filepath.Join(src, "skills", "example")
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("skill content"), 0644); err != nil {
			t.Fatal(err)
		}

		// Destination already has a trial-modified version of the workflow
		dstWorkflowDir := filepath.Join(dst, "workflows")
		if err := os.MkdirAll(dstWorkflowDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dstWorkflowDir, "example.md"), []byte("trial workflow with source field"), 0644); err != nil {
			t.Fatal(err)
		}

		if err := mergeDirectory(src, dst); err != nil {
			t.Fatalf("mergeDirectory() unexpected error: %v", err)
		}

		// Workflow must not be overwritten
		wfContent, err := os.ReadFile(filepath.Join(dst, "workflows", "example.md"))
		if err != nil {
			t.Fatal(err)
		}
		if string(wfContent) != "trial workflow with source field" {
			t.Fatalf("trial workflow was overwritten; got %q", string(wfContent))
		}

		// Skill file must be copied
		skillContent, err := os.ReadFile(filepath.Join(dst, "skills", "example", "SKILL.md"))
		if err != nil {
			t.Fatalf("expected SKILL.md to be copied: %v", err)
		}
		if string(skillContent) != "skill content" {
			t.Fatalf("expected 'skill content', got %q", string(skillContent))
		}
	})

	t.Run("returns nil for empty src directory", func(t *testing.T) {
		src := t.TempDir()
		dst := t.TempDir()

		if err := mergeDirectory(src, dst); err != nil {
			t.Fatalf("mergeDirectory() unexpected error for empty src: %v", err)
		}
	})
}
