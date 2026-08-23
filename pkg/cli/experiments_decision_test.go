//go:build !integration

package cli

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecideExperiment(t *testing.T) {
	t.Parallel()
	pValueStrong := 0.01
	pValueWeak := 0.20
	bayesianStrong := 0.98
	bayesianWeak := 0.80
	passed := true
	failed := false

	tests := []struct {
		name       string
		analysis   ExperimentAnalysis
		want       ExperimentDecision
		reasonCode ExperimentDecisionReasonCode
		normalized float64
		hasEffect  bool
	}{
		{
			name:     "below minimum samples extends",
			analysis: decisionTestAnalysis("max", 0.2, &pValueStrong, nil, 0.05, nil),
			want:     ExperimentDecisionExtend, reasonCode: ExperimentDecisionReasonInsufficientSamples,
		},
		{
			name:     "max direction material improvement promotes",
			analysis: readyDecisionTestAnalysis("max", 0.2, &pValueStrong, nil, 0.05, nil),
			want:     ExperimentDecisionPromote, reasonCode: ExperimentDecisionReasonCandidateImproved, normalized: 0.2, hasEffect: true,
		},
		{
			name:     "min direction material improvement promotes",
			analysis: readyDecisionTestAnalysis("min", -1200, &pValueStrong, nil, 500, nil),
			want:     ExperimentDecisionPromote, reasonCode: ExperimentDecisionReasonCandidateImproved, normalized: 1200, hasEffect: true,
		},
		{
			name:     "max direction material regression rejects",
			analysis: readyDecisionTestAnalysis("max", -0.2, &pValueStrong, nil, 0.05, nil),
			want:     ExperimentDecisionReject, reasonCode: ExperimentDecisionReasonCandidateRegressed, normalized: -0.2, hasEffect: true,
		},
		{
			name:     "min direction material regression rejects",
			analysis: readyDecisionTestAnalysis("min", 1200, &pValueStrong, nil, 500, nil),
			want:     ExperimentDecisionReject, reasonCode: ExperimentDecisionReasonCandidateRegressed, normalized: -1200, hasEffect: true,
		},
		{
			name:     "significant tiny effect is inconclusive",
			analysis: readyDecisionTestAnalysis("max", 0.001, &pValueStrong, nil, 0.05, nil),
			want:     ExperimentDecisionInconclusive, reasonCode: ExperimentDecisionReasonEffectBelowThreshold,
		},
		{
			name:     "large effect without evidence is inconclusive",
			analysis: readyDecisionTestAnalysis("max", 0.2, &pValueWeak, nil, 0.05, nil),
			want:     ExperimentDecisionInconclusive, reasonCode: ExperimentDecisionReasonEvidenceInsufficient,
		},
		{
			name: "failed guardrail rejects before primary promotion",
			analysis: withDecisionTestGuardrail(
				readyDecisionTestAnalysis("max", 0.2, &pValueStrong, nil, 0.05, nil),
				GuardrailStatus{Name: "grader:failures", Status: GuardrailStatusFail, Passed: &failed},
			),
			want: ExperimentDecisionReject, reasonCode: ExperimentDecisionReasonGuardrailFailed,
		},
		{
			name: "failed guardrail rejects without primary comparison",
			analysis: withoutDecisionTestComparison(withDecisionTestGuardrail(
				readyDecisionTestAnalysis("max", 0.2, &pValueStrong, nil, 0.05, nil),
				GuardrailStatus{Name: "grader:failures", Status: GuardrailStatusFail, Passed: &failed},
			)),
			want: ExperimentDecisionReject, reasonCode: ExperimentDecisionReasonGuardrailFailed,
		},
		{
			name: "missing guardrail extends",
			analysis: withDecisionTestGuardrail(
				readyDecisionTestAnalysis("max", 0.2, &pValueStrong, nil, 0.05, nil),
				GuardrailStatus{Name: "grader:failures", Status: GuardrailStatusMissing},
			),
			want: ExperimentDecisionExtend, reasonCode: ExperimentDecisionReasonInsufficientObservations,
		},
		{
			name: "passed guardrail allows promotion",
			analysis: withDecisionTestGuardrail(
				readyDecisionTestAnalysis("max", 0.2, &pValueStrong, nil, 0.05, nil),
				GuardrailStatus{Name: "grader:failures", Status: GuardrailStatusPass, Passed: &passed},
			),
			want: ExperimentDecisionPromote, reasonCode: ExperimentDecisionReasonCandidateImproved,
		},
		{
			name:     "bayesian material improvement promotes",
			analysis: readyDecisionTestAnalysis("max", 0.2, nil, &bayesianStrong, 0.05, nil),
			want:     ExperimentDecisionPromote, reasonCode: ExperimentDecisionReasonCandidateImproved,
		},
		{
			name:     "bayesian uncertain result is inconclusive",
			analysis: readyDecisionTestAnalysis("max", 0.2, nil, &bayesianWeak, 0.05, nil),
			want:     ExperimentDecisionInconclusive, reasonCode: ExperimentDecisionReasonEvidenceInsufficient,
		},
		{
			name: "multi variant decision is explicitly unsupported",
			analysis: withThirdDecisionTestVariant(
				readyDecisionTestAnalysis("max", 0.2, &pValueStrong, nil, 0.05, nil),
			),
			want: ExperimentDecisionInconclusive, reasonCode: ExperimentDecisionReasonUnsupportedMultiVariant,
		},
		{
			name: "multi variant with undersampled arm is still unsupported",
			analysis: withThirdDecisionTestVariant(
				decisionTestAnalysis("max", 0.2, &pValueStrong, nil, 0.05, nil),
			),
			want: ExperimentDecisionInconclusive, reasonCode: ExperimentDecisionReasonUnsupportedMultiVariant,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := DecideExperiment(test.analysis)
			assert.Equal(t, test.want, result.Decision)
			assert.Equal(t, test.reasonCode, result.ReasonCode)
			assert.NotEmpty(t, result.DecisionReason)
			if test.hasEffect {
				require.NotNil(t, result.Effect)
				assert.InDelta(t, test.normalized, result.Effect.NormalizedAbsolute, 0.000001)
			}
		})
	}
}

func decisionTestAnalysis(
	direction string,
	delta float64,
	pValue *float64,
	probability *float64,
	minimumEffect float64,
	regressionTolerance *float64,
) ExperimentAnalysis {
	analysis := readyDecisionTestAnalysis(direction, delta, pValue, probability, minimumEffect, regressionTolerance)
	analysis.Variants[1].BelowMinSamples = true
	analysis.Variants[1].ObservationCount = 19
	return analysis
}

func readyDecisionTestAnalysis(
	direction string,
	delta float64,
	pValue *float64,
	probability *float64,
	minimumEffect float64,
	regressionTolerance *float64,
) ExperimentAnalysis {
	controlMean := 1.0
	candidateMean := controlMean + delta
	tolerance := minimumEffect
	if regressionTolerance != nil {
		tolerance = *regressionTolerance
	}
	analysisType := "t_test"
	if probability != nil {
		analysisType = "bayesian_ab"
	}
	return ExperimentAnalysis{
		ExperimentName:  "prompt_v2",
		Metric:          "grader:score",
		MetricDirection: direction,
		MinSamples:      20,
		Variants: []VariantAnalysis{
			{Name: "control", Count: 20, ObservationCount: 20, Mean: &controlMean},
			{Name: "candidate", Count: 20, ObservationCount: 20, Mean: &candidateMean},
		},
		Comparisons: []MetricComparison{{
			ControlVariant:         "control",
			Variant:                "candidate",
			AnalysisType:           analysisType,
			Delta:                  delta,
			PValue:                 pValue,
			ProbabilitySuperiority: probability,
		}},
		DecisionPolicy: ExperimentDecisionPolicy{
			MinimumEffect:       minimumEffect,
			RegressionTolerance: tolerance,
			Confidence:          defaultDecisionConfidence,
		},
	}
}

func withDecisionTestGuardrail(analysis ExperimentAnalysis, guardrail GuardrailStatus) ExperimentAnalysis {
	analysis.Guardrails = []GuardrailStatus{guardrail}
	return analysis
}

func withoutDecisionTestComparison(analysis ExperimentAnalysis) ExperimentAnalysis {
	analysis.Comparisons = nil
	return analysis
}

func withThirdDecisionTestVariant(analysis ExperimentAnalysis) ExperimentAnalysis {
	mean := 1.1
	analysis.Variants = append(analysis.Variants, VariantAnalysis{
		Name: "candidate_b", Count: 20, ObservationCount: 20, Mean: &mean,
	})
	return analysis
}

func TestApplyExperimentGuardrails(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		direction     string
		threshold     string
		controlValues []float64
		candidateVals []float64
		wantStatus    GuardrailStatusCode
		wantPassed    bool
		hasPassed     bool
	}{
		{
			name: "max guardrail passes", direction: "max", threshold: ">=0.8",
			controlValues: []float64{1, 1}, candidateVals: []float64{1, 1},
			wantStatus: GuardrailStatusPass, wantPassed: true, hasPassed: true,
		},
		{
			name: "min guardrail fails", direction: "min", threshold: "0",
			controlValues: []float64{0, 0}, candidateVals: []float64{1, 1},
			wantStatus: GuardrailStatusFail, wantPassed: false, hasPassed: true,
		},
		{
			name: "missing observations do not pass", direction: "max", threshold: ">=0.8",
			controlValues: []float64{1, 1}, candidateVals: []float64{1},
			wantStatus: GuardrailStatusInsufficientObservations,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			analysis := ExperimentAnalysis{
				MinSamples: 2,
				Variants:   []VariantAnalysis{{Name: "control"}, {Name: "candidate"}},
				Guardrails: []GuardrailStatus{{
					Name: "grader:guardrail", Direction: test.direction, Threshold: test.threshold,
					Status: GuardrailStatusUnsupported,
				}},
			}
			set := &graderMetricObservationSet{ByVariant: map[string][]GraderMetricObservation{
				"control":   decisionTestObservations("control", test.controlValues),
				"candidate": decisionTestObservations("candidate", test.candidateVals),
			}}
			applyExperimentGuardrails(&analysis, map[string]*graderMetricObservationSet{"grader:guardrail": set})

			require.Len(t, analysis.Guardrails, 1)
			assert.Equal(t, test.wantStatus, analysis.Guardrails[0].Status)
			if !test.hasPassed {
				assert.Nil(t, analysis.Guardrails[0].Passed)
			} else {
				require.NotNil(t, analysis.Guardrails[0].Passed)
				assert.Equal(t, test.wantPassed, *analysis.Guardrails[0].Passed)
			}
		})
	}
}

func decisionTestObservations(variant string, values []float64) []GraderMetricObservation {
	observations := make([]GraderMetricObservation, len(values))
	for index, value := range values {
		observations[index] = GraderMetricObservation{RunID: strconv.Itoa(index + 1), Variant: variant, Value: value}
	}
	return observations
}
