//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeOutputsEnvIntegration(t *testing.T) {
	tests := []struct {
		name               string
		frontmatter        map[string]any
		expectedEnvVars    []string
		expectedSafeOutput string
	}{
		{
			name: "Create issue job with custom env vars",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"on":   "push",
				"safe-outputs": map[string]any{
					"create-issue": nil,
					"env": map[string]any{
						"GITHUB_TOKEN": "${{ secrets.SOME_PAT_FOR_AGENTIC_WORKFLOWS }}",
						"DEBUG_MODE":   "true",
					},
				},
			},
			expectedEnvVars: []string{
				"GITHUB_TOKEN: ${{ secrets.SOME_PAT_FOR_AGENTIC_WORKFLOWS }}",
				"DEBUG_MODE: true",
			},
			expectedSafeOutput: "create-issue",
		},
		{
			name: "Create pull request job with custom env vars",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"on":   "push",
				"safe-outputs": map[string]any{
					"create-pull-request": nil,
					"env": map[string]any{
						"CUSTOM_API_KEY": "${{ secrets.CUSTOM_API_KEY }}",
						"ENVIRONMENT":    "production",
					},
				},
			},
			expectedEnvVars: []string{
				"CUSTOM_API_KEY: ${{ secrets.CUSTOM_API_KEY }}",
				"ENVIRONMENT: production",
			},
			expectedSafeOutput: "create-pull-request",
		},
		{
			name: "Add issue comment job with custom env vars",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"on":   "issues",
				"safe-outputs": map[string]any{
					"add-comment": nil,
					"env": map[string]any{
						"NOTIFICATION_URL": "${{ secrets.WEBHOOK_URL }}",
						"COMMENT_TEMPLATE": "template-v2",
					},
				},
			},
			expectedEnvVars: []string{
				"NOTIFICATION_URL: ${{ secrets.WEBHOOK_URL }}",
				"COMMENT_TEMPLATE: template-v2",
			},
			expectedSafeOutput: "add-comment",
		},
		{
			name: "Multiple safe outputs with shared env vars",
			frontmatter: map[string]any{
				"name": "Test Workflow",
				"on":   "push",
				"safe-outputs": map[string]any{
					"create-issue":        nil,
					"create-pull-request": nil,
					"env": map[string]any{
						"SHARED_TOKEN": "${{ secrets.SHARED_TOKEN }}",
						"WORKFLOW_ID":  "multi-output-test",
					},
				},
			},
			expectedEnvVars: []string{
				"SHARED_TOKEN: ${{ secrets.SHARED_TOKEN }}",
				"WORKFLOW_ID: multi-output-test",
			},
			expectedSafeOutput: "create-issue,create-pull-request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")

			// Extract the safe outputs configuration
			config := compiler.extractSafeOutputsConfig(tt.frontmatter)
			if config == nil {
				t.Fatal("Expected SafeOutputsConfig to be parsed")
			}

			// Verify env configuration is parsed correctly
			if config.Env == nil {
				t.Fatal("Expected Env to be parsed")
			}

			// Build workflow data
			data := &WorkflowData{
				Name:            "Test",
				FrontmatterName: "Test Workflow",
				SafeOutputs:     config,
			}

			// Test job generation for each safe output type
			if strings.Contains(tt.expectedSafeOutput, "create-issue") {
				job, err := compiler.buildCreateOutputIssueJob(data, "main_job", false, tt.frontmatter)
				if err != nil {
					t.Errorf("Error building create issue job: %v", err)
				}

				// Verify environment variables are included in job steps
				jobYAML := strings.Join(job.Steps, "")
				for _, expectedEnvVar := range tt.expectedEnvVars {
					if !strings.Contains(jobYAML, expectedEnvVar) {
						t.Errorf("Expected env var %q not found in create issue job YAML", expectedEnvVar)
					}
				}
			}

			if strings.Contains(tt.expectedSafeOutput, "create-pull-request") {
				job, err := compiler.buildCreateOutputPullRequestJob(data, "main_job")
				if err != nil {
					t.Errorf("Error building create pull request job: %v", err)
				}

				// Verify environment variables are included in job steps
				jobYAML := strings.Join(job.Steps, "")
				for _, expectedEnvVar := range tt.expectedEnvVars {
					if !strings.Contains(jobYAML, expectedEnvVar) {
						t.Errorf("Expected env var %q not found in create pull request job YAML", expectedEnvVar)
					}
				}
			}

			if strings.Contains(tt.expectedSafeOutput, "add-comment") {
				job, err := compiler.buildCreateOutputAddCommentJob(data, "main_job")
				if err != nil {
					t.Errorf("Error building add issue comment job: %v", err)
				}

				// Verify environment variables are included in job steps
				jobYAML := strings.Join(job.Steps, "")
				for _, expectedEnvVar := range tt.expectedEnvVars {
					if !strings.Contains(jobYAML, expectedEnvVar) {
						t.Errorf("Expected env var %q not found in add issue comment job YAML", expectedEnvVar)
					}
				}
			}
		})
	}
}

func TestSafeOutputsEnvFullWorkflowCompilation(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test workflow file
	workflowContent := `---
name: Test Environment Variables
on: push
safe-outputs:
  create-issue:
    title-prefix: "[env-test] "
    labels: ["automated", "env-test"]
  env:
    GITHUB_TOKEN: ${{ secrets.SOME_PAT_FOR_AGENTIC_WORKFLOWS }}
    DEBUG_MODE: "true"
    CUSTOM_API_KEY: ${{ secrets.CUSTOM_API_KEY }}
---

# Environment Variables Test Workflow

This workflow tests that custom environment variables are properly passed through
to safe output jobs.

Create an issue with test results.
`

	workflowFile := filepath.Join(tmpDir, "test-env-workflow.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow file: %v", err)
	}

	// Parse the workflow data to get the structure (using the same approach as existing tests)
	compiler := NewCompiler(false, "", "test")
	workflowData, err := compiler.ParseWorkflowFile(workflowFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify the SafeOutputs configuration includes our environment variables
	if workflowData.SafeOutputs == nil {
		t.Fatal("Expected SafeOutputs to be parsed")
	}

	if workflowData.SafeOutputs.Env == nil {
		t.Fatal("Expected Env to be parsed")
	}

	expectedEnvVars := map[string]string{
		"GITHUB_TOKEN":   "${{ secrets.SOME_PAT_FOR_AGENTIC_WORKFLOWS }}",
		"DEBUG_MODE":     "true",
		"CUSTOM_API_KEY": "${{ secrets.CUSTOM_API_KEY }}",
	}

	for key, expectedValue := range expectedEnvVars {
		if actualValue, exists := workflowData.SafeOutputs.Env[key]; !exists {
			t.Errorf("Expected env key %s to exist", key)
		} else if actualValue != expectedValue {
			t.Errorf("Expected env[%s] to be %q, got %q", key, expectedValue, actualValue)
		}
	}

	// Build the create issue job and verify it includes our environment variables
	job, err := compiler.buildCreateOutputIssueJob(workflowData, "main_job", false, nil)
	if err != nil {
		t.Fatalf("Failed to build create issue job: %v", err)
	}

	jobYAML := strings.Join(job.Steps, "")

	for key, expectedValue := range expectedEnvVars {
		expectedEnvLine := key + ": " + expectedValue
		if !strings.Contains(jobYAML, expectedEnvLine) {
			t.Errorf("Expected environment variable %q not found in job YAML", expectedEnvLine)
		}
	}

	// Verify issue configuration is present
	if !strings.Contains(jobYAML, "GITHUB_AW_ISSUE_TITLE_PREFIX: \"[env-test] \"") {
		t.Error("Expected issue title prefix not found in job YAML")
	}

	if !strings.Contains(jobYAML, "GITHUB_AW_ISSUE_LABELS: \"automated,env-test\"") {
		t.Error("Expected issue labels not found in job YAML")
	}

	t.Logf("✓ %s", workflowFile)
}

func TestSafeOutputsEnvWithStagedMode(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test workflow file with staged mode and env vars
	workflowContent := `---
name: Test Environment Variables with Staged Mode
on: push
safe-outputs:
  create-issue:
  env:
    GITHUB_TOKEN: ${{ secrets.SOME_PAT_FOR_AGENTIC_WORKFLOWS }}
    DEBUG_MODE: "true"
  staged: true
---

# Environment Variables with Staged Mode Test

This workflow tests that custom environment variables work with staged mode.
`

	workflowFile := filepath.Join(tmpDir, "test-env-staged-workflow.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow file: %v", err)
	}

	// Parse the workflow data
	compiler := NewCompiler(false, "", "test")
	workflowData, err := compiler.ParseWorkflowFile(workflowFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify staged mode is enabled
	if !workflowData.SafeOutputs.Staged {
		t.Error("Expected staged mode to be enabled")
	}

	// Build the create issue job and verify it includes our environment variables and staged flag
	job, err := compiler.buildCreateOutputIssueJob(workflowData, "main_job", false, nil)
	if err != nil {
		t.Fatalf("Failed to build create issue job: %v", err)
	}

	jobYAML := strings.Join(job.Steps, "")

	expectedEnvVars := []string{
		"GITHUB_TOKEN: ${{ secrets.SOME_PAT_FOR_AGENTIC_WORKFLOWS }}",
		"DEBUG_MODE: true",
	}

	for _, expectedEnvVar := range expectedEnvVars {
		if !strings.Contains(jobYAML, expectedEnvVar) {
			t.Errorf("Expected environment variable %q not found in job YAML", expectedEnvVar)
		}
	}

	// Verify staged mode is enabled
	if !strings.Contains(jobYAML, "GITHUB_AW_SAFE_OUTPUTS_STAGED: \"true\"") {
		t.Error("Expected staged mode flag not found in job YAML")
	}

	t.Logf("✓ %s", workflowFile)
}
