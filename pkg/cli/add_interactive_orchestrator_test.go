//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddInteractiveConfig_determineFilesToAdd(t *testing.T) {
	tests := []struct {
		name          string
		workflowSpecs []string
		resolved      *ResolvedWorkflows
		wantFiles     []string
		wantErr       bool
	}{
		{
			name:          "single workflow",
			workflowSpecs: []string{"owner/repo/test-workflow"},
			wantFiles:     []string{"test-workflow.md", "test-workflow.lock.yml"},
			wantErr:       false,
		},
		{
			name:          "multiple workflows",
			workflowSpecs: []string{"owner/repo/workflow-one", "owner/repo/workflow-two"},
			wantFiles:     []string{"workflow-one.md", "workflow-one.lock.yml", "workflow-two.md", "workflow-two.lock.yml"},
			wantErr:       false,
		},
		{
			name:          "workflow with org/repo",
			workflowSpecs: []string{"owner/repo/workflow"},
			wantFiles:     []string{"workflow.md", "workflow.lock.yml"},
			wantErr:       false,
		},
		{
			name:          "invalid spec",
			workflowSpecs: []string{"invalid-spec"},
			wantErr:       true,
		},
		{
			name:          "repository package uses resolved workflows",
			workflowSpecs: []string{"owner/repo"},
			resolved: &ResolvedWorkflows{
				Workflows: []*ResolvedWorkflow{
					{
						Spec: &WorkflowSpec{WorkflowName: "review"},
					},
					{
						Spec: &WorkflowSpec{WorkflowName: "nightly-review"},
					},
				},
			},
			wantFiles: []string{"review.md", "review.lock.yml", "nightly-review.md", "nightly-review.lock.yml"},
			wantErr:   false,
		},
		{
			name:          "invalid resolved workflow fails loudly",
			workflowSpecs: []string{"owner/repo"},
			resolved: &ResolvedWorkflows{
				Workflows: []*ResolvedWorkflow{
					{},
				},
			},
			wantErr: true,
		},
		{
			name:          "resolved workflow with blank name fails loudly",
			workflowSpecs: []string{"owner/repo"},
			resolved: &ResolvedWorkflows{
				Workflows: []*ResolvedWorkflow{
					{
						Spec: &WorkflowSpec{WorkflowName: "   "},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AddInteractiveConfig{
				WorkflowSpecs:     tt.workflowSpecs,
				resolvedWorkflows: tt.resolved,
			}

			workflowFiles, initFiles, err := config.determineFilesToAdd()

			if tt.wantErr {
				assert.Error(t, err, "Expected error but got none")
			} else {
				require.NoError(t, err, "Unexpected error")
				assert.Equal(t, tt.wantFiles, workflowFiles, "Workflow files should match")
				assert.Empty(t, initFiles, "Init files should be empty")
			}
		})
	}
}

func TestAddInteractiveConfig_primaryWorkflowName(t *testing.T) {
	t.Run("uses resolved workflow for repository package", func(t *testing.T) {
		config := &AddInteractiveConfig{
			WorkflowSpecs: []string{"owner/repo"},
			resolvedWorkflows: &ResolvedWorkflows{
				Workflows: []*ResolvedWorkflow{
					{
						Spec: &WorkflowSpec{WorkflowName: "review"},
					},
				},
			},
		}

		assert.Equal(t, "review", config.primaryWorkflowName())
	})

	t.Run("falls back to parsed workflow spec", func(t *testing.T) {
		config := &AddInteractiveConfig{
			WorkflowSpecs: []string{"owner/repo/test-workflow"},
		}

		assert.Equal(t, "test-workflow", config.primaryWorkflowName())
	})
}

func TestAddInteractiveConfig_showWorkflowDescriptions(t *testing.T) {
	tests := []struct {
		name              string
		resolvedWorkflows *ResolvedWorkflows
		expectOutput      bool
	}{
		{
			name:              "nil resolved workflows",
			resolvedWorkflows: nil,
			expectOutput:      false,
		},
		{
			name: "empty workflows",
			resolvedWorkflows: &ResolvedWorkflows{
				Workflows: []*ResolvedWorkflow{},
			},
			expectOutput: false,
		},
		{
			name: "workflow with description",
			resolvedWorkflows: &ResolvedWorkflows{
				Workflows: []*ResolvedWorkflow{
					{
						Description: "Test workflow description",
					},
				},
			},
			expectOutput: true,
		},
		{
			name: "workflow without description",
			resolvedWorkflows: &ResolvedWorkflows{
				Workflows: []*ResolvedWorkflow{
					{
						Description: "",
					},
				},
			},
			expectOutput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AddInteractiveConfig{
				resolvedWorkflows: tt.resolvedWorkflows,
			}

			// This function prints to stderr, so we just verify it doesn't panic
			require.NotPanics(t, func() {
				config.showWorkflowDescriptions()
			}, "showWorkflowDescriptions should not panic")
		})
	}
}

func TestAddInteractiveConfig_showFinalInstructions(t *testing.T) {
	tests := []struct {
		name              string
		resolvedWorkflows *ResolvedWorkflows
	}{
		{
			name:              "no workflows",
			resolvedWorkflows: nil,
		},
		{
			name: "with workflow",
			resolvedWorkflows: &ResolvedWorkflows{
				Workflows: []*ResolvedWorkflow{
					{
						Spec: &WorkflowSpec{
							WorkflowName: "test-workflow",
						},
						Description: "Test description",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AddInteractiveConfig{
				resolvedWorkflows: tt.resolvedWorkflows,
			}

			// This function prints to stderr, so we just verify it doesn't panic
			require.NotPanics(t, func() {
				config.showFinalInstructions()
			}, "showFinalInstructions should not panic")
		})
	}
}
