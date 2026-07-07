package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestBuiltinJobNeedsAugmentation(t *testing.T) {
	tmpDir := testutil.TempDir(t, "builtin-job-needs-augmentation")
	compiler := NewCompiler()

	workflowContent := `---
on:
  issue_comment:
    types: [created]
  needs: [select_model]
engine: copilot
strict: false
safe-outputs:
  add-comment:
jobs:
  activation:
    needs: [select_model]
  agent:
    needs: [select_model]
  detection:
    needs: [select_model]
  safe_outputs:
    needs: [select_model]
  conclusion:
    needs: [select_model]
  select_model:
    runs-on: ubuntu-latest
    steps:
      - id: pick
        run: |
          echo "model=claude-sonnet-4.6" >> "$GITHUB_OUTPUT"
    outputs:
      model: ${{ steps.pick.outputs.model }}
---
Builtin job needs augmentation
`

	workflowFile := filepath.Join(tmpDir, "builtin-job-needs-augmentation.md")
	require.NoError(t, os.WriteFile(workflowFile, []byte(workflowContent), 0644))
	require.NoError(t, compiler.CompileWorkflow(workflowFile))

	lockFile := filepath.Join(tmpDir, "builtin-job-needs-augmentation.lock.yml")
	lockBytes, err := os.ReadFile(lockFile)
	require.NoError(t, err)

	var lock map[string]any
	require.NoError(t, yaml.Unmarshal(lockBytes, &lock))
	jobs, ok := lock["jobs"].(map[string]any)
	require.True(t, ok)

	for _, jobName := range []string{"activation", "agent", "detection", "safe_outputs", "conclusion"} {
		job, ok := jobs[jobName].(map[string]any)
		require.True(t, ok, "expected %s job in compiled workflow", jobName)
		assert.Contains(t, job["needs"], "select_model", "%s should include jobs.%s.needs augmentation", jobName, jobName)
	}
}

func TestBuiltinJobNeedsAugmentationUnknownJob(t *testing.T) {
	tmpDir := testutil.TempDir(t, "builtin-job-needs-unknown")
	compiler := NewCompiler()

	workflowContent := `---
on:
  issue_comment:
    types: [created]
engine: copilot
strict: false
safe-outputs:
  add-comment:
jobs:
  detection:
    needs: [missing_job]
---
Builtin job needs augmentation validation
`

	workflowFile := filepath.Join(tmpDir, "builtin-job-needs-unknown.md")
	require.NoError(t, os.WriteFile(workflowFile, []byte(workflowContent), 0644))

	err := compiler.CompileWorkflow(workflowFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `jobs.detection.needs: unknown job "missing_job"`)
}
