//go:build !integration

package cli

import (
	"context"
	"path/filepath"
	"testing"
)

func TestPrepareAddTargetCheckoutWithRuntime_CreatesAndClones(t *testing.T) {
	tempDir := t.TempDir()
	var createdRepo string
	var clonedRepo string
	var clonedDir string

	checkoutDir, err := prepareAddTargetCheckoutWithRuntime(context.Background(), addCreateOptions{
		Repo:             "octo/platform-ops",
		Visibility:       "internal",
		RequireOwnerType: "org",
	}, bootstrapRuntime{
		setupRepositoryRuntime: setupRepositoryRuntime{
			checkAuth:          func(context.Context) error { return nil },
			repoExists:         func(context.Context, string) (bool, error) { return false, nil },
			ownerType:          func(context.Context, string) (string, error) { return "Organization", nil },
			createRepo:         func(_ context.Context, repo string, visibility string) error { createdRepo = repo + ":" + visibility; return nil },
			cloneRepo:          func(_ context.Context, repo string, dir string) error { clonedRepo = repo; clonedDir = dir; return nil },
			checkCleanWorktree: func(bool) error { return nil },
		},
	}, tempDir)
	if err != nil {
		t.Fatalf("prepareAddTargetCheckoutWithRuntime returned error: %v", err)
	}

	expectedDir := filepath.Join(tempDir, "platform-ops")
	if checkoutDir != expectedDir {
		t.Fatalf("expected checkout dir %q, got %q", expectedDir, checkoutDir)
	}
	if createdRepo != "octo/platform-ops:internal" {
		t.Fatalf("expected repo creation call, got %q", createdRepo)
	}
	if clonedRepo != "octo/platform-ops" {
		t.Fatalf("expected clone repo octo/platform-ops, got %q", clonedRepo)
	}
	if clonedDir != expectedDir {
		t.Fatalf("expected clone dir %q, got %q", expectedDir, clonedDir)
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
