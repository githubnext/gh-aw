package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCreateProjectsConfig(t *testing.T) {
	tests := []struct {
		name           string
		outputMap      map[string]any
		expectedConfig *CreateProjectsConfig
		expectedNil    bool
	}{
		{
			name: "basic config with max",
			outputMap: map[string]any{
				"create-project": map[string]any{
					"max": 2,
				},
			},
			expectedConfig: &CreateProjectsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 2,
				},
			},
		},
		{
			name: "config with all fields",
			outputMap: map[string]any{
				"create-project": map[string]any{
					"max":          1,
					"github-token": "${{ secrets.PROJECTS_PAT }}",
					"target-owner": "myorg",
					"title-prefix": "Campaign",
				},
			},
			expectedConfig: &CreateProjectsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
				GitHubToken: "${{ secrets.PROJECTS_PAT }}",
				TargetOwner: "myorg",
				TitlePrefix: "Campaign",
			},
		},
		{
			name: "config with views",
			outputMap: map[string]any{
				"create-project": map[string]any{
					"max": 1,
					"views": []any{
						map[string]any{
							"name":   "Campaign Roadmap",
							"layout": "roadmap",
							"filter": "is:issue,is:pull_request",
						},
						map[string]any{
							"name":   "Task Tracker",
							"layout": "table",
							"filter": "is:open",
						},
					},
				},
			},
			expectedConfig: &CreateProjectsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
				Views: []ProjectView{
					{
						Name:   "Campaign Roadmap",
						Layout: "roadmap",
						Filter: "is:issue,is:pull_request",
					},
					{
						Name:   "Task Tracker",
						Layout: "table",
						Filter: "is:open",
					},
				},
			},
		},
		{
			name: "config with views including visible-fields",
			outputMap: map[string]any{
				"create-project": map[string]any{
					"max": 1,
					"views": []any{
						map[string]any{
							"name":           "Task Board",
							"layout":         "board",
							"filter":         "is:issue",
							"visible-fields": []any{1, 2, 3},
							"description":    "Main task board",
						},
					},
				},
			},
			expectedConfig: &CreateProjectsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
				Views: []ProjectView{
					{
						Name:          "Task Board",
						Layout:        "board",
						Filter:        "is:issue",
						VisibleFields: []int{1, 2, 3},
						Description:   "Main task board",
					},
				},
			},
		},
		{
			name: "config with default max when not specified",
			outputMap: map[string]any{
				"create-project": map[string]any{
					"target-owner": "testorg",
				},
			},
			expectedConfig: &CreateProjectsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
				TargetOwner: "testorg",
			},
		},
		{
			name: "no create-project config",
			outputMap: map[string]any{
				"create-issue": map[string]any{},
			},
			expectedNil: true,
		},
		{
			name:        "empty outputMap",
			outputMap:   map[string]any{},
			expectedNil: true,
		},
		{
			name: "views with missing required fields are skipped",
			outputMap: map[string]any{
				"create-project": map[string]any{
					"max": 1,
					"views": []any{
						map[string]any{
							"name":   "Valid View",
							"layout": "table",
						},
						map[string]any{
							// Missing layout - should be skipped
							"name": "Invalid View",
						},
						map[string]any{
							// Missing name - should be skipped
							"layout": "board",
						},
					},
				},
			},
			expectedConfig: &CreateProjectsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
				Views: []ProjectView{
					{
						Name:   "Valid View",
						Layout: "table",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			config := compiler.parseCreateProjectsConfig(tt.outputMap)

			if tt.expectedNil {
				assert.Nil(t, config, "Expected nil config")
			} else {
				require.NotNil(t, config, "Expected non-nil config")
				assert.Equal(t, tt.expectedConfig.Max, config.Max, "Max should match")
				assert.Equal(t, tt.expectedConfig.GitHubToken, config.GitHubToken, "GitHubToken should match")
				assert.Equal(t, tt.expectedConfig.TargetOwner, config.TargetOwner, "TargetOwner should match")
				assert.Equal(t, tt.expectedConfig.TitlePrefix, config.TitlePrefix, "TitlePrefix should match")
				assert.Len(t, config.Views, len(tt.expectedConfig.Views), "Views count should match")

				// Check views details
				for i, expectedView := range tt.expectedConfig.Views {
					assert.Equal(t, expectedView.Name, config.Views[i].Name, "View name should match")
					assert.Equal(t, expectedView.Layout, config.Views[i].Layout, "View layout should match")
					assert.Equal(t, expectedView.Filter, config.Views[i].Filter, "View filter should match")
					assert.Equal(t, expectedView.VisibleFields, config.Views[i].VisibleFields, "View visible fields should match")
					assert.Equal(t, expectedView.Description, config.Views[i].Description, "View description should match")
				}
			}
		})
	}
}

func TestCreateProjectsConfig_DefaultMax(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	outputMap := map[string]any{
		"create-project": map[string]any{
			"target-owner": "myorg",
		},
	}

	config := compiler.parseCreateProjectsConfig(outputMap)
	require.NotNil(t, config)

	// Default max should be 1 when not specified
	assert.Equal(t, 1, config.Max, "Default max should be 1")
}

func TestCreateProjectsConfig_ViewsParsing(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	outputMap := map[string]any{
		"create-project": map[string]any{
			"max": 1,
			"views": []any{
				map[string]any{
					"name":   "Sprint Board",
					"layout": "board",
					"filter": "is:open label:sprint",
				},
				map[string]any{
					"name":   "Timeline",
					"layout": "roadmap",
				},
			},
		},
	}

	config := compiler.parseCreateProjectsConfig(outputMap)
	require.NotNil(t, config)
	require.Len(t, config.Views, 2, "Should parse 2 views")

	// Check first view
	assert.Equal(t, "Sprint Board", config.Views[0].Name)
	assert.Equal(t, "board", config.Views[0].Layout)
	assert.Equal(t, "is:open label:sprint", config.Views[0].Filter)

	// Check second view
	assert.Equal(t, "Timeline", config.Views[1].Name)
	assert.Equal(t, "roadmap", config.Views[1].Layout)
	assert.Empty(t, config.Views[1].Filter) // No filter specified
}
