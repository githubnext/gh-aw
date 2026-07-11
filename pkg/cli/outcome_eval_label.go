package cli

import (
	"fmt"
	"slices"

	"github.com/github/gh-aw/pkg/logger"
)

var outcomeEvalLabelLog = logger.New("cli:outcome_eval_label")

// evalReplaceLabel checks whether the label delta applied at execution time is
// still intact in the current issue state.  It computes the set of labels added
// and removed by the replacement and verifies only that delta, ignoring any
// unrelated labels that may have been added or removed since execution.
func evalReplaceLabel(item CreatedItemReport, repoOverride string) OutcomeReport {
	repo := resolveItemRepo(item, repoOverride)
	num := resolveItemNumber(item)
	outcomeEvalLabelLog.Printf("Evaluating replace_label outcome: repo=%s, number=%d", repo, num)
	report := OutcomeReport{
		Type:         item.Type,
		ObjectURL:    item.URL,
		ObjectNumber: num,
		Repo:         repo,
	}
	if num == 0 || repo == "" || item.BeforeState == nil || item.AfterState == nil {
		report.Result = OutcomeUnknown
		report.Detail = "missing execution state"
		report.OutcomeEvaluation = OutcomeEvaluation{
			OutcomeStatus:    OutcomeStatusUnknown,
			EvidenceStrength: EvidenceNone,
			Signal:           "missing_execution_state",
		}
		return report
	}

	beforeLabels := mutableStringSlice(item.BeforeState["labels"])
	afterLabels := mutableStringSlice(item.AfterState["labels"])

	// Compute the replacement delta: labels added and labels removed.
	added := labelSetDiff(afterLabels, beforeLabels)
	removed := labelSetDiff(beforeLabels, afterLabels)

	if len(added) == 0 && len(removed) == 0 {
		report.Result = OutcomeUnknown
		report.Detail = "no label delta"
		report.OutcomeEvaluation = OutcomeEvaluation{
			OutcomeStatus:    OutcomeStatusUnknown,
			EvidenceStrength: EvidenceNone,
			Signal:           "no_state_delta",
		}
		return report
	}

	currentState, _, err := extractCurrentIssueUpdateState(repo, num)
	if err != nil {
		report.Result = OutcomeError
		report.EvalError = err.Error()
		return report
	}
	currentLabels := mutableStringSlice(currentState["labels"])

	// The replacement is retained when all added labels are still present and
	// all removed labels are still absent, regardless of any other label changes.
	addedRetained := labelSetContainsAll(currentLabels, added)
	removedStillAbsent := !labelSetContainsAny(currentLabels, removed)

	if addedRetained && removedStillAbsent {
		report.Result = OutcomeAccepted
		report.Detail = "label replacement retained"
		report.OutcomeEvaluation = OutcomeEvaluation{
			OutcomeStatus:    OutcomeStatusAccepted,
			EvidenceStrength: EvidenceMedium,
			Signal:           "state_retained",
		}
		return report
	}

	// Reverted: all added labels are gone and all removed labels are back.
	addedReverted := !labelSetContainsAny(currentLabels, added)
	removedBack := labelSetContainsAll(currentLabels, removed)
	if addedReverted && removedBack {
		report.Result = OutcomeRejected
		report.Detail = "label replacement reverted"
		report.OutcomeEvaluation = OutcomeEvaluation{
			OutcomeStatus:    OutcomeStatusRejected,
			EvidenceStrength: EvidenceStrong,
			Signal:           "state_reverted",
		}
		return report
	}

	report.Result = OutcomeRejected
	report.Detail = "label replacement replaced"
	report.OutcomeEvaluation = OutcomeEvaluation{
		OutcomeStatus:    OutcomeStatusRejected,
		EvidenceStrength: EvidenceStrong,
		Signal:           "state_replaced",
	}
	return report
}

// labelSetDiff returns the elements of a that are not in b.
// Both slices must be sorted (as produced by mutableStringSlice).
// Uses binary search for O(n log m) performance.
func labelSetDiff(a, b []string) []string {
	var out []string
	for _, v := range a {
		if _, found := slices.BinarySearch(b, v); !found {
			out = append(out, v)
		}
	}
	return out
}

// labelSetContainsAll reports whether current contains every element of want.
// Both slices must be sorted. Uses binary search for O(n log m) performance.
func labelSetContainsAll(current, want []string) bool {
	for _, v := range want {
		if _, found := slices.BinarySearch(current, v); !found {
			return false
		}
	}
	return true
}

// labelSetContainsAny reports whether current contains at least one element of want.
// Both slices must be sorted. Uses binary search for O(n log m) performance.
func labelSetContainsAny(current, want []string) bool {
	for _, v := range want {
		if _, found := slices.BinarySearch(current, v); found {
			return true
		}
	}
	return false
}

// evalAddLabels checks whether labels added by the workflow are still present.
func evalAddLabels(item CreatedItemReport, repoOverride string) OutcomeReport {
	repo := resolveItemRepo(item, repoOverride)
	num := resolveItemNumber(item)
	outcomeEvalLabelLog.Printf("Evaluating add_labels outcome: repo=%s, number=%d", repo, num)
	report := OutcomeReport{
		Type:         item.Type,
		ObjectURL:    item.URL,
		ObjectNumber: num,
		Repo:         repo,
	}
	if num == 0 || repo == "" {
		outcomeEvalLabelLog.Print("Missing issue number or repo, returning error outcome")
		report.Result = OutcomeError
		report.EvalError = "missing issue number or repo"
		return report
	}

	labels, err := ghAPIGetArray(fmt.Sprintf("issues/%d/labels", num), repo)
	if err != nil {
		outcomeEvalLabelLog.Printf("Failed to fetch labels for %s#%d: %v", repo, num, err)
		report.Result = OutcomeError
		report.EvalError = err.Error()
		return report
	}

	// We don't know exactly which labels were added (the manifest doesn't record them),
	// so we cannot reliably verify retention. If labels are still present we report
	// pending rather than accepted, because the current labels could differ entirely
	// from the ones we added. Only an empty label list is a clear rejection signal.
	if len(labels) > 0 {
		report.Result = OutcomePending
		report.Detail = "cannot evaluate label retention (added labels not recorded; extend manifest to include label names)"
	} else {
		report.Result = OutcomeRejected
		report.Detail = "all labels removed"
	}

	outcomeEvalLabelLog.Printf("Label evaluation result: result=%s, label_count=%d", report.Result, len(labels))
	return report
}
