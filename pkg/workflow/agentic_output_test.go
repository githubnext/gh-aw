package workflow

import (
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

	// Test case with agentic output collection
	testContent := `---
on: push
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [list_issues]
engine: claude
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

	// Verify pre-step: Setup agentic output file step exists
	if !strings.Contains(lockContent, "- name: Setup agentic output file") {
		t.Error("Expected 'Setup agentic output file' step to be in generated workflow")
	}

	// Verify the step uses github-script and sets up the output file
	if !strings.Contains(lockContent, "uses: actions/github-script@v7") {
		t.Error("Expected github-script action to be used for output file setup")
	}

	if !strings.Contains(lockContent, "const outputFile = `/tmp/aw_output_${randomId}.txt`;") {
		t.Error("Expected output file creation in setup step")
	}

	if !strings.Contains(lockContent, "fs2.appendFileSync(process.env.GITHUB_ENV, `GITHUB_AW_OUTPUT=${outputFile}\\n`);") {
		t.Error("Expected GITHUB_AW_OUTPUT environment variable to be set")
	}

	// Verify prompt injection: Check for output instructions in the prompt
	if !strings.Contains(lockContent, "**IMPORTANT**: If you need to provide output that should be captured as a workflow output variable") {
		t.Error("Expected output instructions to be injected into prompt")
	}

	if !strings.Contains(lockContent, "write it to the file specified by the environment variable GITHUB_AW_OUTPUT") {
		t.Error("Expected GITHUB_AW_OUTPUT instructions in prompt")
	}

	// Verify environment variable is passed to agentic engine
	if !strings.Contains(lockContent, "env:\n          GITHUB_AW_OUTPUT: ${{ env.GITHUB_AW_OUTPUT }}") {
		t.Error("Expected GITHUB_AW_OUTPUT environment variable to be passed to agentic engine")
	}

	// Verify post-step: Collect agentic output step exists
	if !strings.Contains(lockContent, "- name: Collect agentic output") {
		t.Error("Expected 'Collect agentic output' step to be in generated workflow")
	}

	if !strings.Contains(lockContent, "id: collect_output") {
		t.Error("Expected collect_output step ID")
	}

	if !strings.Contains(lockContent, "const outputFile = process.env.GITHUB_AW_OUTPUT;") {
		t.Error("Expected output file reading in collection step")
	}

	if !strings.Contains(lockContent, "core.setOutput('output', outputContent.trim());") {
		t.Error("Expected output to be set in collection step")
	}

	// Verify job output declaration
	if !strings.Contains(lockContent, "outputs:\n      output: ${{ steps.collect_output.outputs.output }}") {
		t.Error("Expected job output declaration for 'output'")
	}

	// Verify step order: setup should come before agentic execution, collection should come after
	setupIndex := strings.Index(lockContent, "- name: Setup agentic output file")
	executeIndex := strings.Index(lockContent, "- name: Execute Claude Code Action")
	collectIndex := strings.Index(lockContent, "- name: Collect agentic output")

	if setupIndex == -1 || executeIndex == -1 || collectIndex == -1 {
		t.Fatal("Could not find expected steps in generated workflow")
	}

	if setupIndex >= executeIndex {
		t.Error("Setup step should appear before agentic execution step")
	}

	if collectIndex <= executeIndex {
		t.Error("Collection step should appear after agentic execution step")
	}

	t.Logf("Step order verified: Setup (%d) < Execute (%d) < Collect (%d)",
		setupIndex, executeIndex, collectIndex)
}
