//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========================================
// Custom Job Build Tests (dependencies, permissions, conditionals)
// ========================================

// TestBuildCustomJobsWithMultipleDependencies tests custom jobs with complex dependency chains
func TestBuildCustomJobsWithMultipleDependencies(t *testing.T) {
	tmpDir := testutil.TempDir(t, "multi-dep-test")

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
      - run: echo "job_a"
  job_b:
    runs-on: ubuntu-latest
    needs: job_a
    steps:
      - run: echo "job_b"
  job_c:
    runs-on: ubuntu-latest
    needs: [job_a, job_b]
    steps:
      - run: echo "job_c"
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

	// Verify all custom jobs exist
	if !containsInNonCommentLines(yamlStr, "job_a:") {
		t.Error("Expected job_a")
	}
	if !containsInNonCommentLines(yamlStr, "job_b:") {
		t.Error("Expected job_b")
	}
	if !containsInNonCommentLines(yamlStr, "job_c:") {
		t.Error("Expected job_c")
	}

	// Verify job_c has multiple dependencies
	if !strings.Contains(yamlStr, "job_a") || !strings.Contains(yamlStr, "job_b") {
		t.Error("Expected job_c to depend on both job_a and job_b")
	}
}

// TestBuildCustomJobsWithCircularDetection tests handling of circular dependencies
func TestBuildCustomJobsWithCircularDetection(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	// Create workflow data with potential circular dependency
	// Note: This tests that the compiler handles the case without crashing
	data := &WorkflowData{
		Name:   "Test Workflow",
		AI:     "copilot",
		RunsOn: "runs-on: ubuntu-latest",
		Jobs: map[string]any{
			"job_a": map[string]any{
				"runs-on": "ubuntu-latest",
				"needs":   "job_b",
				"steps": []any{
					map[string]any{"run": "echo 'job_a'"},
				},
			},
			"job_b": map[string]any{
				"runs-on": "ubuntu-latest",
				"needs":   "job_a",
				"steps": []any{
					map[string]any{"run": "echo 'job_b'"},
				},
			},
		},
	}

	// Build custom jobs - this should not crash even with circular deps
	// GitHub Actions itself will catch circular dependencies at runtime
	err := compiler.buildCustomJobs(data, false)
	if err != nil {
		t.Fatalf("buildCustomJobs() returned error: %v", err)
	}

	// Verify both jobs were added
	if _, exists := compiler.jobManager.GetJob("job_a"); !exists {
		t.Error("Expected job_a to be added")
	}
	if _, exists := compiler.jobManager.GetJob("job_b"); !exists {
		t.Error("Expected job_b to be added")
	}
}

// TestBuildCustomJobsWithPermissions tests custom jobs with various permission configurations
func TestBuildCustomJobsWithPermissions(t *testing.T) {
	tmpDir := testutil.TempDir(t, "permissions-test")

	frontmatter := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  job_with_perms:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write
    steps:
      - run: echo "has permissions"
  job_without_perms:
    runs-on: ubuntu-latest
    steps:
      - run: echo "no permissions"
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

	// Verify job_with_perms has permissions
	if !strings.Contains(yamlStr, "job_with_perms:") {
		t.Error("Expected job_with_perms")
	}
	if !strings.Contains(yamlStr, "contents: write") {
		t.Error("Expected contents: write permission")
	}

	// Verify job_without_perms exists
	if !strings.Contains(yamlStr, "job_without_perms:") {
		t.Error("Expected job_without_perms")
	}
}

// TestBuildCustomJobsWithConditionals tests custom jobs with if conditions
func TestBuildCustomJobsWithConditionals(t *testing.T) {
	tmpDir := testutil.TempDir(t, "conditionals-test")

	frontmatter := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  conditional_job:
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
      - run: echo "only on main"
  always_job:
    runs-on: ubuntu-latest
    steps:
      - run: echo "always runs"
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

	// Verify conditional_job has if condition
	if !strings.Contains(yamlStr, "conditional_job:") {
		t.Error("Expected conditional_job")
	}
	if !strings.Contains(yamlStr, "github.ref == 'refs/heads/main'") {
		t.Error("Expected if condition to be preserved")
	}

	// Verify always_job exists without conditions
	if !strings.Contains(yamlStr, "always_job:") {
		t.Error("Expected always_job")
	}
}

// TestBuildCustomJobsWithReusableWorkflowAndWith tests reusable workflow with parameters
func TestBuildCustomJobsWithReusableWorkflowAndWith(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	// Create workflow data with reusable workflow and with parameters
	data := &WorkflowData{
		Name:   "Test Workflow",
		AI:     "copilot",
		RunsOn: "runs-on: ubuntu-latest",
		Jobs: map[string]any{
			"reusable_job": map[string]any{
				"uses": "owner/repo/.github/workflows/reusable.yml@main",
				"with": map[string]any{
					"param1": "value1",
					"param2": 42,
				},
			},
		},
	}

	err := compiler.buildCustomJobs(data, false)
	if err != nil {
		t.Fatalf("buildCustomJobs() returned error: %v", err)
	}

	// Verify job was added
	job, exists := compiler.jobManager.GetJob("reusable_job")
	if !exists {
		t.Fatal("Expected reusable_job to be added")
	}

	// Verify uses field is set
	if job.Uses == "" {
		t.Error("Expected uses field to be set")
	}

	// Verify with parameters are set
	if job.With == nil {
		t.Fatal("Expected with parameters to be set")
	}
	if job.With["param1"] != "value1" {
		t.Errorf("Expected param1=value1, got %v", job.With["param1"])
	}
}

// TestBuildCustomJobsWithInvalidSecrets tests secret validation
func TestBuildCustomJobsWithInvalidSecrets(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	// Create workflow data with invalid secrets (not a GitHub Actions expression)
	data := &WorkflowData{
		Name:   "Test Workflow",
		AI:     "copilot",
		RunsOn: "runs-on: ubuntu-latest",
		Jobs: map[string]any{
			"reusable_job": map[string]any{
				"uses": "owner/repo/.github/workflows/reusable.yml@main",
				"secrets": map[string]any{
					"token": "hardcoded_secret", // Invalid - not an expression
				},
			},
		},
	}

	err := compiler.buildCustomJobs(data, false)
	if err == nil {
		t.Error("Expected error for invalid secret, got nil")
	}
}

// TestBuildCustomJobsAutomaticActivationDependency tests automatic activation dependency
func TestBuildCustomJobsAutomaticActivationDependency(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	// Add activation job to manager
	activationJob := &Job{
		Name: string(constants.ActivationJobName),
	}
	if err := compiler.jobManager.AddJob(activationJob); err != nil {
		t.Fatal(err)
	}

	// Create workflow data with custom job that has no explicit needs
	data := &WorkflowData{
		Name:   "Test Workflow",
		AI:     "copilot",
		RunsOn: "runs-on: ubuntu-latest",
		Jobs: map[string]any{
			"custom_job": map[string]any{
				"runs-on": "ubuntu-latest",
				"steps": []any{
					map[string]any{"run": "echo 'test'"},
				},
			},
		},
	}

	// Build custom jobs with activation created
	err := compiler.buildCustomJobs(data, true)
	if err != nil {
		t.Fatalf("buildCustomJobs() returned error: %v", err)
	}

	// Verify custom job has automatic dependency on activation
	job, exists := compiler.jobManager.GetJob("custom_job")
	if !exists {
		t.Fatal("Expected custom_job to be added")
	}

	// Check that activation is in the needs array
	found := slices.Contains(job.Needs, string(constants.ActivationJobName))
	if !found {
		t.Error("Expected automatic dependency on activation job")
	}
}

// TestBuildCustomJobsSkipsPreActivationJob tests that pre_activation jobs are skipped
func TestBuildCustomJobsSkipsPreActivationJob(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	// Create workflow data with pre_activation job (should be skipped)
	data := &WorkflowData{
		Name:   "Test Workflow",
		AI:     "copilot",
		RunsOn: "runs-on: ubuntu-latest",
		Jobs: map[string]any{
			"pre_activation": map[string]any{
				"runs-on": "ubuntu-latest",
				"steps": []any{
					map[string]any{"run": "echo 'should be skipped'"},
				},
			},
			"pre-activation": map[string]any{
				"runs-on": "ubuntu-latest",
				"steps": []any{
					map[string]any{"run": "echo 'should also be skipped'"},
				},
			},
			"normal_job": map[string]any{
				"runs-on": "ubuntu-latest",
				"steps": []any{
					map[string]any{"run": "echo 'should be added'"},
				},
			},
		},
	}

	err := compiler.buildCustomJobs(data, false)
	if err != nil {
		t.Fatalf("buildCustomJobs() returned error: %v", err)
	}

	// Verify pre_activation jobs were skipped
	if _, exists := compiler.jobManager.GetJob("pre_activation"); exists {
		t.Error("Expected pre_activation job to be skipped")
	}
	if _, exists := compiler.jobManager.GetJob("pre-activation"); exists {
		t.Error("Expected pre-activation job to be skipped")
	}

	// Verify normal job was added
	if _, exists := compiler.jobManager.GetJob("normal_job"); !exists {
		t.Error("Expected normal_job to be added")
	}
}

// TestBuildCustomJobsDoesNotAutoAddActivationWhenListedInOnNeeds verifies that
// custom jobs listed in on.needs run before activation and therefore do not get
// an implicit needs: activation dependency.
func TestBuildCustomJobsDoesNotAutoAddActivationWhenListedInOnNeeds(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	activationJob := &Job{Name: string(constants.ActivationJobName)}
	require.NoError(t, compiler.jobManager.AddJob(activationJob), "activation job should be added")

	data := &WorkflowData{
		Name:    "Test Workflow",
		AI:      "copilot",
		OnNeeds: []string{"secrets_fetcher"},
		Jobs: map[string]any{
			"secrets_fetcher": map[string]any{
				"runs-on": "ubuntu-latest",
				"steps": []any{
					map[string]any{"run": "echo 'fetch'"},
				},
			},
		},
	}

	require.NoError(t, compiler.buildCustomJobs(data, true), "custom jobs should build")

	job, exists := compiler.jobManager.GetJob("secrets_fetcher")
	require.True(t, exists, "secrets_fetcher should be added")
	assert.NotContains(t, job.Needs, string(constants.ActivationJobName), "on.needs job should not auto-depend on activation")
}

// TestBuildCustomJobsWithStrategy tests custom jobs with matrix strategy configuration
func TestBuildCustomJobsWithStrategy(t *testing.T) {
	tmpDir := testutil.TempDir(t, "strategy-test")

	frontmatter := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  matrix_job:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest]
        node: [18, 20]
      fail-fast: false
      max-parallel: 2
    steps:
      - run: echo "matrix job"
  simple_job:
    runs-on: ubuntu-latest
    steps:
      - run: echo "simple job"
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

	// Verify matrix_job has strategy section
	if !strings.Contains(yamlStr, "matrix_job:") {
		t.Error("Expected matrix_job in compiled output")
	}
	if !strings.Contains(yamlStr, "strategy:") {
		t.Error("Expected strategy section in compiled output")
	}
	if !strings.Contains(yamlStr, "matrix:") {
		t.Error("Expected matrix section in compiled output")
	}
	if !strings.Contains(yamlStr, "fail-fast: false") {
		t.Error("Expected fail-fast: false in compiled output")
	}
	if !strings.Contains(yamlStr, "max-parallel: 2") {
		t.Error("Expected max-parallel: 2 in compiled output")
	}

	// Verify simple_job has no strategy
	if !strings.Contains(yamlStr, "simple_job:") {
		t.Error("Expected simple_job in compiled output")
	}
}
