//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// agentJobStepsYAML is a reusable YAML snippet that defines two agent-job steps
// with IDs "prepare" and "fetch_data". Used across multiple subtests.
const agentJobStepsYAML = `steps:
  - name: Prepare workspace
    id: prepare
    run: |
      mkdir -p /tmp/gh-aw/agent
  - name: Fetch data
    id: fetch_data
    run: |
      echo "result=42" >> $GITHUB_OUTPUT
`

func TestValidateStepsOutputsNotInPrompt(t *testing.T) {
	tests := []struct {
		name          string
		markdownBody  string
		customSteps   string
		shouldError   bool
		errorContains string
	}{
		{
			name:         "no steps, no expressions",
			markdownBody: "# My Workflow\n\nDo something useful.",
			customSteps:  "",
			shouldError:  false,
		},
		{
			name:         "steps with no outputs in prompt",
			markdownBody: "# My Workflow\n\nRead results from /tmp/gh-aw/agent/result.txt.",
			customSteps:  agentJobStepsYAML,
			shouldError:  false,
		},
		{
			name: "steps with unrelated expressions in prompt",
			markdownBody: "# My Workflow\n\n" +
				"Repository: ${{ github.repository }}\n" +
				"PR number: ${{ github.event.pull_request.number }}\n",
			customSteps: agentJobStepsYAML,
			shouldError: false,
		},
		{
			name: "steps.sanitized in prompt without agent-job steps",
			markdownBody: "# My Workflow\n\n" +
				"Content: ${{ steps.sanitized.outputs.text }}\n",
			customSteps: "",
			shouldError: false,
		},
		{
			name: "steps.sanitized in prompt with different agent-job steps",
			markdownBody: "# My Workflow\n\n" +
				"Content: ${{ steps.sanitized.outputs.text }}\n",
			customSteps: agentJobStepsYAML,
			shouldError: false,
		},
		{
			name: "agent-job step output referenced in prompt",
			markdownBody: "# My Workflow\n\n" +
				"The result is: ${{ steps.fetch_data.outputs.result }}\n",
			customSteps:   agentJobStepsYAML,
			shouldError:   true,
			errorContains: "fetch_data",
		},
		{
			name: "multiple agent-job step outputs referenced in prompt",
			markdownBody: "# My Workflow\n\n" +
				"Prep: ${{ steps.prepare.outputs.status }}\n" +
				"Result: ${{ steps.fetch_data.outputs.result }}\n",
			customSteps:   agentJobStepsYAML,
			shouldError:   true,
			errorContains: "steps-output-in-prompt",
		},
		{
			name: "step without id not flagged even if referenced",
			markdownBody: "# My Workflow\n\n" +
				"Value: ${{ steps.unnamed_step.outputs.value }}\n",
			customSteps: `steps:
  - name: A step without an id
    run: echo "no id here"
`,
			shouldError: false,
		},
		{
			name: "needs.* expressions not affected",
			markdownBody: "# My Workflow\n\n" +
				"CI status: ${{ needs.check_ci.outputs.status }}\n",
			customSteps: agentJobStepsYAML,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				Name:            "Test",
				MarkdownContent: tt.markdownBody,
				CustomSteps:     tt.customSteps,
			}

			err := validateStepsOutputsNotInPrompt(workflowData)
			if tt.shouldError {
				require.Error(t, err, "expected validateStepsOutputsNotInPrompt to return an error")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExtractAgentJobStepIDs(t *testing.T) {
	tests := []struct {
		name           string
		customSteps    string
		preSteps       string
		preAgentSteps  string
		postSteps      string
		expectedIDs    []string
		notExpectedIDs []string
	}{
		{
			name:        "empty steps",
			customSteps: "",
			expectedIDs: []string{},
		},
		{
			name:        "custom steps with IDs",
			customSteps: agentJobStepsYAML,
			expectedIDs: []string{"prepare", "fetch_data"},
		},
		{
			name: "step without id is not collected",
			customSteps: `steps:
  - name: Setup
    run: echo hi
`,
			expectedIDs:    []string{},
			notExpectedIDs: []string{"Setup"},
		},
		{
			name: "IDs collected from multiple step sections",
			customSteps: `steps:
  - name: Main
    id: main_step
    run: echo main
`,
			preSteps: `pre-steps:
  - name: Pre
    id: pre_step
    run: echo pre
`,
			postSteps: `post-steps:
  - name: Post
    id: post_step
    run: echo post
`,
			expectedIDs: []string{"main_step", "pre_step", "post_step"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				CustomSteps:   tt.customSteps,
				PreSteps:      tt.preSteps,
				PreAgentSteps: tt.preAgentSteps,
				PostSteps:     tt.postSteps,
			}

			ids := extractAgentJobStepIDs(workflowData)

			for _, expectedID := range tt.expectedIDs {
				assert.Contains(t, ids, expectedID, "expected step ID %q to be collected", expectedID)
			}
			for _, notExpectedID := range tt.notExpectedIDs {
				assert.NotContains(t, ids, notExpectedID, "step ID %q should not be collected", notExpectedID)
			}
		})
	}
}

// TestValidateStepsOutputsNotInPromptViaCompiler verifies that the validation
// is wired into the compilation pipeline and produces a compile error when a
// step output is referenced in the prompt body.
func TestValidateStepsOutputsNotInPromptViaCompiler(t *testing.T) {
	tests := []struct {
		name          string
		workflow      string
		shouldError   bool
		errorContains string
	}{
		{
			name: "agent-job step output referenced in prompt produces error",
			workflow: `---
on: push
permissions:
  contents: read
engine: copilot
steps:
  - name: Compute result
    id: compute
    run: echo "value=42" >> $GITHUB_OUTPUT
---

# My Workflow

Process this value: ${{ steps.compute.outputs.value }}
`,
			shouldError:   true,
			errorContains: "compute",
		},
		{
			name: "file-based approach compiles without error",
			workflow: `---
on: push
permissions:
  contents: read
engine: copilot
steps:
  - name: Compute result
    id: compute
    run: echo "42" > /tmp/gh-aw/agent/result.txt
---

# My Workflow

Read the result from /tmp/gh-aw/agent/result.txt.
`,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			workflowPath := filepath.Join(tmpDir, "test-workflow.md")
			require.NoError(t, os.WriteFile(workflowPath, []byte(tt.workflow), 0644))

			compiler := NewCompiler()
			err := compiler.CompileWorkflow(workflowPath)
			if tt.shouldError {
				require.Error(t, err, "expected compilation to fail")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
