//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/stretchr/testify/require"
)

func TestCompileRuntimeCheckoutBaseManifest(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "runtime-base.md")
	content := `---
on: workflow_dispatch
engine: copilot
pre-steps:
  - id: resolve-target
    run: echo "branch=release/1" >> "$GITHUB_OUTPUT"
checkout:
  - repository: owner/side
    path: target
    ref: ${{ steps.resolve-target.outputs.branch || 'main' }}
    fetch-depth: 0
safe-outputs:
  create-pull-request:
    target-repo: owner/side
    allowed-base-branches: ["release/*"]
---
Fix the selected release branch.
`
	require.NoError(t, os.WriteFile(workflowPath, []byte(content), 0o644))
	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowPath))
	lock, err := os.ReadFile(stringutil.MarkdownToLockFile(workflowPath))
	require.NoError(t, err)
	require.Equal(t, 2, strings.Count(string(lock), "GH_AW_CHECKOUT_REF_0: ${{ steps.resolve-target.outputs.branch || 'main' }}"))
	require.Equal(t, 1, strings.Count(string(lock), "id: resolve-target"))
}

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
  - multi: true
    repository: ${{ github.event.inputs.trigger_ref }}
    path: targets
    allowed-repos: ["org/*"]
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
	require.Contains(t, lock, `GH_AW_DYNAMIC_CHECKOUT_INDEXES: "0"`)
	require.Contains(t, lock, `GH_AW_DYNAMIC_REPOSITORIES: "${{ github.event.inputs.trigger_ref }}"`)
	require.NotContains(t, lock, "repo='${{ github.event.inputs.trigger_ref }}'")
	require.NotContains(t, lock, "Build checkout manifest for safe-outputs handlers\n        run: |", "manifest step must not use run block")
}
