//go:build !integration

package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRunStateBranchRef(t *testing.T) {
	t.Run("finds matching commit for run ID", func(t *testing.T) {
		fakeBinDir := t.TempDir()
		fakeGH := filepath.Join(fakeBinDir, "gh")
		script := "#!/bin/sh\ncat <<'EOF'\n[{\"sha\":\"abc123\",\"commit\":{\"message\":\"Update evals results from workflow run 123\"}}]\nEOF\n"
		require.NoError(t, os.WriteFile(fakeGH, []byte(script), 0o755))
		t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		ref, err := resolveRunStateBranchRef(context.Background(), "github/gh-aw", "evals/workflow", 123, "", "evals results")
		require.NoError(t, err)
		assert.Equal(t, "abc123", ref)
	})

	t.Run("returns not found when no matching commit exists", func(t *testing.T) {
		fakeBinDir := t.TempDir()
		fakeGH := filepath.Join(fakeBinDir, "gh")
		script := "#!/bin/sh\ncat <<'EOF'\n[{\"sha\":\"abc123\",\"commit\":{\"message\":\"Update evals results from workflow run 999\"}}]\nEOF\n"
		require.NoError(t, os.WriteFile(fakeGH, []byte(script), 0o755))
		t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

		_, err := resolveRunStateBranchRef(context.Background(), "github/gh-aw", "evals/workflow", 123, "", "evals results")
		assert.ErrorIs(t, err, errRunStateCommitNotFound)
	})
}
