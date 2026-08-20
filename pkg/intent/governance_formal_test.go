//go:build !integration

package intent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/intent"
)

// Formal test suite derived from specs/intent-attribution-agent-governance.md,
// focusing on the Risk classification (ResolveRisk) and Enforcement
// (Authorizer.AuthorizeTool) sections, plus fail-closed policy compilation for
// unlinked/ambiguous attribution. Each test corresponds to a named predicate or
// invariant in the behavioral coverage map.

// TestResolveRisk_ExplicitOverride (P1/P2 — RiskExplicitOverride)
// Invariant: an explicit intent.Risk always wins over derived rules, even with
// conflicting domains/priority that would otherwise resolve differently.
func TestResolveRisk_ExplicitOverride(t *testing.T) {
	rec := intent.IntentRecord{
		Risk:     "low",
		Domains:  []string{"security", "production"},
		Priority: "critical",
	}
	assert.Equal(t, "low", intent.ResolveRisk(rec),
		"P1/P2: explicit risk must win over derived rules")
}

// TestResolveRisk_SecurityCriticalIsHigh (P3 — RiskSecurityCriticalHigh)
// Invariant: domains contains security AND priority == critical => high.
func TestResolveRisk_SecurityCriticalIsHigh(t *testing.T) {
	rec := intent.IntentRecord{
		Domains:  []string{"security"},
		Priority: "critical",
	}
	assert.Equal(t, "high", intent.ResolveRisk(rec),
		"P3: security+critical must resolve to high")
}

// TestResolveRisk_ProductionIsHigh (P4 — RiskProductionHigh)
// Invariant: domains contains production => high, independent of priority.
func TestResolveRisk_ProductionIsHigh(t *testing.T) {
	cases := []string{"", "low", "critical", "unrecognized"}
	for _, priority := range cases {
		t.Run("priority="+priority, func(t *testing.T) {
			rec := intent.IntentRecord{
				Domains:  []string{"production"},
				Priority: priority,
			}
			assert.Equal(t, "high", intent.ResolveRisk(rec),
				"P4: production domain must resolve to high regardless of priority")
		})
	}
}

// TestResolveRisk_InfrastructureIsMedium (P5 — RiskInfrastructureMedium)
// Invariant: domains contains infrastructure => medium.
func TestResolveRisk_InfrastructureIsMedium(t *testing.T) {
	rec := intent.IntentRecord{Domains: []string{"infrastructure"}}
	assert.Equal(t, "medium", intent.ResolveRisk(rec),
		"P5: infrastructure domain must resolve to medium")
}

// TestResolveRisk_DocumentationIsLow (P6 — RiskDocumentationLow)
// Invariant: domains contains documentation => low.
func TestResolveRisk_DocumentationIsLow(t *testing.T) {
	rec := intent.IntentRecord{Domains: []string{"documentation"}}
	assert.Equal(t, "low", intent.ResolveRisk(rec),
		"P6: documentation domain must resolve to low")
}

// TestResolveRisk_UnknownDefault (P7 — RiskUnknownDefault)
// Invariant: no matching rule (empty, unrecognized domain, security without
// critical priority) => unknown.
func TestResolveRisk_UnknownDefault(t *testing.T) {
	cases := []struct {
		name string
		rec  intent.IntentRecord
	}{
		{"empty", intent.IntentRecord{}},
		{"unrecognized_domain", intent.IntentRecord{Domains: []string{"marketing"}}},
		{"security_without_critical", intent.IntentRecord{Domains: []string{"security"}, Priority: "low"}},
		{"security_no_priority", intent.IntentRecord{Domains: []string{"security"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, "unknown", intent.ResolveRisk(tc.rec),
				"P7: non-matching input must resolve to unknown")
		})
	}
}

// TestResolveRisk_PrecedenceOrder (P8 — RiskPrecedenceOrder)
// Invariant: security+critical takes precedence when multiple domains overlap.
func TestResolveRisk_PrecedenceOrder(t *testing.T) {
	rec := intent.IntentRecord{
		Domains:  []string{"documentation", "infrastructure", "production", "security"},
		Priority: "critical",
	}
	assert.Equal(t, "high", intent.ResolveRisk(rec),
		"P8: security+critical must take precedence over other overlapping domains")
}

// TestAuthorizeTool_DeniedWins (P9 — AuthorizeToolDeniedWins)
// Invariant: a tool in DeniedTools is rejected even if it also appears in
// AllowedTools.
func TestAuthorizeTool_DeniedWins(t *testing.T) {
	policy := intent.ExecutionPolicy{
		AllowedTools: []string{"read", "write"},
		DeniedTools:  []string{"write"},
	}
	err := intent.Authorizer{}.AuthorizeTool(policy, "write")
	require.Error(t, err, "P9: denied tool must be rejected")
	assert.ErrorIs(t, err, intent.ErrToolDenied,
		"P9: denied tool must return ErrToolDenied")
}

// TestAuthorizeTool_AllowlistGate (P10 — AuthorizeToolAllowlistGate)
// Invariant: a non-nil allow list rejects tools not listed.
func TestAuthorizeTool_AllowlistGate(t *testing.T) {
	policy := intent.ExecutionPolicy{AllowedTools: []string{"read"}}
	err := intent.Authorizer{}.AuthorizeTool(policy, "exec")
	require.Error(t, err, "P10: tool absent from a restricted allow list must be rejected")
	require.ErrorIs(t, err, intent.ErrToolNotAllowed,
		"P10: tool absent from allow list must return ErrToolNotAllowed")

	require.NoError(t, intent.Authorizer{}.AuthorizeTool(policy, "read"),
		"P10: tool present in the allow list must be authorized")
}

// TestAuthorizeTool_UnrestrictedWhenAllowedToolsNil (P11 — AuthorizeToolUnrestricted)
// Invariant: nil AllowedTools means unrestricted (except explicit denies).
func TestAuthorizeTool_UnrestrictedWhenAllowedToolsNil(t *testing.T) {
	policy := intent.ExecutionPolicy{AllowedTools: nil, DeniedTools: []string{"exec"}}

	require.NoError(t, intent.Authorizer{}.AuthorizeTool(policy, "read"),
		"P11: nil AllowedTools must permit any tool that isn't denied")
	require.NoError(t, intent.Authorizer{}.AuthorizeTool(policy, "anything"),
		"P11: nil AllowedTools must permit any tool that isn't denied")

	err := intent.Authorizer{}.AuthorizeTool(policy, "exec")
	require.Error(t, err, "P11: an explicit deny must still be rejected even when unrestricted")
	assert.ErrorIs(t, err, intent.ErrToolDenied)
}

// TestAuthorizeTool_EmptyAllowedToolsDeniesAll (P12 — AuthorizeToolEmptyDenyAll)
// Invariant: a non-nil, empty AllowedTools denies every tool, distinct from nil.
func TestAuthorizeTool_EmptyAllowedToolsDeniesAll(t *testing.T) {
	policy := intent.ExecutionPolicy{AllowedTools: []string{}}
	err := intent.Authorizer{}.AuthorizeTool(policy, "read")
	require.Error(t, err, "P12: non-nil empty AllowedTools must deny all tools")
	assert.ErrorIs(t, err, intent.ErrToolNotAllowed)
}

// TestSafestDefaultPolicy_FailClosedForIndeterminateStatus (P13 — SafestDefaultFailClosed)
// Invariant: unlinked/ambiguous status forces the safest policy regardless of
// configured rules.
func TestSafestDefaultPolicy_FailClosedForIndeterminateStatus(t *testing.T) {
	autoMerge := true
	permissive := intent.PolicyRule{
		ID: "wildcard-permissive",
		Set: intent.ExecutionPolicy{
			Autonomy:              "bounded",
			WriteScope:            "any_branch",
			HumanApprovalRequired: false,
			AutoMergeAllowed:      &autoMerge,
			MaxAttempts:           10,
		},
	}
	compiler := intent.PolicyCompiler{Rules: []intent.PolicyRule{permissive}}
	repo := intent.RepositoryContext{Owner: "owner", Name: "repo"}

	for _, status := range []intent.AttributionStatus{intent.AttributionUnlinked, intent.AttributionAmbiguous} {
		t.Run(string(status), func(t *testing.T) {
			rec := intent.IntentRecord{Status: status}
			policy := compiler.Compile(rec, repo)

			assert.Equal(t, "propose_only", policy.Autonomy, "P13: indeterminate status must force propose_only")
			assert.Equal(t, "none", policy.WriteScope, "P13: indeterminate status must force no write scope")
			assert.True(t, policy.HumanApprovalRequired, "P13: indeterminate status must force human approval")
			require.NotNil(t, policy.AutoMergeAllowed)
			assert.False(t, *policy.AutoMergeAllowed, "P13: indeterminate status must force auto-merge denial")
			assert.Equal(t, 1, policy.MaxAttempts, "P13: indeterminate status must force a single attempt")
		})
	}
}

// TestEdgeCase_EmptyDomainsAndPriority validates that a fully empty intent
// record resolves to unknown, not a panic or empty string.
func TestEdgeCase_EmptyDomainsAndPriority(t *testing.T) {
	risk := intent.ResolveRisk(intent.IntentRecord{})
	assert.Equal(t, "unknown", risk, "edge case: fully empty record must resolve to unknown")
	assert.NotEmpty(t, risk, "edge case: ResolveRisk must never return an empty string")
}

// TestEdgeCase_NilDeniedAndAllowedTools validates that AuthorizeTool does not
// panic on a zero-value policy.
func TestEdgeCase_NilDeniedAndAllowedTools(t *testing.T) {
	require.NotPanics(t, func() {
		err := intent.Authorizer{}.AuthorizeTool(intent.ExecutionPolicy{}, "read")
		assert.NoError(t, err, "edge case: zero-value policy (nil AllowedTools/DeniedTools) must be unrestricted")
	})
}

// TestEdgeCase_MultipleMatchingRulesPreserveStricterConstraint validates that a
// stricter constraint from an earlier rule isn't overridden by a later, more
// lenient rule.
func TestEdgeCase_MultipleMatchingRulesPreserveStricterConstraint(t *testing.T) {
	strict := intent.PolicyRule{
		ID: "strict-first",
		Set: intent.ExecutionPolicy{
			Autonomy:   "propose_only",
			WriteScope: "none",
		},
	}
	lenient := intent.PolicyRule{
		ID: "lenient-second",
		Set: intent.ExecutionPolicy{
			Autonomy:   "bounded",
			WriteScope: "any_branch",
		},
	}
	compiler := intent.PolicyCompiler{Rules: []intent.PolicyRule{strict, lenient}}
	rec := intent.IntentRecord{Status: intent.AttributionMapped, Labels: []string{"security"}}
	repo := intent.RepositoryContext{Owner: "owner", Name: "repo"}

	policy := compiler.Compile(rec, repo)

	assert.Equal(t, "propose_only", policy.Autonomy,
		"edge case: a later lenient rule must not override an earlier stricter autonomy constraint")
	assert.Equal(t, "none", policy.WriteScope,
		"edge case: a later lenient rule must not override an earlier stricter write-scope constraint")
}
