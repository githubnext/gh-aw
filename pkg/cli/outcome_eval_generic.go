package cli

import "fmt"

// evalCloseSticky checks whether a closed issue or PR stayed closed.
func evalCloseSticky(item CreatedItemReport, repoOverride string) OutcomeReport {
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
		report.EvalError = "missing number or repo"
		return report
	}

	data, err := ghAPIGet(fmt.Sprintf("issues/%d", num), repo)
	if err != nil {
		report.Result = OutcomeError
		report.EvalError = err.Error()
		return report
	}

	state, _ := data["state"].(string)
	if state == "closed" {
		report.Result = OutcomeAccepted
		report.Detail = "still closed"
	} else {
		report.Result = OutcomeRejected
		report.Detail = "reopened"
	}
	return report
}

// evalCloseDiscussion checks whether a closed discussion stayed closed.
// Uses REST API approximation since discussions don't have a direct REST endpoint.
func evalCloseDiscussion(item CreatedItemReport, repoOverride string) OutcomeReport {
	// Discussions require GraphQL; for now return pending with a note
	return OutcomeReport{
		Type:      item.Type,
		ObjectURL: item.URL,
		Repo:      resolveItemRepo(item, repoOverride),
		Result:    OutcomePending,
		Detail:    "discussion outcome check requires GraphQL (not yet implemented)",
	}
}

// evalCreateDiscussion checks whether a discussion received replies.
func evalCreateDiscussion(item CreatedItemReport, repoOverride string) OutcomeReport {
	return OutcomeReport{
		Type:      item.Type,
		ObjectURL: item.URL,
		Repo:      resolveItemRepo(item, repoOverride),
		Result:    OutcomePending,
		Detail:    "discussion outcome check requires GraphQL (not yet implemented)",
	}
}

// evalHideComment checks whether a hidden comment is still hidden.
func evalHideComment(item CreatedItemReport, repoOverride string) OutcomeReport {
	return OutcomeReport{
		Type:      item.Type,
		ObjectURL: item.URL,
		Repo:      resolveItemRepo(item, repoOverride),
		Result:    OutcomePending,
		Detail:    "hidden comment check requires GraphQL (not yet implemented)",
	}
}

// evalAssignMilestone checks whether a milestone assignment stuck.
func evalAssignMilestone(item CreatedItemReport, repoOverride string) OutcomeReport {
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
		report.EvalError = "missing number or repo"
		return report
	}

	data, err := ghAPIGet(fmt.Sprintf("issues/%d", num), repo)
	if err != nil {
		report.Result = OutcomeError
		report.EvalError = err.Error()
		return report
	}

	if data["milestone"] != nil {
		report.Result = OutcomeAccepted
		report.Detail = "milestone still assigned"
	} else {
		report.Result = OutcomeRejected
		report.Detail = "milestone removed"
	}
	return report
}

// evalReviewComment checks whether a PR review comment thread was resolved or engaged.
func evalReviewComment(item CreatedItemReport, repoOverride string) OutcomeReport {
	return OutcomeReport{
		Type:      item.Type,
		ObjectURL: item.URL,
		Repo:      resolveItemRepo(item, repoOverride),
		Result:    OutcomePending,
		Detail:    "review thread check requires GraphQL (not yet implemented)",
	}
}

// evalResolveThread checks whether a resolved review thread stayed resolved.
func evalResolveThread(item CreatedItemReport, repoOverride string) OutcomeReport {
	return OutcomeReport{
		Type:      item.Type,
		ObjectURL: item.URL,
		Repo:      resolveItemRepo(item, repoOverride),
		Result:    OutcomePending,
		Detail:    "resolve thread check requires GraphQL (not yet implemented)",
	}
}

// evalMarkReady checks whether a PR marked as ready received reviews.
func evalMarkReady(item CreatedItemReport, repoOverride string) OutcomeReport {
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
		report.EvalError = "missing number or repo"
		return report
	}

	reviews, err := ghAPIGetArray(fmt.Sprintf("pulls/%d/reviews", num), repo)
	if err != nil {
		report.Result = OutcomeError
		report.EvalError = err.Error()
		return report
	}

	if len(reviews) > 0 {
		report.Result = OutcomeAccepted
		report.Detail = fmt.Sprintf("%d reviews submitted", len(reviews))
	} else {
		data, derr := ghAPIGet(fmt.Sprintf("pulls/%d", num), repo)
		if derr == nil {
			state, _ := data["state"].(string)
			if state == "open" {
				report.Result = OutcomePending
				report.Detail = "awaiting review"
			} else {
				report.Result = OutcomeIgnored
				report.Detail = "closed/merged without review"
			}
		} else {
			report.Result = OutcomePending
			report.Detail = "no reviews yet"
		}
	}
	return report
}

// evalPushToPRBranch checks whether the PR the code was pushed to got merged.
func evalPushToPRBranch(item CreatedItemReport, repoOverride string) OutcomeReport {
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
		report.EvalError = "missing PR number or repo"
		return report
	}

	data, err := ghAPIGet(fmt.Sprintf("pulls/%d", num), repo)
	if err != nil {
		report.Result = OutcomeError
		report.EvalError = err.Error()
		return report
	}

	merged, _ := data["merged"].(bool)
	state, _ := data["state"].(string)

	switch {
	case merged:
		report.Result = OutcomeAccepted
		report.Detail = "PR merged"
	case state == "closed":
		report.Result = OutcomeRejected
		report.Detail = "PR closed without merge"
	default:
		report.Result = OutcomePending
		report.Detail = "PR still open"
	}
	return report
}

// evalGenericSticky is a fallback evaluator for types that modify an existing object.
// It simply checks whether the target issue/PR still exists and is accessible.
func evalGenericSticky(item CreatedItemReport, repoOverride string) OutcomeReport {
	repo := resolveItemRepo(item, repoOverride)
	num := resolveItemNumber(item)
	report := OutcomeReport{
		Type:         item.Type,
		ObjectURL:    item.URL,
		ObjectNumber: num,
		Repo:         repo,
	}

	if num == 0 || repo == "" {
		// No number to check — just report what we know
		report.Result = OutcomePending
		report.Detail = "no object reference to check"
		return report
	}

	_, err := ghAPIGet(fmt.Sprintf("issues/%d", num), repo)
	if err != nil {
		report.Result = OutcomeError
		report.EvalError = err.Error()
		return report
	}

	report.Result = OutcomeAccepted
	report.Detail = "object still exists"
	return report
}
