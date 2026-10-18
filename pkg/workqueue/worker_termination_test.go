package workqueue

import "testing"

func TestWorkerTerminationReleaseParity(t *testing.T) {
	commits, assignment := boundAssignment(t)
	commits = finishMember(t, commits, assignment, 0, "completed")
	state, err := Replay(commits)
	if err != nil {
		t.Fatal(err)
	}
	proof := terminalFor(state, assignment)
	proof.Kind, proof.Status, proof.Conclusion, proof.Receipt = "worker_termination", "", "", "verified-quiescent-jobs"
	actor := workerActor(assignment, "")
	actor.Role = "reconciler"
	operations := []Operation{}
	for _, member := range assignment.Claims[1:] {
		operations = append(operations, mustOp(t, map[string]any{
			"kind": "ClaimCancellation", "work_id": member.WorkID, "claim_id": member.ClaimID,
			"reason": "worker_termination", "retry_not_before": 34000,
		}))
	}
	operations = append(operations, mustOp(t, map[string]any{"kind": "Release", "dispatch_id": assignment.DispatchID, "evidence": proof}))
	released := testOperations(t, commits, actor, "inline-termination", "release", operations...)
	result, err := Replay(released)
	if err != nil || result.Stats.Dispatches != 0 || result.Stats.Completed != 1 ||
		result.Works[assignment.Claims[0].WorkID].Barrier != "pending" {
		t.Fatalf("inline termination changed delivery or retained capacity: %+v %v", result.Stats, err)
	}
	assertRecoveryReplayParity(t, released, "")
	for _, kind := range []string{"Result", "DeliveryFailure"} {
		t.Run("not-"+kind, func(t *testing.T) {
			member := assignment.Claims[0]
			operation := map[string]any{
				"kind": kind, "work_id": member.WorkID, "claim_id": member.ClaimID,
				"completion_id": state.Works[member.WorkID].CompletionID, "evidence": proof,
			}
			requestKind := "result"
			if kind == "Result" {
				operation["descriptor"] = map[string]any{"ok": true}
			} else {
				requestKind = "delivery_failure"
				operation["reason"], operation["disposition"] = "verification_exhausted", "unknown"
			}
			invalid := []Operation{mustOp(t, operation)}
			request, err := NewRequest("not-"+kind, requestKind, actor, OperationsParameters{Operations: invalid})
			if err != nil {
				t.Fatal(err)
			}
			if _, _, _, err := BuildCandidate(commits, actor, request, 4000); err == nil {
				t.Fatal("worker termination manufactured a delivery outcome")
			}
			last := released[len(released)-1]
			last.Actor, last.Request, last.Operations = actor, request, invalid
			assertRecoveryReplayParity(t, append(append([]QueueCommit(nil), commits...), last), "evidence_invalid")
		})
	}
	for _, test := range []string{"foreign-run", "rerun", "foreign-workflow", "foreign-principal", "no-receipt", "terminal-status", "terminal-conclusion", "wrong-source", "wrong-bound-run"} {
		t.Run(test, func(t *testing.T) {
			publisher, evidence := actor, proof
			switch test {
			case "foreign-run":
				publisher.RunID = "999"
			case "rerun":
				publisher.RunAttempt = 2
			case "foreign-workflow":
				publisher.Workflow = ".github/workflows/other.yml"
			case "foreign-principal":
				publisher.Principal = "999"
			case "no-receipt":
				evidence.Receipt = ""
			case "terminal-status":
				evidence.Status = "completed"
			case "terminal-conclusion":
				evidence.Conclusion = "success"
			case "wrong-source":
				evidence.Source = "trusted_publisher"
			case "wrong-bound-run":
				evidence.RunID = "999"
			}
			invalid := append([]Operation(nil), operations...)
			invalid[len(invalid)-1] = mustOp(t, map[string]any{"kind": "Release", "dispatch_id": assignment.DispatchID, "evidence": evidence})
			request, err := NewRequest("invalid-"+test, "release", publisher, OperationsParameters{Operations: invalid})
			if err != nil {
				t.Fatal(err)
			}
			if _, _, _, err := BuildCandidate(commits, publisher, request, 4000); err == nil {
				t.Fatal("invalid worker termination released capacity")
			}
			last := released[len(released)-1]
			last.Actor, last.Request, last.Operations = publisher, request, invalid
			assertRecoveryReplayParity(t, append(append([]QueueCommit(nil), commits...), last), "evidence_invalid")
		})
	}
}
