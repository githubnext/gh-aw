//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEmitGitHubLockdownGuardPolicyWarningExactText validates that the
// compile-time lockdown/guard-policy conflict warning matches, byte-for-byte,
// the example message documented in
// scratchpad/github-mcp-access-control-specification.md §9.5.2.
func TestEmitGitHubLockdownGuardPolicyWarningExactText(t *testing.T) {
	tools := NewTools(map[string]any{
		"github": map[string]any{
			"lockdown":      true,
			"allowed-repos": "all",
			"min-integrity": "approved",
		},
	})
	require.NoError(t, validateGitHubGuardPolicy(tools, "test-workflow"))

	compiler := NewCompiler()
	stderrOutput := captureStderr(func() {
		emitGitHubLockdownGuardPolicyWarning(compiler, tools, "test-workflow.md")
	})

	const specExampleMessage = `'tools.github.lockdown: true' is set; GitHub guard policy fields ('allowed-repos', 'min-integrity', 'blocked-users', 'trusted-users', 'approval-labels') will be ignored.
Guard policies are only evaluated when lockdown is not active.`

	assert.Equal(t, specExampleMessage, githubLockdownGuardPolicyWarningMessage,
		"the implementation warning message must stay byte-identical to the §9.5.2 example")
	assert.Contains(t, stderrOutput, specExampleMessage,
		"the emitted warning must contain the exact §9.5.2 example message")
}
