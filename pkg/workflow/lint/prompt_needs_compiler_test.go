package lint_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/github/gh-aw/pkg/workflow/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptNeedsCompilerDeferredBindingContract(t *testing.T) {
	dir, err := os.MkdirTemp(".", ".prompt-needs-compiler-")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(dir)) })
	source := `---
on: workflow_dispatch
engine: copilot
jobs:
  release:
    needs: activation
    runs-on: ubuntu-latest
    outputs:
      release_id: ${{ steps.release.outputs.id }}
    steps:
      - id: release
        run: echo "id=123" >> "$GITHUB_OUTPUT"
---
Release ID: ${{ needs.release.outputs.release_id }}
`
	path := filepath.Join(dir, "release.md")
	require.NoError(t, os.WriteFile(path, []byte(source), 0o600))
	compiler := workflow.NewCompiler()
	compiler.SetQuiet(true)
	require.NoError(t, compiler.CompileWorkflow(path))
	lock, err := os.ReadFile(filepath.Join(dir, "release.lock.yml"))
	require.NoError(t, err)
	require.Contains(t, string(lock), "id: "+string(constants.DeferredPromptStepID))
	document, err := aw.Parse(source)
	require.NoError(t, err)
	compiled, err := actions.Parse(lock)
	require.NoError(t, err)
	input := lint.Input{Workflow: compiled, Document: document}
	diagnostics, err := lint.NeedsConsistencyRule().Check(context.Background(), input)
	require.NoError(t, err)
	assert.Empty(t, diagnostics, "the linter must recognize real compiler-generated deferred bindings")

	withoutContract := strings.Replace(string(lock), "id: "+string(constants.DeferredPromptStepID), "id: unrecognized_resolver", 1)
	input.Workflow, err = actions.Parse([]byte(withoutContract))
	require.NoError(t, err)
	diagnostics, err = lint.NeedsConsistencyRule().Check(context.Background(), input)
	require.NoError(t, err)
	require.Len(t, diagnostics, 1, "removing the generated binding contract must restore the activation finding")
	assert.Equal(t, "prompt", diagnostics[0].Path)
}
