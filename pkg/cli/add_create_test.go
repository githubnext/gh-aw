//go:build !integration

package cli

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

func TestPrepareAddTargetCheckoutWithRuntime_CreatesAndClones(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	var createdRepo string
	var clonedRepo string
	var clonedDir string
	initCalls := 0
	var initOpts InitOptions

	checkoutDir, err := prepareAddTargetCheckoutWithRuntime(context.Background(), addCreateOptions{
		Repo:             "octo/platform-ops",
		Visibility:       "internal",
		License:          "mit",
		RequireOwnerType: "org",
	}, bootstrapRuntime{
		setupRepositoryRuntime: setupRepositoryRuntime{
			checkAuth:  func(context.Context) error { return nil },
			repoExists: func(context.Context, string) (bool, error) { return false, nil },
			ownerType:  func(context.Context, string) (string, error) { return "Organization", nil },
			createRepo: func(_ context.Context, repo string, opts setupRepositoryCreateOptions) error {
				createdRepo = repo + ":" + opts.Visibility + ":" + opts.License
				return nil
			},
			cloneRepo: func(_ context.Context, repo string, dir string) error {
				clonedRepo = repo
				clonedDir = dir
				return os.MkdirAll(dir, 0o755)
			},
			checkCleanWorktree: func(bool) error { return nil },
		},
		initRepo: func(got InitOptions) error {
			initCalls++
			initOpts = got
			return nil
		},
	}, tempDir)
	if err != nil {
		t.Fatalf("prepareAddTargetCheckoutWithRuntime returned error: %v", err)
	}

	expectedDir := "platform-ops"
	if checkoutDir != expectedDir {
		t.Fatalf("expected checkout dir %q, got %q", expectedDir, checkoutDir)
	}
	if createdRepo != "octo/platform-ops:internal:mit" {
		t.Fatalf("expected repo creation call, got %q", createdRepo)
	}
	if clonedRepo != "octo/platform-ops" {
		t.Fatalf("expected clone repo octo/platform-ops, got %q", clonedRepo)
	}
	if clonedDir != expectedDir {
		t.Fatalf("expected clone dir %q, got %q", expectedDir, clonedDir)
	}
	if initCalls != 1 {
		t.Fatalf("expected init to run once, got %d", initCalls)
	}
	if initOpts.CodespaceEnabled {
		t.Fatal("expected add create init to leave codespaces disabled (matches bootstrap default)")
	}
	if len(initOpts.CodespaceRepos) != 0 {
		t.Fatalf("expected add create init to configure current repo only, got %v", initOpts.CodespaceRepos)
	}
}

func TestPrepareAddTargetCheckoutWithRuntime_SkipsInitWhenRequested(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	initCalls := 0

	_, err = prepareAddTargetCheckoutWithRuntime(context.Background(), addCreateOptions{
		Repo:       "octo/platform-ops",
		Visibility: "private",
		SkipInit:   true,
	}, bootstrapRuntime{
		setupRepositoryRuntime: setupRepositoryRuntime{
			checkAuth:  func(context.Context) error { return nil },
			repoExists: func(context.Context, string) (bool, error) { return false, nil },
			ownerType:  func(context.Context, string) (string, error) { return "User", nil },
			createRepo: func(_ context.Context, _ string, _ setupRepositoryCreateOptions) error { return nil },
			cloneRepo: func(_ context.Context, _ string, dir string) error {
				return os.MkdirAll(dir, 0o755)
			},
			checkCleanWorktree: func(bool) error { return nil },
		},
		initRepo: func(InitOptions) error {
			initCalls++
			return nil
		},
	}, tempDir)
	if err != nil {
		t.Fatalf("prepareAddTargetCheckoutWithRuntime returned error: %v", err)
	}
	if initCalls != 0 {
		t.Fatalf("expected init to be skipped (SkipInit=true), but it ran %d time(s)", initCalls)
	}
}

func TestPrepareAddTargetCheckoutWithRuntime_RejectsInvalidRepo(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})

	// Validation happens before any runtime methods are called, so we can pass
	// a minimal runtime. The normalizeAddCreateOptions will not add an owner
	// because there's no git remote in the test temp dir.
	_, err = prepareAddTargetCheckoutWithRuntime(context.Background(), addCreateOptions{
		Repo: "not-a-slug",
	}, bootstrapRuntime{
		setupRepositoryRuntime: setupRepositoryRuntime{
			checkAuth:  func(context.Context) error { return nil },
			repoExists: func(context.Context, string) (bool, error) { return false, nil },
			ownerType:  func(context.Context, string) (string, error) { return "Organization", nil },
			createRepo: func(_ context.Context, _ string, _ setupRepositoryCreateOptions) error { return nil },
			cloneRepo:  func(_ context.Context, _ string, _ string) error { return nil },
		},
		initRepo: func(InitOptions) error { return nil },
	}, tempDir)
	if err == nil {
		t.Fatal("expected invalid --create error")
	}
	expectedErr := "invalid repository format \"not-a-slug\" after normalization. Expected OWNER/REPO format"
	if err.Error() != expectedErr {
		t.Fatalf("unexpected error: got %q, want %q", err.Error(), expectedErr)
	}
}

func TestNormalizeAddCreateOptions_InfersOwner(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})

	// Initialize a git repo with a remote using actual git commands
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}
	if err := exec.Command("git", "config", "user.name", "Test User").Run(); err != nil {
		t.Skip("Git not available")
	}
	if err := exec.Command("git", "config", "user.email", "test@example.com").Run(); err != nil {
		t.Skip("Git not available")
	}
	if err := exec.Command("git", "remote", "add", "origin", "https://github.com/test-org/test-repo.git").Run(); err != nil {
		t.Skip("Git not available")
	}

	opts := normalizeAddCreateOptions(addCreateOptions{
		Repo: "new-repo",
	})

	if opts.Repo != "test-org/new-repo" {
		t.Fatalf("expected repo to be test-org/new-repo, got %q", opts.Repo)
	}
}

func TestNormalizeAddCreateOptions_SkipsPromptValue(t *testing.T) {
	opts := normalizeAddCreateOptions(addCreateOptions{
		Repo: "prompt",
	})

	if opts.Repo != "prompt" {
		t.Fatalf("expected repo to remain 'prompt', got %q", opts.Repo)
	}
}

func TestNormalizeAddCreateOptions_PreservesFullSlug(t *testing.T) {
	opts := normalizeAddCreateOptions(addCreateOptions{
		Repo: "owner/repo",
	})

	if opts.Repo != "owner/repo" {
		t.Fatalf("expected repo to remain 'owner/repo', got %q", opts.Repo)
	}
}

func TestNormalizeAddCreateOptions_FailsWhenNoRemote(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})

	// Initialize a git repo WITHOUT a remote
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}
	if err := exec.Command("git", "config", "user.name", "Test User").Run(); err != nil {
		t.Skip("Git not available")
	}
	if err := exec.Command("git", "config", "user.email", "test@example.com").Run(); err != nil {
		t.Skip("Git not available")
	}

	// Try to normalize without a remote - should leave repo unchanged
	opts := normalizeAddCreateOptions(addCreateOptions{
		Repo: "new-repo",
	})

	// Without a remote, owner inference should fail and repo should remain unchanged
	if opts.Repo != "new-repo" {
		t.Fatalf("expected repo to remain 'new-repo' when no remote exists, got %q", opts.Repo)
	}
}
