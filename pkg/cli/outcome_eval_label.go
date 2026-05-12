package cli

import "fmt"

// evalAddLabels checks whether labels added by the workflow are still present.
func evalAddLabels(item CreatedItemReport, repoOverride string) OutcomeReport {
	repo := resolveItemRepo(item, repoOverride)
	num := resolveItemNumber(item)
	report := OutcomeReport{
		Type:         item.Type,
		ObjectURL:    item.URL,
		ObjectNumber: num,
		Repo:         repo,
	}
	if num == 0 || repo == "" {
		report.Result = OutcomeError
		report.EvalError = "missing issue number or repo"
		return report
	}

	labels, err := ghAPIGetArray(fmt.Sprintf("issues/%d/labels", num), repo)
	if err != nil {
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

	return report
}
