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

func TestAmbientContextWorkflowOptimizations(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	t.Run("mattpocock workflow uses gh-proxy and compact skill guidance", func(t *testing.T) {
		workflowPath := filepath.Join(repoRoot, ".github", "workflows", "mattpocock-skills-reviewer.md")
		content, readErr := os.ReadFile(workflowPath)
		require.NoError(t, readErr)
		text := string(content)

		assert.Contains(t, text, "mode: gh-proxy")
		assert.Contains(t, text, "Apply each selected skill according to its SKILL.md definition.")
		assert.NotContains(t, text, "**`/diagnose` guidance:**")
		assert.NotContains(t, text, "**`/tdd` guidance:**")
	})

	t.Run("contribution-check workflow references extracted agents", func(t *testing.T) {
		workflowPath := filepath.Join(repoRoot, ".github", "workflows", "contribution-check.md")
		content, readErr := os.ReadFile(workflowPath)
		require.NoError(t, readErr)
		text := string(content)

		assert.Contains(t, text, ".github/agents/report-formatter.agent.md")
		assert.Contains(t, text, ".github/agents/comment-dispatcher.agent.md")
		assert.NotContains(t, text, "## Important")
		assert.NotContains(t, text, "## agent: `report-formatter`")
		assert.NotContains(t, text, "## agent: `comment-dispatcher`")
	})

	t.Run("contribution helper agents exist", func(t *testing.T) {
		reportFormatterPath := filepath.Join(repoRoot, ".github", "agents", "report-formatter.agent.md")
		commentDispatcherPath := filepath.Join(repoRoot, ".github", "agents", "comment-dispatcher.agent.md")

		reportFormatter, readErr := os.ReadFile(reportFormatterPath)
		require.NoError(t, readErr)
		assert.Contains(t, string(reportFormatter), "Lead with the takeaway")

		commentDispatcher, readErr := os.ReadFile(commentDispatcherPath)
		require.NoError(t, readErr)
		assert.Contains(t, string(commentDispatcher), "\"issue_number\": <number>")
	})

	t.Run("test quality sentinel uses compact report skeleton", func(t *testing.T) {
		workflowPath := filepath.Join(repoRoot, ".github", "workflows", "test-quality-sentinel.md")
		content, readErr := os.ReadFile(workflowPath)
		require.NoError(t, readErr)
		text := string(content)

		assert.Contains(t, text, "Use this compact skeleton and fill in real values:")
		assert.NotContains(t, text, "📖 Understanding Test Classifications")
	})
}
