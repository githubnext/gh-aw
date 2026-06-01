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
		"Dynamic input env vars should be available anywhere the runtime needs to resolve placeholders in memory")
	assert.GreaterOrEqual(t, strings.Count(compiled, "GH_AW_INPUT_BASE_BRANCH: ${{ inputs.base_branch }}"), 2,
		"Dynamic input env vars should be available anywhere the runtime needs to resolve placeholders in memory")
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
	assert.GreaterOrEqual(t, strings.Count(compiled, "GH_AW_SECRET_WRITE_PROJECT_PAT: ${{ secrets.WRITE_PROJECT_PAT }}"), 2,
		"Secret env vars should be available anywhere the runtime needs to resolve the placeholder in memory")
	assert.Contains(t, compiled, `"github-token":"${GH_AW_SECRET_WRITE_PROJECT_PAT}"`,
		"config.json payload should preserve the prefixed secret placeholder instead of the secret value")

	quotedHeredocPattern := regexp.MustCompile(`cat > "\$\{RUNNER_TEMP\}/gh-aw/safeoutputs/config\.json" << 'GH_AW_SAFE_OUTPUTS_CONFIG_[0-9a-f]{16}_EOF'`)
	assert.True(t, quotedHeredocPattern.MatchString(compiled),
		"Safe outputs config heredoc should be single-quoted so secret placeholders are not expanded onto disk")

	unquotedHeredocPattern := regexp.MustCompile(`cat > "\$\{RUNNER_TEMP\}/gh-aw/safeoutputs/config\.json" << GH_AW_SAFE_OUTPUTS_CONFIG_[0-9a-f]{16}_EOF`)
	assert.False(t, unquotedHeredocPattern.MatchString(compiled),
		"Safe outputs config heredoc should not be unquoted when secrets are present")
}
