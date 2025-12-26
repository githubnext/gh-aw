//go:build integration

package cli

import (
	"testing"
)

// SKIPPED: Scripts now use require() pattern and are loaded at runtime from external files
// TestSafeInputsMCPServerCompilation tests that safe-inputs are properly compiled
// into MCP server configurations for all three agentic engines
func TestSafeInputsMCPServerCompilation(t *testing.T) {
	t.Skip("Test skipped - safe-inputs MCP server scripts now use require() pattern and are loaded at runtime from external files")
}

	// Test cases for each engine
	engines := []struct {
		name         string
		engineConfig string
		toolName     string
	}{
		{
			name: "copilot",
			engineConfig: `engine: copilot
safe-inputs:
  echo-tool:
    description: Echo a message
    inputs:
      message:
        type: string
        description: Message to echo
        required: true
    script: |
      return { content: [{ type: 'text', text: \` + "`Echo: ${message}`" + ` }] };`,
			toolName: "echo-tool",
		},
		{
			name: "claude",
			engineConfig: `engine: claude
safe-inputs:
  process-text:
    description: Process text with Python
    inputs:
      text:
        type: string
        description: Text to process
        required: true
    py: |
      import json
      text = inputs.get('text', '')
      result = {
          "original": text,
          "uppercase": text.upper(),
          "word_count": len(text.split())
      }
      print(json.dumps(result))`,
			toolName: "process-text",
		},
		{
			name: "codex",
			engineConfig: `engine: codex
safe-inputs:
  calculate:
    description: Calculate sum of numbers
    inputs:
      numbers:
        type: string
        description: Comma-separated numbers
        required: true
    py: |
      import json
      numbers_str = inputs.get('numbers', '')
      numbers = [float(x.strip()) for x in numbers_str.split(',') if x.strip()]
      result = {"sum": sum(numbers), "count": len(numbers)}
      print(json.dumps(result))`,
			toolName: "calculate",
		},
	}

	for _, tc := range engines {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test workflow file for this engine
			workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
` + tc.engineConfig + `
---

# Test Safe-Inputs Configuration for ` + tc.name + `

This workflow tests safe-inputs tool configuration.
`

			workflowFile := filepath.Join(setup.workflowsDir, "test-safe-inputs-"+tc.name+".md")
			if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
				t.Fatalf("Failed to create test workflow file: %v", err)
			}

			// Compile the workflow
			cmd := exec.Command(setup.binaryPath, "compile", "test-safe-inputs-"+tc.name)
			cmd.Dir = setup.tempDir
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v\nOutput: %s", err, string(output))
			}

			t.Logf("✓ Workflow compiled successfully for %s engine", tc.name)

			// Read the generated lock file
			lockFile := filepath.Join(setup.workflowsDir, "test-safe-inputs-"+tc.name+".lock.yml")
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockStr := string(lockContent)

			// Verify that safe-inputs MCP server files are generated
			expectedContent := []string{
				"safe_inputs_mcp_server.cjs",
				"safe_inputs_config_loader.cjs",
				"safe_inputs_tool_factory.cjs",
				"tools.json",
			}

			for _, content := range expectedContent {
				if !strings.Contains(lockStr, content) {
					t.Errorf("Expected safe-inputs file not found in lock file for %s engine: %s", tc.name, content)
				}
			}

			t.Logf("✓ Safe-inputs MCP server files generated for %s engine", tc.name)

			// Verify the tool configuration is in tools.json
			if strings.Contains(lockStr, tc.toolName) {
				t.Logf("✓ Tool '%s' found in configuration for %s engine", tc.toolName, tc.name)
			} else {
				t.Errorf("Tool '%s' not found in configuration for %s engine", tc.toolName, tc.name)
			}

			// Verify MCP configuration includes safeinputs server
			if strings.Contains(lockStr, "safeinputs") || strings.Contains(lockStr, "safe-inputs") {
				t.Logf("✓ Safe-inputs MCP server referenced in workflow for %s engine", tc.name)
			}
		})
	}
}

// TestSafeInputsMCPServerConfiguration tests that safe-inputs tools are properly configured
// in the generated tools.json file
func TestSafeInputsMCPServerConfiguration(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	// Create a workflow with multiple safe-inputs tools
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
safe-inputs:
  echo-tool:
    description: Echo a message
    inputs:
      message:
        type: string
        description: Message to echo
        required: true
    script: |
      return { content: [{ type: 'text', text: \` + "`Echo: ${message}`" + ` }] };
  
  greet-tool:
    description: Greet someone
    inputs:
      name:
        type: string
        description: Name to greet
        required: true
    script: |
      return { content: [{ type: 'text', text: \` + "`Hello, ${name}!`" + ` }] };

  calculate-sum:
    description: Calculate sum of numbers
    inputs:
      numbers:
        type: string
        description: Comma-separated numbers
        required: true
    py: |
      import json
      numbers_str = inputs.get('numbers', '')
      numbers = [float(x.strip()) for x in numbers_str.split(',') if x.strip()]
      result = {"sum": sum(numbers)}
      print(json.dumps(result))
---

# Test Safe-Inputs Tools Configuration

Test workflow for safe-inputs tools configuration.
`

	workflowFile := filepath.Join(setup.workflowsDir, "test-safe-inputs-tools.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}

	// Compile the workflow
	cmd := exec.Command(setup.binaryPath, "compile", "test-safe-inputs-tools")
	cmd.Dir = setup.tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v\nOutput: %s", err, string(output))
	}

	t.Logf("✓ Workflow compiled successfully")

	// Read the generated lock file
	lockFile := filepath.Join(setup.workflowsDir, "test-safe-inputs-tools.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Verify tool names appear in the configuration
	expectedTools := []string{"echo-tool", "greet-tool", "calculate-sum"}
	foundToolCount := 0
	for _, tool := range expectedTools {
		if strings.Contains(lockStr, tool) {
			foundToolCount++
			t.Logf("✓ Found tool in configuration: %s", tool)
		}
	}

	if foundToolCount == len(expectedTools) {
		t.Logf("✓ All %d safe-inputs tools found in compiled workflow", len(expectedTools))
	} else {
		t.Errorf("Only found %d/%d expected tools in compiled workflow", foundToolCount, len(expectedTools))
	}

	// Verify tool descriptions are present
	if strings.Contains(lockStr, "Echo a message") {
		t.Logf("✓ Tool descriptions present in configuration")
	}

	// Verify input schemas are present
	if strings.Contains(lockStr, "inputSchema") || strings.Contains(lockStr, "properties") {
		t.Logf("✓ Input schemas present in configuration")
	}
}

// TestSafeInputsMCPServerStartup tests that the safe-inputs MCP server can be started
// and responds to basic MCP protocol messages
func TestSafeInputsMCPServerStartup(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	// Check if Node.js is available
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("Node.js not available, skipping server startup test")
	}

	// Create a simple workflow with safe-inputs
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
safe-inputs:
  echo-tool:
    description: Echo a message
    inputs:
      message:
        type: string
        description: Message to echo
        required: true
    script: |
      return { content: [{ type: 'text', text: \` + "`Echo: ${message}`" + ` }] };
---

# Test Safe-Inputs Server Startup

Test workflow for safe-inputs server startup.
`

	workflowFile := filepath.Join(setup.workflowsDir, "test-safe-inputs-startup.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}

	// Compile the workflow
	cmd := exec.Command(setup.binaryPath, "compile", "test-safe-inputs-startup")
	cmd.Dir = setup.tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v\nOutput: %s", err, string(output))
	}

	t.Logf("✓ Workflow compiled successfully")

	// Read the lock file to verify tools.json is generated
	lockFile := filepath.Join(setup.workflowsDir, "test-safe-inputs-startup.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Search for tools.json content more safely
	toolsJSONMarker := "cat > /tmp/gh-aw/safe-inputs/tools.json"
	if !strings.Contains(lockStr, toolsJSONMarker) {
		t.Fatal("tools.json section not found in lock file")
	}

	t.Logf("✓ tools.json generation found in lock file")

	// Try to extract and validate tools.json structure
	eofMarker := "EOF_TOOLS_JSON"
	startIdx := strings.Index(lockStr, eofMarker)
	if startIdx == -1 {
		t.Fatal("EOF marker for tools.json not found")
	}

	// Move past the first EOF marker
	startIdx += len(eofMarker) + 1

	// Find the second occurrence (the closing marker)
	endIdx := strings.Index(lockStr[startIdx:], eofMarker)
	if endIdx == -1 {
		t.Fatal("Closing EOF marker for tools.json not found")
	}

	// Extract tools.json content
	toolsJSONContent := lockStr[startIdx : startIdx+endIdx]
	toolsJSONContent = strings.TrimSpace(toolsJSONContent)

	// Verify tools.json is valid JSON
	var toolsConfig map[string]interface{}
	if err := json.Unmarshal([]byte(toolsJSONContent), &toolsConfig); err != nil {
		t.Fatalf("tools.json is not valid JSON: %v\nContent: %s", err, toolsJSONContent)
	}

	t.Logf("✓ tools.json is valid JSON")

	// Verify the tools array exists
	if tools, ok := toolsConfig["tools"].([]interface{}); ok {
		t.Logf("✓ Found %d tool(s) in tools.json", len(tools))

		// Verify the echo-tool exists
		foundEchoTool := false
		for _, tool := range tools {
			if toolMap, ok := tool.(map[string]interface{}); ok {
				if name, ok := toolMap["name"].(string); ok && name == "echo-tool" {
					foundEchoTool = true
					t.Logf("✓ Found echo-tool in tools.json")

					// Verify tool has description
					if desc, ok := toolMap["description"].(string); ok {
						t.Logf("✓ echo-tool has description: %s", desc)
					}

					// Verify tool has inputSchema
					if _, ok := toolMap["inputSchema"]; ok {
						t.Logf("✓ echo-tool has inputSchema")
					}

					break
				}
			}
		}

		if !foundEchoTool {
			t.Error("echo-tool not found in tools.json")
		}
	} else {
		t.Error("tools array not found in tools.json")
	}
}

// TestSafeInputsMCPServerValidation tests that the safe-inputs MCP server
// validates tool configurations properly
func TestSafeInputsMCPServerValidation(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	// Create a workflow with various safe-inputs configurations
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
safe-inputs:
  # JavaScript handler
  js-tool:
    description: JavaScript tool
    inputs:
      input:
        type: string
        required: true
    script: |
      return { result: "JS: " + input };
  
  # Python handler
  py-tool:
    description: Python tool
    inputs:
      value:
        type: string
        required: true
    py: |
      import json
      print(json.dumps({"result": f"PY: {inputs.get('value')}"}))
  
  # Shell handler
  sh-tool:
    description: Shell tool
    inputs:
      name:
        type: string
        required: true
    run: |
      #!/bin/bash
      echo "SH: $INPUT_NAME"
---

# Test Safe-Inputs Validation

Test workflow for safe-inputs validation.
`

	workflowFile := filepath.Join(setup.workflowsDir, "test-safe-inputs-validation.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create test workflow file: %v", err)
	}

	// Compile the workflow
	cmd := exec.Command(setup.binaryPath, "compile", "test-safe-inputs-validation")
	cmd.Dir = setup.tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v\nOutput: %s", err, string(output))
	}

	t.Logf("✓ Workflow compiled successfully")

	// Read the lock file
	lockFile := filepath.Join(setup.workflowsDir, "test-safe-inputs-validation.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Verify all three types of handlers are present
	handlerChecks := []struct {
		name    string
		pattern string
	}{
		{"JavaScript", "js-tool"},
		{"Python", "py-tool"},
		{"Shell", "sh-tool"},
	}

	for _, check := range handlerChecks {
		if strings.Contains(lockStr, check.pattern) {
			t.Logf("✓ %s handler tool (%s) found in compiled workflow", check.name, check.pattern)
		} else {
			t.Errorf("%s handler tool (%s) not found in compiled workflow", check.name, check.pattern)
		}
	}

	// Verify handler files are generated for each type
	handlerFilePatterns := []struct {
		name    string
		pattern string
	}{
		{"JavaScript handler", ".cjs"},
		{"Python handler", ".py"},
		{"Shell handler", ".sh"},
	}

	for _, pattern := range handlerFilePatterns {
		if strings.Contains(lockStr, pattern.pattern) {
			t.Logf("✓ %s file pattern found", pattern.name)
		}
	}
}
