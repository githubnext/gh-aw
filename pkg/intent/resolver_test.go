package intent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolverResolvePullRequestSingleClosingIssueMapped(t *testing.T) {
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

func TestResolverResolvePullRequestNoSourcesUnlinked(t *testing.T) {
	resolver := Resolver{}

	intent := resolver.ResolvePullRequest(PullRequestData{})

	assert.Equal(t, AttributionUnlinked, intent.Status)
	assert.Equal(t, SourceNone, intent.Source)
	assert.Equal(t, "no_supported_intent_source", intent.Rule)
}

func TestResolverResolvePullRequestMultipleClosingIssuesAmbiguous(t *testing.T) {
	resolver := Resolver{}

	intent := resolver.ResolvePullRequest(PullRequestData{
		ClosingIssues: []RootReference{{URL: "https://github.com/owner/repo/issues/1"}, {URL: "https://github.com/owner/repo/issues/2"}},
	})

	assert.Equal(t, AttributionAmbiguous, intent.Status)
	assert.Equal(t, SourceClosingIssue, intent.Source)
	assert.Equal(t, "multiple_closing_issues", intent.Rule)
	assert.Empty(t, intent.RootURL)
}

func TestResolverResolveIssueMapped(t *testing.T) {
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
