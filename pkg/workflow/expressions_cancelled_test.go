package workflow

import (
	"strings"
	"testing"
)

// TestBuildSafeOutputTypeWithCancelled verifies that BuildSafeOutputType uses !cancelled()
// instead of always() to properly respect workflow cancellation.
//
// Background:
// - always() runs even when the workflow is cancelled (incorrect behavior)
// - !cancelled() runs unless the workflow is cancelled (correct behavior)
//
// This test ensures safe-output jobs:
// 1. Run when dependencies succeed
// 2. Run when dependencies fail (for error reporting)
// 3. Skip when the workflow is cancelled (respecting user intent)
func TestBuildSafeOutputTypeWithCancelled(t *testing.T) {
	tests := []struct {
		name               string
		outputType         string
		min                int
		expectedContains   []string
		unexpectedContains []string
	}{
		{
			name:       "min=0 should use !cancelled() with contains check",
			outputType: "create_issue",
			min:        0,
			expectedContains: []string{
				"!(cancelled())",
				"contains(needs.agent.outputs.output_types, 'create_issue')",
			},
			unexpectedContains: []string{
				"always()",
			},
		},
		{
			name:       "min>0 should use !cancelled() without contains check",
			outputType: "create_issue",
			min:        1,
			expectedContains: []string{
				"!(cancelled())",
			},
			unexpectedContains: []string{
				"always()",
				"contains(needs.agent.outputs.output_types, 'create_issue')",
			},
		},
		{
			name:       "push-to-pull-request-branch should use !cancelled()",
			outputType: "push_to_pull_request_branch",
			min:        0,
			expectedContains: []string{
				"!(cancelled())",
				"contains(needs.agent.outputs.output_types, 'push_to_pull_request_branch')",
			},
			unexpectedContains: []string{
				"always()",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := BuildSafeOutputType(tt.outputType, tt.min).Render()

			// Verify expected strings are present
			for _, expected := range tt.expectedContains {
				if !strings.Contains(condition, expected) {
					t.Errorf("Expected condition to contain '%s', but got: %s", expected, condition)
				}
			}

			// Verify unexpected strings are NOT present
			for _, unexpected := range tt.unexpectedContains {
				if strings.Contains(condition, unexpected) {
					t.Errorf("Expected condition NOT to contain '%s', but got: %s", unexpected, condition)
				}
			}
		})
	}
}
