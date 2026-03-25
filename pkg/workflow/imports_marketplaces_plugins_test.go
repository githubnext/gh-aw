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

// TestExtractMarketplacesFromFrontmatter tests parsing of imports.marketplaces from frontmatter.
func TestExtractMarketplacesFromFrontmatter(t *testing.T) {
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
			name: "no marketplaces subfield returns nil",
			frontmatter: map[string]any{
				"imports": map[string]any{"apm-packages": []any{"org/pkg"}},
			},
			expected: nil,
		},
		{
			name: "array form extracts marketplace URLs",
			frontmatter: map[string]any{
				"imports": map[string]any{
					"marketplaces": []any{
						"https://marketplace.example.com",
						"https://another.example.com",
					},
				},
			},
			expected: []string{
				"https://marketplace.example.com",
				"https://another.example.com",
			},
		},
		{
			name: "empty strings in array are skipped",
			frontmatter: map[string]any{
				"imports": map[string]any{
					"marketplaces": []any{"https://ok.example.com", "", "https://also-ok.example.com"},
				},
			},
			expected: []string{"https://ok.example.com", "https://also-ok.example.com"},
		},
		{
			name: "invalid type returns error",
			frontmatter: map[string]any{
				"imports": map[string]any{
					"marketplaces": "not-an-array",
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
					"marketplaces": []any{"https://market.example.com"},
				},
			},
			expected: []string{"https://market.example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractMarketplacesFromFrontmatter(tt.frontmatter)
			if tt.wantErr {
				assert.Error(t, err, "Expected error for invalid marketplaces field")
				return
			}
			require.NoError(t, err, "Should not error for valid input")
			assert.Equal(t, tt.expected, result, "Marketplace URLs should match")
		})
	}
}

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
			name: "coexists with aw, apm-packages, and marketplaces",
			frontmatter: map[string]any{
				"imports": map[string]any{
					"aw":           []any{"shared/tools.md"},
					"apm-packages": []any{"org/pkg"},
					"marketplaces": []any{"https://market.example.com"},
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

// TestCopilotEngineGetMarketplaceSetupSteps tests the Copilot engine's marketplace setup step generation.
func TestCopilotEngineGetMarketplaceSetupSteps(t *testing.T) {
	engine := NewCopilotEngine()

	t.Run("empty list returns nil", func(t *testing.T) {
		steps := engine.GetMarketplaceSetupSteps(nil, &WorkflowData{})
		assert.Nil(t, steps, "Should return nil for empty marketplaces")
	})

	t.Run("single marketplace emits one step", func(t *testing.T) {
		steps := engine.GetMarketplaceSetupSteps([]string{"https://market.example.com"}, &WorkflowData{})
		require.Len(t, steps, 1, "Should emit one step per marketplace")
		step := steps[0]
		found := false
		for _, line := range step {
			if line == "        run: copilot marketplace add https://market.example.com" {
				found = true
			}
		}
		assert.True(t, found, "Step should contain copilot marketplace add command")
		// Verify GITHUB_TOKEN is set
		hasToken := false
		for _, line := range step {
			if line == "          GITHUB_TOKEN: ${{ github.token }}" {
				hasToken = true
			}
		}
		assert.True(t, hasToken, "Step should set GITHUB_TOKEN from github.token")
	})

	t.Run("multiple marketplaces emit one step each", func(t *testing.T) {
		steps := engine.GetMarketplaceSetupSteps([]string{"https://a.example.com", "https://b.example.com"}, &WorkflowData{})
		assert.Len(t, steps, 2, "Should emit one step per marketplace URL")
	})
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
			if line == "        run: copilot extension install my-extension" {
				found = true
			}
		}
		assert.True(t, found, "Step should contain copilot extension install command")
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
}

// TestClaudeEngineGetMarketplaceSetupSteps tests the Claude engine's marketplace setup step generation.
func TestClaudeEngineGetMarketplaceSetupSteps(t *testing.T) {
	engine := NewClaudeEngine()

	t.Run("empty list returns nil", func(t *testing.T) {
		steps := engine.GetMarketplaceSetupSteps(nil, &WorkflowData{})
		assert.Nil(t, steps, "Should return nil for empty marketplaces")
	})

	t.Run("single marketplace emits one step", func(t *testing.T) {
		steps := engine.GetMarketplaceSetupSteps([]string{"https://market.example.com"}, &WorkflowData{})
		require.Len(t, steps, 1, "Should emit one step per marketplace")
		step := steps[0]
		found := false
		for _, line := range step {
			if line == "        run: claude marketplace add https://market.example.com" {
				found = true
			}
		}
		assert.True(t, found, "Step should contain claude marketplace add command")
		// Verify GITHUB_TOKEN is set
		hasToken := false
		for _, line := range step {
			if line == "          GITHUB_TOKEN: ${{ github.token }}" {
				hasToken = true
			}
		}
		assert.True(t, hasToken, "Step should set GITHUB_TOKEN from github.token")
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
}

// TestCompileWorkflow_MarketplacesAndPlugins tests that imports.marketplaces and
// imports.plugins are compiled into setup steps before agent execution.
func TestCompileWorkflow_MarketplacesAndPlugins(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compile-imports-marketplaces-plugins-test*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	workflowContent := `---
name: Test Marketplaces and Plugins
on:
  workflow_dispatch:
engine: copilot
imports:
  marketplaces:
    - https://marketplace.example.com
  plugins:
    - my-extension
---

# Test Marketplaces and Plugins

Register a marketplace and install a plugin before the agent runs.
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

	// Marketplace and plugin steps should appear before agent execution
	assert.Contains(t, yamlStr, "copilot marketplace add https://marketplace.example.com",
		"Should contain marketplace registration step")
	assert.Contains(t, yamlStr, "copilot extension install my-extension",
		"Should contain plugin installation step")
	assert.Contains(t, yamlStr, "GITHUB_TOKEN: ${{ github.token }}",
		"Marketplace/plugin steps should include GITHUB_TOKEN from github.token")

	// Steps should appear before the agent execution step
	marketplaceIdx := strings.Index(yamlStr, "copilot marketplace add")
	pluginIdx := strings.Index(yamlStr, "copilot extension install")
	agentIdx := strings.Index(yamlStr, "copilot --add-dir")
	if agentIdx == -1 {
		agentIdx = strings.Index(yamlStr, "copilot --")
	}
	require.Positive(t, agentIdx, "Agent execution step should be present")
	assert.Less(t, marketplaceIdx, agentIdx,
		"Marketplace registration step should appear before agent execution")
	assert.Less(t, pluginIdx, agentIdx,
		"Plugin installation step should appear before agent execution")
}
