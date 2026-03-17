//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddAllSafeOutputConfigEnvVars tests environment variable generation for all safe output types
func TestAddAllSafeOutputConfigEnvVars(t *testing.T) {
	tests := []struct {
		name             string
		safeOutputs      *SafeOutputsConfig
		trialMode        bool
		checkContains    []string
		checkNotContains []string
	}{
		{
			name: "create issues with staged flag",
			safeOutputs: &SafeOutputsConfig{
				Staged: true,
				CreateIssues: &CreateIssuesConfig{
					TitlePrefix: "[Test] ",
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_STAGED: \"true\"",
			},
		},
		{
			name: "create issues without staged flag",
			safeOutputs: &SafeOutputsConfig{
				Staged: false,
				CreateIssues: &CreateIssuesConfig{
					TitlePrefix: "[Test] ",
				},
			},
			checkNotContains: []string{
				"GH_AW_SAFE_OUTPUTS_STAGED",
			},
		},
		{
			name: "add comments with staged flag",
			safeOutputs: &SafeOutputsConfig{
				Staged: true,
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("5"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_STAGED: \"true\"",
			},
		},
		{
			name: "add labels with staged flag",
			safeOutputs: &SafeOutputsConfig{
				Staged: true,
				AddLabels: &AddLabelsConfig{
					Allowed: []string{"bug"},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_STAGED: \"true\"",
			},
		},
		{
			name: "update issues with staged flag",
			safeOutputs: &SafeOutputsConfig{
				Staged:       true,
				UpdateIssues: &UpdateIssuesConfig{},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_STAGED: \"true\"",
			},
		},
		{
			name: "update discussions with staged flag",
			safeOutputs: &SafeOutputsConfig{
				Staged:            true,
				UpdateDiscussions: &UpdateDiscussionsConfig{},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_STAGED: \"true\"",
			},
		},
		{
			name: "create pull requests with staged flag",
			safeOutputs: &SafeOutputsConfig{
				Staged: true,
				CreatePullRequests: &CreatePullRequestsConfig{
					TitlePrefix: "[PR] ",
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_STAGED: \"true\"",
			},
		},
		{
			name: "multiple types only add staged flag once",
			safeOutputs: &SafeOutputsConfig{
				Staged: true,
				CreateIssues: &CreateIssuesConfig{
					TitlePrefix: "[Issue] ",
				},
				AddComments: &AddCommentsConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{
						Max: strPtr("3"),
					},
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_STAGED: \"true\"",
			},
		},
		{
			name:      "trial mode does not add staged flag",
			trialMode: true,
			safeOutputs: &SafeOutputsConfig{
				Staged: true,
				CreateIssues: &CreateIssuesConfig{
					TitlePrefix: "[Test] ",
				},
			},
			checkNotContains: []string{
				"GH_AW_SAFE_OUTPUTS_STAGED",
			},
		},
		{
			name: "target-repo specified does not add staged flag",
			safeOutputs: &SafeOutputsConfig{
				Staged: true,
				CreateIssues: &CreateIssuesConfig{
					TargetRepoSlug: "org/repo",
					TitlePrefix:    "[Test] ",
				},
			},
			checkNotContains: []string{
				"GH_AW_SAFE_OUTPUTS_STAGED",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			if tt.trialMode {
				compiler.SetTrialMode(true)
			}

			workflowData := &WorkflowData{
				Name:        "Test Workflow",
				SafeOutputs: tt.safeOutputs,
			}

			var steps []string
			compiler.addAllSafeOutputConfigEnvVars(&steps, workflowData)

			stepsContent := strings.Join(steps, "")

			for _, expected := range tt.checkContains {
				assert.Contains(t, stepsContent, expected, "Expected to find: "+expected)
			}

			for _, notExpected := range tt.checkNotContains {
				assert.NotContains(t, stepsContent, notExpected, "Should not contain: "+notExpected)
			}
		})
	}
}

// TestStagedFlagOnlyAddedOnce tests that staged flag is not duplicated
func TestStagedFlagOnlyAddedOnce(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			Staged: true,
			CreateIssues: &CreateIssuesConfig{
				TitlePrefix: "[Issue] ",
			},
			AddComments: &AddCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("3"),
				},
			},
			AddLabels: &AddLabelsConfig{
				Allowed: []string{"bug"},
			},
		},
	}

	var steps []string
	compiler.addAllSafeOutputConfigEnvVars(&steps, workflowData)

	stepsContent := strings.Join(steps, "")

	// Count occurrences of staged flag
	count := strings.Count(stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED")
	assert.Equal(t, 1, count, "Staged flag should appear exactly once")
}

// TestNoEnvVarsWhenNoSafeOutputs tests empty output when safe outputs is nil
func TestNoEnvVarsWhenNoSafeOutputs(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name:        "Test Workflow",
		SafeOutputs: nil,
	}

	var steps []string
	compiler.addAllSafeOutputConfigEnvVars(&steps, workflowData)

	// Should not add any steps
	assert.Empty(t, steps)
}

// TestStagedFlagWithTargetRepo tests staged flag behavior with target-repo
func TestStagedFlagWithTargetRepo(t *testing.T) {
	tests := []struct {
		name           string
		targetRepoSlug string
		shouldAddFlag  bool
	}{
		{
			name:           "no target-repo",
			targetRepoSlug: "",
			shouldAddFlag:  true,
		},
		{
			name:           "with target-repo",
			targetRepoSlug: "org/repo",
			shouldAddFlag:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()

			workflowData := &WorkflowData{
				Name: "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{
					Staged: true,
					CreateIssues: &CreateIssuesConfig{
						TargetRepoSlug: tt.targetRepoSlug,
					},
				},
			}

			var steps []string
			compiler.addAllSafeOutputConfigEnvVars(&steps, workflowData)

			stepsContent := strings.Join(steps, "")

			if tt.shouldAddFlag {
				assert.Contains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED")
			} else {
				assert.NotContains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED")
			}
		})
	}
}

// TestTrialModeOverridesStagedFlag tests that trial mode prevents staged flag
func TestTrialModeOverridesStagedFlag(t *testing.T) {
	compiler := NewCompiler()
	compiler.SetTrialMode(true)
	compiler.SetTrialLogicalRepoSlug("org/trial-repo")

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			Staged: true,
			CreateIssues: &CreateIssuesConfig{
				TitlePrefix: "[Test] ",
			},
		},
	}

	var steps []string
	compiler.addAllSafeOutputConfigEnvVars(&steps, workflowData)

	stepsContent := strings.Join(steps, "")

	// Trial mode should prevent staged flag from being added
	assert.NotContains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED")
}

// TestEnvVarsWithMultipleSafeOutputTypes tests comprehensive env var generation
func TestEnvVarsWithMultipleSafeOutputTypes(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			Staged: true,
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
			UpdateIssues:      &UpdateIssuesConfig{},
			UpdateDiscussions: &UpdateDiscussionsConfig{},
		},
	}

	var steps []string
	compiler.addAllSafeOutputConfigEnvVars(&steps, workflowData)

	require.NotEmpty(t, steps)

	stepsContent := strings.Join(steps, "")

	// Should contain staged flag exactly once
	assert.Contains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED")

	// Count occurrences
	count := strings.Count(stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED")
	assert.Equal(t, 1, count, "Staged flag should appear exactly once")
}

// TestEnvVarsWithNoStagedConfig tests that no staged flag is added when staged is false
func TestEnvVarsWithNoStagedConfig(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			Staged: false,
			CreateIssues: &CreateIssuesConfig{
				TitlePrefix: "[Test] ",
			},
			AddComments: &AddCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("5"),
				},
			},
		},
	}

	var steps []string
	compiler.addAllSafeOutputConfigEnvVars(&steps, workflowData)

	stepsContent := strings.Join(steps, "")

	// Should not contain staged flag
	assert.NotContains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED")
}

// TestEnvVarFormatting tests that environment variables are correctly formatted
func TestEnvVarFormatting(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			Staged: true,
			CreateIssues: &CreateIssuesConfig{
				TitlePrefix: "[Test] ",
			},
		},
	}

	var steps []string
	compiler.addAllSafeOutputConfigEnvVars(&steps, workflowData)

	require.NotEmpty(t, steps)

	// Check that env vars are properly indented and formatted
	for _, step := range steps {
		if strings.Contains(step, "GH_AW_SAFE_OUTPUTS_STAGED") {
			// Should have proper indentation (10 spaces for env vars in steps)
			assert.True(t, strings.HasPrefix(step, "          "), "Env var should be properly indented")
			// Should have proper format: KEY: "value"\n
			assert.True(t, strings.HasSuffix(step, "\n"), "Env var should end with newline")
			assert.Contains(t, step, ": ", "Env var should have key: value format")
		}
	}
}

// TestStagedFlagPrecedence tests staged flag behavior across different configurations
func TestStagedFlagPrecedence(t *testing.T) {
	tests := []struct {
		name           string
		staged         bool
		trialMode      bool
		targetRepoSlug string
		expectFlag     bool
	}{
		{
			name:       "staged true, no trial, no target-repo",
			staged:     true,
			trialMode:  false,
			expectFlag: true,
		},
		{
			name:       "staged true, trial mode",
			staged:     true,
			trialMode:  true,
			expectFlag: false,
		},
		{
			name:           "staged true, target-repo set",
			staged:         true,
			targetRepoSlug: "org/repo",
			expectFlag:     false,
		},
		{
			name:       "staged false",
			staged:     false,
			expectFlag: false,
		},
		{
			name:           "staged true, trial mode and target-repo",
			staged:         true,
			trialMode:      true,
			targetRepoSlug: "org/repo",
			expectFlag:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			if tt.trialMode {
				compiler.SetTrialMode(true)
			}

			workflowData := &WorkflowData{
				Name: "Test Workflow",
				SafeOutputs: &SafeOutputsConfig{
					Staged: tt.staged,
					CreateIssues: &CreateIssuesConfig{
						TargetRepoSlug: tt.targetRepoSlug,
					},
				},
			}

			var steps []string
			compiler.addAllSafeOutputConfigEnvVars(&steps, workflowData)

			stepsContent := strings.Join(steps, "")

			if tt.expectFlag {
				assert.Contains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED", "Expected staged flag to be present")
			} else {
				assert.NotContains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED", "Expected staged flag to be absent")
			}
		})
	}
}

// TestAddCommentsTargetRepoStagedBehavior tests staged flag behavior for add_comments with target-repo
func TestAddCommentsTargetRepoStagedBehavior(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			Staged: true,
			AddComments: &AddCommentsConfig{
				TargetRepoSlug: "org/target",
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("5"),
				},
			},
		},
	}

	var steps []string
	compiler.addAllSafeOutputConfigEnvVars(&steps, workflowData)

	stepsContent := strings.Join(steps, "")

	// Should not add staged flag when target-repo is set
	assert.NotContains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED")
}

// TestAddLabelsTargetRepoStagedBehavior tests staged flag behavior for add_labels with target-repo
func TestAddLabelsTargetRepoStagedBehavior(t *testing.T) {
	compiler := NewCompiler()

	workflowData := &WorkflowData{
		Name: "Test Workflow",
		SafeOutputs: &SafeOutputsConfig{
			Staged: true,
			AddLabels: &AddLabelsConfig{
				Allowed: []string{"bug"},
				SafeOutputTargetConfig: SafeOutputTargetConfig{
					TargetRepoSlug: "org/target",
				},
			},
		},
	}

	var steps []string
	compiler.addAllSafeOutputConfigEnvVars(&steps, workflowData)

	stepsContent := strings.Join(steps, "")

	// Should not add staged flag when target-repo is set
	assert.NotContains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED")
}

// TestStagedFlagForAllHandlerTypes tests that the staged flag is emitted for all safe output handler types.
func TestStagedFlagForAllHandlerTypes(t *testing.T) {
	tests := []struct {
		name        string
		safeOutputs *SafeOutputsConfig
		expectFlag  bool
	}{
		// Handlers without a target-repo field always qualify
		{
			name: "autofix-code-scanning-alert with staged",
			safeOutputs: &SafeOutputsConfig{
				Staged:                   true,
				AutofixCodeScanningAlert: &AutofixCodeScanningAlertConfig{},
			},
			expectFlag: true,
		},
		{
			name: "upload-asset with staged",
			safeOutputs: &SafeOutputsConfig{
				Staged:       true,
				UploadAssets: &UploadAssetsConfig{},
			},
			expectFlag: true,
		},
		{
			name: "update-project with staged",
			safeOutputs: &SafeOutputsConfig{
				Staged:         true,
				UpdateProjects: &UpdateProjectConfig{},
			},
			expectFlag: true,
		},
		{
			name: "create-project with staged",
			safeOutputs: &SafeOutputsConfig{
				Staged:         true,
				CreateProjects: &CreateProjectsConfig{},
			},
			expectFlag: true,
		},
		{
			name: "create-project-status-update with staged",
			safeOutputs: &SafeOutputsConfig{
				Staged:                     true,
				CreateProjectStatusUpdates: &CreateProjectStatusUpdateConfig{},
			},
			expectFlag: true,
		},
		{
			name: "call-workflow with staged",
			safeOutputs: &SafeOutputsConfig{
				Staged:       true,
				CallWorkflow: &CallWorkflowConfig{},
			},
			expectFlag: true,
		},
		{
			name: "noop with staged",
			safeOutputs: &SafeOutputsConfig{
				Staged: true,
				NoOp:   &NoOpConfig{},
			},
			expectFlag: true,
		},
		{
			name: "missing-tool with staged",
			safeOutputs: &SafeOutputsConfig{
				Staged:      true,
				MissingTool: &MissingToolConfig{},
			},
			expectFlag: true,
		},
		{
			name: "missing-data with staged",
			safeOutputs: &SafeOutputsConfig{
				Staged:      true,
				MissingData: &MissingDataConfig{},
			},
			expectFlag: true,
		},
		// Handlers with target-repo: qualify when no target-repo is set
		{
			name: "create-discussion without target-repo",
			safeOutputs: &SafeOutputsConfig{
				Staged:            true,
				CreateDiscussions: &CreateDiscussionsConfig{},
			},
			expectFlag: true,
		},
		{
			name: "create-discussion with target-repo",
			safeOutputs: &SafeOutputsConfig{
				Staged: true,
				CreateDiscussions: &CreateDiscussionsConfig{
					TargetRepoSlug: "org/other",
				},
			},
			expectFlag: false,
		},
		{
			name: "close-issue without target-repo",
			safeOutputs: &SafeOutputsConfig{
				Staged:      true,
				CloseIssues: &CloseIssuesConfig{},
			},
			expectFlag: true,
		},
		{
			name: "close-pull-request without target-repo",
			safeOutputs: &SafeOutputsConfig{
				Staged:            true,
				ClosePullRequests: &ClosePullRequestsConfig{},
			},
			expectFlag: true,
		},
		{
			name: "mark-pr-ready-for-review without target-repo",
			safeOutputs: &SafeOutputsConfig{
				Staged:                          true,
				MarkPullRequestAsReadyForReview: &MarkPullRequestAsReadyForReviewConfig{},
			},
			expectFlag: true,
		},
		{
			name: "create-pull-request-review-comment without target-repo",
			safeOutputs: &SafeOutputsConfig{
				Staged:                          true,
				CreatePullRequestReviewComments: &CreatePullRequestReviewCommentsConfig{},
			},
			expectFlag: true,
		},
		{
			name: "submit-pull-request-review without target-repo",
			safeOutputs: &SafeOutputsConfig{
				Staged:                  true,
				SubmitPullRequestReview: &SubmitPullRequestReviewConfig{},
			},
			expectFlag: true,
		},
		{
			name: "dispatch-workflow without target-repo",
			safeOutputs: &SafeOutputsConfig{
				Staged:           true,
				DispatchWorkflow: &DispatchWorkflowConfig{},
			},
			expectFlag: true,
		},
		{
			name: "dispatch-workflow with target-repo",
			safeOutputs: &SafeOutputsConfig{
				Staged: true,
				DispatchWorkflow: &DispatchWorkflowConfig{
					TargetRepoSlug: "org/other",
				},
			},
			expectFlag: false,
		},
		// Only cross-repo handlers configured: no staged flag
		{
			name: "only cross-repo handlers configured",
			safeOutputs: &SafeOutputsConfig{
				Staged: true,
				CreateIssues: &CreateIssuesConfig{
					TargetRepoSlug: "org/repo",
				},
				CreateDiscussions: &CreateDiscussionsConfig{
					TargetRepoSlug: "org/other",
				},
			},
			expectFlag: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			workflowData := &WorkflowData{
				Name:        "Test Workflow",
				SafeOutputs: tt.safeOutputs,
			}

			var steps []string
			compiler.addAllSafeOutputConfigEnvVars(&steps, workflowData)
			stepsContent := strings.Join(steps, "")

			if tt.expectFlag {
				assert.Contains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED", "Expected staged flag to be present")
			} else {
				assert.NotContains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED", "Expected staged flag to be absent")
			}
		})
	}
}
