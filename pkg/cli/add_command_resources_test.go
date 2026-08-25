//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddAgentFileWithTracking_WritesAgentFile(t *testing.T) {
	t.Parallel()
	gitRoot := testutil.TempDir(t, "test-add-agent-file-*")
	resolved := &ResolvedWorkflow{
		Spec: &WorkflowSpec{
			WorkflowPath: "agents/reviewer.md",
		},
		Content: []byte("agent instructions"),
	}

	err := addAgentFileWithTracking(resolved, nil, AddOptions{Quiet: true}, gitRoot)
	require.NoError(t, err)

	destFile := filepath.Join(gitRoot, workflow.GetEngineSubAgentDir(""), "reviewer.md")
	content, err := os.ReadFile(destFile)
	require.NoError(t, err)
	assert.Equal(t, "agent instructions", string(content))
}

func TestAddAgentFileWithTracking_SkipsExistingWithoutForce(t *testing.T) {
	t.Parallel()
	gitRoot := testutil.TempDir(t, "test-add-agent-skip-*")
	destFile := filepath.Join(gitRoot, workflow.GetEngineSubAgentDir(""), "reviewer.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(destFile), 0o755))
	require.NoError(t, os.WriteFile(destFile, []byte("existing"), 0o644))
	resolved := &ResolvedWorkflow{
		Spec: &WorkflowSpec{
			WorkflowPath: "agents/reviewer.md",
		},
		Content: []byte("new"),
	}

	err := addAgentFileWithTracking(resolved, nil, AddOptions{Quiet: true}, gitRoot)
	require.NoError(t, err)

	content, err := os.ReadFile(destFile)
	require.NoError(t, err)
	assert.Equal(t, "existing", string(content))
}
