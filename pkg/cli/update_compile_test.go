package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewUpdateCompileConfigMatchesCompileCommandDefaults(t *testing.T) {
	files := []string{".github/workflows/first.md", ".github/workflows/second.md"}

	got := newUpdateCompileConfig(files, "custom/workflows", "copilot", true, true)

	require.Equal(t, CompileConfig{
		MarkdownFiles:  files,
		Verbose:        true,
		EngineOverride: "copilot",
		WorkflowDir:    "custom/workflows",
		Approve:        true,
	}, got)
	require.False(t, got.RefreshStopTime, "update compilation must preserve stop times like compile")
}

func TestCompileWorkflowsForUpdatePropagatesCompileErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := compileWorkflowsForUpdate(ctx, []string{"workflow.md"}, "", "", false, false)

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	var compilationErr *updateCompilationError
	require.ErrorAs(t, err, &compilationErr)
}

func TestRecompileAllWorkflowsPropagatesCompileErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := recompileAllWorkflows(ctx, ".github/workflows", "", false, false)

	require.ErrorIs(t, err, context.Canceled)
}
