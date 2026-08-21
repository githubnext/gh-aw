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

	// Each component installs its own packages additively into the shared portable environment.
	packageInstalls := map[string]string{
		"python-dataviz.md":         "/tmp/gh-aw/python/venv/bin/pip install --quiet numpy pandas matplotlib seaborn scipy",
		"python-nlp.md":             "/tmp/gh-aw/python/venv/bin/pip install --quiet nltk scikit-learn textblob wordcloud",
		"trending-charts-simple.md": "uv pip install --quiet --python /tmp/gh-aw/python/venv/bin/python numpy pandas matplotlib seaborn scipy",
	}

	for sharedWorkflow, packageInstall := range packageInstalls {
		t.Run(sharedWorkflow, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "shared", sharedWorkflow))
			require.NoError(t, err)
			workflowContent := string(content)

			require.Contains(t, workflowContent, "UV_PYTHON_INSTALL_DIR: /tmp/gh-aw/python/uv-python")
			require.Contains(t, workflowContent, portableVenvCommand)
			require.Contains(t, workflowContent, packageInstall)
			require.NotContains(t, workflowContent, "python3 -m venv")
			// Recreating the shared environment would discard a sibling import's packages.
			require.NotContains(t, workflowContent, "rm -rf /tmp/gh-aw/python/venv")
		})
	}
}

// Consumers that import more than one shared Python component compile both setup steps into the
// same job, in import order. Whichever step runs first must create the portable environment.
func TestCompiledConsumersOnlyCreatePortableSharedVenv(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	require.NoError(t, err)

	lockFiles := []string{
		// python-dataviz.md + python-nlp.md
		"copilot-pr-nlp-analysis.lock.yml",
		"daily-issues-report.lock.yml",
		// python-nlp.md + trending-charts-simple.md
		"prompt-clustering-analysis.lock.yml",
		// python-dataviz.md + trending-charts-simple.md
		"daily-security-observability.lock.yml",
	}

	for _, lockFile := range lockFiles {
		t.Run(lockFile, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", lockFile))
			require.NoError(t, err)
			lockContent := string(content)

			require.NotContains(t, lockContent, "python3 -m venv /tmp/gh-aw/python/venv")
			require.NotContains(t, lockContent, "rm -rf /tmp/gh-aw/python/venv")
			require.GreaterOrEqual(t, strings.Count(lockContent, portableVenvCommand), 1)
		})
	}
}
