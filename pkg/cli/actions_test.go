package cli

import (
	"testing"
)

func TestConvertToGitHubActionsEnv(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected map[string]string
	}{
		{
			name: "shell syntax conversion",
			input: map[string]interface{}{
				"API_TOKEN":    "${API_TOKEN}",
				"NOTION_TOKEN": "${NOTION_TOKEN}",
			},
			expected: map[string]string{
				"API_TOKEN":    "${{ secrets.API_TOKEN }}",
				"NOTION_TOKEN": "${{ secrets.NOTION_TOKEN }}",
			},
		},
		{
			name: "mixed syntax",
			input: map[string]interface{}{
				"API_TOKEN":  "${API_TOKEN}",
				"PLAIN_VAR":  "plain_value",
				"GITHUB_VAR": "${{ secrets.EXISTING }}",
			},
			expected: map[string]string{
				"API_TOKEN":  "${{ secrets.API_TOKEN }}",
				"PLAIN_VAR":  "plain_value",
				"GITHUB_VAR": "${{ secrets.EXISTING }}",
			},
		},
		{
			name: "no shell syntax",
			input: map[string]interface{}{
				"PLAIN_VAR": "plain_value",
				"NUMBER":    "123",
			},
			expected: map[string]string{
				"PLAIN_VAR": "plain_value",
				"NUMBER":    "123",
			},
		},
		{
			name:     "empty input",
			input:    map[string]interface{}{},
			expected: map[string]string{},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: map[string]string{},
		},
		{
			name: "non-string values ignored",
			input: map[string]interface{}{
				"STRING_VAR": "${TOKEN}",
				"INT_VAR":    123,
				"BOOL_VAR":   true,
			},
			expected: map[string]string{
				"STRING_VAR": "${{ secrets.TOKEN }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToGitHubActionsEnv(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d environment variables, got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("Expected key '%s' not found in result", key)
				} else if actualValue != expectedValue {
					t.Errorf("For key '%s', expected '%s', got '%s'", key, expectedValue, actualValue)
				}
			}
		})
	}
}
