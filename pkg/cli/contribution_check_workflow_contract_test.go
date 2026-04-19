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
}
