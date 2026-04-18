//go:build !integration

package cli

import (
	"context"
	"fmt"
	"os/exec"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockExecWithStdoutAndStderr(stdoutText, stderrText string) execCmdFunc {
	return func(ctx context.Context, args ...string) *exec.Cmd {
		script := fmt.Sprintf("printf %q; printf %q 1>&2", stdoutText, stderrText)
		return exec.CommandContext(ctx, "sh", "-c", script)
	}
}

func extractTextResult(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	require.NotNil(t, result, "Tool result should not be nil")
	require.NotEmpty(t, result.Content, "Tool result should contain content")

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok, "Tool result content should be text")
	return textContent.Text
}

func TestCompileTool_UsesOnlyStdoutOnSuccess(t *testing.T) {
	const (
		expectedStdout = `[{"workflow":"test.md","valid":true,"errors":[],"warnings":[]}]`
		stderrNoise    = "diagnostic noise should not be returned"
	)

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	err := registerCompileTool(server, mockExecWithStdoutAndStderr(expectedStdout, stderrNoise), "")
	require.NoError(t, err, "registerCompileTool should succeed")

	session := connectInMemory(t, server)
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "compile",
		Arguments: map[string]any{},
	})
	require.NoError(t, err, "compile tool call should succeed")

	output := extractTextResult(t, result)
	assert.JSONEq(t, expectedStdout, output, "compile tool should return subprocess stdout only")
	assert.NotContains(t, output, stderrNoise, "compile tool output should not contain stderr noise")
}

func TestCommandBackedTools_UseOnlyStdoutOnSuccess(t *testing.T) {
	tests := []struct {
		name         string
		toolName     string
		args         map[string]any
		expectedOut  string
		stderrNoise  string
		registerTool func(server *mcp.Server)
	}{
		{
			name:     "add",
			toolName: "add",
			args: map[string]any{
				"workflows": []string{"owner/repo/workflow"},
			},
			expectedOut: "add-stdout",
			stderrNoise: "add-stderr",
			registerTool: func(server *mcp.Server) {
				registerAddTool(server, mockExecWithStdoutAndStderr("add-stdout", "add-stderr"))
			},
		},
		{
			name:        "update",
			toolName:    "update",
			args:        map[string]any{},
			expectedOut: "update-stdout",
			stderrNoise: "update-stderr",
			registerTool: func(server *mcp.Server) {
				registerUpdateTool(server, mockExecWithStdoutAndStderr("update-stdout", "update-stderr"))
			},
		},
		{
			name:     "fix",
			toolName: "fix",
			args: map[string]any{
				"list_codemods": true,
			},
			expectedOut: "fix-stdout",
			stderrNoise: "fix-stderr",
			registerTool: func(server *mcp.Server) {
				registerFixTool(server, mockExecWithStdoutAndStderr("fix-stdout", "fix-stderr"))
			},
		},
		{
			name:        "mcp-inspect",
			toolName:    "mcp-inspect",
			args:        map[string]any{},
			expectedOut: "inspect-stdout",
			stderrNoise: "inspect-stderr",
			registerTool: func(server *mcp.Server) {
				registerMCPInspectTool(server, mockExecWithStdoutAndStderr("inspect-stdout", "inspect-stderr"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
			tt.registerTool(server)

			session := connectInMemory(t, server)
			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      tt.toolName,
				Arguments: tt.args,
			})
			require.NoError(t, err, "%s tool call should succeed", tt.toolName)

			output := extractTextResult(t, result)
			assert.Equal(t, tt.expectedOut, output, "%s tool should return subprocess stdout only", tt.toolName)
			assert.NotContains(t, output, tt.stderrNoise, "%s tool output should not contain stderr noise", tt.toolName)
		})
	}
}
