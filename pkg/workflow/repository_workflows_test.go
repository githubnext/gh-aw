//go:build !integration

package workflow

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepositoryWorkflowsCompileDryRun(t *testing.T) {
	// Resolve remote imports from the checked-in repository cache, not pkg/workflow.
	t.Chdir(filepath.Join("..", ".."))
	sources, err := filepath.Glob(filepath.Join(".github", "workflows", "*.md"))
	require.NoError(t, err)
	require.NotEmpty(t, sources, "should discover repository agentic workflows")

	for _, source := range sources {
		name := strings.TrimSuffix(filepath.Base(source), ".md")
		t.Run(name, func(t *testing.T) {
			compiler := NewCompiler(
				WithVersion("integration"),
				WithWorkflowIdentifier(name),
				WithNoEmit(true),
			)
			compiler.SetDryRun(true)
			compiler.SetQuiet(true)

			require.NoError(t, compiler.CompileWorkflow(source), "repository workflow should compile in dry-run mode")
		})
	}
}
