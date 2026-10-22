package workqueue

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/github/gh-aw/pkg/logger"
)

var checkpointLog = logger.New("workqueue:checkpoint")

func firstQueueCommit(commits []QueueCommit) (QueueCommit, error) {
	if len(commits) > 0 {
		return commits[0], nil
	}
	return QueueCommit{}, queueError("checkpoint_invalid", "checkpoint history is empty")
}

func firstCheckpointOperation(commit QueueCommit) (Operation, error) {
	if len(commit.Operations) > 0 {
		return commit.Operations[0], nil
	}
	return nil, queueError("checkpoint_invalid", "checkpoint operation is missing")
}

func decodeCheckpointOperation(commit QueueCommit) (CheckpointOperation, error) {
	var operation CheckpointOperation
	raw, err := firstCheckpointOperation(commit)
	if err != nil {
		return operation, err
	}
	err = json.Unmarshal(raw, &operation)
	return operation, err
}

func checkpointDeployments(commit QueueCommit) (json.RawMessage, error) {
	raw, err := firstCheckpointOperation(commit)
	if err != nil {
		return nil, err
	}
	var operation struct {
		State map[string]json.RawMessage `json:"state"`
	}
	if err := json.Unmarshal(raw, &operation); err != nil {
		return nil, err
	}
	return operation.State["deployments"], nil
}

func withCheckpointDeployments(commit QueueCommit, deployments json.RawMessage) (QueueCommit, error) {
	raw, err := firstCheckpointOperation(commit)
	if err != nil {
		return QueueCommit{}, err
	}
	var operation map[string]json.RawMessage
	if err := json.Unmarshal(raw, &operation); err != nil {
		return QueueCommit{}, err
	}
	var state map[string]json.RawMessage
	if err := json.Unmarshal(operation["state"], &state); err != nil {
		return QueueCommit{}, err
	}
	state["deployments"] = deployments
	operation["state"], err = canonicalValue(state)
	if err != nil {
		return QueueCommit{}, err
	}
	updated, err := canonicalValue(operation)
	if err != nil {
		return QueueCommit{}, err
	}
	commit.Operations = append([]Operation{updated}, commit.Operations[1:]...)
	return commit, nil
}

func (b Branch) checkpointGitParent(ctx context.Context, head string) (string, error) {
	var gitCommit struct {
		Parents []struct {
			SHA string `json:"sha"`
		} `json:"parents"`
	}
	if err := b.ledgerRequest(ctx, http.MethodGet, "git/commits/"+head, nil, &gitCommit); err != nil {
		return "", err
	}
	if len(gitCommit.Parents) != 1 {
		return "", queueError("checkpoint_invalid", "checkpoint requires exactly one Git parent")
	}
	var parent string
	for _, value := range gitCommit.Parents {
		parent = value.SHA
	}
	if parent == "" || parent == head {
		return "", queueError("checkpoint_invalid", "checkpoint Git parent is invalid")
	}
	return parent, nil
}

func (b Branch) checkpointHistoryParent(ctx context.Context, head, rootID string, visited map[string]struct{}) (string, []QueueCommit, error) {
	for {
		if _, ok := visited[head]; ok {
			return "", nil, queueError("checkpoint_invalid", "checkpoint Git history contains a cycle")
		}
		visited[head] = struct{}{}
		parent, err := b.checkpointGitParent(ctx, head)
		if err != nil {
			return "", nil, err
		}
		previous, _, err := b.readAt(ctx, parent, false, true)
		if err != nil {
			return "", nil, err
		}
		root, err := firstQueueCommit(previous)
		if err != nil {
			return "", nil, err
		}
		if !isCheckpoint(root) || root.ID != rootID {
			return parent, previous, nil
		}
		head = parent
	}
}

// Recover only deployment aliasing proven against the unchanged checkpoint in
// the immediate Git parent. All accepted requests and other state stay intact.
func (b Branch) recoverDeploymentCheckpoint(ctx context.Context, head string, commits []QueueCommit, failure error) ([]QueueCommit, error) {
	ordered, err := causalOrder(commits)
	if err != nil {
		return nil, failure
	}
	if len(ordered) < 2 {
		return nil, failure
	}
	damaged, err := firstQueueCommit(ordered)
	if err != nil {
		return nil, failure
	}
	if !isCheckpoint(damaged) {
		return nil, failure
	}
	parent, err := b.checkpointGitParent(ctx, head)
	if err != nil {
		return nil, err
	}
	original, _, err := b.readAt(ctx, parent, false, false)
	if err != nil {
		return nil, err
	}
	original, err = causalOrder(original)
	if err != nil {
		return nil, failure
	}
	root, err := firstQueueCommit(original)
	if err != nil {
		return nil, failure
	}
	if !isCheckpoint(root) || root.ID != damaged.ID {
		return nil, failure
	}
	deployments, err := checkpointDeployments(root)
	if err != nil || len(deployments) == 0 {
		return nil, failure
	}
	normalized, err := withCheckpointDeployments(damaged, deployments)
	if err != nil || !sameJSON(normalized, root) {
		return nil, failure
	}
	repaired := append([]QueueCommit{root}, ordered[1:]...)
	if !extending(original, repaired) {
		return nil, queueError("ledger_invalid", "checkpoint recovery cannot change accepted history")
	}
	state, err := ReplayForHost(repaired, b.Host)
	if err != nil {
		return nil, err
	}
	damagedDeployments, err := checkpointDeployments(damaged)
	if err != nil || !sameJSON(damagedDeployments, state.Deployments) {
		return nil, failure
	}
	if err := b.verifyCheckpointHistory(ctx, parent, original); err != nil {
		return nil, err
	}
	checkpointLog.Print("Recovered aliased checkpoint deployments from verified Git parent")
	return repaired, nil
}
