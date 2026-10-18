package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workqueue"
)

func workHardClearExecute(t *testing.T, extra ...string) (workqueue.HardClearResult, error) {
	t.Helper()
	cmd := NewWorkCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(append([]string{"--repo", "owner/repo", "--json", "clear", "--hard",
		"--reason", "operator_clear", "--request-id", "hard-action"}, extra...))
	err := cmd.Execute()
	var result workqueue.HardClearResult
	if err == nil {
		if err := json.Unmarshal(output.Bytes(), &result); err != nil {
			t.Fatalf("invalid hard-clear JSON: %v (%s)", err, output.String())
		}
	}
	return result, err
}

func workClearPause(t *testing.T, commits []workqueue.QueueCommit) []workqueue.QueueCommit {
	t.Helper()
	return workClearAppend(t, commits, workqueue.Actor{Role: "administrator", Principal: "1001", Repository: "owner/repo"},
		"pause-before-reset", "control", workqueue.OperationsParameters{Operations: []workqueue.Operation{
			mustWorkQueueOperation(t, map[string]any{"kind": "Control", "control": "admission_paused", "value": true, "reason": "operator_pause"}),
			mustWorkQueueOperation(t, map[string]any{"kind": "Control", "control": "grants_paused", "value": true, "reason": "operator_pause"}),
		}}, 9000)
}

func TestWorkClearHardResetsOnlyActiveLedgerAndPreservesGitHistory(t *testing.T) {
	workClearOperatorEnvironment(t, true)
	_, commits, _ := workOperatorFixture(t, false)
	commits = workClearPause(t, commits)
	data, err := workqueue.Serialize(commits)
	if err != nil {
		t.Fatal(err)
	}
	queue := newWorkQueueGitRemote(t, string(data))
	head := queue.head(t)
	preview, err := workHardClearExecute(t, "--dry-run")
	if err != nil || preview.Status != "dry_run" || preview.WorkRemoved != 3 || queue.pushes(t) != 0 || queue.head(t) != head {
		t.Fatalf("unsafe hard preview: %+v %v", preview, err)
	}
	result, err := workHardClearExecute(t)
	if err != nil || !result.Changed || result.Status != "hard_cleared" || result.PriorGitSHA != head ||
		!result.IdentityReset || !result.HistoryInGit || queue.pushes(t) != 1 {
		t.Fatalf("hard reset failed: %+v %v", result, err)
	}
	current := workClearRead(t, queue)
	if len(current.Works) != 0 || len(current.Claims) != 0 || len(current.Dispatches) != 0 ||
		len(current.Clocks) != 0 || !current.AdmissionPaused || !current.GrantsPaused {
		t.Fatalf("hard clear did not produce a fresh paused queue: %+v", current.Stats)
	}
	original, err := workqueue.Replay(commits)
	if err != nil {
		t.Fatal(err)
	}
	assertWorkClearEqual(t, original.Policy, current.Policy)
	if archived := queue.run(t, "", "show", head+":"+workqueue.FileName); archived != string(data) {
		t.Fatal("hard clear erased the archived ledger")
	}
	if parent := strings.TrimSpace(queue.run(t, "", "rev-parse", queue.head(t)+"^")); parent != head {
		t.Fatal("hard clear did not preserve the prior Git head as its parent")
	}
	retry, err := workHardClearExecute(t)
	if err != nil || retry.Status != "already_committed" || retry.Changed || queue.pushes(t) != 1 {
		t.Fatalf("hard retry reset the queue again: %+v %v", retry, err)
	}
	branch := workqueue.Branch{Remote: "owner/repo", Name: workqueue.DefaultBranch}
	if _, err := branch.Compact(context.Background()); err != nil {
		t.Fatal(err)
	}
	pushes := queue.pushes(t)
	retry, err = workHardClearExecute(t)
	if err != nil || retry.Status != "already_committed" || queue.pushes(t) != pushes {
		t.Fatalf("hard retry after checkpoint lost its accepted reset: %+v %v", retry, err)
	}
}

func TestWorkClearHardRefusesUnpausedAndUnsettledQueues(t *testing.T) {
	for _, scenario := range []string{"unpaused", "reserved", "pending", "failed", "not administrator", "missing", "malformed"} {
		t.Run(scenario, func(t *testing.T) {
			workClearOperatorEnvironment(t, scenario != "not administrator")
			_, commits, _ := workOperatorFixture(t, scenario == "reserved")
			if scenario == "pending" || scenario == "failed" {
				_, commits, _ = workClearMixedFixture(t, scenario == "failed")
			}
			if scenario != "unpaused" {
				commits = workClearPause(t, commits)
			}
			data, err := workqueue.Serialize(commits)
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "missing":
				data = nil
			case "malformed":
				data = []byte("malformed\n")
			}
			queue := newWorkQueueGitRemote(t, string(data))
			head := queue.head(t)
			if _, err := workHardClearExecute(t); err == nil || queue.head(t) != head || queue.pushes(t) != 0 {
				t.Fatalf("unsafe hard reset accepted: %v", err)
			}
		})
	}
}
