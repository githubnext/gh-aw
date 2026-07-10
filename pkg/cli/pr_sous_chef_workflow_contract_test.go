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
	assert.Contains(t, text, "Process at most 4 nudges per run.", "Workflow should cap nudges per run")
	assert.Contains(t, text, "add-comment:\n    max: 4", "Workflow should hard-cap add_comment safe-output calls to 4 per run")
	assert.Contains(t, text, "Prioritize which PRs to nudge, in this order:", "Workflow should define deterministic PR prioritization for nudges")
	assert.Contains(t, text, "stop creating new nudge comments once 4 PRs have been nudged", "Workflow should enforce a hard per-run nudge limit")
	assert.Contains(t, text, "Make at most 8 tool calls total.", "Sub-agent should have a hard tool-call budget")
	assert.Contains(t, text, "model: sonnet", "Sub-agent should use a Sonnet model alias")
	assert.Contains(t, text, "skip_reason: \"sub_agent_error\"", "Workflow should skip failed sub-agent responses without retry")
	assert.Contains(t, text, "eligible_count=", "fetch-prs step must export eligible_count output")
	assert.Contains(t, text, ".prs | length", "eligible_count should reflect the number of eligible PRs")
	assert.Contains(t, text, "dismiss-pull-request-review", "Workflow should configure the native safe-output path for dismissing stale github-actions reviews")
	assert.Contains(t, text, "safeoutputs dismiss_pull_request_review", "Workflow should call the native dismiss_pull_request_review safe-output tool")
	assert.Contains(t, text, "resolve-pull-request-review-thread", "Workflow should configure native safe-output support for resolving review threads")
	assert.Contains(t, text, "safeoutputs resolve_pull_request_review_thread", "Workflow should call the native resolve_pull_request_review_thread safe-output tool")
	assert.Contains(t, text, "resolve_review_threads", "Workflow should track and process resolvable review threads")
	assert.Contains(t, text, "skip_reason: \"resolve_review_thread_failed\"", "Workflow should explicitly record failed review-thread resolution attempts")
	assert.Contains(t, text, "all review threads on the PR are resolved", "Workflow must dismiss reviews only when all PR review threads are resolved")
	assert.Contains(t, text, "slash-command runs are acknowledgment nudges and must not perform automated review cleanup", "Workflow must require slash-command runs to skip dismissal")
	assert.Contains(t, text, "dismissed_reviews", "Noop summary must include dismissed_reviews counter")
	assert.Contains(t, text, "Slash-command acknowledgement requirement (mandatory)", "Workflow must define slash-command acknowledgement handling")
	assert.Contains(t, text, "you must always post a comment on the same PR as that triggering comment", "Workflow must require comment acknowledgement on slash-command PR comments")
	assert.Contains(t, text, "Do not skip this acknowledgement due to cooldown, pending checks, or duplicate-comment safeguards", "Workflow must make slash-command acknowledgement unconditional")
	assert.Contains(t, text, "now - 3600", "Workflow must define a 1-hour cutoff for long-running checks")
	assert.Contains(t, text, "fromdateiso8601", "Workflow must parse timestamps to implement long-running check guard")
	assert.Contains(t, text, ".startedAt // .createdAt) as $ts", "Workflow must bind timestamp (startedAt with createdAt fallback) to a variable for null-safe comparison")
	assert.Contains(t, text, "$ts == null or (($ts | fromdateiso8601) > $cutoff)", "Workflow must treat null timestamps as pending and ignore checks running > 1 hour")
	assert.Contains(t, text, "Long-running checks (running > 1 hour) are intentionally ignored", "Workflow instructions must document the long-running check exception")
}
