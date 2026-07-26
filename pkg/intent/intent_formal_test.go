//go:build !integration

package intent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
