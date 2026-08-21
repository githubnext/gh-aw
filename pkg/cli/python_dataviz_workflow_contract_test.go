//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/stretchr/testify/require"
)

func TestPythonDatavizUsesPortableManagedPython(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	require.NoError(t, err)

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "shared", "python-dataviz.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err)
	workflowContent := string(content)

	require.Contains(t, workflowContent, "UV_PYTHON_INSTALL_DIR: /tmp/gh-aw/python/uv-python")
	require.Contains(t, workflowContent, "uv venv --python 3.12 --python-preference only-managed --seed /tmp/gh-aw/python/venv")
	require.Contains(t, workflowContent, "/tmp/gh-aw/python/venv/bin/pip install --quiet numpy pandas matplotlib seaborn scipy")
	require.NotContains(t, workflowContent, "python3 -m venv")
}
