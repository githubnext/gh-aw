package cli

import (
	"cmp"
	"encoding/json"
	"slices"
	"sort"
	"time"
)

func buildOperationalValueReport(evaluator *operationalValueReportEvaluator, observations []operationalValueReportObservation, generatedAt time.Time, stats operationalValueReportBackfillStats) operationalValueReport {
	values := make([]float64, 0, len(observations))
	distinctNumericOpportunities := make(map[string]struct{})
	coverage := operationalValueReportCoverage{
		RunCount:        len(observations),
		WeeklyCacheHits: stats.CacheHits,
		EvaluatedCount:  stats.Evaluated,
	}
	for _, observation := range observations {
		if observation.Mature {
			coverage.MatureCount++
		}
		switch observation.Status {
		case "error":
			coverage.ErrorCount++
		case "unavailable":
			coverage.UnavailableCount++
		}
		if observation.Value != nil {
			coverage.NumericCount++
			values = append(values, *observation.Value)
			opportunityKey := observation.OpportunityKey
			if opportunityKey == "" {
				opportunityKey = "run:" + observation.Run.ID
			}
			distinctNumericOpportunities[opportunityKey] = struct{}{}
		}
	}
	coverage.DistinctOpportunityCount = len(distinctNumericOpportunities)
	coverage.DuplicateOpportunityCount = coverage.NumericCount - len(distinctNumericOpportunities)

	startAt := evaluator.Definition.Adoption.AdoptedAt
	report := operationalValueReport{
		SchemaVersion:    operationalValueReportSchemaVersion,
		GeneratedAt:      generatedAt.UTC().Format(time.RFC3339),
		Repository:       evaluator.Definition.Repository,
		WorkflowID:       evaluator.WorkflowID,
		WorkflowName:     evaluator.Definition.WorkflowName,
		SourcePath:       evaluator.Definition.SourcePath,
		OperationalValue: evaluator.Definition.OperationalValue,
		Window: operationalValueReportWindow{
			StartAt: startAt,
			EndAt:   generatedAt.UTC().Format(time.RFC3339),
		},
		Evaluator: operationalValueReportEvaluatorReference{
			Path:       evaluator.EvaluatorRun,
			SHA256:     evaluator.EvaluatorDigest,
			Definition: append(json.RawMessage(nil), evaluator.Definition.Raw...),
		},
		Baseline:     evaluator.Definition.Baseline,
		Coverage:     coverage,
		Summary:      summarizeOperationalValueValues(values, evaluator.Definition.Baseline.Value),
		Weekly:       buildOperationalValueReportWeeks(observations),
		Observations: observations,
		Caveat:       "Operational value is observational. Replayed historical values apply one frozen evaluator digest to every run and do not establish that the workflow caused the measured outcome.",
	}
	return report
}

func summarizeOperationalValueValues(values []float64, baseline *float64) operationalValueReportSummary {
	if len(values) == 0 {
		return operationalValueReportSummary{}
	}
	ordered := slices.Clone(values)
	slices.Sort(ordered)
	mean := averageOperationalValue(ordered)
	median := ordered[len(ordered)/2]
	if len(ordered)%2 == 0 {
		median = (ordered[len(ordered)/2-1] + ordered[len(ordered)/2]) / 2
	}
	first := values[0]
	latest := values[len(values)-1]
	change := latest - first
	summary := operationalValueReportSummary{
		Mean: &mean, Median: &median, Minimum: &ordered[0], Maximum: &ordered[len(ordered)-1],
		First: &first, Latest: &latest, Change: &change,
	}
	if baseline != nil {
		delta := latest - *baseline
		summary.LatestDeltaFromBaseline = &delta
	}
	return summary
}

func buildOperationalValueReportWeeks(observations []operationalValueReportObservation) []operationalValueReportWeek {
	grouped := make(map[time.Time][]operationalValueReportObservation)
	for _, observation := range observations {
		weekStart := operationalValueUTCWeekStart(observation.Run.CreatedAt)
		grouped[weekStart] = append(grouped[weekStart], observation)
	}
	weekStarts := make([]time.Time, 0, len(grouped))
	for weekStart := range grouped {
		weekStarts = append(weekStarts, weekStart)
	}
	slices.SortFunc(weekStarts, func(left, right time.Time) int { return cmp.Compare(left.Unix(), right.Unix()) })
	weeks := make([]operationalValueReportWeek, 0, len(weekStarts))
	for _, weekStart := range weekStarts {
		weekObservations := grouped[weekStart]
		latestByOpportunity := make(map[string]operationalValueReportObservation)
		for _, observation := range weekObservations {
			if observation.Value == nil {
				continue
			}
			key := observation.OpportunityKey
			if key == "" {
				key = "run:" + observation.Run.ID
			}
			previous, exists := latestByOpportunity[key]
			if !exists || previous.Run.CreatedAt.Before(observation.Run.CreatedAt) {
				latestByOpportunity[key] = observation
			}
		}
		values := make([]float64, 0, len(latestByOpportunity))
		for _, observation := range latestByOpportunity {
			values = append(values, *observation.Value)
		}
		sort.Float64s(values)
		week := operationalValueReportWeek{
			WeekStart:                weekStart.Format(time.RFC3339),
			WeekEnd:                  weekStart.AddDate(0, 0, 7).Format(time.RFC3339),
			RunCount:                 len(weekObservations),
			NumericCount:             len(values),
			DistinctOpportunityCount: len(latestByOpportunity),
		}
		if len(values) > 0 {
			mean := averageOperationalValue(values)
			week.Mean = &mean
			week.Minimum = &values[0]
			week.Maximum = &values[len(values)-1]
		}
		weeks = append(weeks, week)
	}
	return weeks
}

func averageOperationalValue(values []float64) float64 {
	var total float64
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}
