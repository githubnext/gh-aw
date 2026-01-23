package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUpdateProjectConfig(t *testing.T) {
	tests := []struct {
		name           string
		outputMap      map[string]any
		expectedConfig *UpdateProjectConfig
		expectedNil    bool
	}{
		{
			name: "basic config with max",
			outputMap: map[string]any{
				"update-project": map[string]any{
					"max": 5,
				},
			},
			expectedConfig: &UpdateProjectConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 5,
				},
				GitHubToken: "",
			},
		},
		{
			name: "config with custom github-token",
			outputMap: map[string]any{
				"update-project": map[string]any{
					"max":          3,
					"github-token": "${{ secrets.PROJECTS_PAT }}",
				},
			},
			expectedConfig: &UpdateProjectConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 3,
				},
				GitHubToken: "${{ secrets.PROJECTS_PAT }}",
			},
		},
		{
			name: "config with default max when not specified",
			outputMap: map[string]any{
				"update-project": map[string]any{},
			},
			expectedConfig: &UpdateProjectConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 10,
				},
				GitHubToken: "",
			},
		},
		{
			name: "config with only github-token",
			outputMap: map[string]any{
				"update-project": map[string]any{
					"github-token": "${{ secrets.MY_TOKEN }}",
				},
			},
			expectedConfig: &UpdateProjectConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 10,
				},
				GitHubToken: "${{ secrets.MY_TOKEN }}",
			},
		},
		{
			name: "config with field-definitions",
			outputMap: map[string]any{
				"update-project": map[string]any{
					"field-definitions": []any{
						map[string]any{
							"name":      "status",
							"data-type": "SINGLE_SELECT",
							"options":   []any{"Todo", "Done"},
						},
						map[string]any{
							"name":      "campaign_id",
							"data-type": "TEXT",
						},
					},
				},
			},
			expectedConfig: &UpdateProjectConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: 10},
				FieldDefinitions: []ProjectFieldDefinition{
					{Name: "status", DataType: "SINGLE_SELECT", Options: []string{"Todo", "Done"}},
					{Name: "campaign_id", DataType: "TEXT"},
				},
			},
		},
		{
			name: "no update-project config",
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
			name: "update-project is nil",
			outputMap: map[string]any{
				"update-project": nil,
			},
			expectedConfig: &UpdateProjectConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 10,
				},
				GitHubToken: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			config := compiler.parseUpdateProjectConfig(tt.outputMap)

			if tt.expectedNil {
				assert.Nil(t, config, "Expected nil config")
			} else {
				require.NotNil(t, config, "Expected non-nil config")
				assert.Equal(t, tt.expectedConfig.Max, config.Max, "Max should match")
				assert.Equal(t, tt.expectedConfig.GitHubToken, config.GitHubToken, "GitHubToken should match")
				assert.Equal(t, tt.expectedConfig.FieldDefinitions, config.FieldDefinitions, "FieldDefinitions should match")
			}
		})
	}
}

func TestUpdateProjectConfig_DefaultMax(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	outputMap := map[string]any{
		"update-project": map[string]any{
			"github-token": "${{ secrets.TOKEN }}",
		},
	}

	config := compiler.parseUpdateProjectConfig(outputMap)
	require.NotNil(t, config)

	// Default max should be 10 when not specified
	assert.Equal(t, 10, config.Max, "Default max should be 10")
}

func TestUpdateProjectConfig_TokenPrecedence(t *testing.T) {
	tests := []struct {
		name          string
		configToken   string
		expectedToken string
	}{
		{
			name:          "custom token specified",
			configToken:   "${{ secrets.CUSTOM_PAT }}",
			expectedToken: "${{ secrets.CUSTOM_PAT }}",
		},
		{
			name:          "empty token",
			configToken:   "",
			expectedToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")

			outputMap := map[string]any{
				"update-project": map[string]any{
					"github-token": tt.configToken,
				},
			}

			config := compiler.parseUpdateProjectConfig(outputMap)
			require.NotNil(t, config)
			assert.Equal(t, tt.expectedToken, config.GitHubToken)
		})
	}
}

func TestBuildUpdateProjectJob(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		expectError  bool
		errorMsg     string
	}{
		{
			name: "valid config",
			workflowData: &WorkflowData{
				Name: "test-workflow",
				SafeOutputs: &SafeOutputsConfig{
					UpdateProjects: &UpdateProjectConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Max: 5,
						},
						GitHubToken: "${{ secrets.PROJECTS_PAT }}",
					},
				},
			},
			expectError: false,
		},
		{
			name: "missing safe outputs config",
			workflowData: &WorkflowData{
				Name:        "test-workflow",
				SafeOutputs: nil,
			},
			expectError: true,
			errorMsg:    "safe-outputs.update-project configuration is required",
		},
		{
			name: "missing update project config",
			workflowData: &WorkflowData{
				Name: "test-workflow",
				SafeOutputs: &SafeOutputsConfig{
					UpdateProjects: nil,
				},
			},
			expectError: true,
			errorMsg:    "safe-outputs.update-project configuration is required",
		},
		{
			name: "with default max",
			workflowData: &WorkflowData{
				Name: "test-workflow",
				SafeOutputs: &SafeOutputsConfig{
					UpdateProjects: &UpdateProjectConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Max: 10,
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")

			job, err := compiler.buildUpdateProjectJob(tt.workflowData, "main")

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, job)
			} else {
				require.NoError(t, err)
				require.NotNil(t, job)

				// Verify job has basic structure
				assert.NotEmpty(t, job.Steps, "Job should have steps")
			}
		})
	}
}

func TestUpdateProjectJob_EnvironmentVariables(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			UpdateProjects: &UpdateProjectConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 5,
				},
				GitHubToken: "${{ secrets.PROJECTS_PAT }}",
			},
		},
	}

	job, err := compiler.buildUpdateProjectJob(workflowData, "main")
	require.NoError(t, err)
	require.NotNil(t, job)

	// Job should contain steps
	assert.NotEmpty(t, job.Steps, "Job should have steps")

	// Check that GH_AW_PROJECT_GITHUB_TOKEN is set in the environment
	hasProjectToken := false
	for _, step := range job.Steps {
		if strings.Contains(step, "GH_AW_PROJECT_GITHUB_TOKEN") {
			hasProjectToken = true
			break
		}
	}
	assert.True(t, hasProjectToken, "Job should set GH_AW_PROJECT_GITHUB_TOKEN environment variable")
}

func TestUpdateProjectJob_Permissions(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			UpdateProjects: &UpdateProjectConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 10,
				},
			},
		},
	}

	job, err := compiler.buildUpdateProjectJob(workflowData, "main")
	require.NoError(t, err)
	require.NotNil(t, job)

	// Verify permissions are set correctly
	// update_project requires contents: read permission
	require.NotEmpty(t, job.Permissions, "Job should have permissions")
	assert.Contains(t, job.Permissions, "contents: read", "Should have contents: read permission")
}

func TestUpdateProjectJob_ViewsEscaping(t *testing.T) {
	tests := []struct {
		name         string
		views        []ProjectView
		shouldEscape bool
		checkContent string
	}{
		{
			name: "simple views without special characters",
			views: []ProjectView{
				{Name: "status", Layout: "board", Filter: "status:Todo"},
				{Name: "priority", Layout: "table", Filter: "priority:High"},
			},
			shouldEscape: false,
			checkContent: "status",
		},
		{
			name: "views with single quotes in filter",
			views: []ProjectView{
				{Name: "broken", Layout: "board", Filter: "label:\"can't fix\""},
				{Name: "issues", Layout: "table", Description: "It's broken"},
			},
			shouldEscape: true,
			checkContent: `\'`,
		},
		{
			name: "views with backslashes in description",
			views: []ProjectView{
				{Name: "regex", Layout: "board", Description: `Pattern: \d+\.\d+`},
				{Name: "path", Layout: "table", Filter: `path:C:\\Windows`},
			},
			shouldEscape: true,
			checkContent: `\\`,
		},
		{
			name: "views with mixed special characters",
			views: []ProjectView{
				{Name: "complex", Layout: "board", Description: `It's a "test" with \backslash`},
			},
			shouldEscape: true,
			checkContent: `\\`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")

			workflowData := &WorkflowData{
				Name: "test-workflow",
				SafeOutputs: &SafeOutputsConfig{
					UpdateProjects: &UpdateProjectConfig{
						BaseSafeOutputConfig: BaseSafeOutputConfig{
							Max: 10,
						},
						Views: tt.views,
					},
				},
			}

			job, err := compiler.buildUpdateProjectJob(workflowData, "main")
			require.NoError(t, err)
			require.NotNil(t, job)

			// Find the step containing GH_AW_PROJECT_VIEWS
			var viewsEnvVar string
			for _, step := range job.Steps {
				if strings.Contains(step, "GH_AW_PROJECT_VIEWS:") {
					viewsEnvVar = step
					break
				}
			}

			require.NotEmpty(t, viewsEnvVar, "Should contain GH_AW_PROJECT_VIEWS environment variable")

			// Verify that the JSON is properly escaped
			if tt.shouldEscape {
				// Should use single-quoted YAML string
				assert.Contains(t, viewsEnvVar, "GH_AW_PROJECT_VIEWS: '", "Should use single-quoted YAML string")

				// Verify the expected escape sequences are present
				assert.Contains(t, viewsEnvVar, tt.checkContent, "Should contain escaped characters")
			}

			// Verify the environment variable is properly formatted as YAML
			assert.Contains(t, viewsEnvVar, "GH_AW_PROJECT_VIEWS:", "Should contain environment variable key")
		})
	}
}

func TestUpdateProjectJob_ViewsNoInjection(t *testing.T) {
	// Test that malicious input cannot break out of the YAML string
	compiler := NewCompiler(false, "", "test")

	maliciousViews := []ProjectView{
		{Name: "injection", Layout: "board", Filter: "'; echo 'injected'; echo '"},
		{Name: "path", Layout: "table", Description: `\'; rm -rf /; echo '`},
	}

	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			UpdateProjects: &UpdateProjectConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 10,
				},
				Views: maliciousViews,
			},
		},
	}

	job, err := compiler.buildUpdateProjectJob(workflowData, "main")
	require.NoError(t, err)
	require.NotNil(t, job)

	// Find the step containing GH_AW_PROJECT_VIEWS
	var viewsEnvVar string
	for _, step := range job.Steps {
		if strings.Contains(step, "GH_AW_PROJECT_VIEWS:") {
			viewsEnvVar = step
			break
		}
	}

	require.NotEmpty(t, viewsEnvVar, "Should contain GH_AW_PROJECT_VIEWS environment variable")

	// Verify that all single quotes and backslashes are properly escaped
	// The environment variable should be wrapped in single quotes
	assert.Contains(t, viewsEnvVar, "GH_AW_PROJECT_VIEWS: '", "Should use single-quoted YAML string")

	// Count opening quotes
	openQuotes := strings.Count(viewsEnvVar, "GH_AW_PROJECT_VIEWS: '")
	assert.Equal(t, 1, openQuotes, "Should have exactly one opening quote for the environment variable")

	// Verify all single quotes in the JSON are escaped
	lines := strings.Split(viewsEnvVar, "\n")
	for _, line := range lines {
		if strings.Contains(line, "GH_AW_PROJECT_VIEWS:") {
			// Extract the value part after the colon
			parts := strings.SplitN(line, "GH_AW_PROJECT_VIEWS: '", 2)
			if len(parts) == 2 {
				value := parts[1]
				// Remove the trailing quote
				value = strings.TrimSuffix(value, "'")

				// Verify backslashes are escaped (doubled)
				if strings.Contains(value, `\`) {
					// Should contain escaped backslashes
					assert.Contains(t, value, `\\`, "Backslashes should be escaped")
				}

				// Verify single quotes are escaped
				if strings.Contains(value, "'") {
					// Should contain escaped quotes
					assert.Contains(t, value, `\'`, "Single quotes should be escaped")
				}
			}
		}
	}
}
