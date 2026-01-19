//go:build integration

package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

// TestCompileWorkflows_PurgeCampaignLockFiles tests that campaign .g.lock.yml files
// are NOT purged by the general lock file purge function
func TestCompileWorkflows_PurgeCampaignLockFiles(t *testing.T) {
	// Create temporary directory structure for testing
	tempDir := testutil.TempDir(t, "test-purge-campaign-lock-*")
	workflowsDir := filepath.Join(tempDir, ".github/workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Change to temp directory to simulate being in a git repo
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .git directory and initialize it properly
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = tempDir
	if err := gitCmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	// Configure git for the test
	exec.Command("git", "-C", tempDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", tempDir, "config", "user.name", "Test User").Run()

	// Create a valid campaign definition file
	campaignMd := filepath.Join(workflowsDir, "test-campaign.campaign.md")
	campaignContent := `---
id: test-campaign
name: Test Campaign
project-url: "https://github.com/orgs/example/projects/1"
version: v1
---

# Test Campaign

This is a test campaign.`

	if err := os.WriteFile(campaignMd, []byte(campaignContent), 0644); err != nil {
		t.Fatalf("Failed to create campaign file: %v", err)
	}

	// Create a regular workflow file
	workflowMd := filepath.Join(workflowsDir, "regular-workflow.md")
	workflowContent := `---
name: Regular Workflow
on: push
engine: copilot
---

# Regular Workflow

This is a regular workflow.`

	if err := os.WriteFile(workflowMd, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create workflow file: %v", err)
	}

	// Create a campaign orchestrator lock file (generated from .campaign.md)
	campaignLockYml := filepath.Join(workflowsDir, "test-campaign.campaign.lock.yml")
	campaignLockContent := `name: Test Campaign Orchestrator
on:
  schedule:
    - cron: "0 0 * * *"
jobs:
  orchestrate:
    runs-on: ubuntu-latest
    steps:
      - run: echo "campaign"`

	if err := os.WriteFile(campaignLockYml, []byte(campaignLockContent), 0644); err != nil {
		t.Fatalf("Failed to create campaign lock file: %v", err)
	}

	// Create a regular workflow lock file
	workflowLockYml := filepath.Join(workflowsDir, "regular-workflow.lock.yml")
	workflowLockContent := `name: Regular Workflow
on:
  push:
    branches: [main]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "regular"`

	if err := os.WriteFile(workflowLockYml, []byte(workflowLockContent), 0644); err != nil {
		t.Fatalf("Failed to create workflow lock file: %v", err)
	}

	// Create an orphaned regular lock file (no corresponding .md file)
	orphanedLockYml := filepath.Join(workflowsDir, "orphaned-workflow.lock.yml")
	orphanedLockContent := `name: Orphaned Workflow
on:
  push:
    branches: [main]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "orphaned"`

	if err := os.WriteFile(orphanedLockYml, []byte(orphanedLockContent), 0644); err != nil {
		t.Fatalf("Failed to create orphaned lock file: %v", err)
	}

	// Create agentic-campaign-generator source in .github/workflows and its lock file.
	// This file should NOT be purged when --purge is used.
	generatorMd := filepath.Join(workflowsDir, "agentic-campaign-generator.md")
	generatorContent := `---
name: "Agentic Campaign Generator"
on:
  issues:
    types: [labeled]
engine: copilot
---

# Agentic Campaign Generator
`
	if err := os.WriteFile(generatorMd, []byte(generatorContent), 0644); err != nil {
		t.Fatalf("Failed to create generator source file: %v", err)
	}

	generatorLockYml := filepath.Join(workflowsDir, "agentic-campaign-generator.lock.yml")
	generatorLockContent := `name: Agentic Campaign Generator
on:
  issues:
    types: [labeled]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "generator"`
	if err := os.WriteFile(generatorLockYml, []byte(generatorLockContent), 0644); err != nil {
		t.Fatalf("Failed to create generator lock file: %v", err)
	}

	// Verify files exist before purge
	if _, err := os.Stat(campaignLockYml); os.IsNotExist(err) {
		t.Fatal("Campaign lock file should exist before purge")
	}
	if _, err := os.Stat(workflowLockYml); os.IsNotExist(err) {
		t.Fatal("Regular workflow lock file should exist before purge")
	}
	if _, err := os.Stat(orphanedLockYml); os.IsNotExist(err) {
		t.Fatal("Orphaned lock file should exist before purge")
	}
	if _, err := os.Stat(generatorLockYml); os.IsNotExist(err) {
		t.Fatal("Generator lock file should exist before purge")
	}

	// Run compilation with purge flag
	config := CompileConfig{
		MarkdownFiles: []string{}, // Empty to compile all files
		Verbose:       true,       // Enable verbose to see what's happening
		NoEmit:        true,       // Skip emit to avoid validation issues
		Purge:         true,
		WorkflowDir:   "",
		Validate:      false, // Skip validation to avoid test failures
	}

	// Compile workflows with purge enabled
	result, err := CompileWorkflows(context.Background(), config)
	if err != nil {
		t.Logf("Compilation error (expected): %v", err)
	}
	if result != nil {
		t.Logf("Compilation completed with %d results", len(result))
	}

	// Verify campaign lock file was NOT purged (even though it's a .lock.yml file)
	// Campaign .g.lock.yml files should only be handled by purgeOrphanedCampaignOrchestratorLockFiles
	if _, err := os.Stat(campaignLockYml); os.IsNotExist(err) {
		t.Error("Campaign .g.lock.yml file should NOT be purged by general lock file purge")
	}

	// Verify regular workflow lock file was NOT purged (has source .md)
	if _, err := os.Stat(workflowLockYml); os.IsNotExist(err) {
		t.Error("Regular workflow lock file should NOT be purged (has source .md)")
	}

	// Verify orphaned lock file WAS purged (no source .md)
	if _, err := os.Stat(orphanedLockYml); !os.IsNotExist(err) {
		t.Error("Orphaned lock file should have been purged")
	}

	// Verify agentic-campaign-generator lock file was NOT purged (source exists)
	if _, err := os.Stat(generatorLockYml); os.IsNotExist(err) {
		t.Error("agentic-campaign-generator.lock.yml should NOT be purged when source exists")
	}

	// Verify source files still exist
	if _, err := os.Stat(campaignMd); os.IsNotExist(err) {
		t.Error("Campaign .md file should still exist")
	}
	if _, err := os.Stat(workflowMd); os.IsNotExist(err) {
		t.Error("Regular workflow .md file should still exist")
	}
}
