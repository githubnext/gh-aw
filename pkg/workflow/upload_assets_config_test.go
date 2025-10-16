package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUploadAssetsConfigDefaults(t *testing.T) {
	compiler := NewCompiler(false, "", "")

	// Test default configuration
	outputMap := map[string]any{
		"upload-assets": nil,
	}

	config := compiler.parseUploadAssetConfig(outputMap)
	if config == nil {
		t.Fatal("Expected config to be created with defaults")
	}

	// Check default extensions match problem statement requirement
	expectedExts := []string{".png", ".jpg", ".jpeg"}
	if len(config.AllowedExts) != len(expectedExts) {
		t.Errorf("Expected %d default extensions, got %d", len(expectedExts), len(config.AllowedExts))
	}

	for i, ext := range expectedExts {
		if i >= len(config.AllowedExts) || config.AllowedExts[i] != ext {
			t.Errorf("Expected extension %s at position %d, got %v", ext, i, config.AllowedExts)
		}
	}

	// Check default max size
	if config.MaxSizeKB != 10240 {
		t.Errorf("Expected default max size 10240, got %d", config.MaxSizeKB)
	}
}

func TestUploadAssetsConfigCustomExtensions(t *testing.T) {
	compiler := NewCompiler(false, "", "")

	// Test custom configuration like dev.md
	outputMap := map[string]any{
		"upload-assets": map[string]any{
			"allowed-exts": []any{".txt"},
			"max-size":     1024,
		},
	}

	config := compiler.parseUploadAssetConfig(outputMap)
	if config == nil {
		t.Fatal("Expected config to be created")
	}

	// Check custom extensions
	expectedExts := []string{".txt"}
	if len(config.AllowedExts) != len(expectedExts) {
		t.Errorf("Expected %d custom extensions, got %d", len(expectedExts), len(config.AllowedExts))
	}

	if config.AllowedExts[0] != ".txt" {
		t.Errorf("Expected custom extension .txt, got %s", config.AllowedExts[0])
	}

	// Check custom max size
	if config.MaxSizeKB != 1024 {
		t.Errorf("Expected custom max size 1024, got %d", config.MaxSizeKB)
	}
}

func TestUploadAssetsBranchNameNormalizationInWorkflow(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a workflow with upload-assets that uses github.workflow expression
	// which could have spaces (like "Documentation Unbloat")
	workflowContent := `---
on: workflow_dispatch
engine: claude
safe-outputs:
  upload-assets:
---

# Test Workflow

This workflow tests branch name normalization.
`

	workflowFile := filepath.Join(tmpDir, "test-workflow.md")
	err := os.WriteFile(workflowFile, []byte(workflowContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "")
	err = compiler.CompileWorkflow(workflowFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := strings.TrimSuffix(workflowFile, ".md") + ".lock.yml"
	compiled, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	compiledStr := string(compiled)

	// Verify that the bash normalization step is present
	// The normalization is now done in Go (via bash step), not in JavaScript
	if !strings.Contains(compiledStr, "Normalize branch name") {
		t.Error("Expected compiled workflow to include 'Normalize branch name' step")
	}

	// Verify that the normalization uses sed to remove special characters
	if !strings.Contains(compiledStr, "sed 's/[^a-zA-Z0-9/_-]//g'") {
		t.Error("Expected compiled workflow to include sed normalization command")
	}

	// Verify that the branch name is passed through environment variable from the normalize step
	if !strings.Contains(compiledStr, "GITHUB_AW_ASSETS_BRANCH: ${{ steps.normalize_branch.outputs.branch }}") {
		t.Error("Expected compiled workflow to use normalized branch from step output")
	}

	// Verify that the default branch name uses github.workflow in the RAW_BRANCH
	if !strings.Contains(compiledStr, "RAW_BRANCH=\"assets/${{ github.workflow }}\"") {
		t.Error("Expected RAW_BRANCH to use github.workflow expression")
	}
}
