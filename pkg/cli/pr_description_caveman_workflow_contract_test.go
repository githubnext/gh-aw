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

func TestPRDescriptionCavemanWorkflowSubAgentModelContract(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "pr-description-caveman.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Should read pr-description-caveman workflow")

	text := string(content)
	assert.Contains(t, text, "## agent: `chunk-analyzer`", "Workflow should define the chunk-analyzer sub-agent")
	assert.Contains(t, text, "model: claude-haiku-4.5", "chunk-analyzer sub-agent should pin a supported Haiku model")
	assert.NotContains(t, text, "model: small", "chunk-analyzer sub-agent should not use the unresolved 'small' alias")
}
