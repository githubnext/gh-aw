package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixCommand_TimeoutMinutesMigration(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	workflowFile := filepath.Join(tmpDir, "test-workflow.md")

	// Create a workflow with deprecated timeout_minutes field
	content := `---
on:
  workflow_dispatch:

timeout_minutes: 30

permissions:
  contents: read
---

# Test Workflow

This is a test workflow.
`

	if err := os.WriteFile(workflowFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run the codemod
	codemods := GetAllCodemods()
	var timeoutCodemod *Codemod
	for _, cm := range codemods {
		if cm.ID == "timeout-minutes-migration" {
			timeoutCodemod = &cm
			break
		}
	}

	if timeoutCodemod == nil {
		t.Fatal("timeout-minutes-migration codemod not found")
	}

	// Process the file
	fixed, err := processWorkflowFile(workflowFile, []Codemod{*timeoutCodemod}, true, false)
	if err != nil {
		t.Fatalf("Failed to process workflow file: %v", err)
	}

	if !fixed {
		t.Error("Expected file to be fixed, but no changes were made")
	}

	// Read the updated content
	updatedContent, err := os.ReadFile(workflowFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	updatedStr := string(updatedContent)

	// Verify the change
	if strings.Contains(updatedStr, "timeout_minutes:") {
		t.Error("Expected timeout_minutes to be replaced, but it still exists")
	}

	if !strings.Contains(updatedStr, "timeout-minutes: 30") {
		t.Errorf("Expected timeout-minutes: 30 in updated content, got:\n%s", updatedStr)
	}
}

func TestFixCommand_NoChangesNeeded(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	workflowFile := filepath.Join(tmpDir, "test-workflow.md")

	// Create a workflow with no deprecated fields
	content := `---
on:
  workflow_dispatch:

timeout-minutes: 30

permissions:
  contents: read
---

# Test Workflow

This is a test workflow.
`

	if err := os.WriteFile(workflowFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run all codemods
	codemods := GetAllCodemods()

	// Process the file
	fixed, err := processWorkflowFile(workflowFile, codemods, false, false)
	if err != nil {
		t.Fatalf("Failed to process workflow file: %v", err)
	}

	if fixed {
		t.Error("Expected no changes, but file was marked as fixed")
	}

	// Read the content to verify it's unchanged
	updatedContent, err := os.ReadFile(workflowFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(updatedContent) != content {
		t.Error("Expected content to be unchanged")
	}
}

func TestFixCommand_NetworkFirewallMigration(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	workflowFile := filepath.Join(tmpDir, "test-workflow.md")

	// Create a workflow with deprecated network.firewall field
	content := `---
on:
  workflow_dispatch:

network:
  allowed:
    - "*.example.com"
  firewall: null

permissions:
  contents: read
---

# Test Workflow

This is a test workflow.
`

	if err := os.WriteFile(workflowFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run the codemod
	codemods := GetAllCodemods()
	var firewallCodemod *Codemod
	for _, cm := range codemods {
		if cm.ID == "network-firewall-migration" {
			firewallCodemod = &cm
			break
		}
	}

	if firewallCodemod == nil {
		t.Fatal("network-firewall-migration codemod not found")
	}

	// Process the file
	fixed, err := processWorkflowFile(workflowFile, []Codemod{*firewallCodemod}, true, false)
	if err != nil {
		t.Fatalf("Failed to process workflow file: %v", err)
	}

	if !fixed {
		t.Error("Expected file to be fixed, but no changes were made")
	}

	// Read the updated content
	updatedContent, err := os.ReadFile(workflowFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	updatedStr := string(updatedContent)

	// Verify the change
	if strings.Contains(updatedStr, "firewall:") {
		t.Error("Expected firewall field to be removed, but it still exists")
	}

	if !strings.Contains(updatedStr, "sandbox:") {
		t.Errorf("Expected sandbox field to be added, got:\n%s", updatedStr)
	}

	if !strings.Contains(updatedStr, "agent: false") {
		t.Errorf("Expected agent: false in updated content, got:\n%s", updatedStr)
	}
}

func TestFixCommand_PreservesFormatting(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	workflowFile := filepath.Join(tmpDir, "test-workflow.md")

	// Create a workflow with comments and specific formatting
	content := `---
on:
  workflow_dispatch:

# Timeout configuration
timeout_minutes: 30  # 30 minutes should be enough

permissions:
  contents: read
---

# Test Workflow

This is a test workflow.
`

	if err := os.WriteFile(workflowFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run the timeout migration codemod
	codemods := GetAllCodemods()
	var timeoutCodemod *Codemod
	for _, cm := range codemods {
		if cm.ID == "timeout-minutes-migration" {
			timeoutCodemod = &cm
			break
		}
	}

	if timeoutCodemod == nil {
		t.Fatal("timeout-minutes-migration codemod not found")
	}

	// Process the file
	fixed, err := processWorkflowFile(workflowFile, []Codemod{*timeoutCodemod}, true, false)
	if err != nil {
		t.Fatalf("Failed to process workflow file: %v", err)
	}

	if !fixed {
		t.Error("Expected file to be fixed, but no changes were made")
	}

	// Read the updated content
	updatedContent, err := os.ReadFile(workflowFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	updatedStr := string(updatedContent)

	// Verify the comment is preserved
	if !strings.Contains(updatedStr, "# 30 minutes should be enough") {
		t.Error("Expected inline comment to be preserved")
	}

	// Verify the block comment is preserved
	if !strings.Contains(updatedStr, "# Timeout configuration") {
		t.Error("Expected block comment to be preserved")
	}

	// Verify the field was changed
	if !strings.Contains(updatedStr, "timeout-minutes: 30") {
		t.Errorf("Expected timeout-minutes: 30 in updated content, got:\n%s", updatedStr)
	}
}

func TestGetAllCodemods(t *testing.T) {
	codemods := GetAllCodemods()

	if len(codemods) == 0 {
		t.Fatal("Expected at least one codemod, got none")
	}

	// Check for required codemods
	expectedIDs := []string{
		"timeout-minutes-migration",
		"network-firewall-migration",
	}

	foundIDs := make(map[string]bool)
	for _, cm := range codemods {
		foundIDs[cm.ID] = true

		// Verify each codemod has required fields
		if cm.ID == "" {
			t.Error("Codemod has empty ID")
		}
		if cm.Name == "" {
			t.Error("Codemod has empty Name")
		}
		if cm.Description == "" {
			t.Error("Codemod has empty Description")
		}
		if cm.Apply == nil {
			t.Error("Codemod has nil Apply function")
		}
	}

	for _, expectedID := range expectedIDs {
		if !foundIDs[expectedID] {
			t.Errorf("Expected codemod with ID %s not found", expectedID)
		}
	}
}
