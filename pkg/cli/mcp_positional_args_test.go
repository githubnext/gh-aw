//go:build !integration

package cli

import (
	"context"
	"os/exec"
	"sync/atomic"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func positionalArgsTestSession(t *testing.T) (*mcp.ClientSession, *atomic.Int32) {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "positional-args-test", Version: "1"}, nil)
	calls := new(atomic.Int32)
	execCmd := func(ctx context.Context, args ...string) *exec.Cmd {
		calls.Add(1)
		return exec.CommandContext(ctx, "echo", "[]")
	}
	registerAddTool(server, execCmd)
	registerUpdateTool(server, execCmd)
	registerFixTool(server, execCmd)
	require.NoError(t, registerCompileTool(server, execCmd, "/cached/manifest.json"))
	registerMCPInspectTool(server, execCmd)
	require.NoError(t, registerAuditTool(server, execCmd, "", false))
	require.NoError(t, registerAuditDiffTool(server, execCmd, "", false))
	return connectInMemory(t, server), calls
}

func TestValidateMCPPositionalArgs(t *testing.T) {
	for _, tt := range []struct {
		name      string
		values    []string
		wantError bool
	}{
		{"no values", nil, false},
		{"empty value", []string{""}, false},
		{"valid values", []string{"work-flow", "./-workflow.md", "123", "https://github.com/owner/repo/actions/runs/456"}, false},
		{"single dash", []string{"-"}, true},
		{"option boundary", []string{"--"}, true},
		{"short option", []string{"-n"}, true},
		{"long option", []string{"--force"}, true},
		{"option with value", []string{"--repo=owner/private"}, true},
		{"invalid first value", []string{"--write", "workflow"}, true},
		{"invalid later value", []string{"workflow", "--write"}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMCPPositionalArgs("workflows", tt.values...)
			if !tt.wantError {
				require.NoError(t, err)
				return
			}
			var rpcErr *jsonrpc.Error
			require.ErrorAs(t, err, &rpcErr)
			assert.Equal(t, int64(jsonrpc.CodeInvalidParams), rpcErr.Code)
			assert.Contains(t, rpcErr.Message, "workflows")
		})
	}
}

func TestMCPPositionalArgsRejectFlags(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "")
	session, calls := positionalArgsTestSession(t)
	for _, tool := range []struct {
		name  string
		field string
		args  map[string]any
	}{
		{"add", "workflows", map[string]any{"workflows": []string{"--force", "--disable-security-scanner", "--create-pull-request", "owner/repo/workflow"}}},
		{"update", "workflows", map[string]any{"workflows": []string{"workflow", "--force"}, "major": true, "force": true}},
		{"fix", "workflows", map[string]any{"workflows": []string{"--write", "workflow"}}},
		{"compile", "workflows", map[string]any{"workflows": []string{"workflow", "--approve"}}},
		{"mcp-inspect", "workflow_file", map[string]any{"workflow_file": "--inspector"}},
		{"audit", "run_ids_or_urls", map[string]any{"run_ids_or_urls": []string{"123", "--repo=owner/private"}}},
		{"audit", "run_id", map[string]any{"run_id": "--repo=owner/private"}},
		{"audit", "run_id_or_url", map[string]any{"run_id_or_url": "--repo=owner/private"}},
		{"audit-diff", "base_run_id", map[string]any{"base_run_id": "--repo=owner/private", "compare_run_ids": []string{"456"}}},
		{"audit-diff", "compare_run_ids", map[string]any{"base_run_id": "123", "compare_run_ids": []string{"456", "--repo=owner/private"}}},
	} {
		t.Run(tool.name+"/"+tool.field, func(t *testing.T) {
			before := calls.Load()
			_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
				Name: tool.name, Arguments: tool.args,
			})
			require.Error(t, err)
			var rpcErr *jsonrpc.Error
			require.ErrorAs(t, err, &rpcErr)
			assert.Equal(t, int64(jsonrpc.CodeInvalidParams), rpcErr.Code)
			assert.Contains(t, rpcErr.Message, tool.field)
			assert.Equal(t, before, calls.Load(), "invalid positionals must not launch a subprocess")
		})
	}
}

func TestMCPPositionalArgsRejectFlagsWithRepository(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	session, calls := positionalArgsTestSession(t)
	for _, tt := range []struct {
		name string
		args map[string]any
	}{
		{"audit", map[string]any{"run_id": -123}},
		{"audit", map[string]any{"run_id_or_url": -123}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tt.name, Arguments: tt.args})
			require.Error(t, err)
			var rpcErr *jsonrpc.Error
			require.ErrorAs(t, err, &rpcErr)
			assert.Equal(t, int64(jsonrpc.CodeInvalidParams), rpcErr.Code)
			assert.Zero(t, calls.Load(), "invalid positionals must not launch a subprocess")
		})
	}
}

func TestMCPPositionalArgsPreserveValidInputs(t *testing.T) {
	session, calls := positionalArgsTestSession(t)
	for _, tt := range []struct {
		name string
		args map[string]any
	}{
		{"add", map[string]any{"workflows": []string{"owner/repo/work-flow@v1", "./-workflow.md"}, "number": 2, "name": "copy"}},
		{"update", map[string]any{"workflows": []string{"work-flow"}, "major": true, "force": true}},
		{"update", map[string]any{}},
		{"fix", map[string]any{"workflows": []string{"work-flow"}, "write": true}},
		{"fix", map[string]any{"list_codemods": true}},
		{"compile", map[string]any{"workflows": []string{"work-flow", "./-workflow.md"}}},
		{"compile", map[string]any{}},
		{"mcp-inspect", map[string]any{"workflow_file": "./-workflow.md", "server": "github", "tool": "get_issue"}},
		{"mcp-inspect", map[string]any{}},
		{"audit", map[string]any{"run_ids_or_urls": []string{"123", "https://github.com/owner/repo/actions/runs/456"}}},
		{"audit", map[string]any{"run_id": 123}},
		{"audit", map[string]any{"run_id_or_url": "https://github.com/owner/repo/actions/runs/123/job/456#step:1:1"}},
		{"audit-diff", map[string]any{"base_run_id": "123", "compare_run_ids": []string{"456"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			before := calls.Load()
			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tt.name, Arguments: tt.args})
			require.NoError(t, err)
			assert.False(t, result.IsError)
			assert.Equal(t, before+1, calls.Load())
		})
	}
}
