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

// TestApplyPullRequestStackFilter_DualTriggerListFormat tests the common scenario where
// a workflow has both push and pull_request triggers using list syntax.
// This is the blast-radius case described in the bug: workflows with on: [push, pull_request]
// were getting startup_failure on push events due to unguarded arithmetic.
func TestApplyPullRequestStackFilter_DualTriggerListFormat(t *testing.T) {
	compiler := NewCompiler()
	workflowData := &WorkflowData{}
	frontmatter := map[string]any{
		"on": []any{"push", "pull_request"},
	}

	compiler.applyPullRequestStackFilter(workflowData, frontmatter)

	// The condition must be guarded with an event_name check so that it is
	// safe on push events (where github.event.pull_request is absent).
	assert.Contains(t, workflowData.If, "github.event_name != 'pull_request'",
		"condition must start with event_name guard to be safe on non-PR events")
	assert.Contains(t, workflowData.If, "github.event.pull_request.stack == null")
	assert.Contains(t, workflowData.If, "github.event.pull_request.stack.position == github.event.pull_request.stack.size")
	// Arithmetic operators (+, -, *, /) are not supported by GitHub Actions expressions.
	assert.NotContains(t, workflowData.If, "+", "job-level if must not contain arithmetic operators")
	assert.Empty(t, workflowData.PreSteps, "max-stack defaults to 1 — no PreStep needed")
}

// TestApplyPullRequestStackFilter_DualTriggerMapFormat tests the scenario where
// a workflow uses the map form of on: with both push and pull_request keys.
func TestApplyPullRequestStackFilter_DualTriggerMapFormat(t *testing.T) {
	compiler := NewCompiler()
	workflowData := &WorkflowData{}
	frontmatter := map[string]any{
		"on": map[string]any{
			"push": map[string]any{
				"branches": []any{"main"},
			},
			"pull_request": map[string]any{
				"types": []any{"opened", "synchronize"},
			},
		},
	}

	compiler.applyPullRequestStackFilter(workflowData, frontmatter)

	// The condition must be guarded with an event_name check so that it is
	// safe on push events (where github.event.pull_request is absent).
	assert.Contains(t, workflowData.If, "github.event_name != 'pull_request'",
		"condition must start with event_name guard to be safe on non-PR events")
	assert.Contains(t, workflowData.If, "github.event.pull_request.stack == null")
	assert.Contains(t, workflowData.If, "github.event.pull_request.stack.position == github.event.pull_request.stack.size")
	// Arithmetic operators (+, -, *, /) are not supported by GitHub Actions expressions.
	assert.NotContains(t, workflowData.If, "+", "job-level if must not contain arithmetic operators")
	assert.Empty(t, workflowData.PreSteps, "max-stack defaults to 1 — no PreStep needed")
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
