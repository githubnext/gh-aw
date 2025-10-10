package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestAddCommentTargetRepoIntegration(t *testing.T) {
	tests := []struct {
		name                     string
		frontmatter              map[string]any
		expectedTargetRepoInYAML string
		shouldHaveTargetRepo     bool
		trialLogicalRepoSlug     string
		expectedTargetRepoValue  string
	}{
		{
			name: "target-repo configuration should set GITHUB_AW_TARGET_REPO_SLUG",
			frontmatter: map[string]any{
				"name":   "Test Workflow",
				"engine": "copilot",
				"safe-outputs": map[string]any{
					"add-comment": map[string]any{
						"max":         5,
						"target":      "*",
						"target-repo": "github/customer-feedback",
					},
				},
			},
			shouldHaveTargetRepo:    true,
			expectedTargetRepoValue: "github/customer-feedback",
		},
		{
			name: "target-repo should take precedence over trial target repo",
			frontmatter: map[string]any{
				"name":   "Test Workflow",
				"engine": "copilot",
				"safe-outputs": map[string]any{
					"add-comment": map[string]any{
						"max":         5,
						"target":      "*",
						"target-repo": "github/customer-feedback",
					},
				},
			},
			trialLogicalRepoSlug:    "trial/repo",
			shouldHaveTargetRepo:    true,
			expectedTargetRepoValue: "github/customer-feedback", // Should prefer config over trial
		},
		{
			name: "no target-repo should fall back to trial target repo",
			frontmatter: map[string]any{
				"name":   "Test Workflow",
				"engine": "copilot",
				"safe-outputs": map[string]any{
					"add-comment": map[string]any{
						"max":    5,
						"target": "*",
					},
				},
			},
			trialLogicalRepoSlug:    "trial/repo",
			shouldHaveTargetRepo:    true,
			expectedTargetRepoValue: "trial/repo",
		},
		{
			name: "no target-repo and no trial should not set GITHUB_AW_TARGET_REPO_SLUG",
			frontmatter: map[string]any{
				"name":   "Test Workflow",
				"engine": "copilot",
				"safe-outputs": map[string]any{
					"add-comment": map[string]any{
						"max":    5,
						"target": "*",
					},
				},
			},
			trialLogicalRepoSlug: "", // explicitly empty
			shouldHaveTargetRepo: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for this test
			tempDir := t.TempDir()
			workflowPath := tempDir + "/test-workflow.md"

			// Create a simple workflow content
			workflowContent := "# Test Workflow\n\nThis is a test workflow for target-repo functionality."

			// Write the workflow file
			err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write workflow file: %v", err)
			}

			// Create compiler with trial mode if needed
			compiler := NewCompiler(false, "", "")
			if tt.trialLogicalRepoSlug != "" {
				compiler.SetTrialMode(true)
				compiler.SetTrialLogicalRepoSlug(tt.trialLogicalRepoSlug)
			}

			// Parse workflow data
			workflowData := &WorkflowData{
				Name: tt.frontmatter["name"].(string),
			}

			// Extract safe outputs configuration
			workflowData.SafeOutputs = compiler.extractSafeOutputsConfig(tt.frontmatter)

			if workflowData.SafeOutputs == nil || workflowData.SafeOutputs.AddComments == nil {
				t.Fatal("Expected AddComments configuration to be parsed")
			}

			// Build the add comment job
			job, err := compiler.buildCreateOutputAddCommentJob(workflowData, "main")
			if err != nil {
				t.Fatalf("Failed to build add comment job: %v", err)
			}

			// Convert steps to string to check for GITHUB_AW_TARGET_REPO_SLUG
			jobYAML := strings.Join(job.Steps, "")

			if tt.shouldHaveTargetRepo {
				expectedEnvVar := "GITHUB_AW_TARGET_REPO_SLUG: \"" + tt.expectedTargetRepoValue + "\""
				if !strings.Contains(jobYAML, expectedEnvVar) {
					t.Errorf("Expected to find %s in job YAML, but didn't.\nActual job YAML:\n%s", expectedEnvVar, jobYAML)
				}
			} else {
				// Check specifically for the environment variable declaration, not the JavaScript reference
				if strings.Contains(jobYAML, "GITHUB_AW_TARGET_REPO_SLUG: \"") {
					t.Errorf("Expected not to find GITHUB_AW_TARGET_REPO_SLUG environment variable declaration in job YAML when no target-repo is configured.\nActual job YAML:\n%s", jobYAML)
				}
			}
		})
	}
}
