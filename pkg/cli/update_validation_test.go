//go:build !integration

package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noopResolvers returns resolvers that perform no network calls.
// Tests that want specific resolver behaviour should override individual fields.
func noopResolvers() validationResolvers {
	return validationResolvers{
		verifyActionCommitExists: func(_ context.Context, _, _ string) error {
			return nil
		},
		resolveActionVersionToSHA: func(_ context.Context, _, ref string) (string, error) {
			return ref, nil
		},
		resolveContainerDigest: func(_ context.Context, image string) (string, error) {
			if i := strings.Index(image, "@sha256:"); i >= 0 {
				return image[i+1:], nil
			}
			return "sha256:" + strings.Repeat("a", 64), nil
		},
	}
}

func TestValidateUpdateSHAEntries_NoActionsLock(t *testing.T) {
	t.Parallel()
	tmpDir := testutil.TempDir(t, "validate-update-sha-*")
	require.NoError(t, validateUpdateSHAEntriesWithResolvers(context.Background(), tmpDir, noopResolvers()))
}

func TestValidateUpdateSHAEntries_ValidEntries(t *testing.T) {
	t.Parallel()
	tmpDir := testutil.TempDir(t, "validate-update-sha-*")
	cache := workflow.NewActionCache(tmpDir)
	cache.Set("actions/checkout", "v5", "93cb6efe18208431cddfb8368fd83d5badbf9bfd")
	digest := "sha256:" + strings.Repeat("a", 64)
	image := "ghcr.io/github/gh-aw-firewall/agent:0.27.9"
	cache.SetContainerPin(image, digest, image+"@"+digest)
	require.NoError(t, cache.Save())

	r := validationResolvers{
		verifyActionCommitExists: func(_ context.Context, _, _ string) error {
			return nil
		},
		resolveActionVersionToSHA: func(_ context.Context, _, ref string) (string, error) {
			if ref == "v5" {
				return "93cb6efe18208431cddfb8368fd83d5badbf9bfd", nil
			}
			return ref, nil
		},
		resolveContainerDigest: func(_ context.Context, image string) (string, error) {
			if image == "ghcr.io/github/gh-aw-firewall/agent:0.27.9" {
				return digest, nil
			}
			return "", fmt.Errorf("unexpected image %q", image)
		},
	}

	require.NoError(t, validateUpdateSHAEntriesWithResolvers(context.Background(), tmpDir, r))
}

func TestValidateUpdateSHAEntries_InvalidEntries(t *testing.T) {
	t.Parallel()
	tmpDir := testutil.TempDir(t, "validate-update-sha-*")
	awDir := filepath.Join(tmpDir, ".github", "aw")
	require.NoError(t, os.MkdirAll(awDir, 0o755))

	const invalidActionsLock = `{
  "entries": {
    "actions/checkout@v5": {
      "repo": "actions/checkout",
      "version": "v5",
      "sha": "short"
    },
    "actions/setup-node@v6": {
      "repo": "actions/setup-node",
      "version": "v7",
      "sha": "395ad3262231945c25e8478fd5baf05154b1d79f"
    }
  },
  "containers": {
    "ghcr.io/test/image:v1": {
      "image": "ghcr.io/test/other:v1",
      "digest": "sha256:XYZ",
      "pinned_image": "ghcr.io/test/image:v1@sha256:bad"
    }
  }
}
`
	require.NoError(t, os.WriteFile(filepath.Join(awDir, "actions-lock.json"), []byte(invalidActionsLock), 0o644))

	r := validationResolvers{
		// Commit existence check: setup-node SHA is not found; checkout SHA passes.
		verifyActionCommitExists: func(_ context.Context, repo, sha string) error {
			if repo == "actions/setup-node" && sha == "395ad3262231945c25e8478fd5baf05154b1d79f" {
				return errors.New("commit not found")
			}
			return nil
		},
		// Version resolution: setup-node v7 resolves to a different SHA (mismatch).
		resolveActionVersionToSHA: func(_ context.Context, repo, ref string) (string, error) {
			if repo == "actions/setup-node" && ref == "v7" {
				return "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", nil
			}
			return ref, nil
		},
	}

	err := validateUpdateSHAEntriesWithResolvers(context.Background(), tmpDir, r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `action entry "actions/checkout@v5" has invalid SHA`)
	assert.Contains(t, err.Error(), `action entry key/version mismatch: key "actions/setup-node@v6" should be "actions/setup-node@v7"`)
	assert.Contains(t, err.Error(), `action entry "actions/setup-node@v6": commit SHA`)
	assert.Contains(t, err.Error(), `action entry "actions/setup-node@v6" SHA/version mismatch`)
	assert.Contains(t, err.Error(), `container pin key/image mismatch`)
	assert.Contains(t, err.Error(), `container pin "ghcr.io/test/image:v1" has invalid digest`)
	assert.Contains(t, err.Error(), `container pin "ghcr.io/test/image:v1" has inconsistent pinned_image`)
}

func TestValidateUpdateSHAEntries_NonFatalErrors(t *testing.T) {
	t.Parallel()
	tmpDir := testutil.TempDir(t, "validate-update-sha-*")
	cache := workflow.NewActionCache(tmpDir)
	cache.Set("actions/checkout", "v5", "93cb6efe18208431cddfb8368fd83d5badbf9bfd")
	digest := "sha256:" + strings.Repeat("a", 64)
	image := "ghcr.io/github/gh-aw-firewall/agent:0.27.9"
	cache.SetContainerPin(image, digest, image+"@"+digest)
	require.NoError(t, cache.Save())

	r := validationResolvers{
		// Auth error on commit existence check — should be skipped (non-fatal).
		verifyActionCommitExists: func(_ context.Context, _, _ string) error {
			return fmt.Errorf("%w: auth error", parser.ErrVerificationSkipped)
		},
		// Network error on version resolution — should be skipped (non-fatal).
		resolveActionVersionToSHA: func(_ context.Context, _, _ string) (string, error) {
			return "", errors.New("network timeout")
		},
		resolveContainerDigest: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("registry timeout")
		},
	}

	// All network failures should be non-fatal; validation should still pass.
	require.NoError(t, validateUpdateSHAEntriesWithResolvers(context.Background(), tmpDir, r))
}

func TestValidateUpdateSHAEntries_ContainerStructuralOnly(t *testing.T) {
	t.Parallel()
	tmpDir := testutil.TempDir(t, "validate-update-sha-*")
	digest := "sha256:" + strings.Repeat("a", 64)
	image := "ghcr.io/github/gh-aw-firewall/agent:latest"
	cache := workflow.NewActionCache(tmpDir)
	cache.SetContainerPin(image, digest, image+"@"+digest)
	require.NoError(t, cache.Save())

	// Structural-only mode skips the live digest lookup and validates only the on-disk shape.
	require.NoError(t, validateUpdateSHAEntriesWithResolvers(context.Background(), tmpDir, structuralOnlyResolvers()))
}

func TestValidateUpdateSHAEntries_ContainerDigestMismatch(t *testing.T) {
	t.Parallel()
	tmpDir := testutil.TempDir(t, "validate-update-sha-*")
	storedDigest := "sha256:" + strings.Repeat("a", 64)
	resolvedDigest := "sha256:" + strings.Repeat("b", 64)
	image := "ghcr.io/github/gh-aw-firewall/agent:0.27.9"
	cache := workflow.NewActionCache(tmpDir)
	cache.SetContainerPin(image, storedDigest, image+"@"+storedDigest)
	require.NoError(t, cache.Save())

	r := noopResolvers()
	r.resolveContainerDigest = func(_ context.Context, gotImage string) (string, error) {
		require.Equal(t, image, gotImage)
		return resolvedDigest, nil
	}

	err := validateUpdateSHAEntriesWithResolvers(context.Background(), tmpDir, r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `container pin "ghcr.io/github/gh-aw-firewall/agent:0.27.9" digest mismatch`)
}
