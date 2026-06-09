//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestCompileDynamicCheckoutRepositoryManifestStepUsesGitHubScript(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "repro.md")
	content := `---
on:
  workflow_dispatch:
    inputs:
      trigger_ref:
        type: string
        required: true
engine: copilot
timeout-minutes: 5
checkout:
  - repository: ${{ github.event.inputs.trigger_ref }}
    current: true
safe-outputs:
  noop:
    report-as-issue: false
---
# Repro
Do nothing.
`
	require.NoError(t, os.WriteFile(workflowPath, []byte(content), 0o644))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowPath))

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	lockBytes, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	lock := string(lockBytes)

	require.Contains(t, lock, "name: Build checkout manifest for safe-outputs handlers")
	require.Contains(t, lock, "uses: actions/github-script")
	require.Contains(t, lock, "build_checkout_manifest.cjs")
	require.Contains(t, lock, "GH_AW_CHECKOUT_REPO_0: ${{ github.event.inputs.trigger_ref }}")
	require.NotContains(t, lock, "repo='${{ github.event.inputs.trigger_ref }}'")
	require.NotContains(t, lock, "Build checkout manifest for safe-outputs handlers\n        run: |", "manifest step must not use run block")
}
