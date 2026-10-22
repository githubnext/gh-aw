package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workqueue"
)

func workClearOperatorEnvironment(t *testing.T, admin bool) {
	t.Helper()
	role := "write"
	if admin {
		role = "admin"
	}
	workClearRoleEnvironment(t, role)
}

func workClearRoleEnvironment(t *testing.T, role string) func(string) {
	t.Helper()
	for _, name := range []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "COPILOT_AGENT_SESSION_ID"} {
		t.Setenv(name, "")
	}
	t.Setenv("GH_HOST", "github.com")
	t.Setenv("GH_TOKEN", "test-token")
	originalConfirm := workClearConfirmAction
	t.Cleanup(func() { workClearConfirmAction = originalConfirm })
	workClearConfirmAction = func(string, string, string) (bool, error) { return true, nil }
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	http.DefaultTransport = workAPIRoundTripper(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			t.Fatalf("clear attempted a native action or API ledger write: %s %s", r.Method, r.URL.Path)
		}
		var result any
		switch r.URL.Path {
		case "/user":
			result = map[string]any{"id": 1001}
		case "/repos/owner/repo":
			result = map[string]any{"full_name": "owner/repo", "default_branch": "main", "permissions": map[string]bool{
				"push": role == "write" || role == "maintain" || role == "admin", "admin": role == "admin", "maintain": role == "maintain",
			}}
		case "/repos/owner/repo/git/ref/heads/main":
			result = map[string]any{"ref": "refs/heads/main", "object": map[string]string{"type": "commit", "sha": strings.Repeat("a", 40)}}
		case "/repos/owner/repo/contents/.github/workflows/aw.json":
			return &http.Response{StatusCode: http.StatusNotFound, Header: http.Header{"Content-Type": {"application/json"}},
				Body: io.NopCloser(strings.NewReader(`{"message":"not found"}`)), Request: r}, nil
		default:
			t.Fatalf("clear attempted unexpected API read: %s", r.URL.Path)
		}
		data, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(bytes.NewReader(data)), Request: r}, nil
	})
	return func(next string) { role = next }
}

func workClearExecute(t *testing.T, extra ...string) (workClearResult, error) {
	t.Helper()
	cmd := NewWorkCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(append([]string{"--repo", "owner/repo", "--json", "clear", "--reason", "operator_clear", "--request-id", "clear-action"}, extra...))
	err := cmd.Execute()
	var result workClearResult
	if err == nil {
		if err := json.Unmarshal(output.Bytes(), &result); err != nil {
			t.Fatalf("invalid clear output: %v (%s)", err, output.String())
		}
		if strings.Contains(output.String(), "PAYLOAD-MUST") || strings.Contains(output.String(), "SECRET-MUST") {
			t.Fatal("clear output leaked Work payload")
		}
	}
	return result, err
}

func workClearAppend(t *testing.T, commits []workqueue.QueueCommit, actor workqueue.Actor, id, kind string, params any, at int64) []workqueue.QueueCommit {
	t.Helper()
	request, err := workqueue.NewRequest(id, kind, actor, params)
	if err != nil {
		t.Fatal(err)
	}
	commits, _, _, err = workqueue.BuildCandidate(commits, actor, request, at)
	if err != nil {
		t.Fatal(err)
	}
	return commits
}

func workClearMixedFixture(t *testing.T, failed bool) (workqueue.Projection, []workqueue.QueueCommit, string) {
	t.Helper()
	return workClearMixedFixtureWithKeepOpen(t, failed, false)
}

func workClearMixedFixtureWithKeepOpen(t *testing.T, failed, keepOpen bool) (workqueue.Projection, []workqueue.QueueCommit, string) {
	t.Helper()
	state, commits, _ := workOperatorFixture(t, true)
	var dispatch *workqueue.DispatchState
	for _, item := range state.Dispatches {
		dispatch = item
	}
	actor := workqueue.Actor{Role: "administrator", Principal: "1001", Repository: "owner/repo"}
	sender := actor
	sender.Role, sender.Workflow, sender.RunID, sender.RunAttempt = "dispatcher", ".github/workflows/dispatcher.lock.yml", "101", 1
	commits = workClearAppend(t, commits, sender, "start", "dispatch", workqueue.OperationsParameters{Operations: []workqueue.Operation{
		mustWorkQueueOperation(t, map[string]any{"kind": "Dispatch", "dispatch_id": dispatch.DispatchID, "state": "started", "sender": sender}),
	}}, 4000)
	profile := dispatch.Profile
	reconciler := actor
	reconciler.Role = "reconciler"
	evidence := workqueue.Evidence{Kind: "reconciliation", Source: "github_api", Repository: actor.Repository,
		Workflow: profile.Workflow, Ref: profile.Ref, Principal: actor.Principal, CheckedAt: 5000, RunID: "202", RunAttempt: 1}
	commits = workClearAppend(t, commits, reconciler, "bind", "dispatch", workqueue.OperationsParameters{Operations: []workqueue.Operation{
		mustWorkQueueOperation(t, map[string]any{"kind": "Dispatch", "dispatch_id": dispatch.DispatchID, "state": "bound",
			"run": workqueue.RunBinding{RunID: "202", RunAttempt: 1, Repository: actor.Repository, Workflow: profile.Workflow,
				Ref: profile.Ref, Principal: actor.Principal, Event: "workflow_dispatch"}, "evidence": evidence}),
	}}, 5000)
	worker := actor
	worker.Role, worker.Workflow, worker.RunID, worker.RunAttempt = "worker", profile.Workflow, "202", 1
	worker.DispatchID, worker.ClaimHandle = dispatch.DispatchID, "h1"
	commits = workClearAppend(t, commits, worker, "finish", "finish", workqueue.FinishParameters{
		DispatchID: dispatch.DispatchID, ClaimHandle: "h1", Outcome: "completed", KeepOpen: keepOpen,
	}, 6000)
	completedID := dispatch.Claims[0].WorkID
	state, err := workqueue.Replay(commits)
	if err != nil {
		t.Fatal(err)
	}
	if failed {
		evidence.Kind, evidence.Status, evidence.Conclusion = "terminal_run", "completed", "failure"
		evidence.CheckedAt, evidence.Attempts, evidence.Effects = 7000, state.Policy.Pools["default"].Reconciliation.MaxAttempts, "unknown"
		commits = workClearAppend(t, commits, reconciler, "failed", "delivery_failure", workqueue.OperationsParameters{Operations: []workqueue.Operation{
			mustWorkQueueOperation(t, map[string]any{"kind": "DeliveryFailure", "work_id": completedID,
				"claim_id": state.Works[completedID].ClaimID, "completion_id": state.Works[completedID].CompletionID,
				"reason": "verification_failed", "disposition": "unknown", "evidence": evidence}),
		}}, 7000)
	}
	available, err := workqueue.NewWork([]byte(`{"task":"available"}`), "later", "root", "default", *state.Policy, 7000)
	if err != nil {
		t.Fatal(err)
	}
	commits = workClearAppend(t, commits, actor, "later-work", "submit", workqueue.SubmitParameters{Nodes: []workqueue.WorkDefinition{available}}, 8000)
	state, err = workqueue.Replay(commits)
	if err != nil {
		t.Fatal(err)
	}
	return state, commits, completedID
}

func TestWorkClearPreservesHistoryAndUnresolvedEffects(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(strconv.FormatBool(failed), func(t *testing.T) {
			workClearOperatorEnvironment(t, true)
			before, commits, completedID := workClearMixedFixture(t, failed)
			data, err := workqueue.Serialize(commits)
			if err != nil {
				t.Fatal(err)
			}
			queue := newWorkQueueGitRemote(t, string(data))
			preview, err := workClearExecute(t, "--dry-run")
			if err != nil || preview.Status != "dry_run" || len(preview.WorkIDs) != 3 ||
				preview.CompletedRetained != 1 || preview.ReservationsRetained != 1 || queue.pushes(t) != 0 || queue.read(t) != string(data) {
				t.Fatalf("unsafe or incomplete preview: %+v %v", preview, err)
			}
			result, err := workClearExecute(t)
			if err != nil || result.Status != "cancelled" || result.Publication == nil || !result.Publication.Changed ||
				!slices.Equal(result.WorkIDs, preview.WorkIDs) || queue.pushes(t) != 1 {
				t.Fatalf("clear failed: %+v %v", result, err)
			}
			after := workClearRead(t, queue)
			if after.Stats.Cancelled != 3 || after.Stats.Completed != 1 || after.Stats.Available != 0 || after.Stats.Claimed != 0 {
				t.Fatalf("unexpected cleared state: %+v", after.Stats)
			}
			assertWorkClearEqual(t, before.Works[completedID], after.Works[completedID])
			assertWorkClearEqual(t, before.Dispatches, after.Dispatches)
			assertWorkClearEqual(t, before.Clocks, after.Clocks)
			for id, work := range after.Works {
				assertWorkClearEqual(t, before.Works[id].WorkDefinition, work.WorkDefinition)
			}
			if !strings.HasPrefix(queue.read(t), string(data)) {
				t.Fatal("clear deleted historical transactions")
			}
			retry, err := workClearExecute(t)
			if err != nil || retry.Status != "already_committed" || queue.pushes(t) != 1 {
				t.Fatalf("clear retry did not recover acknowledgment: %+v %v", retry, err)
			}
		})
	}
}

func TestWorkClearPreservesPendingKeepOpenDelivery(t *testing.T) {
	for _, pendingOnly := range []bool{false, true} {
		t.Run(strconv.FormatBool(pendingOnly), func(t *testing.T) {
			workClearOperatorEnvironment(t, true)
			before, commits, retainedID := workClearMixedFixtureWithKeepOpen(t, false, true)
			if before.Works[retainedID].State != "available" || before.Works[retainedID].Barrier != "pending" {
				t.Fatal("fixture did not retain a pending keep-open completion")
			}
			for _, reservations := range []int{0, 2} {
				reportState := before
				reportState.Dispatches = map[string]*workqueue.DispatchState{}
				for index := range reservations {
					reportState.Dispatches[strconv.Itoa(index)] = &workqueue.DispatchState{}
				}
				report, err := workClearReport(reportState, workqueue.OperationsParameters{}, "report-only")
				if err != nil || report.PendingDeliveriesRetained != 1 || report.ReservationsRetained != reservations {
					t.Fatalf("retained delivery count depended on reservations: %+v %v", report, err)
				}
			}
			if pendingOnly {
				params, err := workClearIntent(before, "prior_clear", "prior-clear")
				if err != nil {
					t.Fatal(err)
				}
				commits = workClearAppend(t, commits, workqueue.Actor{Role: "administrator", Principal: "1001", Repository: "owner/repo"},
					"prior-clear", "cancel_work", params, 9000)
				before, err = workqueue.Replay(commits)
				if err != nil {
					t.Fatal(err)
				}
			}
			data, err := workqueue.Serialize(commits)
			if err != nil {
				t.Fatal(err)
			}
			queue := newWorkQueueGitRemote(t, string(data))
			preview, err := workClearExecute(t, "--dry-run")
			wantStatus, wantSelected := "dry_run", 3
			if pendingOnly {
				wantStatus, wantSelected = "no_work", 0
			}
			if err != nil || preview.Status != wantStatus || len(preview.WorkIDs) != wantSelected ||
				preview.PendingDeliveriesRetained != 1 || preview.CompletedRetained != 0 ||
				preview.ReservationsRetained != 1 || slices.Contains(preview.WorkIDs, retainedID) ||
				queue.pushes(t) != 0 || queue.read(t) != string(data) {
				t.Fatalf("pending keep-open delivery broke preview: %+v %v", preview, err)
			}
			result, err := workClearExecute(t)
			wantStatus, wantPushes := "cancelled", 1
			if pendingOnly {
				wantStatus, wantPushes = "no_work", 0
			}
			if err != nil || result.Status != wantStatus || result.PendingDeliveriesRetained != 1 ||
				!slices.Equal(result.WorkIDs, preview.WorkIDs) || queue.pushes(t) != wantPushes {
				t.Fatalf("pending keep-open delivery broke clear: %+v %v", result, err)
			}
			after := workClearRead(t, queue)
			if after.Stats.Available != 1 || after.Stats.Claimed != 0 || after.Stats.Cancelled != 3 ||
				after.Stats.Completed != 0 {
				t.Fatalf("clear did not preserve only the retained delivery: %+v", after.Stats)
			}
			assertWorkClearEqual(t, before.Works[retainedID], after.Works[retainedID])
			assertWorkClearEqual(t, before.Claims[before.Works[retainedID].ClaimID], after.Claims[before.Works[retainedID].ClaimID])
			assertWorkClearEqual(t, before.Dispatches, after.Dispatches)
			assertWorkClearEqual(t, before.Clocks, after.Clocks)
			if pendingOnly {
				assertWorkClearEqual(t, before, after)
				if _, ok := after.Requests["clear-action"]; ok || result.Publication != nil {
					t.Fatal("pending-only no-op recorded a clear request")
				}
			}
		})
	}
}

func assertWorkClearEqual(t *testing.T, before, after any) {
	t.Helper()
	left, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}
	right, err := json.Marshal(after)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(left, right) {
		t.Fatalf("clear rewrote retained state:\nbefore %s\nafter %s", left, right)
	}
}

func workClearRead(t *testing.T, queue *workQueueGitRemote) workqueue.Projection {
	t.Helper()
	commits, err := workqueue.Parse([]byte(queue.read(t)))
	if err != nil {
		t.Fatal(err)
	}
	state, err := workqueue.Replay(commits)
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func TestWorkClearRejectsCIWithoutIO(t *testing.T) {
	for _, name := range []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "COPILOT_AGENT_SESSION_ID"} {
		for _, extra := range [][]string{nil, {"--dry-run"}, {"--hard"}, {"--hard", "--dry-run"}} {
			t.Run(name+strings.Join(extra, ""), func(t *testing.T) {
				workClearOperatorEnvironment(t, true)
				t.Setenv(name, "true")
				http.DefaultTransport = workAPIRoundTripper(func(*http.Request) (*http.Response, error) {
					t.Fatal("CI refusal performed network I/O")
					return nil, nil
				})
				if _, err := workClearExecute(t, extra...); err == nil || !strings.Contains(err.Error(), "clear_ci_forbidden") {
					t.Fatalf("CI clear was not refused: %v", err)
				}
			})
		}
	}
}

func TestWorkClearNoopAndRefusals(t *testing.T) {
	for _, scenario := range []string{"empty", "missing", "malformed", "not administrator", "reason", "limit"} {
		t.Run(scenario, func(t *testing.T) {
			workClearOperatorEnvironment(t, scenario != "not administrator")
			state, commits, _ := workOperatorFixture(t, false)
			if scenario == "empty" {
				commits = commits[:1]
			}
			if scenario == "limit" {
				state.Policy.Limits.Operations = 2
				if _, err := workClearIntent(state, "operator_clear", "clear-action"); err == nil || !strings.Contains(err.Error(), "clear_limit") {
					t.Fatalf("oversized atomic clear accepted: %v", err)
				}
				return
			}
			data, err := workqueue.Serialize(commits)
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "missing":
				data = nil
			case "malformed":
				data = []byte("not a valid ledger\n")
			}
			queue := newWorkQueueGitRemote(t, string(data))
			extra := []string{}
			if scenario == "reason" {
				extra = []string{"--reason", "unsafe reason\n"}
			}
			result, err := workClearExecute(t, extra...)
			if scenario == "empty" {
				if err != nil || result.Status != "no_work" || result.Publication != nil || len(result.WorkIDs) != 0 {
					t.Fatalf("empty clear did not no-op: %+v %v", result, err)
				}
			} else if err == nil {
				t.Fatal("unsafe clear accepted")
			}
			if queue.pushes(t) != 0 || queue.read(t) != string(data) {
				t.Fatal("refused/no-op clear modified the queue")
			}
		})
	}
}

func TestWorkClearClaimRecoveryAfterCheckpoint(t *testing.T) {
	state, commits, _ := workOperatorFixture(t, true)
	params, err := workClearIntent(state, "operator_clear", "clear-action")
	if err != nil {
		t.Fatal(err)
	}
	actor := workqueue.Actor{Role: "administrator", Principal: "1001", Repository: "owner/repo"}
	commits = workClearAppend(t, commits, actor, "clear-action", "cancel_work", params, 4000)
	checkpoint, err := workqueue.CompactCheckpoint(commits, strings.Repeat("a", 40), actor, 5000)
	if err != nil {
		t.Fatal(err)
	}
	state, err = workqueue.Replay(checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := workClearIntent(state, "operator_clear", "clear-action")
	if err != nil {
		t.Fatal(err)
	}
	assertWorkClearEqual(t, params, recovered)
}

func TestWorkClearAtomicOperationBoundary(t *testing.T) {
	state, _, _ := workOperatorFixture(t, false)
	state.Works = map[string]*workqueue.WorkState{}
	for index := range 257 {
		id := fmt.Sprintf("work_%03d", index)
		state.Works[id] = &workqueue.WorkState{WorkDefinition: workqueue.WorkDefinition{WorkID: id}, State: "available"}
		if index < 255 {
			continue
		}
		params, err := workClearIntent(state, "operator_clear", "boundary")
		if index == 255 && (err != nil || len(params.Operations) != 256) {
			t.Fatalf("exact 256-Work bound rejected: %v", err)
		}
		if index == 256 && (err == nil || !strings.Contains(err.Error(), "clear_limit")) {
			t.Fatalf("257-Work snapshot was partially accepted: %v", err)
		}
	}
}

func TestWorkClearConcurrentCompletionFailsAtomically(t *testing.T) {
	state, commits, _ := workClearMixedFixture(t, false)
	params, err := workClearIntent(state, "operator_clear", "stale-clear")
	if err != nil {
		t.Fatal(err)
	}
	var selected *workqueue.ClaimState
	for _, claim := range state.Claims {
		if claim.State == "open" {
			selected = claim
			break
		}
	}
	dispatch := state.Dispatches[selected.DispatchID]
	worker := workqueue.Actor{Role: "worker", Principal: "1001", Repository: "owner/repo",
		Workflow: dispatch.Profile.Workflow, RunID: "202", RunAttempt: 1,
		DispatchID: dispatch.DispatchID, ClaimHandle: selected.Handle}
	changed := workClearAppend(t, commits, worker, "concurrent-finish", "finish",
		workqueue.FinishParameters{DispatchID: dispatch.DispatchID, ClaimHandle: selected.Handle, Outcome: "completed"}, 9000)
	actor := workqueue.Actor{Role: "administrator", Principal: "1001", Repository: "owner/repo"}
	request, err := workqueue.NewRequest("stale-clear", "cancel_work", actor, params)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := workqueue.BuildCandidate(changed, actor, request, 10000); err == nil {
		t.Fatal("stale clear cancelled a completed owner")
	}
	latest, err := workqueue.Replay(changed)
	if err != nil || latest.Stats.Cancelled != 0 {
		t.Fatal("failed atomic clear cancelled a sibling")
	}
}

func TestWorkClearRejectsResetReadmissionBeforePublication(t *testing.T) {
	for _, compact := range []bool{false, true} {
		t.Run(strconv.FormatBool(compact), func(t *testing.T) {
			workClearOperatorEnvironment(t, true)
			before, commits, ids := workOperatorFixture(t, false)
			data, err := workqueue.Serialize(commits)
			if err != nil {
				t.Fatal(err)
			}
			queue := newWorkQueueGitRemote(t, string(data))
			transport := http.DefaultTransport
			authentications, resetPushes := 0, 0
			var replacement string
			http.DefaultTransport = workAPIRoundTripper(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/user" {
					authentications++
					if authentications == 2 {
						// Publish reauthenticates after clear has captured and validated its selection.
						actor := workqueue.Actor{Role: "administrator", Principal: "1001", Repository: "owner/repo"}
						paused := workClearAppend(t, commits, actor, "pause-before-reset", "control",
							workqueue.OperationsParameters{Operations: []workqueue.Operation{
								mustWorkQueueOperation(t, map[string]any{"kind": "Control", "control": "admission_paused", "value": true, "reason": "reset"}),
								mustWorkQueueOperation(t, map[string]any{"kind": "Control", "control": "grants_paused", "value": true, "reason": "reset"}),
							}}, 3000)
						pausedData, err := workqueue.Serialize(paused)
						if err != nil {
							t.Fatal(err)
						}
						queue.write(t, string(pausedData))
						branch := workqueue.Branch{Remote: "owner/repo", Name: workqueue.DefaultBranch}
						if _, err := branch.HardClear(context.Background(), "hard-reset", "operator_reset", false); err != nil {
							t.Fatal(err)
						}
						reset, err := workqueue.Parse([]byte(queue.read(t)))
						if err != nil {
							t.Fatal(err)
						}
						at := reset[len(reset)-1].At
						reset = workClearAppend(t, reset, actor, "resume-after-reset", "control",
							workqueue.OperationsParameters{Operations: []workqueue.Operation{
								mustWorkQueueOperation(t, map[string]any{"kind": "Control", "control": "admission_paused", "value": false, "reason": "resume"}),
								mustWorkQueueOperation(t, map[string]any{"kind": "Control", "control": "grants_paused", "value": false, "reason": "resume"}),
							}}, at)
						nodes := []workqueue.WorkDefinition{}
						for _, id := range ids {
							nodes = append(nodes, before.Works[id].WorkDefinition)
						}
						reset = workClearAppend(t, reset, actor, "submit", "submit", workqueue.SubmitParameters{Nodes: nodes}, at)
						resetData, err := workqueue.Serialize(reset)
						if err != nil {
							t.Fatal(err)
						}
						queue.write(t, string(resetData))
						if compact {
							if _, err := branch.Compact(context.Background()); err != nil {
								t.Fatal(err)
							}
						}
						replacement, resetPushes = queue.read(t), queue.pushes(t)
					}
				}
				return transport.RoundTrip(r)
			})
			if _, err := workClearExecute(t); err == nil || !strings.HasPrefix(err.Error(), "ledger_nonextending:") {
				t.Fatalf("clear accepted a replacement queue before its initial publication read: %v", err)
			}
			after := workClearRead(t, queue)
			if replacement == "" || queue.read(t) != replacement || queue.pushes(t) != resetPushes ||
				after.Stats.Available != len(ids) || after.Stats.Cancelled != 0 {
				t.Fatalf("clear changed readmitted Work: %+v", after.Stats)
			}
			if _, ok := after.Requests["clear-action"]; ok {
				t.Fatal("clear appended a cancellation to the replacement queue")
			}
			for _, id := range ids {
				if after.Works[id].State != "available" {
					t.Fatalf("readmitted Work %s was cancelled", id)
				}
			}
		})
	}
}

func TestWorkClearRecoveryAfterCheckpointDoesNotSweepNewWork(t *testing.T) {
	state, commits, ids := workOperatorFixture(t, false)
	actor := workqueue.Actor{Role: "administrator", Principal: "1001", Repository: "owner/repo"}
	params, err := workClearIntent(state, "operator_clear", "clear-action")
	if err != nil {
		t.Fatal(err)
	}
	commits = workClearAppend(t, commits, actor, "clear-action", "cancel_work", params, 4000)
	for _, compact := range []bool{false, true} {
		t.Run(strconv.FormatBool(compact), func(t *testing.T) {
			history := commits
			if compact {
				var err error
				history, err = workqueue.CompactCheckpoint(commits, strings.Repeat("a", 40), actor, 5000)
				if err != nil {
					t.Fatal(err)
				}
			}
			node, err := workqueue.NewWork([]byte(`{"task":"new"}`), "new", "root", "default", *state.Policy, 6000)
			if err != nil {
				t.Fatal(err)
			}
			history = workClearAppend(t, history, actor, "new-work", "submit", workqueue.SubmitParameters{Nodes: []workqueue.WorkDefinition{node}}, 6000)
			latest, err := workqueue.Replay(history)
			if err != nil {
				t.Fatal(err)
			}
			recovered, err := workClearIntent(latest, "operator_clear", "clear-action")
			if err != nil {
				t.Fatal(err)
			}
			assertWorkClearEqual(t, params, recovered)
			request, err := workqueue.NewRequest("clear-action", "cancel_work", actor, recovered)
			if err != nil {
				t.Fatal(err)
			}
			next, _, _, err := workqueue.BuildCandidate(history, actor, request, 7000)
			if err != nil || len(next) != len(history) || latest.Works[node.WorkID].State != "available" {
				t.Fatalf("recovery swept newly admitted Work: %v", err)
			}
			for _, id := range ids {
				if latest.Works[id].State != "cancelled" {
					t.Fatal("original cancellation lost")
				}
			}
			if _, err := workClearIntent(latest, "different_reason", "clear-action"); err == nil {
				t.Fatal("recovery silently changed the reason")
			}
			if _, err := workClearIntent(latest, "operator_clear", "submit"); err == nil {
				t.Fatal("clear reused another kind of request")
			}
		})
	}
}
