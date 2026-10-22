package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workqueue"
)

func TestWorkSyncIssuesCommandResults(t *testing.T) {
	for _, test := range []struct {
		name   string
		result workqueue.IssueSynchronization
		err    error
		human  string
	}{
		{"positive", workqueue.IssueSynchronization{Synchronized: 2}, nil, "Synchronized 2 existing terminal Work mirrors; request=exact-terminal"},
		{"zero", workqueue.IssueSynchronization{}, nil, "Synchronized 0 existing terminal Work mirrors"},
		{"disabled", workqueue.IssueSynchronization{Disabled: true}, nil, "Issue synchronization is disabled"},
		{"error", workqueue.IssueSynchronization{}, errors.New("projection_unavailable: synchronization pending"), ""},
		{"partial", workqueue.IssueSynchronization{Synchronized: 1, Pending: []workqueue.IssueSynchronizationPending{{WorkID: "pending-work", Reason: "projection_receipt_pending"}}}, errors.New("projection_receipt_pending: synchronization pending"), "1 terminal Work pending existing mirror receipts"},
	} {
		for _, jsonOutput := range []bool{false, true} {
			t.Run(test.name+map[bool]string{false: "/human", true: "/json"}[jsonOutput], func(t *testing.T) {
				root := NewWorkCommand()
				root.SilenceUsage = true
				for _, command := range root.Commands() {
					if command.Name() == "sync-issues" {
						root.RemoveCommand(command)
					}
				}
				calls := 0
				expected := test.result
				expected.RequestID = "exact-terminal"
				root.AddCommand(workSyncIssuesCommandWithSynchronizer(func(ctx context.Context, branch workqueue.Branch, id string) (workqueue.IssueSynchronization, error) {
					calls++
					if ctx == nil || branch.Remote != "owner/repo" || branch.Name != "selected-queue" || id != expected.RequestID {
						t.Fatal("sync-issues lost exact request, repository, branch, or context")
					}
					return expected, test.err
				}))
				var output bytes.Buffer
				root.SetOut(&output)
				root.SetErr(io.Discard)
				args := []string{"--repo", "owner/repo", "--branch", "selected-queue", "sync-issues", "--request-id", expected.RequestID}
				if jsonOutput {
					args = append(args, "--json")
				}
				root.SetArgs(args)
				err := root.Execute()
				if !errors.Is(err, test.err) || calls != 1 {
					t.Fatalf("sync result/error routing failed: calls=%d err=%v", calls, err)
				}
				if test.human == "" {
					if output.Len() != 0 {
						t.Fatal("failed synchronization reported successful output")
					}
				} else if jsonOutput {
					var actual workqueue.IssueSynchronization
					if err := json.Unmarshal(output.Bytes(), &actual); err != nil {
						t.Fatalf("invalid sync JSON: %v (%s)", err, output.String())
					}
					want, _ := json.Marshal(expected)
					got, _ := json.Marshal(actual)
					if !bytes.Equal(got, want) {
						t.Fatalf("sync JSON lost structured progress: %s != %s", got, want)
					}
				} else if !strings.Contains(output.String(), test.human) {
					t.Fatalf("missing human result: %s", output.String())
				}
			})
		}
	}
}

func TestWorkSyncIssuesExplicitRecoveryStillRequiresMaintainOrAdmin(t *testing.T) {
	workClearRoleEnvironment(t, "write")
	command := NewWorkCommand()
	command.SilenceUsage = true
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"--repo", "owner/repo", "sync-issues", "--request-id", "terminal"})
	if err := command.Execute(); err == nil || output.Len() != 0 {
		t.Fatal("ordinary writer obtained explicit arbitrary-request recovery authority")
	}
}

func TestWorkAutomaticTerminalSyncDisabledPreservesWriterResult(t *testing.T) {
	workClearRoleEnvironment(t, "write")
	command := NewWorkCommand()
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	// Parse persistent flags through Cobra to exercise the same inherited surface.
	command.SetArgs([]string{"--repo", "owner/repo", "--json", "stats", "--help"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	publication := &workqueue.Publication{Commit: &workqueue.QueueCommit{
		Request:    workqueue.Request{ID: "terminal"},
		Actor:      workqueue.Actor{Role: "reconciler", Principal: "1001", Repository: "owner/repo"},
		Operations: []workqueue.Operation{mustWorkQueueOperation(t, map[string]string{"kind": "DeliveryFailure"})},
	}}
	var output bytes.Buffer
	command.SetOut(&output)
	result := map[string]string{"status": "committed", "reason": "released"}
	if err := workPrintTerminalPublication(command, publication, result, "queue transition committed"); err != nil {
		t.Fatalf("disabled automatic projection failed existing writer reconciliation: %v", err)
	}
	if !strings.Contains(output.String(), `"status": "committed"`) {
		t.Fatal("automatic synchronization suppressed the successful queue result")
	}
	publication.Commit.Actor.Principal = "999"
	output.Reset()
	if err := workPrintTerminalPublication(command, publication, result, "queue transition committed"); err == nil ||
		!strings.Contains(err.Error(), "sync-issues --request-id terminal") || !strings.Contains(output.String(), `"status": "committed"`) {
		t.Fatal("pending mirror error concealed the committed queue transition")
	}
}
