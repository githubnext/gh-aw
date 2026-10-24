//go:build !integration

package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestResolveRepoOverrideForRunKeepsUnknownHostUnset(t *testing.T) {
	t.Setenv("GH_HOST", "ghe.example.com")

	repo, hostname := resolveRepoOverrideForRun(WorkflowRun{}, "octo", "repo", "")

	assert.Equal(t, "octo/repo", repo)
	assert.Empty(t, hostname)
}

func TestResolveRunStateBranchRefPreservesExplicitHost(t *testing.T) {
	for _, tt := range []struct {
		name     string
		hostname string
	}{
		{name: "ambient host"},
		{name: "public host", hostname: "github.com"},
		{name: "enterprise host", hostname: "ghe.example.com"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fakeBinDir := t.TempDir()
			fakeGH := filepath.Join(fakeBinDir, "gh")
			argsFile := filepath.Join(fakeBinDir, "args")
			script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\nprintf '%%s\\n' '[{\"sha\":\"abc123\",\"commit\":{\"message\":\"Update evals results from workflow run 123\"}}]'\n", argsFile)
			require.NoError(t, os.WriteFile(fakeGH, []byte(script), 0o755))
			t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("GH_HOST", "ghe.example.com")

			ref, err := resolveRunStateBranchRef(context.Background(), "octo/repo", "evals/workflow", 123, tt.hostname, "evals results")

			require.NoError(t, err)
			assert.Equal(t, "abc123", ref)
			args, err := os.ReadFile(argsFile)
			require.NoError(t, err)
			expected := []string{"api", "repos/octo/repo/commits?sha=evals%2Fworkflow&per_page=100&page=1", "--method", "GET"}
			if tt.hostname != "" {
				expected = append(expected, "--hostname", tt.hostname)
			}
			assert.Equal(t, expected, strings.Fields(string(args)))
		})
	}
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
