//go:build !integration

package workflow

import (
	"testing"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testLog = logger.New("workflow:test")

// TestParseProjectViews verifies that parseProjectViews behaves identically
// for the create-project and update-project call paths by using the shared
// helper directly with a neutral logger.
func TestParseProjectViews(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected []ProjectView
	}{
		{
			name:     "no views key",
			input:    map[string]any{},
			expected: nil,
		},
		{
			name: "empty views list",
			input: map[string]any{
				"views": []any{},
			},
			expected: nil,
		},
		{
			name: "views value is wrong type (string)",
			input: map[string]any{
				"views": "not-a-list",
			},
			expected: nil,
		},
		{
			name: "views value is wrong type (map)",
			input: map[string]any{
				"views": map[string]any{"name": "Board", "layout": "board"},
			},
			expected: nil,
		},
		{
			name: "single valid view with required fields only",
			input: map[string]any{
				"views": []any{
					map[string]any{
						"name":   "Board",
						"layout": "board",
					},
				},
			},
			expected: []ProjectView{
				{Name: "Board", Layout: "board"},
			},
		},
		{
			name: "view with all optional fields",
			input: map[string]any{
				"views": []any{
					map[string]any{
						"name":           "Task Board",
						"layout":         "board",
						"filter":         "is:issue",
						"visible-fields": []any{1, 2, 3},
						"description":    "Main board",
					},
				},
			},
			expected: []ProjectView{
				{
					Name:          "Task Board",
					Layout:        "board",
					Filter:        "is:issue",
					VisibleFields: []int{1, 2, 3},
					Description:   "Main board",
				},
			},
		},
		{
			name: "visible-fields with mixed types: non-int elements are ignored",
			input: map[string]any{
				"views": []any{
					map[string]any{
						"name":           "Mixed",
						"layout":         "table",
						"visible-fields": []any{1, "bad", 3, nil},
					},
				},
			},
			expected: []ProjectView{
				{Name: "Mixed", Layout: "table", VisibleFields: []int{1, 3}},
			},
		},
		{
			name: "view missing name is skipped",
			input: map[string]any{
				"views": []any{
					map[string]any{"layout": "board"},
				},
			},
			expected: nil,
		},
		{
			name: "view missing layout is skipped",
			input: map[string]any{
				"views": []any{
					map[string]any{"name": "Board"},
				},
			},
			expected: nil,
		},
		{
			name: "mix of valid and invalid views",
			input: map[string]any{
				"views": []any{
					map[string]any{
						"name":   "Valid",
						"layout": "table",
					},
					map[string]any{
						// Missing layout
						"name": "Invalid",
					},
					map[string]any{
						"name":   "Also Valid",
						"layout": "roadmap",
					},
				},
			},
			expected: []ProjectView{
				{Name: "Valid", Layout: "table"},
				{Name: "Also Valid", Layout: "roadmap"},
			},
		},
		{
			name: "non-map view item is skipped",
			input: map[string]any{
				"views": []any{
					"not-a-map",
					map[string]any{
						"name":   "Good",
						"layout": "board",
					},
				},
			},
			expected: []ProjectView{
				{Name: "Good", Layout: "board"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseProjectViews(tt.input, testLog)
			if tt.expected == nil {
				assert.Empty(t, result, "parseProjectViews result should be empty")
			} else {
				assert.Equal(t, tt.expected, result, "parseProjectViews result should match expected")
			}
		})
	}
}

// TestParseProjectFieldDefinitions verifies that parseProjectFieldDefinitions
// behaves identically for both create-project and update-project paths.
func TestParseProjectFieldDefinitions(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected []ProjectFieldDefinition
	}{
		{
			name:     "no field-definitions key",
			input:    map[string]any{},
			expected: nil,
		},
		{
			name: "empty field-definitions list",
			input: map[string]any{
				"field-definitions": []any{},
			},
			expected: nil,
		},
		{
			name: "field-definitions value is wrong type (string)",
			input: map[string]any{
				"field-definitions": "not-a-list",
			},
			expected: nil,
		},
		{
			name: "field-definitions value is wrong type (map)",
			input: map[string]any{
				"field-definitions": map[string]any{"name": "Foo", "data-type": "TEXT"},
			},
			expected: nil,
		},
		{
			name: "single TEXT field using hyphen key",
			input: map[string]any{
				"field-definitions": []any{
					map[string]any{
						"name":      "Tracking Id",
						"data-type": "TEXT",
					},
				},
			},
			expected: []ProjectFieldDefinition{
				{Name: "Tracking Id", DataType: "TEXT"},
			},
		},
		{
			name: "single TEXT field using underscore key variant",
			input: map[string]any{
				"field_definitions": []any{
					map[string]any{
						"name":      "Worker Workflow",
						"data_type": "TEXT",
					},
				},
			},
			expected: []ProjectFieldDefinition{
				{Name: "Worker Workflow", DataType: "TEXT"},
			},
		},
		{
			name: "SINGLE_SELECT field with options",
			input: map[string]any{
				"field-definitions": []any{
					map[string]any{
						"name":      "Priority",
						"data-type": "SINGLE_SELECT",
						"options":   []any{"High", "Medium", "Low"},
					},
				},
			},
			expected: []ProjectFieldDefinition{
				{Name: "Priority", DataType: "SINGLE_SELECT", Options: []string{"High", "Medium", "Low"}},
			},
		},
		{
			name: "options with mixed types: non-string elements are ignored",
			input: map[string]any{
				"field-definitions": []any{
					map[string]any{
						"name":      "Size",
						"data-type": "SINGLE_SELECT",
						"options":   []any{"S", 42, "L", nil},
					},
				},
			},
			expected: []ProjectFieldDefinition{
				{Name: "Size", DataType: "SINGLE_SELECT", Options: []string{"S", "L"}},
			},
		},
		{
			name: "field missing name is skipped",
			input: map[string]any{
				"field-definitions": []any{
					map[string]any{"data-type": "TEXT"},
				},
			},
			expected: nil,
		},
		{
			name: "field missing data-type is skipped",
			input: map[string]any{
				"field-definitions": []any{
					map[string]any{"name": "MyField"},
				},
			},
			expected: nil,
		},
		{
			name: "non-map field item is skipped",
			input: map[string]any{
				"field-definitions": []any{
					"not-a-map",
					map[string]any{
						"name":      "Good",
						"data-type": "DATE",
					},
				},
			},
			expected: []ProjectFieldDefinition{
				{Name: "Good", DataType: "DATE"},
			},
		},
		{
			name: "multiple valid fields",
			input: map[string]any{
				"field-definitions": []any{
					map[string]any{
						"name":      "Tracking Id",
						"data-type": "TEXT",
					},
					map[string]any{
						"name":      "Priority",
						"data-type": "SINGLE_SELECT",
						"options":   []any{"High", "Medium", "Low"},
					},
					map[string]any{
						"name":      "Start Date",
						"data-type": "DATE",
					},
				},
			},
			expected: []ProjectFieldDefinition{
				{Name: "Tracking Id", DataType: "TEXT"},
				{Name: "Priority", DataType: "SINGLE_SELECT", Options: []string{"High", "Medium", "Low"}},
				{Name: "Start Date", DataType: "DATE"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseProjectFieldDefinitions(tt.input, testLog)
			if tt.expected == nil {
				assert.Empty(t, result, "parseProjectFieldDefinitions result should be empty")
			} else {
				assert.Equal(t, tt.expected, result, "parseProjectFieldDefinitions result should match expected")
			}
		})
	}
}

// TestParseProjectViewsAndFieldDefinitionsParity verifies that create-project
// and update-project both use the shared helpers and produce identical results
// for the same views/field-definitions input.
func TestParseProjectViewsAndFieldDefinitionsParity(t *testing.T) {
	viewsInput := []any{
		map[string]any{
			"name":           "Task Board",
			"layout":         "board",
			"filter":         "is:open",
			"visible-fields": []any{1, 2},
			"description":    "Main board",
		},
	}
	fieldsInput := []any{
		map[string]any{
			"name":      "Size",
			"data-type": "SINGLE_SELECT",
			"options":   []any{"S", "M", "L"},
		},
	}

	configMap := map[string]any{
		"views":             viewsInput,
		"field-definitions": fieldsInput,
	}

	compiler := NewCompiler()

	createOutputMap := map[string]any{
		"create-project": map[string]any{
			"views":             viewsInput,
			"field-definitions": fieldsInput,
		},
	}
	updateOutputMap := map[string]any{
		"update-project": map[string]any{
			"views":             viewsInput,
			"field-definitions": fieldsInput,
		},
	}

	createConfig := compiler.parseCreateProjectsConfig(createOutputMap)
	updateConfig := compiler.parseUpdateProjectConfig(updateOutputMap)

	require.NotNil(t, createConfig, "create config should not be nil")
	require.NotNil(t, updateConfig, "update config should not be nil")

	// Views must be identical from both paths
	expectedViews := parseProjectViews(configMap, testLog)
	assert.Equal(t, expectedViews, createConfig.Views, "create-project views should match shared helper output")
	assert.Equal(t, expectedViews, updateConfig.Views, "update-project views should match shared helper output")

	// Field definitions must be identical from both paths
	expectedFields := parseProjectFieldDefinitions(configMap, testLog)
	assert.Equal(t, expectedFields, createConfig.FieldDefinitions, "create-project field definitions should match shared helper output")
	assert.Equal(t, expectedFields, updateConfig.FieldDefinitions, "update-project field definitions should match shared helper output")
}
