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

func TestDeepReportWorkflowIssueResolutionContract(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "deep-report.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Should read deep-report workflow")

	text := string(content)
	assert.Contains(t, text, "expires: false", "Deep Report issues must not close before their fixes are verified")
	assert.Contains(t, text, "implementing pull request was merged", "Workflow must require merged implementation evidence")
	assert.Contains(t, text, "Re-check the current artifact or rendered state", "Workflow must require current-state verification")
	assert.Contains(t, text, "treat the problem as unresolved", "Workflow must re-file failures that were closed without a durable fix")
}
