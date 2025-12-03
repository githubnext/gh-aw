package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestInitRepository_WithCodespace(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := testutil.TempDir(t, "test-*")

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

	// Initialize git repo with a remote
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := exec.Command("git", "remote", "add", "origin", "https://github.com/test/example-repo.git").Run(); err != nil {
		t.Fatalf("Failed to add git remote: %v", err)
	}

	// Call the function with Codespace flag
	err = InitRepository(false, false, true)
	if err != nil {
		t.Fatalf("InitRepository() with Codespace returned error: %v", err)
	}

	// Verify standard files were created
	gitAttributesPath := filepath.Join(tempDir, ".gitattributes")
	if _, err := os.Stat(gitAttributesPath); os.IsNotExist(err) {
		t.Errorf("Expected .gitattributes file to exist")
	}

	// Verify devcontainer.json was created
	devcontainerPath := filepath.Join(tempDir, ".devcontainer", "devcontainer.json")
	if _, err := os.Stat(devcontainerPath); os.IsNotExist(err) {
		t.Errorf("Expected .devcontainer/devcontainer.json to exist")
	} else {
		// Verify content contains key elements
		content, err := os.ReadFile(devcontainerPath)
		if err != nil {
			t.Fatalf("Failed to read devcontainer.json: %v", err)
		}

		var config Devcontainer
		if err := json.Unmarshal(content, &config); err != nil {
			t.Fatalf("Failed to parse devcontainer.json: %v", err)
		}

		// Verify basic structure
		if config.Image == "" {
			t.Errorf("Expected image to be set in devcontainer.json")
		}

		if config.Customizations == nil {
			t.Fatalf("Expected customizations to exist in devcontainer.json")
		}

		if config.Customizations.Codespaces == nil {
			t.Fatalf("Expected codespaces section to exist in devcontainer.json")
		}

		repos := config.Customizations.Codespaces.Repositories
		if repos == nil {
			t.Fatalf("Expected repositories to exist in codespaces section")
		}

		// Verify current repository has correct permissions
		currentRepo, exists := repos["test/example-repo"]
		if !exists {
			t.Errorf("Expected test/example-repo to be in repositories")
		} else {
			perms := currentRepo.Permissions
			if perms.Actions != "write" {
				t.Errorf("Expected actions permission to be 'write', got '%s'", perms.Actions)
			}
			if perms.Contents != "write" {
				t.Errorf("Expected contents permission to be 'write', got '%s'", perms.Contents)
			}
			if perms.Workflows != "write" {
				t.Errorf("Expected workflows permission to be 'write', got '%s'", perms.Workflows)
			}
			if perms.Issues != "write" {
				t.Errorf("Expected issues permission to be 'write', got '%s'", perms.Issues)
			}
			if perms.PullRequests != "write" {
				t.Errorf("Expected pull-requests permission to be 'write', got '%s'", perms.PullRequests)
			}
			if perms.Discussions != "write" {
				t.Errorf("Expected discussions permission to be 'write', got '%s'", perms.Discussions)
			}
		}

		// Verify githubnext/gh-aw has read permissions
		ghAwRepo, exists := repos["githubnext/gh-aw"]
		if !exists {
			t.Errorf("Expected githubnext/gh-aw to be in repositories")
		} else {
			perms := ghAwRepo.Permissions
			if perms.Contents != "read" {
				t.Errorf("Expected contents permission to be 'read' for gh-aw, got '%s'", perms.Contents)
			}
			if perms.Metadata != "read" {
				t.Errorf("Expected metadata permission to be 'read' for gh-aw, got '%s'", perms.Metadata)
			}
		}
	}
}

func TestInitRepository_Codespace_Idempotent(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := testutil.TempDir(t, "test-*")

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

	// Initialize git repo with a remote
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := exec.Command("git", "remote", "add", "origin", "https://github.com/test/example-repo.git").Run(); err != nil {
		t.Fatalf("Failed to add git remote: %v", err)
	}

	// Call the function first time with Codespace
	err = InitRepository(false, false, true)
	if err != nil {
		t.Fatalf("InitRepository() with Codespace returned error on first call: %v", err)
	}

	// Call the function second time with Codespace
	err = InitRepository(false, false, true)
	if err != nil {
		t.Fatalf("InitRepository() with Codespace returned error on second call: %v", err)
	}

	// Verify files still exist
	devcontainerPath := filepath.Join(tempDir, ".devcontainer", "devcontainer.json")
	if _, err := os.Stat(devcontainerPath); os.IsNotExist(err) {
		t.Errorf("Expected devcontainer.json to exist after second call")
	}
}

func TestEnsureDevcontainerCodespace_UpdatesExisting(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := testutil.TempDir(t, "test-*")

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

	// Initialize git repo with a remote
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := exec.Command("git", "remote", "add", "origin", "https://github.com/test/example-repo.git").Run(); err != nil {
		t.Fatalf("Failed to add git remote: %v", err)
	}

	// Create .devcontainer directory
	if err := os.MkdirAll(".devcontainer", 0755); err != nil {
		t.Fatalf("Failed to create .devcontainer directory: %v", err)
	}

	// Create initial devcontainer.json with existing content
	initialConfig := Devcontainer{
		Image: "mcr.microsoft.com/devcontainers/go:1-bookworm",
		Customizations: &DevcontainerCustomizations{
			Codespaces: &DevcontainerCodespaces{
				Repositories: map[string]DevcontainerRepositoryConfig{
					"other-org/other-repo": {
						Permissions: DevcontainerRepositoryPermissions{
							Contents: "read",
						},
					},
				},
			},
		},
	}
	initialData, _ := json.MarshalIndent(initialConfig, "", "\t")
	devcontainerPath := filepath.Join(tempDir, ".devcontainer", "devcontainer.json")
	if err := os.WriteFile(devcontainerPath, initialData, 0644); err != nil {
		t.Fatalf("Failed to write initial devcontainer.json: %v", err)
	}

	// Call ensureDevcontainerCodespace
	if err := ensureDevcontainerCodespace(false); err != nil {
		t.Fatalf("ensureDevcontainerCodespace() returned error: %v", err)
	}

	// Verify the config now contains both repositories
	content, err := os.ReadFile(devcontainerPath)
	if err != nil {
		t.Fatalf("Failed to read devcontainer.json: %v", err)
	}

	var config Devcontainer
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatalf("Failed to parse devcontainer.json: %v", err)
	}

	repos := config.Customizations.Codespaces.Repositories

	// Check existing repository is preserved
	if _, exists := repos["other-org/other-repo"]; !exists {
		t.Errorf("Expected existing 'other-org/other-repo' to be preserved")
	}

	// Check current repository was added
	if _, exists := repos["test/example-repo"]; !exists {
		t.Errorf("Expected 'test/example-repo' to be added")
	}

	// Check gh-aw repository was added
	if _, exists := repos["githubnext/gh-aw"]; !exists {
		t.Errorf("Expected 'githubnext/gh-aw' to be added")
	}

	// Verify original image is preserved
	if config.Image != "mcr.microsoft.com/devcontainers/go:1-bookworm" {
		t.Errorf("Expected original image to be preserved, got '%s'", config.Image)
	}
}

func TestEnsureDevcontainerCodespace_PreservesExistingPermissions(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := testutil.TempDir(t, "test-*")

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

	// Initialize git repo with a remote
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := exec.Command("git", "remote", "add", "origin", "https://github.com/test/example-repo.git").Run(); err != nil {
		t.Fatalf("Failed to add git remote: %v", err)
	}

	// Create .devcontainer directory
	if err := os.MkdirAll(".devcontainer", 0755); err != nil {
		t.Fatalf("Failed to create .devcontainer directory: %v", err)
	}

	// Create initial devcontainer.json with current repo having some permissions
	initialConfig := Devcontainer{
		Image: "mcr.microsoft.com/devcontainers/universal:2",
		Customizations: &DevcontainerCustomizations{
			Codespaces: &DevcontainerCodespaces{
				Repositories: map[string]DevcontainerRepositoryConfig{
					"test/example-repo": {
						Permissions: DevcontainerRepositoryPermissions{
							Contents: "read", // Should remain read
							Actions:  "write",
						},
					},
				},
			},
		},
	}
	initialData, _ := json.MarshalIndent(initialConfig, "", "\t")
	devcontainerPath := filepath.Join(tempDir, ".devcontainer", "devcontainer.json")
	if err := os.WriteFile(devcontainerPath, initialData, 0644); err != nil {
		t.Fatalf("Failed to write initial devcontainer.json: %v", err)
	}

	// Call ensureDevcontainerCodespace
	if err := ensureDevcontainerCodespace(false); err != nil {
		t.Fatalf("ensureDevcontainerCodespace() returned error: %v", err)
	}

	// Verify the config
	content, err := os.ReadFile(devcontainerPath)
	if err != nil {
		t.Fatalf("Failed to read devcontainer.json: %v", err)
	}

	var config Devcontainer
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatalf("Failed to parse devcontainer.json: %v", err)
	}

	repos := config.Customizations.Codespaces.Repositories
	currentRepo := repos["test/example-repo"]

	// Verify existing permissions were preserved
	if currentRepo.Permissions.Contents != "read" {
		t.Errorf("Expected existing contents permission 'read' to be preserved, got '%s'", currentRepo.Permissions.Contents)
	}

	// Verify new permissions were added
	if currentRepo.Permissions.Workflows != "write" {
		t.Errorf("Expected workflows permission to be added as 'write', got '%s'", currentRepo.Permissions.Workflows)
	}
}

func TestInitRepository_CodespaceVerbose(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := testutil.TempDir(t, "test-*")

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

	// Initialize git repo with a remote
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := exec.Command("git", "remote", "add", "origin", "https://github.com/test/example-repo.git").Run(); err != nil {
		t.Fatalf("Failed to add git remote: %v", err)
	}

	// Call the function with verbose=true and codespace=true (should not error)
	err = InitRepository(true, false, true)
	if err != nil {
		t.Fatalf("InitRepository() returned error with verbose=true and codespace=true: %v", err)
	}

	// Verify files were created
	devcontainerPath := filepath.Join(tempDir, ".devcontainer", "devcontainer.json")
	if _, err := os.Stat(devcontainerPath); os.IsNotExist(err) {
		t.Errorf("Expected devcontainer.json to exist with verbose=true and codespace=true")
	}
}

func TestEnsureDevcontainerCodespace_CreatesBasicConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := testutil.TempDir(t, "test-*")

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

	// Initialize git repo with a remote
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}
	if err := exec.Command("git", "remote", "add", "origin", "https://github.com/test/example-repo.git").Run(); err != nil {
		t.Fatalf("Failed to add git remote: %v", err)
	}

	// Call ensureDevcontainerCodespace when no devcontainer exists
	if err := ensureDevcontainerCodespace(false); err != nil {
		t.Fatalf("ensureDevcontainerCodespace() returned error: %v", err)
	}

	// Verify the config was created with basic content
	devcontainerPath := filepath.Join(tempDir, ".devcontainer", "devcontainer.json")
	content, err := os.ReadFile(devcontainerPath)
	if err != nil {
		t.Fatalf("Failed to read devcontainer.json: %v", err)
	}

	var config Devcontainer
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatalf("Failed to parse devcontainer.json: %v", err)
	}

	// Verify basic image was set
	if !strings.Contains(config.Image, "mcr.microsoft.com/devcontainers") {
		t.Errorf("Expected Microsoft devcontainer image, got '%s'", config.Image)
	}

	// Verify repositories were added
	repos := config.Customizations.Codespaces.Repositories
	if len(repos) < 2 {
		t.Errorf("Expected at least 2 repositories, got %d", len(repos))
	}
}
