package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/testutil"
)

// ========================================
// extractJobsFromFrontmatter Tests
// ========================================

// TestExtractJobsFromFrontmatter tests the extractJobsFromFrontmatter method
func TestExtractJobsFromFrontmatter(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name        string
		frontmatter map[string]any
		expectedLen int
	}{
		{
			name:        "no jobs in frontmatter",
			frontmatter: map[string]any{"on": "push"},
			expectedLen: 0,
		},
		{
			name: "jobs present",
			frontmatter: map[string]any{
				"on": "push",
				"jobs": map[string]any{
					"job1": map[string]any{"runs-on": "ubuntu-latest"},
					"job2": map[string]any{"runs-on": "windows-latest"},
				},
			},
			expectedLen: 2,
		},
		{
			name: "jobs is not a map",
			frontmatter: map[string]any{
				"on":   "push",
				"jobs": "invalid",
			},
			expectedLen: 0,
		},
		{
			name:        "nil frontmatter",
			frontmatter: nil,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.extractJobsFromFrontmatter(tt.frontmatter)
			if len(result) != tt.expectedLen {
				t.Errorf("extractJobsFromFrontmatter() returned %d jobs, want %d", len(result), tt.expectedLen)
			}
		})
	}
}

// ========================================
// Integration Tests
// ========================================

// TestBuildPreActivationJobWithPermissionCheck tests building a pre-activation job with permission checks
func TestBuildPreActivationJobWithPermissionCheck(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	workflowData := &WorkflowData{
		Name:    "Test Workflow",
		Command: "/test",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{},
		},
	}

	job, err := compiler.buildPreActivationJob(workflowData, true, nil)
	if err != nil {
		t.Fatalf("buildPreActivationJob() returned error: %v", err)
	}

	if job.Name != constants.PreActivationJobName {
		t.Errorf("Job name = %q, want %q", job.Name, constants.PreActivationJobName)
	}

	// Check that it has outputs
	if job.Outputs == nil {
		t.Error("Expected job to have outputs")
	}

	// Check for activated output
	if _, ok := job.Outputs["activated"]; !ok {
		t.Error("Expected 'activated' output")
	}

	// Check steps exist
	if len(job.Steps) == 0 {
		t.Error("Expected job to have steps")
	}
}

// TestBuildPreActivationJobWithStopTime tests building a pre-activation job with stop-time
func TestBuildPreActivationJobWithStopTime(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		StopTime:    "2024-12-31T23:59:59Z",
		SafeOutputs: &SafeOutputsConfig{},
	}

	job, err := compiler.buildPreActivationJob(workflowData, false, nil)
	if err != nil {
		t.Fatalf("buildPreActivationJob() returned error: %v", err)
	}

	// Check that steps include stop-time check
	stepsContent := strings.Join(job.Steps, "")
	if !strings.Contains(stepsContent, "Check stop-time limit") {
		t.Error("Expected 'Check stop-time limit' step")
	}
}

// TestBuildActivationJob tests building an activation job
func TestBuildActivationJob(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{},
	}

	job, err := compiler.buildActivationJob(workflowData, false, "", "test.lock.yml", nil)
	if err != nil {
		t.Fatalf("buildActivationJob() returned error: %v", err)
	}

	if job.Name != constants.ActivationJobName {
		t.Errorf("Job name = %q, want %q", job.Name, constants.ActivationJobName)
	}

	// Check for timestamp check step
	stepsContent := strings.Join(job.Steps, "")
	if !strings.Contains(stepsContent, "Check workflow file timestamps") {
		t.Error("Expected 'Check workflow file timestamps' step")
	}
}

// TestBuildActivationJobWithReaction tests building an activation job with AI reaction
func TestBuildActivationJobWithReaction(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		AIReaction:  "rocket",
		SafeOutputs: &SafeOutputsConfig{},
	}

	job, err := compiler.buildActivationJob(workflowData, false, "", "test.lock.yml", nil)
	if err != nil {
		t.Fatalf("buildActivationJob() returned error: %v", err)
	}

	// Check that outputs include reaction-related outputs
	if _, ok := job.Outputs["reaction_id"]; !ok {
		t.Error("Expected 'reaction_id' output")
	}
	if _, ok := job.Outputs["comment_id"]; !ok {
		t.Error("Expected 'comment_id' output")
	}

	// Check for reaction step
	stepsContent := strings.Join(job.Steps, "")
	if !strings.Contains(stepsContent, "rocket reaction") {
		t.Error("Expected reaction step with 'rocket'")
	}
}

// TestBuildMainJobWithActivation tests building the main job with activation dependency
func TestBuildMainJobWithActivation(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	// Initialize stepOrderTracker
	compiler.stepOrderTracker = NewStepOrderTracker()

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		AI:          "copilot",
		RunsOn:      "runs-on: ubuntu-latest",
		Permissions: "permissions:\n  contents: read",
	}

	job, err := compiler.buildMainJob(workflowData, true, nil)
	if err != nil {
		t.Fatalf("buildMainJob() returned error: %v", err)
	}

	if job.Name != constants.AgentJobName {
		t.Errorf("Job name = %q, want %q", job.Name, constants.AgentJobName)
	}

	// Check that it depends on activation job
	found := false
	for _, need := range job.Needs {
		if need == constants.ActivationJobName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected job to depend on %s, got needs: %v", constants.ActivationJobName, job.Needs)
	}
}

// TestBuildCustomJobsWithActivation tests building custom jobs with activation dependency
func TestBuildCustomJobsWithActivation(t *testing.T) {
	tmpDir := testutil.TempDir(t, "custom-jobs-test")

	frontmatter := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  custom_lint:
    runs-on: ubuntu-latest
    steps:
      - run: echo "lint"
  custom_build:
    runs-on: ubuntu-latest
    needs: custom_lint
    steps:
      - run: echo "build"
---

# Test Workflow

Test content`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
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

	// Check that custom jobs exist
	if !strings.Contains(yamlStr, "custom_lint:") {
		t.Error("Expected custom_lint job")
	}
	if !strings.Contains(yamlStr, "custom_build:") {
		t.Error("Expected custom_build job")
	}

	// custom_lint without explicit needs should depend on activation
	// custom_build has explicit needs so should keep that
}

// TestBuildSafeOutputsJobsCreatesExpectedJobs tests that safe output jobs are created correctly
func TestBuildSafeOutputsJobsCreatesExpectedJobs(t *testing.T) {
	tmpDir := testutil.TempDir(t, "safe-outputs-jobs-test")

	frontmatter := `---
on: issues
permissions:
  contents: read
engine: copilot
strict: false
safe-outputs:
  create-issue:
    title-prefix: "[bot] "
  add-comment:
    max: 3
  add-labels:
    allowed: [bug, enhancement]
---

# Test Workflow

Test content`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
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

	// Check that expected safe output jobs are created
	expectedJobs := []string{"create_issue:", "add_comment:", "add_labels:"}
	for _, job := range expectedJobs {
		if !containsInNonCommentLines(yamlStr, job) {
			t.Errorf("Expected job %q not found in output", job)
		}
	}

	// Check that jobs have correct timeout
	// Each safe output job should have timeout-minutes: 10
	if !strings.Contains(yamlStr, "timeout-minutes: 10") {
		t.Error("Expected timeout-minutes: 10 for safe output jobs")
	}
}

// TestBuildJobsWithThreatDetection tests job building with threat detection enabled
func TestBuildJobsWithThreatDetection(t *testing.T) {
	tmpDir := testutil.TempDir(t, "threat-detection-test")

	frontmatter := `---
on: issues
permissions:
  contents: read
engine: copilot
strict: false
safe-outputs:
  create-issue:
  threat-detection:
    enabled: true
---

# Test Workflow

Test content`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
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

	// Check that detection job is created
	if !containsInNonCommentLines(yamlStr, "detection:") {
		t.Error("Expected detection job to be created")
	}

	// Check that create_issue job depends on detection
	if !strings.Contains(yamlStr, constants.DetectionJobName) {
		t.Error("Expected safe output jobs to depend on detection job")
	}
}

// TestBuildJobsWithReusableWorkflow tests custom jobs using reusable workflows
func TestBuildJobsWithReusableWorkflow(t *testing.T) {
	tmpDir := testutil.TempDir(t, "reusable-workflow-test")

	frontmatter := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  call-other:
    uses: owner/repo/.github/workflows/reusable.yml@main
    with:
      param1: value1
    secrets:
      token: ${{ secrets.MY_TOKEN }}
---

# Test Workflow

Test content`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
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

	// Check that reusable workflow job is created
	if !containsInNonCommentLines(yamlStr, "call-other:") {
		t.Error("Expected call-other job")
	}

	// Check for uses directive
	if !strings.Contains(yamlStr, "uses: owner/repo/.github/workflows/reusable.yml@main") {
		t.Error("Expected uses directive for reusable workflow")
	}
}

// TestBuildJobsJobConditionExtraction tests that if conditions are properly extracted
func TestBuildJobsJobConditionExtraction(t *testing.T) {
	tmpDir := testutil.TempDir(t, "job-condition-test")

	frontmatter := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  conditional_job:
    if: github.event_name == 'push'
    runs-on: ubuntu-latest
    steps:
      - run: echo "conditional"
---

# Test Workflow

Test content`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
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

	// Check that job has if condition
	if !strings.Contains(yamlStr, "github.event_name == 'push'") {
		t.Error("Expected if condition to be preserved")
	}
}

// TestBuildJobsWithOutputs tests custom jobs with outputs
func TestBuildJobsWithOutputs(t *testing.T) {
	tmpDir := testutil.TempDir(t, "job-outputs-test")

	frontmatter := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  generate_output:
    runs-on: ubuntu-latest
    outputs:
      result: ${{ steps.compute.outputs.value }}
    steps:
      - id: compute
        run: echo "value=test" >> $GITHUB_OUTPUT
---

# Test Workflow

Test content`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(frontmatter), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
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

	// Check that job has outputs section
	if !strings.Contains(yamlStr, "outputs:") {
		t.Error("Expected outputs section")
	}

	// Check that result output is defined
	if !strings.Contains(yamlStr, "result:") {
		t.Error("Expected 'result' output")
	}
}

// TestBuildPreActivationJobWithCustomConfig tests importing steps and outputs from jobs.pre-activation
func TestBuildPreActivationJobWithCustomConfig(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Test with custom steps and outputs
	customConfig := map[string]any{
		"steps": []any{
			map[string]any{
				"name": "Custom Step",
				"run":  "echo 'Hello from custom step'",
				"id":   "custom_step",
			},
		},
		"outputs": map[string]any{
			"custom_output": "${{ steps.custom_step.outputs.value }}",
		},
	}

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{},
	}

	job, err := compiler.buildPreActivationJob(workflowData, false, customConfig)
	if err != nil {
		t.Fatalf("buildPreActivationJob() returned error: %v", err)
	}

	// Check that custom steps were added
	stepsContent := strings.Join(job.Steps, "")
	if !strings.Contains(stepsContent, "Custom Step") {
		t.Error("Expected 'Custom Step' to be present in steps")
	}
	if !strings.Contains(stepsContent, "echo 'Hello from custom step'") {
		t.Error("Expected custom step command to be present")
	}

	// Check that custom outputs were added
	if job.Outputs == nil {
		t.Fatal("Expected job to have outputs")
	}
	if customOutput, ok := job.Outputs["custom_output"]; !ok {
		t.Error("Expected 'custom_output' in outputs")
	} else if !strings.Contains(customOutput, "steps.custom_step.outputs.value") {
		t.Errorf("Expected custom output expression, got: %s", customOutput)
	}
}

// TestBuildPreActivationJobWithCustomConfigAndBuiltInChecks tests combining custom config with built-in checks
func TestBuildPreActivationJobWithCustomConfigAndBuiltInChecks(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Custom config with steps and outputs
	customConfig := map[string]any{
		"steps": []any{
			map[string]any{
				"name": "Custom Validation",
				"run":  "echo 'Validating'",
				"id":   "custom_validation",
			},
		},
		"outputs": map[string]any{
			"validation_result": "${{ steps.custom_validation.outputs.result }}",
		},
	}

	workflowData := &WorkflowData{
		Name:     "Test Workflow",
		StopTime: "2024-12-31T23:59:59Z",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{},
		},
	}

	job, err := compiler.buildPreActivationJob(workflowData, true, customConfig)
	if err != nil {
		t.Fatalf("buildPreActivationJob() returned error: %v", err)
	}

	// Check that built-in steps are present
	stepsContent := strings.Join(job.Steps, "")
	if !strings.Contains(stepsContent, "Check stop-time limit") {
		t.Error("Expected built-in 'Check stop-time limit' step")
	}

	// Check that custom steps were added
	if !strings.Contains(stepsContent, "Custom Validation") {
		t.Error("Expected custom 'Custom Validation' step")
	}

	// Check that both built-in and custom outputs are present
	if job.Outputs == nil {
		t.Fatal("Expected job to have outputs")
	}
	if _, ok := job.Outputs["activated"]; !ok {
		t.Error("Expected built-in 'activated' output")
	}
	if _, ok := job.Outputs["validation_result"]; !ok {
		t.Error("Expected custom 'validation_result' output")
	}
}

// TestBuildPreActivationJobWithInvalidConfig tests error handling for unsupported fields
func TestBuildPreActivationJobWithInvalidConfig(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Custom config with unsupported field
	customConfig := map[string]any{
		"steps": []any{
			map[string]any{
				"name": "Test Step",
				"run":  "echo 'test'",
			},
		},
		"needs": []string{"some-job"}, // This should cause an error
	}

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{},
	}

	_, err := compiler.buildPreActivationJob(workflowData, false, customConfig)
	if err == nil {
		t.Fatal("Expected error for unsupported field 'needs'")
	}
	if !strings.Contains(err.Error(), "only supports 'steps', 'outputs', and 'permissions' fields") {
		t.Errorf("Expected error message about unsupported fields, got: %v", err)
	}
	if !strings.Contains(err.Error(), "needs") {
		t.Errorf("Expected error message to mention 'needs', got: %v", err)
	}
}

// TestBuildPreActivationJobWithOnlyCustomConfig tests pre-activation with only custom config (no built-in checks)
func TestBuildPreActivationJobWithOnlyCustomConfig(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Only custom config, no built-in checks
	customConfig := map[string]any{
		"steps": []any{
			map[string]any{
				"name": "Custom Check",
				"run":  "echo 'Checking'",
				"id":   "check",
			},
		},
		"outputs": map[string]any{
			"check_result": "${{ steps.check.outputs.result }}",
		},
	}

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{},
	}

	job, err := compiler.buildPreActivationJob(workflowData, false, customConfig)
	if err != nil {
		t.Fatalf("buildPreActivationJob() returned error: %v", err)
	}

	// Check that custom steps and outputs are present
	stepsContent := strings.Join(job.Steps, "")
	if !strings.Contains(stepsContent, "Custom Check") {
		t.Error("Expected 'Custom Check' step")
	}

	if job.Outputs == nil {
		t.Fatal("Expected job to have outputs")
	}

	// Should NOT have the activated output since there are no built-in checks
	if _, ok := job.Outputs["activated"]; ok {
		t.Error("Did not expect 'activated' output when no built-in checks are present")
	}

	// Should have custom output
	if _, ok := job.Outputs["check_result"]; !ok {
		t.Error("Expected 'check_result' output")
	}
}

// TestBuildPreActivationJobWithCustomPermissions tests importing permissions
func TestBuildPreActivationJobWithCustomPermissions(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Custom config with permissions
	customConfig := map[string]any{
		"permissions": map[string]any{
			"contents": "read",
			"issues":   "write",
		},
	}

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{},
	}

	job, err := compiler.buildPreActivationJob(workflowData, false, customConfig)
	if err != nil {
		t.Fatalf("buildPreActivationJob() returned error: %v", err)
	}

	// Check that permissions were imported
	if job.Permissions == "" {
		t.Error("Expected permissions to be set")
	}
	if !strings.Contains(job.Permissions, "contents: read") {
		t.Error("Expected 'contents: read' in permissions")
	}
	if !strings.Contains(job.Permissions, "issues: write") {
		t.Error("Expected 'issues: write' in permissions")
	}
}

// TestAgentJobNoCyclic tests that agent job with custom config doesn't create circular dependencies
func TestAgentJobNoCyclic(t *testing.T) {
compiler := NewCompiler(false, "", "test")

// Simulate workflow data with agent job custom config
workflowData := &WorkflowData{
Name: "Test Workflow",
Jobs: map[string]any{
"agent": map[string]any{
"steps": []any{
map[string]any{
"name": "Custom step",
"run":  "echo test",
},
},
},
},
SafeOutputs: &SafeOutputsConfig{},
}

// Build the main job
job, err := compiler.buildMainJob(workflowData, false, map[string]any{
"steps": []any{
map[string]any{
"name": "Custom step",
"run":  "echo test",
},
},
})

if err != nil {
t.Fatalf("buildMainJob() returned error: %v", err)
}

// Verify the agent job doesn't depend on itself
for _, dep := range job.Needs {
if dep == constants.AgentJobName {
t.Errorf("Agent job should not depend on itself, found in Needs: %v", job.Needs)
}
}
}
