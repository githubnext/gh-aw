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

func TestActivationPermissionsPullRequestReactionRequiresPullRequestsWrite(t *testing.T) {
	tmpDir := testutil.TempDir(t, "activation-perms-pull-request-reaction")
	testFile := filepath.Join(tmpDir, "pull-request-reaction.md")
	testContent := `---
on:
  reaction: eyes
  status-comment: false
  pull_request:
    types: [opened]
engine: copilot
---

# Pull request reaction permissions
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "failed to compile workflow")

	lockContent, err := os.ReadFile(stringutil.MarkdownToLockFile(testFile))
	require.NoError(t, err, "failed to read generated lock file")

	activationJobSection := extractJobSection(string(lockContent), string(constants.ActivationJobName))
	assert.Contains(t, activationJobSection, "issues: write", "activation job should include issues: write for pull_request reactions")
	assert.Contains(t, activationJobSection, "pull-requests: write", "activation job should include pull-requests: write for pull_request reactions")
	assert.NotContains(t, activationJobSection, "discussions: write", "activation job should not include discussions: write for pull_request reactions")
}

func TestActivationPermissionsReactionPullRequestsDisabled(t *testing.T) {
	tmpDir := testutil.TempDir(t, "activation-perms-reaction-pr-disabled")
	testFile := filepath.Join(tmpDir, "reaction-pr-disabled.md")
	testContent := `---
on:
  reaction:
    type: eyes
    pull-requests: false
  status-comment: false
  pull_request:
    types: [opened]
engine: copilot
---

# Reaction pull_request target disabled
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "failed to compile workflow")

	lockContent, err := os.ReadFile(stringutil.MarkdownToLockFile(testFile))
	require.NoError(t, err, "failed to read generated lock file")

	activationJobSection := extractJobSection(string(lockContent), string(constants.ActivationJobName))
	assert.Contains(t, activationJobSection, "Add eyes reaction for immediate feedback", "activation job should include reaction step")
	assert.Contains(t, activationJobSection, "github.event_name == 'issues'", "reaction condition should still include issues when reaction.issues is enabled")
	assert.Contains(t, activationJobSection, "github.event_name == 'discussion'", "reaction condition should still include discussions when reaction.discussions is enabled")
	assert.NotContains(t, activationJobSection, "github.event_name == 'pull_request'", "reaction condition should exclude pull_request when reaction.pull-requests is false")
	assert.NotContains(t, activationJobSection, "github.event_name == 'pull_request_review_comment'", "reaction condition should exclude pull_request_review_comment when reaction.pull-requests is false")
	assert.NotContains(t, activationJobSection, "issues: write", "activation job should not include issues: write when pull request reactions are disabled")
	assert.NotContains(t, activationJobSection, "pull-requests: write", "activation job should not include pull-requests: write when pull request reactions are disabled")
	assert.NotContains(t, activationJobSection, "discussions: write", "activation job should not include discussions: write when no discussion triggers are configured")
}

func TestActivationPermissionsStatusCommentDiscussionsDisabled(t *testing.T) {
	tmpDir := testutil.TempDir(t, "activation-perms-status-comment-discussions-disabled")
	testFile := filepath.Join(tmpDir, "status-comment-discussions-disabled.md")
	testContent := `---
on:
  reaction: none
  status-comment:
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

func TestAddActivationInteractionPermissionsMapFallsBackOnInvalidOnYAML(t *testing.T) {
	permsMap := map[PermissionScope]PermissionLevel{}

	addActivationInteractionPermissionsMap(permsMap, "on: [",
		/* hasReaction */ true,
		/* reactionIncludesIssues */ true,
		/* reactionIncludesPullRequests */ true,
		/* reactionIncludesDiscussions */ true,
		/* hasStatusComment */ true,
		/* statusCommentIncludesIssues */ true,
		/* statusCommentIncludesPullRequests */ true,
		/* statusCommentIncludesDiscussions */ true,
	)

	assert.Equal(t, PermissionWrite, permsMap[PermissionIssues], "fallback should include issues:write")
	assert.Equal(t, PermissionWrite, permsMap[PermissionPullRequests], "fallback should include pull-requests:write")
	assert.Equal(t, PermissionWrite, permsMap[PermissionDiscussions], "fallback should include discussions:write when enabled")
}

func TestAddActivationInteractionPermissionsMapFallbackRespectsStatusCommentDiscussionsToggle(t *testing.T) {
	permsMap := map[PermissionScope]PermissionLevel{}

	addActivationInteractionPermissionsMap(permsMap, "name: no-on-key",
		/* hasReaction */ false,
		/* reactionIncludesIssues */ true,
		/* reactionIncludesPullRequests */ true,
		/* reactionIncludesDiscussions */ true,
		/* hasStatusComment */ true,
		/* statusCommentIncludesIssues */ true,
		/* statusCommentIncludesPullRequests */ true,
		/* statusCommentIncludesDiscussions */ false,
	)

	assert.Equal(t, PermissionWrite, permsMap[PermissionIssues], "fallback should include issues:write for status comments")
	_, hasPullRequests := permsMap[PermissionPullRequests]
	assert.False(t, hasPullRequests, "fallback should omit pull-requests:write when only status comments are enabled")
	_, hasDiscussions := permsMap[PermissionDiscussions]
	assert.False(t, hasDiscussions, "fallback should omit discussions:write when status-comment.discussions is false and reactions are disabled")
}

func TestActivationPermissionsStatusCommentIssuesDisabled(t *testing.T) {
	tmpDir := testutil.TempDir(t, "activation-perms-status-comment-issues-disabled")
	testFile := filepath.Join(tmpDir, "status-comment-issues-disabled.md")
	testContent := `---
on:
  reaction: none
  status-comment:
    issues: false
  issues:
    types: [opened]
  discussion:
    types: [created]
engine: copilot
---

# Status comment issues disabled
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "failed to compile workflow")

	lockContent, err := os.ReadFile(stringutil.MarkdownToLockFile(testFile))
	require.NoError(t, err, "failed to read generated lock file")

	activationJobSection := extractJobSection(string(lockContent), string(constants.ActivationJobName))
	assert.Contains(t, activationJobSection, "discussions: write", "activation job should include discussions: write for discussion status comments")
	assert.NotContains(t, activationJobSection, "issues: write", "activation job should not include issues: write when status-comment.issues is false and reactions are disabled")
	assert.NotContains(t, activationJobSection, "pull-requests: write", "activation job should not include pull-requests: write when status-comment.issues is false and reactions are disabled")
	assert.Contains(t, activationJobSection, "github.event_name == 'discussion'", "status comment condition should include discussion events when status-comment.issues is false")
	assert.NotContains(t, activationJobSection, "github.event_name == 'issues'", "status comment condition should not include issue events when status-comment.issues is false")
	assert.NotContains(t, activationJobSection, "github.event_name == 'issue_comment'", "status comment condition should not include issue_comment events when status-comment.issues is false")
}

func TestAddActivationInteractionPermissionsMapFallbackRespectsStatusCommentIssuesToggle(t *testing.T) {
	permsMap := map[PermissionScope]PermissionLevel{}

	addActivationInteractionPermissionsMap(permsMap, "name: no-on-key",
		/* hasReaction */ false,
		/* reactionIncludesIssues */ true,
		/* reactionIncludesPullRequests */ true,
		/* reactionIncludesDiscussions */ true,
		/* hasStatusComment */ true,
		/* statusCommentIncludesIssues */ false,
		/* statusCommentIncludesPullRequests */ false,
		/* statusCommentIncludesDiscussions */ true,
	)

	_, hasIssues := permsMap[PermissionIssues]
	_, hasPullRequests := permsMap[PermissionPullRequests]
	assert.False(t, hasIssues, "fallback should omit issues:write when status-comment.issues is false and reactions are disabled")
	assert.False(t, hasPullRequests, "fallback should omit pull-requests:write when status-comment.issues is false and reactions are disabled")
	assert.Equal(t, PermissionWrite, permsMap[PermissionDiscussions], "fallback should include discussions:write when status-comment.discussions is true")
}

func TestStatusCommentObjectRejectsAllTargetsDisabled(t *testing.T) {
	tmpDir := testutil.TempDir(t, "status-comment-object-all-disabled")
	testFile := filepath.Join(tmpDir, "status-comment-object-all-disabled.md")
	testContent := `---
on:
  status-comment:
    issues: false
    pull-requests: false
    discussions: false
  issues:
    types: [opened]
engine: copilot
---

# Invalid status comment object
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.Error(t, err, "compilation should fail when status-comment object disables all targets")
	assert.Contains(t, err.Error(), "status-comment object requires at least one target to be enabled", "error should explain invalid status-comment object configuration")
}

func TestActivationPermissionsStatusCommentPullRequestsDisabled(t *testing.T) {
	tmpDir := testutil.TempDir(t, "activation-perms-status-comment-pull-requests-disabled")
	testFile := filepath.Join(tmpDir, "status-comment-pull-requests-disabled.md")
	testContent := `---
on:
  reaction: none
  status-comment:
    pull-requests: false
  issues:
    types: [opened]
  pull_request:
    types: [opened]
engine: copilot
---

# Status comment pull-requests disabled
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err, "failed to write test workflow")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "failed to compile workflow")

	lockContent, err := os.ReadFile(stringutil.MarkdownToLockFile(testFile))
	require.NoError(t, err, "failed to read generated lock file")

	activationJobSection := extractJobSection(string(lockContent), string(constants.ActivationJobName))
	assert.Contains(t, activationJobSection, "issues: write", "activation job should include issues: write for issue status comments")
	assert.NotContains(t, activationJobSection, "pull-requests: write", "activation job should not include pull-requests: write when reactions are disabled")
	assert.Contains(t, activationJobSection, "github.event_name == 'issues'", "status comment condition should include issue events")
	assert.Contains(t, activationJobSection, "github.event_name == 'issue_comment'", "status comment condition should include issue_comment events")
	assert.NotContains(t, activationJobSection, "github.event_name == 'pull_request_review_comment'", "status comment condition should not include pull_request_review_comment when status-comment.pull-requests is false")
}

func TestAddActivationInteractionPermissionsMapFallbackRespectsStatusCommentPullRequestsToggle(t *testing.T) {
	permsMap := map[PermissionScope]PermissionLevel{}

	addActivationInteractionPermissionsMap(permsMap, "name: no-on-key",
		/* hasReaction */ false,
		/* reactionIncludesIssues */ true,
		/* reactionIncludesPullRequests */ true,
		/* reactionIncludesDiscussions */ true,
		/* hasStatusComment */ true,
		/* statusCommentIncludesIssues */ false,
		/* statusCommentIncludesPullRequests */ false,
		/* statusCommentIncludesDiscussions */ true,
	)

	_, hasIssues := permsMap[PermissionIssues]
	_, hasPullRequests := permsMap[PermissionPullRequests]
	assert.False(t, hasIssues, "fallback should omit issues:write when status-comment.issues and status-comment.pull-requests are false and reactions are disabled")
	assert.False(t, hasPullRequests, "fallback should omit pull-requests:write when reactions are disabled")
	assert.Equal(t, PermissionWrite, permsMap[PermissionDiscussions], "fallback should include discussions:write when status-comment.discussions is true")
}
