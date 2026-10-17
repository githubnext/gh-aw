//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"regexp"
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
