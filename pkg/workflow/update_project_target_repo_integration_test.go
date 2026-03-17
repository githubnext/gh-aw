//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateProjectTargetRepoInCompiledConfig verifies that when a workflow is compiled with
// update-project.target-repo and update-project.allowed-repos, those values are present in
// both the config.json written to disk and in the GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG env var.
func TestUpdateProjectTargetRepoInCompiledConfig(t *testing.T) {
	tests := []struct {
		name              string
		workflowContent   string
		expectedInYAML    []string
		notExpectedInYAML []string
	}{
		{
			name: "target-repo and allowed-repos appear in compiled config",
			workflowContent: `---
name: Test Update Project Cross Repo
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
safe-outputs:
  update-project:
    github-token: ${{ secrets.GH_AW_WRITE_PROJECT_TOKEN }}
    project: "https://github.com/orgs/myorg/projects/42"
    target-repo: myorg/backend
    allowed-repos:
      - myorg/docs
      - myorg/frontend
---

Test workflow for cross-repo project item resolution.
`,
			expectedInYAML: []string{
				// config.json written in safe_outputs job
				`"target-repo":"myorg/backend"`,
				`"allowed_repos":["myorg/docs","myorg/frontend"]`,
				// handler config env var (JSON-encoded, quotes escaped)
				`\"target-repo\":\"myorg/backend\"`,
				`\"allowed_repos\":[\"myorg/docs\",\"myorg/frontend\"]`,
			},
			notExpectedInYAML: nil,
		},
		{
			name: "without target-repo and allowed-repos, neither appears in compiled config",
			workflowContent: `---
name: Test Update Project No Cross Repo
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
safe-outputs:
  update-project:
    github-token: ${{ secrets.GH_AW_WRITE_PROJECT_TOKEN }}
    project: "https://github.com/orgs/myorg/projects/42"
---

Test workflow without cross-repo configuration.
`,
			expectedInYAML: []string{
				// project URL and update_project key are still present
				`update_project`,
				`https://github.com/orgs/myorg/projects/42`,
			},
			notExpectedInYAML: []string{
				`"target-repo"`,
				`"allowed_repos"`,
				`\"target-repo\"`,
				`\"allowed_repos\"`,
			},
		},
		{
			name: "target-repo only (no allowed-repos) appears in compiled config",
			workflowContent: `---
name: Test Update Project Target Repo Only
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
safe-outputs:
  update-project:
    github-token: ${{ secrets.GH_AW_WRITE_PROJECT_TOKEN }}
    project: "https://github.com/orgs/myorg/projects/42"
    target-repo: myorg/backend
---

Test workflow with only target-repo (no allowed-repos list).
`,
			expectedInYAML: []string{
				`"target-repo":"myorg/backend"`,
				`\"target-repo\":\"myorg/backend\"`,
			},
			notExpectedInYAML: []string{
				`"allowed_repos"`,
				`\"allowed_repos\"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "update-project-target-repo-test")

			mdFile := filepath.Join(tmpDir, "test-workflow.md")
			err := os.WriteFile(mdFile, []byte(tt.workflowContent), 0600)
			require.NoError(t, err, "Failed to write test markdown file")

			compiler := NewCompiler()
			err = compiler.CompileWorkflow(mdFile)
			require.NoError(t, err, "Failed to compile workflow")

			lockFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
			compiledBytes, err := os.ReadFile(lockFile)
			require.NoError(t, err, "Failed to read compiled lock file")

			compiledStr := string(compiledBytes)

			for _, expected := range tt.expectedInYAML {
				assert.True(t,
					strings.Contains(compiledStr, expected),
					"Expected compiled YAML to contain %q\nCompiled output:\n%s", expected, compiledStr,
				)
			}

			for _, notExpected := range tt.notExpectedInYAML {
				assert.False(t,
					strings.Contains(compiledStr, notExpected),
					"Expected compiled YAML NOT to contain %q\nCompiled output:\n%s", notExpected, compiledStr,
				)
			}
		})
	}
}

// TestUpdateProjectTargetRepoWorkflowFile verifies that the sample workflow file in
// pkg/cli/workflows compiles successfully and produces config with target-repo set.
func TestUpdateProjectTargetRepoWorkflowFile(t *testing.T) {
	workflowFile := "../cli/workflows/test-copilot-update-project-cross-repo.md"

	compiler := NewCompiler()
	// Set a temporary output dir so we don't write .lock.yml next to the source
	tmpDir := testutil.TempDir(t, "update-project-workflow-file-test")
	mdDst := filepath.Join(tmpDir, "test-copilot-update-project-cross-repo.md")

	src, err := os.ReadFile(workflowFile)
	require.NoError(t, err, "Failed to read sample workflow file %s", workflowFile)

	err = os.WriteFile(mdDst, src, 0600)
	require.NoError(t, err, "Failed to copy sample workflow file")

	err = compiler.CompileWorkflow(mdDst)
	require.NoError(t, err, "Sample workflow %s should compile without errors", workflowFile)

	lockFile := filepath.Join(tmpDir, "test-copilot-update-project-cross-repo.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read compiled lock file")

	compiledStr := string(compiledBytes)

	// The workflow declares target-repo: myorg/backend and allowed-repos: [myorg/docs, myorg/frontend]
	assert.Contains(t, compiledStr, `"target-repo":"myorg/backend"`,
		"config.json should contain target-repo")
	assert.Contains(t, compiledStr, `"allowed_repos":["myorg/docs","myorg/frontend"]`,
		"config.json should contain allowed_repos list")
	assert.Contains(t, compiledStr, `\"target-repo\":\"myorg/backend\"`,
		"GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG should contain target-repo")
	assert.Contains(t, compiledStr, `update_project`,
		"Compiled workflow should reference update_project handler")
}
