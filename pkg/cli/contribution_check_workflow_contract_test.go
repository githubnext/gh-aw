//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContributionCheckWorkflowSafeOutputContract(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "contribution-check.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Should read contribution-check workflow")

	text := string(content)
	assert.True(t, strings.Contains(text, "emit exactly") && strings.Contains(text, "one consolidated noop"), "Workflow must instruct single consolidated noop emission")

	assert.Contains(t, text, "set a `temporary_id` (for example `aw_summary`)", "Workflow must instruct create_issue temporary_id for summary issue")

	assert.Contains(t, text, "set `add_labels.item_number` to `#<temporary_id>`", "Workflow must instruct add_labels item_number linkage to temporary_id")
}
