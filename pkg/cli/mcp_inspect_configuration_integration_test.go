//go:build integration

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPInspect_ConfigurationOnlyPreventsExecution(t *testing.T) {
	binary := testutil.RequireGhAwBinary(t)
	dir := t.TempDir()
	workflowsDir := filepath.Join(dir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0o755))
	marker := filepath.Join(dir, "executed")
	var requests atomic.Int32
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer httpServer.Close()

	quotedMarker, err := json.Marshal(marker)
	require.NoError(t, err)
	content := fmt.Sprintf(`---
on: workflow_dispatch
mcp-servers:
  shell:
    type: stdio
    command: /bin/sh
    args: ["-c", "touch \"$1\"", "sh", %s]
  package:
    type: stdio
    command: npx
    args: ["untrusted-package"]
  container:
    type: stdio
    container: untrusted-image
  endpoint:
    type: http
    url: %s
---
# Configuration test
`, quotedMarker, httpServer.URL)
	workflowPath := filepath.Join(workflowsDir, "untrusted.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(content), 0o600))
	require.NoError(t, initTestGitRepo(dir))
	for _, launcher := range []string{"npx", "docker"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, launcher), []byte("#!/bin/sh\n: > \"$GH_AW_INSPECT_MARKER\"\n"), 0o700))
	}

	for _, allowWrite := range []bool{false, true} {
		t.Run(fmt.Sprintf("allow-write=%t", allowWrite), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			args := []string{"mcp-server", "--cmd", binary}
			if allowWrite {
				args = append(args, "--allow-write-tools")
			}
			cmd := exec.CommandContext(ctx, binary, args...)
			cmd.Dir = dir
			cmd.Env = withNonInteractiveCIEnv(nil)
			cmd.Env = append(cmd.Env, "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"), "GH_AW_INSPECT_MARKER="+marker)
			client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil)
			session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
			require.NoError(t, err)
			defer session.Close()

			if !allowWrite {
				for _, tool := range []string{"add", "update", "fix", "compile"} {
					_, err := session.CallTool(ctx, &mcp.CallToolParams{
						Name: tool, Arguments: map[string]any{"workflows": []string{"attacker/evil/payload"}},
					})
					require.ErrorContains(t, err, "workflow-writing tools disabled")
				}
			}
			for _, workflow := range []string{"", "untrusted"} {
				result, err := session.CallTool(ctx, &mcp.CallToolParams{
					Name: "mcp-inspect", Arguments: map[string]any{"workflow_file": workflow},
				})
				require.NoError(t, err)
				require.False(t, result.IsError)
				require.NotEmpty(t, result.Content)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "untrusted")
				if workflow != "" {
					assert.Contains(t, text.Text, "shell")
					assert.Contains(t, text.Text, "package")
					assert.Contains(t, text.Text, "container")
					assert.Contains(t, text.Text, "endpoint")
				}
			}
			_, err = os.Stat(marker)
			require.ErrorIs(t, err, os.ErrNotExist, "inspection must never run the configured command")
			assert.Zero(t, requests.Load(), "inspection must never connect to configured HTTP endpoints")
			got, err := os.ReadFile(workflowPath)
			require.NoError(t, err)
			assert.Equal(t, content, string(got))
			entries, err := os.ReadDir(workflowsDir)
			require.NoError(t, err)
			assert.Len(t, entries, 1, "denied management calls must not plant or compile workflows")
		})
	}
}
