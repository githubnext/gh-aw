package workqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func hardClearFixture(t *testing.T) []QueueCommit {
	t.Helper()
	commits := testGenesis(t, nil)
	commits = testSubmit(t, commits, "before-hard", testNode(t, commits, "before-hard"))
	return hardClearPause(t, commits)
}

func hardClearPause(t *testing.T, commits []QueueCommit) []QueueCommit {
	t.Helper()
	return testOperations(t, commits, testActor("administrator"), "pause-hard", "control",
		mustOp(t, map[string]any{"kind": "Control", "control": "admission_paused", "value": true, "reason": "operator_pause"}),
		mustOp(t, map[string]any{"kind": "Control", "control": "grants_paused", "value": true, "reason": "operator_pause"}))
}

func TestHardClearRejectsConcurrentResetAndRecoversLostAcknowledgment(t *testing.T) {
	for _, scenario := range []string{"concurrent", "lost_ack", "forbidden"} {
		t.Run(scenario, func(t *testing.T) {
			branch, mock := newQueueAPI(t)
			commits := hardClearFixture(t)
			installMockLog(t, mock, commits)
			if scenario == "concurrent" {
				mock.concurrent = func() {
					next := testOperations(t, commits, testActor("administrator"), "concurrent-resume", "control",
						mustOp(t, map[string]any{"kind": "Control", "control": "grants_paused", "value": false, "reason": "operator_resume"}))
					installMockLog(t, mock, next)
				}
			}
			mock.ambiguous = scenario == "lost_ack"
			mock.noAdmin = scenario == "forbidden"
			result, err := branch.HardClear(context.Background(), "hard-action", "operator_clear", false)
			if scenario == "lost_ack" {
				if err != nil || result.Status != "already_committed" || mock.refWrites != 1 {
					t.Fatalf("lost hard-clear acknowledgment was not recovered: %+v %v writes=%d", result, err, mock.refWrites)
				}
			} else if err == nil {
				t.Fatal("conflicting or unauthorized hard clear accepted")
			}
			latest, err := branch.Read(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			state, err := Replay(latest)
			if err != nil {
				t.Fatal(err)
			}
			if scenario != "lost_ack" && len(state.Works) != 1 {
				t.Fatal("hard clear discarded a newer/unauthorized snapshot")
			}
		})
	}
}

func TestHardClearRetryDoesNotDeleteLaterAdmissions(t *testing.T) {
	commits := hardClearFixture(t)
	state, err := Replay(commits)
	if err != nil {
		t.Fatal(err)
	}
	actor := testActor("administrator")
	reset, _, err := prepareHardClear(state, actor, "hard-action", "operator_clear", strings.Repeat("a", 40))
	if err != nil {
		t.Fatal(err)
	}
	reset = testOperations(t, reset, actor, "resume-admission", "control",
		mustOp(t, map[string]any{"kind": "Control", "control": "admission_paused", "value": false, "reason": "operator_resume"}))
	reset = testSubmit(t, reset, "after-hard", testNode(t, reset, "after-hard"))
	branch, mock := newQueueAPI(t)
	installMockLog(t, mock, reset)
	result, err := branch.HardClear(context.Background(), "hard-action", "operator_clear", false)
	if err != nil || result.Status != "already_committed" || mock.refWrites != 0 {
		t.Fatalf("hard retry did not recover accepted identity: %+v %v", result, err)
	}
	latest, err := branch.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	retained, err := Replay(latest)
	if err != nil || len(retained.Works) != 1 {
		t.Fatal("hard retry discarded later Work")
	}
	if _, err := branch.HardClear(context.Background(), "hard-action", "different", false); err == nil {
		t.Fatal("hard retry silently changed reason")
	}
	if _, err := branch.HardClear(context.Background(), "after-hard", "operator_clear", false); err == nil {
		t.Fatal("hard clear reused a submission request ID")
	}
}

func TestHardClearRecoversOriginalCredentialGenerationAfterCheckpoint(t *testing.T) {
	actor := testActor("administrator")
	commits := testOperations(t, hardClearFixture(t), actor, "rotate-before-clear", "control",
		mustOp(t, map[string]any{"kind": "Control", "control": "credential_generation", "value": "rotated_before_clear", "reason": "operator_rotate"}))
	state, err := Replay(commits)
	if err != nil {
		t.Fatal(err)
	}
	priorGitSHA := strings.Repeat("a", 40)
	reset, _, err := prepareHardClear(state, actor, "hard-action", "operator_clear", priorGitSHA)
	if err != nil {
		t.Fatal(err)
	}
	state, err = Replay(reset)
	if err != nil {
		t.Fatal(err)
	}
	if state.CredentialGeneration != "rotated_before_clear" {
		t.Fatal("hard clear did not retain the current credential generation")
	}
	for index, name := range []string{"admission_paused", "grants_paused", "credential_generation"} {
		id := fmt.Sprintf("hard_clear_%s_control_%d", hashBytes([]byte("hard-action")), index)
		receipt := state.Requests[id]
		if len(receipt.Operations) != 1 {
			t.Fatalf("hard clear changed stable control receipt %s", id)
		}
		var control struct {
			Name string `json:"control"`
		}
		if err := json.Unmarshal(receipt.Operations[0], &control); err != nil {
			t.Fatal(err)
		}
		if control.Name != name {
			t.Fatalf("control receipt %s identifies %s, want %s", id, control.Name, name)
		}
	}
	reset = testOperations(t, reset, actor, "rotate-after-clear", "control",
		mustOp(t, map[string]any{"kind": "Control", "control": "credential_generation", "value": "rotated_after_clear", "reason": "operator_rotate"}))
	for _, compact := range []bool{false, true} {
		t.Run(fmt.Sprintf("checkpoint=%t", compact), func(t *testing.T) {
			current := reset
			if compact {
				current, err = CompactCheckpoint(current, strings.Repeat("b", 40), actor, time.Now().UnixMilli()+1)
				if err != nil {
					t.Fatal(err)
				}
			}
			state, err := Replay(current)
			if err != nil {
				t.Fatal(err)
			}
			result, ok, err := recoverHardClear(state, actor, "hard-action", "operator_clear")
			if err != nil || !ok || result.Status != "already_committed" || result.PriorGitSHA != priorGitSHA || result.Changed {
				t.Fatalf("hard clear did not recover its original credential generation: %+v %t %v", result, ok, err)
			}
			if state.CredentialGeneration != "rotated_after_clear" {
				t.Fatal("hard-clear recovery changed the later credential generation")
			}
			if _, _, err := recoverHardClear(state, actor, "hard-action", "different"); err == nil {
				t.Fatal("hard-clear recovery accepted a different reason")
			}
		})
	}
}

func TestHardClearRetainsCurrentUnavailableWorkerRoute(t *testing.T) {
	commits := testGenesis(t, deploymentPolicy)
	commits = deployTestWorker(t, commits, "new-route", "default", strings.Repeat("b", 40), strings.Repeat("a", 64), false)
	commits = hardClearPause(t, commits)
	state, err := Replay(commits)
	if err != nil {
		t.Fatal(err)
	}
	next, _, err := prepareHardClear(state, testActor("administrator"), "hard-action", "operator_clear", strings.Repeat("a", 40))
	if err != nil {
		t.Fatal(err)
	}
	current, err := Replay(next)
	if err != nil {
		t.Fatal(err)
	}
	deployment := current.Deployments["default"]["default"]
	if deployment.CurrentRef != strings.Repeat("b", 40) || deployment.Revisions[deployment.CurrentRef].Available ||
		len(deployment.Revisions) != 1 {
		t.Fatal("hard clear resurrected an old or unavailable worker route")
	}
	assertOperatorJSProjection(t, next)
}

func TestHardClearRefusesUnresolvedStateEvenAfterReservationRelease(t *testing.T) {
	for _, scenario := range []string{"open_claim", "pending", "failed", "unknown", "retained_failed", "retained_pending"} {
		t.Run(scenario, func(t *testing.T) {
			state, err := Replay(hardClearFixture(t))
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "open_claim":
				state.Claims["open"] = &ClaimState{State: "open"}
			case "retained_failed", "retained_pending":
				state.Claims["retained"] = &ClaimState{State: "completed", KeepOpen: true, Barrier: strings.TrimPrefix(scenario, "retained_")}
			default:
				for _, work := range state.Works {
					work.State = "completed"
					if scenario == "unknown" {
						work.Disposition = "unknown"
					} else {
						work.Barrier = scenario
					}
				}
			}
			if err := validateHardClearState(state); err == nil || !strings.Contains(err.Error(), "hard_clear_unresolved") {
				t.Fatalf("hard clear discarded an unresolved obligation: %v", err)
			}
		})
	}
}
