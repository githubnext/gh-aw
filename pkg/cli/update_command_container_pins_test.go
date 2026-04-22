//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowFilesForExistingLocks(t *testing.T) {
	t.Run("returns only workflow markdown files with matching lock files", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		require.NoError(t, os.MkdirAll(workflowsDir, 0o755))

		require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "a.md"), []byte("---\non: issues\n---"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "a.lock.yml"), []byte("name: a"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "b.md"), []byte("---\non: issues\n---"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "b.lock.yml"), []byte("name: b"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "c.md"), []byte("---\non: issues\n---"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "notes.md"), []byte("# include"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "notes.lock.yml"), []byte("name: notes"), 0o644))

		files, err := workflowFilesForExistingLocks(workflowsDir)
		require.NoError(t, err, "workflow file discovery should succeed")
		assert.Equal(t, []string{
			filepath.Join(workflowsDir, "a.md"),
			filepath.Join(workflowsDir, "b.md"),
			filepath.Join(workflowsDir, "notes.md"),
		}, files, "only markdown files with lock counterparts should be returned in sorted order")
	})

	t.Run("returns empty slice when no matching pairs exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		require.NoError(t, os.MkdirAll(workflowsDir, 0o755))

		require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "only-md.md"), []byte("---\non: issues\n---"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "only-lock.lock.yml"), []byte("name: only-lock"), 0o644))

		files, err := workflowFilesForExistingLocks(workflowsDir)
		require.NoError(t, err, "workflow file discovery should succeed")
		assert.Empty(t, files, "no matching markdown/lock pairs should produce an empty result")
	})
}
