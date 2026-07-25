//go:build !integration

package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateUpdateSHAEntries_NoActionsLock(t *testing.T) {
	tmpDir := testutil.TempDir(t, "validate-update-sha-*")
	require.NoError(t, validateUpdateSHAEntries(context.Background(), tmpDir))
}

func TestValidateUpdateSHAEntries_ValidEntries(t *testing.T) {
	tmpDir := testutil.TempDir(t, "validate-update-sha-*")
	cache := workflow.NewActionCache(tmpDir)
	cache.Set("actions/checkout", "v5", "93cb6efe18208431cddfb8368fd83d5badbf9bfd")
	digest := "sha256:" + strings.Repeat("a", 64)
	image := "ghcr.io/github/gh-aw-firewall/agent:0.27.9"
	cache.SetContainerPin(image, digest, image+"@"+digest)
	require.NoError(t, cache.Save())

	origResolveAction := resolveActionRefSHAForValidation
	origResolveContainer := resolveContainerDigestForValidation
	resolveActionRefSHAForValidation = func(_ context.Context, _, ref string) (string, error) {
		if ref == "v5" {
			return "93cb6efe18208431cddfb8368fd83d5badbf9bfd", nil
		}
		return ref, nil
	}
	resolveContainerDigestForValidation = func(_ context.Context, _ string) (string, error) {
		return digest, nil
	}
	t.Cleanup(func() {
		resolveActionRefSHAForValidation = origResolveAction
		resolveContainerDigestForValidation = origResolveContainer
	})

	require.NoError(t, validateUpdateSHAEntries(context.Background(), tmpDir))
}

func TestValidateUpdateSHAEntries_InvalidEntries(t *testing.T) {
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

	origResolveAction := resolveActionRefSHAForValidation
	origResolveContainer := resolveContainerDigestForValidation
	resolveActionRefSHAForValidation = func(_ context.Context, repo, ref string) (string, error) {
		switch {
		case repo == "actions/setup-node" && ref == "v7":
			return "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", nil
		case ref == "395ad3262231945c25e8478fd5baf05154b1d79f":
			return "", errors.New("commit not found")
		default:
			return ref, nil
		}
	}
	resolveContainerDigestForValidation = func(_ context.Context, _ string) (string, error) {
		return "sha256:" + strings.Repeat("b", 64), nil
	}
	t.Cleanup(func() {
		resolveActionRefSHAForValidation = origResolveAction
		resolveContainerDigestForValidation = origResolveContainer
	})

	err := validateUpdateSHAEntries(context.Background(), tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `action entry "actions/checkout@v5" has invalid SHA`)
	assert.Contains(t, err.Error(), `action entry key/version mismatch: key "actions/setup-node@v6" should be "actions/setup-node@v7"`)
	assert.Contains(t, err.Error(), `action entry "actions/setup-node@v6" has unresolved commit SHA`)
	assert.Contains(t, err.Error(), `action entry "actions/setup-node@v6" SHA/version mismatch`)
	assert.Contains(t, err.Error(), `container pin key/image mismatch`)
	assert.Contains(t, err.Error(), `container pin "ghcr.io/test/image:v1" has invalid digest`)
	assert.Contains(t, err.Error(), `container pin "ghcr.io/test/image:v1" has inconsistent pinned_image`)
}
