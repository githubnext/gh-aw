package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// TestRenderBuiltinMCPServerBlock verifies the shared helper function that eliminated code duplication
func TestRenderBuiltinMCPServerBlock(t *testing.T) {
	tests := []struct {
		name                 string
		serverID             string
		command              string
		args                 []string
		envVars              []string
		isLast               bool
		includeCopilotFields bool
		expectedContent      []string
		unexpectedContent    []string
	}{
		{
			name:     "SafeOutputs Copilot format",
			serverID: constants.SafeOutputsMCPServerID,
			command:  "node",
			args:     []string{"/opt/gh-aw/safeoutputs/mcp-server.cjs"},
			envVars: []string{
				"GH_AW_SAFE_OUTPUTS",
				"GH_AW_ASSETS_BRANCH",
			},
			isLast:               true,
			includeCopilotFields: true,
			expectedContent: []string{
				`"safeoutputs": {`,
				`"type": "local"`,
				`"command": "node"`,
				`"args": ["/opt/gh-aw/safeoutputs/mcp-server.cjs"]`,
				`"tools": ["*"]`,
				`"env": {`,
				`"GH_AW_SAFE_OUTPUTS": "\${GH_AW_SAFE_OUTPUTS}"`,
				`"GH_AW_ASSETS_BRANCH": "\${GH_AW_ASSETS_BRANCH}"`,
				`              }`, // isLast = true, no comma
			},
			unexpectedContent: []string{},
		},
		{
			name:     "SafeOutputs Claude format",
			serverID: constants.SafeOutputsMCPServerID,
			command:  "node",
			args:     []string{"/opt/gh-aw/safeoutputs/mcp-server.cjs"},
			envVars: []string{
				"GH_AW_SAFE_OUTPUTS",
				"GH_AW_ASSETS_BRANCH",
			},
			isLast:               false,
			includeCopilotFields: false,
			expectedContent: []string{
				`"safeoutputs": {`,
				`"command": "node"`,
				`"args": ["/opt/gh-aw/safeoutputs/mcp-server.cjs"]`,
				`"env": {`,
				`"GH_AW_SAFE_OUTPUTS": "$GH_AW_SAFE_OUTPUTS"`,
				`"GH_AW_ASSETS_BRANCH": "$GH_AW_ASSETS_BRANCH"`,
				`              },`, // isLast = false, with comma
			},
			unexpectedContent: []string{
				`"type"`,
				`"tools"`,
				`\\${`, // Should not have backslash-escaped variables in Claude format
			},
		},
		{
			name:                 "AgenticWorkflows Copilot format",
			serverID:             "agentic_workflows",
			command:              "gh",
			args:                 []string{"aw", "mcp-server"},
			envVars:              []string{"GITHUB_TOKEN"},
			isLast:               false,
			includeCopilotFields: true,
			expectedContent: []string{
				`"agentic_workflows": {`,
				`"type": "local"`,
				`"command": "gh"`,
				`"args": ["aw", "mcp-server"]`,
				`"tools": ["*"]`,
				`"env": {`,
				`"GITHUB_TOKEN": "\${GITHUB_TOKEN}"`,
				`              },`, // isLast = false, with comma
			},
			unexpectedContent: []string{},
		},
		{
			name:                 "AgenticWorkflows Claude format",
			serverID:             "agentic_workflows",
			command:              "gh",
			args:                 []string{"aw", "mcp-server"},
			envVars:              []string{"GITHUB_TOKEN"},
			isLast:               true,
			includeCopilotFields: false,
			expectedContent: []string{
				`"agentic_workflows": {`,
				`"command": "gh"`,
				`"args": ["aw", "mcp-server"]`,
				`"env": {`,
				`"GITHUB_TOKEN": "$GITHUB_TOKEN"`,
				`              }`, // isLast = true, no comma
			},
			unexpectedContent: []string{
				`"type"`,
				`"tools"`,
				`\\${`, // Should not have backslash-escaped variables in Claude format
			},
		},
		{
			name:                 "Multiple args formatting",
			serverID:             "test_server",
			command:              "testcmd",
			args:                 []string{"arg1", "arg2", "arg3"},
			envVars:              []string{"VAR1", "VAR2"},
			isLast:               false,
			includeCopilotFields: true,
			expectedContent: []string{
				`"test_server": {`,
				`"args": ["arg1", "arg2", "arg3"]`,
				`"VAR1": "\${VAR1}"`,
				`"VAR2": "\${VAR2}"`,
			},
			unexpectedContent: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder

			renderBuiltinMCPServerBlock(&output, tt.serverID, tt.command, tt.args, tt.envVars, tt.isLast, tt.includeCopilotFields)

			result := output.String()

			// Check expected content
			for _, expected := range tt.expectedContent {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected content not found: %q\nActual output:\n%s", expected, result)
				}
			}

			// Check unexpected content
			for _, unexpected := range tt.unexpectedContent {
				if strings.Contains(result, unexpected) {
					t.Errorf("Unexpected content found: %q\nActual output:\n%s", unexpected, result)
				}
			}
		})
	}
}

// TestBuiltinMCPServerBlockCommaHandling specifically tests comma handling for isLast parameter
func TestBuiltinMCPServerBlockCommaHandling(t *testing.T) {
	tests := []struct {
		name           string
		isLast         bool
		expectedEnding string
	}{
		{
			name:           "Not last - should have comma",
			isLast:         false,
			expectedEnding: "              },\n",
		},
		{
			name:           "Is last - should not have comma",
			isLast:         true,
			expectedEnding: "              }\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder

			renderBuiltinMCPServerBlock(&output, "test", "node", []string{"arg"}, []string{"VAR"}, tt.isLast, false)

			result := output.String()

			if !strings.HasSuffix(result, tt.expectedEnding) {
				t.Errorf("Expected ending %q but got:\n%s", tt.expectedEnding, result)
			}
		})
	}
}

// TestBuiltinMCPServerBlockEnvVarOrdering tests that environment variables maintain order
func TestBuiltinMCPServerBlockEnvVarOrdering(t *testing.T) {
	envVars := []string{"VAR_A", "VAR_B", "VAR_C", "VAR_D"}

	var output strings.Builder
	renderBuiltinMCPServerBlock(&output, "test", "cmd", []string{"arg"}, envVars, true, false)

	result := output.String()

	// Find positions of each variable in the output
	positions := make(map[string]int)
	for _, envVar := range envVars {
		pos := strings.Index(result, `"`+envVar+`"`)
		if pos == -1 {
			t.Errorf("Environment variable %s not found in output", envVar)
			continue
		}
		positions[envVar] = pos
	}

	// Verify ordering
	for i := 0; i < len(envVars)-1; i++ {
		currentVar := envVars[i]
		nextVar := envVars[i+1]

		if positions[currentVar] >= positions[nextVar] {
			t.Errorf("Environment variables out of order: %s should come before %s", currentVar, nextVar)
		}
	}
}
