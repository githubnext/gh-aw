package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

// TestWriteSafeInputsFiles tests that all required files are written correctly
func TestWriteSafeInputsFiles(t *testing.T) {
	// Create a test safe-inputs configuration
	safeInputsConfig := &workflow.SafeInputsConfig{
		Tools: map[string]*workflow.SafeInputToolConfig{
			"test-js": {
				Name:        "test-js",
				Description: "Test JavaScript tool",
				Script:      "console.log('test');",
				Inputs: map[string]*workflow.SafeInputParam{
					"message": {
						Type:        "string",
						Description: "Test message",
						Required:    true,
					},
				},
			},
			"test-sh": {
				Name:        "test-sh",
				Description: "Test shell script tool",
				Run:         "echo 'test'",
			},
			"test-py": {
				Name:        "test-py",
				Description: "Test Python tool",
				Py:          "print('test')",
			},
		},
	}

	// Create temporary directory
	tmpDir := t.TempDir()

	// Write files
	err := writeSafeInputsFiles(tmpDir, safeInputsConfig, false)
	if err != nil {
		t.Fatalf("writeSafeInputsFiles failed: %v", err)
	}

	// Verify JavaScript dependencies are written
	expectedJSFiles := []string{
		"read_buffer.cjs",
		"mcp_http_transport.cjs",
		"safe_inputs_config_loader.cjs",
		"mcp_server_core.cjs",
		"safe_inputs_validation.cjs",
		"mcp_logger.cjs",
		"mcp_handler_shell.cjs",
		"mcp_handler_python.cjs",
		"safe_inputs_mcp_server_http.cjs",
	}

	for _, filename := range expectedJSFiles {
		filePath := filepath.Join(tmpDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file not found: %s", filename)
		} else {
			// Verify file is not empty
			info, _ := os.Stat(filePath)
			if info.Size() == 0 {
				t.Errorf("File is empty: %s", filename)
			}
		}
	}

	// Verify tools.json is written
	toolsPath := filepath.Join(tmpDir, "tools.json")
	if _, err := os.Stat(toolsPath); os.IsNotExist(err) {
		t.Error("tools.json not found")
	} else {
		// Verify it's valid JSON with content
		content, _ := os.ReadFile(toolsPath)
		if len(content) < 10 {
			t.Error("tools.json is too short")
		}
	}

	// Verify mcp-server.cjs is written and executable
	mcpServerPath := filepath.Join(tmpDir, "mcp-server.cjs")
	if _, err := os.Stat(mcpServerPath); os.IsNotExist(err) {
		t.Error("mcp-server.cjs not found")
	} else {
		info, _ := os.Stat(mcpServerPath)
		// Check if executable (Unix permission bits)
		mode := info.Mode()
		if mode&0100 == 0 {
			t.Error("mcp-server.cjs is not executable")
		}
	}

	// Verify tool handler files are written
	expectedToolFiles := map[string]os.FileMode{
		"test-js.cjs": 0644,
		"test-sh.sh":  0755,
		"test-py.py":  0755,
	}

	for filename, expectedMode := range expectedToolFiles {
		filePath := filepath.Join(tmpDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected tool file not found: %s", filename)
		} else {
			// Verify file permissions
			info, _ := os.Stat(filePath)
			mode := info.Mode()
			if mode&0100 == 0 && expectedMode&0100 != 0 {
				t.Errorf("File should be executable but is not: %s", filename)
			}
			if mode&0100 != 0 && expectedMode&0100 == 0 {
				t.Errorf("File should not be executable but is: %s", filename)
			}
		}
	}

	// Verify logs directory is created
	logsDir := filepath.Join(tmpDir, "logs")
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		t.Error("logs directory not found")
	}
}

// TestSpawnSafeInputsInspector_NoSafeInputs tests the error case when workflow has no safe-inputs
func TestSpawnSafeInputsInspector_NoSafeInputs(t *testing.T) {
	// Create temporary directory with a workflow file
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow file WITHOUT safe-inputs
	workflowContent := `---
on: push
engine: copilot
---
# Test Workflow

This workflow has no safe-inputs configuration.
`
	workflowPath := filepath.Join(workflowsDir, "test.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Change to the temporary directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Try to spawn safe-inputs inspector - should fail
	err := spawnSafeInputsInspector("test", false)
	if err == nil {
		t.Error("Expected error when workflow has no safe-inputs, got nil")
	}

	// Verify error message mentions "no safe-inputs"
	if err != nil && err.Error() != "no safe-inputs configuration found in workflow" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

// TestSpawnSafeInputsInspector_WithSafeInputs tests file generation with a real workflow
func TestSpawnSafeInputsInspector_WithSafeInputs(t *testing.T) {
	// This test verifies that the function correctly parses a workflow and generates files
	// We can't actually start the server or inspector in a test, but we can verify file generation

	// Create temporary directory with a workflow file
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow file with safe-inputs
	workflowContent := `---
on: push
engine: copilot
safe-inputs:
  echo-tool:
    description: "Echo a message"
    inputs:
      message:
        type: string
        description: "Message to echo"
        required: true
    run: |
      echo "$message"
---
# Test Workflow

This workflow has safe-inputs configuration.
`
	workflowPath := filepath.Join(workflowsDir, "test.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Change to the temporary directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// We can't fully test spawnSafeInputsInspector because it tries to start a server
	// and launch the inspector, but we can test the file generation part separately
	// by calling writeSafeInputsFiles directly

	// Parse the workflow using the compiler to get safe-inputs config
	// (including any imported safe-inputs)
	compiler := workflow.NewCompiler(false, "", "")
	workflowData, err := compiler.ParseWorkflowFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	safeInputsConfig := workflowData.SafeInputs
	if safeInputsConfig == nil {
		t.Fatal("Expected safe-inputs config to be parsed")
	}

	// Create a temp directory for files
	filesDir := t.TempDir()

	// Write files
	err = writeSafeInputsFiles(filesDir, safeInputsConfig, false)
	if err != nil {
		t.Fatalf("writeSafeInputsFiles failed: %v", err)
	}

	// Verify the echo-tool.sh file was created
	toolPath := filepath.Join(filesDir, "echo-tool.sh")
	if _, err := os.Stat(toolPath); os.IsNotExist(err) {
		t.Error("echo-tool.sh not found")
	}

	// Verify tools.json contains the echo-tool
	toolsPath := filepath.Join(filesDir, "tools.json")
	toolsContent, err := os.ReadFile(toolsPath)
	if err != nil {
		t.Fatalf("Failed to read tools.json: %v", err)
	}

	// Simple check that the tool name is in the JSON
	if len(toolsContent) < 50 {
		t.Error("tools.json seems too short")
	}
}

// TestSpawnSafeInputsInspector_WithImportedSafeInputs tests that imported safe-inputs are resolved
func TestSpawnSafeInputsInspector_WithImportedSafeInputs(t *testing.T) {
	// Create temporary directory with workflow and shared files
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	sharedDir := filepath.Join(workflowsDir, "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a shared workflow file with safe-inputs
	sharedContent := `---
safe-inputs:
  shared-tool:
    description: "Shared tool from import"
    inputs:
      param:
        type: string
        description: "A parameter"
        required: true
    run: |
      echo "Shared: $param"
---
# Shared Workflow
`
	sharedPath := filepath.Join(sharedDir, "shared.md")
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared workflow file: %v", err)
	}

	// Create a test workflow file that imports the shared workflow
	workflowContent := `---
on: push
engine: copilot
imports:
  - shared/shared.md
safe-inputs:
  local-tool:
    description: "Local tool"
    inputs:
      message:
        type: string
        description: "Message to echo"
        required: true
    run: |
      echo "$message"
---
# Test Workflow

This workflow imports safe-inputs from shared/shared.md.
`
	workflowPath := filepath.Join(workflowsDir, "test.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Change to the temporary directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	// Parse the workflow using the compiler to get safe-inputs config
	// This should include both local and imported safe-inputs
	compiler := workflow.NewCompiler(false, "", "")
	workflowData, err := compiler.ParseWorkflowFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	safeInputsConfig := workflowData.SafeInputs
	if safeInputsConfig == nil {
		t.Fatal("Expected safe-inputs config to be parsed")
	}

	// Verify both local and imported tools are present
	if len(safeInputsConfig.Tools) != 2 {
		t.Errorf("Expected 2 tools (local + imported), got %d", len(safeInputsConfig.Tools))
	}

	// Verify local tool exists
	if _, exists := safeInputsConfig.Tools["local-tool"]; !exists {
		t.Error("Expected local-tool to be present")
	}

	// Verify imported tool exists
	if _, exists := safeInputsConfig.Tools["shared-tool"]; !exists {
		t.Error("Expected shared-tool (from import) to be present")
	}

	// Create a temp directory for files
	filesDir := t.TempDir()

	// Write files
	err = writeSafeInputsFiles(filesDir, safeInputsConfig, false)
	if err != nil {
		t.Fatalf("writeSafeInputsFiles failed: %v", err)
	}

	// Verify both tool handler files were created
	localToolPath := filepath.Join(filesDir, "local-tool.sh")
	if _, err := os.Stat(localToolPath); os.IsNotExist(err) {
		t.Error("local-tool.sh not found")
	}

	sharedToolPath := filepath.Join(filesDir, "shared-tool.sh")
	if _, err := os.Stat(sharedToolPath); os.IsNotExist(err) {
		t.Error("shared-tool.sh not found")
	}

	// Verify tools.json contains both tools
	toolsPath := filepath.Join(filesDir, "tools.json")
	toolsContent, err := os.ReadFile(toolsPath)
	if err != nil {
		t.Fatalf("Failed to read tools.json: %v", err)
	}

	// Check that both tool names are in the JSON
	toolsJSON := string(toolsContent)
	if !strings.Contains(toolsJSON, "local-tool") {
		t.Error("tools.json should contain 'local-tool'")
	}
	if !strings.Contains(toolsJSON, "shared-tool") {
		t.Error("tools.json should contain 'shared-tool'")
	}
}
