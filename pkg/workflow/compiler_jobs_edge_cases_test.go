//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/testutil"
)

// ========================================
// Custom Job Edge Case Tests
// ========================================

// TestEmptyCustomJobs tests handling of empty custom jobs array
func TestEmptyCustomJobs(t *testing.T) {
	tmpDir := testutil.TempDir(t, "empty-jobs-test")

	frontmatter := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
jobs: {}
---

# Test Workflow

Test content`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("CompileWorkflow() error: %v", err)
	}

	// Read compiled output
	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(content)

	// Should still have standard jobs (activation, agent)
	if !containsInNonCommentLines(yamlStr, "activation:") {
		t.Error("Expected activation job")
	}
	if !containsInNonCommentLines(yamlStr, string(constants.AgentJobName)) {
		t.Error("Expected agent job")
	}
}

// TestJobWithInvalidDependency tests handling of jobs with non-existent dependencies
func TestJobWithInvalidDependency(t *testing.T) {
	tmpDir := testutil.TempDir(t, "invalid-dep-test")

	// Note: The compiler now validates job dependencies and will fail
	// This test verifies that the error is properly reported
	frontmatter := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  dependent:
    runs-on: ubuntu-latest
    needs: non_existent_job
    steps:
      - run: echo "test"
---

# Test Workflow

Test content`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	// Should fail with validation error
	err := compiler.CompileWorkflow(testFile)
	if err == nil {
		t.Fatal("Expected CompileWorkflow() to return error for non-existent job dependency")
	}

	// Verify error message mentions the invalid dependency
	if !strings.Contains(err.Error(), "non_existent_job") {
		t.Errorf("Expected error to mention 'non_existent_job', got: %v", err)
	}
}

// TestJobWithMissingRequiredFields tests handling of jobs missing required fields
func TestJobWithMissingRequiredFields(t *testing.T) {
	tmpDir := testutil.TempDir(t, "missing-fields-test")

	// Job with no runs-on and no uses (invalid but should compile)
	frontmatter := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  minimal:
    steps:
      - run: echo "test"
---

# Test Workflow

Test content`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	// Should compile (GitHub Actions validates at runtime)
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("CompileWorkflow() error: %v", err)
	}

	// Read compiled output
	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(content)

	// Verify job exists
	if !containsInNonCommentLines(yamlStr, "minimal:") {
		t.Error("Expected minimal job")
	}
}

// TestMultipleJobsWithComplexDependencies tests a realistic complex scenario
func TestMultipleJobsWithComplexDependencies(t *testing.T) {
	tmpDir := testutil.TempDir(t, "complex-deps-test")

	frontmatter := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  lint:
    runs-on: ubuntu-latest
    outputs:
      passed: ${{ steps.check.outputs.result }}
    steps:
      - id: check
        run: echo "result=true" >> $GITHUB_OUTPUT
  test:
    runs-on: ubuntu-latest
    needs: lint
    steps:
      - run: npm test
  build:
    runs-on: ubuntu-latest
    needs: [lint, test]
    if: needs.lint.outputs.passed == 'true'
    steps:
      - run: npm build
  deploy:
    runs-on: ubuntu-latest
    needs: build
    steps:
      - run: echo "deploying"
---

# Test Workflow

Test content`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("CompileWorkflow() error: %v", err)
	}

	// Read compiled output
	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(content)

	// Verify all jobs exist
	expectedJobs := []string{"lint:", "test:", "build:", "deploy:"}
	for _, job := range expectedJobs {
		if !containsInNonCommentLines(yamlStr, job) {
			t.Errorf("Expected job %q not found", job)
		}
	}

	// Verify conditional logic is preserved
	if !strings.Contains(yamlStr, "needs.lint.outputs.passed") {
		t.Error("Expected conditional reference to lint output")
	}

	// Verify multi-dependency structure
	// The build job needs array should contain both lint and test
	// Look for the needs section within the build job
	if !strings.Contains(yamlStr, "build:") {
		t.Fatal("build job not found")
	}

	// Check if build job has dependencies (either as array or single)
	// Since jobs auto-depend on activation, we should see lint and test referenced
	hasBothDeps := (strings.Contains(yamlStr, "needs.lint.") || strings.Contains(yamlStr, "- lint")) &&
		(strings.Contains(yamlStr, "needs.test.") || strings.Contains(yamlStr, "- test"))

	if !hasBothDeps {
		t.Error("Expected build job to depend on both lint and test")
	}
}

// TestJobManagerStateValidation tests that job manager maintains correct state
func TestJobManagerStateValidation(t *testing.T) {
	tmpDir := testutil.TempDir(t, "job-manager-state-test")

	frontmatter := `---
on:
  slash_command: test
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  custom1:
    runs-on: ubuntu-latest
    needs: pre_activation
    steps:
      - run: echo "custom1"
  custom2:
    runs-on: ubuntu-latest
    needs: custom1
    steps:
      - run: echo "custom2"
safe-outputs:
  create-issue:
    title-prefix: "[bot] "
  threat-detection: {}
---

# Test Workflow

Test content`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("CompileWorkflow() error: %v", err)
	}

	// Read compiled output
	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(content)

	// Verify expected job structure:
	// 1. pre_activation (command configured)
	// 2. activation (depends on pre_activation + custom1)
	// 3. agent (depends on activation)
	// 4. safe_outputs (depends on agent)
	// 5. detection (depends on safe_outputs)
	// 6. conclusion (depends on safe_outputs)
	// 7. custom1 (depends on pre_activation)
	// 8. custom2 (depends on custom1)

	expectedJobs := []string{
		"pre_activation:",
		"activation:",
		string(constants.AgentJobName),
		"safe_outputs:",
		"conclusion:",
		"custom1:",
		"custom2:",
	}

	for _, job := range expectedJobs {
		if !containsInNonCommentLines(yamlStr, job) {
			t.Errorf("Expected job %q not found", job)
		}
	}

	// Verify inline detection is present in agent job (no separate detection job)
	if containsInNonCommentLines(yamlStr, "\n  detection:\n") {
		t.Error("Expected no separate detection job (detection is inline in agent)")
	}
	if !strings.Contains(yamlStr, "detection_guard") {
		t.Error("Expected inline detection_guard step in agent job")
	}

	// Verify custom2 depends on custom1
	if !strings.Contains(yamlStr, "needs: custom1") && !strings.Contains(yamlStr, "- custom1") {
		t.Error("Expected custom2 to depend on custom1")
	}
}
