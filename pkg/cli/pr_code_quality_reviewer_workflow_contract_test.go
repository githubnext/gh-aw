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

func TestPRCodeQualityReviewerWorkflowSubAgentModelContract(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "pr-code-quality-reviewer.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Should read pr-code-quality-reviewer workflow")

	text := string(content)
	assert.Contains(t, text, "## agent: `grumpy-coder`", "Workflow should define the grumpy-coder sub-agent")
	assert.Contains(t, text, "model: claude-haiku-4.5", "Sub-agent should pin a supported Haiku model")
	assert.NotContains(t, text, "model: inherited", "Sub-agent should not inherit an unsupported tier-specific model")
}
