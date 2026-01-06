package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSandboxMCPContainerConfiguration(t *testing.T) {
	// Test workflow with exact configuration from problem statement
	workflow := `---
name: Test Sandbox MCP Container
engine: copilot
on: workflow_dispatch
sandbox:
  agent: awf
  mcp:
    container: "ghcr.io/githubnext/gh-aw-mcpg:latest"
    args:
      - "--rm"
      - "-i"
      - "-v"
      - "/var/run/docker.sock:/var/run/docker.sock"
      - "-p"
      - "8000:8000"
      - "--entrypoint"
      - "/app/flowguard-go"
    entrypointArgs:
      - "--routed"
      - "--listen"
      - "0.0.0.0:8000"
      - "--config-stdin"
    port: 8000
    env:
      DOCKER_API_VERSION: "1.44"
      GITHUB_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
tools:
  github:
    mode: remote
    toolsets: [default]
permissions:
  issues: read
  pull-requests: read
---

Test workflow for sandbox MCP container configuration.
`

	tmpDir := testutil.TempDir(t, "sandbox-mcp-container-test")
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	err := os.WriteFile(testFile, []byte(workflow), 0644)
	require.NoError(t, err)

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	compiler.SetStrictMode(false)
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err)

	// Read the compiled lock file
	lockFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
	lockFileContent, err := os.ReadFile(lockFile)
	require.NoError(t, err)
	require.NotEmpty(t, lockFileContent)

	// Verify the compiled workflow contains the correct container configuration
	lockFileStr := string(lockFileContent)

	// Check container start command
	assert.Contains(t, lockFileStr, "Start MCP Gateway")
	assert.Contains(t, lockFileStr, "docker run --rm -i -v /var/run/docker.sock:/var/run/docker.sock -p 8000:8000 --entrypoint /app/flowguard-go")
	assert.Contains(t, lockFileStr, "ghcr.io/githubnext/gh-aw-mcpg:latest")
	assert.Contains(t, lockFileStr, "--routed --listen 0.0.0.0:8000 --config-stdin")

	// Check environment variables
	assert.Contains(t, lockFileStr, `DOCKER_API_VERSION="1.44"`)
	assert.Contains(t, lockFileStr, `GITHUB_TOKEN="${{ secrets.GITHUB_TOKEN }}"`)

	// Check health check uses correct port
	assert.Contains(t, lockFileStr, "Verify MCP Gateway Health")
	assert.Contains(t, lockFileStr, "http://localhost:8000")
	
	// Ensure we're NOT using the default port
	healthCheckLine := ""
	for _, line := range strings.Split(lockFileStr, "\n") {
		if strings.Contains(line, "Verify MCP Gateway Health") {
			// Find the next line with the health check URL
			idx := strings.Index(lockFileStr, line)
			remaining := lockFileStr[idx:]
			lines := strings.Split(remaining, "\n")
			for _, l := range lines[1:] {
				if strings.Contains(l, "http://localhost:") {
					healthCheckLine = l
					break
				}
			}
			break
		}
	}
	require.NotEmpty(t, healthCheckLine, "Health check line not found")
	assert.Contains(t, healthCheckLine, "http://localhost:8000", "Health check should use configured port 8000, not default 8080")
	assert.NotContains(t, healthCheckLine, "http://localhost:8080", "Health check should not use default port 8080")
}

func TestSandboxMCPCommandConfiguration(t *testing.T) {
	// Test workflow with command mode (not container mode)
	workflow := `---
name: Test Sandbox MCP Command
engine: copilot
on: workflow_dispatch
sandbox:
  agent: awf
  mcp:
    command: "./custom-gateway"
    args:
      - "--port"
      - "9000"
    port: 9000
    env:
      LOG_LEVEL: "debug"
tools:
  github:
    mode: remote
    toolsets: [default]
permissions:
  issues: read
  pull-requests: read
---

Test workflow for sandbox MCP command configuration.
`

	tmpDir := testutil.TempDir(t, "sandbox-mcp-command-test")
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	err := os.WriteFile(testFile, []byte(workflow), 0644)
	require.NoError(t, err)

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	compiler.SetStrictMode(false)
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err)

	// Read the compiled lock file
	lockFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
	lockFileContent, err := os.ReadFile(lockFile)
	require.NoError(t, err)
	require.NotEmpty(t, lockFileContent)

	// Verify the compiled workflow contains the correct command configuration
	lockFileStr := string(lockFileContent)

	// Check command start
	assert.Contains(t, lockFileStr, "Start MCP Gateway")
	assert.Contains(t, lockFileStr, "./custom-gateway --port 9000")
	
	// Check environment variables
	assert.Contains(t, lockFileStr, `LOG_LEVEL="debug"`)

	// Check health check uses correct port
	assert.Contains(t, lockFileStr, "Verify MCP Gateway Health")
	assert.Contains(t, lockFileStr, "http://localhost:9000")
	assert.NotContains(t, lockFileStr, "http://localhost:8080", "Health check should not use default port 8080")
}
