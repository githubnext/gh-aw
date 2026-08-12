package authz

import (
	"testing"

	"github.com/github/gh-aw/pkg/intent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthorizerAuthorizeToolRejectsDeniedTool(t *testing.T) {
	err := (Authorizer{}).AuthorizeTool(intent.ExecutionPolicy{
		AllowedTools: []string{"compile", "add"},
		DeniedTools:  []string{"add"},
	}, "add", ToolContext{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrToolDenied)
}

func TestAuthorizerAuthorizeToolRejectsToolOutsideAllowedSet(t *testing.T) {
	err := (Authorizer{}).AuthorizeTool(intent.ExecutionPolicy{
		AllowedTools: []string{"compile"},
	}, "add", ToolContext{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrToolNotAllowed)
}

func TestAuthorizerAuthorizeToolEnforcesExecutionPolicyGates(t *testing.T) {
	autoMergeDenied := false
	policy := intent.ExecutionPolicy{
		Autonomy:              "bounded",
		AllowedTools:          []string{"merge_pull_request"},
		WriteScope:            "feature_branch",
		HumanApprovalRequired: true,
		RequiredChecks:        []string{"unit-tests"},
		AutoMergeAllowed:      &autoMergeDenied,
		MaxAttempts:           2,
	}

	t.Run("max attempts", func(t *testing.T) {
		err := (Authorizer{}).AuthorizeTool(policy, "merge_pull_request", ToolContext{
			IsWrite:      true,
			IsAutoMerge:  true,
			Approved:     true,
			PassedChecks: []string{"unit-tests"},
			Attempt:      3,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMaxAttemptsExceeded)
	})

	t.Run("auto merge", func(t *testing.T) {
		err := (Authorizer{}).AuthorizeTool(policy, "merge_pull_request", ToolContext{
			IsWrite:      true,
			IsAutoMerge:  true,
			Approved:     true,
			PassedChecks: []string{"unit-tests"},
			Attempt:      1,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAutoMergeDenied)
	})

	t.Run("required checks", func(t *testing.T) {
		err := (Authorizer{}).AuthorizeTool(policy, "merge_pull_request", ToolContext{
			IsWrite:  true,
			Approved: true,
			Attempt:  1,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrRequiredChecks)
	})
}
