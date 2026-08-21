//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/stretchr/testify/require"
)

const portableVenvCommand = "uv venv --python 3.12 --python-preference only-managed --seed /tmp/gh-aw/python/venv"

// Shared Python imports write into the same /tmp/gh-aw/python/venv, so every component that can
// create it first must provision the same uv-managed CPython. Otherwise a runner-CPython venv
// created by one import is reused by the others and fails to load native wheels in the sandbox.
func TestSharedPythonImportsUsePortableManagedPython(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	require.NoError(t, err)

	for _, sharedWorkflow := range []string{"python-dataviz.md", "python-nlp.md"} {
		t.Run(sharedWorkflow, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "shared", sharedWorkflow))
			require.NoError(t, err)
			workflowContent := string(content)

			require.Contains(t, workflowContent, "UV_PYTHON_INSTALL_DIR: /tmp/gh-aw/python/uv-python")
			require.Contains(t, workflowContent, portableVenvCommand)
			require.NotContains(t, workflowContent, "python3 -m venv")
		})
	}
}

// Consumers that import more than one shared Python component compile both setup steps into the
// same job, in import order. Whichever step runs first must create the portable environment.
func TestCompiledConsumersOnlyCreatePortableSharedVenv(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	require.NoError(t, err)

	for _, lockFile := range []string{"copilot-pr-nlp-analysis.lock.yml", "daily-issues-report.lock.yml"} {
		t.Run(lockFile, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", lockFile))
			require.NoError(t, err)
			lockContent := string(content)

			require.NotContains(t, lockContent, "python3 -m venv /tmp/gh-aw/python/venv")
			require.GreaterOrEqual(t, strings.Count(lockContent, portableVenvCommand), 1)
		})
	}
}
