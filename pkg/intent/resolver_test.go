package intent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolverResolvePullRequestSingleClosingIssueMapped(t *testing.T) {
	t.Parallel()
	resolver := Resolver{
		ResolverVersion: "test-v1",
		MatchLabels: func(labels []string) []string {
			if len(labels) == 0 {
				return nil
			}
			return []string{"security"}
		},
	}

	intent := resolver.ResolvePullRequest(PullRequestData{
		NodeID: "PR_kwDOAAABCD4",
		URL:    "https://github.com/owner/repo/pull/77",
		ClosingIssues: []RootReference{{
			NodeID: "I_kwDOAAABCQ4",
			Type:   "issue",
			URL:    "https://github.com/owner/repo/issues/1234",
			Labels: []string{"security", "critical"},
		}},
	})

	assert.Equal(t, AttributionMapped, intent.Status)
	assert.Equal(t, SourceClosingIssue, intent.Source)
	assert.Equal(t, "I_kwDOAAABCQ4", intent.RootNodeID)
	assert.Equal(t, "issue", intent.RootType)
	assert.Equal(t, "https://github.com/owner/repo/issues/1234", intent.RootURL)
	assert.Equal(t, []string{"security", "critical"}, intent.Labels)
	assert.Equal(t, "single_closing_issue", intent.Rule)
	assert.Equal(t, "test-v1", intent.ResolverVersion)
}

func TestResolverResolvePullRequestSingleClosingIssueUnmapped(t *testing.T) {
	t.Parallel()
	resolver := Resolver{
		MatchLabels: func(labels []string) []string {
			return nil
		},
	}

	intent := resolver.ResolvePullRequest(PullRequestData{
		ClosingIssues: []RootReference{{
			Type:   "issue",
			URL:    "https://github.com/owner/repo/issues/1234",
			Labels: []string{"triage"},
		}},
	})

	assert.Equal(t, AttributionUnmapped, intent.Status)
	assert.Equal(t, SourceClosingIssue, intent.Source)
	assert.Equal(t, "single_closing_issue", intent.Rule)
}

func TestResolverResolvePullRequestArtifactFallbackMapped(t *testing.T) {
	t.Parallel()
	resolver := Resolver{
		MatchLabels: func(labels []string) []string {
			return []string{"automation"}
		},
	}

	intent := resolver.ResolvePullRequest(PullRequestData{
		NodeID: "PR_kwDOAAABCD4",
		URL:    "https://github.com/owner/repo/pull/77",
		Labels: []string{"automation"},
	})

	assert.Equal(t, AttributionMapped, intent.Status)
	assert.Equal(t, SourceArtifactLabels, intent.Source)
	assert.Equal(t, "pull_request_label_fallback", intent.Rule)
	assert.Equal(t, "artifact", intent.RootType)
	assert.Equal(t, "https://github.com/owner/repo/pull/77", intent.RootURL)
}

func TestResolverResolvePullRequestArtifactFallbackClonesLabels(t *testing.T) {
	t.Parallel()
	resolver := Resolver{ResolverVersion: "test-v1"}

	labels := []string{"automation", "maintenance"}
	intent := resolver.ResolvePullRequest(PullRequestData{
		NodeID: "PR_kwDOAAABCD4",
		URL:    "https://github.com/owner/repo/pull/77",
		Labels: labels,
	})
	labels[0] = "mutated"

	assert.Equal(t, []string{"automation", "maintenance"}, intent.Labels)
}

func TestResolverResolvePullRequestExplicitIntentPreservesVersion(t *testing.T) {
	t.Parallel()
	resolver := Resolver{ResolverVersion: "test-v1"}

	explicit := IntentRecord{
		Status:          AttributionMapped,
		Source:          SourceExplicitMetadata,
		RootNodeID:      "I_kwDOAAABCQ4",
		RootType:        "issue",
		RootURL:         "https://github.com/owner/repo/issues/1234",
		Labels:          []string{"security"},
		Rule:            "explicit_metadata",
		ResolverVersion: "explicit-v9",
	}

	intent := resolver.ResolvePullRequest(PullRequestData{ExplicitIntent: &explicit})

	assert.Equal(t, explicit, intent)
}

func TestResolverResolvePullRequestExplicitIntentFillsVersion(t *testing.T) {
	t.Parallel()
	resolver := Resolver{ResolverVersion: "test-v1"}

	explicit := IntentRecord{
		Status: AttributionMapped,
		Source: SourceExplicitMetadata,
		Rule:   "explicit_metadata",
	}

	intent := resolver.ResolvePullRequest(PullRequestData{ExplicitIntent: &explicit})

	assert.Equal(t, IntentRecord{
		Status:          AttributionMapped,
		Source:          SourceExplicitMetadata,
		Rule:            "explicit_metadata",
		ResolverVersion: "test-v1",
	}, intent)
	assert.Empty(t, explicit.ResolverVersion, "explicit intent must not be mutated")
}

func TestResolverResolvePullRequestNoSourcesUnlinked(t *testing.T) {
	t.Parallel()
	resolver := Resolver{ResolverVersion: "test-v1"}

	intent := resolver.ResolvePullRequest(PullRequestData{})

	assert.Equal(t, IntentRecord{
		Status:          AttributionUnlinked,
		Source:          SourceNone,
		Rule:            "no_supported_intent_source",
		ResolverVersion: "test-v1",
	}, intent)
}

func TestResolverResolvePullRequestMultipleClosingIssuesAmbiguous(t *testing.T) {
	t.Parallel()
	resolver := Resolver{ResolverVersion: "test-v1"}

	intent := resolver.ResolvePullRequest(PullRequestData{
		ClosingIssues: []RootReference{{URL: "https://github.com/owner/repo/issues/1"}, {URL: "https://github.com/owner/repo/issues/2"}},
	})

	assert.Equal(t, IntentRecord{
		Status:          AttributionAmbiguous,
		Source:          SourceClosingIssue,
		Rule:            "multiple_closing_issues",
		ResolverVersion: "test-v1",
	}, intent)
}

func TestResolverResolvePullRequestNilMatchLabelsUnmapped(t *testing.T) {
	t.Parallel()
	resolver := Resolver{ResolverVersion: "test-v1"}

	intent := resolver.ResolvePullRequest(PullRequestData{
		NodeID: "PR_kwDOAAABCD4",
		URL:    "https://github.com/owner/repo/pull/77",
		Labels: []string{"automation"},
	})

	assert.Equal(t, IntentRecord{
		Status:          AttributionUnmapped,
		Source:          SourceArtifactLabels,
		RootNodeID:      "PR_kwDOAAABCD4",
		RootType:        "artifact",
		RootURL:         "https://github.com/owner/repo/pull/77",
		Labels:          []string{"automation"},
		Rule:            "pull_request_label_fallback",
		ResolverVersion: "test-v1",
	}, intent)
}

func TestResolverResolveIssueMapped(t *testing.T) {
	t.Parallel()
	resolver := Resolver{
		MatchLabels: func(labels []string) []string {
			return []string{"documentation"}
		},
	}

	intent := resolver.ResolveIssue("I_kwDOAAABCQ4", "https://github.com/owner/repo/issues/42", []string{"documentation"})

	assert.Equal(t, AttributionMapped, intent.Status)
	assert.Equal(t, SourceIssueLabels, intent.Source)
	assert.Equal(t, "issue_label_fallback", intent.Rule)
	assert.Equal(t, "issue", intent.RootType)
	assert.Equal(t, []string{"documentation"}, intent.Labels)
}

func TestResolverResolveIssueNoLabelsUnlinked(t *testing.T) {
	t.Parallel()
	resolver := Resolver{ResolverVersion: "test-v1"}

	intent := resolver.ResolveIssue("I_kwDOAAABCQ4", "https://github.com/owner/repo/issues/42", nil)

	assert.Equal(t, IntentRecord{
		Status:          AttributionUnlinked,
		Source:          SourceNone,
		Rule:            "no_supported_intent_source",
		ResolverVersion: "test-v1",
	}, intent)
}

func TestResolverResolveIssueUnmapped(t *testing.T) {
	t.Parallel()
	resolver := Resolver{
		ResolverVersion: "test-v1",
		MatchLabels: func(labels []string) []string {
			return nil
		},
	}

	intent := resolver.ResolveIssue("I_kwDOAAABCQ4", "https://github.com/owner/repo/issues/42", []string{"triage"})

	assert.Equal(t, IntentRecord{
		Status:          AttributionUnmapped,
		Source:          SourceIssueLabels,
		RootNodeID:      "I_kwDOAAABCQ4",
		RootType:        "issue",
		RootURL:         "https://github.com/owner/repo/issues/42",
		Labels:          []string{"triage"},
		Rule:            "issue_label_fallback",
		ResolverVersion: "test-v1",
	}, intent)
}

func TestResolverResolveIssueNilMatchLabelsUnmapped(t *testing.T) {
	t.Parallel()
	resolver := Resolver{ResolverVersion: "test-v1"}

	intent := resolver.ResolveIssue("I_kwDOAAABCQ4", "https://github.com/owner/repo/issues/42", []string{"triage"})

	assert.Equal(t, IntentRecord{
		Status:          AttributionUnmapped,
		Source:          SourceIssueLabels,
		RootNodeID:      "I_kwDOAAABCQ4",
		RootType:        "issue",
		RootURL:         "https://github.com/owner/repo/issues/42",
		Labels:          []string{"triage"},
		Rule:            "issue_label_fallback",
		ResolverVersion: "test-v1",
	}, intent)
}

func TestResolverResolveIssueClonesLabels(t *testing.T) {
	t.Parallel()
	resolver := Resolver{ResolverVersion: "test-v1"}

	labels := []string{"triage", "bug"}
	intent := resolver.ResolveIssue("I_kwDOAAABCQ4", "https://github.com/owner/repo/issues/42", labels)
	labels[0] = "mutated"

	assert.Equal(t, []string{"triage", "bug"}, intent.Labels)
}
