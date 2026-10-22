//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// explicitMainAgentModelPattern matches a top-level, provider-qualified model
// declaration (for example "model: openai/gpt-5.4") in the workflow frontmatter.
var explicitMainAgentModelPattern = regexp.MustCompile(`(?m)^model: \S+/\S+$`)

func TestPRCodeQualityReviewerWorkflowSubAgentModelContract(t *testing.T) {
	t.Parallel()
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "pr-code-quality-reviewer.md")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Should read pr-code-quality-reviewer workflow")

	text := string(content)
	assert.Regexp(t, explicitMainAgentModelPattern, text, "Main agent should use an explicit provider-qualified model")
	assert.Contains(t, text, "## agent: `grumpy-coder`", "Workflow should define the grumpy-coder sub-agent")
	assert.Contains(t, text, "model: small", "Sub-agent should use the portable small alias")
	assert.NotContains(t, text, "model: inherited", "Sub-agent should not inherit an unsupported tier-specific model")
}

func TestPRCodeQualityReviewerWorkflowPublicationContract(t *testing.T) {
	t.Parallel()
	repoRoot, err := gitutil.FindGitRoot()
	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "pr-code-quality-reviewer.md"))
	require.NoError(t, err)

	text := string(content)
	assert.Contains(t, text, "Keep publication commands stdout-only")
	assert.Contains(t, text, "Do not write temporary payload files, use shell redirection or heredocs")
	assert.Contains(t, text, "Send each safe-output command separately from analysis commands")
	assert.Contains(t, text, "without widening permissions")
	assert.Contains(t, text, "emit `report_incomplete` through the available safe-output transport")
	assert.Contains(t, text, "do not call `noop` or claim the review was published")
	assert.Contains(t, text, `| safeoutputs create_pull_request_review_comment .`)
	assert.Contains(t, text, `| safeoutputs submit_pull_request_review .`)
	assert.Contains(t, text, `| safeoutputs report_incomplete .`)
	assert.Contains(t, text, "Make at most one corrected publication attempt")
	assert.Contains(t, text, "before exhausting the five-denial budget")
	assert.Contains(t, text, "no terminal safe output was emitted")
	assert.NotContains(t, text, "name the exact allowlist entry needed: `tools.bash: - safeoutputs`")
}

func TestPRCodeQualityReviewerWorkflowToolContract(t *testing.T) {
	t.Parallel()
	repoRoot, err := gitutil.FindGitRoot()
	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "pr-code-quality-reviewer.md"))
	require.NoError(t, err)

	text := string(content)
	assert.NotContains(t, text, "command -v safeoutputs", "Publication must not depend on an unallowed availability probe")
	assert.NotContains(t, text, "    - command:*\n")
	assert.Contains(t, text, "do not prepend availability probes or other commands")
	assert.Contains(t, text, "run exactly `wc -c /tmp/gh-aw/agent/pr-review-comments.json`")
	assert.Contains(t, text, "Use `glob` to check whether")
	assert.Contains(t, text, "do not execute repository code, tests, package managers, or build commands")
}

func TestPRCodeQualityReviewerWorkflowReviewSequencingContract(t *testing.T) {
	t.Parallel()
	repoRoot, err := gitutil.FindGitRoot()
	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "pr-code-quality-reviewer.md"))
	require.NoError(t, err)
	text := string(content)

	// A pending specialist must be joined or explicitly excluded before any writes.
	_, adjudication, ok := strings.Cut(text, "### Step 3: Judge Agent-to-Agent Findings")
	require.True(t, ok)
	barrier, publication, ok := strings.Cut(adjudication, "### Step 4: Write Review Comments")
	require.True(t, ok)
	assert.Contains(t, barrier, "use a bounded wait for completion")
	assert.Contains(t, barrier, "ending no later than 10 minutes elapsed")
	assert.Contains(t, barrier, "`completed`")
	assert.Contains(t, barrier, "`unavailable`")
	assert.Contains(t, barrier, "Empty output means no specialist findings only after completion")
	assert.Contains(t, barrier, "launch/wait fails, output is invalid, or the 10-minute deadline expires")
	assert.Contains(t, barrier, "note the concrete limitation in the overall review")
	assert.Contains(t, barrier, "Do not enter Step 4 or Step 5 until")
	assert.Contains(t, barrier, "freeze the final finding set and choose the overall verdict")
	assert.Contains(t, barrier, "Late specialist output must not reopen adjudication")
	assert.NotContains(t, text, "if it has not yet responded, continue without it")
	assert.NotContains(t, text, "do not block if it has not responded")

	comments, submission, ok := strings.Cut(publication, "### Step 5: Submit the Overall Review")
	require.True(t, ok)
	assert.Contains(t, comments, "Publish only the frozen, adjudicated findings")
	assert.Contains(t, comments, "before the single overall review")
	assert.Contains(t, submission, "Submit exactly one overall review, only after Step 3's completion barrier")
	assert.Contains(t, submission, "never submit a provisional review or attempt a second review")
	assert.Contains(t, submission, "The deadline is not permission to bypass the completion barrier")
	assert.Contains(t, submission, "After a successful submission, publication is closed")
	assert.Contains(t, submission, "or emit `report_incomplete` merely because late specialist output arrived")

	assert.Contains(t, text, "submit-pull-request-review:\n    max: 1\n    allowed-events: [COMMENT, REQUEST_CHANGES]")
	_, specialist, ok := strings.Cut(text, "## agent: `grumpy-coder`")
	require.True(t, ok)
	assert.Contains(t, specialist, "do not publish comments, submit reviews, or invoke safe outputs")
}
