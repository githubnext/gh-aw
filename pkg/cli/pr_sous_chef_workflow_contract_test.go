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

func TestPRSousChefWorkflowAddCommentTargetContract(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "pr-sous-chef.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Should read pr-sous-chef workflow")

	text := string(content)
	assert.Contains(t, text, "Every `add_comment` must include `pr_number`", "Workflow must require explicit pr_number in add_comment")
	assert.Contains(t, text, "Never emit `add_comment` without a numeric target field", "Workflow must forbid targetless add_comment items")
	assert.Contains(t, text, "pr_number 12345", "Workflow should include a concrete add_comment pr_number example")
	assert.Contains(t, text, "include an explicit unresolved-reviews list", "Workflow should require explicit unresolved review listing in nudge comments")
	assert.Contains(t, text, "Process at most **5 PRs** per run.", "Workflow should cap per-run PR processing to 5")
	assert.Contains(t, text, "Make at most 8 tool calls total.", "Sub-agent should have a hard tool-call budget")
	assert.Contains(t, text, "model: claude-sonnet-4.5", "Sub-agent should use a Sonnet model")
	assert.Contains(t, text, "skip_reason: \"sub_agent_error\"", "Workflow should skip failed sub-agent responses without retry")
	assert.Contains(t, text, "eligible_count=", "fetch-prs step must export eligible_count output")
	assert.Contains(t, text, ".prs | length", "eligible_count should reflect the number of eligible PRs")
	assert.Contains(t, text, "dismiss_github_actions_reviews", "Workflow should provide a safe-output path for dismissing stale github-actions reviews")
	assert.Contains(t, text, "missing_tool --tool \"dismiss_pull_request_review\"", "Workflow should request a dedicated dismiss-review safe-output when unavailable")
}
