package workqueue

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestWorkExpirationRestoresOlderCheckpointTime(t *testing.T) {
	commits := testGenesis(t, nil)
	commits = testOperations(t, commits, testActor("administrator"), "old-tick", "control", mustOp(t, map[string]any{
		"kind": "Control", "control": "grants_paused", "value": false, "reason": "time_observed",
	}))
	commits = testSubmit(t, commits, "backdated-submit", testNode(t, commits, "original"))
	checkpoint, err := CompactCheckpoint(commits, strings.Repeat("c", 40), testActor("administrator"), 4000)
	if err != nil {
		t.Fatal(err)
	}
	var op CheckpointOperation
	if err := json.Unmarshal(checkpoint[0].Operations[0], &op); err != nil {
		t.Fatal(err)
	}
	snapshot, _, err := decodeCheckpointSnapshot(op.State)
	if err != nil {
		t.Fatal(err)
	}
	// Older writers retained the last timestamp instead of the maximum.
	snapshot.LastAt, checkpoint[0].At = 2000, 2000
	op.State, err = encodeCheckpointSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	logical, err := canonicalValue(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	op.StateSHA256 = hashBytes(logical)
	checkpoint[0].Operations[0] = mustOp(t, op)
	checkpoint[0].Request, err = NewRequest(checkpoint[0].Request.ID, "checkpoint", checkpoint[0].Actor, CheckpointParameters{
		PriorGitSHA: op.PriorGitSHA, PriorTip: op.PriorTip, HistorySHA256: op.HistorySHA256, StateSHA256: op.StateSHA256,
	})
	if err != nil {
		t.Fatal(err)
	}
	work := testNode(t, checkpoint, "after-old-checkpoint")
	deadline := int64(3500)
	work.ExpiresAt = &deadline
	next := testSubmit(t, checkpoint, "new-expiring-submit", work)
	state, err := Replay(next)
	if err != nil || state.timeWatermark != 4000 || state.Works[work.WorkID].CancellationReason != "work_expired" {
		t.Fatalf("older checkpoint discarded its receipts' maximum time: %v", err)
	}
	assertRecoveryReplayParity(t, next, "")
}

func expiringWorkFixture(t *testing.T) ([]QueueCommit, WorkDefinition, WorkDefinition) {
	t.Helper()
	commits := testGenesis(t, nil)
	old, fresh := testNode(t, commits, "old"), testNode(t, commits, "fresh")
	deadline := int64(3500)
	old.ExpiresAt = &deadline
	return testSubmit(t, commits, "submit", old, fresh), old, fresh
}

func TestWorkExpirationBoundaryAndHistoricalReplay(t *testing.T) {
	commits, old, fresh := expiringWorkFixture(t)
	for _, at := range []int64{3499, 3500, 3501} {
		state, err := ReplayAt(commits, at)
		if err != nil {
			t.Fatal(err)
		}
		want := "available"
		if at >= 3500 {
			want = "cancelled"
		}
		if state.Works[old.WorkID].State != want || state.Works[fresh.WorkID].State != "available" {
			t.Fatalf("at %d: %+v", at, state.Works)
		}
		next, err := PlanNext(state, "default", at)
		if err != nil {
			t.Fatal(err)
		}
		wantID := old.WorkID
		if at >= 3500 {
			wantID = fresh.WorkID
		}
		if next.WorkID != wantID {
			t.Fatalf("at %d selected %s, want %s", at, next.WorkID, wantID)
		}
	}
	state, err := Replay(commits)
	if err != nil || state.Works[old.WorkID].State != "available" {
		t.Fatalf("current-time view changed historical replay: %v", err)
	}
	next, err := PlanNext(state, "default", 3500)
	if err != nil || next.WorkID != fresh.WorkID || state.Works[old.WorkID].State != "available" {
		t.Fatal("selection did not skip expiry purely at the exact boundary")
	}
	for _, at := range []int64{-1, MaxTimestamp + 1} {
		if _, err := ReplayAt(commits, at); err == nil {
			t.Fatalf("accepted invalid replay timestamp %d", at)
		}
	}
	assertRecoveryReplayParity(t, commits, "", 3500)
}

func TestWorkExpirationCheckpointEnvelopeAndStableRequest(t *testing.T) {
	commits, old, _ := expiringWorkFixture(t)
	checkpoint, err := CompactCheckpoint(commits, strings.Repeat("b", 40), testActor("administrator"), 4000)
	if err != nil {
		t.Fatal(err)
	}
	state, err := Replay(checkpoint)
	if err != nil || state.Works[old.WorkID].CancellationReason != "work_expired" {
		t.Fatalf("checkpoint envelope did not advance committed time: %v", err)
	}
	assertRecoveryReplayParity(t, checkpoint, "")
	next, receipt, _, err := BuildCandidate(commits, testActor("producer"), commits[1].Request, 9000)
	if err != nil || receipt == nil || receipt.ID != commits[1].ID || !sameJSON(next, commits) {
		t.Fatalf("stable request retry changed its immutable admission: %v", err)
	}
	// The automatic cancellation is idempotent with the same explicit decision.
	checkpoint = testOperations(t, checkpoint, testActor("administrator"), "cancel-expired", "cancel_work", mustOp(t, map[string]any{
		"kind": "WorkCancellation", "work_id": old.WorkID, "reason": "work_expired",
	}))
	assertRecoveryReplayParity(t, checkpoint, "")
}

func TestWorkExpirationWaitsForRetainedDelivery(t *testing.T) {
	deadline := int64(3500)
	commits, assignment := boundAssignmentWithWork(t, func(work *WorkDefinition) { work.ExpiresAt = &deadline })
	member := assignment.Claims[0]
	actor := workerActor(assignment, member.Handle)
	request, err := NewRequest("retained-expiration", "finish", actor, FinishParameters{
		DispatchID: assignment.DispatchID, ClaimHandle: member.Handle, Outcome: "completed", KeepOpen: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	commits, _, _, err = BuildCandidate(commits, actor, request, 4000)
	if err != nil {
		t.Fatal(err)
	}
	state, err := ReplayAt(commits, 9000)
	if err != nil || state.Works[member.WorkID].State != "available" || state.Works[member.WorkID].Barrier != "pending" {
		t.Fatalf("expiry erased retained delivery authority: %v", err)
	}
	assertRecoveryReplayParity(t, commits, "", 9000)
	evidence := terminalFor(state, assignment)
	evidence.Kind, evidence.Source, evidence.Receipt = "delivery", "verified_receipts", "retained-expiry-delivery"
	commits = testOperations(t, commits, testActor("reconciler"), "retained-result", "result", mustOp(t, map[string]any{
		"kind": "Result", "work_id": member.WorkID, "claim_id": member.ClaimID,
		"completion_id": state.Works[member.WorkID].CompletionID, "descriptor": map[string]any{"ok": true}, "evidence": evidence,
	}))
	state, err = Replay(commits)
	if err != nil || state.Works[member.WorkID].CancellationReason != "work_expired" ||
		state.Claims[member.ClaimID].State != "completed" || state.Claims[member.ClaimID].Barrier != "verified" || state.Stats.Dispatches != 1 {
		t.Fatalf("retained delivery failed to settle before expiry: %v", err)
	}
	assertRecoveryReplayParity(t, commits, "")
}

func TestWorkExpirationCommittedTimeCheckpointAndRollbackParity(t *testing.T) {
	commits, old, fresh := expiringWorkFixture(t)
	commits = testOperations(t, commits, testActor("administrator"), "tick", "control", mustOp(t, map[string]any{
		"kind": "Control", "control": "grants_paused", "value": false, "reason": "time_observed",
	}))
	// A later admission with a backdated timestamp cannot revive eligibility.
	late := testNode(t, commits, "late")
	late.ExpiresAt = old.ExpiresAt
	commits = testSubmit(t, commits, "late-submit", late)
	state, err := Replay(commits)
	if err != nil || state.Works[old.WorkID].CancellationReason != "work_expired" ||
		state.Works[late.WorkID].CancellationReason != "work_expired" || state.Works[fresh.WorkID].State != "available" {
		t.Fatalf("committed expiration was lost on clock rollback: %v", err)
	}
	checkpoint, err := CompactCheckpoint(commits, strings.Repeat("a", 40), testActor("administrator"), 4000)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := ReplayAt(checkpoint, 0)
	if err != nil || !sameJSON(state.Works, restored.Works) || restored.timeWatermark != 4000 {
		t.Fatalf("checkpoint changed deadlines or watermark: %v", err)
	}
	if _, err := CompactCheckpoint(commits, strings.Repeat("a", 40), testActor("administrator"), 2000); err == nil {
		t.Fatal("checkpoint discarded the committed time watermark")
	}
	reordered := slices.Clone(commits)
	slices.Reverse(reordered)
	reordered = append(reordered, commits[1])
	assertRecoveryReplayParity(t, reordered, "")
	assertRecoveryReplayParity(t, checkpoint, "")
}

func TestWorkExpirationPreservesClaimsCompletionAndNativeCapacity(t *testing.T) {
	deadline := int64(3500)
	commits, assignment := boundAssignmentWithWork(t, func(work *WorkDefinition) { work.ExpiresAt = &deadline })
	state, err := ReplayAt(commits, 9000)
	if err != nil || state.Stats.Claimed != 3 || state.Stats.Dispatches != 1 {
		t.Fatalf("deadline revoked running Claims or released capacity: %+v %v", state.Stats, err)
	}
	clocks := state.Clocks
	if !expiringGrant(Projection{Works: map[string]*WorkState{
		assignment.Claims[0].WorkID: {WorkDefinition: WorkDefinition{ExpiresAt: &deadline}, State: "available"},
	}}, []Operation{mustOp(t, ClaimOperation{Kind: "Claim", WorkID: assignment.Claims[0].WorkID})}, deadline) {
		t.Fatal("publication failed to recheck the exact deadline")
	}
	commits = finishMember(t, commits, assignment, 0, "completed")
	commits = finishMember(t, commits, assignment, 1, "cancelled")
	state, err = Replay(commits)
	if err != nil || state.Claims[assignment.Claims[0].ClaimID].State != "completed" ||
		state.Works[assignment.Claims[0].WorkID].Barrier != "pending" ||
		state.Works[assignment.Claims[1].WorkID].CancellationReason != "work_expired" ||
		state.Claims[assignment.Claims[1].ClaimID].State != "cancelled" ||
		state.Claims[assignment.Claims[2].ClaimID].State != "open" ||
		state.Stats.Dispatches != 1 || !sameJSON(clocks, state.Clocks) {
		t.Fatalf("expiry changed independent outcomes, charges or reservation: %v", err)
	}
	assertRecoveryReplayParity(t, commits, "")
}

func TestWorkExpirationImmutableAndDoesNotSatisfyDependencies(t *testing.T) {
	commits, old, fresh := expiringWorkFixture(t)
	child := testNode(t, commits, "child")
	child.DependsOn = []Dependency{{Kind: "work", WorkID: old.WorkID}}
	commits = testSubmit(t, commits, "child-submit", child)
	commits = testOperations(t, commits, testActor("administrator"), "tick", "control", mustOp(t, map[string]any{
		"kind": "Control", "control": "grants_paused", "value": false, "reason": "time_observed",
	}))
	state, err := Replay(commits)
	if err != nil {
		t.Fatal(err)
	}
	next, err := PlanNext(state, "default", 4000)
	if err != nil || next.WorkID != fresh.WorkID || state.Works[old.WorkID].ResultCommitID != "" {
		t.Fatal("expired predecessor unlocked its child or fabricated a Result")
	}
	changed := old
	deadline := int64(6000)
	changed.ExpiresAt = &deadline
	request, err := NewRequest("change-expiration", "submit", testActor("producer"), SubmitParameters{Nodes: []WorkDefinition{changed}})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := BuildCandidate(commits, testActor("producer"), request, 4000); err == nil {
		t.Fatal("resubmission extended immutable expiration")
	}
	for _, at := range []int64{-1, MaxTimestamp + 1} {
		invalid := old
		invalid.ExpiresAt = &at
		request, err := NewRequest("invalid-expiration", "submit", testActor("producer"), SubmitParameters{Nodes: []WorkDefinition{invalid}})
		if err == nil {
			_, _, _, err = BuildCandidate(commits, testActor("producer"), request, 4000)
		}
		if err == nil {
			t.Fatalf("accepted invalid expiration %d", at)
		}
	}
	assertRecoveryReplayParity(t, commits, "")
}
