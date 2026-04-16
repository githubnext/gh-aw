//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivationPermissionsIssueOnlyReactionAndStatusComment(t *testing.T) {
	tmpDir := testutil.TempDir(t, "activation-perms-issues-only")
	testFile := filepath.Join(tmpDir, "issue-only.md")
	testContent := `---
on:
  reaction: eyes
  status-comment: true
  issues:
    types: [opened]
engine: copilot
safe-outputs:
  add-comment:
---

# Issue-only activation permissions
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "failed to compile workflow")

	lockContent, err := os.ReadFile(stringutil.MarkdownToLockFile(testFile))
	require.NoError(t, err, "failed to read generated lock file")

	activationJobSection := extractJobSection(string(lockContent), string(constants.ActivationJobName))
	assert.Contains(t, activationJobSection, "issues: write", "activation job should include issues: write for issue trigger reactions/comments")
	assert.NotContains(t, activationJobSection, "pull-requests: write", "activation job should not include pull-requests: write for issue-only triggers")
	assert.NotContains(t, activationJobSection, "discussions: write", "activation job should not include discussions: write for issue-only triggers")
}

func TestActivationPermissionsPRReviewReactionOnly(t *testing.T) {
	tmpDir := testutil.TempDir(t, "activation-perms-pr-review")
	testFile := filepath.Join(tmpDir, "pr-review-reaction.md")
	testContent := `---
on:
  reaction: eyes
  status-comment: false
  pull_request_review_comment:
    types: [created]
engine: copilot
---

# PR review reaction permissions
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "failed to compile workflow")

	lockContent, err := os.ReadFile(stringutil.MarkdownToLockFile(testFile))
	require.NoError(t, err, "failed to read generated lock file")

	activationJobSection := extractJobSection(string(lockContent), string(constants.ActivationJobName))
	assert.Contains(t, activationJobSection, "pull-requests: write", "activation job should include pull-requests: write for PR review comment reactions")
	assert.NotContains(t, activationJobSection, "issues: write", "activation job should not include issues: write for PR review comment reactions without status comments")
	assert.NotContains(t, activationJobSection, "discussions: write", "activation job should not include discussions: write for PR review comment reactions")
}

func TestActivationPermissionsStatusCommentDiscussionsDisabled(t *testing.T) {
	tmpDir := testutil.TempDir(t, "activation-perms-status-comment-discussions-disabled")
	testFile := filepath.Join(tmpDir, "status-comment-discussions-disabled.md")
	testContent := `---
on:
  reaction: none
  status-comment:
    enabled: true
    discussions: false
  discussion:
    types: [created]
engine: copilot
---

# Status comment discussions disabled
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "failed to compile workflow")

	lockContent, err := os.ReadFile(stringutil.MarkdownToLockFile(testFile))
	require.NoError(t, err, "failed to read generated lock file")

	activationJobSection := extractJobSection(string(lockContent), string(constants.ActivationJobName))
	assert.NotContains(t, activationJobSection, "discussions: write", "activation job should not include discussions: write when status-comment.discussions is false")
	assert.Contains(t, activationJobSection, "Add comment with workflow run link", "activation job should still include status comment step when enabled")
	assert.Contains(t, activationJobSection, "github.event_name == 'issues'", "status comment condition should still include issue events")
	assert.Contains(t, activationJobSection, "github.event_name == 'issue_comment'", "status comment condition should still include issue_comment events")
	assert.NotContains(t, activationJobSection, "github.event_name == 'discussion'", "status comment condition should not include discussion events when status-comment.discussions is false")
	assert.NotContains(t, activationJobSection, "github.event_name == 'discussion_comment'", "status comment condition should not include discussion_comment events when status-comment.discussions is false")
}
