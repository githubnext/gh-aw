package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

// TestSafeOutputsMCPServerUsesCjsExtension verifies that safe-outputs MCP server
// files are written with .cjs extension to avoid Node.js ESM vs CommonJS confusion
func TestSafeOutputsMCPServerUsesCjsExtension(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "test-safe-outputs-cjs")

	// Create a minimal workflow with safe-outputs enabled
	workflowSource := `---
on: issues
engine: copilot
safe-outputs:
  create-issue: {}
---

Test workflow for .cjs extension verification`

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(workflowSource), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated .lock.yml file
	lockFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
	yamlContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}
	yaml := string(yamlContent)

	// Verify that the setup-safe-outputs action is used (files are no longer written inline)
	if !strings.Contains(yaml, "uses: ./actions/setup-safe-outputs") {
		t.Error("Expected safe-outputs to use the setup-safe-outputs action")
	}

	// Verify that JavaScript files are NOT written inline
	if strings.Contains(yaml, "cat > /tmp/gh-aw/safeoutputs/mcp-server.cjs") {
		t.Error("Expected mcp-server.cjs to be copied by setup-safe-outputs action, not written inline")
	}

	// Verify that no .js files are written inline (should all come from action)
	if err := verifyNoDotJSFiles(yaml, "/tmp/gh-aw/safeoutputs/", t); err != nil {
		t.Error(err)
	}

	// Verify MCP config references .cjs
	if !strings.Contains(yaml, "\"/tmp/gh-aw/safeoutputs/mcp-server.cjs\"") {
		t.Error("Expected MCP config to reference mcp-server.cjs")
	}
}

// TestSafeInputsMCPServerUsesCjsExtension verifies that safe-inputs MCP server
// files and tool scripts are written with .cjs extension
func TestSafeInputsMCPServerUsesCjsExtension(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "test-safe-inputs-cjs")

	// Create a minimal workflow with safe-inputs enabled
	workflowSource := `---
on: issues
engine: copilot
safe-inputs:
  test_tool:
    description: "Test tool for .cjs verification"
    script: |
      return "hello";
---

Test workflow for .cjs extension verification`

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(workflowSource), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated .lock.yml file
	lockFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
	yamlContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}
	yaml := string(yamlContent)

	// Verify that the main MCP server entry point uses .cjs extension
	if !strings.Contains(yaml, "cat > /tmp/gh-aw/safe-inputs/mcp-server.cjs") {
		t.Error("Expected safe-inputs MCP server to be written as mcp-server.cjs, not mcp-server.js")
	}

	// Verify that JavaScript tool files use .cjs extension
	if !strings.Contains(yaml, "cat > /tmp/gh-aw/safe-inputs/test_tool.cjs") {
		t.Error("Expected JavaScript tool to be written as test_tool.cjs, not test_tool.js")
	}

	// Verify that no .js files are written (should be .cjs or other extensions like .sh, .py, .json)
	if err := verifyNoDotJSFiles(yaml, "/tmp/gh-aw/safe-inputs/", t); err != nil {
		t.Error(err)
	}

	// Verify the chmod command uses .cjs
	if !strings.Contains(yaml, "chmod +x /tmp/gh-aw/safe-inputs/mcp-server.cjs") {
		t.Error("Expected chmod command to reference mcp-server.cjs")
	}

	// Verify that all embedded JavaScript modules use .cjs extension
	expectedModules := []string{
		"read_buffer.cjs",
		"mcp_server_core.cjs",
		"mcp_handler_shell.cjs",
		"mcp_handler_python.cjs",
		"safe_inputs_config_loader.cjs",
		"safe_inputs_tool_factory.cjs",
		"safe_inputs_validation.cjs",
		"safe_inputs_mcp_server.cjs",
		"safe_inputs_mcp_server_http.cjs",
	}

	for _, module := range expectedModules {
		if !strings.Contains(yaml, "cat > /tmp/gh-aw/safe-inputs/"+module) {
			t.Errorf("Expected module %s to be written with .cjs extension", module)
		}
	}

	// Verify require statements in generated entry point use .cjs
	if !strings.Contains(yaml, `require("./safe_inputs_mcp_server_http.cjs")`) {
		t.Error("Expected require statement to reference .cjs extension")
	}
}

// TestSafeInputsToolsConfigUsesCjsExtension verifies that the tools.json configuration
// references JavaScript handlers with .cjs extension
func TestSafeInputsToolsConfigUsesCjsExtension(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "test-safe-inputs-config-cjs")

	// Create a workflow with safe-inputs JavaScript tool
	workflowSource := `---
on: issues
engine: copilot
safe-inputs:
  my_tool:
    description: "Test tool"
    script: |
      return "test";
---

Test workflow`

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(workflowSource), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated .lock.yml file
	lockFile := filepath.Join(tmpDir, "test-workflow.lock.yml")
	yamlContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}
	yaml := string(yamlContent)

	// Verify that the tools.json references .cjs handler
	if !strings.Contains(yaml, `"handler": "my_tool.cjs"`) {
		t.Error("Expected tools.json to reference my_tool.cjs as handler, not my_tool.js")
	}
}

// TestJavaScriptSourcesUseCjsExtension verifies that all embedded JavaScript sources
// in the GetJavaScriptSources map use .cjs extension
func TestJavaScriptSourcesUseCjsExtension(t *testing.T) {
	sources := GetJavaScriptSources()

	for filename := range sources {
		// All JavaScript files should use .cjs extension
		if strings.HasSuffix(filename, ".js") && !strings.HasSuffix(filename, ".cjs") {
			t.Errorf("JavaScript source file %s should use .cjs extension, not .js", filename)
		}
	}
}

// verifyNoDotJSFiles checks that no .js files (only .cjs or other extensions) are written
// to the specified directory path in the generated YAML
func verifyNoDotJSFiles(yaml, dirPath string, t *testing.T) error {
	lines := strings.Split(yaml, "\n")
	for i, line := range lines {
		// Check for file creation commands with .js extension (but not .cjs or .json)
		if strings.Contains(line, "cat > "+dirPath) {
			// Use word boundary check for .js to avoid false positives
			// Match .js followed by space, <, ", or end of string
			if strings.Contains(line, ".js ") || strings.Contains(line, ".js<") ||
				strings.Contains(line, ".js\"") || strings.HasSuffix(strings.TrimSpace(line), ".js") {
				// Make sure it's not .cjs or .json
				if !strings.Contains(line, ".cjs") && !strings.Contains(line, ".json") {
					return fmt.Errorf("line %d: Found .js file being written in %s: %s", i+1, dirPath, line)
				}
			}
		}
	}
	return nil
}
