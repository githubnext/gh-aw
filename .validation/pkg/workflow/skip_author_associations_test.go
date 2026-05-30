//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestExtractSkipAuthorAssociations(t *testing.T) {
	compiler := NewCompiler()

	frontmatter := map[string]any{
		"on": map[string]any{
			"skip-author-associations": map[string]any{
				"issue_comment":                "contributor",
				"pull_request_review_comment":  []any{"OWNER", "member", "owner", ""},
				"discussion_comment":           []string{"first_timer"},
				"pull_request_review":          "",
				"pull_request_review_threaded": []any{},
			},
		},
	}

	got := compiler.extractSkipAuthorAssociations(frontmatter)
	want := map[string][]string{
		"issue_comment":               {"CONTRIBUTOR"},
		"pull_request_review_comment": {"OWNER", "MEMBER"},
		"discussion_comment":          {"FIRST_TIMER"},
	}
	assert.Equal(t, want, got)
}

func TestSkipAuthorAssociationsCompilesToPreActivationIf(t *testing.T) {
	tmpDir := testutil.TempDir(t, "skip-author-associations-test")
	compiler := NewCompiler()

	workflowContent := `---
on:
  issue_comment:
    types: [created]
  pull_request_review_comment:
    types: [created]
  pull_request_review:
    types: [submitted]
  issues:
    types: [opened]
  pull_request:
    types: [opened]
  roles: all
  skip-author-associations:
    issue_comment: contributor
    pull_request_review_comment: [first_time_contributor, none]
    pull_request_review: collaborator
    issues: owner
    pull_request: member
engine: copilot
---

# Skip Author Associations Workflow
`

	workflowFile := filepath.Join(tmpDir, "skip-author-associations.md")
	err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
	require.NoError(t, err)

	err = compiler.CompileWorkflow(workflowFile)
	require.NoError(t, err)

	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err)

	lockContentStr := string(lockContent)
	preActivationSection := extractJobSection(lockContentStr, "pre_activation")
	require.NotEmpty(t, preActivationSection)

	assert.Contains(t, preActivationSection, "github.event.comment.author_association")
	assert.Contains(t, preActivationSection, "github.event.review.author_association")
	assert.Contains(t, preActivationSection, "github.event.issue.author_association")
	assert.Contains(t, preActivationSection, "github.event.pull_request.author_association")
	assert.Contains(t, preActivationSection, "github.event_name == 'issue_comment'")
	assert.Contains(t, preActivationSection, "github.event_name == 'pull_request_review_comment'")
	assert.Contains(t, preActivationSection, "github.event_name == 'pull_request_review'")
	assert.Contains(t, preActivationSection, "github.event_name == 'issues'")
	assert.Contains(t, preActivationSection, "github.event_name == 'pull_request'")
	assert.Contains(t, preActivationSection, "CONTRIBUTOR")
	assert.Contains(t, preActivationSection, "FIRST_TIME_CONTRIBUTOR")
	assert.Contains(t, preActivationSection, "NONE")
	assert.Contains(t, preActivationSection, "OWNER")
	assert.Contains(t, preActivationSection, "MEMBER")
	assert.Contains(t, preActivationSection, "COLLABORATOR")
	assert.Contains(t, preActivationSection, "!(")
	assert.Contains(t, preActivationSection, "||")
	assert.Contains(t, preActivationSection, "&&")

	assert.Contains(t, lockContentStr, "# skip-author-associations:")
	assert.Contains(t, lockContentStr, "    # issue_comment: contributor")
	assert.Contains(t, lockContentStr, "    # pull_request_review_comment:")
	assert.Contains(t, lockContentStr, "    # pull_request_review: collaborator")
	assert.Contains(t, lockContentStr, "    # issues: owner")
	assert.Contains(t, lockContentStr, "    # pull_request: member")
	assert.Contains(t, lockContentStr, "    # - first_time_contributor")
	assert.Contains(t, lockContentStr, "    # - none")
	assert.NotContains(t, lockContentStr, "skip-author-association:")

	var workflow map[string]any
	require.NoError(t, yaml.Unmarshal(lockContent, &workflow), "compiled lock file should be valid YAML")
}
