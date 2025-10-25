package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/goccy/go-yaml"
)

func TestCompileDependabotIntegration(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("failed to create workflows directory: %v", err)
	}

	// Change to temp directory
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tempDir)

	// Initialize git repo
	initGitRepo(t, tempDir)

	// Create a test workflow with npm dependencies
	workflowContent := `---
on: push
permissions:
  contents: read
steps:
  - run: npx @playwright/mcp@latest --help
---

# Test Workflow

This workflow uses npx to run Playwright MCP.
`
	workflowPath := filepath.Join(workflowsDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("failed to write workflow file: %v", err)
	}

	// Compile with Dependabot flag
	config := CompileConfig{
		MarkdownFiles:  []string{workflowPath},
		Verbose:        true,
		Validate:       false, // Skip validation for faster test
		WorkflowDir:    ".github/workflows",
		Dependabot:     true,
		ForceOverwrite: false,
		Strict:         false,
	}

	workflowDataList, err := CompileWorkflows(config)
	if err != nil {
		t.Fatalf("compilation failed: %v", err)
	}

	if len(workflowDataList) != 1 {
		t.Fatalf("expected 1 workflow, got %d", len(workflowDataList))
	}

	// Verify package.json was created
	packageJSONPath := filepath.Join(workflowsDir, "package.json")
	if _, err := os.Stat(packageJSONPath); os.IsNotExist(err) {
		t.Fatal("package.json was not created")
	}

	// Verify package.json content
	packageData, err := os.ReadFile(packageJSONPath)
	if err != nil {
		t.Fatalf("failed to read package.json: %v", err)
	}

	var pkgJSON workflow.PackageJSON
	if err := json.Unmarshal(packageData, &pkgJSON); err != nil {
		t.Fatalf("failed to parse package.json: %v", err)
	}

	if pkgJSON.Name != "gh-aw-workflows-deps" {
		t.Errorf("expected name 'gh-aw-workflows-deps', got %q", pkgJSON.Name)
	}

	if len(pkgJSON.Dependencies) == 0 {
		t.Error("expected at least one dependency (@playwright/mcp)")
	}

	// Verify package-lock.json was created
	packageLockPath := filepath.Join(workflowsDir, "package-lock.json")
	if _, err := os.Stat(packageLockPath); os.IsNotExist(err) {
		t.Error("package-lock.json was not created")
	}

	// Verify dependabot.yml was created
	dependabotPath := filepath.Join(tempDir, ".github", "dependabot.yml")
	if _, err := os.Stat(dependabotPath); os.IsNotExist(err) {
		t.Fatal("dependabot.yml was not created")
	}

	// Verify dependabot.yml content
	dependabotData, err := os.ReadFile(dependabotPath)
	if err != nil {
		t.Fatalf("failed to read dependabot.yml: %v", err)
	}

	var dependabotConfig workflow.DependabotConfig
	if err := yaml.Unmarshal(dependabotData, &dependabotConfig); err != nil {
		t.Fatalf("failed to parse dependabot.yml: %v", err)
	}

	if dependabotConfig.Version != 2 {
		t.Errorf("expected version 2, got %d", dependabotConfig.Version)
	}

	npmFound := false
	for _, update := range dependabotConfig.Updates {
		if update.PackageEcosystem == "npm" && update.Directory == "/.github/workflows" {
			npmFound = true
			if update.Schedule.Interval != "weekly" {
				t.Errorf("expected interval 'weekly', got %q", update.Schedule.Interval)
			}
			break
		}
	}

	if !npmFound {
		t.Error("npm ecosystem not found in dependabot.yml")
	}
}

func TestCompileDependabotNoDependencies(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("failed to create workflows directory: %v", err)
	}

	// Change to temp directory
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tempDir)

	// Initialize git repo
	initGitRepo(t, tempDir)

	// Create a test workflow WITHOUT npm dependencies
	workflowContent := `---
on: push
permissions:
  contents: read
---

# Test Workflow

This workflow does not use npm.
`
	workflowPath := filepath.Join(workflowsDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("failed to write workflow file: %v", err)
	}

	// Compile with Dependabot flag
	config := CompileConfig{
		MarkdownFiles:  []string{workflowPath},
		Verbose:        true,
		Validate:       false,
		WorkflowDir:    ".github/workflows",
		Dependabot:     true,
		ForceOverwrite: false,
		Strict:         false,
	}

	_, err := CompileWorkflows(config)
	if err != nil {
		t.Fatalf("compilation failed: %v", err)
	}

	// Verify package.json was NOT created (no dependencies)
	packageJSONPath := filepath.Join(workflowsDir, "package.json")
	if _, err := os.Stat(packageJSONPath); !os.IsNotExist(err) {
		t.Error("package.json should not be created when there are no npm dependencies")
	}

	// Verify dependabot.yml was NOT created (no dependencies)
	dependabotPath := filepath.Join(tempDir, ".github", "dependabot.yml")
	if _, err := os.Stat(dependabotPath); !os.IsNotExist(err) {
		t.Error("dependabot.yml should not be created when there are no npm dependencies")
	}
}

func TestCompileDependabotPreserveExisting(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	workflowsDir := filepath.Join(tempDir, ".github", "workflows")
	githubDir := filepath.Join(tempDir, ".github")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("failed to create workflows directory: %v", err)
	}

	// Change to temp directory
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tempDir)

	// Initialize git repo
	initGitRepo(t, tempDir)

	// Create existing dependabot.yml with custom config
	existingDependabot := workflow.DependabotConfig{
		Version: 2,
		Updates: []workflow.DependabotUpdateEntry{
			{
				PackageEcosystem: "pip",
				Directory:        "/",
			},
		},
	}
	existingDependabot.Updates[0].Schedule.Interval = "daily"

	dependabotPath := filepath.Join(githubDir, "dependabot.yml")
	dependabotData, _ := yaml.Marshal(&existingDependabot)
	if err := os.WriteFile(dependabotPath, dependabotData, 0644); err != nil {
		t.Fatalf("failed to write existing dependabot.yml: %v", err)
	}

	// Create a test workflow with npm dependencies
	workflowContent := `---
on: push
permissions:
  contents: read
steps:
  - run: npx @playwright/mcp@latest --help
---

# Test Workflow
`
	workflowPath := filepath.Join(workflowsDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("failed to write workflow file: %v", err)
	}

	// Compile with Dependabot flag (without force)
	config := CompileConfig{
		MarkdownFiles:  []string{workflowPath},
		Verbose:        true,
		Validate:       false,
		WorkflowDir:    ".github/workflows",
		Dependabot:     true,
		ForceOverwrite: false, // Don't force overwrite
		Strict:         false,
	}

	_, err := CompileWorkflows(config)
	if err != nil {
		t.Fatalf("compilation failed: %v", err)
	}

	// Verify existing dependabot.yml was preserved
	dependabotData, err = os.ReadFile(dependabotPath)
	if err != nil {
		t.Fatalf("failed to read dependabot.yml: %v", err)
	}

	var dependabotConfig workflow.DependabotConfig
	if err := yaml.Unmarshal(dependabotData, &dependabotConfig); err != nil {
		t.Fatalf("failed to parse dependabot.yml: %v", err)
	}

	// Should still have the original pip entry
	pipFound := false
	for _, update := range dependabotConfig.Updates {
		if update.PackageEcosystem == "pip" {
			pipFound = true
			break
		}
	}

	if !pipFound {
		t.Error("existing pip ecosystem should be preserved")
	}
}

// Helper function to initialize a git repo for testing
func initGitRepo(t *testing.T, dir string) {
	// Create .git directory to simulate a git repo
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git directory: %v", err)
	}
}
