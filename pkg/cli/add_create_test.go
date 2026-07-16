//go:build !integration

package cli

import (
	"context"
	"os"
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
		initRepo: func(InitOptions) error {
			initCalls++
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
}

func TestPrepareAddTargetCheckoutWithRuntime_RejectsInvalidRepo(t *testing.T) {
	_, err := prepareAddTargetCheckoutWithRuntime(context.Background(), addCreateOptions{
		Repo: "not-a-slug",
	}, bootstrapRuntime{}, t.TempDir())
	if err == nil {
		t.Fatal("expected invalid --create error")
	}
	if err.Error() != "--create must use the OWNER/REPO format. Example: --create github/gh-aw" {
		t.Fatalf("unexpected error: %v", err)
	}
}
