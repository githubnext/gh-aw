//go:build !integration

package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBootstrapCommitAndPushAction_CommitsAndPushesChanges(t *testing.T) {
	repoDir := initBootstrapGitRepo(t)
	remoteDir := t.TempDir()

	runRepoGit := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
		}
		return string(output)
	}

	cmd := exec.Command("git", "init", "--bare", remoteDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare failed: %v\n%s", err, string(output))
	}

	runRepoGit("config", "user.name", "Bootstrap Test")
	runRepoGit("config", "user.email", "bootstrap@example.com")
	runRepoGit("remote", "add", "origin", remoteDir)
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatalf("write initial file: %v", err)
	}
	runRepoGit("add", "README.md")
	runRepoGit("commit", "-m", "initial")
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("updated\n"), 0o644); err != nil {
		t.Fatalf("write updated file: %v", err)
	}

	err := runBootstrapCommitAndPushAction(context.Background(), repoDir, repositoryPackageBootstrapAction{
		Type:    "commit-and-push",
		Message: "Bootstrap repository changes",
	})
	if err != nil {
		t.Fatalf("runBootstrapCommitAndPushAction returned error: %v", err)
	}

	if got := strings.TrimSpace(runRepoGit("log", "--format=%s", "-1")); got != "Bootstrap repository changes" {
		t.Fatalf("unexpected commit message: %q", got)
	}
	if got := strings.TrimSpace(runRepoGit("status", "--porcelain")); got != "" {
		t.Fatalf("expected clean worktree after commit-and-push, got %q", got)
	}

	branch, err := getCurrentBranchIn(repoDir)
	if err != nil {
		t.Fatalf("getCurrentBranchIn returned error: %v", err)
	}
	localHead := strings.TrimSpace(runRepoGit("rev-parse", "HEAD"))
	remoteHeadCmd := exec.Command("git", "--git-dir", remoteDir, "rev-parse", "refs/heads/"+branch)
	remoteHeadOutput, err := remoteHeadCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to read remote branch head: %v\n%s", err, string(remoteHeadOutput))
	}
	if got := strings.TrimSpace(string(remoteHeadOutput)); got != localHead {
		t.Fatalf("expected remote head %q to match local head %q", got, localHead)
	}
}

func TestRunBootstrapCommitAndPushAction_RequiresRepoDir(t *testing.T) {
	if err := runBootstrapCommitAndPushAction(context.Background(), "", repositoryPackageBootstrapAction{
		Type:    "commit-and-push",
		Message: "Bootstrap repository changes",
	}); err == nil {
		t.Fatal("expected missing repoDir error")
	}
}

func TestRunBootstrapCommitAndPushAction_SkipsCleanCheckout(t *testing.T) {
	repoDir := initBootstrapGitRepo(t)
	runRepoGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
		}
	}

	runRepoGit("config", "user.name", "Bootstrap Test")
	runRepoGit("config", "user.email", "bootstrap@example.com")
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("initial\n"), 0o644); err != nil {
		t.Fatalf("write initial file: %v", err)
	}
	runRepoGit("add", "README.md")
	runRepoGit("commit", "-m", "initial")

	if err := runBootstrapCommitAndPushAction(context.Background(), repoDir, repositoryPackageBootstrapAction{
		Type:    "commit-and-push",
		Message: "Bootstrap repository changes",
	}); err != nil {
		t.Fatalf("runBootstrapCommitAndPushAction returned error: %v", err)
	}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git status failed: %v\n%s", err, string(output))
	}
	if strings.TrimSpace(string(output)) != "" {
		t.Fatalf("expected clean checkout, got %q", string(output))
	}
}
