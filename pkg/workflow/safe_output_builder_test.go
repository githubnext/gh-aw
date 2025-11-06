package workflow

import (
	"strings"
	"testing"
)

// TestSafeOutputBuilderConsistency validates that the fluent builder pattern
// produces consistent results across all safe output job types
func TestSafeOutputBuilderConsistency(t *testing.T) {
	tests := []struct {
		name                string
		buildFunc           func(*Compiler, *WorkflowData, string) (*Job, error)
		safeOutputsConfig   *SafeOutputsConfig
		expectedJobName     string
		expectedStepID      string
		expectedOutputKey   string
		expectedEnvVarCount int // Approximate number of env vars
	}{
		{
			name: "create_issue builder",
			buildFunc: func(c *Compiler, data *WorkflowData, mainJob string) (*Job, error) {
				return c.buildCreateOutputIssueJob(data, mainJob)
			},
			safeOutputsConfig: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 1},
					TitlePrefix:          "[bot] ",
					Labels:               []string{"automation"},
				},
				Staged: true,
			},
			expectedJobName:     "create_issue",
			expectedStepID:      "create_issue",
			expectedOutputKey:   "issue_number",
			expectedEnvVarCount: 4,
		},
		{
			name: "update_issue builder",
			buildFunc: func(c *Compiler, data *WorkflowData, mainJob string) (*Job, error) {
				return c.buildCreateOutputUpdateIssueJob(data, mainJob)
			},
			safeOutputsConfig: &SafeOutputsConfig{
				UpdateIssues: &UpdateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 1},
					Status:               new(bool),
					Title:                new(bool),
				},
				Staged: true,
			},
			expectedJobName:     "update_issue",
			expectedStepID:      "update_issue",
			expectedOutputKey:   "issue_number",
			expectedEnvVarCount: 4,
		},
		{
			name: "create_discussion builder",
			buildFunc: func(c *Compiler, data *WorkflowData, mainJob string) (*Job, error) {
				return c.buildCreateOutputDiscussionJob(data, mainJob)
			},
			safeOutputsConfig: &SafeOutputsConfig{
				CreateDiscussions: &CreateDiscussionsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 1},
					TitlePrefix:          "[report] ",
					Category:             "General",
				},
				Staged: true,
			},
			expectedJobName:     "create_discussion",
			expectedStepID:      "create_discussion",
			expectedOutputKey:   "discussion_number",
			expectedEnvVarCount: 3,
		},
		{
			name: "create_pull_request builder",
			buildFunc: func(c *Compiler, data *WorkflowData, mainJob string) (*Job, error) {
				return c.buildCreateOutputPullRequestJob(data, mainJob)
			},
			safeOutputsConfig: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{},
					TitlePrefix:          "[auto] ",
					Labels:               []string{"automated"},
				},
				Staged: true,
			},
			expectedJobName:     "create_pull_request",
			expectedStepID:      "create_pull_request",
			expectedOutputKey:   "pull_request_number",
			expectedEnvVarCount: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler(false, "", "test")
			data := &WorkflowData{
				Name:        "test-workflow",
				SafeOutputs: tt.safeOutputsConfig,
			}

			job, err := tt.buildFunc(c, data, "main_job")
			if err != nil {
				t.Fatalf("Unexpected error building job: %v", err)
			}

			// Verify job name
			if job.Name != tt.expectedJobName {
				t.Errorf("Expected job name %q, got %q", tt.expectedJobName, job.Name)
			}

			// Verify steps contain the expected step ID
			stepsContent := strings.Join(job.Steps, "")
			if !strings.Contains(stepsContent, "id: "+tt.expectedStepID) {
				t.Errorf("Expected step ID %q not found in job steps", tt.expectedStepID)
			}

			// Verify outputs contain the expected key
			if _, exists := job.Outputs[tt.expectedOutputKey]; !exists {
				t.Errorf("Expected output key %q not found in job outputs", tt.expectedOutputKey)
			}

			// Verify staged flag is set when configured
			if tt.safeOutputsConfig.Staged {
				if !strings.Contains(stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED: \"true\"") {
					t.Error("Expected GH_AW_SAFE_OUTPUTS_STAGED to be set when staged is true")
				}
			}

			// Verify timeout is set to 10 minutes (standard for all safe output jobs)
			if job.TimeoutMinutes != 10 {
				t.Errorf("Expected timeout of 10 minutes, got %d", job.TimeoutMinutes)
			}

			// Verify needs includes main job
			foundMainJob := false
			for _, need := range job.Needs {
				if need == "main_job" {
					foundMainJob = true
					break
				}
			}
			if !foundMainJob {
				t.Error("Expected job to depend on 'main_job'")
			}

			// Verify job condition is set
			if job.If == "" {
				t.Error("Expected job to have a condition (if clause)")
			}

			// Verify standard components exist
			if !strings.Contains(stepsContent, "Download agent output artifact") {
				t.Error("Expected 'Download agent output artifact' step not found")
			}
		})
	}
}

// TestBuilderErrorHandling verifies that the builder properly handles error cases
func TestBuilderErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		buildFunc     func(*Compiler, *WorkflowData, string) (*Job, error)
		data          *WorkflowData
		expectedError string
	}{
		{
			name: "create_issue with nil config",
			buildFunc: func(c *Compiler, data *WorkflowData, mainJob string) (*Job, error) {
				return c.buildCreateOutputIssueJob(data, mainJob)
			},
			data: &WorkflowData{
				Name:        "test",
				SafeOutputs: nil,
			},
			expectedError: "safe-outputs.create-issue configuration is required",
		},
		{
			name: "update_issue with nil config",
			buildFunc: func(c *Compiler, data *WorkflowData, mainJob string) (*Job, error) {
				return c.buildCreateOutputUpdateIssueJob(data, mainJob)
			},
			data: &WorkflowData{
				Name: "test",
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{},
				},
			},
			expectedError: "safe-outputs.update-issue configuration is required",
		},
		{
			name: "create_discussion with nil config",
			buildFunc: func(c *Compiler, data *WorkflowData, mainJob string) (*Job, error) {
				return c.buildCreateOutputDiscussionJob(data, mainJob)
			},
			data: &WorkflowData{
				Name:        "test",
				SafeOutputs: &SafeOutputsConfig{},
			},
			expectedError: "safe-outputs.create-discussion configuration is required",
		},
		{
			name: "create_pull_request with nil config",
			buildFunc: func(c *Compiler, data *WorkflowData, mainJob string) (*Job, error) {
				return c.buildCreateOutputPullRequestJob(data, mainJob)
			},
			data: &WorkflowData{
				Name:        "test",
				SafeOutputs: nil,
			},
			expectedError: "safe-outputs.create-pull-request configuration is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler(false, "", "test")
			_, err := tt.buildFunc(c, tt.data, "main_job")

			if err == nil {
				t.Fatal("Expected error but got nil")
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

// TestBuilderTokenAndTargetRepo verifies that token and target repo are properly handled
func TestBuilderTokenAndTargetRepo(t *testing.T) {
	c := NewCompiler(false, "", "test")

	tests := []struct {
		name              string
		token             string
		targetRepoSlug    string
		expectedToken     string
		expectedTargetEnv string
	}{
		{
			name:              "with custom token and target repo",
			token:             "${{ secrets.CUSTOM_TOKEN }}",
			targetRepoSlug:    "owner/target-repo",
			expectedToken:     "${{ secrets.CUSTOM_TOKEN }}",
			expectedTargetEnv: "GH_AW_TARGET_REPO_SLUG: \"owner/target-repo\"",
		},
		{
			name:              "without token or target repo",
			token:             "",
			targetRepoSlug:    "",
			expectedToken:     "",
			expectedTargetEnv: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{
				Name: "test-workflow",
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							GitHubToken: tt.token,
						},
						TargetRepoSlug: tt.targetRepoSlug,
					},
				},
			}

			job, err := c.buildCreateOutputIssueJob(data, "main_job")
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			stepsContent := strings.Join(job.Steps, "")

			// Verify target repo slug in env vars if set
			if tt.targetRepoSlug != "" {
				if !strings.Contains(stepsContent, tt.expectedTargetEnv) {
					t.Errorf("Expected target repo env var %q not found", tt.expectedTargetEnv)
				}
			} else {
				// If no target repo, the env var should not be present
				if strings.Contains(stepsContent, "GH_AW_TARGET_REPO_SLUG:") {
					t.Error("Expected GH_AW_TARGET_REPO_SLUG to not be set")
				}
			}
		})
	}
}

// TestBuilderPreAndPostSteps verifies that pre-steps and post-steps are correctly applied
func TestBuilderPreAndPostSteps(t *testing.T) {
	c := NewCompiler(false, "", "test")

	// Test create_issue with assignees (post-steps)
	t.Run("create_issue with assignees", func(t *testing.T) {
		data := &WorkflowData{
			Name: "test-workflow",
			SafeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					Assignees: []string{"copilot", "user123"},
				},
			},
		}

		job, err := c.buildCreateOutputIssueJob(data, "main_job")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		stepsContent := strings.Join(job.Steps, "")
		if !strings.Contains(stepsContent, "Assign issue to copilot") {
			t.Error("Expected assignee step for copilot not found")
		}
		if !strings.Contains(stepsContent, "Assign issue to user123") {
			t.Error("Expected assignee step for user123 not found")
		}
	})

	// Test create_pull_request with pre-steps (checkout, git config)
	t.Run("create_pull_request with pre-steps", func(t *testing.T) {
		data := &WorkflowData{
			Name: "test-workflow",
			SafeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
		}

		job, err := c.buildCreateOutputPullRequestJob(data, "main_job")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		stepsContent := strings.Join(job.Steps, "")
		if !strings.Contains(stepsContent, "Download patch artifact") {
			t.Error("Expected 'Download patch artifact' pre-step not found")
		}
		if !strings.Contains(stepsContent, "actions/checkout") {
			t.Error("Expected 'actions/checkout' pre-step not found")
		}
	})

	// Test add_comment with pre-steps (debug output)
	t.Run("add_comment with debug pre-steps", func(t *testing.T) {
		data := &WorkflowData{
			Name: "test-workflow",
			SafeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{},
			},
		}

		job, err := c.buildCreateOutputAddCommentJob(data, "main_job", "", "", "")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		stepsContent := strings.Join(job.Steps, "")
		if !strings.Contains(stepsContent, "Debug agent outputs") {
			t.Error("Expected 'Debug agent outputs' pre-step not found")
		}
	})
}
