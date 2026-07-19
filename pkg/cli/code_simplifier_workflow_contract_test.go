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

func TestCodeSimplifierWorkflowSubAgentModelContract(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "code-simplifier.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Should read code-simplifier workflow")

	text := string(content)
	assert.Contains(t, text, "## agent: `scope-filter`", "Workflow should define the scope-filter sub-agent")
	assert.Contains(t, text, "## agent: `simplification-scout`", "Workflow should define the simplification-scout sub-agent")
	assert.Contains(t, text, "model: claude-haiku-4.5", "Sub-agents should pin a supported Haiku model")
	assert.NotContains(t, text, "model: small", "Sub-agents should not use the unresolved 'small' alias")
}
