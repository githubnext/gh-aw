package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestNotifyWithCommentJob(t *testing.T) {
	tests := []struct {
		name                   string
		addCommentConfig       bool
		threatDetectionEnabled bool
		expectJob              bool
		expectConditions       []string
		expectNeeds            []string
	}{
		{
			name:                   "notify job created when add-comment is configured",
			addCommentConfig:       true,
			threatDetectionEnabled: false,
			expectJob:              true,
			expectConditions: []string{
				"always()",
				"needs.activation.outputs.comment_id",
				"!(contains(needs.agent.outputs.output_types, 'add_comment'))",
				"!(contains(needs.agent.outputs.output_types, 'create_pull_request'))",
			},
			expectNeeds: []string{constants.AgentJobName, constants.ActivationJobName},
		},
		{
			name:                   "notify job includes detection dependency when threat detection is enabled",
			addCommentConfig:       true,
			threatDetectionEnabled: true,
			expectJob:              true,
			expectConditions: []string{
				"always()",
				"needs.activation.outputs.comment_id",
				"!(contains(needs.agent.outputs.output_types, 'add_comment'))",
				"!(contains(needs.agent.outputs.output_types, 'create_pull_request'))",
			},
			expectNeeds: []string{constants.AgentJobName, constants.ActivationJobName, constants.DetectionJobName},
		},
		{
			name:                   "notify job not created when add-comment is not configured",
			addCommentConfig:       false,
			threatDetectionEnabled: false,
			expectJob:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test workflow
			compiler := NewCompiler(false, "", "")
			workflowData := &WorkflowData{
				Name: "Test Workflow",
			}

			if tt.addCommentConfig {
				workflowData.SafeOutputs = &SafeOutputsConfig{
					AddComments: &AddCommentsConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Max: 1,
						},
					},
				}
			}

			// Build the notify job
			job, err := compiler.buildNotifyWithCommentJob(workflowData, constants.AgentJobName, tt.threatDetectionEnabled)
			if err != nil {
				t.Fatalf("Failed to build notify_with_comment job: %v", err)
			}

			if tt.expectJob {
				if job == nil {
					t.Fatal("Expected notify_with_comment job to be created, but got nil")
				}

				// Check job name
				if job.Name != "notify_with_comment" {
					t.Errorf("Expected job name 'notify_with_comment', got '%s'", job.Name)
				}

				// Check conditions
				for _, expectedCond := range tt.expectConditions {
					if !strings.Contains(job.If, expectedCond) {
						t.Errorf("Expected condition '%s' to be in job.If, but it wasn't.\nActual If: %s", expectedCond, job.If)
					}
				}

				// Check needs
				if len(job.Needs) != len(tt.expectNeeds) {
					t.Errorf("Expected %d needs, got %d: %v", len(tt.expectNeeds), len(job.Needs), job.Needs)
				}
				for _, expectedNeed := range tt.expectNeeds {
					found := false
					for _, need := range job.Needs {
						if need == expectedNeed {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected need '%s' not found in job.Needs: %v", expectedNeed, job.Needs)
					}
				}

				// Check permissions
				if !strings.Contains(job.Permissions, "issues: write") {
					t.Error("Expected 'issues: write' permission in notify_with_comment job")
				}
				if !strings.Contains(job.Permissions, "pull-requests: write") {
					t.Error("Expected 'pull-requests: write' permission in notify_with_comment job")
				}
				if !strings.Contains(job.Permissions, "discussions: write") {
					t.Error("Expected 'discussions: write' permission in notify_with_comment job")
				}

				// Check that the job has the update comment step
				stepsYAML := strings.Join(job.Steps, "")
				if !strings.Contains(stepsYAML, "Update comment with error notification") {
					t.Error("Expected 'Update comment with error notification' step in notify_with_comment job")
				}
				if !strings.Contains(stepsYAML, "GITHUB_AW_COMMENT_ID") {
					t.Error("Expected GITHUB_AW_COMMENT_ID environment variable in notify_with_comment job")
				}
				if !strings.Contains(stepsYAML, "GITHUB_AW_AGENT_CONCLUSION") {
					t.Error("Expected GITHUB_AW_AGENT_CONCLUSION environment variable in notify_with_comment job")
				}
			} else {
				if job != nil {
					t.Errorf("Expected no notify_with_comment job, but got one: %v", job)
				}
			}
		})
	}
}

func TestNotifyWithCommentJobIntegration(t *testing.T) {
	// Test that the job is properly integrated with activation job outputs
	compiler := NewCompiler(false, "", "")
	workflowData := &WorkflowData{
		Name:       "Test Workflow",
		AIReaction: "eyes", // This causes the activation job to create a comment
		SafeOutputs: &SafeOutputsConfig{
			AddComments: &AddCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
			},
		},
	}

	// Build the notify job
	job, err := compiler.buildNotifyWithCommentJob(workflowData, constants.AgentJobName, false)
	if err != nil {
		t.Fatalf("Failed to build notify_with_comment job: %v", err)
	}

	if job == nil {
		t.Fatal("Expected notify_with_comment job to be created")
	}

	// Convert job to YAML string for checking
	jobYAML := strings.Join(job.Steps, "")

	// Check that the job references activation outputs
	if !strings.Contains(job.If, "needs.activation.outputs.comment_id") {
		t.Error("Expected notify_with_comment to reference activation.outputs.comment_id")
	}

	// Check that environment variables reference activation outputs
	if !strings.Contains(jobYAML, "needs.activation.outputs.comment_id") {
		t.Error("Expected GITHUB_AW_COMMENT_ID to reference activation.outputs.comment_id")
	}
	if !strings.Contains(jobYAML, "needs.activation.outputs.comment_repo") {
		t.Error("Expected GITHUB_AW_COMMENT_REPO to reference activation.outputs.comment_repo")
	}

	// Check that agent result is referenced
	if !strings.Contains(jobYAML, "needs.agent.result") {
		t.Error("Expected GITHUB_AW_AGENT_CONCLUSION to reference needs.agent.result")
	}

	// Check all four conditions are present
	if !strings.Contains(job.If, "always()") {
		t.Error("Expected always() in notify_with_comment condition")
	}
	if !strings.Contains(job.If, "needs.activation.outputs.comment_id") {
		t.Error("Expected comment_id check in notify_with_comment condition")
	}
	if !strings.Contains(job.If, "!(contains(needs.agent.outputs.output_types, 'add_comment'))") {
		t.Error("Expected NOT contains add_comment check in notify_with_comment condition")
	}
	if !strings.Contains(job.If, "!(contains(needs.agent.outputs.output_types, 'create_pull_request'))") {
		t.Error("Expected NOT contains create_pull_request check in notify_with_comment condition")
	}
}
