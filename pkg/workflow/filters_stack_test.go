package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyPullRequestStackFilter_DefaultMaxStack(t *testing.T) {
	compiler := NewCompiler()
	workflowData := &WorkflowData{}
	frontmatter := map[string]any{
		"on": map[string]any{
			"pull_request": map[string]any{
				"types": []any{"opened"},
			},
		},
	}

	compiler.applyPullRequestStackFilter(workflowData, frontmatter)

	assert.Contains(t, workflowData.If, "github.event.pull_request.stack == null")
	assert.Contains(t, workflowData.If, "github.event.pull_request.stack.position + 1 > github.event.pull_request.stack.size")
}

func TestApplyPullRequestStackFilter_ConfiguredMaxStack(t *testing.T) {
	compiler := NewCompiler()
	workflowData := &WorkflowData{}
	frontmatter := map[string]any{
		"on": map[string]any{
			"pull_request": map[string]any{
				"types":     []any{"opened"},
				"max-stack": 3,
			},
		},
	}

	compiler.applyPullRequestStackFilter(workflowData, frontmatter)

	assert.Contains(t, workflowData.If, "github.event.pull_request.stack.position + 3 > github.event.pull_request.stack.size")
}

func TestApplyPullRequestStackFilter_Disabled(t *testing.T) {
	compiler := NewCompiler()
	workflowData := &WorkflowData{If: "github.actor != 'dependabot[bot]'"}
	frontmatter := map[string]any{
		"on": map[string]any{
			"pull_request": map[string]any{
				"types":     []any{"opened"},
				"max-stack": -1,
			},
		},
	}

	compiler.applyPullRequestStackFilter(workflowData, frontmatter)

	assert.Equal(t, "github.actor != 'dependabot[bot]'", workflowData.If)
}

func TestApplyPullRequestStackFilter_SimplePullRequestTrigger(t *testing.T) {
	compiler := NewCompiler()
	workflowData := &WorkflowData{}
	frontmatter := map[string]any{
		"on": "pull_request",
	}

	compiler.applyPullRequestStackFilter(workflowData, frontmatter)

	assert.Contains(t, workflowData.If, "github.event.pull_request.stack.position + 1 > github.event.pull_request.stack.size")
}

func TestApplyPullRequestStackFilter_NoPullRequestTrigger(t *testing.T) {
	compiler := NewCompiler()
	workflowData := &WorkflowData{If: "github.actor != 'dependabot[bot]'"}
	frontmatter := map[string]any{
		"on": map[string]any{
			"push": map[string]any{
				"branches": []any{"main"},
			},
		},
	}

	compiler.applyPullRequestStackFilter(workflowData, frontmatter)

	assert.Equal(t, "github.actor != 'dependabot[bot]'", workflowData.If)
}
