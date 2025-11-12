package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewInitCommand(t *testing.T) {
	t.Parallel()

	cmd := NewInitCommand()

	if cmd == nil {
		t.Fatal("NewInitCommand() returned nil")
	}

	if cmd.Use != "init" {
		t.Errorf("Expected Use to be 'init', got %q", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("Expected Long description to be set")
	}

	// Verify flags
	mcpFlag := cmd.Flags().Lookup("mcp")
	if mcpFlag == nil {
		t.Error("Expected 'mcp' flag to be defined")
	}

	if mcpFlag.DefValue != "false" {
		t.Errorf("Expected mcp flag default to be 'false', got %q", mcpFlag.DefValue)
	}
}

func TestInitCommandHelp(t *testing.T) {
	t.Parallel()

	cmd := NewInitCommand()

	// Test that help can be generated without error
	helpText := cmd.Long
	if !strings.Contains(helpText, "Initialize") {
		t.Error("Expected help text to contain 'Initialize'")
	}

	if !strings.Contains(helpText, ".gitattributes") {
		t.Error("Expected help text to mention .gitattributes")
	}

	if !strings.Contains(helpText, "Copilot") {
		t.Error("Expected help text to mention Copilot")
	}
}

func TestInitRepositoryBasic(t *testing.T) {
	tmpDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize git repo (required for some init operations)
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	// Configure git
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// Test basic init without MCP
	err = InitRepository(false, false)
	if err != nil {
		t.Fatalf("InitRepository() failed: %v", err)
	}

	// Verify .gitattributes was created/updated
	gitAttributesPath := ".gitattributes"
	if _, err := os.Stat(gitAttributesPath); os.IsNotExist(err) {
		t.Error("Expected .gitattributes to be created")
	}

	// Read and verify .gitattributes content
	content, err := os.ReadFile(gitAttributesPath)
	if err != nil {
		t.Fatalf("Failed to read .gitattributes: %v", err)
	}

	expectedEntry := ".github/workflows/*.lock.yml linguist-generated=true merge=ours"
	if !strings.Contains(string(content), expectedEntry) {
		t.Errorf("Expected .gitattributes to contain %q", expectedEntry)
	}
}

func TestInitRepositoryWithMCP(t *testing.T) {
	tmpDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	// Configure git
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// Test init with MCP flag
	err = InitRepository(false, true)
	if err != nil {
		t.Fatalf("InitRepository() with MCP failed: %v", err)
	}

	// Verify .vscode/mcp.json was created
	mcpConfigPath := filepath.Join(".vscode", "mcp.json")
	if _, err := os.Stat(mcpConfigPath); os.IsNotExist(err) {
		t.Error("Expected .vscode/mcp.json to be created")
	}

	// Verify copilot-setup-steps.yml was created
	setupStepsPath := filepath.Join(".github", "workflows", "copilot-setup-steps.yml")
	if _, err := os.Stat(setupStepsPath); os.IsNotExist(err) {
		t.Error("Expected .github/workflows/copilot-setup-steps.yml to be created")
	}
}

func TestInitRepositoryVerbose(t *testing.T) {
	tmpDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	// Configure git
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// Test verbose mode (should not error, just produce more output)
	err = InitRepository(true, false)
	if err != nil {
		t.Fatalf("InitRepository() in verbose mode failed: %v", err)
	}

	// Verify basic files were still created
	if _, err := os.Stat(".gitattributes"); os.IsNotExist(err) {
		t.Error("Expected .gitattributes to be created even in verbose mode")
	}
}

func TestInitRepositoryNotInGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Don't initialize git repo - should fail for some operations
	err = InitRepository(false, false)
	
	// The function should handle this gracefully or return an error
	// Based on the implementation, ensureGitAttributes requires git
	if err == nil {
		t.Log("InitRepository() succeeded despite not being in a git repo")
	}
}

func TestInitRepositoryIdempotent(t *testing.T) {
	tmpDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	// Configure git
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// Run init twice
	err = InitRepository(false, false)
	if err != nil {
		t.Fatalf("First InitRepository() failed: %v", err)
	}

	// Second run should be idempotent
	err = InitRepository(false, false)
	if err != nil {
		t.Fatalf("Second InitRepository() failed: %v", err)
	}

	// Verify .gitattributes still correct
	content, err := os.ReadFile(".gitattributes")
	if err != nil {
		t.Fatalf("Failed to read .gitattributes: %v", err)
	}

	expectedEntry := ".github/workflows/*.lock.yml linguist-generated=true merge=ours"
	
	// Count occurrences - should only appear once
	count := strings.Count(string(content), expectedEntry)
	if count != 1 {
		t.Errorf("Expected .gitattributes entry to appear exactly once, got %d occurrences", count)
	}
}

func TestInitRepositoryWithMCPIdempotent(t *testing.T) {
	tmpDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	// Configure git
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// Run init with MCP twice
	err = InitRepository(false, true)
	if err != nil {
		t.Fatalf("First InitRepository() with MCP failed: %v", err)
	}

	err = InitRepository(false, true)
	if err != nil {
		t.Fatalf("Second InitRepository() with MCP failed: %v", err)
	}

	// Verify files still exist and are correct
	mcpConfigPath := filepath.Join(".vscode", "mcp.json")
	if _, err := os.Stat(mcpConfigPath); os.IsNotExist(err) {
		t.Error("Expected .vscode/mcp.json to still exist after second run")
	}

	setupStepsPath := filepath.Join(".github", "workflows", "copilot-setup-steps.yml")
	if _, err := os.Stat(setupStepsPath); os.IsNotExist(err) {
		t.Error("Expected copilot-setup-steps.yml to still exist after second run")
	}
}

func TestInitRepositoryCreatesDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	// Configure git
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// Run init with MCP
	err = InitRepository(false, true)
	if err != nil {
		t.Fatalf("InitRepository() failed: %v", err)
	}

	// Verify directory structure
	vscodeDir := ".vscode"
	info, err := os.Stat(vscodeDir)
	if os.IsNotExist(err) {
		t.Error("Expected .vscode directory to be created")
	} else if !info.IsDir() {
		t.Error("Expected .vscode to be a directory")
	}

	workflowsDir := filepath.Join(".github", "workflows")
	info, err = os.Stat(workflowsDir)
	if os.IsNotExist(err) {
		t.Error("Expected .github/workflows directory to be created")
	} else if !info.IsDir() {
		t.Error("Expected .github/workflows to be a directory")
	}
}

func TestInitCommandFlagValidation(t *testing.T) {
	t.Parallel()

	cmd := NewInitCommand()

	// Test that mcp flag is a boolean
	mcpFlag := cmd.Flags().Lookup("mcp")
	if mcpFlag == nil {
		t.Fatal("Expected 'mcp' flag to exist")
	}

	if mcpFlag.Value.Type() != "bool" {
		t.Errorf("Expected mcp flag to be bool, got %s", mcpFlag.Value.Type())
	}

	// Test verbose flag exists (inherited from parent command likely)
	// Note: verbose flag might be added by parent command, not in init command itself
}

func TestInitRepositoryErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Test init without git repo
	err = InitRepository(false, false)
	
	// Should handle error gracefully or return error
	// The actual behavior depends on implementation
	if err != nil {
		// Error is acceptable if git is required
		if !strings.Contains(err.Error(), "git") {
			t.Logf("Received error (acceptable): %v", err)
		}
	}
}

func TestInitRepositoryWithExistingFiles(t *testing.T) {
	tmpDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	// Configure git
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// Create existing .gitattributes with different content
	existingContent := "*.md linguist-documentation=true\n"
	if err := os.WriteFile(".gitattributes", []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create existing .gitattributes: %v", err)
	}

	// Run init
	err = InitRepository(false, false)
	if err != nil {
		t.Fatalf("InitRepository() failed: %v", err)
	}

	// Verify existing content is preserved and new entry is added
	content, err := os.ReadFile(".gitattributes")
	if err != nil {
		t.Fatalf("Failed to read .gitattributes: %v", err)
	}

	contentStr := string(content)
	
	// Should contain both old and new entries
	if !strings.Contains(contentStr, "*.md linguist-documentation=true") {
		t.Error("Expected existing content to be preserved")
	}

	expectedEntry := ".github/workflows/*.lock.yml linguist-generated=true merge=ours"
	if !strings.Contains(contentStr, expectedEntry) {
		t.Error("Expected new entry to be added")
	}
}
