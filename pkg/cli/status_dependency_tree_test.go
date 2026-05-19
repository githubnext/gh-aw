package cli

import (
	"strings"
	"testing"
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

	if len(got) != len(want) {
		t.Fatalf("extractWorkflowDependencies() len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("extractWorkflowDependencies()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
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
		if !strings.Contains(result, part) {
			t.Fatalf("expected dependency tree to contain %q, got:\n%s", part, result)
		}
	}
}

func TestRenderWorkflowDependencyTree_Empty(t *testing.T) {
	statuses := []WorkflowStatus{{Workflow: "standalone"}}
	if result := renderWorkflowDependencyTree(statuses); result != "" {
		t.Fatalf("expected empty dependency tree output, got %q", result)
	}
}
