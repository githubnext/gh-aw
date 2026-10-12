//go:build integration

package workflow

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNVXStagedMuslCopilotInvokesAPIProxy(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker is required")
	}
	require.NoError(t, exec.Command("docker", "version").Run())
	if runtime.GOOS == "linux" {
		if _, err := os.Stat("/usr/bin/patchelf"); err != nil {
			t.Skip("/usr/bin/patchelf is required to stage the portable Node runtime")
		}
	}

	temp := t.TempDir()
	workspace := filepath.Join(temp, "workspace")
	runnerTemp := filepath.Join(temp, "runner")
	hostRoot := filepath.Join(runnerTemp, "gh-aw")
	stateRoot := filepath.Join(temp, "state")
	require.NoError(t, os.MkdirAll(filepath.Join(hostRoot, "actions"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(hostRoot, "nvx-engine"), 0o755))
	require.NoError(t, os.MkdirAll(workspace, 0o755))
	require.NoError(t, os.MkdirAll(stateRoot, 0o755))
	require.NoError(t, copyFile(
		filepath.Join("..", "..", "actions", "setup", "js", "nvx_assets.cjs"),
		filepath.Join(hostRoot, "actions", "nvx_assets.cjs"),
		0o644,
	))

	version := strings.TrimPrefix(string(constants.DefaultCopilotVersion), "v")
	archiveURL := fmt.Sprintf("https://github.com/github/copilot-cli/releases/download/v%s/copilot-linuxmusl-x64.tar.gz", version)
	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Get(archiveURL)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NoError(t, extractCopilotBinary(response.Body, filepath.Join(hostRoot, "nvx-engine", "copilot")))

	stage := exec.Command("node", filepath.Join(hostRoot, "actions", "nvx_assets.cjs"), "stage")
	stage.Env = append(os.Environ(), "GITHUB_WORKSPACE="+workspace, "RUNNER_TEMP="+runnerTemp)
	output, err := stage.CombinedOutput()
	require.NoError(t, err, "%s", output)

	requests := make(chan map[string]any, 1)
	serverErrors := make(chan error, 1)
	listener, err := net.Listen("tcp4", "0.0.0.0:0")
	require.NoError(t, err)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(request.URL.Path, "/models") {
			_, _ = io.WriteString(w, `{"object":"list","data":[{"id":"test-model","object":"model","owned_by":"test"}]}`)
			return
		}
		if strings.HasSuffix(request.URL.Path, "/chat/completions") {
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				serverErrors <- err
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			requests <- body
			_, _ = io.WriteString(w, `{"id":"chatcmpl-test","object":"chat.completion","created":1,"model":"test-model","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":1,"total_tokens":6}}`)
			return
		}
		http.NotFound(w, request)
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	})

	port := listener.Addr().(*net.TCPAddr).Port
	stagedCopilot := "/workspace/.gh-aw-nvx/gh-aw/bin/copilot"
	run := exec.Command(
		"docker", "run", "--rm", "--platform", "linux/amd64",
		"--add-host", "host.docker.internal:host-gateway",
		"-v", workspace+":/workspace",
		"-w", "/workspace",
		"-e", "HOME=/tmp/awf-home",
		"-e", fmt.Sprintf("COPILOT_PROVIDER_BASE_URL=http://host.docker.internal:%d/v1", port),
		"-e", "COPILOT_PROVIDER_API_KEY=test-only",
		"-e", "COPILOT_MODEL=test-model",
		"alpine:3.22",
		stagedCopilot, "-p", "Reply with ok", "--allow-all-tools", "--no-ask-user", "--silent",
	)
	output, err = run.CombinedOutput()
	require.NoError(t, err, "%s", output)
	assert.Contains(t, string(output), "ok")

	select {
	case request := <-requests:
		assert.Equal(t, "test-model", request["model"])
		assert.NotEmpty(t, request["messages"])
	case err := <-serverErrors:
		t.Fatalf("failed to decode API proxy request: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("staged musl Copilot did not invoke the API proxy")
	}
}

func copyFile(source, destination string, mode os.FileMode) error {
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(destination, content, mode)
}

func extractCopilotBinary(reader io.Reader, destination string) error {
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return fmt.Errorf("copilot binary not found in archive")
		}
		if err != nil {
			return err
		}
		if header.Name != "copilot" || header.Typeflag != tar.TypeReg {
			continue
		}
		file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(file, tarReader)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}
}
