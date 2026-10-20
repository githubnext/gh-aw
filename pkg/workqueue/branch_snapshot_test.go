package workqueue

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func snapshotCancellation(t *testing.T, node WorkDefinition) Request {
	t.Helper()
	request, err := NewRequest("snapshot-clear", "cancel_work", testActor("administrator"), OperationsParameters{Operations: []Operation{
		mustOp(t, map[string]any{"kind": "WorkCancellation", "work_id": node.WorkID, "reason": "operator_clear"}),
	}})
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func snapshotReadmission(t *testing.T, selected []QueueCommit, priorGitSHA string, node WorkDefinition) []QueueCommit {
	t.Helper()
	actor := testActor("administrator")
	paused := testOperations(t, selected, actor, "pause-before-reset", "control",
		mustOp(t, map[string]any{"kind": "Control", "control": "admission_paused", "value": true, "reason": "reset"}),
		mustOp(t, map[string]any{"kind": "Control", "control": "grants_paused", "value": true, "reason": "reset"}))
	state, err := Replay(paused)
	if err != nil {
		t.Fatal(err)
	}
	reset, _, err := prepareHardClear(state, actor, "hard-reset", "operator_reset", priorGitSHA)
	if err != nil {
		t.Fatal(err)
	}
	resume, err := NewRequest("resume-after-reset", "control", actor, OperationsParameters{Operations: []Operation{
		mustOp(t, map[string]any{"kind": "Control", "control": "admission_paused", "value": false, "reason": "resume"}),
		mustOp(t, map[string]any{"kind": "Control", "control": "grants_paused", "value": false, "reason": "resume"}),
	}})
	if err != nil {
		t.Fatal(err)
	}
	at := reset[len(reset)-1].At + 1
	reset, _, _, err = BuildCandidate(reset, actor, resume, at)
	if err != nil {
		t.Fatal(err)
	}
	// Reuse the original submission and graph/node identities, as hard clear permits.
	producer := testActor("producer")
	submit, err := NewRequest("submit", "submit", producer, SubmitParameters{Nodes: []WorkDefinition{node}})
	if err != nil {
		t.Fatal(err)
	}
	reset, _, _, err = BuildCandidate(reset, producer, submit, at+1)
	if err != nil {
		t.Fatal(err)
	}
	return reset
}

func installSnapshotProgression(t *testing.T, mock *queueAPI, commits []QueueCommit, checkpoints int) []QueueCommit {
	t.Helper()
	prior := mock.head
	installMockLog(t, mock, commits)
	gitCommit := mock.commits[mock.head]
	gitCommit.Parents = []string{prior}
	mock.commits[mock.head] = gitCommit
	for range checkpoints {
		prior = mock.head
		var err error
		commits, err = CompactCheckpoint(commits, prior, testActor("administrator"), commits[len(commits)-1].At+1)
		if err != nil {
			t.Fatal(err)
		}
		installMockLog(t, mock, commits)
		gitCommit = mock.commits[mock.head]
		gitCommit.Parents = []string{prior}
		mock.commits[mock.head] = gitCommit
	}
	return commits
}

func TestBranchSnapshotPublicationRejectsResetReadmission(t *testing.T) {
	for _, stage := range []string{"initial_read", "conflict_retry", "final_acknowledgment"} {
		for _, checkpoints := range []int{0, 1} {
			t.Run(fmt.Sprintf("%s/checkpoints_%d", stage, checkpoints), func(t *testing.T) {
				branch, mock := newQueueAPI(t)
				selected := testGenesis(t, nil)
				node := testNode(t, selected, "reused")
				selected = testSubmit(t, selected, "submit", node)
				installMockLog(t, mock, selected)
				request := snapshotCancellation(t, node)
				replacement := snapshotReadmission(t, selected, mock.head, node)
				var expected []QueueCommit
				replace := func() { expected = installSnapshotProgression(t, mock, replacement, checkpoints) }
				wantWrites := 0
				switch stage {
				case "initial_read":
					replace()
				case "conflict_retry":
					mock.concurrent = replace
					wantWrites = 1
				case "final_acknowledgment":
					mock.conflicts = maxRetries
					read := branch.ledgerRequestFn
					reads := 0
					branch.ledgerRequestFn = func(ctx context.Context, method, endpoint string, body, response any) error {
						if method == http.MethodGet && endpoint == "git/ref/heads/"+DefaultBranch {
							reads++
							if reads == maxRetries+1 {
								replace()
							}
						}
						return read(ctx, method, endpoint, body, response)
					}
					wantWrites = maxRetries
				}
				_, err := branch.PublishFromSnapshot(context.Background(), testActor("administrator"), request, selected)
				if err == nil || !strings.HasPrefix(err.Error(), "ledger_nonextending:") || mock.refWrites != wantWrites {
					t.Fatalf("replacement ledger accepted at %s: writes=%d err=%v", stage, mock.refWrites, err)
				}
				current, err := branch.Read(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				state, err := Replay(current)
				if err != nil || !sameJSON(expected, current) || state.Works[node.WorkID].State != "available" ||
					state.Stats.Cancelled != 0 || state.Requests[request.ID].ID != "" {
					t.Fatalf("snapshot cancellation changed readmitted Work: %+v %v", state.Stats, err)
				}
			})
		}
	}
}

func TestBranchSnapshotPublicationAllowsProgressionAndCheckpoints(t *testing.T) {
	for _, scenario := range []struct {
		name               string
		conflict           bool
		selectedCheckpoint bool
	}{
		{name: "initial_read"},
		{name: "conflict_retry", conflict: true},
		{name: "checkpoint_selection", selectedCheckpoint: true},
		{name: "checkpoint_selection_conflict", selectedCheckpoint: true, conflict: true},
	} {
		for _, checkpoints := range []int{0, 1, 2} {
			t.Run(fmt.Sprintf("%s/checkpoints_%d", scenario.name, checkpoints), func(t *testing.T) {
				branch, mock := newQueueAPI(t)
				selected := testGenesis(t, nil)
				node := testNode(t, selected, "selected")
				selected = testSubmit(t, selected, "submit", node)
				installMockLog(t, mock, selected)
				if scenario.selectedCheckpoint {
					selected = installSnapshotProgression(t, mock, selected, 1)
				}
				later := testNode(t, selected, "later")
				submit, err := NewRequest("later-submit", "submit", testActor("producer"), SubmitParameters{Nodes: []WorkDefinition{later}})
				if err != nil {
					t.Fatal(err)
				}
				progressed, _, _, err := BuildCandidate(selected, testActor("producer"), submit, selected[len(selected)-1].At+1)
				if err != nil {
					t.Fatal(err)
				}
				progress := func() { installSnapshotProgression(t, mock, progressed, checkpoints) }
				wantWrites := 1
				if scenario.conflict {
					mock.concurrent, wantWrites = progress, 2
				} else {
					progress()
				}
				publication, err := branch.PublishFromSnapshot(context.Background(), testActor("administrator"), snapshotCancellation(t, node), selected)
				if err != nil || !publication.Changed || mock.refWrites != wantWrites {
					t.Fatalf("same-ledger progression rejected: %+v writes=%d err=%v", publication, mock.refWrites, err)
				}
				current, err := branch.Read(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				state, err := Replay(current)
				if err != nil || state.Works[node.WorkID].State != "cancelled" || state.Works[later.WorkID].State != "available" ||
					state.Stats.Cancelled != 1 || state.Stats.Available != 1 {
					t.Fatalf("publication swept beyond its snapshot: %+v %v", state.Stats, err)
				}
				installSnapshotProgression(t, mock, current, 1)
				recovered, err := branch.PublishFromSnapshot(context.Background(), testActor("administrator"), snapshotCancellation(t, node), selected)
				if err != nil || recovered.Changed || recovered.Commit == nil || recovered.Commit.ID != publication.Commit.ID ||
					mock.refWrites != wantWrites {
					t.Fatalf("checkpoint lost the original publication acknowledgment: %+v %v", recovered, err)
				}
			})
		}
	}
}

func TestBranchSnapshotPublicationRequiresSnapshot(t *testing.T) {
	branch, mock := newQueueAPI(t)
	if _, err := branch.PublishFromSnapshot(context.Background(), testActor("administrator"), Request{}, nil); err == nil ||
		!strings.HasPrefix(err.Error(), "queue_missing:") || mock.ledgerCalls != 0 || mock.refWrites != 0 {
		t.Fatalf("empty snapshot initialized a ledger: %v", err)
	}
}
