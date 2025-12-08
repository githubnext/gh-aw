package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSafeInputsStdioMode verifies that stdio mode generates correct configuration
func TestSafeInputsStdioMode(t *testing.T) {
	// Create a temporary workflow file
	tempDir := t.TempDir()
	workflowPath := filepath.Join(tempDir, "test-workflow.md")

	workflowContent := `---
on: workflow_dispatch
engine: copilot
safe-inputs:
  mode: stdio
  test-tool:
    description: Test tool
    script: |
      return { result: "test" };
---

Test safe-inputs stdio mode
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
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(lockContent)

	// Verify that HTTP server startup steps are NOT present
	unexpectedSteps := []string{
		"Generate Safe Inputs MCP Server Config",
		"Start Safe Inputs MCP HTTP Server",
	}

	for _, stepName := range unexpectedSteps {
		if strings.Contains(yamlStr, stepName) {
			t.Errorf("Unexpected HTTP server step found in stdio mode: %q", stepName)
		}
	}

	// Verify stdio configuration in MCP setup
	if !strings.Contains(yamlStr, `"safeinputs"`) {
		t.Error("Safe-inputs MCP server config not found")
	}

	// Should use stdio transport
	if !strings.Contains(yamlStr, `"type": "stdio"`) {
		t.Error("Expected type field set to 'stdio' in MCP config")
	}

	if !strings.Contains(yamlStr, `"command": "node"`) {
		t.Error("Expected command field in stdio config")
	}

	if !strings.Contains(yamlStr, `"/tmp/gh-aw/safe-inputs/mcp-server.cjs"`) {
		t.Error("Expected mcp-server.cjs in args for stdio mode")
	}

	// Should NOT have HTTP-specific fields
	safeinputsConfig := extractSafeinputsConfigSection(yamlStr)
	if strings.Contains(safeinputsConfig, `"url"`) {
		t.Error("Stdio mode should not have URL field")
	}

	if strings.Contains(safeinputsConfig, `"headers"`) {
		t.Error("Stdio mode should not have headers field")
	}

	// Verify the entry point script uses stdio
	if !strings.Contains(yamlStr, "startSafeInputsServer") {
		t.Error("Expected stdio entry point to use startSafeInputsServer")
	}

	// Check the actual mcp-server.cjs entry point uses stdio server
	entryPointSection := extractMCPServerEntryPoint(yamlStr)
	if !strings.Contains(entryPointSection, "startSafeInputsServer(configPath") {
		t.Error("Entry point should call startSafeInputsServer for stdio mode")
	}

	if strings.Contains(entryPointSection, "startHttpServer") {
		t.Error("Stdio mode entry point should not call startHttpServer")
	}

	t.Logf("✓ Stdio mode correctly configured without HTTP server steps")
}

// TestSafeInputsHTTPMode verifies that HTTP mode generates correct configuration
func TestSafeInputsHTTPMode(t *testing.T) {
	testCases := []struct {
		name string
		mode string // empty string tests default behavior
	}{
		{
			name: "explicit_http_mode",
			mode: "http",
		},
		{
			name: "default_mode",
			mode: "", // No mode specified, should default to HTTP
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary workflow file
			tempDir := t.TempDir()
			workflowPath := filepath.Join(tempDir, "test-workflow.md")

			modeField := ""
			if tc.mode != "" {
				modeField = "  mode: " + tc.mode + "\n"
			}

			workflowContent := `---
on: workflow_dispatch
engine: copilot
safe-inputs:
` + modeField + `  test-tool:
    description: Test tool
    script: |
      return { result: "test" };
---

Test safe-inputs HTTP mode
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
			lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
			lockContent, err := os.ReadFile(lockPath)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			yamlStr := string(lockContent)

			// Verify that HTTP server startup steps ARE present
			expectedSteps := []string{
				"Generate Safe Inputs MCP Server Config",
				"Start Safe Inputs MCP HTTP Server",
			}

			for _, stepName := range expectedSteps {
				if !strings.Contains(yamlStr, stepName) {
					t.Errorf("Expected HTTP server step not found: %q", stepName)
				}
			}

			// Verify HTTP configuration in MCP setup
			if !strings.Contains(yamlStr, `"safeinputs"`) {
				t.Error("Safe-inputs MCP server config not found")
			}

			// Should use HTTP transport
			if !strings.Contains(yamlStr, `"type": "http"`) {
				t.Error("Expected type field set to 'http' in MCP config")
			}

			if !strings.Contains(yamlStr, `"url": "http://host.docker.internal`) {
				t.Error("Expected HTTP URL in config")
			}

			if !strings.Contains(yamlStr, `"headers"`) {
				t.Error("Expected headers field in HTTP config")
			}

			// Verify the entry point script uses HTTP
			if !strings.Contains(yamlStr, "startHttpServer") {
				t.Error("Expected HTTP entry point to use startHttpServer")
			}

			// Check the actual mcp-server.cjs entry point uses HTTP server
			entryPointSection := extractMCPServerEntryPoint(yamlStr)
			if !strings.Contains(entryPointSection, "startHttpServer(configPath") {
				t.Error("Entry point should call startHttpServer for HTTP mode")
			}

			if strings.Contains(entryPointSection, "startSafeInputsServer(configPath") {
				t.Error("HTTP mode entry point should not call startSafeInputsServer")
			}

			t.Logf("✓ HTTP mode correctly configured with HTTP server steps")
		})
	}
}

// TestSafeInputsModeInImport verifies that mode can be set via imports
func TestSafeInputsModeInImport(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	sharedDir := filepath.Join(tempDir, "shared")
	err := os.Mkdir(sharedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	// Create import file with stdio mode
	importPath := filepath.Join(sharedDir, "tool.md")
	importContent := `---
safe-inputs:
  mode: stdio
  imported-tool:
    description: Imported tool
    script: |
      return { result: "imported" };
---

Imported tool
`

	err = os.WriteFile(importPath, []byte(importContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write import file: %v", err)
	}

	// Create main workflow that imports the tool
	workflowPath := filepath.Join(tempDir, "workflow.md")
	workflowContent := `---
on: workflow_dispatch
engine: copilot
imports:
  - shared/tool.md
---

Test mode via import
`

	err = os.WriteFile(workflowPath, []byte(workflowContent), 0644)
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
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(lockContent)

	// Verify stdio mode is used from import
	if !strings.Contains(yamlStr, `"type": "stdio"`) {
		t.Error("Expected stdio mode from imported configuration")
	}

	// Verify HTTP server steps are NOT present
	if strings.Contains(yamlStr, "Start Safe Inputs MCP HTTP Server") {
		t.Error("Should not have HTTP server step when mode is stdio via import")
	}

	t.Logf("✓ Mode correctly inherited from import")
}

// extractSafeinputsConfigSection extracts the safeinputs configuration section from the YAML
func extractSafeinputsConfigSection(yamlStr string) string {
	start := strings.Index(yamlStr, `"safeinputs"`)
	if start == -1 {
		return ""
	}

	// Find the closing brace for the safeinputs object
	// This is a simple heuristic - we look for the next server or closing brace
	end := strings.Index(yamlStr[start:], `},`)
	if end == -1 {
		end = strings.Index(yamlStr[start:], `}`)
	}

	if end == -1 {
		return yamlStr[start:]
	}

	return yamlStr[start : start+end+1]
}

// extractMCPServerEntryPoint extracts the mcp-server.cjs entry point script from the YAML
func extractMCPServerEntryPoint(yamlStr string) string {
	// Find the mcp-server.cjs section
	start := strings.Index(yamlStr, "cat > /tmp/gh-aw/safe-inputs/mcp-server.cjs")
	if start == -1 {
		return ""
	}

	// Find the heredoc start marker
	heredocStart := strings.Index(yamlStr[start:], "<< 'EOFSI'")
	if heredocStart == -1 {
		return ""
	}
	// Move past the heredoc start and newline to the actual content
	contentStart := start + heredocStart + len("<< 'EOFSI'\n")

	// Find the EOFSI marker that ends the heredoc (should be at start of a line)
	end := strings.Index(yamlStr[contentStart:], "\n          EOFSI")
	if end == -1 {
		// Try without the leading spaces (in case formatting is different)
		end = strings.Index(yamlStr[contentStart:], "\nEOFSI")
		if end == -1 {
			return ""
		}
	}

	return yamlStr[contentStart : contentStart+end]
}
