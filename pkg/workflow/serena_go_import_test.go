//go:build !integration

package workflow_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"
)

// TestImportSerenaGoMD tests that importing shared/mcp/serena-go.md results in the
// Serena MCP server being present in the compiled MCP config (container, entrypoint,
// entrypointArgs, mounts). This is the end-to-end chain:
//
//	main.md → shared/mcp/serena-go.md (uses/with) → shared/mcp/serena.md (explicit MCP config)
func TestImportSerenaGoMD(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-serena-go-*")

	// Create directory structure mirroring .github/workflows/shared/mcp/
	sharedMCPDir := filepath.Join(tempDir, "shared", "mcp")
	require.NoError(t, os.MkdirAll(sharedMCPDir, 0755), "create shared/mcp dir")

	// serena.md: parameterized shared workflow with complete explicit MCP server config
	serenaPath := filepath.Join(sharedMCPDir, "serena.md")
	serenaContent := `---
import-schema:
  languages:
    type: array
    items:
      type: string
    required: true

mcp-servers:
  serena:
    container: "ghcr.io/github/serena-mcp-server:latest"
    args:
      - "--network"
      - "host"
    entrypoint: "serena"
    entrypointArgs:
      - "start-mcp-server"
      - "--context"
      - "codex"
      - "--project"
      - \${GITHUB_WORKSPACE}
    mounts:
      - \${GITHUB_WORKSPACE}:\${GITHUB_WORKSPACE}:rw
---

## Serena Code Analysis

The Serena MCP server is configured for code analysis.
`
	require.NoError(t, os.WriteFile(serenaPath, []byte(serenaContent), 0644), "write serena.md")

	// serena-go.md: convenience wrapper that imports serena.md for Go
	serenaGoPath := filepath.Join(sharedMCPDir, "serena-go.md")
	serenaGoContent := `---
imports:
  - uses: shared/mcp/serena.md
    with:
      languages: ["go"]
---

## Serena Go Code Analysis

The Serena MCP server is configured for Go analysis.
`
	require.NoError(t, os.WriteFile(serenaGoPath, []byte(serenaGoContent), 0644), "write serena-go.md")

	// main workflow that imports serena-go.md
	workflowPath := filepath.Join(tempDir, "main-workflow.md")
	workflowContent := `---
on: issues
engine: copilot
imports:
  - shared/mcp/serena-go.md
permissions:
  contents: read
  issues: read
  pull-requests: read
---

# Main Workflow

Uses Serena for Go code analysis.
`
	require.NoError(t, os.WriteFile(workflowPath, []byte(workflowContent), 0644), "write main workflow")

	// Compile the workflow
	compiler := workflow.NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowPath), "CompileWorkflow")

	// Read the generated lock file
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "read lock file")

	lockContent := string(lockFileContent)

	// Serena MCP server must be present in the gateway config
	assert.Contains(t, lockContent, `"serena"`, "lock file should contain serena MCP server entry")

	// Container image must be correct
	assert.Contains(t, lockContent, "ghcr.io/github/serena-mcp-server:latest",
		"lock file should contain serena Docker container image")

	// Entrypoint must be set
	assert.Contains(t, lockContent, "serena", "lock file should contain serena entrypoint")

	// Docker image download step must include serena-mcp-server
	assert.Contains(t, lockContent, "download_docker_images.sh",
		"lock file should have docker image download step")
	assert.Contains(t, lockContent, "ghcr.io/github/serena-mcp-server",
		"docker image download step should include serena-mcp-server image")

	// Verify start-mcp-server entrypoint args are present
	assert.Contains(t, lockContent, "start-mcp-server",
		"lock file should contain start-mcp-server entrypoint arg")

	// Verify workspace mount is present
	assert.Contains(t, lockContent, "GITHUB_WORKSPACE",
		"lock file should reference GITHUB_WORKSPACE for workspace mount")
}

// TestImportSerenaWithLanguagesMD tests that importing shared/mcp/serena.md with
// explicit languages=[go, typescript] produces a working MCP config.
func TestImportSerenaWithLanguagesMD(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-serena-langs-*")

	sharedMCPDir := filepath.Join(tempDir, "shared", "mcp")
	require.NoError(t, os.MkdirAll(sharedMCPDir, 0755), "create shared/mcp dir")

	// serena.md: parameterized shared workflow
	serenaPath := filepath.Join(sharedMCPDir, "serena.md")
	serenaContent := `---
import-schema:
  languages:
    type: array
    items:
      type: string
    required: true

mcp-servers:
  serena:
    container: "ghcr.io/github/serena-mcp-server:latest"
    args:
      - "--network"
      - "host"
    entrypoint: "serena"
    entrypointArgs:
      - "start-mcp-server"
      - "--context"
      - "codex"
      - "--project"
      - \${GITHUB_WORKSPACE}
    mounts:
      - \${GITHUB_WORKSPACE}:\${GITHUB_WORKSPACE}:rw
---

## Serena Code Analysis
`
	require.NoError(t, os.WriteFile(serenaPath, []byte(serenaContent), 0644), "write serena.md")

	workflowPath := filepath.Join(tempDir, "main-workflow.md")
	workflowContent := `---
on: issues
engine: copilot
imports:
  - uses: shared/mcp/serena.md
    with:
      languages: ["go", "typescript"]
permissions:
  contents: read
  issues: read
  pull-requests: read
---

# Main Workflow

Uses Serena for Go and TypeScript analysis.
`
	require.NoError(t, os.WriteFile(workflowPath, []byte(workflowContent), 0644), "write main workflow")

	compiler := workflow.NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowPath), "CompileWorkflow")

	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	require.NoError(t, err, "read lock file")

	lockContent := string(lockFileContent)

	assert.Contains(t, lockContent, `"serena"`, "lock file should contain serena MCP entry")
	assert.Contains(t, lockContent, "ghcr.io/github/serena-mcp-server:latest",
		"lock file should contain serena container image")
	assert.Contains(t, lockContent, "start-mcp-server",
		"lock file should contain serena entrypoint args")
}
