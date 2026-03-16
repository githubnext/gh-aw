//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildConsolidatedSafeOutputsJob tests the main job builder function
func TestBuildConsolidatedSafeOutputsJob(t *testing.T) {
	tests := []struct {
		name             string
		safeOutputs      *SafeOutputsConfig
		threatDetection  bool
		expectedJobName  string
		expectedSteps    int
		expectNil        bool
		checkPermissions bool
		expectedPerms    []string
	}{
		{
			name:          "no safe outputs configured",
			safeOutputs:   nil,
			expectNil:     true,
			expectedSteps: 0,
		},
		{
			name: "create issues only",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					TitlePrefix: "[Test] ",
					Labels:      []string{"test"},
				},
			},
			expectedJobName:  "safe_outputs",
			checkPermissions: true,
			expectedPerms:    []string{"contents: read", "issues: write"},
		},
		{
			name: "add comments only",
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
			},
			expectedJobName:  "safe_outputs",
			checkPermissions: true,
			expectedPerms:    []string{"contents: read", "issues: write", "discussions: write"},
		},
		{
			name: "create pull requests with patch",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					TitlePrefix: "[Test] ",
					Labels:      []string{"test"},
				},
			},
			expectedJobName:  "safe_outputs",
			checkPermissions: true,
			expectedPerms:    []string{"contents: write", "issues: write", "pull-requests: write"},
		},
		{
			name: "multiple safe output types",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					TitlePrefix: "[Issue] ",
				},
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("3"),
					},
				},
				AddLabels: &AddLabelsConfig{
					Allowed: []string{"bug", "enhancement"},
				},
			},
			expectedJobName:  "safe_outputs",
			checkPermissions: true,
			expectedPerms:    []string{"contents: read", "issues: write", "discussions: write"},
		},
		{
			name: "with threat detection enabled",
			safeOutputs: &SafeOutputsConfig{
				ThreatDetection: &ThreatDetectionConfig{},
				CreateIssues: &CreateIssuesConfig{
					TitlePrefix: "[Test] ",
				},
			},
			threatDetection:  true,
			expectedJobName:  "safe_outputs",
			checkPermissions: false,
		},
		{
			name: "with GitHub App token",
			safeOutputs: &SafeOutputsConfig{
				GitHubApp: &GitHubAppConfig{
					AppID:      "12345",
					PrivateKey: "test-key",
				},
				CreateIssues: &CreateIssuesConfig{
					TitlePrefix: "[Test] ",
				},
			},
			expectedJobName:  "safe_outputs",
			checkPermissions: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.jobManager = NewJobManager()

			workflowData := &WorkflowData{
				Name:        "Test Workflow",
				SafeOutputs: tt.safeOutputs,
			}

			job, stepNames, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, string(constants.AgentJobName), "test-workflow.md")

			if tt.expectNil {
				assert.Nil(t, job)
				assert.Nil(t, stepNames)
				assert.NoError(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, job)
			assert.Equal(t, tt.expectedJobName, job.Name)
			assert.NotEmpty(t, job.Steps)
			assert.NotEmpty(t, job.Env)

			// Check job dependencies — detection is inline in agent, no separate dependency
			assert.Contains(t, job.Needs, string(constants.AgentJobName))
			if tt.threatDetection {
				assert.NotContains(t, job.Needs, string(constants.DetectionJobName), "safe_outputs should not depend on detection (detection is inline in agent)")
			}

			// Check permissions if specified
			if tt.checkPermissions {
				jobYAML := job.Permissions
				for _, perm := range tt.expectedPerms {
					assert.Contains(t, jobYAML, perm, "Expected permission: "+perm)
				}
			}

			// Verify timeout is set
			assert.Equal(t, 15, job.TimeoutMinutes)

			// Verify job condition is set
			assert.NotEmpty(t, job.If)
		})
	}
}

// TestBuildConsolidatedSafeOutputsJobConcurrencyGroup tests that the concurrency-group field
// is correctly applied to the safe_outputs job
func TestBuildConsolidatedSafeOutputsJobConcurrencyGroup(t *testing.T) {
	tests := []struct {
		name              string
		concurrencyGroup  string
		expectConcurrency bool
	}{
		{
			name:              "no concurrency group",
			concurrencyGroup:  "",
			expectConcurrency: false,
		},
		{
			name:              "simple concurrency group",
			concurrencyGroup:  "my-safe-outputs",
			expectConcurrency: true,
		},
		{
			name:              "concurrency group with expression",
			concurrencyGroup:  "safe-outputs-${{ github.repository }}",
			expectConcurrency: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.jobManager = NewJobManager()

			workflowData := &WorkflowData{
				Name: "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues:     &CreateIssuesConfig{TitlePrefix: "[Test] "},
					ConcurrencyGroup: tt.concurrencyGroup,
				},
			}

			job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, string(constants.AgentJobName), "test-workflow.md")
			require.NoError(t, err, "Should build job without error")
			require.NotNil(t, job, "Job should not be nil")

			if tt.expectConcurrency {
				assert.NotEmpty(t, job.Concurrency, "Job should have concurrency set")
				assert.Contains(t, job.Concurrency, tt.concurrencyGroup, "Concurrency should contain the group value")
				assert.Contains(t, job.Concurrency, "cancel-in-progress: false", "Concurrency should have cancel-in-progress: false")
			} else {
				assert.Empty(t, job.Concurrency, "Job should have no concurrency set")
			}
		})
	}
}

func TestBuildJobLevelSafeOutputEnvVars(t *testing.T) {
	tests := []struct {
		name          string
		workflowData  *WorkflowData
		workflowID    string
		trialMode     bool
		trialRepo     string
		expectedVars  map[string]string
		checkContains bool
	}{
		{
			name: "basic env vars",
			workflowData: &WorkflowData{
				Name:        "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{},
			},
			workflowID: "test-workflow",
			expectedVars: map[string]string{
				"GH_AW_WORKFLOW_ID":        `"test-workflow"`,
				"GH_AW_WORKFLOW_NAME":      `"Test Workflow"`,
				"GH_AW_CALLER_WORKFLOW_ID": `"${{ github.repository }}/test-workflow"`,
			},
			checkContains: true,
		},
		{
			name: "with source metadata",
			workflowData: &WorkflowData{
				Name:        "Test Workflow",
				Source:      "user/repo",
				SafeOutputs: &SafeOutputsConfig{},
			},
			workflowID: "test-workflow",
			expectedVars: map[string]string{
				"GH_AW_WORKFLOW_SOURCE": `"user/repo"`,
			},
			checkContains: true,
		},
		{
			name: "with tracker ID",
			workflowData: &WorkflowData{
				Name:        "Test Workflow",
				TrackerID:   "tracker-123",
				SafeOutputs: &SafeOutputsConfig{},
			},
			workflowID: "test-workflow",
			expectedVars: map[string]string{
				"GH_AW_TRACKER_ID": `"tracker-123"`,
			},
			checkContains: true,
		},
		{
			name: "with engine config",
			workflowData: &WorkflowData{
				Name: "Test Workflow",
				EngineConfig: &EngineConfig{
					ID:      "copilot",
					Version: "0.0.375",
					Model:   "gpt-4",
				},
				SafeOutputs: &SafeOutputsConfig{},
			},
			workflowID: "test-workflow",
			expectedVars: map[string]string{
				"GH_AW_ENGINE_ID":      `"copilot"`,
				"GH_AW_ENGINE_VERSION": `"0.0.375"`,
				"GH_AW_ENGINE_MODEL":   `"gpt-4"`,
			},
			checkContains: true,
		},
		{
			name: "staged mode",
			workflowData: &WorkflowData{
				Name: "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{
					Staged: true,
				},
			},
			workflowID: "test-workflow",
			expectedVars: map[string]string{
				"GH_AW_SAFE_OUTPUTS_STAGED": `"true"`,
			},
			checkContains: true,
		},
		{
			name: "trial mode with target repo",
			workflowData: &WorkflowData{
				Name:        "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{},
			},
			workflowID: "test-workflow",
			trialMode:  true,
			trialRepo:  "org/test-repo",
			expectedVars: map[string]string{
				"GH_AW_TARGET_REPO_SLUG": `"org/test-repo"`,
			},
			checkContains: true,
		},
		{
			name: "with messages config",
			workflowData: &WorkflowData{
				Name: "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{
					Messages: &SafeOutputMessagesConfig{
						Footer: "Custom footer",
					},
				},
			},
			workflowID:    "test-workflow",
			checkContains: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			if tt.trialMode {
				compiler.SetTrialMode(true)
			}
			if tt.trialRepo != "" {
				compiler.SetTrialLogicalRepoSlug(tt.trialRepo)
			}

			envVars := compiler.buildJobLevelSafeOutputEnvVars(tt.workflowData, tt.workflowID)

			require.NotNil(t, envVars)

			if tt.checkContains {
				for key, expectedValue := range tt.expectedVars {
					actualValue, exists := envVars[key]
					assert.True(t, exists, "Expected env var %s to exist", key)
					if exists {
						assert.Equal(t, expectedValue, actualValue, "Env var %s has incorrect value", key)
					}
				}
			}
		})
	}
}

// TestBuildDetectionSuccessCondition tests the detection condition builder
func TestBuildDetectionSuccessCondition(t *testing.T) {
	condition := buildDetectionSuccessCondition()

	require.NotNil(t, condition)

	rendered := condition.Render()

	// Should check agent job's detection_success output
	assert.Contains(t, rendered, "needs."+string(constants.AgentJobName))
	assert.Contains(t, rendered, "outputs.detection_success")
	assert.Contains(t, rendered, "'true'")
}

// TestJobConditionWithThreatDetection tests job condition building with threat detection
func TestJobConditionWithThreatDetection(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
			CreateIssues: &CreateIssuesConfig{
				TitlePrefix: "[Test] ",
			},
		},
	}

	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, string(constants.AgentJobName), "test.md")

	require.NoError(t, err)
	require.NotNil(t, job)

	// Job condition should include detection check referencing agent job
	assert.Contains(t, job.If, "needs."+string(constants.AgentJobName))
	assert.Contains(t, job.If, "outputs.detection_success")

	// Job should NOT depend on detection job (detection is inline in agent)
	assert.NotContains(t, job.Needs, string(constants.DetectionJobName), "safe_outputs job should not depend on detection job")
}

// TestJobWithGitHubApp tests job building with GitHub App configuration
func TestJobWithGitHubApp(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			GitHubApp: &GitHubAppConfig{
				AppID:      "12345",
				PrivateKey: "test-key",
			},
			CreateIssues: &CreateIssuesConfig{
				TitlePrefix: "[Test] ",
			},
		},
	}

	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, string(constants.AgentJobName), "test.md")

	require.NoError(t, err)
	require.NotNil(t, job)

	stepsContent := strings.Join(job.Steps, "")

	// Should include app token minting step
	assert.Contains(t, stepsContent, "Generate GitHub App token")

	// Should include app token invalidation step
	assert.Contains(t, stepsContent, "Invalidate GitHub App token")
}

// TestAssignToAgentWithGitHubAppUsesAgentToken tests that when github-app: is configured,
// assign-to-agent uses GH_AW_AGENT_TOKEN rather than the App installation token.
// The Copilot assignment API only accepts PATs, not GitHub App tokens.
func TestAssignToAgentWithGitHubAppUsesAgentToken(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			GitHubApp: &GitHubAppConfig{
				AppID:      "12345",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			},
			AssignToAgent: &AssignToAgentConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
			},
		},
	}

	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, string(constants.AgentJobName), "test.md")

	require.NoError(t, err)
	require.NotNil(t, job)

	stepsContent := strings.Join(job.Steps, "")

	// App token minting step should be present (github-app: is configured)
	assert.Contains(t, stepsContent, "Generate GitHub App token", "App token minting step should be present")

	// Find the assign_to_agent step section
	assignToAgentStart := strings.Index(stepsContent, "id: assign_to_agent")
	require.Greater(t, assignToAgentStart, -1, "assign_to_agent step should exist")

	// Find the end of the assign_to_agent step (next step starts with "      - ")
	nextStepOffset := strings.Index(stepsContent[assignToAgentStart:], "\n      - ")
	var assignToAgentSection string
	if nextStepOffset == -1 {
		assignToAgentSection = stepsContent[assignToAgentStart:]
	} else {
		assignToAgentSection = stepsContent[assignToAgentStart : assignToAgentStart+nextStepOffset]
	}

	// The assign_to_agent step should use GH_AW_AGENT_TOKEN, NOT the App token
	assert.Contains(t, assignToAgentSection, "GH_AW_AGENT_TOKEN",
		"assign_to_agent step should use GH_AW_AGENT_TOKEN, not the App token")
	assert.NotContains(t, assignToAgentSection, "safe-outputs-app-token.outputs.token",
		"assign_to_agent step should not use the GitHub App token")
}

// TestAssignToAgentWithGitHubAppAndExplicitToken tests that an explicit github-token
// on assign-to-agent takes precedence over both the App token and GH_AW_AGENT_TOKEN.
func TestAssignToAgentWithGitHubAppAndExplicitToken(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			GitHubApp: &GitHubAppConfig{
				AppID:      "12345",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			},
			AssignToAgent: &AssignToAgentConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max:         strPtr("1"),
					GitHubToken: "${{ secrets.MY_CUSTOM_TOKEN }}",
				},
			},
		},
	}

	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, string(constants.AgentJobName), "test.md")

	require.NoError(t, err)
	require.NotNil(t, job)

	stepsContent := strings.Join(job.Steps, "")

	// Find the assign_to_agent step section
	assignToAgentStart := strings.Index(stepsContent, "id: assign_to_agent")
	require.Greater(t, assignToAgentStart, -1, "assign_to_agent step should exist")

	nextStepOffset := strings.Index(stepsContent[assignToAgentStart:], "\n      - ")
	var assignToAgentSection string
	if nextStepOffset == -1 {
		assignToAgentSection = stepsContent[assignToAgentStart:]
	} else {
		assignToAgentSection = stepsContent[assignToAgentStart : assignToAgentStart+nextStepOffset]
	}

	// The explicit token should take precedence
	assert.Contains(t, assignToAgentSection, "secrets.MY_CUSTOM_TOKEN",
		"assign_to_agent step should use the explicitly configured github-token")
	assert.NotContains(t, assignToAgentSection, "safe-outputs-app-token.outputs.token",
		"assign_to_agent step should not use the GitHub App token even with explicit token")
	assert.NotContains(t, assignToAgentSection, "GH_AW_AGENT_TOKEN",
		"assign_to_agent step should not use GH_AW_AGENT_TOKEN when explicit token is set")
}

// TestJobOutputs tests that job outputs are correctly configured
func TestJobOutputs(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{
				TitlePrefix: "[Test] ",
			},
		},
	}

	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, string(constants.AgentJobName), "test.md")

	require.NoError(t, err)
	require.NotNil(t, job)

	// Handler manager outputs
	assert.Contains(t, job.Outputs, "process_safe_outputs_temporary_id_map")
	assert.Contains(t, job.Outputs, "process_safe_outputs_processed_count")

	// Check output format
	assert.Contains(t, job.Outputs["process_safe_outputs_temporary_id_map"], "steps.process_safe_outputs.outputs")
}

// TestJobDependencies tests that job dependencies are correctly set
func TestJobDependencies(t *testing.T) {
	tests := []struct {
		name             string
		safeOutputs      *SafeOutputsConfig
		expectedNeeds    []string
		notExpectedNeeds []string
	}{
		{
			name: "basic safe outputs",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
			expectedNeeds:    []string{string(constants.AgentJobName)},
			notExpectedNeeds: []string{string(constants.DetectionJobName), string(constants.ActivationJobName)},
		},
		{
			name: "with threat detection",
			safeOutputs: &SafeOutputsConfig{
				ThreatDetection: &ThreatDetectionConfig{},
				CreateIssues:    &CreateIssuesConfig{},
			},
			expectedNeeds:    []string{string(constants.AgentJobName)},
			notExpectedNeeds: []string{string(constants.DetectionJobName)}, // detection is inline in agent
		},
		{
			name: "with create pull request",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expectedNeeds: []string{string(constants.AgentJobName), string(constants.ActivationJobName)},
		},
		{
			name: "with push to PR branch",
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{},
			},
			expectedNeeds: []string{string(constants.AgentJobName), string(constants.ActivationJobName)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.jobManager = NewJobManager()

			workflowData := &WorkflowData{
				Name:        "Test Workflow",
				SafeOutputs: tt.safeOutputs,
			}

			job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, string(constants.AgentJobName), "test.md")

			require.NoError(t, err)
			require.NotNil(t, job)

			for _, need := range tt.expectedNeeds {
				assert.Contains(t, job.Needs, need)
			}

			for _, notNeed := range tt.notExpectedNeeds {
				assert.NotContains(t, job.Needs, notNeed)
			}
		})
	}
}

// TestGitHubAppWithPushToPRBranch tests that GitHub App token step is not duplicated
// when both app and push-to-pull-request-branch are configured
// Regression test for duplicate step bug reported in issue
func TestGitHubAppWithPushToPRBranch(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			GitHubApp: &GitHubAppConfig{
				AppID:      "${{ vars.ACTIONS_APP_ID }}",
				PrivateKey: "${{ secrets.ACTIONS_PRIVATE_KEY }}",
			},
			PushToPullRequestBranch: &PushToPullRequestBranchConfig{},
		},
	}

	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, string(constants.AgentJobName), "test.md")

	require.NoError(t, err, "Should successfully build job")
	require.NotNil(t, job, "Job should not be nil")

	stepsContent := strings.Join(job.Steps, "")

	// Should include app token minting step exactly once
	tokenMintCount := strings.Count(stepsContent, "Generate GitHub App token")
	assert.Equal(t, 1, tokenMintCount, "App token minting step should appear exactly once, found %d times", tokenMintCount)

	// Should include app token invalidation step exactly once
	tokenInvalidateCount := strings.Count(stepsContent, "Invalidate GitHub App token")
	assert.Equal(t, 1, tokenInvalidateCount, "App token invalidation step should appear exactly once, found %d times", tokenInvalidateCount)

	// Token step should come before checkout step (checkout references the token)
	tokenIndex := strings.Index(stepsContent, "Generate GitHub App token")
	checkoutIndex := strings.Index(stepsContent, "Checkout repository")
	assert.Less(t, tokenIndex, checkoutIndex, "Token minting step should come before checkout step")

	// Verify step ID is set correctly
	assert.Contains(t, stepsContent, "id: safe-outputs-app-token")
}

// TestJobWithGitHubAppWorkflowCallUsesTargetRepoNameFallback is a regression test verifying that
// a safe-output job compiled for a workflow_call trigger uses
// needs.activation.outputs.target_repo_name (repo name only, no owner prefix) as the repositories
// fallback for the GitHub App token mint step, instead of the full target_repo slug.
// This prevents actions/create-github-app-token from receiving an invalid owner/repo slug
// in the repositories field when owner is also set.
func TestJobWithGitHubAppWorkflowCallUsesTargetRepoNameFallback(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		On: `"on":
  workflow_call:`,
		SafeOutputs: &SafeOutputsConfig{
			GitHubApp: &GitHubAppConfig{
				AppID:      "${{ vars.APP_ID }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			},
			CreateIssues: &CreateIssuesConfig{
				TitlePrefix: "[Test] ",
			},
		},
	}

	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, string(constants.AgentJobName), "test.md")

	require.NoError(t, err, "Should successfully build job")
	require.NotNil(t, job, "Job should not be nil")

	stepsContent := strings.Join(job.Steps, "")

	// Must use the repo-name-only output, NOT the full slug
	assert.Contains(t, stepsContent, "repositories: ${{ needs.activation.outputs.target_repo_name }}",
		"GitHub App token step must use target_repo_name (repo name only) for workflow_call workflows")
	assert.NotContains(t, stepsContent, "repositories: ${{ needs.activation.outputs.target_repo }}",
		"GitHub App token step must not use target_repo (full slug) for workflow_call workflows")
}

// TestConclusionJobWithGitHubAppWorkflowCallUsesTargetRepoNameFallback is a regression test
// verifying that the conclusion job compiled for a workflow_call trigger uses
// needs.activation.outputs.target_repo_name (repo name only) as the repositories fallback
// for the GitHub App token mint step.
func TestConclusionJobWithGitHubAppWorkflowCallUsesTargetRepoNameFallback(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()
	compiler.SetActionMode(ActionModeDev)

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		On: `"on":
  workflow_call:`,
		SafeOutputs: &SafeOutputsConfig{
			GitHubApp: &GitHubAppConfig{
				AppID:      "${{ vars.APP_ID }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			},
			AddComments: &AddCommentsConfig{},
		},
	}

	job, err := compiler.buildConclusionJob(workflowData, string(constants.AgentJobName), nil)

	require.NoError(t, err, "Should successfully build conclusion job")
	require.NotNil(t, job, "Conclusion job should not be nil")

	stepsContent := strings.Join(job.Steps, "")

	// Must use the repo-name-only output, NOT the full slug
	assert.Contains(t, stepsContent, "repositories: ${{ needs.activation.outputs.target_repo_name }}",
		"Conclusion job GitHub App token step must use target_repo_name (repo name only) for workflow_call workflows")
	assert.NotContains(t, stepsContent, "repositories: ${{ needs.activation.outputs.target_repo }}",
		"Conclusion job GitHub App token step must not use target_repo (full slug) for workflow_call workflows")
}

// TestCallWorkflowOnly_UsesHandlerManagerStep asserts that a workflow configured with only
// call-workflow (no other handler-manager types) still compiles a "Process Safe Outputs" step.
func TestCallWorkflowOnly_UsesHandlerManagerStep(t *testing.T) {
	compiler := NewCompiler()
	compiler.jobManager = NewJobManager()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
				Workflows: []string{"worker-a"},
			},
		},
	}

	job, stepNames, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, string(constants.AgentJobName), "test-workflow.md")
	require.NoError(t, err, "Should compile without error")
	require.NotNil(t, job, "safe_outputs job should be generated when only call-workflow is configured")
	require.NotNil(t, stepNames, "Step names should not be nil")

	stepsContent := strings.Join(job.Steps, "")
	assert.Contains(t, stepsContent, "Process Safe Outputs", "Compiled job should include 'Process Safe Outputs' step")
	assert.Contains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG", "Compiled job should include handler config env var")
	assert.Contains(t, stepsContent, "call_workflow", "Handler config should reference call_workflow")
}
