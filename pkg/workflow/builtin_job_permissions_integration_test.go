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

// TestBuiltinJobPermissionsAugmentation verifies that user-declared permissions under
// jobs.<built-in>.permissions (e.g. safe_outputs, conclusion) are merged additively into the
// compiled built-in jobs, so scopes such as id-token: write are retained in the lock file.
func TestBuiltinJobPermissionsAugmentation(t *testing.T) {
	tmpDir := testutil.TempDir(t, "builtin-job-permissions-augmentation")
	compiler := NewCompiler()

	workflowContent := `---
on:
  issue_comment:
    types: [created]
engine: copilot
strict: false
permissions:
  contents: read
  id-token: write
safe-outputs:
  add-comment:
jobs:
  safe_outputs:
    permissions:
      id-token: write
      contents: read
      issues: write
  conclusion:
    permissions:
      id-token: write
      contents: read
      issues: write
---
Builtin job permissions augmentation
`

	workflowFile := filepath.Join(tmpDir, "builtin-job-permissions-augmentation.md")
	require.NoError(t, os.WriteFile(workflowFile, []byte(workflowContent), 0644))
	require.NoError(t, compiler.CompileWorkflow(workflowFile))

	lockFile := filepath.Join(tmpDir, "builtin-job-permissions-augmentation.lock.yml")
	lockBytes, err := os.ReadFile(lockFile)
	require.NoError(t, err)

	var lock map[string]any
	require.NoError(t, yaml.Unmarshal(lockBytes, &lock))
	jobs, ok := lock["jobs"].(map[string]any)
	require.True(t, ok)

	for _, jobName := range []string{"safe_outputs", "conclusion"} {
		job, ok := jobs[jobName].(map[string]any)
		require.True(t, ok, "expected %s job in compiled workflow", jobName)
		perms, ok := job["permissions"].(map[string]any)
		require.True(t, ok, "expected %s permissions to be a map", jobName)
		assert.Equal(t, "write", perms["id-token"], "%s should retain id-token: write from jobs.%s.permissions", jobName, jobName)
	}
}
