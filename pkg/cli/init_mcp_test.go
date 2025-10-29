package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitRepository_WithMCP(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create go.mod for the copilot-setup-steps.yml to reference
	goModContent := []byte("module github.com/test/repo\n\ngo 1.23\n")
	if err := os.WriteFile("go.mod", goModContent, 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Call the function with MCP flag
	err = InitRepository(false, true)
	if err != nil {
		t.Fatalf("InitRepository() with MCP returned error: %v", err)
	}

	// Verify standard files were created
	gitAttributesPath := filepath.Join(tempDir, ".gitattributes")
	if _, err := os.Stat(gitAttributesPath); os.IsNotExist(err) {
		t.Errorf("Expected .gitattributes file to exist")
	}

	// Verify copilot-setup-steps.yml was created
	setupStepsPath := filepath.Join(tempDir, ".github", "workflows", "copilot-setup-steps.yml")
	if _, err := os.Stat(setupStepsPath); os.IsNotExist(err) {
		t.Errorf("Expected copilot-setup-steps.yml to exist")
	} else {
		// Verify content contains key elements
		content, err := os.ReadFile(setupStepsPath)
		if err != nil {
			t.Fatalf("Failed to read copilot-setup-steps.yml: %v", err)
		}
		contentStr := string(content)

		if !strings.Contains(contentStr, "name: \"Copilot Setup Steps\"") {
			t.Errorf("Expected copilot-setup-steps.yml to contain workflow name")
		}
		if !strings.Contains(contentStr, "copilot-setup-steps:") {
			t.Errorf("Expected copilot-setup-steps.yml to contain job name")
		}
		if !strings.Contains(contentStr, "gh extension install githubnext/gh-aw") {
			t.Errorf("Expected copilot-setup-steps.yml to contain gh-aw installation steps")
		}
	}

	// Verify .vscode/mcp.json was created
	mcpConfigPath := filepath.Join(tempDir, ".vscode", "mcp.json")
	if _, err := os.Stat(mcpConfigPath); os.IsNotExist(err) {
		t.Errorf("Expected .vscode/mcp.json to exist")
	} else {
		// Verify content is valid JSON with gh-aw server
		content, err := os.ReadFile(mcpConfigPath)
		if err != nil {
			t.Fatalf("Failed to read .vscode/mcp.json: %v", err)
		}

		var config MCPConfig
		if err := json.Unmarshal(content, &config); err != nil {
			t.Fatalf("Failed to parse .vscode/mcp.json: %v", err)
		}

		if _, exists := config.Servers["github-agentic-workflows"]; !exists {
			t.Errorf("Expected .vscode/mcp.json to contain github-agentic-workflows server")
		}

		server := config.Servers["github-agentic-workflows"]
		if server.Command != "gh" {
			t.Errorf("Expected command to be 'gh', got %s", server.Command)
		}
		if len(server.Args) != 2 || server.Args[0] != "aw" || server.Args[1] != "mcp-server" {
			t.Errorf("Expected args to be ['aw', 'mcp-server'], got %v", server.Args)
		}
	}
}

func TestInitRepository_MCP_Idempotent(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create go.mod
	goModContent := []byte("module github.com/test/repo\n\ngo 1.23\n")
	if err := os.WriteFile("go.mod", goModContent, 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Call the function first time with MCP
	err = InitRepository(false, true)
	if err != nil {
		t.Fatalf("InitRepository() with MCP returned error on first call: %v", err)
	}

	// Call the function second time with MCP
	err = InitRepository(false, true)
	if err != nil {
		t.Fatalf("InitRepository() with MCP returned error on second call: %v", err)
	}

	// Verify files still exist
	setupStepsPath := filepath.Join(tempDir, ".github", "workflows", "copilot-setup-steps.yml")
	if _, err := os.Stat(setupStepsPath); os.IsNotExist(err) {
		t.Errorf("Expected copilot-setup-steps.yml to exist after second call")
	}

	mcpConfigPath := filepath.Join(tempDir, ".vscode", "mcp.json")
	if _, err := os.Stat(mcpConfigPath); os.IsNotExist(err) {
		t.Errorf("Expected .vscode/mcp.json to exist after second call")
	}
}

func TestEnsureMCPConfig_UpdatesExisting(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Create .vscode directory
	if err := os.MkdirAll(".vscode", 0755); err != nil {
		t.Fatalf("Failed to create .vscode directory: %v", err)
	}

	// Create initial mcp.json with a different server
	initialConfig := MCPConfig{
		Servers: map[string]MCPServerConfig{
			"other-server": {
				Command: "other-command",
				Args:    []string{"arg1"},
			},
		},
	}
	initialData, _ := json.MarshalIndent(initialConfig, "", "  ")
	mcpConfigPath := filepath.Join(tempDir, ".vscode", "mcp.json")
	if err := os.WriteFile(mcpConfigPath, initialData, 0644); err != nil {
		t.Fatalf("Failed to write initial mcp.json: %v", err)
	}

	// Call ensureMCPConfig
	if err := ensureMCPConfig(false); err != nil {
		t.Fatalf("ensureMCPConfig() returned error: %v", err)
	}

	// Verify the config now contains both servers
	content, err := os.ReadFile(mcpConfigPath)
	if err != nil {
		t.Fatalf("Failed to read mcp.json: %v", err)
	}

	var config MCPConfig
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatalf("Failed to parse mcp.json: %v", err)
	}

	// Check both servers exist
	if _, exists := config.Servers["other-server"]; !exists {
		t.Errorf("Expected existing 'other-server' to be preserved")
	}

	if _, exists := config.Servers["github-agentic-workflows"]; !exists {
		t.Errorf("Expected 'github-agentic-workflows' server to be added")
	}
}

func TestEnsureCopilotSetupSteps_SkipsExisting(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Create .github/workflows directory
	workflowsDir := filepath.Join(".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create custom copilot-setup-steps.yml
	setupStepsPath := filepath.Join(workflowsDir, "copilot-setup-steps.yml")
	customContent := []byte("# Custom setup steps\nname: Custom\n")
	if err := os.WriteFile(setupStepsPath, customContent, 0644); err != nil {
		t.Fatalf("Failed to write custom setup steps: %v", err)
	}

	// Call ensureCopilotSetupSteps
	if err := ensureCopilotSetupSteps(false); err != nil {
		t.Fatalf("ensureCopilotSetupSteps() returned error: %v", err)
	}

	// Verify the file was not overwritten
	content, err := os.ReadFile(setupStepsPath)
	if err != nil {
		t.Fatalf("Failed to read setup steps file: %v", err)
	}

	if string(content) != string(customContent) {
		t.Errorf("Expected custom content to be preserved, but it was overwritten")
	}
}
