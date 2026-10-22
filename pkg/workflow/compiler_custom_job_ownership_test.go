//go:build !integration

package workflow

import (
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomJobDetectionReservedNameCompilation(t *testing.T) {
	document, err := aw.Parse("---\non: workflow_dispatch\nengine: copilot\npermissions: {contents: read}\nsafe-outputs:\n  create-issue:\njobs:\n  evals:\n    uses: ./.github/workflows/report.yml\n    needs: agent\n    if: always()\n---\nPublic prompt.\n")
	require.NoError(t, err)
	compiler := NewCompiler(WithNoEmit(true))
	data, err := compiler.ParseWorkflowDocument(document, filepath.Join(t.TempDir(), "test.md"))
	require.NoError(t, err)
	_, err = compiler.CompileToYAML(data, "test.md")
	require.ErrorContains(t, err, "jobs.evals: this name is reserved")
}

func TestCustomJobDetectionBuiltinCustomization(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()
	agent := &Job{Name: "agent", Steps: []string{"      - run: echo generated\n"}}
	require.NoError(t, compiler.jobManager.AddJob(agent))
	require.NoError(t, compiler.buildCustomJobs(&WorkflowData{
		Jobs: map[string]any{"agent": map[string]any{"needs": "prepare"}},
	}, false))
	actual, exists := compiler.jobManager.GetJob("agent")
	require.True(t, exists)
	assert.Same(t, agent, actual, "existing generated jobs remain customization-only")
}

func TestCustomJobDetectionAuthoredMaintenanceCompilation(t *testing.T) {
	for _, name := range []string{"push_evals_state", "push_ledger_changes", "push_experiments_state", "push_repo_memory", "update_cache_memory", "update_drive_memory"} {
		t.Run(name, func(t *testing.T) {
			document, err := aw.Parse("---\non: workflow_dispatch\nengine: copilot\npermissions: {contents: read}\nsafe-outputs:\n  create-issue:\njobs:\n  " + name + ":\n    uses: ./.github/workflows/report.yml\n    needs: agent\n    if: always()\n---\nPublic prompt.\n")
			require.NoError(t, err)
			compiler := NewCompiler(WithNoEmit(true))
			data, err := compiler.ParseWorkflowDocument(document, filepath.Join(t.TempDir(), "test.md"))
			require.NoError(t, err)
			_, err = compiler.CompileToYAML(data, "test.md")
			require.NoError(t, err)
			job, exists := compiler.jobManager.GetJob(name)
			require.True(t, exists)
			assert.Contains(t, job.Needs, "detection")
			assert.Contains(t, job.If, "needs.detection.outputs.detection_success == 'true'")
		})
	}
}
