package workflow

import (
	"strings"
	"testing"
)

// TestUpdateProjectGitHubTokenEnvVar verifies that GH_AW_PROJECT_GITHUB_TOKEN
// is exposed as an environment variable for all update_project jobs
func TestUpdateProjectGitHubTokenEnvVar(t *testing.T) {
	tests := []struct {
		name                string
		frontmatter         map[string]any
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
			expectedEnvVarValue: "GH_AW_PROJECT_GITHUB_TOKEN: ${{ secrets.PROJECTS_PAT }}",
		},
		{
			name: "update-project without custom github-token (uses default fallback)",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"update-project": nil,
				},
			},
			expectedEnvVarValue: "GH_AW_PROJECT_GITHUB_TOKEN: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}",
		},
		{
			name: "update-project with top-level github-token",
			frontmatter: map[string]any{
				"name":         "Test Workflow",
				"github-token": "${{ secrets.CUSTOM_TOKEN }}",
				"safe-outputs": map[string]any{
					"update-project": nil,
				},
			},
			expectedEnvVarValue: "GH_AW_PROJECT_GITHUB_TOKEN: ${{ secrets.CUSTOM_TOKEN }}",
		},
		{
			name: "update-project with per-config token overrides top-level",
			frontmatter: map[string]any{
				"name":         "Test Workflow",
				"github-token": "${{ secrets.GLOBAL_TOKEN }}",
				"safe-outputs": map[string]any{
					"update-project": map[string]any{
						"github-token": "${{ secrets.PROJECT_SPECIFIC_TOKEN }}",
					},
				},
			},
			expectedEnvVarValue: "GH_AW_PROJECT_GITHUB_TOKEN: ${{ secrets.PROJECT_SPECIFIC_TOKEN }}",
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

			// Set top-level github-token if present in frontmatter
			if githubToken, ok := tt.frontmatter["github-token"].(string); ok {
				workflowData.GitHubToken = githubToken
			}

			// Build the update_project job
			job, err := compiler.buildUpdateProjectJob(workflowData, "main")
			if err != nil {
				t.Fatalf("Failed to build update_project job: %v", err)
			}

			// Check steps for environment variable and github-token parameter
			foundEnv := false
			foundWith := false
			expectedEnvVarKeyVal := strings.SplitN(tt.expectedEnvVarValue, ": ", 2)
			expectedEnvVarKey := expectedEnvVarKeyVal[0]
			expectedEnvVarVal := ""
			if len(expectedEnvVarKeyVal) > 1 {
				expectedEnvVarVal = expectedEnvVarKeyVal[1]
			}
			expectedWithVal := strings.TrimPrefix(tt.expectedEnvVarValue, "GH_AW_PROJECT_GITHUB_TOKEN: ")

			for _, step := range job.Steps {
				// Try to assert step is a map[string]interface{}
				stepMap, ok := step.(map[string]interface{})
				if !ok {
					continue
				}

				// Check for env
				if env, ok := stepMap["env"].(map[string]interface{}); ok {
					if val, ok := env[expectedEnvVarKey]; ok && val == expectedEnvVarVal {
						foundEnv = true
					}
				}

				// Check for github-token in with
				if with, ok := stepMap["with"].(map[string]interface{}); ok {
					if val, ok := with["github-token"]; ok && val == expectedWithVal {
						foundWith = true
					}
				}
			}

			if !foundEnv {
				t.Errorf("Expected environment variable %q to be set in update_project job, but it was not found.\nJob steps: %+v",
					tt.expectedEnvVarValue, job.Steps)
			}

			if !foundWith {
				t.Errorf("Expected github-token parameter to be set to %q, but it was not found.\nJob steps: %+v",
					expectedWithVal, job.Steps)
			}
		})
	}
}
