package intent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ───────────────────────────────────────────────────
// safestDefaultPolicy
// ───────────────────────────────────────────────────

func TestSafestDefaultPolicy(t *testing.T) {
	p := safestDefaultPolicy()
	assert.Equal(t, "propose_only", p.Autonomy, "safestDefaultPolicy Autonomy")
	assert.Equal(t, "none", p.WriteScope, "safestDefaultPolicy WriteScope")
	assert.True(t, p.HumanApprovalRequired, "safestDefaultPolicy HumanApprovalRequired")
	require.NotNil(t, p.AutoMergeAllowed, "safestDefaultPolicy AutoMergeAllowed must not be nil")
	assert.False(t, *p.AutoMergeAllowed, "safestDefaultPolicy AutoMergeAllowed must be false")
	assert.Equal(t, 1, p.MaxAttempts, "safestDefaultPolicy MaxAttempts")
}

// ───────────────────────────────────────────────────
// PolicyCondition.Matches
// ───────────────────────────────────────────────────

func TestPolicyCondition_Empty_MatchesAll(t *testing.T) {
	cond := PolicyCondition{}
	record := IntentRecord{Status: AttributionMapped, Labels: []string{"security"}}
	assert.True(t, cond.Matches(record, RepositoryContext{}))
}

func TestPolicyCondition_Domain_RequiresLabel(t *testing.T) {
	cond := PolicyCondition{Domain: "security"}
	withLabel := IntentRecord{Labels: []string{"security", "high"}}
	withoutLabel := IntentRecord{Labels: []string{"documentation"}}

	assert.True(t, cond.Matches(withLabel, RepositoryContext{}))
	assert.False(t, cond.Matches(withoutLabel, RepositoryContext{}))
}

func TestPolicyCondition_Org_RequiresOrgMatch(t *testing.T) {
	cond := PolicyCondition{Org: "github"}
	matchRepo := RepositoryContext{Org: "github"}
	otherRepo := RepositoryContext{Org: "external"}

	assert.True(t, cond.Matches(IntentRecord{}, matchRepo))
	assert.False(t, cond.Matches(IntentRecord{}, otherRepo))
}

// ───────────────────────────────────────────────────
// mergeAllowedTools
// ───────────────────────────────────────────────────

func TestMergeAllowedTools_BothNil(t *testing.T) {
	assert.Nil(t, mergeAllowedTools(nil, nil),
		"nil × nil → nil (unrestricted)")
}

func TestMergeAllowedTools_OneNil(t *testing.T) {
	tools := []string{"read-file"}
	assert.Equal(t, tools, mergeAllowedTools(nil, tools),
		"nil × slice → adopt slice")
	assert.Equal(t, tools, mergeAllowedTools(tools, nil),
		"slice × nil → adopt slice")
}

func TestMergeAllowedTools_Intersection(t *testing.T) {
	a := []string{"read-file", "write-file", "deploy"}
	b := []string{"read-file", "write-file", "test"}
	got := mergeAllowedTools(a, b)
	assert.ElementsMatch(t, []string{"read-file", "write-file"}, got,
		"slice × slice → intersection")
}

func TestMergeAllowedTools_EmptySliceIsNotNil(t *testing.T) {
	tools := []string{"read-file"}
	got := mergeAllowedTools(tools, []string{})
	assert.NotNil(t, got, "intersection of non-empty × empty must not be nil (deny-all)")
	assert.Empty(t, got, "intersection of non-empty × empty must be empty slice (deny-all)")
}

// ───────────────────────────────────────────────────
// mergePolicy
// ───────────────────────────────────────────────────

func TestMergePolicy_DeniedToolsUnion(t *testing.T) {
	base := ExecutionPolicy{DeniedTools: []string{"deploy"}}
	override := ExecutionPolicy{DeniedTools: []string{"delete-branch"}}
	result := mergePolicy(base, override)
	assert.ElementsMatch(t, []string{"deploy", "delete-branch"}, result.DeniedTools)
}

func TestMergePolicy_RequiredChecksUnion(t *testing.T) {
	base := ExecutionPolicy{RequiredChecks: []string{"unit-tests"}}
	override := ExecutionPolicy{RequiredChecks: []string{"security-scan"}}
	result := mergePolicy(base, override)
	assert.ElementsMatch(t, []string{"unit-tests", "security-scan"}, result.RequiredChecks)
}

func TestMergePolicy_HumanApprovalOR(t *testing.T) {
	cases := []struct {
		base, override, want bool
	}{
		{false, false, false},
		{true, false, true},
		{false, true, true},
		{true, true, true},
	}
	for _, c := range cases {
		result := mergePolicy(
			ExecutionPolicy{HumanApprovalRequired: c.base},
			ExecutionPolicy{HumanApprovalRequired: c.override},
		)
		assert.Equal(t, c.want, result.HumanApprovalRequired,
			"OR(%v, %v)", c.base, c.override)
	}
}

func TestMergePolicy_AutoMergeAND(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	cases := []struct {
		base, override *bool
		want           *bool
	}{
		{nil, nil, nil},
		{boolPtr(true), nil, boolPtr(true)},
		{nil, boolPtr(true), boolPtr(true)},
		{boolPtr(true), boolPtr(true), boolPtr(true)},
		{boolPtr(true), boolPtr(false), boolPtr(false)},
		{boolPtr(false), boolPtr(true), boolPtr(false)},
		{boolPtr(false), boolPtr(false), boolPtr(false)},
	}
	for _, c := range cases {
		result := mergePolicy(
			ExecutionPolicy{AutoMergeAllowed: c.base},
			ExecutionPolicy{AutoMergeAllowed: c.override},
		)
		if c.want == nil {
			assert.Nil(t, result.AutoMergeAllowed)
		} else {
			require.NotNil(t, result.AutoMergeAllowed)
			assert.Equal(t, *c.want, *result.AutoMergeAllowed)
		}
	}
}

func TestMergePolicy_MaxAttemptsMin(t *testing.T) {
	cases := []struct {
		base, override, want int
	}{
		{0, 0, 0},  // 0 = unlimited, result is unlimited
		{0, 3, 3},  // override's non-zero limit wins
		{3, 0, 3},  // base's non-zero limit wins
		{5, 3, 3},  // smaller value wins
		{1, 10, 1}, // smaller value wins
	}
	for _, c := range cases {
		result := mergePolicy(
			ExecutionPolicy{MaxAttempts: c.base},
			ExecutionPolicy{MaxAttempts: c.override},
		)
		assert.Equal(t, c.want, result.MaxAttempts,
			"min(%d, %d)", c.base, c.override)
	}
}

func TestMergePolicy_WriteScopeMoreRestrictive(t *testing.T) {
	cases := []struct {
		base, override, want string
	}{
		{"none", "none", "none"},
		{"none", "feature_branch", "none"},
		{"none", "any_branch", "none"},
		{"feature_branch", "none", "none"},
		{"feature_branch", "feature_branch", "feature_branch"},
		{"feature_branch", "any_branch", "feature_branch"},
		{"any_branch", "none", "none"},
		{"any_branch", "feature_branch", "feature_branch"},
		{"any_branch", "any_branch", "any_branch"},
	}
	for _, c := range cases {
		result := mergePolicy(
			ExecutionPolicy{WriteScope: c.base},
			ExecutionPolicy{WriteScope: c.override},
		)
		assert.Equal(t, c.want, result.WriteScope,
			"moreRestrictive(%q, %q)", c.base, c.override)
	}
}

func TestMergePolicy_AutonomyMoreRestrictive(t *testing.T) {
	cases := []struct {
		base, override, want string
	}{
		{"propose_only", "propose_only", "propose_only"},
		{"propose_only", "supervised", "propose_only"},
		{"propose_only", "bounded", "propose_only"},
		{"supervised", "propose_only", "propose_only"},
		{"supervised", "supervised", "supervised"},
		{"supervised", "bounded", "supervised"},
		{"bounded", "propose_only", "propose_only"},
		{"bounded", "supervised", "supervised"},
		{"bounded", "bounded", "bounded"},
	}
	for _, c := range cases {
		result := mergePolicy(
			ExecutionPolicy{Autonomy: c.base},
			ExecutionPolicy{Autonomy: c.override},
		)
		assert.Equal(t, c.want, result.Autonomy,
			"moreRestrictiveAutonomy(%q, %q)", c.base, c.override)
	}
}

// ───────────────────────────────────────────────────
// PolicyCompiler.Compile
// ───────────────────────────────────────────────────

func TestPolicyCompiler_NoRules_ReturnsSafestDefault(t *testing.T) {
	compiler := PolicyCompiler{}
	record := IntentRecord{Status: AttributionMapped, Labels: []string{"security"}}
	policy := compiler.Compile(record, RepositoryContext{})

	assert.Equal(t, "propose_only", policy.Autonomy)
	assert.Equal(t, "none", policy.WriteScope)
	assert.True(t, policy.HumanApprovalRequired)
	require.NotNil(t, policy.AutoMergeAllowed)
	assert.False(t, *policy.AutoMergeAllowed)
	assert.Equal(t, 1, policy.MaxAttempts)
}

func TestPolicyCompiler_Unlinked_AlwaysSafestDefault(t *testing.T) {
	autoMerge := true
	compiler := PolicyCompiler{
		Rules: []PolicyRule{{
			ID:   "grant-bounded",
			When: PolicyCondition{},
			Set: ExecutionPolicy{
				Autonomy:         "bounded",
				WriteScope:       "any_branch",
				AutoMergeAllowed: &autoMerge,
			},
		}},
	}
	record := IntentRecord{Status: AttributionUnlinked, Labels: []string{"security"}}
	policy := compiler.Compile(record, RepositoryContext{})

	assert.Equal(t, "propose_only", policy.Autonomy,
		"unlinked status must always produce safest policy")
	assert.Equal(t, "none", policy.WriteScope)
	assert.True(t, policy.HumanApprovalRequired)
}

func TestPolicyCompiler_Ambiguous_AlwaysSafestDefault(t *testing.T) {
	autoMerge := true
	compiler := PolicyCompiler{
		Rules: []PolicyRule{{
			ID:   "grant-bounded",
			When: PolicyCondition{},
			Set: ExecutionPolicy{
				Autonomy:         "bounded",
				AutoMergeAllowed: &autoMerge,
			},
		}},
	}
	record := IntentRecord{Status: AttributionAmbiguous}
	policy := compiler.Compile(record, RepositoryContext{})

	assert.Equal(t, "propose_only", policy.Autonomy,
		"ambiguous status must always produce safest policy")
}

func TestPolicyCompiler_SingleRule_GrantsBounded(t *testing.T) {
	autoMerge := true
	compiler := PolicyCompiler{
		Rules: []PolicyRule{{
			ID:   "security-bounded",
			When: PolicyCondition{Domain: "security"},
			Set: ExecutionPolicy{
				Autonomy:         "bounded",
				WriteScope:       "feature_branch",
				AutoMergeAllowed: &autoMerge,
				MaxAttempts:      3,
			},
		}},
	}
	record := IntentRecord{Status: AttributionMapped, Labels: []string{"security"}}
	policy := compiler.Compile(record, RepositoryContext{})

	assert.Equal(t, "bounded", policy.Autonomy)
	assert.Equal(t, "feature_branch", policy.WriteScope)
	require.NotNil(t, policy.AutoMergeAllowed)
	assert.True(t, *policy.AutoMergeAllowed)
	assert.Equal(t, 3, policy.MaxAttempts)
	assert.Equal(t, []string{"security-bounded"}, policy.RuleIDs)
}

func TestPolicyCompiler_OrgRuleBeforeIntentRule_MoreRestrictiveWins(t *testing.T) {
	// Org rule grants bounded; intent rule restricts to propose_only.
	// More-restrictive-wins: intent rule's propose_only overrides org's bounded.
	compiler := PolicyCompiler{
		Rules: []PolicyRule{
			{
				ID:    "org-bounded",
				Scope: "organization",
				When:  PolicyCondition{},
				Set:   ExecutionPolicy{Autonomy: "bounded", WriteScope: "any_branch"},
			},
			{
				ID:    "intent-restricted",
				Scope: "intent",
				When:  PolicyCondition{Domain: "risky"},
				Set:   ExecutionPolicy{Autonomy: "propose_only", WriteScope: "none"},
			},
		},
	}
	record := IntentRecord{Status: AttributionMapped, Labels: []string{"risky"}}
	policy := compiler.Compile(record, RepositoryContext{})

	assert.Equal(t, "propose_only", policy.Autonomy,
		"intent rule restricting to propose_only must override org's bounded")
	assert.Equal(t, "none", policy.WriteScope)
	assert.ElementsMatch(t, []string{"org-bounded", "intent-restricted"}, policy.RuleIDs)
}

func TestPolicyCompiler_RuleNotMatched_SafeDefault(t *testing.T) {
	// Rule requires "security" label; record only has "documentation" label.
	compiler := PolicyCompiler{
		Rules: []PolicyRule{{
			ID:   "security-only",
			When: PolicyCondition{Domain: "security"},
			Set:  ExecutionPolicy{Autonomy: "bounded"},
		}},
	}
	record := IntentRecord{Status: AttributionMapped, Labels: []string{"documentation"}}
	policy := compiler.Compile(record, RepositoryContext{})

	assert.Equal(t, "propose_only", policy.Autonomy,
		"no matching rule must produce safest default")
}

func TestPolicyCompiler_RequiredChecksUnionAcrossRules(t *testing.T) {
	compiler := PolicyCompiler{
		Rules: []PolicyRule{
			{
				ID:   "org-checks",
				When: PolicyCondition{},
				Set:  ExecutionPolicy{RequiredChecks: []string{"unit-tests"}},
			},
			{
				ID:   "security-checks",
				When: PolicyCondition{Domain: "security"},
				Set:  ExecutionPolicy{RequiredChecks: []string{"security-scan"}},
			},
		},
	}
	record := IntentRecord{Status: AttributionMapped, Labels: []string{"security"}}
	policy := compiler.Compile(record, RepositoryContext{})

	assert.ElementsMatch(t, []string{"unit-tests", "security-scan"}, policy.RequiredChecks)
}

func TestPolicyCompiler_RuleIDsRecorded(t *testing.T) {
	compiler := PolicyCompiler{
		Rules: []PolicyRule{
			{ID: "r1", When: PolicyCondition{}, Set: ExecutionPolicy{}},
			{ID: "r2", When: PolicyCondition{Domain: "x"}, Set: ExecutionPolicy{}},
			{ID: "r3", When: PolicyCondition{}, Set: ExecutionPolicy{}},
		},
	}
	record := IntentRecord{Status: AttributionMapped, Labels: []string{"x"}}
	policy := compiler.Compile(record, RepositoryContext{})

	assert.ElementsMatch(t, []string{"r1", "r2", "r3"}, policy.RuleIDs,
		"all matching rule IDs must be recorded")
}
