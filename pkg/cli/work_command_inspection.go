package cli

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/github/gh-aw/pkg/workqueue"
)

type workQueueWorkMetadata struct {
	WorkID             string                 `json:"work_id"`
	GraphID            string                 `json:"graph_id"`
	NodeKey            string                 `json:"node_key"`
	Pool               string                 `json:"pool"`
	WorkerProfile      string                 `json:"worker_profile"`
	State              string                 `json:"state"`
	Priority           int                    `json:"priority"`
	AdmittedPriority   int                    `json:"admitted_priority"`
	FairnessKey        string                 `json:"fairness_key"`
	Position           workqueue.Position     `json:"position"`
	EnqueuedAt         int64                  `json:"enqueued_at"`
	AdmittedAt         *int64                 `json:"admitted_at"`
	Attempts           int                    `json:"attempts"`
	MaxAttempts        *int                   `json:"max_attempts"`
	RetryNotBefore     int64                  `json:"retry_not_before"`
	ClaimID            string                 `json:"claim_id,omitempty"`
	Delivery           string                 `json:"delivery"`
	CompletionID       string                 `json:"completion_id,omitempty"`
	ResultCommitID     string                 `json:"result_commit_id,omitempty"`
	Disposition        string                 `json:"disposition,omitempty"`
	CancellationReason string                 `json:"cancellation_reason,omitempty"`
	DependsOn          []workqueue.Dependency `json:"depends_on"`
	Issue              *workqueue.Resource    `json:"issue,omitempty"`
}

type workQueueClaimMetadata struct {
	ClaimID               string `json:"claim_id"`
	WorkID                string `json:"work_id"`
	DispatchID            string `json:"dispatch_id"`
	Handle                string `json:"claim_handle"`
	State                 string `json:"state"`
	Current               bool   `json:"current"`
	RequestID             string `json:"request_id"`
	CommitID              string `json:"commit_id"`
	TerminalCommitID      string `json:"terminal_commit_id,omitempty"`
	GrantedAt             *int64 `json:"granted_at"`
	TerminalAt            *int64 `json:"terminal_at"`
	CancellationReason    string `json:"cancellation_reason,omitempty"`
	RetryNotBefore        int64  `json:"retry_not_before"`
	KeepOpen              bool   `json:"keep_open"`
	Delivery              string `json:"delivery"`
	DeliveryFailureReason string `json:"delivery_failure_reason,omitempty"`
	Disposition           string `json:"disposition,omitempty"`
}

type workQueueAssignmentMember struct {
	ClaimID string `json:"claim_id"`
	WorkID  string `json:"work_id"`
	Handle  string `json:"claim_handle"`
}

type workQueueDispatchMetadata struct {
	DispatchID          string                      `json:"dispatch_id"`
	Pool                string                      `json:"pool"`
	WorkerProfile       string                      `json:"worker_profile"`
	Workflow            string                      `json:"workflow"`
	Ref                 string                      `json:"ref"`
	State               string                      `json:"state"`
	Reason              string                      `json:"reason,omitempty"`
	ReservationRetained bool                        `json:"reservation_retained"`
	Run                 *workqueue.RunBinding       `json:"run"`
	Claims              []workQueueAssignmentMember `json:"claims"`
}

type workQueueInspection struct {
	workQueueSnapshot
	Selection  workQueueRow               `json:"selection"`
	Work       workQueueWorkMetadata      `json:"work"`
	Scheduling workqueue.WorkExplanation  `json:"scheduling"`
	Claim      *workQueueClaimMetadata    `json:"claim"`
	Dispatch   *workQueueDispatchMetadata `json:"dispatch"`
	Details    string                     `json:"details"`
}

func workQueueInspectAt(state workqueue.Projection, branch string, row workQueueRow, at int64) (workQueueInspection, error) {
	result := workQueueInspection{workQueueSnapshot: workQueueSnapshotAt(state, branch, at), Selection: row}
	if row.Kind != "graph" && state.Works[row.WorkID] == nil {
		return result, errors.New("work_missing: selected Work is absent; refresh state")
	}
	var err error
	result.Details, err = workQueueDetails(state, row, at)
	if err != nil {
		return result, err
	}
	if row.Kind == "graph" {
		return result, nil
	}
	work := state.Works[row.WorkID]
	result.Scheduling, err = workQueueExplainWork(state, work, at)
	if err != nil {
		return result, err
	}
	result.Work = workQueueWorkMetadataAt(state, work)
	claimID := work.ClaimID
	if row.Kind == "claim" {
		claimID = row.ID
	}
	if claimID == "" {
		return result, nil
	}
	claim := state.Claims[claimID]
	metadata, err := workQueueClaimMetadataAt(state, claim)
	if err != nil {
		return result, err
	}
	result.Claim = &metadata
	if dispatch := state.Dispatches[claim.DispatchID]; dispatch != nil {
		result.Dispatch = &workQueueDispatchMetadata{
			DispatchID: dispatch.DispatchID, Pool: dispatch.Pool, WorkerProfile: dispatch.WorkerProfile,
			Workflow: dispatch.Profile.Workflow, Ref: dispatch.Profile.Ref, State: dispatch.State,
			Reason: dispatch.Reason, ReservationRetained: !dispatch.Released, Run: dispatch.Run,
			Claims: []workQueueAssignmentMember{},
		}
		for _, member := range dispatch.Claims {
			result.Dispatch.Claims = append(result.Dispatch.Claims, workQueueAssignmentMember{
				ClaimID: member.ClaimID, WorkID: member.WorkID, Handle: member.Handle,
			})
		}
	}
	return result, nil
}

func workQueueExplainWork(state workqueue.Projection, work *workqueue.WorkState, at int64) (workqueue.WorkExplanation, error) {
	if state.Policy != nil {
		if _, ok := state.Policy.Pools[work.Pool]; ok {
			return workqueue.ExplainWork(state, work.WorkID, at)
		}
	}
	reason := "pool_not_installed"
	if work.State != "available" {
		reason = "ownership_" + work.State
	}
	return workqueue.WorkExplanation{
		WorkID: work.WorkID, State: work.State, Barrier: work.Barrier,
		Reason: reason, Path: []string{work.NodeKey}, Observations: []string{},
	}, nil
}

func workQueueClaimMetadataAt(state workqueue.Projection, claim *workqueue.ClaimState) (workQueueClaimMetadata, error) {
	work := state.Works[claim.WorkID]
	result := workQueueClaimMetadata{
		ClaimID: claim.ClaimID, WorkID: claim.WorkID, DispatchID: claim.DispatchID, Handle: claim.Handle,
		State: claim.State, Current: work.ClaimID == claim.ClaimID && work.State != "cancelled",
		RequestID: claim.RequestID, CommitID: claim.CommitID, TerminalCommitID: claim.TerminalCommitID,
		GrantedAt: workQueueCommitAt(state, claim.CommitID), TerminalAt: workQueueCommitAt(state, claim.TerminalCommitID),
		CancellationReason: claim.CancellationReason, RetryNotBefore: claim.RetryNotBefore,
		KeepOpen: claim.KeepOpen, Delivery: workQueueClaimDelivery(state, claim),
	}
	delivery, err := workQueueClaimTerminal(state, claim)
	if err != nil {
		return result, err
	}
	result.DeliveryFailureReason, result.Disposition = delivery.Reason, delivery.Disposition
	return result, nil
}

func workQueueWorkMetadataAt(state workqueue.Projection, work *workqueue.WorkState) workQueueWorkMetadata {
	var maxAttempts *int
	if state.Policy != nil {
		if pool, ok := state.Policy.Pools[work.Pool]; ok {
			maxAttempts = &pool.Retry.MaxAttempts
		}
	}
	return workQueueWorkMetadata{
		WorkID: work.WorkID, GraphID: work.GraphID, NodeKey: work.NodeKey, Pool: work.Pool,
		WorkerProfile: work.WorkerProfile, State: work.State, Priority: work.SchedulingPriority(),
		AdmittedPriority: work.Priority, FairnessKey: work.FairnessKey, Position: work.Position,
		EnqueuedAt: work.Enqueued, AdmittedAt: workQueueAdmittedAt(state, work), Attempts: work.Attempts,
		MaxAttempts: maxAttempts, RetryNotBefore: work.RetryNotBefore,
		ClaimID: work.ClaimID, Delivery: work.Barrier, CompletionID: work.CompletionID,
		ResultCommitID: work.ResultCommitID, Disposition: work.Disposition, CancellationReason: work.CancellationReason,
		DependsOn: append([]workqueue.Dependency{}, work.DependsOn...), Issue: work.IssueLink,
	}
}

type workQueueTerminalMetadata struct {
	Kind        string `json:"kind"`
	ClaimID     string `json:"claim_id"`
	Reason      string `json:"reason"`
	Disposition string `json:"disposition"`
}

func workQueueClaimTerminal(state workqueue.Projection, claim *workqueue.ClaimState) (workQueueTerminalMetadata, error) {
	terminal := state.TerminalBarriers[claim.ClaimID]
	if len(terminal) == 0 {
		terminal = state.TerminalBarriers[claim.WorkID]
	}
	var delivery workQueueTerminalMetadata
	if len(terminal) == 0 {
		return delivery, nil
	}
	if err := json.Unmarshal(terminal, &delivery); err != nil {
		return delivery, fmt.Errorf("read Claim delivery metadata: %w", err)
	}
	if delivery.ClaimID != claim.ClaimID {
		return workQueueTerminalMetadata{}, nil
	}
	return delivery, nil
}

func workQueueCommitAt(state workqueue.Projection, id string) *int64 {
	if id == "" {
		return nil
	}
	for _, commit := range state.Requests {
		if commit.ID == id {
			return &commit.At
		}
	}
	return nil
}
