//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCodexWebFetchMCPServer verifies that the Codex engine exposes web-fetch as a
// containerised MCP server in both the TOML (preliminary) and JSON (gateway) configs.
// Other engines use web-fetch as a native built-in tool and must NOT include this server.
func TestCodexWebFetchMCPServer(t *testing.T) {
	tempDir := t.TempDir()
	workflowPath := filepath.Join(tempDir, "test-workflow.md")

	workflowContent := `---
on: workflow_dispatch
engine: codex
tools:
  web-fetch: null
---
Use the web-fetch MCP tool to fetch a URL.
`
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0o600)
	require.NoError(t, err, "Should write workflow file")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(workflowPath)
	require.NoError(t, err, "Should compile successfully")

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	lockBytes, err := os.ReadFile(lockPath)
	require.NoError(t, err, "Should read lock file")
	result := string(lockBytes)

	// TOML preliminary config: the [mcp_servers.web-fetch] section must appear
	assert.Contains(t, result, "[mcp_servers.web-fetch]", "Preliminary TOML must include web-fetch section")
	assert.Contains(t, result, `container = "node:lts-alpine"`, "web-fetch section must use node:lts-alpine container")
	assert.Contains(t, result, "web_fetch_server.cjs", "web-fetch section must reference web_fetch_server.cjs")

	// JSON gateway config: "web-fetch" server must appear with the same container
	assert.Contains(t, result, `"web-fetch": {`, "Gateway JSON must include web-fetch server entry")
	assert.Contains(t, result, `"node:lts-alpine"`, "Gateway JSON web-fetch entry must use node:lts-alpine")
}

// TestNonCodexEnginesDoNotGetWebFetchMCPServer verifies that Claude and Copilot engines,
// which provide web-fetch as a native built-in, do NOT include a web-fetch MCP server.
func TestNonCodexEnginesDoNotGetWebFetchMCPServer(t *testing.T) {
	for _, engine := range []string{"claude", "copilot"} {
		t.Run(engine, func(t *testing.T) {
			tempDir := t.TempDir()
			workflowPath := filepath.Join(tempDir, "test-workflow.md")

			workflowContent := `---
on: workflow_dispatch
engine: ` + engine + `
tools:
  web-fetch: null
---
Use the web-fetch tool.
`
			err := os.WriteFile(workflowPath, []byte(workflowContent), 0o600)
			require.NoError(t, err, "Should write workflow file")

			compiler := NewCompiler()
			err = compiler.CompileWorkflow(workflowPath)
			require.NoError(t, err, "Should compile successfully")

			lockPath := stringutil.MarkdownToLockFile(workflowPath)
			lockBytes, err := os.ReadFile(lockPath)
			require.NoError(t, err, "Should read lock file")
			result := string(lockBytes)

			// These engines must NOT include a containerised web-fetch MCP server
			assert.False(t,
				strings.Contains(result, `"web-fetch": {`),
				"Engine %q must not add a web-fetch MCP server entry", engine,
			)
			assert.False(t,
				strings.Contains(result, "web_fetch_server.cjs"),
				"Engine %q must not reference web_fetch_server.cjs", engine,
			)
		})
	}
}
