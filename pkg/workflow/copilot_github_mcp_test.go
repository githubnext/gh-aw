package workflow

import (
	"testing"
)

func TestGetGitHubAllowedTools(t *testing.T) {
	tests := []struct {
		name       string
		githubTool any
		expected   []string
	}{
		{
			name: "Specific allowed tools",
			githubTool: map[string]any{
				"allowed": []string{"get_repository", "list_commits"},
			},
			expected: []string{"get_repository", "list_commits"},
		},
		{
			name: "Empty allowed array",
			githubTool: map[string]any{
				"allowed": []string{},
			},
			expected: []string{},
		},
		{
			name:       "No allowed field",
			githubTool: map[string]any{},
			expected:   nil,
		},
		{
			name: "Allowed with []any type",
			githubTool: map[string]any{
				"allowed": []any{"tool1", "tool2", "tool3"},
			},
			expected: []string{"tool1", "tool2", "tool3"},
		},
		{
			name:       "Not a map",
			githubTool: "invalid",
			expected:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getGitHubAllowedTools(tt.githubTool)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected %v, got nil", tt.expected)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d tools, got %d", len(tt.expected), len(result))
				return
			}

			for i, tool := range tt.expected {
				if result[i] != tool {
					t.Errorf("Expected tool[%d] = %s, got %s", i, tool, result[i])
				}
			}
		})
	}
}
