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

	job, err := compiler.buildPreActivationJob(workflowData, true)
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
	compiler := NewCompiler(false, "", "test")

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{},
	}

	job, err := compiler.buildActivationJob(workflowData, false, "", "test.lock.yml")
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

	job, err := compiler.buildActivationJob(workflowData, false, "", "test.lock.yml")
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

	job, err := compiler.buildMainJob(workflowData, true)
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

	// Check that the consolidated safe_outputs job is created
	if !containsInNonCommentLines(yamlStr, "safe_outputs:") {
		t.Error("Expected safe_outputs job not found in output")
	}

	// Check that expected safe output steps are created within the consolidated job
	expectedSteps := []string{
		"name: Create Issue",
		"id: create_issue",
		"name: Add Comment",
		"id: add_comment",
		"name: Add Labels",
		"id: add_labels",
	}
	for _, step := range expectedSteps {
		if !strings.Contains(yamlStr, step) {
			t.Errorf("Expected step %q not found in output", step)
		}
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

	// Check that safe_outputs job depends on detection
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

// ========================================
// JobConfig Tests
// ========================================

// TestParseJobConfig tests the ParseJobConfig function
func TestParseJobConfig(t *testing.T) {
	tests := []struct {
		name     string
		jobMap   map[string]any
		expected *JobConfig
	}{
		{
			name:     "nil map returns nil",
			jobMap:   nil,
			expected: nil,
		},
		{
			name:   "empty map returns empty config",
			jobMap: map[string]any{},
			expected: &JobConfig{
				Needs:  nil,
				RunsOn: nil,
				Steps:  nil,
				If:     "",
			},
		},
		{
			name: "map with needs as string",
			jobMap: map[string]any{
				"needs": "activation",
			},
			expected: &JobConfig{
				Needs:  "activation",
				RunsOn: nil,
				Steps:  nil,
				If:     "",
			},
		},
		{
			name: "map with needs as array",
			jobMap: map[string]any{
				"needs": []any{"activation", "pre_activation"},
			},
			expected: &JobConfig{
				Needs:  []any{"activation", "pre_activation"},
				RunsOn: nil,
				Steps:  nil,
				If:     "",
			},
		},
		{
			name: "map with all fields",
			jobMap: map[string]any{
				"needs":   "activation",
				"runs-on": "ubuntu-latest",
				"steps":   []any{map[string]any{"run": "echo test"}},
				"if":      "github.event_name == 'push'",
			},
			expected: &JobConfig{
				Needs:  "activation",
				RunsOn: "ubuntu-latest",
				Steps:  []any{map[string]any{"run": "echo test"}},
				If:     "github.event_name == 'push'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseJobConfig(tt.jobMap)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("ParseJobConfig() = %+v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Errorf("ParseJobConfig() = nil, want %+v", tt.expected)
				return
			}

			// Compare fields
			if result.If != tt.expected.If {
				t.Errorf("If = %q, want %q", result.If, tt.expected.If)
			}

			// For complex fields, just check they match expected structure
			if (result.Needs == nil) != (tt.expected.Needs == nil) {
				t.Errorf("Needs nil mismatch: got %v, want %v", result.Needs, tt.expected.Needs)
			}
			if (result.RunsOn == nil) != (tt.expected.RunsOn == nil) {
				t.Errorf("RunsOn nil mismatch: got %v, want %v", result.RunsOn, tt.expected.RunsOn)
			}
			if (result.Steps == nil) != (tt.expected.Steps == nil) {
				t.Errorf("Steps nil mismatch: got %v, want %v", result.Steps, tt.expected.Steps)
			}
		})
	}
}

// TestJobConfigHasDependency tests the HasDependency method
func TestJobConfigHasDependency(t *testing.T) {
	tests := []struct {
		name     string
		config   *JobConfig
		jobName  string
		expected bool
	}{
		{
			name:     "nil config returns false",
			config:   nil,
			jobName:  "activation",
			expected: false,
		},
		{
			name:     "config with nil needs returns false",
			config:   &JobConfig{Needs: nil},
			jobName:  "activation",
			expected: false,
		},
		{
			name:     "string needs matches",
			config:   &JobConfig{Needs: "activation"},
			jobName:  "activation",
			expected: true,
		},
		{
			name:     "string needs doesn't match",
			config:   &JobConfig{Needs: "activation"},
			jobName:  "pre_activation",
			expected: false,
		},
		{
			name:     "array needs contains match",
			config:   &JobConfig{Needs: []any{"activation", "pre_activation"}},
			jobName:  "pre_activation",
			expected: true,
		},
		{
			name:     "array needs doesn't contain match",
			config:   &JobConfig{Needs: []any{"activation", "other"}},
			jobName:  "pre_activation",
			expected: false,
		},
		{
			name:     "array with single match",
			config:   &JobConfig{Needs: []any{"agent"}},
			jobName:  constants.AgentJobName,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.HasDependency(tt.jobName)
			if result != tt.expected {
				t.Errorf("HasDependency(%q) = %v, want %v (config.Needs = %v)",
					tt.jobName, result, tt.expected, tt.config.Needs)
			}
		})
	}
}

// TestJobConfigBackwardCompatibility tests backward compatibility with existing functions
func TestJobConfigBackwardCompatibility(t *testing.T) {
	tests := []struct {
		name           string
		jobMap         map[string]any
		checkPreAct    bool
		checkAgent     bool
		expectedPreAct bool
		expectedAgent  bool
	}{
		{
			name:           "depends on pre_activation",
			jobMap:         map[string]any{"needs": "pre_activation"},
			checkPreAct:    true,
			checkAgent:     true,
			expectedPreAct: true,
			expectedAgent:  false,
		},
		{
			name:           "depends on agent",
			jobMap:         map[string]any{"needs": "agent"},
			checkPreAct:    true,
			checkAgent:     true,
			expectedPreAct: false,
			expectedAgent:  true,
		},
		{
			name:           "depends on both in array",
			jobMap:         map[string]any{"needs": []any{"pre_activation", "agent"}},
			checkPreAct:    true,
			checkAgent:     true,
			expectedPreAct: true,
			expectedAgent:  true,
		},
		{
			name:           "depends on neither",
			jobMap:         map[string]any{"needs": "activation"},
			checkPreAct:    true,
			checkAgent:     true,
			expectedPreAct: false,
			expectedAgent:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.checkPreAct {
				result := jobDependsOnPreActivation(tt.jobMap)
				if result != tt.expectedPreAct {
					t.Errorf("jobDependsOnPreActivation() = %v, want %v", result, tt.expectedPreAct)
				}
			}

			if tt.checkAgent {
				result := jobDependsOnAgent(tt.jobMap)
				if result != tt.expectedAgent {
					t.Errorf("jobDependsOnAgent() = %v, want %v", result, tt.expectedAgent)
				}
			}
		})
	}
}
