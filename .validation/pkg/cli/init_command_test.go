//go:build !integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestNewInitCommand(t *testing.T) {
	t.Parallel()

	cmd := NewInitCommand()
	if cmd == nil {
		t.Fatal("NewInitCommand() returned nil")
	}

	assertInitCommandMetadata(t, cmd)
	assertInitCommandFlags(t, cmd)
}

func assertInitCommandMetadata(t *testing.T, cmd *cobra.Command) {
	t.Helper()

	if cmd.Use != "init" {
		t.Errorf("Expected Use to be 'init', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("Expected Short description to be set")
	}
	if cmd.Long == "" {
		t.Error("Expected Long description to be set")
	}
}

func assertInitCommandFlags(t *testing.T, cmd *cobra.Command) {
	t.Helper()

	noMcpFlag := requireInitCommandFlag(t, cmd, "no-mcp")
	noSkillFlag := requireInitCommandFlag(t, cmd, "no-skill")
	noAgentFlag := requireInitCommandFlag(t, cmd, "no-agent")
	mcpFlag := requireInitCommandFlag(t, cmd, "mcp")
	engineFlag := requireInitCommandFlag(t, cmd, "engine")
	codespaceFlag := requireInitCommandFlag(t, cmd, "codespaces")
	_ = requireInitCommandFlag(t, cmd, "create-pull-request")
	prFlag := requireInitCommandFlag(t, cmd, "pr")

	if engineFlag.Hidden {
		t.Error("Expected 'engine' flag to be visible")
	}
	if !mcpFlag.Hidden {
		t.Error("Expected 'mcp' flag to be hidden")
	}
	if !prFlag.Hidden {
		t.Error("Expected 'pr' flag to be hidden")
	}
	assertInitCommandFlagDefault(t, noMcpFlag, "false")
	assertInitCommandFlagDefault(t, noSkillFlag, "false")
	assertInitCommandFlagDefault(t, noAgentFlag, "false")
	assertInitCommandFlagDefault(t, mcpFlag, "false")
	if !strings.Contains(codespaceFlag.Usage, "or use with an empty value for the current repo only") {
		t.Errorf("Expected codespaces flag help text to include article fixes, got %q", codespaceFlag.Usage)
	}
	if codespaceFlag.DefValue != "" {
		t.Errorf("Expected codespaces flag default to be '', got %q", codespaceFlag.DefValue)
	}
	if codespaceFlag.NoOptDefVal != "" {
		t.Errorf("Expected codespaces flag NoOptDefVal to be '' (empty), got %q", codespaceFlag.NoOptDefVal)
	}
}

func requireInitCommandFlag(t *testing.T, cmd *cobra.Command, name string) *pflag.Flag {
	t.Helper()

	flag := cmd.Flags().Lookup(name)
	if flag == nil {
		t.Fatalf("Expected %q flag to be defined", name)
	}
	return flag
}

func assertInitCommandFlagDefault(t *testing.T, flag *pflag.Flag, expected string) {
	t.Helper()
	if flag.DefValue != expected {
		t.Errorf("Expected %s flag default to be %q, got %q", flag.Name, expected, flag.DefValue)
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
	restore := chdirInitTestDir(t, tmpDir)
	defer restore()
	initInitTestRepo(t)

	if err := InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: false, Completions: false, CreatePR: false, RootCmd: nil}); err != nil {
		t.Fatalf("InitRepository(, false, false, false, nil) failed: %v", err)
	}

	assertBasicInitFiles(t)
	assertAgenticWorkflowBootstrapFiles(t)
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
	restore := chdirInitTestDir(t, tmpDir)
	defer restore()
	initInitTestRepo(t)

	additionalRepos := []string{"org/repo1", "owner/repo2"}
	if err := InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: true, CodespaceRepos: additionalRepos, CodespaceEnabled: true, Completions: false, CreatePR: false, RootCmd: nil}); err != nil {
		t.Fatalf("InitRepository(, false, false, false, nil) with codespaces failed: %v", err)
	}

	assertDevcontainerContains(t, additionalRepos)
	assertBasicInitFiles(t)
}

func chdirInitTestDir(t *testing.T, dir string) func() {
	t.Helper()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	return func() { _ = os.Chdir(originalDir) }
}

func initInitTestRepo(t *testing.T) {
	t.Helper()

	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}
	if err := exec.Command("git", "config", "user.name", "Test User").Run(); err != nil {
		t.Fatalf("Failed to set git user.name: %v", err)
	}
	if err := exec.Command("git", "config", "user.email", "test@example.com").Run(); err != nil {
		t.Fatalf("Failed to set git user.email: %v", err)
	}
}

func assertBasicInitFiles(t *testing.T) {
	t.Helper()

	content, err := os.ReadFile(".gitattributes")
	if err != nil {
		t.Fatalf("Failed to read .gitattributes: %v", err)
	}
	if !strings.Contains(string(content), ".github/workflows/*.lock.yml linguist-generated=true merge=ours") {
		t.Errorf("Expected .gitattributes to contain %q", ".github/workflows/*.lock.yml linguist-generated=true merge=ours")
	}
}

func assertAgenticWorkflowBootstrapFiles(t *testing.T) {
	t.Helper()

	for _, path := range []string{mcpConfigFilePath, filepath.Join(".github", "workflows", "copilot-setup-steps.yml"), filepath.Join(".github", "skills", "agentic-workflows", "SKILL.md")} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected %s to be created", path)
		}
	}
	agentContent, err := os.ReadFile(filepath.Join(".github", "agents", "agentic-workflows.md"))
	if err != nil {
		t.Fatalf("Expected Agentic Workflows custom agent file to be created: %v", err)
	}
	if !strings.Contains(string(agentContent), "name: Agentic Workflows") {
		t.Error("Expected Agentic Workflows custom agent file to use the Agentic Workflows name")
	}
	if !strings.Contains(string(agentContent), "`.github/skills/agentic-workflows/SKILL.md`") {
		t.Error("Expected Agentic Workflows custom agent file to reference the dispatcher skill path")
	}
	if strings.Contains(string(agentContent), ".github/aw/") {
		t.Error("Expected generic init repositories without .github/aw prompts to omit .github/aw prompt entries")
	}
}

func assertDevcontainerContains(t *testing.T, expectedRepos []string) {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(".devcontainer", "devcontainer.json"))
	if err != nil {
		t.Fatalf("Failed to read devcontainer.json: %v", err)
	}
	for _, repo := range expectedRepos {
		if !strings.Contains(string(content), repo) {
			t.Errorf("Expected %s to be in devcontainer.json", repo)
		}
	}
}

func TestInitCommandWithCodespacesNoArgs(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")
	restore := chdirInitTestDir(t, tempDir)
	defer restore()
	initInitTestRepo(t)

	if err := exec.Command("git", "remote", "add", "origin", "https://github.com/testorg/testrepo.git").Run(); err != nil {
		t.Skip("Git not available")
	}
	if err := InitRepository(InitOptions{Verbose: false, Skill: true, Agent: true, MCP: true, CodespaceRepos: []string{}, CodespaceEnabled: true, Completions: false, CreatePR: false, RootCmd: nil}); err != nil {
		t.Fatalf("InitRepository(, false, false, false, nil) with codespaces (no args) failed: %v", err)
	}

	assertDevcontainerContains(t, []string{"testorg/testrepo"})
	assertBasicInitFiles(t)
}
