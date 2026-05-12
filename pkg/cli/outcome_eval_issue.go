package cli

import "fmt"

// evalCreateIssue checks whether an issue was resolved, dismissed, or is still open.
// Bot-initiated closes (e.g. close-older-issues) are classified as lifecycle, not rejection.
func evalCreateIssue(item CreatedItemReport, repoOverride string) OutcomeReport {
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

	data, err := ghAPIGet(fmt.Sprintf("issues/%d", num), repo)
	if err != nil {
		report.Result = OutcomeError
		report.EvalError = err.Error()
		return report
	}

	state, _ := data["state"].(string)
	stateReason, _ := data["state_reason"].(string)
	closedAt, _ := data["closed_at"].(string)

	// Count human comments
	comments, _ := data["comments"].(float64)
	commentList, cerr := ghAPIGetArray(fmt.Sprintf("issues/%d/comments", num), repo)
	if cerr == nil {
		for _, c := range commentList {
			user, _ := c["user"].(map[string]any)
			login, _ := user["login"].(string)
			if !isBotUser(login) {
				report.HumanComments++
			}
		}
	}

	switch {
	case state == "closed" && stateReason == "completed":
		report.Result = OutcomeAccepted
		report.Detail = "completed"
		if closedAt != "" && item.Timestamp != "" {
			report.TimeToOutcomeHours = timeBetween(item.Timestamp, closedAt)
		}

	case state == "closed" && stateReason == "not_planned":
		// Check if closed by a bot (lifecycle) or human (rejection)
		closedByBot := isClosedByBot(num, repo)
		if closedByBot {
			report.Result = OutcomeLifecycle
			report.Detail = "closed by bot (lifecycle)"
		} else {
			report.Result = OutcomeRejected
			report.Detail = "closed as not planned"
		}
		if closedAt != "" && item.Timestamp != "" {
			report.TimeToOutcomeHours = timeBetween(item.Timestamp, closedAt)
		}

	case state == "closed":
		report.Result = OutcomeAccepted
		report.Detail = "closed"
		if closedAt != "" && item.Timestamp != "" {
			report.TimeToOutcomeHours = timeBetween(item.Timestamp, closedAt)
		}

	case state == "open" && report.HumanComments > 0:
		report.Result = OutcomePending
		report.Detail = fmt.Sprintf("open, %d human comments", report.HumanComments)

	case state == "open" && int(comments) > 0:
		report.Result = OutcomePending
		report.Detail = "open with comments"

	default:
		report.Result = OutcomeIgnored
		report.Detail = "open, no engagement"
	}

	return report
}

// isClosedByBot checks the issue timeline to determine if the close event was performed by a bot.
func isClosedByBot(issueNumber int, repo string) bool {
	events, err := ghAPIGetArray(fmt.Sprintf("issues/%d/events", issueNumber), repo)
	if err != nil {
		return false
	}
	// Walk backward to find the most recent close event
	for i := len(events) - 1; i >= 0; i-- {
		event, _ := events[i]["event"].(string)
		if event == "closed" {
			actor, _ := events[i]["actor"].(map[string]any)
			login, _ := actor["login"].(string)
			return isBotUser(login)
		}
	}
	return false
}
