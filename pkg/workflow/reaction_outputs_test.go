package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReactionJobOutputs tests that the add_reaction job includes comment outputs
func TestReactionJobOutputs(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "reaction-outputs-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file with reaction
	testContent := `---
on:
  issues:
    types: [opened]
  pull_request:
    types: [opened]
  reaction: eyes
permissions:
  contents: read
  issues: write
  pull-requests: write
tools:
  github:
    allowed: [get_issue]
---

# Test Reaction Outputs

This workflow should generate add_reaction job with comment outputs.
`

	testFile := filepath.Join(tmpDir, "test-reaction-outputs.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Generate YAML
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check for reaction job outputs
	expectedOutputs := []string{
		"reaction_id:",
		"comment_id:",
		"comment_url:",
	}

	for _, expected := range expectedOutputs {
		if !strings.Contains(yamlContent, expected) {
			t.Errorf("Generated YAML does not contain expected output: %s", expected)
		}
	}

	// Verify the outputs reference the react step
	if !strings.Contains(yamlContent, "steps.react.outputs.reaction-id") {
		t.Error("Generated YAML should contain reaction-id output reference")
	}
	if !strings.Contains(yamlContent, "steps.react.outputs.comment-id") {
		t.Error("Generated YAML should contain comment-id output reference")
	}
	if !strings.Contains(yamlContent, "steps.react.outputs.comment-url") {
		t.Error("Generated YAML should contain comment-url output reference")
	}
}
