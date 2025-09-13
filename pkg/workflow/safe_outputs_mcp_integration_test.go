package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeOutputsMCPServerIntegration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "safe-outputs-integration-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file with safe-outputs configuration
	testContent := `---
name: Test Safe Outputs MCP
engine: claude
safe-outputs:
  create-issue:
    max: 3
  missing-tool: {}
---

Test safe outputs workflow with MCP server integration.
`

	testFile := filepath.Join(tmpDir, "test-safe-outputs.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Read the generated .lock.yml file
	lockFile := filepath.Join(tmpDir, "test-safe-outputs.lock.yml")
	yamlContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}
	yamlStr := string(yamlContent)

	// Check that safe-outputs MCP server file is written
	if !strings.Contains(yamlStr, "cat > /tmp/safe-outputs/mcp-server.cjs") {
		t.Error("Expected safe-outputs MCP server to be written to temp file")
	}

	// Check that the new actions/github-script format is used
	if !strings.Contains(yamlStr, "uses: actions/github-script@v7") {
		t.Error("Expected actions/github-script to be used for MCP configuration")
	}

	// Check that safe_outputs environment variable is configured
	if !strings.Contains(yamlStr, "MCP_SAFE_OUTPUTS_CONFIG") {
		t.Error("Expected MCP_SAFE_OUTPUTS_CONFIG environment variable")
	}

	// Check that the MCP generation script is included
	if !strings.Contains(yamlStr, "generateJSONConfig") ||
		!strings.Contains(yamlStr, "safe_outputs") {
		t.Error("Expected safe_outputs MCP server to be configured with node command")
	}

	// Check that safe outputs config is properly set
	if !strings.Contains(yamlStr, "GITHUB_AW_SAFE_OUTPUTS_CONFIG") {
		t.Error("Expected GITHUB_AW_SAFE_OUTPUTS_CONFIG environment variable to be set")
	}

	t.Log("Safe outputs MCP server integration test passed")
}

func TestSafeOutputsMCPServerDisabled(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "safe-outputs-disabled-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file without safe-outputs configuration
	testContent := `---
name: Test Without Safe Outputs
engine: claude
---

Test workflow without safe outputs.
`

	testFile := filepath.Join(tmpDir, "test-no-safe-outputs.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Read the generated .lock.yml file
	lockFile := filepath.Join(tmpDir, "test-no-safe-outputs.lock.yml")
	yamlContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}
	yamlStr := string(yamlContent)

	// Check that safe-outputs MCP server file is NOT written
	if strings.Contains(yamlStr, "cat > /tmp/safe-outputs/mcp-server.cjs") {
		t.Error("Expected safe-outputs MCP server to NOT be written when safe-outputs are disabled")
	}

	// Check that safe_outputs is NOT included in MCP configuration
	if strings.Contains(yamlStr, `"safe_outputs": {`) {
		t.Error("Expected safe_outputs to NOT be in MCP server configuration when disabled")
	}

	t.Log("Safe outputs MCP server disabled test passed")
}

func TestSafeOutputsMCPServerCodex(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "safe-outputs-codex-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file with safe-outputs configuration for Codex
	testContent := `---
name: Test Safe Outputs MCP with Codex
engine: codex
safe-outputs:
  create-issue: {}
  missing-tool: {}
---

Test safe outputs workflow with Codex engine.
`

	testFile := filepath.Join(tmpDir, "test-safe-outputs-codex.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Read the generated .lock.yml file
	lockFile := filepath.Join(tmpDir, "test-safe-outputs-codex.lock.yml")
	yamlContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}
	yamlStr := string(yamlContent)

	// Check that safe-outputs MCP server file is written
	if !strings.Contains(yamlStr, "cat > /tmp/safe-outputs/mcp-server.cjs") {
		t.Error("Expected safe-outputs MCP server to be written to temp file")
	}

	// Check that the new actions/github-script format is used
	if !strings.Contains(yamlStr, "uses: actions/github-script@v7") {
		t.Error("Expected actions/github-script to be used for MCP configuration")
	}

	// Check that TOML format is configured for Codex
	if !strings.Contains(yamlStr, "MCP_CONFIG_FORMAT: toml") {
		t.Error("Expected TOML format for Codex MCP configuration")
	}

	// Check that the MCP generation script is included
	if !strings.Contains(yamlStr, "generateTOMLConfig") {
		t.Error("Expected generateTOMLConfig function in script for Codex")
	}

	t.Log("Safe outputs MCP server Codex integration test passed")
}
