package cli

import (
	"math"
	"strings"

	"github.com/github/gh-aw/pkg/workflow"
)

const experimentOutcomeSchemaVersion = 1

type ExperimentOutcomeRecord struct {
	SchemaVersion  int                `json:"schema_version"`
	RunID          string             `json:"run_id"`
	Experiment     string             `json:"experiment"`
	Variant        string             `json:"variant"`
	HarnessVersion string             `json:"harness_version"`
	Assignment     AssignmentInfo     `json:"assignment"`
	Features       map[string]string  `json:"features,omitempty"`
	Outcomes       map[string]float64 `json:"outcomes"`
	TraceReference string             `json:"trace_reference,omitempty"`
}

type ContinualVariantSummary struct {
	Variant     string             `json:"variant"`
	Count       int                `json:"count"`
	Successes   int                `json:"successes"`
	QualityMean float64            `json:"quality_mean"`
	Metrics     map[string]float64 `json:"metrics,omitempty"`
}

type SegmentRetentionResult struct {
	Feature               string  `json:"feature"`
	Value                 string  `json:"value"`
	ControlCount          int     `json:"control_count"`
	CandidateCount        int     `json:"candidate_count"`
	QualityDelta          float64 `json:"quality_delta"`
	RegressionProbability float64 `json:"regression_probability"`
	Passed                bool    `json:"passed"`
}

type ContinualExperimentAnalysis struct {
	State                       string                    `json:"state"`
	Decision                    string                    `json:"decision"`
	Control                     ContinualVariantSummary   `json:"control"`
	Candidate                   ContinualVariantSummary   `json:"candidate"`
	QualityDelta                float64                   `json:"quality_delta"`
	ImprovementProbability      float64                   `json:"improvement_probability"`
	HarmProbability             float64                   `json:"harm_probability"`
	CriticalSegments            []SegmentRetentionResult  `json:"critical_segments,omitempty"`
	RecommendedCandidatePercent int                       `json:"recommended_candidate_percent,omitempty"`
	Rationale                   string                    `json:"rationale"`
	Ledger                      []ExperimentOutcomeRecord `json:"ledger,omitempty"`
}

func buildExperimentOutcomeLedger(
	experiment string,
	metric string,
	runs []ExperimentRunRecord,
	evals []evalResultRecord,
) []ExperimentOutcomeRecord {
	evalID, isEval := workflow.ParseExperimentMetricEvalReference(metric)
	if !isEval {
		evalID = metric
	}
	answers := make(map[string]string)
	for _, record := range evals {
		if record.ID == evalID && record.RunID != "" {
			answers[record.RunID] = strings.ToUpper(strings.TrimSpace(record.Answer))
		}
	}
	var ledger []ExperimentOutcomeRecord
	for _, run := range runs {
		variant, ok := run.Assignments[experiment]
		if !ok {
			continue
		}
		answer, ok := answers[run.RunID]
		if !ok || answer == "UNKNOWN" {
			continue
		}
		success := 0.0
		if answer == "YES" {
			success = 1
		} else if answer != "NO" {
			continue
		}
		ledger = append(ledger, ExperimentOutcomeRecord{
			SchemaVersion:  experimentOutcomeSchemaVersion,
			RunID:          run.RunID,
			Experiment:     experiment,
			Variant:        variant,
			HarnessVersion: run.HarnessVersion,
			Assignment:     run.Assignment[experiment],
			Features:       run.Features,
			Outcomes:       map[string]float64{metric: success, "success": success},
			TraceReference: run.RunID,
		})
	}
	return ledger
}

func evaluateContinualExperiment(name string, cfg *workflow.ExperimentConfig, ledger []ExperimentOutcomeRecord) *ContinualExperimentAnalysis {
	if cfg == nil || cfg.Continual == nil || len(cfg.Variants) < 2 {
		return nil
	}
	controlName, candidateName := cfg.Variants[0], cfg.Variants[1]
	metric := cfg.Continual.Objective.Metric
	control := summarizeContinualVariant(controlName, metric, ledger)
	candidate := summarizeContinualVariant(candidateName, metric, ledger)
	result := &ContinualExperimentAnalysis{
		State:     "RUNNING",
		Decision:  "INSUFFICIENT_DATA",
		Control:   control,
		Candidate: candidate,
		Ledger:    ledger,
		Rationale: "no comparable control and candidate outcomes",
	}
	if control.Count == 0 || candidate.Count == 0 {
		return result
	}

	result.QualityDelta = candidate.QualityMean - control.QualityMean
	minimum := cfg.Continual.Decision.MinimumObservations
	if minimum == 0 {
		minimum = cfg.MinSamples
	}
	if minimum == 0 {
		minimum = defaultMinSamples
	}
	confidence := cfg.Continual.Decision.Confidence
	if confidence == 0 {
		confidence = 0.95
	}
	improvement := cfg.Continual.Objective.MinimumImprovement
	tolerance := cfg.Continual.Decision.RegressionTolerance
	result.ImprovementProbability = betaDifferenceProbability(candidate, control, improvement)
	result.HarmProbability = betaDifferenceProbability(control, candidate, tolerance)
	result.CriticalSegments = evaluateCriticalSegments(cfg, ledger, metric, controlName, candidateName, minimum, confidence)

	for _, segment := range result.CriticalSegments {
		if !segment.Passed {
			result.State = "REJECTED"
			result.Decision = "REJECT"
			result.Rationale = "candidate regressed on a critical pre-treatment segment"
			return result
		}
	}
	if result.HarmProbability >= confidence {
		result.State = "REJECTED"
		result.Decision = "REJECT"
		result.Rationale = "candidate quality is materially worse than control"
		return result
	}
	if control.Count < minimum || candidate.Count < minimum {
		result.Decision = "CONTINUE"
		result.Rationale = "minimum observations have not been reached"
		return result
	}
	if result.ImprovementProbability >= confidence || costEquivalentPromotion(cfg, control, candidate, tolerance) {
		result.State = "VALIDATED"
		result.Decision = "PROMOTE"
		result.Rationale = "candidate satisfies quality and critical-segment retention thresholds"
		result.RecommendedCandidatePercent = nextRampStage(cfg.Continual)
		return result
	}
	result.Decision = "CONTINUE"
	result.Rationale = "evidence does not yet meet promotion or rejection thresholds"
	return result
}

func summarizeContinualVariant(variant, metric string, ledger []ExperimentOutcomeRecord) ContinualVariantSummary {
	result := ContinualVariantSummary{Variant: variant, Metrics: map[string]float64{}}
	sums := map[string]float64{}
	for _, record := range ledger {
		if record.Variant != variant {
			continue
		}
		value, ok := record.Outcomes[metric]
		if !ok {
			continue
		}
		result.Count++
		if value >= 0.5 {
			result.Successes++
		}
		for name, metricValue := range record.Outcomes {
			sums[name] += metricValue
		}
	}
	if result.Count > 0 {
		result.QualityMean = sums[metric] / float64(result.Count)
		for name, sum := range sums {
			result.Metrics[name] = sum / float64(result.Count)
		}
	}
	return result
}

func betaDifferenceProbability(a, b ContinualVariantSummary, threshold float64) float64 {
	alphaA, betaA := float64(a.Successes+1), float64(a.Count-a.Successes+1)
	alphaB, betaB := float64(b.Successes+1), float64(b.Count-b.Successes+1)
	mean := alphaA/(alphaA+betaA) - alphaB/(alphaB+betaB)
	variance := betaVariance(alphaA, betaA) + betaVariance(alphaB, betaB)
	if variance == 0 {
		if mean > threshold {
			return 1
		}
		return 0
	}
	z := (mean - threshold) / math.Sqrt(variance)
	return 0.5 * math.Erfc(-z/math.Sqrt2)
}

func betaVariance(alpha, beta float64) float64 {
	total := alpha + beta
	return alpha * beta / (total * total * (total + 1))
}

func evaluateCriticalSegments(
	cfg *workflow.ExperimentConfig,
	ledger []ExperimentOutcomeRecord,
	metric, control, candidate string,
	minimum int,
	confidence float64,
) []SegmentRetentionResult {
	var results []SegmentRetentionResult
	for _, feature := range cfg.Continual.Segments.Critical {
		values := map[string]struct{}{}
		for _, record := range ledger {
			if value := record.Features[feature]; value != "" {
				values[value] = struct{}{}
			}
		}
		for value := range values {
			var segmentLedger []ExperimentOutcomeRecord
			for _, record := range ledger {
				if record.Features[feature] == value {
					segmentLedger = append(segmentLedger, record)
				}
			}
			controlSummary := summarizeContinualVariant(control, metric, segmentLedger)
			candidateSummary := summarizeContinualVariant(candidate, metric, segmentLedger)
			tolerance := cfg.Continual.Decision.RegressionTolerance
			for _, guardrail := range cfg.GuardrailMetrics {
				if guardrail.Segment == feature+"="+value {
					tolerance = guardrail.MaxRegression
				}
			}
			probability := betaDifferenceProbability(controlSummary, candidateSummary, tolerance)
			enough := controlSummary.Count >= minimum && candidateSummary.Count >= minimum
			results = append(results, SegmentRetentionResult{
				Feature: feature, Value: value,
				ControlCount: controlSummary.Count, CandidateCount: candidateSummary.Count,
				QualityDelta:          candidateSummary.QualityMean - controlSummary.QualityMean,
				RegressionProbability: probability,
				Passed:                !enough || probability < confidence,
			})
		}
	}
	return results
}

func costEquivalentPromotion(cfg *workflow.ExperimentConfig, control, candidate ContinualVariantSummary, tolerance float64) bool {
	if !cfg.Continual.Decision.AllowCostPromotion || candidate.QualityMean < control.QualityMean-tolerance {
		return false
	}
	controlCost, controlOK := control.Metrics["aic"]
	candidateCost, candidateOK := candidate.Metrics["aic"]
	return controlOK && candidateOK && controlCost > 0 && candidateCost <= controlCost*0.8
}

func nextRampStage(cfg *workflow.ContinualExperimentConfig) int {
	if len(cfg.Ramp) == 0 {
		return 0
	}
	next := cfg.CurrentStage + 1
	if next >= len(cfg.Ramp) {
		next = len(cfg.Ramp) - 1
	}
	return cfg.Ramp[next]
}
