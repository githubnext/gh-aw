//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildSafeOutputsSectionsCustomTools verifies that custom jobs, scripts, and actions
// defined in safe-outputs are included in the compiled <safe-output-tools> prompt block.
// This prevents silent drift between the runtime configuration surface and the
// agent-facing compiled instructions.
func TestBuildSafeOutputsSectionsCustomTools(t *testing.T) {
	tests := []struct {
		name          string
		safeOutputs   *SafeOutputsConfig
		expectedTools []string
		expectNil     bool
	}{
		{
			name:      "nil safe outputs returns nil",
			expectNil: true,
		},
		{
			name:        "empty safe outputs returns nil",
			safeOutputs: &SafeOutputsConfig{},
			expectNil:   true,
		},
		{
			name: "custom job appears in tools list",
			safeOutputs: &SafeOutputsConfig{
				NoOp: &NoOpConfig{},
				Jobs: map[string]*SafeJobConfig{
					"deploy": {Description: "Deploy to production"},
				},
			},
			expectedTools: []string{"noop", "deploy"},
		},
		{
			name: "custom job name with dashes is normalized to underscores",
			safeOutputs: &SafeOutputsConfig{
				NoOp: &NoOpConfig{},
				Jobs: map[string]*SafeJobConfig{
					"send-notification": {Description: "Send a notification"},
				},
			},
			expectedTools: []string{"noop", "send_notification"},
		},
		{
			name: "multiple custom jobs are sorted and appear in tools list",
			safeOutputs: &SafeOutputsConfig{
				NoOp: &NoOpConfig{},
				Jobs: map[string]*SafeJobConfig{
					"zebra-job": {},
					"alpha-job": {},
				},
			},
			expectedTools: []string{"noop", "alpha_job", "zebra_job"},
		},
		{
			name: "custom script appears in tools list",
			safeOutputs: &SafeOutputsConfig{
				NoOp: &NoOpConfig{},
				Scripts: map[string]*SafeScriptConfig{
					"my-script": {Description: "Run my script"},
				},
			},
			expectedTools: []string{"noop", "my_script"},
		},
		{
			name: "custom action appears in tools list",
			safeOutputs: &SafeOutputsConfig{
				NoOp: &NoOpConfig{},
				Actions: map[string]*SafeOutputActionConfig{
					"my-action": {Description: "Run my custom action"},
				},
			},
			expectedTools: []string{"noop", "my_action"},
		},
		{
			name: "custom jobs are listed even without predefined tools",
			safeOutputs: &SafeOutputsConfig{
				NoOp: &NoOpConfig{},
				Jobs: map[string]*SafeJobConfig{
					"custom-deploy": {},
				},
			},
			expectedTools: []string{"noop", "custom_deploy"},
		},
		{
			name: "mix of predefined tools and custom jobs both appear in tools list",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
				AddComments:  &AddCommentsConfig{},
				NoOp:         &NoOpConfig{},
				Jobs: map[string]*SafeJobConfig{
					"deploy": {},
				},
				Scripts: map[string]*SafeScriptConfig{
					"notify": {},
				},
			},
			expectedTools: []string{"add_comment", "create_issue", "noop", "deploy", "notify"},
		},
		{
			name: "mix of predefined tools, custom jobs, scripts, and actions all appear",
			safeOutputs: &SafeOutputsConfig{
				CreateIssues: &CreateIssuesConfig{},
				NoOp:         &NoOpConfig{},
				Jobs: map[string]*SafeJobConfig{
					"custom-job": {},
				},
				Scripts: map[string]*SafeScriptConfig{
					"custom-script": {},
				},
				Actions: map[string]*SafeOutputActionConfig{
					"custom-action": {},
				},
			},
			expectedTools: []string{"create_issue", "noop", "custom_job", "custom_script", "custom_action"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sections := buildSafeOutputsSections(tt.safeOutputs)

			if tt.expectNil {
				assert.Nil(t, sections, "Expected nil sections for empty/nil config")
				return
			}

			require.NotNil(t, sections, "Expected non-nil sections")

			// Find the opening section with the Tools: list
			var toolsLine string
			for _, section := range sections {
				if !section.IsFile && strings.HasPrefix(section.Content, "<safe-output-tools>") {
					toolsLine = section.Content
					break
				}
			}

			require.NotEmpty(t, toolsLine, "Expected to find <safe-output-tools> opening section")

			// Extract tool names from "Tools: tool1, tool2, ..." line
			lines := strings.Split(toolsLine, "\n")
			require.GreaterOrEqual(t, len(lines), 2, "Expected at least two lines in tools section")

			toolsListLine := lines[1]
			assert.True(t, strings.HasPrefix(toolsListLine, "Tools: "), "Second line should start with 'Tools: '")

			toolsList := strings.TrimPrefix(toolsListLine, "Tools: ")
			actualToolNames := strings.Split(toolsList, ", ")

			// Strip any max budget annotations like "noop(max:5)" → "noop"
			for i, t := range actualToolNames {
				if name, _, found := strings.Cut(t, "("); found {
					actualToolNames[i] = name
				}
			}

			assert.ElementsMatch(t, tt.expectedTools, actualToolNames,
				"Tool names in <safe-output-tools> should match expected set")
		})
	}
}

// TestBuildSafeOutputsSectionsCustomToolsConsistency verifies that every custom
// tool type registered in the runtime configuration has a corresponding entry in
// the compiled <safe-output-tools> prompt block — preventing silent drift.
func TestBuildSafeOutputsSectionsCustomToolsConsistency(t *testing.T) {
	config := &SafeOutputsConfig{
		NoOp: &NoOpConfig{},
		Jobs: map[string]*SafeJobConfig{
			"job-alpha": {Description: "Alpha job"},
			"job-beta":  {Description: "Beta job"},
		},
		Scripts: map[string]*SafeScriptConfig{
			"script-one": {Description: "Script one"},
		},
		Actions: map[string]*SafeOutputActionConfig{
			"action-x": {Description: "Action X"},
		},
	}

	sections := buildSafeOutputsSections(config)
	require.NotNil(t, sections, "Expected non-nil sections")

	// Concatenate all non-file section content to find the tools block
	var toolsBuilder strings.Builder
	for _, section := range sections {
		if !section.IsFile {
			toolsBuilder.WriteString(section.Content)
			toolsBuilder.WriteString("\n")
		}
	}
	toolsContent := toolsBuilder.String()

	// Every custom job name (normalized) must appear in the tools list
	for jobName := range config.Jobs {
		normalizedName := strings.ReplaceAll(jobName, "-", "_")
		assert.Contains(t, toolsContent, normalizedName,
			"Custom job %q (normalized: %q) should appear in <safe-output-tools>", jobName, normalizedName)
	}

	// Every custom script name (normalized) must appear in the tools list
	for scriptName := range config.Scripts {
		normalizedName := strings.ReplaceAll(scriptName, "-", "_")
		assert.Contains(t, toolsContent, normalizedName,
			"Custom script %q (normalized: %q) should appear in <safe-output-tools>", scriptName, normalizedName)
	}

	// Every custom action name (normalized) must appear in the tools list
	for actionName := range config.Actions {
		normalizedName := strings.ReplaceAll(actionName, "-", "_")
		assert.Contains(t, toolsContent, normalizedName,
			"Custom action %q (normalized: %q) should appear in <safe-output-tools>", actionName, normalizedName)
	}
}
