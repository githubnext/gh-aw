//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinterMinerWorkflowSubAgentModelContract(t *testing.T) {
	t.Parallel()
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "linter-miner.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Should read linter-miner workflow")

	text := string(content)
	assert.Contains(t, text, "## agent: `code-pattern-scanner`", "Workflow should define the code-pattern-scanner sub-agent")
	assert.Contains(t, text, "## agent: `linter-writer`", "Workflow should define the linter-writer sub-agent")
	assert.NotContains(t, text, "model: inherited", "Sub-agents should not use the unsupported model: inherited value")
}
