//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
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

			actualToolNames := extractToolNamesFromSections(t, sections)

			assert.Equal(t, tt.expectedTools, actualToolNames,
				"Tool names in <safe-output-tools> should match expected order and set")
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

	actualToolNames := extractToolNamesFromSections(t, sections)
	actualToolSet := make(map[string]bool, len(actualToolNames))
	for _, name := range actualToolNames {
		actualToolSet[name] = true
	}

	// Every custom job name (normalized) must appear as an exact tool identifier.
	for jobName := range config.Jobs {
		normalized := stringutil.NormalizeSafeOutputIdentifier(jobName)
		assert.True(t, actualToolSet[normalized],
			"Custom job %q (normalized: %q) should appear as an exact tool identifier in <safe-output-tools>", jobName, normalized)
	}

	// Every custom script name (normalized) must appear as an exact tool identifier.
	for scriptName := range config.Scripts {
		normalized := stringutil.NormalizeSafeOutputIdentifier(scriptName)
		assert.True(t, actualToolSet[normalized],
			"Custom script %q (normalized: %q) should appear as an exact tool identifier in <safe-output-tools>", scriptName, normalized)
	}

	// Every custom action name (normalized) must appear as an exact tool identifier.
	for actionName := range config.Actions {
		normalized := stringutil.NormalizeSafeOutputIdentifier(actionName)
		assert.True(t, actualToolSet[normalized],
			"Custom action %q (normalized: %q) should appear as an exact tool identifier in <safe-output-tools>", actionName, normalized)
	}
}

// TestBuildSafeOutputsSectionsMaxExpressionExtraction verifies that ${{ }} expressions
// in safe-output max: values are extracted to GH_AW_* env vars and replaced with
// __GH_AW_*__ placeholders in the <safe-output-tools> prompt block.
// This prevents ${{ }} from appearing in the run: heredoc, which is subject to the
// GitHub Actions 21KB expression-size limit (regression guard for gh-aw#21158).
func TestBuildSafeOutputsSectionsMaxExpressionExtraction(t *testing.T) {
	maxExpr := "${{ inputs.review-comment-max }}"
	sections := buildSafeOutputsSections(&SafeOutputsConfig{
		CreatePullRequestReviewComments: &CreatePullRequestReviewCommentsConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{
				Max: &maxExpr,
			},
		},
		NoOp: &NoOpConfig{},
	})

	require.NotNil(t, sections, "Expected non-nil sections")

	// Find the opening <safe-output-tools> section
	var openingSection *PromptSection
	for i := range sections {
		if !sections[i].IsFile && strings.HasPrefix(sections[i].Content, "<safe-output-tools>") {
			openingSection = &sections[i]
			break
		}
	}
	require.NotNil(t, openingSection, "Expected to find <safe-output-tools> opening section")

	// The raw ${{ }} expression must NOT appear in the content (would hit the 21KB limit)
	assert.NotContains(t, openingSection.Content, "${{",
		"${{ }} expressions must not appear in the tools content (triggers 21KB expression-size limit)")

	// A __GH_AW_*__ placeholder must appear instead
	assert.Contains(t, openingSection.Content, "__GH_AW_",
		"A __GH_AW_*__ placeholder should replace the ${{ }} expression")

	// The EnvVars map must have an entry mapping the placeholder key to the original expression
	require.NotEmpty(t, openingSection.EnvVars,
		"EnvVars must be populated so the substitution step can resolve the placeholder")

	var foundExpr bool
	for _, v := range openingSection.EnvVars {
		if v == "${{ inputs.review-comment-max }}" {
			foundExpr = true
			break
		}
	}
	assert.True(t, foundExpr, "EnvVars must contain the original ${{ inputs.review-comment-max }} expression")
}

func TestBuildSafeOutputsSections_IncludesCommentMemoryPromptFile(t *testing.T) {
	sections := buildSafeOutputsSections(&SafeOutputsConfig{
		CommentMemory: &CommentMemoryConfig{},
		NoOp:          &NoOpConfig{},
	})

	require.NotNil(t, sections, "Expected non-nil sections")

	found := false
	for _, section := range sections {
		if section.IsFile && section.Content == safeOutputsCommentMemoryFile {
			found = true
			break
		}
	}

	assert.True(t, found, "Expected comment-memory guidance file to be included when comment_memory is enabled")

	actualToolNames := extractToolNamesFromSections(t, sections)
	assert.NotContains(t, actualToolNames, "comment_memory", "comment_memory should not be exposed as an agent tool when file-based sync is enabled")
}

// the list of tool names in the order they appear, stripping any max-budget annotations
// (e.g. "noop(max:5)" → "noop").
func extractToolNamesFromSections(t *testing.T, sections []PromptSection) []string {
	t.Helper()

	var toolsLine string
	for _, section := range sections {
		if !section.IsFile && strings.HasPrefix(section.Content, "<safe-output-tools>") {
			toolsLine = section.Content
			break
		}
	}

	require.NotEmpty(t, toolsLine, "Expected to find <safe-output-tools> opening section")

	lines := strings.Split(toolsLine, "\n")
	require.GreaterOrEqual(t, len(lines), 2, "Expected at least two lines in tools section")

	toolsListLine := lines[1]
	require.True(t, strings.HasPrefix(toolsListLine, "Tools: "),
		"Second line should start with 'Tools: ', got: %q", toolsListLine)

	toolsList := strings.TrimPrefix(toolsListLine, "Tools: ")
	toolEntries := strings.Split(toolsList, ", ")

	names := make([]string, 0, len(toolEntries))
	for _, entry := range toolEntries {
		// Strip optional budget annotation: "noop(max:5)" → "noop"
		if name, _, found := strings.Cut(entry, "("); found {
			names = append(names, name)
		} else {
			names = append(names, entry)
		}
	}
	return names
}
