package workflow

import (
	"strings"
	"testing"
)

func TestSafeOutputsGitHubTokenConfiguration(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	t.Run("Should parse github-token configuration in safe-outputs", func(t *testing.T) {
		frontmatter := map[string]any{
			"name": "Test Workflow",
			"safe-outputs": map[string]any{
				"create-issue": nil,
				"github-token": "${{ secrets.CUSTOM_PAT }}",
			},
		}

		config := compiler.extractSafeOutputsConfig(frontmatter)
		if config == nil {
			t.Fatal("Expected SafeOutputsConfig to be parsed")
		}

		if config.GitHubToken != "${{ secrets.CUSTOM_PAT }}" {
			t.Errorf("Expected GitHubToken to be '${{ secrets.CUSTOM_PAT }}', got '%s'", config.GitHubToken)
		}
	})

	t.Run("Should handle missing github-token field", func(t *testing.T) {
		frontmatter := map[string]any{
			"name": "Test Workflow",
			"safe-outputs": map[string]any{
				"create-issue": nil,
			},
		}

		config := compiler.extractSafeOutputsConfig(frontmatter)
		if config == nil {
			t.Fatal("Expected SafeOutputsConfig to be parsed")
		}

		if config.GitHubToken != "" {
			t.Errorf("Expected GitHubToken to be empty, got '%s'", config.GitHubToken)
		}
	})

	t.Run("Should handle non-string github-token field", func(t *testing.T) {
		frontmatter := map[string]any{
			"name": "Test Workflow",
			"safe-outputs": map[string]any{
				"create-issue": nil,
				"github-token": 123, // Invalid type
			},
		}

		config := compiler.extractSafeOutputsConfig(frontmatter)
		if config == nil {
			t.Fatal("Expected SafeOutputsConfig to be parsed")
		}

		if config.GitHubToken != "" {
			t.Errorf("Expected GitHubToken to be empty when non-string, got '%s'", config.GitHubToken)
		}
	})
}

func TestSafeOutputsGitHubTokenIntegration(t *testing.T) {
	tests := []struct {
		name             string
		frontmatter      map[string]any
		expectedInWith   []string
		unexpectedInWith []string
	}{
		{
			name: "create-issue with github-token",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"create-issue": nil,
					"github-token": "${{ secrets.CUSTOM_PAT }}",
				},
			},
			expectedInWith: []string{
				"github-token: ${{ secrets.CUSTOM_PAT }}",
			},
			unexpectedInWith: []string{},
		},
		{
			name: "create-issue without github-token",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"create-issue": nil,
				},
			},
			expectedInWith:   []string{},
			unexpectedInWith: []string{"github-token:"},
		},
		{
			name: "multiple safe outputs with github-token",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"safe-outputs": map[string]any{
					"create-issue":        nil,
					"add-comment":         nil,
					"create-pull-request": nil,
					"github-token":        "${{ secrets.GITHUB_TOKEN }}",
				},
			},
			expectedInWith: []string{
				"github-token: ${{ secrets.GITHUB_TOKEN }}",
			},
			unexpectedInWith: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			config := compiler.extractSafeOutputsConfig(tt.frontmatter)

			// Test the config generation in safe output jobs by creating a mock workflow data
			workflowData := &WorkflowData{
				Name:        "Test Workflow",
				SafeOutputs: config,
			}

			// Test create-issue job if configured
			if config != nil && config.CreateIssues != nil {
				job, err := compiler.buildCreateOutputIssueJob(workflowData, "main")
				if err != nil {
					t.Fatalf("Failed to build create issue job: %v", err)
				}

				jobYAML := strings.Join(job.Steps, "")

				// Check expected strings are present
				for _, expected := range tt.expectedInWith {
					if !strings.Contains(jobYAML, expected) {
						t.Errorf("Expected '%s' to be present in job YAML, but it was not found", expected)
					}
				}

				// Check unexpected strings are not present
				for _, unexpected := range tt.unexpectedInWith {
					if strings.Contains(jobYAML, unexpected) {
						t.Errorf("Expected '%s' to NOT be present in job YAML, but it was found", unexpected)
					}
				}
			}

			// Test add-comment job if configured
			if config != nil && config.AddComments != nil {
				job, err := compiler.buildCreateOutputAddCommentJob(workflowData, "main")
				if err != nil {
					t.Fatalf("Failed to build add issue comment job: %v", err)
				}

				jobYAML := strings.Join(job.Steps, "")

				// Check expected strings are present
				for _, expected := range tt.expectedInWith {
					if !strings.Contains(jobYAML, expected) {
						t.Errorf("Expected '%s' to be present in add comment job YAML, but it was not found", expected)
					}
				}

				// Check unexpected strings are not present
				for _, unexpected := range tt.unexpectedInWith {
					if strings.Contains(jobYAML, unexpected) {
						t.Errorf("Expected '%s' to NOT be present in add comment job YAML, but it was found", unexpected)
					}
				}
			}

			// Test create-pull-request job if configured
			if config != nil && config.CreatePullRequests != nil {
				job, err := compiler.buildCreateOutputPullRequestJob(workflowData, "main")
				if err != nil {
					t.Fatalf("Failed to build create pull request job: %v", err)
				}

				jobYAML := strings.Join(job.Steps, "")

				// Check expected strings are present
				for _, expected := range tt.expectedInWith {
					if !strings.Contains(jobYAML, expected) {
						t.Errorf("Expected '%s' to be present in create PR job YAML, but it was not found", expected)
					}
				}

				// Check unexpected strings are not present
				for _, unexpected := range tt.unexpectedInWith {
					if strings.Contains(jobYAML, unexpected) {
						t.Errorf("Expected '%s' to NOT be present in create PR job YAML, but it was found", unexpected)
					}
				}
			}
		})
	}
}

func TestAddSafeOutputGitHubTokenFunction(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	t.Run("Should add github-token when configured", func(t *testing.T) {
		workflowData := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				GitHubToken: "${{ secrets.CUSTOM_PAT }}",
			},
		}

		var steps []string
		compiler.addSafeOutputGitHubToken(&steps, workflowData)

		if len(steps) != 1 {
			t.Fatalf("Expected 1 step to be added, got %d", len(steps))
		}

		expectedStep := "          github-token: ${{ secrets.CUSTOM_PAT }}\n"
		if steps[0] != expectedStep {
			t.Errorf("Expected step '%s', got '%s'", expectedStep, steps[0])
		}
	})

	t.Run("Should add default github-token when not configured", func(t *testing.T) {
		workflowData := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				GitHubToken: "",
			},
		}

		var steps []string
		compiler.addSafeOutputGitHubToken(&steps, workflowData)

		if len(steps) != 1 {
			t.Fatalf("Expected 1 step to be added, got %d", len(steps))
		}

		expectedStep := "          github-token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}\n"
		if steps[0] != expectedStep {
			t.Errorf("Expected step '%s', got '%s'", expectedStep, steps[0])
		}
	})

	t.Run("Should add default github-token when SafeOutputs is nil", func(t *testing.T) {
		workflowData := &WorkflowData{
			SafeOutputs: nil,
		}

		var steps []string
		compiler.addSafeOutputGitHubToken(&steps, workflowData)

		if len(steps) != 1 {
			t.Fatalf("Expected 1 step to be added, got %d", len(steps))
		}

		expectedStep := "          github-token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}\n"
		if steps[0] != expectedStep {
			t.Errorf("Expected step '%s', got '%s'", expectedStep, steps[0])
		}
	})
}

func TestIndividualConfigGitHubTokenConfiguration(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	t.Run("Should parse github-token in create-issue configuration", func(t *testing.T) {
		frontmatter := map[string]any{
			"name": "Test Workflow",
			"safe-outputs": map[string]any{
				"create-issue": map[string]any{
					"github-token": "${{ secrets.ISSUE_PAT }}",
				},
			},
		}

		config := compiler.extractSafeOutputsConfig(frontmatter)
		if config == nil {
			t.Fatal("Expected SafeOutputsConfig to be parsed")
		}

		if config.CreateIssues == nil {
			t.Fatal("Expected CreateIssues config to be parsed")
		}

		if config.CreateIssues.GitHubToken != "${{ secrets.ISSUE_PAT }}" {
			t.Errorf("Expected CreateIssues GitHubToken to be '${{ secrets.ISSUE_PAT }}', got '%s'", config.CreateIssues.GitHubToken)
		}
	})

	t.Run("Should parse github-token in create-pull-request configuration", func(t *testing.T) {
		frontmatter := map[string]any{
			"name": "Test Workflow",
			"safe-outputs": map[string]any{
				"create-pull-request": map[string]any{
					"draft":        true,
					"github-token": "${{ secrets.PR_PAT }}",
				},
			},
		}

		config := compiler.extractSafeOutputsConfig(frontmatter)
		if config == nil {
			t.Fatal("Expected SafeOutputsConfig to be parsed")
		}

		if config.CreatePullRequests == nil {
			t.Fatal("Expected CreatePullRequests config to be parsed")
		}

		if config.CreatePullRequests.GitHubToken != "${{ secrets.PR_PAT }}" {
			t.Errorf("Expected CreatePullRequests GitHubToken to be '${{ secrets.PR_PAT }}', got '%s'", config.CreatePullRequests.GitHubToken)
		}
	})

	t.Run("Should parse github-token in add-comment configuration", func(t *testing.T) {
		frontmatter := map[string]any{
			"name": "Test Workflow",
			"safe-outputs": map[string]any{
				"add-comment": map[string]any{
					"max":          3,
					"github-token": "${{ secrets.COMMENT_PAT }}",
				},
			},
		}

		config := compiler.extractSafeOutputsConfig(frontmatter)
		if config == nil {
			t.Fatal("Expected SafeOutputsConfig to be parsed")
		}

		if config.AddComments == nil {
			t.Fatal("Expected AddComments config to be parsed")
		}

		if config.AddComments.GitHubToken != "${{ secrets.COMMENT_PAT }}" {
			t.Errorf("Expected AddComments GitHubToken to be '${{ secrets.COMMENT_PAT }}', got '%s'", config.AddComments.GitHubToken)
		}
	})

	t.Run("Should parse github-token in add-labels configuration", func(t *testing.T) {
		frontmatter := map[string]any{
			"name": "Test Workflow",
			"safe-outputs": map[string]any{
				"add-labels": map[string]any{
					"allowed":      []string{"bug", "feature"},
					"github-token": "${{ secrets.LABELS_PAT }}",
				},
			},
		}

		config := compiler.extractSafeOutputsConfig(frontmatter)
		if config == nil {
			t.Fatal("Expected SafeOutputsConfig to be parsed")
		}

		if config.AddLabels == nil {
			t.Fatal("Expected AddLabels config to be parsed")
		}

		if config.AddLabels.GitHubToken != "${{ secrets.LABELS_PAT }}" {
			t.Errorf("Expected AddLabels GitHubToken to be '${{ secrets.LABELS_PAT }}', got '%s'", config.AddLabels.GitHubToken)
		}
	})

	t.Run("Should prefer per-config token over global token", func(t *testing.T) {
		frontmatter := map[string]any{
			"name": "Test Workflow",
			"safe-outputs": map[string]any{
				"github-token": "${{ secrets.GLOBAL_PAT }}",
				"create-issue": map[string]any{
					"github-token": "${{ secrets.SPECIFIC_PAT }}",
				},
				"create-pull-request": map[string]any{
					"draft": true,
					// No github-token specified, should use global
				},
			},
		}

		config := compiler.extractSafeOutputsConfig(frontmatter)
		if config == nil {
			t.Fatal("Expected SafeOutputsConfig to be parsed")
		}

		// Global token should be set
		if config.GitHubToken != "${{ secrets.GLOBAL_PAT }}" {
			t.Errorf("Expected global GitHubToken to be '${{ secrets.GLOBAL_PAT }}', got '%s'", config.GitHubToken)
		}

		// create-issue should have specific token
		if config.CreateIssues == nil {
			t.Fatal("Expected CreateIssues config to be parsed")
		}
		if config.CreateIssues.GitHubToken != "${{ secrets.SPECIFIC_PAT }}" {
			t.Errorf("Expected CreateIssues GitHubToken to be '${{ secrets.SPECIFIC_PAT }}', got '%s'", config.CreateIssues.GitHubToken)
		}

		// create-pull-request should have empty token (will fall back to global during job generation)
		if config.CreatePullRequests == nil {
			t.Fatal("Expected CreatePullRequests config to be parsed")
		}
		if config.CreatePullRequests.GitHubToken != "" {
			t.Errorf("Expected CreatePullRequests GitHubToken to be empty, got '%s'", config.CreatePullRequests.GitHubToken)
		}
	})

	t.Run("Should test addSafeOutputGitHubTokenForConfig method", func(t *testing.T) {
		data := &WorkflowData{
			SafeOutputs: &SafeOutputsConfig{
				GitHubToken: "${{ secrets.GLOBAL_PAT }}",
			},
		}

		var steps []string

		// Test with per-config token
		compiler.addSafeOutputGitHubTokenForConfig(&steps, data, "${{ secrets.SPECIFIC_PAT }}")

		if len(steps) != 1 {
			t.Fatalf("Expected 1 step to be added, got %d", len(steps))
		}

		expectedStep := "          github-token: ${{ secrets.SPECIFIC_PAT }}\n"
		if steps[0] != expectedStep {
			t.Errorf("Expected step to be '%s', got '%s'", expectedStep, steps[0])
		}

		// Test fallback to global token
		steps = []string{} // Reset
		compiler.addSafeOutputGitHubTokenForConfig(&steps, data, "")

		if len(steps) != 1 {
			t.Fatalf("Expected 1 step to be added, got %d", len(steps))
		}

		expectedStep = "          github-token: ${{ secrets.GLOBAL_PAT }}\n"
		if steps[0] != expectedStep {
			t.Errorf("Expected step to be '%s', got '%s'", expectedStep, steps[0])
		}

		// Test with no tokens
		steps = []string{} // Reset
		data.SafeOutputs.GitHubToken = ""
		compiler.addSafeOutputGitHubTokenForConfig(&steps, data, "")

		if len(steps) != 0 {
			t.Fatalf("Expected 0 steps to be added when no tokens available, got %d", len(steps))
		}
	})
}
