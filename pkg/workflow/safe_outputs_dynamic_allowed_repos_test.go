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
	assert.Contains(t, compiled, `"allowed_repos":"${GH_AW_INPUT_TARGET_REPO}"`,
		"config.json payload should use shell env var for allowed_repos")
	assert.Contains(t, compiled, `"allowed_base_branches":"${GH_AW_INPUT_BASE_BRANCH}"`,
		"config.json payload should use shell env var for allowed_base_branches")

	quotedHeredocPattern := regexp.MustCompile(`cat > "\$\{RUNNER_TEMP\}/gh-aw/safeoutputs/config\.json" << 'GH_AW_SAFE_OUTPUTS_CONFIG_[0-9a-f]{16}_EOF'`)
	assert.False(t, quotedHeredocPattern.MatchString(compiled),
		"Safe outputs config heredoc should not be single-quoted when using dynamic input expressions")

	unquotedHeredocPattern := regexp.MustCompile(`cat > "\$\{RUNNER_TEMP\}/gh-aw/safeoutputs/config\.json" << GH_AW_SAFE_OUTPUTS_CONFIG_[0-9a-f]{16}_EOF`)
	assert.True(t, unquotedHeredocPattern.MatchString(compiled),
		"Safe outputs config heredoc should be unquoted when dynamic input expressions are present")
	normalizedCompiled := strings.ReplaceAll(compiled, `\"`, `"`)
	assert.NotContains(t, normalizedCompiled, `"allowed_repos":["${{ inputs.target_repo }}"]`,
		"config.json payload should not keep unresolved workflow input expression")
}
