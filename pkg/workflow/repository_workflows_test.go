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

func TestDailyCodeMetricsUsesArtifactAICAccounting(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	compiler := NewCompiler(WithWorkflowIdentifier("daily-code-metrics"))
	data, err := compiler.ParseWorkflowFile(filepath.Join(".github", "workflows", "daily-code-metrics.md"))
	require.NoError(t, err)

	require.NotNil(t, data.MaxDailyAICredits)
	require.Equal(t, "10000", *data.MaxDailyAICredits)
	require.Empty(t, data.MaxDailyAICBackend, "daily credits should use the default artifact backend")
	require.False(t, data.MaxDailyAICContinueOnError, "unknown accounting must still block activation")
	require.NotNil(t, data.RepoMemoryConfig, "historical metrics must retain repo-memory")
	require.NotEmpty(t, data.RepoMemoryConfig.Memories)
	require.Equal(t, "daily/daily-code-metrics", data.RepoMemoryConfig.Memories[0].BranchName)

	steps := strings.Join(compiler.buildActivationDailyAICGuardrailStep(data), "")
	require.Contains(t, steps, "Restore daily AIC scan observations")
	require.Contains(t, steps, "Publish daily AIC scan observations")
	require.NotContains(t, steps, "GH_AW_DAILY_AIC_REPO_MEMORY_TRUSTED")
	require.NotContains(t, steps, "GH_AW_ALLOW_INSECURE_REPO_MEMORY_AIC")
}
