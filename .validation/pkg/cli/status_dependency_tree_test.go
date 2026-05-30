//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractWorkflowDependencies(t *testing.T) {
	frontmatter := map[string]any{
		"imports": []any{
			"shared/base.md#section",
			map[string]any{"uses": "owner/repo/.github/workflows/common.md@main"},
		},
	}
	content := `
@include local/helpers.md#part
@import shared/base.md
`

	got := extractWorkflowDependencies(content, frontmatter)
	want := []string{
		"local/helpers.md",
		"owner/repo/.github/workflows/common.md@main",
		"shared/base.md",
	}

	assert.Len(t, got, len(want), "dependency count should match expected unique set")
	for i := range want {
		assert.Equal(t, want[i], got[i], "dependency should match normalized and sorted value")
	}
}

func TestExtractWorkflowDependencies_ImportsObjectAW(t *testing.T) {
	frontmatter := map[string]any{
		"imports": map[string]any{
			"aw": []any{
				map[string]any{"path": "shared/one.md#setup"},
				map[string]any{"uses": "shared/two.md"},
				"shared/three.md",
			},
		},
	}

	got := extractWorkflowDependencies("", frontmatter)
	want := []string{"shared/one.md", "shared/three.md", "shared/two.md"}
	assert.Equal(t, want, got)
}

func TestRenderWorkflowDependencyTree(t *testing.T) {
	statuses := []WorkflowStatus{
		{
			Workflow:     "main-workflow",
			Dependencies: []string{"shared/base.md", "local/helpers.md"},
		},
	}

	result := renderWorkflowDependencyTree(statuses)
	expected := []string{"Workflow Dependencies", "main-workflow", "shared/base.md", "local/helpers.md"}
	for _, part := range expected {
		assert.Contains(t, result, part, "dependency tree should include expected node")
	}
}

func TestRenderWorkflowDependencyTree_Empty(t *testing.T) {
	statuses := []WorkflowStatus{{Workflow: "standalone"}}
	assert.Empty(t, renderWorkflowDependencyTree(statuses), "dependency tree should be empty when no dependencies exist")
}
