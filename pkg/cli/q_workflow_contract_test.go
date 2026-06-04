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

func TestQWorkflowSafeOutputContract(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "q.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Should read q workflow")

	text := string(content)
	assert.Contains(t, text, "Permission denied and could not request permission from user", "Q workflow must handle non-interactive permission-denied failures")
	assert.Contains(t, text, "call `report_incomplete`", "Q workflow must require report_incomplete for blocked execution")
	assert.Contains(t, text, "Every run must end with at least one safe-output call", "Q workflow must require at least one safe output call")
	assert.Contains(t, text, "`create-pull-request`", "Q workflow safe-output list must include create-pull-request")
	assert.Contains(t, text, "`add-comment`", "Q workflow safe-output list must include add-comment")
	assert.Contains(t, text, "`add-labels`", "Q workflow safe-output list must include add-labels")
	assert.Contains(t, text, "`noop`", "Q workflow safe-output list must include noop")
	assert.Contains(t, text, "`report_incomplete`", "Q workflow safe-output list must include report_incomplete")
}
