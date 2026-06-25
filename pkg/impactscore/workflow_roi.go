package impactscore

import (
	"slices"
	"sort"
)

// RankWorkflows attributes value from every workflow-linked item and joins it
// with observed workflow-run cost. Duplicate cost runs are collapsed by workflow
// and run ID, keeping the highest-cost observation as the most complete record.
func RankWorkflows(items []ItemRank, costRuns []WorkflowCostRun) []WorkflowRank {
	valueByWorkflow := map[string]float64{}
	itemsByWorkflow := map[string]int{}
	for _, item := range items {
		workflows := cleanStrings(item.SourceWorkflows)
		if len(workflows) == 0 || item.ImpactScore <= 0 {
			continue
		}
		share := item.ImpactScore / float64(len(workflows))
		for _, workflow := range workflows {
			valueByWorkflow[workflow] += share
			itemsByWorkflow[workflow]++
		}
	}

	dedupedRuns := dedupeCostRuns(costRuns)
	ranks := map[string]*WorkflowRank{}
	for workflow, value := range valueByWorkflow {
		ranks[workflow] = &WorkflowRank{Workflow: workflow, AttributedImpactScore: value, LinkedItems: itemsByWorkflow[workflow]}
	}
	for _, item := range items {
		for _, workflow := range cleanStrings(item.SourceWorkflows) {
			rank := ranks[workflow]
			if rank == nil {
				rank = &WorkflowRank{Workflow: workflow}
				ranks[workflow] = rank
			}
			if item.State == "open" {
				rank.OpenItems++
			}
			if item.Released {
				rank.ReleasedItems++
			}
		}
	}
	for _, run := range dedupedRuns {
		if run.Workflow == "" {
			continue
		}
		rank := ranks[run.Workflow]
		if rank == nil {
			rank = &WorkflowRank{Workflow: run.Workflow}
			ranks[run.Workflow] = rank
		}
		rank.RunCount++
		if run.AICCost > 0 {
			rank.CostedRunCount++
		}
		rank.TotalAICCost += run.AICCost
		rank.TotalTokens += run.TokenUsage
		rank.TotalTurns += run.Turns
		rank.ActionMinutes += run.ActionMinutes
		rank.Errors += run.Errors
		if run.Source != "" && !containsString(rank.CostSources, run.Source) {
			rank.CostSources = append(rank.CostSources, run.Source)
		}
	}

	result := []WorkflowRank{}
	for _, rank := range ranks {
		if rank.RunCount > 0 {
			rank.AverageAICCostPerRun = rank.TotalAICCost / float64(rank.RunCount)
		}
		if rank.TotalAICCost > 0 {
			rank.ImpactPerAIC = rank.AttributedImpactScore / rank.TotalAICCost
			rank.ImpactPerThousandAIC = rank.ImpactPerAIC * 1000
		}
		if rank.AttributedImpactScore > 0 {
			rank.AICPerImpact = rank.TotalAICCost / rank.AttributedImpactScore
		}
		result = append(result, *rank)
	}

	valueThreshold, costThreshold := workflowThresholds(result)
	for index := range result {
		result[index].ActionZone = workflowActionZone(result[index], valueThreshold, costThreshold)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].ImpactPerAIC != result[j].ImpactPerAIC {
			return result[i].ImpactPerAIC > result[j].ImpactPerAIC
		}
		if result[i].AttributedImpactScore != result[j].AttributedImpactScore {
			return result[i].AttributedImpactScore > result[j].AttributedImpactScore
		}
		return result[i].Workflow < result[j].Workflow
	})
	return result
}

func dedupeCostRuns(costRuns []WorkflowCostRun) []WorkflowCostRun {
	best := map[string]WorkflowCostRun{}
	withoutID := []WorkflowCostRun{}
	for _, run := range costRuns {
		if run.Workflow == "" || run.RunID == "" {
			withoutID = append(withoutID, run)
			continue
		}
		key := run.Workflow + "\x00" + run.RunID
		current, ok := best[key]
		if !ok || run.AICCost > current.AICCost || (run.AICCost == current.AICCost && run.TokenUsage > current.TokenUsage) {
			best[key] = run
		}
	}
	result := make([]WorkflowCostRun, 0, len(best)+len(withoutID))
	for _, run := range best {
		result = append(result, run)
	}
	result = append(result, withoutID...)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Workflow != result[j].Workflow {
			return result[i].Workflow < result[j].Workflow
		}
		return result[i].RunID < result[j].RunID
	})
	return result
}

func workflowThresholds(ranks []WorkflowRank) (float64, float64) {
	values := []float64{}
	costs := []float64{}
	for _, rank := range ranks {
		if rank.AttributedImpactScore > 0 && rank.TotalAICCost > 0 {
			values = append(values, rank.AttributedImpactScore)
		}
		if rank.TotalAICCost > 0 {
			costs = append(costs, rank.TotalAICCost)
		}
	}
	return median(values), median(costs)
}

func workflowActionZone(rank WorkflowRank, impactThreshold, costThreshold float64) string {
	highImpact := rank.AttributedImpactScore >= impactThreshold && rank.AttributedImpactScore > 0
	highCost := rank.TotalAICCost >= costThreshold && rank.TotalAICCost > 0
	switch {
	case rank.AttributedImpactScore > 0 && rank.TotalAICCost <= 0:
		return "needs cost"
	case highImpact && !highCost:
		return "keep / scale"
	case highImpact && highCost:
		return "optimize"
	case highCost:
		return "waste review"
	default:
		return "monitor"
	}
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	ordered := append([]float64{}, values...)
	sort.Float64s(ordered)
	mid := len(ordered) / 2
	if len(ordered)%2 == 1 {
		return ordered[mid]
	}
	return (ordered[mid-1] + ordered[mid]) / 2
}

func containsString(values []string, target string) bool {
	return slices.Contains(values, target)
}
