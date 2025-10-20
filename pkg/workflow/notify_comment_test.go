package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestUpdateReactionJob(t *testing.T) {
	tests := []struct {
		name               string
		addCommentConfig   bool
		safeOutputJobNames []string
		expectJob          bool
		expectConditions   []string
		expectNeeds        []string
	}{
		{
			name:               "update_reaction job created when add-comment is configured",
			addCommentConfig:   true,
			safeOutputJobNames: []string{"add_comment", "missing_tool"},
			expectJob:          true,
			expectConditions: []string{
				"always()",
				"needs.agent.result != 'skipped'",
				"needs.activation.outputs.comment_id",
				"!(contains(needs.agent.outputs.output_types, 'add_comment'))",
				"!(contains(needs.agent.outputs.output_types, 'create_pull_request'))",
				"!(contains(needs.agent.outputs.output_types, 'push_to_pull_request_branch'))",
			},
			expectNeeds: []string{constants.AgentJobName, constants.ActivationJobName, "add_comment", "missing_tool"},
		},
		{
			name:               "update_reaction job depends on all safe output jobs",
			addCommentConfig:   true,
			safeOutputJobNames: []string{"add_comment", "create_issue", "missing_tool"},
			expectJob:          true,
			expectConditions: []string{
				"always()",
				"needs.agent.result != 'skipped'",
				"needs.activation.outputs.comment_id",
				"!(contains(needs.agent.outputs.output_types, 'add_comment'))",
				"!(contains(needs.agent.outputs.output_types, 'create_pull_request'))",
				"!(contains(needs.agent.outputs.output_types, 'push_to_pull_request_branch'))",
			},
			expectNeeds: []string{constants.AgentJobName, constants.ActivationJobName, "add_comment", "create_issue", "missing_tool"},
		},
		{
			name:               "update_reaction job not created when add-comment is not configured",
			addCommentConfig:   false,
			safeOutputJobNames: []string{},
			expectJob:          false,
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

			// Build the update_reaction job
			job, err := compiler.buildUpdateReactionJob(workflowData, constants.AgentJobName, tt.safeOutputJobNames)
			if err != nil {
				t.Fatalf("Failed to build update_reaction job: %v", err)
			}

			if tt.expectJob {
				if job == nil {
					t.Fatal("Expected update_reaction job to be created, but got nil")
				}

				// Check job name
				if job.Name != "update_reaction" {
					t.Errorf("Expected job name 'update_reaction', got '%s'", job.Name)
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
					t.Error("Expected 'issues: write' permission in update_reaction job")
				}
				if !strings.Contains(job.Permissions, "pull-requests: write") {
					t.Error("Expected 'pull-requests: write' permission in update_reaction job")
				}
				if !strings.Contains(job.Permissions, "discussions: write") {
					t.Error("Expected 'discussions: write' permission in update_reaction job")
				}

				// Check that the job has the update reaction step
				stepsYAML := strings.Join(job.Steps, "")
				if !strings.Contains(stepsYAML, "Update reaction comment with completion status") {
					t.Error("Expected 'Update reaction comment with completion status' step in update_reaction job")
				}
				if !strings.Contains(stepsYAML, "GH_AW_COMMENT_ID") {
					t.Error("Expected GH_AW_COMMENT_ID environment variable in update_reaction job")
				}
				if !strings.Contains(stepsYAML, "GH_AW_AGENT_CONCLUSION") {
					t.Error("Expected GH_AW_AGENT_CONCLUSION environment variable in update_reaction job")
				}
			} else {
				if job != nil {
					t.Errorf("Expected no update_reaction job, but got one: %v", job)
				}
			}
		})
	}
}

func TestUpdateReactionJobIntegration(t *testing.T) {
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

	// Build the update_reaction job with sample safe output job names
	safeOutputJobNames := []string{"add_comment", "missing_tool"}
	job, err := compiler.buildUpdateReactionJob(workflowData, constants.AgentJobName, safeOutputJobNames)
	if err != nil {
		t.Fatalf("Failed to build update_reaction job: %v", err)
	}

	if job == nil {
		t.Fatal("Expected update_reaction job to be created")
	}

	// Convert job to YAML string for checking
	jobYAML := strings.Join(job.Steps, "")

	// Check that the job references activation outputs
	if !strings.Contains(job.If, "needs.activation.outputs.comment_id") {
		t.Error("Expected update_reaction to reference activation.outputs.comment_id")
	}

	// Check that environment variables reference activation outputs
	if !strings.Contains(jobYAML, "needs.activation.outputs.comment_id") {
		t.Error("Expected GH_AW_COMMENT_ID to reference activation.outputs.comment_id")
	}
	if !strings.Contains(jobYAML, "needs.activation.outputs.comment_repo") {
		t.Error("Expected GH_AW_COMMENT_REPO to reference activation.outputs.comment_repo")
	}

	// Check that agent result is referenced
	if !strings.Contains(jobYAML, "needs.agent.result") {
		t.Error("Expected GH_AW_AGENT_CONCLUSION to reference needs.agent.result")
	}

	// Check all six conditions are present
	if !strings.Contains(job.If, "always()") {
		t.Error("Expected always() in update_reaction condition")
	}
	if !strings.Contains(job.If, "needs.agent.result != 'skipped'") {
		t.Error("Expected agent not skipped check in update_reaction condition")
	}
	if !strings.Contains(job.If, "needs.activation.outputs.comment_id") {
		t.Error("Expected comment_id check in update_reaction condition")
	}
	if !strings.Contains(job.If, "!(contains(needs.agent.outputs.output_types, 'add_comment'))") {
		t.Error("Expected NOT contains add_comment check in update_reaction condition")
	}
	if !strings.Contains(job.If, "!(contains(needs.agent.outputs.output_types, 'create_pull_request'))") {
		t.Error("Expected NOT contains create_pull_request check in update_reaction condition")
	}
	if !strings.Contains(job.If, "!(contains(needs.agent.outputs.output_types, 'push_to_pull_request_branch'))") {
		t.Error("Expected NOT contains push_to_pull_request_branch check in update_reaction condition")
	}

	// Verify job depends on the safe output jobs
	for _, expectedNeed := range safeOutputJobNames {
		found := false
		for _, need := range job.Needs {
			if need == expectedNeed {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected update_reaction job to depend on '%s'", expectedNeed)
		}
	}
}
