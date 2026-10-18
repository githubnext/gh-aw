package cli

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/workqueue"
)

type workQueueSnapshot struct {
	EvaluatedAt     int64  `json:"evaluated_at"`
	Status          string `json:"status"`
	Tip             string `json:"tip"`
	PolicyEpoch     string `json:"policy_epoch"`
	Repository      string `json:"repository"`
	Branch          string `json:"branch"`
	AdmissionPaused bool   `json:"admission_paused"`
	GrantsPaused    bool   `json:"grants_paused"`
}

func workQueueSnapshotAt(state workqueue.Projection, branch string, at int64) workQueueSnapshot {
	return workQueueSnapshot{
		EvaluatedAt: at, Status: "committed_snapshot", Tip: state.Tip, PolicyEpoch: state.PolicyEpoch,
		Repository: state.Repository, Branch: branch, AdmissionPaused: state.AdmissionPaused, GrantsPaused: state.GrantsPaused,
	}
}

type workQueueOldestWork struct {
	WorkID     string `json:"work_id"`
	AdmittedAt int64  `json:"admitted_at"`
	AgeMS      int64  `json:"age_ms"`
}

type workQueuePoolStats struct {
	Pool               string                    `json:"pool"`
	PolicyInstalled    bool                      `json:"policy_installed"`
	WorkByState        map[string]int            `json:"work_by_state"`
	ClaimsByState      map[string]int            `json:"claims_by_state"`
	DeliveryByState    map[string]int            `json:"delivery_by_state"`
	NativeReservations int                       `json:"native_reservations"`
	LogicalLimit       int                       `json:"logical_limit"`
	NativeLimit        int                       `json:"native_limit"`
	MaxAttempts        int                       `json:"max_attempts"`
	SchedulingEligible int                       `json:"scheduling_eligible"`
	IneligibleReasons  map[string]int            `json:"ineligible_reasons"`
	OldestAvailable    *workQueueOldestWork      `json:"oldest_available"`
	UnknownAge         int                       `json:"available_age_unknown"`
	Prediction         workQueueReplayPrediction `json:"prediction"`
}

type workQueueActivity struct {
	Since        int64          `json:"since"`
	Until        int64          `json:"until"`
	Transactions int            `json:"transactions"`
	Operations   map[string]int `json:"operations"`
	LastCommitID string         `json:"last_commit_id,omitempty"`
	LastAt       *int64         `json:"last_at"`
}

type workQueueStatsReport struct {
	workqueue.Stats
	workQueueSnapshot
	PolicyMode               string               `json:"policy_mode"`
	ClaimsByState            map[string]int       `json:"claims_by_state"`
	ClaimCancellationReasons map[string]int       `json:"claim_cancellation_reasons"`
	DeliveryByState          map[string]int       `json:"delivery_by_state"`
	DeliveryFailureReasons   map[string]int       `json:"delivery_failure_reasons"`
	NativeReservations       int                  `json:"native_reservations"`
	DispatchesTotal          int                  `json:"dispatches_total"`
	DispatchesByState        map[string]int       `json:"dispatches_by_state"`
	ReservationsByState      map[string]int       `json:"reservations_by_state"`
	Pools                    []workQueuePoolStats `json:"pools"`
	Activity                 workQueueActivity    `json:"activity"`
}

func workQueueWorkStates() map[string]int {
	return map[string]int{"available": 0, "claimed": 0, "completed": 0, "cancelled": 0}
}

func workQueueClaimStates() map[string]int {
	return map[string]int{"open": 0, "completed": 0, "cancelled": 0}
}

func workQueueDeliveryStates() map[string]int {
	return map[string]int{"pending": 0, "verified": 0, "failed": 0}
}

func workQueueAdmittedAt(state workqueue.Projection, work *workqueue.WorkState) *int64 {
	if work.Position.Commit < 0 || work.Position.Commit >= len(state.RequestOrder) {
		return nil
	}
	commit, ok := state.Requests[state.RequestOrder[work.Position.Commit]]
	if !ok || commit.Request.Kind != "submit" {
		return nil
	}
	return &commit.At
}

func workQueueClaimDelivery(state workqueue.Projection, claim *workqueue.ClaimState) string {
	if claim.State != "completed" {
		return "none"
	}
	if claim.Barrier != "" {
		return claim.Barrier
	}
	if work := state.Works[claim.WorkID]; work != nil && work.ClaimID == claim.ClaimID {
		return work.Barrier
	}
	return "pending"
}

func workQueueStatsAt(state workqueue.Projection, branch string, at int64) (workQueueStatsReport, error) {
	report := workQueueStatsReport{
		Stats: state.Stats, workQueueSnapshot: workQueueSnapshotAt(state, branch, at),
		ClaimsByState: workQueueClaimStates(), ClaimCancellationReasons: map[string]int{},
		DeliveryByState: workQueueDeliveryStates(), DeliveryFailureReasons: map[string]int{},
		NativeReservations: state.Stats.Dispatches, DispatchesTotal: len(state.Dispatches),
		DispatchesByState: map[string]int{}, ReservationsByState: map[string]int{}, Pools: []workQueuePoolStats{},
	}
	if state.Policy != nil {
		report.PolicyMode = state.Policy.Mode
	}
	pools, err := workQueuePoolsAt(state, at)
	if err != nil {
		return report, err
	}
	if err := workQueueClaimStats(state, pools, &report); err != nil {
		return report, err
	}
	names := make([]string, 0, len(pools))
	for name := range pools {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if !pools[name].PolicyInstalled {
			pools[name].Prediction = workQueueReplayPrediction{Reason: "pool_not_installed", Assignments: []workQueueReplayAssignment{}}
			report.Pools = append(report.Pools, *pools[name])
			continue
		}
		decision, err := workqueue.PlanDispatch(state, workqueue.DispatchParameters{
			Pool: name, MaxClaims: 1, MaxDispatches: 1, MaxBytes: state.Policy.Limits.AssignmentBytes,
		}, "snapshot_prediction", "snapshot_commit", at)
		if err != nil {
			return report, err
		}
		pools[name].Prediction = workQueueCompactPrediction(decision)
		report.Pools = append(report.Pools, *pools[name])
	}
	report.Activity, err = workQueueActivityAt(state, at)
	return report, err
}

func workQueuePoolsAt(state workqueue.Projection, at int64) (map[string]*workQueuePoolStats, error) {
	pools := map[string]*workQueuePoolStats{}
	for _, work := range state.Works {
		pools[work.Pool] = nil
	}
	for _, dispatch := range state.Dispatches {
		pools[dispatch.Pool] = nil
	}
	if state.Policy != nil {
		for name := range state.Policy.Pools {
			pools[name] = nil
		}
	}
	for name := range pools {
		pool := &workQueuePoolStats{
			Pool: name, WorkByState: workQueueWorkStates(), ClaimsByState: workQueueClaimStates(),
			DeliveryByState: workQueueDeliveryStates(), IneligibleReasons: map[string]int{},
		}
		if state.Policy != nil {
			if policy, ok := state.Policy.Pools[name]; ok {
				pool.PolicyInstalled = true
				pool.LogicalLimit, pool.NativeLimit, pool.MaxAttempts = policy.LogicalLimit, policy.NativeLimit, policy.Retry.MaxAttempts
			}
		}
		pools[name] = pool
	}
	for _, work := range state.Works {
		pool := pools[work.Pool]
		pool.WorkByState[work.State]++
		if work.State != "available" {
			continue
		}
		if pool.PolicyInstalled {
			explanation, err := workqueue.ExplainWork(state, work.WorkID, at)
			if err != nil {
				return nil, err
			}
			if explanation.SchedulingEligible {
				pool.SchedulingEligible++
			} else {
				pool.IneligibleReasons[explanation.Reason]++
			}
		} else {
			pool.IneligibleReasons["pool_not_installed"]++
		}
		admitted := workQueueAdmittedAt(state, work)
		if admitted == nil || *admitted > at {
			pool.UnknownAge++
		} else if pool.OldestAvailable == nil || *admitted < pool.OldestAvailable.AdmittedAt ||
			*admitted == pool.OldestAvailable.AdmittedAt && work.WorkID < pool.OldestAvailable.WorkID {
			pool.OldestAvailable = &workQueueOldestWork{WorkID: work.WorkID, AdmittedAt: *admitted, AgeMS: at - *admitted}
		}
	}
	return pools, nil
}

func workQueueClaimStats(state workqueue.Projection, pools map[string]*workQueuePoolStats, report *workQueueStatsReport) error {
	for _, claim := range state.Claims {
		pool := pools[state.Works[claim.WorkID].Pool]
		report.ClaimsByState[claim.State]++
		pool.ClaimsByState[claim.State]++
		if claim.State == "cancelled" {
			report.ClaimCancellationReasons[claim.CancellationReason]++
		}
		if claim.State == "completed" {
			delivery := workQueueClaimDelivery(state, claim)
			report.DeliveryByState[delivery]++
			pool.DeliveryByState[delivery]++
			terminal, err := workQueueClaimTerminal(state, claim)
			if err != nil {
				return err
			}
			if terminal.Kind == "DeliveryFailure" {
				report.DeliveryFailureReasons[terminal.Reason]++
			}
		}
	}
	for _, dispatch := range state.Dispatches {
		report.DispatchesByState[dispatch.State]++
		if !dispatch.Released {
			report.ReservationsByState[dispatch.State]++
			pools[dispatch.Pool].NativeReservations++
		}
	}
	return nil
}

func workQueueActivityAt(state workqueue.Projection, at int64) (workQueueActivity, error) {
	activity := workQueueActivity{Since: max(0, at-int64(24*time.Hour/time.Millisecond)), Until: at, Operations: map[string]int{}}
	requestOrder := make(map[string]int, len(state.RequestOrder))
	for index, requestID := range state.RequestOrder {
		requestOrder[requestID] = index
	}
	lastRequestOrder := -1
	for _, commit := range state.Requests {
		if commit.At > at {
			continue
		}
		order := -1
		if index, ok := requestOrder[commit.Request.ID]; ok {
			order = index
		}
		if activity.LastAt == nil || commit.At > *activity.LastAt ||
			commit.At == *activity.LastAt && (order > lastRequestOrder ||
				order == lastRequestOrder && commit.ID > activity.LastCommitID) {
			timestamp := commit.At
			activity.LastAt, activity.LastCommitID = &timestamp, commit.ID
			lastRequestOrder = order
		}
		if commit.At < activity.Since {
			continue
		}
		activity.Transactions++
		for _, operation := range commit.Operations {
			var event struct {
				Kind string `json:"kind"`
			}
			if err := json.Unmarshal(operation, &event); err != nil {
				return activity, fmt.Errorf("read work queue activity: %w", err)
			}
			activity.Operations[event.Kind]++
		}
	}
	return activity, nil
}

func workQueueCompactPrediction(decision workqueue.Decision) workQueueReplayPrediction {
	prediction := workQueueReplayPrediction{Reason: decision.Reason, Assignments: []workQueueReplayAssignment{}}
	for _, assignment := range decision.Assignments {
		member := workQueueReplayAssignment{WorkerProfile: assignment.WorkerProfile, WorkIDs: []string{}}
		for _, claim := range assignment.Claims {
			member.WorkIDs = append(member.WorkIDs, claim.WorkID)
		}
		prediction.Assignments = append(prediction.Assignments, member)
	}
	return prediction
}

func workQueueStatsText(report workQueueStatsReport) string {
	s := report.Stats
	lines := []string{fmt.Sprintf("Work: %d; available: %d; claimed: %d; completed: %d; cancelled: %d; Claims: %d; native reservations: %d; graph nodes: %d",
		s.Work, s.Available, s.Claimed, s.Completed, s.Cancelled, s.Claims, s.Dispatches, s.Nodes),
		fmt.Sprintf("Claims: open=%d completed=%d cancelled=%d | delivery: pending=%d verified=%d failed=%d",
			report.ClaimsByState["open"], report.ClaimsByState["completed"], report.ClaimsByState["cancelled"],
			report.DeliveryByState["pending"], report.DeliveryByState["verified"], report.DeliveryByState["failed"]),
		fmt.Sprintf("Admission paused=%t | grants paused=%t | policy=%s | tip=%s",
			report.AdmissionPaused, report.GrantsPaused, workQueueText(report.PolicyMode), workQueueText(report.Tip))}
	for _, pool := range report.Pools {
		if !pool.PolicyInstalled {
			lines = append(lines, fmt.Sprintf("Pool %s: historical only (not in current Policy)", workQueueText(pool.Pool)))
			continue
		}
		lines = append(lines, fmt.Sprintf("Pool %s: available=%d eligible=%d reservations=%d/%d | prediction=%s (not a grant)",
			workQueueText(pool.Pool), pool.WorkByState["available"], pool.SchedulingEligible,
			pool.NativeReservations, pool.NativeLimit, workQueueText(pool.Prediction.Reason)))
	}
	lines = append(lines, "Completion is not verified delivery. Reservations and run bindings are ledger evidence, not live Actions status.")
	return strings.Join(lines, "\n")
}
