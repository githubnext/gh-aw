//go:build !integration

package intent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/intent"
)

// boolPtr returns a pointer to b, used to set *bool fields in ExecutionPolicy literals.
func boolPtr(b bool) *bool { return new(b) }

// Tests for PolicyCompiler.Compile() covering organization > repository > intent
// precedence. A lower-precedence rule MUST NOT weaken a constraint imposed by a
// higher-precedence rule.
//
// Spec (intent-attribution-agent-governance.md §Policy precedence):
//
//	organization constraints
//	> repository constraints
//	> intent-specific rules
//	> workflow defaults
//	> agent request

// TestPolicyCompilerOrgConstraintsPreservedOverRepo verifies that organization-scoped
// constraints cannot be weakened by a later repository-scoped rule.
//
// Scenario: an org rule denies a dangerous tool and requires human approval;
// a repository rule then tries to remove the denied tool and clear the approval
// requirement. The compiled policy MUST preserve both constraints from the org rule.
func TestPolicyCompilerOrgConstraintsPreservedOverRepo(t *testing.T) {
	compiler := &intent.PolicyCompiler{
		Rules: []intent.PolicyRule{
			{
				ID:    "org-security-policy",
				Scope: "organization",
				When:  intent.PolicyCondition{},
				Set: intent.ExecutionPolicy{
					HumanApprovalRequired: true,
					AutoMergeAllowed:      boolPtr(false),
					DeniedTools:           []string{"delete_repository", "push_direct_to_main"},
					RequiredChecks:        []string{"security-scan"},
					MaxAttempts:           2,
				},
			},
			{
				ID:    "repo-permissive-policy",
				Scope: "repository",
				When:  intent.PolicyCondition{},
				Set: intent.ExecutionPolicy{
					// Lower-precedence attempt to weaken org constraints:
					HumanApprovalRequired: false,         // tries to remove approval requirement
					AutoMergeAllowed:      boolPtr(true), // tries to enable auto-merge
					DeniedTools:           []string{},    // tries to clear denied tools
					RequiredChecks:        []string{},    // tries to clear required checks
					MaxAttempts:           10,            // tries to increase max attempts
				},
			},
		},
	}

	record := intent.IntentRecord{Status: intent.AttributionMapped}
	repo := intent.RepositoryContext{Owner: "github", Name: "gh-aw"}

	policy := compiler.Compile(record, repo)

	// Org rule's HumanApprovalRequired=true must NOT be cleared by the repo rule.
	assert.True(t, policy.HumanApprovalRequired,
		"organization HumanApprovalRequired=true must not be weakened by repository rule")

	// Org rule's AutoMergeAllowed=false must NOT be enabled by the repo rule.
	require.NotNil(t, policy.AutoMergeAllowed,
		"AutoMergeAllowed must be explicitly set by the org rule")
	assert.False(t, *policy.AutoMergeAllowed,
		"organization AutoMergeAllowed=false must not be enabled by repository rule")

	// Org's denied tools must be preserved even when the repo rule passes an empty list.
	assert.Contains(t, policy.DeniedTools, "delete_repository",
		"organization denied tool 'delete_repository' must not be removed by repository rule")
	assert.Contains(t, policy.DeniedTools, "push_direct_to_main",
		"organization denied tool 'push_direct_to_main' must not be removed by repository rule")

	// Org's required check must be preserved.
	assert.Contains(t, policy.RequiredChecks, "security-scan",
		"organization required check 'security-scan' must not be removed by repository rule")

	// MaxAttempts must not be increased beyond the org limit.
	// Use Equal (not LessOrEqual) to verify the org rule's value (2) was actually applied,
	// not silently replaced by the safe default (1).
	assert.Equal(t, 2, policy.MaxAttempts,
		"organization MaxAttempts=2 must be applied and not raised by repository rule")

	// Both rules should appear in the applied rule IDs.
	require.Contains(t, policy.RuleIDs, "org-security-policy")
	require.Contains(t, policy.RuleIDs, "repo-permissive-policy")
}

// TestPolicyCompilerRepoConstraintsPreservedOverIntent verifies that repository-scoped
// constraints cannot be weakened by a lower-precedence intent-specific rule.
//
// Scenario: a repository rule enforces AllowedTools to a restricted set and adds
// required checks; an intent-specific rule for "low" priority tries to widen
// AllowedTools and does not add checks. The repo's AllowedTools restriction and
// required checks must be preserved.
func TestPolicyCompilerRepoConstraintsPreservedOverIntent(t *testing.T) {
	compiler := &intent.PolicyCompiler{
		Rules: []intent.PolicyRule{
			{
				ID:    "repo-tool-restriction",
				Scope: "repository",
				When:  intent.PolicyCondition{},
				Set: intent.ExecutionPolicy{
					AllowedTools:   []string{"issue_read", "list_issues", "list_prs"},
					RequiredChecks: []string{"unit-tests", "lint"},
					MaxAttempts:    3,
				},
			},
			{
				ID:    "intent-low-priority",
				Scope: "intent",
				When: intent.PolicyCondition{
					Priority: "low",
				},
				Set: intent.ExecutionPolicy{
					// Lower-precedence attempt to widen tool access and add a check.
					AllowedTools:   []string{"issue_read", "list_issues", "list_prs", "create_pr"},
					RequiredChecks: []string{"docs-build"},
					MaxAttempts:    5,
				},
			},
		},
	}

	record := intent.IntentRecord{
		Status: intent.AttributionMapped,
		Labels: []string{"low"},
	}
	repo := intent.RepositoryContext{Owner: "github", Name: "gh-aw"}

	policy := compiler.Compile(record, repo)

	// AllowedTools: the repo restricts to 3 tools; intent tries to add "create_pr".
	// The intersection (repo's tighter set) must be used — "create_pr" must NOT appear.
	assert.NotContains(t, policy.AllowedTools, "create_pr",
		"intent rule must not expand AllowedTools beyond what the repository rule allows")
	assert.Contains(t, policy.AllowedTools, "issue_read",
		"tools allowed by both repo and intent rules must be preserved")
	assert.Contains(t, policy.AllowedTools, "list_issues",
		"tools allowed by both repo and intent rules must be preserved")

	// RequiredChecks: union of both rules; all checks must be present.
	assert.Contains(t, policy.RequiredChecks, "unit-tests",
		"repository required check must be preserved")
	assert.Contains(t, policy.RequiredChecks, "lint",
		"repository required check must be preserved")
	assert.Contains(t, policy.RequiredChecks, "docs-build",
		"intent required check must be added to the union")

	// MaxAttempts: intent tries to raise it; repo's tighter limit must win.
	assert.LessOrEqual(t, policy.MaxAttempts, 3,
		"repository MaxAttempts=3 must not be increased by lower-precedence intent rule")

	// Both rules matched.
	require.Contains(t, policy.RuleIDs, "repo-tool-restriction")
	require.Contains(t, policy.RuleIDs, "intent-low-priority")
}

// TestPolicyCompilerNoRulesMatchReturnsSafeDefault verifies that when no rules
// match the intent, the compiled policy equals the safe fail-closed default.
//
// Spec: "Unknown or ambiguous intent must not grant elevated authority."
func TestPolicyCompilerNoRulesMatchReturnsSafeDefault(t *testing.T) {
	compiler := &intent.PolicyCompiler{
		Rules: []intent.PolicyRule{
			{
				ID:    "security-critical-rule",
				Scope: "intent",
				When: intent.PolicyCondition{
					Domain:   "security",
					Priority: "critical",
				},
				Set: intent.ExecutionPolicy{
					RequiredChecks: []string{"security-tests", "dependency-review"},
					DeniedTools:    []string{"delete_repository"},
					MaxAttempts:    2,
				},
			},
			{
				ID:    "documentation-low-risk",
				Scope: "intent",
				When: intent.PolicyCondition{
					Domain: "documentation",
				},
				Set: intent.ExecutionPolicy{
					RequiredChecks: []string{"docs-build"},
					MaxAttempts:    3,
				},
			},
		},
	}

	// Ambiguous intent has no matching labels for either rule.
	record := intent.IntentRecord{
		Status: intent.AttributionAmbiguous,
		Source: intent.SourceNone,
		Labels: nil,
	}
	repo := intent.RepositoryContext{Owner: "github", Name: "gh-aw"}

	policy := compiler.Compile(record, repo)

	// No rules matched → safe default must govern the policy.
	assert.Equal(t, "propose_only", policy.Autonomy,
		"ambiguous intent with no matching rules must produce propose_only autonomy")
	assert.Equal(t, "none", policy.WriteScope,
		"ambiguous intent with no matching rules must produce write_scope=none")
	assert.True(t, policy.HumanApprovalRequired,
		"ambiguous intent with no matching rules must require human approval")
	assert.False(t, policy.AutoMergeAllowed == nil || *policy.AutoMergeAllowed,
		"ambiguous intent with no matching rules must not allow auto-merge")
	assert.Equal(t, 1, policy.MaxAttempts,
		"ambiguous intent with no matching rules must produce max_attempts=1")
	assert.Empty(t, policy.RuleIDs,
		"no applied rule IDs should be recorded when no rules match")
}

// TestPolicyCompilerRulesCanGrantLessRestrictiveThanSafeDefault verifies that
// matching rules can produce a policy less restrictive than the safe default
// (e.g. supervised autonomy, auto_merge_allowed=true, max_attempts>1).
// Previously the compiler always seeded from safestDefaultPolicy(), making this
// impossible.
func TestPolicyCompilerRulesCanGrantLessRestrictiveThanSafeDefault(t *testing.T) {
	compiler := &intent.PolicyCompiler{
		Rules: []intent.PolicyRule{
			{
				ID:    "supervised-docs-rule",
				Scope: "intent",
				When: intent.PolicyCondition{
					Domain: "documentation",
				},
				Set: intent.ExecutionPolicy{
					Autonomy:              "supervised",
					WriteScope:            "feature_branch",
					AutoMergeAllowed:      boolPtr(true),
					HumanApprovalRequired: false,
					MaxAttempts:           5,
				},
			},
		},
	}

	record := intent.IntentRecord{
		Status: intent.AttributionMapped,
		// Labels carries the domain value used by PolicyCondition.Matches to satisfy
		// the When.Domain="documentation" condition on the rule above.
		Labels: []string{"documentation"},
	}
	repo := intent.RepositoryContext{Owner: "github", Name: "gh-aw"}

	policy := compiler.Compile(record, repo)

	assert.Equal(t, "supervised", policy.Autonomy,
		"a matching rule must be able to grant supervised autonomy (not just propose_only)")
	assert.Equal(t, "feature_branch", policy.WriteScope,
		"a matching rule must be able to grant feature_branch write scope")
	require.NotNil(t, policy.AutoMergeAllowed,
		"AutoMergeAllowed must be explicitly set by the matching rule")
	assert.True(t, *policy.AutoMergeAllowed,
		"a matching rule must be able to enable auto_merge_allowed")
	assert.Equal(t, 5, policy.MaxAttempts,
		"a matching rule must be able to set max_attempts > 1")
	assert.Contains(t, policy.RuleIDs, "supervised-docs-rule")
}

// TestPolicyCompilerScopeOrderingEnforced verifies that rules are applied in
// scope-precedence order (organization > repository > intent) regardless of
// the declaration order in the Rules slice. A lower-precedence scope declared
// first must not override a higher-precedence scope declared last.
func TestPolicyCompilerScopeOrderingEnforced(t *testing.T) {
	// Rules are intentionally listed in reverse precedence order: intent first,
	// then repository, then organization. The compiled policy must still apply
	// them highest-precedence-first (org seeds the policy, intent only narrows).
	compiler := &intent.PolicyCompiler{
		Rules: []intent.PolicyRule{
			{
				// Declared first but lowest precedence: should NOT seed the policy.
				ID:    "intent-rule",
				Scope: "intent",
				When:  intent.PolicyCondition{},
				Set: intent.ExecutionPolicy{
					Autonomy:    "bounded",
					MaxAttempts: 10,
				},
			},
			{
				ID:    "repo-rule",
				Scope: "repository",
				When:  intent.PolicyCondition{},
				Set: intent.ExecutionPolicy{
					Autonomy:    "supervised",
					MaxAttempts: 5,
				},
			},
			{
				// Declared last but highest precedence: MUST seed the policy.
				ID:    "org-rule",
				Scope: "organization",
				When:  intent.PolicyCondition{},
				Set: intent.ExecutionPolicy{
					Autonomy:    "propose_only",
					MaxAttempts: 2,
				},
			},
		},
	}

	record := intent.IntentRecord{Status: intent.AttributionMapped}
	repo := intent.RepositoryContext{Owner: "github", Name: "gh-aw"}

	policy := compiler.Compile(record, repo)

	// The org rule (propose_only, MaxAttempts=2) must dominate.
	// If scope ordering were ignored, the intent rule (bounded, MaxAttempts=10) would
	// seed the policy and produce a different autonomy level.
	assert.Equal(t, "propose_only", policy.Autonomy,
		"organization scope must take precedence over lower scopes")
	assert.Equal(t, 2, policy.MaxAttempts,
		"organization MaxAttempts=2 must win; declaration order must not override scope precedence")
}

// rules restrict AllowedTools to non-overlapping sets, the compiled policy denies
// all tools ([]string{} sentinel) rather than silently reverting to unrestricted (nil).
func TestPolicyCompilerAllowedToolsDenyAllOnEmptyIntersection(t *testing.T) {
	compiler := &intent.PolicyCompiler{
		Rules: []intent.PolicyRule{
			{
				ID:    "repo-allows-read-tools",
				Scope: "repository",
				When:  intent.PolicyCondition{},
				Set: intent.ExecutionPolicy{
					AllowedTools: []string{"issue_read", "list_issues"},
				},
			},
			{
				ID:    "intent-allows-write-tools",
				Scope: "intent",
				When:  intent.PolicyCondition{},
				Set: intent.ExecutionPolicy{
					AllowedTools: []string{"create_pr", "push_branch"},
				},
			},
		},
	}

	record := intent.IntentRecord{Status: intent.AttributionMapped}
	repo := intent.RepositoryContext{Owner: "github", Name: "gh-aw"}

	policy := compiler.Compile(record, repo)

	// The intersection is empty (no tool appears in both sets).
	// Result MUST be deny-all (non-nil empty slice), not unrestricted (nil).
	assert.Equal(t, []string{}, policy.AllowedTools,
		"non-overlapping AllowedTools sets must produce deny-all ([]string{}), not unrestricted (nil)")
}
