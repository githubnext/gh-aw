//go:build !integration

package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/impactscore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteUIArtifact(t *testing.T) {
	outDir := t.TempDir()
	result := output{
		Repo:        "owner/repo",
		GeneratedAt: "2026-06-23T00:00:00Z",
		WorkflowRanks: []impactscore.WorkflowRank{{
			Workflow:              "workflow one",
			ActionZone:            "keep / scale",
			AttributedImpactScore: 5,
			LinkedItems:           2,
			TotalAICCost:          1,
			ActionMinutes:         4,
			ImpactPerAIC:          5,
		}},
		ItemRanks: []impactscore.ItemRank{
			{Number: 1, ItemType: "issue", State: "closed", StateReason: "completed", Title: "security fix", ImpactScore: 3, ScoreSource: "aw.json:security work", ScoreExplanation: impactscore.ScoreExplanation{PolicyPath: ".github/workflows/aw.json", PolicyVersion: 1, PolicySHA256: "0123456789abcdef", MatchedRules: []string{"security work"}}, SourceWorkflows: []string{"workflow one"}},
			{Number: 2, ItemType: "pr", State: "open", Title: "human fix", ImpactScore: 2, ScoreSource: "aw.json:bug work"},
		},
		Features: []impactscore.ItemFeatures{{
			Item:       impactscore.WorkItem{Repo: "owner/repo", Number: 1, Type: "issue", Title: "security fix", State: "closed"},
			Dimensions: map[string][]string{"label": {"security"}},
			Measures:   map[string]float64{"changed_files": 2},
		}},
	}

	require.NoError(t, writeUIArtifact(outDir, result))
	content, err := os.ReadFile(filepath.Join(outDir, "impact_score_dashboard.html"))
	require.NoError(t, err)
	html := string(content)

	assert.Contains(t, html, "Impact Score Dashboard")
	assert.Contains(t, html, "impactCostChart")
	assert.Contains(t, html, "impactRankChart")
	assert.Contains(t, html, "data-tab=\"workItems\">Work Items")
	assert.NotContains(t, html, "data-tab=\"otherWork\">Other Work")
	assert.Contains(t, html, "aw.json:security work")
	assert.Contains(t, html, "score_explanation")
	assert.Contains(t, html, "state_reason")
	assert.Contains(t, html, "itemStateText")
	assert.Contains(t, html, "policy_sha256")
	assert.Contains(t, html, "scoreExplanationText")
	assert.Contains(t, html, "Workflow Work Items")
	assert.Contains(t, html, "workflowWorkItems")
	assert.Contains(t, html, "workflowWorkFilter")
	assert.Contains(t, html, "renderScoredWorkItems")
	assert.Contains(t, html, "renderItemTable")
	assert.Contains(t, html, "Other Work Items")
	assert.Contains(t, html, "workflow one")
	assert.Contains(t, html, "human fix")
	assert.Contains(t, html, "no linked agentic workflow")
	assert.Contains(t, html, "renderOtherWorkItems")
	assert.Contains(t, html, "displayZone")
	assert.Contains(t, html, "quadrantLabel")
	assert.Contains(t, html, "quadrantRect")
	assert.Contains(t, html, "target=\"_blank\"")
	assert.Contains(t, html, "rel=\"noopener noreferrer\"")
	assert.NotContains(t, html, "Top workflow")
	assert.NotContains(t, html, "data-tab=\"scoring\">Scoring")
	assert.NotContains(t, html, "workflowScoringItems")
	assert.NotContains(t, html, "scoringItems")
	assert.NotContains(t, html, "itemFilter")
	assert.NotContains(t, html, "workflowItemFilter")

	assert.NotContains(t, html, "Config Graph Preview")
	assert.NotContains(t, html, "Impact tuning")
	assert.NotContains(t, html, "addDirectImpactRule")
	assert.NotContains(t, html, "graphEditor")
	assert.NotContains(t, html, "Bootstrap starter model")
	assert.NotContains(t, html, "Bootstrap creates an executable")
	assert.NotContains(t, html, "Scoring Code")
	assert.NotContains(t, html, "scoring-layout")
	assert.NotContains(t, html, "highlightCode")
	assert.NotContains(t, html, "copyBootstrapCommand")
	assert.NotContains(t, html, "copyRerunCommand")
	assert.NotContains(t, html, "Download policy")
	assert.NotContains(t, html, "These are the highest item scores from the current run")
	assert.NotContains(t, html, "--score-command")
	assert.NotContains(t, html, "status-key")
}
