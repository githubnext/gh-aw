package workqueue

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func aliasedCheckpointBranch(t *testing.T) (Branch, *queueAPI, []QueueCommit) {
	t.Helper()
	branch, mock := newQueueAPI(t)
	history := testGenesis(t, deploymentPolicy)
	history = testSubmit(t, history, "submit-test-work", testNode(t, history, "test-work"))
	history, _ = testGrant(t, history, "reserve-test-work", 1, 1)
	prior := strings.Repeat("a", 40)
	parent := strings.Repeat("b", 40)
	head := strings.Repeat("c", 40)
	checkpoint, err := CompactCheckpoint(history, prior, testActor("administrator"), 3001)
	if err != nil {
		t.Fatal(err)
	}
	valid := deployTestWorker(t, checkpoint, "upgrade-worker", "default", strings.Repeat("d", 40), strings.Repeat("a", 64), true)
	state, err := Replay(valid)
	if err != nil {
		t.Fatal(err)
	}
	deployments, err := canonicalValue(state.Deployments)
	if err != nil {
		t.Fatal(err)
	}
	damaged := append([]QueueCommit(nil), valid...)
	damaged[0], err = withCheckpointDeployments(damaged[0], deployments)
	if err != nil {
		t.Fatal(err)
	}
	for sha, commits := range map[string][]QueueCommit{prior: history, parent: checkpoint, head: damaged} {
		data, err := Serialize(commits)
		if err != nil {
			t.Fatal(err)
		}
		mock.logs[sha] = string(data)
		mock.commits[sha] = gitQueueCommit{Tree: sha}
	}
	mock.commits[parent] = gitQueueCommit{Tree: parent, Parents: []string{prior}}
	mock.commits[head] = gitQueueCommit{Tree: head, Parents: []string{parent}}
	mock.head = head
	return branch, mock, valid
}

func assertCheckpointBranchParity(t *testing.T, mock *queueAPI, expected []QueueCommit, expectedCode string) {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Log("JavaScript unavailable; native checkpoint assertions still run")
		return
	}
	commits := map[string]any{}
	for sha, commit := range mock.commits {
		commits[sha] = map[string]any{"tree": commit.Tree, "parents": commit.Parents}
	}
	input, err := json.Marshal(map[string]any{"head": mock.head, "logs": mock.logs, "commits": commits})
	if err != nil {
		t.Fatal(err)
	}
	const script = `
const fs = require("node:fs");
const input = JSON.parse(fs.readFileSync(0, "utf8"));
const { readWorkQueueLog } = require("../../actions/setup/js/work_queue_store.cjs");
const { serializeProjection, serializeTransactionLog } = require("../../actions/setup/js/work_queue_replay.cjs");
const githubClient = { rest: {
  repos: { get: async () => ({ data: { full_name: "owner/repo", size: 1, default_branch: "main" } }) },
  git: {
    getRef: async () => ({ data: { object: { sha: input.head } } }),
    getCommit: async ({ commit_sha }) => {
      const commit = input.commits[commit_sha];
      return { data: { tree: { sha: commit?.tree }, parents: (commit?.parents || []).map(sha => ({ sha })) } };
    },
    getTree: async ({ tree_sha }) => ({ data: { tree: [{ path: "work-queue.jsonl", mode: "100644", type: "blob", sha: tree_sha }] } }),
    getBlob: async ({ file_sha }) => ({ data: { encoding: "base64", content: Buffer.from(input.logs[file_sha]).toString("base64") } })
  }
} };
readWorkQueueLog({ githubClient, owner: "owner", repo: "repo" })
  .then(result => {
    const state = serializeProjection(result.state);
    state.requests = Object.fromEntries(Object.entries(state.requests).map(([id, commit]) => {
      const { checkpoint_events, checkpoint_prior_tip, compacted_parameters_digest, ...wire } = commit;
      return [id, wire];
    }));
    process.stdout.write(JSON.stringify({ canonical: serializeTransactionLog(result.transactions), state }));
  })
  .catch(error => process.stdout.write(JSON.stringify({ code: error.code })));
`
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, node, "-e", script)
	command.Stdin = bytes.NewReader(input)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		t.Fatalf("JavaScript checkpoint branch parity: %v\n%s", err, stderr.String())
	}
	var result struct {
		Canonical string          `json:"canonical"`
		Code      string          `json:"code"`
		State     json.RawMessage `json:"state"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatal(err)
	}
	if result.Code != expectedCode {
		t.Fatalf("JavaScript checkpoint rejection = %q, want %q", result.Code, expectedCode)
	}
	if expectedCode == "" {
		data, err := Serialize(expected)
		if err != nil || result.Canonical != string(data) {
			t.Fatalf("Go and JavaScript recovered different accepted histories: %v", err)
		}
		state, err := Replay(expected)
		if err != nil {
			t.Fatal(err)
		}
		if !sameJSON(state, result.State) {
			data, err := canonicalValue(state)
			if err != nil {
				t.Fatal(err)
			}
			var native, js any
			_ = json.Unmarshal(data, &native)
			_ = json.Unmarshal(result.State, &js)
			t.Fatalf("checkpoint recovery projection parity: %s", firstParityDifference(native, js, "projection"))
		}
	}
}

func TestBranchCheckpointDeploymentRecoveryParity(t *testing.T) {
	branch, mock, valid := aliasedCheckpointBranch(t)
	originalHead := mock.head
	originalLog := mock.logs[originalHead]
	for range 2 {
		recovered, err := branch.Read(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if !sameJSON(recovered, valid) || mock.refWrites != 0 || mock.head != originalHead || mock.logs[originalHead] != originalLog {
			t.Fatal("recovery changed accepted history or wrote during a read")
		}
		state, err := Replay(recovered)
		if err != nil || state.Stats.Claimed != 1 || state.Stats.Dispatches != 1 {
			t.Fatalf("recovery lost original Claim or native reservation: %v", err)
		}
		assertCheckpointBranchParity(t, mock, recovered, "")
	}
	request, err := NewRequest("pause-after-recovery", "control", testActor("administrator"), OperationsParameters{
		Operations: []Operation{mustOp(t, map[string]any{"kind": "Control", "control": "admission_paused", "value": true, "reason": "operator"})},
	})
	if err != nil {
		t.Fatal(err)
	}
	mock.ambiguous = true
	if _, err := branch.Publish(context.Background(), testActor("administrator"), request); err != nil {
		t.Fatal(err)
	}
	healed, err := branch.Read(context.Background())
	if err != nil || !sameJSON(healed[0], valid[0]) {
		t.Fatalf("checked publication failed to persist the unchanged checkpoint: %v", err)
	}
	assertCheckpointBranchParity(t, mock, healed, "")
	writes := mock.refWrites
	if _, err := branch.Publish(context.Background(), testActor("administrator"), request); err != nil || mock.refWrites != writes {
		t.Fatalf("stable retry republished recovered history: %v", err)
	}
}

func TestBranchNestedCheckpointDeploymentRecoveryParity(t *testing.T) {
	branch, mock, valid := aliasedCheckpointBranch(t)
	parent := mock.head
	nested, err := CompactCheckpoint(valid, parent, testActor("administrator"), 4001)
	if err != nil {
		t.Fatal(err)
	}
	data, err := Serialize(nested)
	if err != nil {
		t.Fatal(err)
	}
	head := strings.Repeat("e", 40)
	mock.logs[head] = string(data)
	mock.commits[head] = gitQueueCommit{Tree: head, Parents: []string{parent}}
	mock.head = head
	recovered, err := branch.Read(context.Background())
	if err != nil || !sameJSON(recovered, nested) || mock.refWrites != 0 {
		t.Fatalf("valid current checkpoint could not verify its recovered ancestor: %v", err)
	}
	assertCheckpointBranchParity(t, mock, recovered, "")
}

func TestBranchCheckpointDeploymentRecoveryRejectsCorruptionParity(t *testing.T) {
	for _, defect := range []string{"work", "deployment", "metadata", "digest", "tail", "foreign-parent", "merge-parent", "missing-parent", "unverified-history", "lost-parent-request"} {
		t.Run(defect, func(t *testing.T) {
			branch, mock, valid := aliasedCheckpointBranch(t)
			head := mock.head
			parent := mock.commits[head].Parents[0]
			damaged, err := Parse([]byte(mock.logs[head]))
			if err != nil {
				t.Fatal(err)
			}

			var operation CheckpointOperation
			if err := json.Unmarshal(damaged[0].Operations[0], &operation); err != nil {
				t.Fatal(err)
			}
			var snapshot map[string]any
			if err := json.Unmarshal(operation.State, &snapshot); err != nil {
				t.Fatal(err)
			}
			switch defect {
			case "work":
				for _, work := range snapshot["works"].(map[string]any) {
					work.(map[string]any)["priority"] = float64(2)
				}
			case "deployment":
				snapshot["deployments"].(map[string]any)["default"].(map[string]any)["default"].(map[string]any)["current_contract"] = strings.Repeat("e", 64)
			case "metadata":
				damaged[0].At++
			case "digest":
				operation.StateSHA256 = strings.Repeat("0", 64)
			case "tail":
				var update DeploymentOperation
				if err := json.Unmarshal(damaged[1].Operations[0], &update); err != nil {
					t.Fatal(err)
				}
				update.Available = false
				damaged[1].Operations[0] = mustOp(t, update)
				damaged[1].Request, err = NewRequest(damaged[1].Request.ID, "deployment", damaged[1].Actor, OperationsParameters{Operations: damaged[1].Operations})
				if err != nil {
					t.Fatal(err)
				}
			case "foreign-parent":
				mock.commits[head] = gitQueueCommit{Tree: head, Parents: []string{strings.Repeat("a", 40)}}
			case "merge-parent":
				mock.commits[head] = gitQueueCommit{Tree: head, Parents: []string{parent, strings.Repeat("a", 40)}}
			case "missing-parent":
				mock.commits[head] = gitQueueCommit{Tree: head}
			case "unverified-history":
				prior := strings.Repeat("a", 40)
				history, err := Parse([]byte(mock.logs[prior]))
				if err != nil {
					t.Fatal(err)
				}
				data, _ := Serialize(history[:1])
				mock.logs[prior] = string(data)
			case "lost-parent-request":
				extra := testOperations(t, valid, testActor("administrator"), "must-retain-parent-request", "control",
					mustOp(t, map[string]any{"kind": "Control", "control": "admission_paused", "value": true, "reason": "operator"}))
				data, _ := Serialize(extra)
				mock.logs[parent] = string(data)
			}
			operation.State = mustOp(t, snapshot)
			damaged[0].Operations[0] = mustOp(t, operation)
			data, err := Serialize(damaged)
			if err != nil {
				t.Fatal(err)
			}
			mock.logs[head] = string(data)
			_, err = branch.Read(context.Background())
			if err == nil {
				t.Fatal("accepted corruption outside proven deployment aliasing")
			}
			code := "checkpoint_invalid"
			if defect == "lost-parent-request" {
				code = "ledger_invalid"
			}
			if !strings.Contains(err.Error(), code) || mock.refWrites != 0 {
				t.Fatalf("unexpected recovery rejection or write: %v", err)
			}
			assertCheckpointBranchParity(t, mock, nil, code)
		})
	}
}
