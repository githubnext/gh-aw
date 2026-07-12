//go:build !integration

// Package intent_test contains compliance fixture tests for the intent
// attribution resolver and policy compiler.
//
// Each fixture in specs/intent-attribution-compliance/ is loaded and validated
// against the public Resolver and PolicyCompiler APIs. Adding a new YAML fixture
// to that directory is sufficient to add it to this test run — no Go changes
// required.
package intent_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yamlv3 "gopkg.in/yaml.v3"

	"github.com/github/gh-aw/pkg/intent"
)

// ───────────────────────────────────────────────────
// Fixture schema types
// ───────────────────────────────────────────────────

type iaFixtureFile struct {
	FixtureID   string       `yaml:"fixture_id"`
	Description string       `yaml:"description"`
	Scenarios   []iaScenario `yaml:"scenarios"`
}

type iaScenario struct {
	ScenarioID  string     `yaml:"scenario_id"`
	Description string     `yaml:"description"`
	Input       iaInput    `yaml:"input"`
	Expected    iaExpected `yaml:"expected"`
}

type iaInput struct {
	PullRequest iaPullRequest `yaml:"pull_request"`
}

type iaPullRequest struct {
	NodeID         string          `yaml:"node_id"`
	URL            string          `yaml:"url"`
	Labels         []string        `yaml:"labels"`
	ExplicitIntent *iaIntentRecord `yaml:"explicit_intent"`
	ClosingIssues  []iaRootRef     `yaml:"closing_issues"`
}

type iaIntentRecord struct {
	Status          string `yaml:"status"`
	Source          string `yaml:"source"`
	Rule            string `yaml:"rule"`
	ResolverVersion string `yaml:"resolver_version"`
}

type iaRootRef struct {
	NodeID string   `yaml:"node_id"`
	Type   string   `yaml:"type"`
	URL    string   `yaml:"url"`
	Labels []string `yaml:"labels"`
}

type iaExpected struct {
	Attribution iaExpectedAttribution `yaml:"attribution"`
	Policy      *iaExpectedPolicy     `yaml:"policy"`
}

type iaExpectedAttribution struct {
	Status string `yaml:"status"`
	Source string `yaml:"source"`
	Rule   string `yaml:"rule"`
}

type iaExpectedPolicy struct {
	Autonomy              string `yaml:"autonomy"`
	WriteScope            string `yaml:"write_scope"`
	HumanApprovalRequired bool   `yaml:"human_approval_required"`
	AutoMergeAllowed      bool   `yaml:"auto_merge_allowed"`
	MaxAttempts           int    `yaml:"max_attempts"`
}

// ───────────────────────────────────────────────────
// Fixture runner
// ───────────────────────────────────────────────────

// TestCompliance_IntentAttributionFixtures loads every YAML fixture from
// specs/intent-attribution-compliance/ and validates each scenario against the
// Resolver and PolicyCompiler APIs.
func TestCompliance_IntentAttributionFixtures(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller failed")

	fixtureDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "specs", "intent-attribution-compliance")
	entries, err := os.ReadDir(fixtureDir)
	require.NoError(t, err, "reading compliance fixture directory")

	resolver := intent.Resolver{
		ResolverVersion: "test",
		// MatchLabels maps every non-empty label as matched (for fixture purposes).
		MatchLabels: func(labels []string) []string { return labels },
	}
	compiler := intent.PolicyCompiler{} // no rules → safe default for all

	var fixtureCount int

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		path := filepath.Join(fixtureDir, entry.Name())
		data, err := os.ReadFile(path)
		require.NoError(t, err, "reading %s", entry.Name())

		var fixture iaFixtureFile
		require.NoError(t, yamlv3.Unmarshal(data, &fixture),
			"parsing %s", entry.Name())

		for _, s := range fixture.Scenarios {
			t.Run(fixture.FixtureID+"/"+s.ScenarioID, func(t *testing.T) {
				fixtureCount++
				pr := buildPullRequestData(s.Input.PullRequest)
				record := resolver.ResolvePullRequest(pr)

				// Attribution assertions.
				exp := s.Expected.Attribution
				if exp.Status != "" {
					assert.Equal(t, intent.AttributionStatus(exp.Status), record.Status,
						"fixture %s/%s: attribution status", fixture.FixtureID, s.ScenarioID)
				}
				if exp.Source != "" {
					assert.Equal(t, intent.AttributionSource(exp.Source), record.Source,
						"fixture %s/%s: attribution source", fixture.FixtureID, s.ScenarioID)
				}
				if exp.Rule != "" {
					assert.Equal(t, exp.Rule, record.Rule,
						"fixture %s/%s: attribution rule", fixture.FixtureID, s.ScenarioID)
				}

				// Policy assertions (only when the fixture specifies an expected policy).
				if s.Expected.Policy != nil {
					policy := compiler.Compile(record, intent.RepositoryContext{})
					ep := s.Expected.Policy
					assert.Equal(t, ep.Autonomy, policy.Autonomy,
						"fixture %s/%s: policy.Autonomy", fixture.FixtureID, s.ScenarioID)
					assert.Equal(t, ep.WriteScope, policy.WriteScope,
						"fixture %s/%s: policy.WriteScope", fixture.FixtureID, s.ScenarioID)
					assert.Equal(t, ep.HumanApprovalRequired, policy.HumanApprovalRequired,
						"fixture %s/%s: policy.HumanApprovalRequired", fixture.FixtureID, s.ScenarioID)
					if policy.AutoMergeAllowed != nil {
						assert.Equal(t, ep.AutoMergeAllowed, *policy.AutoMergeAllowed,
							"fixture %s/%s: policy.AutoMergeAllowed", fixture.FixtureID, s.ScenarioID)
					} else {
						assert.False(t, ep.AutoMergeAllowed,
							"fixture %s/%s: policy.AutoMergeAllowed is nil, expected false",
							fixture.FixtureID, s.ScenarioID)
					}
					if ep.MaxAttempts > 0 {
						assert.Equal(t, ep.MaxAttempts, policy.MaxAttempts,
							"fixture %s/%s: policy.MaxAttempts", fixture.FixtureID, s.ScenarioID)
					}
				}
			})
		}
	}

	assert.Positive(t, fixtureCount,
		"at least one fixture scenario must be present in %s", fixtureDir)
}

// buildPullRequestData converts the YAML fixture's pull request input into the
// intent.PullRequestData type used by the Resolver.
func buildPullRequestData(pr iaPullRequest) intent.PullRequestData {
	data := intent.PullRequestData{
		NodeID: pr.NodeID,
		URL:    pr.URL,
		Labels: pr.Labels,
	}

	if pr.ExplicitIntent != nil {
		record := intent.IntentRecord{
			Status:          intent.AttributionStatus(pr.ExplicitIntent.Status),
			Source:          intent.AttributionSource(pr.ExplicitIntent.Source),
			Rule:            pr.ExplicitIntent.Rule,
			ResolverVersion: pr.ExplicitIntent.ResolverVersion,
		}
		data.ExplicitIntent = &record
	}

	for _, ref := range pr.ClosingIssues {
		data.ClosingIssues = append(data.ClosingIssues, intent.RootReference{
			NodeID: ref.NodeID,
			Type:   ref.Type,
			URL:    ref.URL,
			Labels: ref.Labels,
		})
	}

	return data
}
