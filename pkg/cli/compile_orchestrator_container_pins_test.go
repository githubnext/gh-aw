//go:build !integration

package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/require"
)

func TestMaybeForceRefreshContainerPins_PropagatesRefreshError(t *testing.T) {
	original := compileUpdateContainerPins
	t.Cleanup(func() { compileUpdateContainerPins = original })

	compileUpdateContainerPins = func(_ context.Context, _ containerPinUpdateDeps, _ string, _ bool, _ containerPinUpdateOptions) (bool, error) {
		return false, errors.New("refresh failed")
	}

	err := maybeForceRefreshContainerPins(context.Background(), CompileConfig{
		ForceRefreshContainerPins: true,
	}, ".github/workflows")
	if err == nil {
		t.Fatal("expected error when refresh fails")
	}
	if !strings.Contains(err.Error(), "failed to refresh container pins") {
		t.Fatalf("expected wrapped refresh error, got: %v", err)
	}
}

func TestMaybeForceRefreshContainerPins_SkipsWhenNoEmit(t *testing.T) {
	original := compileUpdateContainerPins
	t.Cleanup(func() { compileUpdateContainerPins = original })

	called := false
	compileUpdateContainerPins = func(_ context.Context, _ containerPinUpdateDeps, _ string, _ bool, _ containerPinUpdateOptions) (bool, error) {
		called = true
		return false, nil
	}

	err := maybeForceRefreshContainerPins(context.Background(), CompileConfig{
		ForceRefreshContainerPins: true,
		NoEmit:                    true,
	}, ".github/workflows")
	if err != nil {
		t.Fatalf("expected no error when --no-emit skips refresh, got: %v", err)
	}
	if called {
		t.Fatal("expected refresh to be skipped when --no-emit is set")
	}
}

func TestMaybeForceRefreshContainerPins_UsesRefreshExistingMode(t *testing.T) {
	original := compileUpdateContainerPins
	t.Cleanup(func() { compileUpdateContainerPins = original })

	called := false
	compileUpdateContainerPins = func(_ context.Context, _ containerPinUpdateDeps, _ string, _ bool, opts containerPinUpdateOptions) (bool, error) {
		called = true
		if !opts.refreshExisting {
			t.Fatal("expected refreshExisting=true when forcing container pin refresh")
		}

		return false, nil
	}

	err := maybeForceRefreshContainerPins(context.Background(), CompileConfig{
		ForceRefreshContainerPins: true,
	}, ".github/workflows")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !called {
		t.Fatal("expected refresh function to be called")
	}
}

func TestForceRefreshContainerPins_KreuzbergScanUsesRefreshedDigest(t *testing.T) {
	sharedSource, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "shared", "mcp", "kreuzberg.md"))
	require.NoError(t, err)

	tmpDir := t.TempDir()
	t.Chdir(tmpDir)
	workflowsDir := filepath.Join(".github", "workflows")
	sharedDir := filepath.Join(workflowsDir, "shared", "mcp")
	require.NoError(t, os.MkdirAll(sharedDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sharedDir, "kreuzberg.md"), sharedSource, 0644))
	markdownPath := filepath.Join(workflowsDir, "test.md")
	require.NoError(t, os.WriteFile(markdownPath, []byte(`---
on: workflow_dispatch
engine: copilot
network:
  allowed:
    - defaults
imports:
  - shared/mcp/kreuzberg.md
---
Extract a document.
`), 0644))

	const image = "ghcr.io/xberg-io/xberg:1.3.6"
	oldDigest := "sha256:" + strings.Repeat("1", 64)
	newDigest := "sha256:" + strings.Repeat("2", 64)
	cache := workflow.NewActionCache(".")
	cache.SetContainerPin(image, oldDigest, image+"@"+oldDigest)
	require.NoError(t, cache.Save())
	compile := func() {
		t.Helper()
		compiler := createAndConfigureCompiler(CompileConfig{})
		require.NoError(t, compiler.CompileWorkflow(markdownPath))
	}
	compile()

	original := compileUpdateContainerPins
	t.Cleanup(func() { compileUpdateContainerPins = original })
	compileUpdateContainerPins = func(ctx context.Context, deps containerPinUpdateDeps, dir string, verbose bool, opts containerPinUpdateOptions) (bool, error) {
		deps.fetchDigest = func(_ context.Context, ref string, _ bool) (string, error) {
			if ref == image {
				return newDigest, nil
			}
			return oldDigest, nil
		}
		return updateContainerPins(ctx, deps, dir, verbose, opts)
	}
	require.NoError(t, maybeForceRefreshContainerPins(context.Background(), CompileConfig{
		ForceRefreshContainerPins: true,
	}, workflowsDir))
	compile()

	lockPath := filepath.Join(workflowsDir, "test.lock.yml")
	scanned := collectContainerImagesFromLockFiles([]string{lockPath})
	require.Contains(t, scanned, workflow.GHAWManifestContainer{
		Image: image, Digest: newDigest, PinnedImage: image + "@" + newDigest,
	})
	lock, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	require.NotContains(t, string(lock), image+"@"+oldDigest)
	require.Contains(t, string(lock), image+"@"+newDigest)
}
