package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestEnsureDevcontainerConfig(t *testing.T) {
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

	// Initialize git repo (required for getCurrentRepoName)
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	// Configure git
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// Test creating devcontainer.json
	err = ensureDevcontainerConfig(false, []string{})
	if err != nil {
		t.Fatalf("ensureDevcontainerConfig() failed: %v", err)
	}

	// Verify .devcontainer/devcontainer.json was created
	devcontainerPath := filepath.Join(".devcontainer", "gh-aw", "devcontainer.json")
	if _, err := os.Stat(devcontainerPath); os.IsNotExist(err) {
		t.Fatal("Expected .devcontainer/devcontainer.json to be created")
	}

	// Read and parse the created file
	data, err := os.ReadFile(devcontainerPath)
	if err != nil {
		t.Fatalf("Failed to read devcontainer.json: %v", err)
	}

	var config DevcontainerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse devcontainer.json: %v", err)
	}

	// Verify basic structure
	if config.Name == "" {
		t.Error("Expected name to be set")
	}

	if config.Image != "mcr.microsoft.com/devcontainers/universal:latest" {
		t.Errorf("Expected universal image, got %q", config.Image)
	}

	// Verify Codespaces configuration
	if config.Customizations == nil || config.Customizations.Codespaces == nil {
		t.Fatal("Expected Codespaces customizations to be set")
	}

	// Verify VSCode extensions
	if config.Customizations.VSCode == nil {
		t.Fatal("Expected VSCode customizations to be set")
	}

	extensions := config.Customizations.VSCode.Extensions
	hasGitHubCopilot := false
	hasCopilotChat := false
	for _, ext := range extensions {
		if ext == "GitHub.copilot" {
			hasGitHubCopilot = true
		}
		if ext == "GitHub.copilot-chat" {
			hasCopilotChat = true
		}
	}

	if !hasGitHubCopilot {
		t.Error("Expected GitHub.copilot extension to be included")
	}

	if !hasCopilotChat {
		t.Error("Expected GitHub.copilot-chat extension to be included")
	}

	// Verify GitHub CLI feature
	if config.Features == nil {
		t.Fatal("Expected features to be set")
	}

	if _, exists := config.Features["ghcr.io/devcontainers/features/github-cli:1"]; !exists {
		t.Error("Expected GitHub CLI feature to be included")
	}

	// Verify Copilot CLI feature
	if _, exists := config.Features["ghcr.io/devcontainers/features/copilot-cli:latest"]; !exists {
		t.Error("Expected Copilot CLI feature to be included")
	}

	// Verify postCreateCommand
	if config.PostCreateCommand == "" {
		t.Error("Expected postCreateCommand to be set")
	}

	// Test that running again doesn't fail (idempotency)
	err = ensureDevcontainerConfig(false, []string{})
	if err != nil {
		t.Fatalf("ensureDevcontainerConfig() should be idempotent, but failed: %v", err)
	}
}

func TestEnsureDevcontainerConfigWithAdditionalRepos(t *testing.T) {
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

	// Test creating devcontainer.json with additional repos
	additionalRepos := []string{"org/additional-repo1", "owner/additional-repo2"}
	err = ensureDevcontainerConfig(false, additionalRepos)
	if err != nil {
		t.Fatalf("ensureDevcontainerConfig() failed: %v", err)
	}

	// Read and parse the created file
	devcontainerPath := filepath.Join(".devcontainer", "gh-aw", "devcontainer.json")
	data, err := os.ReadFile(devcontainerPath)
	if err != nil {
		t.Fatalf("Failed to read devcontainer.json: %v", err)
	}

	var config DevcontainerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse devcontainer.json: %v", err)
	}

	// Verify additional repos are included
	if config.Customizations == nil || config.Customizations.Codespaces == nil {
		t.Fatal("Expected Codespaces customizations to be set")
	}

	if _, exists := config.Customizations.Codespaces.Repositories["org/additional-repo1"]; !exists {
		t.Error("Expected org/additional-repo1 to be in repositories")
	}

	if _, exists := config.Customizations.Codespaces.Repositories["owner/additional-repo2"]; !exists {
		t.Error("Expected owner/additional-repo2 to be in repositories")
	}

	// Verify read permissions for additional repos
	repo1 := config.Customizations.Codespaces.Repositories["org/additional-repo1"]
	if repo1.Permissions["contents"] != "read" {
		t.Errorf("Expected contents: read for org/additional-repo1, got %q", repo1.Permissions["contents"])
	}

	repo2 := config.Customizations.Codespaces.Repositories["owner/additional-repo2"]
	if repo2.Permissions["contents"] != "read" {
		t.Errorf("Expected contents: read for owner/additional-repo2, got %q", repo2.Permissions["contents"])
	}
}

func TestEnsureDevcontainerConfigWithCurrentRepo(t *testing.T) {
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

	// Test creating devcontainer.json
	err = ensureDevcontainerConfig(false, []string{})
	if err != nil {
		t.Fatalf("ensureDevcontainerConfig() failed: %v", err)
	}

	// Read and parse the created file
	devcontainerPath := filepath.Join(".devcontainer", "gh-aw", "devcontainer.json")
	data, err := os.ReadFile(devcontainerPath)
	if err != nil {
		t.Fatalf("Failed to read devcontainer.json: %v", err)
	}

	var config DevcontainerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse devcontainer.json: %v", err)
	}

	// Verify that current repo has workflows: write permission
	if config.Customizations == nil || config.Customizations.Codespaces == nil {
		t.Fatal("Expected Codespaces customizations to be set")
	}

	// Check if any repository has workflows: write (should be current repo)
	hasWorkflowsWrite := false
	for _, repo := range config.Customizations.Codespaces.Repositories {
		if repo.Permissions["workflows"] == "write" {
			hasWorkflowsWrite = true
			break
		}
	}

	if !hasWorkflowsWrite {
		t.Error("Expected at least one repository to have workflows: write permission")
	}
}

func TestEnsureDevcontainerConfigWithOwnerValidation(t *testing.T) {
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

	// Add remote with specific owner
	exec.Command("git", "remote", "add", "origin", "https://github.com/testowner/testrepo.git").Run()

	// Test that same owner succeeds
	err = ensureDevcontainerConfig(false, []string{"testowner/repo1"})
	if err != nil {
		t.Fatalf("ensureDevcontainerConfig() with same owner should succeed: %v", err)
	}

	// Clean up for next test
	os.RemoveAll(filepath.Join(".devcontainer", "gh-aw"))

	// Test that different owner fails
	err = ensureDevcontainerConfig(false, []string{"differentowner/repo2"})
	if err == nil {
		t.Fatal("ensureDevcontainerConfig() with different owner should fail")
	}

	// Check that error message contains expected text
	if err.Error() == "" || len(err.Error()) < 10 {
		t.Errorf("Expected meaningful error message, got: %v", err)
	}
}

func TestEnsureDevcontainerConfigUpdatesOldVersion(t *testing.T) {
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

	// Create .devcontainer/gh-aw directory
	devcontainerDir := filepath.Join(".devcontainer", "gh-aw")
	if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Create a devcontainer.json with old copilot-cli version
	oldConfig := DevcontainerConfig{
		Name:  "Agentic Workflows Development",
		Image: "mcr.microsoft.com/devcontainers/universal:latest",
		Features: DevcontainerFeatures{
			"ghcr.io/devcontainers/features/github-cli:1":  map[string]any{},
			"ghcr.io/devcontainers/features/copilot-cli:1": map[string]any{}, // Old version
		},
		PostCreateCommand: "curl -fsSL https://raw.githubusercontent.com/githubnext/gh-aw/refs/heads/main/install-gh-aw.sh | bash",
	}

	devcontainerPath := filepath.Join(devcontainerDir, "devcontainer.json")
	data, err := json.MarshalIndent(oldConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal old config: %v", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(devcontainerPath, data, 0644); err != nil {
		t.Fatalf("Failed to write old config: %v", err)
	}

	// Run ensureDevcontainerConfig - should update the version
	err = ensureDevcontainerConfig(false, []string{})
	if err != nil {
		t.Fatalf("ensureDevcontainerConfig() failed: %v", err)
	}

	// Read and verify the updated config
	updatedData, err := os.ReadFile(devcontainerPath)
	if err != nil {
		t.Fatalf("Failed to read updated config: %v", err)
	}

	var updatedConfig DevcontainerConfig
	if err := json.Unmarshal(updatedData, &updatedConfig); err != nil {
		t.Fatalf("Failed to parse updated config: %v", err)
	}

	// Verify copilot-cli was updated to :latest
	if _, exists := updatedConfig.Features["ghcr.io/devcontainers/features/copilot-cli:latest"]; !exists {
		t.Error("Expected copilot-cli feature to be updated to :latest")
	}

	// Verify old version is gone
	if _, exists := updatedConfig.Features["ghcr.io/devcontainers/features/copilot-cli:1"]; exists {
		t.Error("Expected old copilot-cli:1 version to be removed")
	}
}

func TestGetCurrentRepoName(t *testing.T) {
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

	// Get repo name
	repoName := getCurrentRepoName()
	if repoName == "" {
		t.Error("Expected getCurrentRepoName() to return a non-empty string")
	}
}
