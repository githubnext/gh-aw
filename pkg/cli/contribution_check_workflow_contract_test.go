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

func TestContributionCheckWorkflowSafeOutputContract(t *testing.T) {
	t.Parallel()
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "contribution-check.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Should read contribution-check workflow")

	text := string(content)
	assert.Contains(t, text, "emit exactly", "Workflow must explicitly limit noop emission")
	assert.Contains(t, text, "one consolidated noop", "Workflow must require a single consolidated noop")

	assert.Contains(t, text, "temporary_id", "Workflow must mention temporary_id for summary issue linkage")
	assert.Contains(t, text, "aw_summary", "Workflow should provide a concrete temporary_id example")
	assert.Contains(t, text, "add_labels.item_number", "Workflow must mention add_labels item_number linkage")
	assert.Contains(t, text, "#<temporary_id>", "Workflow must describe item_number temporary_id reference format")
	assert.Contains(t, text, "Never emit `add_comment` without a numeric target field", "Workflow must forbid targetless add_comment items")
	assert.Contains(t, text, "\"issue_number\":35304", "Workflow should include a concrete add_comment issue_number example")
	assert.Contains(t, text, "model: claude-haiku-4.5", "Workflow should require small model for contribution-checker subagent calls")
}

func TestContributionCheckWorkflowAllowsRequiredShellCommands(t *testing.T) {
	t.Parallel()
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "contribution-check.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Should read contribution-check workflow")

	text := string(content)
	assert.Contains(t, text, `"git"`, "Workflow must allow git fetch/diff commands used by contribution-checker subagents")
	assert.Contains(t, text, `"jq *"`, "Workflow must allow jq payload construction for safeoutputs create_issue")

	lockPath := filepath.Join(repoRoot, ".github", "workflows", "contribution-check.lock.yml")
	lockContent, err := os.ReadFile(lockPath)
	require.NoError(t, err, "Should read compiled contribution-check workflow")

	lockText := string(lockContent)
	for _, token := range []string{
		"--allow-tool '\\''shell(git:*)'\\''",
		"--allow-tool '\\''shell(jq)'\\''",
		"--allow-tool '\\''shell(safeoutputs:*)'\\''",
	} {
		assert.Containsf(t, lockText, token, "Compiled workflow must contain %s", token)
	}
}
