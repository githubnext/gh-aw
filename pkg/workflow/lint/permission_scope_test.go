package lint

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionScopeRule(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value any
		want  int
		level Severity
	}{
		{"absent", nil, 0, ""},
		{"empty explicit mapping", map[string]any{}, 0, ""},
		{"all reads", "read-all", 0, ""},
		{"intentional scoped writes", map[string]any{"contents": "write", "issues": "read"}, 0, ""},
		{"OIDC issuance is not a repository write", map[string]string{"id-token": "write"}, 0, ""},
		{"OIDC none", map[string]any{"id-token": "none"}, 0, ""},
		{"blanket writes", "write-all", 1, Warning},
		{"invalid OIDC read", map[string]any{"id-token": "read"}, 1, Error},
	} {
		t.Run(tt.name, func(t *testing.T) {
			workflow := &actions.Workflow{Permissions: tt.value, Jobs: []*actions.Job{{ID: "test", Permissions: tt.value}}}
			before, err := json.Marshal(workflow)
			require.NoError(t, err)
			runner, err := New(PermissionScopeRule())
			require.NoError(t, err)
			diagnostics, err := runner.Run(context.Background(), Input{Workflow: workflow})
			require.NoError(t, err)
			require.Len(t, diagnostics, 2*tt.want)
			for _, diagnostic := range diagnostics {
				assert.Equal(t, tt.level, diagnostic.Severity)
			}
			after, err := json.Marshal(workflow)
			require.NoError(t, err)
			assert.Equal(t, before, after)
		})
	}
}

func TestPermissionScopeAuthoredPolicy(t *testing.T) {
	for _, strict := range []string{"", "strict: true", "strict: false"} {
		input := policyTestInput(t, strict)
		input.Workflow.Permissions = "write-all"
		diagnostics, err := PermissionScopeRule().Check(context.Background(), input)
		require.NoError(t, err)
		require.Len(t, diagnostics, 1)
		want := Error
		if strict == "strict: false" {
			want = Warning
		}
		assert.Equal(t, want, diagnostics[0].Severity)
	}
}

func TestPermissionScopePartialMalformedAndCancellation(t *testing.T) {
	for _, value := range []any{"private-value", 42, []string{"contents"}, map[string]any{"contents": "private-value"}, map[string]any{"contents": true}} {
		input := Input{Workflow: &actions.Workflow{Permissions: value, Jobs: []*actions.Job{{ID: "test", Permissions: "write-all"}}}}
		diagnostics, err := PermissionScopeRule().Check(context.Background(), input)
		require.Error(t, err)
		require.Len(t, diagnostics, 1, "later valid declarations must still be inspected")
		assert.NotContains(t, err.Error(), "private-value")
	}
	ctx := &integrityCancelContext{Context: context.Background(), remaining: 4}
	_, err := PermissionScopeRule().Check(ctx, Input{Workflow: &actions.Workflow{
		Permissions: map[string]any{"contents": "read", "issues": "read", "actions": "read", "checks": "read"},
	}})
	require.ErrorIs(t, err, context.Canceled)
}
