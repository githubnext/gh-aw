package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAgenticWorkflowEnginesByName(t *testing.T) {
	workflowsDir := t.TempDir()
	t.Setenv("GH_AW_WORKFLOWS_DIR", workflowsDir)

	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "one.lock.yml"), []byte("name: Workflow One\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "one.md"), []byte("---\nengine: claude\n---\n# Workflow One\n"), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "two.lock.yml"), []byte("name: Workflow Two\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "two.md"), []byte("---\n---\n# Workflow Two\n"), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "missing-md.lock.yml"), []byte("name: Missing Source\n"), 0o644))

	engines, err := getAgenticWorkflowEnginesByName(false)
	require.NoError(t, err)

	assert.Equal(t, "claude", engines["Workflow One"])
	assert.Equal(t, "copilot", engines["Workflow Two"])
	_, hasMissing := engines["Missing Source"]
	assert.False(t, hasMissing)
}
