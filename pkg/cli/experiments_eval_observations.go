package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/workflow"
)

const (
	exclusionEvalRecordUnavailable = "eval record unavailable"
	exclusionEvalMissing           = "eval missing"
)

// resolveEvalMetricReferences normalizes eval:<id> and evals.<id>.value metric references
// declared on experiments, validating that each referenced eval question is declared.
func resolveEvalMetricReferences(configs map[string]*workflow.ExperimentConfig, evals *workflow.EvalsConfig) (map[string]string, error) {
	refs := make(map[string]string)
	for experimentName, cfg := range configs {
		if cfg == nil {
			continue
		}
		evalID, isEval := workflow.ParseExperimentMetricEvalReference(cfg.Metric)
		if !isEval {
			continue
		}
		if evalID == "" {
			return nil, fmt.Errorf("experiments.%s.metric: expected eval reference format eval:<question_id> or evals.<question_id>", experimentName)
		}
		if evals == nil || !evalQuestionDeclared(evals, evalID) {
			if evals == nil || len(evals.Questions) == 0 {
				return nil, fmt.Errorf("experiments.%s.metric: references eval %q but no evals are declared", experimentName, evalID)
			}
			return nil, fmt.Errorf("experiments.%s.metric: references unknown eval %q", experimentName, evalID)
		}
		refs[experimentName] = evalID
	}
	return refs, nil
}

func evalQuestionDeclared(evals *workflow.EvalsConfig, evalID string) bool {
	for _, q := range evals.Questions {
		if q.ID == evalID {
			return true
		}
	}
	return false
}

// buildEvalMetricObservationSets attributes per-run eval answers (YES/NO) recorded in
// evals.jsonl to variants using the persisted assignment history, mirroring the
// grader-backed observation pipeline so eval-backed metrics support the same
// statistical comparisons and readiness diagnostics.
func buildEvalMetricObservationSets(
	experiments []ExperimentVariantStats,
	runs []ExperimentRunRecord,
	refs map[string]string,
	evalRecords []evalResultRecord,
) map[string]*graderMetricObservationSet {
	assignedCounts, sets, recordedCounts, seen := initializeGraderObservationSets(experiments, refs)

	byRunAndEval := make(map[string]map[string]evalResultRecord, len(evalRecords))
	for _, record := range evalRecords {
		if record.RunID == "" || record.ID == "" {
			continue
		}
		m, ok := byRunAndEval[record.RunID]
		if !ok {
			m = make(map[string]evalResultRecord)
			byRunAndEval[record.RunID] = m
		}
		m[record.ID] = record
	}

	for _, run := range runs {
		for experimentName, evalID := range refs {
			appendEvalObservation(run, experimentName, evalID, sets, recordedCounts, seen, byRunAndEval)
		}
	}
	addMissingAssignmentHistory(assignedCounts, sets, recordedCounts)
	return sets
}

func appendEvalObservation(
	run ExperimentRunRecord,
	experimentName string,
	evalID string,
	sets map[string]*graderMetricObservationSet,
	recordedCounts map[string]map[string]int,
	seen map[string]map[string]struct{},
	byRunAndEval map[string]map[string]evalResultRecord,
) {
	variant, assigned := run.Assignments[experimentName]
	if !assigned {
		return
	}
	recordedCounts[experimentName][variant]++
	set := sets[experimentName]
	if _, duplicate := seen[experimentName][run.RunID]; duplicate {
		addObservationExclusion(set, variant, exclusionDuplicateAssignment, run.RunID, 1)
		return
	}
	seen[experimentName][run.RunID] = struct{}{}

	records, ok := byRunAndEval[run.RunID]
	if !ok {
		addObservationExclusion(set, variant, exclusionEvalRecordUnavailable, run.RunID, 1)
		return
	}
	record, ok := records[evalID]
	if !ok {
		addObservationExclusion(set, variant, exclusionEvalMissing, run.RunID, 1)
		return
	}
	value, ok := parseEvalAnswerValue(record.Answer)
	if !ok {
		addObservationExclusion(set, variant, exclusionInvalidValue, run.RunID, 1)
		return
	}
	set.ByVariant[variant] = append(set.ByVariant[variant], GraderMetricObservation{
		RunID:        run.RunID,
		Variant:      variant,
		GraderID:     evalID,
		GraderStatus: strings.ToUpper(strings.TrimSpace(record.Answer)),
		Value:        value,
		Binary:       true,
	})
}

// parseEvalAnswerValue converts a YES/NO eval answer into a binary 1/0 value.
// Returns ok=false for UNKNOWN or any other unrecognized answer.
func parseEvalAnswerValue(answer string) (float64, bool) {
	switch strings.ToUpper(strings.TrimSpace(answer)) {
	case "YES":
		return 1, true
	case "NO":
		return 0, true
	default:
		return 0, false
	}
}
