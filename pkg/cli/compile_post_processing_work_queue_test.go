//go:build !integration

package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/require"
)

func TestGenerateWorkQueueEventWorkflowSkipsRepoConfigWhenIssueMirroringIsUnused(t *testing.T) {
	repoRoot := t.TempDir()
	workflowDir := filepath.Join(repoRoot, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(workflowDir, "aw.json"), []byte("{"), 0o600))

	generatedPath := filepath.Join(workflowDir, "agentic-workqueue.yml")
	require.NoError(t, os.WriteFile(generatedPath, []byte("stale"), 0o600))

	err := generateWorkQueueEventWorkflowWrapper(context.Background(), []*workflow.WorkflowData{{
		Tools: map[string]any{"work-queue": true},
	}}, workflowDir, repoRoot, true, false)

	require.NoError(t, err)
	_, err = os.Stat(generatedPath)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestRepoConfigWithNoLogo(t *testing.T) {
	for _, logo := range []*bool{nil, new(true), new(false)} {
		for _, noLogo := range []bool{false, true} {
			config := &workflow.RepoConfig{LockFileLogo: logo, AutoUpgrade: new(true)}
			result := repoConfigWithNoLogo(config, noLogo)
			require.Equal(t, !noLogo && config.IsLockFileLogoEnabled(), result.IsLockFileLogoEnabled())
			require.Equal(t, logo, config.LockFileLogo, "the original config must not be mutated")
			require.True(t, result.IsAutoUpgradeEnabled(), "other settings must be preserved")
		}
	}
	require.Nil(t, repoConfigWithNoLogo(nil, false))
	require.False(t, repoConfigWithNoLogo(nil, true).IsLockFileLogoEnabled())
}
