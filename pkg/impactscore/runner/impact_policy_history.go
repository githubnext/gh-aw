package runner

import (
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/impactscore"
)

type historyScoreCandidate struct {
	Kind     string
	Signal   string
	Support  int
	Mean     float64
	Baseline float64
	Boost    float64
}

type historyAggregate struct {
	kind    string
	value   string
	support int
	sum     float64
}

func historicalItems(items []impactscore.WorkItem) []impactscore.WorkItem {
	history := []impactscore.WorkItem{}
	for _, item := range items {
		if item.State == "closed" || item.Released {
			history = append(history, item)
		}
	}
	if len(history) == 0 {
		return items
	}
	return history
}

func historyScoreCandidates(items []impactscore.WorkItem) []historyScoreCandidate {
	if len(items) == 0 {
		return nil
	}
	aggregates := map[string]*historyAggregate{}
	total := 0.0
	observed := 0
	for _, item := range items {
		value, ok := observedHistoryImpact(item)
		if !ok {
			continue
		}
		total += value
		observed++
		addCandidateSignals(aggregates, item, value)
	}
	if observed == 0 {
		return nil
	}
	baseline := total / float64(observed)
	minSupport := 2
	if len(items) < 8 {
		minSupport = 1
	}
	candidates := []historyScoreCandidate{}
	for _, aggregate := range aggregates {
		if aggregate.support < minSupport {
			continue
		}
		mean := aggregate.sum / float64(aggregate.support)
		delta := mean - baseline
		if delta < 0.75 {
			continue
		}
		candidates = append(candidates, historyScoreCandidate{Kind: aggregate.kind, Signal: aggregate.value, Support: aggregate.support, Mean: mean, Baseline: baseline, Boost: minFloat(2, maxFloat(0.5, roundHalf(delta)))})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Boost != candidates[j].Boost {
			return candidates[i].Boost > candidates[j].Boost
		}
		if candidates[i].Support != candidates[j].Support {
			return candidates[i].Support > candidates[j].Support
		}
		return candidates[i].Signal < candidates[j].Signal
	})
	if len(candidates) > 8 {
		candidates = candidates[:8]
	}
	return candidates
}

func observedHistoryImpact(item impactscore.WorkItem) (float64, bool) {
	for _, key := range []string{"impact_score", "score"} {
		if value := item.Measures[key]; value > 0 {
			return minFloat(10, maxFloat(0, value)), true
		}
	}
	if item.Released && item.ReleaseNoteImportance > 0 {
		return minFloat(10, maxFloat(0, item.ReleaseNoteImportance)), true
	}
	return 0, false
}

func addCandidateSignals(aggregates map[string]*historyAggregate, item impactscore.WorkItem, impact float64) {
	for _, label := range item.Labels {
		addHistoryCandidate(aggregates, "label", label, impact)
	}
	for _, signal := range item.ContextSignals {
		addHistoryCandidate(aggregates, "signal", signal, impact)
	}
	for _, component := range item.Components {
		addHistoryCandidate(aggregates, "component", component, impact)
	}
	for _, area := range item.Areas {
		addHistoryCandidate(aggregates, "area", area, impact)
	}
	for _, workflow := range item.SourceWorkflows {
		addHistoryCandidate(aggregates, "source workflow", workflow, impact)
	}
	if item.SensitivePathCount > 0 {
		addHistoryCandidate(aggregates, "measure", "sensitive_path_count > 0", impact)
	}
	if item.ComponentCount >= 3 {
		addHistoryCandidate(aggregates, "measure", "component_count >= 3", impact)
	}
}

func addHistoryCandidate(aggregates map[string]*historyAggregate, kind, value string, impact float64) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	key := kind + "\x00" + value
	aggregate := aggregates[key]
	if aggregate == nil {
		aggregate = &historyAggregate{kind: kind, value: value}
		aggregates[key] = aggregate
	}
	aggregate.support++
	aggregate.sum += impact
}

func roundHalf(value float64) float64 {
	return float64(int(value*2+0.5)) / 2
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}
