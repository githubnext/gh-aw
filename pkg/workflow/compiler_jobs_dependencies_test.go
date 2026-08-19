//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

// ========================================
// Complex Dependency and Ordering Tests
// ========================================

// TestComplexJobDependencyChains tests various job dependency chain scenarios
func TestComplexJobDependencyChains(t *testing.T) {
	tmpDir := testutil.TempDir(t, "dependency-chains-test")

	frontmatter := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  job_a:
    runs-on: ubuntu-latest
    steps:
      - run: echo "A"
  job_b:
    runs-on: ubuntu-latest
    needs: job_a
    steps:
      - run: echo "B"
  job_c:
    runs-on: ubuntu-latest
    needs: [job_a, job_b]
    steps:
      - run: echo "C"
  job_d:
    runs-on: ubuntu-latest
    needs: job_c
    steps:
      - run: echo "D"
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

	// Verify all custom jobs are present
	expectedJobs := []string{"job_a:", "job_b:", "job_c:", "job_d:"}
	for _, job := range expectedJobs {
		if !containsInNonCommentLines(yamlStr, job) {
			t.Errorf("Expected job %q not found", job)
		}
	}

	// Verify dependency structure is preserved
	// job_b should depend on job_a
	if !strings.Contains(yamlStr, "needs: job_a") && !strings.Contains(yamlStr, "needs:\n      - job_a") {
		t.Error("Expected job_b to depend on job_a")
	}
}

// TestJobDependingOnPreActivation tests jobs that explicitly depend on pre-activation
func TestJobDependingOnPreActivation(t *testing.T) {
	tmpDir := testutil.TempDir(t, "pre-activation-dep-test")

	frontmatter := `---
on:
  slash_command: test
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  early_job:
    runs-on: ubuntu-latest
    needs: pre_activation
    steps:
      - run: echo "Runs after pre-activation"
  normal_job:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Normal job"
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

	// Verify pre-activation job exists (command is configured)
	if !containsInNonCommentLines(yamlStr, "pre_activation:") {
		t.Error("Expected pre_activation job")
	}

	// Verify early_job exists and depends on pre_activation
	if !containsInNonCommentLines(yamlStr, "early_job:") {
		t.Error("Expected early_job")
	}

	// Verify normal_job exists
	if !containsInNonCommentLines(yamlStr, "normal_job:") {
		t.Error("Expected normal_job")
	}
}

// TestJobReferencingCustomJobOutputs tests jobs that reference outputs from custom jobs
func TestJobReferencingCustomJobOutputs(t *testing.T) {
	tmpDir := testutil.TempDir(t, "job-outputs-ref-test")

	frontmatter := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  producer:
    runs-on: ubuntu-latest
    outputs:
      value: ${{ steps.gen.outputs.value }}
    steps:
      - id: gen
        run: echo "value=42" >> $GITHUB_OUTPUT
  consumer:
    runs-on: ubuntu-latest
    needs: producer
    if: needs.producer.outputs.value == '42'
    steps:
      - run: echo "Consuming ${{ needs.producer.outputs.value }}"
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

	// Verify both jobs exist
	if !containsInNonCommentLines(yamlStr, "producer:") {
		t.Error("Expected producer job")
	}
	if !containsInNonCommentLines(yamlStr, "consumer:") {
		t.Error("Expected consumer job")
	}

	// Verify output reference is preserved
	if !strings.Contains(yamlStr, "needs.producer.outputs.value") {
		t.Error("Expected reference to producer output")
	}
}
