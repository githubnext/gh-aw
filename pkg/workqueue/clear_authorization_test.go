package workqueue

import (
	"context"
	"strings"
	"testing"
)

func TestClearAuthorizationDoesNotWidenOtherAdministratorOperations(t *testing.T) {
	for _, scenario := range []string{"admin", "maintain", "write", "no_push"} {
		t.Run(scenario, func(t *testing.T) {
			branch, mock := newQueueAPI(t)
			mock.noAdmin = scenario != "admin"
			mock.maintain = scenario == "maintain" || scenario == "no_push"
			mock.noPush = scenario == "no_push"
			actor, err := branch.AuthenticateClear(context.Background())
			allowed := scenario == "admin" || scenario == "maintain"
			if (err == nil) != allowed || mock.ledgerCalls != 0 || mock.refWrites != 0 {
				t.Fatalf("clear authorization performed I/O or accepted the wrong role: %+v %v", actor, err)
			}
			if !allowed {
				if !strings.HasPrefix(err.Error(), "actor_unauthorized:") {
					t.Fatal(err)
				}
				return
			}
			if actor.Role != "administrator" || actor.Principal != testPrincipal || actor.Repository != testRepository {
				t.Fatalf("clear authorization changed audit actor: %+v", actor)
			}
			selected := testGenesis(t, nil)
			node := testNode(t, selected, "selected")
			selected = testSubmit(t, selected, "submit", node)
			installMockLog(t, mock, selected)
			control, err := NewRequest("pause", "control", actor, OperationsParameters{Operations: []Operation{
				mustOp(t, map[string]any{"kind": "Control", "control": "grants_paused", "value": true, "reason": "pause"}),
			}})
			if err != nil {
				t.Fatal(err)
			}
			_, genericErr := branch.PublishFromSnapshot(context.Background(), actor, control, selected)
			if (genericErr == nil) != (scenario == "admin") {
				t.Fatalf("maintainer access widened another administrator operation: %v", genericErr)
			}
			_, genericErr = branch.Publish(context.Background(), actor, snapshotCancellation(t, node))
			if scenario == "maintain" && genericErr == nil {
				t.Fatal("unfenced administrator publication accepted maintainer access")
			}
			if _, err := branch.PublishFromSnapshot(context.Background(), actor, snapshotCancellation(t, node), selected); err != nil {
				t.Fatalf("authorized snapshot clear failed: %v", err)
			}
		})
	}
}

func TestHardClearFromSnapshotChecksConfirmedRevision(t *testing.T) {
	for _, scenario := range []string{"unchanged", "changed", "invalid", "maintain", "write"} {
		t.Run(scenario, func(t *testing.T) {
			branch, mock := newQueueAPI(t)
			mock.noAdmin = scenario == "maintain" || scenario == "write"
			mock.maintain = scenario == "maintain"
			commits := hardClearFixture(t)
			installMockLog(t, mock, commits)
			preview, err := branch.HardClear(context.Background(), "hard-action", "operator_clear", true)
			if scenario == "write" {
				if err == nil || mock.ledgerCalls != 0 || mock.refWrites != 0 {
					t.Fatalf("writer could preview a hard clear: %+v %v", preview, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "changed" {
				commits = testOperations(t, commits, testActor("administrator"), "new-generation", "control",
					mustOp(t, map[string]any{"kind": "Control", "control": "credential_generation", "value": "new_generation", "reason": "rotate"}))
				installMockLog(t, mock, commits)
			}
			revision := preview.PriorGitSHA
			if scenario == "invalid" {
				revision = ""
			}
			_, err = branch.HardClearFromSnapshot(context.Background(), "hard-action", "operator_clear", revision)
			if scenario == "changed" || scenario == "invalid" {
				if err == nil || mock.refWrites != 0 {
					t.Fatalf("hard clear accepted an unconfirmed snapshot: %v", err)
				}
				return
			}
			if err != nil || mock.refWrites != 1 {
				t.Fatalf("confirmed hard clear failed: %v", err)
			}
			result, err := branch.HardClearFromSnapshot(context.Background(), "hard-action", "operator_clear", revision)
			if err != nil || result.Status != "already_committed" || mock.refWrites != 1 {
				t.Fatalf("confirmed hard-clear retry performed another reset: %+v %v", result, err)
			}
		})
	}
}
