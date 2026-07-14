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

func TestWorkflowIDFromRunPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "empty", path: "", want: ""},
		{name: "markdown workflow", path: ".github/workflows/release.md", want: "release"},
		{name: "lock workflow", path: ".github/workflows/release.lock.yml", want: "release"},
		{name: "yaml workflow", path: ".github/workflows/release.yml", want: "release"},
		{name: "yml workflow", path: ".github/workflows/release.yaml", want: "release"},
		{name: "uppercase and hyphen normalized", path: ".github/workflows/My-Release.yaml", want: "myrelease"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, workflowIDFromRunPath(tt.path))
		})
	}
}
