//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModelsCheckStepGeneratedForClaude verifies that the models check step is emitted
// in the agent job when the engine is claude.
func TestModelsCheckStepGeneratedForClaude(t *testing.T) {
	testDir := testutil.TempDir(t, "test-models-check-claude-*")
	workflowFile := filepath.Join(testDir, "test-workflow.md")

	workflow := `---
on: workflow_dispatch
engine: claude
---

Test workflow`

	require.NoError(t, os.WriteFile(workflowFile, []byte(workflow), 0644), "write workflow file")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowFile), "compile workflow")

	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "read lock file")

	lockStr := string(lockContent)

	assert.Contains(t, lockStr, "id: "+string(constants.ModelsCheckStepID),
		"Expected models check step ID in agent job")
	assert.Contains(t, lockStr, "api.anthropic.com/v1/models",
		"Expected Anthropic models URL in models check step")
	assert.Contains(t, lockStr, "ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}",
		"Expected ANTHROPIC_API_KEY env var in models check step")
	assert.Contains(t, lockStr, "models_check_failed",
		"Expected models_check_failed output in models check step")
}

// TestModelsCheckStepGeneratedForCodex verifies that the models check step is emitted
// in the agent job when the engine is codex.
func TestModelsCheckStepGeneratedForCodex(t *testing.T) {
	testDir := testutil.TempDir(t, "test-models-check-codex-*")
	workflowFile := filepath.Join(testDir, "test-workflow.md")

	workflow := `---
on: workflow_dispatch
engine: codex
---

Test workflow`

	require.NoError(t, os.WriteFile(workflowFile, []byte(workflow), 0644), "write workflow file")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowFile), "compile workflow")

	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "read lock file")

	lockStr := string(lockContent)

	assert.Contains(t, lockStr, "id: "+string(constants.ModelsCheckStepID),
		"Expected models check step ID in agent job")
	assert.Contains(t, lockStr, "api.openai.com/v1/models",
		"Expected OpenAI models URL in models check step")
	assert.Contains(t, lockStr, "OPENAI_API_KEY: ${{ secrets.CODEX_API_KEY || secrets.OPENAI_API_KEY }}",
		"Expected OPENAI_API_KEY env var in models check step")
	assert.Contains(t, lockStr, "models_check_failed",
		"Expected models_check_failed output in models check step")
}

// TestModelsCheckJobOutputForClaude verifies that the models_check_failed job output
// is exposed on the agent job for engines that implement ModelsProvider.
func TestModelsCheckJobOutputForClaude(t *testing.T) {
	testDir := testutil.TempDir(t, "test-models-check-output-*")
	workflowFile := filepath.Join(testDir, "test-workflow.md")

	workflow := `---
on: workflow_dispatch
engine: claude
---

Test workflow`

	require.NoError(t, os.WriteFile(workflowFile, []byte(workflow), 0644), "write workflow file")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowFile), "compile workflow")

	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "read lock file")

	lockStr := string(lockContent)

	assert.Contains(t, lockStr,
		"models_check_failed: ${{ steps.models-check.outputs.models_check_failed || 'false' }}",
		"Expected models_check_failed job output referencing models-check step")
}

// TestModelsCheckPassedToConclusionJob verifies that the models_check_failed output is
// passed to the handle_agent_failure.cjs step in the conclusion job.
func TestModelsCheckPassedToConclusionJob(t *testing.T) {
	testDir := testutil.TempDir(t, "test-models-check-conclusion-*")
	workflowFile := filepath.Join(testDir, "test-workflow.md")

	workflow := `---
on: workflow_dispatch
engine: claude
safe-outputs:
  add-comment:
    max: 5
---

Test workflow`

	require.NoError(t, os.WriteFile(workflowFile, []byte(workflow), 0644), "write workflow file")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowFile), "compile workflow")

	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "read lock file")

	lockStr := string(lockContent)

	assert.Contains(t, lockStr, "GH_AW_MODELS_CHECK_FAILED: ${{ needs.agent.outputs.models_check_failed }}",
		"Expected GH_AW_MODELS_CHECK_FAILED env var in conclusion job referencing agent job output")
}

// TestModelsCheckNotGeneratedForCopilot verifies that the models check step is NOT
// generated for the copilot engine (which does not implement ModelsProvider).
func TestModelsCheckNotGeneratedForCopilot(t *testing.T) {
	testDir := testutil.TempDir(t, "test-models-check-copilot-*")
	workflowFile := filepath.Join(testDir, "test-workflow.md")

	workflow := `---
on: workflow_dispatch
engine: copilot
---

Test workflow`

	require.NoError(t, os.WriteFile(workflowFile, []byte(workflow), 0644), "write workflow file")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowFile), "compile workflow")

	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "read lock file")

	lockStr := string(lockContent)

	assert.NotContains(t, lockStr, "id: "+string(constants.ModelsCheckStepID),
		"Expected no models check step for copilot engine")
	assert.NotContains(t, lockStr, "models_check_failed",
		"Expected no models_check_failed output for copilot engine")
}

// TestModelsCheckSkippedWithCustomCommand verifies that the models check step is NOT
// generated when a custom engine command is specified.
func TestModelsCheckSkippedWithCustomCommand(t *testing.T) {
	testDir := testutil.TempDir(t, "test-models-check-custom-cmd-*")
	workflowFile := filepath.Join(testDir, "test-workflow.md")

	workflow := `---
on: workflow_dispatch
engine:
  id: claude
  command: /custom/claude
---

Test workflow`

	require.NoError(t, os.WriteFile(workflowFile, []byte(workflow), 0644), "write workflow file")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowFile), "compile workflow")

	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "read lock file")

	lockStr := string(lockContent)

	assert.NotContains(t, lockStr, "id: "+string(constants.ModelsCheckStepID),
		"Expected no models check step when custom command is specified")
}

// TestModelsCheckSkippedWithEnvironment verifies that the models check step is NOT
// generated when a top-level environment is configured (secret validation is skipped too).
func TestModelsCheckSkippedWithEnvironment(t *testing.T) {
	testDir := testutil.TempDir(t, "test-models-check-env-*")
	workflowFile := filepath.Join(testDir, "test-workflow.md")

	workflow := `---
on: workflow_dispatch
engine: claude
environment: production
---

Test workflow`

	require.NoError(t, os.WriteFile(workflowFile, []byte(workflow), 0644), "write workflow file")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowFile), "compile workflow")

	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "read lock file")

	lockStr := string(lockContent)

	assert.NotContains(t, lockStr, "id: "+string(constants.ModelsCheckStepID),
		"Expected no models check step when top-level environment is configured")
}

// TestModelsCheckStepBeforeAgentExecution verifies that the models check step appears
// before the agent execution step in the compiled YAML.
func TestModelsCheckStepBeforeAgentExecution(t *testing.T) {
	testDir := testutil.TempDir(t, "test-models-check-order-*")
	workflowFile := filepath.Join(testDir, "test-workflow.md")

	workflow := `---
on: workflow_dispatch
engine: claude
---

Test workflow`

	require.NoError(t, os.WriteFile(workflowFile, []byte(workflow), 0644), "write workflow file")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowFile), "compile workflow")

	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "read lock file")

	lockStr := string(lockContent)

	modelsCheckPos := strings.Index(lockStr, "id: "+string(constants.ModelsCheckStepID))
	// Claude execution step runs "claude" CLI
	claudeExecPos := strings.Index(lockStr, "name: Execute Claude Code CLI")

	require.Positive(t, modelsCheckPos, "models check step should be present")
	require.Positive(t, claudeExecPos, "Claude execution step should be present")
	assert.Less(t, modelsCheckPos, claudeExecPos,
		"models check step should appear before the agent execution step")
}
