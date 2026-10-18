package workqueue

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func checkpointRegressionJS(t *testing.T, mode string, input any, output any) {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal("checkpoint regression requires JavaScript parity tooling")
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, node, "../../actions/setup/js/work_queue_checkpoint_regression_helpers.cjs", mode)
	command.Stdin = bytes.NewReader(data)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	raw, err := command.Output()
	if err != nil {
		t.Fatalf("checkpoint JavaScript regression: %v\n%s", err, stderr.String())
	}
	if err := json.Unmarshal(raw, output); err != nil {
		t.Fatalf("checkpoint JavaScript result: %v\n%s", err, raw)
	}
}

func assertCheckpointGitRegression(t *testing.T, branch Branch, expected []QueueCommit, watermark int64, code string) {
	t.Helper()
	commits, err := branch.Read(context.Background())
	if code != "" {
		if err == nil || !strings.Contains(err.Error(), code) {
			t.Fatalf("native Git checkpoint rejection: %v, want %s", err, code)
		}
	} else {
		if err != nil || !sameJSON(commits, expected) {
			t.Fatalf("native Git checkpoint verification: %v", err)
		}
		state, err := Replay(commits)
		if err != nil || state.timeWatermark != watermark {
			t.Fatalf("native restored watermark: %d, want %d: %v", state.timeWatermark, watermark, err)
		}
	}
	remote, err := filepath.Abs(branch.ledger.remote)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Code          string          `json:"code"`
		Transactions  []QueueCommit   `json:"transactions"`
		State         json.RawMessage `json:"state"`
		TimeWatermark int64           `json:"time_watermark"`
	}
	checkpointRegressionJS(t, "--git-read", map[string]any{"remote": remote}, &result)
	if result.Code != code {
		t.Fatalf("JavaScript Git checkpoint rejection: %q, want %q", result.Code, code)
	}
	if code == "" {
		state, err := Replay(expected)
		if err != nil || !sameJSON(result.Transactions, expected) || !sameJSON(state, result.State) || result.TimeWatermark != watermark {
			t.Fatalf("JavaScript Git checkpoint projection/watermark parity: %d, want %d: %v", result.TimeWatermark, watermark, err)
		}
	}
}

func rewriteCheckpointRegression(t *testing.T, checkpoint []QueueCommit, change func(*checkpointSnapshot, *QueueCommit, *CheckpointOperation)) []QueueCommit {
	t.Helper()
	result := append([]QueueCommit(nil), checkpoint...)
	operation, err := decodeCheckpointOperation(result[0])
	if err != nil {
		t.Fatal(err)
	}
	snapshot, _, err := decodeCheckpointSnapshot(operation.State)
	if err != nil {
		t.Fatal(err)
	}
	change(&snapshot, &result[0], &operation)
	logical, err := canonicalValue(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	operation.StateSHA256 = hashBytes(logical)
	operation.State, err = encodeCheckpointSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	result[0].Operations = []Operation{mustOp(t, operation)}
	result[0].Request, err = NewRequest(result[0].Request.ID, "checkpoint", result[0].Actor, CheckpointParameters{
		PriorGitSHA: operation.PriorGitSHA, PriorTip: operation.PriorTip,
		HistorySHA256: operation.HistorySHA256, StateSHA256: operation.StateSHA256,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestCheckpointLegacyGitHistoryAndHashParity(t *testing.T) {
	branch, _ := gitQueueBranch(t)
	history := testGenesis(t, nil)
	history = testOperations(t, history, testActor("administrator"), "clock", "control", mustOp(t, map[string]any{
		"kind": "Control", "control": "grants_paused", "value": false, "reason": "clock",
	}))
	work := testNode(t, history, "legacy-work")
	history = testSubmit(t, history, "backdated-submit", work)
	prior := writeRawQueueGit(t, branch, history)
	assertCheckpointGitRegression(t, branch, history, 4000, "")
	current, err := CompactCheckpoint(history, prior, testActor("administrator"), 4000)
	if err != nil {
		t.Fatal(err)
	}
	legacy := rewriteCheckpointRegression(t, current, func(snapshot *checkpointSnapshot, root *QueueCommit, _ *CheckpointOperation) {
		snapshot.LastAt, root.At = 2000, 2000
	})
	currentOperation, _ := decodeCheckpointOperation(current[0])
	legacyOperation, _ := decodeCheckpointOperation(legacy[0])
	if currentOperation.StateSHA256 == legacyOperation.StateSHA256 {
		t.Fatal("legacy fixture did not exercise the historical final timestamp hash")
	}
	var js struct {
		Checkpoint []QueueCommit `json:"checkpoint"`
	}
	checkpointRegressionJS(t, "--checkpoint", map[string]any{
		"transactions": history, "prior_git_sha": prior, "actor": testActor("administrator"), "at": 2000, "legacy": true,
	}, &js)
	if !sameJSON(js.Checkpoint[0].Request, legacy[0].Request) {
		t.Fatal("independent legacy builders disagree on checkpoint hash")
	}
	parent := writeRawQueueGit(t, branch, legacy, prior)
	assertCheckpointGitRegression(t, branch, legacy, 4000, "")
	if !extending(history, legacy) {
		t.Fatal("verified legacy checkpoint is not a publication extension")
	}
	if _, err := CompactCheckpoint(history, prior, testActor("administrator"), 2000); err == nil {
		t.Fatal("new checkpoint writer accepted an envelope below its watermark")
	}
	suffix := testNode(t, legacy, "after-legacy")
	deadline := int64(3500)
	suffix.ExpiresAt = &deadline
	extended := testSubmit(t, legacy, "after-legacy-submit", suffix)
	state, err := Replay(extended)
	if err != nil || state.Works[suffix.WorkID].CancellationReason != "work_expired" {
		t.Fatalf("verified legacy restore discarded the receipts' watermark: %v", err)
	}
	next := writeRawQueueGit(t, branch, extended, parent)
	assertCheckpointGitRegression(t, branch, extended, 4000, "")
	nested, err := CompactCheckpoint(extended, next, testActor("administrator"), 4001)
	if err != nil {
		t.Fatal(err)
	}
	writeRawQueueGit(t, branch, nested, next)
	assertCheckpointGitRegression(t, branch, nested, 4001, "")

	for _, defect := range []string{"state", "watermark", "hash", "history", "prior-envelope"} {
		t.Run(defect, func(t *testing.T) {
			damaged := rewriteCheckpointRegression(t, legacy, func(snapshot *checkpointSnapshot, root *QueueCommit, op *CheckpointOperation) {
				switch defect {
				case "state":
					snapshot.GrantsPaused = true
				case "watermark":
					snapshot.LastAt = 1999
				case "history":
					op.HistorySHA256 = strings.Repeat("0", 64)
				case "prior-envelope":
					root.At = 1999
				}
			})
			if defect == "hash" {
				operation, err := decodeCheckpointOperation(damaged[0])
				if err != nil {
					t.Fatal(err)
				}
				operation.StateSHA256 = strings.Repeat("0", 64)
				damaged[0].Operations = []Operation{mustOp(t, operation)}
				damaged[0].Request, err = NewRequest(damaged[0].Request.ID, "checkpoint", damaged[0].Actor, CheckpointParameters{
					PriorGitSHA: operation.PriorGitSHA, PriorTip: operation.PriorTip,
					HistorySHA256: operation.HistorySHA256, StateSHA256: operation.StateSHA256,
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			writeRawQueueGit(t, branch, damaged, prior)
			assertCheckpointGitRegression(t, branch, nil, 0, "checkpoint_invalid")
		})
	}
}

func TestCheckpointDeadlineHistoryCannotUseLegacyHash(t *testing.T) {
	for _, compacted := range []bool{false, true} {
		t.Run(map[bool]string{false: "full-history", true: "retained-checkpoint"}[compacted], func(t *testing.T) {
			branch, _ := gitQueueBranch(t)
			history, _, _ := expiringWorkFixture(t)
			prior := writeRawQueueGit(t, branch, history)
			if compacted {
				checkpoint, err := CompactCheckpoint(history, prior, testActor("administrator"), 2500)
				if err != nil {
					t.Fatal(err)
				}
				history, prior = checkpoint, writeRawQueueGit(t, branch, checkpoint, prior)
			}
			history = testOperations(t, history, testActor("administrator"), "clock", "control", mustOp(t, map[string]any{
				"kind": "Control", "control": "grants_paused", "value": false, "reason": "clock",
			}))
			history = testSubmit(t, history, "backdated", testNode(t, history, "backdated"))
			parent := writeRawQueueGit(t, branch, history, prior)
			checkpoint, err := CompactCheckpoint(history, parent, testActor("administrator"), 4000)
			if err != nil {
				t.Fatal(err)
			}
			for _, at := range []int64{2000, 4000} {
				forged := rewriteCheckpointRegression(t, checkpoint, func(snapshot *checkpointSnapshot, root *QueueCommit, _ *CheckpointOperation) {
					snapshot.LastAt, root.At = 2000, at
				})
				writeRawQueueGit(t, branch, forged, parent)
				assertCheckpointGitRegression(t, branch, nil, 0, "checkpoint_invalid")
			}
			writeRawQueueGit(t, branch, checkpoint, parent)
			assertCheckpointGitRegression(t, branch, checkpoint, 4000, "")
		})
	}
}

func TestCheckpointExplicitExpirationCancellationHashParity(t *testing.T) {
	branch, _ := gitQueueBranch(t)
	history, old, _ := expiringWorkFixture(t)
	operation := mustOp(t, map[string]any{"kind": "WorkCancellation", "work_id": old.WorkID, "reason": "work_expired"})
	history = testOperations(t, history, testActor("administrator"), "cancel-expired", "cancel_work", operation)
	state, err := Replay(history)
	if err != nil || !sameJSON(state.Cancellations[old.WorkID], operation) {
		t.Fatalf("accepted cancellation was not retained: %v", err)
	}
	prior := writeRawQueueGit(t, branch, history)
	checkpoint, err := CompactCheckpoint(history, prior, testActor("administrator"), 4000)
	if err != nil {
		t.Fatal(err)
	}
	var js struct {
		Checkpoint []QueueCommit `json:"checkpoint"`
	}
	checkpointRegressionJS(t, "--checkpoint", map[string]any{
		"transactions": history, "prior_git_sha": prior, "actor": testActor("administrator"), "at": 4000,
	}, &js)
	if !sameJSON(js.Checkpoint[0].Request, checkpoint[0].Request) {
		t.Fatal("explicit expiration cancellation produced different checkpoint state hashes")
	}
	for _, candidate := range [][]QueueCommit{checkpoint, js.Checkpoint} {
		writeRawQueueGit(t, branch, candidate, prior)
		assertCheckpointGitRegression(t, branch, candidate, 4000, "")
	}
}
