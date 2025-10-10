package workflow

import (
	"strings"
	"testing"
)

func TestGetGitHubToolsets(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "No toolsets configured",
			input:    map[string]any{},
			expected: "",
		},
		{
			name: "Toolsets as array of strings",
			input: map[string]any{
				"toolset": []string{"repos", "issues", "pull_requests"},
			},
			expected: "repos,issues,pull_requests",
		},
		{
			name: "Toolsets as array of any",
			input: map[string]any{
				"toolset": []any{"repos", "issues", "actions"},
			},
			expected: "repos,issues,actions",
		},
		{
			name: "Special 'all' toolset as array",
			input: map[string]any{
				"toolset": []string{"all"},
			},
			expected: "all",
		},
		{
			name:     "Non-map input returns empty",
			input:    "not a map",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getGitHubToolsets(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestClaudeEngineGitHubToolsetsRendering(t *testing.T) {
	tests := []struct {
		name           string
		githubTool     any
		expectedInYAML []string
		notInYAML      []string
	}{
		{
			name: "Toolsets configured with array",
			githubTool: map[string]any{
				"toolset": []string{"repos", "issues", "pull_requests"},
			},
			expectedInYAML: []string{
				`"GITHUB_TOOLSETS": "repos,issues,pull_requests"`,
			},
			notInYAML: []string{},
		},
		{
			name:       "No toolsets configured",
			githubTool: map[string]any{},
			expectedInYAML: []string{
				`"GITHUB_PERSONAL_ACCESS_TOKEN"`,
				"GITHUB_TOOLSETS",
			},
			notInYAML: []string{},
		},
		{
			name: "All toolset as array",
			githubTool: map[string]any{
				"toolset": []string{"all"},
			},
			expectedInYAML: []string{
				`"GITHUB_TOOLSETS": "all"`,
			},
			notInYAML: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &ClaudeEngine{}
			var yaml strings.Builder
			engine.renderGitHubClaudeMCPConfig(&yaml, tt.githubTool, true)

			result := yaml.String()

			for _, expected := range tt.expectedInYAML {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected YAML to contain %q, but it didn't.\nYAML:\n%s", expected, result)
				}
			}

			for _, notExpected := range tt.notInYAML {
				if strings.Contains(result, notExpected) {
					t.Errorf("Expected YAML to NOT contain %q, but it did.\nYAML:\n%s", notExpected, result)
				}
			}
		})
	}
}

func TestCopilotEngineGitHubToolsetsRendering(t *testing.T) {
	tests := []struct {
		name           string
		githubTool     any
		expectedInYAML []string
		notInYAML      []string
	}{
		{
			name: "Toolsets configured with array",
			githubTool: map[string]any{
				"toolset": []string{"repos", "issues", "pull_requests"},
			},
			expectedInYAML: []string{
				`"GITHUB_TOOLSETS=repos,issues,pull_requests"`,
			},
			notInYAML: []string{},
		},
		{
			name:       "No toolsets configured",
			githubTool: map[string]any{},
			expectedInYAML: []string{
				`GITHUB_PERSONAL_ACCESS_TOKEN`,
				"GITHUB_TOOLSETS",
			},
			notInYAML: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &CopilotEngine{}
			var yaml strings.Builder
			engine.renderGitHubCopilotMCPConfig(&yaml, tt.githubTool, true)

			result := yaml.String()

			for _, expected := range tt.expectedInYAML {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected YAML to contain %q, but it didn't.\nYAML:\n%s", expected, result)
				}
			}

			for _, notExpected := range tt.notInYAML {
				if strings.Contains(result, notExpected) {
					t.Errorf("Expected YAML to NOT contain %q, but it did.\nYAML:\n%s", notExpected, result)
				}
			}
		})
	}
}

func TestCodexEngineGitHubToolsetsRendering(t *testing.T) {
	tests := []struct {
		name           string
		githubTool     any
		expectedInYAML []string
		notInYAML      []string
	}{
		{
			name: "Toolsets configured with array",
			githubTool: map[string]any{
				"toolset": []string{"repos", "issues"},
			},
			expectedInYAML: []string{
				`GITHUB_TOOLSETS = "repos,issues"`,
			},
			notInYAML: []string{},
		},
		{
			name:       "No toolsets configured",
			githubTool: map[string]any{},
			expectedInYAML: []string{
				`GITHUB_PERSONAL_ACCESS_TOKEN`,
				"GITHUB_TOOLSETS",
			},
			notInYAML: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &CodexEngine{}
			var yaml strings.Builder
			workflowData := &WorkflowData{Name: "test-workflow"}
			engine.renderGitHubCodexMCPConfig(&yaml, tt.githubTool, workflowData)

			result := yaml.String()

			for _, expected := range tt.expectedInYAML {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected YAML to contain %q, but it didn't.\nYAML:\n%s", expected, result)
				}
			}

			for _, notExpected := range tt.notInYAML {
				if strings.Contains(result, notExpected) {
					t.Errorf("Expected YAML to NOT contain %q, but it did.\nYAML:\n%s", notExpected, result)
				}
			}
		})
	}
}

func TestGitHubToolsetsWithOtherConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		githubTool     any
		expectedInYAML []string
	}{
		{
			name: "Toolsets with read-only mode",
			githubTool: map[string]any{
				"toolset":   []string{"repos", "issues"},
				"read-only": true,
			},
			expectedInYAML: []string{
				`GITHUB_TOOLSETS`,
				`repos,issues`,
				`GITHUB_READ_ONLY`,
			},
		},
		{
			name: "Toolsets with custom token",
			githubTool: map[string]any{
				"toolset":      []string{"all"},
				"github-token": "${{ secrets.CUSTOM_PAT }}",
			},
			expectedInYAML: []string{
				`GITHUB_TOOLSETS`,
				`all`,
				`secrets.CUSTOM_PAT`,
			},
		},
		{
			name: "Toolsets with custom Docker version",
			githubTool: map[string]any{
				"toolset": []string{"repos", "issues", "pull_requests"},
				"version": "latest",
			},
			expectedInYAML: []string{
				`GITHUB_TOOLSETS`,
				`repos,issues,pull_requests`,
				`github-mcp-server:latest`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &ClaudeEngine{}
			var yaml strings.Builder
			engine.renderGitHubClaudeMCPConfig(&yaml, tt.githubTool, true)

			result := yaml.String()

			for _, expected := range tt.expectedInYAML {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected YAML to contain %q, but it didn't.\nYAML:\n%s", expected, result)
				}
			}
		})
	}
}
