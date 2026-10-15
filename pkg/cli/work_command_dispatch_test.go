package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestWorkQueueDispatchPinsVerifiedRevisionWhenDefaultBranchMoves(t *testing.T) {
	for _, preview := range []bool{false, true} {
		t.Run(fmt.Sprintf("preview=%t", preview), func(t *testing.T) {
			readCalls := mockWorkDispatcherAPI(t)
			calls := 0
			command := newWorkDispatchTestCommand(func(ctx context.Context, name string, options RunOptions) error {
				calls++
				if name != "dispatcher" || options.RepoOverride != "owner/repo" ||
					options.RefOverride != "refs/tags/gh-aw/work-queue/"+strings.Repeat("f", 40) ||
					options.ExpectedRevision != strings.Repeat("f", 40) ||
					options.JSON || !options.WaitForCompletion || options.Push || options.Enable || options.Approve ||
					len(options.Inputs) != 0 || options.RepeatCount != 0 {
					t.Fatalf("dispatch changed trusted run options: %s %+v", name, options)
				}
				return nil
			})
			var output bytes.Buffer
			command.SetOut(&output)
			args := []string{"--repo", "owner/repo", "--json", "dispatch", "dispatcher", "--wait"}
			if preview {
				args = append(args, "--dry-run")
			}
			command.SetArgs(args)
			if err := command.Execute(); err != nil {
				t.Fatal(err)
			}
			expectedAPICalls, expectedRef := 7, "refs/tags/gh-aw/work-queue/"+strings.Repeat("f", 40)
			if preview {
				expectedAPICalls, expectedRef = 4, "main"
			}
			if *readCalls != expectedAPICalls {
				t.Fatalf("unexpected API calls: got %d, want %d", *readCalls, expectedAPICalls)
			}
			var result struct {
				Status, Revision, Ref, Workflow, Repository string
			}
			if err := json.Unmarshal(output.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			expectedCalls, expectedStatus := 1, "activated"
			if preview {
				expectedCalls, expectedStatus = 0, "preview"
			}
			if calls != expectedCalls || result.Status != expectedStatus || result.Revision != strings.Repeat("f", 40) ||
				result.Repository != "owner/repo" || result.Ref != expectedRef || result.Workflow != "dispatcher.lock.yml" {
				t.Fatalf("incorrect activation count or result: calls=%d output=%s", calls, output.String())
			}
		})
	}
}

func TestWorkQueueDispatchPropagatesRunFailure(t *testing.T) {
	mockWorkDispatcherAPI(t)
	denied := errors.New("workflow dispatch denied")
	calls := 0
	command := newWorkDispatchTestCommand(func(context.Context, string, RunOptions) error {
		calls++
		return denied
	})
	command.SetArgs([]string{"--repo", "owner/repo", "dispatch", "dispatcher"})
	if err := command.Execute(); !errors.Is(err, denied) || calls != 1 {
		t.Fatalf("dispatch failure was hidden or retried: %v calls=%d", err, calls)
	}
}

func TestWorkQueueDispatchRejectsAuthorityOverrides(t *testing.T) {
	for _, args := range [][]string{
		{"dispatch"},
		{"dispatch", "dispatcher", "another"},
		{"dispatch", "dispatcher", "--claim-id", "chosen"},
		{"dispatch", "dispatcher", "--work-id", "chosen"},
		{"dispatch", "dispatcher", "--worker-profile", "worker"},
		{"dispatch", "dispatcher", "--ref", "unreviewed"},
		{"--branch", "other-queue", "dispatch", "dispatcher"},
		{"--request-id", "forged-origin", "dispatch", "dispatcher"},
	} {
		command := newWorkDispatchTestCommand(func(context.Context, string, RunOptions) error {
			t.Fatal("invalid command reached workflow dispatch")
			return nil
		})
		command.SetOut(io.Discard)
		command.SetErr(io.Discard)
		command.SetArgs(args)
		if err := command.Execute(); err == nil {
			t.Fatalf("unsafe assignment override was accepted: %v", args)
		}
	}
}

func TestWorkQueueDispatchRequiresCorrelatedRun(t *testing.T) {
	info, err := resolveWorkflowRunInfo("dispatcher.lock.yml", "", RunOptions{ExpectedRevision: strings.Repeat("f", 40)})
	if err == nil || info != nil || !strings.Contains(err.Error(), "refusing to select the latest") {
		t.Fatalf("unidentified activation selected another workflow run: %+v %v", info, err)
	}
}

func TestWorkQueueDispatchWaitRequiresIdentity(t *testing.T) {
	opts := RunOptions{WaitForCompletion: true, ExpectedRevision: strings.Repeat("f", 40)}
	unresolved := errors.New("unresolved native run")
	if err := waitForWorkflowRunCompletion(context.Background(), opts, nil, unresolved, time.Time{}, "main"); !errors.Is(err, unresolved) {
		t.Fatalf("unresolved activation wait succeeded: %v", err)
	}
	if err := waitForWorkflowRunCompletion(context.Background(), opts, nil, nil, time.Time{}, "main"); err == nil {
		t.Fatal("missing run identity wait succeeded")
	}
	if err := waitForWorkflowRunCompletion(context.Background(), opts, &WorkflowRunInfo{DatabaseID: 202}, nil, time.Time{}, "main"); err == nil {
		t.Fatal("missing repository wait succeeded")
	}
}

func TestWorkQueueDispatchVerifiesNativeRevision(t *testing.T) {
	for _, mismatch := range []string{"", "revision", "repository", "workflow", "event", "id", "denied"} {
		t.Run(mismatch, func(t *testing.T) {
			t.Setenv("GH_HOST", "github.com")
			t.Setenv("GH_TOKEN", "test-token")
			original := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = original })
			http.DefaultTransport = workAPIRoundTripper(func(request *http.Request) (*http.Response, error) {
				if request.Method != http.MethodGet || request.URL.Path != "/repos/owner/repo/actions/runs/202" {
					t.Fatalf("unexpected native run verification: %s %s", request.Method, request.URL.Path)
				}
				run := map[string]any{"id": 202, "head_sha": strings.Repeat("f", 40),
					"path": ".github/workflows/dispatcher.lock.yml", "event": "workflow_dispatch",
					"repository": map[string]string{"full_name": "owner/repo"}}
				switch mismatch {
				case "revision":
					run["head_sha"] = strings.Repeat("a", 40)
				case "repository":
					run["repository"] = map[string]string{"full_name": "other/repo"}
				case "workflow":
					run["path"] = ".github/workflows/worker.lock.yml"
				case "event":
					run["event"] = "push"
				case "id":
					run["id"] = 203
				}
				status := http.StatusOK
				if mismatch == "denied" {
					status = http.StatusForbidden
				}
				data, _ := json.Marshal(run)
				return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}},
					Body: io.NopCloser(bytes.NewReader(data)), Request: request}, nil
			})
			err := verifyExpectedWorkflowRun(context.Background(), "dispatcher.lock.yml", &WorkflowRunInfo{DatabaseID: 202},
				RunOptions{WorkflowRunFlags: WorkflowRunFlags{RepoOverride: "owner/repo"}, ExpectedRevision: strings.Repeat("f", 40)})
			if (err != nil) != (mismatch != "") {
				t.Fatalf("native revision mismatch=%q error=%v", mismatch, err)
			}
		})
	}
}

func newWorkDispatchTestCommand(run func(context.Context, string, RunOptions) error) *cobra.Command {
	command := NewWorkCommand()
	existing, _, _ := command.Find([]string{"dispatch"})
	command.RemoveCommand(existing)
	command.AddCommand(workDispatchCommandWithRunner(run))
	return command
}

func mockWorkDispatcherAPI(t *testing.T) *int {
	return mockWorkDispatcherActivationAPI(t, false)
}

func TestWorkQueueDispatchRejectsSnapshotMismatch(t *testing.T) {
	mockWorkDispatcherActivationAPI(t, true)
	command := newWorkDispatchTestCommand(func(context.Context, string, RunOptions) error {
		t.Fatal("snapshot mismatch reached workflow dispatch")
		return nil
	})
	command.SetArgs([]string{"--repo", "owner/repo", "dispatch", "dispatcher"})
	if err := command.Execute(); err == nil || !strings.Contains(err.Error(), "snapshot tag does not match") {
		t.Fatalf("snapshot mismatch did not fail closed: %v", err)
	}
}

func mockWorkDispatcherActivationAPI(t *testing.T, mismatch bool) *int {
	t.Helper()
	t.Setenv("GH_HOST", "github.com")
	t.Setenv("GH_TOKEN", "test-token")
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	calls := new(int)
	tagCreated := false
	defaultRevision := strings.Repeat("f", 40)
	http.DefaultTransport = workAPIRoundTripper(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Authorization") != "token test-token" {
			t.Fatal("dispatcher request was not authenticated")
		}
		*calls++
		var value any
		status := http.StatusOK
		tag := "refs/tags/gh-aw/work-queue/" + strings.Repeat("f", 40)
		if request.Method == http.MethodPost {
			if request.URL.Path != "/repos/owner/repo/git/refs" || tagCreated {
				t.Fatalf("unexpected or repeated mutation: %s", request.URL.Path)
			}
			var body map[string]string
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if len(body) != 2 || body["ref"] != tag || body["sha"] != strings.Repeat("f", 40) {
				t.Fatalf("snapshot creation changed verified route: %v", body)
			}
			tagCreated = true
			defaultRevision = strings.Repeat("a", 40)
			status = http.StatusCreated
		} else if request.Method != http.MethodGet {
			t.Fatalf("unexpected dispatcher mutation: %s", request.Method)
		}
		switch request.URL.Path {
		case "/repos/owner/repo/git/refs":
			value = map[string]any{}
		case "/repos/owner/repo/git/ref/" + strings.TrimPrefix(tag, "refs/"):
			if !tagCreated && !mismatch {
				status = http.StatusNotFound
				value = map[string]string{"message": "Not Found"}
			} else {
				sha := strings.Repeat("f", 40)
				if mismatch {
					sha = strings.Repeat("a", 40)
				}
				value = map[string]any{"ref": tag, "object": map[string]string{"type": "commit", "sha": sha}}
			}
		case "/repos/owner/repo":
			value = map[string]any{"full_name": "owner/repo", "default_branch": "main"}
		case "/repos/owner/repo/git/ref/heads/main":
			value = map[string]any{"ref": "refs/heads/main", "object": map[string]string{"sha": defaultRevision}}
		case "/repos/owner/repo/contents/.github/workflows/dispatcher.md",
			"/repos/owner/repo/contents/.github/workflows/dispatcher.lock.yml":
			if request.URL.Query().Get("ref") != strings.Repeat("f", 40) {
				t.Fatal("dispatcher artifacts were read at a moving ref")
			}
			content := "---\ntools:\n  work-queue: true\nsafe-outputs:\n  dispatch-workflow:\n    workflows: [worker]\n---\nDispatch"
			if strings.HasSuffix(request.URL.Path, ".yml") {
				content = "on:\n  workflow_dispatch:\njobs:\n  agent:\n    env:\n      GH_AW_WORK_QUEUE_ROLE: dispatcher\n" +
					"    steps:\n      - id: work_queue_controls\n        env:\n" +
					"          GH_AW_WORK_QUEUE_CONTROL_CONFIG: '{\"work_queue_workflows\":[\"worker\"],\"aw_context_workflows\":[\"worker\"]}'\n"
			}
			value = map[string]string{"type": "file", "path": strings.TrimPrefix(request.URL.Path, "/repos/owner/repo/contents/"),
				"encoding": "base64", "content": base64.StdEncoding.EncodeToString([]byte(content))}
		default:
			t.Fatalf("unexpected dispatcher read: %s", request.URL.Path)
		}
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(bytes.NewReader(data)), Request: request}, nil
	})
	return calls
}
