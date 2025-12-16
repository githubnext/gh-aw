package workflow

import (
	"strings"
	"testing"
)

// TestUpdateProjectGitHubTokenEnvVar verifies that GH_AW_PROJECT_GITHUB_TOKEN
// is exposed as an environment variable when a custom token is configured
func TestUpdateProjectGitHubTokenEnvVar(t *testing.T) {
	tests := []struct {
		name                string
		frontmatter         map[string]any
		expectedEnvVar      bool
		expectedEnvVarValue string
	}{
		{
			name: "update-project with custom github-token",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"update-project": map[string]any{
						"github-token": "${{ secrets.PROJECTS_PAT }}",
					},
				},
			},
			expectedEnvVar:      true,
			expectedEnvVarValue: "GH_AW_PROJECT_GITHUB_TOKEN: ${{ secrets.PROJECTS_PAT }}",
		},
		{
			name: "update-project without custom github-token",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"update-project": nil,
				},
			},
			expectedEnvVar: false,
		},
		{
			name: "update-project with empty github-token",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"update-project": map[string]any{
						"github-token": "",
					},
				},
			},
			expectedEnvVar: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")

			// Parse frontmatter
			workflowData := &WorkflowData{
				Name:        "test-workflow",
				SafeOutputs: compiler.extractSafeOutputsConfig(tt.frontmatter),
			}

			// Build the update_project job
			job, err := compiler.buildUpdateProjectJob(workflowData, "main")
			if err != nil {
				t.Fatalf("Failed to build update_project job: %v", err)
			}

			// Convert job to YAML to check for environment variables
			yamlStr := strings.Join(job.Steps, "")

			if tt.expectedEnvVar {
				// Check that the environment variable is present
				if !strings.Contains(yamlStr, tt.expectedEnvVarValue) {
					t.Errorf("Expected environment variable %q to be set in update_project job, but it was not found.\nGenerated YAML:\n%s",
						tt.expectedEnvVarValue, yamlStr)
				}
			} else {
				// Check that the environment variable is NOT present
				if strings.Contains(yamlStr, "GH_AW_PROJECT_GITHUB_TOKEN:") {
					t.Errorf("Expected GH_AW_PROJECT_GITHUB_TOKEN environment variable to NOT be set, but it was found.\nGenerated YAML:\n%s",
						yamlStr)
				}
			}

			// Also verify the token is passed to github-token parameter
			if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.UpdateProjects != nil {
				token := workflowData.SafeOutputs.UpdateProjects.GitHubToken
				if token != "" {
					expectedWith := "github-token: " + token
					if !strings.Contains(yamlStr, expectedWith) {
						t.Errorf("Expected github-token parameter to be set to %q, but it was not found.\nGenerated YAML:\n%s",
							token, yamlStr)
					}
				}
			}
		})
	}
}
