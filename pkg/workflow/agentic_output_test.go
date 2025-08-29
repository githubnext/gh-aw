package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgenticOutputCollection(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "agentic-output-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case with agentic output collection for Claude engine
	testContent := `---
on: push
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [list_issues]
engine: claude
safe-outputs:
  add-issue-labels:
    allowed: ["bug", "enhancement"]
---

# Test Agentic Output Collection

This workflow tests the agentic output collection functionality.
`

	testFile := filepath.Join(tmpDir, "test-agentic-output.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow with agentic output: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-agentic-output.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify GITHUB_AW_OUTPUT functionality (should be present for all engines)
	if !strings.Contains(lockContent, "- name: Setup agent output") {
		t.Error("Expected 'Setup agent output' step to be in generated workflow")
	}

	if !strings.Contains(lockContent, "- name: Collect agent output") {
		t.Error("Expected 'Collect agent output' step to be in generated workflow")
	}

	if !strings.Contains(lockContent, "- name: Upload agentic output file") {
		t.Error("Expected 'Upload agentic output file' step to be in generated workflow")
	}

	// Verify job output declaration for GITHUB_AW_OUTPUT
	if !strings.Contains(lockContent, "outputs:\n      output: ${{ steps.collect_output.outputs.output }}") {
		t.Error("Expected job output declaration for 'output'")
	}

	// Verify GITHUB_AW_OUTPUT is passed to Claude
	if !strings.Contains(lockContent, "GITHUB_AW_OUTPUT: ${{ env.GITHUB_AW_OUTPUT }}") {
		t.Error("Expected GITHUB_AW_OUTPUT environment variable to be passed to engine")
	}

	// Verify prompt contains output instructions
	if !strings.Contains(lockContent, "## Adding Labels to Issues or Pull Requests") {
		t.Error("Expected output instructions to be injected into prompt")
	}

	// Verify Claude engine declared outputs are uploaded separately
	if !strings.Contains(lockContent, "- name: Upload engine output files") {
		t.Error("Expected 'Upload engine output files' step for Claude engine")
	}

	if !strings.Contains(lockContent, "name: agent_outputs") {
		t.Error("Expected engine output artifact to be named 'agent_outputs'")
	}

	// Verify that both artifacts are uploaded
	if !strings.Contains(lockContent, fmt.Sprintf("name: %s", OutputArtifactName)) {
		t.Errorf("Expected GITHUB_AW_OUTPUT artifact name to be '%s'", OutputArtifactName)
	}

	t.Log("Claude workflow correctly includes both GITHUB_AW_OUTPUT and engine output collection")
}

func TestCodexEngineNoOutputSteps(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "codex-no-output-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case with Codex engine (should have GITHUB_AW_OUTPUT but no engine output collection)
	testContent := `---
on: push
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [list_issues]
engine: codex
safe-outputs:
  add-issue-labels:
    allowed: ["bug", "enhancement"]
---

# Test Codex No Engine Output Collection

This workflow tests that Codex engine gets GITHUB_AW_OUTPUT but not engine output collection.
`

	testFile := filepath.Join(tmpDir, "test-codex-no-output.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Unexpected error compiling workflow with Codex: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-codex-no-output.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify that Codex workflow DOES have GITHUB_AW_OUTPUT functionality
	if !strings.Contains(lockContent, "- name: Setup agent output") {
		t.Error("Codex workflow should have 'Setup agent output' step (GITHUB_AW_OUTPUT functionality)")
	}

	if !strings.Contains(lockContent, "- name: Collect agent output") {
		t.Error("Codex workflow should have 'Collect agent output' step (GITHUB_AW_OUTPUT functionality)")
	}

	if !strings.Contains(lockContent, "- name: Upload agentic output file") {
		t.Error("Codex workflow should have 'Upload agentic output file' step (GITHUB_AW_OUTPUT functionality)")
	}

	if !strings.Contains(lockContent, "GITHUB_AW_OUTPUT") {
		t.Error("Codex workflow should reference GITHUB_AW_OUTPUT environment variable")
	}

	if !strings.Contains(lockContent, fmt.Sprintf("name: %s", OutputArtifactName)) {
		t.Errorf("Codex workflow should reference %s artifact (GITHUB_AW_OUTPUT)", OutputArtifactName)
	}

	// Verify that job outputs section includes output for GITHUB_AW_OUTPUT
	if !strings.Contains(lockContent, "outputs:\n      output: ${{ steps.collect_output.outputs.output }}") {
		t.Error("Codex workflow should have job output declaration for 'output' (GITHUB_AW_OUTPUT)")
	}

	// Verify that Codex workflow does NOT have engine output collection steps
	if strings.Contains(lockContent, "- name: Collect engine output files") {
		t.Error("Codex workflow should NOT have 'Collect engine output files' step")
	}

	if strings.Contains(lockContent, "- name: Upload engine output files") {
		t.Error("Codex workflow should NOT have 'Upload engine output files' step")
	}

	if strings.Contains(lockContent, "name: agent_outputs") {
		t.Error("Codex workflow should NOT reference 'agent_outputs' artifact")
	}

	// Verify that the Codex execution step is still present
	if !strings.Contains(lockContent, "- name: Run Codex") {
		t.Error("Expected 'Run Codex' step to be in generated workflow")
	}

	t.Log("Codex workflow correctly includes GITHUB_AW_OUTPUT functionality but excludes engine output collection")
}

func TestEngineOutputFileDeclarations(t *testing.T) {
	// Test Claude engine declares output files
	claudeEngine := NewClaudeEngine()
	claudeOutputFiles := claudeEngine.GetDeclaredOutputFiles()

	if len(claudeOutputFiles) == 0 {
		t.Error("Claude engine should declare at least one output file")
	}

	if !stringSliceContains(claudeOutputFiles, "output.txt") {
		t.Errorf("Claude engine should declare 'output.txt' as an output file, got: %v", claudeOutputFiles)
	}

	// Test Codex engine declares no output files
	codexEngine := NewCodexEngine()
	codexOutputFiles := codexEngine.GetDeclaredOutputFiles()

	if len(codexOutputFiles) != 0 {
		t.Errorf("Codex engine should declare no output files, got: %v", codexOutputFiles)
	}

	t.Logf("Claude engine declares: %v", claudeOutputFiles)
	t.Logf("Codex engine declares: %v", codexOutputFiles)
}

// Helper function to check if a string slice contains a specific string
func stringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
