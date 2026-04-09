//go:build integration

package workflow

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddCommentDiscussionsFieldIntegration verifies that the discussions: true/false
// field on add-comment is accepted and properly controls discussions:write permission.
func TestAddCommentDiscussionsFieldIntegration(t *testing.T) {
	tests := []struct {
		name                   string
		frontmatter            map[string]any
		expectDiscussionsWrite bool
		shouldCompile          bool
	}{
		{
			name: "discussions defaults to true - includes discussions:write",
			frontmatter: map[string]any{
				"name":   "Test Workflow",
				"engine": "copilot",
				"safe-outputs": map[string]any{
					"add-comment": map[string]any{},
				},
			},
			expectDiscussionsWrite: true,
			shouldCompile:          true,
		},
		{
			name: "discussions: true - explicitly includes discussions:write",
			frontmatter: map[string]any{
				"name":   "Test Workflow",
				"engine": "copilot",
				"safe-outputs": map[string]any{
					"add-comment": map[string]any{
						"discussions": true,
					},
				},
			},
			expectDiscussionsWrite: true,
			shouldCompile:          true,
		},
		{
			name: "discussions: false - excludes discussions:write permission",
			frontmatter: map[string]any{
				"name":   "Test Workflow",
				"engine": "copilot",
				"safe-outputs": map[string]any{
					"add-comment": map[string]any{
						"discussions": false,
					},
				},
			},
			expectDiscussionsWrite: false,
			shouldCompile:          true,
		},
		{
			name: "discussions: false with issues: false - only pull-requests:write",
			frontmatter: map[string]any{
				"name":   "Test Workflow",
				"engine": "copilot",
				"safe-outputs": map[string]any{
					"add-comment": map[string]any{
						"discussions": false,
						"issues":      false,
					},
				},
			},
			expectDiscussionsWrite: false,
			shouldCompile:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			workflowData := &WorkflowData{
				Name: tt.frontmatter["name"].(string),
			}
			workflowData.SafeOutputs = compiler.extractSafeOutputsConfig(tt.frontmatter)

			require.NotNil(t, workflowData.SafeOutputs, "Expected SafeOutputs to be parsed")
			require.NotNil(t, workflowData.SafeOutputs.AddComments, "Expected AddComments to be configured")

			job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, "main", "")
			if tt.shouldCompile {
				require.NoError(t, err, "Expected workflow to compile successfully")
				require.NotNil(t, job, "Expected consolidated job to be built")

				if tt.expectDiscussionsWrite {
					assert.Contains(t, job.Permissions, "discussions: write", "Expected discussions:write permission to be present")
				} else {
					assert.NotContains(t, job.Permissions, "discussions: write", "Expected discussions:write permission to be absent")
				}
			} else {
				assert.Error(t, err, "Expected compilation to fail")
			}
		})
	}
}

// TestAddCommentDiscussionsFieldSchemaIntegration verifies that the discussions field
// is accepted in full compilation (frontmatter parsing + schema validation).
func TestAddCommentDiscussionsFieldSchemaIntegration(t *testing.T) {
	tests := []struct {
		name          string
		workflowMD    string
		shouldCompile bool
	}{
		{
			name: "discussions: true is accepted",
			workflowMD: `---
name: Test Workflow
engine: copilot
safe-outputs:
  add-comment:
    discussions: true
---

# Test Workflow
Handle issues.
`,
			shouldCompile: true,
		},
		{
			name: "discussions: false is accepted",
			workflowMD: `---
name: Test Workflow
engine: copilot
safe-outputs:
  add-comment:
    discussions: false
---

# Test Workflow
Handle issues.
`,
			shouldCompile: true,
		},
		{
			name: "add-comment without discussions field is accepted",
			workflowMD: `---
name: Test Workflow
engine: copilot
safe-outputs:
  add-comment: {}
---

# Test Workflow
Handle issues.
`,
			shouldCompile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			compiler.SetSkipValidation(false)

			workflowPath := t.TempDir() + "/test-workflow.md"
			err := os.WriteFile(workflowPath, []byte(tt.workflowMD), 0644)
			require.NoError(t, err, "Failed to write test workflow file")

			err = compiler.CompileWorkflow(workflowPath)
			if tt.shouldCompile {
				assert.NoError(t, err, "Expected workflow to compile successfully")
			} else {
				assert.Error(t, err, "Expected workflow to fail compilation")
			}
		})
	}
}
