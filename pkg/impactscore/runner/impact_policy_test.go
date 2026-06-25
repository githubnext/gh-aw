//go:build !integration

package runner

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/impactscore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImpactPolicyScoresRules(t *testing.T) {
	outDir := t.TempDir()
	policyPath := filepath.Join(outDir, "aw.json")
	require.NoError(t, os.WriteFile(policyPath, []byte(`{
  "impact": {
    "version": 1,
    "base": 1,
    "clamp": { "min": 0, "max": 10 },
    "rules": [
      { "name": "ignore duplicate work", "when": { "any_label": ["duplicate"] }, "score": 0, "stop": true },
      { "name": "security work", "when": { "any_signal": ["security"] }, "min": 7 },
      { "name": "sensitive path boost", "when": { "measure_gt": { "sensitive_path_count": 0 } }, "add": 2 }
    ]
  }
}`), 0o644))

	policy, err := loadImpactPolicy(policyPath)
	require.NoError(t, err)
	options := applyImpactPolicy(impactscore.DefaultRankOptions(), policy, policyPath)

	securityScore := options.Score(impactscore.ItemFeatures{
		Item:     impactscore.WorkItem{Number: 1, Type: "issue", ContextSignals: []string{"security"}, SensitivePathCount: 1},
		Measures: map[string]float64{impactscore.MeasureSensitivePathCount: 1},
	})
	duplicateScore := options.Score(impactscore.ItemFeatures{
		Item:     impactscore.WorkItem{Number: 2, Type: "issue", Labels: []string{"duplicate"}, ContextSignals: []string{"security"}, SensitivePathCount: 1},
		Measures: map[string]float64{impactscore.MeasureSensitivePathCount: 1},
	})

	assert.InEpsilon(t, 9.0, securityScore.Score, 0.001)
	assert.Equal(t, "aw.json:sensitive path boost", securityScore.Source)
	assert.Equal(t, policyPath, securityScore.Explanation.PolicyPath)
	assert.Equal(t, 1, securityScore.Explanation.PolicyVersion)
	assert.Len(t, securityScore.Explanation.PolicySHA256, 64)
	assert.Equal(t, []string{"security work", "sensitive path boost"}, securityScore.Explanation.MatchedRules)
	assert.InDelta(t, 0.0, duplicateScore.Score, 0.001)
	assert.Equal(t, "aw.json:ignore duplicate work", duplicateScore.Source)
	assert.Equal(t, []string{"ignore duplicate work"}, duplicateScore.Explanation.MatchedRules)
}

func TestImpactPolicyMatchesDimensions(t *testing.T) {
	outDir := t.TempDir()
	policyPath := filepath.Join(outDir, "aw.json")
	require.NoError(t, os.WriteFile(policyPath, []byte(`{
  "impact": {
    "version": 1,
    "base": 1,
    "rules": [
      { "name": "ignore non-delivered closed work", "when": { "any_dimension": { "state_reason": ["not_planned", "closed_unmerged"] } }, "score": 0, "stop": true },
      { "name": "security work", "when": { "any_label": ["security"] }, "min": 7 }
    ]
  }
}`), 0o644))

	policy, err := loadImpactPolicy(policyPath)
	require.NoError(t, err)
	options := applyImpactPolicy(impactscore.DefaultRankOptions(), policy, policyPath)

	score := options.Score(impactscore.ItemFeatures{
		Item:       impactscore.WorkItem{Number: 1, Type: "issue", State: "closed", StateReason: "not_planned", Labels: []string{"security"}},
		Dimensions: map[string][]string{},
		Measures:   map[string]float64{},
	})

	assert.InDelta(t, 0.0, score.Score, 0.001)
	assert.Equal(t, []string{"ignore non-delivered closed work"}, score.Explanation.MatchedRules)
}

func TestImpactPolicyRejectsUnknownFields(t *testing.T) {
	outDir := t.TempDir()
	policyPath := filepath.Join(outDir, "aw.json")
	require.NoError(t, os.WriteFile(policyPath, []byte(`{
  "impact": {
    "version": 1,
    "rules": [
      { "name": "typo", "when": { "label_any": ["security"] }, "min": 7 }
    ]
  }
}`), 0o644))

	_, err := loadImpactPolicy(policyPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")
}

func TestImpactPolicyRejectsRuleWithoutOperation(t *testing.T) {
	outDir := t.TempDir()
	policyPath := filepath.Join(outDir, "aw.json")
	require.NoError(t, os.WriteFile(policyPath, []byte(`{
  "impact": {
    "version": 1,
    "rules": [
      { "name": "empty rule", "when": { "any_label": ["security"] } }
    ]
  }
}`), 0o644))

	_, err := loadImpactPolicy(policyPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `rule "empty rule" contains no operation`)
}

func TestLoadImpactPolicyAllowsRepoConfigFields(t *testing.T) {
	outDir := t.TempDir()
	policyPath := filepath.Join(outDir, "aw.json")
	repoConfig := `{
  "ghes": true,
  "utc": "-08:00",
  "maintenance": { "runs_on": "self-hosted" },
  "impact": {
    "version": 1,
    "rules": [
      { "name": "security work", "when": { "any_label": ["security"] }, "min": 7 }
    ]
  }
}`
	require.NoError(t, os.WriteFile(policyPath, []byte(repoConfig), 0o644))

	policy, err := loadImpactPolicy(policyPath)

	require.NoError(t, err)
	require.NotNil(t, policy)
	assert.Contains(t, impactPolicyRuleNames(policy.Rules), "security work")
}

func TestWriteImpactPolicyPreservesRepoConfigFields(t *testing.T) {
	outDir := t.TempDir()
	policyPath := filepath.Join(outDir, impactPolicyPath)
	repoConfig := `{
  "ghes": true,
  "utc": "-08:00",
  "maintenance": { "action_failure_issue_expires": 12 }
}`
	require.NoError(t, os.MkdirAll(filepath.Dir(policyPath), 0o755))
	require.NoError(t, os.WriteFile(policyPath, []byte(repoConfig), 0o644))

	err := writeImpactPolicy(policyPath, generateHistoryImpactPolicy("owner/repo", sourceData{}))

	require.NoError(t, err)
	data, err := os.ReadFile(policyPath)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(data, &doc))
	assert.Equal(t, true, doc["ghes"])
	assert.Equal(t, "-08:00", doc["utc"])
	assert.Contains(t, doc, "maintenance")
	assert.Contains(t, doc, "impact")
}

func TestGenerateHistoryImpactPolicyUsesRepoSignals(t *testing.T) {
	config := generateHistoryImpactPolicy("owner/repo", sourceData{Items: []impactscore.WorkItem{
		{Number: 1, Type: "issue", State: "closed", Labels: []string{"security"}, ContextSignals: []string{"security"}, Measures: map[string]float64{"impact_score": 9}},
		{Number: 2, Type: "issue", State: "closed", Labels: []string{"security"}, ContextSignals: []string{"security"}, Measures: map[string]float64{"impact_score": 8}},
		{Number: 3, Type: "issue", State: "closed", Labels: []string{"docs"}, Measures: map[string]float64{"impact_score": 2}},
	}})

	require.NotNil(t, config.Impact)
	assert.Equal(t, 1, config.Impact.Version)
	assert.NotEmpty(t, config.Impact.Rules)
	assert.Contains(t, impactPolicyRuleNames(config.Impact.Rules), "security work")
	assert.Contains(t, impactPolicyRuleNames(config.Impact.Rules), "history: label security")
}

func TestRunOnceUsesAWJSONImpactPolicy(t *testing.T) {
	outDir := t.TempDir()
	t.Chdir(outDir)
	require.NoError(t, os.MkdirAll(filepath.Dir(impactPolicyPath), 0o755))
	require.NoError(t, os.WriteFile(impactPolicyPath, []byte(`{
  "impact": {
    "version": 1,
    "base": 1,
    "clamp": { "min": 0, "max": 10 },
    "rules": [
      { "name": "security work", "when": { "any_label": ["security"] }, "min": 7 },
      { "name": "sensitive path boost", "when": { "measure_gt": { "sensitive_path_count": 0 } }, "add": 1 }
    ]
  }
}`), 0o644))
	require.NoError(t, writeJSON(filepath.Join(outDir, "items.json"), []impactscore.WorkItem{{
		Repo:               "owner/repo",
		Number:             1,
		Type:               "issue",
		State:              "open",
		Title:              "security workflow issue",
		Labels:             []string{"security"},
		Dimensions:         map[string][]string{},
		Measures:           map[string]float64{},
		SensitivePathCount: 1,
		SourceWorkflows:    []string{"triage"},
	}}))
	require.NoError(t, writeJSON(filepath.Join(outDir, "workflows.json"), []impactscore.WorkflowDefinition{{Name: "triage"}}))
	require.NoError(t, writeJSON(filepath.Join(outDir, "cost_runs.json"), []impactscore.WorkflowCostRun{{Workflow: "triage", RunID: "1", AICCost: 2, Source: "test"}}))

	result, err := runOnce(context.Background(), config{Repo: "owner/repo", OutDir: outDir, ReportFormat: "text"})

	require.NoError(t, err)
	require.Len(t, result.ItemRanks, 1)
	assert.InEpsilon(t, 8.0, result.ItemRanks[0].ImpactScore, 0.001)
	assert.Equal(t, "aw.json:sensitive path boost", result.ItemRanks[0].ScoreSource)
	assert.Equal(t, impactPolicyPath, result.ItemRanks[0].ScoreExplanation.PolicyPath)
	assert.Equal(t, 1, result.ItemRanks[0].ScoreExplanation.PolicyVersion)
	assert.Len(t, result.ItemRanks[0].ScoreExplanation.PolicySHA256, 64)
	assert.Equal(t, []string{"security work", "sensitive path boost"}, result.ItemRanks[0].ScoreExplanation.MatchedRules)
	assertCSVContains(t, filepath.Join(outDir, "items.csv"), "score_policy_sha256")
	assertCSVContains(t, filepath.Join(outDir, "items.csv"), "security work;sensitive path boost")
	assertCSVContains(t, filepath.Join(outDir, "impact_score_report.txt"), "rules=security work;sensitive path boost")
}

func TestRunOnceInitializesMissingImpactPolicy(t *testing.T) {
	outDir := t.TempDir()
	t.Chdir(outDir)
	policyPath := filepath.Join(outDir, impactPolicyPath)
	require.NoError(t, writeJSON(filepath.Join(outDir, "items.json"), []impactscore.WorkItem{{
		Repo:            "owner/repo",
		Number:          1,
		Type:            "issue",
		State:           "open",
		Title:           "docs issue",
		Labels:          []string{"docs"},
		Dimensions:      map[string][]string{},
		Measures:        map[string]float64{},
		SourceWorkflows: []string{"triage"},
	}}))
	require.NoError(t, writeJSON(filepath.Join(outDir, "workflows.json"), []impactscore.WorkflowDefinition{{Name: "triage"}}))
	require.NoError(t, writeJSON(filepath.Join(outDir, "cost_runs.json"), []impactscore.WorkflowCostRun{{Workflow: "triage", RunID: "1", AICCost: 2, Source: "test"}}))

	result, err := runOnce(context.Background(), config{Repo: "owner/repo", OutDir: outDir, ReportFormat: "text"})

	require.NoError(t, err)
	assert.FileExists(t, policyPath)
	require.Len(t, result.ItemRanks, 1)
	assert.InDelta(t, 3.0, result.ItemRanks[0].ImpactScore, 0.001)
	assert.Equal(t, "aw.json:docs work", result.ItemRanks[0].ScoreSource)
}

func impactPolicyRuleNames(rules []impactRule) []string {
	names := make([]string, 0, len(rules))
	for _, rule := range rules {
		names = append(names, rule.Name)
	}
	return names
}
