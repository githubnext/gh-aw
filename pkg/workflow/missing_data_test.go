//go:build !integration

package workflow

import (
	"testing"
)

func TestMissingDataSafeOutput(t *testing.T) {
	tests := []struct {
		name         string
		frontmatter  map[string]any
		expectConfig bool
		expectJob    bool
		expectMax    int
	}{
		{
			name:         "No safe-outputs config should NOT enable missing-data by default",
			frontmatter:  map[string]any{"name": "Test"},
			expectConfig: false,
			expectJob:    false,
			expectMax:    0,
		},
		{
			name: "Safe-outputs with other config should enable missing-data by default",
			frontmatter: map[string]any{
				"name": "Test",
				"safe-outputs": map[string]any{
					"create-issue": nil,
				},
			},
			expectConfig: true,
			expectJob:    false, // Job not built in config test
			expectMax:    0,
		},
		{
			name: "Explicit missing-data: false should disable it",
			frontmatter: map[string]any{
				"name": "Test",
				"safe-outputs": map[string]any{
					"create-issue": nil,
					"missing-data": false,
				},
			},
			expectConfig: false,
			expectJob:    false,
			expectMax:    0,
		},
		{
			name: "Explicit missing-data config with max",
			frontmatter: map[string]any{
				"name": "Test",
				"safe-outputs": map[string]any{
					"missing-data": map[string]any{
						"max": 5,
					},
				},
			},
			expectConfig: true,
			expectJob:    false,
			expectMax:    5,
		},
		{
			name: "Missing-data with other safe outputs",
			frontmatter: map[string]any{
				"name": "Test",
				"safe-outputs": map[string]any{
					"create-issue": nil,
					"missing-data": nil,
				},
			},
			expectConfig: true,
			expectJob:    false,
			expectMax:    0,
		},
		{
			name: "Empty missing-data config",
			frontmatter: map[string]any{
				"name": "Test",
				"safe-outputs": map[string]any{
					"missing-data": nil,
				},
			},
			expectConfig: true,
			expectJob:    false,
			expectMax:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()

			// Extract safe outputs config
			safeOutputs := compiler.extractSafeOutputsConfig(tt.frontmatter)

			// Verify config expectations
			if tt.expectConfig {
				if safeOutputs == nil {
					t.Fatal("Expected SafeOutputsConfig to be created, but it was nil")
				}
				if safeOutputs.MissingData == nil {
					t.Error("Expected MissingData config to be present, but it was nil")
				} else {
					if templatableIntValue(safeOutputs.MissingData.Max) != tt.expectMax {
						t.Errorf("Expected Max=%d, got Max=%v", tt.expectMax, safeOutputs.MissingData.Max)
					}
				}
			} else {
				if safeOutputs != nil && safeOutputs.MissingData != nil {
					t.Error("Expected MissingData config to be nil, but it was present")
				}
			}
		})
	}
}

func TestMissingDataConfigParsing(t *testing.T) {
	tests := []struct {
		name         string
		configData   map[string]any
		expectNil    bool
		expectMax    int
		expectIssue  *string
		expectTitle  string
		expectLabels []string
	}{
		{
			name: "Default config with nil value",
			configData: map[string]any{
				"missing-data": nil,
			},
			expectNil:    false,
			expectMax:    0,
			expectIssue:  strPtr("true"),
			expectTitle:  "[missing data]",
			expectLabels: []string{},
		},
		{
			name: "Config with create-issue disabled",
			configData: map[string]any{
				"missing-data": map[string]any{
					"create-issue": false,
				},
			},
			expectNil:    false,
			expectMax:    0,
			expectIssue:  strPtr("false"),
			expectTitle:  "[missing data]",
			expectLabels: []string{},
		},
		{
			name: "Config with create-issue as expression",
			configData: map[string]any{
				"missing-data": map[string]any{
					"create-issue": "${{ inputs.create-missing-data-issue }}",
				},
			},
			expectNil:    false,
			expectMax:    0,
			expectIssue:  strPtr("${{ inputs.create-missing-data-issue }}"),
			expectTitle:  "[missing data]",
			expectLabels: []string{},
		},
		{
			name: "Config with custom title and labels",
			configData: map[string]any{
				"missing-data": map[string]any{
					"title-prefix": "[data needed]",
					"labels":       []any{"data", "blocked"},
					"max":          10,
				},
			},
			expectNil:    false,
			expectMax:    10,
			expectIssue:  strPtr("true"),
			expectTitle:  "[data needed]",
			expectLabels: []string{"data", "blocked"},
		},
		{
			name: "Config explicitly disabled",
			configData: map[string]any{
				"missing-data": false,
			},
			expectNil:    true,
			expectMax:    0,
			expectIssue:  nil,
			expectTitle:  "",
			expectLabels: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()
			config := compiler.parseMissingDataConfig(tt.configData)

			if tt.expectNil {
				if config != nil {
					t.Error("Expected nil config, but got non-nil")
				}
				return
			}

			if config == nil {
				t.Fatal("Expected non-nil config, but got nil")
			}

			if templatableIntValue(config.Max) != tt.expectMax {
				t.Errorf("Expected Max=%d, got Max=%v", tt.expectMax, config.Max)
			}

			if tt.expectIssue == nil {
				if config.CreateIssue != nil {
					t.Errorf("Expected CreateIssue=nil, got CreateIssue=%q", *config.CreateIssue)
				}
			} else {
				if config.CreateIssue == nil {
					t.Errorf("Expected CreateIssue=%q, got CreateIssue=nil", *tt.expectIssue)
				} else if *config.CreateIssue != *tt.expectIssue {
					t.Errorf("Expected CreateIssue=%q, got CreateIssue=%q", *tt.expectIssue, *config.CreateIssue)
				}
			}

			if config.TitlePrefix != tt.expectTitle {
				t.Errorf("Expected TitlePrefix=%q, got TitlePrefix=%q", tt.expectTitle, config.TitlePrefix)
			}

			if len(config.Labels) != len(tt.expectLabels) {
				t.Errorf("Expected %d labels, got %d", len(tt.expectLabels), len(config.Labels))
			}
		})
	}
}
