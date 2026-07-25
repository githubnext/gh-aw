//go:build !integration

package cli

import (
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
	require.NoError(t, validateUpdateSHAEntries(tmpDir))
}

func TestValidateUpdateSHAEntries_ValidEntries(t *testing.T) {
	tmpDir := testutil.TempDir(t, "validate-update-sha-*")
	cache := workflow.NewActionCache(tmpDir)
	cache.Set("actions/checkout", "v5", "93cb6efe18208431cddfb8368fd83d5badbf9bfd")
	digest := "sha256:" + strings.Repeat("a", 64)
	image := "ghcr.io/github/gh-aw-firewall/agent:0.27.9"
	cache.SetContainerPin(image, digest, image+"@"+digest)
	require.NoError(t, cache.Save())

	require.NoError(t, validateUpdateSHAEntries(tmpDir))
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

	err := validateUpdateSHAEntries(tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `action entry "actions/checkout@v5" has invalid SHA`)
	assert.Contains(t, err.Error(), `action entry key/version mismatch: key "actions/setup-node@v6" should be "actions/setup-node@v7"`)
	assert.Contains(t, err.Error(), `container pin key/image mismatch`)
	assert.Contains(t, err.Error(), `container pin "ghcr.io/test/image:v1" has invalid digest`)
	assert.Contains(t, err.Error(), `container pin "ghcr.io/test/image:v1" has inconsistent pinned_image`)
}
