package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/stringutil"
)

// TestCodexSafeInputsStdioTransport verifies that Codex engine uses stdio transport for safe-inputs
// (following the same pattern as safeoutputs) for consistency
func TestCodexSafeInputsStdioTransport(t *testing.T) {
	// Create a temporary workflow file
	tempDir := t.TempDir()
	workflowPath := filepath.Join(tempDir, "test-workflow.md")

	workflowContent := `---
on: workflow_dispatch
engine: codex
safe-inputs:
  test-tool:
    description: Test tool
    script: |
      return { result: "test" };
---

Test safe-inputs stdio transport for Codex
`

	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(lockContent)

	// Verify that the safe-inputs configuration steps are generated
	expectedSteps := []string{
		"Generate Safe Inputs MCP Server Config",
		"Start MCP gateway",
	}

	for _, stepName := range expectedSteps {
		if !strings.Contains(yamlStr, stepName) {
			t.Errorf("Expected step not found in workflow: %q", stepName)
		}
	}

	// Verify stdio transport in TOML config (not HTTP)
	if !strings.Contains(yamlStr, "[mcp_servers.safeinputs]") {
		t.Error("Safe-inputs MCP server config section not found")
	}

	// Should use stdio transport (container + entrypoint + entrypointArgs)
	codexConfigSection := extractCodexConfigSection(yamlStr)
	
	// Check for containerized stdio transport pattern (like safeoutputs)
	if !strings.Contains(codexConfigSection, `container = "node:lts-alpine"`) {
		t.Error("Expected container field for stdio transport not found in TOML format")
	}

	if !strings.Contains(codexConfigSection, `entrypoint = "node"`) {
		t.Error("Expected entrypoint field for stdio transport not found in TOML format")
	}

	if !strings.Contains(codexConfigSection, `entrypointArgs = ["/opt/gh-aw/safe-inputs/mcp-server.cjs"]`) {
		t.Error("Expected entrypointArgs field for stdio transport not found in TOML format")
	}

	if !strings.Contains(codexConfigSection, `mounts = ["/opt/gh-aw:/opt/gh-aw:ro", "/tmp/gh-aw:/tmp/gh-aw:rw"]`) {
		t.Error("Expected mounts field for stdio transport not found in TOML format")
	}

	// Should NOT use HTTP transport (url + headers)
	if strings.Contains(codexConfigSection, `type = "http"`) {
		t.Error("Codex config should not use HTTP transport, should use stdio")
	}

	if strings.Contains(codexConfigSection, `url = "http://`) {
		t.Error("Codex config should not use HTTP transport with url field, should use stdio")
	}

	if strings.Contains(codexConfigSection, `headers = {`) {
		t.Error("Codex config should not use HTTP transport with headers field, should use stdio")
	}

	// Verify environment variables ARE in the MCP config (env_vars supported for stdio transport)
	if !strings.Contains(codexConfigSection, "env_vars") {
		t.Error("stdio MCP servers should have env_vars in config")
	}

	t.Logf("✓ Codex engine correctly uses stdio transport for safe-inputs")
}

// extractCodexConfigSection extracts the Codex MCP config section from the workflow YAML
func extractCodexConfigSection(yamlContent string) string {
	// Find the start of the safeinputs config
	start := strings.Index(yamlContent, "[mcp_servers.safeinputs]")
	if start == -1 {
		return ""
	}

	// Find the end (next section or EOF)
	end := strings.Index(yamlContent[start:], "EOF")
	if end == -1 {
		return yamlContent[start:]
	}

	return yamlContent[start : start+end]
}

// TestCodexSafeInputsWithSecretsStdioTransport verifies that environment variables
// from safe-inputs tools are properly passed through with stdio transport
func TestCodexSafeInputsWithSecretsStdioTransport(t *testing.T) {
	tempDir := t.TempDir()
	workflowPath := filepath.Join(tempDir, "test-workflow.md")

	workflowContent := `---
on: workflow_dispatch
engine: codex
safe-inputs:
  api-call:
    description: Call an API
    env:
      API_KEY: ${{ secrets.API_KEY }}
      GH_TOKEN: ${{ github.token }}
    script: |
      return { result: "test" };
---

Test safe-inputs with secrets
`

	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	lockPath := stringutil.MarkdownToLockFile(workflowPath)
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(lockContent)
	codexConfigSection := extractCodexConfigSection(yamlStr)

	// Verify tool-specific env vars ARE in the MCP config (env_vars supported for stdio)
	if !strings.Contains(codexConfigSection, "env_vars") {
		t.Error("stdio MCP servers should have env_vars in config")
	}

	// Verify the specific env vars are included
	if !strings.Contains(codexConfigSection, "API_KEY") {
		t.Error("Expected API_KEY in env_vars list")
	}

	if !strings.Contains(codexConfigSection, "GH_TOKEN") {
		t.Error("Expected GH_TOKEN in env_vars list")
	}

	// Verify env vars are also set in Start MCP gateway step
	if !strings.Contains(yamlStr, "API_KEY: ${{ secrets.API_KEY }}") {
		t.Error("Expected API_KEY secret in Start MCP gateway env section")
	}

	if !strings.Contains(yamlStr, "GH_TOKEN: ${{ github.token }}") {
		t.Error("Expected GH_TOKEN in Start MCP gateway env section")
	}

	t.Logf("✓ Codex engine correctly passes secrets through stdio transport (via env_vars and job env)")
}
