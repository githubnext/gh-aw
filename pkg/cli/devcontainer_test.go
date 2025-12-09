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
	err = ensureDevcontainerConfig(false)
	if err != nil {
		t.Fatalf("ensureDevcontainerConfig() failed: %v", err)
	}

	// Verify .devcontainer/devcontainer.json was created
	devcontainerPath := filepath.Join(".devcontainer", "devcontainer.json")
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

	// Verify gh-aw repository permissions
	ghAwRepo, exists := config.Customizations.Codespaces.Repositories["githubnext/gh-aw"]
	if !exists {
		t.Fatal("Expected githubnext/gh-aw repository to be configured")
	}

	if ghAwRepo.Permissions["contents"] != "read" {
		t.Errorf("Expected contents: read permission for githubnext/gh-aw, got %q", ghAwRepo.Permissions["contents"])
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

	// Verify postCreateCommand
	if config.PostCreateCommand == "" {
		t.Error("Expected postCreateCommand to be set")
	}

	// Test that running again doesn't fail (idempotency)
	err = ensureDevcontainerConfig(false)
	if err != nil {
		t.Fatalf("ensureDevcontainerConfig() should be idempotent, but failed: %v", err)
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
	err = ensureDevcontainerConfig(false)
	if err != nil {
		t.Fatalf("ensureDevcontainerConfig() failed: %v", err)
	}

	// Read and parse the created file
	devcontainerPath := filepath.Join(".devcontainer", "devcontainer.json")
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
