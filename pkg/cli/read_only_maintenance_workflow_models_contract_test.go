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

func TestReadOnlyMaintenanceWorkflowsUseHaikuModels(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	t.Run("pr-description-updater", func(t *testing.T) {
		workflowPath := filepath.Join(repoRoot, ".github", "workflows", "pr-description-caveman.md")
		content, err := os.ReadFile(workflowPath)
		require.NoError(t, err, "Should read pr-description-caveman workflow")

		text := string(content)
		assert.Contains(t, text, "engine:\n  id: copilot", "Workflow should declare a top-level Copilot engine block")
		assert.Contains(t, text, "\n  model: claude-haiku-4.5\n", "Workflow should pin a cheaper Haiku top-level model")
		assert.Contains(t, text, "## agent: `pr-description-synthesizer`", "Workflow should keep the dedicated synthesizer sub-agent")
		assert.Contains(t, text, "## agent: `pr-description-synthesizer`\n---\ndescription: Combines per-chunk analysis results and diff metadata into a final structured PR description optimised for agentic analysis.\nmodel: claude-haiku-4.5", "Workflow synthesizer should use a cheaper Haiku model")
		assert.NotContains(t, text, "model: large", "Workflow should no longer use the expensive large alias for PR description synthesis")
	})

	for _, workflowName := range []string{
		"impeccable-skills-reviewer.md",
		"mattpocock-skills-reviewer.md",
	} {
		t.Run(workflowName, func(t *testing.T) {
			workflowPath := filepath.Join(repoRoot, ".github", "workflows", workflowName)
			content, err := os.ReadFile(workflowPath)
			require.NoError(t, err, "Should read %s workflow", workflowName)

			text := string(content)
			assert.Contains(t, text, "model: claude-haiku-4.5", "Read-only maintenance workflow should pin a cheaper Haiku model")
			assert.NotContains(t, text, "model: claude-sonnet-4.6", "Read-only maintenance workflow should no longer use a Sonnet frontier model")
		})
	}
}
