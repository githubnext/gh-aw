//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractPluginsFromFrontmatter tests parsing of imports.plugins from frontmatter.
func TestExtractPluginsFromFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    []string
		wantErr     bool
	}{
		{
			name:        "no imports field returns nil",
			frontmatter: map[string]any{},
			expected:    nil,
		},
		{
			name:        "imports is not a map returns nil",
			frontmatter: map[string]any{"imports": []any{"some.md"}},
			expected:    nil,
		},
		{
			name: "no plugins subfield returns nil",
			frontmatter: map[string]any{
				"imports": map[string]any{"apm-packages": []any{"org/pkg"}},
			},
			expected: nil,
		},
		{
			name: "array form extracts plugin names",
			frontmatter: map[string]any{
				"imports": map[string]any{
					"plugins": []any{"my-plugin", "another-plugin"},
				},
			},
			expected: []string{"my-plugin", "another-plugin"},
		},
		{
			name: "empty strings in array are skipped",
			frontmatter: map[string]any{
				"imports": map[string]any{
					"plugins": []any{"my-plugin", "", "other-plugin"},
				},
			},
			expected: []string{"my-plugin", "other-plugin"},
		},
		{
			name: "invalid type returns error",
			frontmatter: map[string]any{
				"imports": map[string]any{
					"plugins": 42,
				},
			},
			wantErr: true,
		},
		{
			name: "coexists with aw and apm-packages",
			frontmatter: map[string]any{
				"imports": map[string]any{
					"aw":           []any{"shared/tools.md"},
					"apm-packages": []any{"org/pkg"},
					"plugins":      []any{"my-plugin"},
				},
			},
			expected: []string{"my-plugin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractPluginsFromFrontmatter(tt.frontmatter)
			if tt.wantErr {
				assert.Error(t, err, "Expected error for invalid plugins field")
				return
			}
			require.NoError(t, err, "Should not error for valid input")
			assert.Equal(t, tt.expected, result, "Plugin names should match")
		})
	}
}

// TestCopilotEngineGetPluginInstallSteps tests the Copilot engine's plugin install step generation.
func TestCopilotEngineGetPluginInstallSteps(t *testing.T) {
	engine := NewCopilotEngine()

	t.Run("empty list returns nil", func(t *testing.T) {
		steps := engine.GetPluginInstallSteps(nil, &WorkflowData{})
		assert.Nil(t, steps, "Should return nil for empty plugins")
	})

	t.Run("single plugin emits one step", func(t *testing.T) {
		steps := engine.GetPluginInstallSteps([]string{"my-extension"}, &WorkflowData{})
		require.Len(t, steps, 1, "Should emit one step per plugin")
		step := steps[0]
		found := false
		for _, line := range step {
			if line == "        run: copilot plugin install my-extension" {
				found = true
			}
		}
		assert.True(t, found, "Step should contain copilot plugin install command")
		// Verify GITHUB_TOKEN is set
		hasToken := false
		for _, line := range step {
			if line == "          GITHUB_TOKEN: ${{ github.token }}" {
				hasToken = true
			}
		}
		assert.True(t, hasToken, "Step should set GITHUB_TOKEN from github.token")
	})

	t.Run("multiple plugins emit one step each", func(t *testing.T) {
		steps := engine.GetPluginInstallSteps([]string{"plugin-a", "plugin-b", "plugin-c"}, &WorkflowData{})
		assert.Len(t, steps, 3, "Should emit one step per plugin name")
	})

	t.Run("OWNER/REPO:PATH spec is passed through", func(t *testing.T) {
		spec := "github/copilot-plugins:plugins/advanced-security/skills/secret-scanning"
		steps := engine.GetPluginInstallSteps([]string{spec}, &WorkflowData{})
		require.Len(t, steps, 1, "Should emit one step")
		found := false
		for _, line := range steps[0] {
			if line == "        run: copilot plugin install "+spec {
				found = true
			}
		}
		assert.True(t, found, "Step should contain the OWNER/REPO:PATH spec verbatim")
	})
}

// TestClaudeEngineGetPluginInstallSteps tests the Claude engine's plugin install step generation.
func TestClaudeEngineGetPluginInstallSteps(t *testing.T) {
	engine := NewClaudeEngine()

	t.Run("empty list returns nil", func(t *testing.T) {
		steps := engine.GetPluginInstallSteps(nil, &WorkflowData{})
		assert.Nil(t, steps, "Should return nil for empty plugins")
	})

	t.Run("single plugin emits one step", func(t *testing.T) {
		steps := engine.GetPluginInstallSteps([]string{"my-plugin"}, &WorkflowData{})
		require.Len(t, steps, 1, "Should emit one step per plugin")
		step := steps[0]
		found := false
		for _, line := range step {
			if line == "        run: claude plugin install my-plugin" {
				found = true
			}
		}
		assert.True(t, found, "Step should contain claude plugin install command")
	})
}

// TestImportsProviderInterface verifies that Copilot and Claude engines implement ImportsProvider.
func TestImportsProviderInterface(t *testing.T) {
	t.Run("CopilotEngine implements ImportsProvider", func(t *testing.T) {
		engine := NewCopilotEngine()
		_, ok := any(engine).(ImportsProvider)
		assert.True(t, ok, "CopilotEngine should implement ImportsProvider")
	})

	t.Run("ClaudeEngine implements ImportsProvider", func(t *testing.T) {
		engine := NewClaudeEngine()
		_, ok := any(engine).(ImportsProvider)
		assert.True(t, ok, "ClaudeEngine should implement ImportsProvider")
	})

	// Codex and Gemini CLIs do not have native plugin CLI commands,
	// so they intentionally do not implement ImportsProvider.
	t.Run("CodexEngine does not implement ImportsProvider", func(t *testing.T) {
		engine := NewCodexEngine()
		_, ok := any(engine).(ImportsProvider)
		assert.False(t, ok, "CodexEngine should not implement ImportsProvider (no native plugin CLI support)")
	})

	t.Run("GeminiEngine does not implement ImportsProvider", func(t *testing.T) {
		engine := NewGeminiEngine()
		_, ok := any(engine).(ImportsProvider)
		assert.False(t, ok, "GeminiEngine should not implement ImportsProvider (no native plugin CLI support)")
	})
}

// TestCompileWorkflow_Plugins tests that imports.plugins is compiled into install steps
// before agent execution.
func TestCompileWorkflow_Plugins(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compile-imports-plugins-test*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	workflowContent := `---
name: Test Plugins
on:
  workflow_dispatch:
engine: copilot
imports:
  plugins:
    - my-extension
---

# Test Plugins

Install a plugin before the agent runs.
`
	workflowFile := filepath.Join(workflowsDir, "test-workflow.md")
	err = os.WriteFile(workflowFile, []byte(workflowContent), 0600)
	require.NoError(t, err, "Failed to write test workflow")

	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	defer os.Chdir(originalDir) //nolint:errcheck

	err = os.Chdir(tmpDir)
	require.NoError(t, err, "Failed to change to temp directory")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(workflowFile)
	require.NoError(t, err, "Failed to compile workflow")

	lockFile := strings.Replace(workflowFile, ".md", ".lock.yml", 1)
	yamlOutput, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")
	require.NotEmpty(t, yamlOutput, "Compiled YAML should not be empty")

	yamlStr := string(yamlOutput)

	assert.Contains(t, yamlStr, "copilot plugin install my-extension",
		"Should contain plugin installation step")
	assert.Contains(t, yamlStr, "GITHUB_TOKEN: ${{ github.token }}",
		"Plugin step should include GITHUB_TOKEN from github.token")

	// Step should appear before the agent execution step
	pluginIdx := strings.Index(yamlStr, "copilot plugin install")
	agentIdx := strings.Index(yamlStr, "copilot --add-dir")
	if agentIdx == -1 {
		agentIdx = strings.Index(yamlStr, "copilot --")
	}
	require.NotEqual(t, -1, agentIdx, "Agent execution step should be present")
	assert.Less(t, pluginIdx, agentIdx,
		"Plugin installation step should appear before agent execution")
}

// TestCompileWorkflow_PluginsFromSharedImport verifies that imports.plugins defined in a
// shared workflow file are merged into the main workflow's Plugins.
func TestCompileWorkflow_PluginsFromSharedImport(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compile-shared-imports-test*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Shared workflow that defines plugins
	sharedContent := `---
imports:
  plugins:
    - shared-plugin
---
`
	sharedFile := filepath.Join(workflowsDir, "shared-tools.md")
	err = os.WriteFile(sharedFile, []byte(sharedContent), 0600)
	require.NoError(t, err, "Failed to write shared workflow")

	// Main workflow imports the shared file and adds its own plugin
	workflowContent := `---
name: Test Shared Imports Merge
on:
  workflow_dispatch:
engine: copilot
imports:
  aw:
    - shared-tools.md
  plugins:
    - main-plugin
---

# Test Shared Imports Merge

Verify that plugins from imported workflows are merged.
`
	workflowFile := filepath.Join(workflowsDir, "test-workflow.md")
	err = os.WriteFile(workflowFile, []byte(workflowContent), 0600)
	require.NoError(t, err, "Failed to write test workflow")

	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	defer os.Chdir(originalDir) //nolint:errcheck

	err = os.Chdir(tmpDir)
	require.NoError(t, err, "Failed to change to temp directory")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(workflowFile)
	require.NoError(t, err, "Failed to compile workflow")

	lockFile := strings.Replace(workflowFile, ".md", ".lock.yml", 1)
	yamlOutput, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	yamlStr := string(yamlOutput)

	assert.Contains(t, yamlStr, "copilot plugin install main-plugin",
		"Main workflow plugin should be present")
	assert.Contains(t, yamlStr, "copilot plugin install shared-plugin",
		"Shared workflow plugin should be merged in")
}

// TestCompileWorkflow_PluginsDeduplication verifies that importing the same plugin name
// from multiple shared workflows does not produce duplicate install steps.
func TestCompileWorkflow_PluginsDeduplication(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compile-dedup-test*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	// Two shared files that both declare the same plugin
	for _, name := range []string{"shared-a.md", "shared-b.md"} {
		content := `---
imports:
  plugins:
    - common-plugin
---
`
		require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, name), []byte(content), 0600))
	}

	workflowContent := `---
name: Test Dedup
on:
  workflow_dispatch:
engine: copilot
imports:
  aw:
    - shared-a.md
    - shared-b.md
---

# Test Dedup
`
	workflowFile := filepath.Join(workflowsDir, "test-workflow.md")
	require.NoError(t, os.WriteFile(workflowFile, []byte(workflowContent), 0600))

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir) //nolint:errcheck
	require.NoError(t, os.Chdir(tmpDir))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowFile))

	lockFile := strings.Replace(workflowFile, ".md", ".lock.yml", 1)
	yamlOutput, err := os.ReadFile(lockFile)
	require.NoError(t, err)
	yamlStr := string(yamlOutput)

	// Should appear exactly once despite being defined in two shared imports
	count := strings.Count(yamlStr, "copilot plugin install common-plugin")
	assert.Equal(t, 1, count, "Duplicate plugin name should appear only once")
}

// TestValidateImportsProviderSupport tests that engines without ImportsProvider support
// return an error when plugins are specified.
func TestValidateImportsProviderSupport(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name        string
		engine      CodingAgentEngine
		plugins     []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "copilot with plugin - no error",
			engine:  &CopilotEngine{BaseEngine: BaseEngine{id: "copilot", displayName: "GitHub Copilot"}},
			plugins: []string{"my-plugin"},
			wantErr: false,
		},
		{
			name:    "claude with plugin - no error",
			engine:  &ClaudeEngine{BaseEngine: BaseEngine{id: "claude", displayName: "Claude"}},
			plugins: []string{"my-plugin"},
			wantErr: false,
		},
		{
			name:        "codex with plugin - error",
			engine:      &CodexEngine{BaseEngine: BaseEngine{id: "codex", displayName: "Codex"}},
			plugins:     []string{"my-plugin"},
			wantErr:     true,
			errContains: "imports.plugins",
		},
		{
			name:        "gemini with plugin - error",
			engine:      &GeminiEngine{BaseEngine: BaseEngine{id: "gemini", displayName: "Gemini"}},
			plugins:     []string{"my-plugin"},
			wantErr:     true,
			errContains: "imports.plugins",
		},
		{
			name:    "any engine with empty list - no error",
			engine:  &CodexEngine{BaseEngine: BaseEngine{id: "codex", displayName: "Codex"}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := compiler.validateImportsProviderSupport(tt.plugins, tt.engine)
			if tt.wantErr {
				require.Error(t, err, "Expected error for engine %s", tt.engine.GetID())
				assert.Contains(t, err.Error(), tt.errContains, "Error message should mention the unsupported field")
				assert.Contains(t, err.Error(), tt.engine.GetID(), "Error message should mention the engine ID")
			} else {
				assert.NoError(t, err, "Expected no error for engine %s", tt.engine.GetID())
			}
		})
	}
}

// TestCompileWorkflow_PluginsWithUnsupportedEngine verifies that using imports.plugins
// with an engine that does not implement ImportsProvider fails with a clear error.
func TestCompileWorkflow_PluginsWithUnsupportedEngine(t *testing.T) {
	tmpDir := t.TempDir()
	workflowFile := filepath.Join(tmpDir, ".github", "workflows", "test-workflow.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(workflowFile), 0755))

	workflowContent := `---
on: issues
permissions:
  contents: read
  issues: read
engine: gemini
imports:
  plugins:
    - my-plugin
---

# Test Gemini With Plugin

Process the issue.
`
	require.NoError(t, os.WriteFile(workflowFile, []byte(workflowContent), 0600))

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir) //nolint:errcheck
	require.NoError(t, os.Chdir(tmpDir))

	compiler := NewCompiler()
	err := compiler.CompileWorkflow(workflowFile)
	require.Error(t, err, "Should fail when plugins are used with gemini engine")
	assert.Contains(t, err.Error(), "imports.plugins", "Error should mention the unsupported field")
}
