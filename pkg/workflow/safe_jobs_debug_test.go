package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractScriptFromSafeJob(t *testing.T) {
	c := NewCompiler(false, "", "test")

	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "test-workflow.md")

	// Create the workflow file (empty is fine for this test)
	if err := os.WriteFile(workflowPath, []byte("---\nname: Test\n---\nTest"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test extracting a script
	config := ExtractScriptConfig{
		WorkflowName: "Test Workflow",
		WorkflowPath: workflowPath,
		JobName:      "test_job",
		StepIndex:    0,
		StepName:     "Test Step",
		Script: `const fs = require('fs');
console.log('Hello from test script');`,
	}

	scriptPath, err := c.ExtractScriptFromSafeJob(config)
	if err != nil {
		t.Fatalf("Failed to extract script: %v", err)
	}

	// Verify the script file was created
	if scriptPath == "" {
		t.Fatal("Expected script path to be returned")
	}

	if !filepath.IsAbs(scriptPath) {
		t.Errorf("Expected absolute path, got: %s", scriptPath)
	}

	// Verify the file exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Fatalf("Script file was not created at: %s", scriptPath)
	}

	// Read the file and verify its contents
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("Failed to read script file: %v", err)
	}

	contentStr := string(content)

	// Check for header comment
	if !strings.Contains(contentStr, "GitHub Agentic Workflow Script") {
		t.Error("Script file should contain header comment")
	}

	if !strings.Contains(contentStr, "Workflow: Test Workflow") {
		t.Error("Script file should contain workflow name in header")
	}

	if !strings.Contains(contentStr, "Job: test_job") {
		t.Error("Script file should contain job name in header")
	}

	if !strings.Contains(contentStr, "Step: Test Step") {
		t.Error("Script file should contain step name in header")
	}

	// Check for the actual script content
	if !strings.Contains(contentStr, "const fs = require('fs');") {
		t.Error("Script file should contain the original script content")
	}

	if !strings.Contains(contentStr, "console.log('Hello from test script');") {
		t.Error("Script file should contain the original script content")
	}

	// Verify the filename follows the expected pattern
	expectedFilename := "test_job_0_test_step.cjs"
	if !strings.HasSuffix(scriptPath, expectedFilename) {
		t.Errorf("Expected filename to be %s, but path is %s", expectedFilename, scriptPath)
	}

	// Verify the script is in the correct directory
	expectedDir := filepath.Join(tmpDir, ".gh-aw", "scripts", "test_workflow")
	if !strings.Contains(scriptPath, expectedDir) {
		t.Errorf("Expected script to be in directory %s, but got %s", expectedDir, scriptPath)
	}
}

func TestExtractScriptFromSafeJobEmptyScript(t *testing.T) {
	c := NewCompiler(false, "", "test")

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "test-workflow.md")

	config := ExtractScriptConfig{
		WorkflowName: "Test Workflow",
		WorkflowPath: workflowPath,
		JobName:      "test_job",
		StepIndex:    0,
		StepName:     "Test Step",
		Script:       "", // Empty script
	}

	scriptPath, err := c.ExtractScriptFromSafeJob(config)
	if err != nil {
		t.Fatalf("Should not error on empty script: %v", err)
	}

	if scriptPath != "" {
		t.Error("Should return empty path for empty script")
	}
}

func TestExtractScriptFromStep(t *testing.T) {
	c := NewCompiler(false, "", "test")

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "test-workflow.md")

	// Create the workflow file
	if err := os.WriteFile(workflowPath, []byte("---\nname: Test\n---\nTest"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test with a github-script step
	stepMap := map[string]any{
		"name": "Test Script Step",
		"uses": "actions/github-script@v8",
		"with": map[string]any{
			"script": "console.log('test');",
		},
	}

	err := c.extractScriptFromStep(stepMap, "Test Workflow", workflowPath, "test_job", 0)
	if err != nil {
		t.Fatalf("Failed to extract script from step: %v", err)
	}

	// Verify the file was created
	expectedPath := filepath.Join(tmpDir, ".gh-aw", "scripts", "test_workflow", "test_job_0_test_script_step.cjs")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("Script file was not created at: %s", expectedPath)
	}

	// Read and verify content
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read script file: %v", err)
	}

	if !strings.Contains(string(content), "console.log('test');") {
		t.Error("Script file should contain the script content")
	}
}

func TestExtractScriptFromStepNonGitHubScript(t *testing.T) {
	c := NewCompiler(false, "", "test")

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "test-workflow.md")

	// Test with a non-github-script step
	stepMap := map[string]any{
		"name": "Test Run Step",
		"run":  "echo 'test'",
	}

	err := c.extractScriptFromStep(stepMap, "Test Workflow", workflowPath, "test_job", 0)
	if err != nil {
		t.Fatalf("Should not error on non-github-script step: %v", err)
	}

	// Verify no file was created
	scriptsDir := filepath.Join(tmpDir, ".gh-aw", "scripts")
	if _, err := os.Stat(scriptsDir); err == nil {
		// Directory exists, check if it's empty
		files, _ := os.ReadDir(scriptsDir)
		if len(files) > 0 {
			t.Error("Should not create script file for non-github-script steps")
		}
	}
}

func TestSanitizeForFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Simple Name",
			expected: "simple_name",
		},
		{
			input:    "Name With: Special*Characters?",
			expected: "name_with_special_characters",
		},
		{
			input:    "Name/With/Slashes",
			expected: "name_with_slashes",
		},
		{
			input:    "Multiple___Underscores",
			expected: "multiple_underscores",
		},
		{
			input:    "__Leading_Trailing__",
			expected: "leading_trailing",
		},
		{
			input:    ".Leading.Dots.",
			expected: "leading.dots",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeForFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeForFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
