//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeOutputsConfigUsesWorkflowInputEnvVarsForDynamicAllowedRepos(t *testing.T) {
	tmpDir := testutil.TempDir(t, "safe-outputs-dynamic-allowed-repos")
	mdFile := filepath.Join(tmpDir, "dynamic-safe-outputs.md")

	content := `---
name: Dynamic Safe Outputs
on:
  workflow_dispatch:
    inputs:
      target_repo:
        required: true
        type: string
      base_branch:
        required: true
        type: string
engine: copilot
safe-outputs:
  create-pull-request:
    allowed-repos:
      - ${{ inputs.target_repo }}
    allowed-base-branches:
      - ${{ inputs.base_branch }}
---

Test workflow
`

	err := os.WriteFile(mdFile, []byte(content), 0600)
	require.NoError(t, err, "Failed to write test workflow markdown")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(mdFile)
	require.NoError(t, err, "Failed to compile workflow")

	lockFile := stringutil.MarkdownToLockFile(mdFile)
	compiledBytes, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read compiled workflow")
	compiled := string(compiledBytes)

	assert.Contains(t, compiled, "GH_AW_INPUT_TARGET_REPO: ${{ inputs.target_repo }}",
		"Generate Safe Outputs Config step should map inputs.target_repo to an env var")
	assert.Contains(t, compiled, "GH_AW_INPUT_BASE_BRANCH: ${{ inputs.base_branch }}",
		"Generate Safe Outputs Config step should map inputs.base_branch to an env var")
	assert.GreaterOrEqual(t, strings.Count(compiled, "GH_AW_INPUT_TARGET_REPO: ${{ inputs.target_repo }}"), 2,
		"GH_AW_INPUT_TARGET_REPO should appear in both the Generate Safe Outputs Config step and the Start MCP Gateway step")
	assert.GreaterOrEqual(t, strings.Count(compiled, "GH_AW_INPUT_BASE_BRANCH: ${{ inputs.base_branch }}"), 2,
		"GH_AW_INPUT_BASE_BRANCH should appear in both the Generate Safe Outputs Config step and the Start MCP Gateway step")
	assert.Contains(t, compiled, `"allowed_repos":"${GH_AW_INPUT_TARGET_REPO}"`,
		"config.json payload should preserve env placeholder for allowed_repos")
	assert.Contains(t, compiled, `"allowed_base_branches":"${GH_AW_INPUT_BASE_BRANCH}"`,
		"config.json payload should preserve env placeholder for allowed_base_branches")

	quotedHeredocPattern := regexp.MustCompile(`cat > "\$\{RUNNER_TEMP\}/gh-aw/safeoutputs/config\.json" << 'GH_AW_SAFE_OUTPUTS_CONFIG_[0-9a-f]{16}_EOF'`)
	assert.True(t, quotedHeredocPattern.MatchString(compiled),
		"Safe outputs config heredoc should be single-quoted so placeholders are not expanded onto disk")

	unquotedHeredocPattern := regexp.MustCompile(`cat > "\$\{RUNNER_TEMP\}/gh-aw/safeoutputs/config\.json" << GH_AW_SAFE_OUTPUTS_CONFIG_[0-9a-f]{16}_EOF`)
	assert.False(t, unquotedHeredocPattern.MatchString(compiled),
		"Safe outputs config heredoc should never be unquoted for dynamic config placeholders")

	// Verify GH_AW_INPUT_* env vars are forwarded to the MCP gateway container via -e flags.
	// Without this the safe-outputs MCP server cannot resolve ${GH_AW_INPUT_…} placeholders in
	// config.json, causing failures such as "No remote refs available for merge-base calculation".
	assert.Contains(t, compiled, "-e GH_AW_INPUT_TARGET_REPO",
		"MCP gateway docker run command should include -e GH_AW_INPUT_TARGET_REPO so the container can resolve the placeholder")
	assert.Contains(t, compiled, "-e GH_AW_INPUT_BASE_BRANCH",
		"MCP gateway docker run command should include -e GH_AW_INPUT_BASE_BRANCH so the container can resolve the placeholder")
}

func TestSafeOutputsConfigPreservesSecretPlaceholdersOnDisk(t *testing.T) {
	tmpDir := testutil.TempDir(t, "safe-outputs-secret-placeholders")
	mdFile := filepath.Join(tmpDir, "secret-safe-outputs.md")

	content := `---
name: Secret Safe Outputs
on:
  workflow_dispatch:
engine: copilot
safe-outputs:
  update-project:
    github-token: ${{ secrets.WRITE_PROJECT_PAT }}
    project: https://github.com/orgs/github/projects/24263
---

Test workflow
`

	err := os.WriteFile(mdFile, []byte(content), 0600)
	require.NoError(t, err, "Failed to write test workflow markdown")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(mdFile)
	require.NoError(t, err, "Failed to compile workflow")

	lockFile := stringutil.MarkdownToLockFile(mdFile)
	compiledBytes, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read compiled workflow")
	compiled := string(compiledBytes)

	assert.Contains(t, compiled, "GH_AW_SECRET_WRITE_PROJECT_PAT: ${{ secrets.WRITE_PROJECT_PAT }}",
		"Generate Safe Outputs Config step should map secret expressions to prefixed env vars")
	assert.GreaterOrEqual(t, strings.Count(compiled, "GH_AW_SECRET_WRITE_PROJECT_PAT: ${{ secrets.WRITE_PROJECT_PAT }}"), 1,
		"Secret env vars should be exported anywhere the runtime still resolves the placeholder in memory")
	assert.Contains(t, compiled, `"github-token":"${GH_AW_SECRET_WRITE_PROJECT_PAT}"`,
		"config.json payload should preserve the prefixed secret placeholder instead of the secret value")

	quotedHeredocPattern := regexp.MustCompile(`cat > "\$\{RUNNER_TEMP\}/gh-aw/safeoutputs/config\.json" << 'GH_AW_SAFE_OUTPUTS_CONFIG_[0-9a-f]{16}_EOF'`)
	assert.True(t, quotedHeredocPattern.MatchString(compiled),
		"Safe outputs config heredoc should be single-quoted so secret placeholders are not expanded onto disk")

	unquotedHeredocPattern := regexp.MustCompile(`cat > "\$\{RUNNER_TEMP\}/gh-aw/safeoutputs/config\.json" << GH_AW_SAFE_OUTPUTS_CONFIG_[0-9a-f]{16}_EOF`)
	assert.False(t, unquotedHeredocPattern.MatchString(compiled),
		"Safe outputs config heredoc should not be unquoted when secrets are present")
}

// TestSafeOutputsDynamicBaseBranchPassedToMCPContainer is a regression test for
// github/gh-aw#47795: when create-pull-request uses a dynamic base-branch such as
// ${{ inputs.base_branch }}, the compiled workflow must forward the resolved value to
// the MCP gateway container both in the step env block and in the docker run -e allowlist.
// Without this the safe-outputs MCP server cannot resolve the ${GH_AW_INPUT_BASE_BRANCH}
// placeholder in config.json and fails with "No remote refs available for merge-base calculation".
func TestSafeOutputsDynamicBaseBranchPassedToMCPContainer(t *testing.T) {
	tmpDir := testutil.TempDir(t, "safe-outputs-dynamic-base-branch")
	mdFile := filepath.Join(tmpDir, "dynamic-base-branch.md")

	content := `---
name: Dynamic Base Branch
on:
  workflow_dispatch:
    inputs:
      base_branch:
        required: false
        default: develop
        type: string
engine: copilot
safe-outputs:
  create-pull-request:
    base-branch: ${{ inputs.base_branch }}
---

Test workflow
`

	err := os.WriteFile(mdFile, []byte(content), 0600)
	require.NoError(t, err, "Failed to write test workflow markdown")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(mdFile)
	require.NoError(t, err, "Failed to compile workflow")

	lockFile := stringutil.MarkdownToLockFile(mdFile)
	compiledBytes, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read compiled workflow")
	compiled := string(compiledBytes)

	// The placeholder must appear in config.json on disk so no shell expansion happens at write-time.
	assert.Contains(t, compiled, `"base_branch":"${GH_AW_INPUT_BASE_BRANCH}"`,
		"config.json payload should preserve the ${GH_AW_INPUT_BASE_BRANCH} placeholder for runtime resolution")

	// The env var must be set in both the "Generate Safe Outputs Config" step and the
	// "Start MCP Gateway" step so the MCP container process can resolve the placeholder.
	assert.GreaterOrEqual(t, strings.Count(compiled, "GH_AW_INPUT_BASE_BRANCH: ${{ inputs.base_branch }}"), 2,
		"GH_AW_INPUT_BASE_BRANCH must appear in at least two step env blocks: Generate Safe Outputs Config and Start MCP Gateway")

	// The docker run command must include -e GH_AW_INPUT_BASE_BRANCH so the container
	// inherits the value from the runner step environment.
	assert.Contains(t, compiled, "-e GH_AW_INPUT_BASE_BRANCH",
		"MCP gateway docker run command must include -e GH_AW_INPUT_BASE_BRANCH so the containerised MCP server can resolve the placeholder")
}
