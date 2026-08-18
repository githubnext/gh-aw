//go:build !integration

package workflow

import (
	"slices"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

// ========================================
// Built-in Job Construction Tests (pre-activation, activation, main)
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
	if !strings.Contains(stepsContent, "Check workflow lock file") {
		t.Error("Expected 'Check workflow lock file' step")
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

func TestBuildMainJobSkipsBuiltInJobCustomizationsFromNeeds(t *testing.T) {
	tests := []struct {
		name        string
		jobName     string
		forbidden   string
		verifyCount bool
	}{
		{
			name:        "activation pre-steps does not duplicate activation needs",
			jobName:     string(constants.ActivationJobName),
			forbidden:   string(constants.ActivationJobName),
			verifyCount: true,
		},
		{
			name:      "agent pre-steps does not create self-cycle",
			jobName:   string(constants.AgentJobName),
			forbidden: string(constants.AgentJobName),
		},
		{
			name:      "safe_outputs pre-steps does not create cycle with agent",
			jobName:   string(constants.SafeOutputsJobName),
			forbidden: string(constants.SafeOutputsJobName),
		},
		{
			name:      "conclusion pre-steps does not create cycle with agent",
			jobName:   string(constants.ConclusionJobName),
			forbidden: string(constants.ConclusionJobName),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.stepOrderTracker = NewStepOrderTracker()

			workflowData := &WorkflowData{
				Name:        "Test Workflow",
				AI:          "copilot",
				RunsOn:      "runs-on: ubuntu-latest",
				Permissions: "permissions:\n  contents: read",
				Jobs: map[string]any{
					tt.jobName: map[string]any{
						"pre-steps": []any{
							map[string]any{
								"name": "Pre-step",
								"run":  "echo test",
							},
						},
					},
				},
			}

			job, err := compiler.buildMainJob(workflowData, true)
			if err != nil {
				t.Fatalf("buildMainJob() returned error: %v", err)
			}

			if tt.verifyCount {
				count := 0
				for _, need := range job.Needs {
					if need == tt.forbidden {
						count++
					}
				}
				if count != 1 {
					t.Fatalf("Expected exactly one %q dependency, got %d (needs: %v)", tt.forbidden, count, job.Needs)
				}
			} else if slices.Contains(job.Needs, tt.forbidden) {
				t.Fatalf("Did not expect %q in agent needs, got: %v", tt.forbidden, job.Needs)
			}
		})
	}
}

func TestIsBuiltinJobName(t *testing.T) {
	tests := []struct {
		name     string
		jobName  string
		expected bool
	}{
		{name: "pre_activation canonical", jobName: string(constants.PreActivationJobName), expected: true},
		{name: "pre-activation alias", jobName: "pre-activation", expected: true},
		{name: "activation", jobName: string(constants.ActivationJobName), expected: true},
		{name: "agent", jobName: string(constants.AgentJobName), expected: true},
		{name: "safe_outputs canonical", jobName: string(constants.SafeOutputsJobName), expected: true},
		{name: "safe-outputs alias", jobName: "safe-outputs", expected: true},
		{name: "conclusion", jobName: string(constants.ConclusionJobName), expected: true},
		{name: "detection", jobName: string(constants.DetectionJobName), expected: true},
		{name: "upload_assets", jobName: string(constants.UploadAssetsJobName), expected: true},
		{name: "upload_code_scanning_sarif", jobName: string(constants.UploadCodeScanningJobName), expected: true},
		{name: "unlock", jobName: string(constants.UnlockJobName), expected: true},
		{name: "empty string", jobName: "", expected: false},
		{name: "different casing activation", jobName: "ACTIVATION", expected: false},
		{name: "different casing agent", jobName: "Agent", expected: false},
		{name: "partial pre-activation match", jobName: "pre-activation-custom", expected: false},
		{name: "partial agent match", jobName: "agent-step", expected: false},
		{name: "custom job", jobName: "custom_job", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := isBuiltinJobName(tt.jobName)
			if actual != tt.expected {
				t.Fatalf("isBuiltinJobName(%q) = %t, want %t", tt.jobName, actual, tt.expected)
			}
		})
	}
}

// TestBuildCustomJobsWithActivation tests building custom jobs with activation dependency
