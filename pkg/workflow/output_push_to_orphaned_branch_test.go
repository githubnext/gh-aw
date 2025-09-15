package workflow

import (
	"strings"
	"testing"
)

func TestBuildCreateOutputPushToOrphanedBranchJob(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	t.Run("basic_configuration", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			SafeOutputs: &SafeOutputsConfig{
				PushToOrphanedBranch: &PushToOrphanedBranchConfig{
					Max: 3,
				},
			},
		}

		job, err := compiler.buildCreateOutputPushToOrphanedBranchJob(workflowData, "main_job")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if job.Name != "push_to_orphaned_branch" {
			t.Errorf("Expected job name 'push_to_orphaned_branch', got: %s", job.Name)
		}

		if job.If != "always()" {
			t.Errorf("Expected job condition 'always()', got: %s", job.If)
		}

		if !strings.Contains(job.Permissions, "contents: write") {
			t.Errorf("Expected job to have contents: write permission")
		}

		if job.TimeoutMinutes != 10 {
			t.Errorf("Expected timeout of 10 minutes, got: %d", job.TimeoutMinutes)
		}

		// Check for default branch name in environment variables
		stepsStr := strings.Join(job.Steps, "")
		if !strings.Contains(stepsStr, "GITHUB_AW_ORPHANED_BRANCH_NAME: assets/test-workflow") {
			t.Errorf("Expected default branch name 'assets/test-workflow' in steps")
		}

		// Check that the main job is a dependency
		found := false
		for _, need := range job.Needs {
			if need == "main_job" {
				t.Logf("Found expected dependency: %s", need)
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected 'main_job' to be in needs, got: %v", job.Needs)
		}

		// Check for expected outputs
		if _, ok := job.Outputs["uploaded_files"]; !ok {
			t.Errorf("Expected 'uploaded_files' output to be present")
		}
		if _, ok := job.Outputs["file_urls"]; !ok {
			t.Errorf("Expected 'file_urls' output to be present")
		}

		// Check that steps contain expected elements
		stepsString := strings.Join(job.Steps, "")
		if !strings.Contains(stepsString, "Checkout repository") {
			t.Errorf("Expected checkout step")
		}
		if !strings.Contains(stepsString, "Push to Orphaned Branch") {
			t.Errorf("Expected push to orphaned branch step")
		}
		if !strings.Contains(stepsString, "GITHUB_AW_ORPHANED_BRANCH_MAX_COUNT: 3") {
			t.Errorf("Expected max count environment variable to be set")
		}
	})

	t.Run("default_max_count", func(t *testing.T) {
		workflowData := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				PushToOrphanedBranch: &PushToOrphanedBranchConfig{},
			},
		}

		job, err := compiler.buildCreateOutputPushToOrphanedBranchJob(workflowData, "main_job")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		stepsStr := strings.Join(job.Steps, "")
		if !strings.Contains(stepsStr, "GITHUB_AW_ORPHANED_BRANCH_MAX_COUNT: 1") {
			t.Errorf("Expected default max count of 1")
		}
	})

	t.Run("command_workflow_condition", func(t *testing.T) {
		workflowData := &WorkflowData{
			Command: "upload-files",
			SafeOutputs: &SafeOutputsConfig{
				PushToOrphanedBranch: &PushToOrphanedBranchConfig{},
			},
		}

		job, err := compiler.buildCreateOutputPushToOrphanedBranchJob(workflowData, "main_job")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should have command trigger condition
		if !strings.Contains(job.If, "upload-files") {
			t.Errorf("Expected command condition in job.If, got: %s", job.If)
		}
	})

	t.Run("missing_configuration", func(t *testing.T) {
		workflowData := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{},
		}

		_, err := compiler.buildCreateOutputPushToOrphanedBranchJob(workflowData, "main_job")
		if err == nil {
			t.Fatalf("Expected error for missing configuration")
		}

		if !strings.Contains(err.Error(), "safe-outputs.push-to-orphaned-branch configuration is required") {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})
}

func TestBuildCreateOutputPushToOrphanedBranchJobWithCustomBranch(t *testing.T) {
	compiler := NewCompiler(false, "", "1.0.0")

	t.Run("custom_branch_configuration", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			SafeOutputs: &SafeOutputsConfig{
				PushToOrphanedBranch: &PushToOrphanedBranchConfig{
					Max:    2,
					Branch: "custom-uploads",
				},
			},
		}

		job, err := compiler.buildCreateOutputPushToOrphanedBranchJob(workflowData, "main_job")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Check for custom branch name in environment variables
		stepsStr := strings.Join(job.Steps, "")
		if !strings.Contains(stepsStr, "GITHUB_AW_ORPHANED_BRANCH_NAME: custom-uploads") {
			t.Errorf("Expected custom branch name 'custom-uploads' in steps, got: %s", stepsStr)
		}

		// Check that max count is correctly set
		if !strings.Contains(stepsStr, "GITHUB_AW_ORPHANED_BRANCH_MAX_COUNT: 2") {
			t.Errorf("Expected max count 2 in environment variables")
		}
	})
}

func TestHasSafeOutputsEnabledWithOrphanedBranch(t *testing.T) {
	t.Run("enabled_with_orphaned_branch", func(t *testing.T) {
		config := &SafeOutputsConfig{
			PushToOrphanedBranch: &PushToOrphanedBranchConfig{},
		}

		if !HasSafeOutputsEnabled(config) {
			t.Errorf("Expected safe outputs to be enabled with orphaned branch config")
		}
	})

	t.Run("disabled_without_orphaned_branch", func(t *testing.T) {
		config := &SafeOutputsConfig{}

		if HasSafeOutputsEnabled(config) {
			t.Errorf("Expected safe outputs to be disabled without any config")
		}
	})
}
