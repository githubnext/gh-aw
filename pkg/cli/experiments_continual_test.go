//go:build !integration

package cli

import (
	"fmt"
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func continualTestConfig(minimum int) *workflow.ExperimentConfig {
	return &workflow.ExperimentConfig{
		Variants: []string{"control", "candidate"},
		Continual: &workflow.ContinualExperimentConfig{
			Seed:      "stable-seed",
			Objective: workflow.ContinualObjectiveConfig{Metric: "eval:quality", MinimumImprovement: 0.02},
			Decision: workflow.ContinualDecisionConfig{
				MinimumObservations: minimum,
				Confidence:          0.95,
				RegressionTolerance: 0.03,
			},
			Segments: workflow.ContinualSegmentsConfig{Critical: []string{"task_type"}},
			Ramp:     []int{10, 25, 50},
		},
	}
}

func continualLedger(controlSuccess, controlTotal, candidateSuccess, candidateTotal int) []ExperimentOutcomeRecord {
	var ledger []ExperimentOutcomeRecord
	appendVariant := func(variant string, success, total int) {
		for i := range total {
			value := 0.0
			if i < success {
				value = 1
			}
			ledger = append(ledger, ExperimentOutcomeRecord{
				SchemaVersion: experimentOutcomeSchemaVersion,
				RunID:         fmt.Sprintf("%s-%d", variant, i),
				Experiment:    "optimization",
				Variant:       variant,
				Features:      map[string]string{"task_type": "general"},
				Outcomes:      map[string]float64{"eval:quality": value},
			})
		}
	}
	appendVariant("control", controlSuccess, controlTotal)
	appendVariant("candidate", candidateSuccess, candidateTotal)
	return ledger
}

func TestEvaluateContinualExperimentPromotesClearWinner(t *testing.T) {
	result := evaluateContinualExperiment("optimization", continualTestConfig(20), continualLedger(12, 30, 28, 30))
	require.NotNil(t, result)
	assert.Equal(t, "PROMOTE", result.Decision)
	assert.Equal(t, 25, result.RecommendedCandidatePercent)
}

func TestEvaluateContinualExperimentRejectsHarm(t *testing.T) {
	result := evaluateContinualExperiment("optimization", continualTestConfig(20), continualLedger(28, 30, 10, 30))
	require.NotNil(t, result)
	assert.Equal(t, "REJECT", result.Decision)
}

func TestEvaluateContinualExperimentContinuesWithInsufficientEvidence(t *testing.T) {
	result := evaluateContinualExperiment("optimization", continualTestConfig(20), continualLedger(4, 5, 5, 5))
	require.NotNil(t, result)
	assert.Equal(t, "CONTINUE", result.Decision)
}

func TestEvaluateContinualExperimentRejectsCriticalSegmentRegression(t *testing.T) {
	cfg := continualTestConfig(10)
	ledger := continualLedger(10, 20, 18, 20)
	for range 10 {
		ledger = append(ledger,
			ExperimentOutcomeRecord{Variant: "control", Features: map[string]string{"task_type": "security"}, Outcomes: map[string]float64{"eval:quality": 1}},
			ExperimentOutcomeRecord{Variant: "candidate", Features: map[string]string{"task_type": "security"}, Outcomes: map[string]float64{"eval:quality": 0}},
		)
	}
	result := evaluateContinualExperiment("optimization", cfg, ledger)
	require.NotNil(t, result)
	assert.Equal(t, "REJECT", result.Decision)
}

func TestEvaluateContinualExperimentCanPromoteEquivalentCheaperCandidate(t *testing.T) {
	cfg := continualTestConfig(20)
	cfg.Continual.Decision.AllowCostPromotion = true
	ledger := continualLedger(16, 20, 16, 20)
	for i := range ledger {
		if ledger[i].Variant == "control" {
			ledger[i].Outcomes["aic"] = 10
		} else {
			ledger[i].Outcomes["aic"] = 7
		}
	}
	result := evaluateContinualExperiment("optimization", cfg, ledger)
	require.NotNil(t, result)
	assert.Equal(t, "PROMOTE", result.Decision)
}

func TestBuildExperimentOutcomeLedgerJoinsByRunIDAndSkipsUnknown(t *testing.T) {
	runs := []ExperimentRunRecord{
		{RunID: "1", HarnessVersion: "sha256:a", Assignments: map[string]string{"optimization": "control"}, Features: map[string]string{"event": "issues"}},
		{RunID: "2", HarnessVersion: "sha256:b", Assignments: map[string]string{"optimization": "candidate"}},
	}
	evals := []evalResultRecord{
		{ID: "quality", Answer: "YES", RunID: "1"},
		{ID: "quality", Answer: "UNKNOWN", RunID: "2"},
	}
	ledger := buildExperimentOutcomeLedger("optimization", "eval:quality", runs, evals)
	require.Len(t, ledger, 1)
	assert.Equal(t, "control", ledger[0].Variant)
	assert.Equal(t, "sha256:a", ledger[0].HarnessVersion)
	assert.InDelta(t, 1.0, ledger[0].Outcomes["eval:quality"], 0.0001)
}
