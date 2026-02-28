//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectWorkflowCallOutputs(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name            string
		onSection       string
		safeOutputs     *SafeOutputsConfig
		expectContains  []string
		expectAbsent    []string
		expectUnchanged bool
	}{
		{
			name: "no workflow_call - no injection",
			onSection: `"on":
  push:
  workflow_dispatch:`,
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
			expectUnchanged: true,
		},
		{
			name: "nil safe outputs - no injection",
			onSection: `"on":
  workflow_call:`,
			safeOutputs:     nil,
			expectUnchanged: true,
		},
		{
			name: "empty SafeOutputsConfig - no injection",
			onSection: `"on":
  workflow_call:`,
			safeOutputs:     &SafeOutputsConfig{},
			expectUnchanged: true,
		},
		{
			name: "workflow_call with create-issue outputs",
			onSection: `"on":
  workflow_call:
  workflow_dispatch:`,
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
			expectContains: []string{
				"workflow_call:",
				"outputs:",
				"created_issue_number:",
				"created_issue_url:",
				"jobs.safe_outputs.outputs.created_issue_number",
				"jobs.safe_outputs.outputs.created_issue_url",
			},
		},
		{
			name: "workflow_call with create-pull-request outputs",
			onSection: `"on":
  workflow_call:`,
			safeOutputs: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{},
			},
			expectContains: []string{
				"created_pr_number:",
				"created_pr_url:",
				"jobs.safe_outputs.outputs.created_pr_number",
				"jobs.safe_outputs.outputs.created_pr_url",
			},
			expectAbsent: []string{
				"created_issue_number:",
			},
		},
		{
			name: "workflow_call with add-comment outputs",
			onSection: `"on":
  workflow_call:`,
			safeOutputs: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{},
			},
			expectContains: []string{
				"comment_id:",
				"comment_url:",
				"jobs.safe_outputs.outputs.comment_id",
				"jobs.safe_outputs.outputs.comment_url",
			},
		},
		{
			name: "workflow_call with push-to-pull-request-branch outputs",
			onSection: `"on":
  workflow_call:`,
			safeOutputs: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{},
			},
			expectContains: []string{
				"push_commit_sha:",
				"push_commit_url:",
				"jobs.safe_outputs.outputs.push_commit_sha",
				"jobs.safe_outputs.outputs.push_commit_url",
			},
		},
		{
			name: "workflow_call with multiple safe output types",
			onSection: `"on":
  workflow_call:
  issues:
    types: [opened]`,
			safeOutputs: &SafeOutputsConfig{
				CreateIssues:       &CreateIssuesConfig{},
				CreatePullRequests: &CreatePullRequestsConfig{},
				AddComments:        &AddCommentsConfig{},
			},
			expectContains: []string{
				"created_issue_number:",
				"created_issue_url:",
				"created_pr_number:",
				"created_pr_url:",
				"comment_id:",
				"comment_url:",
			},
		},
		{
			name: "workflow_call with no relevant safe output types",
			onSection: `"on":
  workflow_call:`,
			safeOutputs: &SafeOutputsConfig{
				AssignToAgent: &AssignToAgentConfig{},
			},
			expectUnchanged: true,
		},
		{
			name: "user-defined outputs are preserved when merged",
			onSection: `"on":
  workflow_call:
    outputs:
      my_custom_output:
        description: Custom output
        value: ${{ jobs.my_job.outputs.my_value }}`,
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
			},
			expectContains: []string{
				"my_custom_output:",
				"created_issue_number:",
				"created_issue_url:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.injectWorkflowCallOutputs(tt.onSection, tt.safeOutputs)

			if tt.expectUnchanged {
				assert.Equal(t, tt.onSection, result, "on section should be unchanged")
				return
			}

			require.NotEmpty(t, result, "result should not be empty")

			for _, expected := range tt.expectContains {
				assert.Contains(t, result, expected,
					"expected result to contain %q", expected)
			}
			for _, absent := range tt.expectAbsent {
				assert.NotContains(t, result, absent,
					"expected result NOT to contain %q", absent)
			}
		})
	}
}

func TestBuildWorkflowCallOutputsMap(t *testing.T) {
	tests := []struct {
		name        string
		safeOutputs *SafeOutputsConfig
		expectKeys  []string
		absentKeys  []string
	}{
		{
			name:        "create-issue adds 2 outputs",
			safeOutputs: &SafeOutputsConfig{CreateIssues: &CreateIssuesConfig{}},
			expectKeys:  []string{"created_issue_number", "created_issue_url"},
			absentKeys:  []string{"created_pr_number", "comment_id", "push_commit_sha"},
		},
		{
			name:        "create-pull-request adds 2 outputs",
			safeOutputs: &SafeOutputsConfig{CreatePullRequests: &CreatePullRequestsConfig{}},
			expectKeys:  []string{"created_pr_number", "created_pr_url"},
			absentKeys:  []string{"created_issue_number"},
		},
		{
			name:        "add-comment adds 2 outputs",
			safeOutputs: &SafeOutputsConfig{AddComments: &AddCommentsConfig{}},
			expectKeys:  []string{"comment_id", "comment_url"},
		},
		{
			name:        "push-to-pull-request-branch adds 2 outputs",
			safeOutputs: &SafeOutputsConfig{PushToPullRequestBranch: &PushToPullRequestBranchConfig{}},
			expectKeys:  []string{"push_commit_sha", "push_commit_url"},
		},
		{
			name:        "no relevant types returns empty map",
			safeOutputs: &SafeOutputsConfig{AssignToAgent: &AssignToAgentConfig{}},
			absentKeys:  []string{"created_issue_number", "created_pr_number", "comment_id", "push_commit_sha"},
		},
		{
			name: "multiple types produce all outputs",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues:            &CreateIssuesConfig{},
				CreatePullRequests:      &CreatePullRequestsConfig{},
				AddComments:             &AddCommentsConfig{},
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{},
			},
			expectKeys: []string{
				"created_issue_number", "created_issue_url",
				"created_pr_number", "created_pr_url",
				"comment_id", "comment_url",
				"push_commit_sha", "push_commit_url",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildWorkflowCallOutputsMap(tt.safeOutputs)

			for _, key := range tt.expectKeys {
				entry, ok := result[key]
				assert.True(t, ok, "expected key %q in outputs map", key)
				assert.NotEmpty(t, entry.Description, "description for %q should not be empty", key)
				assert.NotEmpty(t, entry.Value, "value for %q should not be empty", key)
				assert.Contains(t, entry.Value, "jobs.safe_outputs.outputs.",
					"value for %q should reference safe_outputs job", key)
			}
			for _, key := range tt.absentKeys {
				_, ok := result[key]
				assert.False(t, ok, "key %q should not be in outputs map", key)
			}
		})
	}
}

// TestWorkflowCallOutputsEndToEnd tests that compiling a workflow with workflow_call + safe-outputs
// produces the correct on.workflow_call.outputs section in the compiled YAML.
func TestWorkflowCallOutputsEndToEnd(t *testing.T) {
	workflowContent := `---
on:
  workflow_call:
  workflow_dispatch:
engine: copilot
safe-outputs:
  create-issue:
    title-prefix: "[ai] "
  add-comment:
    max: 1
---

# Test Workflow

Create an issue and add a comment.
`

	tmpDir := testutil.TempDir(t, "workflow-call-outputs-test")
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(workflowContent), 0644)
	require.NoError(t, err, "failed to write test file")

	compiler := NewCompiler()
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "failed to parse workflow file")

	yamlOutput, err := compiler.generateYAML(workflowData, testFile)
	require.NoError(t, err, "failed to generate YAML")

	// Verify workflow_call outputs section is present
	assert.Contains(t, yamlOutput, "workflow_call:", "should contain workflow_call trigger")
	assert.Contains(t, yamlOutput, "outputs:", "should contain outputs section")

	// Verify create-issue outputs
	assert.Contains(t, yamlOutput, "created_issue_number:", "should contain created_issue_number")
	assert.Contains(t, yamlOutput, "created_issue_url:", "should contain created_issue_url")
	assert.Contains(t, yamlOutput, "jobs.safe_outputs.outputs.created_issue_number", "should reference safe_outputs job created_issue_number")

	// Verify add-comment outputs
	assert.Contains(t, yamlOutput, "comment_id:", "should contain comment_id")
	assert.Contains(t, yamlOutput, "comment_url:", "should contain comment_url")

	// Verify safe_outputs job outputs contain the new individual outputs
	assert.Contains(t, yamlOutput, "created_issue_number: ${{ steps.process_safe_outputs.outputs.created_issue_number }}",
		"safe_outputs job should expose created_issue_number")
	assert.Contains(t, yamlOutput, "comment_id: ${{ steps.process_safe_outputs.outputs.comment_id }}",
		"safe_outputs job should expose comment_id")
}
