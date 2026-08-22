//go:build !integration

package intent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/github/gh-aw/pkg/intent"
)

func TestFormalFixture_AmbiguousIgnoresPermissiveRuleConfig(t *testing.T) {
	fixture := loadIntentComplianceFixture(t, "ambiguous-root-closing-issues.yaml")

	rec := matchingResolver().ResolvePullRequest(buildFixturePullRequest(fixture))
	policy := permissiveWildcardCompiler().Compile(rec, intent.RepositoryContext{})

	assertFixturePolicy(t, fixture.Expected.Policy, policy)
	assert.Empty(t, policy.RuleIDs, "fail-closed policy must not apply permissive rules")
}

func TestFormalFixture_UnlinkedIgnoresPermissiveRuleConfig(t *testing.T) {
	fixture := loadIntentComplianceFixture(t, "unlinked-pr-fail-closed.yaml")

	rec := matchingResolver().ResolvePullRequest(buildFixturePullRequest(fixture))
	policy := permissiveWildcardCompiler().Compile(rec, intent.RepositoryContext{})

	assertFixturePolicy(t, fixture.Expected.Policy, policy)
	assert.Empty(t, policy.RuleIDs, "fail-closed policy must not apply permissive rules")
}

func TestFormalFixture_ExplicitIntentWinsEvenWithMatchingLabels(t *testing.T) {
	fixture := loadIntentComplianceFixture(t, "explicit-intent-wins.yaml")
	pr := buildFixturePullRequest(fixture)
	pr.Labels = []string{"bug", "priority-high"}

	labelOnlyPR := pr
	labelOnlyPR.ExplicitIntent = nil
	labelOnlyPR.ClosingIssues = nil
	labelOnly := matchingResolver().ResolvePullRequest(labelOnlyPR)
	assert.Equal(t, intent.SourceArtifactLabels, labelOnly.Source)
	assert.Equal(t, intent.AttributionMapped, labelOnly.Status)

	rec := matchingResolver().ResolvePullRequest(pr)

	assertFixtureAttribution(t, fixture.Expected.Attribution, rec)
	assert.Equal(t, intent.SourceExplicitMetadata, rec.Source)
	assert.Equal(t, intent.AttributionMapped, rec.Status)
}

func TestFormalFixture_NoRulesMatchYieldsSafestPolicyForMappedStatus(t *testing.T) {
	fixture := loadIntentComplianceFixture(t, "explicit-intent-wins.yaml")

	rec := matchingResolver().ResolvePullRequest(buildFixturePullRequest(fixture))
	policy := (intent.PolicyCompiler{}).Compile(rec, intent.RepositoryContext{})

	assert.Equal(t, "propose_only", policy.Autonomy)
	assert.Equal(t, "none", policy.WriteScope)
	assert.True(t, policy.HumanApprovalRequired)
	assert.NotNil(t, policy.AutoMergeAllowed)
	assert.False(t, *policy.AutoMergeAllowed)
	assert.Equal(t, 1, policy.MaxAttempts)
	assert.Empty(t, policy.RuleIDs)
}

func permissiveWildcardCompiler() intent.PolicyCompiler {
	autoMerge := true
	return intent.PolicyCompiler{
		Rules: []intent.PolicyRule{{
			ID: "permissive-wildcard",
			Set: intent.ExecutionPolicy{
				Autonomy:         "bounded",
				WriteScope:       "any_branch",
				AutoMergeAllowed: &autoMerge,
				MaxAttempts:      10,
			},
		}},
	}
}
