//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnNeedsCompilesAndWiresActivationDependencies(t *testing.T) {
	tmpDir := testutil.TempDir(t, "on-needs-integration")
	compiler := NewCompiler()

	workflowContent := `---
on:
  workflow_dispatch:
  needs: [secrets_fetcher]
  github-app:
    client-id: ${{ needs.secrets_fetcher.outputs.app_id }}
    private-key: ${{ needs.secrets_fetcher.outputs.private_key }}
engine: copilot
jobs:
  secrets_fetcher:
    runs-on: ubuntu-latest
    outputs:
      app_id: ${{ steps.fetch.outputs.app_id }}
      private_key: ${{ steps.fetch.outputs.private_key }}
    steps:
      - id: fetch
        run: |
          echo "app_id=123" >> "$GITHUB_OUTPUT"
          echo "private_key=key" >> "$GITHUB_OUTPUT"
---
Run with on.needs
`

	workflowFile := filepath.Join(tmpDir, "test-on-needs.md")
	require.NoError(t, os.WriteFile(workflowFile, []byte(workflowContent), 0644), "should write test workflow")

	require.NoError(t, compiler.CompileWorkflow(workflowFile), "workflow should compile with on.needs and on.github-app needs expression")

	lockFile := filepath.Join(tmpDir, "test-on-needs.lock.yml")
	lockBytes, err := os.ReadFile(lockFile)
	require.NoError(t, err, "should read compiled lock file")

	var lock map[string]any
	require.NoError(t, yaml.Unmarshal(lockBytes, &lock), "compiled lock file should be valid YAML")

	jobs, ok := lock["jobs"].(map[string]any)
	require.True(t, ok, "compiled workflow should contain jobs map")

	preActivation, ok := jobs["pre_activation"].(map[string]any)
	require.True(t, ok, "compiled workflow should contain pre_activation job")
	assert.Contains(t, preActivation["needs"], "secrets_fetcher", "pre_activation should depend on on.needs job")

	activation, ok := jobs["activation"].(map[string]any)
	require.True(t, ok, "compiled workflow should contain activation job")
	assert.Contains(t, activation["needs"], "secrets_fetcher", "activation should depend on on.needs job")
}
