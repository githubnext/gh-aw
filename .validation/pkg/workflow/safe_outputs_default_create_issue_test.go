//go:build !integration

package workflow

import (
	"fmt"
	"os"
	"path/filepath"
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
			name:                 "nil safe-outputs - inject create-issue",
			workflowID:           "my-workflow",
			safeOutputs:          nil,
			expectInjection:      true,
			expectedLabel:        "my-workflow",
			expectedTitlePrefix:  "[my-workflow]",
			expectedAutoInjected: true,
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

// TestAutoInjectCreateIssueWithImportedNonBuiltin verifies that when a workflow imports a shared
// file that already provides a non-builtin safe output (e.g. add-comment), create-issue is NOT
// auto-injected. The imported non-builtin is treated the same as an explicitly configured one.
func TestAutoInjectCreateIssueWithImportedNonBuiltin(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	// Shared workflow provides a non-builtin safe output (add-comment)
	sharedContent := `---
safe-outputs:
  add-comment:
    target: "triggering"
---
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "shared.md"), []byte(sharedContent), 0644))

	// Main workflow has NO safe-outputs: section — non-builtin comes entirely from import
	mainContent := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared.md
---

# Workflow relying on imported safe outputs
`
	mainFile := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mainFile, []byte(mainContent), 0644))

	workflowData, err := compiler.ParseWorkflowFile(mainFile)
	require.NoError(t, err, "ParseWorkflowFile should not error")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")

	// Non-builtin from import: add-comment should be present
	require.NotNil(t, workflowData.SafeOutputs.AddComments, "AddComments should be imported")

	// Auto-injection must NOT happen because import already provides a non-builtin
	assert.Nil(t, workflowData.SafeOutputs.CreateIssues,
		"create-issue should NOT be auto-injected when a non-builtin is already imported")
	assert.False(t, workflowData.SafeOutputs.AutoInjectedCreateIssue,
		"AutoInjectedCreateIssue should be false when non-builtin is imported")
}

// TestAutoInjectCreateIssueWithImportedCreateIssue verifies that when a workflow imports a shared
// file that explicitly configures create-issue, the imported config is preserved unchanged and
// auto-injection does NOT overwrite it.
func TestAutoInjectCreateIssueWithImportedCreateIssue(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	// Shared workflow provides create-issue with a specific title prefix and labels.
	// Note: max must be an integer (not a quoted string) to pass schema validation.
	sharedContent := `---
safe-outputs:
  create-issue:
    title-prefix: "[imported] "
    labels: [imported-label]
    max: 3
---
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "shared.md"), []byte(sharedContent), 0644))

	// Main workflow has NO safe-outputs section — create-issue comes from import
	mainContent := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared.md
---

# Workflow using imported create-issue
`
	mainFile := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mainFile, []byte(mainContent), 0644))

	workflowData, err := compiler.ParseWorkflowFile(mainFile)
	require.NoError(t, err, "ParseWorkflowFile should not error")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
	require.NotNil(t, workflowData.SafeOutputs.CreateIssues, "CreateIssues should be imported")

	// Imported config must be preserved — auto-injection must not overwrite it
	assert.Equal(t, "[imported] ", workflowData.SafeOutputs.CreateIssues.TitlePrefix,
		"Imported title-prefix must not be overwritten by auto-injection")
	assert.Equal(t, []string{"imported-label"}, workflowData.SafeOutputs.CreateIssues.Labels,
		"Imported labels must not be overwritten by auto-injection")
	assert.Equal(t, "3", *workflowData.SafeOutputs.CreateIssues.Max,
		"Imported max must not be overwritten by auto-injection")
	assert.False(t, workflowData.SafeOutputs.AutoInjectedCreateIssue,
		"AutoInjectedCreateIssue should be false when create-issue comes from import")
}

// TestAutoInjectCreateIssueWithImportedBuiltinsOnly verifies that when a workflow imports ONLY
// builtin safe outputs (noop, missing-tool, missing-data), create-issue is still auto-injected
// because builtins alone do not constitute a meaningful non-builtin output.
func TestAutoInjectCreateIssueWithImportedBuiltinsOnly(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	// Shared workflow provides ONLY builtin safe outputs (using explicit empty maps)
	sharedContent := `---
safe-outputs:
  noop:
    max: 1
  missing-tool: {}
  missing-data: {}
---
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "shared.md"), []byte(sharedContent), 0644))

	// Main workflow has NO safe-outputs section
	mainContent := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared.md
---

# Workflow importing only builtins
`
	mainFile := filepath.Join(workflowsDir, "main.md")
	require.NoError(t, os.WriteFile(mainFile, []byte(mainContent), 0644))

	workflowData, err := compiler.ParseWorkflowFile(mainFile)
	require.NoError(t, err, "ParseWorkflowFile should not error")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")

	// create-issue must be auto-injected since only builtins are imported
	require.NotNil(t, workflowData.SafeOutputs.CreateIssues,
		"create-issue should be auto-injected when only builtins are imported")
	assert.True(t, workflowData.SafeOutputs.AutoInjectedCreateIssue,
		"AutoInjectedCreateIssue should be true when auto-injected")
	assert.Equal(t, "main", workflowData.SafeOutputs.CreateIssues.Labels[0],
		"Label should be the workflow ID (basename without extension)")
}
