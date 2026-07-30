//go:build !integration

// Package workflow – intent attribution & agent governance formal model tests.
//
// This file encodes the formal specification predicates (P1–P11 and a
// cross-cutting safety invariant) from
// specs/intent-attribution-agent-governance.md.
//
// Types and functions marked "stub — replace with real implementation" are
// placeholder implementations that directly encode the spec semantics.  They
// must be replaced with the real intent-attribution resolver once Phase 2+
// ("honest attribution model") lands.
//
// Usage:
//
//	go test ./pkg/workflow/... -run TestFormal_P
//	go test ./pkg/workflow/... -run TestFormal_SAFETY
package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Stub types — replace with real implementation
// ---------------------------------------------------------------------------

// pullRequestData represents the inputs used by the intent resolver.
// stub — replace with real implementation
type pullRequestData struct {
	// ExplicitIntent, when non-nil, signals that the workflow explicitly
	// declared its intent in metadata. It overrides all other sources.
	ExplicitIntent *formalIntentKey

	// ClosingIssues lists the issue numbers linked to the pull request
	// as "closes #N" references.
	ClosingIssues []int

	// Labels are the labels attached to the pull request artifact.
	Labels []string
}

// formalIntentKey carries the key for an explicit intent declaration.
// stub — replace with real implementation
type formalIntentKey struct {
	Key string
}

// formalIntentRecord is the normalized output of the intent resolver.
// stub — replace with real implementation
type formalIntentRecord struct {
	// Status is the attribution resolution result:
	// "mapped", "ambiguous", "artifact_label", "unlinked", "suggested", "unmapped"
	Status string

	// Source is the provenance:
	// "explicit_metadata", "closing_issue", "artifact_label", "none"
	Source string

	// Key is the resolved intent key (empty when unresolved).
	Key string

	// Risk, when non-empty, is an explicit risk override.
	Risk string

	// Priority qualifies the intent for risk derivation.
	Priority string

	// Domains are the domain tags attached to the intent.
	Domains []string
}

// formalExecutionPolicy encodes what an agent is permitted and required to do.
// stub — replace with real implementation
type formalExecutionPolicy struct {
	Autonomy              string // "propose_only", "supervised", "bounded"
	WriteScope            string // "none", "feature_branch", "limited"
	HumanApprovalRequired bool
	AutoMergeAllowed      bool
	MaxAttempts           int
}

// ---------------------------------------------------------------------------
// Stub functions — replace with real implementation
// ---------------------------------------------------------------------------

// formalResolve applies the attribution-resolution order specified in
// specs/intent-attribution-agent-governance.md § Attribution-Resolution Order.
// stub — replace with real implementation
func formalResolve(pr pullRequestData) formalIntentRecord {
	// Step 1: explicit intent wins.
	if pr.ExplicitIntent != nil {
		return formalIntentRecord{
			Status: "mapped",
			Source: "explicit_metadata",
			Key:    pr.ExplicitIntent.Key,
		}
	}

	// Step 2: exactly one closing issue → mapped via closing issue.
	if len(pr.ClosingIssues) == 1 {
		return formalIntentRecord{
			Status: "mapped",
			Source: "closing_issue",
		}
	}

	// Step 3: two or more closing issues → ambiguous.
	if len(pr.ClosingIssues) > 1 {
		return formalIntentRecord{
			Status: "ambiguous",
			Source: "closing_issue",
		}
	}

	// Step 4: labels present → artifact-label fallback.
	if len(pr.Labels) > 0 {
		return formalIntentRecord{
			Status: "artifact_label",
			Source: "artifact_label",
		}
	}

	// Step 5: no source → unlinked.
	return formalIntentRecord{
		Status: "unlinked",
		Source: "none",
	}
}

// formalSafestPolicy returns the unconditional fail-closed execution policy
// defined in specs/intent-attribution-agent-governance.md § Fail-Closed Behavior.
// stub — replace with real implementation
func formalSafestPolicy() formalExecutionPolicy {
	return formalExecutionPolicy{
		Autonomy:              "propose_only",
		WriteScope:            "none",
		HumanApprovalRequired: true,
		AutoMergeAllowed:      false,
		MaxAttempts:           1,
	}
}

// formalDerivePolicy compiles an execution policy from an intent record.
// Indeterminate attribution statuses always produce the safest policy,
// regardless of any caller-requested policy escalation.
// stub — replace with real implementation
func formalDerivePolicy(rec formalIntentRecord) formalExecutionPolicy {
	switch rec.Status {
	case "unlinked", "ambiguous", "suggested", "unmapped":
		return formalSafestPolicy()
	case "mapped", "artifact_label":
		return formalExecutionPolicy{
			Autonomy:              "supervised",
			WriteScope:            "feature_branch",
			HumanApprovalRequired: true,
			AutoMergeAllowed:      false,
			MaxAttempts:           2,
		}
	default:
		return formalSafestPolicy()
	}
}

// formalResolveRisk derives a risk classification from an intent record.
// Explicit risk wins; otherwise rules are derived from domain and priority.
// stub — replace with real implementation
func formalResolveRisk(rec formalIntentRecord) string {
	if rec.Risk != "" {
		return rec.Risk
	}

	for _, d := range rec.Domains {
		if d == "security" && rec.Priority == "critical" {
			return "high"
		}
	}

	for _, d := range rec.Domains {
		switch d {
		case "production":
			return "high"
		case "infrastructure":
			return "medium"
		case "documentation":
			return "low"
		}
	}

	return "unknown"
}

// formalPolicyPrecedenceLevel returns a numeric precedence for a policy source
// name.  Higher value means higher precedence (wins when two sources conflict).
//
// Ordering: agent_request(0) < workflow(1) < intent(2) < repository(3) < organization(4)
// stub — replace with real implementation
func formalPolicyPrecedenceLevel(source string) int {
	switch source {
	case "agent_request":
		return 0
	case "workflow":
		return 1
	case "intent":
		return 2
	case "repository":
		return 3
	case "organization":
		return 4
	default:
		return -1
	}
}

// ---------------------------------------------------------------------------
// Formal predicates (P1 – P11) and cross-cutting safety invariant
// ---------------------------------------------------------------------------

// TestFormal_P1_ExplicitIntentPrecedence verifies that explicit workflow intent
// always resolves first and overrides otherwise-ambiguous closing-issue
// candidates.
func TestFormal_P1_ExplicitIntentPrecedence(t *testing.T) {
	pr := pullRequestData{
		ExplicitIntent: &formalIntentKey{Key: "security"},
		ClosingIssues:  []int{101, 102}, // would be ambiguous without explicit intent
		Labels:         []string{"docs"},
	}
	rec := formalResolve(pr)
	require.Equal(t, "mapped", rec.Status)
	require.Equal(t, "explicit_metadata", rec.Source)
	require.Equal(t, "security", rec.Key)
}

// TestFormal_P2_SingleClosingIssueMapped verifies that exactly one closing
// issue resolves to status "mapped" with source "closing_issue".
func TestFormal_P2_SingleClosingIssueMapped(t *testing.T) {
	pr := pullRequestData{
		ExplicitIntent: nil,
		ClosingIssues:  []int{55},
		Labels:         []string{"enhancement"},
	}
	rec := formalResolve(pr)
	require.Equal(t, "mapped", rec.Status)
	require.Equal(t, "closing_issue", rec.Source)
}

// TestFormal_P3_AmbiguousOnMultipleRoots verifies that two or more closing
// issues always resolve to "ambiguous", regardless of the order in which
// issues are listed.
func TestFormal_P3_AmbiguousOnMultipleRoots(t *testing.T) {
	cases := []struct {
		name   string
		issues []int
	}{
		{"ascending", []int{1, 2}},
		{"descending", []int{2, 1}},
		{"three issues", []int{3, 1, 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := formalResolve(pullRequestData{ClosingIssues: tc.issues})
			assert.Equal(t, "ambiguous", rec.Status)
			assert.Equal(t, "closing_issue", rec.Source)
		})
	}
}

// TestFormal_P4_ArtifactLabelFallback verifies that zero closing issues with
// labels present falls back to artifact-label mapping.
func TestFormal_P4_ArtifactLabelFallback(t *testing.T) {
	pr := pullRequestData{
		ExplicitIntent: nil,
		ClosingIssues:  nil,
		Labels:         []string{"documentation"},
	}
	rec := formalResolve(pr)
	require.Equal(t, "artifact_label", rec.Status)
	require.Equal(t, "artifact_label", rec.Source)
}

// TestFormal_P5_UnlinkedWhenNoSource verifies that zero closing issues and
// zero labels resolves to "unlinked".
func TestFormal_P5_UnlinkedWhenNoSource(t *testing.T) {
	pr := pullRequestData{}
	rec := formalResolve(pr)
	require.Equal(t, "unlinked", rec.Status)
	require.Equal(t, "none", rec.Source)
}

// TestFormal_P6_AmbiguousNotMapped verifies that an ambiguous attribution
// status is never equivalent to mapped for authorization purposes.
func TestFormal_P6_AmbiguousNotMapped(t *testing.T) {
	ambiguous := formalResolve(pullRequestData{ClosingIssues: []int{10, 20}})
	mapped := formalResolve(pullRequestData{ClosingIssues: []int{10}})

	require.Equal(t, "ambiguous", ambiguous.Status)
	require.Equal(t, "mapped", mapped.Status)
	assert.NotEqual(t, ambiguous.Status, mapped.Status)
}

// TestFormal_P7_FailClosedPolicy verifies that indeterminate attribution
// statuses always yield the safest execution policy, regardless of any
// elevated request.  Table-driven with 4 cases.
func TestFormal_P7_FailClosedPolicy(t *testing.T) {
	safe := formalSafestPolicy()

	cases := []struct {
		status string
	}{
		{"unlinked"},
		{"ambiguous"},
		{"suggested"},
		{"unmapped"},
	}
	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			policy := formalDerivePolicy(formalIntentRecord{Status: tc.status})
			assert.Equal(t, safe.Autonomy, policy.Autonomy)
			assert.Equal(t, safe.WriteScope, policy.WriteScope)
			assert.Equal(t, safe.HumanApprovalRequired, policy.HumanApprovalRequired)
			assert.Equal(t, safe.AutoMergeAllowed, policy.AutoMergeAllowed)
			assert.Equal(t, safe.MaxAttempts, policy.MaxAttempts)
		})
	}
}

// TestFormal_P8_PolicyDeterminism verifies that identical attribution inputs
// always produce identical policy output across repeated calls.
func TestFormal_P8_PolicyDeterminism(t *testing.T) {
	const iterations = 10

	inputs := []formalIntentRecord{
		{Status: "mapped", Source: "explicit_metadata", Key: "security"},
		{Status: "ambiguous", Source: "closing_issue"},
		{Status: "unlinked", Source: "none"},
		{Status: "artifact_label", Source: "artifact_label"},
	}

	for _, rec := range inputs {
		first := formalDerivePolicy(rec)
		for range iterations - 1 {
			got := formalDerivePolicy(rec)
			assert.Equal(t, first, got, "non-deterministic policy for status=%s", rec.Status)
		}
	}
}

// TestFormal_P9_RiskClassificationOrder verifies that explicit risk wins;
// otherwise rules are derived from domain and priority.  Table-driven with
// 6 cases covering all documented derivation paths including the unknown-domain
// edge case.
func TestFormal_P9_RiskClassificationOrder(t *testing.T) {
	cases := []struct {
		name     string
		rec      formalIntentRecord
		wantRisk string
	}{
		{
			name:     "explicit risk overrides domain rules",
			rec:      formalIntentRecord{Risk: "low", Domains: []string{"security"}, Priority: "critical"},
			wantRisk: "low",
		},
		{
			name:     "security domain + critical priority = high",
			rec:      formalIntentRecord{Domains: []string{"security"}, Priority: "critical"},
			wantRisk: "high",
		},
		{
			name:     "production domain = high",
			rec:      formalIntentRecord{Domains: []string{"production"}},
			wantRisk: "high",
		},
		{
			name:     "infrastructure domain = medium",
			rec:      formalIntentRecord{Domains: []string{"infrastructure"}},
			wantRisk: "medium",
		},
		{
			name:     "documentation domain = low",
			rec:      formalIntentRecord{Domains: []string{"documentation"}},
			wantRisk: "low",
		},
		{
			name:     "unknown domain = unknown",
			rec:      formalIntentRecord{Domains: []string{"custom-domain"}},
			wantRisk: "unknown",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.wantRisk, formalResolveRisk(tc.rec))
		})
	}
}

// TestFormal_P10_PrecedenceOrdering verifies that
// organization > repository > intent > workflow > agent request.  Includes a
// non-adjacent domination check (organization dominates agent_request directly).
func TestFormal_P10_PrecedenceOrdering(t *testing.T) {
	adjacent := [][2]string{
		{"organization", "repository"},
		{"repository", "intent"},
		{"intent", "workflow"},
		{"workflow", "agent_request"},
	}

	for _, pair := range adjacent {
		higher, lower := pair[0], pair[1]
		assert.Greater(
			t,
			formalPolicyPrecedenceLevel(higher),
			formalPolicyPrecedenceLevel(lower),
			"%s must dominate %s", higher, lower,
		)
	}

	// Non-adjacent domination: organization must dominate agent_request.
	assert.Greater(
		t,
		formalPolicyPrecedenceLevel("organization"),
		formalPolicyPrecedenceLevel("agent_request"),
		"organization must dominate agent_request (non-adjacent)",
	)
}

// TestFormal_P11_NoElevatedAuthorityOnAbsentAttribution verifies that
// unresolved or unlinked attribution never grants elevated autonomy, auto-merge,
// or a max-attempts value greater than 1.
func TestFormal_P11_NoElevatedAuthorityOnAbsentAttribution(t *testing.T) {
	cases := []struct {
		name string
		pr   pullRequestData
	}{
		{"unlinked", pullRequestData{}},
		{"ambiguous", pullRequestData{ClosingIssues: []int{1, 2}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := formalResolve(tc.pr)
			policy := formalDerivePolicy(rec)

			assert.Equal(t, "propose_only", policy.Autonomy,
				"autonomy must not be elevated for status=%s", rec.Status)
			assert.False(t, policy.AutoMergeAllowed,
				"auto-merge must not be allowed for status=%s", rec.Status)
			assert.Equal(t, 1, policy.MaxAttempts,
				"max-attempts must be 1 for status=%s", rec.Status)
		})
	}
}

// TestFormal_SAFETY_AmbiguousAlwaysFailsClosed encodes the cross-cutting safety
// property: ambiguous attribution must fail closed even when a caller requests
// an elevated policy.  This invariant is unconditional and cuts across all
// other predicates.
func TestFormal_SAFETY_AmbiguousAlwaysFailsClosed(t *testing.T) {
	rec := formalResolve(pullRequestData{ClosingIssues: []int{10, 20}})
	require.Equal(t, "ambiguous", rec.Status)

	policy := formalDerivePolicy(rec)
	safe := formalSafestPolicy()

	assert.Equal(t, safe.Autonomy, policy.Autonomy,
		"SAFETY: ambiguous attribution must never grant autonomy above propose_only")
	assert.Equal(t, safe.WriteScope, policy.WriteScope,
		"SAFETY: ambiguous attribution must never grant write scope above none")
	assert.True(t, policy.HumanApprovalRequired,
		"SAFETY: ambiguous attribution must always require human approval")
	assert.False(t, policy.AutoMergeAllowed,
		"SAFETY: ambiguous attribution must never allow auto-merge")
	assert.Equal(t, safe.MaxAttempts, policy.MaxAttempts,
		"SAFETY: ambiguous attribution must limit max-attempts to 1")
}
