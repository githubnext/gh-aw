//go:build !integration

package cli

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workqueue"
	"github.com/stretchr/testify/require"
)

func workQueueReplayTestSnapshot(t *testing.T) []byte {
	t.Helper()
	snapshot := currentWorkQueueFixture(t)
	commits, err := workqueue.Parse([]byte(snapshot["transactionLog"].(string)))
	require.NoError(t, err)
	ledger, err := workqueue.Serialize(commits[:2])
	require.NoError(t, err)
	snapshot["transactionLog"] = string(ledger)
	snapshot["worker"] = nil
	snapshot["role"] = "dispatcher"
	snapshot["origin"] = workqueue.Actor{
		Role: "dispatcher", Principal: "42", Repository: "owner/repo",
		Workflow: ".github/workflows/dispatcher.lock.yml", RunID: "10", RunAttempt: 1,
	}
	data, err := json.Marshal(snapshot)
	require.NoError(t, err)
	return data
}

func workQueueReplayTestZIP(t *testing.T, data []byte, names ...string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, name := range names {
		entry, err := writer.Create(name)
		require.NoError(t, err)
		_, err = entry.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

func TestWorkQueueArtifactReplayLocalPredictsWithoutPublishingOrExposingPayloads(t *testing.T) {
	data := workQueueReplayTestSnapshot(t)
	file := filepath.Join(t.TempDir(), workQueueSnapshotFile)
	require.NoError(t, os.WriteFile(file, data, 0o600))
	command := NewWorkCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"--repo", "owner/repo", "--json", "replay", "--file", file, "--pool", "default", "--max-claims", "2"})
	require.NoError(t, command.Execute())
	var result struct {
		Status     string
		Stats      workqueue.Stats
		Prediction workQueueReplayPrediction
	}
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	require.Equal(t, "artifact_replay", result.Status)
	require.Equal(t, 2, result.Stats.Work)
	require.Zero(t, result.Stats.Claims)
	require.Len(t, result.Prediction.Assignments, 1)
	require.Len(t, result.Prediction.Assignments[0].WorkIDs, 2)
	require.NotContains(t, output.String(), "private-task-content")
	require.NotContains(t, output.String(), "payload")
	after, err := os.ReadFile(file)
	require.NoError(t, err)
	require.Equal(t, data, after)
}

func TestWorkQueueArtifactReplayDownloadsOnlyExactRunAndActivationSnapshot(t *testing.T) {
	data := workQueueReplayTestSnapshot(t)
	archive := workQueueReplayTestZIP(t, data, "../ignored", workQueueSnapshotFile)
	for _, failure := range []string{"", "missing", "expired", "ambiguous", "run_repository", "snapshot_origin", "denied"} {
		t.Run(failure, func(t *testing.T) {
			t.Setenv("GH_HOST", "github.com")
			t.Setenv("GH_TOKEN", "test-token")
			original := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = original })
			var calls []string
			http.DefaultTransport = workAPIRoundTripper(func(request *http.Request) (*http.Response, error) {
				require.Equal(t, http.MethodGet, request.Method, "replay must never publish or launch")
				calls = append(calls, request.URL.Path)
				status, contentType := http.StatusOK, "application/json"
				var body []byte
				switch request.URL.Path {
				case "/repos/owner/repo/actions/runs/10":
					run := workQueueReplayRun{ID: 10, RunAttempt: 1, Path: ".github/workflows/dispatcher.lock.yml"}
					run.Repository.FullName = "owner/repo"
					if failure == "run_repository" {
						run.Repository.FullName = "other/repo"
					}
					if failure == "snapshot_origin" {
						run.RunAttempt = 2
					}
					body, _ = json.Marshal(run)
					if failure == "denied" {
						status = http.StatusForbidden
					}
				case "/repos/owner/repo/actions/runs/10/artifacts":
					require.Equal(t, "100", request.URL.Query().Get("per_page"))
					require.Equal(t, "1", request.URL.Query().Get("page"))
					artifact := fmt.Sprintf(`{"id":20,"name":"activation","expired":%t,"size_in_bytes":%d}`, failure == "expired", len(archive))
					switch failure {
					case "missing":
						body = []byte(`{"artifacts":[]}`)
					case "ambiguous":
						body = []byte(`{"artifacts":[` + artifact + "," + artifact + "]}")
					default:
						body = []byte(`{"artifacts":[` + artifact + "]}")
					}
				case "/repos/owner/repo/actions/artifacts/20/zip":
					contentType, body = "application/zip", archive
				default:
					t.Fatalf("unexpected replay request: %s", request.URL)
				}
				return &http.Response{StatusCode: status, Request: request, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(bytes.NewReader(body))}, nil
			})
			command := NewWorkCommand()
			command.SilenceUsage = true
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetErr(io.Discard)
			command.SetArgs([]string{"--repo", "owner/repo", "--json", "replay", "--run-id", "10"})
			err := command.Execute()
			if failure != "" {
				require.Error(t, err)
				require.Empty(t, output.String())
				if failure == "denied" {
					require.Contains(t, err.Error(), "GitHub denied access")
					require.Contains(t, err.Error(), "can read this repository's Actions runs and artifacts")
					require.Contains(t, err.Error(), "does not require write permissions")
				}
				if failure == "missing" {
					require.Contains(t, err.Error(), "this run has no activation artifact")
					require.Contains(t, err.Error(), "--file /path/to/work-queue.snapshot.json")
				}
			} else {
				require.NoError(t, err)
				require.Contains(t, output.String(), `"run_id": "10"`)
				require.Len(t, calls, 3)
			}
		})
	}
}

func TestWorkQueueArtifactReplayDownloadErrorsExplainRecovery(t *testing.T) {
	for _, test := range []struct {
		status   int
		problem  string
		recovery string
	}{
		{401, "GitHub denied access", "Authenticate gh"},
		{403, "GitHub denied access", "does not require write permissions"},
		{404, "missing, expired, or inaccessible", "Verify --repo and --run-id"},
		{410, "missing, expired, or inaccessible", "--file"},
		{429, "rate-limited the download", "Wait for the rate limit to reset"},
		{503, "did not return the requested run or artifact", "connectivity before retrying"},
	} {
		t.Run(strconv.Itoa(test.status), func(t *testing.T) {
			err := workQueueReplayDownloadError(test.status)
			require.Contains(t, err.Error(), fmt.Sprintf("HTTP %d", test.status))
			require.Contains(t, err.Error(), test.problem)
			require.Contains(t, err.Error(), test.recovery)
		})
	}
}

func TestWorkQueueArtifactReplayRejectsInvalidSelectorsAndSnapshots(t *testing.T) {
	for _, flags := range [][]string{
		{"--run-id", "0"}, {"--run-id", "-1"}, {"--file", ""},
		{"--file", "snapshot", "--run-id", "10"},
		{"--file", "snapshot", "--branch", "other"},
		{"--file", "snapshot", "--request-id", "forged"},
		{"--file", "snapshot", "--max-claims", "2"},
		{"--file", "snapshot", "--pool", ""},
		{"--file", "snapshot", "--pool", "default", "--max-claims", "0"},
		{"--pool", "default"},
	} {
		command := NewWorkCommand()
		command.SetOut(io.Discard)
		command.SetErr(io.Discard)
		command.SetArgs(append([]string{"replay"}, flags...))
		require.Error(t, command.Execute(), "%v", flags)
	}
	data := workQueueReplayTestSnapshot(t)
	for _, failure := range []string{"version", "chain", "repository", "unknown"} {
		t.Run(failure, func(t *testing.T) {
			var snapshot map[string]any
			require.NoError(t, json.Unmarshal(data, &snapshot))
			switch failure {
			case "version":
				snapshot["version"] = 2
			case "chain":
				snapshot["transactionLog"] = `{"private":"secret"}`
			case "repository":
				snapshot["origin"].(map[string]any)["repository"] = "other/repo"
			case "unknown":
				snapshot["private"] = "secret"
			}
			invalid, err := json.Marshal(snapshot)
			require.NoError(t, err)
			file := filepath.Join(t.TempDir(), workQueueSnapshotFile)
			require.NoError(t, os.WriteFile(file, invalid, 0o600))
			command := NewWorkCommand()
			command.SetOut(io.Discard)
			command.SetErr(io.Discard)
			command.SetArgs([]string{"--repo", "owner/repo", "replay", "--file", file})
			err = command.Execute()
			require.Error(t, err)
			require.NotContains(t, err.Error(), "secret")
		})
	}
}

func TestWorkQueueArtifactReplayZIPBoundsAndExactMember(t *testing.T) {
	for _, names := range [][]string{
		{"../" + workQueueSnapshotFile}, {"nested/" + workQueueSnapshotFile},
		{workQueueSnapshotFile, workQueueSnapshotFile},
	} {
		_, err := workQueueReplaySnapshotFromZIP(workQueueReplayTestZIP(t, []byte("{}"), names...))
		require.Error(t, err)
	}
	_, err := workQueueReplaySnapshotFromZIP([]byte("not a zip"))
	require.Error(t, err)
	_, err = readWorkQueueReplayBytes(strings.NewReader("oversized"), 4)
	require.Error(t, err)
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	header := &zip.FileHeader{Name: workQueueSnapshotFile}
	header.SetMode(os.ModeSymlink | 0o777)
	entry, err := writer.CreateHeader(header)
	require.NoError(t, err)
	_, err = entry.Write([]byte("private-target"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	_, err = workQueueReplaySnapshotFromZIP(buffer.Bytes())
	require.Error(t, err)
}
