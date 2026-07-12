//go:build !integration

package authz_test

import (
	"testing"

	"github.com/github/gh-aw/pkg/intent"
	"github.com/github/gh-aw/pkg/intent/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ───────────────────────────────────────────────────
// Advisory mode (FeatureEnabled = false)
// ───────────────────────────────────────────────────

func TestAuthorizeTool_AdvisoryMode_AlwaysNil(t *testing.T) {
	a := authz.Authorizer{FeatureEnabled: false}
	policy := intent.ExecutionPolicy{
		DeniedTools:  []string{"deploy"},
		AllowedTools: []string{"read-file"},
	}
	assert.NoError(t, a.AuthorizeTool(policy, "deploy"),
		"advisory mode must not block a denied tool")
	assert.NoError(t, a.AuthorizeTool(policy, "write-file"),
		"advisory mode must not block a tool outside the allow list")
}

func TestAuthorizeWriteScope_AdvisoryMode_AlwaysNil(t *testing.T) {
	a := authz.Authorizer{FeatureEnabled: false}
	policy := intent.ExecutionPolicy{WriteScope: "none"}
	assert.NoError(t, a.AuthorizeWriteScope(policy, "any_branch"))
}

func TestCheckHumanApproval_AdvisoryMode_AlwaysNil(t *testing.T) {
	a := authz.Authorizer{FeatureEnabled: false}
	policy := intent.ExecutionPolicy{HumanApprovalRequired: true}
	assert.NoError(t, a.CheckHumanApproval(policy))
}

func TestCheckAttemptLimit_AdvisoryMode_AlwaysNil(t *testing.T) {
	a := authz.Authorizer{FeatureEnabled: false}
	policy := intent.ExecutionPolicy{MaxAttempts: 1}
	assert.NoError(t, a.CheckAttemptLimit(policy, 5))
}

// ───────────────────────────────────────────────────
// AuthorizeTool — enforcement mode
// ───────────────────────────────────────────────────

func TestAuthorizeTool_DeniedTool_Blocked(t *testing.T) {
	a := authz.Authorizer{FeatureEnabled: true}
	policy := intent.ExecutionPolicy{
		DeniedTools: []string{"deploy", "delete-branch"},
	}
	err := a.AuthorizeTool(policy, "deploy")
	require.Error(t, err)
	assert.ErrorIs(t, err, authz.ErrToolDenied,
		"a tool in DeniedTools must return ErrToolDenied")
}

func TestAuthorizeTool_NotInAllowList_Blocked(t *testing.T) {
	a := authz.Authorizer{FeatureEnabled: true}
	policy := intent.ExecutionPolicy{
		AllowedTools: []string{"read-file", "write-file"},
	}
	err := a.AuthorizeTool(policy, "deploy")
	require.Error(t, err)
	assert.ErrorIs(t, err, authz.ErrToolNotAllowed,
		"a tool absent from AllowedTools must return ErrToolNotAllowed")
}

func TestAuthorizeTool_InAllowList_Permitted(t *testing.T) {
	a := authz.Authorizer{FeatureEnabled: true}
	policy := intent.ExecutionPolicy{
		AllowedTools: []string{"read-file", "write-file"},
	}
	assert.NoError(t, a.AuthorizeTool(policy, "read-file"))
	assert.NoError(t, a.AuthorizeTool(policy, "write-file"))
}

func TestAuthorizeTool_NilAllowList_UnrestrictedExceptDenied(t *testing.T) {
	a := authz.Authorizer{FeatureEnabled: true}
	policy := intent.ExecutionPolicy{
		AllowedTools: nil, // nil means unrestricted
		DeniedTools:  []string{"delete-branch"},
	}
	assert.NoError(t, a.AuthorizeTool(policy, "any-tool"),
		"nil AllowedTools must not restrict any tool")
	err := a.AuthorizeTool(policy, "delete-branch")
	require.Error(t, err)
	assert.ErrorIs(t, err, authz.ErrToolDenied)
}

func TestAuthorizeTool_EmptyAllowList_DenyAll(t *testing.T) {
	a := authz.Authorizer{FeatureEnabled: true}
	policy := intent.ExecutionPolicy{
		AllowedTools: []string{}, // empty non-nil means deny-all
	}
	err := a.AuthorizeTool(policy, "any-tool")
	require.Error(t, err)
	assert.ErrorIs(t, err, authz.ErrToolNotAllowed,
		"an empty non-nil AllowedTools must deny all tools")
}

func TestAuthorizeTool_DeniedBeforeAllowed(t *testing.T) {
	// DeniedTools is checked before AllowedTools; a tool in both lists is denied.
	a := authz.Authorizer{FeatureEnabled: true}
	policy := intent.ExecutionPolicy{
		AllowedTools: []string{"deploy"},
		DeniedTools:  []string{"deploy"},
	}
	err := a.AuthorizeTool(policy, "deploy")
	require.Error(t, err)
	assert.ErrorIs(t, err, authz.ErrToolDenied,
		"DeniedTools must be checked before AllowedTools")
}

// ───────────────────────────────────────────────────
// AuthorizeWriteScope — enforcement mode
// ───────────────────────────────────────────────────

func TestAuthorizeWriteScope_Ordering(t *testing.T) {
	a := authz.Authorizer{FeatureEnabled: true}

	cases := []struct {
		policy    string
		requested string
		wantErr   bool
	}{
		{"none", "none", false},
		{"none", "feature_branch", true},
		{"none", "any_branch", true},
		{"feature_branch", "none", false},
		{"feature_branch", "feature_branch", false},
		{"feature_branch", "any_branch", true},
		{"any_branch", "none", false},
		{"any_branch", "feature_branch", false},
		{"any_branch", "any_branch", false},
	}

	for _, c := range cases {
		policy := intent.ExecutionPolicy{WriteScope: c.policy}
		err := a.AuthorizeWriteScope(policy, c.requested)
		if c.wantErr {
			require.Error(t, err, "policy=%s requested=%s should fail", c.policy, c.requested)
			assert.ErrorIs(t, err, authz.ErrWriteScopeExceeded)
		} else {
			assert.NoError(t, err, "policy=%s requested=%s should pass", c.policy, c.requested)
		}
	}
}

// ───────────────────────────────────────────────────
// CheckHumanApproval — enforcement mode
// ───────────────────────────────────────────────────

func TestCheckHumanApproval_Required_ReturnsError(t *testing.T) {
	a := authz.Authorizer{FeatureEnabled: true}
	policy := intent.ExecutionPolicy{HumanApprovalRequired: true}
	err := a.CheckHumanApproval(policy)
	require.Error(t, err)
	assert.ErrorIs(t, err, authz.ErrHumanApprovalRequired)
}

func TestCheckHumanApproval_NotRequired_NoError(t *testing.T) {
	a := authz.Authorizer{FeatureEnabled: true}
	policy := intent.ExecutionPolicy{HumanApprovalRequired: false}
	assert.NoError(t, a.CheckHumanApproval(policy))
}

// ───────────────────────────────────────────────────
// CheckAttemptLimit — enforcement mode
// ───────────────────────────────────────────────────

func TestCheckAttemptLimit_Exceeded_ReturnsError(t *testing.T) {
	a := authz.Authorizer{FeatureEnabled: true}
	policy := intent.ExecutionPolicy{MaxAttempts: 3}

	assert.NoError(t, a.CheckAttemptLimit(policy, 0), "0 attempts ≤ limit=3")
	assert.NoError(t, a.CheckAttemptLimit(policy, 2), "2 attempts ≤ limit=3")

	err := a.CheckAttemptLimit(policy, 3)
	require.Error(t, err)
	assert.ErrorIs(t, err, authz.ErrAttemptsExceeded,
		"3 attempts == limit=3 must be exceeded")

	err = a.CheckAttemptLimit(policy, 10)
	require.Error(t, err)
	assert.ErrorIs(t, err, authz.ErrAttemptsExceeded)
}

func TestCheckAttemptLimit_ZeroLimit_Unlimited(t *testing.T) {
	a := authz.Authorizer{FeatureEnabled: true}
	policy := intent.ExecutionPolicy{MaxAttempts: 0}
	assert.NoError(t, a.CheckAttemptLimit(policy, 1000),
		"MaxAttempts=0 means unlimited")
}
