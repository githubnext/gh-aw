package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDefaultsSectionExtraction tests that the defaults section is correctly extracted from frontmatter
func TestDefaultsSectionExtraction(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	frontmatter := map[string]any{
		"on": map[string]any{
			"push": nil,
		},
		"permissions": map[string]any{
			"contents": "read",
		},
		"defaults": map[string]any{
			"run": map[string]any{
				"shell":             "bash",
				"working-directory": "./src",
			},
		},
	}

	result := compiler.extractTopLevelYAMLSection(frontmatter, "defaults")

	// Verify the result contains the expected structure
	if !strings.Contains(result, "defaults:") {
		t.Errorf("Expected result to contain 'defaults:', got: %s", result)
	}
	if !strings.Contains(result, "run:") {
		t.Errorf("Expected result to contain 'run:', got: %s", result)
	}
	if !strings.Contains(result, "shell: bash") {
		t.Errorf("Expected result to contain 'shell: bash', got: %s", result)
	}
	if !strings.Contains(result, "working-directory: ./src") {
		t.Errorf("Expected result to contain 'working-directory: ./src', got: %s", result)
	}
}

// TestDefaultsSectionEmission tests that the defaults section is correctly emitted in the compiled workflow
func TestDefaultsSectionEmission(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "defaults-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file with defaults section
	testContent := `---
on: push
permissions:
  contents: read
defaults:
  run:
    shell: bash
    working-directory: ./src
---

# Test Defaults Workflow

This is a test workflow with defaults section.
`

	testFile := filepath.Join(tmpDir, "test-defaults.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContent := string(content)

	// Verify the defaults section is present
	if !strings.Contains(lockContent, "defaults:") {
		t.Error("Expected lock file to contain 'defaults:'")
	}
	if !strings.Contains(lockContent, "run:") {
		t.Error("Expected lock file to contain 'run:'")
	}
	if !strings.Contains(lockContent, "shell: bash") {
		t.Error("Expected lock file to contain 'shell: bash'")
	}
	if !strings.Contains(lockContent, "working-directory: ./src") {
		t.Error("Expected lock file to contain 'working-directory: ./src'")
	}

	// Verify defaults appears in the correct position (after run-name, before jobs)
	runNameIndex := strings.Index(lockContent, "run-name:")
	defaultsIndex := strings.Index(lockContent, "defaults:")
	jobsIndex := strings.Index(lockContent, "jobs:")

	if runNameIndex == -1 {
		t.Error("Expected lock file to contain 'run-name:'")
	}
	if defaultsIndex == -1 {
		t.Error("Expected lock file to contain 'defaults:'")
	}
	if jobsIndex == -1 {
		t.Error("Expected lock file to contain 'jobs:'")
	}

	if runNameIndex > 0 && defaultsIndex > 0 && jobsIndex > 0 {
		if defaultsIndex < runNameIndex {
			t.Error("Expected 'defaults:' to appear after 'run-name:'")
		}
		if defaultsIndex > jobsIndex {
			t.Error("Expected 'defaults:' to appear before 'jobs:'")
		}
	}
}

// TestDefaultsSectionWithNoDefaults tests that workflows without defaults still compile correctly
func TestDefaultsSectionWithNoDefaults(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "no-defaults-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file WITHOUT defaults section
	testContent := `---
on: push
permissions:
  contents: read
---

# Test Workflow Without Defaults

This is a test workflow without defaults section.
`

	testFile := filepath.Join(tmpDir, "test-no-defaults.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContent := string(content)

	// Verify the workflow compiled successfully and contains jobs
	if !strings.Contains(lockContent, "jobs:") {
		t.Error("Expected lock file to contain 'jobs:'")
	}

	// Verify defaults section is not present (since we didn't define it)
	if strings.Contains(lockContent, "defaults:") {
		t.Error("Expected lock file to NOT contain 'defaults:' when not defined in frontmatter")
	}
}

// TestDefaultsSectionFieldOrdering tests that fields within defaults.run are ordered correctly
func TestDefaultsSectionFieldOrdering(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Test with multiple fields in non-alphabetical order in input
	frontmatter := map[string]any{
		"on": map[string]any{
			"push": nil,
		},
		"defaults": map[string]any{
			"run": map[string]any{
				"working-directory": "./src",
				"shell":             "bash",
			},
		},
	}

	result := compiler.extractTopLevelYAMLSection(frontmatter, "defaults")

	// The fields should be in alphabetical order
	shellIndex := strings.Index(result, "shell:")
	workingDirIndex := strings.Index(result, "working-directory:")

	if shellIndex == -1 || workingDirIndex == -1 {
		t.Errorf("Expected both 'shell:' and 'working-directory:' in result: %s", result)
	}

	// shell should come before working-directory alphabetically
	if shellIndex > workingDirIndex {
		t.Errorf("Expected 'shell:' to appear before 'working-directory:' (alphabetical order), got:\n%s", result)
	}
}
