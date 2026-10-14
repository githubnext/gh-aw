package workqueue

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestKeepOpenCompletionDeliveryAndReclaim(t *testing.T) {
	commits, assignment, target := boundEffectAssignment(t)
	member := assignment.Claims[0]
	child := testNode(t, commits, "retained-child")
	child.DependsOn = []Dependency{{Kind: "work", WorkID: member.WorkID}}
	commits = testSubmit(t, commits, "retained-child-submit", child)
	actor := workerActor(assignment, member.Handle)
	parameters := FinishParameters{DispatchID: assignment.DispatchID, ClaimHandle: member.Handle, Outcome: "completed", KeepOpen: true}
	request, _ := NewRequest("retained-finish", "finish", actor, parameters)
	next, published, _, err := BuildCandidate(commits, actor, request, 4000)
	if err != nil || published == nil {
		t.Fatalf("keep-open completion: %v", err)
	}
	commits = next
	assertKeepOpenRuntimeParity(t, commits)
	state, err := Replay(commits)
	if err != nil {
		t.Fatal(err)
	}
	work := state.Works[member.WorkID]
	if work.State != "available" || work.Barrier != "pending" || !state.Claims[member.ClaimID].KeepOpen {
		t.Fatalf("retained completion lost its barrier: %+v", work)
	}
	if err := AuthorizeEffect(state, actor, target); err != nil {
		t.Fatal(err)
	}
	explanation, _ := ExplainWork(state, member.WorkID, 4000)
	if explanation.SchedulingEligible || explanation.Reason != "delivery_pending" {
		t.Fatal("retained completion was granted before delivery")
	}
	retry, _ := NewRequest("retained-finish-retry", "finish", actor, parameters)
	if _, repeated, _, err := BuildCandidate(commits, actor, retry, 4000); err != nil || repeated != nil {
		t.Fatalf("same-choice retry must be a no-op: %v", err)
	}
	parameters.KeepOpen = false
	conflict, _ := NewRequest("retained-finish-conflict", "finish", actor, parameters)
	if _, _, _, err := BuildCandidate(commits, actor, conflict, 4000); err == nil {
		t.Fatal("conflicting retention choice accepted")
	}
	checkpoint, err := CompactCheckpoint(commits, strings.Repeat("a", 40), testActor("administrator"), 4000)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Replay(checkpoint)
	if err != nil || !sameJSON(state.Claims, restored.Claims) || !sameJSON(state.Works, restored.Works) {
		t.Fatalf("pending retention checkpoint changed projection: %v", err)
	}
	evidence := terminalFor(state, assignment)
	evidence.Kind, evidence.Source, evidence.Receipt = "delivery", "verified_receipts", "retained-delivery"
	result := mustOp(t, map[string]any{"kind": "Result", "work_id": member.WorkID, "claim_id": member.ClaimID,
		"completion_id": work.CompletionID, "descriptor": map[string]any{"ok": true}, "evidence": evidence})
	commits = testOperations(t, commits, testActor("reconciler"), "retained-result", "result", result)
	assertKeepOpenRuntimeParity(t, commits)
	state, err = Replay(commits)
	if err != nil {
		t.Fatal(err)
	}
	work = state.Works[member.WorkID]
	if work.State != "available" || work.ClaimID != "" || work.Barrier != "none" || work.Attempts != 0 ||
		state.Claims[member.ClaimID].Barrier != "verified" {
		t.Fatalf("verified retained completion did not release delivery barrier: %+v", work)
	}
	if err := AuthorizeEffect(state, actor, target); err == nil {
		t.Fatal("settled historical Claim authorized effects twice")
	}
	childExplanation, _ := ExplainWork(state, child.WorkID, 4000)
	if childExplanation.Ready || childExplanation.Reason != "dependency_result_unavailable" {
		t.Fatal("retained successful Result satisfied DAG dependency")
	}
	commits, decision := testGrant(t, commits, "retained-reclaim", 1, 1)
	assertKeepOpenRuntimeParity(t, commits)
	if len(decision.Assignments) != 1 || decision.Assignments[0].Claims[0].WorkID != member.WorkID {
		t.Fatal("retained Work was not claimable after verified delivery")
	}
	state, _ = Replay(commits)
	if state.Claims[member.ClaimID].Barrier != "verified" || state.Works[member.WorkID].ClaimID == member.ClaimID {
		t.Fatal("new Claim rewrote historical completed Claim")
	}
	checkpoint, err = CompactCheckpoint(commits, strings.Repeat("b", 40), testActor("administrator"), 4000)
	restored, replayErr := Replay(checkpoint)
	if err != nil || replayErr != nil || !sameJSON(state.Claims, restored.Claims) || !sameJSON(state.TerminalBarriers, restored.TerminalBarriers) {
		t.Fatalf("reclaimed retention checkpoint lost history: %v %v", err, replayErr)
	}
}

func assertKeepOpenRuntimeParity(t *testing.T, commits []QueueCommit) {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("cross-runtime parity requires Node")
	}
	input, _ := json.Marshal(commits)
	script := `
	const q = require("../../actions/setup/js/work_queue_replay.cjs");
	const log = JSON.parse(require("fs").readFileSync(0, "utf8"));
	const state = q.replayTransactions(log);
	const checkpoint = q.prepareCheckpoint(log, "c".repeat(40), {role:"administrator",principal:"42",repository:"owner/repo"}, 4000);
	console.log(require("../../actions/setup/js/work_queue_codec.cjs").canonical({works: Object.fromEntries([...state.works].map(([id,w]) => {
	const {completion_at, admission_request_id, ...wire} = w; return [id,wire];
	})), claims: Object.fromEntries(state.claims), barriers: Object.fromEntries(state.terminalBarriers), checkpoint}));
	`
	command := exec.Command(node, "-e", script)
	command.Stdin = bytes.NewReader(input)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("JS retained replay: %s %v", output, err)
	}
	var result struct {
		Works      map[string]*WorkState  `json:"works"`
		Claims     map[string]*ClaimState `json:"claims"`
		Barriers   map[string]Operation   `json:"barriers"`
		Checkpoint []QueueCommit          `json:"checkpoint"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatal(err)
	}
	state, _ := Replay(commits)
	if !sameJSON(state.Works, result.Works) || !sameJSON(state.Claims, result.Claims) || !sameJSON(state.TerminalBarriers, result.Barriers) {
		t.Fatal("Go/JS retained projections diverged")
	}
	restored, err := Replay(result.Checkpoint)
	if err != nil || !sameJSON(state.Works, restored.Works) || !sameJSON(state.Claims, restored.Claims) {
		t.Fatalf("JS checkpoint did not restore Go retained projection: %v", err)
	}
}

func TestKeepOpenRequestValidationAndDefaultNormalization(t *testing.T) {
	commits, assignment := boundAssignment(t)
	actor := workerActor(assignment, "h1")
	for _, value := range []any{nil, "true", 1} {
		request, _ := NewRequest("invalid-retention", "finish", actor, map[string]any{
			"dispatch_id": assignment.DispatchID, "claim_handle": "h1", "outcome": "completed", "keep_open": value,
		})
		if _, _, _, err := BuildCandidate(commits, actor, request, 4000); err == nil {
			t.Fatalf("invalid keep_open %v accepted", value)
		}

	}
	request, _ := NewRequest("cancel-retained", "finish", actor, FinishParameters{
		DispatchID: assignment.DispatchID, ClaimHandle: "h1", Outcome: "cancelled", KeepOpen: true,
	})
	if _, _, _, err := BuildCandidate(commits, actor, request, 4000); err == nil {
		t.Fatal("cancelled keep_open accepted")
	}
	absent := json.RawMessage(`{"dispatch_id":"dispatch","claim_handle":"h1","outcome":"completed"}`)
	explicit := json.RawMessage(`{"dispatch_id":"dispatch","claim_handle":"h1","outcome":"completed","keep_open":false}`)
	a, _ := Fingerprint(actor, "finish", absent)
	b, _ := Fingerprint(actor, "finish", explicit)
	if a != b {
		t.Fatal("absent and false keep_open have different meaning")
	}
}

func TestKeepOpenFailedDeliveryRequiresExplicitDisposition(t *testing.T) {
	commits, assignment := boundAssignment(t, func(policy *Policy) {
		pool := policy.Pools["default"]
		pool.Retry.MaxAttempts = 1
		policy.Pools["default"] = pool
	})
	member := assignment.Claims[0]
	actor := workerActor(assignment, member.Handle)
	request, _ := NewRequest("retained-final-attempt", "finish", actor, FinishParameters{
		DispatchID: assignment.DispatchID, ClaimHandle: member.Handle, Outcome: "completed", KeepOpen: true,
	})
	commits, _, _, err := BuildCandidate(commits, actor, request, 4000)
	if err != nil {
		t.Fatal(err)
	}
	state, _ := Replay(commits)
	work := state.Works[member.WorkID]
	terminal := terminalFor(state, assignment)
	terminal.Attempts, terminal.Effects = state.Policy.Pools["default"].Reconciliation.MaxAttempts, "unknown"
	failure := mustOp(t, map[string]any{"kind": "DeliveryFailure", "work_id": member.WorkID, "claim_id": member.ClaimID,
		"completion_id": work.CompletionID, "reason": "verification_exhausted", "disposition": "unknown", "evidence": terminal})
	commits = testOperations(t, commits, testActor("reconciler"), "retained-failure", "delivery_failure", failure)
	state, _ = Replay(commits)
	explanation, _ := ExplainWork(state, member.WorkID, 4000)
	if explanation.SchedulingEligible || explanation.Reason != "delivery_failed" {
		t.Fatal("failed retained delivery permitted another Claim")
	}
	assertKeepOpenRuntimeParity(t, commits)
	commits = testOperations(t, commits, testActor("administrator"), "retained-disposition", "cancel_work",
		mustOp(t, map[string]any{"kind": "WorkCancellation", "work_id": member.WorkID, "reason": "inspected"}))
	state, _ = Replay(commits)
	if state.Works[member.WorkID].State != "cancelled" || state.Claims[member.ClaimID].State != "completed" {
		t.Fatal("explicit disposition rewrote completed Claim")
	}
}

func TestKeepOpenCheckpointCannotRewriteChoice(t *testing.T) {
	commits, assignment := boundAssignment(t)
	member := assignment.Claims[0]
	actor := workerActor(assignment, member.Handle)
	request, _ := NewRequest("retained-choice", "finish", actor, FinishParameters{
		DispatchID: assignment.DispatchID, ClaimHandle: member.Handle, Outcome: "completed", KeepOpen: true,
	})
	commits, _, _, err := BuildCandidate(commits, actor, request, 4000)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err := CompactCheckpoint(commits, strings.Repeat("a", 40), testActor("administrator"), 4000)
	if err != nil {
		t.Fatal(err)
	}
	var operation CheckpointOperation
	if err := json.Unmarshal(checkpoint[0].Operations[0], &operation); err != nil {
		t.Fatal(err)
	}
	snapshot, _, err := decodeCheckpointSnapshot(operation.State)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Claims[member.ClaimID].KeepOpen = false
	for index := range snapshot.Requests {
		for eventIndex, event := range snapshot.Requests[index].Events {
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(event, &fields); err != nil {
				t.Fatal(err)
			}
			if string(fields["kind"]) == `"Completion"` {
				delete(fields, "keep_open")
				snapshot.Requests[index].Events[eventIndex], _ = json.Marshal(fields)
			}
		}
	}
	logical, _ := canonicalValue(snapshot)
	operation.State, _ = encodeCheckpointSnapshot(snapshot)
	operation.StateSHA256 = hashBytes(logical)
	checkpoint[0].Operations[0] = mustOp(t, operation)
	checkpoint[0].Request, _ = NewRequest(checkpoint[0].Request.ID, "checkpoint", checkpoint[0].Actor, CheckpointParameters{
		PriorGitSHA: operation.PriorGitSHA, PriorTip: operation.PriorTip,
		HistorySHA256: operation.HistorySHA256, StateSHA256: operation.StateSHA256,
	})
	if _, err := Replay(checkpoint); err == nil {
		t.Fatal("checkpoint rewrote retained Completion's request-bound choice")
	}
}
