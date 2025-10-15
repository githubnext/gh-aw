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

	// Verify the outputs reference the react step - now in activation job
	if !strings.Contains(yamlContent, "steps.react.outputs.reaction-id") {
		t.Error("Generated YAML should contain reaction-id output reference")
	}
	if !strings.Contains(yamlContent, "steps.react.outputs.comment-id") {
		t.Error("Generated YAML should contain comment-id output reference")
	}
	if !strings.Contains(yamlContent, "steps.react.outputs.comment-url") {
		t.Error("Generated YAML should contain comment-url output reference")
	}

	// Verify reaction step is in activation job, not a separate add_reaction job
	if strings.Contains(yamlContent, "add_reaction:") {
		t.Error("Generated YAML should not contain separate add_reaction job")
	}
}

// TestReactionJobWorkflowName tests that the add_reaction job includes GITHUB_AW_WORKFLOW_NAME environment variable
func TestReactionJobWorkflowName(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "reaction-workflow-name-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file with reaction and a specific workflow name
	testContent := `---
name: Test Workflow Name
on:
  issues:
    types: [opened]
  reaction: rocket
permissions:
  contents: read
  issues: write
  pull-requests: write
---

# Test Workflow

This workflow should generate add_reaction job with GITHUB_AW_WORKFLOW_NAME environment variable.
`

	testFile := filepath.Join(tmpDir, "test-reaction-workflow-name.md")
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

	// Check that GITHUB_AW_WORKFLOW_NAME is set
	if !strings.Contains(yamlContent, "GITHUB_AW_WORKFLOW_NAME:") {
		t.Error("Generated YAML should contain GITHUB_AW_WORKFLOW_NAME environment variable")
	}

	// Verify the workflow name is correctly set
	if !strings.Contains(yamlContent, `GITHUB_AW_WORKFLOW_NAME: "Test Workflow Name"`) {
		t.Error("Generated YAML should contain the correct workflow name value")
	}

	// Ensure it's in the activation job section (not a separate add_reaction job)
	// Find the activation job section (must be exact match, not pre_activation)
	activationJobStart := strings.Index(yamlContent, "\n  activation:")
	if activationJobStart == -1 {
		// Try from the beginning in case it's the first job
		if strings.HasPrefix(yamlContent, "  activation:") {
			activationJobStart = 0
		} else {
			t.Fatal("Could not find activation job in generated YAML")
		}
	} else {
		activationJobStart += 1 // Skip the leading newline
	}

	// Find the next job or end of file
	nextJobStart := len(yamlContent)
	lines := strings.Split(yamlContent[activationJobStart:], "\n")
	for i, line := range lines[1:] {
		if strings.HasPrefix(line, "  ") && strings.HasSuffix(line, ":") && !strings.HasPrefix(line, "    ") {
			nextJobStart = activationJobStart + strings.Index(yamlContent[activationJobStart:], lines[i+1])
			break
		}
	}

	activationJobSection := yamlContent[activationJobStart:nextJobStart]

	// Verify GITHUB_AW_WORKFLOW_NAME is in the activation job section
	if !strings.Contains(activationJobSection, "GITHUB_AW_WORKFLOW_NAME:") {
		t.Errorf("GITHUB_AW_WORKFLOW_NAME should be in the activation job section\n%s", activationJobSection)
	}

	// Verify no separate add_reaction job exists
	if strings.Contains(yamlContent, "add_reaction:") {
		t.Error("Generated YAML should not contain separate add_reaction job")
	}
}
