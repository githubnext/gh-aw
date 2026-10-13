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
	require.NoError(t, registerLogsTool(server, execCmd, "", false))
	require.NoError(t, registerAuditTool(server, execCmd, "", false))
	require.NoError(t, registerAuditDiffTool(server, execCmd, "", false))
	return connectInMemory(t, server), calls
}

func TestMCPPositionalArgsRejectFlags(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "")
	session, calls := positionalArgsTestSession(t)
	for _, tool := range []struct {
		name  string
		field string
		args  func(string, bool) map[string]any
	}{
		{"add", "workflows", func(value string, after bool) map[string]any {
			items := []string{value, "owner/repo/workflow"}
			if after {
				items = []string{"owner/repo/workflow", value}
			}
			return map[string]any{"workflows": items, "number": 2, "name": "copy"}
		}},
		{"update", "workflows", func(value string, after bool) map[string]any {
			items := []string{value, "workflow"}
			if after {
				items = []string{"workflow", value}
			}
			return map[string]any{"workflows": items, "major": true, "force": true}
		}},
		{"fix", "workflows", func(value string, after bool) map[string]any {
			items := []string{value, "workflow"}
			if after {
				items = []string{"workflow", value}
			}
			return map[string]any{"workflows": items}
		}},
		{"compile", "workflows", func(value string, after bool) map[string]any {
			items := []string{value, "workflow"}
			if after {
				items = []string{"workflow", value}
			}
			return map[string]any{"workflows": items}
		}},
		{"mcp-inspect", "workflow_file", func(value string, _ bool) map[string]any {
			return map[string]any{"workflow_file": value}
		}},
		{"logs", "workflow_name", func(value string, _ bool) map[string]any {
			return map[string]any{"workflow_name": value}
		}},
		{"audit", "run_ids_or_urls", func(value string, after bool) map[string]any {
			items := []string{value, "123"}
			if after {
				items = []string{"123", value}
			}
			return map[string]any{"run_ids_or_urls": items}
		}},
		{"audit", "run_id", func(value string, _ bool) map[string]any {
			return map[string]any{"run_id": value}
		}},
		{"audit", "run_id_or_url", func(value string, _ bool) map[string]any {
			return map[string]any{"run_id_or_url": value}
		}},
		{"audit-diff", "base_run_id", func(value string, _ bool) map[string]any {
			return map[string]any{"base_run_id": value, "compare_run_ids": []string{"456"}}
		}},
		{"audit-diff", "compare_run_ids", func(value string, after bool) map[string]any {
			items := []string{value, "456"}
			if after {
				items = []string{"456", value}
			}
			return map[string]any{"base_run_id": "123", "compare_run_ids": items}
		}},
	} {
		t.Run(tool.name+"/"+tool.field, func(t *testing.T) {
			for _, value := range []string{"-", "--", "-n", "--force", "--disable-security-scanner", "--create-pull-request", "--repo=owner/private", "--approve", "--dir=../outside", "--write", "--inspector"} {
				for _, after := range []bool{false, true} {
					t.Run(value, func(t *testing.T) {
						before := calls.Load()
						_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
							Name: tool.name, Arguments: tool.args(value, after),
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
		})
	}

	t.Run("reported add payload", func(t *testing.T) {
		before := calls.Load()
		_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "add",
			Arguments: map[string]any{"workflows": []string{
				"--force", "--disable-security-scanner", "--create-pull-request",
				"owner/repo/workflow",
			}},
		})
		require.Error(t, err)
		assert.Equal(t, before, calls.Load())
	})
}

func TestMCPPositionalArgsRejectFlagsWithRepository(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	session, calls := positionalArgsTestSession(t)
	for _, tt := range []struct {
		name string
		args map[string]any
	}{
		{"logs", map[string]any{"workflow_name": "--repo=owner/private"}},
		{"audit", map[string]any{"run_ids_or_urls": []string{"123", "--repo=owner/private"}}},
		{"audit", map[string]any{"run_id": -123}},
		{"audit", map[string]any{"run_id_or_url": -123}},
		{"audit-diff", map[string]any{"base_run_id": "123", "compare_run_ids": []string{"--repo=owner/private"}}},
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
