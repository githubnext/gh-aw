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
	assert.Contains(t, text, "\"pr_number\":12345", "Workflow should include a concrete add_comment pr_number example")
}
