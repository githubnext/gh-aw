//go:build !integration

package parser

import (
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

func TestFilterIgnoredFields(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    map[string]any
	}{
		{
			name:        "nil frontmatter",
			frontmatter: nil,
			expected:    nil,
		},
		{
			name:        "empty frontmatter",
			frontmatter: map[string]any{},
			expected:    map[string]any{},
		},
		{
			name: "frontmatter with description - no longer filtered",
			frontmatter: map[string]any{
				"description": "This is a test workflow",
				"on":          "push",
			},
			expected: map[string]any{
				"description": "This is a test workflow",
				"on":          "push",
			},
		},
		{
			name: "frontmatter with applyTo - no longer filtered",
			frontmatter: map[string]any{
				"applyTo": "some-value",
				"on":      "push",
			},
			expected: map[string]any{
				"applyTo": "some-value",
				"on":      "push",
			},
		},
		{
			name: "frontmatter with both description and applyTo - no longer filtered",
			frontmatter: map[string]any{
				"description": "This is a test workflow",
				"applyTo":     "some-value",
				"on":          "push",
				"engine":      "claude",
			},
			expected: map[string]any{
				"description": "This is a test workflow",
				"applyTo":     "some-value",
				"on":          "push",
				"engine":      "claude",
			},
		},
		{
			name: "frontmatter with only valid fields",
			frontmatter: map[string]any{
				"on":     "push",
				"engine": "claude",
			},
			expected: map[string]any{
				"on":     "push",
				"engine": "claude",
			},
		},
		{
			name: "frontmatter with user-invokable - should be filtered",
			frontmatter: map[string]any{
				"user-invokable": true,
				"on":             "push",
				"engine":         "claude",
			},
			expected: map[string]any{
				"on":     "push",
				"engine": "claude",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterIgnoredFields(tt.frontmatter)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d fields, got %d fields", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				if actualValue, ok := result[key]; !ok {
					t.Errorf("Expected field %q not found in result", key)
				} else {
					// For simple types, compare directly
					// For maps, we need to compare keys (simple check for this test)
					switch v := expectedValue.(type) {
					case map[string]any:
						if actualMap, ok := actualValue.(map[string]any); !ok {
							t.Errorf("Field %q: expected map, got %T", key, actualValue)
						} else if len(actualMap) != len(v) {
							t.Errorf("Field %q: expected map with %d keys, got %d keys", key, len(v), len(actualMap))
						}
					default:
						if actualValue != expectedValue {
							t.Errorf("Field %q: expected %v, got %v", key, expectedValue, actualValue)
						}
					}
				}
			}

			// Check that ignored fields are not present
			for _, ignoredField := range constants.IgnoredFrontmatterFields {
				if _, ok := result[ignoredField]; ok {
					t.Errorf("Ignored field %q should not be present in result", ignoredField)
				}
			}
		})
	}
}
