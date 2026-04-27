//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractLastConsoleMessage verifies that extractLastConsoleMessage correctly
// filters debug log lines and returns only user-facing console messages.
func TestExtractLastConsoleMessage(t *testing.T) {
	tests := []struct {
		name     string
		stderr   string
		expected string
	}{
		{
			name: "filters debug logs and returns error message",
			stderr: `workflow:script_registry Creating new script registry +151ns
workflow:domains Loading ecosystem domains from embedded JSON +760µs
workflow:domains Loaded 31 ecosystem categories +161µs
cli:audit Starting audit for workflow run: runID=99999999999 +916µs
cli:audit Using output directory: /tmp/gh-aw/aw-mcp/logs/run-99999999999 +14µs
✗ failed to fetch run metadata: workflow run 99999999999 not found. Please verify the run ID is correct`,
			expected: "✗ failed to fetch run metadata: workflow run 99999999999 not found. Please verify the run ID is correct",
		},
		{
			name:     "empty stderr returns empty string",
			stderr:   "",
			expected: "",
		},
		{
			name:     "only whitespace returns empty string",
			stderr:   "   \n\n  ",
			expected: "",
		},
		{
			name:     "only debug logs falls back to last non-empty line",
			stderr:   "workflow:foo Starting +100ns\ncli:bar Processing +200µs",
			expected: "cli:bar Processing +200µs",
		},
		{
			name:     "console error message with no debug logs",
			stderr:   "✗ some error occurred",
			expected: "✗ some error occurred",
		},
		{
			name:     "console success message",
			stderr:   "✓ operation completed",
			expected: "✓ operation completed",
		},
		{
			name:     "console info message",
			stderr:   "ℹ loading configuration",
			expected: "ℹ loading configuration",
		},
		{
			name:     "console warning message",
			stderr:   "⚠ deprecated option",
			expected: "⚠ deprecated option",
		},
		{
			name: "multiple console messages returns last one",
			stderr: `ℹ starting up
✗ first error
✗ second error`,
			expected: "✗ second error",
		},
		{
			name: "debug logs after console message are skipped (last console returned)",
			stderr: `✗ some error
workflow:foo Cleanup +50ms`,
			expected: "✗ some error",
		},
		{
			name: "authentication error from logs command (GitHub Actions context)",
			stderr: `cli:logs_orchestrator Starting workflow log download: workflow=, count=100
ℹ Fetching workflow runs from GitHub Actions...
✗ GitHub CLI authentication required. Run 'gh auth login' first`,
			expected: "✗ GitHub CLI authentication required. Run 'gh auth login' first",
		},
		{
			name:     "cobra error format without console symbols",
			stderr:   "Error: GitHub CLI authentication required. Run 'gh auth login' first",
			expected: "Error: GitHub CLI authentication required. Run 'gh auth login' first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractLastConsoleMessage(tt.stderr)
			assert.Equal(t, tt.expected, result, "should extract correct message from stderr")
		})
	}
}

// connectInMemory creates an in-memory MCP client-server connection for testing.
// The session is closed automatically when the test ends via t.Cleanup.
func connectInMemory(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	t1, t2 := mcp.NewInMemoryTransports()
	_, err := server.Connect(ctx, t1, nil)
	require.NoError(t, err, "server.Connect should succeed")
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	require.NoError(t, err, "client.Connect should succeed")
	t.Cleanup(func() { session.Close() })
	return session
}

// TestLogsToolPassesGithubRepositoryAsRepoFlag verifies that the logs MCP tool
// appends --repo <owner/repo> to the subprocess command when GITHUB_REPOSITORY
// is set, allowing gh run list to work in environments without git installed.
func TestLogsToolPassesGithubRepositoryAsRepoFlag(t *testing.T) {
	tests := []struct {
		name             string
		githubRepository string
		wantRepoFlag     bool
	}{
		{
			name:             "passes --repo when GITHUB_REPOSITORY is set",
			githubRepository: "github/gh-aw",
			wantRepoFlag:     true,
		},
		{
			name:             "omits --repo when GITHUB_REPOSITORY is empty",
			githubRepository: "",
			wantRepoFlag:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_REPOSITORY", tt.githubRepository)

			var capturedArgs []string
			mockExecCmd := func(ctx context.Context, args ...string) *exec.Cmd {
				capturedArgs = append([]string(nil), args...)
				// Use a non-existent command so the subprocess fails on all platforms
				// without depending on Unix-specific commands like "false".
				// cmd.Output() will return a "executable file not found" error, which
				// the handler treats as a failure — we only care about the captured args.
				return exec.CommandContext(ctx, "nonexistent-command-for-testing-only")
			}

			server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
			err := registerLogsTool(server, mockExecCmd, "", false)
			require.NoError(t, err, "registerLogsTool should succeed")

			session := connectInMemory(t, server)

			// Call the tool — it will fail because the mock command is not found,
			// but we only care about the captured args.
			ctx := context.Background()
			_, _ = session.CallTool(ctx, &mcp.CallToolParams{
				Name:      "logs",
				Arguments: map[string]any{},
			})

			require.NotNil(t, capturedArgs, "execCmd should have been called")

			// Locate --repo flag in captured args
			var repoValue string
			for i, arg := range capturedArgs {
				if arg == "--repo" && i+1 < len(capturedArgs) {
					repoValue = capturedArgs[i+1]
					break
				}
			}

			if tt.wantRepoFlag {
				assert.Equal(t, tt.githubRepository, repoValue,
					"--repo flag should be set to GITHUB_REPOSITORY value; args: %v", capturedArgs)
			} else {
				assert.Empty(t, repoValue,
					"--repo flag should not be present when GITHUB_REPOSITORY is empty; args: %v", capturedArgs)
			}
		})
	}
}

// TestAuditToolPassesGithubRepositoryAsRepoFlag verifies that the audit MCP tool
// appends --repo <owner/repo> to the subprocess command when GITHUB_REPOSITORY
// is set, allowing the audit command to resolve the repository without git.
func TestAuditToolPassesGithubRepositoryAsRepoFlag(t *testing.T) {
	tests := []struct {
		name             string
		githubRepository string
		wantRepoFlag     bool
	}{
		{
			name:             "passes --repo when GITHUB_REPOSITORY is set",
			githubRepository: "github/gh-aw",
			wantRepoFlag:     true,
		},
		{
			name:             "omits --repo when GITHUB_REPOSITORY is empty",
			githubRepository: "",
			wantRepoFlag:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_REPOSITORY", tt.githubRepository)

			var capturedArgs []string
			mockExecCmd := func(ctx context.Context, args ...string) *exec.Cmd {
				capturedArgs = append([]string(nil), args...)
				// Use a non-existent command so the subprocess fails on all platforms
				// without depending on Unix-specific commands like "false".
				return exec.CommandContext(ctx, "nonexistent-command-for-testing-only")
			}

			server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
			err := registerAuditTool(server, mockExecCmd, "", false)
			require.NoError(t, err, "registerAuditTool should succeed")

			session := connectInMemory(t, server)

			ctx := context.Background()
			_, _ = session.CallTool(ctx, &mcp.CallToolParams{
				Name:      "audit",
				Arguments: map[string]any{"run_id_or_url": "1234567890"},
			})

			require.NotNil(t, capturedArgs, "execCmd should have been called")

			var repoValue string
			for i, arg := range capturedArgs {
				if arg == "--repo" && i+1 < len(capturedArgs) {
					repoValue = capturedArgs[i+1]
					break
				}
			}

			if tt.wantRepoFlag {
				assert.Equal(t, tt.githubRepository, repoValue,
					"--repo flag should be set to GITHUB_REPOSITORY value; args: %v", capturedArgs)
			} else {
				assert.Empty(t, repoValue,
					"--repo flag should not be present when GITHUB_REPOSITORY is empty; args: %v", capturedArgs)
			}
		})
	}
}

// TestAuditToolErrorEnvelopeSetsIsErrorFalse verifies that audit command failures
// returned as JSON envelopes use IsError=false so callers receive graceful JSON
// rather than a fatal MCP protocol error.
func TestAuditToolErrorEnvelopeSetsIsErrorFalse(t *testing.T) {
	mockExecCmd := func(ctx context.Context, args ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestAuditToolErrorEnvelopeHelperProcess")
		cmd.Env = append(os.Environ(), "GH_AW_AUDIT_HELPER_PROCESS=1")
		return cmd
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	err := registerAuditTool(server, mockExecCmd, "", false)
	require.NoError(t, err, "registerAuditTool should succeed")

	session := connectInMemory(t, server)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "audit",
		Arguments: map[string]any{"run_id_or_url": "9999999999"},
	})
	require.NoError(t, err, "audit tool should return result envelope without protocol error")
	require.NotNil(t, result, "result should not be nil")
	assert.False(t, result.IsError, "audit error envelope should set IsError=false (graceful JSON error)")
	require.NotEmpty(t, result.Content, "result should contain text content")

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok, "expected text content in audit error response")

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(textContent.Text), &envelope), "error response should be valid JSON")
	runIDsRaw, hasRunIDs := envelope["run_ids_or_urls"]
	require.True(t, hasRunIDs, "error envelope should include run_ids_or_urls field")
	runIDs, ok := runIDsRaw.([]any)
	require.True(t, ok, "run_ids_or_urls should be an array")
	require.Len(t, runIDs, 1, "run_ids_or_urls should contain the single run ID")
	assert.Equal(t, "9999999999", runIDs[0], "error envelope should include original run ID")
	errorMessage, ok := envelope["error"].(string)
	require.True(t, ok, "error envelope should include string error field")
	assert.Contains(t, errorMessage, "failed to audit workflow run", "error envelope should include contextual prefix")
	suggestions, hasSuggestions := envelope["suggestions"]
	assert.True(t, hasSuggestions, "error envelope should include suggestions")
	assert.NotEmpty(t, suggestions, "suggestions should not be empty")
}

func TestAuditToolErrorEnvelopeHelperProcess(t *testing.T) {
	if os.Getenv("GH_AW_AUDIT_HELPER_PROCESS") != "1" {
		return
	}

	_, _ = fmt.Fprintln(os.Stderr, "✗ failed to fetch run metadata")
	os.Exit(1)
}

func TestAuditTool_AcceptsDeprecatedMaxTokensParameter(t *testing.T) {
	const expectedStdout = `{"overview":{"run_id":"1234567890"}}`

	var capturedArgs []string
	mockExecCmd := func(ctx context.Context, args ...string) *exec.Cmd {
		capturedArgs = slices.Clone(args)
		return exec.CommandContext(ctx, "sh", "-c", `printf '%s' "$1"`, "sh", expectedStdout)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	err := registerAuditTool(server, mockExecCmd, "", false)
	require.NoError(t, err, "registerAuditTool should succeed")

	session := connectInMemory(t, server)
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "audit",
		Arguments: map[string]any{
			"run_id_or_url": "1234567890",
			"max_tokens":    5000,
		},
	})
	require.NoError(t, err, "audit tool should accept deprecated max_tokens parameter")
	require.NotNil(t, result, "result should not be nil")

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok, "expected text content in audit response")
	assert.JSONEq(t, expectedStdout, textContent.Text, "audit tool should return subprocess stdout")
	assert.NotContains(t, strings.Join(capturedArgs, " "), "max_tokens", "audit command args should ignore max_tokens")
}

// TestAuditTool_MultiRunDiffMode verifies that when run_ids_or_urls contains
// multiple entries the audit tool passes all of them as positional arguments
// to the audit command (which then runs in diff mode).
func TestAuditTool_MultiRunDiffMode(t *testing.T) {
	const expectedStdout = `[{"base_run_id":111,"compare_run_id":222}]`

	var capturedArgs []string
	mockExecCmd := func(ctx context.Context, args ...string) *exec.Cmd {
		capturedArgs = slices.Clone(args)
		return exec.CommandContext(ctx, "sh", "-c", `printf '%s' "$1"`, "sh", expectedStdout)
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	err := registerAuditTool(server, mockExecCmd, "", false)
	require.NoError(t, err, "registerAuditTool should succeed")

	session := connectInMemory(t, server)
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "audit",
		Arguments: map[string]any{
			"run_ids_or_urls": []string{"111", "222", "333"},
		},
	})
	require.NoError(t, err, "audit tool should succeed with multiple run IDs")
	require.NotNil(t, result, "result should not be nil")
	assert.False(t, result.IsError, "result should not be an error")

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok, "expected text content in audit response")
	assert.JSONEq(t, expectedStdout, textContent.Text, "audit tool should return subprocess stdout")

	// All three run IDs must appear as positional args immediately after "audit"
	require.GreaterOrEqual(t, len(capturedArgs), 4, "captured args should include audit + 3 run IDs: %v", capturedArgs)
	assert.Equal(t, "audit", capturedArgs[0], "first arg should be 'audit'")
	assert.Equal(t, "111", capturedArgs[1], "second arg should be first run ID")
	assert.Equal(t, "222", capturedArgs[2], "third arg should be second run ID")
	assert.Equal(t, "333", capturedArgs[3], "fourth arg should be third run ID")
}

// TestAuditTool_FailsWhenNoRunIDProvided verifies that the audit tool
// returns an error when neither run_id_or_url nor run_ids_or_urls is provided.
func TestAuditTool_FailsWhenNoRunIDProvided(t *testing.T) {
	mockExecCmd := func(ctx context.Context, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "nonexistent-command-for-testing-only")
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	err := registerAuditTool(server, mockExecCmd, "", false)
	require.NoError(t, err, "registerAuditTool should succeed")

	session := connectInMemory(t, server)
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "audit",
		Arguments: map[string]any{},
	})
	// The MCP SDK surfaces InvalidParams as a protocol-level error
	assert.True(t, err != nil || (result != nil && result.IsError),
		"audit tool should return an error when no run ID is provided")
}

func TestAuditDiffToolErrorEnvelopeSetsIsErrorFalse(t *testing.T) {
	mockExecCmd := func(ctx context.Context, args ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestAuditDiffToolErrorEnvelopeHelperProcess")
		cmd.Env = append(os.Environ(), "GH_AW_AUDIT_DIFF_HELPER_PROCESS=1")
		return cmd
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	err := registerAuditDiffTool(server, mockExecCmd, "", false)
	require.NoError(t, err, "registerAuditDiffTool should succeed")

	session := connectInMemory(t, server)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "audit-diff",
		Arguments: map[string]any{
			"base_run_id":     "100",
			"compare_run_ids": []string{"200"},
		},
	})
	require.NoError(t, err, "audit-diff tool should return result envelope without protocol error")
	require.NotNil(t, result, "result should not be nil")
	assert.False(t, result.IsError, "audit-diff error envelope should set IsError=false (graceful JSON error)")
	require.NotEmpty(t, result.Content, "result should contain text content")

	textContent, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok, "expected text content in audit-diff error response")

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(textContent.Text), &envelope), "error response should be valid JSON")
	assert.Equal(t, "100", envelope["base_run_id"], "error envelope should include base run ID")
	errorMessage, ok := envelope["error"].(string)
	require.True(t, ok, "error envelope should include string error field")
	assert.Contains(t, errorMessage, "failed to diff workflow runs", "error envelope should include contextual prefix")
	suggestions, hasSuggestions := envelope["suggestions"]
	assert.True(t, hasSuggestions, "error envelope should include suggestions")
	assert.NotEmpty(t, suggestions, "suggestions should not be empty")
}

func TestAuditDiffToolErrorEnvelopeHelperProcess(t *testing.T) {
	if os.Getenv("GH_AW_AUDIT_DIFF_HELPER_PROCESS") != "1" {
		return
	}

	_, _ = fmt.Fprintln(os.Stderr, "✗ failed to diff workflow runs")
	os.Exit(1)
}
