package cli

import (
	"context"
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
)

var outcomeEvalIssueLog = logger.New("cli:outcome_eval_issue")

// evalCreateIssue checks whether an issue was resolved, dismissed, or is still open.
// Bot-initiated closes (e.g. close-older-issues) are classified as lifecycle, not rejection.
func evalCreateIssue(ctx context.Context, item CreatedItemReport, repoOverride string) OutcomeReport {
	repo := resolveItemRepo(item, repoOverride)
	num := resolveItemNumber(item)
	outcomeEvalIssueLog.Printf("Evaluating create_issue: repo=%s, num=%d, url=%s", repo, num, item.URL)
	report := OutcomeReport{
		Type:         item.Type,
		ObjectURL:    item.URL,
		ObjectNumber: num,
		Repo:         repo,
	}
	if num == 0 || repo == "" {
		outcomeEvalIssueLog.Printf("Missing issue number or repo: num=%d, repo=%s", num, repo)
		report.Result = OutcomeError
		report.EvalError = "missing issue number or repo"
		return report
	}

	data, err := ghAPIGet(ctx, fmt.Sprintf("issues/%d", num), repo)
	if err != nil {
		report.Result = OutcomeError
		report.EvalError = err.Error()
		return report
	}

	state, _ := data["state"].(string)
	stateReason, _ := data["state_reason"].(string)
	closedAt, _ := data["closed_at"].(string)

	comments, _ := data["comments"].(float64)
	commentList, cerr := ghAPIGetArray(ctx, fmt.Sprintf("issues/%d/comments", num), repo)
	if cerr == nil {
		report.HumanComments = countHumanComments(commentList)
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
		closedByBot := isClosedByBot(ctx, num, repo)
		outcomeEvalIssueLog.Printf("Issue #%d closed as not_planned, closed_by_bot=%v", num, closedByBot)
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
func isClosedByBot(ctx context.Context, issueNumber int, repo string) bool {
	closedByBot, err := isLatestCloseByBot(ctx, issueNumber, repo, ghAPIGetArray)
	return err == nil && closedByBot
}
