//go:build !integration

package cli

import (
	"context"
	"fmt"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeRedirectToSourceSpec(t *testing.T) {
	t.Run("parses workflow spec", func(t *testing.T) {
		spec, err := normalizeRedirectToSourceSpec("owner/repo/workflows/new.md@main")
		require.NoError(t, err, "redirect spec should parse")
		assert.Equal(t, "owner/repo", spec.Repo, "repo should be parsed from redirect spec")
		assert.Equal(t, "workflows/new.md", spec.Path, "path should be parsed from redirect spec")
		assert.Equal(t, "main", spec.Ref, "ref should be parsed from redirect spec")
	})

	t.Run("parses GitHub URL", func(t *testing.T) {
		spec, err := normalizeRedirectToSourceSpec("https://github.com/owner/repo/blob/main/workflows/new.md")
		require.NoError(t, err, "redirect URL should parse")
		assert.Equal(t, "owner/repo", spec.Repo, "repo should be parsed from redirect URL")
		assert.Equal(t, "workflows/new.md", spec.Path, "path should be parsed from redirect URL")
		assert.Equal(t, "main", spec.Ref, "ref should be parsed from redirect URL")
	})
}

func TestResolveRedirectedUpdateLocation(t *testing.T) {
	originalResolveLatestRef := resolveLatestRefFn
	originalDownloadWorkflow := downloadWorkflowContentFn
	t.Cleanup(func() {
		resolveLatestRefFn = originalResolveLatestRef
		downloadWorkflowContentFn = originalDownloadWorkflow
	})

	t.Run("follows redirect chain", func(t *testing.T) {
		resolveLatestRefFn = func(_ context.Context, _ string, currentRef string, _ bool, _ bool) (string, error) {
			return currentRef, nil
		}
		downloadWorkflowContentFn = func(_ context.Context, repo, path, ref string, _ bool) ([]byte, error) {
			key := fmt.Sprintf("%s/%s@%s", repo, path, ref)
			switch key {
			case "owner/repo/workflows/original.md@main":
				return []byte("---\nredirect: owner/repo/workflows/new.md@main\n---\n"), nil
			case "owner/repo/workflows/new.md@main":
				return []byte("---\nname: New Workflow\n---\n"), nil
			default:
				return nil, fmt.Errorf("unexpected download key %s", key)
			}
		}

		var result *resolvedUpdateLocation
		output := testutil.CaptureStderr(t, func() {
			var err error
			result, err = resolveRedirectedUpdateLocation(
				context.Background(),
				"test-workflow",
				&SourceSpec{Repo: "owner/repo", Path: "workflows/original.md", Ref: "main"},
				false,
				false,
				false,
			)
			require.NoError(t, err, "redirect chain should resolve")
		})
		assert.Equal(t, "owner/repo", result.sourceSpec.Repo, "final redirect repo should be tracked")
		assert.Equal(t, "workflows/new.md", result.sourceSpec.Path, "final redirect path should be tracked")
		assert.Equal(t, "main", result.sourceFieldRef, "branch ref should be preserved in source field")
		assert.Len(t, result.redirectHistory, 1, "redirect history should include the followed hop")
		assert.Contains(t, output, "Workflow test-workflow redirect: owner/repo/workflows/original.md@main → owner/repo/workflows/new.md@main", "redirect warning should explain source movement")
	})

	t.Run("detects redirect loops", func(t *testing.T) {
		resolveLatestRefFn = func(_ context.Context, _ string, currentRef string, _ bool, _ bool) (string, error) {
			return currentRef, nil
		}
		downloadWorkflowContentFn = func(_ context.Context, repo, path, ref string, _ bool) ([]byte, error) {
			key := fmt.Sprintf("%s/%s@%s", repo, path, ref)
			switch key {
			case "owner/repo/workflows/a.md@main":
				return []byte("---\nredirect: owner/repo/workflows/b.md@main\n---\n"), nil
			case "owner/repo/workflows/b.md@main":
				return []byte("---\nredirect: owner/repo/workflows/a.md@main\n---\n"), nil
			default:
				return nil, fmt.Errorf("unexpected download key %s", key)
			}
		}

		_, err := resolveRedirectedUpdateLocation(
			context.Background(),
			"loop-workflow",
			&SourceSpec{Repo: "owner/repo", Path: "workflows/a.md", Ref: "main"},
			false,
			false,
			false,
		)
		require.Error(t, err, "redirect loop should return an error")
		assert.Contains(t, err.Error(), "redirect loop detected", "error should explain redirect loop")
	})

	t.Run("refuses redirect when no-redirect is enabled", func(t *testing.T) {
		resolveLatestRefFn = func(_ context.Context, _ string, currentRef string, _ bool, _ bool) (string, error) {
			return currentRef, nil
		}
		downloadWorkflowContentFn = func(_ context.Context, repo, path, ref string, _ bool) ([]byte, error) {
			key := fmt.Sprintf("%s/%s@%s", repo, path, ref)
			switch key {
			case "owner/repo/workflows/original.md@main":
				return []byte("---\nredirect: owner/repo/workflows/new.md@main\n---\n"), nil
			default:
				return nil, fmt.Errorf("unexpected download key %s", key)
			}
		}

		_, err := resolveRedirectedUpdateLocation(
			context.Background(),
			"no-redirect-workflow",
			&SourceSpec{Repo: "owner/repo", Path: "workflows/original.md", Ref: "main"},
			false,
			false,
			true,
		)
		require.Error(t, err, "redirect should be refused with --no-redirect")
		assert.Contains(t, err.Error(), "redirect is disabled by --no-redirect", "error should explain redirect refusal")
	})
}
