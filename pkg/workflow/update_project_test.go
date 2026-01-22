package workflow

import (
	"encoding/base64"
	"encoding/json"
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

func TestUpdateProjectJob_ViewsConfigurationBase64(t *testing.T) {
	tests := []struct {
		name         string
		views        []map[string]any
		expectViews  bool
		description  string
	}{
		{
			name:         "no views configured",
			views:        nil,
			expectViews:  false,
			description:  "Should not set GH_AW_PROJECT_VIEWS when no views are configured",
		},
		{
			name:         "empty views array",
			views:        []map[string]any{},
			expectViews:  false,
			description:  "Should not set GH_AW_PROJECT_VIEWS when views array is empty",
		},
		{
			name: "single view with simple name",
			views: []map[string]any{
				{"name": "My View", "id": "123"},
			},
			expectViews: true,
			description: "Should base64 encode views configuration with simple names",
		},
		{
			name: "view with special characters including single quotes",
			views: []map[string]any{
				{"name": "User's View", "id": "456", "description": "Contains 'quotes'"},
			},
			expectViews: true,
			description: "Should safely handle views with single quotes via base64 encoding",
		},
		{
			name: "view with double quotes and backslashes",
			views: []map[string]any{
				{"name": "Test\"View", "path": "\\path\\to\\view"},
			},
			expectViews: true,
			description: "Should safely handle views with double quotes and backslashes via base64",
		},
		{
			name: "multiple views with mixed special characters",
			views: []map[string]any{
				{"name": "View 1", "id": "1"},
				{"name": "User's \"View\"", "id": "2"},
				{"name": "Complex\\Path'View", "id": "3"},
			},
			expectViews: true,
			description: "Should safely handle multiple views with various special characters",
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
			require.NoError(t, err, tt.description)
			require.NotNil(t, job, tt.description)

			// Join all steps to search for environment variable
			allSteps := strings.Join(job.Steps, "\n")

			if tt.expectViews {
				// Should contain the GH_AW_PROJECT_VIEWS environment variable
				assert.Contains(t, allSteps, "GH_AW_PROJECT_VIEWS:", "Should set GH_AW_PROJECT_VIEWS when views are configured")

				// Extract the base64 value from the steps
				lines := strings.Split(allSteps, "\n")
				var viewsBase64 string
				for _, line := range lines {
					if strings.Contains(line, "GH_AW_PROJECT_VIEWS:") {
						parts := strings.SplitN(line, ":", 2)
						if len(parts) == 2 {
							viewsBase64 = strings.TrimSpace(parts[1])
							break
						}
					}
				}

				require.NotEmpty(t, viewsBase64, "Should have base64-encoded views value")

				// Verify the base64 value can be decoded back to the original JSON
				decodedBytes, err := base64.StdEncoding.DecodeString(viewsBase64)
				require.NoError(t, err, "Base64 value should be valid")

				var decodedViews []map[string]any
				err = json.Unmarshal(decodedBytes, &decodedViews)
				require.NoError(t, err, "Decoded value should be valid JSON")

				// Verify the decoded views match the original
				assert.Equal(t, tt.views, decodedViews, "Decoded views should match original configuration")

				// Security check: Verify no raw single quotes in the environment variable value
				// This ensures the base64 encoding prevents any quoting issues
				assert.NotContains(t, viewsBase64, "'", "Base64 value should not contain single quotes")
				assert.NotContains(t, viewsBase64, "\"", "Base64 value should not contain double quotes")
			} else {
				// Should not contain GH_AW_PROJECT_VIEWS when no views configured
				assert.NotContains(t, allSteps, "GH_AW_PROJECT_VIEWS:", "Should not set GH_AW_PROJECT_VIEWS when no views are configured")
			}
		})
	}
}

func TestUpdateProjectJob_Base64EncodingPreventsSQLInjection(t *testing.T) {
	// Test that base64 encoding prevents potential injection attacks
	compiler := NewCompiler(false, "", "test")

	maliciousViews := []map[string]any{
		{
			"name": "'; DROP TABLE projects; --",
			"id":   "injection-test",
		},
		{
			"name": "${INJECTION}",
			"id":   "var-expansion-test",
		},
		{
			"name": "`rm -rf /`",
			"id":   "command-injection-test",
		},
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
	require.NoError(t, err, "Should handle malicious input without error")
	require.NotNil(t, job)

	allSteps := strings.Join(job.Steps, "\n")

	// Verify malicious strings are not present in raw form
	assert.NotContains(t, allSteps, "DROP TABLE", "SQL injection attempt should be encoded")
	assert.NotContains(t, allSteps, "rm -rf", "Command injection attempt should be encoded")

	// Verify the base64 value is present
	assert.Contains(t, allSteps, "GH_AW_PROJECT_VIEWS:", "Environment variable should be set")

	// Extract and verify the base64 value contains only safe characters
	lines := strings.Split(allSteps, "\n")
	for _, line := range lines {
		if strings.Contains(line, "GH_AW_PROJECT_VIEWS:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				viewsBase64 := strings.TrimSpace(parts[1])
				// Base64 should only contain alphanumeric, +, /, and = characters
				for _, char := range viewsBase64 {
					assert.True(t,
						(char >= 'A' && char <= 'Z') ||
							(char >= 'a' && char <= 'z') ||
							(char >= '0' && char <= '9') ||
							char == '+' || char == '/' || char == '=',
						"Base64 value should only contain safe characters")
				}
				break
			}
		}
	}
}
