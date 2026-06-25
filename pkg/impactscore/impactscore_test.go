//go:build !integration

package impactscore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRankWorkflowsUsesAllWorkflowLinkedItems(t *testing.T) {
	items := []ItemRank{
		{Number: 1, State: "closed", Released: true, ImpactScore: 4.0, SourceWorkflows: []string{"triage"}},
		{Number: 2, State: "open", ImpactScore: 2.0, SourceWorkflows: []string{"triage"}},
		{Number: 3, State: "closed", Released: true, ImpactScore: 3.0, SourceWorkflows: []string{"expensive"}},
	}
	costRuns := []WorkflowCostRun{
		{Workflow: "triage", RunID: "1", AICCost: 2.0, Source: "logs"},
		{Workflow: "expensive", RunID: "2", AICCost: 30.0, Source: "logs"},
	}

	ranks := RankWorkflows(items, costRuns)
	byWorkflow := map[string]WorkflowRank{}
	for _, rank := range ranks {
		byWorkflow[rank.Workflow] = rank
	}

	require.Contains(t, byWorkflow, "triage")
	assert.InDelta(t, 6.0, byWorkflow["triage"].AttributedImpactScore, 0.001)
	assert.Equal(t, 2, byWorkflow["triage"].LinkedItems)
	assert.Equal(t, 1, byWorkflow["triage"].OpenItems)
	assert.Equal(t, 1, byWorkflow["triage"].ReleasedItems)
	assert.InDelta(t, 3.0, byWorkflow["triage"].ImpactPerAIC, 0.001)
}

func TestRankWorkflowsMarksImpactWithoutCostAsNeedsCost(t *testing.T) {
	ranks := RankWorkflows([]ItemRank{
		{Number: 1, State: "open", ImpactScore: 2.0, SourceWorkflows: []string{"triage"}},
	}, nil)

	require.Len(t, ranks, 1)
	assert.Equal(t, "needs cost", ranks[0].ActionZone)
}

func TestRankWorkflowsKeepsZeroImpactMetricsFinite(t *testing.T) {
	ranks := RankWorkflows(nil, []WorkflowCostRun{{Workflow: "expensive", RunID: "1", AICCost: 10}})

	require.Len(t, ranks, 1)
	assert.InDelta(t, 0.0, ranks[0].AICPerImpact, 0.001)
	assert.Equal(t, "waste review", ranks[0].ActionZone)
}

func TestCustomScoreSupportsRepoSpecificPolicy(t *testing.T) {
	graph := BuildGraph([]WorkItem{
		{
			Number:          1,
			Type:            "issue",
			State:           "closed",
			Title:           "customer escalation fixed",
			Labels:          []string{"customer-impact", "sev1"},
			Components:      []string{"api"},
			SourceWorkflows: []string{"support-triage"},
		},
		{
			Number:          2,
			Type:            "issue",
			State:           "open",
			Title:           "new customer escalation",
			Labels:          []string{"customer-impact", "sev1"},
			Components:      []string{"api"},
			SourceWorkflows: []string{"support-triage"},
		},
	}, nil)
	options := RankOptions{
		Score: func(features ItemFeatures) ItemScore {
			if containsString(features.Item.Labels, "sev1") {
				return ItemScore{Score: 5.0, Prior: 5.0, Confidence: 1, Support: 1, Source: "sev1_policy", Explanation: ScoreExplanation{PolicyPath: ".github/workflows/aw.json", PolicyVersion: 1, MatchedRules: []string{"sev1"}}}
			}
			return ItemScore{Source: "repo_policy"}
		},
	}

	ranks := RankItemsWithOptions(graph, options)

	require.Len(t, ranks, 2)
	assert.Equal(t, "sev1_policy", ranks[0].ScoreSource)
	assert.InDelta(t, 5.0, ranks[0].ImpactScore, 0.001)
	assert.Equal(t, ".github/workflows/aw.json", ranks[0].ScoreExplanation.PolicyPath)
	assert.Equal(t, []string{"sev1"}, ranks[0].ScoreExplanation.MatchedRules)
}

func TestBuildGraphIncludesCustomNodesAndEdges(t *testing.T) {
	graph := BuildGraph([]WorkItem{{
		Number: 1,
		Type:   "issue",
		State:  "open",
		GraphNodes: []Node{{
			ID:   "customer:enterprise",
			Type: "customer_segment",
			Name: "enterprise",
		}},
		GraphEdges: []Edge{{
			Target:   "customer:enterprise",
			Type:     "IMPACTS_CUSTOMER_SEGMENT",
			Evidence: "repo adapter",
		}},
	}}, nil)

	assert.Equal(t, "enterprise", graph.Nodes["customer:enterprise"].Name)
	require.Len(t, graph.In["customer:enterprise"], 1)
	assert.Equal(t, "work_item:issue:1", graph.In["customer:enterprise"][0].Source)
	assert.Equal(t, "IMPACTS_CUSTOMER_SEGMENT", graph.In["customer:enterprise"][0].Type)
}

func TestCustomScoreCanUseRepoSpecificMeasures(t *testing.T) {
	graph := BuildGraph([]WorkItem{
		{Number: 1, Type: "issue", State: "closed", Measures: map[string]float64{"customer_accounts": 12}},
	}, nil)
	options := RankOptions{
		Score: func(features ItemFeatures) ItemScore {
			accounts := features.Measures["customer_accounts"]
			return ItemScore{Score: accounts / 3, Prior: accounts / 3, Confidence: 1, Support: 1, Source: "customer_accounts"}
		},
	}

	ranks := RankItemsWithOptions(graph, options)

	require.Len(t, ranks, 1)
	assert.InDelta(t, 4.0, ranks[0].ImpactScore, 0.001)
	assert.Equal(t, "customer_accounts", ranks[0].ScoreSource)
	assert.Contains(t, StandardMeasureKeys(), MeasureReleaseNoteImportance)
}

func TestRankItemsAndFeaturesWithOptionsReturnsScoredFeatures(t *testing.T) {
	graph := BuildGraph([]WorkItem{{Number: 1, Type: "issue", State: "open", Labels: []string{"security"}}}, nil)
	options := RankOptions{Score: func(features ItemFeatures) ItemScore {
		if containsString(features.Dimensions["label"], "security") {
			return ItemScore{Score: 7, Source: "labels"}
		}
		return ItemScore{Source: "labels"}
	}}

	ranks, features := RankItemsAndFeaturesWithOptions(graph, options)

	require.Len(t, ranks, 1)
	require.Len(t, features, 1)
	assert.InDelta(t, 7.0, ranks[0].ImpactScore, 0.001)
	assert.Equal(t, []string{"security"}, features[0].Dimensions["label"])
}

func TestStateReasonIsExtractedAsStandardDimension(t *testing.T) {
	graph := BuildGraph([]WorkItem{{Number: 1, Type: "issue", State: "closed", StateReason: "not_planned"}}, nil)

	features := FeaturesForItem(graph, "work_item:issue:1")
	ranks := RankItemsWithOptions(graph, RankOptions{Score: func(features ItemFeatures) ItemScore {
		if containsString(features.Dimensions[DimensionStateReason], "not_planned") {
			return ItemScore{Score: 1, Source: "state_reason"}
		}
		return ItemScore{Source: "state_reason"}
	}})

	assert.Equal(t, []string{"not_planned"}, features.Dimensions[DimensionStateReason])
	require.Len(t, ranks, 1)
	assert.Equal(t, "not_planned", ranks[0].StateReason)
	assert.InDelta(t, 1.0, ranks[0].ImpactScore, 0.001)
}
