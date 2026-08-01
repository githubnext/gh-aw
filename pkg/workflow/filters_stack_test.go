package workflow

import (
	"strings"
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
	// max-stack: 1 (default) uses equality (== size) rather than arithmetic (+ N > size)
	// GitHub Actions expressions do not support arithmetic operators.
	assert.Contains(t, workflowData.If, "github.event.pull_request.stack.position == github.event.pull_request.stack.size")
	assert.NotContains(t, workflowData.If, "+", "job-level if must not contain arithmetic operators")
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

	// For max-stack: N > 1, the arithmetic check must be in a PreStep, not in job-level if:
	assert.Empty(t, workflowData.If, "job-level if should not contain arithmetic for max-stack > 1")
	assert.Contains(t, workflowData.PreSteps, "Stack position gate (max-stack: 3)")
	assert.Contains(t, workflowData.PreSteps, "max_stack=3")
	assert.Contains(t, workflowData.PreSteps, "STACK_POSITION + max_stack <= STACK_SIZE")
	assert.NotContains(t, workflowData.If, "+", "job-level if must not contain arithmetic operators")
}

func TestApplyPullRequestStackFilter_ConfiguredMaxStackInArrayTriggerForm(t *testing.T) {
	compiler := NewCompiler()
	workflowData := &WorkflowData{}
	frontmatter := map[string]any{
		"on": []any{
			map[string]any{
				"pull_request": map[string]any{
					"types":     []any{"opened"},
					"max-stack": 2,
				},
			},
		},
	}

	compiler.applyPullRequestStackFilter(workflowData, frontmatter)

	assert.Empty(t, workflowData.If, "job-level if should not contain arithmetic for max-stack > 1")
	assert.Contains(t, workflowData.PreSteps, "Stack position gate (max-stack: 2)")
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
	assert.Empty(t, workflowData.PreSteps)
}

func TestApplyPullRequestStackFilter_SimplePullRequestTrigger(t *testing.T) {
	compiler := NewCompiler()
	workflowData := &WorkflowData{}
	frontmatter := map[string]any{
		"on": "pull_request",
	}

	compiler.applyPullRequestStackFilter(workflowData, frontmatter)

	// String trigger form — max-stack defaults to 1, so equality expression is used
	assert.Contains(t, workflowData.If, "github.event.pull_request.stack.position == github.event.pull_request.stack.size")
	assert.NotContains(t, workflowData.If, "+", "job-level if must not contain arithmetic operators")
}

func TestApplyPullRequestStackFilter_SimplePullRequestReviewTrigger(t *testing.T) {
	compiler := NewCompiler()
	workflowData := &WorkflowData{}
	frontmatter := map[string]any{
		"on": "pull_request_review",
	}

	compiler.applyPullRequestStackFilter(workflowData, frontmatter)

	assert.Contains(t, workflowData.If, "github.event_name != 'pull_request_review'")
	assert.Contains(t, workflowData.If, "github.event.pull_request.stack.position == github.event.pull_request.stack.size")
	assert.NotContains(t, workflowData.If, "+", "job-level if must not contain arithmetic operators")
}

func TestApplyPullRequestStackFilter_PullRequestReviewConfiguredMaxStack(t *testing.T) {
	compiler := NewCompiler()
	workflowData := &WorkflowData{}
	frontmatter := map[string]any{
		"on": map[string]any{
			"pull_request_review": map[string]any{
				"types":     []any{"submitted"},
				"max-stack": 2,
			},
		},
	}

	compiler.applyPullRequestStackFilter(workflowData, frontmatter)

	assert.Empty(t, workflowData.If, "job-level if should not contain arithmetic for max-stack > 1")
	assert.Contains(t, workflowData.PreSteps, "Stack position gate (max-stack: 2)")
	assert.Contains(t, workflowData.PreSteps, "max_stack=2")
	// Intermediate positions are gated via inequality: skip when position + N <= size
	assert.Contains(t, workflowData.PreSteps, "STACK_POSITION + max_stack <= STACK_SIZE")
	assert.Contains(t, workflowData.PreSteps, "github.event_name == 'pull_request_review'")
	assert.NotContains(t, workflowData.If, "+", "job-level if must not contain arithmetic operators")
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
	assert.Empty(t, workflowData.PreSteps)
}

func TestApplyPullRequestStackFilter_ExistingPreStepsAppended(t *testing.T) {
	compiler := NewCompiler()
	workflowData := &WorkflowData{
		PreSteps: "pre-steps:\n- name: existing-step\n  run: |\n    echo hello\n",
	}
	frontmatter := map[string]any{
		"on": map[string]any{
			"pull_request": map[string]any{
				"types":     []any{"opened"},
				"max-stack": 2,
			},
		},
	}

	compiler.applyPullRequestStackFilter(workflowData, frontmatter)

	assert.Contains(t, workflowData.PreSteps, "existing-step")
	assert.Contains(t, workflowData.PreSteps, "Stack position gate (max-stack: 2)")
	// Both steps should be present
	idx1 := strings.Index(workflowData.PreSteps, "existing-step")
	idx2 := strings.Index(workflowData.PreSteps, "Stack position gate")
	assert.Greater(t, idx2, idx1, "stack gate step should appear after existing pre-steps")
}
