//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file formalizes the §12 Compliance Test Matrix Gap Analysis for
// Permission Management (T-PM-003, T-PM-005, T-PM-007) described in
// specs/security-architecture-spec-validation.md. It derives predicates
// directly from the exported/unexported implementation in role_checks.go,
// agent_validation.go, github_token.go and compiler_mutators.go.

// P1: WorkflowRunRepoSafetyCondition
// The compiled if: guard must check repo-id equality and non-fork status
// only for workflow_run events, and must allow all other events unconditionally.
func TestFormalPM005_WorkflowRunRepoSafetyCondition(t *testing.T) {
	c := NewCompiler()
	condition := c.buildWorkflowRunRepoSafetyCondition()

	assert.True(t, strings.HasPrefix(condition, "${{ "), "condition must be wrapped in an expression")
	assert.True(t, strings.HasSuffix(condition, " }}"), "condition must be wrapped in an expression")
	assert.Contains(t, condition, "github.event_name != 'workflow_run'")
	assert.Contains(t, condition, "github.event.workflow_run.repository.id == github.repository_id")
	assert.Contains(t, condition, "!(github.event.workflow_run.repository.fork)")
	// OR-combination: non-workflow_run events pass unconditionally.
	assert.Contains(t, condition, "||")
	// AND-combination: workflow_run events require both repo-id match and non-fork.
	assert.Contains(t, condition, "&&")
}

// P2: WorkflowRunRepoSafetyOnlyAppliesWhenTriggerPresent
// hasWorkflowRunTrigger must correctly detect map/string/absent "on:" forms.
func TestFormalPM005_WorkflowRunRepoSafetyOnlyWhenTriggerDeclared(t *testing.T) {
	c := NewCompiler()

	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    bool
	}{
		{
			name:        "nil frontmatter",
			frontmatter: nil,
			expected:    false,
		},
		{
			name:        "no on section",
			frontmatter: map[string]any{},
			expected:    false,
		},
		{
			name: "map form with workflow_run",
			frontmatter: map[string]any{
				"on": map[string]any{"workflow_run": map[string]any{}},
			},
			expected: true,
		},
		{
			name: "map form without workflow_run",
			frontmatter: map[string]any{
				"on": map[string]any{"push": map[string]any{}},
			},
			expected: false,
		},
		{
			name: "string form matching workflow_run",
			frontmatter: map[string]any{
				"on": "workflow_run",
			},
			expected: true,
		},
		{
			name: "string form not matching workflow_run",
			frontmatter: map[string]any{
				"on": "push",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, c.hasWorkflowRunTrigger(tt.frontmatter))
		})
	}
}

// P3: WorkflowRunRequiresNonEmptyWorkflowsField
func TestFormalPM_WorkflowRunRequiresNonEmptyWorkflows(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"non-empty string", "my-workflow", true},
		{"blank string", "   ", false},
		{"empty string", "", false},
		{"non-empty []string", []string{"my-workflow"}, true},
		{"[]string with only blanks", []string{"  ", ""}, false},
		{"empty []string", []string{}, false},
		{"non-empty []any of strings", []any{"my-workflow"}, true},
		{"empty []any", []any{}, false},
		{"nil value", nil, false},
		{"unsupported type", 42, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, hasNonEmptyWorkflowRunWorkflows(tt.value))
		})
	}
}

// Shared workflow_run "on:" fixtures used across the P4-P6 predicates below.
const (
	workflowRunOnMissingBranches = "on:\n  workflow_run:\n    workflows: [\"ci\"]\n    types: [completed]\n"
	workflowRunOnWithBranches    = "on:\n  workflow_run:\n    workflows: [\"ci\"]\n    types: [completed]\n    branches: [main]\n"
	pushOnNoWorkflowRun          = "on:\n  push:\n    branches: [main]\n"
)

// P4: WorkflowRunBranchRestrictionModeSensitive
// Missing branch restrictions must error in strict mode and warn otherwise.
func TestFormalPM003_StrictModeGatesMissingBranchRestriction(t *testing.T) {
	workflowData := &WorkflowData{On: workflowRunOnMissingBranches}

	t.Run("strict mode errors on missing branches", func(t *testing.T) {
		c := NewCompiler()
		c.SetStrictMode(true)
		err := c.validateWorkflowRunBranches(workflowData, "workflow.md")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "branch restrictions")
	})

	t.Run("non-strict mode warns on missing branches", func(t *testing.T) {
		c := NewCompiler()
		c.SetStrictMode(false)
		err := c.validateWorkflowRunBranches(workflowData, "workflow.md")
		require.NoError(t, err)
	})
}

// P5: WorkflowRunBranchRestrictionSatisfiedNoOp
// Present branch restrictions never trigger the strict/warn path, regardless
// of strict mode.
func TestFormalPM_BranchRestrictionPresentIsNoOp(t *testing.T) {
	workflowData := &WorkflowData{On: workflowRunOnWithBranches}

	for _, strict := range []bool{true, false} {
		c := NewCompiler()
		c.SetStrictMode(strict)
		err := c.validateWorkflowRunBranches(workflowData, "workflow.md")
		require.NoError(t, err, "strict=%v", strict)
	}
}

// P6: NoWorkflowRunTriggerIsNoOp
// Non-workflow_run triggers must bypass validation entirely, even in strict mode.
func TestFormalPM_NoWorkflowRunTriggerSkipsValidation(t *testing.T) {
	workflowData := &WorkflowData{On: pushOnNoWorkflowRun}

	c := NewCompiler()
	c.SetStrictMode(true)
	err := c.validateWorkflowRunBranches(workflowData, "workflow.md")
	require.NoError(t, err)
}

// P7: DefaultGitHubTokenPrecedence
// A custom token always wins; otherwise the 3-tier secret fallback chain is used.
func TestFormalPM007_DefaultGitHubTokenPrecedence(t *testing.T) {
	require.Equal(t, "custom-token", getEffectiveGitHubToken("custom-token"))

	fallback := getEffectiveGitHubToken("")
	require.Equal(t, "${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN || secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}", fallback)
}

// P8: SafeOutputGitHubTokenPrecedence
// A custom token always wins; otherwise the 2-tier safe-output fallback chain is used.
func TestFormalPM007_SafeOutputTokenPrecedence(t *testing.T) {
	require.Equal(t, "custom-safe-output-token", getEffectiveSafeOutputGitHubToken("custom-safe-output-token"))

	fallback := getEffectiveSafeOutputGitHubToken("")
	require.Equal(t, "${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}", fallback)
}

// P9: TokenChainsAreDistinctByJobRole
// The tool-token chain includes the MCP-server secret; the safe-output chain
// deliberately excludes it.
func TestFormalPM007_TokenChainsDifferByRole(t *testing.T) {
	toolChain := getEffectiveGitHubToken("")
	safeOutputChain := getEffectiveSafeOutputGitHubToken("")

	assert.Contains(t, toolChain, "GH_AW_GITHUB_MCP_SERVER_TOKEN")
	assert.NotContains(t, safeOutputChain, "GH_AW_GITHUB_MCP_SERVER_TOKEN")

	// Both chains share the common tail of the fallback chain.
	assert.Contains(t, toolChain, "GH_AW_GITHUB_TOKEN")
	assert.Contains(t, toolChain, "GITHUB_TOKEN")
	assert.Contains(t, safeOutputChain, "GH_AW_GITHUB_TOKEN")
	assert.Contains(t, safeOutputChain, "GITHUB_TOKEN")
}

// P10: StrictModeIsPerCompilerInstance
// SetStrictMode toggles the strictMode field deterministically and does not
// leak across compiler instances.
func TestFormalPM003_SetStrictModeIsIdempotentSetter(t *testing.T) {
	c1 := NewCompiler()
	c2 := NewCompiler()

	c1.SetStrictMode(true)
	c2.SetStrictMode(false)
	assert.True(t, c1.strictMode)
	assert.False(t, c2.strictMode)

	// Idempotent: setting the same value repeatedly is a no-op.
	c1.SetStrictMode(true)
	assert.True(t, c1.strictMode)

	// Toggling flips the field.
	c1.SetStrictMode(false)
	assert.False(t, c1.strictMode)
}

// P11: BashRestrictionWildcardSafe
// Boundary matrix for HasBashExplicitRestriction: nil/wildcard/false/empty-list.
func TestFormalPM_BashExplicitRestrictionBoundary(t *testing.T) {
	tests := []struct {
		name     string
		tools    map[string]any
		expected bool
	}{
		{"nil tools", nil, false},
		{"no bash key", map[string]any{}, false},
		{"bash true (no restriction)", map[string]any{"bash": true}, false},
		{"bash false (explicit restriction)", map[string]any{"bash": false}, true},
		{"bash nil value", map[string]any{"bash": nil}, false},
		{"bash wildcard list", map[string]any{"bash": []any{"*"}}, false},
		{"bash empty list (explicit restriction)", map[string]any{"bash": []any{}}, true},
		{"bash named list (explicit restriction)", map[string]any{"bash": []any{"ls"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, HasBashExplicitRestriction(tt.tools))
		})
	}
}
