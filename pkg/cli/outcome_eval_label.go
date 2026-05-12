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

	// We don't know exactly which labels were added (the manifest doesn't record them).
	// We can only confirm that the issue still has labels. If the issue has zero labels
	// and we know we added some, that is a rejection signal.
	if len(labels) > 0 {
		report.Result = OutcomeAccepted
		report.Detail = fmt.Sprintf("%d labels present", len(labels))
	} else {
		report.Result = OutcomeRejected
		report.Detail = "all labels removed"
	}

	return report
}
