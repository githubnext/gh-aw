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

// TestModelsCheckUsesCustomBaseURLFromEngineEnv verifies that when ANTHROPIC_BASE_URL is
// configured in engine.env, the models check step uses it at runtime for the API call.
func TestModelsCheckUsesCustomBaseURLFromEngineEnv(t *testing.T) {
	testDir := testutil.TempDir(t, "test-models-check-base-url-*")
	workflowFile := filepath.Join(testDir, "test-workflow.md")

	workflow := `---
on: workflow_dispatch
engine:
  id: claude
  env:
    ANTHROPIC_BASE_URL: "https://custom.anthropic.example.com/v1"
---

Test workflow`

	require.NoError(t, os.WriteFile(workflowFile, []byte(workflow), 0644), "write workflow file")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowFile), "compile workflow")

	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "read lock file")

	lockStr := string(lockContent)

	// The step should include ANTHROPIC_BASE_URL in the env section
	assert.Contains(t, lockStr, "ANTHROPIC_BASE_URL: https://custom.anthropic.example.com/v1",
		"Expected ANTHROPIC_BASE_URL in step env when set in engine.env")

	// The step should use dynamic URL construction (bash conditional)
	assert.Contains(t, lockStr, "MODELS_URL=",
		"Expected dynamic MODELS_URL variable in bash script")
	assert.Contains(t, lockStr, "ANTHROPIC_BASE_URL",
		"Expected ANTHROPIC_BASE_URL reference in bash URL construction")
	assert.Contains(t, lockStr, "https://api.anthropic.com/v1/models",
		"Expected default fallback URL in bash script")
}

// TestModelsCheckUsesCustomBaseURLForCodex verifies that when OPENAI_BASE_URL is
// configured in engine.env for Codex, the models check step uses it at runtime.
func TestModelsCheckUsesCustomBaseURLForCodex(t *testing.T) {
	testDir := testutil.TempDir(t, "test-models-check-codex-base-url-*")
	workflowFile := filepath.Join(testDir, "test-workflow.md")

	workflow := `---
on: workflow_dispatch
engine:
  id: codex
  env:
    OPENAI_BASE_URL: "https://my-openai-proxy.example.com/v1"
---

Test workflow`

	require.NoError(t, os.WriteFile(workflowFile, []byte(workflow), 0644), "write workflow file")

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowFile), "compile workflow")

	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "read lock file")

	lockStr := string(lockContent)

	// The step should include OPENAI_BASE_URL in the env section
	assert.Contains(t, lockStr, "OPENAI_BASE_URL: https://my-openai-proxy.example.com/v1",
		"Expected OPENAI_BASE_URL in step env when set in engine.env")

	// The step should use dynamic URL construction
	assert.Contains(t, lockStr, "MODELS_URL=",
		"Expected dynamic MODELS_URL variable in bash script")
	assert.Contains(t, lockStr, "OPENAI_BASE_URL",
		"Expected OPENAI_BASE_URL reference in bash URL construction")
	assert.Contains(t, lockStr, "https://api.openai.com/v1/models",
		"Expected default fallback URL in bash script")
}

// TestModelsCheckWithoutCustomBaseURLUsesDefaultURL verifies that when no custom base URL
// is configured, the models check step uses the static default URL directly.
func TestModelsCheckWithoutCustomBaseURLUsesDefaultURL(t *testing.T) {
	testDir := testutil.TempDir(t, "test-models-check-default-url-*")
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

	assert.Contains(t, lockStr, "https://api.anthropic.com/v1/models",
		"Expected default Anthropic models URL in generated step")
	// Without custom base URL, MODELS_URL is still set (via the unconditional branch)
	assert.Contains(t, lockStr, "MODELS_URL=",
		"Expected MODELS_URL to be set in bash script")
}
