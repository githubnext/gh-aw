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
)

// ========================================
// extractJobsFromFrontmatter Tests
// ========================================

// TestExtractJobsFromFrontmatter tests the extractJobsFromFrontmatter method
func TestExtractJobsFromFrontmatter(t *testing.T) {
	compiler := NewCompiler()

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
// Helper Function Tests
// ========================================

// TestReferencesCustomJobOutputsAdditional tests additional edge cases for referencesCustomJobOutputs method
func TestReferencesCustomJobOutputsAdditional(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name       string
		condition  string
		customJobs map[string]any
		expected   bool
	}{
		{
			name:       "references non-existent job",
			condition:  "needs.job2.outputs.value",
			customJobs: map[string]any{"job1": map[string]any{}},
			expected:   false,
		},
		{
			name:       "multiple custom jobs with reference",
			condition:  "needs.producer.outputs.result",
			customJobs: map[string]any{"producer": map[string]any{}, "consumer": map[string]any{}},
			expected:   true,
		},
		{
			name:       "complex condition with output reference",
			condition:  "needs.test.outputs.status == 'pass' && github.ref == 'refs/heads/main'",
			customJobs: map[string]any{"test": map[string]any{}},
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.referencesCustomJobOutputs(tt.condition, tt.customJobs)
			if result != tt.expected {
				t.Errorf("referencesCustomJobOutputs() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestJobDependsOnPreActivationEdgeCases tests edge cases for jobDependsOnPreActivation function
func TestJobDependsOnPreActivationEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		jobConfig map[string]any
		expected  bool
	}{
		{
			name: "needs is invalid type",
			jobConfig: map[string]any{
				"needs": 123,
			},
			expected: false,
		},
		{
			name: "array with non-string element",
			jobConfig: map[string]any{
				"needs": []any{123, "pre_activation"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jobDependsOnPreActivation(tt.jobConfig)
			if result != tt.expected {
				t.Errorf("jobDependsOnPreActivation() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestJobDependsOnAgentEdgeCases tests edge cases for jobDependsOnAgent function
func TestJobDependsOnAgentEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		jobConfig map[string]any
		expected  bool
	}{
		{
			name: "array with mixed types including agent",
			jobConfig: map[string]any{
				"needs": []any{123, "agent", "job2"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jobDependsOnAgent(tt.jobConfig)
			if result != tt.expected {
				t.Errorf("jobDependsOnAgent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestGetCustomJobsDependingOnPreActivationEdgeCases tests edge cases for getCustomJobsDependingOnPreActivation method
func TestGetCustomJobsDependingOnPreActivationEdgeCases(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name           string
		customJobs     map[string]any
		expectedCount  int
		expectedJobIDs []string
	}{
		{
			name: "job with invalid config type",
			customJobs: map[string]any{
				"job1": "invalid",
				"job2": map[string]any{"needs": "pre_activation"},
			},
			expectedCount:  1,
			expectedJobIDs: []string{"job2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.getCustomJobsDependingOnPreActivation(tt.customJobs)
			if len(result) != tt.expectedCount {
				t.Errorf("getCustomJobsDependingOnPreActivation() returned %d jobs, want %d", len(result), tt.expectedCount)
			}
			// Check that expected job IDs are present
			for _, expectedID := range tt.expectedJobIDs {
				found := slices.Contains(result, expectedID)
				if !found {
					t.Errorf("Expected job %q not found in result", expectedID)
				}
			}
		})
	}
}

// TestJobDependsOnActivation tests the jobDependsOnActivation function
func TestJobDependsOnActivation(t *testing.T) {
	tests := []struct {
		name      string
		jobConfig map[string]any
		expected  bool
	}{
		{
			name:      "no needs field",
			jobConfig: map[string]any{"runs-on": "ubuntu-latest"},
			expected:  false,
		},
		{
			name:      "needs: activation as string",
			jobConfig: map[string]any{"needs": "activation"},
			expected:  true,
		},
		{
			name:      "needs: pre_activation only",
			jobConfig: map[string]any{"needs": "pre_activation"},
			expected:  false,
		},
		{
			name:      "needs: agent only",
			jobConfig: map[string]any{"needs": "agent"},
			expected:  false,
		},
		{
			name: "needs: [activation, pre_activation] array",
			jobConfig: map[string]any{
				"needs": []any{"pre_activation", "activation"},
			},
			expected: true,
		},
		{
			name: "needs: array without activation",
			jobConfig: map[string]any{
				"needs": []any{"pre_activation", "config"},
			},
			expected: false,
		},
		{
			name: "needs: array with mixed types including activation",
			jobConfig: map[string]any{
				"needs": []any{123, "activation"},
			},
			expected: true,
		},
		{
			name:      "needs: invalid type",
			jobConfig: map[string]any{"needs": 123},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jobDependsOnActivation(tt.jobConfig)
			if result != tt.expected {
				t.Errorf("jobDependsOnActivation() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestGetCustomJobsDependingOnPreActivationExcludesActivationDependents tests that
// getCustomJobsDependingOnPreActivation excludes jobs that also depend on activation.
// This prevents the compiler from adding such jobs to activation's needs (which would
// create a circular dependency: activation → job → activation).
func TestGetCustomJobsDependingOnPreActivationExcludesActivationDependents(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name         string
		customJobs   map[string]any
		expectedJobs []string
		excludedJobs []string
	}{
		{
			name: "job with both pre_activation and activation is excluded",
			customJobs: map[string]any{
				"config": map[string]any{
					"runs-on": "ubuntu-latest",
					"needs":   []any{"pre_activation", "activation"},
				},
			},
			expectedJobs: []string{},
			excludedJobs: []string{"config"},
		},
		{
			name: "job with only pre_activation is included",
			customJobs: map[string]any{
				"precompute": map[string]any{
					"runs-on": "ubuntu-latest",
					"needs":   []any{"pre_activation"},
				},
			},
			expectedJobs: []string{"precompute"},
			excludedJobs: []string{},
		},
		{
			name: "mixed: pre_activation-only included, pre_activation+activation excluded",
			customJobs: map[string]any{
				"precompute": map[string]any{
					"runs-on": "ubuntu-latest",
					"needs":   "pre_activation",
				},
				"config": map[string]any{
					"runs-on": "ubuntu-latest",
					"needs":   []any{"pre_activation", "activation"},
				},
				"release": map[string]any{
					"runs-on": "ubuntu-latest",
					"needs":   []any{"pre_activation", "activation", "config"},
				},
			},
			expectedJobs: []string{"precompute"},
			excludedJobs: []string{"config", "release"},
		},
		{
			name: "job with only activation dependency is excluded",
			customJobs: map[string]any{
				"post_job": map[string]any{
					"runs-on": "ubuntu-latest",
					"needs":   "activation",
				},
			},
			expectedJobs: []string{},
			excludedJobs: []string{"post_job"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.getCustomJobsDependingOnPreActivation(tt.customJobs)
			resultSet := make(map[string]bool, len(result))
			for _, j := range result {
				resultSet[j] = true
			}
			for _, expected := range tt.expectedJobs {
				if !resultSet[expected] {
					t.Errorf("Expected job %q in result, got: %v", expected, result)
				}
			}
			for _, excluded := range tt.excludedJobs {
				if resultSet[excluded] {
					t.Errorf("Job %q should be excluded from result, got: %v", excluded, result)
				}
			}
		})
	}
}

// TestBuildCustomJobsDoesNotAutoAddActivationToOutputReferencedJobs tests that
// buildCustomJobs does NOT auto-add needs: activation to custom jobs whose outputs
// are referenced in the markdown body (they must run before activation).
func TestBuildCustomJobsDoesNotAutoAddActivationToOutputReferencedJobs(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	// Add activation job to manager
	activationJob := &Job{Name: string(constants.ActivationJobName)}
	if err := compiler.jobManager.AddJob(activationJob); err != nil {
		t.Fatal(err)
	}

	data := &WorkflowData{
		Name:   "Test Workflow",
		AI:     "copilot",
		RunsOn: "runs-on: ubuntu-latest",
		// precompute has no explicit needs and its output is referenced in the markdown
		MarkdownContent: "Action: ${{ needs.precompute.outputs.action }}",
		Jobs: map[string]any{
			"precompute": map[string]any{
				"runs-on": "ubuntu-latest",
				"steps": []any{
					map[string]any{"run": "echo 'precompute'"},
				},
				// No explicit needs — normally would auto-get needs: activation
				// But since precompute's output is referenced in markdown, it should NOT
			},
		},
	}

	err := compiler.buildCustomJobs(data, true)
	if err != nil {
		t.Fatalf("buildCustomJobs() returned error: %v", err)
	}

	job, exists := compiler.jobManager.GetJob("precompute")
	if !exists {
		t.Fatal("Expected precompute job to be added")
	}

	for _, need := range job.Needs {
		if need == string(constants.ActivationJobName) {
			t.Errorf("precompute job should NOT have needs: activation when its output is referenced in markdown (it must run before activation)")
		}
	}
}

func TestGetReferencedCustomJobs(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name           string
		content        string
		customJobs     map[string]any
		expectedCount  int
		expectedJobIDs []string
	}{
		{
			name:           "empty content",
			content:        "",
			customJobs:     map[string]any{"job1": map[string]any{}},
			expectedCount:  0,
			expectedJobIDs: []string{},
		},
		{
			name:           "nil custom jobs",
			content:        "needs.job1.outputs.value",
			customJobs:     nil,
			expectedCount:  0,
			expectedJobIDs: []string{},
		},
		{
			name:           "references one job output",
			content:        "needs.producer.outputs.value",
			customJobs:     map[string]any{"producer": map[string]any{}, "consumer": map[string]any{}},
			expectedCount:  1,
			expectedJobIDs: []string{"producer"},
		},
		{
			name:           "references job result",
			content:        "needs.test_job.result == 'success'",
			customJobs:     map[string]any{"test_job": map[string]any{}},
			expectedCount:  1,
			expectedJobIDs: []string{"test_job"},
		},
		{
			name:           "references multiple jobs",
			content:        "needs.job1.outputs.a && needs.job2.outputs.b",
			customJobs:     map[string]any{"job1": map[string]any{}, "job2": map[string]any{}},
			expectedCount:  2,
			expectedJobIDs: []string{"job1", "job2"},
		},
		{
			name:           "no job references",
			content:        "github.event_name == 'push'",
			customJobs:     map[string]any{"job1": map[string]any{}},
			expectedCount:  0,
			expectedJobIDs: []string{},
		},
		{
			name:           "references non-existent job",
			content:        "needs.unknown.outputs.value",
			customJobs:     map[string]any{"job1": map[string]any{}},
			expectedCount:  0,
			expectedJobIDs: []string{},
		},
		{
			name:           "github expression format",
			content:        "${{ needs.check.outputs.status }}",
			customJobs:     map[string]any{"check": map[string]any{}},
			expectedCount:  1,
			expectedJobIDs: []string{"check"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.getReferencedCustomJobs(tt.content, tt.customJobs)
			if len(result) != tt.expectedCount {
				t.Errorf("getReferencedCustomJobs() returned %d jobs, want %d", len(result), tt.expectedCount)
			}
			// Check that expected job IDs are present
			for _, expectedID := range tt.expectedJobIDs {
				found := slices.Contains(result, expectedID)
				if !found {
					t.Errorf("Expected job %q not found in result", expectedID)
				}
			}
		})
	}
}

// TestShouldAddCheckoutStep tests the shouldAddCheckoutStep method
func TestShouldAddCheckoutStep(t *testing.T) {
	tests := []struct {
		name       string
		data       *WorkflowData
		actionMode ActionMode
		expected   bool
	}{
		{
			name: "custom steps with checkout",
			data: &WorkflowData{
				CustomSteps: "- uses: actions/checkout@v4",
			},
			actionMode: ActionModeDev,
			expected:   false,
		},
		{
			name: "custom steps without checkout",
			data: &WorkflowData{
				CustomSteps: "- run: echo 'test'",
			},
			actionMode: ActionModeDev,
			expected:   true,
		},
		{
			name: "agent file specified",
			data: &WorkflowData{
				AgentFile: ".github/agents/custom.md",
			},
			actionMode: ActionModeRelease,
			expected:   true,
		},
		{
			name: "release mode without agent file",
			data: &WorkflowData{
				CustomSteps: "",
			},
			actionMode: ActionModeRelease,
			expected:   true, // Checkout always needed unless already in steps
		},
		{
			name: "dev mode without agent file",
			data: &WorkflowData{
				CustomSteps: "",
			},
			actionMode: ActionModeDev,
			expected:   true,
		},
		{
			name: "script mode without agent file",
			data: &WorkflowData{
				CustomSteps: "",
			},
			actionMode: ActionModeScript,
			expected:   true,
		},
		{
			name:       "uninitialized mode",
			data:       &WorkflowData{},
			actionMode: ActionMode(""),
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.actionMode = tt.actionMode
			result := compiler.shouldAddCheckoutStep(tt.data)
			if result != tt.expected {
				t.Errorf("shouldAddCheckoutStep() = %v, want %v (actionMode=%v)", result, tt.expected, tt.actionMode)
			}
		})
	}
}

// ========================================
// Integration Tests
// ========================================

// TestBuildPreActivationJobWithPermissionCheck tests building a pre-activation job with permission checks
func TestBuildPreActivationJobWithPermissionCheck(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name:    "Test Workflow",
		Command: []string{"test"},
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{},
		},
	}

	job, err := compiler.buildPreActivationJob(workflowData, true)
	if err != nil {
		t.Fatalf("buildPreActivationJob() returned error: %v", err)
	}

	if job.Name != string(constants.PreActivationJobName) {
		t.Errorf("Job name = %q, want %q", job.Name, string(constants.PreActivationJobName))
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
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		StopTime:    "2024-12-31T23:59:59Z",
		SafeOutputs: &SafeOutputsConfig{},
	}

	job, err := compiler.buildPreActivationJob(workflowData, false)
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
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{},
	}

	job, err := compiler.buildActivationJob(workflowData, false, "", "test.lock.yml")
	if err != nil {
		t.Fatalf("buildActivationJob() returned error: %v", err)
	}

	if job.Name != string(constants.ActivationJobName) {
		t.Errorf("Job name = %q, want %q", job.Name, string(constants.ActivationJobName))
	}

	// Check for timestamp check step
	stepsContent := strings.Join(job.Steps, "")
	if !strings.Contains(stepsContent, "Check workflow file timestamps") {
		t.Error("Expected 'Check workflow file timestamps' step")
	}
}

// TestBuildActivationJobWithReaction tests building an activation job with AI reaction
func TestBuildActivationJobWithReaction(t *testing.T) {
	compiler := NewCompiler()

	statusCommentTrue := true
	workflowData := &WorkflowData{
		Name:          "Test Workflow",
		AIReaction:    "rocket",
		StatusComment: &statusCommentTrue,
		SafeOutputs:   &SafeOutputsConfig{},
	}

	job, err := compiler.buildActivationJob(workflowData, false, "", "test.lock.yml")
	if err != nil {
		t.Fatalf("buildActivationJob() returned error: %v", err)
	}

	// Check that outputs include comment-related outputs (but not reaction_id since reaction is in pre-activation)
	if _, ok := job.Outputs["comment_id"]; !ok {
		t.Error("Expected 'comment_id' output")
	}

	// Check for comment step (not reaction, since reaction moved to pre-activation)
	stepsContent := strings.Join(job.Steps, "")
	if !strings.Contains(stepsContent, "Add comment with workflow run link") {
		t.Error("Expected comment step in activation job")
	}
}

// TestBuildActivationJobLockFilename tests that lock filenames are passed through
// unchanged to the activation job environment.
func TestBuildActivationJobLockFilename(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{},
	}

	job, err := compiler.buildActivationJob(workflowData, false, "", "example.workflow.lock.yml")
	if err != nil {
		t.Fatalf("buildActivationJob() returned error: %v", err)
	}

	// Check that GH_AW_WORKFLOW_FILE uses the lock filename exactly
	stepsContent := strings.Join(job.Steps, "")
	if !strings.Contains(stepsContent, `GH_AW_WORKFLOW_FILE: "example.workflow.lock.yml"`) {
		t.Errorf("Expected GH_AW_WORKFLOW_FILE to be 'example.workflow.lock.yml', got steps content:\n%s", stepsContent)
	}
	// Verify it does NOT contain the incorrect .g. version
	if strings.Contains(stepsContent, "example.workflow.g.lock.yml") {
		t.Error("GH_AW_WORKFLOW_FILE should not contain '.g.' in the filename")
	}
}

// TestBuildMainJobWithActivation tests building the main job with activation dependency
func TestBuildMainJobWithActivation(t *testing.T) {
	compiler := NewCompiler()
	// Initialize stepOrderTracker
	compiler.stepOrderTracker = NewStepOrderTracker()

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		AI:          "copilot",
		RunsOn:      "runs-on: ubuntu-latest",
		Permissions: "permissions:\n  contents: read",
	}

	job, err := compiler.buildMainJob(workflowData, true)
	if err != nil {
		t.Fatalf("buildMainJob() returned error: %v", err)
	}

	if job.Name != string(constants.AgentJobName) {
		t.Errorf("Job name = %q, want %q", job.Name, string(constants.AgentJobName))
	}

	// Check that it depends on activation job
	found := slices.Contains(job.Needs, string(constants.ActivationJobName))
	if !found {
		t.Errorf("Expected job to depend on %s, got needs: %v", string(constants.ActivationJobName), job.Needs)
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

// TestBuildSafeOutputsJobsCreatesExpectedJobs tests that safe output steps are created correctly
// in the consolidated safe_outputs job
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

	// Check that the consolidated safe_outputs job is created
	if !containsInNonCommentLines(yamlStr, "safe_outputs:") {
		t.Error("Expected safe_outputs job not found in output")
	}

	// Check that the handler manager step is created (since create-issue, add-comment, and add-labels are now handled by the handler manager)
	expectedSteps := []string{
		"name: Process Safe Outputs",
		"id: process_safe_outputs",
	}
	for _, step := range expectedSteps {
		if !strings.Contains(yamlStr, step) {
			t.Errorf("Expected step %q not found in output", step)
		}
	}

	// Verify handler config contains all three enabled safe outputs
	if !strings.Contains(yamlStr, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG") {
		t.Error("Expected GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG in output")
	}
	if !strings.Contains(yamlStr, "create_issue") {
		t.Error("Expected create_issue in handler config")
	}
	if !strings.Contains(yamlStr, "add_comment") {
		t.Error("Expected add_comment in handler config")
	}
	if !strings.Contains(yamlStr, "add_labels") {
		t.Error("Expected add_labels in handler config")
	}

	// Check that the consolidated job has correct timeout (15 minutes for consolidated job)
	if !strings.Contains(yamlStr, "timeout-minutes: 15") {
		t.Error("Expected timeout-minutes: 15 for consolidated safe_outputs job")
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

	// Check that detection is inline in agent job (not a separate job)
	if containsInNonCommentLines(yamlStr, "\n  detection:\n") {
		t.Error("Expected detection to be inline in agent job, not a separate job")
	}

	// Check that agent job contains inline detection steps
	if !strings.Contains(yamlStr, "detection_guard") {
		t.Error("Expected agent job to contain detection_guard step")
	}
	if !strings.Contains(yamlStr, "detection_conclusion") {
		t.Error("Expected agent job to contain detection_conclusion step")
	}

	// Check that safe_outputs job references agent detection output
	if !strings.Contains(yamlStr, "needs.agent.outputs.detection_success") {
		t.Error("Expected safe output jobs to check needs.agent.outputs.detection_success")
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

	// Check that reusable workflow job is created
	if !containsInNonCommentLines(yamlStr, "call-other:") {
		t.Error("Expected call-other job")
	}

	// Check for uses directive
	if !strings.Contains(yamlStr, "uses: owner/repo/.github/workflows/reusable.yml@main") {
		t.Error("Expected uses directive for reusable workflow")
	}
}

// TestBuildJobsWithReusableWorkflowSecretsInherit tests that secrets: inherit is correctly emitted
// for reusable workflow call jobs.
func TestBuildJobsWithReusableWorkflowSecretsInherit(t *testing.T) {
	tmpDir := testutil.TempDir(t, "reusable-workflow-secrets-inherit-test")

	frontmatter := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  call-other:
    uses: owner/repo/.github/workflows/reusable.yml@main
    secrets: inherit
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

	content, err := os.ReadFile(filepath.Join(tmpDir, "test.lock.yml"))
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	yamlStr := string(content)

	if !strings.Contains(yamlStr, "uses: owner/repo/.github/workflows/reusable.yml@main") {
		t.Error("Expected uses directive for reusable workflow")
	}
	if !strings.Contains(yamlStr, "secrets: inherit") {
		t.Error("Expected 'secrets: inherit' directive")
	}
	if strings.Contains(yamlStr, "secrets:\n      ") {
		t.Error("Should not emit secrets map when using inherit")
	}
}

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

	// Check that job has outputs section
	if !strings.Contains(yamlStr, "outputs:") {
		t.Error("Expected outputs section")
	}

	// Check that result output is defined
	if !strings.Contains(yamlStr, "result:") {
		t.Error("Expected 'result' output")
	}
}

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
on: push
permissions:
  contents: read
engine: copilot
strict: false
command: /test
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

// TestJobsWithRepoMemoryDependencies tests push_repo_memory job positioning
// This tests the job creation logic when repo-memory config is present
func TestJobsWithRepoMemoryDependencies(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	// Create workflow data with repo-memory config
	data := &WorkflowData{
		Name:        "Test Workflow",
		AI:          "copilot",
		RunsOn:      "runs-on: ubuntu-latest",
		Permissions: "permissions:\n  contents: read",
		RepoMemoryConfig: &RepoMemoryConfig{
			Memories: []RepoMemoryEntry{
				{
					ID:         "test-memory",
					BranchName: "memory-branch",
					FileGlob:   []string{"data/**"},
				},
			},
		},
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				TitlePrefix: "[bot] ",
			},
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	// Build activation and agent jobs first
	compiler.stepOrderTracker = NewStepOrderTracker()
	activationJob, _ := compiler.buildActivationJob(data, false, "", "test.lock.yml")
	compiler.jobManager.AddJob(activationJob)

	agentJob, _ := compiler.buildMainJob(data, true)
	compiler.jobManager.AddJob(agentJob)

	// Build safe outputs jobs (creates detection job when threat detection is enabled)
	compiler.buildSafeOutputsJobs(data, string(constants.AgentJobName), "test.md")

	// Build push_repo_memory job
	threatDetectionEnabledForSafeJobs := data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil
	pushRepoMemoryJob, err := compiler.buildPushRepoMemoryJob(data, threatDetectionEnabledForSafeJobs)
	if err != nil {
		t.Fatalf("buildPushRepoMemoryJob() error: %v", err)
	}

	// Verify job was created
	if pushRepoMemoryJob == nil {
		t.Fatal("Expected push_repo_memory job to be created")
	}

	// Detection is inline in agent — no separate dependency
	// Verify push_repo_memory does NOT depend on detection job
	if threatDetectionEnabledForSafeJobs {
		hasDetectionDep := slices.Contains(pushRepoMemoryJob.Needs, string(constants.DetectionJobName))
		if hasDetectionDep {
			t.Error("push_repo_memory should not depend on detection job (detection is inline in agent)")
		}
	}

	// Verify job name
	if pushRepoMemoryJob.Name != "push_repo_memory" {
		t.Errorf("Expected job name 'push_repo_memory', got %q", pushRepoMemoryJob.Name)
	}
}

// TestJobsWithCacheMemoryDependencies tests update_cache_memory job positioning
// This tests the job creation logic when cache-memory config is present
func TestJobsWithCacheMemoryDependencies(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	// Create workflow data with cache-memory config
	data := &WorkflowData{
		Name:        "Test Workflow",
		AI:          "copilot",
		RunsOn:      "runs-on: ubuntu-latest",
		Permissions: "permissions:\n  contents: read",
		CacheMemoryConfig: &CacheMemoryConfig{
			Caches: []CacheMemoryEntry{
				{
					ID:  "test-cache",
					Key: "test-key",
				},
			},
		},
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				TitlePrefix: "[bot] ",
			},
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	// Build activation and agent jobs first
	compiler.stepOrderTracker = NewStepOrderTracker()
	activationJob, _ := compiler.buildActivationJob(data, false, "", "test.lock.yml")
	compiler.jobManager.AddJob(activationJob)

	agentJob, _ := compiler.buildMainJob(data, true)
	compiler.jobManager.AddJob(agentJob)

	// Build safe outputs jobs (creates detection job when threat detection is enabled)
	compiler.buildSafeOutputsJobs(data, string(constants.AgentJobName), "test.md")

	// Build update_cache_memory job (only created with threat detection)
	threatDetectionEnabledForSafeJobs := data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil
	if threatDetectionEnabledForSafeJobs {
		updateCacheMemoryJob, err := compiler.buildUpdateCacheMemoryJob(data, threatDetectionEnabledForSafeJobs)
		if err != nil {
			t.Fatalf("buildUpdateCacheMemoryJob() error: %v", err)
		}

		// Verify job was created
		if updateCacheMemoryJob == nil {
			t.Fatal("Expected update_cache_memory job to be created when threat detection is enabled")
		}

		// Verify dependencies — detection is inline in agent, no separate dependency
		hasDetectionDep := slices.Contains(updateCacheMemoryJob.Needs, string(constants.DetectionJobName))
		if hasDetectionDep {
			t.Error("update_cache_memory should not depend on detection job (detection is inline in agent)")
		}
		// Should depend on agent job
		hasAgentDep := slices.Contains(updateCacheMemoryJob.Needs, string(constants.AgentJobName))
		if !hasAgentDep {
			t.Error("Expected update_cache_memory to depend on agent job")
		}

		// Verify job name
		if updateCacheMemoryJob.Name != "update_cache_memory" {
			t.Errorf("Expected job name 'update_cache_memory', got %q", updateCacheMemoryJob.Name)
		}
	}
}

// TestUpdateCacheMemoryJobHasWorkflowIDEnv verifies that the update_cache_memory job
// includes GH_AW_WORKFLOW_ID_SANITIZED in its env block so cache keys match the agent job.
func TestUpdateCacheMemoryJobHasWorkflowIDEnv(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	data := &WorkflowData{
		Name:       "Test Workflow",
		WorkflowID: "daily-repo-status",
		AI:         "copilot",
		RunsOn:     "runs-on: ubuntu-latest",
		CacheMemoryConfig: &CacheMemoryConfig{
			Caches: []CacheMemoryEntry{
				{ID: "default"},
			},
		},
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	compiler.stepOrderTracker = NewStepOrderTracker()
	activationJob, _ := compiler.buildActivationJob(data, false, "", "test.lock.yml")
	compiler.jobManager.AddJob(activationJob)

	agentJob, _ := compiler.buildMainJob(data, true)
	compiler.jobManager.AddJob(agentJob)

	compiler.buildSafeOutputsJobs(data, string(constants.AgentJobName), "test.md")

	updateCacheMemoryJob, err := compiler.buildUpdateCacheMemoryJob(data, true)
	if err != nil {
		t.Fatalf("buildUpdateCacheMemoryJob() error: %v", err)
	}
	if updateCacheMemoryJob == nil {
		t.Fatal("Expected update_cache_memory job to be created")
	}

	// GH_AW_WORKFLOW_ID_SANITIZED must be present so the save key matches the restore key
	sanitizedID, ok := updateCacheMemoryJob.Env["GH_AW_WORKFLOW_ID_SANITIZED"]
	if !ok {
		t.Error("update_cache_memory job is missing GH_AW_WORKFLOW_ID_SANITIZED env var; cache keys will not match")
	}
	// "daily-repo-status" -> lowercase + hyphens removed -> "dailyrepostatus"
	if sanitizedID != "dailyrepostatus" {
		t.Errorf("GH_AW_WORKFLOW_ID_SANITIZED = %q, want %q", sanitizedID, "dailyrepostatus")
	}
}

// ========================================
// Edge Case Tests
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
on: push
permissions:
  contents: read
engine: copilot
strict: false
command: /test
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

// ========================================
// Additional Edge Case Tests
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

// TestBuildCustomJobsRunsOnForms tests that runs-on string, array, and object forms
// are all correctly handled in buildCustomJobs.
func TestBuildCustomJobsRunsOnForms(t *testing.T) {
	tests := []struct {
		name             string
		runsOn           any
		expectedRunsOn   string
		expectedContains []string
		shouldErr        bool
	}{
		{
			name:           "string form",
			runsOn:         "ubuntu-latest",
			expectedRunsOn: "runs-on: ubuntu-latest",
		},
		{
			name:             "array form",
			runsOn:           []any{"self-hosted", "linux", "large"},
			expectedContains: []string{"runs-on:", "- self-hosted", "- linux", "- large"},
		},
		{
			name:             "object form",
			runsOn:           map[string]any{"group": "my-runners"},
			expectedContains: []string{"runs-on:", "group: my-runners"},
		},
		{
			name:      "unmarshalable value returns error",
			runsOn:    make(chan int), // channels cannot be marshaled to YAML
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.jobManager = NewJobManager()

			data := &WorkflowData{
				Name:   "Test Workflow",
				AI:     "copilot",
				RunsOn: "runs-on: ubuntu-latest",
				Jobs: map[string]any{
					"my_job": map[string]any{
						"runs-on": tt.runsOn,
						"steps":   []any{map[string]any{"run": "echo hi"}},
					},
				},
			}

			err := compiler.buildCustomJobs(data, false)
			if tt.shouldErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), "my_job") {
					t.Errorf("Expected error to mention job name 'my_job', got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("buildCustomJobs() returned unexpected error: %v", err)
			}

			job, exists := compiler.jobManager.GetJob("my_job")
			if !exists {
				t.Fatal("Expected my_job to be added")
			}

			if tt.expectedRunsOn != "" {
				if job.RunsOn != tt.expectedRunsOn {
					t.Errorf("RunsOn = %q, want %q", job.RunsOn, tt.expectedRunsOn)
				}
			}
			for _, want := range tt.expectedContains {
				if !strings.Contains(job.RunsOn, want) {
					t.Errorf("RunsOn %q does not contain %q", job.RunsOn, want)
				}
			}
		})
	}
}

// TestBuildCustomJobsNewSimpleFields tests extraction of simple job fields via CompileWorkflow
func TestBuildCustomJobsNewSimpleFields(t *testing.T) {
	tmpDir := testutil.TempDir(t, "new-simple-fields-test")

	frontmatter := `---
on: push
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  featured_job:
    runs-on: ubuntu-latest
    name: My Display Name
    timeout-minutes: 30
    continue-on-error: true
    concurrency: my-group
    env:
      MY_VAR: hello
      OTHER_VAR: world
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
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("CompileWorkflow() error: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	yamlStr := string(content)

	// Verify display name
	if !strings.Contains(yamlStr, "name: My Display Name") {
		t.Errorf("Expected 'name: My Display Name' in output, got:\n%s", yamlStr)
	}

	// Verify timeout-minutes
	if !strings.Contains(yamlStr, "timeout-minutes: 30") {
		t.Errorf("Expected 'timeout-minutes: 30' in output, got:\n%s", yamlStr)
	}

	// Verify continue-on-error
	if !strings.Contains(yamlStr, "continue-on-error: true") {
		t.Errorf("Expected 'continue-on-error: true' in output, got:\n%s", yamlStr)
	}

	// Verify concurrency (string form)
	if !strings.Contains(yamlStr, "concurrency: my-group") {
		t.Errorf("Expected 'concurrency: my-group' in output, got:\n%s", yamlStr)
	}

	// Verify env variables
	if !strings.Contains(yamlStr, "MY_VAR: hello") {
		t.Errorf("Expected 'MY_VAR: hello' in output, got:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "OTHER_VAR: world") {
		t.Errorf("Expected 'OTHER_VAR: world' in output, got:\n%s", yamlStr)
	}
}

// TestBuildCustomJobsWithContainerAndServices tests extraction of container and services fields
func TestBuildCustomJobsWithContainerAndServices(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	data := &WorkflowData{
		Name:   "Test Workflow",
		AI:     "copilot",
		RunsOn: "runs-on: ubuntu-latest",
		Jobs: map[string]any{
			"services_job": map[string]any{
				"runs-on": "ubuntu-latest",
				"container": map[string]any{
					"image": "node:18",
				},
				"services": map[string]any{
					"redis": map[string]any{
						"image": "redis",
						"ports": []any{"6379:6379"},
					},
				},
				"steps": []any{
					map[string]any{"run": "echo 'test'"},
				},
			},
		},
	}

	err := compiler.buildCustomJobs(data, false)
	if err != nil {
		t.Fatalf("buildCustomJobs() returned error: %v", err)
	}

	job, exists := compiler.jobManager.GetJob("services_job")
	if !exists {
		t.Fatal("Expected services_job to be added")
	}

	// Verify container map form
	if !strings.Contains(job.Container, "container:") {
		t.Errorf("Expected 'container:' header, got: %q", job.Container)
	}
	if !strings.Contains(job.Container, "image: node:18") {
		t.Errorf("Expected 'image: node:18' in container, got: %q", job.Container)
	}

	// Verify services
	if !strings.Contains(job.Services, "services:") {
		t.Errorf("Expected 'services:' header, got: %q", job.Services)
	}
	if !strings.Contains(job.Services, "redis:") {
		t.Errorf("Expected 'redis:' in services, got: %q", job.Services)
	}
}

// TestBuildCustomJobsMapConcurrencyAndContainer tests map-form concurrency and string container
func TestBuildCustomJobsMapConcurrencyAndContainer(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	data := &WorkflowData{
		Name:   "Test Workflow",
		AI:     "copilot",
		RunsOn: "runs-on: ubuntu-latest",
		Jobs: map[string]any{
			"map_fields_job": map[string]any{
				"runs-on": "ubuntu-latest",
				"concurrency": map[string]any{
					"group":              "my-group",
					"cancel-in-progress": true,
				},
				"container": "node:18",
				"steps": []any{
					map[string]any{"run": "echo 'test'"},
				},
			},
		},
	}

	err := compiler.buildCustomJobs(data, false)
	if err != nil {
		t.Fatalf("buildCustomJobs() returned error: %v", err)
	}

	job, exists := compiler.jobManager.GetJob("map_fields_job")
	if !exists {
		t.Fatal("Expected map_fields_job to be added")
	}

	// Verify concurrency map form
	if !strings.Contains(job.Concurrency, "concurrency:") {
		t.Errorf("Expected 'concurrency:' header, got: %q", job.Concurrency)
	}
	if !strings.Contains(job.Concurrency, "group: my-group") {
		t.Errorf("Expected 'group: my-group' in concurrency, got: %q", job.Concurrency)
	}

	// Verify container string form
	if job.Container != "container: node:18" {
		t.Errorf("Expected 'container: node:18', got: %q", job.Container)
	}
}

// TestBuildCustomJobsMapConcurrencyDefaultsCancelInProgress tests that map-form concurrency
// without cancel-in-progress defaults to cancel-in-progress: false for non-agent jobs.
func TestBuildCustomJobsMapConcurrencyDefaultsCancelInProgress(t *testing.T) {
	tests := []struct {
		name            string
		concurrencyMap  map[string]any
		wantCancelFalse bool
		wantCancelTrue  bool
	}{
		{
			name: "no cancel-in-progress defaults to false",
			concurrencyMap: map[string]any{
				"group": "my-group",
			},
			wantCancelFalse: true,
			wantCancelTrue:  false,
		},
		{
			name: "explicit cancel-in-progress true is preserved",
			concurrencyMap: map[string]any{
				"group":              "my-group",
				"cancel-in-progress": true,
			},
			wantCancelFalse: false,
			wantCancelTrue:  true,
		},
		{
			name: "explicit cancel-in-progress false is preserved",
			concurrencyMap: map[string]any{
				"group":              "my-group",
				"cancel-in-progress": false,
			},
			wantCancelFalse: true,
			wantCancelTrue:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.jobManager = NewJobManager()

			data := &WorkflowData{
				Name:   "Test Workflow",
				AI:     "copilot",
				RunsOn: "runs-on: ubuntu-latest",
				Jobs: map[string]any{
					"custom_job": map[string]any{
						"runs-on":     "ubuntu-latest",
						"concurrency": tt.concurrencyMap,
						"steps":       []any{map[string]any{"run": "echo 'test'"}},
					},
				},
			}

			err := compiler.buildCustomJobs(data, false)
			if err != nil {
				t.Fatalf("buildCustomJobs() returned error: %v", err)
			}

			job, exists := compiler.jobManager.GetJob("custom_job")
			if !exists {
				t.Fatal("Expected custom_job to be added")
			}

			if !strings.Contains(job.Concurrency, "concurrency:") {
				t.Errorf("Expected 'concurrency:' header, got: %q", job.Concurrency)
			}
			if !strings.Contains(job.Concurrency, "group: my-group") {
				t.Errorf("Expected 'group: my-group' in concurrency, got: %q", job.Concurrency)
			}
			if tt.wantCancelFalse && !strings.Contains(job.Concurrency, "cancel-in-progress: false") {
				t.Errorf("Expected 'cancel-in-progress: false' in concurrency, got: %q", job.Concurrency)
			}
			if tt.wantCancelTrue && !strings.Contains(job.Concurrency, "cancel-in-progress: true") {
				t.Errorf("Expected 'cancel-in-progress: true' in concurrency, got: %q", job.Concurrency)
			}
			if !tt.wantCancelFalse && strings.Contains(job.Concurrency, "cancel-in-progress: false") {
				t.Errorf("Did not expect 'cancel-in-progress: false' in concurrency, got: %q", job.Concurrency)
			}
			if !tt.wantCancelTrue && strings.Contains(job.Concurrency, "cancel-in-progress: true") {
				t.Errorf("Did not expect 'cancel-in-progress: true' in concurrency, got: %q", job.Concurrency)
			}
		})
	}
}

// TestBuildCustomJobsAllNewFieldsViaWorkflowData tests all 7 new fields via direct WorkflowData
func TestBuildCustomJobsAllNewFieldsViaWorkflowData(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	data := &WorkflowData{
		Name:   "Test Workflow",
		AI:     "copilot",
		RunsOn: "runs-on: ubuntu-latest",
		Jobs: map[string]any{
			"full_job": map[string]any{
				"runs-on":           "ubuntu-latest",
				"name":              "My Display Name",
				"timeout-minutes":   30,
				"continue-on-error": true,
				"concurrency":       "ci-group",
				"env": map[string]any{
					"KEY1": "val1",
					"KEY2": "val2",
				},
				"container": map[string]any{
					"image": "ubuntu:22.04",
				},
				"services": map[string]any{
					"postgres": map[string]any{
						"image": "postgres:15",
					},
				},
				"steps": []any{
					map[string]any{"run": "echo 'test'"},
				},
			},
		},
	}

	err := compiler.buildCustomJobs(data, false)
	if err != nil {
		t.Fatalf("buildCustomJobs() returned error: %v", err)
	}

	job, exists := compiler.jobManager.GetJob("full_job")
	if !exists {
		t.Fatal("Expected full_job to be added")
	}

	// Verify all 7 fields
	if job.DisplayName != "My Display Name" {
		t.Errorf("DisplayName = %q, want 'My Display Name'", job.DisplayName)
	}
	if job.TimeoutMinutes != 30 {
		t.Errorf("TimeoutMinutes = %d, want 30", job.TimeoutMinutes)
	}
	if job.ContinueOnError == nil || !*job.ContinueOnError {
		t.Error("Expected ContinueOnError to be true")
	}
	if job.Concurrency != "concurrency: ci-group" {
		t.Errorf("Concurrency = %q, want 'concurrency: ci-group'", job.Concurrency)
	}
	if job.Env["KEY1"] != "val1" {
		t.Errorf("Env[KEY1] = %q, want 'val1'", job.Env["KEY1"])
	}
	if job.Env["KEY2"] != "val2" {
		t.Errorf("Env[KEY2] = %q, want 'val2'", job.Env["KEY2"])
	}
	if !strings.Contains(job.Container, "container:") {
		t.Errorf("Expected 'container:' in Container, got: %q", job.Container)
	}
	if !strings.Contains(job.Container, "image: ubuntu:22.04") {
		t.Errorf("Expected 'image: ubuntu:22.04' in Container, got: %q", job.Container)
	}
	if !strings.Contains(job.Services, "services:") {
		t.Errorf("Expected 'services:' in Services, got: %q", job.Services)
	}
	if !strings.Contains(job.Services, "postgres:") {
		t.Errorf("Expected 'postgres:' in Services, got: %q", job.Services)
	}

	// Verify the rendered YAML contains all fields
	jm := NewJobManager()
	if err := jm.AddJob(job); err != nil {
		t.Fatalf("AddJob() error: %v", err)
	}
	rendered := jm.RenderToYAML()

	renderedChecks := []string{
		"name: My Display Name",
		"timeout-minutes: 30",
		"continue-on-error: true",
		"concurrency: ci-group",
		"KEY1: val1",
		"KEY2: val2",
		"container:",
		"image: ubuntu:22.04",
		"services:",
		"postgres:",
	}
	for _, check := range renderedChecks {
		if !strings.Contains(rendered, check) {
			t.Errorf("Expected %q in rendered YAML, got:\n%s", check, rendered)
		}
	}
}
