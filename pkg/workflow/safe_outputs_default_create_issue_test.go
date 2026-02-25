//go:build !integration

package workflow

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHasNonBuiltinSafeOutputsEnabled verifies that only non-builtin safe outputs are counted
func TestHasNonBuiltinSafeOutputsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *SafeOutputsConfig
		expected bool
	}{
		{
			name:     "nil config returns false",
			config:   nil,
			expected: false,
		},
		{
			name:     "empty config returns false",
			config:   &SafeOutputsConfig{},
			expected: false,
		},
		{
			name: "only noop returns false (builtin)",
			config: &SafeOutputsConfig{
				NoOp: &NoOpConfig{},
			},
			expected: false,
		},
		{
			name: "only missing-data returns false (builtin)",
			config: &SafeOutputsConfig{
				MissingData: &MissingDataConfig{},
			},
			expected: false,
		},
		{
			name: "only missing-tool returns false (builtin)",
			config: &SafeOutputsConfig{
				MissingTool: &MissingToolConfig{},
			},
			expected: false,
		},
		{
			name: "all builtins returns false",
			config: &SafeOutputsConfig{
				NoOp:        &NoOpConfig{},
				MissingData: &MissingDataConfig{},
				MissingTool: &MissingToolConfig{},
			},
			expected: false,
		},
		{
			name: "create-issue is non-builtin returns true",
			config: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
			expected: true,
		},
		{
			name: "add-comment is non-builtin returns true",
			config: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{},
			},
			expected: true,
		},
		{
			name: "create-pull-request is non-builtin returns true",
			config: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expected: true,
		},
		{
			name: "non-builtin alongside builtins returns true",
			config: &SafeOutputsConfig{
				NoOp:         &NoOpConfig{},
				MissingData:  &MissingDataConfig{},
				MissingTool:  &MissingToolConfig{},
				CreateIssues: &CreateIssuesConfig{},
			},
			expected: true,
		},
		{
			name: "custom safe-job returns true",
			config: &SafeOutputsConfig{
				Jobs: map[string]*SafeJobConfig{
					"my_custom_job": {},
				},
			},
			expected: true,
		},
		{
			name: "custom safe-job alongside builtins returns true",
			config: &SafeOutputsConfig{
				NoOp: &NoOpConfig{},
				Jobs: map[string]*SafeJobConfig{
					"my_custom_job": {},
				},
			},
			expected: true,
		},
		{
			name: "create-discussion is non-builtin returns true",
			config: &SafeOutputsConfig{
				CreateDiscussions: &CreateDiscussionsConfig{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasNonBuiltinSafeOutputsEnabled(tt.config)
			assert.Equal(t, tt.expected, result, "hasNonBuiltinSafeOutputsEnabled(%v)", tt.config)
		})
	}
}

// TestAutoInjectCreateIssue verifies that create-issues is auto-injected when no non-builtin
// safe outputs are configured, and uses the workflow ID for labels and title-prefix.
func TestAutoInjectCreateIssue(t *testing.T) {
	tests := []struct {
		name                 string
		workflowID           string
		safeOutputs          *SafeOutputsConfig
		expectInjection      bool
		expectedLabel        string
		expectedTitlePrefix  string
		expectedAutoInjected bool
	}{
		{
			name:            "nil safe-outputs - no injection",
			workflowID:      "my-workflow",
			safeOutputs:     nil,
			expectInjection: false,
		},
		{
			name:       "only builtins configured - inject create-issue",
			workflowID: "my-workflow",
			safeOutputs: &SafeOutputsConfig{
				NoOp:        &NoOpConfig{},
				MissingData: &MissingDataConfig{},
				MissingTool: &MissingToolConfig{},
			},
			expectInjection:      true,
			expectedLabel:        "my-workflow",
			expectedTitlePrefix:  "[my-workflow]",
			expectedAutoInjected: true,
		},
		{
			name:       "empty safe-outputs - inject create-issue",
			workflowID: "daily-report",
			safeOutputs: &SafeOutputsConfig{
				NoOp: &NoOpConfig{},
			},
			expectInjection:      true,
			expectedLabel:        "daily-report",
			expectedTitlePrefix:  "[daily-report]",
			expectedAutoInjected: true,
		},
		{
			name:       "create-issue already configured - no injection",
			workflowID: "my-workflow",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					TitlePrefix: "[existing]",
				},
			},
			expectInjection: false,
		},
		{
			name:       "add-comment configured - no injection",
			workflowID: "my-workflow",
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{},
			},
			expectInjection: false,
		},
		{
			name:       "create-pull-request configured - no injection",
			workflowID: "my-workflow",
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expectInjection: false,
		},
		{
			name:       "custom safe-job configured - no injection",
			workflowID: "my-workflow",
			safeOutputs: &SafeOutputsConfig{
				Jobs: map[string]*SafeJobConfig{
					"my_job": {},
				},
			},
			expectInjection: false,
		},
		{
			name:                 "empty safe-outputs config struct - inject create-issue",
			workflowID:           "status-checker",
			safeOutputs:          &SafeOutputsConfig{},
			expectInjection:      true,
			expectedLabel:        "status-checker",
			expectedTitlePrefix:  "[status-checker]",
			expectedAutoInjected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				WorkflowID:  tt.workflowID,
				SafeOutputs: tt.safeOutputs,
			}

			// Simulate the auto-injection logic
			applyDefaultCreateIssue(workflowData)

			if !tt.expectInjection {
				// If no injection expected, check the original state is preserved
				if tt.safeOutputs == nil {
					assert.Nil(t, workflowData.SafeOutputs, "SafeOutputs should remain nil")
				} else if tt.safeOutputs.CreateIssues != nil {
					// Original create-issues should be preserved unchanged
					assert.Equal(t, tt.safeOutputs.CreateIssues.TitlePrefix, workflowData.SafeOutputs.CreateIssues.TitlePrefix,
						"Existing create-issues config should be unchanged")
				} else {
					// No create-issues should be injected
					assert.Nil(t, workflowData.SafeOutputs.CreateIssues, "create-issues should not be injected")
				}
				return
			}

			// Injection expected
			require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil after injection")
			require.NotNil(t, workflowData.SafeOutputs.CreateIssues, "CreateIssues should be injected")

			assert.Equal(t, strPtr("1"), workflowData.SafeOutputs.CreateIssues.Max,
				"Injected create-issues should have max=1")
			assert.Equal(t, []string{tt.expectedLabel}, workflowData.SafeOutputs.CreateIssues.Labels,
				"Injected create-issues should have workflow ID as label")
			assert.Equal(t, tt.expectedTitlePrefix, workflowData.SafeOutputs.CreateIssues.TitlePrefix,
				"Injected create-issues should have [workflowID] as title prefix")
			assert.True(t, workflowData.SafeOutputs.AutoInjectedCreateIssue,
				"AutoInjectedCreateIssue should be true when injected")
		})
	}
}

// TestAutoInjectedCreateIssuePrompt verifies that the auto-injected create-issue produces
// a specific prompt instruction to create an issue with results or call noop.
func TestAutoInjectedCreateIssuePrompt(t *testing.T) {
	tests := []struct {
		name           string
		safeOutputs    *SafeOutputsConfig
		expectSpecific bool // expect the auto_create_issue file reference
	}{
		{
			name: "auto-injected create-issue produces specific prompt",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
					Labels:               []string{"my-workflow"},
					TitlePrefix:          "[my-workflow]",
				},
				AutoInjectedCreateIssue: true,
			},
			expectSpecific: true,
		},
		{
			name: "user-configured create-issue does NOT produce specific prompt",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{
					TitlePrefix: "[custom]",
				},
				AutoInjectedCreateIssue: false,
			},
			expectSpecific: false,
		},
		{
			name: "no create-issue configured",
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{},
			},
			expectSpecific: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &Compiler{}
			var yaml strings.Builder
			data := &WorkflowData{
				ParsedTools: NewTools(map[string]any{}),
				SafeOutputs: tt.safeOutputs,
			}
			compiler.generateUnifiedPromptStep(&yaml, data)
			output := yaml.String()

			if tt.expectSpecific {
				assert.Contains(t, output, safeOutputsAutoCreateIssueFile,
					"Auto-injected create-issue should include the auto_create_issue file reference")
			} else {
				assert.NotContains(t, output, safeOutputsAutoCreateIssueFile,
					"Non-auto-injected create-issue should not include the auto_create_issue file reference")
			}
		})
	}
}

// TestAutoInjectCreateIssueWithVariousWorkflowIDs verifies correct label/prefix generation
func TestAutoInjectCreateIssueWithVariousWorkflowIDs(t *testing.T) {
	workflowIDs := []string{
		"daily-status",
		"code-review",
		"security-scan",
		"my_workflow",
		"workflow123",
	}

	for _, wfID := range workflowIDs {
		t.Run("workflowID="+wfID, func(t *testing.T) {
			workflowData := &WorkflowData{
				WorkflowID: wfID,
				SafeOutputs: &SafeOutputsConfig{
					NoOp: &NoOpConfig{},
				},
			}

			applyDefaultCreateIssue(workflowData)

			require.NotNil(t, workflowData.SafeOutputs.CreateIssues, "create-issues should be injected")
			assert.Equal(t, []string{wfID}, workflowData.SafeOutputs.CreateIssues.Labels,
				"Label should be the workflow ID")
			assert.Equal(t, fmt.Sprintf("[%s]", wfID), workflowData.SafeOutputs.CreateIssues.TitlePrefix,
				"Title prefix should be [workflowID]")
		})
	}
}
