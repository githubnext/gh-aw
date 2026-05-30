//go:build !integration

package workflow

import (
	"reflect"
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
			// staged is independent of target-repo: staged flag is emitted even when target-repo is set
			name: "target-repo specified still adds staged flag",
			safeOutputs: &SafeOutputsConfig{
				Staged: true,
				CreateIssues: &CreateIssuesConfig{
					TargetRepoSlug: "org/repo",
					TitlePrefix:    "[Test] ",
				},
			},
			checkContains: []string{
				"GH_AW_SAFE_OUTPUTS_STAGED: \"true\"",
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

// TestStagedFlagWithTargetRepo tests that staged flag is emitted regardless of target-repo
func TestStagedFlagWithTargetRepo(t *testing.T) {
	tests := []struct {
		name          string
		targetRepo    string
		shouldAddFlag bool
	}{
		{
			name:          "no target-repo",
			targetRepo:    "",
			shouldAddFlag: true,
		},
		{
			// staged is independent of target-repo
			name:          "with target-repo",
			targetRepo:    "org/repo",
			shouldAddFlag: true,
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
						TargetRepoSlug: tt.targetRepo,
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
		name       string
		staged     bool
		trialMode  bool
		targetRepo string
		expectFlag bool
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
			// staged is independent of target-repo
			name:       "staged true, target-repo set",
			staged:     true,
			targetRepo: "org/repo",
			expectFlag: true,
		},
		{
			name:       "staged false",
			staged:     false,
			expectFlag: false,
		},
		{
			// trial mode suppresses staged regardless of target-repo
			name:       "staged true, trial mode and target-repo",
			staged:     true,
			trialMode:  true,
			targetRepo: "org/repo",
			expectFlag: false,
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
						TargetRepoSlug: tt.targetRepo,
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

	// staged is independent of target-repo: flag is emitted even with target-repo set
	assert.Contains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED")
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

	// staged is independent of target-repo: flag is emitted even with target-repo set
	assert.Contains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED")
}

// TestStagedFlagForAllHandlerTypes tests that the staged flag is emitted for every handler type
// registered in safeOutputFieldMapping. The test cases are generated via reflection so the test
// stays complete automatically when new handler types are added to safeOutputFieldMapping.
func TestStagedFlagForAllHandlerTypes(t *testing.T) {
	soType := reflect.TypeFor[SafeOutputsConfig]()

	// One sub-test per field registered in safeOutputFieldMapping.
	// Each sub-test sets staged:true + that one handler, and verifies the env var is emitted.
	for fieldName := range safeOutputFieldMapping {
		t.Run(fieldName, func(t *testing.T) {
			f, ok := soType.FieldByName(fieldName)
			require.True(t, ok, "safeOutputFieldMapping references field %q which does not exist in SafeOutputsConfig", fieldName)

			so := &SafeOutputsConfig{Staged: true}
			soVal := reflect.ValueOf(so).Elem()
			field := soVal.FieldByName(fieldName)
			require.True(t, field.IsValid(), "Field %q not found in SafeOutputsConfig", fieldName)
			require.Equal(t, reflect.Pointer, field.Kind(), "Expected pointer field for %q, got %v", fieldName, f.Type)

			// Set the field to a zero-valued instance of the target struct.
			field.Set(reflect.New(field.Type().Elem()))

			compiler := NewCompiler()
			workflowData := &WorkflowData{
				Name:        "Test Workflow",
				SafeOutputs: so,
			}

			var steps []string
			compiler.addAllSafeOutputConfigEnvVars(&steps, workflowData)
			stepsContent := strings.Join(steps, "")

			assert.Contains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED",
				"Expected staged flag to be emitted for handler %q", fieldName)
		})
	}

	// Verify the flag is not emitted when staged is set but no handler is configured.
	t.Run("no handlers configured", func(t *testing.T) {
		compiler := NewCompiler()
		workflowData := &WorkflowData{
			Name:        "Test Workflow",
			SafeOutputs: &SafeOutputsConfig{Staged: true},
		}

		var steps []string
		compiler.addAllSafeOutputConfigEnvVars(&steps, workflowData)
		stepsContent := strings.Join(steps, "")

		assert.NotContains(t, stepsContent, "GH_AW_SAFE_OUTPUTS_STAGED",
			"Staged flag should not be emitted when no handlers are configured")
	})
}
