package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var outcomeEvalAgentLog = logger.New("cli:outcome_eval_agent")

// evalAssignToAgent checks whether an agent assignment led to a PR that was merged.
func evalAssignToAgent(item CreatedItemReport, repoOverride string) OutcomeReport {
	repo := resolveItemRepo(item, repoOverride)
	num := resolveItemNumber(item)
	outcomeEvalAgentLog.Printf("Evaluating assign_to_agent: repo=%s, num=%d, url=%s", repo, num, item.URL)
	report := OutcomeReport{
		Type:         item.Type,
		ObjectURL:    item.URL,
		ObjectNumber: num,
		Repo:         repo,
	}
	if num == 0 || repo == "" {
		outcomeEvalAgentLog.Printf("Missing issue number or repo: num=%d, repo=%s", num, repo)
		report.Result = OutcomeError
		report.EvalError = "missing issue number or repo"
		return report
	}

	// Check issue state first
	issueData, err := ghAPIGet(fmt.Sprintf("issues/%d", num), repo)
	if err != nil {
		report.Result = OutcomeError
		report.EvalError = err.Error()
		return report
	}

	state, _ := issueData["state"].(string)
	stateReason, _ := issueData["state_reason"].(string)

	// Search for linked PRs from copilot-swe-agent via timeline events
	events, err := ghAPIGetArray(fmt.Sprintf("issues/%d/timeline", num), repo)
	if err == nil {
		agentPR := evalAssignToAgentFindLinkedPR(events)
		if prReport, ok := evalAssignToAgentEvaluateLinkedPR(agentPR, repo, item, report); ok {
			return prReport
		}
	}

	return evalAssignToAgentFallbackIssue(issueData, item, report, state, stateReason)
}

func evalAssignToAgentFindLinkedPR(events []map[string]any) map[string]any {
	for _, event := range events {
		eventType, _ := event["event"].(string)
		if eventType != "cross-referenced" {
			continue
		}
		source, _ := event["source"].(map[string]any)
		if source == nil {
			continue
		}
		issue, _ := source["issue"].(map[string]any)
		if issue == nil {
			continue
		}
		// Check if it's a PR (has pull_request field)
		if _, hasPR := issue["pull_request"]; !hasPR {
			continue
		}
		user, _ := issue["user"].(map[string]any)
		login, _ := user["login"].(string)
		if strings.Contains(login, "copilot") || strings.Contains(login, "github-actions") {
			return issue
		}
	}
	return nil
}

func evalAssignToAgentEvaluateLinkedPR(agentPR map[string]any, repo string, item CreatedItemReport, report OutcomeReport) (OutcomeReport, bool) {
	if agentPR == nil {
		return report, false
	}
	prNumber := 0
	if n, ok := agentPR["number"].(float64); ok {
		prNumber = int(n)
	}
	outcomeEvalAgentLog.Printf("Found agent-linked PR for issue #%d: pr=%d", report.ObjectNumber, prNumber)
	if prNumber == 0 {
		return report, false
	}
	prData, perr := ghAPIGet(fmt.Sprintf("pulls/%d", prNumber), repo)
	if perr != nil {
		return report, false
	}
	return evalAssignToAgentReportForPR(prData, prNumber, item, report), true
}

func evalAssignToAgentReportForPR(prData map[string]any, prNumber int, item CreatedItemReport, report OutcomeReport) OutcomeReport {
	merged, _ := prData["merged"].(bool)
	prState, _ := prData["state"].(string)
	mergedAt, _ := prData["merged_at"].(string)
	closedAt, _ := prData["closed_at"].(string)
	switch {
	case merged:
		report.Result = OutcomeAccepted
		report.Detail = fmt.Sprintf("agent PR #%d merged", prNumber)
		if mergedAt != "" && item.Timestamp != "" {
			report.TimeToOutcomeHours = timeBetween(item.Timestamp, mergedAt)
		}
	case prState == "closed":
		report.Result = OutcomeRejected
		report.Detail = fmt.Sprintf("agent PR #%d closed without merge", prNumber)
		if closedAt != "" && item.Timestamp != "" {
			report.TimeToOutcomeHours = timeBetween(item.Timestamp, closedAt)
		}
	default:
		report.Result = OutcomePending
		report.Detail = fmt.Sprintf("agent PR #%d open", prNumber)
	}
	return report
}

func evalAssignToAgentFallbackIssue(issueData map[string]any, item CreatedItemReport, report OutcomeReport, state string, stateReason string) OutcomeReport {
	switch {
	case state == "closed" && stateReason == "completed":
		report.Result = OutcomeAccepted
		report.Detail = "issue resolved (no agent PR found)"
		closedAt, _ := issueData["closed_at"].(string)
		if closedAt != "" && item.Timestamp != "" {
			report.TimeToOutcomeHours = timeBetween(item.Timestamp, closedAt)
		}
	case state == "closed":
		report.Result = OutcomeRejected
		report.Detail = "issue closed without resolution, no agent PR"
	default:
		report.Result = OutcomeIgnored
		report.Detail = "no agent PR created"
	}

	return report
}
