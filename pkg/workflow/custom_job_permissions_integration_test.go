//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func compileWorkflowAndReadLock(t *testing.T, frontmatter string) string {
	t.Helper()

	tmpDir := testutil.TempDir(t, "custom-job-permissions")
	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	content := frontmatter + "\n\n# Test Workflow\n\nTest permissions support.\n"
	require.NoError(t, os.WriteFile(workflowPath, []byte(content), 0o644))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowPath))

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	lockContent, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	return string(lockContent)
}

func TestCustomJobPermissionsIntegration(t *testing.T) {
	tests := []struct {
		name              string
		permissionsConfig string
		expectedParts     []string
	}{
		{
			name: "object permissions",
			permissionsConfig: `permissions:
      contents: write
      pull-requests: write`,
			expectedParts: []string{"contents: write", "pull-requests: write"},
		},
		{
			name:              "shorthand permissions",
			permissionsConfig: "permissions: read-all",
			expectedParts:     []string{"permissions: read-all"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lockContent := compileWorkflowAndReadLock(t, `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  custom_job:
    runs-on: ubuntu-latest
    `+tt.permissionsConfig+`
    steps:
      - run: echo "custom"
---`)

			jobSection := extractJobSection(lockContent, "custom_job")
			require.NotEmpty(t, jobSection)
			for _, expected := range tt.expectedParts {
				assert.Contains(t, jobSection, expected)
			}
		})
	}
}

func TestSafeJobPermissionsIntegration(t *testing.T) {
	tests := []struct {
		name              string
		permissionsConfig string
		expectedParts     []string
	}{
		{
			name: "object permissions",
			permissionsConfig: `permissions:
        contents: write
        issues: read`,
			expectedParts: []string{"contents: write", "issues: read"},
		},
		{
			name:              "shorthand permissions",
			permissionsConfig: "permissions: read-all",
			expectedParts:     []string{"permissions: read-all"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lockContent := compileWorkflowAndReadLock(t, `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
strict: false
safe-outputs:
  create-issue:
    title-prefix: "[test] "
  jobs:
    publish:
      runs-on: ubuntu-latest
      `+tt.permissionsConfig+`
      steps:
        - run: echo "publish"
---`)

			jobSection := extractJobSection(lockContent, "publish")
			require.NotEmpty(t, jobSection)
			for _, expected := range tt.expectedParts {
				assert.Contains(t, jobSection, expected)
			}
			assert.False(t, strings.Contains(jobSection, "permissions: {}"))
		})
	}
}
