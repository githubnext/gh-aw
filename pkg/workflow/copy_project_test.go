package workflow

import (
	"strings"
	"testing"
)

// TestCopyProjectConfiguration verifies that copy-project configuration
// is parsed correctly and integrated into the workflow
func TestCopyProjectConfiguration(t *testing.T) {
	tests := []struct {
		name                string
		frontmatter         map[string]any
		expectedConfigured  bool
		expectedMaxDefault  int
		expectedCustomToken bool
	}{
		{
			name: "copy-project with default configuration",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"copy-project": nil,
				},
			},
			expectedConfigured: true,
			expectedMaxDefault: 10,
		},
		{
			name: "copy-project with custom max",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"copy-project": map[string]any{
						"max": 5,
					},
				},
			},
			expectedConfigured: true,
			expectedMaxDefault: 5,
		},
		{
			name: "copy-project with custom github-token",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"copy-project": map[string]any{
						"github-token": "${{ secrets.PROJECTS_PAT }}",
					},
				},
			},
			expectedConfigured:  true,
			expectedMaxDefault:  10,
			expectedCustomToken: true,
		},
		{
			name: "no copy-project configuration",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"create-issue": nil,
				},
			},
			expectedConfigured: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")

			// Parse frontmatter
			config := compiler.extractSafeOutputsConfig(tt.frontmatter)

			if tt.expectedConfigured {
				if config == nil || config.CopyProjects == nil {
					t.Fatalf("Expected copy-project to be configured, but it was not")
				}

				if config.CopyProjects.Max != tt.expectedMaxDefault {
					t.Errorf("Expected max to be %d, got %d", tt.expectedMaxDefault, config.CopyProjects.Max)
				}

				if tt.expectedCustomToken {
					if config.CopyProjects.GitHubToken == "" {
						t.Errorf("Expected custom github-token to be set, but it was empty")
					}
				}
			} else {
				if config != nil && config.CopyProjects != nil {
					t.Errorf("Expected copy-project not to be configured, but it was")
				}
			}
		})
	}
}

// TestCopyProjectGitHubTokenEnvVar verifies that the github-token
// is passed correctly to the copy_project step
func TestCopyProjectGitHubTokenEnvVar(t *testing.T) {
	tests := []struct {
		name                string
		frontmatter         map[string]any
		expectedTokenValue  string
	}{
		{
			name: "copy-project with custom github-token",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"copy-project": map[string]any{
						"github-token": "${{ secrets.PROJECTS_PAT }}",
					},
				},
			},
			expectedTokenValue: "github-token: ${{ secrets.PROJECTS_PAT }}",
		},
		{
			name: "copy-project without custom github-token (uses default fallback)",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"copy-project": nil,
				},
			},
			expectedTokenValue: "github-token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}",
		},
		{
			name: "copy-project with top-level github-token",
			frontmatter: map[string]any{
				"name":         "Test Workflow",
				"github-token": "${{ secrets.CUSTOM_TOKEN }}",
				"safe-outputs": map[string]any{
					"copy-project": nil,
				},
			},
			expectedTokenValue: "github-token: ${{ secrets.CUSTOM_TOKEN }}",
		},
		{
			name: "copy-project with per-config token overrides top-level",
			frontmatter: map[string]any{
				"name":         "Test Workflow",
				"github-token": "${{ secrets.GLOBAL_TOKEN }}",
				"safe-outputs": map[string]any{
					"copy-project": map[string]any{
						"github-token": "${{ secrets.PROJECT_SPECIFIC_TOKEN }}",
					},
				},
			},
			expectedTokenValue: "github-token: ${{ secrets.PROJECT_SPECIFIC_TOKEN }}",
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

			// Build the consolidated safe outputs job
			job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, "agent", "test.md")
			if err != nil {
				t.Fatalf("Failed to build consolidated safe outputs job: %v", err)
			}

			if job == nil {
				t.Fatalf("Expected consolidated safe outputs job to be created, but it was nil")
			}

			// Convert job to YAML to check for token configuration
			yamlStr := strings.Join(job.Steps, "")

			// Check that the github-token is passed correctly
			if !strings.Contains(yamlStr, tt.expectedTokenValue) {
				t.Errorf("Expected github-token %q to be set in copy_project step, but it was not found.\nGenerated YAML:\n%s",
					tt.expectedTokenValue, yamlStr)
			}

			// Verify that the copy_project step is present
			if !strings.Contains(yamlStr, "Copy Project") {
				t.Errorf("Expected 'Copy Project' step to be present in generated YAML, but it was not found")
			}
		})
	}
}

// TestCopyProjectStepCondition verifies that the copy_project step has the correct condition
func TestCopyProjectStepCondition(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	frontmatter := map[string]any{
		"name": "Test Workflow",
		"safe-outputs": map[string]any{
			"copy-project": nil,
		},
	}

	workflowData := &WorkflowData{
		Name:        "test-workflow",
		SafeOutputs: compiler.extractSafeOutputsConfig(frontmatter),
	}

	// Build the consolidated safe outputs job
	job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, "agent", "test.md")
	if err != nil {
		t.Fatalf("Failed to build consolidated safe outputs job: %v", err)
	}

	if job == nil {
		t.Fatalf("Expected consolidated safe outputs job to be created, but it was nil")
	}

	// Convert job to YAML to check for step condition
	yamlStr := strings.Join(job.Steps, "")

	// The copy_project step should have a condition that checks for the copy_project type
	// The condition references needs.agent.outputs.output_types (not safe_output_types)
	expectedCondition := "contains(needs.agent.outputs.output_types, 'copy_project')"
	if !strings.Contains(yamlStr, expectedCondition) {
		t.Errorf("Expected condition %q to be present in copy_project step, but it was not found.\nGenerated YAML:\n%s",
			expectedCondition, yamlStr)
	}
}
