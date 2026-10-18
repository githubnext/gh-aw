package workqueue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// HardClearResult distinguishes a new queue authority from history-preserving cancellation.
type HardClearResult struct {
	Status        string `json:"status"`
	RequestID     string `json:"request_id"`
	PriorGitSHA   string `json:"prior_git_sha"`
	PriorTip      string `json:"prior_tip,omitempty"`
	Tip           string `json:"tip"`
	Changed       bool   `json:"changed"`
	WorkRemoved   int    `json:"work_removed,omitempty"`
	HistoryInGit  bool   `json:"history_in_git"`
	IdentityReset bool   `json:"identity_reset"`
}

// HardClear replaces only a settled, paused queue, retaining its prior ledger in
// the Git parent. It never retries a conflicting reset against a newer snapshot.
func (b Branch) HardClear(ctx context.Context, requestID, reason string, dryRun bool) (HardClearResult, error) {
	return b.hardClear(ctx, requestID, reason, dryRun, "")
}

// HardClearFromSnapshot resets only the exact Git snapshot the operator previewed.
// An already accepted request is recovered without resetting any newer Work.
func (b Branch) HardClearFromSnapshot(ctx context.Context, requestID, reason, priorGitSHA string) (HardClearResult, error) {
	if !revisionPattern.MatchString(priorGitSHA) {
		return HardClearResult{}, queueError("hard_clear_snapshot_invalid", "hard clear requires the previewed Git revision")
	}
	return b.hardClear(ctx, requestID, reason, false, priorGitSHA)
}

func (b Branch) hardClear(ctx context.Context, requestID, reason string, dryRun bool, expectedGitSHA string) (HardClearResult, error) {
	_, actor, session, err := b.withAuthenticatedLedgerSessionAccess(ctx, "administrator", true)
	if err != nil {
		return HardClearResult{}, err
	}
	defer session.Close()
	b = session.branch
	commits, snapshot, err := b.read(ctx)
	if err != nil {
		return HardClearResult{}, err
	}
	if snapshot.head == "" {
		return HardClearResult{}, queueError("queue_missing", "hard clear requires an existing queue")
	}
	state, err := Replay(commits)
	if err != nil {
		return HardClearResult{}, err
	}
	if recovered, ok, err := recoverHardClear(state, actor, requestID, reason); ok || err != nil {
		return recovered, err
	}
	if expectedGitSHA != "" && snapshot.head != expectedGitSHA {
		return HardClearResult{}, queueError("hard_clear_conflict", "queue changed since confirmation preview; no newer snapshot was reset")
	}
	next, result, err := prepareHardClear(state, actor, requestID, reason, snapshot.head)
	if err != nil {
		return result, err
	}
	if dryRun {
		result.Status = "dry_run"
		return result, nil
	}
	if _, err := b.publish(ctx, snapshot, next); err != nil {
		// A lost acknowledgment must not cause a second reset of later admissions.
		latest, _, readErr := b.read(ctx)
		if readErr != nil {
			return result, fmt.Errorf("hard_clear_uncertain: inspect request %s before retrying: %w", requestID, errors.Join(err, readErr))
		}
		current, replayErr := Replay(latest)
		if replayErr != nil {
			return result, replayErr
		}
		if recovered, ok, recoverErr := recoverHardClear(current, actor, requestID, reason); ok || recoverErr != nil {
			return recovered, recoverErr
		}
		return result, fmt.Errorf("hard_clear_conflict: queue changed or publication failed; no newer snapshot was reset: %w", err)
	}
	result.Status, result.Changed = "hard_cleared", true
	return result, nil
}

func validateHardClearState(state Projection) error {
	if !state.AdmissionPaused || !state.GrantsPaused {
		return queueError("hard_clear_not_paused", "pause both admission and grants with control before hard clearing")
	}
	for _, dispatch := range state.Dispatches {
		if !dispatch.Released {
			return queueError("hard_clear_unresolved", "native reservations remain; reconcile definitive termination or nonlaunch evidence first")
		}
	}
	for _, claim := range state.Claims {
		if claim.State == "open" || claim.Barrier == "pending" || claim.Barrier == "failed" {
			return queueError("hard_clear_unresolved", "open Claims or unsettled retained deliveries remain")
		}
	}
	for _, work := range state.Works {
		if work.State == "claimed" || work.Barrier == "pending" || work.Barrier == "failed" ||
			work.Disposition != "" && work.Disposition != "none" {
			return queueError("hard_clear_unresolved", "delivery barriers or uncertain effects remain; hard clear cannot discard reconciliation evidence")
		}
	}
	return nil
}

// This index is part of accepted hard-clear request IDs, including checkpoint receipts.
const hardClearCredentialGenerationControlIndex = 2

func hardClearControls(reason, generation string) (OperationsParameters, error) {
	params := OperationsParameters{Operations: []Operation{}}
	for _, control := range []struct {
		name  string
		value any
	}{
		{"admission_paused", true},
		{"grants_paused", true},
		hardClearCredentialGenerationControlIndex: {"credential_generation", generation},
	} {
		operation, err := Op(map[string]any{"kind": "Control", "control": control.name, "value": control.value, "reason": reason})
		if err != nil {
			return params, err
		}
		params.Operations = append(params.Operations, operation)
	}
	return params, nil
}

func prepareHardClear(state Projection, actor Actor, requestID, reason, priorGitSHA string) ([]QueueCommit, HardClearResult, error) {
	result := HardClearResult{RequestID: requestID, PriorGitSHA: priorGitSHA, PriorTip: state.Tip,
		WorkRemoved: len(state.Works), HistoryInGit: true, IdentityReset: true}
	if !ValidReason(reason) {
		return nil, result, queueError("reason_invalid", "reason must be a bounded sanitized code")
	}
	if err := validateHardClearState(state); err != nil {
		return nil, result, err
	}
	epoch := "hard_clear_" + hashBytes([]byte(requestID)) + "_" + priorGitSHA
	policy := FuturePolicy(state)
	at := time.Now().UnixMilli()
	genesis, err := Genesis(actor, policy, requestID, epoch, at)
	if err != nil {
		return nil, result, err
	}
	next := []QueueCommit{genesis}
	params, err := hardClearControls(reason, state.CredentialGeneration)
	if err != nil {
		return nil, result, err
	}
	// One control per request also supports policies with the smallest operation budget.
	for index, operation := range params.Operations {
		request, err := NewRequest(hardClearControlID(requestID, index), "control", actor, OperationsParameters{Operations: []Operation{operation}})
		if err != nil {
			return nil, result, err
		}
		next, _, _, err = BuildCandidate(next, actor, request, at)
		if err != nil {
			return nil, result, err
		}
	}
	next, err = hardClearAvailability(next, state, actor, requestID, at)
	if err != nil {
		return nil, result, err
	}
	current, err := Replay(next)
	if err != nil {
		return nil, result, err
	}
	result.Tip = current.Tip
	return next, result, nil
}

func hardClearControlID(requestID string, index int) string {
	return fmt.Sprintf("hard_clear_%s_control_%d", hashBytes([]byte(requestID)), index)
}

func hardClearAvailability(next []QueueCommit, state Projection, actor Actor, requestID string, at int64) ([]QueueCommit, error) {
	for _, pool := range sortedHardClearPools(state) {
		profiles := []string{}
		for name := range state.Deployments[pool] {
			profiles = append(profiles, name)
		}
		slices.Sort(profiles)
		for _, name := range profiles {
			deployment := state.Deployments[pool][name]
			current := deployment.Revisions[deployment.CurrentRef]
			if current.Available {
				continue
			}
			activate := false
			operation, err := Op(DeploymentOperation{Kind: "Deployment", Pool: pool, WorkerProfile: name,
				ExpectedRef: current.Profile.Ref, ExpectedContract: current.Profile.LogicalContract,
				Profile: current.Profile, Available: false, Activate: &activate, Reason: "hard_clear_preserve_unavailable"})
			if err != nil {
				return nil, err
			}
			request, err := NewRequest("hard_clear_deploy_"+hashBytes([]byte(requestID+"\n"+pool+"\n"+name)), "deployment", actor,
				OperationsParameters{Operations: []Operation{operation}})
			if err != nil {
				return nil, err
			}
			next, _, _, err = BuildCandidate(next, actor, request, at)
			if err != nil {
				return nil, err
			}
		}
	}
	return next, nil
}

func sortedHardClearPools(state Projection) []string {
	pools := []string{}
	for pool := range state.Deployments {
		pools = append(pools, pool)
	}
	slices.Sort(pools)
	return pools
}

func recoverHardClear(state Projection, actor Actor, requestID, reason string) (HardClearResult, bool, error) {
	accepted, ok := state.Requests[requestID]
	if !ok {
		return HardClearResult{}, false, nil
	}
	prefix := "hard_clear_" + hashBytes([]byte(requestID)) + "_"
	priorGitSHA, valid := strings.CutPrefix(accepted.PolicyEpoch, prefix)
	if !valid || !revisionPattern.MatchString(priorGitSHA) || accepted.Request.Kind != "policy" || !sameJSON(accepted.Actor, actor) {
		return HardClearResult{}, false, queueError("request_reuse", "request does not identify this administrator's accepted hard clear")
	}
	var generation string
	control := state.Requests[hardClearControlID(requestID, hardClearCredentialGenerationControlIndex)]
	for _, operation := range control.Operations {
		var value struct {
			Value string `json:"value"`
		}
		if err := json.Unmarshal(operation, &value); err != nil {
			return HardClearResult{}, false, err
		}
		generation = value.Value
	}
	params, err := hardClearControls(reason, generation)
	if err != nil {
		return HardClearResult{}, false, err
	}
	for index, operation := range params.Operations {
		id := hardClearControlID(requestID, index)
		request, err := NewRequest(id, "control", actor, OperationsParameters{Operations: []Operation{operation}})
		if err != nil || request.Fingerprint != state.Requests[id].Request.Fingerprint {
			return HardClearResult{}, false, queueError("request_reuse", "hard clear reason or controls differ from the accepted reset")
		}
	}
	return HardClearResult{Status: "already_committed", RequestID: requestID, PriorGitSHA: priorGitSHA,
		Tip: state.Tip, HistoryInGit: true, IdentityReset: true}, true, nil
}
