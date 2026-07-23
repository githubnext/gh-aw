package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildYamllintContainerPaths(t *testing.T) {
	t.Run("normalizes in-repo path", func(t *testing.T) {
		gitRoot := t.TempDir()
		lockFile := filepath.Join(gitRoot, ".github", "workflows", "static-analysis-report.lock.yml")

		paths, err := buildYamllintContainerPaths(gitRoot, []string{lockFile})

		require.NoError(t, err)
		assert.Equal(t, []string{"./.github/workflows/static-analysis-report.lock.yml"}, paths)
	})

	t.Run("rejects path outside repository", func(t *testing.T) {
		gitRoot := t.TempDir()
		lockFile := filepath.Join(filepath.Dir(gitRoot), "outside.lock.yml")

		_, err := buildYamllintContainerPaths(gitRoot, []string{lockFile})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "outside repository root")
	})

	t.Run("prefixes option-looking root file", func(t *testing.T) {
		gitRoot := t.TempDir()
		lockFile := filepath.Join(gitRoot, "-workflow.lock.yml")

		paths, err := buildYamllintContainerPaths(gitRoot, []string{lockFile})

		require.NoError(t, err)
		assert.Equal(t, []string{"./-workflow.lock.yml"}, paths)
	})
}
