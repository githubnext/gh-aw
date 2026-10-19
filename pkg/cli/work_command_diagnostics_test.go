package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/workqueue"
)

func workDiagnosticsFixture(t *testing.T) (workqueue.Projection, []workqueue.QueueCommit, []string) {
	t.Helper()
	state, commits, ids := workOperatorFixture(t, true)
	var dispatch *workqueue.DispatchState
	for _, candidate := range state.Dispatches {
		dispatch = candidate
	}
	actor := workqueue.Actor{Role: "dispatcher", Principal: "1001", Repository: "owner/repo",
		Workflow: ".github/workflows/dispatcher.lock.yml", RunID: "101", RunAttempt: 1}
	appendRequest := func(id, kind string, parameters any, at int64) {
		t.Helper()
		request, err := workqueue.NewRequest(id, kind, actor, parameters)
		if err != nil {
			t.Fatal(err)
		}
		commits, _, _, err = workqueue.BuildCandidate(commits, actor, request, at)
		if err != nil {
			t.Fatal(err)
		}
	}
	operations := func(value any) workqueue.OperationsParameters {
		return workqueue.OperationsParameters{Operations: []workqueue.Operation{mustWorkQueueOperation(t, value)}}
	}
	appendRequest("start", "dispatch", operations(map[string]any{
		"kind": "Dispatch", "dispatch_id": dispatch.DispatchID, "state": "started", "sender": actor,
	}), 4000)
	binding := workqueue.RunBinding{
		RunID: "202", RunAttempt: 1, Repository: actor.Repository, Workflow: dispatch.Profile.Workflow,
		Ref: dispatch.Profile.Ref, Principal: actor.Principal, Event: "workflow_dispatch",
	}
	evidence := workqueue.Evidence{
		Kind: "reconciliation", Source: "github_api", Repository: binding.Repository, Workflow: binding.Workflow,
		Ref: binding.Ref, Principal: binding.Principal, RunID: binding.RunID, RunAttempt: 1, CheckedAt: 5000,
	}
	actor.Role, actor.Workflow, actor.RunID, actor.RunAttempt = "reconciler", "", "", 0
	appendRequest("bind", "dispatch", operations(map[string]any{
		"kind": "Dispatch", "dispatch_id": dispatch.DispatchID, "state": "bound", "run": binding, "evidence": evidence,
	}), 5000)
	actor.Role, actor.Workflow, actor.RunID, actor.RunAttempt = "worker", binding.Workflow, binding.RunID, 1
	actor.DispatchID = dispatch.DispatchID
	for index, member := range dispatch.Claims {
		actor.ClaimHandle = member.Handle
		outcome := "completed"
		if index == 2 {
			outcome = "cancelled"
		}
		appendRequest("finish-"+member.Handle, "finish", workqueue.FinishParameters{
			DispatchID: dispatch.DispatchID, ClaimHandle: member.Handle, Outcome: outcome,
		}, 6000)
	}
	state, err := workqueue.Replay(commits)
	if err != nil {
		t.Fatal(err)
	}
	actor = workqueue.Actor{Role: "reconciler", Principal: "1001", Repository: "owner/repo"}
	evidence.Kind, evidence.Source, evidence.Receipt = "delivery", "verified_receipts", "RECEIPT-MUST-STAY-PRIVATE"
	appendRequest("result", "result", operations(map[string]any{
		"kind": "Result", "work_id": ids[0], "claim_id": state.Works[ids[0]].ClaimID,
		"completion_id": state.Works[ids[0]].CompletionID,
		"descriptor":    map[string]any{"detail": "RESULT-MUST-STAY-PRIVATE"}, "evidence": evidence,
	}), 7000)
	evidence.Kind, evidence.Source, evidence.Status, evidence.Conclusion = "terminal_run", "github_api", "completed", "failure"
	evidence.CheckedAt, evidence.Attempts, evidence.Effects = 400000, 5, "unknown"
	appendRequest("delivery-failed", "delivery_failure", operations(map[string]any{
		"kind": "DeliveryFailure", "work_id": ids[1], "claim_id": state.Works[ids[1]].ClaimID,
		"completion_id": state.Works[ids[1]].CompletionID, "reason": "verification_exhausted",
		"disposition": "unknown", "evidence": evidence,
	}), 400000)
	state, err = workqueue.Replay(commits)
	if err != nil {
		t.Fatal(err)
	}
	return state, commits, ids
}

func assertWorkDiagnosticsPrivate(t *testing.T, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"PAYLOAD-MUST", "SECRET-MUST", "RESULT-MUST", "RECEIPT-MUST", `"payload"`, `"descriptor"`, `"parameters"`} {
		if bytes.Contains(data, []byte(forbidden)) {
			t.Fatalf("diagnostics exposed %s", forbidden)
		}
	}
}

func TestWorkQueueDiagnosticStatsSeparateDeliveryAndReservations(t *testing.T) {
	state, _, _ := workDiagnosticsFixture(t)
	before, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	report, err := workQueueStatsAt(state, "work-queue", 500000)
	if err != nil {
		t.Fatal(err)
	}
	if report.Completed != 2 || report.ClaimsByState["open"] != 0 || report.ClaimsByState["completed"] != 2 ||
		report.ClaimsByState["cancelled"] != 1 || report.Dispatches != 1 || report.NativeReservations != 1 || report.DispatchesTotal != 1 ||
		report.DispatchesByState["bound"] != 1 || report.ReservationsByState["bound"] != 1 {
		t.Fatalf("ownership and native reservation counts were conflated: %+v", report)
	}
	if report.DeliveryByState["verified"] != 1 || report.DeliveryByState["failed"] != 1 || report.DeliveryByState["pending"] != 0 {
		t.Fatalf("Completion was treated as verified Result: %+v", report.DeliveryByState)
	}
	if report.DeliveryFailureReasons["verification_exhausted"] != 1 || report.ClaimCancellationReasons["worker_cancelled"] != 1 {
		t.Fatalf("failure/cancellation reasons are missing: %+v", report)
	}
	pool := report.Pools[0]
	if pool.WorkByState["completed"] != 2 || pool.ClaimsByState["completed"] != 2 || pool.NativeReservations != 1 ||
		!pool.PolicyInstalled || pool.LogicalLimit != state.Policy.Pools["default"].LogicalLimit || pool.MaxAttempts != 3 ||
		pool.WorkByState["available"] != 1 || len(pool.Prediction.Assignments) != 1 {
		t.Fatalf("pool diagnostics are incomplete: %+v", pool)
	}
	if report.Activity.Operations["Completion"] != 2 || report.Activity.Operations["Result"] != 1 ||
		report.Activity.Operations["DeliveryFailure"] != 1 || report.Activity.LastAt == nil || *report.Activity.LastAt != 400000 {
		t.Fatalf("activity did not count committed lifecycle operations: %+v", report.Activity)
	}
	assertWorkDiagnosticsPrivate(t, report)
	after, err := json.Marshal(state)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("stats prediction mutated the authoritative projection")
	}
	text := workQueueStatsText(report)
	if !strings.Contains(text, "verified=1 failed=1") || !strings.Contains(text, "not live Actions status") {
		t.Fatalf("human stats hide delivery/evidence distinctions: %s", text)
	}
}

func TestWorkQueueDiagnosticStatsEligibilityAgesAndEmptyQueue(t *testing.T) {
	state, _, ids := workOperatorFixture(t, false)
	work := state.Works[ids[0]]
	work.RetryNotBefore = 10000
	work.Enqueued = 0
	report, err := workQueueStatsAt(state, "alternate", 5000)
	if err != nil {
		t.Fatal(err)
	}
	pool := report.Pools[0]
	if report.Branch != "alternate" || report.Tip != state.Tip || report.PolicyMode != state.Policy.Mode ||
		pool.SchedulingEligible != 2 || pool.IneligibleReasons["retry_delayed"] != 1 ||
		pool.OldestAvailable == nil || pool.OldestAvailable.AdmittedAt != 2000 || pool.OldestAvailable.AgeMS != 3000 ||
		len(pool.Prediction.Assignments) != 1 {
		t.Fatalf("readiness or admission time diagnostics are incorrect: %+v", report)
	}
	assertWorkDiagnosticsPrivate(t, report)
	state.GrantsPaused = true
	report, err = workQueueStatsAt(state, "work-queue", 5000)
	if err != nil || report.Pools[0].SchedulingEligible != 0 || report.Pools[0].Prediction.Reason != "grants_paused" {
		t.Fatal("paused grants still predict a Claim")
	}
	state.GrantsPaused, state.RequestOrder = false, nil
	report, err = workQueueStatsAt(state, "work-queue", 5000)
	if err != nil || report.Pools[0].UnknownAge != 3 || report.Pools[0].OldestAvailable != nil {
		t.Fatal("missing trusted admission time was replaced with client enqueue time")
	}
	empty, err := workQueueStatsAt(workqueue.Projection{}, "work-queue", 5000)
	if err != nil || empty.Pools == nil || empty.ClaimsByState["open"] != 0 || empty.Activity.LastAt != nil {
		t.Fatal("empty queue diagnostics were unavailable or fabricated activity")
	}
}

func TestWorkQueueInspectionHandlesMissingAndUninstalledPolicy(t *testing.T) {
	state := workqueue.Projection{
		Works: map[string]*workqueue.WorkState{
			"w1": {
				WorkDefinition: workqueue.WorkDefinition{WorkID: "w1", Pool: "retired", NodeKey: "node"},
				State:          "available",
			},
		},
	}
	inspection, err := workQueueInspectAt(state, "work-queue",
		workQueueRow{Kind: "work", ID: "w1", WorkID: "w1"}, 5000)
	if err != nil || inspection.Scheduling.Reason != "pool_not_installed" || inspection.Work.MaxAttempts != nil {
		t.Fatalf("inspection did not tolerate missing Policy: %+v %v", inspection, err)
	}
	report, err := workQueueStatsAt(state, "work-queue", 5000)
	if err != nil || report.Pools[0].IneligibleReasons["pool_not_installed"] != 1 {
		t.Fatalf("stats did not classify a Work without Policy: %+v %v", report, err)
	}
	_, err = workQueueInspectAt(state, "work-queue",
		workQueueRow{Kind: "work", ID: "missing", WorkID: "missing"}, 5000)
	if err == nil || !strings.Contains(err.Error(), "work_missing") {
		t.Fatalf("missing Work did not return work_missing: %v", err)
	}
}

func TestWorkQueueDiagnosticStatsHistoricalRetainedDelivery(t *testing.T) {
	state, _, ids := workDiagnosticsFixture(t)
	claim := state.Claims[state.Works[ids[0]].ClaimID]
	claim.KeepOpen, claim.Barrier = true, "verified"
	work := state.Works[ids[0]]
	work.State, work.ClaimID, work.Barrier = "available", "", "none"
	work.CompletionID, work.ResultCommitID, work.Result = "", "", nil
	report, err := workQueueStatsAt(state, "work-queue", 500000)
	if err != nil || report.DeliveryByState["verified"] != 1 {
		t.Fatal("historical keep-open delivery disappeared when Work became available")
	}
	claim.Barrier, work.Barrier, work.ClaimID = "pending", "pending", claim.ClaimID
	report, err = workQueueStatsAt(state, "work-queue", 500000)
	if err != nil || report.DeliveryByState["pending"] != 1 || report.Pools[0].IneligibleReasons["delivery_pending"] != 1 {
		t.Fatal("retained pending delivery was treated as eligible Work")
	}
	failed := state.Works[ids[1]]
	state.TerminalBarriers[failed.ClaimID] = state.TerminalBarriers[failed.WorkID]
	report, err = workQueueStatsAt(state, "work-queue", 500000)
	if err != nil || report.DeliveryFailureReasons["verification_exhausted"] != 1 {
		t.Fatal("Work/Claim barrier indexes double-counted one failed delivery")
	}
}

func TestWorkQueueDiagnosticActivityWindow(t *testing.T) {
	at := int64(48 * time.Hour / time.Millisecond)
	since := at - int64(24*time.Hour/time.Millisecond)
	state := workqueue.Projection{Requests: map[string]workqueue.QueueCommit{}}
	for id, timestamp := range map[string]int64{"before": since - 1, "start": since, "end": at, "future": at + 1} {
		state.Requests[id] = workqueue.QueueCommit{ID: id, Request: workqueue.Request{ID: id}, At: timestamp, Operations: []workqueue.Operation{
			mustWorkQueueOperation(t, map[string]any{"kind": "Checkpoint"}),
		}}
	}
	state.RequestOrder = []string{"before", "start", "end", "future"}
	activity, err := workQueueActivityAt(state, at)
	if err != nil || activity.Since != since || activity.Until != at || activity.Transactions != 2 ||
		activity.Operations["Checkpoint"] != 2 || activity.LastCommitID != "end" {
		t.Fatalf("activity window includes future/old transactions: %+v %v", activity, err)
	}
	state.RequestOrder = []string{"older", "newer"}
	state.Requests = map[string]workqueue.QueueCommit{
		"older": {ID: "z-older", Request: workqueue.Request{ID: "older"}, At: at, Operations: []workqueue.Operation{
			mustWorkQueueOperation(t, map[string]any{"kind": "Checkpoint"}),
		}},
		"newer": {ID: "a-newer", Request: workqueue.Request{ID: "newer"}, At: at, Operations: []workqueue.Operation{
			mustWorkQueueOperation(t, map[string]any{"kind": "Checkpoint"}),
		}},
	}
	activity, err = workQueueActivityAt(state, at)
	if err != nil || activity.LastCommitID != "a-newer" {
		t.Fatalf("equal-timestamp commits ignored ledger order: %+v %v", activity, err)
	}
	state.Requests["invalid"] = workqueue.QueueCommit{At: at, Operations: []workqueue.Operation{json.RawMessage(`invalid`)}}
	if _, err := workQueueActivityAt(state, at); err == nil {
		t.Fatal("malformed activity was silently ignored")
	}
}

func TestWorkQueueDiagnosticCheckpointProvenance(t *testing.T) {
	state, commits, ids := workDiagnosticsFixture(t)
	checkpoint, err := workqueue.CompactCheckpoint(commits, strings.Repeat("a", 40),
		workqueue.Actor{Role: "administrator", Principal: "1001", Repository: "owner/repo"}, 500000)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := workqueue.Replay(checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	before, err := workQueueStatsAt(state, "work-queue", 600000)
	if err != nil {
		t.Fatal(err)
	}
	after, err := workQueueStatsAt(restored, "work-queue", 600000)
	if err != nil || !reflect.DeepEqual(before.ClaimsByState, after.ClaimsByState) ||
		!reflect.DeepEqual(before.DeliveryByState, after.DeliveryByState) ||
		!reflect.DeepEqual(before.DeliveryFailureReasons, after.DeliveryFailureReasons) ||
		after.Activity.Operations["Completion"] != before.Activity.Operations["Completion"] ||
		after.Activity.Operations["Result"] != before.Activity.Operations["Result"] {
		t.Fatalf("compaction lost outcome/activity provenance: %v", err)
	}
	inspection, err := workQueueInspectAt(restored, "work-queue",
		workQueueRow{Kind: "work", ID: ids[1], WorkID: ids[1]}, 600000)
	if err != nil || inspection.Work.AdmittedAt == nil || *inspection.Work.AdmittedAt != 2000 ||
		inspection.Claim.GrantedAt == nil || *inspection.Claim.GrantedAt != 3000 ||
		inspection.Claim.DeliveryFailureReason != "verification_exhausted" {
		t.Fatalf("compaction lost structured inspection provenance: %v", err)
	}
	assertWorkDiagnosticsPrivate(t, inspection)
}

func TestWorkQueueDiagnosticInspectionAndRows(t *testing.T) {
	state, _, ids := workDiagnosticsFixture(t)
	work := state.Works[ids[1]]
	result, err := workQueueInspectAt(state, "alternate", workQueueRow{Kind: "work", ID: work.WorkID, WorkID: work.WorkID}, 500000)
	if err != nil {
		t.Fatal(err)
	}
	if result.Work.NodeKey != "analyze" || result.Work.Delivery != "failed" || result.Work.MaxAttempts == nil || *result.Work.MaxAttempts != 3 ||
		result.Work.AdmittedAt == nil || *result.Work.AdmittedAt != 2000 || result.Scheduling.Reason != "ownership_completed" ||
		result.Claim.DeliveryFailureReason != "verification_exhausted" || result.Claim.Disposition != "unknown" ||
		result.Claim.GrantedAt == nil || *result.Claim.GrantedAt != 3000 ||
		result.Claim.TerminalAt == nil || *result.Claim.TerminalAt != 6000 {
		t.Fatalf("structured inspection lost delivery/attempt provenance: %+v", result)
	}
	if result.Dispatch.Run == nil || result.Dispatch.Run.RunID != "202" || !result.Dispatch.ReservationRetained ||
		len(result.Dispatch.Claims) != 3 || result.Work.DependsOn == nil || result.Branch != "alternate" {
		t.Fatalf("structured inspection lost original assignment/run binding: %+v", result.Dispatch)
	}
	assertWorkDiagnosticsPrivate(t, result)
	work.ClaimID = ""
	historical, err := workQueueInspectAt(state, "work-queue", workQueueRow{
		Kind: "claim", ID: result.Claim.ClaimID, WorkID: work.WorkID,
	}, 500000)
	if err != nil || historical.Claim.Current || historical.Claim.DeliveryFailureReason != "verification_exhausted" {
		t.Fatal("historical Claim inspection used only current Work ownership")
	}
	rows := workQueueRows(state, workQueueFilter{}, nil)
	for _, row := range rows {
		switch row.Kind {
		case "work":
			if row.Pool != "default" || row.WorkerProfile != "default" || row.NodeKey == "" || row.Attempts == nil ||
				*row.Attempts != 1 || row.AdmittedAt == nil || row.RetryNotBefore == nil {
				t.Fatalf("Work row requires parsing display text: %+v", row)
			}
		case "claim":
			if row.Run == nil || row.Run.RunID != "202" || row.ReservationRetained == nil ||
				!*row.ReservationRetained || row.GrantedAt == nil || *row.GrantedAt != 3000 || row.Pool != "default" {
				t.Fatalf("Claim row lacks worker metadata: %+v", row)
			}
		}
	}
	assertWorkDiagnosticsPrivate(t, rows)
}

func TestWorkQueueDiagnosticRetiredPoolHistory(t *testing.T) {
	_, commits, ids := workOperatorFixture(t, false)
	actor := workqueue.Actor{Role: "administrator", Principal: "1001", Repository: "owner/repo"}
	var cancellations []workqueue.Operation
	for _, id := range ids {
		cancellations = append(cancellations, mustWorkQueueOperation(t, map[string]any{
			"kind": "WorkCancellation", "work_id": id, "reason": "operator_cancelled",
		}))
	}
	request, err := workqueue.NewRequest("cancel-all", "cancel_work", actor, workqueue.OperationsParameters{Operations: cancellations})
	if err != nil {
		t.Fatal(err)
	}
	commits, _, _, err = workqueue.BuildCandidate(commits, actor, request, 3000)
	if err != nil {
		t.Fatal(err)
	}
	policy := workqueue.DefaultPolicy(actor.Principal, actor.Repository)
	policy.Pools["replacement"] = policy.Pools["default"]
	delete(policy.Pools, "default")
	rule := policy.Producers[actor.Principal]
	rule.Pools = []string{"replacement"}
	policy.Producers[actor.Principal] = rule
	request, err = workqueue.NewRequest("retire-pool", "policy", actor, workqueue.OperationsParameters{Operations: []workqueue.Operation{
		mustWorkQueueOperation(t, map[string]any{"kind": "Policy", "epoch": "replacement", "policy": policy}),
	}})
	if err != nil {
		t.Fatal(err)
	}
	commits, _, _, err = workqueue.BuildCandidate(commits, actor, request, 4000)
	if err != nil {
		t.Fatal(err)
	}
	state, err := workqueue.Replay(commits)
	if err != nil {
		t.Fatal(err)
	}
	state.Works[ids[0]].State = "available"
	report, err := workQueueStatsAt(state, "work-queue", 5000)
	if err != nil || len(report.Pools) != 2 || report.Pools[0].Pool != "default" || report.Pools[0].PolicyInstalled ||
		report.Pools[0].WorkByState["available"] != 1 || report.Pools[0].WorkByState["cancelled"] != 2 ||
		report.Pools[0].SchedulingEligible != 0 || report.Pools[0].IneligibleReasons["pool_not_installed"] != 1 ||
		report.Pools[0].Prediction.Reason != "pool_not_installed" ||
		!report.Pools[1].PolicyInstalled {
		t.Fatalf("retired pool Work was lost or treated as schedulable: %+v %v", report, err)
	}
	inspection, err := workQueueInspectAt(state, "work-queue", workQueueRow{Kind: "work", ID: ids[0], WorkID: ids[0]}, 5000)
	if err != nil || inspection.Work.MaxAttempts != nil || inspection.Scheduling.Reason != "pool_not_installed" {
		t.Fatalf("retired Work inspection invented a current retry policy: %+v %v", inspection, err)
	}
	assertWorkDiagnosticsPrivate(t, report)
	assertWorkDiagnosticsPrivate(t, inspection)
}

func TestWorkQueueDiagnosticRetainedDispatchFromHistoricalPool(t *testing.T) {
	state, _, _ := workOperatorFixture(t, false)
	state.Works = map[string]*workqueue.WorkState{}
	state.Claims = map[string]*workqueue.ClaimState{}
	state.Dispatches = map[string]*workqueue.DispatchState{
		"historical": {
			Assignment: workqueue.Assignment{DispatchID: "historical", Pool: "retired"},
			State:      "bound",
		},
	}
	report, err := workQueueStatsAt(state, "work-queue", 5000)
	if err != nil {
		t.Fatal(err)
	}
	for _, pool := range report.Pools {
		if pool.Pool == "retired" {
			if pool.PolicyInstalled || pool.NativeReservations != 1 {
				t.Fatalf("retired dispatch history was not retained: %+v", pool)
			}
			return
		}
	}
	t.Fatal("historical dispatch pool was omitted from stats")
}

func TestWorkQueueDiagnosticCommandsReadOnlyAndLegacyFields(t *testing.T) {
	t.Setenv("GH_HOST", "github.com")
	t.Setenv("GH_TOKEN", "test-token")
	state, commits, ids := workDiagnosticsFixture(t)
	data, err := workqueue.Serialize(commits)
	if err != nil {
		t.Fatal(err)
	}
	queue := newWorkQueueGitRemote(t, string(data))
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	http.DefaultTransport = workAPIRoundTripper(func(r *http.Request) (*http.Response, error) {
		t.Fatalf("diagnostic command unexpectedly contacted Actions/API: %s %s", r.Method, r.URL.Path)
		return nil, nil
	})
	for _, args := range [][]string{{"stats"}, {"state", "--limit", "2"}, {"inspect", "--work-id", ids[1]}} {
		command := NewWorkCommand()
		var output bytes.Buffer
		command.SetOut(&output)
		command.SetErr(io.Discard)
		command.SetArgs(append([]string{"--repo", "owner/repo", "--json"}, args...))
		if err := command.Execute(); err != nil {
			t.Fatalf("%v failed: %v (%s)", args, err, output.String())
		}
		var result map[string]json.RawMessage
		if err := json.Unmarshal(output.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		for _, required := range []string{"status", "tip", "repository", "branch", "evaluated_at", "policy_epoch"} {
			if _, ok := result[required]; !ok {
				t.Fatalf("%v omitted snapshot field %s", args, required)
			}
		}
		switch args[0] {
		case "stats":
			var legacy workqueue.Stats
			if err := json.Unmarshal(output.Bytes(), &legacy); err != nil || !reflect.DeepEqual(legacy, state.Stats) {
				t.Fatal("additive stats broke existing root counters")
			}
		case "inspect":
			for _, required := range []string{"selection", "details", "work", "claim", "dispatch", "scheduling"} {
				if _, ok := result[required]; !ok {
					t.Fatalf("inspection omitted %s", required)
				}
			}
		case "state":
			var page workQueuePage
			if err := json.Unmarshal(output.Bytes(), &page); err != nil || page.NextOffset == nil || *page.NextOffset != 2 {
				t.Fatal("structured row metadata broke pagination")
			}
		}
		assertWorkDiagnosticsPrivate(t, result)
	}
	if queue.pushes(t) != 0 || queue.read(t) != string(data) {
		t.Fatal("read-only diagnostics changed queue state")
	}
}
