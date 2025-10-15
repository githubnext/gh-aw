package workflow

import (
	"strings"
	"testing"
)

// TestSafeOutputConditionWithMin tests that when min > 0, the job condition
// does not check if output_types contains the message type
func TestSafeOutputConditionWithMin(t *testing.T) {
	tests := []struct {
		name                string
		frontmatter         map[string]any
		expectedCondition   string
		unexpectedCondition string
	}{
		{
			name: "missing-tool without min should check contains",
			frontmatter: map[string]any{
				"name": "Test",
				"safe-outputs": map[string]any{
					"missing-tool": map[string]any{
						"max": 5,
					},
				},
			},
			expectedCondition:   "contains(needs.agent.outputs.output_types, 'missing_tool')",
			unexpectedCondition: "",
		},
		{
			name: "missing-tool with min > 0 should not check contains",
			frontmatter: map[string]any{
				"name": "Test",
				"safe-outputs": map[string]any{
					"missing-tool": map[string]any{
						"min": 1,
						"max": 5,
					},
				},
			},
			expectedCondition:   "always()",
			unexpectedCondition: "contains(needs.agent.outputs.output_types, 'missing_tool')",
		},
		{
			name: "create-issue without min should check contains",
			frontmatter: map[string]any{
				"name": "Test",
				"safe-outputs": map[string]any{
					"create-issue": map[string]any{
						"max": 3,
					},
					"missing-tool": false,
				},
			},
			expectedCondition:   "contains(needs.agent.outputs.output_types, 'create_issue')",
			unexpectedCondition: "",
		},
		{
			name: "create-issue with min > 0 should not check contains",
			frontmatter: map[string]any{
				"name": "Test",
				"safe-outputs": map[string]any{
					"create-issue": map[string]any{
						"min": 2,
						"max": 5,
					},
					"missing-tool": false,
				},
			},
			expectedCondition:   "always()",
			unexpectedCondition: "contains(needs.agent.outputs.output_types, 'create_issue')",
		},
		{
			name: "add-comment with min > 0 should not check contains",
			frontmatter: map[string]any{
				"name": "Test",
				"safe-outputs": map[string]any{
					"add-comment": map[string]any{
						"min": 1,
					},
					"missing-tool": false,
				},
			},
			expectedCondition:   "always()",
			unexpectedCondition: "contains(needs.agent.outputs.output_types, 'add-comment')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")

			// Extract safe outputs config
			safeOutputs := compiler.extractSafeOutputsConfig(tt.frontmatter)
			if safeOutputs == nil {
				t.Fatal("Expected SafeOutputsConfig to be created")
			}

			// Build the appropriate job based on what's configured
			var job *Job
			var err error

			workflowData := &WorkflowData{
				SafeOutputs: safeOutputs,
			}

			if safeOutputs.MissingTool != nil {
				job, err = compiler.buildCreateOutputMissingToolJob(workflowData, "agent")
			} else if safeOutputs.CreateIssues != nil {
				job, err = compiler.buildCreateOutputIssueJob(workflowData, "agent")
			} else if safeOutputs.AddComments != nil {
				job, err = compiler.buildCreateOutputAddCommentJob(workflowData, "agent")
			}

			if err != nil {
				t.Fatalf("Failed to build job: %v", err)
			}
			if job == nil {
				t.Fatal("Expected job to be created")
			}

			// Check the job condition
			condition := job.If
			if tt.expectedCondition != "" && !strings.Contains(condition, tt.expectedCondition) {
				t.Errorf("Expected condition to contain '%s', but got: %s", tt.expectedCondition, condition)
			}
			if tt.unexpectedCondition != "" && strings.Contains(condition, tt.unexpectedCondition) {
				t.Errorf("Expected condition NOT to contain '%s', but got: %s", tt.unexpectedCondition, condition)
			}
		})
	}
}

// TestBuildSafeOutputTypeWithMin tests the BuildSafeOutputType function directly
func TestBuildSafeOutputTypeWithMin(t *testing.T) {
	tests := []struct {
		name                string
		outputType          string
		min                 int
		expectedCondition   string
		unexpectedCondition string
	}{
		{
			name:                "with min=0 should include contains check",
			outputType:          "create-issue",
			min:                 0,
			expectedCondition:   "contains(needs.agent.outputs.output_types, 'create-issue')",
			unexpectedCondition: "",
		},
		{
			name:                "with min>0 should only have always()",
			outputType:          "create-issue",
			min:                 1,
			expectedCondition:   "always()",
			unexpectedCondition: "contains(needs.agent.outputs.output_types, 'create-issue')",
		},
		{
			name:                "missing-tool with min=0",
			outputType:          "missing-tool",
			min:                 0,
			expectedCondition:   "contains(needs.agent.outputs.output_types, 'missing-tool')",
			unexpectedCondition: "",
		},
		{
			name:                "missing-tool with min>0",
			outputType:          "missing-tool",
			min:                 2,
			expectedCondition:   "always()",
			unexpectedCondition: "contains(needs.agent.outputs.output_types, 'missing-tool')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := BuildSafeOutputType(tt.outputType, tt.min).Render()

			if tt.expectedCondition != "" && !strings.Contains(condition, tt.expectedCondition) {
				t.Errorf("Expected condition to contain '%s', but got: %s", tt.expectedCondition, condition)
			}
			if tt.unexpectedCondition != "" && strings.Contains(condition, tt.unexpectedCondition) {
				t.Errorf("Expected condition NOT to contain '%s', but got: %s", tt.unexpectedCondition, condition)
			}
		})
	}
}

// TestMinConditionInCompiledWorkflow tests that a compiled workflow with min > 0
// generates the correct job condition
func TestMinConditionInCompiledWorkflow(t *testing.T) {
	// Create a temporary workflow with min configuration
	frontmatter := map[string]any{
		"name": "Test Min Workflow",
		"on":   map[string]any{"workflow_dispatch": nil},
		"safe-outputs": map[string]any{
			"missing-tool": map[string]any{
				"min": 1,
				"max": 5,
			},
		},
	}

	compiler := NewCompiler(false, "", "test")
	safeOutputs := compiler.extractSafeOutputsConfig(frontmatter)

	if safeOutputs == nil || safeOutputs.MissingTool == nil {
		t.Fatal("Expected MissingTool config to be created")
	}

	if safeOutputs.MissingTool.Min != 1 {
		t.Errorf("Expected min to be 1, got %d", safeOutputs.MissingTool.Min)
	}

	workflowData := &WorkflowData{
		SafeOutputs: safeOutputs,
	}

	job, err := compiler.buildCreateOutputMissingToolJob(workflowData, "agent")
	if err != nil {
		t.Fatalf("Failed to build job: %v", err)
	}

	// Verify that the condition only contains always() and not the contains check
	if !strings.Contains(job.If, "always()") {
		t.Error("Expected condition to contain 'always()'")
	}
	if strings.Contains(job.If, "contains(needs.agent.outputs.output_types, 'missing-tool')") {
		t.Error("Expected condition NOT to contain contains check when min > 0")
	}
}
