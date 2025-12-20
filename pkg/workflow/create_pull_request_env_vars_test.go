package workflow

import (
	"strings"
	"testing"
)

func TestCreatePullRequestStepConfigEnvVars(t *testing.T) {
	// Create a compiler instance for consolidated mode
	c := NewCompiler(false, "", "test")

	// Create workflow data with create-pull-request configuration
	workflowData := &WorkflowData{
		Name:       "test-workflow",
		AIReaction: "emoji", // Enable reaction to test comment env vars
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{
				TitlePrefix:   "[TEST] ",
				Labels:        []string{"automated", "test"},
				AllowedLabels: []string{"automated", "test", "bug"},
				Draft:         boolPtr(false),
				IfNoChanges:   "error",
				AllowEmpty:    true,
				Expires:       7,
			},
			MaximumPatchSize: 2048,
		},
	}

	// Build the step config
	stepConfig := c.buildCreatePullRequestStepConfig(workflowData, "main_job", false)

	// Convert custom env vars to a single string for testing
	envVarsContent := strings.Join(stepConfig.CustomEnvVars, "")

	// Verify required environment variables are present
	requiredEnvVars := map[string]string{
		"GH_AW_WORKFLOW_ID":       `GH_AW_WORKFLOW_ID: "main_job"`,
		"GH_AW_BASE_BRANCH":       "GH_AW_BASE_BRANCH: ${{ github.ref_name }}",
		"GH_AW_PR_TITLE_PREFIX":   `GH_AW_PR_TITLE_PREFIX: "[TEST] "`,
		"GH_AW_PR_LABELS":         `GH_AW_PR_LABELS: "automated,test"`,
		"GH_AW_PR_ALLOWED_LABELS": `GH_AW_PR_ALLOWED_LABELS: "automated,test,bug"`,
		"GH_AW_PR_DRAFT":          `GH_AW_PR_DRAFT: "false"`,
		"GH_AW_PR_IF_NO_CHANGES":  `GH_AW_PR_IF_NO_CHANGES: "error"`,
		"GH_AW_PR_ALLOW_EMPTY":    `GH_AW_PR_ALLOW_EMPTY: "true"`,
		"GH_AW_MAX_PATCH_SIZE":    "GH_AW_MAX_PATCH_SIZE: 2048",
		"GH_AW_PR_EXPIRES":        `GH_AW_PR_EXPIRES: "7"`,
	}

	for envVarName, expectedContent := range requiredEnvVars {
		if !strings.Contains(envVarsContent, expectedContent) {
			t.Errorf("Expected %s to be present in environment variables.\nExpected: %s\nGot env vars:\n%s",
				envVarName, expectedContent, envVarsContent)
		}
	}

	// Verify comment-related env vars are present when reaction is enabled
	if !strings.Contains(envVarsContent, "GH_AW_COMMENT_ID") {
		t.Error("Expected GH_AW_COMMENT_ID to be present when AIReaction is enabled")
	}
	if !strings.Contains(envVarsContent, "GH_AW_COMMENT_REPO") {
		t.Error("Expected GH_AW_COMMENT_REPO to be present when AIReaction is enabled")
	}
}

func TestCreatePullRequestStepConfigDefaultValues(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Create minimal workflow data to test defaults
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{
				// Using all defaults
			},
		},
	}

	// Build the step config
	stepConfig := c.buildCreatePullRequestStepConfig(workflowData, "agent", false)

	// Convert custom env vars to a single string for testing
	envVarsContent := strings.Join(stepConfig.CustomEnvVars, "")

	// Verify default values
	defaultEnvVars := map[string]string{
		"GH_AW_WORKFLOW_ID":      `GH_AW_WORKFLOW_ID: "agent"`,
		"GH_AW_BASE_BRANCH":      "GH_AW_BASE_BRANCH: ${{ github.ref_name }}",
		"GH_AW_PR_DRAFT":         `GH_AW_PR_DRAFT: "true"`,         // Default is true
		"GH_AW_PR_IF_NO_CHANGES": `GH_AW_PR_IF_NO_CHANGES: "warn"`, // Default is warn
		"GH_AW_PR_ALLOW_EMPTY":   `GH_AW_PR_ALLOW_EMPTY: "false"`,  // Default is false
		"GH_AW_MAX_PATCH_SIZE":   "GH_AW_MAX_PATCH_SIZE: 1024",     // Default is 1024
	}

	for envVarName, expectedContent := range defaultEnvVars {
		if !strings.Contains(envVarsContent, expectedContent) {
			t.Errorf("Expected default %s to be present.\nExpected: %s\nGot env vars:\n%s",
				envVarName, expectedContent, envVarsContent)
		}
	}

	// Verify expires is NOT present when not set
	if strings.Contains(envVarsContent, "GH_AW_PR_EXPIRES") {
		t.Error("Expected GH_AW_PR_EXPIRES to NOT be present when Expires is 0")
	}

	// Verify comment env vars are NOT present when reaction is not enabled
	if strings.Contains(envVarsContent, "GH_AW_COMMENT_ID") {
		t.Error("Expected GH_AW_COMMENT_ID to NOT be present when AIReaction is not enabled")
	}
}

func TestCreatePullRequestStepConfigWithTargetRepo(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Create workflow data with target-repo (cross-repo PR)
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{
				TargetRepoSlug: "owner/repo",
				Expires:        7, // Should be ignored for cross-repo PRs
			},
		},
	}

	// Build the step config
	stepConfig := c.buildCreatePullRequestStepConfig(workflowData, "main_job", false)

	// Convert custom env vars to a single string for testing
	envVarsContent := strings.Join(stepConfig.CustomEnvVars, "")

	// Verify expires is NOT present for cross-repo PRs
	if strings.Contains(envVarsContent, "GH_AW_PR_EXPIRES") {
		t.Error("Expected GH_AW_PR_EXPIRES to NOT be present for cross-repo PRs (when TargetRepoSlug is set)")
	}

	// Verify target repo is passed to standard env vars builder
	if !strings.Contains(envVarsContent, "GH_AW_TARGET_REPO_SLUG") {
		t.Error("Expected GH_AW_TARGET_REPO_SLUG to be present for cross-repo PRs")
	}
}
