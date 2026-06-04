//go:build !integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
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
	noMcpFlag := cmd.Flags().Lookup("no-mcp")
	if noMcpFlag == nil {
		t.Error("Expected 'no-mcp' flag to be defined")
		return
	}

	noSkillFlag := cmd.Flags().Lookup("no-skill")
	if noSkillFlag == nil {
		t.Error("Expected 'no-skill' flag to be defined")
		return
	}

	noAgentFlag := cmd.Flags().Lookup("no-agent")
	if noAgentFlag == nil {
		t.Error("Expected 'no-agent' flag to be defined")
		return
	}

	// Verify hidden --mcp flag still exists for backward compatibility
	mcpFlag := cmd.Flags().Lookup("mcp")
	if mcpFlag == nil {
		t.Error("Expected 'mcp' flag to be defined (for backward compatibility)")
		return
	}

	engineFlag := cmd.Flags().Lookup("engine")
	if engineFlag == nil {
		t.Error("Expected 'engine' flag to be defined")
		return
	}
	if engineFlag.Hidden {
		t.Error("Expected 'engine' flag to be visible")
	}

	// Verify --mcp flag is hidden
	if !mcpFlag.Hidden {
		t.Error("Expected 'mcp' flag to be hidden")
	}

	if noMcpFlag.DefValue != "false" {
		t.Errorf("Expected no-mcp flag default to be 'false', got %q", noMcpFlag.DefValue)
	}
	if noSkillFlag.DefValue != "false" {
		t.Errorf("Expected no-skill flag default to be 'false', got %q", noSkillFlag.DefValue)
	}
	if noAgentFlag.DefValue != "false" {
		t.Errorf("Expected no-agent flag default to be 'false', got %q", noAgentFlag.DefValue)
	}

	if mcpFlag.DefValue != "false" {
		t.Errorf("Expected mcp flag default to be 'false', got %q", mcpFlag.DefValue)
	}

	codespaceFlag := cmd.Flags().Lookup("codespaces")
	if codespaceFlag == nil {
		t.Error("Expected 'codespaces' flag to be defined")
		return
	}
	if !strings.Contains(codespaceFlag.Usage, "or use with an empty value for the current repo only") {
		t.Errorf("Expected codespaces flag help text to include article fixes, got %q", codespaceFlag.Usage)
	}

	// String flags without NoOptDefVal require an explicit value
	if codespaceFlag.DefValue != "" {
		t.Errorf("Expected codespaces flag default to be '', got %q", codespaceFlag.DefValue)
	}

	// Verify NoOptDefVal is empty (flag always requires an explicit value, avoids string[=" "] in help)
	if codespaceFlag.NoOptDefVal != "" {
		t.Errorf("Expected codespaces flag NoOptDefVal to be '' (empty), got %q", codespaceFlag.NoOptDefVal)
	}

	// Check create-pull-request flags
	createPRFlag := cmd.Flags().Lookup("create-pull-request")
	if createPRFlag == nil {
		t.Error("Expected 'create-pull-request' flag to be defined")
		return
	}

	prFlag := cmd.Flags().Lookup("pr")
	if prFlag == nil {
		t.Error("Expected 'pr' flag to be defined (alias)")
		return
	}

	// Verify --pr flag is hidden
	if !prFlag.Hidden {
		t.Error("Expected 'pr' flag to be hidden")
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

	if !strings.Contains(helpText, ".github/agents/agentic-workflows.md") {
		t.Error("Expected help text to mention the Agentic Workflows custom agent")
	}

	if !strings.Contains(helpText, ".github/skills/agentic-workflow-designer/SKILL.md") {
		t.Error("Expected help text to mention the agentic workflow designer skill")
	}

	if !strings.Contains(helpText, "Copilot") {
		t.Error("Expected help text to mention Copilot")
	}

	if !strings.Contains(helpText, "non-interactive repository setup") {
		t.Error("Expected help text to mention non-interactive setup")
	}

	if strings.Contains(helpText, "Usage:") {
		t.Error("Expected init long help text to not embed a Usage section")
	}
}

func TestInitCommandInteractiveModeDetection(t *testing.T) {
	t.Parallel()

	// Test that interactive mode is triggered when no flags are set
	// We can't test the actual interactive prompts in unit tests, but we can
	// verify that the command structure supports the detection logic

	cmd := NewInitCommand()

	// Verify that all the flags exist that are checked for interactive mode detection
	requiredFlags := []string{"mcp", "no-mcp", "no-skill", "no-agent", "codespaces", "completions", "create-pull-request", "pr"}
	for _, flagName := range requiredFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag %q to exist for interactive mode detection", flagName)
		}
	}
}

func TestInitRepositoryBasic(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

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
	if err := exec.Command("git", "config", "user.name", "Test User").Run(); err != nil {
		t.Fatalf("Failed to set git user.name: %v", err)
	}
	if err := exec.Command("git", "config", "user.email", "test@example.com").Run(); err != nil {
		t.Fatalf("Failed to set git user.email: %v", err)
	}

	// Test basic init with MCP enabled by default (mcp=true, noMcp=false behavior)
	err = InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil})
	if err != nil {
		t.Fatalf("InitRepository(, false, false, false, nil) failed: %v", err)
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

	// Verify MCP files were created by default
	mcpConfigPath := mcpConfigFilePath
	if _, err := os.Stat(mcpConfigPath); os.IsNotExist(err) {
		t.Error("Expected .github/mcp.json to be created by default")
	}

	setupStepsPath := filepath.Join(".github", "workflows", "copilot-setup-steps.yml")
	if _, err := os.Stat(setupStepsPath); os.IsNotExist(err) {
		t.Error("Expected .github/workflows/copilot-setup-steps.yml to be created by default")
	}

	skillPath := filepath.Join(".github", "skills", "agentic-workflows", "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Errorf("Expected dispatcher skill file to be created at %s", skillPath)
	}
	designerSkillPath := filepath.Join(".github", "skills", "agentic-workflow-designer", "SKILL.md")
	if _, err := os.Stat(designerSkillPath); os.IsNotExist(err) {
		t.Errorf("Expected workflow designer skill file to be created at %s", designerSkillPath)
	}

	agentPath := filepath.Join(".github", "agents", "agentic-workflows.md")
	agentContent, err := os.ReadFile(agentPath)
	if err != nil {
		t.Fatalf("Expected Agentic Workflows custom agent file to be created at %s: %v", agentPath, err)
	}
	if !strings.Contains(string(agentContent), "name: Agentic Workflows") {
		t.Error("Expected Agentic Workflows custom agent file to use the Agentic Workflows name")
	}
	if !strings.Contains(string(agentContent), ".github/aw/create-agentic-workflow.md") {
		t.Error("Expected Agentic Workflows custom agent file to include routing instructions")
	}
	if strings.Contains(string(agentContent), ".github/skills/agentic-workflows/SKILL.md") {
		t.Error("Expected Agentic Workflows custom agent file to avoid skill cross-references")
	}
}

func TestInitRepositoryWithMCP(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

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

	// Test init with MCP explicitly enabled (same as default)
	err = InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil})
	if err != nil {
		t.Fatalf("InitRepository(, false, false, false, nil) with MCP failed: %v", err)
	}

	// Verify .github/mcp.json was created
	mcpConfigPath := mcpConfigFilePath
	if _, err := os.Stat(mcpConfigPath); os.IsNotExist(err) {
		t.Error("Expected .github/mcp.json to be created")
	}

	// Verify copilot-setup-steps.yml was created
	setupStepsPath := filepath.Join(".github", "workflows", "copilot-setup-steps.yml")
	if _, err := os.Stat(setupStepsPath); os.IsNotExist(err) {
		t.Error("Expected .github/workflows/copilot-setup-steps.yml to be created")
	}
}

func TestInitRepositoryWithNoMCP(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

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

	// Test init with --no-mcp flag (mcp=false)
	err = InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: false, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil})
	if err != nil {
		t.Fatalf("InitRepository(, false, false, false, nil) with --no-mcp failed: %v", err)
	}

	// Verify .github/mcp.json was NOT created
	mcpConfigPath := mcpConfigFilePath
	if _, err := os.Stat(mcpConfigPath); err == nil {
		t.Error("Expected .github/mcp.json to NOT be created with --no-mcp flag")
	}

	// Verify copilot-setup-steps.yml was NOT created
	setupStepsPath := filepath.Join(".github", "workflows", "copilot-setup-steps.yml")
	if _, err := os.Stat(setupStepsPath); err == nil {
		t.Error("Expected .github/workflows/copilot-setup-steps.yml to NOT be created with --no-mcp flag")
	}

	// Verify basic files were still created
	if _, err := os.Stat(".gitattributes"); os.IsNotExist(err) {
		t.Error("Expected .gitattributes to be created even with --no-mcp flag")
	}
	if _, err := os.Stat(filepath.Join(".github", "skills", "agentic-workflows", "SKILL.md")); os.IsNotExist(err) {
		t.Error("Expected dispatcher skill file to still be created with --no-mcp flag")
	}
	if _, err := os.Stat(filepath.Join(".github", "skills", "agentic-workflow-designer", "SKILL.md")); os.IsNotExist(err) {
		t.Error("Expected workflow designer skill file to still be created with --no-mcp flag")
	}
	if _, err := os.Stat(filepath.Join(".github", "agents", "agentic-workflows.md")); os.IsNotExist(err) {
		t.Error("Expected Agentic Workflows custom agent file to still be created with --no-mcp flag")
	}
}

func TestInitRepositoryWithNoSkill(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

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

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	_ = exec.Command("git", "config", "user.name", "Test User").Run()
	_ = exec.Command("git", "config", "user.email", "test@example.com").Run()

	err = InitRepository(InitOptions{Verbose: false, Skill: false, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil})
	if err != nil {
		t.Fatalf("InitRepository() with no skill failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(".github", "skills", "agentic-workflows", "SKILL.md")); err == nil {
		t.Error("Expected dispatcher skill file to NOT be created with skill disabled")
	}
	if _, err := os.Stat(filepath.Join(".github", "skills", "agentic-workflow-designer", "SKILL.md")); err == nil {
		t.Error("Expected workflow designer skill file to NOT be created with skill disabled")
	}
	if _, err := os.Stat(filepath.Join(".github", "agents", "agentic-workflows.md")); os.IsNotExist(err) {
		t.Error("Expected Agentic Workflows custom agent file to still be created with skill disabled")
	}
}

func TestInitRepositoryWithNoAgent(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

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

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	_ = exec.Command("git", "config", "user.name", "Test User").Run()
	_ = exec.Command("git", "config", "user.email", "test@example.com").Run()

	err = InitRepository(InitOptions{Verbose: false, Skill: true, Agent: false, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil})
	if err != nil {
		t.Fatalf("InitRepository() with no agent failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(".github", "skills", "agentic-workflows", "SKILL.md")); os.IsNotExist(err) {
		t.Error("Expected dispatcher skill file to still be created with agent disabled")
	}
	if _, err := os.Stat(filepath.Join(".github", "skills", "agentic-workflow-designer", "SKILL.md")); os.IsNotExist(err) {
		t.Error("Expected workflow designer skill file to still be created with agent disabled")
	}
	if _, err := os.Stat(filepath.Join(".github", "agents", "agentic-workflows.md")); err == nil {
		t.Error("Expected Agentic Workflows custom agent file to NOT be created with agent disabled")
	}
}

func TestInitRepositoryWithNonCopilotEngineSkipsCopilotArtifacts(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

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

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	err = InitRepository(InitOptions{Verbose: false, Engine: "claude", Skill: true, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil})
	if err != nil {
		t.Fatalf("InitRepository with --engine claude failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(".github", "skills", "agentic-workflows", "SKILL.md")); err == nil {
		t.Error("Expected dispatcher skill file to NOT be created for non-Copilot engine")
	}
	if _, err := os.Stat(filepath.Join(".github", "skills", "agentic-workflow-designer", "SKILL.md")); err == nil {
		t.Error("Expected workflow designer skill file to NOT be created for non-Copilot engine")
	}
	if _, err := os.Stat(filepath.Join(".github", "agents", "agentic-workflows.md")); err == nil {
		t.Error("Expected Agentic Workflows custom agent file to NOT be created for non-Copilot engine")
	}
	if _, err := os.Stat(mcpConfigFilePath); err == nil {
		t.Error("Expected .github/mcp.json to NOT be created for non-Copilot engine")
	}
	if _, err := os.Stat(filepath.Join(".github", "workflows", "copilot-setup-steps.yml")); err == nil {
		t.Error("Expected copilot-setup-steps.yml to NOT be created for non-Copilot engine")
	}
	if _, err := os.Stat(".gitattributes"); os.IsNotExist(err) {
		t.Error("Expected .gitattributes to be created for non-Copilot engine")
	}
}

func TestInitRepositoryRemovesLegacyDispatcherAgentFile(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

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

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	_ = exec.Command("git", "config", "user.name", "Test User").Run()
	_ = exec.Command("git", "config", "user.email", "test@example.com").Run()

	legacyPath := filepath.Join(".github", "agents", "agentic-workflows.agent.md")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0755); err != nil {
		t.Fatalf("Failed to create legacy agent directory: %v", err)
	}
	if err := os.WriteFile(legacyPath, []byte("legacy dispatcher"), 0644); err != nil {
		t.Fatalf("Failed to create legacy agent file: %v", err)
	}

	err = InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil})
	if err != nil {
		t.Fatalf("InitRepository() failed: %v", err)
	}

	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("Expected legacy dispatcher agent file to be removed, got err=%v", err)
	}

	skillPath := filepath.Join(".github", "skills", "agentic-workflows", "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Fatalf("Expected dispatcher skill file to be created at %s", skillPath)
	}
	if _, err := os.Stat(filepath.Join(".github", "skills", "agentic-workflow-designer", "SKILL.md")); os.IsNotExist(err) {
		t.Fatalf("Expected workflow designer skill file to be created")
	}
}

func TestInitRepositoryWithMCPBackwardCompatibility(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

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

	// Test init with deprecated --mcp flag for backward compatibility (mcp=true)
	err = InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil})
	if err != nil {
		t.Fatalf("InitRepository(, false, false, false, nil) with deprecated --mcp flag failed: %v", err)
	}

	// Verify .github/mcp.json was created
	mcpConfigPath := mcpConfigFilePath
	if _, err := os.Stat(mcpConfigPath); os.IsNotExist(err) {
		t.Error("Expected .github/mcp.json to be created with --mcp flag (backward compatibility)")
	}

	// Verify copilot-setup-steps.yml was created
	setupStepsPath := filepath.Join(".github", "workflows", "copilot-setup-steps.yml")
	if _, err := os.Stat(setupStepsPath); os.IsNotExist(err) {
		t.Error("Expected .github/workflows/copilot-setup-steps.yml to be created with --mcp flag (backward compatibility)")
	}
}

func TestInitRepositoryVerbose(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

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

	// Test verbose mode with MCP enabled by default (should not error, just produce more output)
	err = InitRepository(InitOptions{Verbose: true, Skill: true, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil})
	if err != nil {
		t.Fatalf("InitRepository(, false, false, false, nil) in verbose mode failed: %v", err)
	}

	// Verify basic files were still created
	if _, err := os.Stat(".gitattributes"); os.IsNotExist(err) {
		t.Error("Expected .gitattributes to be created even in verbose mode")
	}
}

func TestInitRepositoryNotInGitRepo(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

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
	err = InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil})

	// The function should handle this gracefully or return an error
	// Based on the implementation, ensureGitAttributes requires git
	if err == nil {
		t.Log("InitRepository(, false, false, false, nil) succeeded despite not being in a git repo")
	}
}

func TestInitRepositoryIdempotent(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

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

	// Run init twice with MCP enabled by default
	err = InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil})
	if err != nil {
		t.Fatalf("First InitRepository(, false, false, false, nil) failed: %v", err)
	}

	// Second run should be idempotent
	err = InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil})
	if err != nil {
		t.Fatalf("Second InitRepository(, false, false, false, nil) failed: %v", err)
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
	tmpDir := testutil.TempDir(t, "test-*")

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
	err = InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil})
	if err != nil {
		t.Fatalf("First InitRepository(, false, false, false, nil) with MCP failed: %v", err)
	}

	err = InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil})
	if err != nil {
		t.Fatalf("Second InitRepository(, false, false, false, nil) with MCP failed: %v", err)
	}

	// Verify files still exist and are correct
	mcpConfigPath := mcpConfigFilePath
	if _, err := os.Stat(mcpConfigPath); os.IsNotExist(err) {
		t.Error("Expected .github/mcp.json to still exist after second run")
	}

	setupStepsPath := filepath.Join(".github", "workflows", "copilot-setup-steps.yml")
	if _, err := os.Stat(setupStepsPath); os.IsNotExist(err) {
		t.Error("Expected copilot-setup-steps.yml to still exist after second run")
	}
}

func TestInitRepositoryCreatesDirectories(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

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
	err = InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil})
	if err != nil {
		t.Fatalf("InitRepository(, false, false, false, nil) failed: %v", err)
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

	// Test that no-mcp flag is a boolean
	noMcpFlag := cmd.Flags().Lookup("no-mcp")
	if noMcpFlag == nil {
		t.Fatal("Expected 'no-mcp' flag to exist")
	}

	if noMcpFlag.Value.Type() != "bool" {
		t.Errorf("Expected no-mcp flag to be bool, got %s", noMcpFlag.Value.Type())
	}

	// Test verbose flag exists (inherited from parent command likely)
	// Note: verbose flag might be added by parent command, not in init command itself
}

func TestInitRepositoryErrorHandling(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

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

	// Test init without git repo (with MCP enabled by default)
	err = InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil})

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
	tmpDir := testutil.TempDir(t, "test-*")

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

	// Run init with MCP enabled by default
	err = InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil})
	if err != nil {
		t.Fatalf("InitRepository(, false, false, false, nil) failed: %v", err)
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

func TestInitRepositoryWithCodespace(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

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

	// Test init with --codespaces flag (with MCP enabled by default and additional repos)
	additionalRepos := []string{"org/repo1", "owner/repo2"}
	err = InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: true, CodespaceRepos: additionalRepos, CodespaceEnabled: true, Completions: false, CreatePR: false, RootCmd: nil})
	if err != nil {
		t.Fatalf("InitRepository(, false, false, false, nil) with codespaces failed: %v", err)
	}

	// Verify .devcontainer/devcontainer.json was created at default location
	devcontainerPath := filepath.Join(".devcontainer", "devcontainer.json")
	if _, err := os.Stat(devcontainerPath); os.IsNotExist(err) {
		t.Error("Expected .devcontainer/devcontainer.json to be created")
	}

	// Verify additional repos were added
	data, err := os.ReadFile(devcontainerPath)
	if err != nil {
		t.Fatalf("Failed to read devcontainer.json: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "org/repo1") {
		t.Error("Expected org/repo1 to be in devcontainer.json")
	}
	if !strings.Contains(content, "owner/repo2") {
		t.Error("Expected owner/repo2 to be in devcontainer.json")
	}

	// Verify basic files are still created
	gitAttributesPath := ".gitattributes"
	if _, err := os.Stat(gitAttributesPath); os.IsNotExist(err) {
		t.Error("Expected .gitattributes to be created")
	}
}

func TestInitCommandWithCodespacesNoArgs(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := testutil.TempDir(t, "test-*")

	// Save and restore original directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize a git repository
	err = exec.Command("git", "init").Run()
	if err != nil {
		t.Skip("Git not available")
	}

	// Create a mock git remote to test owner extraction
	err = exec.Command("git", "remote", "add", "origin", "https://github.com/testorg/testrepo.git").Run()
	if err != nil {
		t.Skip("Git not available")
	}

	// Configure git
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// Test init with --codespaces flag (no additional repos, MCP enabled by default)
	err = InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: true, Completions: false, CreatePR: false, RootCmd: nil})
	if err != nil {
		t.Fatalf("InitRepository(, false, false, false, nil) with codespaces (no args) failed: %v", err)
	}

	// Verify .devcontainer/devcontainer.json was created at default location
	devcontainerPath := filepath.Join(".devcontainer", "devcontainer.json")
	if _, err := os.Stat(devcontainerPath); os.IsNotExist(err) {
		t.Error("Expected .devcontainer/devcontainer.json to be created")
	}

	// Verify only current repo is configured
	data, err := os.ReadFile(devcontainerPath)
	if err != nil {
		t.Fatalf("Failed to read devcontainer.json: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "testorg/testrepo") {
		t.Error("Expected testorg/testrepo to be in devcontainer.json")
	}

	// Verify basic files are still created
	gitAttributesPath := ".gitattributes"
	if _, err := os.Stat(gitAttributesPath); os.IsNotExist(err) {
		t.Error("Expected .gitattributes to be created")
	}
}
