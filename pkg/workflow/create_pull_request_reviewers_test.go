package workflow

import (
	"strings"
	"testing"
)

func TestCreatePullRequestJobWithReviewers(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test with reviewers configured
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{
				Reviewers: []string{"user1", "user2", "bot-reviewer"},
			},
		},
	}

	job, err := c.buildCreateOutputPullRequestJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building create pull request job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Check that checkout step is included before reviewer steps
	if !strings.Contains(stepsContent, "Checkout repository for gh CLI") {
		t.Error("Expected checkout step for gh CLI")
	}

	// Verify that checkout step is conditional on PR creation
	checkoutPattern := "Checkout repository for gh CLI"
	checkoutIndex := strings.Index(stepsContent, checkoutPattern)
	if checkoutIndex == -1 {
		t.Error("Expected checkout step")
	} else {
		// Check that conditional appears after the checkout step name
		afterCheckout := stepsContent[checkoutIndex:]
		if !strings.Contains(afterCheckout, "if: steps.create_pull_request.outputs.pull_request_url != ''") {
			t.Error("Expected checkout step to be conditional on PR creation")
		}
	}

	// Check that reviewer steps are included
	if !strings.Contains(stepsContent, "Add user1 as reviewer") {
		t.Error("Expected reviewer step for user1")
	}
	if !strings.Contains(stepsContent, "Add user2 as reviewer") {
		t.Error("Expected reviewer step for user2")
	}
	if !strings.Contains(stepsContent, "Add bot-reviewer as reviewer") {
		t.Error("Expected reviewer step for bot-reviewer")
	}

	// Check that gh pr edit command is present
	if !strings.Contains(stepsContent, "gh pr edit") {
		t.Error("Expected gh pr edit command in steps")
	}

	// Check that --add-reviewer flag is present
	if !strings.Contains(stepsContent, "--add-reviewer") {
		t.Error("Expected --add-reviewer flag in gh pr edit command")
	}

	// Check that PR_URL environment variable is set from step output
	if !strings.Contains(stepsContent, "PR_URL: ${{ steps.create_pull_request.outputs.pull_request_url }}") {
		t.Error("Expected PR_URL to be set from create_pull_request step output")
	}

	// Check that condition is set to only run if pull_request_url is not empty
	if !strings.Contains(stepsContent, "if: steps.create_pull_request.outputs.pull_request_url != ''") {
		t.Error("Expected conditional if statement for reviewer steps")
	}

	// Verify that GH_TOKEN is set with proper token expression
	if !strings.Contains(stepsContent, "GH_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}") {
		t.Error("Expected GH_TOKEN environment variable to be set with proper token expression")
	}

	// Verify that checkout uses actions/checkout@v5
	if !strings.Contains(stepsContent, "uses: actions/checkout@v5") {
		t.Error("Expected checkout to use actions/checkout@v5")
	}
}

func TestCreatePullRequestJobWithoutReviewers(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test without reviewers
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{
				// No reviewers configured
			},
		},
	}

	job, err := c.buildCreateOutputPullRequestJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building create pull request job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Check that no reviewer steps are included
	if strings.Contains(stepsContent, "Add") && strings.Contains(stepsContent, "as reviewer") {
		t.Error("Did not expect reviewer steps when no reviewers configured")
	}
	if strings.Contains(stepsContent, "gh pr edit") {
		t.Error("Did not expect gh pr edit command when no reviewers configured")
	}
}

func TestCreatePullRequestJobWithSingleReviewer(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test with a single reviewer
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{
				Reviewers: []string{"single-user"},
			},
		},
	}

	job, err := c.buildCreateOutputPullRequestJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building create pull request job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Check that single reviewer step is included
	if !strings.Contains(stepsContent, "Add single-user as reviewer") {
		t.Error("Expected reviewer step for single-user")
	}

	// Check that gh pr edit command is present
	if !strings.Contains(stepsContent, "gh pr edit") {
		t.Error("Expected gh pr edit command in steps")
	}

	// Verify environment variable for reviewer
	if !strings.Contains(stepsContent, `REVIEWER: "single-user"`) {
		t.Error("Expected REVIEWER environment variable to be set")
	}
}

func TestParsePullRequestsConfigWithReviewers(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test parsing reviewers from config (array format)
	outputMap := map[string]any{
		"create-pull-request": map[string]any{
			"title-prefix": "[test] ",
			"labels":       []any{"bug", "enhancement"},
			"reviewers":    []any{"user1", "user2", "github-bot"},
		},
	}

	config := c.parsePullRequestsConfig(outputMap)
	if config == nil {
		t.Fatal("Expected parsePullRequestsConfig to return non-nil config")
	}

	if len(config.Reviewers) != 3 {
		t.Errorf("Expected 3 reviewers, got %d", len(config.Reviewers))
	}

	expectedReviewers := []string{"user1", "user2", "github-bot"}
	for i, expected := range expectedReviewers {
		if i >= len(config.Reviewers) {
			t.Errorf("Missing reviewer at index %d, expected %s", i, expected)
			continue
		}
		if config.Reviewers[i] != expected {
			t.Errorf("Reviewer at index %d: expected %s, got %s", i, expected, config.Reviewers[i])
		}
	}
}

func TestParsePullRequestsConfigWithSingleStringReviewer(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test parsing reviewers from config (string format)
	outputMap := map[string]any{
		"create-pull-request": map[string]any{
			"title-prefix": "[test] ",
			"labels":       []any{"bug"},
			"reviewers":    "single-user",
		},
	}

	config := c.parsePullRequestsConfig(outputMap)
	if config == nil {
		t.Fatal("Expected parsePullRequestsConfig to return non-nil config")
	}

	if len(config.Reviewers) != 1 {
		t.Errorf("Expected 1 reviewer, got %d", len(config.Reviewers))
	}

	if config.Reviewers[0] != "single-user" {
		t.Errorf("Expected reviewer 'single-user', got %s", config.Reviewers[0])
	}
}

func TestCreatePullRequestJobWithCopilotReviewer(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test with "copilot" as reviewer (should use GitHub API with copilot-pull-request-reviewer[bot])
	workflowData := &WorkflowData{
		Name: "test-workflow",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{
				Reviewers: []string{"copilot"},
			},
		},
	}

	job, err := c.buildCreateOutputPullRequestJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building create pull request job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Check that the step name shows "copilot"
	if !strings.Contains(stepsContent, "Add copilot as reviewer") {
		t.Error("Expected reviewer step name to show 'copilot'")
	}

	// Check that it uses the GitHub API (not gh pr edit)
	if !strings.Contains(stepsContent, "gh api --method POST") {
		t.Error("Expected GitHub API call for copilot reviewer")
	}

	// Check that it uses the correct API endpoint and bot name
	if !strings.Contains(stepsContent, "/requested_reviewers") {
		t.Error("Expected /requested_reviewers API endpoint")
	}

	if !strings.Contains(stepsContent, "copilot-pull-request-reviewer[bot]") {
		t.Error("Expected copilot-pull-request-reviewer[bot] as the reviewer")
	}

	// Check that PR_NUMBER environment variable is used (not PR_URL)
	if !strings.Contains(stepsContent, "PR_NUMBER: ${{ steps.create_pull_request.outputs.pull_request_number }}") {
		t.Error("Expected PR_NUMBER to be set from create_pull_request step output")
	}

	// Verify that gh pr edit is NOT used for copilot
	if strings.Contains(stepsContent, "gh pr edit") && strings.Contains(stepsContent, "copilot") {
		t.Error("Should not use gh pr edit for copilot reviewer")
	}
}

func TestCreatePullRequestJobWithCustomGitHubToken(t *testing.T) {
	// Create a compiler instance
	c := NewCompiler(false, "", "test")

	// Test with custom GitHub token configuration
	workflowData := &WorkflowData{
		Name:        "test-workflow",
		GitHubToken: "${{ secrets.CUSTOM_PAT }}",
		SafeOutputs: &SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					GitHubToken: "${{ secrets.PR_SPECIFIC_PAT }}",
				},
				Reviewers: []string{"user1"},
			},
		},
	}

	job, err := c.buildCreateOutputPullRequestJob(workflowData, "main_job")
	if err != nil {
		t.Fatalf("Unexpected error building create pull request job: %v", err)
	}

	// Convert steps to a single string for testing
	stepsContent := strings.Join(job.Steps, "")

	// Check that the PR-specific token is used (highest precedence)
	if !strings.Contains(stepsContent, "GH_TOKEN: ${{ secrets.PR_SPECIFIC_PAT }}") {
		t.Error("Expected PR-specific GitHub token to be used in reviewer steps")
	}

	// Verify default token is NOT used
	if strings.Contains(stepsContent, "GH_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN") {
		t.Error("Did not expect default token when custom token is configured")
	}
}
