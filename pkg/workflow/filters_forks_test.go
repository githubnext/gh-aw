//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyPullRequestForkFilter(t *testing.T) {
	const sameRepo = "github.event_name != 'pull_request' || github.event.pull_request.head.repo.id == github.repository_id"
	tests := []struct {
		name     string
		pr       map[string]any
		expected string
	}{
		{"default", map[string]any{}, sameRepo},
		{"empty patterns", map[string]any{"forks": []any{}}, sameRepo},
		{"wildcard string", map[string]any{"forks": "*"}, ""},
		{"wildcard list", map[string]any{"forks": []any{"trusted/repo", "*"}}, ""},
		{
			"repository pattern",
			map[string]any{"forks": "trusted/*"},
			sameRepo + " || startsWith(github.event.pull_request.head.repo.full_name, 'trusted/')",
		},
		{
			"trusted associations",
			map[string]any{"forks": map[string]any{"author-association": []any{"OWNER", "MEMBER", "COLLABORATOR"}}},
			sameRepo + ` || contains(fromJSON('["OWNER","MEMBER","COLLABORATOR"]'), github.event.pull_request.author_association)`,
		},
		{
			"member excluded",
			map[string]any{"forks": map[string]any{"author-association": []any{"OWNER", "COLLABORATOR"}}},
			sameRepo + ` || contains(fromJSON('["OWNER","COLLABORATOR"]'), github.event.pull_request.author_association)`,
		},
		{
			"empty associations",
			map[string]any{"forks": map[string]any{"author-association": []any{}}},
			sameRepo,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, existing := range []string{"", "github.actor != 'dependabot[bot]'"} {
				data := &WorkflowData{If: existing}
				NewCompiler().applyPullRequestForkFilter(data, map[string]any{
					"on": map[string]any{"pull_request": tt.pr, "push": nil, "workflow_dispatch": nil},
				})
				expected := tt.expected
				if existing != "" {
					expected = existing
					if tt.expected != "" {
						expected = "(" + existing + ") && (" + tt.expected + ")"
					}
				}
				assert.Equal(t, expected, data.If)
			}
		})
	}
}

func TestPullRequestForkAuthorAssociationCompilation(t *testing.T) {
	for _, associations := range []string{
		"author-association: [OWNER, MEMBER, COLLABORATOR]",
		"author-association:\n        - OWNER\n        - MEMBER\n        - COLLABORATOR",
	} {
		t.Run(associations, func(t *testing.T) {
			workflowFile := filepath.Join(testutil.TempDir(t, "fork-associations"), "forks.md")
			require.NoError(t, os.WriteFile(workflowFile, []byte(`---
on:
  pull_request:
    forks:
      `+associations+`
    types: [opened, synchronize]
    branches: [main]
  push:
    branches: [main]
  workflow_dispatch:
if: github.actor != 'dependabot[bot]'
engine: copilot
---
# Trusted fork authors
`), 0644))
			require.NoError(t, NewCompiler().CompileWorkflow(workflowFile))
			content, err := os.ReadFile(filepath.Join(filepath.Dir(workflowFile), "forks.lock.yml"))
			require.NoError(t, err)

			var compiled map[string]any
			require.NoError(t, yaml.Unmarshal(content, &compiled))
			on := compiled["on"].(map[string]any)
			pr := on["pull_request"].(map[string]any)
			assert.NotContains(t, pr, "forks")
			assert.NotContains(t, pr, "author-association")
			assert.Equal(t, []any{"opened", "synchronize"}, pr["types"])
			assert.Equal(t, []any{"main"}, pr["branches"])
			assert.Contains(t, on, "push")
			assert.Contains(t, on, "workflow_dispatch")
			assert.Contains(t, string(content), "# author-association:")

			jobs := compiled["jobs"].(map[string]any)
			activation := jobs["activation"].(map[string]any)
			condition := activation["if"].(string)
			assert.Contains(t, condition, "github.actor != 'dependabot[bot]'")
			assert.Contains(t, condition, "github.event_name != 'pull_request' || github.event.pull_request.head.repo.id == github.repository_id")
			assert.Contains(t, condition, `|| contains(fromJSON('["OWNER","MEMBER","COLLABORATOR"]'), github.event.pull_request.author_association)`)
			assert.Contains(t, string(content), "github.event.pull_request.base.sha || github.sha")
		})
	}
}
