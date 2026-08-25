//go:build !integration

package workflow

import (
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
)

func TestWorkflowDispatchConcurrencyWarning(t *testing.T) {
	tests := []struct {
		name          string
		on            string
		discriminator string
		expectWarning bool
	}{
		{
			name:          "workflow dispatch without discriminator",
			on:            "on: workflow_dispatch",
			expectWarning: false,
		},
		{
			name:          "workflow dispatch with empty inputs without discriminator",
			on:            "on:\n  workflow_dispatch:\n    inputs: {}",
			expectWarning: false,
		},
		{
			name:          "workflow dispatch with inputs without discriminator",
			on:            "on:\n  workflow_dispatch:\n    inputs:\n      target_repo:\n        type: string",
			expectWarning: true,
		},
		{
			name:          "mixed workflow dispatch without discriminator",
			on:            "on:\n  workflow_dispatch:\n  schedule:\n    - cron: '0 0 * * *'",
			expectWarning: false,
		},
		{
			name:          "mixed workflow dispatch with inputs without discriminator",
			on:            "on:\n  workflow_dispatch:\n    inputs:\n      target_repo:\n        type: string\n  schedule:\n    - cron: '0 0 * * *'",
			expectWarning: true,
		},
		{
			name:          "workflow dispatch with inputs and discriminator",
			on:            "on:\n  workflow_dispatch:\n    inputs:\n      target_repo:\n        type: string",
			discriminator: "${{ github.run_id }}",
			expectWarning: false,
		},
		{
			name:          "nested workflow dispatch value without discriminator",
			on:            "on:\n  workflow_run:\n    workflows: [workflow_dispatch]\n    types: [completed]",
			expectWarning: false,
		},
		{
			name:          "schedule without discriminator",
			on:            "on:\n  schedule:\n    - cron: '0 0 * * *'",
			expectWarning: false,
		},
	}

	const warning = "workflow_dispatch workflow has no concurrency.job-discriminator"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			workflowData := &WorkflowData{
				On:                          tt.on,
				ConcurrencyJobDiscriminator: tt.discriminator,
			}

			output := testutil.CaptureStderr(t, func() {
				compiler.emitGeneralToolWarnings(workflowData, "test.md")
			})

			if tt.expectWarning {
				assert.Contains(t, output, warning)
				assert.Equal(t, 1, compiler.GetWarningCount())
			} else {
				assert.NotContains(t, output, warning)
				assert.Zero(t, compiler.GetWarningCount())
			}
		})
	}
}
