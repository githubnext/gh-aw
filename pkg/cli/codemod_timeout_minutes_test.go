//go:build !integration

package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTimeoutMinutesCodemod(t *testing.T) {
	t.Parallel()
	codemod := getTimeoutMinutesCodemod()

	// Verify codemod metadata
	assert.Equal(t, "timeout-minutes-migration", codemod.ID, "Codemod ID should match")
	assert.Equal(t, "Migrate timeout_minutes to timeout-minutes", codemod.Name, "Codemod name should match")
	assert.NotEmpty(t, codemod.Description, "Codemod should have a description")
	assert.Equal(t, "0.1.0", codemod.IntroducedIn, "Codemod version should match")
	require.NotNil(t, codemod.Apply, "Codemod should have an Apply function")
}

func TestTimeoutMinutesCodemod_BasicMigration(t *testing.T) {
	t.Parallel()
	codemod := getTimeoutMinutesCodemod()

	content := `---
on: workflow_dispatch
timeout_minutes: 30
permissions:
  contents: read
---

# Test Workflow`

	frontmatter := map[string]any{
		"on":              "workflow_dispatch",
		"timeout_minutes": 30,
		"permissions": map[string]any{
			"contents": "read",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	assert.Contains(t, result, "timeout-minutes: 30", "Result should contain new field name")
	assert.NotContains(t, result, "timeout_minutes:", "Result should not contain old field name")
}

func TestTimeoutMinutesCodemod_PreservesIndentation(t *testing.T) {
	t.Parallel()
	codemod := getTimeoutMinutesCodemod()

	content := `---
on: workflow_dispatch
timeout_minutes: 45
---

# Test`

	frontmatter := map[string]any{
		"on":              "workflow_dispatch",
		"timeout_minutes": 45,
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	assert.Contains(t, result, "timeout-minutes: 45", "Result should contain migrated field")
}

func TestTimeoutMinutesCodemod_PreservesComments(t *testing.T) {
	t.Parallel()
	codemod := getTimeoutMinutesCodemod()

	content := `---
on: workflow_dispatch
timeout_minutes: 30  # 30 minutes should be enough
permissions:
  contents: read
---

# Test Workflow`

	frontmatter := map[string]any{
		"on":              "workflow_dispatch",
		"timeout_minutes": 30,
		"permissions": map[string]any{
			"contents": "read",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	assert.Contains(t, result, "timeout-minutes: 30  # 30 minutes should be enough", "Result should preserve inline comment")
	assert.NotContains(t, result, "timeout_minutes:", "Result should not contain old field name")
}

func TestTimeoutMinutesCodemod_NoFieldPresent(t *testing.T) {
	t.Parallel()
	codemod := getTimeoutMinutesCodemod()

	content := `---
on: workflow_dispatch
timeout-minutes: 30
permissions:
  contents: read
---

# Test Workflow`

	frontmatter := map[string]any{
		"on":              "workflow_dispatch",
		"timeout-minutes": 30,
		"permissions": map[string]any{
			"contents": "read",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.False(t, applied, "Codemod should not report changes when field is not present")
	assert.Equal(t, content, result, "Content should remain unchanged")
}

func TestTimeoutMinutesCodemod_PreservesMarkdownBody(t *testing.T) {
	t.Parallel()
	codemod := getTimeoutMinutesCodemod()

	content := `---
on: workflow_dispatch
timeout_minutes: 60
---

# Test Workflow

This is a test workflow with:
- Multiple lines
- Markdown formatting
- Code blocks

` + "```yaml" + `
key: value
` + "```"

	frontmatter := map[string]any{
		"on":              "workflow_dispatch",
		"timeout_minutes": 60,
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	assert.Contains(t, result, "timeout-minutes: 60", "Result should contain new field name")
	assert.Contains(t, result, "# Test Workflow", "Result should preserve markdown body")
	assert.Contains(t, result, "- Multiple lines", "Result should preserve markdown content")
	assert.Contains(t, result, "```yaml", "Result should preserve code blocks")
}

func TestTimeoutMinutesCodemod_DifferentValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		value int
	}{
		{"small timeout", 5},
		{"medium timeout", 30},
		{"large timeout", 120},
		{"very large timeout", 360},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codemod := getTimeoutMinutesCodemod()

			content := fmt.Sprintf(`---
on: workflow_dispatch
timeout_minutes: %d
---

# Test`, tt.value)

			frontmatter := map[string]any{
				"on":              "workflow_dispatch",
				"timeout_minutes": tt.value,
			}

			result, applied, err := codemod.Apply(content, frontmatter)

			require.NoError(t, err, "Apply should not return an error")
			assert.True(t, applied, "Codemod should report changes")
			assert.Contains(t, result, fmt.Sprintf("timeout-minutes: %d", tt.value), "Result should contain new field with correct value")
			assert.NotContains(t, result, "timeout_minutes:", "Result should not contain old field name")
		})
	}
}

func TestTimeoutMinutesCodemod_OnlyReplacesExactMatch(t *testing.T) {
	t.Parallel()
	codemod := getTimeoutMinutesCodemod()

	content := `---
on: workflow_dispatch
timeout_minutes: 30
custom_timeout_minutes: 60
---

# Test`

	frontmatter := map[string]any{
		"on":                     "workflow_dispatch",
		"timeout_minutes":        30,
		"custom_timeout_minutes": 60,
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")

	lines := strings.Split(result, "\n")
	foundTimeoutMinutes := false
	foundCustomTimeoutMinutes := false

	for _, line := range lines {
		if strings.Contains(line, "timeout-minutes: 30") {
			foundTimeoutMinutes = true
		}
		if strings.Contains(line, "custom_timeout_minutes: 60") {
			foundCustomTimeoutMinutes = true
		}
	}

	assert.True(t, foundTimeoutMinutes, "Should replace timeout_minutes")
	assert.True(t, foundCustomTimeoutMinutes, "Should not replace custom_timeout_minutes")
}

func TestTimeoutMinutesCodemod_DoesNotRenameNestedKeys(t *testing.T) {
	t.Parallel()
	codemod := getTimeoutMinutesCodemod()

	content := `---
on: workflow_dispatch
timeout_minutes: 30
jobs:
  setup:
    timeout_minutes: 45
---

# Test`

	frontmatter := map[string]any{
		"on":              "workflow_dispatch",
		"timeout_minutes": 30,
		"jobs": map[string]any{
			"setup": map[string]any{
				"timeout_minutes": 45,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "Apply should not return an error")
	assert.True(t, applied, "Codemod should report changes")
	assert.Equal(t, 1, strings.Count(result, "timeout-minutes:"), "Only the top-level workflow field should be renamed")
	assert.Contains(t, result, "    timeout_minutes: 45", "Nested keys should remain unchanged")
}

func TestFrontmatterRenameCodemods_AWAST(t *testing.T) {
	t.Parallel()
	tests := []struct {
		codemod Codemod
		input   string
		want    string
	}{
		{getTimeoutMinutesCodemod(),
			`{"timeout_minutes": 30, nested: {timeout_minutes: 45}} # keep`,
			`{"timeout-minutes": 30, nested: {timeout_minutes: 45}} # keep`},
		{getGitHubReposToAllowedReposCodemod(),
			`tools: {"github": {'repos': [org/repo], nested: {repos: all}}} # keep`,
			`tools: {"github": {'allowed-repos': [org/repo], nested: {repos: all}}} # keep`},
		{getToolsetSingularToToolsetsCodemod(),
			`tools: {'github': {"toolset": [issues], nested: {toolset: [pull_requests]}}} # keep`,
			`tools: {'github': {"toolsets": [issues], nested: {toolset: [pull_requests]}}} # keep`},
		{getEngineCopilotSDKDriverToDriverCodemod(),
			`"engine": {'copilot-sdk-driver': sdk, nested: {copilot-sdk-driver: untouched}} # keep`,
			`"engine": {'driver': sdk, nested: {copilot-sdk-driver: untouched}} # keep`},
		{getMaxRunsToMaxTurnsCodemod(),
			`{'max-runs': 20, nested: {max-runs: 30}} # keep`,
			`{'max-turns': 20, nested: {max-runs: 30}} # keep`},
		{getGitHubAppCodemod(),
			`{"app": {app-id: 1}, "tools": {github: {'app': null, nested: {app: untouched}}}, safe-outputs: {app: {}}, checkout: {app: {}}} # keep`,
			`{"github-app": {app-id: 1}, "tools": {github: {'github-app': null, nested: {app: untouched}}}, safe-outputs: {github-app: {}}, checkout: {github-app: {}}} # keep`},
		{getGitHubAppClientIDCodemod(),
			`"github-app": {'app-id': 1, nested: {app-id: untouched}} # keep`,
			`"github-app": {'client-id': 1, nested: {app-id: untouched}} # keep`},
		{getMCPModeToTypeCodemod(),
			`mcp-servers: {"custom": {'mode': local, nested: {mode: untouched}}, other: {mode: remote}} # keep`,
			`mcp-servers: {"custom": {'type': local, nested: {mode: untouched}}, other: {type: remote}} # keep`},
		{getAssignToAgentDefaultAgentCodemod(),
			`'safe-outputs': {"assign-to-agent": {'default-agent': copilot, nested: {default-agent: untouched}}} # keep`,
			`'safe-outputs': {"assign-to-agent": {'name': copilot, nested: {default-agent: untouched}}} # keep`},
		{getUploadAssetsCodemod(),
			`"safe-outputs": {'upload-assets': {}, nested: {upload-assets: untouched}} # keep`,
			`"safe-outputs": {'upload-asset': {}, nested: {upload-assets: untouched}} # keep`},
		{getSafeInputsToMCPScriptsCodemod(),
			`{'safe-inputs': {}, nested: {safe-inputs: untouched}} # keep`,
			`{'mcp-scripts': {}, nested: {safe-inputs: untouched}} # keep`},
		{getCommandToSlashCommandCodemod(),
			`"on": {'command': test, nested: {command: untouched}} # keep`,
			`"on": {'slash_command': test, nested: {command: untouched}} # keep`},
	}
	for _, tt := range tests {
		t.Run(tt.codemod.ID, func(t *testing.T) {
			t.Parallel()
			body := "\n---\n\n# Body\n\n```yaml\n" + tt.input + "\n```\n"
			content := "---\n" + tt.input + body
			// The authored AST, not a potentially stale caller map, is authoritative.
			result, applied, err := tt.codemod.Apply(content, nil)
			require.NoError(t, err)
			require.True(t, applied)
			assert.Equal(t, "---\n"+tt.want+body, result)

			repeated, applied, err := tt.codemod.Apply(result, map[string]any{"stale": true})
			require.NoError(t, err)
			assert.False(t, applied)
			assert.Equal(t, result, repeated)
		})
	}
}

func TestFrontmatterRenameCodemods_ConservativeConflicts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		codemod Codemod
		input   string
	}{
		{getTimeoutMinutesCodemod(), `{timeout_minutes: 30, timeout-minutes: null}`},
		{getGitHubReposToAllowedReposCodemod(), `tools: {github: {repos: all, allowed-repos: null}}`},
		{getToolsetSingularToToolsetsCodemod(), `tools: {github: {toolset: [issues], toolsets: null}}`},
		{getEngineCopilotSDKDriverToDriverCodemod(), `engine: {copilot-sdk-driver: sdk, driver: null}`},
		{getGitHubAppCodemod(), `{app: {}, github-app: null, tools: {github: {app: {}, github-app: null}}, safe-outputs: {app: {}, github-app: null}, checkout: {app: {}, github-app: null}}`},
		{getGitHubAppClientIDCodemod(), `github-app: {app-id: 1, client-id: null}`},
		{getMCPModeToTypeCodemod(), `mcp-servers: {custom: {mode: local, type: null}}`},
		{getAssignToAgentDefaultAgentCodemod(), `safe-outputs: {assign-to-agent: {default-agent: copilot, name: null}}`},
		{getUploadAssetsCodemod(), `safe-outputs: {upload-assets: {}, upload-asset: null}`},
		{getSafeInputsToMCPScriptsCodemod(), `{safe-inputs: {}, mcp-scripts: null}`},
		{getCommandToSlashCommandCodemod(), `on: {command: test, slash_command: null}`},
	}
	for _, tt := range tests {
		t.Run(tt.codemod.ID, func(t *testing.T) {
			t.Parallel()
			content := "---\n" + tt.input + "\n---\n\n# Body\n"
			result, applied, err := tt.codemod.Apply(content, nil)
			require.NoError(t, err)
			assert.False(t, applied)
			assert.Equal(t, content, result)
		})
	}
}
