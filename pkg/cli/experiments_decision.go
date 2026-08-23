package cli

import (
	"fmt"
	"math"
)

const defaultDecisionConfidence = 0.95

// ExperimentDecision is the stable deterministic conclusion for an experiment.
type ExperimentDecision string

const (
	ExperimentDecisionExtend       ExperimentDecision = "EXTEND"
	ExperimentDecisionPromote      ExperimentDecision = "PROMOTE"
	ExperimentDecisionReject       ExperimentDecision = "REJECT"
	ExperimentDecisionInconclusive ExperimentDecision = "INCONCLUSIVE"
)

// ExperimentDecisionReasonCode is the stable machine-readable reason for a decision.
type ExperimentDecisionReasonCode string

const (
	ExperimentDecisionReasonInsufficientSamples      ExperimentDecisionReasonCode = "insufficient_samples"
	ExperimentDecisionReasonInsufficientObservations ExperimentDecisionReasonCode = "insufficient_observations"
	ExperimentDecisionReasonCandidateImproved        ExperimentDecisionReasonCode = "candidate_improved"
	ExperimentDecisionReasonCandidateRegressed       ExperimentDecisionReasonCode = "candidate_regressed"
	ExperimentDecisionReasonGuardrailFailed          ExperimentDecisionReasonCode = "guardrail_failed"
	ExperimentDecisionReasonGuardrailUnsupported     ExperimentDecisionReasonCode = "guardrail_unsupported"
	ExperimentDecisionReasonEffectBelowThreshold     ExperimentDecisionReasonCode = "effect_below_threshold"
	ExperimentDecisionReasonEvidenceInsufficient     ExperimentDecisionReasonCode = "evidence_insufficient"
	ExperimentDecisionReasonUnsupportedMultiVariant  ExperimentDecisionReasonCode = "unsupported_multi_variant"
	ExperimentDecisionReasonAnalysisUnavailable      ExperimentDecisionReasonCode = "analysis_unavailable"
)

// ExperimentDecisionPolicy is the normalized deterministic policy used to interpret analysis.
type ExperimentDecisionPolicy struct {
	MinimumEffect       float64 `json:"minimum_effect"`
	RegressionTolerance float64 `json:"regression_tolerance"`
	Confidence          float64 `json:"confidence"`
}

// ExperimentDecisionEffect reports the candidate effect in primary-metric units.
type ExperimentDecisionEffect struct {
	Absolute           float64  `json:"absolute"`
	Relative           *float64 `json:"relative,omitempty"`
	NormalizedAbsolute float64  `json:"normalized_absolute"`
}

// ExperimentDecisionEvidence reports the analyzer evidence consumed by the decision layer.
type ExperimentDecisionEvidence struct {
	AnalysisType           string   `json:"analysis_type"`
	Significant            bool     `json:"significant"`
	PValue                 *float64 `json:"p_value,omitempty"`
	ProbabilitySuperiority *float64 `json:"probability_superiority,omitempty"`
}

// ExperimentDecisionGuardrails summarizes mandatory guardrail outcomes.
type ExperimentDecisionGuardrails struct {
	Configured bool  `json:"configured"`
	Passed     *bool `json:"passed,omitempty"`
}

// ExperimentDecisionResult is the stable, machine-readable deterministic recommendation.
type ExperimentDecisionResult struct {
	Decision       ExperimentDecision           `json:"decision"`
	ReasonCode     ExperimentDecisionReasonCode `json:"reason_code"`
	DecisionReason string                       `json:"reason"`
	Control        string                       `json:"control,omitempty"`
	Candidate      string                       `json:"candidate,omitempty"`
	Direction      string                       `json:"direction,omitempty"`
	Samples        map[string]int               `json:"samples,omitempty"`
	Effect         *ExperimentDecisionEffect    `json:"effect,omitempty"`
	Evidence       *ExperimentDecisionEvidence  `json:"evidence,omitempty"`
	Guardrails     ExperimentDecisionGuardrails `json:"decision_guardrails"`
	Policy         ExperimentDecisionPolicy     `json:"decision_policy"`
}

// DecideExperiment deterministically transforms an existing analysis result into a decision.
// It performs no I/O, statistical tests, artifact loading, or experiment mutation.
func DecideExperiment(analysis ExperimentAnalysis) ExperimentDecisionResult {
	result := newExperimentDecisionResult(analysis)
	result.Samples = decisionSampleCounts(analysis.Variants, analysis.UsesMetricObservations)
	result.Control, result.Candidate = decisionVariants(analysis)
	if len(analysis.Variants) < 2 {
		return result.withDecision(ExperimentDecisionExtend, ExperimentDecisionReasonInsufficientObservations,
			"at least two variants are required for a decision")
	}
	if len(analysis.Variants) != 2 {
		return result.withDecision(ExperimentDecisionInconclusive, ExperimentDecisionReasonUnsupportedMultiVariant,
			"automatic decisions currently require exactly two variants")
	}
	if hasVariantBelowMinimum(analysis.Variants) {
		return result.withDecision(ExperimentDecisionExtend, ExperimentDecisionReasonInsufficientSamples,
			"one or more variants have fewer usable observations than min_samples")
	}

	if guardrailDecision, decided := decideGuardrails(result, analysis.Guardrails); decided {
		return guardrailDecision
	}

	comparison := decisionComparison(analysis.Comparisons)
	if comparison == nil {
		return result.withDecision(ExperimentDecisionExtend, ExperimentDecisionReasonInsufficientObservations,
			"primary metric observations are not available for statistical comparison")
	}
	result.Control = comparison.ControlVariant
	result.Candidate = comparison.Variant

	if comparison.Error != "" {
		return result.withDecision(ExperimentDecisionExtend, ExperimentDecisionReasonAnalysisUnavailable,
			"statistical analysis is not computable: "+comparison.Error)
	}

	result.Effect = decisionEffect(analysis, *comparison)
	result.Evidence = decisionEvidence(result.Policy, *comparison)
	if result.Effect == nil || result.Evidence == nil {
		return result.withDecision(ExperimentDecisionExtend, ExperimentDecisionReasonAnalysisUnavailable,
			"effect or statistical evidence is unavailable")
	}

	normalized := result.Effect.NormalizedAbsolute
	improvementMaterial := normalized > 0 && normalized >= result.Policy.MinimumEffect
	regressionMaterial := normalized < 0 && -normalized > result.Policy.RegressionTolerance
	if result.Evidence.Significant && regressionMaterial {
		return result.withDecision(ExperimentDecisionReject, ExperimentDecisionReasonCandidateRegressed,
			"candidate materially regresses the primary metric with sufficient evidence")
	}
	if result.Evidence.Significant && improvementMaterial {
		return result.withDecision(ExperimentDecisionPromote, ExperimentDecisionReasonCandidateImproved,
			"candidate materially improves the primary metric with sufficient evidence and all guardrails pass")
	}
	if !result.Evidence.Significant {
		return result.withDecision(ExperimentDecisionInconclusive, ExperimentDecisionReasonEvidenceInsufficient,
			"minimum samples are available but the configured evidence threshold is not satisfied")
	}
	return result.withDecision(ExperimentDecisionInconclusive, ExperimentDecisionReasonEffectBelowThreshold,
		"the primary metric effect does not exceed the configured practical threshold")
}

func decideGuardrails(result ExperimentDecisionResult, guardrails []GuardrailStatus) (ExperimentDecisionResult, bool) {
	if code, reason, pending := pendingGuardrailDecision(guardrails); pending {
		return result.withDecision(ExperimentDecisionExtend, code, reason), true
	}
	if failedGuardrail := firstFailedGuardrail(guardrails); failedGuardrail != "" {
		return result.withDecision(ExperimentDecisionReject, ExperimentDecisionReasonGuardrailFailed,
			fmt.Sprintf("mandatory guardrail %q failed", failedGuardrail)), true
	}
	return ExperimentDecisionResult{}, false
}

func decisionVariants(analysis ExperimentAnalysis) (string, string) {
	if len(analysis.VariantOrder) == 2 {
		return analysis.VariantOrder[0], analysis.VariantOrder[1]
	}
	if len(analysis.Variants) == 2 {
		return analysis.Variants[0].Name, analysis.Variants[1].Name
	}
	return "", ""
}

func newExperimentDecisionResult(analysis ExperimentAnalysis) ExperimentDecisionResult {
	policy := analysis.DecisionPolicy
	if policy.Confidence <= 0 || policy.Confidence >= 1 {
		policy.Confidence = defaultDecisionConfidence
	}
	return ExperimentDecisionResult{
		Direction:  analysis.MetricDirection,
		Guardrails: ExperimentDecisionGuardrails{Configured: len(analysis.Guardrails) > 0, Passed: guardrailPassSummary(analysis.Guardrails)},
		Policy:     policy,
	}
}

func (result ExperimentDecisionResult) withDecision(
	decision ExperimentDecision,
	code ExperimentDecisionReasonCode,
	reason string,
) ExperimentDecisionResult {
	result.Decision = decision
	result.ReasonCode = code
	result.DecisionReason = reason
	return result
}

func hasVariantBelowMinimum(variants []VariantAnalysis) bool {
	for _, variant := range variants {
		if variant.BelowMinSamples {
			return true
		}
	}
	return false
}

func decisionComparison(comparisons []MetricComparison) *MetricComparison {
	if len(comparisons) != 1 {
		return nil
	}
	return &comparisons[0]
}

func decisionSampleCounts(variants []VariantAnalysis, usesMetricObservations bool) map[string]int {
	samples := make(map[string]int, len(variants))
	for _, variant := range variants {
		count := variant.Count
		if usesMetricObservations {
			count = variant.ObservationCount
		}
		samples[variant.Name] = count
	}
	return samples
}

func decisionEffect(analysis ExperimentAnalysis, comparison MetricComparison) *ExperimentDecisionEffect {
	direction := analysis.MetricDirection
	if direction != "min" && direction != "max" {
		return nil
	}
	normalized := comparison.Delta
	if direction == "min" {
		normalized = -normalized
	}
	effect := &ExperimentDecisionEffect{Absolute: comparison.Delta, NormalizedAbsolute: normalized}
	if controlMean, ok := variantMean(analysis.Variants, comparison.ControlVariant); ok && controlMean != 0 {
		relative := comparison.Delta / math.Abs(controlMean)
		if direction == "min" {
			relative = -relative
		}
		effect.Relative = &relative
	}
	return effect
}

func variantMean(variants []VariantAnalysis, name string) (float64, bool) {
	for _, variant := range variants {
		if variant.Name == name && variant.Mean != nil {
			return *variant.Mean, true
		}
	}
	return 0, false
}

func decisionEvidence(policy ExperimentDecisionPolicy, comparison MetricComparison) *ExperimentDecisionEvidence {
	evidence := &ExperimentDecisionEvidence{
		AnalysisType:           comparison.AnalysisType,
		PValue:                 comparison.PValue,
		ProbabilitySuperiority: comparison.ProbabilitySuperiority,
	}
	switch comparison.AnalysisType {
	case "bayesian_ab":
		if comparison.ProbabilitySuperiority == nil {
			return nil
		}
		probability := *comparison.ProbabilitySuperiority
		evidence.Significant = probability >= policy.Confidence ||
			probability <= 1-policy.Confidence
	default:
		if comparison.PValue == nil {
			return nil
		}
		evidence.Significant = *comparison.PValue <= 1-policy.Confidence
	}
	return evidence
}

func guardrailPassSummary(guardrails []GuardrailStatus) *bool {
	if len(guardrails) == 0 {
		passed := true
		return &passed
	}
	for _, guardrail := range guardrails {
		if guardrail.Passed == nil {
			return nil
		}
		if !*guardrail.Passed {
			passed := false
			return &passed
		}
	}
	passed := true
	return &passed
}

func pendingGuardrailDecision(guardrails []GuardrailStatus) (ExperimentDecisionReasonCode, string, bool) {
	for _, guardrail := range guardrails {
		if guardrail.Status == GuardrailStatusInsufficientObservations || guardrail.Status == GuardrailStatusMissing {
			return ExperimentDecisionReasonInsufficientObservations,
				fmt.Sprintf("mandatory guardrail %q lacks sufficient usable observations", guardrail.Name), true
		}
		if guardrail.Status == GuardrailStatusUnsupported {
			return ExperimentDecisionReasonGuardrailUnsupported,
				fmt.Sprintf("mandatory guardrail %q is not backed by a supported metric observation", guardrail.Name), true
		}
	}
	return "", "", false
}

func firstFailedGuardrail(guardrails []GuardrailStatus) string {
	for _, guardrail := range guardrails {
		if guardrail.Passed != nil && !*guardrail.Passed {
			return guardrail.Name
		}
	}
	return ""
}
