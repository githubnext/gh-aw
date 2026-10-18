package cli

import (
	"bytes"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/workqueue"
)

func TestWorkClearRequiresMaintainOrAdmin(t *testing.T) {
	for _, role := range []string{"admin", "maintain", "write", "triage", "read", ""} {
		for _, hard := range []bool{false, true} {
			t.Run(role+"/hard_"+strconv.FormatBool(hard), func(t *testing.T) {
				workClearRoleEnvironment(t, role)
				_, commits, _ := workOperatorFixture(t, false)
				if hard {
					commits = workClearPause(t, commits)
				}
				data, err := workqueue.Serialize(commits)
				if err != nil {
					t.Fatal(err)
				}
				queue := newWorkQueueGitRemote(t, string(data))
				confirmations := 0
				workClearConfirmAction = func(string, string, string) (bool, error) {
					confirmations++
					if queue.pushes(t) != 0 {
						t.Fatal("clear wrote before confirmation")
					}
					return true, nil
				}
				for _, dryRun := range []bool{true, false} {
					extra := []string{}
					if dryRun {
						extra = append(extra, "--dry-run")
					}
					if hard {
						_, err = workHardClearExecute(t, extra...)
					} else {
						_, err = workClearExecute(t, extra...)
					}
					allowed := role == "admin" || role == "maintain"
					if (err == nil) != allowed {
						t.Fatalf("role %q authorization differs: %v", role, err)
					}
					if !allowed && (!strings.Contains(err.Error(), "maintain or admin") || queue.pushes(t) != 0 || confirmations != 0) {
						t.Fatalf("unauthorized clear accessed confirmation or publication: %v", err)
					}
				}
				if (role == "admin" || role == "maintain") && (confirmations != 1 || queue.pushes(t) != 1) {
					t.Fatal("authorized clear did not confirm exactly once before writing")
				}
			})
		}
	}
}

func TestWorkClearConfirmationAndPreview(t *testing.T) {
	for _, hard := range []bool{false, true} {
		for _, scenario := range []string{"yes", "no", "error", "dry_run"} {
			t.Run("hard_"+strconv.FormatBool(hard)+"/"+scenario, func(t *testing.T) {
				workClearOperatorEnvironment(t, true)
				state, commits, ids := workOperatorFixture(t, false)
				if hard {
					commits = workClearPause(t, commits)
				}
				data, err := workqueue.Serialize(commits)
				if err != nil {
					t.Fatal(err)
				}
				queue := newWorkQueueGitRemote(t, string(data))
				head := queue.head(t)
				cmd := NewWorkCommand()
				var output, preview bytes.Buffer
				cmd.SetOut(&output)
				cmd.SetErr(&preview)
				args := []string{"--repo", "owner/repo", "--json", "clear", "--reason", "operator_clear", "--request-id", "confirm-action"}
				if hard {
					args = append(args, "--hard")
				}
				if scenario == "dry_run" {
					args = append(args, "--dry-run")
				}
				cmd.SetArgs(args)
				confirmations := 0
				workClearConfirmAction = func(string, string, string) (bool, error) {
					confirmations++
					text := preview.String()
					for _, want := range append(ids, `repository="owner/repo"`, "branch="+workQueueText(workqueue.DefaultBranch), "Snapshot:", "Work: 3", "unreleased native reservations: 0") {
						if !strings.Contains(text, want) {
							t.Fatalf("confirmation omitted queue preview %q: %s", want, text)
						}
					}
					if strings.Contains(text, "PAYLOAD-MUST") || strings.Contains(text, "SECRET-MUST") ||
						queue.head(t) != head || queue.pushes(t) != 0 || output.Len() != 0 {
						t.Fatal("preview leaked payload or wrote before confirmation")
					}
					if hard && (!strings.Contains(text, "HARD RESET") || !strings.Contains(text, "effects may repeat")) {
						t.Fatal("hard confirmation omitted destructive identity warning")
					}
					if !hard && !strings.Contains(text, workQueueText(state.Tip)) {
						t.Fatal("confirmation did not show the selected snapshot")
					}
					if scenario == "error" {
						return false, errors.New("prompt unavailable")
					}
					return scenario == "yes", nil
				}
				err = cmd.Execute()
				if scenario == "no" || scenario == "error" {
					if err == nil || confirmations != 1 || queue.pushes(t) != 0 || queue.head(t) != head || queue.read(t) != string(data) {
						t.Fatalf("unconfirmed clear mutated queue: %v", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				want := 1
				if scenario == "dry_run" {
					want = 0
				}
				if confirmations != want || queue.pushes(t) != want || !strings.HasPrefix(output.String(), "{") {
					t.Fatalf("confirmation count, writes or JSON output changed: %d %d %s", confirmations, queue.pushes(t), output.String())
				}
			})
		}
	}
}

func TestWorkClearRefusesNonInteractiveConfirmation(t *testing.T) {
	for _, hard := range []bool{false, true} {
		t.Run(strconv.FormatBool(hard), func(t *testing.T) {
			workClearOperatorEnvironment(t, true)
			_, commits, _ := workOperatorFixture(t, false)
			if hard {
				commits = workClearPause(t, commits)
			}
			data, err := workqueue.Serialize(commits)
			if err != nil {
				t.Fatal(err)
			}
			queue := newWorkQueueGitRemote(t, string(data))
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			stdin := os.Stdin
			os.Stdin = reader
			t.Cleanup(func() { os.Stdin = stdin })
			t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })
			workClearConfirmAction = console.ConfirmAction
			if hard {
				_, err = workHardClearExecute(t)
			} else {
				_, err = workClearExecute(t)
			}
			if err == nil || !strings.Contains(err.Error(), "clear_confirmation_failed") || !strings.Contains(err.Error(), "not a TTY") ||
				queue.pushes(t) != 0 || queue.read(t) != string(data) {
				t.Fatalf("noninteractive clear did not fail closed: %v", err)
			}
		})
	}
}

func TestWorkClearRechecksRoleAfterConfirmation(t *testing.T) {
	for _, hard := range []bool{false, true} {
		t.Run(strconv.FormatBool(hard), func(t *testing.T) {
			setRole := workClearRoleEnvironment(t, "maintain")
			_, commits, _ := workOperatorFixture(t, false)
			if hard {
				commits = workClearPause(t, commits)
			}
			data, err := workqueue.Serialize(commits)
			if err != nil {
				t.Fatal(err)
			}
			queue := newWorkQueueGitRemote(t, string(data))
			workClearConfirmAction = func(string, string, string) (bool, error) {
				setRole("write")
				return true, nil
			}
			if hard {
				_, err = workHardClearExecute(t)
			} else {
				_, err = workClearExecute(t)
			}
			if err == nil || !strings.Contains(err.Error(), "maintain or admin") || queue.pushes(t) != 0 || queue.read(t) != string(data) {
				t.Fatalf("revoked role retained clear authority: %v", err)
			}
		})
	}
}

func TestWorkClearNoopAndRecoveryDoNotPrompt(t *testing.T) {
	for _, hard := range []bool{false, true} {
		t.Run(strconv.FormatBool(hard), func(t *testing.T) {
			workClearOperatorEnvironment(t, true)
			_, commits, _ := workOperatorFixture(t, false)
			if hard {
				commits = workClearPause(t, commits)
			}
			data, err := workqueue.Serialize(commits)
			if err != nil {
				t.Fatal(err)
			}
			queue := newWorkQueueGitRemote(t, string(data))
			if hard {
				_, err = workHardClearExecute(t)
			} else {
				_, err = workClearExecute(t)
			}
			if err != nil {
				t.Fatal(err)
			}
			workClearConfirmAction = func(string, string, string) (bool, error) {
				t.Fatal("read-only recovery or no-op requested confirmation")
				return false, nil
			}
			if hard {
				_, err = workHardClearExecute(t)
			} else {
				_, err = workClearExecute(t)
			}
			if err != nil || queue.pushes(t) != 1 {
				t.Fatalf("recovery wrote or failed: %v", err)
			}
			if !hard {
				if _, err := workClearExecute(t, "--request-id", "empty-action"); err != nil || queue.pushes(t) != 1 {
					t.Fatalf("empty clear wrote or failed: %v", err)
				}
			}
		})
	}
}

func TestWorkClearHardRejectsChangesAfterConfirmationPreview(t *testing.T) {
	workClearOperatorEnvironment(t, true)
	_, commits, _ := workOperatorFixture(t, false)
	commits = workClearPause(t, commits)
	data, err := workqueue.Serialize(commits)
	if err != nil {
		t.Fatal(err)
	}
	queue := newWorkQueueGitRemote(t, string(data))
	var changed string
	workClearConfirmAction = func(string, string, string) (bool, error) {
		commits = workClearAppend(t, commits, workqueue.Actor{Role: "administrator", Principal: "1001", Repository: "owner/repo"},
			"changed-during-confirmation", "control", workqueue.OperationsParameters{Operations: []workqueue.Operation{
				mustWorkQueueOperation(t, map[string]any{"kind": "Control", "control": "credential_generation", "value": "new_generation", "reason": "rotate"}),
			}}, 10000)
		updated, err := workqueue.Serialize(commits)
		if err != nil {
			t.Fatal(err)
		}
		changed = string(updated)
		queue.write(t, changed)
		return true, nil
	}
	if _, err := workHardClearExecute(t); err == nil || !strings.Contains(err.Error(), "hard_clear_conflict") ||
		queue.pushes(t) != 0 || changed == "" || queue.read(t) != changed {
		t.Fatalf("hard clear reset an unconfirmed snapshot: %v", err)
	}
}
