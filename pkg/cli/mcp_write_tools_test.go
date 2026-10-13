//go:build !integration

package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPWriteTools_DefaultServerDenies(t *testing.T) {
	t.Parallel()
	server := createMCPServer("must-not-run", "operator", false, false, "", nil)
	session := connectInMemory(t, server)
	for _, tool := range []string{"add", "update", "fix", "compile"} {
		_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: tool, Arguments: map[string]any{
				"workflows": []string{"owner/repo/workflow"}, "allow_write_tools": true,
			},
		})
		require.ErrorContains(t, err, "workflow-writing tools disabled")
	}
}

func TestMCPWriteTools_HTTPServerDenies(t *testing.T) {
	t.Parallel()
	server := createMCPServer("must-not-run", "operator", false, false, "", nil)
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, nil)
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: httpServer.URL, DisableStandaloneSSE: true,
	}, nil)
	require.NoError(t, err)
	defer session.Close()
	for _, tool := range []string{"add", "update", "fix", "compile"} {
		_, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: tool, Arguments: map[string]any{"workflows": []string{"owner/repo/workflow"}},
		})
		require.ErrorContains(t, err, "workflow-writing tools disabled")
	}
}

func TestMCPWriteTools_StartupCapabilityAndActorChecks(t *testing.T) {
	oldCache := mcpCache
	mcpCache = newMCPCacheStore()
	t.Cleanup(func() { mcpCache = oldCache })
	mcpCache.SetRepo("owner/repo")
	mcpCache.SetPermission("writer", "owner/repo", "write")
	mcpCache.SetPermission("reader", "owner/repo", "read")

	for _, tc := range []struct {
		name          string
		allowWrite    bool
		actor         string
		validateActor bool
		wantError     string
	}{
		{name: "default", wantError: "workflow-writing tools disabled"},
		{name: "write role cannot enable tools", actor: "writer", validateActor: true, wantError: "workflow-writing tools disabled"},
		{name: "operator opt-in", allowWrite: true},
		{name: "missing actor", allowWrite: true, validateActor: true, wantError: "permission denied"},
		{name: "read role", allowWrite: true, actor: "reader", validateActor: true, wantError: "permission denied"},
		{name: "write role and opt-in", allowWrite: true, actor: "writer", validateActor: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var invocations atomic.Int32
			execCmd := func(ctx context.Context, args ...string) *exec.Cmd {
				invocations.Add(1)
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMCPWriteToolsHelper$")
				cmd.Env = append(os.Environ(), "GH_AW_MCP_WRITE_HELPER=1")
				return cmd
			}
			server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, nil)
			server.AddReceivingMiddleware(mcpWriteToolsMiddleware(tc.allowWrite, tc.actor, tc.validateActor))
			require.NoError(t, registerMCPTools(server, execCmd, tc.actor, tc.validateActor, ""))
			session := connectInMemory(t, server)

			for _, tool := range []string{"add", "update", "fix", "compile"} {
				t.Run(tool, func(t *testing.T) {
					before := invocations.Load()
					result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
						Name: tool, Arguments: map[string]any{"workflows": []string{"owner/repo/workflow"}},
					})
					if tc.wantError != "" {
						require.ErrorContains(t, err, tc.wantError)
						assert.Equal(t, before, invocations.Load(), "denial must precede subprocess creation")
					} else {
						require.NoError(t, err)
						require.NotNil(t, result)
						assert.False(t, result.IsError)
						assert.Equal(t, before+1, invocations.Load())
					}
				})
			}
		})
	}
}

func TestMCPWriteToolsHelper(t *testing.T) {
	if os.Getenv("GH_AW_MCP_WRITE_HELPER") != "1" {
		return
	}
	fmt.Print(`[{"workflow":"test.md","valid":true,"errors":[],"warnings":[]}]`)
	os.Exit(0)
}

func TestMCPInspect_ConfigurationOnlyArguments(t *testing.T) {
	t.Parallel()
	var gotArgs []string
	execCmd := func(ctx context.Context, args ...string) *exec.Cmd {
		gotArgs = append([]string(nil), args...)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMCPWriteToolsHelper$")
		cmd.Env = append(os.Environ(), "GH_AW_MCP_WRITE_HELPER=1")
		return cmd
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	registerMCPInspectTool(server, execCmd)
	session := connectInMemory(t, server)
	for _, workflow := range []string{"", "test", "--inspector"} {
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "mcp-inspect", Arguments: map[string]any{"workflow_file": workflow},
		})
		require.NoError(t, err)
		assert.False(t, result.IsError)
		wantArgs := []string{"mcp", "list", "--verbose", "--"}
		if workflow != "" {
			wantArgs = append(wantArgs, workflow)
		}
		assert.Equal(t, wantArgs, gotArgs)
	}
	for _, filter := range []string{"server", "tool"} {
		gotArgs = nil
		_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "mcp-inspect", Arguments: map[string]any{filter: "test"},
		})
		require.ErrorContains(t, err, "live MCP discovery is not available")
		assert.Nil(t, gotArgs, "unsupported live discovery must not execute a command")
	}
}

func TestMCPServerCommand_WriteToolsRequireOptIn(t *testing.T) {
	t.Parallel()
	flag := NewMCPServerCommand().Flags().Lookup("allow-write-tools")
	require.NotNil(t, flag)
	assert.Equal(t, "false", flag.DefValue)
}
