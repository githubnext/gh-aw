//go:build !integration

package intent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/intent"
)

// Formal test suite derived from specs/intent-attribution-compliance/README.md.
// Each test corresponds to a named invariant in the behavioral coverage map.

// matchingResolver returns a Resolver whose MatchLabels accepts every non-empty label slice.
func matchingResolver() intent.Resolver {
	return intent.Resolver{
		ResolverVersion: "formal-v1",
		MatchLabels: func(labels []string) []string {
			return labels
		},
	}
}

// TestFormal_ExplicitIntentWins (P1 — ExplicitIntentWins)
// Invariant: ExplicitIntent wins over any other source (closing issues, labels).
func TestFormal_ExplicitIntentWins(t *testing.T) {
	r := matchingResolver()
	explicit := &intent.IntentRecord{
		Status: intent.AttributionMapped,
		Source: intent.SourceExplicitMetadata,
		Rule:   "explicit",
	}
	rec := r.ResolvePullRequest(intent.PullRequestData{
		NodeID:         "PR_explicit",
		ExplicitIntent: explicit,
		ClosingIssues: []intent.RootReference{
			{NodeID: "I_1", Labels: []string{"security"}},
			{NodeID: "I_2", Labels: []string{"automation"}},
		},
		Labels: []string{"bug"},
	})

	assert.Equal(t, intent.AttributionMapped, rec.Status, "P1: explicit intent status must win")
	assert.Equal(t, intent.SourceExplicitMetadata, rec.Source, "P1: explicit intent source must win")
}

// TestFormal_SingleClosingIssue (P2 — SingleClosingIssueAttributed)
// Invariant: One closing issue produces source = closing_issue.
func TestFormal_SingleClosingIssue(t *testing.T) {
	r := matchingResolver()
	rec := r.ResolvePullRequest(intent.PullRequestData{
		NodeID: "PR_single",
		ClosingIssues: []intent.RootReference{
			{NodeID: "I_kwDO1", Type: "issue", URL: "https://github.com/owner/repo/issues/1", Labels: []string{"security"}},
		},
	})

	assert.Equal(t, intent.SourceClosingIssue, rec.Source, "P2: single closing issue must produce closing_issue source")
	assert.Equal(t, intent.AttributionMapped, rec.Status, "P2: single closing issue with matching label must be mapped")
}

// TestFormal_MultipleClosingIssuesAmbiguous (P3 — AmbiguousOnMultipleRoots)
// Invariant: Two or more closing issues produce status = ambiguous.
func TestFormal_MultipleClosingIssuesAmbiguous(t *testing.T) {
	r := matchingResolver()
	rec := r.ResolvePullRequest(intent.PullRequestData{
		NodeID: "PR_ambiguous",
		ClosingIssues: []intent.RootReference{
			{NodeID: "I_1", Labels: []string{"security"}},
			{NodeID: "I_2", Labels: []string{"automation"}},
		},
	})

	assert.Equal(t, intent.AttributionAmbiguous, rec.Status, "P3: two closing issues must produce ambiguous status")
	assert.Equal(t, intent.SourceClosingIssue, rec.Source, "P3: ambiguous record must report closing_issue as source")
}

// TestFormal_AmbiguousNeverMapped (P4 — NoArbitraryAmbiguityResolution)
// Invariant: An ambiguous record is never reported as mapped.
func TestFormal_AmbiguousNeverMapped(t *testing.T) {
	r := matchingResolver()
	rec := r.ResolvePullRequest(intent.PullRequestData{
		NodeID: "PR_ambig2",
		ClosingIssues: []intent.RootReference{
			{NodeID: "I_a", Labels: []string{"security"}},
			{NodeID: "I_b", Labels: []string{"security"}},
		},
	})

	assert.Equal(t, intent.AttributionAmbiguous, rec.Status, "P4: multiple closing issues must produce ambiguous, not mapped")
	assert.NotEqual(t, intent.AttributionMapped, rec.Status, "P4: ambiguous status must never equal mapped")
}

// TestFormal_LabelFallback (P5 — LabelFallbackWhenNoClosingIssue)
// Invariant: When there are no closing issues, PR labels are used as fallback.
func TestFormal_LabelFallback(t *testing.T) {
	r := matchingResolver()
	rec := r.ResolvePullRequest(intent.PullRequestData{
		NodeID: "PR_label",
		URL:    "https://github.com/owner/repo/pull/10",
		Labels: []string{"automation"},
	})

	assert.Equal(t, intent.SourceArtifactLabels, rec.Source, "P5: no closing issue must fall back to artifact_labels")
	assert.Equal(t, intent.AttributionMapped, rec.Status, "P5: label fallback with matching label must be mapped")
}

// TestFormal_UnlinkedWhenNoSource (P6 — UnlinkedWhenNoSource)
// Invariant: With no closing issues, no labels, and no explicit intent, status is unlinked.
func TestFormal_UnlinkedWhenNoSource(t *testing.T) {
	r := matchingResolver()
	rec := r.ResolvePullRequest(intent.PullRequestData{NodeID: "PR_empty"})

	assert.Equal(t, intent.AttributionUnlinked, rec.Status, "P6: no source must produce unlinked status")
	assert.Equal(t, intent.SourceNone, rec.Source, "P6: unlinked record must have source none")
}

// TestFormal_SafestPolicyFields (P7 — FailClosedForUnlinked)
// Invariant: A PolicyCompiler with no rules produces the safest execution policy.
func TestFormal_SafestPolicyFields(t *testing.T) {
	compiler := intent.PolicyCompiler{}
	resolver := matchingResolver()

	unlinked := resolver.ResolvePullRequest(intent.PullRequestData{})
	require.Equal(t, intent.AttributionUnlinked, unlinked.Status)

	policy := compiler.Compile(unlinked, intent.RepositoryContext{})

	assert.Equal(t, "propose_only", policy.Autonomy, "P7: safest policy must be propose_only")
	assert.Equal(t, "none", policy.WriteScope, "P7: safest policy must have no write scope")
	assert.True(t, policy.HumanApprovalRequired, "P7: safest policy must require human approval")
	require.NotNil(t, policy.AutoMergeAllowed, "P7: safest policy must carry an explicit auto-merge value")
	assert.False(t, *policy.AutoMergeAllowed, "P7: safest policy must deny auto-merge")
	assert.Equal(t, 1, policy.MaxAttempts, "P7: safest policy must allow only one attempt")
}

// TestFormal_FailClosedForIndeterminate (P7b — FailClosedAlsoWithRules)
// Invariant: Unlinked and ambiguous records always receive the safest execution
// policy even when a permissive wildcard rule (empty conditions) is present.
func TestFormal_FailClosedForIndeterminate(t *testing.T) {
	autoMerge := true
	permissiveRule := intent.PolicyRule{
		ID: "wildcard-permissive",
		Set: intent.ExecutionPolicy{
			Autonomy:              "autonomous",
			WriteScope:            "bounded",
			HumanApprovalRequired: false,
			AutoMergeAllowed:      &autoMerge,
			MaxAttempts:           10,
		},
	}
	compiler := intent.PolicyCompiler{Rules: []intent.PolicyRule{permissiveRule}}
	resolver := matchingResolver()
	repo := intent.RepositoryContext{Owner: "owner", Name: "repo"}

	cases := []struct {
		name string
		pr   intent.PullRequestData
	}{
		{
			name: "unlinked",
			pr:   intent.PullRequestData{NodeID: "PR_unlinked"},
		},
		{
			name: "ambiguous",
			pr: intent.PullRequestData{
				NodeID: "PR_ambiguous",
				ClosingIssues: []intent.RootReference{
					{NodeID: "I_1", Labels: []string{"security"}},
					{NodeID: "I_2", Labels: []string{"automation"}},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := resolver.ResolvePullRequest(tc.pr)
			policy := compiler.Compile(rec, repo)

			assert.Equal(t, "propose_only", policy.Autonomy, "P7b: indeterminate records must always be propose_only")
			assert.Equal(t, "none", policy.WriteScope, "P7b: indeterminate records must always have no write scope")
			assert.True(t, policy.HumanApprovalRequired, "P7b: indeterminate records must always require human approval")
			require.NotNil(t, policy.AutoMergeAllowed, "P7b: indeterminate policy must carry explicit auto-merge value")
			assert.False(t, *policy.AutoMergeAllowed, "P7b: indeterminate records must always deny auto-merge")
			assert.Equal(t, 1, policy.MaxAttempts, "P7b: indeterminate records must always allow only one attempt")
		})
	}
}

// TestFormal_PolicyDeterminism (P8 — PolicyDeterminism)
// Invariant: Compiling the same intent record twice yields identical policies.
func TestFormal_PolicyDeterminism(t *testing.T) {
	compiler := intent.PolicyCompiler{}
	resolver := matchingResolver()

	rec := resolver.ResolvePullRequest(intent.PullRequestData{
		ClosingIssues: []intent.RootReference{
			{NodeID: "I_1", Labels: []string{"security"}},
		},
	})
	repo := intent.RepositoryContext{Owner: "owner", Name: "repo"}

	p1 := compiler.Compile(rec, repo)
	p2 := compiler.Compile(rec, repo)

	assert.Equal(t, p1, p2, "P8: identical inputs must produce identical policies")
}

// TestFormal_ExplicitOverridesSuggested (P9 — SuggestedNotOfficial)
// Invariant: Explicit metadata never returns a suggested status.
func TestFormal_ExplicitOverridesSuggested(t *testing.T) {
	r := matchingResolver()
	explicit := &intent.IntentRecord{
		Status: intent.AttributionMapped,
		Source: intent.SourceExplicitMetadata,
	}
	rec := r.ResolvePullRequest(intent.PullRequestData{
		ExplicitIntent: explicit,
	})

	assert.NotEqual(t, intent.AttributionSuggested, rec.Status, "P9: explicit intent must never yield suggested status")
	assert.Equal(t, intent.SourceExplicitMetadata, rec.Source, "P9: explicit intent source must be preserved")
}

// TestFormal_SingleSourcePerRecord (P10 — MixingSourcesForbidden)
// Invariant: Every resolved record carries exactly one attribution source.
func TestFormal_SingleSourcePerRecord(t *testing.T) {
	r := matchingResolver()

	cases := []struct {
		name string
		pr   intent.PullRequestData
	}{
		{
			name: "explicit_intent",
			pr: intent.PullRequestData{
				ExplicitIntent: &intent.IntentRecord{Status: intent.AttributionMapped, Source: intent.SourceExplicitMetadata},
			},
		},
		{
			name: "single_closing_issue",
			pr: intent.PullRequestData{
				ClosingIssues: []intent.RootReference{{NodeID: "I_1", Labels: []string{"security"}}},
			},
		},
		{
			name: "label_fallback",
			pr: intent.PullRequestData{
				NodeID: "PR_label",
				URL:    "https://github.com/owner/repo/pull/42",
				Labels: []string{"automation"},
			},
		},
		{
			name: "unlinked",
			pr:   intent.PullRequestData{},
		},
		{
			name: "ambiguous",
			pr: intent.PullRequestData{
				ClosingIssues: []intent.RootReference{
					{NodeID: "I_1", Labels: []string{"security"}},
					{NodeID: "I_2", Labels: []string{"automation"}},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := r.ResolvePullRequest(tc.pr)
			assert.NotEmpty(t, string(rec.Source), "P10: every record must carry exactly one attribution source")
		})
	}
}

// TestFormal_StricterWinsForAutonomyAndWriteScope (P11 — StricterAutonomyWins)
// Invariant: mergePolicy never lets a lower-precedence rule relax Autonomy or WriteScope;
// the more-restrictive value must always survive.
func TestFormal_StricterWinsForAutonomyAndWriteScope(t *testing.T) {
	// Two rules: the first grants broad access; the second is more restrictive.
	// Stricter-wins means the second (more restrictive) values must be retained.
	broadRule := intent.PolicyRule{
		ID: "broad",
		Set: intent.ExecutionPolicy{
			Autonomy:    "bounded",    // less restrictive
			WriteScope:  "any_branch", // less restrictive
			MaxAttempts: 5,
		},
	}
	strictRule := intent.PolicyRule{
		ID: "strict",
		Set: intent.ExecutionPolicy{
			Autonomy:    "propose_only", // more restrictive
			WriteScope:  "none",         // more restrictive
			MaxAttempts: 2,
		},
	}

	// Use a mapped record (labels required for non-unlinked status).
	rec := matchingResolver().ResolvePullRequest(intent.PullRequestData{
		ClosingIssues: []intent.RootReference{
			{NodeID: "I_1", Labels: []string{"security"}},
		},
	})
	repo := intent.RepositoryContext{Owner: "owner", Name: "repo"}

	// broad first, strict second: strict values must win.
	compiler := intent.PolicyCompiler{Rules: []intent.PolicyRule{broadRule, strictRule}}
	policy := compiler.Compile(rec, repo)

	assert.Equal(t, "propose_only", policy.Autonomy,
		"P11: stricter autonomy must survive when a lower-precedence rule is more permissive")
	assert.Equal(t, "none", policy.WriteScope,
		"P11: stricter write scope must survive when a lower-precedence rule is more permissive")
	assert.Equal(t, 2, policy.MaxAttempts,
		"P11: minimum max_attempts must be kept")

	// strict first, broad second: broad rule must not weaken the strict values.
	compiler2 := intent.PolicyCompiler{Rules: []intent.PolicyRule{strictRule, broadRule}}
	policy2 := compiler2.Compile(rec, repo)

	assert.Equal(t, "propose_only", policy2.Autonomy,
		"P11: broad rule must not weaken a stricter preceding rule's autonomy")
	assert.Equal(t, "none", policy2.WriteScope,
		"P11: broad rule must not weaken a stricter preceding rule's write scope")
}

// TestFormal_AllowedToolsIntersection (P12 — AllowedToolsIntersected)
// Invariant: mergePolicy intersects AllowedTools so that only tools permitted by
// all matching rules remain; nil (unrestricted) defers to the non-nil side.
func TestFormal_AllowedToolsIntersection(t *testing.T) {
	repo := intent.RepositoryContext{Owner: "owner", Name: "repo"}
	// Use a mapped record (labels required for non-unlinked status).
	rec := matchingResolver().ResolvePullRequest(intent.PullRequestData{
		ClosingIssues: []intent.RootReference{
			{NodeID: "I_1", Labels: []string{"security"}},
		},
	})

	t.Run("intersection_of_overlapping_lists", func(t *testing.T) {
		ruleA := intent.PolicyRule{
			ID:  "rule-a",
			Set: intent.ExecutionPolicy{AllowedTools: []string{"read", "write"}},
		}
		ruleB := intent.PolicyRule{
			ID:  "rule-b",
			Set: intent.ExecutionPolicy{AllowedTools: []string{"write", "exec"}},
		}
		compiler := intent.PolicyCompiler{Rules: []intent.PolicyRule{ruleA, ruleB}}
		policy := compiler.Compile(rec, repo)

		assert.Equal(t, []string{"write"}, policy.AllowedTools,
			"P12: only tools present in both rules must be allowed")
	})

	t.Run("nil_defers_to_restricted_side", func(t *testing.T) {
		ruleA := intent.PolicyRule{
			ID:  "rule-a",
			Set: intent.ExecutionPolicy{AllowedTools: nil}, // unrestricted
		}
		ruleB := intent.PolicyRule{
			ID:  "rule-b",
			Set: intent.ExecutionPolicy{AllowedTools: []string{"read"}},
		}
		compiler := intent.PolicyCompiler{Rules: []intent.PolicyRule{ruleA, ruleB}}
		policy := compiler.Compile(rec, repo)

		assert.Equal(t, []string{"read"}, policy.AllowedTools,
			"P12: unrestricted nil must defer to the non-nil restriction")
	})

	t.Run("deny_all_empty_slice_preserved", func(t *testing.T) {
		ruleA := intent.PolicyRule{
			ID:  "rule-a",
			Set: intent.ExecutionPolicy{AllowedTools: []string{"read"}},
		}
		ruleB := intent.PolicyRule{
			ID:  "rule-b",
			Set: intent.ExecutionPolicy{AllowedTools: []string{}}, // deny-all
		}
		compiler := intent.PolicyCompiler{Rules: []intent.PolicyRule{ruleA, ruleB}}
		policy := compiler.Compile(rec, repo)

		require.NotNil(t, policy.AllowedTools,
			"P12: deny-all empty slice must not be collapsed to unrestricted nil")
		assert.Empty(t, policy.AllowedTools,
			"P12: intersection with deny-all must yield deny-all")
	})
}
