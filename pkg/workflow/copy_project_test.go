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
			expectedMaxDefault: 1,
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
			expectedMaxDefault:  1,
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
// is passed correctly to the project handler manager step when copy-project is enabled.
func TestCopyProjectGitHubTokenEnvVar(t *testing.T) {
	tests := []struct {
		name               string
		frontmatter        map[string]any
		expectedTokenValue string
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
				t.Errorf("Expected github-token %q to be set in project handler manager step, but it was not found.\nGenerated YAML:\n%s",
					tt.expectedTokenValue, yamlStr)
			}

			// Verify that the project handler manager step is present
			if !strings.Contains(yamlStr, "id: process_project_safe_outputs") {
				t.Errorf("Expected project handler manager step (process_project_safe_outputs) to be present in generated YAML, but it was not found")
			}
		})
	}
}

// TestCopyProjectStepCondition verifies that copy_project configuration is wired into the
// project handler manager step.
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

	// copy_project should be included in the project handler manager configuration.
	if !strings.Contains(yamlStr, "GH_AW_SAFE_OUTPUTS_PROJECT_HANDLER_CONFIG") {
		t.Errorf("Expected project handler manager config env var to be present, but it was not found.\nGenerated YAML:\n%s", yamlStr)
	}
	if !strings.Contains(yamlStr, "copy_project") {
		t.Errorf("Expected copy_project configuration to be present in project handler manager config, but it was not found.\nGenerated YAML:\n%s", yamlStr)
	}
}

// TestCopyProjectSourceAndTargetConfiguration verifies that source-project and target-owner
// configuration is parsed correctly and passed as environment variables
func TestCopyProjectSourceAndTargetConfiguration(t *testing.T) {
	tests := []struct {
		name                  string
		frontmatter           map[string]any
		expectedSourceProject string
		expectedTargetOwner   string
		shouldHaveSource      bool
		shouldHaveTarget      bool
	}{
		{
			name: "copy-project with source-project configured",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"copy-project": map[string]any{
						"source-project": "https://github.com/orgs/myorg/projects/42",
					},
				},
			},
			expectedSourceProject: "https://github.com/orgs/myorg/projects/42",
			shouldHaveSource:      true,
			shouldHaveTarget:      false,
		},
		{
			name: "copy-project with target-owner configured",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"copy-project": map[string]any{
						"target-owner": "myorg",
					},
				},
			},
			expectedTargetOwner: "myorg",
			shouldHaveSource:    false,
			shouldHaveTarget:    true,
		},
		{
			name: "copy-project with both source-project and target-owner configured",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"copy-project": map[string]any{
						"source-project": "https://github.com/orgs/myorg/projects/99",
						"target-owner":   "targetorg",
					},
				},
			},
			expectedSourceProject: "https://github.com/orgs/myorg/projects/99",
			expectedTargetOwner:   "targetorg",
			shouldHaveSource:      true,
			shouldHaveTarget:      true,
		},
		{
			name: "copy-project without source-project and target-owner",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"copy-project": nil,
				},
			},
			shouldHaveSource: false,
			shouldHaveTarget: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")

			// Parse frontmatter
			config := compiler.extractSafeOutputsConfig(tt.frontmatter)

			if config == nil || config.CopyProjects == nil {
				t.Fatalf("Expected copy-project to be configured, but it was not")
			}

			// Check source-project parsing
			if tt.shouldHaveSource {
				if config.CopyProjects.SourceProject != tt.expectedSourceProject {
					t.Errorf("Expected source-project to be %q, got %q",
						tt.expectedSourceProject, config.CopyProjects.SourceProject)
				}
			} else {
				if config.CopyProjects.SourceProject != "" {
					t.Errorf("Expected source-project to be empty, got %q", config.CopyProjects.SourceProject)
				}
			}

			// Check target-owner parsing
			if tt.shouldHaveTarget {
				if config.CopyProjects.TargetOwner != tt.expectedTargetOwner {
					t.Errorf("Expected target-owner to be %q, got %q",
						tt.expectedTargetOwner, config.CopyProjects.TargetOwner)
				}
			} else {
				if config.CopyProjects.TargetOwner != "" {
					t.Errorf("Expected target-owner to be empty, got %q", config.CopyProjects.TargetOwner)
				}
			}

			// Build the consolidated safe outputs job and check environment variables
			workflowData := &WorkflowData{
				Name:        "test-workflow",
				SafeOutputs: config,
			}

			job, _, err := compiler.buildConsolidatedSafeOutputsJob(workflowData, "agent", "test.md")
			if err != nil {
				t.Fatalf("Failed to build consolidated safe outputs job: %v", err)
			}

			if job == nil {
				t.Fatalf("Expected consolidated safe outputs job to be created, but it was nil")
			}

			// Convert job to YAML to check for configuration in the project handler manager JSON
			yamlStr := strings.Join(job.Steps, "")

			if !strings.Contains(yamlStr, "GH_AW_SAFE_OUTPUTS_PROJECT_HANDLER_CONFIG") {
				t.Fatalf("Expected project handler manager config env var to be present, but it was not found.\nGenerated YAML:\n%s", yamlStr)
			}

			// Check for source-project configuration in JSON
			if tt.shouldHaveSource {
				expectedFragment := `\"source_project\":\"` + tt.expectedSourceProject + `\"`
				if !strings.Contains(yamlStr, expectedFragment) {
					// Fallback: tolerate unescaped JSON if output format changes
					fallbackFragment := `"source_project":"` + tt.expectedSourceProject + `"`
					if !strings.Contains(yamlStr, fallbackFragment) {
						t.Errorf("Expected source_project (%q or %q) to be present in project handler manager config, but it was not found.\nGenerated YAML:\n%s",
							expectedFragment, fallbackFragment, yamlStr)
					}
				}
			}

			// Check for target-owner configuration in JSON
			if tt.shouldHaveTarget {
				expectedFragment := `\"target_owner\":\"` + tt.expectedTargetOwner + `\"`
				if !strings.Contains(yamlStr, expectedFragment) {
					// Fallback: tolerate unescaped JSON if output format changes
					fallbackFragment := `"target_owner":"` + tt.expectedTargetOwner + `"`
					if !strings.Contains(yamlStr, fallbackFragment) {
						t.Errorf("Expected target_owner (%q or %q) to be present in project handler manager config, but it was not found.\nGenerated YAML:\n%s",
							expectedFragment, fallbackFragment, yamlStr)
					}
				}
			}
		})
	}
}
