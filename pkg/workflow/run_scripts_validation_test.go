//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRunScripts(t *testing.T) {
	tests := []struct {
		name              string
		frontmatter       map[string]any
		runtimes          map[string]any
		mergedRunScripts  bool
		expectedRunScript bool
	}{
		{
			name:              "default: run_scripts is false",
			frontmatter:       map[string]any{},
			runtimes:          map[string]any{},
			mergedRunScripts:  false,
			expectedRunScript: false,
		},
		{
			name: "global run-scripts: true",
			frontmatter: map[string]any{
				"run-scripts": true,
			},
			runtimes:          map[string]any{},
			mergedRunScripts:  false,
			expectedRunScript: true,
		},
		{
			name:        "global run-scripts: false",
			frontmatter: map[string]any{"run-scripts": false},
			runtimes:    map[string]any{},

			mergedRunScripts:  false,
			expectedRunScript: false,
		},
		{
			name:        "per-runtime node run-scripts: true",
			frontmatter: map[string]any{},
			runtimes: map[string]any{
				"node": map[string]any{
					"run-scripts": true,
				},
			},
			mergedRunScripts:  false,
			expectedRunScript: true,
		},
		{
			name:        "per-runtime python run-scripts: true (no effect for npm installs)",
			frontmatter: map[string]any{},
			runtimes: map[string]any{
				"python": map[string]any{
					"run-scripts": true,
				},
			},
			mergedRunScripts:  false,
			expectedRunScript: false,
		},
		{
			name:              "merged run-scripts from imported shared workflow",
			frontmatter:       map[string]any{},
			runtimes:          map[string]any{},
			mergedRunScripts:  true,
			expectedRunScript: true,
		},
		{
			name: "merged run-scripts OR global: both true",
			frontmatter: map[string]any{
				"run-scripts": true,
			},
			runtimes:          map[string]any{},
			mergedRunScripts:  true,
			expectedRunScript: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveRunScripts(tt.frontmatter, tt.runtimes, tt.mergedRunScripts)
			assert.Equal(t, tt.expectedRunScript, result, "resolveRunScripts() result mismatch")
		})
	}
}

func TestGenerateNpmInstallSteps_IgnoreScriptsByDefault(t *testing.T) {
	steps := GenerateNpmInstallSteps(
		"@anthropic-ai/claude-code",
		"latest",
		"Install Claude Code CLI",
		"claude",
		false, // no Node.js setup step
		false, // runScripts=false (default)
	)

	require.Len(t, steps, 1, "Expected 1 install step")
	installStep := strings.Join([]string(steps[0]), "\n")

	assert.Contains(t, installStep, "--ignore-scripts", "Expected --ignore-scripts flag by default")
	assert.Contains(t, installStep, "npm install --ignore-scripts -g @anthropic-ai/claude-code@latest")
}

func TestGenerateNpmInstallSteps_RunScriptsEnabled(t *testing.T) {
	steps := GenerateNpmInstallSteps(
		"@anthropic-ai/claude-code",
		"latest",
		"Install Claude Code CLI",
		"claude",
		false, // no Node.js setup step
		true,  // runScripts=true
	)

	require.Len(t, steps, 1, "Expected 1 install step")
	installStep := strings.Join([]string(steps[0]), "\n")

	assert.NotContains(t, installStep, "--ignore-scripts", "Expected no --ignore-scripts flag when runScripts=true")
	assert.Contains(t, installStep, "npm install -g @anthropic-ai/claude-code@latest")
}

func TestGenerateNpmInstallStepsWithScope_LocalInstall(t *testing.T) {
	steps := GenerateNpmInstallStepsWithScope(
		"@tobilu/qmd",
		"2.0.1",
		"Install qmd",
		"qmd",
		false, // no Node.js setup
		false, // local install (not global)
		false, // runScripts=false
	)

	require.Len(t, steps, 1, "Expected 1 install step")
	installStep := strings.Join([]string(steps[0]), "\n")

	assert.Contains(t, installStep, "--ignore-scripts", "Expected --ignore-scripts flag")
	assert.NotContains(t, installStep, " -g ", "Expected local install (no -g flag)")
}

func TestValidateRunScripts_Warning(t *testing.T) {
	c := &Compiler{strictMode: false}
	c.ResetWarningCount()

	workflowData := &WorkflowData{RunScripts: true}
	err := c.validateRunScripts(workflowData)

	require.NoError(t, err, "Should not return error in non-strict mode")
	assert.Equal(t, 1, c.GetWarningCount(), "Should increment warning count")
}

func TestValidateRunScripts_StrictModeError(t *testing.T) {
	c := &Compiler{strictMode: true}

	workflowData := &WorkflowData{RunScripts: true}
	err := c.validateRunScripts(workflowData)

	require.Error(t, err, "Should return error in strict mode")
	assert.Contains(t, err.Error(), "strict mode", "Error should mention strict mode")
	assert.Contains(t, err.Error(), "supply chain", "Error should mention supply chain risk")
}

func TestValidateRunScripts_NotSet(t *testing.T) {
	c := &Compiler{strictMode: false}
	c.ResetWarningCount()

	workflowData := &WorkflowData{RunScripts: false}
	err := c.validateRunScripts(workflowData)

	require.NoError(t, err, "Should not return error when run-scripts is not set")
	assert.Equal(t, 0, c.GetWarningCount(), "Should not increment warning count")
}

func TestFrontmatterConfig_RunScripts(t *testing.T) {
	frontmatter := map[string]any{
		"run-scripts": true,
		"engine":      "claude",
	}

	config, err := ParseFrontmatterConfig(frontmatter)
	require.NoError(t, err, "Should parse frontmatter without error")
	require.NotNil(t, config, "Config should not be nil")

	require.NotNil(t, config.RunScripts, "RunScripts should be set")
	assert.True(t, *config.RunScripts, "RunScripts should be true")
}

func TestRuntimeConfig_RunScripts(t *testing.T) {
	runtimes := map[string]any{
		"node": map[string]any{
			"version":     "20",
			"run-scripts": true,
		},
	}

	config, err := parseRuntimesConfig(runtimes)
	require.NoError(t, err, "Should parse runtimes config without error")
	require.NotNil(t, config.Node, "Node config should be set")
	require.NotNil(t, config.Node.RunScripts, "Node RunScripts should be set")
	assert.True(t, *config.Node.RunScripts, "Node RunScripts should be true")
}
