//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasBashRestrictedAllowlist(t *testing.T) {
	tests := []struct {
		name     string
		tools    map[string]any
		expected bool
	}{
		{
			name:     "nil tools",
			tools:    nil,
			expected: false,
		},
		{
			name:     "no bash key",
			tools:    map[string]any{"edit": nil},
			expected: false,
		},
		{
			name:     "bash nil (unrestricted)",
			tools:    map[string]any{"bash": nil},
			expected: false,
		},
		{
			name:     "bash empty array",
			tools:    map[string]any{"bash": []any{}},
			expected: false,
		},
		{
			name:     "bash wildcard star",
			tools:    map[string]any{"bash": []any{"*"}},
			expected: false,
		},
		{
			name:     "bash wildcard colon-star",
			tools:    map[string]any{"bash": []any{":*"}},
			expected: false,
		},
		{
			name:     "bash with specific commands",
			tools:    map[string]any{"bash": []any{"echo", "ls"}},
			expected: true,
		},
		{
			name:     "bash with single command",
			tools:    map[string]any{"bash": []any{"git:*"}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasBashRestrictedAllowlist(tt.tools)
			assert.Equal(t, tt.expected, result, "unexpected result for hasBashRestrictedAllowlist")
		})
	}
}

func TestWithMountedCLIShellCommandsInRestrictedBash_PlaywrightCLIMode(t *testing.T) {
	t.Run("playwright cli mode adds playwright-cli:* to restricted bash", func(t *testing.T) {
		workflowData := &WorkflowData{
			Tools: map[string]any{
				"bash": []any{"echo"},
				"playwright": map[string]any{
					"mode": "cli",
				},
			},
		}
		result := withMountedCLIShellCommandsInRestrictedBash(workflowData)
		require.NotNil(t, result, "result should not be nil")
		bash, ok := result["bash"].([]any)
		require.True(t, ok, "bash should be a []any")
		assert.Contains(t, bash, "playwright-cli:*", "playwright-cli:* should be auto-added")
		assert.Contains(t, bash, "echo", "original command should be preserved")
	})

	t.Run("playwright cli mode with unrestricted bash (nil) does not add playwright-cli:*", func(t *testing.T) {
		workflowData := &WorkflowData{
			Tools: map[string]any{
				"bash": nil,
				"playwright": map[string]any{
					"mode": "cli",
				},
			},
		}
		result := withMountedCLIShellCommandsInRestrictedBash(workflowData)
		require.NotNil(t, result, "result should not be nil")
		// bash is nil (unrestricted), so tools map should be unchanged
		bash, hasBash := result["bash"]
		assert.True(t, hasBash, "bash key should still exist")
		assert.Nil(t, bash, "bash should remain nil (unrestricted)")
	})

	t.Run("playwright cli mode with wildcard bash does not add playwright-cli:*", func(t *testing.T) {
		workflowData := &WorkflowData{
			Tools: map[string]any{
				"bash": []any{"*"},
				"playwright": map[string]any{
					"mode": "cli",
				},
			},
		}
		result := withMountedCLIShellCommandsInRestrictedBash(workflowData)
		require.NotNil(t, result, "result should not be nil")
		bash, ok := result["bash"].([]any)
		require.True(t, ok, "bash should be a []any")
		// Wildcard must be preserved and playwright-cli:* must not be injected
		assert.Equal(t, []any{"*"}, bash, "bash should remain exactly [\"*\"] — wildcard preserved and nothing injected")
	})

	t.Run("playwright mcp mode (not cli) does not add playwright-cli:*", func(t *testing.T) {
		workflowData := &WorkflowData{
			Tools: map[string]any{
				"bash":       []any{"echo"},
				"playwright": true,
			},
		}
		result := withMountedCLIShellCommandsInRestrictedBash(workflowData)
		require.NotNil(t, result, "result should not be nil")
		bash, ok := result["bash"].([]any)
		// No servers, no playwright CLI → no changes; bash might be unchanged
		if ok {
			for _, cmd := range bash {
				if cmdStr, ok := cmd.(string); ok {
					assert.NotEqual(t, "playwright-cli:*", cmdStr, "playwright-cli:* should not be injected in MCP mode")
				}
			}
		}
	})

	t.Run("playwright-cli:* not duplicated when already present", func(t *testing.T) {
		workflowData := &WorkflowData{
			Tools: map[string]any{
				"bash": []any{"echo", "playwright-cli:*"},
				"playwright": map[string]any{
					"mode": "cli",
				},
			},
		}
		result := withMountedCLIShellCommandsInRestrictedBash(workflowData)
		require.NotNil(t, result, "result should not be nil")
		bash, ok := result["bash"].([]any)
		require.True(t, ok, "bash should be a []any")
		count := 0
		for _, cmd := range bash {
			if cmdStr, ok := cmd.(string); ok && cmdStr == "playwright-cli:*" {
				count++
			}
		}
		assert.Equal(t, 1, count, "playwright-cli:* should appear exactly once")
	})

	t.Run("github gh-proxy mode adds gh:* to restricted bash", func(t *testing.T) {
		workflowData := &WorkflowData{
			Tools: map[string]any{
				"bash": []any{"echo"},
				"github": map[string]any{
					"mode": "gh-proxy",
				},
			},
		}
		result := withMountedCLIShellCommandsInRestrictedBash(workflowData)
		require.NotNil(t, result, "result should not be nil")
		bash, ok := result["bash"].([]any)
		require.True(t, ok, "bash should be a []any")
		assert.Contains(t, bash, "gh:*", "gh:* should be auto-added in gh-proxy mode")
		assert.Contains(t, bash, "echo", "original command should be preserved")
	})

	t.Run("github local mode does not add gh:* to restricted bash", func(t *testing.T) {
		workflowData := &WorkflowData{
			Tools: map[string]any{
				"bash": []any{"echo"},
				"github": map[string]any{
					"mode": "local",
				},
			},
		}
		result := withMountedCLIShellCommandsInRestrictedBash(workflowData)
		require.NotNil(t, result, "result should not be nil")
		bash, ok := result["bash"].([]any)
		require.True(t, ok, "bash should be a []any")
		assert.NotContains(t, bash, "gh:*", "gh:* should not be auto-added outside gh-proxy mode")
	})

	t.Run("nil workflowData returns nil", func(t *testing.T) {
		result := withMountedCLIShellCommandsInRestrictedBash(nil)
		assert.Nil(t, result, "nil input should return nil")
	})

	t.Run("workflowData with nil tools returns nil tools", func(t *testing.T) {
		workflowData := &WorkflowData{Tools: nil}
		result := withMountedCLIShellCommandsInRestrictedBash(workflowData)
		assert.Nil(t, result, "nil tools should be returned unchanged")
	})
}

func TestBuildMCPCLIPromptSection_PromptFileUsesNonHeadingLabels(t *testing.T) {
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CreateIssues: &CreateIssuesConfig{},
		},
	}

	section := buildMCPCLIPromptSection(data)
	require.NotNil(t, section)
	assert.Equal(t, mcpCLIToolsPromptFile, section.Content)

	wd, err := os.Getwd()
	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Clean(filepath.Join(wd, "../../actions/setup/md", section.Content)))
	require.NoError(t, err)

	prompt := string(content)
	assert.NotRegexp(t, `(?m)^\s*(>\s*)?##\s+`, prompt, "prompt must not contain H2 Markdown headings")
	assert.NotRegexp(t, `(?m)^\s*(>\s*)?###\s+`, prompt, "prompt must not contain H3 Markdown headings")
	assert.Contains(t, prompt, "Use `<server> --help` for tool names, parameters, and examples before calling any command.")
}
