package workqueue

import (
	"bytes"
	"encoding/json"
	"slices"
)

type issueTransitionTarget struct {
	WorkID    string   `json:"work_id"`
	Resource  Resource `json:"resource"`
	SummaryID string   `json:"summary_id"`
	Status    string   `json:"status"`
}

// A committed terminal transition grants no admission/Claim or creation
// authority. Only existing, same-repository AW mirrors can be synchronized.
func (state Projection) issueTransitionTargets(requestID string) ([]issueTransitionTarget, error) {
	if state.Policy == nil || state.Policy.Authorization != "aw" {
		return nil, queueError("projection_unauthorized", "terminal synchronization requires an AW-managed queue")
	}
	receipt, ok := state.Requests[requestID]
	if !ok {
		return nil, queueError("request_missing", "terminal synchronization requires an exact committed request")
	}
	targets := []issueTransitionTarget{}
	seen := map[string]struct{}{}
	for _, operation := range receipt.Operations {
		var fields struct {
			Kind         string `json:"kind"`
			WorkID       string `json:"work_id"`
			ClaimID      string `json:"claim_id"`
			CompletionID string `json:"completion_id"`
		}
		if err := json.Unmarshal(operation, &fields); err != nil {
			return nil, err
		}
		work := state.Works[fields.WorkID]
		if work == nil || work.BackingIssue != nil {
			continue
		}
		if _, exists := seen[work.WorkID]; exists {
			continue
		}
		terminal := fields.Kind == "WorkCancellation" && work.State == "cancelled" && work.CancellationCommitID == receipt.ID
		if slices.Contains([]string{"Result", "DeliveryFailure"}, fields.Kind) && work.State == "completed" &&
			work.ClaimID == fields.ClaimID && work.CompletionID == fields.CompletionID {
			left, leftErr := canonicalValue(operation)
			right, rightErr := canonicalValue(state.TerminalBarriers[work.WorkID])
			terminal = leftErr == nil && rightErr == nil && bytes.Equal(left, right)
		}
		if !terminal {
			continue
		}
		if work.IssueLink == nil || work.IssueSummary == "" {
			return nil, queueError("projection_receipt_pending", "terminal synchronization requires existing Issue and canonical summary receipts; recreation is forbidden")
		}
		if work.IssueLink.Repository != state.Repository || !slices.Contains(state.Policy.Pools[work.Pool].AllowedRepositories, state.Repository) {
			return nil, queueError("projection_unauthorized", "terminal synchronization is outside the existing queue mirror repository")
		}
		status, err := terminalIssueStatus(work)
		if err != nil {
			return nil, err
		}
		seen[work.WorkID] = struct{}{}
		targets = append(targets, issueTransitionTarget{work.WorkID, *work.IssueLink, work.IssueSummary, status})
	}
	if len(targets) > 256 {
		return nil, queueError("projection_limit", "terminal synchronization requires at most 256 existing mirrors")
	}
	return targets, nil
}

func terminalIssueStatus(work *WorkState) (string, error) {
	if work.State == "cancelled" {
		return "Cancelled", nil
	}
	if work.Barrier != "verified" {
		return "Needs attention", nil
	}
	var result struct {
		Outputs []struct {
			Type     string    `json:"type"`
			Resource *Resource `json:"resource"`
		} `json:"outputs"`
	}
	if err := json.Unmarshal(work.Result, &result); err != nil {
		return "", err
	}
	for _, output := range result.Outputs {
		if output.Type == "create_pull_request" || output.Resource != nil && output.Resource.Kind == "pull_request" {
			return "Needs review", nil
		}
	}
	return "Done", nil
}
