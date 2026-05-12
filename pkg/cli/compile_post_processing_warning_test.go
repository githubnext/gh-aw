//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/require"
)

func TestDisplayCentralizedSlashCommandRecommendation(t *testing.T) {
	tests := []struct {
		name              string
		workflows         []*workflow.WorkflowData
		jsonOutput        bool
		expectWarning     bool
		expectedWarnCount int
	}{
		{
			name: "warns when three slash commands include non centralized workflows",
			workflows: []*workflow.WorkflowData{
				{Command: []string{"a"}, CommandCentralized: false},
				{Command: []string{"b"}, CommandCentralized: false},
				{Command: []string{"c"}, CommandCentralized: true},
			},
			expectWarning:     true,
			expectedWarnCount: 1,
		},
		{
			name: "does not warn when fewer than three slash commands exist",
			workflows: []*workflow.WorkflowData{
				{Command: []string{"a"}, CommandCentralized: false},
				{Command: []string{"b"}, CommandCentralized: false},
			},
			expectWarning:     false,
			expectedWarnCount: 0,
		},
		{
			name: "does not warn when all slash commands are centralized",
			workflows: []*workflow.WorkflowData{
				{Command: []string{"a"}, CommandCentralized: true},
				{Command: []string{"b"}, CommandCentralized: true},
				{Command: []string{"c"}, CommandCentralized: true},
			},
			expectWarning:     false,
			expectedWarnCount: 0,
		},
		{
			name: "does not warn for json output mode",
			workflows: []*workflow.WorkflowData{
				{Command: []string{"a"}, CommandCentralized: false},
				{Command: []string{"b"}, CommandCentralized: false},
				{Command: []string{"c"}, CommandCentralized: false},
			},
			jsonOutput:        true,
			expectWarning:     false,
			expectedWarnCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := workflow.NewCompiler()

			stderrOutput := testutil.CaptureStderr(t, func() {
				displayCentralizedSlashCommandRecommendation(compiler, tt.workflows, tt.jsonOutput)
			})

			if tt.expectWarning {
				require.Contains(t, stderrOutput, "Consider setting `on.slash_command.strategy: centralized`")
				require.Contains(t, stderrOutput, "Detected 3 slash_command entries")
			} else {
				require.NotContains(t, stderrOutput, "on.slash_command.strategy: centralized")
			}

			require.Equal(t, tt.expectedWarnCount, compiler.GetWarningCount())
		})
	}
}
