package impactscore

import (
	"math"
	"sort"
)

// RankItems scores all work items with the configured impact policy.
func RankItems(graph Graph) []ItemRank {
	return RankItemsWithOptions(graph, RankOptions{})
}

// RankItemsWithOptions scores all work items using a repository-specific impact
// policy over graph-extracted features.
func RankItemsWithOptions(graph Graph, options RankOptions) []ItemRank {
	ranks, _ := RankItemsAndFeaturesWithOptions(graph, options)
	return ranks
}

// RankItemsAndFeaturesWithOptions scores all work items and returns the feature
// records used for scoring, avoiding a second graph feature extraction pass.
func RankItemsAndFeaturesWithOptions(graph Graph, options RankOptions) ([]ItemRank, []ItemFeatures) {
	options = normalizeRankOptions(options)
	ranks := []ItemRank{}

	itemIDs := make([]string, 0, len(graph.Items))
	for itemID := range graph.Items {
		itemIDs = append(itemIDs, itemID)
	}
	sort.Strings(itemIDs)

	featuresList := make([]ItemFeatures, 0, len(itemIDs))
	for _, itemID := range itemIDs {
		features := featuresForItemWithOptions(graph, itemID, options)
		featuresList = append(featuresList, features)
		item := features.Item
		score := options.Score(features)

		ranks = append(ranks, ItemRank{
			Number:           item.Number,
			ItemType:         item.Type,
			State:            item.State,
			StateReason:      item.StateReason,
			Title:            item.Title,
			Released:         item.Released,
			ScoreSource:      score.Source,
			ScoreExplanation: cloneScoreExplanation(score.Explanation),
			ImpactScore:      round(score.Score, 3),
			SourceWorkflows:  features.Dimensions["source_workflow"],
		})
	}

	sort.SliceStable(ranks, func(i, j int) bool {
		if ranks[i].ImpactScore != ranks[j].ImpactScore {
			return ranks[i].ImpactScore > ranks[j].ImpactScore
		}
		return ranks[i].Number < ranks[j].Number
	})
	return ranks, featuresList
}

func cloneScoreExplanation(explanation ScoreExplanation) ScoreExplanation {
	explanation.MatchedRules = append([]string{}, explanation.MatchedRules...)
	return explanation
}

func namesForTargets(graph Graph, source, edgeType string) []string {
	seen := map[string]bool{}
	names := []string{}
	for _, edge := range graph.Out[source] {
		if edge.Type != edgeType || seen[edge.Target] {
			continue
		}
		seen[edge.Target] = true
		name := graph.Nodes[edge.Target].Name
		if name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func round(value float64, digits int) float64 {
	factor := math.Pow10(digits)
	return math.Round(value*factor) / factor
}
